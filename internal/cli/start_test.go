package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// startEnv is a runEnv in a fresh home holding settings, with the keys of
// the providers named, each listing live with one model unless refused; its
// warnings are returned.
func startEnv(t *testing.T, settings string, keys []string, refused ...string) (*runEnv, *syncBuffer) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	t.Chdir(t.TempDir())
	if settings != "" {
		writeFile(t, filepath.Join(home, "settings.yaml"), settings)
	}
	for _, id := range keys {
		t.Setenv(provider.CredentialVar(id), "test-only")
	}
	withLister(t, func(_ context.Context, id, _ string, _ provider.Options) (provider.Listing, error) {
		if slices.Contains(refused, id) {
			return provider.Listing{}, &llm.APIError{Status: 401, Err: fmt.Errorf("%w: refused", llm.ErrAuth)}
		}
		return provider.Listing{Live: true, Source: "live", Models: []string{id + "-1", id + "-2"}, FetchedAt: time.Now()}, nil
	})
	stderr := &syncBuffer{} // background listings warn into it
	out := &Output{out: &bytes.Buffer{}, err: stderr}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out, stderr: stderr}}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	e.processSettings("")
	return e, stderr
}

// A run starts on --model's provider; else the defaultProvider and
// defaultModel settings; else the newest recent model whose provider has a
// credential and is not refused; else the first provider with a key, with a
// note when there were several. A setting or recent model that cannot be
// used falls to the next, with a warning (0025-MADR D3).
func TestStartOrder(t *testing.T) {
	for _, tc := range []struct {
		name, settings  string
		keys, refused   []string
		recent          []string
		pick            string
		provider, model string
		warning, note   string
	}{
		{name: "--model", keys: []string{"openai", "claude"}, pick: "claude", provider: "claude", model: "claude-1"},
		{name: "settings", settings: "defaultProvider: claude\ndefaultModel: claude-2\n", keys: []string{"openai", "claude"},
			recent: []string{"openai/openai-2"}, provider: "claude", model: "claude-2"},
		{name: "a setting with no credential falls to recent", settings: "defaultProvider: gemini\n", keys: []string{"openai", "claude"},
			recent: []string{"claude/claude-2"}, provider: "claude", model: "claude-2", warning: "defaultProvider gemini has no credential"},
		{name: "recent", keys: []string{"openai", "claude"}, recent: []string{"together/t-1", "claude/claude-2", "openai/openai-1"},
			provider: "claude", model: "claude-2"},
		{name: "a refused recent falls to the registry", keys: []string{"openai", "claude"}, refused: []string{"claude"},
			recent: []string{"claude/claude-2"}, provider: "openai", model: "openai-1",
			warning: "claude/claude-2 is not used: key rejected", note: "Starting on OpenAI (openai), the first provider with a key"},
		{name: "the registry, several keys", keys: []string{"openai", "claude"}, provider: "openai", model: "openai-1",
			note: "Starting on OpenAI (openai), the first provider with a key; /model switches"},
		{name: "the registry, one key", keys: []string{"claude"}, provider: "claude", model: "claude-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, stderr := startEnv(t, tc.settings, tc.keys, tc.refused...)
			for _, ref := range slices.Backward(tc.recent) {
				e.rememberModel(ref)
			}
			if tc.pick != "" {
				e.pick = modelPick{Provider: tc.pick, Model: tc.pick + "-1"}
			}
			c := e.modelChoice()
			if c.Provider != tc.provider || c.Default != tc.model {
				t.Fatalf("started on %s/%s, want %s/%s", c.Provider, c.Default, tc.provider, tc.model)
			}
			if tc.warning == "" && strings.Contains(stderr.String(), "Warning:") || !strings.Contains(stderr.String(), tc.warning) {
				t.Errorf("warnings %q, want %q", stderr.String(), tc.warning)
			}
			if got := e.startNote(); tc.note == "" && got != "" || !strings.Contains(got, tc.note) {
				t.Errorf("note %q, want %q", got, tc.note)
			}
			settled(t, e) // the listings end, and keep their files, before the home is removed
		})
	}
}

// The line session says where a run started when it chose the first of
// several providers with a key, before the first prompt (0025-MADR D3).
func TestStartNoteInLineSession(t *testing.T) {
	agent := &acptest.ScriptAgent{}
	h := newLineHarness(t, agent)
	t.Setenv("OPENAI_API_KEY", "test-only")
	t.Setenv("ANTHROPIC_API_KEY", "test-only")
	h.start(Prompt{})
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Starting on OpenAI (openai)") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
}

// With no credential anywhere, a defaultProvider still names its variable,
// as before (0019-MADR D15).
func TestStartNoCredential(t *testing.T) {
	e, _ := startEnv(t, "defaultProvider: openai\n", nil)
	if _, err := modelProvider(e)("s1", llm.DefaultRetry); err == nil || !strings.Contains(err.Error(), "set OPENAI_API_KEY") ||
		!errors.Is(err, llm.ErrAuth) {
		t.Fatalf("no credential at all: %v, want defaultProvider's variable named", err)
	}
}
