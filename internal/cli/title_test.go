package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// smallModel's provider is built by its own id and model, and names the
// session from its first prompt; with no credential for its provider, the
// run warns once and names nothing (0005-PLAN F5-5 decisions 1 and 10).
func TestSmallModelTitles(t *testing.T) {
	small := llmtest.NewScript(llmtest.Text("Fix the login test"))
	var built []string
	old := smallProvider
	smallProvider = func(_ *runEnv, id, model string) func(string, llm.Retry) (llm.Provider, error) {
		built = append(built, id+"/"+model)
		return func(string, llm.Retry) (llm.Provider, error) { return small, nil }
	}
	t.Cleanup(func() { smallProvider = old })

	t.Setenv("OPENAI_API_KEY", "test-key")
	h := newModelHarness(t, llmtest.NewScript(llmtest.Text("done")), t.TempDir())
	home := os.Getenv("GOBBLE_HOME")
	writeFile(t, filepath.Join(home, "settings.yaml"), "smallModel: openai/gpt-5-nano\n")
	h.start(Prompt{})
	h.press("fix the login test\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "done") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if len(built) != 1 || built[0] != "openai/gpt-5-nano" || len(small.Requests()) != 1 {
		t.Fatalf("built %q, %d title requests", built, len(small.Requests()))
	}
	files := storeFiles(t, filepath.Join(home, "sessions"))
	if len(files) != 1 {
		t.Fatalf("session files %q", files)
	}
	b, err := os.ReadFile(files[0])
	if err != nil || !strings.Contains(string(b), `"type":"session_info"`) || !strings.Contains(string(b), `"name":"Fix the login test"`) {
		t.Fatalf("the session file has no title:\n%s", b)
	}

	built = nil
	silent := t.TempDir()
	withModel(t, llmtest.NewScript(llmtest.Text("fine")))
	t.Chdir(t.TempDir())
	if r := mustRun(t, silent, "-p", "hi"); strings.Contains(r.stderr, "smallModel") || len(built) != 0 {
		t.Fatalf("no smallModel, and yet: built %q, stderr %q", built, r.stderr)
	}
	other := t.TempDir()
	writeFile(t, filepath.Join(other, "settings.yaml"), "smallModel: grok/grok-4.6\n")
	withModel(t, llmtest.NewScript(llmtest.Text("fine")))
	t.Chdir(t.TempDir())
	r := mustRun(t, other, "-p", "hi")
	if n := strings.Count(r.stderr, "smallModel grok/grok-4.6: grok has no credential, so no session is named"); n != 1 || len(built) != 0 {
		t.Fatalf("warned %d times, built %q; stderr %q", n, built, r.stderr)
	}
}
