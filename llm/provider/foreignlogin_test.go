package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// The provider build refuses a stored session of a kind no offered method signs
// in to, whoever builds it, in the words the resolver gives for it: another
// application's ChatGPT login, kept by kind and by none; and builds from a
// ChatGPT plan session. The resolver is the first gate, this the second, so a
// stale resolver cannot hand a foreign session to the build (0052-PLAN P4, the
// owner's ruling of 2026-11-16, relayed by the coordinator).
func TestNewRefusesAForeignLogin(t *testing.T) {
	const resolverRefusal = "provider openai has an OAuth login in auth.json, which gobble does not use (0005-PLAN F4-3b)"
	foreign := func(kind string) *Login {
		return &Login{Access: "a", Refresh: "r", Expires: time.Now().Add(time.Hour), Issuer: auth.DefaultOpenAIIssuer,
			ClientID: "app_codex", TokenURL: auth.DefaultOpenAIIssuer + "/oauth/token", Kind: kind}
	}
	plan := foreign(string(auth.SessionChatGPTPlan))
	plan.ClientID, plan.Resource = "oaiapp_gobbletest", "https://api.openai.com/v1"
	for name, c := range map[string]struct {
		login *Login
		ok    bool
	}{
		"another application's ChatGPT login": {foreign(string(auth.SessionChatGPT)), false},
		"a login stored with no kind":         {foreign(""), false},
		"a plan session":                      {plan, true},
	} {
		t.Run(name, func(t *testing.T) {
			store := &memLogins{logins: map[string]*Login{"openai": c.login}}
			_, err := New("openai", Options{Logins: store})
			switch {
			case c.ok && err != nil:
				t.Errorf("a session the policy accepts was refused: %v", err)
			case !c.ok && (err == nil || err.Error() != resolverRefusal):
				t.Errorf("the build gave %v, want exactly the resolver's refusal of a foreign login, %q (the provider named once)", err, resolverRefusal)
			}
		})
	}
}

// failingLoad is a login store whose Load fails with a fixed error.
type failingLoad struct {
	memLogins
	err error
}

func (f *failingLoad) Load(context.Context, string) (*Login, error) { return nil, f.err }

// An error of the store that is not the refusal of a foreign login is still
// the build's own failure: New names the provider once, with its prefix, and
// keeps the store's error in the chain (the refusal alone is returned as it is,
// since it names the provider already).
func TestNewWrapsAStoreError(t *testing.T) {
	boom := errors.New("auth.json is unreadable")
	_, err := New("openai", Options{Logins: &failingLoad{err: boom}})
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("the build gave %v, want an error that wraps the store's, %v", err, boom)
	}
	if want := "provider openai: "; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("the build's error is %q, want the prefix %q", err, want)
	}
}
