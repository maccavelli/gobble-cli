package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
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

// A run's Sessions holds the plan session it loaded; another process then
// replaces auth.json's entry with another application's ChatGPT login. At the
// held session's next token request the refresh fails with an auth failure,
// the foreign login's token is not sent, and the session stays the plan's; a
// provider built afterwards from the same Sessions does not run on the foreign
// login. The reproduction of 0052-PLAN's "Third review" item 3
// (TestHeldSessionRefusesAForeignEntry there), adopted by the pin bump that
// brings go-llmprovider-sdk 0082-MADR (0068-MADR V8; 0052-PLAN step 1d).
func TestHeldPlanSessionRefusesAForeignEntry(t *testing.T) {
	ti := newTestIssuer(t) // in-process: every token URL below is its own
	tokenURL := ti.URL + "/oauth2/token"
	plan := &Login{Access: "plan-a", Refresh: "plan-r", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
		ClientID: "oaiapp_gobbletest", TokenURL: tokenURL, Resource: "https://api.openai.com/v1",
		Kind: string(auth.SessionChatGPTPlan)}
	store := &memLogins{logins: map[string]*Login{"openai": plan}}
	sessions := &Sessions{}
	if _, err := New("openai", Options{Logins: store, Sessions: sessions}); err != nil {
		t.Fatalf("the first build over the plan session: %v", err)
	}
	// Another process replaces the entry: another application's ChatGPT login, its access current.
	foreign := &Login{Access: "codex-a", Refresh: "codex-r", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
		ClientID: "app_codex", TokenURL: tokenURL, Kind: string(auth.SessionChatGPT)}
	if err := store.Save(context.Background(), "openai", foreign); err != nil {
		t.Fatal(err)
	}
	held, err := (Options{Logins: store, Sessions: sessions}).login(context.Background(), "openai")
	if err != nil || held == nil {
		t.Fatalf("the held session: %v, %v", held, err)
	}
	// A 401 on the plan token makes the provider invalidate it; the next Token refreshes.
	held.Invalidate()
	tok, err := held.Token(context.Background())
	if err == nil && tok.Value == "codex-a" {
		t.Errorf("the held plan session sent the foreign login's token")
	}
	if !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Errorf("the held session's token request: %+v, %v; want an auth failure", tok, err)
	}
	if held.SessionKind() != auth.SessionChatGPTPlan {
		t.Errorf("the held session is now of kind %q, want %q", held.SessionKind(), auth.SessionChatGPTPlan)
	}
	if _, err := New("openai", Options{Logins: store, Sessions: sessions}); err == nil && held.SessionKind() == auth.SessionChatGPT {
		t.Errorf("a provider was built over the held session, now another application's ChatGPT login")
	}
}
