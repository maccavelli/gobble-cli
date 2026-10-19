package acpserver

import (
	"encoding/json/v2"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/personality"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// personalityHarness is an agent over the store in dir whose sessions
// start with ss, recording the settings it saves.
type personalityHarness struct {
	c      *acptest.Conn
	ag     *Agent
	client *acptest.Client
	model  *llmtest.Script

	mu    sync.Mutex
	saved []string
}

func newPersonalityHarness(t *testing.T, dir string, ss SessionSettings, client *acptest.Client, turns int) *personalityHarness {
	t.Helper()
	var script []llmtest.Turn
	for i := range turns {
		script = append(script, llmtest.Text(fmt.Sprintf("reply %d", i+1)))
	}
	h := &personalityHarness{client: client, model: llmtest.NewScript(script...)}
	h.ag = New(Options{Version: "1.2.3", Store: jsonl.New(dir, jsonl.Options{}), Models: func() ModelChoice { return testModels },
		Provider: func(string, llm.Retry) (llm.Provider, error) { return h.model, nil },
		Settings: func(string) SessionSettings { return ss },
		SaveSetting: func(path, value string) error {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.saved = append(h.saved, path+"="+value)
			return nil
		}})
	h.c = acptest.Connect(t, h.ag, client)
	h.ag.SetConnection(h.c.Agent)
	t.Cleanup(func() {
		if err := h.ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, h.c.Client)
	return h
}

func (h *personalityHarness) savedSettings() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.saved)
}

// lastPersonality is the last _gobble/personality notification for id.
func (h *personalityHarness) lastPersonality(t *testing.T, id acp.SessionId) personalityNotice {
	t.Helper()
	var last personalityNotice
	for _, n := range h.client.Notices() {
		if n.Method != methodPersonality {
			continue
		}
		var p personalityNotice
		if err := json.Unmarshal(n.Params, &p); err != nil {
			t.Fatal(err)
		}
		if p.SessionID == id {
			last = p
		}
	}
	return last
}

// state is _gobble/get_state's personality.
func (h *personalityHarness) state(t *testing.T, id acp.SessionId) string {
	t.Helper()
	got, _ := mustExt(t, h.c, "_gobble/get_state", map[string]any{"sessionId": id})["personality"].(string) //nolint:errcheck // compared below
	return got
}

// entries are the live session's entries of one custom type.
func (h *personalityHarness) entries(id acp.SessionId, custom string) []session.Entry {
	h.ag.mu.Lock()
	s := h.ag.sessions[id]
	h.ag.mu.Unlock()
	var out []session.Entry
	for _, e := range s.log.Entries() {
		if e.Type == session.TypeCustom && e.CustomType == custom {
			out = append(out, e)
		}
	}
	return out
}

func mustVoice(t *testing.T, name string) *personality.Voice {
	t.Helper()
	v, err := personality.New(personality.Options{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// personalityPatch is the personality section of the session's last prompt
// entry, which must be a patch: its text, and whether the patch has it.
func (h *personalityHarness) personalityPatch(t *testing.T, id acp.SessionId) (*string, bool) {
	t.Helper()
	prompts := h.entries(id, session.CustomPrompt)
	_, patch, err := sysprompt.ParseData(prompts[len(prompts)-1].Data)
	if err != nil || patch == nil {
		t.Fatalf("the last prompt entry is no patch: %s, %v", prompts[len(prompts)-1].Data, err)
	}
	for _, p := range patch {
		if p.Name == "personality" {
			return p.Text, true
		}
	}
	return nil, false
}

// /personality lists the personalities, and switches to and from every
// built-in, boring by its other names too: each switch answers in the new
// voice, is written as a gobble.personality entry, is reported by
// _gobble/personality and get_state, and reaches the model as a prompt
// patch at the next prompt. The level-5 question is asked once; its yes is
// saved (0022-PLAN P3).
func TestPersonalitySwitch(t *testing.T) {
	client := &acptest.Client{Allow: true}
	h := newPersonalityHarness(t, t.TempDir(), SessionSettings{Personality: personality.Dry}, client, 8)
	id := newSession(t, h.c, t.TempDir(), nil).SessionId
	if got := h.lastPersonality(t, id); got != (personalityNotice{SessionID: id, Name: personality.Dry, Level: 3}) {
		t.Fatalf("the start's notice %+v", got)
	}
	if _, err := prompt(t, h.c, id, "hello"); err != nil {
		t.Fatal(err)
	}
	hint := ""
	for _, u := range client.Updates() {
		if a := u.Update.AvailableCommandsUpdate; a != nil {
			for _, c := range a.AvailableCommands {
				if c.Name == cmdPersonality && c.Input != nil && c.Input.Unstructured != nil {
					hint = c.Input.Unstructured.Hint
				}
			}
		}
	}
	if hint != "[boring|dry|snarky|bombastic]" {
		t.Fatalf("/personality's hint %q, want 0022-MADR D6's", hint)
	}
	list, _ := slash(t, h.c, client, id, "/personality")
	for _, line := range []string{"  boring (1) — " + personality.BoringSummary, "  dry (3) — ", "  snarky (4) — ", "  bombastic (5) — "} {
		if !strings.Contains(list, line) {
			t.Errorf("/personality lacks %q:\n%s", line, list)
		}
	}
	if current, _, _ := strings.Cut(list, "\n"); !slices.Contains(mustVoice(t, personality.Dry).Variants(personality.PersonalityCurrent,
		personality.Args{"name": personality.Dry}), current) {
		t.Errorf("/personality's first line %q names no current dry", current)
	}
	steps := []struct{ arg, want string }{
		{"snarky", personality.Snarky}, {"bombastic", personality.Bombastic}, {"Plain", personality.Boring},
		{"dry", personality.Dry}, {"normal", personality.Boring}, {"bombastic", personality.Bombastic}, {"dry", personality.Dry},
	}
	for i, step := range steps {
		v := mustVoice(t, step.want)
		said, _ := slash(t, h.c, client, id, "/personality "+step.arg)
		if !slices.Contains(v.Variants(personality.PersonalitySet, personality.Args{"name": step.want}), said) {
			t.Errorf("/personality %s answered %q, want %s's", step.arg, said, step.want)
		}
		if got := h.lastPersonality(t, id); got != (personalityNotice{SessionID: id, Name: step.want, Level: v.Level()}) {
			t.Errorf("/personality %s: notice %+v", step.arg, got)
		}
		if got := h.state(t, id); got != step.want {
			t.Errorf("/personality %s: get_state %q", step.arg, got)
		}
		if es := h.entries(id, session.CustomPersonality); len(es) != i+1 || string(es[i].Data) != `{"name":"`+step.want+`"}` {
			t.Errorf("/personality %s: entries %d, last %s", step.arg, len(es), es[len(es)-1].Data)
		}
		if _, err := prompt(t, h.c, id, "go on"); err != nil {
			t.Fatal(err)
		}
		text, ok := h.personalityPatch(t, id)
		switch {
		case !ok:
			t.Errorf("/personality %s: the next prompt patched no personality", step.arg)
		case step.want == personality.Boring && text != nil:
			t.Errorf("/personality %s: the patch keeps a personality %q", step.arg, *text)
		case step.want != personality.Boring && (text == nil || *text != "<personality>\n"+v.Brief()+"\n</personality>"):
			t.Errorf("/personality %s: the patch is not %s's brief", step.arg, step.want)
		}
	}
	perms := client.Permissions()
	if len(perms) != 1 || perms[0].ToolCall.ToolCallId != InsultsCallID || *perms[0].ToolCall.Title != insultsQuestion(personality.Bombastic) {
		t.Fatalf("permissions %+v, want the level-5 question once", perms)
	}
	if got := h.savedSettings(); !slices.Equal(got, []string{"bombasticAccepted=true"}) {
		t.Fatalf("saved %q", got)
	}
	if got := insultsQuestion(personality.Bombastic); got !=
		"Bombastic insults you and your decisions. It does what you choose, and it keeps gobble's house rules. Switch to it" {
		t.Fatalf("the question %q is not 0022-MADR D10's", got)
	}
}

// _gobble/set_personality switches as /personality does, and a resumed
// session keeps the personality it last had, over its settings'; a name no
// personality has is invalid params (0022-PLAN P3).
func TestSetPersonalityResumed(t *testing.T) {
	dir := t.TempDir()
	client := &acptest.Client{}
	h := newPersonalityHarness(t, dir, SessionSettings{Personality: personality.Dry}, client, 2)
	cwd := t.TempDir()
	id := newSession(t, h.c, cwd, nil).SessionId
	if _, err := prompt(t, h.c, id, "hello"); err != nil {
		t.Fatal(err)
	}
	res := mustExt(t, h.c, "_gobble/set_personality", map[string]any{"sessionId": id, "name": "snarky"})
	if res["name"] != personality.Snarky || res["level"] != 4.0 {
		t.Fatalf("set_personality %v", res)
	}
	if got := h.lastPersonality(t, id); got.Name != personality.Snarky {
		t.Fatalf("notice %+v", got)
	}
	for _, name := range []string{"zany", ""} {
		if _, err := ext(t, h.c, "_gobble/set_personality", map[string]any{"sessionId": id, "name": name}); err == nil || requestCode(t, err) != -32602 {
			t.Fatalf("set_personality %q: %v, want invalid params", name, err)
		}
	}
	if err := h.ag.Close(); err != nil {
		t.Fatal(err)
	}
	r := newPersonalityHarness(t, dir, SessionSettings{Personality: personality.Dry}, &acptest.Client{}, 1)
	if _, err := r.c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if got := r.lastPersonality(t, id); got != (personalityNotice{SessionID: id, Name: personality.Snarky, Level: 4}) {
		t.Fatalf("the resumed session's notice %+v", got)
	}
	if got := r.state(t, id); got != personality.Snarky {
		t.Fatalf("the resumed session is %q", got)
	}
}

// A session that would start at level 5 asks first, once per user: a no,
// or no answer, gives dry, written and reported, with nothing said in the
// reply; a yes is saved and remembered for the process's other sessions;
// the setting's yes asks nothing; a command that chooses the voice is not
// held up by the question; and a declined /personality bombastic keeps the
// voice (0022-MADR D10).
func TestInsultsQuestion(t *testing.T) {
	bombastic, dry := mustVoice(t, personality.Bombastic), mustVoice(t, personality.Dry)
	hasBrief := func(h *personalityHarness, i int, v *personality.Voice) bool {
		return strings.Contains(systemOf(h.model.Requests()[i]), "<personality>\n"+v.Brief()+"\n</personality>")
	}

	t.Run("declined", func(t *testing.T) {
		client := &acptest.Client{}
		h := newPersonalityHarness(t, t.TempDir(), SessionSettings{Personality: personality.Bombastic}, client, 2)
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		said, _ := slash(t, h.c, client, id, "hello")
		if said != "reply 1" || !hasBrief(h, 0, dry) || h.state(t, id) != personality.Dry || h.lastPersonality(t, id).Name != personality.Dry {
			t.Fatalf("declined: said %q, state %q", said, h.state(t, id))
		}
		if _, err := prompt(t, h.c, id, "again"); err != nil {
			t.Fatal(err)
		}
		if n := len(client.Permissions()); n != 1 || len(h.savedSettings()) != 0 {
			t.Fatalf("declined: %d questions, saved %q", n, h.savedSettings())
		}
	})

	t.Run("accepted", func(t *testing.T) {
		client := &acptest.Client{Allow: true}
		h := newPersonalityHarness(t, t.TempDir(), SessionSettings{Personality: personality.Bombastic}, client, 3)
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		for _, text := range []string{"hello", "again"} {
			if _, err := prompt(t, h.c, id, text); err != nil {
				t.Fatal(err)
			}
		}
		other := newSession(t, h.c, t.TempDir(), nil).SessionId
		if _, err := prompt(t, h.c, other, "hello"); err != nil {
			t.Fatal(err)
		}
		if n := len(client.Permissions()); n != 1 || !hasBrief(h, 0, bombastic) || !hasBrief(h, 2, bombastic) ||
			!slices.Equal(h.savedSettings(), []string{"bombasticAccepted=true"}) {
			t.Fatalf("accepted: %d questions, saved %q", n, h.savedSettings())
		}
	})

	t.Run("setting", func(t *testing.T) {
		client := &acptest.Client{}
		h := newPersonalityHarness(t, t.TempDir(), SessionSettings{Personality: personality.Bombastic, BombasticAccepted: true}, client, 1)
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		if _, err := prompt(t, h.c, id, "hello"); err != nil {
			t.Fatal(err)
		}
		if n := len(client.Permissions()); n != 0 || !hasBrief(h, 0, bombastic) {
			t.Fatalf("setting: %d questions", n)
		}
	})

	t.Run("command first", func(t *testing.T) {
		client := &acptest.Client{}
		h := newPersonalityHarness(t, t.TempDir(), SessionSettings{Personality: personality.Bombastic}, client, 0)
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		said, _ := slash(t, h.c, client, id, "/personality boring")
		if n := len(client.Permissions()); n != 0 || said != "Personality: boring" || h.state(t, id) != personality.Boring {
			t.Fatalf("command first: %d questions, said %q", n, said)
		}
	})

	t.Run("other command", func(t *testing.T) {
		client := &acptest.Client{}
		h := newPersonalityHarness(t, t.TempDir(), SessionSettings{Personality: personality.Bombastic}, client, 0)
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		said, _ := slash(t, h.c, client, id, "/mode plan")
		if n := len(client.Permissions()); n != 1 || !slices.Contains(dry.Variants(personality.ModePlanOn, nil), said) {
			t.Fatalf("other command: %d questions, said %q", n, said)
		}
	})

	t.Run("declined command", func(t *testing.T) {
		client := &acptest.Client{}
		h := newPersonalityHarness(t, t.TempDir(), SessionSettings{Personality: personality.Dry}, client, 0)
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		said, _ := slash(t, h.c, client, id, "/personality bombastic")
		if !slices.Contains(dry.Variants(personality.PersonalityDeclined, personality.Args{"name": personality.Dry}), said) ||
			h.state(t, id) != personality.Dry || len(h.entries(id, session.CustomPersonality)) != 0 || len(client.Permissions()) != 1 {
			t.Fatalf("declined command: said %q, state %q", said, h.state(t, id))
		}
	})
}

// The hidden /humor N takes the built-in of level N: there is no 2, and
// another number is unknown. It is not advertised, nor in /help
// (0022-PLAN P3).
func TestHumor(t *testing.T) {
	client := &acptest.Client{Allow: true}
	h := newPersonalityHarness(t, t.TempDir(), SessionSettings{Personality: personality.Dry}, client, 0)
	id := newSession(t, h.c, t.TempDir(), nil).SessionId
	snarky := mustVoice(t, personality.Snarky)
	for _, c := range []struct {
		arg  string
		want []string
		now  string
	}{
		{"4", snarky.Variants(personality.PersonalitySet, personality.Args{"name": personality.Snarky}), personality.Snarky},
		{"2", snarky.Variants(personality.HumorTwo, nil), personality.Snarky},
		{"9", snarky.Variants(personality.HumorUnknown, personality.Args{"level": `"9"`}), personality.Snarky},
		{"1", []string{"Personality: boring"}, personality.Boring},
		{"", []string{"Personality: boring"}, personality.Boring},
		{"5", mustVoice(t, personality.Bombastic).Variants(personality.PersonalitySet, personality.Args{"name": personality.Bombastic}), personality.Bombastic},
		{"3", mustVoice(t, personality.Dry).Variants(personality.PersonalitySet, personality.Args{"name": personality.Dry}), personality.Dry},
	} {
		said, _ := slash(t, h.c, client, id, strings.TrimSpace("/humor "+c.arg))
		if !slices.Contains(c.want, said) || h.state(t, id) != c.now {
			t.Errorf("/humor %s: said %q, now %q", c.arg, said, h.state(t, id))
		}
	}
	help, _ := slash(t, h.c, client, id, "/help")
	if strings.Contains(help, "/humor") || slices.Contains(advertised(t, client), cmdHumor) {
		t.Fatalf("/humor is shown:\n%s", help)
	}
}
