package acpserver

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// testConsent is the consent a background request under a ChatGPT plan
// session needs, as the SDK names it.
const testConsent = "chatgpt_plan.background_use"

// answerFile is a test's stored consent: a file holding granted or declined,
// which each read of the gate takes as it is, as the gobble CLI's reads
// auth.json. Another writer changes it, as a second process does.
type answerFile struct {
	path string
	// plan is the provider whose credential is a plan session.
	plan string
}

func newAnswerFile(t *testing.T, plan, answer string) *answerFile {
	t.Helper()
	f := &answerFile{path: filepath.Join(t.TempDir(), "answer"), plan: plan}
	f.write(t, answer)
	return f
}

func (f *answerFile) write(t *testing.T, answer string) {
	t.Helper()
	if err := os.WriteFile(f.path, []byte(answer), 0o600); err != nil {
		t.Fatal(err)
	}
}

// read is Options.PlanConsent over the file.
func (f *answerFile) read(id string) (ConsentState, error) {
	if id != f.plan {
		return ConsentState{}, nil
	}
	b, err := os.ReadFile(f.path)
	if err != nil {
		return ConsentState{Consent: testConsent}, err
	}
	switch strings.TrimSpace(string(b)) {
	case "granted":
		return ConsentState{Consent: testConsent, Plan: true, Granted: true}, nil
	case "declined":
		return ConsentState{Consent: testConsent, Plan: true}, nil
	}
	return ConsentState{Consent: testConsent}, errors.New("answer file: the stored answer does not parse")
}

// planLogins is a test's auth.json holding a ChatGPT plan session, which the
// gate reads through the real provider.PlanConsent over auth.Logins, as the CLI
// does, for the one provider id plan; withdraw changes the answer through a
// second auth.Logins on the same file, as another process does.
type planLogins struct {
	path string
	plan string
}

func newPlanLogins(t *testing.T, plan string, granted bool) *planLogins {
	t.Helper()
	f := &planLogins{path: filepath.Join(t.TempDir(), "auth.json"), plan: plan}
	err := auth.Logins{Path: f.path}.Save(t.Context(), "openai", &provider.Login{Access: "a", Refresh: "r", Expires: time.Now().Add(time.Hour),
		Issuer: "https://auth.openai.com", ClientID: "oaiapp_test", Resource: "https://api.openai.com/v1", Kind: "chatgpt_plan"})
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.SetConsent(t.Context(), auth.Logins{Path: f.path}, "openai", "background", granted); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *planLogins) withdraw(t *testing.T) {
	t.Helper()
	if err := provider.SetConsent(t.Context(), auth.Logins{Path: f.path}, "openai", testConsent, false); err != nil {
		t.Fatal(err)
	}
}

func (f *planLogins) read(id string) (ConsentState, error) {
	if id != f.plan {
		return ConsentState{}, nil
	}
	st, err := provider.PlanConsent("openai", auth.Logins{Path: f.path})
	return ConsentState{Consent: st.Consent, Plan: st.Plan, Granted: st.Granted}, err
}

// Under a ChatGPT plan session, the small model's title is a background
// request, asked only with the user's consent, and with none it shows the
// client nothing (0052-MADR D2).
func TestTitleUnderPlan(t *testing.T) {
	for _, answer := range []string{"declined", "granted"} {
		t.Run(answer, func(t *testing.T) {
			small := llmtest.NewScript(llmtest.Text("Plan title"))
			f := newAnswerFile(t, "small", answer)
			h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), small, func(o *Options) { o.PlanConsent = f.read })
			id := newSession(t, h.c, t.TempDir(), nil).SessionId
			if _, err := prompt(t, h.c, id, "Name me"); err != nil {
				t.Fatal(err)
			}
			if answer == "granted" {
				waitFor(t, func() bool { return h.titled("Plan title") })
				return
			}
			h.ag.mu.Lock()
			s := h.ag.sessions[id]
			h.ag.mu.Unlock()
			h.ag.closeTitle(s) // a title, had one started, is waited for
			if n := len(small.Requests()); n != 0 {
				t.Fatalf("%d title requests without consent, want none", n)
			}
			for _, u := range h.client.Updates() {
				if u.Update.SessionInfoUpdate != nil {
					t.Errorf("the client was sent %+v for a title that was not made", u.Update.SessionInfoUpdate)
				}
			}
		})
	}
}

// Under a ChatGPT plan session, automatic compaction is a background
// request, made only with the user's consent: without it the threshold is
// passed by, and an overflow ends the turn, naming /compact and the command
// that allows it; with it, the session compacts as any other (0052-MADR D2).
func TestAutoCompactUnderPlan(t *testing.T) {
	declined := newAnswerFile(t, "fake", "declined")
	h := newCompactHarnessWith(t, harnessConfig{planConsent: declined.read}, SessionSettings{}, full("r1", 110000), llmtest.Text("after"))
	id := h.session()
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2)
	if len(h.notices()) != 0 || len(h.summaries()) != 0 {
		t.Fatalf("no consent, at the threshold: notices %v, %d summaries; want none", h.notices(), len(h.summaries()))
	}

	overflow := llmtest.Turn{Err: llm.ErrContextOverflow}
	o := newCompactHarnessWith(t, harnessConfig{planConsent: declined.read}, SessionSettings{}, full("r1", 10), overflow, llmtest.Text("never"))
	id = o.session()
	o.mustPrompt(id, p1)
	err := o.prompt(id, p2)
	const want = "automatic compaction runs under your ChatGPT plan only with your consent: run /compact, or allow it with " +
		"gobble auth consent fake chatgpt_plan.background_use allow"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("no consent, an overflow: err %v, want the turn ended saying %q", err, want)
	}
	if len(o.summaries()) != 0 {
		t.Fatalf("no consent, an overflow: %d summaries, want none", len(o.summaries()))
	}

	granted := newAnswerFile(t, "fake", "granted")
	with := newCompactHarnessWith(t, harnessConfig{planConsent: granted.read}, SessionSettings{}, full("r1", 110000), llmtest.Text("SUMMARY"),
		llmtest.Text("after"))
	id = with.session()
	with.mustPrompt(id, p1)
	with.mustPrompt(id, p2)
	if len(with.summaries()) != 1 {
		t.Fatalf("with consent: %d summaries, want the threshold's one", len(with.summaries()))
	}
}

// The answer is read at each request, not once for a session: a session that
// started with a grant, and a consent withdrawn by another writer after it,
// makes no automatic compaction and no title from the next request on
// (0052-MADR D2).
func TestConsentAppliesFromTheNextRequest(t *testing.T) {
	t.Run("compaction", func(t *testing.T) {
		f := newPlanLogins(t, "fake", true)
		h := newCompactHarnessWith(t, harnessConfig{planConsent: f.read}, SessionSettings{},
			full("r1", 110000), llmtest.Text("SUMMARY"), full("r2", 110000), full("r3", 110000), full("r4", 110000), full("r5", 110000), full("r6", 110000))
		id := h.session()
		h.mustPrompt(id, p1)
		h.mustPrompt(id, p2)
		if len(h.summaries()) != 1 {
			t.Fatalf("with the grant: %d summaries, want one", len(h.summaries()))
		}
		f.withdraw(t) // a second process withdraws it
		h.mustPrompt(id, p1+" again")
		h.mustPrompt(id, p2+" again")
		if len(h.summaries()) != 1 {
			t.Fatalf("after the withdrawal: %d summaries, want still one", len(h.summaries()))
		}
	})
	t.Run("title", func(t *testing.T) {
		f := newPlanLogins(t, "small", true)
		small := llmtest.NewScript(llmtest.Text("Never made"))
		h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), small, func(o *Options) { o.PlanConsent = f.read })
		id := newSession(t, h.c, t.TempDir(), nil).SessionId // started with the grant
		f.withdraw(t)
		if _, err := prompt(t, h.c, id, "Name me"); err != nil {
			t.Fatal(err)
		}
		h.ag.mu.Lock()
		s := h.ag.sessions[id]
		h.ag.mu.Unlock()
		h.ag.closeTitle(s)
		if n := len(small.Requests()); n != 0 {
			t.Fatalf("%d title requests after the withdrawal, want none", n)
		}
	})
}

// markedProvider records, for each request, whether its context carried the
// background mark and whether it was a compaction's summary.
type markedProvider struct {
	llm.Provider
	mu    sync.Mutex
	calls []markedCall
}

type markedCall struct{ summary, marked bool }

func (m *markedProvider) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	m.mu.Lock()
	m.calls = append(m.calls, markedCall{summary: strings.HasPrefix(systemOf(req), "You are a context summarization assistant."), marked: llm.IsBackground(ctx)})
	m.mu.Unlock()
	return m.Provider.Stream(ctx, req)
}

func (m *markedProvider) seen() []markedCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]markedCall(nil), m.calls...)
}

// A title's request and an automatic compaction's reach the provider
// marked as background, as the SDK needs to refuse one without a consent; a
// user's turn and /compact do not (0052-MADR D2).
func TestBackgroundRequestsAreMarked(t *testing.T) {
	t.Run("title", func(t *testing.T) {
		small := &markedProvider{Provider: llmtest.NewScript(llmtest.Text("A title"))}
		main := &markedProvider{Provider: llmtest.NewScript(llmtest.Text("ok"))}
		h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("unused")), small, func(o *Options) {
			o.Provider = func(string, llm.Retry) (llm.Provider, error) { return main, nil }
		})
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		if _, err := prompt(t, h.c, id, "Name me"); err != nil {
			t.Fatal(err)
		}
		waitFor(t, func() bool { return h.titled("A title") })
		if got := small.seen(); len(got) != 1 || !got[0].marked {
			t.Errorf("the title's requests %+v, want one, marked", got)
		}
		if got := main.seen(); len(got) != 1 || got[0].marked {
			t.Errorf("the user's turn's requests %+v, want one, unmarked", got)
		}
	})
	t.Run("compaction", func(t *testing.T) {
		var rec *markedProvider
		h := newCompactHarnessWith(t, harnessConfig{wrap: func(s *llmtest.Script) llm.Provider {
			rec = &markedProvider{Provider: s}
			return rec
		}}, SessionSettings{}, full("r1", 110000), llmtest.Text("SUMMARY"), llmtest.Text("after"), llmtest.Text("third"), llmtest.Text("MANUAL"))
		id := h.session()
		h.mustPrompt(id, p1)
		h.mustPrompt(id, p2) // over the threshold: an automatic compaction
		h.mustPrompt(id, p1+" third")
		h.mustPrompt(id, "/compact")
		var autoSummaries, manualSummaries int
		for i, c := range rec.seen() {
			switch {
			case c.summary && c.marked:
				autoSummaries++
			case c.summary && !c.marked:
				manualSummaries++
			case c.marked:
				t.Errorf("request %d, a user's turn, was marked", i)
			}
		}
		if autoSummaries != 1 || manualSummaries != 1 {
			t.Errorf("summaries: %d marked and %d unmarked, want the automatic one marked and /compact's not: %+v", autoSummaries, manualSummaries, rec.seen())
		}
	})
}

// With the stored answer unreadable, nothing is a grant: no title and no
// automatic compaction at the threshold is made, each leaving a line in the
// log and nothing for the client; an overflow ends the turn with an error
// that gives the store's error, names /compact and not the command, and
// writes nothing (0052-MADR D2).
func TestBackgroundRefusedWhenAnswerUnreadable(t *testing.T) {
	t.Run("title", func(t *testing.T) {
		f := newAnswerFile(t, "small", "granted")
		small := llmtest.NewScript(llmtest.Text("Never made"))
		h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), small, func(o *Options) { o.PlanConsent = f.read })
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		f.write(t, "not an answer")
		if _, err := prompt(t, h.c, id, "Name me"); err != nil {
			t.Fatal(err)
		}
		h.ag.mu.Lock()
		s := h.ag.sessions[id]
		h.ag.mu.Unlock()
		h.ag.closeTitle(s)
		if n := len(small.Requests()); n != 0 {
			t.Fatalf("%d title requests with an unreadable answer, want none", n)
		}
		if got := h.logs.String(); !strings.Contains(got, "provider=small") || !strings.Contains(got, "does not parse") {
			t.Errorf("the log %q lacks a line naming the provider and the store's error", got)
		}
	})
	t.Run("compaction", func(t *testing.T) {
		f := newAnswerFile(t, "fake", "granted")
		h := newCompactHarnessWith(t, harnessConfig{planConsent: f.read}, SessionSettings{}, full("r1", 110000), llmtest.Text("after"))
		id := h.session()
		f.write(t, "not an answer")
		h.mustPrompt(id, p1)
		h.mustPrompt(id, p2)
		if len(h.notices()) != 0 || len(h.summaries()) != 0 {
			t.Fatalf("an unreadable answer, at the threshold: notices %v, %d summaries; want none", h.notices(), len(h.summaries()))
		}
	})
	t.Run("overflow", func(t *testing.T) {
		f := newAnswerFile(t, "fake", "granted")
		h := newCompactHarnessWith(t, harnessConfig{planConsent: f.read}, SessionSettings{}, full("r1", 10), llmtest.Turn{Err: llm.ErrContextOverflow},
			llmtest.Text("never"))
		id := h.session()
		h.mustPrompt(id, p1)
		f.write(t, "not an answer")
		before, err := os.ReadFile(f.path)
		if err != nil {
			t.Fatal(err)
		}
		err = h.prompt(id, p2)
		if err == nil || !strings.Contains(err.Error(), "could not be read (answer file: the stored answer does not parse)") ||
			!strings.Contains(err.Error(), "run /compact") || strings.Contains(err.Error(), "gobble auth consent") {
			t.Fatalf("an unreadable answer, an overflow: err %v, want the store's error, /compact and not the command", err)
		}
		if len(h.summaries()) != 0 {
			t.Fatalf("%d summaries, want none", len(h.summaries()))
		}
		if after, _ := os.ReadFile(f.path); !bytes.Equal(before, after) {
			t.Errorf("the answer file was written: %q, was %q", after, before)
		}
	})
}

// gateSequence is a Options.PlanConsent whose reads answer in turn, then
// repeat the last.
func gateSequence(plan string, answers ...func() (ConsentState, error)) func(string) (ConsentState, error) {
	var n atomic.Int32
	return func(id string) (ConsentState, error) {
		if id != plan {
			return ConsentState{}, nil
		}
		i := min(int(n.Add(1))-1, len(answers)-1)
		return answers[i]()
	}
}

func granted() (ConsentState, error) {
	return ConsentState{Consent: testConsent, Plan: true, Granted: true}, nil
}
func declined() (ConsentState, error) { return ConsentState{Consent: testConsent, Plan: true}, nil }
func unreadable() (ConsentState, error) {
	return ConsentState{Consent: testConsent}, errors.New("auth.json: unreadable")
}

// A turn refused for a consent ends with session/prompt's error whose data
// has kind not_permitted and, only when allowing the consent would let the
// request through, the consent's id: the gate's overflow refusal, and an
// overflow compaction that a stale grant lets through and the SDK refuses
// while the stored answer reads declined, carry it; an answer that cannot be
// read, and the SDK's refusal while it cannot, do not (0052-MADR D2).
func TestRefusedBackgroundCarriesConsent(t *testing.T) {
	// errPlanOverflow's text, which row 1 of the MADR's table gives for the gate's refusal and for the SDK's after a stale grant.
	overflowText := errPlanOverflow("fake", ConsentState{Consent: testConsent}).Error()
	// The shape production hands acpserver when the SDK refuses a marked request: the SDK's own text (go-llmprovider-sdk's openai
	// checkBackground), wrapped by llm/provider's mapErrorAt as llm.ErrNotPermitted and then by the adapter's attempt in an
	// llm.ConsentError with the consent's id (llm/provider/provider.go).
	const sdkText = "llmprovider: not permitted: openai: a background request on a ChatGPT plan session needs the consent " +
		testConsent + ", which the user has not granted (0075-MADR D5)"
	refused := func() llmtest.Turn {
		return llmtest.Turn{Err: &llm.ConsentError{Consent: testConsent, Err: fmt.Errorf("%w: %w", llm.ErrNotPermitted, errors.New(sdkText))}}
	}
	overflow := llmtest.Turn{Err: llm.ErrContextOverflow}
	// A granted answer the SDK refused anyway has no consent to offer, and ends the turn as an overflow whose compaction failed
	// does: the sentence of an overflow whose compaction failed, then the refusal as the adapter built it.
	exactGrantedRefusal := "the context is over the model's window, and compacting it failed: " + llm.ErrNotPermitted.Error() + ": " + sdkText
	for _, c := range []struct {
		name    string
		reads   []func() (ConsentState, error)
		turns   []llmtest.Turn
		consent string
		text    string
		exact   string // the whole text, when production's is known
	}{
		{"the gate's refusal", []func() (ConsentState, error){declined}, []llmtest.Turn{full("r1", 10), overflow, llmtest.Text("never")},
			testConsent, overflowText, ""},
		{"the SDK's refusal, the answer declined", []func() (ConsentState, error){granted, declined}, []llmtest.Turn{full("r1", 10), overflow, refused()},
			testConsent, overflowText, ""},
		{"the unreadable answer", []func() (ConsentState, error){unreadable}, []llmtest.Turn{full("r1", 10), overflow, llmtest.Text("never")},
			"", "could not be read (auth.json: unreadable): run /compact", ""},
		{"the SDK's refusal, the answer unreadable", []func() (ConsentState, error){granted, unreadable}, []llmtest.Turn{full("r1", 10), overflow, refused()},
			"", "could not be read (auth.json: unreadable): run /compact", ""},
		{"the SDK's refusal, the answer granted", []func() (ConsentState, error){granted, granted}, []llmtest.Turn{full("r1", 10), overflow, refused()},
			"", "a background request on a ChatGPT plan session needs the consent", exactGrantedRefusal},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newCompactHarnessWith(t, harnessConfig{planConsent: gateSequence("fake", c.reads...)}, SessionSettings{}, c.turns...)
			id := h.session()
			h.mustPrompt(id, p1)
			err := h.prompt(id, p2)
			code, data := wireData(t, err)
			var d acpext.ErrorData
			if jerr := json.Unmarshal([]byte(data), &d); jerr != nil {
				t.Fatal(jerr)
			}
			if code != -32603 || d.Kind != acpext.KindNotPermitted || d.Consent != c.consent || d.Provider != "fake" || !strings.Contains(d.Error, c.text) ||
				c.text == overflowText && d.Error != overflowText || c.exact != "" && d.Error != c.exact {
				t.Errorf("error %d %s, want kind not_permitted, consent %q, provider fake, naming %q", code, data, c.consent, c.text)
			}
		})
	}
}

// A title that is not made sends the client nothing and fails no prompt,
// whether the answer is declined or cannot be read; the turn's own reply
// comes back as it would without a title (0052-PLAN P3, the row of
// TestRefusedBackgroundCarriesConsent).
func TestTitleNotMadeSendsNothing(t *testing.T) {
	for name, read := range map[string]func() (ConsentState, error){"declined": declined, "unreadable": unreadable} {
		t.Run(name, func(t *testing.T) {
			small := llmtest.NewScript(llmtest.Text("Never made"))
			h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), small, func(o *Options) {
				o.PlanConsent = gateSequence("small", read)
			})
			id := newSession(t, h.c, t.TempDir(), nil).SessionId
			if resp, err := prompt(t, h.c, id, "Name me"); err != nil || resp.StopReason != acp.StopReasonEndTurn {
				t.Fatalf("prompt %+v, %v; want the turn to end as it would without a title", resp, err)
			}
			h.ag.mu.Lock()
			s := h.ag.sessions[id]
			h.ag.mu.Unlock()
			h.ag.closeTitle(s)
			if n := len(small.Requests()); n != 0 {
				t.Errorf("%d title requests, want none", n)
			}
			for _, u := range h.client.Updates() {
				if u.Update.SessionInfoUpdate != nil {
					t.Errorf("the client was sent %+v for a title that was not made", u.Update.SessionInfoUpdate)
				}
			}
		})
	}
}

// The title's gate is read only when a title would otherwise be asked: once
// for a session's first prompt, and never for the prompts after it, whether
// the title was made or not (0052-MADR D2; 0052-PLAN P3, deviation 12).
func TestTitleGateReadsOnlyWhenATitleIsAsked(t *testing.T) {
	for name, read := range map[string]func() (ConsentState, error){"granted": granted, "declined": declined} {
		t.Run(name, func(t *testing.T) {
			var reads atomic.Int32
			gate := gateSequence("small", read)
			small := llmtest.NewScript(llmtest.Text("A title"))
			h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("one"), llmtest.Text("two"), llmtest.Text("three")), small, func(o *Options) {
				o.PlanConsent = func(id string) (ConsentState, error) {
					if id == "small" {
						reads.Add(1)
					}
					return gate(id)
				}
			})
			id := newSession(t, h.c, t.TempDir(), nil).SessionId
			for _, text := range []string{"Name me", "second", "third"} {
				if _, err := prompt(t, h.c, id, text); err != nil {
					t.Fatal(err)
				}
			}
			if n := reads.Load(); n != 1 {
				t.Errorf("the gate was read %d times over three prompts, want once", n)
			}
		})
	}
}
