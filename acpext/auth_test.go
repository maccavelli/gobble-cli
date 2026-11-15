package acpext_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/acpext"
)

// AuthProviders is the list gobble's internal/cli builds from the provider
// registry, as recorded at 0042-PLAN P1 step 1; gobble's own test keeps the
// two equal.
func TestAuthProviders(t *testing.T) {
	want := []acpext.AuthProvider{
		{ID: "gemini", Name: "Gemini (Google)", Command: "gobble auth login gemini --api-key -"},
		{ID: "openai", Name: "OpenAI", Command: "gobble auth login openai", Login: true},
		{ID: "claude", Name: "Claude (Anthropic)", Command: "gobble auth login claude --api-key -"},
		{ID: "grok", Name: "Grok (xAI)", Command: "gobble auth login grok", Login: true},
		{ID: "opencode-zen", Name: "OpenCode Zen", Command: "gobble auth login opencode-zen --api-key -"},
		{ID: "opencode-go", Name: "OpenCode Go", Command: "gobble auth login opencode-go --api-key -"},
		{ID: "huggingface", Name: "Hugging Face", Command: "gobble auth login huggingface --api-key -"},
		{ID: "kilo", Name: "Kilo Gateway", Command: "gobble auth login kilo", Login: true},
		{ID: "together", Name: "Together AI", Command: "gobble auth login together --api-key -"},
	}
	got := acpext.AuthProviders()
	if !slices.Equal(got, want) {
		t.Fatalf("AuthProviders() = %+v,\nwant %+v", got, want)
	}
	got[0].ID = "changed"
	if acpext.AuthProviders()[0].ID == "changed" {
		t.Error("AuthProviders() returned a shared slice")
	}
}

// AuthenticateMeta is _meta.gobble on an authenticate request: its members
// are method, fresh and remote, each read strictly by both encoders. A
// member it lacks and a member of the wrong type are refused (0051-MADR D4,
// D7).
func TestAuthenticateMetaRoundTrip(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "authenticate-meta.request.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := acpext.AuthenticateMeta{Method: "chatgpt_plan", Fresh: true, Remote: true}
	for _, c := range codecs {
		var got acpext.AuthenticateMeta
		if err := c.decode(raw, &got); err != nil {
			t.Errorf("%s: decode the frame: %v", c.name, err)
			continue
		}
		if got != want {
			t.Errorf("%s: the frame decodes as %+v, want %+v", c.name, got, want)
		}
		b, err := c.encode(got)
		if err != nil {
			t.Errorf("%s: encode: %v", c.name, err)
			continue
		}
		if !bytes.Equal(canonical(t, b), canonical(t, raw)) {
			t.Errorf("%s: encodes back as %s, want %s", c.name, b, raw)
		}
		if b, err := c.encode(acpext.AuthenticateMeta{}); err != nil || string(b) != "{}" {
			t.Errorf("%s: the zero value encodes as %s (%v), want {}", c.name, b, err)
		}
		// fresh and remote false are sent as absent.
		if b, err := c.encode(acpext.AuthenticateMeta{Method: "device_code"}); err != nil ||
			string(b) != `{"method":"device_code"}` {
			t.Errorf("%s: a method alone encodes as %s (%v)", c.name, b, err)
		}
		for _, bad := range []struct{ why, in string }{
			{"an unknown member", `{"methodId":"chatgpt_plan"}`},
			{"a string fresh", `{"fresh":"true"}`},
			{"a number fresh", `{"fresh":1}`},
			{"a string remote", `{"remote":"true"}`},
			{"a number remote", `{"remote":1}`},
			{"a number method", `{"method":7}`},
		} {
			var v acpext.AuthenticateMeta
			if err := c.decode([]byte(bad.in), &v); err == nil {
				t.Errorf("%s: %s, %s, was accepted as %+v", c.name, bad.why, bad.in, v)
			}
		}
	}
}
