package cli

import (
	"bufio"
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// AuthCmd stores, removes and checks model credentials (0005-MADR's CLI
// surface; 0005-PLAN F4-3a).
type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Store a provider's API key in auth.json, or sign in to Grok or Kilo."`
	Logout AuthLogoutCmd `cmd:"" help:"Remove a provider's credential from auth.json."`
	Status AuthStatusCmd `cmd:"" help:"List where each provider's credential comes from, never the credential."`
	Check  AuthCheckCmd  `cmd:"" help:"Report whether a provider has a usable credential: exit 0, 1 without one, 2 on error."`
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
	JSON     bool   `name:"json" help:"Print the login's events as JSON lines on stdout, and nothing on stderr."`
}

// The logins, which a test replaces (0005-PLAN F4-3b).
var (
	browserLogin = provider.BrowserLogin
	deviceLogin  = provider.DeviceLogin
	keyLogin     = provider.KeyLogin
	logoutLogin  = provider.Logout
)

// Run stores the key read for --api-key - as the provider's api_key entry,
// or, without --api-key, signs in to the provider (0005-PLAN F4-3b). With
// --json, its output is loginEvents' events alone: stderr's diagnostics and
// log lines are dropped, the log file keeps the latter, and a failure is an
// error event with the exit code unchanged (0005-PLAN F4-3c decision 6).
func (c *AuthLoginCmd) Run(e *runEnv) error {
	if !c.JSON {
		return c.runWith(e, nil)
	}
	e.out.err = io.Discard
	e.logs.stderr = nil
	ev := &loginEvents{out: e.out}
	err := c.runWith(e, ev)
	if err == nil || errors.Is(err, errBrokenPipe) {
		return err
	}
	id, ierr := keyProvider(c.Provider)
	if ierr != nil {
		id = c.Provider
	}
	code := ExitFailure
	if ee, ok := errors.AsType[*exitError](err); ok {
		code = ee.code
	}
	if werr := ev.emit(loginEvent{Event: "error", Provider: id, Kind: provider.FailureKind(err), Message: err.Error()}); werr != nil {
		return werr
	}
	return &exitError{code: code}
}

// runWith is Run, showing what it shows through ev when ev is not nil.
func (c *AuthLoginCmd) runWith(e *runEnv, ev *loginEvents) error {
	e.ensureSettingsFile()
	id, err := keyProvider(c.Provider)
	if err != nil {
		return err
	}
	switch c.APIKey {
	case "":
		return e.loginWith(id, c.Device, ev)
	case "-":
		if c.Device {
			return usageErrorf("--device is for a login; --api-key - stores a key")
		}
	default:
		return usageErrorf("%s", apiKeyLiteral)
	}
	key, err := e.readAPIKey(id)
	if err != nil {
		return err
	}
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	if err := auth.Set(e.ctx, path, id, auth.Entry{Type: auth.TypeAPIKey, Key: key}); err != nil {
		return failf("%v", err)
	}
	return e.saved(ev, id, auth.TypeAPIKey)
}

// loginEvents writes gobble auth login --json's events to stdout, one JSON
// object a line (0005-PLAN F4-3c decision 6). A nil *loginEvents is text.
type loginEvents struct{ out *Output }

// loginEvent is one event: device_code, browser_url, saved or error.
// Expires is RFC 3339, in UTC; Kind is provider.FailureKind's, and Message
// the error gobble prints without --json, which holds no credential.
type loginEvent struct {
	Event    string `json:"event"`
	Provider string `json:"provider"`
	URL      string `json:"url,omitempty"`
	Code     string `json:"code,omitempty"`
	Expires  string `json:"expires,omitempty"`
	Type     string `json:"type,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Message  string `json:"message,omitempty"`
}

func (l *loginEvents) emit(e loginEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("login event: %w", err)
	}
	return l.out.Result(append(b, '\n'))
}

// saved says provider id's credential, of type typ, is stored: in Pi's
// words, or the saved event.
func (e *runEnv) saved(ev *loginEvents, id, typ string) error {
	switch {
	case ev != nil:
		return ev.emit(loginEvent{Event: "saved", Provider: id, Type: typ})
	case typ == auth.TypeOAuth:
		return e.out.Resultf("Logged in to %s\n", id)
	}
	return e.out.Resultf("Saved API key for %s\n", id)
}

// login is loginWith in text.
func (e *runEnv) login(id string, device bool) error { return e.loginWith(id, device, nil) }

// loginWith signs in to provider id (0005-PLAN F4-3b decision 6): Grok's
// browser login, or its device login with device; Kilo's device login,
// whose key is stored escaped, so Pi's grammar gives it back unchanged. Any
// other provider has none. ev, when not nil, takes what it shows. A login's
// own error is returned as is: Main prints it and exits 1, as failf would,
// and --json's error event names its kind.
func (e *runEnv) loginWith(id string, device bool, ev *loginEvents) error {
	show := func(d provider.DeviceCode) error { return e.showDeviceCode(ev, id, d) }
	path := authPath()
	switch {
	case path == "":
		return failf("auth.json: the config directory cannot be resolved")
	case provider.HasKeyLogin(id):
		key, err := keyLogin(e.ctx, id, provider.LoginOptions{}, show)
		if err != nil {
			return err
		}
		if err := auth.Set(e.ctx, path, id, auth.Entry{Type: auth.TypeAPIKey, Key: catalog.QuoteValue(key)}); err != nil {
			return failf("%v", err)
		}
		return e.saved(ev, id, auth.TypeAPIKey)
	case provider.HasLogin(id):
		store := auth.Logins{Path: path}
		var err error
		if device {
			err = deviceLogin(e.ctx, id, provider.LoginOptions{}, show, store)
		} else {
			o := provider.LoginOptions{OpenURL: func(u string) error { return e.showLoginURL(ev, id, u) }}
			if e.offersPaste(ev) {
				o.InputCode = e.pastedRedirect
			}
			err = browserLogin(e.ctx, id, o, store)
		}
		if err != nil {
			return err
		}
		return e.saved(ev, id, auth.TypeOAuth)
	case id == "openai":
		return usageErrorf("openai's login waits for go-llmprovider-sdk's Sign in with ChatGPT for agents (0005-PLAN F4-3d); use --api-key -")
	}
	return usageErrorf("%s has no login; use --api-key -", id)
}

// offersPaste reports whether a browser login takes a pasted redirect URL:
// on a terminal, and never with --json (0005-PLAN F4-3c decision 6).
func (e *runEnv) offersPaste(ev *loginEvents) bool {
	return ev == nil && e.out.caps.In && e.stdin != nil
}

// showDeviceCode tells the user where to enter provider id's device code,
// or emits the device_code event.
func (e *runEnv) showDeviceCode(ev *loginEvents, id string, d provider.DeviceCode) error {
	if ev != nil {
		c := loginEvent{Event: "device_code", Provider: id, URL: d.URL, Code: d.Code}
		if !d.Expires.IsZero() {
			c.Expires = d.Expires.UTC().Format(time.RFC3339)
		}
		return ev.emit(c)
	}
	if _, err := fmt.Fprintf(e.out.err, "Open %s and enter the code %s\n", d.URL, d.Code); err != nil {
		return fmt.Errorf("write stderr: %w", err)
	}
	return nil
}

// showLoginURL prints a browser login's URL, or emits the browser_url
// event, then opens it: the SDK gives the URL nowhere else, so a machine
// whose browser does not open still shows it (0005-PLAN F4-3b decision 6).
// A browser that does not open is not an error.
func (e *runEnv) showLoginURL(ev *loginEvents, id, u string) error {
	if ev != nil {
		if err := ev.emit(loginEvent{Event: "browser_url", Provider: id, URL: u}); err != nil {
			return err
		}
	} else {
		msg := "Open this URL to sign in to %s:\n  %s\n"
		if e.offersPaste(ev) {
			msg += "If the browser cannot reach this machine, paste the URL it ends on here.\n"
		}
		if _, err := fmt.Fprintf(e.out.err, msg, id, u); err != nil {
			return fmt.Errorf("write stderr: %w", err)
		}
	}
	if err := openURL(e.ctx, u); err != nil {
		e.log().Debug("open browser", slog.String("error", err.Error()))
	}
	return nil
}

// pastedRedirect is the redirect URL the user pastes on a terminal, read
// as one line; ctx ending gives up waiting for it.
func (e *runEnv) pastedRedirect(ctx context.Context) (string, error) {
	line := make(chan string, 1)
	go func() {
		s, _ := bufio.NewReader(e.stdin).ReadString('\n') //nolint:errcheck // an unread line is no paste
		line <- strings.TrimSpace(s)
	}()
	select {
	case s := <-line:
		if s == "" {
			<-ctx.Done() // nothing pasted: the browser's callback may still arrive
			return "", ctx.Err()
		}
		return s, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
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
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	f, err := auth.Load(path)
	if err != nil {
		return failf("%v", err)
	}
	was, _ := f.Entry(id)
	if was.Type == auth.TypeOAuth && provider.HasLogin(id) {
		// Revoked, then removed whether or not revocation succeeded
		// (0005-PLAN F4-3b decision 7).
		revoke, err := logoutLogin(e.ctx, id, provider.LoginOptions{}, auth.Logins{Path: path})
		if err != nil {
			return failf("%v", err)
		}
		if revoke != nil {
			e.out.Warnf("%s's session could not be revoked (%s); it was removed from auth.json", id, provider.FailureKind(revoke))
		}
		return e.out.Resultf("Logged out of %s\n", id)
	}
	removed, err := auth.Delete(e.ctx, path, id)
	switch {
	case err != nil:
		return failf("%v", err)
	case !removed:
		return e.out.Resultf("No stored credentials to remove. gobble auth logout only removes credentials saved by gobble auth login; " +
			"environment variables and models.json config are unchanged.\n")
	case was.Type == auth.TypeOAuth:
		return e.out.Resultf("Logged out of %s\n", id)
	}
	return e.out.Resultf("Removed stored API key for %s. Environment variables and models.json config are unchanged.\n", id)
}

// AuthStatusCmd is gobble auth status.
type AuthStatusCmd struct{}

// Run lists each key provider and where its credential comes from, never
// the credential itself.
func (c *AuthStatusCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	rows := [][2]string{{"provider", "credential"}}
	for _, id := range provider.KeyProviders() {
		rows = append(rows, [2]string{id, e.credentialSource(id)})
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

// credentialSource is where provider id's credential comes from, as status
// shows it: auth.json's entry, models.json, its variable, or none.
func (e *runEnv) credentialSource(id string) string {
	r := e.credentials()
	if r.File != nil {
		if entry, ok := r.File.Entry(id); ok {
			switch entry.Type {
			case auth.TypeAPIKey:
				return "auth.json"
			case auth.TypeOAuth:
				if provider.HasLogin(id) {
					return "auth.json, an OAuth login" + expiry(entry.Expires)
				}
				return "auth.json, an OAuth login" + expiry(entry.Expires) + ", not used (0005-PLAN F4-3b)"
			}
			return fmt.Sprintf("auth.json, a %q entry gobble does not use", entry.Type)
		}
	}
	if src, ok := r.Has(id); ok {
		return string(src)
	}
	return "none"
}

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
		b, err := json.Marshal(r, jsontext.WithIndent("  "))
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
