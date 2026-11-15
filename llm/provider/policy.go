package provider

import (
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
)

// acceptedBorrowed are the providers whose borrowed vendor client gobble
// accepts for a sign-in: Grok's, the client Pi signs in with
// (0005-MADR's second amendment of 2026-10-14; 0051-MADR D2).
var acceptedBorrowed = []string{string(llmprovider.ProviderGrok)}

// honouredConsents are the consents gobble shows in its own words and gates
// on, so a method that requires one of them may be offered: the ChatGPT
// plan's background use (0052-MADR D2).
var honouredConsents = []string{openai.ConsentBackgroundUse}

// sessionKey names the session a sign-in leaves: the provider and whose
// OAuth client it signed in as.
type sessionKey struct {
	provider string
	client   authcontract.ClientIdentity
}

// sessionKinds is the kind of session each OAuth sign-in of a provider
// leaves, by whose client it used. The SDK sets the kind on the session it
// saves and gives no accessor for what a method will leave, so the table is
// gobble's.
var sessionKinds = map[sessionKey]auth.SessionKind{
	{string(llmprovider.ProviderOpenAI), authcontract.ClientVendorBorrowed}: auth.SessionChatGPT,
	{string(llmprovider.ProviderOpenAI), authcontract.ClientAppRegistered}:  auth.SessionChatGPTPlan,
	{string(llmprovider.ProviderGrok), authcontract.ClientVendorBorrowed}:   auth.SessionGrok,
}

// offeredFor is Offered as yields reads it, and a test replaces it with
// methods whose facts the built-in descriptors do not have. Only yields reads
// it, so a test that swaps it changes only what yields feeds: HasLogin,
// HasKeyLogin, Revocable, ImportsPiLogin, consentMethod and storedMethod, and
// through them Accepts, Logout, PlanConsent, Consents, SetConsent and the
// adapter's planConsentOf. Offered itself, DefaultMethod, HasSignIn, Methods and
// skips read Offered directly and do not follow the swap.
var offeredFor = Offered

// Offered is provider id's sign-in methods as the SDK's facts give them
// (signin.Methods), with gobble's policy applied, in the SDK's order; none
// for a provider that takes no key (0051-MADR D2). The policy is these four
// rules and no provider id but acceptedBorrowed:
//
//   - no method gobble cannot drive or does not keep: a token paste, a
//     vendor import, a host-only command;
//   - no borrowed vendor client unless acceptedBorrowed lists the provider;
//   - no method that requires a consent honouredConsents does not list.
//
// An honoured consent's text is gobble's wording, ConsentText's, and keeps
// the SDK's links (0052-MADR D2). Every other predicate of this file is
// derived from it.
func Offered(id string) []authcontract.Method {
	d, ok := keyProvider(id)
	if !ok {
		return nil
	}
	var out []authcontract.Method
	for _, m := range signin.Methods(d) {
		if !kept(id, m) {
			continue
		}
		for i, c := range m.Consents {
			if text, ok := ConsentText[c.ID]; ok && slices.Contains(honouredConsents, c.ID) {
				m.Consents[i].Text = text
			}
		}
		out = append(out, m)
	}
	return out
}

// kept reports whether the policy keeps method m of provider id.
func kept(id string, m authcontract.Method) bool {
	switch m.Kind {
	case authcontract.KindTokenPaste, authcontract.KindVendorImport, authcontract.KindHostOnly:
		return false
	}
	if m.ClientIdentity == authcontract.ClientVendorBorrowed && !slices.Contains(acceptedBorrowed, id) {
		return false
	}
	for _, c := range m.Consents {
		if c.Required && !slices.Contains(honouredConsents, c.ID) {
			return false
		}
	}
	return true
}

// Remote reports whether method m can be completed by a person who is not on
// the agent's host: a device sign-in, or a browser sign-in that takes a paste
// (its Paste fact is set). A browser sign-in that only the host's loopback
// listener completes cannot be (0051-MADR D2, the note of 2026-11-11).
func Remote(m authcontract.Method) bool {
	switch m.Completion {
	case authcontract.CompletionDevice:
		return true
	case authcontract.CompletionBrowserPaste, authcontract.CompletionBrowserLoopback:
		return m.Paste != authcontract.PasteNone
	}
	return false
}

// yields are the offered methods of provider id that yield y.
func yields(id string, y authcontract.Yields) []authcontract.Method {
	var out []authcontract.Method
	for _, m := range offeredFor(id) {
		if m.Yields == y {
			out = append(out, m)
		}
	}
	return out
}

// HasLogin reports whether provider id has an OAuth login gobble keeps and
// refreshes: an offered method yields an OAuth session (Grok's, the ChatGPT
// plan's) (0051-MADR D2).
func HasLogin(id string) bool { return len(yields(id, authcontract.YieldsOAuthSession)) > 0 }

// HasKeyLogin reports whether provider id has a device login that gives an
// API key: an offered method yields a device key (Kilo's).
func HasKeyLogin(id string) bool { return len(yields(id, authcontract.YieldsDeviceKey)) > 0 }

// HasSignIn reports whether provider id has a sign-in other than an API key:
// an offered method that is not api_key. It is acpext's Login.
func HasSignIn(id string) bool { return DefaultMethod(id) != "" }

// Revocable reports whether a session of provider id can be revoked: an
// offered session method is revocable.
func Revocable(id string) bool {
	return slices.ContainsFunc(yields(id, authcontract.YieldsOAuthSession), func(m authcontract.Method) bool { return m.Revocable })
}

// DefaultMethod is the method a bare gobble auth login takes for provider id:
// the first offered method that is not an API key, in the SDK's order, as
// --method names it; "" when there is none (0051-MADR D2).
func DefaultMethod(id string) string {
	for _, m := range Offered(id) {
		if m.ID == string(llmprovider.AuthAPIKey) {
			continue
		}
		if w := wordOf(m.ID); w != "" {
			return w
		}
	}
	return ""
}

// ImportsPiLogin reports whether gobble imports a Pi OAuth login for
// provider id: its offered session method signs in as the vendor's own
// client, which gobble accepts, the one Pi used (0051-MADR D2).
func ImportsPiLogin(id string) bool {
	return slices.ContainsFunc(yields(id, authcontract.YieldsOAuthSession), func(m authcontract.Method) bool {
		return m.ClientIdentity == authcontract.ClientVendorBorrowed && slices.Contains(acceptedBorrowed, id)
	})
}

// Accepts reports whether stored login l is a session gobble uses for
// provider id: its kind, as the SDK's SessionKind gives it (inferred by the
// SDK's rules when l holds none), is the kind an offered method signs in to.
// A session of another kind, such as another application's ChatGPT login,
// is not gobble's (0051-MADR D2).
func Accepts(id string, l *Login) bool {
	_, ok := storedMethod(id, l)
	return ok
}

// wordOf is the --method word of the SDK's method id, or "".
func wordOf(sdkID string) string {
	for w, m := range methodIDs {
		if string(m) == sdkID {
			return w
		}
	}
	return ""
}
