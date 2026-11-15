package provider

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"

	"github.com/maccavelli/gobble-cli/llm"
)

// brokenLogins is a LoginStore whose reads fail once it is broken, as an
// auth.json that no longer parses does.
type brokenLogins struct {
	*memLogins
	broken atomic.Bool
}

func (b *brokenLogins) Load(ctx context.Context, id string) (*Login, error) {
	if b.broken.Load() {
		return nil, errors.New("auth.json: the file does not parse")
	}
	return b.memLogins.Load(ctx, id)
}

// A request marked as background on a ChatGPT plan session whose user has
// not granted the consent is refused before the network, as llm.ErrNotPermitted
// carrying the consent's id: when no answer is stored, when it is declined,
// and when the grant stored can no longer be read, which the SDK reads from
// the store at each request and not from the session it was built with. With
// the grant, an unmarked request, or another credential, the request is
// sent (0052-MADR D2).
func TestMarkReachesTheSDK(t *testing.T) {
	plan := func(consents string) *Login {
		return &Login{Access: "a", Refresh: "r", Expires: time.Now().Add(time.Hour), Issuer: auth.DefaultOpenAIIssuer, ClientID: "oaiapp_gobbletest",
			Resource: "https://api.openai.com/v1", HostID: "urn:uuid:host", Kind: string(auth.SessionChatGPTPlan), Consents: consents}
	}
	for _, c := range []struct {
		name     string
		consents string
		marked   bool
		breakIt  bool
		refused  bool
	}{
		{"no answer", "", true, false, true},
		{"declined", testConsentID + "=declined", true, false, true},
		{"granted", testConsentID + "=granted", true, false, false},
		{"granted, then the store unreadable", testConsentID + "=granted", true, true, true},
		{"no answer, an unmarked request", "", false, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			store := &brokenLogins{memLogins: &memLogins{}}
			if err := store.Save(t.Context(), "openai", plan(c.consents)); err != nil {
				t.Fatal(err)
			}
			wire := &cannedWire{status: http.StatusBadRequest, body: `{"error":{"message":"bad request"}}`}
			p, err := New("openai", Options{Model: "gpt-6-luna", Logins: store, HTTPClient: &http.Client{Transport: wire}, Retry: &llm.Retry{Off: true}})
			if err != nil {
				t.Fatal(err)
			}
			if c.breakIt {
				store.broken.Store(true)
			}
			ctx := t.Context()
			if c.marked {
				ctx = llm.WithBackground(ctx)
			}
			req := wireRequest()
			req.Model = "gpt-6-luna"
			var streamErr error
			for _, err := range p.Stream(ctx, req) {
				if err != nil {
					streamErr = err
					break
				}
			}
			paths, _ := wire.requests()
			ce, isConsent := errors.AsType[*llm.ConsentError](streamErr)
			switch {
			case c.refused:
				if !errors.Is(streamErr, llm.ErrNotPermitted) || !isConsent || ce.Consent != testConsentID || len(paths) != 0 {
					t.Errorf("err %v (consent error %v), %d requests sent; want llm.ErrNotPermitted carrying %s and nothing sent", streamErr, isConsent, len(paths), testConsentID)
				}
			default:
				if errors.Is(streamErr, llm.ErrNotPermitted) || isConsent || len(paths) == 0 {
					t.Errorf("err %v, %d requests sent; want the request sent and no consent refusal", streamErr, len(paths))
				}
			}
		})
	}
	// A key, or a session of another kind, is never refused for a consent.
	wire := &cannedWire{status: http.StatusBadRequest, body: `{"error":{"message":"bad request"}}`}
	p, err := New("openai", Options{APIKey: "k", Model: "gpt-6-luna", HTTPClient: &http.Client{Transport: wire}, Retry: &llm.Retry{Off: true}})
	if err != nil {
		t.Fatal(err)
	}
	req := wireRequest()
	req.Model = "gpt-6-luna"
	for _, err := range p.Stream(llm.WithBackground(t.Context()), req) {
		if errors.Is(err, llm.ErrNotPermitted) {
			t.Errorf("a marked request with an API key was refused: %v", err)
		}
	}
	if paths, _ := wire.requests(); len(paths) == 0 {
		t.Error("a marked request with an API key was not sent")
	}
}

// testConsentID is the consent the ChatGPT plan's background requests need,
// as the SDK names it.
const testConsentID = "chatgpt_plan.background_use"
