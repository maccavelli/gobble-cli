package acpserver

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// multiModels is a choice across four providers: the session's, another
// live one, one whose key was refused, and one with no credential, with two
// recent models (0025-MADR D1, D3, D4).
var multiModels = ModelChoice{Provider: "fake", Models: []string{"m1", "m2"}, Default: "m1",
	Recent: []string{"fake/m2", "other/x9"},
	Providers: []ProviderChoice{
		{ID: "fake", Label: "Fake", State: StateLive, Models: []string{"m1", "m2"}},
		{ID: "other", Label: "Other", State: StateLive, Models: []string{"x9"}},
		{ID: "dead", Label: "Dead", State: StateKeyRejected, Reason: "(HTTP 401) — gobble auth login dead", Models: []string{"d1"}},
		{ID: "none", Label: "None", State: StateNoCredential, Reason: "— set NONE_API_KEY, or gobble auth login none"},
	}}

// remembered records what Options.Remember is told.
type remembered struct {
	mu   sync.Mutex
	refs []string
}

func (r *remembered) add(ref string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refs = append(r.refs, ref)
}

func (r *remembered) got() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.refs)
}

// choiceAgent is an agent over multiModels on model p, remembering into r.
func choiceAgent(t *testing.T, p llm.Provider, r *remembered) (*acptest.Conn, *acptest.Client, acp.SessionId) {
	t.Helper()
	client := &acptest.Client{Allow: true}
	ag := New(Options{Version: "1.2.3", Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Models: func() ModelChoice { return multiModels }, Remember: r.add, Store: jsonl.New(t.TempDir(), jsonl.Options{})})
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c, client, newSession(t, c, t.TempDir(), nil).SessionId
}

// /model lists the current model, the recent ones, then each provider
// under its state, numbered across the list; a provider that is not usable
// shows its state and reason and no model (0025-MADR D4, D5; deviation 2).
func TestModelPicker(t *testing.T) {
	c, client, id := choiceAgent(t, llmtest.NewScript(), &remembered{})
	text, _ := slash(t, c, client, id, "/model")
	for _, want := range []string{
		"Model: fake/m1",
		"| 1 | fake/m2 |", "| 2 | other/x9 |",
		"**Fake** — live", "| 3 | fake/m1 |", "| 4 | fake/m2 |",
		"**Other** — live", "| 5 | other/x9 |",
		"**Dead** — key rejected (HTTP 401) — gobble auth login dead",
		"**None** — no credential — set NONE_API_KEY, or gobble auth login none",
		"/model N",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("/model lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "dead/d1") {
		t.Errorf("a dead provider's model is offered:\n%s", text)
	}
}

// /model takes an entry's number, a ref, or a bare id one provider has, and
// tells Remember; a model of another provider waits for P2, an unknown one
// is refused, and words that are no model filter the list keeping its
// numbers (0025-MADR D1, D3, D5).
func TestModelPick(t *testing.T) {
	r := &remembered{}
	c, client, id := choiceAgent(t, llmtest.NewScript(), r)
	for _, tc := range []struct{ cmd, want string }{
		{"/model 4", "Model: fake/m2"},
		{"/model fake/m1", "Model: fake/m1"},
		{"/model m2", "Model: fake/m2"},
		{"/model 5", "Switching to another provider's model is not available yet; this session's provider is fake."},
		{"/model other/x9", "Switching to another provider's model is not available yet; this session's provider is fake."},
		{"/model 99", "No entry 99: /model lists 5."},
		{"/model dead/d1", "dead is not usable: key rejected (HTTP 401) — gobble auth login dead"},
	} {
		if text, _ := slash(t, c, client, id, tc.cmd); text != tc.want {
			t.Errorf("%s: %q, want %q", tc.cmd, text, tc.want)
		}
	}
	if got, want := r.got(), []string{"fake/m2", "fake/m1", "fake/m2"}; !slices.Equal(got, want) {
		t.Errorf("remembered %q, want %q", got, want)
	}
	if text, _ := slash(t, c, client, id, "/model x9"); text != "Switching to another provider's model is not available yet; this session's provider is fake." {
		t.Errorf("/model x9, an id one provider offers: %q", text)
	}
	text, _ := slash(t, c, client, id, "/model m")
	for _, want := range []string{"| 1 | fake/m2 |", "| 3 | fake/m1 |", "| 4 | fake/m2 |"} {
		if !strings.Contains(text, want) {
			t.Errorf("/model m lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "other/x9") || strings.Contains(text, "**Dead**") {
		t.Errorf("/model m lists more than its matches:\n%s", text)
	}
	if text, _ := slash(t, c, client, id, "/model zz"); text != `Unknown model "zz". Available models: fake/m2, other/x9, fake/m1.` {
		t.Errorf("/model zz: %q", text)
	}
}

// The model option is grouped, recent first, then each usable provider,
// its values refs; a bare id of the session's provider is still taken
// (0025-MADR D1, D5).
func TestModelOptionGrouped(t *testing.T) {
	r := &remembered{}
	c, _, id := choiceAgent(t, llmtest.NewScript(), r)
	resp, err := c.Client.SetSessionConfigOption(t.Context(), acp.SetSessionConfigOptionRequest{ValueId: &acp.SetSessionConfigOptionValueId{
		SessionId: id, ConfigId: "model", Value: "fake/m2"}})
	if err != nil {
		t.Fatal(err)
	}
	sel := resp.ConfigOptions[0].Select
	if sel.CurrentValue != "fake/m2" || sel.Options.Grouped == nil {
		t.Fatalf("model option %+v, want grouped with fake/m2 current", sel)
	}
	var groups []string
	var values []string
	for _, g := range *sel.Options.Grouped {
		groups = append(groups, g.Name)
		for _, o := range g.Options {
			values = append(values, g.Name+":"+string(o.Value))
		}
	}
	if want := []string{"Recent", "Fake", "Other"}; !slices.Equal(groups, want) {
		t.Fatalf("groups %q, want %q", groups, want)
	}
	if want := []string{"Recent:fake/m2", "Recent:other/x9", "Fake:fake/m1", "Fake:fake/m2", "Other:other/x9"}; !slices.Equal(values, want) {
		t.Fatalf("values %q, want %q", values, want)
	}
	if _, err := c.Client.SetSessionConfigOption(t.Context(), acp.SetSessionConfigOptionRequest{ValueId: &acp.SetSessionConfigOptionValueId{
		SessionId: id, ConfigId: "model", Value: "m1"}}); err != nil {
		t.Fatalf("a bare id: %v", err)
	}
	if got := r.got(); !slices.Equal(got, []string{"fake/m2", "fake/m1"}) {
		t.Fatalf("remembered %q", got)
	}
}

// StateOf names each failure's state in gobble's words, with the status
// when there is one and never the error's text; an unclassified failure
// still says the listing failed (0025-MADR D4, deviation 1).
func TestStateOf(t *testing.T) {
	for _, tc := range []struct {
		err    error
		state  ProviderState
		reason string
	}{
		{nil, StateLive, ""},
		{&llm.APIError{Status: 429, Err: fmt.Errorf("%w: sk-pl***nt", llm.ErrQuota)}, StateQuotaExhausted, "(HTTP 429)"},
		{&llm.APIError{Status: 401, Err: fmt.Errorf("%w: sk-pl***nt", llm.ErrAuth)}, StateKeyRejected, "(HTTP 401) — gobble auth login p"},
		{fmt.Errorf("%w", llm.ErrAuth), StateKeyRejected, "— gobble auth login p"},
		{&llm.APIError{Status: 403, Err: llm.ErrNotPermitted}, StateNotPermitted, "(HTTP 403)"},
		{fmt.Errorf("x: %w", llm.ErrOffline), StateNotListed, "(offline)"},
		{fmt.Errorf("x: %w", context.DeadlineExceeded), StateNotListed, "(timed out)"},
		{&llm.APIError{Status: 503, Err: llm.ErrUnavailable}, StateNotListed, "(HTTP 503)"},
		{errors.New("sk-pl***nt went wrong"), StateNotListed, "(listing failed)"},
	} {
		state, reason := StateOf("p", tc.err)
		if state != tc.state || reason != tc.reason || strings.Contains(reason, "sk-") {
			t.Errorf("StateOf(%v) = %s %q, want %s %q", tc.err, state, reason, tc.state, tc.reason)
		}
	}
}

// A turn refused for the key marks its provider, with the status, until
// the process ends (0025-MADR D4, deviation 1).
func TestRefusedKeyMarksProvider(t *testing.T) {
	refused := llmtest.Turn{Err: &llm.APIError{Status: 401, Err: fmt.Errorf("%w: Incorrect API key provided: sk-pl***nt", llm.ErrAuth)}}
	c, client, id := choiceAgent(t, llmtest.NewScript(refused), &remembered{})
	if _, err := prompt(t, c, id, "hello"); err == nil || !errors.Is(err, err) {
		t.Fatalf("the refused turn answered %v", err)
	}
	text, _ := slash(t, c, client, id, "/model")
	if !strings.Contains(text, "**Fake** — key rejected (HTTP 401)") || strings.Contains(text, "sk-pl") {
		t.Fatalf("/model after a refused key:\n%s", text)
	}
}
