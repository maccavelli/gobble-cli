package cli

import (
	"cmp"
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-tui-lib/v2/inline"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// AuthCmd stores, removes and checks model credentials (0005-MADR's CLI
// surface; 0005-PLAN F4-3a).
type AuthCmd struct {
	Login   AuthLoginCmd   `cmd:"" help:"Store a provider's API key in auth.json, or sign in to Grok, Kilo or your ChatGPT plan."`
	Logout  AuthLogoutCmd  `cmd:"" help:"Remove a provider's credential from auth.json."`
	Status  AuthStatusCmd  `cmd:"" help:"List where each provider's credential comes from and, for a sign-in, its kind and consents, never the credential."`
	Check   AuthCheckCmd   `cmd:"" help:"Report whether a provider has a usable credential: exit 0, 1 without one, 2 on error."`
	Consent AuthConsentCmd `cmd:"" help:"List, allow or deny a consent of a provider's stored sign-in, such as background requests under a ChatGPT plan."`
	Import  AuthImportCmd  `cmd:"" help:"Copy credentials from Pi's auth.json."`
}

// keyProvider is a provider argument as gobble's id: an SDK id or Pi's name
// for one, that takes a key.
func keyProvider(name string) (string, error) {
	id, ok := catalog.ProviderID(name)
	if !ok || provider.CredentialVar(id) == "" {
		return "", usageErrorf("unknown provider %q: one of %s", name, strings.Join(provider.KeyProviders(), ", "))
	}
	return id, nil
}

// AuthLoginCmd is gobble auth login.
type AuthLoginCmd struct {
	Provider string `arg:"" complete:"provider" help:"The provider."`
	APIKey   string `name:"api-key" complete:"none" placeholder:"-" help:"Read the key from stdin, or ask for it: -."`
	Device   bool   `name:"device" help:"Sign in with a device code, on a machine without a browser."`
	Method   string `name:"method" complete:"none" help:"How to sign in, with no menu: api-key (the key from stdin), browser, device or chatgpt-plan, as the provider offers."`
	JSON     bool   `name:"json" help:"Print the method and the sign-in's steps as JSON lines on stdout, and nothing on stderr; a command line that fails to parse is one failed step, unless --json follows the argument that failed."`
	// Workspace is the Anthropic workspace a Claude key not scoped to one
	// runs in, stored beside the key (0035-MADR D5, 0035-PLAN PQ1).
	Workspace string `name:"workspace" complete:"none" placeholder:"ID" help:"With --api-key -, the Anthropic workspace a Claude key not scoped to one runs in."`
}

// logoutLogin revokes and removes a login; a test replaces it (0005-PLAN
// F4-3b).
var logoutLogin = provider.Logout

// Run stores the key read for --api-key - as the provider's api_key entry,
// or, without --api-key, signs in to the provider (0005-PLAN F4-3b). With
// --json, its output is jsonSteps' lines alone, the contract's method and
// steps: stderr's diagnostics and log lines are dropped, the log file keeps
// the latter, and a failure is a failed step with the exit code unchanged
// (0051-MADR D5, which amends 0005-PLAN F4-3c decision 6).
func (c *AuthLoginCmd) Run(e *runEnv) error {
	if !c.JSON {
		return c.runWith(e, nil)
	}
	e.out.err = io.Discard
	e.logs.stderr = nil
	id, ierr := keyProvider(c.Provider)
	if ierr != nil {
		id = c.Provider
	}
	js := &jsonSteps{out: e.out, provider: id}
	err := c.runWith(e, js)
	if err == nil || errors.Is(err, errBrokenPipe) {
		return err
	}
	return js.fail(err)
}

// runWith is Run, showing what it shows through js when js is not nil.
func (c *AuthLoginCmd) runWith(e *runEnv, js *jsonSteps) error {
	e.ensureSettingsFile()
	id, err := keyProvider(c.Provider)
	if err != nil {
		return err
	}
	if err := checkWorkspace(id, c.Workspace); err != nil {
		return err
	}
	if c.Method != "" { // 0025-MADR D6
		if c.APIKey != "" || c.Device {
			return usageErrorf("--method is instead of --api-key and --device")
		}
		switch method, err := methodOf(id, c.Method); {
		case err != nil:
			return err
		case method == provider.MethodAPIKey:
			c.APIKey = "-"
		default:
			c.Device = method == provider.MethodDevice
		}
	}
	switch c.APIKey {
	case "":
		return e.loginWith(e.ctx, id, c.Device, js)
	case "-":
		if c.Device {
			return usageErrorf("--device is for a login; --api-key - stores a key")
		}
	default:
		return usageErrorf("%s", apiKeyLiteral)
	}
	// A key that is not given is a usage error, which is one failed step and
	// no method line (0051-PLAN P5 step 2), so the method line follows the read.
	key, err := e.readAPIKey(id)
	if err != nil {
		return err
	}
	if err := js.method(id, provider.MethodAPIKey); err != nil {
		return err
	}
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	if err := auth.SetKey(e.ctx, path, id, key, c.Workspace); err != nil {
		return failf("%v", err)
	}
	return e.saved(js, id, auth.TypeAPIKey)
}

// checkWorkspace refuses --workspace for any provider but Claude, and a
// value that cannot be a header's (0035-MADR D5).
func checkWorkspace(id, workspace string) error {
	switch {
	case workspace == "":
		return nil
	case id != "claude":
		return usageErrorf("--workspace is for claude, an Anthropic workspace")
	case strings.ContainsFunc(workspace, unicode.IsControl):
		return usageErrorf("--workspace: a workspace ID holds no control character")
	}
	return nil
}

// jsonSteps writes gobble auth login --json's lines to stdout, one JSON
// object a line (0051-MADR D5): a method line, `{"event":"method",
// "provider":...,"method":<authcontract.Method>}`, then a step line,
// `{"event":"step","provider":...,"step":<authcontract.Step>}`, for each step
// of the sign-in, the last done or failed. The steps that show something come
// from the SDK, with its seq; the last is gobble's, with the next seq, stated
// once the result is stored or the failure known. A nil *jsonSteps is text.
type jsonSteps struct {
	out      *Output
	provider string
	// seq is the last step's; yields is the method's, which done states.
	seq    int
	yields authcontract.Yields
}

type methodLine struct {
	Event    string              `json:"event"`
	Provider string              `json:"provider"`
	Method   authcontract.Method `json:"method"`
}

type stepLine struct {
	Event    string            `json:"event"`
	Provider string            `json:"provider"`
	Step     authcontract.Step `json:"step"`
}

func (j *jsonSteps) emit(v any) error {
	b, err := jsonx.Marshal(v)
	if err != nil {
		return fmt.Errorf("login step: %w", err)
	}
	return j.out.Result(append(b, '\n'))
}

// method writes the method line for provider id's offered method word. A
// nil *jsonSteps writes nothing. A word the provider does not offer is a
// usage error, which methodOf has refused already.
func (j *jsonSteps) method(id, word string) error {
	if j == nil {
		return nil
	}
	m, ok := provider.OfferedMethod(id, word)
	if !ok {
		return usageErrorf("%s offers no %s sign-in", id, word)
	}
	j.yields = m.Yields
	return j.emit(methodLine{Event: "method", Provider: j.provider, Method: m})
}

// step writes a step of the sign-in as the SDK states it.
func (j *jsonSteps) step(s authcontract.Step) error {
	j.seq = s.Seq
	return j.emit(stepLine{Event: "step", Provider: j.provider, Step: s})
}

// last writes the sign-in's last step, with the next seq.
func (j *jsonSteps) last(s authcontract.Step) error {
	j.seq++
	s.SchemaVersion, s.Seq = authcontract.SchemaVersion, j.seq
	return j.emit(stepLine{Event: "step", Provider: j.provider, Step: s})
}

// done writes the done step, with what the method yields.
func (j *jsonSteps) done() error {
	return j.last(authcontract.Step{Kind: authcontract.StepDone, Yields: j.yields})
}

// fail writes err as the failed step and returns the error that ends the
// command, with the code it exits by. The step's kind is provider.ErrorKind's
// of err, read from an error SignIn returned, which holds the SDK's errors
// wrapped in a *provider.SignInError, as 0051-MADR D6 reads it; a usage error
// is invalid_request, exit 2, and so is a sign-in refused for ErrAsks, kind
// unsupported (D5). The message is the text gobble prints without --json.
func (j *jsonSteps) fail(err error) error {
	code, kind := ExitFailure, provider.ErrorKind(err)
	if ee, ok := errors.AsType[*exitError](err); ok {
		code = ee.code
		if code == ExitUsage {
			kind = string(authcontract.FailureInvalidRequest)
		}
	}
	if errors.Is(err, provider.ErrAsks) {
		code = ExitUsage
	}
	f := authcontract.Failure{Kind: authcontract.ErrorKind(kind), Message: err.Error(), Retryable: provider.Retryable(err)}
	if werr := j.last(authcontract.Step{Kind: authcontract.StepFailed, Error: &f}); werr != nil {
		return werr
	}
	return &exitError{code: code}
}

// saved says provider id's credential, of type typ, is stored: in Pi's
// words, or the done step.
func (e *runEnv) saved(js *jsonSteps, id, typ string) error {
	switch {
	case js != nil:
		return js.done()
	case typ == auth.TypeOAuth:
		return e.out.Resultf("Logged in to %s\n", id)
	}
	return e.out.Resultf("Saved API key for %s\n", id)
}

// login is loginWith in text.
func (e *runEnv) login(id string, device bool) error { return e.loginWith(e.ctx, id, device, nil) }

// loginWith signs in to provider id with no menu (0005-PLAN F4-3b decision
// 6; 0031-MADR D7; 0051-MADR D2): by the provider's default offered method,
// the first that is not an API key, or by its device sign-in with device. A
// provider with none offered is a usage error naming --api-key -. js, when
// not nil, takes what it shows as the contract's steps, in place of the
// terminal's text and the host's browser, and a sign-in that would ask a
// question is refused before it starts (CheckNoQuestion). On a terminal, and
// never with --json, a browser sign-in also takes the URL its browser ends
// on, pasted. ctx ends the sign-in. Its own error is returned as is: Main
// prints it and exits 1, as failf would, and --json's failed step names its
// kind.
func (e *runEnv) loginWith(ctx context.Context, id string, device bool, js *jsonSteps) error {
	method := provider.DefaultMethod(id)
	switch {
	case method == "":
		return usageErrorf("%s has no login; use --api-key -", id)
	case device && !slices.Contains(provider.Methods(id), provider.MethodDevice):
		return usageErrorf("%s has no device sign-in; --device is for one", id)
	case device:
		method = provider.MethodDevice
	}
	o := provider.SignInOptions{Method: method}
	if js != nil {
		if err := js.method(id, method); err != nil {
			return err
		}
		path := authPath()
		if path == "" {
			return failf("auth.json: the config directory cannot be resolved")
		}
		if err := provider.CheckNoQuestion(ctx, id, method, auth.Logins{Path: path}); err != nil {
			return err
		}
		o.Steps, o.NoPaste = js.step, true
		return e.signInTo(ctx, unasked{out: e.out.err}, id, o, js)
	}
	o.NoPaste = !e.offersPaste()
	o.ShowDeviceCode = e.showDeviceCode
	o.OpenURL = e.openLoginURL
	var p provider.Prompter = unasked{out: e.out.err}
	if !o.NoPaste {
		p = prompter{&questions{ctx: ctx, in: inline.NewReader(e.stdin), out: e.out.err}}
	}
	return e.signInTo(ctx, p, id, o, nil)
}

// offersPaste reports whether a browser sign-in takes a pasted redirect
// URL: on a terminal. It is asked only without --json, which prints steps
// and takes no paste (0051-MADR D5).
func (e *runEnv) offersPaste() bool {
	return e.out.caps.In && e.stdin != nil
}

// showDeviceCode tells the user where to enter a device code, and until
// when: "Open <url> and enter the code <code> (expires 14:05)", the time on
// e's clock (0051-MADR D3, F56). The SDK's code always carries one; a code
// with none, as a test's may, names none.
func (e *runEnv) showDeviceCode(d provider.DeviceCode) error {
	until := ""
	if !d.Expires.IsZero() {
		until = " (expires " + d.Expires.In(e.now().Location()).Format("15:04") + ")"
	}
	if _, err := fmt.Fprintf(e.out.err, "Open %s and enter the code %s%s\n", d.URL, d.Code, until); err != nil {
		return fmt.Errorf("write stderr: %w", err)
	}
	return nil
}

// openLoginURL opens a browser sign-in's URL; the wizard has already shown
// it, so a machine whose browser does not open still has it (0005-PLAN F4-3b
// decision 6). A browser that does not open is not an error. A sign-in under
// --json never reaches it: it prints the URL as a step and opens nothing.
func (e *runEnv) openLoginURL(u string) error {
	if err := openURL(e.ctx, u); err != nil {
		e.log().Debug("open browser", slog.String("error", err.Error()))
	}
	return nil
}

// AuthLogoutCmd is gobble auth logout.
type AuthLogoutCmd struct {
	Provider string `arg:"" complete:"provider" help:"The provider."`
}

// Run removes the provider's entry, in Pi's words (interactive-mode.ts).
func (c *AuthLogoutCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	id, err := keyProvider(c.Provider)
	if err != nil {
		return err
	}
	said, err := e.logout(e.ctx, id)
	if err != nil {
		return err
	}
	if said == "" {
		said = "No stored credentials to remove. gobble auth logout only removes credentials saved by gobble auth login; " +
			"environment variables and models.json config are unchanged."
	}
	return e.out.Resultf("%s\n", said)
}

// logout removes provider id's entry from auth.json, revoking a Grok
// session first, and says what it did in Pi's words; "" when there was
// nothing to remove (0005-PLAN F4-3b decision 7; 0025-MADR D6).
func (e *runEnv) logout(ctx context.Context, id string) (string, error) {
	path := authPath()
	if path == "" {
		return "", failf("auth.json: the config directory cannot be resolved")
	}
	f, err := auth.Load(path)
	if err != nil {
		return "", failf("%v", err)
	}
	was, _ := f.Entry(id)
	if was.Type == auth.TypeOAuth && provider.HasLogin(id) {
		// Revoked, then removed whether or not revocation succeeded
		// (0005-PLAN F4-3b decision 7).
		revoke, err := logoutLogin(ctx, id, provider.LoginOptions{ClientName: appName, ClientVersion: identity().Version}, auth.Logins{Path: path})
		if err != nil {
			return "", failf("%v", err)
		}
		if revoke != nil {
			e.out.Warnf("%s's session could not be revoked (%s); it was removed from auth.json", id, provider.ErrorKind(revoke))
		}
		return "Logged out of " + id, nil
	}
	removed, err := auth.Delete(ctx, path, id)
	switch {
	case err != nil:
		return "", failf("%v", err)
	case !removed:
		return "", nil
	case was.Type == auth.TypeOAuth:
		return "Logged out of " + id, nil
	}
	return fmt.Sprintf("Removed stored API key for %s. Environment variables and models.json config are unchanged.", id), nil
}

// consentStore is the store gobble auth consent reads and writes; a test
// replaces it.
var consentStore = func(path string) provider.LoginStore { return auth.Logins{Path: path} }

// AuthConsentCmd is gobble auth consent: it lists the consents of a
// provider's stored sign-in with their answers, or stores one (0052-MADR D3).
// It asks nothing and reads no stdin, so a program may run it: exit 0 when
// the answer is stored or the list printed, 2 for a usage error (an unknown
// provider, no stored sign-in, a consent the sign-in has not, a consent with
// no allow or deny, or any other third word), 1 when auth.json cannot be
// read, parsed or locked. A result is on stdout, an error one line on stderr
// with stdout empty.
type AuthConsentCmd struct {
	Provider string `arg:"" complete:"provider" help:"The provider."`
	Consent  string `arg:"" optional:"" complete:"none" help:"A consent of the provider's stored sign-in: its id, such as chatgpt_plan.background_use, or its word, background."`
	Answer   string `arg:"" optional:"" complete:"none" help:"allow or deny. With neither a consent nor an answer, the consents are listed."`
}

// Run lists the consents, or stores the answer to one.
func (c *AuthConsentCmd) Run(e *runEnv) error {
	id, err := keyProvider(c.Provider)
	if err != nil {
		return err
	}
	var granted bool
	if c.Consent != "" {
		switch c.Answer {
		case "allow":
			granted = true
		case "deny":
		case "":
			return usageErrorf("%s %s needs allow or deny", id, c.Consent)
		default:
			return usageErrorf("%q is not allow or deny", c.Answer)
		}
	}
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	store := consentStore(path)
	answers, err := provider.Consents(e.ctx, store, id)
	if err != nil {
		return consentError(id, c.Consent, err)
	}
	if c.Consent == "" {
		var b strings.Builder
		for _, a := range answers {
			fmt.Fprintf(&b, "%s %s %s\n", id, a.ID, a.Answer)
		}
		if b.Len() == 0 {
			return nil
		}
		return e.out.Result([]byte(b.String()))
	}
	// The consent's id is found before anything is written, so that nothing
	// read afterwards can fail a stored answer.
	consent := ""
	for _, a := range answers {
		if c.Consent == a.ID || a.Word != "" && c.Consent == a.Word {
			consent = a.ID
		}
	}
	if consent == "" {
		return consentError(id, c.Consent, provider.ErrUnknownConsent)
	}
	if err := provider.SetConsent(e.ctx, store, id, consent, granted); err != nil {
		return consentError(id, c.Consent, err)
	}
	answer := provider.AnswerDeclined
	if granted {
		answer = provider.AnswerGranted
	}
	return e.out.Resultf("%s %s %s\n", id, consent, answer)
}

// consentError is err, from a consent function, as the command's error: a
// usage error for the provider having no stored sign-in or the sign-in not
// having the consent, exit 1 with the store's own words for anything else.
func consentError(id, consent string, err error) error {
	switch {
	case errors.Is(err, provider.ErrNoStoredSession):
		return usageErrorf("%s has no stored sign-in to answer a consent for", id)
	case errors.Is(err, provider.ErrUnknownConsent):
		return usageErrorf("%s's stored sign-in has no consent %q", id, consent)
	}
	return failf("%s", oneLine(err.Error()))
}

// AuthStatusCmd is gobble auth status.
type AuthStatusCmd struct {
	// JSON prints the rows as data, for a client such as magic-cli-remote
	// (0045-MADR).
	JSON bool `name:"json" help:"Print JSON: every key provider's credential source and, for a sign-in, sessionKind, consents and consentTerms, never a credential."`
}

// Run lists each key provider and where its credential comes from, never
// the credential itself.
func (c *AuthStatusCmd) Run(e *runEnv) error {
	if c.JSON {
		return c.runJSON(e)
	}
	e.ensureSettingsFile()
	rows := [][2]string{{"provider", "credential"}}
	for _, id := range provider.KeyProviders() {
		rows = append(rows, [2]string{id, e.statusRow(id).text})
	}
	width := 0
	for _, r := range rows {
		width = max(width, utf8.RuneCountInString(r[0]))
	}
	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "%-*s  %s\n", width, r[0], r[1])
	}
	return e.out.Result([]byte(b.String()))
}

// runJSON prints every key provider's row as one JSON object, or, when a
// file cannot be read, an error object and exit 2 (0045-MADR D1, D5).
// stderr and the log's stderr lines are off, as with auth login --json;
// the log file keeps them (D4).
func (c *AuthStatusCmd) runJSON(e *runEnv) error {
	e.out.err = io.Discard
	e.logs.stderr = nil
	e.ensureSettingsFile()
	var report struct {
		Providers []authStatusRow `json:"providers,omitzero"`
		Error     *statusError    `json:"error,omitzero"`
	}
	if report.Error = e.statusError(); report.Error == nil {
		report.Providers = []authStatusRow{}
		for _, id := range provider.KeyProviders() {
			report.Providers = append(report.Providers, e.statusRow(id))
		}
	}
	b, err := jsonx.Marshal(report, jsontext.WithIndent("  "))
	if err != nil {
		return failf("auth status: %v", err)
	}
	if err := e.out.Result(append(b, '\n')); err != nil {
		return err
	}
	if report.Error != nil {
		return &exitError{code: ExitUsage}
	}
	return nil
}

// statusError is auth status --json's error: a file it cannot say anything
// without. File is "auth.json", "models.json", or "" for the config
// directory; Message is gobble's error, which names the file and the fault,
// never a value (0045-MADR D5).
type statusError struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

// statusError is why auth status cannot answer, or nil.
func (e *runEnv) statusError() *statusError {
	if _, _, err := systemDirs(); err != nil {
		return &statusError{File: "", Message: err.Error()}
	}
	if err := e.authJSONError(); err != nil {
		return &statusError{File: statusAuthJSON, Message: err.Error()}
	}
	if err := e.modelsJSONError(); err != nil {
		return &statusError{File: statusModelsJSON, Message: err.Error()}
	}
	return nil
}

// authStatusRow is one key provider's row of auth status: where its credential
// comes from and whether gobble would use it, never the credential
// (0045-MADR D2). The table prints text; --json prints the rest (D3).
type authStatusRow struct {
	Provider string `json:"provider"`
	Source   string `json:"source"`
	Variable string `json:"variable,omitempty"`
	AuthType string `json:"authType,omitempty"`
	Usable   bool   `json:"usable"`
	Reason   string `json:"reason,omitempty"`
	Expires  string `json:"expires,omitempty"`
	// SessionKind is an oauth entry's kind, the SDK's word for it
	// (provider.SessionKindOf); Consents are the answers a session holds, by
	// consent id, granted or declined, and ConsentTerms every consent of the
	// method a stored session of an offered kind signs in to, in gobble's
	// wording, answered or not (0052-MADR D8).
	SessionKind  string                 `json:"sessionKind,omitempty"`
	Consents     map[string]string      `json:"consents,omitempty"`
	ConsentTerms []authcontract.Consent `json:"consentTerms,omitempty"`
	text         string
}

// The sources a row names (0045-MADR D2).
const (
	statusAuthJSON   = "auth.json"
	statusModelsJSON = "models.json"
	statusEnv        = "env"
	statusNone       = "none"
)

// statusRow is provider id's row, in Has's order: auth.json's entry, which
// owns its provider, then models.json, then its variable. Nothing is run:
// Has runs no command. No run's --api-key reaches auth status.
func (e *runEnv) statusRow(id string) authStatusRow {
	r := e.credentials()
	src, ok := r.Has(id)
	row := authStatusRow{Provider: id, Usable: ok}
	if r.File != nil {
		if entry, inFile := r.File.Entry(id); inFile {
			row.Source = statusAuthJSON
			switch entry.Type {
			case auth.TypeAPIKey:
				row.AuthType, row.text = auth.TypeAPIKey, statusAuthJSON
				if !ok {
					row.Reason = "empty_key"
				}
			case auth.TypeOAuth:
				row.AuthType, row.text = auth.TypeOAuth, "auth.json, an OAuth login"+expiry(entry.Expires)
				if entry.Expires > 0 {
					row.Expires = time.UnixMilli(entry.Expires).UTC().Format(time.RFC3339)
				}
				if !ok {
					row.Reason, row.text = "oauth_unused", row.text+", not used (0005-PLAN F4-3b)"
				}
				e.sessionFacts(&row, id)
			default:
				row.Reason, row.text = "unknown_type", fmt.Sprintf("auth.json, a %q entry gobble does not use", entry.Type)
			}
			return row
		}
	}
	switch {
	case !ok:
		row.Source, row.text = statusNone, statusNone
	case src == auth.SourceModelsJSON:
		row.Source, row.text = statusModelsJSON, string(src)
	default:
		row.Source, row.Variable, row.text = statusEnv, string(src), string(src)
	}
	return row
}

// sessionFacts adds to an oauth row what the stored session says of itself:
// its kind; for a session of the kind an offered method signs in to, each
// consent that method has with its terms, and the answers the session holds,
// which the text form puts in words. A store that cannot be read adds
// nothing: runJSON has reported it as the error object (0052-MADR D8).
func (e *runEnv) sessionFacts(row *authStatusRow, id string) {
	store := e.logins()
	if store == nil {
		return
	}
	l, err := store.Load(e.ctx, id)
	if err != nil || l == nil {
		return
	}
	row.SessionKind = provider.SessionKindOf(id, l)
	m, ok := provider.SignedInMethod(id, l)
	if !ok || len(m.Consents) == 0 {
		return
	}
	row.ConsentTerms = m.Consents
	answers, err := provider.Consents(e.ctx, store, id)
	if err != nil {
		return
	}
	var words []string
	for _, a := range answers {
		words = append(words, consentWords(a))
		switch a.Answer {
		case provider.AnswerGranted, provider.AnswerDeclined:
			if row.Consents == nil {
				row.Consents = map[string]string{}
			}
			row.Consents[a.ID] = a.Answer
		}
	}
	row.text += ", " + strings.Join(words, ", ")
}

// consentWords is a consent's answer in the status text: "background use
// allowed", "background use declined" or "background use not asked", the
// consent's name being its id after the dot.
func consentWords(a provider.ConsentAnswer) string {
	name := strings.ReplaceAll(a.ID[strings.LastIndex(a.ID, ".")+1:], "_", " ")
	switch a.Answer {
	case provider.AnswerGranted:
		return name + " allowed"
	case provider.AnswerDeclined:
		return name + " declined"
	}
	return name + " " + a.Answer
}

// credentialSource is where provider id's credential comes from, as status
// shows it: auth.json's entry, models.json, its variable, or none.
func (e *runEnv) credentialSource(id string) string { return e.statusRow(id).text }

// expiry is ", expires <time>" for an entry's milliseconds, or "".
func expiry(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return ", expires " + time.UnixMilli(ms).Local().Format("2006-01-02 15:04")
}

// AuthCheckCmd is gobble auth check, Pi's auth check (cli/auth-check.ts).
type AuthCheckCmd struct {
	Provider string `complete:"provider" help:"The provider."`
	Model    string `complete:"model" placeholder:"PROVIDER/ID" help:"A model, as --model takes it; its provider is checked."`
	JSON     bool   `name:"json" help:"Print JSON."`
}

// Pi's check statuses (cli/auth-check.ts).
const (
	checkReady    = "ready"
	checkNotReady = "not_ready"
	checkInvalid  = "invalid"
)

// checkResult is Pi's check result.
type checkResult struct {
	Status   string `json:"status"`
	Provider string `json:"provider"`
	Reason   string `json:"reason,omitempty"`
	AuthType string `json:"authType,omitempty"`
}

// Run reports ready, not_ready or invalid, and exits 0, 1 or 2, as Pi's
// check does.
func (c *AuthCheckCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	if c.Provider == "" && c.Model == "" {
		return usageErrorf("Auth checks require --provider <provider> or --model <model>")
	}
	r := e.check(c.Provider, c.Model)
	var out []byte
	if c.JSON {
		b, err := jsonx.Marshal(r, jsontext.WithIndent("  "))
		if err != nil {
			return failf("auth check: %v", err)
		}
		b = append(b, '\n')
		out = b
	} else {
		out = []byte(r.Status + "\n")
	}
	if err := e.out.Result(out); err != nil {
		return err
	}
	switch r.Status {
	case checkReady:
		return nil
	case checkNotReady:
		return &exitError{code: ExitFailure}
	}
	return &exitError{code: ExitUsage}
}

// check is the check for a provider, or for a model's provider.
func (e *runEnv) check(name, model string) checkResult {
	var id string
	if model != "" {
		res, err := catalog.ResolveModel(name, model, "", offlineModels(e.ctx, e.modelsJSON()), e.hasCredential)
		if err != nil {
			return checkResult{Status: checkNotReady, Provider: cmp.Or(name, model), Reason: "provider_not_found"}
		}
		id = res.Model.Provider
	} else if canon, ok := catalog.ProviderID(name); ok && provider.CredentialVar(canon) != "" {
		id = canon
	} else {
		return checkResult{Status: checkNotReady, Provider: name, Reason: "provider_not_found"}
	}
	c, ok, err := e.credentials().Credential(id)
	switch {
	case err != nil:
		return checkResult{Status: checkInvalid, Provider: id, Reason: "invalid_state"}
	case !ok:
		return checkResult{Status: checkNotReady, Provider: id, Reason: "credentials_not_configured"}
	case c.Login:
		return checkResult{Status: checkReady, Provider: id, AuthType: auth.TypeOAuth}
	}
	return checkResult{Status: checkReady, Provider: id, AuthType: auth.TypeAPIKey}
}
