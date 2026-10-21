package acpserver

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
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
		{ID: "fake", Label: "Fake", State: StateLive, Models: []string{"m1", "m2"}, Source: "env: FAKE_API_KEY", Methods: "API key"},
		{ID: "other", Label: "Other", State: StateLive, Models: []string{"x9"}, Source: "auth.json: sign-in", Methods: "API key; device sign-in"},
		{ID: "dead", Label: "Dead", State: StateKeyRejected, Reason: "(HTTP 401) — gobble auth login dead", Models: []string{"d1"}},
		{ID: "none", Label: "None", State: StateNoCredential, Reason: "— set NONE_API_KEY, or gobble auth login none",
			Source: "none — gobble auth login none", Methods: "API key", Known: []string{"n1", "n2"}},
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
// tells Remember; another provider's model this agent cannot build is
// refused, the session left as it was; an unknown one is refused, and words
// that are no model filter the list keeping its numbers (0025-MADR D1, D3,
// D5).
func TestModelPick(t *testing.T) {
	r := &remembered{}
	c, client, id := choiceAgent(t, llmtest.NewScript(), r)
	for _, tc := range []struct{ cmd, want string }{
		{"/model 4", "Model: fake/m2"},
		{"/model fake/m1", "Model: fake/m1"},
		{"/model m2", "Model: fake/m2"},
		{"/model 5", "Cannot switch to other/x9: acpserver: no model provider configured"},
		{"/model other/x9", "Cannot switch to other/x9: acpserver: no model provider configured"},
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
	if text, _ := slash(t, c, client, id, "/model x9"); text != "Cannot switch to other/x9: acpserver: no model provider configured" {
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

// /model <ref> --default switches and saves the model as the default a run
// starts on; /model --default saves the session's (0025-MADR D3).
func TestModelSaveDefault(t *testing.T) {
	var saved []string
	client := &acptest.Client{Allow: true}
	ag := New(Options{Version: "1.2.3", Provider: func(string, llm.Retry) (llm.Provider, error) { return llmtest.NewScript(), nil },
		Models: func() ModelChoice { return multiModels }, Store: jsonl.New(t.TempDir(), jsonl.Options{}),
		SaveSetting: func(path, value string) error { saved = append(saved, path+"="+value); return nil }})
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	text, _ := slash(t, c, client, id, "/model fake/m2 --default")
	if !strings.HasPrefix(text, "Model: fake/m2") || !strings.Contains(text, "Saved fake/m2 as the default: new runs start on it.") {
		t.Fatalf("/model fake/m2 --default: %q", text)
	}
	if want := []string{"defaultProvider=fake", "defaultModel=m2"}; !slices.Equal(saved, want) {
		t.Fatalf("saved %q, want %q", saved, want)
	}
	saved = nil
	if text, _ := slash(t, c, client, id, "/model --default"); text != "Saved fake/m2 as the default: new runs start on it." {
		t.Fatalf("/model --default: %q", text)
	}
	if want := []string{"defaultProvider=fake", "defaultModel=m2"}; !slices.Equal(saved, want) {
		t.Fatalf("saved %q, want %q", saved, want)
	}
}

// A window is shown short: tokens below a thousand as they are.
func TestTokensShort(t *testing.T) {
	for n, want := range map[int]string{0: "", 5: "5", 999: "999", 128_000: "128K", 1_000_000: "1M", 1_100_000: "1.1M"} {
		if got := tokensShort(n); got != want {
			t.Errorf("tokensShort(%d) = %q, want %q", n, got, want)
		}
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

// A refused turn's mark holds while the provider's credential is the one it
// was set under: a sign-in, which advances the generation, lifts it
// (0025-MADR D4, D6).
func TestSignInLiftsMark(t *testing.T) {
	refused := llmtest.Turn{Err: &llm.APIError{Status: 401, Err: fmt.Errorf("%w: refused", llm.ErrAuth)}}
	var gen atomic.Uint64
	client := &acptest.Client{Allow: true}
	ag := New(Options{Version: "1.2.3", Provider: func(string, llm.Retry) (llm.Provider, error) { return llmtest.NewScript(refused), nil },
		Models: func() ModelChoice { return multiModels }, Store: jsonl.New(t.TempDir(), jsonl.Options{}),
		CredentialGeneration: func(id string) uint64 {
			if id == "fake" {
				return gen.Load()
			}
			return 0
		}})
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "hello"); err == nil {
		t.Fatal("the refused turn answered")
	}
	if text, _ := slash(t, c, client, id, "/model"); !strings.Contains(text, "**Fake** — key rejected (HTTP 401)") {
		t.Fatalf("/model after a refused key:\n%s", text)
	}
	gen.Add(1)
	if text, _ := slash(t, c, client, id, "/model"); !strings.Contains(text, "**Fake** — live") {
		t.Fatalf("/model after a sign-in:\n%s", text)
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

// /providers lists the session's providers, each with its state and
// reason, its credential's source and how to sign in (0025-MADR D5).
func TestProvidersSlash(t *testing.T) {
	c, client, id := choiceAgent(t, llmtest.NewScript(), &remembered{})
	text, _ := slash(t, c, client, id, "/providers")
	for _, want := range []string{
		"| Provider | State | Credential | Sign-in |",
		"| Fake (fake) | live | env: FAKE_API_KEY | API key |",
		"| Other (other) | live | auth.json: sign-in | API key; device sign-in |",
		"| Dead (dead) | key rejected (HTTP 401) — gobble auth login dead |",
		"| None (none) | no credential — set NONE_API_KEY, or gobble auth login none | none — gobble auth login none | API key |",
		"/models <provider> lists one provider's models.",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("/providers lacks %q:\n%s", want, text)
		}
	}
}

// /models lists one provider's models with /model's numbers, or every
// provider's, with words to match; a provider that offers none shows the
// models gobble knows of it (0025-MADR D5).
func TestModelsSlash(t *testing.T) {
	c, client, id := choiceAgent(t, llmtest.NewScript(), &remembered{})
	text, _ := slash(t, c, client, id, "/models other")
	if !strings.Contains(text, "**Other** — live") || !strings.Contains(text, "| 5 | other/x9 |") || strings.Contains(text, "fake/m1") {
		t.Errorf("/models other:\n%s", text)
	}
	text, _ = slash(t, c, client, id, "/models fake m2")
	if !strings.Contains(text, "| 4 | fake/m2 |") || strings.Contains(text, "| 3 | fake/m1 |") {
		t.Errorf("/models fake m2:\n%s", text)
	}
	text, _ = slash(t, c, client, id, "/models")
	if !strings.Contains(text, "| 3 | fake/m1 |") || !strings.Contains(text, "| 5 | other/x9 |") || strings.Contains(text, "Model: ") {
		t.Errorf("/models:\n%s", text)
	}
	text, _ = slash(t, c, client, id, "/models none")
	if !strings.Contains(text, "**None** — no credential") || !strings.Contains(text, "none/n1, none/n2") {
		t.Errorf("/models none:\n%s", text)
	}
	if text, _ := slash(t, c, client, id, "/models zz"); text != `No model matches "zz".` {
		t.Errorf("/models zz: %q", text)
	}
}
