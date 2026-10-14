package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	xterm "golang.org/x/term"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// authPath is <config>/auth.json, or "" when the config directory cannot be
// resolved (0003-MADR; 0005-PLAN F4-3a).
func authPath() string {
	dirs, _, err := systemDirs()
	if err != nil {
		return ""
	}
	return filepath.Join(dirs.Config, "auth.json")
}

// credentials is the run's credential resolver, built once: --api-key's key
// (set by setRuntimeKey), auth.json, models.json's keys and the providers'
// variables, in Pi's order. A file that cannot be read is not used, with
// one warning (0005-PLAN F4-3a).
func (e *runEnv) credentials() *auth.Resolver {
	e.credsOnce.Do(func() {
		mj := e.modelsJSON()
		r := &auth.Resolver{
			Runtime:       map[string]string{},
			ModelsJSONKey: mj.APIKey,
			Var:           provider.CredentialVar,
			LookupEnv:     os.LookupEnv,
			Run:           func(command string) (string, error) { return e.runCommand(e.ctx, command) },
		}
		if path := authPath(); path != "" {
			f, err := auth.Load(path)
			if err != nil {
				e.log().Warn("auth.json", slog.String("message", err.Error()+"; the file is not used"))
			} else {
				r.File = f
			}
		}
		e.creds = r
	})
	return e.creds
}

// setRuntimeKey is --api-key's key for provider id, for this run alone.
func (e *runEnv) setRuntimeKey(id, key string) { e.credentials().Runtime[id] = key }

// credential is provider id's credential, resolved now. A resolution error
// is warned once per provider, and is no credential.
func (e *runEnv) credential(id string) (auth.Credential, bool) {
	c, ok, err := e.credentials().Credential(id)
	if err != nil {
		e.credsMu.Lock()
		defer e.credsMu.Unlock()
		if !e.credWarned[id] {
			if e.credWarned == nil {
				e.credWarned = map[string]bool{}
			}
			e.credWarned[id] = true
			e.log().Warn("credentials", slog.String("message", err.Error()))
		}
		return auth.Credential{}, false
	}
	return c, ok
}

// knownKey is provider id's key for what only informs, such as a model's
// facts and the model listing: resolved once per provider, so a command is
// not run for every lookup.
func (e *runEnv) knownKey(id string) string {
	e.credsMu.Lock()
	if k, ok := e.knownKeys[id]; ok {
		e.credsMu.Unlock()
		return k
	}
	e.credsMu.Unlock()
	c, _ := e.credential(id)
	e.credsMu.Lock()
	defer e.credsMu.Unlock()
	if e.knownKeys == nil {
		e.knownKeys = map[string]string{}
	}
	e.knownKeys[id] = c.Key
	return c.Key
}

// credentialKey is the provider's key for building it, resolved now, as
// provider.AmbientWith takes one.
func (e *runEnv) credentialKey(id string) (string, bool) {
	c, ok := e.credential(id)
	return c.Key, ok
}

// hasCredential reports whether provider id has a credential, without
// running a command (auth.Resolver.Has).
func (e *runEnv) hasCredential(id string) bool {
	_, ok := e.credentials().Has(id)
	return ok
}

// readAPIKey is --api-key -'s key for provider id: on a terminal it asks,
// without echo; otherwise it is stdin, all of it, trimmed (0005-PLAN F4-3a
// decision 7).
func (e *runEnv) readAPIKey(id string) (string, error) {
	if e.stdin == nil {
		return "", usageErrorf("--api-key -: there is no stdin to read the key from")
	}
	var key string
	if e.out.caps.In {
		if _, err := fmt.Fprintf(e.out.err, "API key for %s: ", id); err != nil {
			return "", fmt.Errorf("write stderr: %w", err)
		}
		b, err := xterm.ReadPassword(int(e.stdin.Fd())) //nolint:gosec // G115: a terminal's descriptor fits an int
		if _, werr := fmt.Fprintln(e.out.err); werr != nil {
			return "", fmt.Errorf("write stderr: %w", werr)
		}
		if err != nil {
			return "", failf("--api-key -: %v", err)
		}
		key = string(b)
	} else {
		b, err := io.ReadAll(io.LimitReader(e.stdin, 64<<10))
		if err != nil {
			return "", failf("--api-key -: %v", err)
		}
		key = string(b)
	}
	if key = strings.TrimSpace(key); key == "" {
		return "", usageErrorf("--api-key -: no key was given")
	}
	return key, nil
}

// authMessage is what the CLI says when the agent answers auth_required:
// that the run's provider rejected its key, naming where the key came from,
// and nothing of the provider's text, which may hold part of the key; that
// the key could not be resolved, naming the command or variable; or, with
// no key, noCredentialMessage (0005-PLAN F4-3a deviations 1 and 3).
func (e *runEnv) authMessage() string {
	sel, err := provider.ChooseWith(e.hasCredential, e.chosenProvider())
	if err != nil {
		return e.noCredentialMessage()
	}
	src, _ := e.credentials().Has(sel.ID)
	_, ok, err := e.credentials().Credential(sel.ID)
	switch {
	case err != nil:
		return fmt.Sprintf("%s's API key from %s could not be resolved: %v", sel.ID, src, err)
	case ok:
		return sel.ID + " rejected the API key from " + string(src)
	}
	return e.noCredentialMessage()
}
