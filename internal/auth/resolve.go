package auth

import (
	"fmt"
	"strings"
	"sync"

	"github.com/maccavelli/gobble-cli/llm/catalog"
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

// Credential is a provider's key and where it came from.
type Credential struct {
	Key    string
	Source Source
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
		default:
			return Credential{Key: key, Source: SourceModelsJSON}, true, nil
		}
	}
	if name := r.varOf(id); name != "" {
		if v, ok := r.LookupEnv(name); ok && v != "" {
			return Credential{Key: v, Source: Source(name)}, true, nil
		}
	}
	return Credential{}, false, nil
}

// Has is where provider id's credential would come from, without running a
// command, as Pi's check counts one configured (provider-composer.ts): the
// run's key; an auth.json api_key with a key, which owns its provider; a
// models.json key that is a command or whose variables are set; the
// provider's variable.
func (r *Resolver) Has(id string) (Source, bool) {
	if r.Runtime[id] != "" {
		return SourceFlag, true
	}
	if e, ok := r.entry(id); ok {
		return SourceFile, e.Type == TypeAPIKey && e.Key != ""
	}
	if raw := r.modelsJSONKey(id); raw != "" {
		if strings.HasPrefix(raw, "!") {
			return SourceModelsJSON, true
		}
		if _, err := catalog.ResolveValue(raw, r.LookupEnv, nil); err == nil {
			return SourceModelsJSON, true
		}
	}
	if name := r.varOf(id); name != "" {
		if v, ok := r.LookupEnv(name); ok && v != "" {
			return Source(name), true
		}
	}
	return "", false
}

// stored is an auth.json entry's credential. An api_key's key takes Pi's
// grammar, a command run once for the process, as Pi's resolveConfigValue
// caches one; an oauth entry waits for F4-3b; any other type, or an empty
// key, is none.
func (r *Resolver) stored(id string, e Entry) (Credential, bool, error) {
	switch {
	case e.Type == TypeOAuth:
		return Credential{}, false, fmt.Errorf("provider %s has an OAuth login in auth.json; gobble uses OAuth logins from 0005-PLAN F4-3b", id)
	case e.Type != TypeAPIKey || e.Key == "":
		return Credential{}, false, nil
	}
	key, err := catalog.ResolveValue(e.Key, r.LookupEnv, r.cachedRun)
	if err != nil {
		return Credential{}, false, fmt.Errorf("auth.json: provider %s: %w", id, err)
	}
	return Credential{Key: key, Source: SourceFile}, true, nil
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
