package cli

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
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
	Login  AuthLoginCmd  `cmd:"" help:"Store a provider's API key in auth.json."`
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
}

// Run stores the key read for --api-key - as the provider's api_key entry.
func (c *AuthLoginCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	id, err := keyProvider(c.Provider)
	if err != nil {
		return err
	}
	switch c.APIKey {
	case "":
		return usageErrorf("gobble auth login takes --api-key - until OAuth logins arrive (0005-PLAN F4-3b)")
	case "-":
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
	return e.out.Resultf("Saved API key for %s\n", id)
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
				return "auth.json, an OAuth login" + expiry(entry.Expires) + ", used from 0005-PLAN F4-3b"
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
	_, ok, err := e.credentials().Credential(id)
	switch {
	case err != nil:
		return checkResult{Status: checkInvalid, Provider: id, Reason: "invalid_state"}
	case !ok:
		return checkResult{Status: checkNotReady, Provider: id, Reason: "credentials_not_configured"}
	}
	return checkResult{Status: checkReady, Provider: id, AuthType: auth.TypeAPIKey}
}
