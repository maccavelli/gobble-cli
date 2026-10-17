package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The agent's provider for a session is built with that session's id
// (0021-MADR D4).
func TestModelProviderSessionID(t *testing.T) {
	t.Setenv("GOBBLE_HOME", t.TempDir())
	var got string
	old := ambientProvider
	ambientProvider = func(_ func(string) (string, bool), _ string, o provider.Options) (llm.Provider, provider.Selection, error) {
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
	if _, err := modelProvider(e)("sess-9", llm.DefaultRetry); err == nil || got != "sess-9" {
		t.Fatalf("built with session %q (err %v), want sess-9", got, err)
	}
}

// cacheRetention reaches a session's requests: short by default, the
// settings file's value, --set's, and the default with a warning in the
// log for a value that is not one of the three (0021-MADR D3).
func TestCacheRetentionSetting(t *testing.T) {
	for _, c := range []struct {
		name, file string
		args       []string
		want       llm.CacheRetention
		warning    bool
	}{
		{name: "default", want: llm.CacheShort},
		{name: "file", file: "cacheRetention: long\n", want: llm.CacheLong},
		{name: "set", args: []string{"--set", "cacheRetention=none"}, want: llm.CacheNone},
		{name: "invalid", file: "cacheRetention: forever\n", want: llm.CacheShort, warning: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			t.Chdir(t.TempDir())
			if c.file != "" {
				writeFile(t, filepath.Join(home, "settings.yaml"), c.file)
			}
			script := llmtest.NewScript(llmtest.Text("ok"))
			withModel(t, script)
			r := runIn(t, home, append(slices.Clone(c.args), "-p", "x")...)
			if r.code != ExitOK {
				t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
			}
			if reqs := script.Requests(); len(reqs) != 1 || reqs[0].Cache != c.want {
				t.Fatalf("%d requests; want one asking %q", len(reqs), c.want)
			}
			if !c.warning {
				return
			}
			dirs, _, err := systemDirs()
			if err != nil {
				t.Fatal(err)
			}
			logged, err := os.ReadFile(filepath.Join(dirs.State, "logs", "gobble.log"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(logged), "cacheRetention: must be one of none, short, long") {
				t.Fatalf("the log lacks the warning:\n%s", logged)
			}
		})
	}
}
