package provider

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// idsOfMethods are the ids of methods, in order.
func idsOfMethods(methods []authcontract.Method) []string {
	var out []string
	for _, m := range methods {
		out = append(out, m.ID)
	}
	return out
}

// Offered is the SDK's facts with gobble's policy applied: Grok's key,
// browser and device sign-ins (an accepted borrowed client); Kilo's key and
// device sign-in; OpenAI's key and ChatGPT plan, whose consent gobble
// honours, and neither its borrowed-client sign-ins nor its token paste nor
// its Codex CLI import; a key alone for the six key-only providers; none for
// Ollama (0051-MADR D2).
func TestOfferedMethods(t *testing.T) {
	for id, want := range map[string][]string{
		"grok":         {"api_key", "browser_oauth", "device_code"},
		"kilo":         {"api_key", "device_code"},
		"openai":       {"api_key", "chatgpt_plan"},
		"gemini":       {"api_key"},
		"claude":       {"api_key"},
		"opencode-zen": {"api_key"},
		"opencode-go":  {"api_key"},
		"huggingface":  {"api_key"},
		"together":     {"api_key"},
		"ollama":       nil,
	} {
		if got := idsOfMethods(Offered(id)); !slices.Equal(got, want) {
			t.Errorf("Offered(%s) = %q, want %q", id, got, want)
		}
	}
	if got := Offered("no-such-provider"); got != nil {
		t.Errorf("Offered of an unknown provider = %v, want none", got)
	}
}

// The predicates are read from what Offered says, for every key provider:
// a login is an offered method that yields an OAuth session, a key login one
// that yields a device key, a sign-in any offered method but the key, and the
// bare method the first of those in the SDK's order, as --method names it.
func TestPredicatesFromOffered(t *testing.T) {
	for _, id := range KeyProviders() {
		var login, keyLogin, revocable, piImport bool
		var first string
		for _, m := range Offered(id) {
			if m.Yields == authcontract.YieldsOAuthSession {
				login = true
				revocable = revocable || m.Revocable
				piImport = piImport || m.ClientIdentity == authcontract.ClientVendorBorrowed && id == "grok"
			}
			if m.Yields == authcontract.YieldsDeviceKey {
				keyLogin = true
			}
			if first == "" && m.ID != "api_key" {
				first = map[string]string{"browser_oauth": MethodBrowser, "device_code": MethodDevice, "chatgpt_plan": MethodChatGPTPlan}[m.ID]
			}
		}
		if HasLogin(id) != login || HasKeyLogin(id) != keyLogin || Revocable(id) != revocable || ImportsPiLogin(id) != piImport {
			t.Errorf("%s: HasLogin %v, HasKeyLogin %v, Revocable %v, ImportsPiLogin %v; Offered says %v, %v, %v, %v",
				id, HasLogin(id), HasKeyLogin(id), Revocable(id), ImportsPiLogin(id), login, keyLogin, revocable, piImport)
		}
		if got := DefaultMethod(id); got != first {
			t.Errorf("%s: DefaultMethod %q, want %q", id, got, first)
		}
		if got := HasSignIn(id); got != (first != "") {
			t.Errorf("%s: HasSignIn %v with the default method %q", id, got, first)
		}
	}
	for id, want := range map[string]string{"grok": MethodBrowser, "kilo": MethodDevice, "openai": MethodChatGPTPlan, "claude": ""} {
		if got := DefaultMethod(id); got != want {
			t.Errorf("DefaultMethod(%s) = %q, want %q", id, got, want)
		}
	}
	if !HasLogin("grok") || !HasLogin("openai") || HasLogin("kilo") || !HasKeyLogin("kilo") || HasKeyLogin("grok") {
		t.Errorf("HasLogin grok %v openai %v kilo %v, HasKeyLogin kilo %v grok %v", HasLogin("grok"), HasLogin("openai"),
			HasLogin("kilo"), HasKeyLogin("kilo"), HasKeyLogin("grok"))
	}
	if !ImportsPiLogin("grok") || ImportsPiLogin("openai") || ImportsPiLogin("kilo") {
		t.Errorf("ImportsPiLogin grok %v, openai %v, kilo %v; want true, false, false", ImportsPiLogin("grok"), ImportsPiLogin("openai"), ImportsPiLogin("kilo"))
	}
}

// Remote accepts a device sign-in, and a browser sign-in that takes a paste;
// a browser sign-in that only the host's loopback listener completes is
// refused (0051-MADR D2, the note of 2026-11-11). At go-llmprovider-sdk
// v1.31.0 Grok's browser sign-in takes a paste (its method is browser_paste,
// url_or_code), so every offered non-key method is remote; the refusal is
// held on facts of the contract's own, which no built-in method has.
func TestRemoteMethods(t *testing.T) {
	for id, want := range map[string][]string{
		"grok":   {"browser_oauth", "device_code"},
		"kilo":   {"device_code"},
		"openai": {"chatgpt_plan"},
		"claude": nil,
	} {
		var got []string
		for _, m := range Offered(id) {
			if m.ID != "api_key" && Remote(m) {
				got = append(got, m.ID)
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: Remote accepts %q, want %q", id, got, want)
		}
	}
	for _, c := range []struct {
		name string
		m    authcontract.Method
		want bool
	}{
		{"a device sign-in", authcontract.Method{Completion: authcontract.CompletionDevice}, true},
		{"a browser sign-in with a paste", authcontract.Method{Completion: authcontract.CompletionBrowserPaste, Paste: authcontract.PasteURLOrCode}, true},
		{"a loopback-only browser sign-in", authcontract.Method{Completion: authcontract.CompletionBrowserLoopback}, false},
		{"a loopback browser sign-in that takes a paste", authcontract.Method{Completion: authcontract.CompletionBrowserLoopback, Paste: authcontract.PasteCode}, true},
		{"a browser sign-in marked for a paste with none", authcontract.Method{Completion: authcontract.CompletionBrowserPaste}, false},
		{"a key", authcontract.Method{Completion: authcontract.CompletionValue}, false},
		{"a host command", authcontract.Method{Completion: authcontract.CompletionHostCommand}, false},
	} {
		if got := Remote(c.m); got != c.want {
			t.Errorf("Remote(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

// An honoured consent's text in Offered is gobble's wording, the one the
// sign-ins are given, and its link the SDK's usage page; a first plan
// sign-in's confirm step carries the same text and link (0052-MADR D2).
func TestOfferedConsentTextIsGobbles(t *testing.T) {
	var plan authcontract.Method
	for _, m := range Offered("openai") {
		if m.ID == "chatgpt_plan" {
			plan = m
		}
	}
	if len(plan.Consents) != 1 {
		t.Fatalf("openai's chatgpt_plan has consents %+v, want one", plan.Consents)
	}
	c := plan.Consents[0]
	want := ConsentText[openai.ConsentBackgroundUse]
	if c.ID != openai.ConsentBackgroundUse || c.Text != want || !c.Required {
		t.Errorf("the consent %+v, want %s required with gobble's text", c, openai.ConsentBackgroundUse)
	}
	if len(c.Links) != 1 || c.Links[0].Rel != authcontract.LinkUsage || c.Links[0].URL != openai.ChatGPTUsageURL {
		t.Errorf("the consent's links %+v, want the SDK's usage page %s", c.Links, openai.ChatGPTUsageURL)
	}
	if !strings.Contains(want, "gobble auth consent openai background deny") || !strings.HasSuffix(want, "compacting them on its own?") {
		t.Errorf("gobble's wording %q lacks the command or the question", want)
	}
	for _, m := range Offered("grok") {
		if len(m.Consents) != 0 {
			t.Errorf("grok's %s has consents %+v", m.ID, m.Consents)
		}
	}

	// A first plan sign-in's confirm step has the same text and link: the
	// wizard shows the step's text and its first link as the question's
	// title, and the test ends the sign-in at it by cancelling. Before it the
	// SDK asks where the browser is, answered "On this machine" (0068-MADR W6).
	s := &script{t: t, choose: func(title string, _ []Choice) int {
		if title == browserQuestion {
			return 0
		}
		return 2 // cancel the sign-in
	}}
	o := SignInOptions{Method: MethodChatGPTPlan, HTTPClient: &http.Client{Transport: refusingTransport{}}}
	if _, err := SignIn(t.Context(), "openai", s, o, &memLogins{}); err == nil {
		t.Fatal("a cancelled sign-in succeeded")
	}
	if want := []string{browserQuestion, c.Text + " " + c.Links[0].URL}; !slices.Equal(s.asked(), want) {
		t.Errorf("the sign-in asked %q, want the question where the browser is, then Offered's text and link %q", s.asked(), want[1])
	}
}

// refusingTransport answers every request with an error: no request leaves
// the process.
type refusingTransport struct{}

func (refusingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("the test's transport answers nothing")
}

// The policy's rules, on methods of their own (a descriptor-free check of
// kept): a consent gobble does not honour, a token paste, a vendor import, a
// host-only command, and a borrowed client for a provider not accepted keep
// a method out.
func TestUnhonouredConsentKeepsMethodOut(t *testing.T) {
	m := authcontract.Method{ID: "x", Kind: authcontract.KindBrowser, Consents: []authcontract.Consent{{ID: "other.consent", Required: true}}}
	if kept("openai", m) {
		t.Error("a method requiring a consent gobble does not honour was kept")
	}
	m.Consents[0].ID = openai.ConsentBackgroundUse
	if !kept("openai", m) {
		t.Error("a method requiring a consent gobble honours was not kept")
	}
	if kept("openai", authcontract.Method{Kind: authcontract.KindTokenPaste}) || kept("grok", authcontract.Method{Kind: authcontract.KindVendorImport}) ||
		kept("grok", authcontract.Method{Kind: authcontract.KindHostOnly}) {
		t.Error("a token paste, a vendor import or a host-only method was kept")
	}
	if kept("openai", authcontract.Method{Kind: authcontract.KindBrowser, ClientIdentity: authcontract.ClientVendorBorrowed}) ||
		!kept("grok", authcontract.Method{Kind: authcontract.KindBrowser, ClientIdentity: authcontract.ClientVendorBorrowed}) {
		t.Error("a borrowed client was kept for openai, or dropped for grok")
	}
}

// Methods of one provider that sign in to one kind of session agree on
// Revocable: Logout reads the first of them, as the SDK keeps no method id on a
// stored session, so a disagreement would make the first one decide silently
// (0051-MADR D2).
func TestSameKindMethodsAgreeOnRevocable(t *testing.T) {
	for _, id := range KeyProviders() {
		first := map[auth.SessionKind]authcontract.Method{}
		for _, m := range yields(id, authcontract.YieldsOAuthSession) {
			kind := sessionKinds[sessionKey{id, m.ClientIdentity}]
			if kind == "" {
				t.Errorf("%s: offered session method %s has no kind in sessionKinds", id, m.ID)
				continue
			}
			if f, ok := first[kind]; ok && f.Revocable != m.Revocable {
				t.Errorf("%s: %s (Revocable %v) and %s (Revocable %v) sign in to the same %s session", id, f.ID, f.Revocable, m.ID, m.Revocable, kind)
			}
			if _, ok := first[kind]; !ok {
				first[kind] = m
			}
		}
	}
}
