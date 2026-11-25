package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// The tests of 0066-MADR's discovery at a session's start and model switch,
// through Options.Discover (0071-MADR P37, the owner's "Add Options.Discover
// (Recommended)"; 0071-PLAN Phase 2b).

// discoverRec is a fake Options.Discover: it records each provider/model it
// is asked for, waits delay (or, with block, its context's end), then calls
// then and returns err.
type discoverRec struct {
	delay time.Duration
	block bool
	err   error
	then  func(provider, model string)

	// blockRef blocks only that provider/model's discovery until its
	// context ends; ignore, when set, blocks every discovery until it is
	// closed, whatever its context does; lag is how long a blocked
	// discovery goes on after its context ends, before it closes ended.
	blockRef string
	ignore   chan struct{}
	lag      time.Duration

	mu    sync.Mutex
	calls []string
	ended chan struct{} // closed when a blocked discovery's context ends
}

func (d *discoverRec) discover(ctx context.Context, provider, model string) error {
	d.mu.Lock()
	d.calls = append(d.calls, provider+"/"+model)
	d.mu.Unlock()
	if d.ignore != nil {
		<-d.ignore
		return nil
	}
	if d.block || provider+"/"+model == d.blockRef {
		<-ctx.Done()
		time.Sleep(d.lag)
		if d.ended != nil {
			close(d.ended)
		}
		return ctx.Err()
	}
	select {
	case <-time.After(d.delay):
	case <-ctx.Done():
		return ctx.Err()
	}
	if d.then != nil {
		d.then(provider, model)
	}
	return d.err
}

func (d *discoverRec) seen() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.calls)
}

// discovered is a limit harness whose discovery states cap for each model
// once it ends, after delay: before it, the catalog knows nothing, and a
// call asks the assumed 16,384.
func discovered(t *testing.T, d *discoverRec, caps map[string]int, mod func(*Options), turns ...llmtest.Turn) *limitHarness {
	t.Helper()
	var h *limitHarness
	d.then = func(provider, model string) {
		h.setInfo(provider+"/"+model, wide(caps[provider+"/"+model]))
		if d.err != nil {
			h.setInfo(provider+"/"+model, llm.Model{})
		}
	}
	h = newLimitHarnessWith(t, SessionSettings{}, map[string]llm.Model{}, func(o *Options) {
		o.Discover = d.discover
		if mod != nil {
			mod(o)
		}
	}, turns...)
	return h
}

// claudeService is Anthropic's service in process: its listing of
// claude-test-1 (output 32,000) answers after 200 ms, and each streamed
// message answers "hello". It records each request in order, with the turn's
// max_tokens.
type claudeService struct {
	mu    sync.Mutex
	order []string
}

func (c *claudeService) RoundTrip(r *http.Request) (*http.Response, error) {
	reply := func(ct, body string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {ct}}, Body: io.NopCloser(strings.NewReader(body)),
			Request: r}, nil
	}
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/v1/models") {
		c.record("GET models")
		select {
		case <-time.After(200 * time.Millisecond):
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
		return reply("application/json", `{"data":[{"id":"claude-test-1","type":"model","display_name":"t","created_at":"2026-10-01T00:00:00Z",`+
			`"max_tokens":32000,"max_input_tokens":200000}],"has_more":false,"first_id":"claude-test-1","last_id":"claude-test-1"}`)
	}
	var body map[string]any
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal(b, &body) //nolint:errcheck // a body that is not JSON records no limit
	}
	limit, _ := body["max_tokens"].(float64)
	c.record(fmt.Sprintf("POST messages %d", int(limit)))
	ev := func(name, data string) string { return "event: " + name + "\ndata: " + data + "\n\n" }
	return reply("text/event-stream", ev("message_start", `{"type":"message_start","message":{"id":"m","type":"message","role":"assistant","content":[],"usage":{"input_tokens":12,"output_tokens":0}}}`)+
		ev("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)+
		ev("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`)+
		ev("content_block_stop", `{"type":"content_block_stop","index":0}`)+
		ev("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`)+
		ev("message_stop", `{"type":"message_stop"}`))
}

func (c *claudeService) record(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.order = append(c.order, s)
}

func (c *claudeService) seen() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.order)
}

// A new session discovers its model before its first call, which waits for
// it and carries what the listing states; a later call waits for nothing
// and lists nothing (0066-MADR S1, S9; through provider.Discover and a Claude
// service whose listing answers after 200 ms).
func TestSession_DiscoversBeforeFirstCall_0066(t *testing.T) {
	svc := &claudeService{}
	cache := &provider.Cache{}
	t.Cleanup(func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	o := provider.Options{APIKey: "sk-acp", Model: "claude-test-1", HTTPClient: &http.Client{Transport: svc}, Cache: cache,
		Retry: &llm.Retry{Off: true}}
	choice := ModelChoice{Provider: "claude", Models: []string{"claude-test-1"}, Default: "claude-test-1"}
	ag := New(Options{Version: "1.2.3", Store: jsonl.New(t.TempDir(), jsonl.Options{}),
		Provider:  func(string, llm.Retry) (llm.Provider, error) { return provider.New("claude", o) },
		Models:    func() ModelChoice { return choice },
		ModelInfo: func(id, model string) llm.Model { return provider.ModelInfoWith(o, id, model, nil) },
		Discover: func(ctx context.Context, id, model string) error {
			return provider.Discover(ctx, o, id, model, nil)
		}})
	client := &acptest.Client{Allow: true}
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	resp, err := prompt(t, c, id, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if got := svc.seen(); !slices.Equal(got, []string{"GET models", "POST messages 32000"}) {
		t.Errorf("requests %q, want the listing, then the turn carrying its 32,000", got)
	}
	wantMeta(t, resp, 32000, "stated")
	if _, err := prompt(t, c, id, "again"); err != nil {
		t.Fatal(err)
	}
	if got := svc.seen(); !slices.Equal(got, []string{"GET models", "POST messages 32000", "POST messages 32000"}) {
		t.Errorf("requests %q, want no listing before the second turn", got)
	}
}

// A loaded or resumed session discovers its model before its first call
// (0066-MADR S2).
func TestSession_LoadDiscovers_0066(t *testing.T) {
	for _, how := range []string{"load", "resume"} {
		t.Run(how, func(t *testing.T) {
			d := &discoverRec{delay: 100 * time.Millisecond}
			h := discovered(t, d, map[string]int{"fake/m1": 24000}, nil, llmtest.Text("one"), llmtest.Text("two"))
			id := h.session()
			h.mustPrompt(id, "first")
			if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
				t.Fatal(err)
			}
			h.setInfo("fake/m1", llm.Model{}) // what the run knew is gone; the reopened session must discover again
			var err error
			if how == "load" {
				_, err = h.c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
			} else {
				_, err = h.c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
			}
			if err != nil {
				t.Fatal(err)
			}
			h.mustPrompt(id, "second")
			if got := d.seen(); !slices.Equal(got, []string{"fake/m1", "fake/m1"}) || h.lastCap() != 24000 {
				t.Errorf("discoveries %q, cap %d; want one at the start and one at the %s, the call waiting for it", got, h.lastCap(), how)
			}
		})
	}
}

// A session's title model is discovered with its model, at its start
// (0066-MADR S1).
func TestSession_TitleModelDiscovered_0066(t *testing.T) {
	d := &discoverRec{}
	small := llmtest.NewScript(llmtest.Text("A title"))
	h := discovered(t, d, map[string]int{}, func(o *Options) {
		o.Title = TitleOptions{Provider: func(string, llm.Retry) (llm.Provider, error) { return small, nil }, ProviderID: "small", Model: "s1"}
	})
	h.session()
	waitFor(t, func() bool { return len(d.seen()) == 2 })
	if got := d.seen(); !slices.Contains(got, "fake/m1") || !slices.Contains(got, "small/s1") {
		t.Errorf("discoveries %q, want the session's model and the title model", got)
	}
}

// A model switch discovers the new model, to the same provider or another,
// and the next call waits for it; a thinking change discovers nothing
// (0066-MADR S5, S6, S41).
func TestSetConfig_SwitchDiscovers_0066(t *testing.T) {
	d := &discoverRec{delay: 100 * time.Millisecond}
	h := discovered(t, d, map[string]int{"fake/m1": 24000, "fake/m2": 12000, "other/x9": 6000}, nil, llmtest.Text("one"))
	id := h.session()
	h.mustPrompt(id, "first")
	if _, err := setOption(t, h.c, id, configModel, "m2"); err != nil {
		t.Fatal(err)
	}
	h.fake.set(llmtest.NewScript(llmtest.Text("two")))
	h.mustPrompt(id, "second")
	if got := h.lastCap(); got != 12000 {
		t.Errorf("after the switch to m2 the turn asked for %d, want m2's discovered 12,000", got)
	}
	if _, err := setOption(t, h.c, id, configModel, "other/x9"); err != nil {
		t.Fatal(err)
	}
	h.mustPrompt(id, "third")
	if r := h.other.Requests(); len(r) != 1 || r[0].MaxTokens != 6000 {
		t.Errorf("after the switch to other/x9 the turn asked %v, want x9's discovered 6,000", maxTokensOf(r))
	}
	if _, err := setOption(t, h.c, id, configThinking, "high"); err != nil {
		t.Fatal(err)
	}
	h.ag.waitDiscovery(t.Context(), h.live(id), "other", "x9") // what the next call would wait for
	if got := d.seen(); !slices.Equal(got, []string{"fake/m1", "fake/m2", "other/x9"}) {
		t.Errorf("discoveries %q, want one at the start and one per model switch, none for thinking", got)
	}
}

// maxTokensOf are rs' MaxTokens.
func maxTokensOf(rs []*llm.Request) []int {
	var out []int
	for _, r := range rs {
		out = append(out, r.MaxTokens)
	}
	return out
}

// A discovery that fails lets the call go on with what is cached (0066-MADR
// S7; S8, the SDK's bound, is llm/provider's TestSession_DiscoveryFails_0066).
func TestSession_DiscoveryFails_0066(t *testing.T) {
	d := &discoverRec{delay: 50 * time.Millisecond, err: errors.New("model limits: listing claude: 503")}
	h := discovered(t, d, map[string]int{"fake/m1": 24000}, nil, llmtest.Text("ok"))
	resp := h.mustPrompt(h.session(), "hello")
	if got := h.lastCap(); got != 16384 {
		t.Errorf("the turn asked for %d, want the assumed 16,384 of what is cached", got)
	}
	wantMeta(t, resp, 16384, "assumed")
}

// A cancel while the first call waits ends the turn, cancelled; closing the
// session cancels its discovery (0066-MADR S40).
func TestSession_CancelAndCloseWhileWaiting_0066(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		d := &discoverRec{block: true, ended: make(chan struct{})}
		h := discovered(t, d, map[string]int{}, nil, llmtest.Text("ok"))
		id := h.session()
		got := make(chan acp.StopReason, 1)
		go func() {
			resp, err := h.c.Client.Prompt(context.Background(), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{acp.TextBlock("hello")}})
			if err != nil {
				got <- acp.StopReason("error: " + err.Error())
				return
			}
			got <- resp.StopReason
		}()
		waitFor(t, func() bool {
			s := h.live(id)
			h.ag.mu.Lock()
			defer h.ag.mu.Unlock()
			return s.cancel != nil
		})
		if err := h.c.Client.Cancel(t.Context(), acp.CancelNotification{SessionId: id}); err != nil {
			t.Fatal(err)
		}
		select {
		case stop := <-got:
			if stop != acp.StopReasonCancelled {
				t.Errorf("stop %q, want cancelled, as a cancel ends a turn today", stop)
			}
			// 0071-PLAN Phase 2b's review: no limit computed or recorded.
			s := h.live(id)
			h.ag.mu.Lock()
			recorded := s.outLimit
			h.ag.mu.Unlock()
			if len(h.calls()) != 0 || recorded != (llm.OutputLimit{}) {
				t.Errorf("%d calls, limit recorded %+v; want none: the turn ended while it waited", len(h.calls()), recorded)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the turn outlived its cancel while it waited for discovery")
		}
	})
	t.Run("close", func(t *testing.T) {
		d := &discoverRec{block: true, ended: make(chan struct{})}
		h := discovered(t, d, map[string]int{}, nil)
		id := h.session()
		waitFor(t, func() bool { return len(d.seen()) == 1 })
		if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
			t.Fatal(err)
		}
		select {
		case <-d.ended:
		case <-time.After(5 * time.Second):
			t.Fatal("the discovery outlived its session's close")
		}
	})
}

// A long session does not discover again: one discovery at its start, none
// before its later calls, which carry what it stored (0066-MADR S9, S37).
func TestSession_NoRediscovery_0066(t *testing.T) {
	d := &discoverRec{delay: 50 * time.Millisecond}
	h := discovered(t, d, map[string]int{"fake/m1": 24000}, nil, llmtest.Text("one"), llmtest.Text("two"), llmtest.Text("three"))
	id := h.session()
	for _, p := range []string{"first", "second", "third"} {
		h.mustPrompt(id, p)
		if got := h.lastCap(); got != 24000 {
			t.Errorf("%s: the turn asked for %d, want the stored 24,000", p, got)
		}
	}
	if got := d.seen(); len(got) != 1 {
		t.Errorf("discoveries %q, want the start's alone", got)
	}
}

// Concurrent sessions each discover their own model, and each first call
// waits for its own session's discovery (0066-MADR S24, its per-session
// half; under -race in the Linux machine's make race).
func TestProvider_ConcurrentSessionsDiscovery_0066(t *testing.T) {
	d := &discoverRec{delay: 50 * time.Millisecond}
	turns := make([]llmtest.Turn, 8)
	for i := range turns {
		turns[i] = llmtest.Text("ok")
	}
	h := discovered(t, d, map[string]int{"fake/m1": 4096, "fake/m2": 2048}, nil, turns...)
	ids := make([]acp.SessionId, 8)
	for i := range ids {
		ids[i] = h.session()
		if i%2 == 1 {
			if _, err := setOption(t, h.c, ids[i], configModel, "m2"); err != nil {
				t.Fatal(err)
			}
		}
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Go(func() {
			if _, err := h.c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{acp.TextBlock("go")}}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	want := map[string]int{"m1": 4096, "m2": 2048}
	for _, r := range h.calls() {
		if r.MaxTokens != want[r.Model] {
			t.Errorf("a %s turn asked for %d, want its own discovered %d", r.Model, r.MaxTokens, want[r.Model])
		}
	}
}

// A session's first request after a load or a resume may be /compact: its
// summary waits for the discovery as a first turn does (0066-MADR S2, D1;
// 0071-PLAN Phase 2b's review).
func TestSession_CompactFirstWaits_0066(t *testing.T) {
	for _, how := range []string{"load", "resume"} {
		t.Run(how, func(t *testing.T) {
			d := &discoverRec{delay: 100 * time.Millisecond}
			h := discovered(t, d, map[string]int{"fake/m1": 24000}, nil, llmtest.Text(strings.Repeat("one ", 200)),
				llmtest.Text(strings.Repeat("two ", 200)), llmtest.Text("a summary"), llmtest.Text("a prefix summary"))
			id := h.session()
			h.mustPrompt(id, strings.Repeat("first ", 200))
			h.mustPrompt(id, strings.Repeat("second ", 200))
			if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
				t.Fatal(err)
			}
			h.setInfo("fake/m1", llm.Model{})
			var err error
			if how == "load" {
				_, err = h.c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
			} else {
				_, err = h.c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
			}
			if err != nil {
				t.Fatal(err)
			}
			text, _ := slash(t, h.c, h.client, id, "/compact")
			if s := maxTokensOf(h.summaries()); len(s) == 0 || slices.ContainsFunc(s, func(n int) bool { return n != 24000 }) {
				t.Errorf("summaries asking %v (%.120q), want each at the discovered 24,000", s, text)
			}
		})
	}
}

// Each call waits for its own model's discovery: the turn not for the
// title model's, which here never ends while the session is open; the
// title call for the title model's, in the background (0071-MADR P40, "Each
// call waits for its own model (Recommended)").
func TestEachCallWaitsForItsModel_0071(t *testing.T) {
	title := func(small *llmtest.Script) func(*Options) {
		return func(o *Options) {
			o.Title = TitleOptions{Provider: func(string, llm.Retry) (llm.Provider, error) { return small, nil }, ProviderID: "small", Model: "s1"}
		}
	}
	t.Run("turn", func(t *testing.T) {
		d := &discoverRec{delay: 50 * time.Millisecond, blockRef: "small/s1"}
		h := discovered(t, d, map[string]int{"fake/m1": 24000}, title(llmtest.NewScript(llmtest.Text("A title"))), llmtest.Text("ok"))
		id := h.session()
		done := make(chan error, 1)
		go func() {
			_, err := h.c.Client.Prompt(context.Background(), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{acp.TextBlock("hello")}})
			done <- err
		}()
		select {
		case err := <-done:
			if err != nil || h.lastCap() != 24000 {
				t.Errorf("err %v, cap %d; want the turn at its model's discovered 24,000", err, h.lastCap())
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the turn waited for the title model's discovery")
		}
	})
	t.Run("title", func(t *testing.T) {
		d := &discoverRec{delay: 100 * time.Millisecond}
		small := llmtest.NewScript(llmtest.Text("A title"))
		h := discovered(t, d, map[string]int{"fake/m1": 24000}, title(small), llmtest.Text("ok"))
		then := d.then
		d.then = func(provider, model string) {
			if provider == "small" {
				h.setInfo("small/s1", llm.Model{MaxTokens: 100, ContextWindow: 1000000})
				return
			}
			then(provider, model)
		}
		h.mustPrompt(h.session(), "hello")
		waitFor(t, func() bool { return len(small.Requests()) == 1 })
		if got := small.Requests()[0].MaxTokens; got != 100 {
			t.Errorf("the title asked for %d, want min(128, the title model's discovered 100)", got)
		}
	})
}

// S8's bound is acpserver's own: a hook that ignores its context still
// lets the call go on after the bound (0066-MADR S8; 0071-PLAN Phase 2b's
// review). The bound is shortened for the test.
func TestSession_DiscoveryBound_0066(t *testing.T) {
	old := discoverBound
	discoverBound = 100 * time.Millisecond
	t.Cleanup(func() { discoverBound = old })
	release := make(chan struct{})
	d := &discoverRec{ignore: release}
	h := discovered(t, d, map[string]int{}, nil, llmtest.Text("ok"))
	t.Cleanup(func() { close(release) }) // runs before the harness's close
	id := h.session()
	done := make(chan error, 1)
	go func() {
		_, err := h.c.Client.Prompt(context.Background(), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{acp.TextBlock("hello")}})
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the call waited past the bound for a hook that ignores its context")
	}
}

// A first turn's tools take images as its model's discovery states: the
// decision is made after the call's wait, not from the facts before it
// (0066-MADR D1, "the session's first model call waits"; 0071-PLAN Phase
// 2b's delta check).
func TestSession_ImagesAfterDiscovery_0066(t *testing.T) {
	d := &discoverRec{delay: 100 * time.Millisecond}
	h := discovered(t, d, map[string]int{}, nil, llmtest.ToolCall("r1", "read", `{"path":"pic.png"}`), llmtest.Text("ok"))
	h.setInfo("fake/m1", llm.Model{Input: []string{"text"}})
	d.then = func(provider, model string) {
		m := wide(24000)
		m.Input = []string{"text", "image"}
		h.setInfo(provider+"/"+model, m)
	}
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "pic.png"), tinyPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	id := newSession(t, h.c, cwd, nil).SessionId
	h.mustPrompt(id, "look")
	calls := h.calls()
	if len(calls) != 2 {
		t.Fatalf("%d calls, want the read and the reply", len(calls))
	}
	var images int
	var results []string
	for _, m := range calls[1].Messages {
		for _, c := range m.Content {
			switch c.Type {
			case llm.ContentImage:
				images++
			case llm.ContentToolResult:
				results = append(results, c.Text)
			}
		}
	}
	if images != 1 {
		t.Errorf("%d images sent after the read, results %q; want the picture, the discovered model taking images", images, results)
	}
}

// Agent.Close returns only after the sessions' discoveries have ended
// (0066-MADR S40; 0046-MADR D1).
func TestCloseWaitsForDiscovery_0066(t *testing.T) {
	d := &discoverRec{block: true, ended: make(chan struct{}), lag: 100 * time.Millisecond}
	h := discovered(t, d, map[string]int{}, nil)
	h.session()
	waitFor(t, func() bool { return len(d.seen()) == 1 })
	if err := h.ag.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-d.ended:
	default:
		t.Error("Agent.Close returned while a discovery still ran")
	}
}
