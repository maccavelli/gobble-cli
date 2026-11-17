package provider

import (
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// SessionKindOf is the SDK's kind of a stored login: a stored Kind as it is,
// even one the SDK does not know; the SDK's inference for a login stored with
// none; "" for no login and for one the SDK infers no kind for.
func TestSessionKindOf(t *testing.T) {
	login := func(kind, issuer string) *Login {
		return &Login{Access: "a", Refresh: "r", Expires: time.Now().Add(time.Hour), Issuer: issuer, ClientID: "c", Kind: kind}
	}
	for _, c := range []struct {
		name string
		id   string
		l    *Login
		want string
	}{
		{"no login", "openai", nil, ""},
		{"a stored plan kind", "openai", login(string(auth.SessionChatGPTPlan), auth.DefaultOpenAIIssuer), "chatgpt_plan"},
		{"a stored kind the SDK does not know", "openai", login("a_future_kind", auth.DefaultOpenAIIssuer), "a_future_kind"},
		{"openai's login with no kind", "openai", login("", auth.DefaultOpenAIIssuer), "chatgpt"},
		{"grok's login with no kind", "grok", login("", auth.DefaultGrokOAuthIssuer), "grok"},
		{"a provider that has no sign-in kind", "claude", login("", "https://example.test"), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := SessionKindOf(c.id, c.l); got != c.want {
				t.Errorf("SessionKindOf(%s) = %q, want %q", c.id, got, c.want)
			}
		})
	}
}
