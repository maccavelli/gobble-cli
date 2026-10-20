package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// ProvidersCmd is gobble providers (0025-MADR D5).
type ProvidersCmd struct {
	Check bool `help:"List each provider's models, to show whether its key works."`
	JSON  bool `name:"json" help:"Print JSON."`
}

// providerRow is one provider as gobble providers lists it.
type providerRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Credential string `json:"credential"`
	SignIn     string `json:"signIn"`
	State      string `json:"state,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// Run lists every provider, in the registry's order: its name, where its
// credential comes from in Pi's words, how to sign in, and with --check the
// state its listing shows, never the credential (0025-MADR D4, D5).
func (c *ProvidersCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	ids := provider.Providers()
	if c.Check { // every listing starts before the first is read
		for _, id := range ids {
			if e.listable(id) {
				e.modelListing(provider.Selection{ID: id}).start()
			}
		}
	}
	rows := make([]providerRow, len(ids))
	for i, id := range ids {
		rows[i] = providerRow{ID: id, Name: provider.Label(id), Credential: e.credentialWords(id), SignIn: signInWords(id)}
		if c.Check {
			rows[i].State, rows[i].Reason = e.checkedState(id)
		}
	}
	if c.JSON {
		b, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return err
		}
		return e.out.Result(append(b, '\n'))
	}
	head := []string{labelProvider, "name", "credential", "sign-in"}
	if c.Check {
		head = append(head, "state")
	}
	cells := [][]string{head}
	for _, r := range rows {
		row := []string{r.ID, r.Name, r.Credential, r.SignIn}
		if c.Check {
			row = append(row, strings.TrimSpace(r.State+" "+r.Reason))
		}
		cells = append(cells, row)
	}
	return e.out.Result(columns(cells))
}

// listable reports whether provider id can be listed: it has a credential,
// or it takes none, as Ollama.
func (e *runEnv) listable(id string) bool {
	return e.hasCredential(id) || provider.CredentialVar(id) == ""
}

// checkedState is provider id's state and reason by its listing, or "no
// credential" with none.
func (e *runEnv) checkedState(id string) (string, string) {
	if !e.listable(id) {
		return "no credential", ""
	}
	state, reason, _ := listingState(id, e.modelListing(provider.Selection{ID: id}).get(), time.Now())
	return string(state), reason
}

// envName is a credential source that is an environment variable's name.
var envName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// credentialWords is where provider id's credential comes from, in Pi's
// words: "env: OPENAI_API_KEY", "auth.json: key", "auth.json: sign-in…",
// or none and how to set one (0025-MADR D5).
func (e *runEnv) credentialWords(id string) string {
	if provider.CredentialVar(id) == "" {
		return "runs on your machine, no key"
	}
	src := e.credentialSource(id)
	switch {
	case src == "none":
		return "none — gobble auth login " + id
	case src == "auth.json":
		return "auth.json: key"
	case strings.HasPrefix(src, "auth.json, an OAuth login"):
		return "auth.json: sign-in" + strings.TrimPrefix(src, "auth.json, an OAuth login")
	case envName.MatchString(src):
		return "env: " + src
	}
	return src
}

// signInWords is how provider id is signed in to (0025-MADR D5, D6).
func signInWords(id string) string {
	switch {
	case provider.CredentialVar(id) == "":
		return "—"
	case provider.HasLogin(id):
		return "API key; browser or device sign-in"
	case provider.HasKeyLogin(id):
		return "API key; device sign-in"
	}
	return "API key"
}

// columns lays cells out two spaces apart, each column as wide as its
// widest cell, no line ending in spaces.
func columns(cells [][]string) []byte {
	widths := make([]int, len(cells[0]))
	for _, row := range cells {
		for i, c := range row {
			widths[i] = max(widths[i], utf8.RuneCountInString(c))
		}
	}
	var b strings.Builder
	for _, row := range cells {
		var line strings.Builder
		for i, c := range row {
			if i > 0 {
				line.WriteString("  ")
			}
			fmt.Fprintf(&line, "%-*s", widths[i], c)
		}
		b.WriteString(strings.TrimRight(line.String(), " ") + "\n")
	}
	return []byte(b.String())
}
