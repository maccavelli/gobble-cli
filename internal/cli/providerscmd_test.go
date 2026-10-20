package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// gobble providers lists the ten providers in the registry's order, each
// with its credential's source in Pi's words and how to sign in, and lists
// no model to do it (0025-MADR D5).
func TestProvidersCommand(t *testing.T) {
	home := t.TempDir()
	t.Chdir(t.TempDir())
	t.Setenv("OPENAI_API_KEY", "test-only")
	var calls atomic.Int64
	withLister(t, func(context.Context, string, string, provider.Options) (provider.Listing, error) {
		calls.Add(1)
		return provider.Listing{}, nil
	})
	r := runIn(t, home, "providers")
	if r.code != 0 {
		t.Fatalf("exit %d: %s", r.code, r.stderr)
	}
	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(r.stdout), "\n")[1:] {
		ids = append(ids, strings.Fields(line)[0])
	}
	if !slices.Equal(ids, provider.Providers()) || len(ids) != 10 {
		t.Fatalf("providers %q, want the ten %q", ids, provider.Providers())
	}
	for id, want := range map[string][]string{
		"openai": {"OpenAI", "env: OPENAI_API_KEY", "API key"},
		"claude": {"none — gobble auth login claude"},
		"grok":   {"API key; browser or device sign-in"},
		"kilo":   {"API key; device sign-in"},
		"ollama": {"runs on your machine, no key"},
	} {
		for _, w := range want {
			if !strings.Contains(rowOf(r.stdout, id), w) {
				t.Errorf("%s's row %q lacks %q", id, rowOf(r.stdout, id), w)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("%d listings without --check", calls.Load())
	}
}

// --check lists each provider with a credential, and Ollama, which takes
// none, and shows its state; --json prints the same rows as JSON (0025-MADR
// D4, D5).
func TestProvidersCheckAndJSON(t *testing.T) {
	home := t.TempDir()
	t.Chdir(t.TempDir())
	t.Setenv("OPENAI_API_KEY", "test-only")
	t.Setenv("ANTHROPIC_API_KEY", "test-only")
	withLister(t, func(_ context.Context, id, _ string, _ provider.Options) (provider.Listing, error) {
		if id == "claude" {
			return provider.Listing{}, &llm.APIError{Status: 401, Err: fmt.Errorf("%w: refused", llm.ErrAuth)}
		}
		return provider.Listing{Live: true, Source: "live", Models: []string{id + "-1"}, FetchedAt: time.Now()}, nil
	})
	r := runIn(t, home, "providers", "--check")
	if !strings.Contains(rowOf(r.stdout, "openai"), "live") || !strings.Contains(rowOf(r.stdout, "claude"), "key rejected (HTTP 401)") ||
		!strings.Contains(rowOf(r.stdout, "gemini"), "no credential") || !strings.Contains(rowOf(r.stdout, "ollama"), "live") {
		t.Fatalf("--check:\n%s", r.stdout)
	}
	r = runIn(t, home, "providers", "--check", "--json")
	var rows []struct {
		ID, Name, Credential, SignIn, State, Reason string
	}
	if err := json.Unmarshal([]byte(r.stdout), &rows); err != nil || len(rows) != 10 {
		t.Fatalf("--json: %v, %d rows:\n%s", err, len(rows), r.stdout)
	}
	if rows[2].ID != "claude" || rows[2].State != "key rejected" || rows[2].Reason != "(HTTP 401) — gobble auth login claude" || rows[1].Credential != "env: OPENAI_API_KEY" {
		t.Fatalf("--json rows %+v", rows[:3])
	}
}

// models list --provider lists one provider's models, its known ones when
// it has no credential, so marked, and Ollama's, which takes none; --all
// lists every provider's (0025-MADR D5).
func TestModelsListProviderAll(t *testing.T) {
	home := t.TempDir()
	t.Chdir(t.TempDir())
	t.Setenv("OPENAI_API_KEY", "test-only")
	withLister(t, (&liveLister{ids: []string{"gpt-listed"}}).list)
	r := runIn(t, home, "models", "list", "--provider", "claude")
	known := provider.KnownModels("claude")
	if r.code != 0 || !strings.Contains(r.stdout, known[0]) || strings.Contains(r.stdout, "gpt-listed") ||
		!strings.Contains(r.stdout, "claude has no credential: these are the models gobble knows of it") {
		t.Fatalf("--provider claude (%d):\n%s%s", r.code, r.stdout, r.stderr)
	}
	r = runIn(t, home, "models", "list", "--all")
	if !strings.Contains(r.stdout, "gpt-listed") || !strings.Contains(r.stdout, known[0]) ||
		!strings.Contains(r.stdout, "No credential, the models gobble knows: gemini, claude,") {
		t.Fatalf("--all:\n%s", r.stdout)
	}
	if r := runIn(t, home, "models", "list", "--provider", "ollama"); r.code != 0 || !strings.Contains(rowOf(r.stdout, "ollama"), "gpt-listed") {
		t.Fatalf("--provider ollama (%d):\n%s%s", r.code, r.stdout, r.stderr)
	}
	if r := runIn(t, home, "models", "list", "--provider", "nope"); r.code != ExitUsage {
		t.Fatalf("an unknown provider: %+v", r)
	}
}
