package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The picker lists a Chat Completions server under its models.json name,
// and gobble models --provider lists its models (0036-PLAN P4).
func TestChatServerPickerAndModels(t *testing.T) {
	e, _ := modelEnv(t)
	writeModelsJSON(t, `{"providers":{"local":{"name":"Local","baseUrl":"http://127.0.0.1:1/v1","models":[{"id":"mine"}]}}}`)
	withLister(t, (&liveLister{ids: []string{"listed-1"}}).list)
	var label string
	for _, p := range e.providerChoices("", nil) {
		if p.ID == "local" {
			label = p.Label
		}
	}
	if label != "Local" {
		t.Fatalf("picker label %q, want Local", label)
	}
	r := runIn(t, os.Getenv("GOBBLE_HOME"), "models", "list", "--provider", "local")
	if r.code != 0 || !strings.Contains(r.stdout, "local") || !strings.Contains(r.stdout, "mine") {
		t.Fatalf("models --provider local (%d):\n%s%s", r.code, r.stdout, r.stderr)
	}
}

// gobble models list --provider local prints models.json's models and the
// live listing's, each once: the merge is printModels', not the parser's
// (0036-PLAN, gap G1).
func TestChatServerModelsListMerges(t *testing.T) {
	c := newCfgEnv(t)
	c.write(filepath.Join(c.home, "models.json"),
		`{"providers":{"local":{"name":"Local","baseUrl":"http://127.0.0.1:1/v1","models":[{"id":"mine"}]}}}`)
	l := &liveLister{ids: []string{"listed-1"}}
	withLister(t, l.list)
	r := c.run("models", "list", "--provider", "local")
	if r.code != ExitOK {
		t.Fatalf("models list --provider local: exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	var ids []string
	for _, row := range rowsOf(r.stdout)[1:] {
		if len(row) > 1 && row[0] == "local" {
			ids = append(ids, row[1])
		}
	}
	slices.Sort(ids)
	if !slices.Equal(ids, []string{"listed-1", "mine"}) {
		t.Fatalf("local's ids %q, want models.json's mine and the listing's listed-1, each once; stdout %q", ids, r.stdout)
	}
	if l.calls.Load() == 0 {
		t.Error("the live listing was never asked")
	}
}

// --model local/m resolves, through the CLI's own offline set, to provider
// local and model m when models.json is the only place that names m, and
// gobble models info resolves it the same way (0036-PLAN, gap G2).
func TestChatServerModelFlagResolves(t *testing.T) {
	const doc = `{"providers":{"local":{"name":"Local","baseUrl":"http://127.0.0.1:1/v1","models":[{"id":"m"}]}}}`
	e, _ := modelEnv(t)
	writeModelsJSON(t, doc)
	pick, err := e.pickModel(&ModelFlags{Model: "local/m"})
	if err != nil {
		t.Fatalf("--model local/m: %v", err)
	}
	if pick.Provider != "local" || pick.Model != "m" {
		t.Fatalf("--model local/m resolved to %+v, want provider local, model m", pick)
	}
	c := newCfgEnv(t)
	c.write(filepath.Join(c.home, "models.json"), doc)
	r := c.run("models", "info", "local/m")
	if r.code != ExitOK || !strings.HasPrefix(r.stdout, "provider  local\nmodel     m\n") {
		t.Fatalf("models info local/m: exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
}
