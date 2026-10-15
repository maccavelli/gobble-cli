package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// AuthImportCmd is gobble auth import --from-pi (0005-PLAN F4-3c decision 7).
type AuthImportCmd struct {
	FromPi   bool     `name:"from-pi" required:"" help:"Copy from Pi's auth.json, in $PI_CODING_AGENT_DIR or ~/.pi/agent."`
	Provider []string `name:"provider" complete:"provider" placeholder:"PROVIDER" help:"A provider to copy without a terminal, by gobble's or Pi's name; repeatable."`
	Replace  bool     `name:"replace" help:"Without a terminal, replace an entry auth.json already has."`
}

// piAuthPath is Pi's auth.json: in $PI_CODING_AGENT_DIR, a leading ~
// expanded, or in ~/.pi/agent (Pi's config.ts).
func piAuthPath() (string, error) {
	if dir := os.Getenv("PI_CODING_AGENT_DIR"); dir != "" {
		return filepath.Join(expandHome(dir), "auth.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pi", "agent", "auth.json"), nil
}

// piEntry is what gobble makes of Pi's entry piID: the provider it is
// copied to, or why it is skipped.
func piEntry(piID string, e auth.Entry) (id, skip string) {
	id, ok := catalog.ProviderID(piID)
	switch {
	case !ok || provider.CredentialVar(id) == "":
		return "", "gobble has no provider " + piID
	case e.Type == auth.TypeOAuth && provider.HasLogin(id):
		return id, ""
	case e.Type == auth.TypeOAuth:
		return "", "an OAuth login gobble does not use"
	case e.Type == auth.TypeAPIKey && e.Key != "":
		return id, ""
	}
	return "", "no key"
}

// answerYes is confirm's yes in full; y is its short form.
const answerYes = "yes"

// confirm asks question on stderr and reads the answer, a line, from
// stdin: y or yes, in any case, is yes; anything else, or none, is no. A
// test replaces it.
var confirm = func(e *runEnv, question string) (bool, error) {
	if _, err := fmt.Fprintf(e.out.err, "%s [y/N] ", question); err != nil {
		return false, fmt.Errorf("write stderr: %w", err)
	}
	line, err := readLine(e.stdin)
	if err != nil {
		return false, err
	}
	a := strings.ToLower(strings.TrimSpace(line))
	return a == "y" || a == answerYes, nil
}

// readLine reads one line from r a byte at a time, so nothing after it is
// taken from the next question; end of file ends it.
func readLine(r io.Reader) (string, error) {
	var b []byte
	one := make([]byte, 1)
	for len(b) < 4096 {
		n, err := r.Read(one)
		if n == 1 {
			if one[0] == '\n' {
				break
			}
			b = append(b, one[0])
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
	}
	return string(b), nil
}

// Run copies Pi's credentials into gobble's auth.json, one entry at a time
// in Pi's order, reporting each. Pi's file is read once, with no lock, and
// never written. On a terminal each copy, and each replacement, is asked;
// otherwise --provider and --replace decide (0005-PLAN F4-3c decision 7).
func (c *AuthImportCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	named := map[string]bool{}
	for _, name := range c.Provider {
		id, err := keyProvider(name)
		if err != nil {
			return err
		}
		named[id] = true
	}
	from, err := piAuthPath()
	if err != nil {
		return failf("Pi's auth.json: %v", err)
	}
	if _, err := os.Stat(from); errors.Is(err, fs.ErrNotExist) {
		return failf("Pi has no auth.json at %s", from)
	}
	pi, err := auth.Load(from)
	if err != nil {
		return failf("%v", err)
	}
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	mine, err := auth.Load(path)
	if err != nil {
		return failf("%v", err)
	}
	has := map[string]bool{}
	for _, id := range mine.Providers() {
		has[id] = true
	}
	ask := e.out.caps.In && e.out.caps.Err.TTY && e.stdin != nil
	for _, piID := range pi.Providers() {
		entry, _ := pi.Entry(piID)
		id, skip := piEntry(piID, entry)
		if skip == "" {
			if skip, err = c.choose(e, ask, named, has, piID, id); err != nil {
				return err
			}
		}
		if skip != "" {
			if err := e.out.Resultf("Skipped %s: %s\n", piID, skip); err != nil {
				return err
			}
			continue
		}
		if err := importEntry(e, path, id, entry); err != nil {
			return failf("%v", err)
		}
		has[id] = true
		if err := e.out.Resultf("Imported %s from Pi's %s\n", id, piID); err != nil {
			return err
		}
	}
	return nil
}

// choose is why Pi's piID is not copied to id, or "" to copy it: asked on
// a terminal, else whether --provider names it and, when auth.json has id
// already, --replace.
func (c *AuthImportCmd) choose(e *runEnv, ask bool, named, has map[string]bool, piID, id string) (string, error) {
	if !ask || len(named) > 0 {
		if !named[id] {
			return "not named by --provider", nil
		}
	}
	if ask {
		ok, err := confirm(e, fmt.Sprintf("Import Pi's %s as %s?", piID, id))
		switch {
		case err != nil:
			return "", err
		case !ok:
			return "not chosen", nil
		case !has[id]:
			return "", nil
		}
		ok, err = confirm(e, fmt.Sprintf("auth.json already has %s; replace it?", id))
		switch {
		case err != nil:
			return "", err
		case !ok:
			return "auth.json already has " + id, nil
		}
		return "", nil
	}
	if has[id] && !c.Replace {
		return "auth.json already has " + id + "; --replace replaces it", nil
	}
	return "", nil
}

// importEntry stores Pi's entry as id's: a key as written, or Grok's login
// with the SDK's issuer and client id.
func importEntry(e *runEnv, path, id string, entry auth.Entry) error {
	if entry.Type == auth.TypeAPIKey {
		return auth.Set(e.ctx, path, id, auth.Entry{Type: auth.TypeAPIKey, Key: entry.Key})
	}
	var exp time.Time
	if entry.Expires > 0 {
		exp = time.UnixMilli(entry.Expires)
	}
	login, ok := provider.PiLogin(id, entry.Access, entry.Refresh, exp)
	if !ok {
		return fmt.Errorf("provider %s has no login gobble imports", id)
	}
	return auth.Logins{Path: path}.Save(e.ctx, id, login)
}
