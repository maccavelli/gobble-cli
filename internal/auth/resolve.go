package auth

import (
	"fmt"
	"strings"
	"sync"

	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// Source is where a credential came from: one of the constants, or the
// name of the environment variable it was read from.
type Source string

// The sources other than a variable.
const (
	SourceFlag       Source = "--api-key"
	SourceFile       Source = "auth.json"
	SourceModelsJSON Source = "models.json"
)

// Credential is a provider's key and where it came from; or, with Login, a
// stored OAuth login, which the provider loads itself (0005-PLAN F4-3b).
// Workspace is the Anthropic workspace an auth.json api_key runs in; a
// key from anywhere else carries none (0035-MADR D5).
type Credential struct {
	Key       string
	Source    Source
	Login     bool
	Workspace string
}

// AcceptsLogin is Resolver.Accepts as gobble sets it: provider.Accepts for
// entry e's session. Its kind is read from the entry's own members, by the
// SDK's rules (0051-MADR D2); provider cannot take an Entry, which this
// package's Logins already makes it import.
func AcceptsLogin(id string, e Entry) bool {
	return provider.Accepts(id, &provider.Login{Kind: e.Kind, Issuer: e.Issuer, Resource: e.Resource})
}

// Resolver finds a provider's credential in Pi's order (ai/src/auth/
// resolve.ts): the run's --api-key, then auth.json's entry, which owns its
// provider, then models.json's apiKey, then the provider's variable
// (0005-PLAN F4-3a decision 5).
type Resolver struct {
	// Runtime are --api-key's keys, by provider, for the run only.
	Runtime map[string]string
	// File is auth.json; nil has no entries.
	File *File
	// ModelsJSONKey is models.json's apiKey for a provider, as written.
	ModelsJSONKey func(id string) string
	// Var is a provider's credential variable, or "".
	Var func(id string) string
	// LookupEnv reads the environment.
	LookupEnv func(string) (string, bool)
	// Run runs a value's !command and returns its output.
	Run func(string) (string, error)
	// Accepts reports whether provider id uses oauth entry e as its login:
	// its session is of a kind an offered sign-in method yields; nil uses
	// none (0005-PLAN F4-3b; 0051-MADR D2).
	Accepts func(id string, e Entry) bool

	mu       sync.Mutex
	commands map[string]commandResult // a stored key's !command, run once
}

type commandResult struct {
	out string
	err error
}

// Credential is provider id's credential, whether there is one, or an error
// naming the provider and the variable or command, never a value. An
// auth.json entry owns its provider: with one, no later source is read.
func (r *Resolver) Credential(id string) (Credential, bool, error) {
	if k := r.Runtime[id]; k != "" {
		return Credential{Key: k, Source: SourceFlag}, true, nil
	}
	if e, ok := r.entry(id); ok {
		return r.stored(id, e)
	}
	if raw := r.modelsJSONKey(id); raw != "" {
		key, err := catalog.ResolveValue(raw, r.LookupEnv, r.Run)
		switch {
		case err != nil && !strings.HasPrefix(raw, "!"):
			// Its variables are unset: no key, as Pi counts one (provider-composer.ts).
		case err != nil:
			return Credential{}, false, fmt.Errorf("models.json: provider %s: apiKey: %w", id, err)
		case blank(key):
			// Only white space: no key, as an empty one (0068-MADR Q2).
		default:
			return Credential{Key: key, Source: SourceModelsJSON}, true, nil
		}
	}
	if name := r.varOf(id); name != "" {
		if v, ok := r.LookupEnv(name); ok && !blank(v) {
			return Credential{Key: v, Source: Source(name)}, true, nil
		}
	}
	return Credential{}, false, nil
}

// blank reports whether a key is empty once trimmed: only spaces, tabs, line
// breaks or a non-breaking space. Such a key is no key, at every source, as an
// empty one is (0068-MADR Q2, "Treat it as no key (Recommended)"); the SDK
// would store it as empty from v1.32.0 (go-llmprovider-sdk 0085-MADR D9).
func blank(key string) bool { return strings.TrimSpace(key) == "" }

// storedKeyBlank reports whether an auth.json key is no key: blank as
// written, or a value with no command whose variables are set and resolve to
// a blank one. A command is not run, and one whose variables are unset still
// counts, as before, for Credential to report.
func (r *Resolver) storedKeyBlank(raw string) bool {
	if blank(raw) {
		return true
	}
	if strings.HasPrefix(raw, "!") {
		return false
	}
	v, err := catalog.ResolveValue(raw, r.LookupEnv, nil)
	return err == nil && blank(v)
}

// Has is where provider id's credential would come from, without running a
// command, as Pi's check counts one configured (provider-composer.ts): the
// run's key; an auth.json api_key with a key, or the oauth login of a
// provider that has one, which owns its provider; a
// models.json key that is a command or whose variables are set; the
// provider's variable.
func (r *Resolver) Has(id string) (Source, bool) {
	if r.Runtime[id] != "" {
		return SourceFlag, true
	}
	if e, ok := r.entry(id); ok {
		return SourceFile, (e.Type == TypeAPIKey && !r.storedKeyBlank(e.Key)) || (e.Type == TypeOAuth && r.usesLogin(id, e))
	}
	if raw := r.modelsJSONKey(id); raw != "" {
		if strings.HasPrefix(raw, "!") {
			return SourceModelsJSON, true
		}
		if v, err := catalog.ResolveValue(raw, r.LookupEnv, nil); err == nil && !blank(v) {
			return SourceModelsJSON, true
		}
	}
	if name := r.varOf(id); name != "" {
		if v, ok := r.LookupEnv(name); ok && !blank(v) {
			return Source(name), true
		}
	}
	return "", false
}

// stored is an auth.json entry's credential. An api_key's key takes Pi's
// grammar, a command run once for the process, as Pi's resolveConfigValue
// caches one; an oauth entry is the login of a provider that has one, and an
// error for any other; any other type, or an empty key, is none.
func (r *Resolver) stored(id string, e Entry) (Credential, bool, error) {
	switch {
	case e.Type == TypeOAuth && r.usesLogin(id, e):
		return Credential{Source: SourceFile, Login: true}, true, nil
	case e.Type == TypeOAuth:
		return Credential{}, false, fmt.Errorf("provider %s has an OAuth login in auth.json, which gobble does not use (0005-PLAN F4-3b)", id)
	case e.Type != TypeAPIKey || blank(e.Key):
		return Credential{}, false, nil
	}
	key, err := catalog.ResolveValue(e.Key, r.LookupEnv, r.cachedRun)
	if err != nil {
		return Credential{}, false, fmt.Errorf("auth.json: provider %s: %w", id, err)
	}
	if blank(key) {
		return Credential{}, false, nil // only white space: no key (0068-MADR Q2)
	}
	return Credential{Key: key, Source: SourceFile, Workspace: e.Workspace}, true, nil
}

func (r *Resolver) cachedRun(command string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.commands[command]; ok {
		return c.out, c.err
	}
	out, err := r.Run(command)
	if r.commands == nil {
		r.commands = map[string]commandResult{}
	}
	r.commands[command] = commandResult{out: out, err: err}
	return out, err
}

func (r *Resolver) usesLogin(id string, e Entry) bool { return r.Accepts != nil && r.Accepts(id, e) }

func (r *Resolver) entry(id string) (Entry, bool) {
	if r.File == nil {
		return Entry{}, false
	}
	return r.File.Entry(id)
}

func (r *Resolver) modelsJSONKey(id string) string {
	if r.ModelsJSONKey == nil {
		return ""
	}
	return r.ModelsJSONKey(id)
}

func (r *Resolver) varOf(id string) string {
	if r.Var == nil {
		return ""
	}
	return r.Var(id)
}
