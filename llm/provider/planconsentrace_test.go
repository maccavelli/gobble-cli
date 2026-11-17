package provider

import (
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

// planConsentOf reads a shared session's kind under the session's own lock, while a refresh of
// the same session, as a run's Sessions shares one between its providers, rewrites its fields
// under it. Run under -race (make race): reading the fields unlocked is a data race.
func TestPlanConsentOfReadsTheKindUnderTheSessionLock(t *testing.T) {
	ti := newTestIssuer(t)
	u, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: toIssuer{u}}
	l := &Login{Access: "a", Refresh: "plan-r", Expires: time.Now().Add(-time.Hour), Issuer: ti.URL, ClientID: "oaiapp_gobbletest",
		TokenURL: ti.URL + "/oauth2/token", Resource: "https://api.openai.com/v1", Kind: "chatgpt_plan"}
	store := &memLogins{}
	if err := store.Save(t.Context(), "openai", l); err != nil {
		t.Fatal(err)
	}
	s := l.session("openai", store, client, nil)
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 20 {
			s.Invalidate()
			_, _ = s.Token(t.Context())
		}
	})
	for range 2000 {
		if planConsentOf("openai", s) == "" {
			t.Error("a plan session needs no consent")
			break
		}
	}
	wg.Wait()
}
