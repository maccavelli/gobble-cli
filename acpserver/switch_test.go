package acpserver

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// switchAgent is an agent over multiModels in store dir: fake's model is
// scripts["fake"], and ProviderFor builds the others from scripts, or fails
// for an id it has none of. other/x9's window is five tokens.
func switchAgent(t *testing.T, dir string, scripts map[string]*llmtest.Script, choice ModelChoice, r *remembered, logs *bytes.Buffer) (*acptest.Conn, *acptest.Client) {
	t.Helper()
	client := &acptest.Client{Allow: true}
	ag := New(Options{Version: "1.2.3",
		Provider: func(string, llm.Retry) (llm.Provider, error) { return scripts["fake"], nil },
		ProviderFor: func(id, _ string, _ llm.Retry) (llm.Provider, error) {
			if s, ok := scripts[id]; ok {
				return s, nil
			}
			return nil, errors.New("no provider " + id)
		},
		ModelInfo: func(p, m string) llm.Model {
			if p == "other" && m == "x9" {
				return llm.Model{Provider: p, ID: m, ContextWindow: 5}
			}
			return llm.Model{Provider: p, ID: m}
		},
		Models: func() ModelChoice { return choice }, Remember: r.add, Store: jsonl.New(dir, jsonl.Options{}),
		Logger: slog.New(slog.NewTextHandler(logs, nil))})
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c, client
}

// A session switched to another provider's model, by /model or the option,
// sends its next turn there, with the earlier turn's origin, records the
// switch, and remembers it (0025-MADR D1, D2).
func TestSwitchProvider(t *testing.T) {
	for _, by := range []string{"command", "option"} {
		t.Run(by, func(t *testing.T) {
			a, b := llmtest.NewScript(llmtest.Text("from a")), llmtest.NewScript(llmtest.Text("from b"))
			r := &remembered{}
			dir := t.TempDir()
			c, client := switchAgent(t, dir, map[string]*llmtest.Script{"fake": a, "other": b}, multiModels, r, &bytes.Buffer{})
			id := newSession(t, c, t.TempDir(), nil).SessionId
			if _, err := prompt(t, c, id, "first"); err != nil {
				t.Fatal(err)
			}
			if by == "command" {
				if text, _ := slash(t, c, client, id, "/model other/x9"); !strings.HasPrefix(text, "Model: other/x9") {
					t.Fatalf("/model other/x9: %q", text)
				}
			} else if resp, err := c.Client.SetSessionConfigOption(t.Context(), acp.SetSessionConfigOptionRequest{ValueId: &acp.SetSessionConfigOptionValueId{
				SessionId: id, ConfigId: "model", Value: "other/x9"}}); err != nil || resp.ConfigOptions[0].Select.CurrentValue != "other/x9" {
				t.Fatalf("the option: %v %+v", err, resp)
			}
			if _, err := prompt(t, c, id, "second"); err != nil {
				t.Fatal(err)
			}
			if len(a.Requests()) != 1 || len(b.Requests()) != 1 {
				t.Fatalf("requests a %d, b %d; want one each", len(a.Requests()), len(b.Requests()))
			}
			req := b.Requests()[0]
			if req.Model != "x9" {
				t.Fatalf("b's request names %q", req.Model)
			}
			i := slices.IndexFunc(req.Messages, func(m llm.Message) bool { return m.Role == llm.RoleAssistant })
			if i < 0 || req.Messages[i].Origin != (llm.Origin{Provider: "fake", Model: "m1"}) {
				var got []string
				for _, m := range req.Messages {
					got = append(got, fmt.Sprintf("%s %+v", m.Role, m.Origin))
				}
				t.Fatalf("b's history %q, want a's answer with its origin", got)
			}
			if !slices.Contains(r.got(), "other/x9") {
				t.Fatalf("remembered %q", r.got())
			}
			if got := entryTypes(t, c, id); !slices.Contains(got, "model_change") {
				t.Fatalf("entries %q, want a model_change", got)
			}
		})
	}
}

// /model with a provider lands on its last model, the newest recent one
// (0025-MADR D1, D3).
func TestSwitchToProviderLandsOnItsLast(t *testing.T) {
	scripts := map[string]*llmtest.Script{"fake": llmtest.NewScript(), "other": llmtest.NewScript()}
	c, client := switchAgent(t, t.TempDir(), scripts, multiModels, &remembered{}, &bytes.Buffer{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if text, _ := slash(t, c, client, id, "/model other"); !strings.HasPrefix(text, "Model: other/x9") {
		t.Fatalf("/model other: %q", text)
	}
}

// A switch to a model whose window is smaller than the context in use says
// so and suggests /compact (0025-MADR D1).
func TestSwitchWarnsOnSmallerWindow(t *testing.T) {
	scripts := map[string]*llmtest.Script{"fake": llmtest.NewScript(llmtest.Text("a long enough answer")), "other": llmtest.NewScript()}
	c, client := switchAgent(t, t.TempDir(), scripts, multiModels, &remembered{}, &bytes.Buffer{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "first"); err != nil {
		t.Fatal(err)
	}
	text, _ := slash(t, c, client, id, "/model other/x9")
	if !strings.Contains(text, "other/x9's window is 5 tokens") || !strings.Contains(text, "/compact") {
		t.Fatalf("/model other/x9 after a turn: %q", text)
	}
}

// A provider that cannot be built leaves the session where it was.
func TestSwitchBuildFails(t *testing.T) {
	scripts := map[string]*llmtest.Script{"fake": llmtest.NewScript()}
	c, client := switchAgent(t, t.TempDir(), scripts, multiModels, &remembered{}, &bytes.Buffer{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if text, _ := slash(t, c, client, id, "/model other/x9"); text != "Cannot switch to other/x9: no provider other" {
		t.Fatalf("/model other/x9: %q", text)
	}
	if text, _ := slash(t, c, client, id, "/model"); !strings.HasPrefix(text, "Model: fake/m1") {
		t.Fatalf("after the failed switch: %q", text)
	}
}

// A session restored from its log goes on with the provider it switched to;
// when that provider has no credential now, with the run's, and a warning
// (0025-MADR D1).
func TestRestoreSwitchedSession(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	b := llmtest.NewScript(llmtest.Text("from b"), llmtest.Text("again"))
	c1, client1 := switchAgent(t, dir, map[string]*llmtest.Script{"fake": llmtest.NewScript(), "other": b}, multiModels, &remembered{}, &bytes.Buffer{})
	id := newSession(t, c1, cwd, nil).SessionId
	slash(t, c1, client1, id, "/model other/x9")
	if _, err := prompt(t, c1, id, "first"); err != nil {
		t.Fatal(err)
	}
	if _, err := c1.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}

	c2, _ := switchAgent(t, dir, map[string]*llmtest.Script{"fake": llmtest.NewScript(), "other": b}, multiModels, &remembered{}, &bytes.Buffer{})
	resp, err := c2.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if got := currentModel(t, resp.ConfigOptions); got != "other/x9" {
		t.Fatalf("restored on %q, want other/x9", got)
	}
	if _, err := prompt(t, c2, id, "second"); err != nil || len(b.Requests()) != 2 {
		t.Fatalf("the restored turn: %v, other's requests %d", err, len(b.Requests()))
	}
	if _, err := c2.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}

	noOther := multiModels
	noOther.Providers = slices.Clone(multiModels.Providers)
	noOther.Providers[1] = ProviderChoice{ID: "other", Label: "Other", State: StateNoCredential, Reason: "— set OTHER_API_KEY"}
	var logs bytes.Buffer
	c3, _ := switchAgent(t, dir, map[string]*llmtest.Script{"fake": llmtest.NewScript()}, noOther, &remembered{}, &logs)
	resp, err = c3.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if got := currentModel(t, resp.ConfigOptions); got != "fake/m1" || !strings.Contains(logs.String(), "the session's provider is not usable") || !strings.Contains(logs.String(), "provider=other") {
		t.Fatalf("restored on %q with logs %q, want fake/m1 and a warning", got, logs.String())
	}
}
