package cli

import (
	"os"
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
