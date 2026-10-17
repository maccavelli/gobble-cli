package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/llm"
)

// retryUpdates are a turn's waits as the client gets them: an offline
// wait and an empty reply's, each started and ended.
func retryUpdates() []acpclient.Update {
	return []acpclient.Update{
		{Kind: acpclient.KindRetry, Retry: acpclient.Retry{Phase: "start", Cause: "offline"}, Params: []byte(`{"phase":"start"}`)},
		{Kind: acpclient.KindRetry, Retry: acpclient.Retry{Phase: "end", Cause: "offline"}, Params: []byte(`{"phase":"end"}`)},
		{Kind: acpclient.KindRetry, Retry: acpclient.Retry{Phase: "start", Cause: "empty_reply", Attempt: 1, MaxAttempts: 2, Delay: 2 * time.Second},
			Params: []byte(`{"phase":"start"}`)},
		{Kind: acpclient.KindRetry, Retry: acpclient.Retry{Phase: "end", Cause: "empty_reply", Error: "retry cancelled"}, Params: []byte(`{"phase":"end"}`)},
	}
}

const retryStderr = "waiting for the network\nretrying: empty reply (attempt 1 of 2)\n"

// The CLI shows each wait's start on stderr, and nothing on stdout: the
// line view, print mode, and stream-json, whose stdout lines are
// session/update params alone (0005-PLAN F5-1 decision 5).
func TestRetryLines(t *testing.T) {
	var out, errw bytes.Buffer
	v := newView(&Output{out: &out, err: &errw}, display{}, stepClock())
	for _, u := range retryUpdates() {
		v.on(u)
	}
	if out.Len() != 0 || errw.String() != retryStderr {
		t.Fatalf("view: stdout %q, stderr %q; want nothing and %q", out.String(), errw.String(), retryStderr)
	}
	for _, stream := range []bool{false, true} {
		var out, errw bytes.Buffer
		s := &printSink{out: &Output{out: &out, err: &errw}, stream: stream, now: time.Now}
		s.begin("s1", "/w")
		header := out.String()
		for _, u := range retryUpdates() {
			s.on(u)
		}
		if out.String() != header || errw.String() != retryStderr {
			t.Fatalf("print (stream %v): stdout %q after its header %q, stderr %q; want nothing more and %q", stream, out.String(), header,
				errw.String(), retryStderr)
		}
	}
}

// The retry keys reach a session's settings: enabled false is Off, and the
// times are milliseconds; with none set, they are llm.DefaultRetry
// (0005-PLAN F5-1 decision 2).
func TestRetrySettingsReachSession(t *testing.T) {
	s := config.Defaults()
	if got := toSession(s).Retry; got != llm.DefaultRetry {
		t.Fatalf("default retry %+v, want %+v", got, llm.DefaultRetry)
	}
	s.Retry = config.Retry{Enabled: false, MaxRetries: 0, BaseDelayMs: 5, MaxDelayMs: 7, OfflineWaitMs: 0}
	want := llm.Retry{Off: true, MaxRetries: 0, BaseDelay: 5 * time.Millisecond, MaxDelay: 7 * time.Millisecond}
	if got := toSession(s).Retry; got != want {
		t.Fatalf("retry %+v, want %+v", got, want)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// _gobble/set_auto_retry's SaveSetting writes the key to the global
// settings file, creating it from the template, as gobble config set does;
// a bad value is refused (0005-PLAN F5-1 decision 6).
func TestSaveGlobalSetting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	save := e.agentOptions().SaveSetting
	if save == nil {
		t.Fatal("no SaveSetting")
	}
	if err := save("retry.enabled", "false"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(home, "settings.yaml")
	text := readFile(t, file)
	if !strings.Contains(text, "retry:\n  enabled: false\n") {
		t.Fatalf("settings.yaml:\n%s", text)
	}
	if err := save("retry.enabled", "maybe"); err == nil {
		t.Fatal("a bad value was written")
	}
	if got := readFile(t, file); got != text {
		t.Fatalf("a refused value changed the file:\n%s", got)
	}
}
