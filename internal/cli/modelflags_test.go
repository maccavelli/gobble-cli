package cli

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// settingsEnv is modelEnv with --set's overrides applied before the
// process settings are first read.
func settingsEnv(t *testing.T, sets ...string) *runEnv {
	t.Helper()
	t.Setenv("GOBBLE_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out, stderr: &syncBuffer{}}}
	for _, s := range sets {
		o, err := config.ParseOverride(s)
		if err != nil {
			t.Fatal(err)
		}
		e.overrides = append(e.overrides, o)
	}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	return e
}

// stderrOf is what e wrote to its error stream.
func stderrOf(e *runEnv) string { return e.out.err.(*bytes.Buffer).String() }

// --model resolves by Pi's rules: the pick reaches the agent's override and
// the model choice, a custom id is warned of, --thinking wins over a
// suffix, and Pi's errors are usage errors (0005-PLAN F4-2c).
func TestPickModel(t *testing.T) {
	e, _ := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	withLister(t, (&liveLister{ids: []string{"gpt-6-astra"}}).list)
	pick, err := e.pickModel(&SharedFlags{Model: "openai/gpt-zz:low"})
	if err != nil || pick != (modelPick{Provider: "openai", Model: "gpt-zz", Thinking: "low"}) {
		t.Fatalf("pick %+v, %v; want openai/gpt-zz at low", pick, err)
	}
	if !strings.Contains(stderrOf(e), `Warning: Model "gpt-zz" not found for provider "openai". Using custom model id.`) {
		t.Errorf("stderr %q lacks Pi's custom-id warning", stderrOf(e))
	}
	e.pick = pick
	if got := e.agentOptions().Override; got != (acpserver.Override{Provider: "openai", Model: "gpt-zz", Thinking: "low"}) {
		t.Errorf("override %+v, want the pick", got)
	}
	if c := e.modelChoice(); c.Provider != "openai" || !slices.Equal(c.Models, []string{"gpt-6-astra", "gpt-zz"}) {
		t.Errorf("model choice %+v, want the listing and the pick", c)
	}
	if pick, err := e.pickModel(&SharedFlags{Model: "openai/gpt-zz:low", Thinking: "high"}); err != nil || pick.Model != "gpt-zz:low" || pick.Thinking != "high" {
		t.Errorf("with --thinking: %+v, %v; want the suffix kept in the id and high", pick, err)
	}
	_, err = e.pickModel(&SharedFlags{Model: "zzz-none"})
	if ee, ok := errors.AsType[*exitError](err); !ok || ee.code != ExitUsage || ee.msg != `Model "zzz-none" not found. Use --list-models to see available models.` {
		t.Errorf("not found: %v", err)
	}
	_, err = e.pickModel(&SharedFlags{Provider: "nope", Model: "x"})
	if ee, ok := errors.AsType[*exitError](err); !ok || ee.code != ExitUsage || ee.msg != `Unknown provider "nope". Use --list-models to see available providers/models.` {
		t.Errorf("an unknown provider: %v", err)
	}
	if pick, err := e.pickModel(&SharedFlags{Thinking: "max"}); err != nil || pick != (modelPick{Thinking: "max"}) {
		t.Errorf("--thinking alone: %+v, %v", pick, err)
	}
}

// The credential message names --model's provider for a pick, else the
// defaultProvider setting's, else every variable.
func TestNoCredentialMessagePick(t *testing.T) {
	e, _ := modelEnv(t)
	if got := e.noCredentialMessage(); !strings.HasPrefix(got, "no model credential: set one of ") {
		t.Errorf("no pick: %q", got)
	}
	e.pick = modelPick{Provider: "claude", Model: "x"}
	if got, want := e.noCredentialMessage(), "no model credential: --model is claude/x; set "+provider.CredentialVar("claude"); got != want {
		t.Errorf("pick: %q, want %q", got, want)
	}
	d := settingsEnv(t, "defaultProvider=grok")
	if got, want := d.noCredentialMessage(), "no model credential: defaultProvider is grok; set "+provider.CredentialVar("grok"); got != want {
		t.Errorf("defaultProvider: %q, want %q", got, want)
	}
}

// defaultModel names a new session's model with defaultProvider, and is
// ignored, warned once, without it or naming a model gobble does not know.
func TestDefaultModel(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-only")
	listed := []string{"gpt-6-astra", "gpt-6-luna"}
	known := provider.KnownModels("openai")
	if len(known) == 0 {
		t.Fatal("openai has no known models")
	}
	t.Run("used", func(t *testing.T) {
		e := settingsEnv(t, "defaultProvider=openai", "defaultModel="+known[len(known)-1])
		withLister(t, (&liveLister{ids: listed}).list)
		if c := e.modelChoice(); c.Default != known[len(known)-1] || !slices.Contains(c.Models, c.Default) {
			t.Errorf("model choice %+v, want defaultModel the default, offered", c)
		}
	})
	t.Run("without defaultProvider", func(t *testing.T) {
		e := settingsEnv(t, "defaultModel=gpt-6-luna")
		withLister(t, (&liveLister{ids: listed}).list)
		e.modelChoice()
		if c := e.modelChoice(); c.Default != "gpt-6-astra" ||
			strings.Count(stderrOf(e), "defaultModel gpt-6-luna is ignored: it needs defaultProvider") != 1 {
			t.Errorf("model choice %+v, stderr %q; want the listing's default and one warning", c, stderrOf(e))
		}
	})
	t.Run("unknown", func(t *testing.T) {
		e := settingsEnv(t, "defaultProvider=openai", "defaultModel=no-such-model")
		withLister(t, (&liveLister{ids: listed}).list)
		e.modelChoice()
		if c := e.modelChoice(); c.Default != "gpt-6-astra" ||
			strings.Count(stderrOf(e), "defaultModel no-such-model is not a model of openai that gobble knows") != 1 {
			t.Errorf("model choice %+v, stderr %q; want the listing's default and one warning", c, stderrOf(e))
		}
	})
	t.Run("--model wins", func(t *testing.T) {
		e := settingsEnv(t, "defaultProvider=openai", "defaultModel=no-such-model")
		withLister(t, (&liveLister{ids: listed}).list)
		e.pick = modelPick{Provider: "openai", Model: "gpt-6-luna"}
		if c := e.modelChoice(); c.Default != "gpt-6-astra" || stderrOf(e) != "" {
			t.Errorf("model choice %+v, stderr %q; want defaultModel unread", c, stderrOf(e))
		}
	})
}

// defaultThinkingLevel's values are --thinking's.
func TestThinkingLevelsMatchTheSetting(t *testing.T) {
	k, ok := config.KeyOf("defaultThinkingLevel")
	if !ok || !slices.Equal(k.Enum, thinkingLevels) {
		t.Fatalf("defaultThinkingLevel's enum %q, want thinkingLevels %q", k.Enum, thinkingLevels)
	}
}
