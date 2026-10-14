package auth

import (
	"errors"
	"strings"
	"testing"
)

// resolverFor is a Resolver over file's entries, models.json keys mj, and
// the environment env; ran counts the commands it runs.
func resolverFor(t *testing.T, file string, mj, env map[string]string) (*Resolver, *int) {
	t.Helper()
	f, err := parse([]byte(file))
	if err != nil {
		t.Fatal(err)
	}
	ran := 0
	vars := map[string]string{"openai": "OPENAI_API_KEY", "claude": "ANTHROPIC_API_KEY", "gemini": "GEMINI_API_KEY", "grok": "XAI_API_KEY"}
	return &Resolver{
		Runtime:       map[string]string{},
		File:          f,
		ModelsJSONKey: func(id string) string { return mj[id] },
		Var:           func(id string) string { return vars[id] },
		LookupEnv: func(name string) (string, bool) {
			v, ok := env[name]
			return v, ok
		},
		Run: func(command string) (string, error) {
			ran++
			if command == "fail" {
				return "", errors.New("exit status 1")
			}
			return "from-" + command + "\n", nil
		},
	}, &ran
}

// Each source wins over those after it, in Pi's order: --api-key, auth.json
// (which owns its provider), models.json, the variable (0005-PLAN F4-3a
// decisions 5 and 6).
func TestResolveOrder(t *testing.T) {
	r, ran := resolverFor(t, `{
  "openai": {"type": "api_key", "key": "stored"},
  "claude": {"type": "oauth", "access": "a", "refresh": "r", "expires": 1},
  "gemini": {"type": "api_key", "key": "!pass gemini"}
}`, map[string]string{"openai": "mj-openai", "grok": "$XAI_FROM_MJ"},
		map[string]string{"OPENAI_API_KEY": "env-openai", "ANTHROPIC_API_KEY": "env-claude", "XAI_API_KEY": "env-grok"})

	r.Runtime["openai"] = "flag"
	check := func(id string, want Credential) {
		t.Helper()
		got, ok, err := r.Credential(id)
		if err != nil || !ok || got != want {
			t.Errorf("%s: %+v, %v, %v; want %+v", id, got, ok, err, want)
		}
	}
	check("openai", Credential{Key: "flag", Source: SourceFlag})
	delete(r.Runtime, "openai")
	check("openai", Credential{Key: "stored", Source: SourceFile})

	// An oauth entry owns its provider and fails closed until F4-3b.
	if _, ok, err := r.Credential("claude"); ok || err == nil || !strings.Contains(err.Error(), "F4-3b") {
		t.Errorf("claude's oauth entry: %v, %v; want an error naming F4-3b, and no fallback to ANTHROPIC_API_KEY", ok, err)
	}

	// A stored command runs once for the process.
	check("gemini", Credential{Key: "from-pass gemini", Source: SourceFile})
	check("gemini", Credential{Key: "from-pass gemini", Source: SourceFile})
	if *ran != 1 {
		t.Errorf("the stored command ran %d times, want once", *ran)
	}

	// models.json's key, its variable unset, is none: the variable is next.
	check("grok", Credential{Key: "env-grok", Source: "XAI_API_KEY"})
	if src, ok := r.Has("grok"); !ok || src != "XAI_API_KEY" {
		t.Errorf("Has(grok) = %q, %v", src, ok)
	}
}

// models.json's key wins over the variable, its command uncached; a failed
// command is an error naming it and no value; Has runs no command.
func TestResolveModelsJSON(t *testing.T) {
	r, ran := resolverFor(t, `{}`, map[string]string{"openai": "!mj", "grok": "!fail"}, map[string]string{"OPENAI_API_KEY": "env"})
	for range 2 {
		if got, ok, err := r.Credential("openai"); err != nil || !ok || got != (Credential{Key: "from-mj", Source: SourceModelsJSON}) {
			t.Errorf("openai: %+v, %v, %v", got, ok, err)
		}
	}
	if *ran != 2 {
		t.Errorf("models.json's command ran %d times, want each time", *ran)
	}
	if _, ok, err := r.Credential("grok"); ok || err == nil || !strings.Contains(err.Error(), `"fail"`) || strings.Contains(err.Error(), "from-") {
		t.Errorf("a failed command: %v, %v; want an error naming it", ok, err)
	}
	before := *ran
	if src, ok := r.Has("openai"); !ok || src != SourceModelsJSON || *ran != before {
		t.Errorf("Has(openai) = %q, %v, ran %d; want models.json and no command run", src, ok, *ran-before)
	}
	if _, ok := r.Has("gemini"); ok {
		t.Error("Has(gemini) with no source")
	}
}
