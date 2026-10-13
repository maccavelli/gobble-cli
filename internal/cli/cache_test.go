package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The agent's provider for a session is built with that session's id
// (0021-MADR D4).
func TestModelProviderSessionID(t *testing.T) {
	t.Setenv("GOBBLE_HOME", t.TempDir())
	var got string
	old := ambientProvider
	ambientProvider = func(_ func(string) string, _ string, o provider.Options) (llm.Provider, provider.Selection, error) {
		got = o.SessionID
		return nil, provider.Selection{}, errors.New("recorded")
	}
	t.Cleanup(func() { ambientProvider = old })
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := modelProvider(e)("sess-9"); err == nil || got != "sess-9" {
		t.Fatalf("built with session %q (err %v), want sess-9", got, err)
	}
}
