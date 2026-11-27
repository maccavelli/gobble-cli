package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// realTrigger installs the automatic refresh's own trigger for t, in place
// of TestMain's, which starts nothing.
func realTrigger(t *testing.T) {
	t.Helper()
	old := startAutoModelData
	startAutoModelData = (*runEnv).autoRefreshModelData
	t.Cleanup(func() { startAutoModelData = old })
}

// settle waits for e's refresh under way to end.
func settle(e *runEnv) { e.modelData.group.Wait() }

// writeSettings writes the global settings.yaml of the run's home.
func writeSettings(t *testing.T, text string) {
	t.Helper()
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirs.Config, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirs.Config, "settings.yaml"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// agentConn is the in-process agent of e, as gobble's modes drive it.
func agentConn(t *testing.T, e *runEnv) *acpclient.Conn {
	t.Helper()
	conn, err := e.startAgentWith(t.Context(), nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
	return conn
}

// stampedAgo is the 2026-10-14-shaped document cliModelDoc makes, stamped
// as fetched from cliDataURL ago before now.
func stampedAgo(t *testing.T, sections map[string]int, ago time.Duration) *provider.ModelData {
	t.Helper()
	return cliParsed(t, cliModelDoc(sections), provider.ModelDataStamp{Source: cliDataURL,
		Fetched: time.Now().UTC().Add(-ago).Truncate(time.Second), ETag: `"seed"`})
}

// M24 to M27, M30 to M32, M67: what starts an automatic refresh, and what
// holds one back (0072-MADR D8, D18; "Only model calls and models cmds
// (Recommended)", "Never fetch; warn once (Recommended)", "Any attempt gates
// an hour (Recommended)").
func TestModelDataRefreshTrigger_0072(t *testing.T) {
	for _, c := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"model_call", triggerModelCall},
		{"models_cmds", triggerModelsCmds},
		{"initialize_no_fetch", triggerInitializeNoFetch},
		{"context_no_fetch", triggerContextNoFetch},
		{"one_in_flight", triggerOneInFlight},
		{"attempt_within_hour", triggerAttemptWithinHour},
		{"other_process_attempt", triggerOtherProcessAttempt},
		{"recheck", triggerRecheck},
		{"pinned", triggerPinned},
		{"clock_ahead", triggerClockAhead},
		{"fresh_copy", triggerFreshCopy},
		{"first_run", triggerFirstRun},
		{"refused_waits", triggerRefusedWaits},
		{"canceled_no_gate", triggerCanceledNoGate},
		{"started_stale", func(t *testing.T) { triggerStarted(t, unfinishedAttemptGate+51*time.Second, 1) }},
		{"started_fresh", func(t *testing.T) { triggerStarted(t, 30*time.Second, 0) }},
	} {
		t.Run(c.name, c.run)
	}
}

// triggerReply is the download the trigger's cases are served.
func triggerReply() dataReply {
	return dataReply{status: http.StatusOK, body: cliModelDoc(map[string]int{"openai": 3})}
}

func triggerModelCall(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	withModel(t, llmtest.NewScript(llmtest.Text("done")))
	s, err := agentConn(t, r.e).NewSession(t.Context(), t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	settle(r.e)
	if n := len(r.wire.requests()); n != 1 {
		t.Fatalf("%d requests after a model call with no copy; want one download", n)
	}
	if _, err := readModelDataCopy(r.copy); err != nil {
		t.Errorf("no copy after the download: %v", err)
	}
}

func triggerModelsCmds(t *testing.T) {
	for name, args := range map[string][]string{
		"models_list": {"models", "list"}, "models_info": {"models", "info", "openai/gpt-5-mini"}, "models_refresh": {"models", "refresh"},
	} {
		t.Run(name, func(t *testing.T) {
			r := newDataRun(t, triggerReply())
			realTrigger(t)
			res := runIn(t, os.Getenv("GOBBLE_HOME"), args...)
			if res.code != ExitOK {
				t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
			}
			if n := len(r.wire.requests()); n != 1 {
				t.Errorf("%d requests; want the command to download the due copy once", n)
			}
		})
	}
}

func triggerInitializeNoFetch(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	agentConn(t, r.e) // initialize reads the model facts (its image check)
	settle(r.e)
	if n := len(r.wire.requests()); n != 0 {
		t.Errorf("initialize made %d requests; want none (0008-MADR D5)", n)
	}
}

func triggerContextNoFetch(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	s, err := agentConn(t, r.e).NewSession(t.Context(), t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "/context"}); err != nil {
		t.Fatal(err)
	}
	settle(r.e)
	if n := len(r.wire.requests()); n != 0 {
		t.Errorf("/context made %d requests; want none", n)
	}
}

func triggerOneInFlight(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	release := make(chan struct{})
	var calls atomic.Int64
	fetchModelData = func(ctx context.Context, _ string, _ provider.Options) (*provider.ModelData, bool, error) {
		calls.Add(1)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil, false, errors.New("released")
	}
	for range 3 {
		r.e.refreshModelDataIfDue()
	}
	close(release)
	settle(r.e)
	if n := calls.Load(); n != 1 {
		t.Errorf("%d fetches for three model calls during one refresh; want one", n)
	}
}

func triggerAttemptWithinHour(t *testing.T) {
	r := newDataRun(t, dataReply{status: http.StatusServiceUnavailable})
	realTrigger(t)
	r.e.refreshModelDataIfDue()
	settle(r.e)
	r.e.refreshModelDataIfDue() // the same process, its attempt failed a moment ago
	settle(r.e)
	if n := len(r.wire.requests()); n != 1 {
		t.Errorf("%d requests; want one: its own failed attempt holds the process back an hour", n)
	}
}

func triggerOtherProcessAttempt(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	started := time.Now().UTC().Add(-50 * time.Minute).Format(time.RFC3339)
	if err := writeModelDataAttempt(r.attempt, modelDataAttempt{Source: cliDataURL, Started: started, Finished: started, Result: attemptFailed,
		Reason: "offline"}); err != nil {
		t.Fatal(err)
	}
	r.e.refreshModelDataIfDue()
	settle(r.e)
	if n := len(r.wire.requests()); n != 0 {
		t.Errorf("%d requests; want none: another process's attempt 50 minutes ago holds every process back", n)
	}
	if a := r.lastAttempt(t); a.Started != started {
		t.Errorf("attempt.json %+v; want the other process's left as it was", a)
	}
}

func triggerRecheck(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	fresh := stampedAgo(t, map[string]int{"openai": 2}, time.Minute)
	if err := writeModelDataCopy(r.copy, fresh); err != nil { // another process's, after this one loaded none
		t.Fatal(err)
	}
	r.e.refreshModelDataIfDue()
	settle(r.e)
	if n := len(r.wire.requests()); n != 0 {
		t.Errorf("%d requests; want none: the copy another process wrote is not due", n)
	}
	if got := provider.CurrentModelData().Stamp(); got != fresh.Stamp() {
		t.Errorf("current %+v; want the other process's copy %+v", got, fresh.Stamp())
	}
}

func triggerPinned(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	writeSettings(t, "modelData:\n  refreshHours: 0\n")
	r.e.refreshModelDataIfDue()
	settle(r.e)
	if n := len(r.wire.requests()); n != 0 {
		t.Errorf("refreshHours 0: %d requests on its own; want none", n)
	}
	if res := runIn(t, os.Getenv("GOBBLE_HOME"), "models", "refresh"); res.code != ExitOK || len(r.wire.requests()) != 1 {
		t.Errorf("gobble models refresh, pinned: exit %d, %d requests; want it to fetch", res.code, len(r.wire.requests()))
	}
}

func triggerClockAhead(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	provider.SetCurrentModelData(stampedAgo(t, map[string]int{"openai": 2}, -time.Hour))
	r.e.refreshModelDataIfDue()
	settle(r.e)
	if n := len(r.wire.requests()); n != 1 {
		t.Errorf("a copy fetched after now: %d requests; want it due", n)
	}
}

func triggerFreshCopy(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	provider.SetCurrentModelData(stampedAgo(t, map[string]int{"openai": 2}, 23*time.Hour))
	r.e.refreshModelDataIfDue()
	settle(r.e)
	if n := len(r.wire.requests()); n != 0 {
		t.Errorf("a copy 23 hours old: %d requests; want none before refreshHours", n)
	}
}

func triggerFirstRun(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	r.e.refreshModelDataIfDue()
	settle(r.e)
	if d := provider.CurrentModelData(); len(r.wire.requests()) != 1 || d.Stamp().Source != cliDataURL || d.Sections()["openai"] != 3 {
		t.Errorf("first run: %d requests, current %+v; want the download current", len(r.wire.requests()), d.Stamp())
	}
}

func triggerRefusedWaits(t *testing.T) {
	r := newDataRun(t, dataReply{status: http.StatusOK, body: cliModelDoc(map[string]int{"openai": 3})})
	home := os.Getenv("GOBBLE_HOME")
	realTrigger(t)
	have := stampedAgo(t, map[string]int{"openai": 3, "xai": 2}, 48*time.Hour)
	if err := writeModelDataCopy(r.copy, have); err != nil {
		t.Fatal(err)
	}
	provider.SetCurrentModelData(have)
	r.e.refreshModelDataIfDue()
	settle(r.e)
	if a := r.lastAttempt(t); a.Result != attemptRefused || !strings.Contains(a.Reason, "xai section is missing") {
		t.Fatalf("attempt %+v; want the refusal recorded with its reason", a)
	}
	if n := strings.Count(r.warnings.String(), "a download was refused"); n != 1 {
		t.Errorf("%d refusal warnings; want one: %q", n, r.warnings.String())
	}
	next := newDataRun(t, dataReply{status: http.StatusOK, body: cliModelDoc(map[string]int{"openai": 3})})
	t.Setenv("GOBBLE_HOME", home) // the first run's home
	next.e.refreshModelDataIfDue()
	settle(next.e)
	if n := len(next.wire.requests()); n != 0 {
		t.Errorf("another run within the hour of a refusal made %d requests; want none", n)
	}
}

// M87: an attempt another process left started, with no finished time,
// holds back every process's automatic refresh for 120 s only: one older
// does not gate (want 1 request), one younger does (want 0) ("Unfinished
// attempts gate only briefly (Recommended)").
func triggerStarted(t *testing.T, ago time.Duration, want int) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	started := time.Now().UTC().Add(-ago).Format(time.RFC3339)
	if err := writeModelDataAttempt(r.attempt, modelDataAttempt{Source: cliDataURL, Started: started, Result: attemptStarted}); err != nil {
		t.Fatal(err)
	}
	r.e.refreshModelDataIfDue()
	settle(r.e)
	if n := len(r.wire.requests()); n != want {
		t.Errorf("an unfinished attempt %v old: %d requests; want %d", ago, n, want)
	}
}

// fetchBlocks makes the engine's fetch block until its context ends; started
// is closed when it begins.
func fetchBlocks(t *testing.T) (started chan struct{}) {
	t.Helper()
	started = make(chan struct{})
	var once sync.Once
	old := fetchModelData
	fetchModelData = func(ctx context.Context, _ string, _ provider.Options) (*provider.ModelData, bool, error) {
		once.Do(func() { close(started) })
		<-ctx.Done()
		return nil, false, ctx.Err()
	}
	t.Cleanup(func() { fetchModelData = old })
	return started
}

// waitClosed waits up to 10 s for ch to close, else fails t with what.
func waitClosed(t *testing.T, ch chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatalf("%s did not happen within 10 s", what)
	}
}

// M84: a refresh the run's end cancels is recorded canceled, and holds no
// process back: another run's due refresh fetches at once ("A cancel
// doesn't gate or warn (Recommended)").
func triggerCanceledNoGate(t *testing.T) {
	r := newDataRun(t)
	home := os.Getenv("GOBBLE_HOME")
	realTrigger(t)
	started := fetchBlocks(t)
	r.e.refreshModelDataIfDue()
	waitClosed(t, started, "the refresh's fetch")
	r.e.endModelData() // gobble acp's or the interactive CLI's exit: cancelled at once
	settle(r.e)
	if a := r.lastAttempt(t); a.Result != attemptCanceled || a.Reason != "" {
		t.Fatalf("attempt %+v; want the cancel recorded as canceled, with no reason", a)
	}
	next := newDataRun(t, triggerReply())
	t.Setenv("GOBBLE_HOME", home) // the first run's home, its attempt.json
	next.e.refreshModelDataIfDue()
	settle(next.e)
	if n := len(next.wire.requests()); n != 1 {
		t.Errorf("another run after a cancelled attempt made %d requests; want one: a cancel gates nothing", n)
	}
}

// cancelAnAttempt leaves in the home t uses now the attempt.json a refresh
// cancelled by its run's end writes: a run whose fetch blocks, ended as
// gobble acp ends. The copy reader and the fetch are left as they were.
func cancelAnAttempt(t *testing.T) {
	t.Helper()
	home, load, fetch := os.Getenv("GOBBLE_HOME"), loadModelData, fetchModelData
	e, _ := modelEnv(t)
	t.Setenv("GOBBLE_HOME", home)
	started := fetchBlocks(t)
	if !e.startModelDataRefresh(modelDataBackground, false) {
		t.Fatal("the refresh did not start")
	}
	waitClosed(t, started, "the refresh's fetch")
	e.endModelData()
	settle(e)
	loadModelData, fetchModelData = load, fetch
}

// M28: at the run's end gobble acp and the interactive CLI cancel a refresh
// under way at once and do not wait; a headless run and the models
// commands wait for it up to 5 s, its write included, then cancel
// (0072-MADR D20; "acp/interactive cancel at once (Recommended)").
func TestModelDataRunEnd_0072(t *testing.T) {
	for _, tc := range []struct {
		name    string
		waits   bool
		fetch   time.Duration // 0 blocks until cancelled
		elapsed time.Duration
		written bool
	}{
		{"acp", false, 0, 0, false},
		{"interactive", false, 0, 0, false},
		{"headless/finishes", true, time.Second, time.Second, true},
		{"headless/blocked", true, 0, 5 * time.Second, false},
		{"models/finishes", true, time.Second, time.Second, true},
		{"models/blocked", true, 0, 5 * time.Second, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newDataRun(t)
				var cancelled atomic.Bool
				fetchModelData = func(ctx context.Context, url string, _ provider.Options) (*provider.ModelData, bool, error) {
					if tc.fetch > 0 {
						time.Sleep(tc.fetch)
						return stampedAgo(t, map[string]int{"openai": 2}, 0), false, nil
					}
					<-ctx.Done()
					cancelled.Store(true)
					return nil, false, ctx.Err()
				}
				if tc.waits {
					r.e.waitModelDataAtEnd()
				}
				if !r.e.startModelDataRefresh(modelDataBackground, false) {
					t.Fatal("the refresh did not start")
				}
				synctest.Wait()
				start := time.Now()
				r.e.endModelData()
				if got := time.Since(start); got != tc.elapsed {
					t.Errorf("the end took %v; want %v", got, tc.elapsed)
				}
				settle(r.e)
				if _, err := readModelDataCopy(r.copy); (err == nil) != tc.written {
					t.Errorf("copy written %v (%v); want %v", err == nil, err, tc.written)
				}
				if tc.fetch == 0 && !cancelled.Load() {
					t.Error("a blocked refresh was not cancelled at the end")
				}
				if a := r.lastAttempt(t); tc.fetch == 0 && a.Result != attemptCanceled {
					t.Errorf("attempt %+v; want the cancelled refresh recorded as canceled (M84)", a)
				}
			})
		})
	}
	// Which commands wait: a headless run and the models commands; gobble
	// acp does not.
	for name, args := range map[string][]string{"headless": {"--no-session", "-p", "hello"}, "models": {"models", "list"}} {
		t.Run("command/"+name, func(t *testing.T) {
			newDataRun(t)
			realTrigger(t)
			withModel(t, llmtest.NewScript(llmtest.Text("done")))
			old := modelDataExitWait
			modelDataExitWait = 300 * time.Millisecond
			t.Cleanup(func() { modelDataExitWait = old })
			fetchModelData = func(ctx context.Context, _ string, _ provider.Options) (*provider.ModelData, bool, error) {
				<-ctx.Done()
				return nil, false, ctx.Err()
			}
			start := time.Now()
			res := runIn(t, os.Getenv("GOBBLE_HOME"), args...)
			if got := time.Since(start); res.code != ExitOK || got < modelDataExitWait {
				t.Errorf("exit %d after %v; want the run to wait %v for its refresh", res.code, got, modelDataExitWait)
			}
			// The run cancels its refresh and returns (D20); the cancelled
			// refresh then records its end. Wait for that write, bounded, so
			// it does not race the removal of the test's home.
			dirs, _, err := systemDirs()
			if err != nil {
				t.Fatal(err)
			}
			_, attemptPath := modelDataPaths(dirs)
			waitRecorded(t, attemptPath)
		})
	}
	t.Run("command/acp", runEndACPCommand)
	t.Run("command/interactive", func(t *testing.T) {
		if interactiveEndCase == nil {
			t.Skip("no pseudo-terminal in this build; on Windows the conpty test is M28's interactive twin")
		}
		interactiveEndCase(t)
	})
}

// interactiveEndCase is M28's command-level interactive case, set where a
// pseudo-terminal can be opened in this process (modeldatatty_unix_test.go).
var interactiveEndCase func(*testing.T)

// waitRecorded waits, bounded at 10 s, for the cancelled refresh to record
// its end in attemptPath, so it does not race the removal of the test's
// home, and checks it is recorded canceled (M84).
func waitRecorded(t *testing.T, attemptPath string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		a, ok, _ := readModelDataAttempt(attemptPath)
		if ok && a.Result != attemptStarted {
			if a.Result != attemptCanceled {
				t.Errorf("attempt %+v; want the cancelled refresh recorded as canceled (M84)", a)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the cancelled refresh never recorded its end")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// acpOverPipe is an acpclient.ServeFunc that runs gobble acp through Main,
// its stdin a pipe fed from the client: eof gets the time the client's
// input ended, exited the time Main returned, and code its exit code.
func acpOverPipe(eof, exited chan time.Time, code chan int) func(context.Context, io.Reader, io.Writer) error {
	return func(ctx context.Context, in io.Reader, out io.Writer) error {
		r, w, err := os.Pipe()
		if err != nil {
			return err
		}
		go func() {
			_, _ = io.Copy(w, in) //nolint:errcheck // the client's end is the only end
			eof <- time.Now()
			_ = w.Close() //nolint:errcheck // the reader sees end of file either way
		}()
		var errw bytes.Buffer
		c := Main(context.WithoutCancel(ctx), []string{"acp"}, r, out, &errw) // the client's grace never cancels it
		exited <- time.Now()
		_ = r.Close() //nolint:errcheck // read to its end
		code <- c
		return nil
	}
}

// runEndACPCommand: gobble acp itself, over its stdin, with the real
// trigger: a prompt's model call starts a refresh whose fetch blocks; when
// the client's input ends, gobble acp exits at once, well within the 2 s
// M28 gives it and far below the 5 s a headless run waits, and the refresh
// is recorded canceled (0072-MADR D20, M28, M84).
func runEndACPCommand(t *testing.T) {
	newDataRun(t)
	realTrigger(t)
	withModel(t, llmtest.NewScript(llmtest.Text("done")))
	started := fetchBlocks(t)
	eof, exited, code := make(chan time.Time, 1), make(chan time.Time, 1), make(chan int, 1)
	conn, err := acpclient.Start(t.Context(), acpOverPipe(eof, exited, code), acpclient.Options{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	s, err := conn.NewSession(t.Context(), t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	waitClosed(t, started, "the model call's refresh")
	_ = conn.Close() //nolint:errcheck // the agent's end is what is measured
	var from, to time.Time
	select {
	case from = <-eof:
	case <-time.After(10 * time.Second):
		t.Fatal("the client's input never ended")
	}
	select {
	case to = <-exited:
	case <-time.After(20 * time.Second):
		t.Fatal("gobble acp did not exit within 20 s of its input's end")
	}
	if c := <-code; c != ExitOK || to.Sub(from) >= 2*time.Second {
		t.Errorf("gobble acp: exit %d, %v after its input ended; want 0 within 2 s, not waiting for its refresh", c, to.Sub(from))
	}
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	_, attemptPath := modelDataPaths(dirs)
	waitRecorded(t, attemptPath)
}

// M29: completion mode loads the copy and never fetches, whatever its age
// (0072-MADR D8; "Only model calls and models cmds (Recommended)").
func TestCompletionStartsNoRefresh_0072(t *testing.T) {
	realTrigger(t)
	stale := cliParsed(t, []byte(copyOnlyDoc), provider.ModelDataStamp{Source: provider.ModelDataURL, Fetched: time.Now().Add(-72 * time.Hour)})
	loads, fetches := withCopy(t, stale)
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	t.Setenv(complete.EnvShell, "fish")
	t.Setenv(complete.EnvProtocol, complete.Protocol)
	var out, errw bytes.Buffer
	code := Main(t.Context(), []string{"--", "--model", "openai/copy-o"}, nil, &out, &errw)
	if code != ExitOK || !strings.Contains(out.String(), "openai/copy-only-1") || errw.Len() != 0 {
		t.Errorf("exit %d, stdout %q, stderr %q; want the copy's model offered", code, out.String(), errw.String())
	}
	if loads.Load() != 1 || fetches.Load() != 0 {
		t.Errorf("%d loads, %d fetches; want one load and no fetch of a copy 72 hours old", loads.Load(), fetches.Load())
	}
	if des, _ := os.ReadDir(home); len(des) != 0 {
		t.Errorf("completion mode wrote %v", des)
	}
}

// M37 (internal/cli's half): the line for a session's document, in each of
// D12's forms (0072-MADR D12).
func TestModelDataLine_0072(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 30, 0, 0, time.UTC)
	at := func(ago time.Duration) *provider.ModelData {
		return cliParsed(t, cliModelDoc(map[string]int{"openai": 1}), provider.ModelDataStamp{Source: provider.ModelDataURL, Fetched: now.Add(-ago)})
	}
	for _, tc := range []struct {
		name string
		d    *provider.ModelData
		want string
	}{
		{"fresh", at(3 * time.Hour), "Model data: models.dev, fetched 2026-11-25 09:30 UTC (3 hours ago)"},
		{"fresh/minutes", at(time.Minute), "Model data: models.dev, fetched 2026-11-25 12:29 UTC (1 minute ago)"},
		{"past_warn", at(8 * 24 * time.Hour), "Model data: models.dev, fetched 2026-11-17 12:30 UTC (8 days ago); gobble models refresh fetches it again"},
		{"past_warn/at_warnDays", at(7 * 24 * time.Hour), "Model data: models.dev, fetched 2026-11-18 12:30 UTC (7 days ago); gobble models refresh fetches it again"},
		{"none", nil, "Model data: none; models are unknown until gobble models refresh downloads them"},
		{"clock_ahead", at(-time.Hour), "Model data: models.dev, fetched 2026-11-25 13:30 UTC (ahead of this clock)"},
	} {
		if got := modelDataLine(tc.d, now, 7); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
	if got := modelDataLine(at(30*24*time.Hour), now, 0); strings.Contains(got, "fetches it again") {
		t.Errorf("warnDays 0: %q; want no advice", got)
	}
}

// warnEnv is a home for one warning case: the copy the run loads, the
// last attempt and the settings, each optional.
func warnEnv(t *testing.T, copyData *provider.ModelData, attempt *modelDataAttempt, settings string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	withCopy(t, copyData)
	if attempt != nil {
		dirs, _, err := systemDirs()
		if err != nil {
			t.Fatal(err)
		}
		_, path := modelDataPaths(dirs)
		if err := writeModelDataAttempt(path, *attempt); err != nil {
			t.Fatal(err)
		}
	}
	if settings != "" {
		writeSettings(t, settings)
	}
	return home
}

// M39, M40, M63: one warning per run on stderr at its start, from the copy
// and the last attempt; gobble acp logs it, at warn, once (0072-MADR D12,
// D13; "Refresh 24h, warn past 7 days (Recommended)", "Never fetch; warn
// once (Recommended)").
func TestModelDataWarning_0072(t *testing.T) {
	old := stampedAgo(t, map[string]int{"openai": 2}, 8*24*time.Hour)
	ago := func(d time.Duration) string { return time.Now().UTC().Add(-d).Format(time.RFC3339) }
	const age = "Warning: model data was fetched 8 days ago from models-dev.invalid; gobble models refresh fetches it again"
	const none = "Warning: no model data: models are unknown; gobble models refresh downloads them"
	for _, tc := range []struct {
		name     string
		copy     *provider.ModelData
		attempt  *modelDataAttempt
		settings string
		want     string // the whole of stderr
	}{
		{"past", old, nil, "", age + "\n"},
		{"refused", old, &modelDataAttempt{Source: cliDataURL, Started: ago(2 * time.Hour), Result: attemptRefused, Reason: "its xai section is missing"}, "",
			age + "; the last download was refused: its xai section is missing; gobble models refresh --force accepts it\n"},
		{"failed", old, &modelDataAttempt{Source: cliDataURL, Started: ago(2 * time.Hour), Result: attemptFailed, Reason: "offline"}, "",
			age + "; the last download failed: offline\n"},
		{"one_day", stampedAgo(t, map[string]int{"openai": 2}, 25*time.Hour), nil, "modelData:\n  warnDays: 1\n",
			"Warning: model data was fetched 1 day ago from models-dev.invalid; gobble models refresh fetches it again\n"},
		{"refused_empty", old, &modelDataAttempt{Source: cliDataURL, Started: ago(2 * time.Hour), Result: attemptRefused, Reason: "it holds no models"}, "",
			age + "; the last download was refused: it holds no models\n"},
		{"off", old, nil, "modelData:\n  warnDays: 0\n", ""},
		{"fresh", stampedAgo(t, map[string]int{"openai": 2}, 6*24*time.Hour), nil, "", ""},
		{"clock_ahead", stampedAgo(t, map[string]int{"openai": 2}, -time.Hour), nil, "", ""},
		{"none_pinned", nil, nil, "modelData:\n  refreshHours: 0\n", none + "\n"},
		{"none_failed", nil, &modelDataAttempt{Source: cliDataURL, Started: ago(2 * time.Hour), Result: attemptFailed, Reason: "offline"}, "",
			none + "; the last download failed: offline\n"},
		{"none_untried", nil, nil, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := warnEnv(t, tc.copy, tc.attempt, tc.settings)
			res := runIn(t, home, "models", "list")
			if res.code != ExitOK || res.stderr != tc.want {
				t.Errorf("exit %d, stderr %q; want %q", res.code, res.stderr, tc.want)
			}
		})
	}
	// M84: the attempt a refresh cancelled at a run's end leaves, as gobble
	// writes it, is named neither as a refusal nor as a failure ("A cancel
	// doesn't gate or warn (Recommended)").
	for _, tc := range []struct {
		name string
		copy *provider.ModelData
		want string
	}{{"canceled", old, age + "\n"}, {"none_canceled", nil, ""}} {
		t.Run(tc.name, func(t *testing.T) {
			home := warnEnv(t, tc.copy, nil, "")
			cancelAnAttempt(t)
			res := runIn(t, home, "models", "list")
			if res.code != ExitOK || res.stderr != tc.want {
				t.Errorf("exit %d, stderr %q; want %q", res.code, res.stderr, tc.want)
			}
		})
	}
	t.Run("list_stdout", func(t *testing.T) {
		warned := runIn(t, warnEnv(t, old, nil, ""), "models", "list")
		quiet := runIn(t, warnEnv(t, stampedAgo(t, map[string]int{"openai": 2}, time.Hour), nil, ""), "models", "list")
		if warned.stderr == "" || warned.stdout != quiet.stdout {
			t.Errorf("stdout %q with the warning, %q without; want the same", warned.stdout, quiet.stdout)
		}
	})
	t.Run("acp_log_once", func(t *testing.T) {
		home := warnEnv(t, old, nil, "")
		res := runIn(t, home, "acp")
		if res.code != ExitOK || res.stderr != "" {
			t.Fatalf("exit %d, stderr %q; want nothing on stderr under gobble acp", res.code, res.stderr)
		}
		dirs, _, err := systemDirs()
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dirs.State, "logs", "gobble.log"))
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(b), "model data was fetched 8 days ago"); n != 1 || !strings.Contains(string(b), "WARN") {
			t.Errorf("%d warnings in the log; want one at warn:\n%s", n, b)
		}
	})
}

// refreshCmdEnv is a home whose run fetches through an in-memory transport
// from cliDataURL and reads its copy as gobble does; seed, when not nil,
// is the copy, fetched at seedAt.
func refreshCmdEnv(t *testing.T, seed []byte, replies ...dataReply) (home string, wire *dataTransport, copyPath string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	wire = &dataTransport{replies: replies}
	oldFetch, oldLoad, oldSource := fetchModelData, loadModelData, modelDataSource
	fetchModelData = func(ctx context.Context, url string, o provider.Options) (*provider.ModelData, bool, error) {
		o.HTTPClient = &http.Client{Transport: wire}
		return provider.FetchModelData(ctx, url, o)
	}
	loadModelData, modelDataSource = readModelDataCopy, cliDataURL
	keepModelData(t)
	t.Cleanup(func() {
		fetchModelData, loadModelData, modelDataSource = oldFetch, oldLoad, oldSource
	})
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	copyPath, _ = modelDataPaths(dirs)
	if seed != nil {
		if err := writeModelDataCopy(copyPath, cliParsed(t, seed, provider.ModelDataStamp{Source: cliDataURL, Fetched: seedAt, ETag: `"seed"`})); err != nil {
			t.Fatal(err)
		}
	}
	return home, wire, copyPath
}

// seedAt is when refreshCmdEnv's copy was fetched.
var seedAt = time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)

// fetchedOf is the copy's fetched time, as the command writes it.
func fetchedOf(t *testing.T, path string) string {
	t.Helper()
	d, err := readModelDataCopy(path)
	if err != nil {
		t.Fatal(err)
	}
	return d.Stamp().Fetched.UTC().Format("2006-01-02 15:04")
}

// M18, M41 to M46, M68: gobble models refresh [--force]: its line, its
// error and its exit code, compared whole (0072-MADR D6, D9; "Add a --force
// flag", "Still refuse an empty document (Recommended)", "Background 30 s,
// refresh cmd 120 s (Recommended)").
func TestModelsRefreshCmd_0072(t *testing.T) {
	const url = cliDataURL
	two := cliModelDoc(map[string]int{"openai": 3, "xai": 2})
	ok := func(body []byte) dataReply { return dataReply{status: http.StatusOK, body: body} }
	kept := "the copy fetched 2026-11-20 10:00 is kept."
	for _, tc := range []struct {
		name   string
		seed   []byte
		reply  dataReply
		args   []string
		code   int
		stdout func(fetched string) string
		stderr string
	}{
		{"first", nil, ok(two), nil, ExitOK, func(f string) string {
			return "Model data: fetched from models-dev.invalid at " + f + " UTC: 5 models in 2 sections; there was no copy.\n"
		}, ""},
		{"saved", two, ok(cliModelDoc(map[string]int{"openai": 4, "xai": 2})), nil, ExitOK, func(f string) string {
			return "Model data: fetched from models-dev.invalid at " + f + " UTC: 6 models in 2 sections; it replaces the copy fetched 2026-11-20 10:00.\n"
		}, ""},
		{"unchanged", two, ok(two), nil, ExitOK, func(f string) string {
			return "Model data: fetched from models-dev.invalid at " + f + " UTC: 5 models in 2 sections; unchanged since the copy fetched 2026-11-20 10:00.\n"
		}, ""},
		{"not_modified", two, dataReply{status: http.StatusNotModified}, nil, ExitOK, func(f string) string {
			return "Model data: fetched from models-dev.invalid at " + f + " UTC: 5 models in 2 sections; not modified since 2026-11-20 10:00.\n"
		}, ""},
		{"offline", two, dataReply{err: errors.New("no route")}, nil, ExitFailure, nil,
			"Error: model data: could not fetch " + url + ": offline; " + kept + "\n"},
		{"offline/no_copy", nil, dataReply{err: errors.New("no route")}, nil, ExitFailure, nil,
			"Error: model data: could not fetch " + url + ": offline; there is no copy.\n"},
		{"status", two, dataReply{status: http.StatusServiceUnavailable}, nil, ExitFailure, nil,
			"Error: model data: could not fetch " + url + ": status 503; " + kept + "\n"},
		{"refused", two, ok(cliModelDoc(map[string]int{"openai": 3})), nil, ExitFailure, nil,
			"Error: model data: the document from " + url + " was not saved: its xai section is missing; " + kept + " gobble models refresh --force saves it.\n"},
		{"force", two, ok(cliModelDoc(map[string]int{"openai": 3})), []string{"--force"}, ExitOK, func(f string) string {
			return "Model data: fetched from models-dev.invalid at " + f + " UTC: 3 models in 1 section; it replaces the copy fetched 2026-11-20 10:00.\n"
		}, ""},
		{"one_model", nil, ok(cliModelDoc(map[string]int{"openai": 1})), nil, ExitOK, func(f string) string {
			return "Model data: fetched from models-dev.invalid at " + f + " UTC: 1 model in 1 section; there was no copy.\n"
		}, ""},
		{"force_empty", two, ok([]byte("{}\n")), []string{"--force"}, ExitFailure, nil,
			"Error: model data: the document from " + url + " was not saved: it holds no models; " + kept + "\n"},
		{"usage", nil, ok(two), []string{"extra"}, ExitUsage, nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, wire, copyPath := refreshCmdEnv(t, tc.seed, tc.reply)
			var before []byte
			if tc.seed != nil {
				before = fileBytes(t, copyPath)
			}
			res := runIn(t, home, append([]string{"models", "refresh"}, tc.args...)...)
			want := ""
			if tc.stdout != nil {
				want = tc.stdout(fetchedOf(t, copyPath))
			}
			stderrOK := res.stderr == tc.stderr || (tc.code == ExitUsage && strings.Contains(res.stderr, "unexpected argument extra"))
			if res.code != tc.code || res.stdout != want || !stderrOK {
				t.Errorf("exit %d, stdout %q, stderr %q; want %d, %q, %q", res.code, res.stdout, res.stderr, tc.code, want, tc.stderr)
			}
			if tc.code != ExitOK && tc.seed != nil && !bytes.Equal(fileBytes(t, copyPath), before) {
				t.Error("the copy changed")
			}
			if tc.name == "not_modified" {
				if reqs := wire.requests(); len(reqs) != 1 || reqs[0].Header.Get("If-None-Match") != `"seed"` {
					t.Error("the conditional GET did not send the copy's ETag")
				}
			}
		})
	}
	t.Run("not_saved", func(t *testing.T) {
		home, _, copyPath := refreshCmdEnv(t, nil, ok(two))
		renameVia(t, func(tmp, name string) error {
			if filepath.Base(name) == "models-dev.json" {
				return errors.New("planted: the copy cannot be saved")
			}
			return os.Rename(tmp, name)
		})
		res := runIn(t, home, "models", "refresh")
		if res.code != ExitFailure || res.stdout != "" || !strings.HasPrefix(res.stderr, "Error: model data: fetched from "+url+" but not saved: ") ||
			!strings.Contains(res.stderr, "planted: the copy cannot be saved") || !strings.HasSuffix(res.stderr, ".\n") {
			t.Errorf("exit %d, stdout %q, stderr %q", res.code, res.stdout, res.stderr)
		}
		if _, err := os.Stat(copyPath); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("a copy was written (%v)", err)
		}
	})
	t.Run("timeout_120s", func(t *testing.T) {
		home, _, _ := refreshCmdEnv(t, two)
		var left time.Duration
		fetchModelData = func(ctx context.Context, _ string, _ provider.Options) (*provider.ModelData, bool, error) {
			dl, _ := ctx.Deadline()
			left = time.Until(dl)
			return nil, false, fmt.Errorf("provider: model data: fetch: %w", context.DeadlineExceeded)
		}
		res := runIn(t, home, "models", "refresh")
		if want := "Error: model data: could not fetch " + url + ": timeout; " + kept + "\n"; res.code != ExitFailure || res.stderr != want {
			t.Errorf("exit %d, stderr %q; want %q", res.code, res.stderr, want)
		}
		if left <= 119*time.Second || left > 120*time.Second {
			t.Errorf("the command's fetch had %v; want 120 s", left)
		}
	})
	t.Run("interrupted", func(t *testing.T) {
		home, _, copyPath := refreshCmdEnv(t, two)
		before := fileBytes(t, copyPath)
		t.Setenv("GOBBLE_HOME", home)
		ctx, cancel := context.WithCancelCause(t.Context())
		// The interrupt arrives once the reply is read: nothing is saved.
		fetchModelData = func(fctx context.Context, _ string, _ provider.Options) (*provider.ModelData, bool, error) {
			cancel(signalCause{sig: os.Interrupt})
			<-fctx.Done()
			return stampedAgo(t, map[string]int{"openai": 9, "xai": 2}, 0), false, nil // one the guards accept
		}
		var out, errw bytes.Buffer
		code := Main(ctx, []string{"models", "refresh"}, nil, &out, &errw)
		if want := exitForSignal(os.Interrupt); code != want || out.Len() != 0 {
			t.Errorf("exit %d, stdout %q; want %d and nothing printed", code, out.String(), want)
		}
		if !bytes.Equal(fileBytes(t, copyPath), before) {
			t.Error("an interrupted refresh changed the copy")
		}
	})
}

// M85: gobble version and gobble config path, which create no state
// directory and no log file (0008-PLAN P4), do not load the model data: an
// unreadable copy logs nothing for them, as it does for every other command
// (0072-MADR D3, D7).
func TestModelDataNotLoadedForLogFree_0072(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"config", "path"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("GOBBLE_HOME", home)
			old := loadModelData
			loadModelData = readModelDataCopy
			t.Cleanup(func() { loadModelData = old })
			dirs, _, err := systemDirs()
			if err != nil {
				t.Fatal(err)
			}
			copyPath, _ := modelDataPaths(dirs)
			if err := os.MkdirAll(filepath.Dir(copyPath), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(copyPath, []byte("not a copy"), 0o600); err != nil {
				t.Fatal(err)
			}
			res := runIn(t, home, args...)
			if res.code != ExitOK || res.stderr != "" {
				t.Errorf("exit %d, stderr %q; want nothing on stderr", res.code, res.stderr)
			}
			if _, err := os.Stat(filepath.Join(dirs.State, "logs")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("the run made its log directory (%v); want no log file", err)
			}
		})
	}
	t.Run("models_list_loads", func(t *testing.T) { // the control: any other command loads it, and its D3 warning logs
		home := t.TempDir()
		t.Setenv("GOBBLE_HOME", home)
		old := loadModelData
		loadModelData = readModelDataCopy
		t.Cleanup(func() { loadModelData = old })
		dirs, _, err := systemDirs()
		if err != nil {
			t.Fatal(err)
		}
		copyPath, _ := modelDataPaths(dirs)
		if err := os.MkdirAll(filepath.Dir(copyPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(copyPath, []byte("not a copy"), 0o600); err != nil {
			t.Fatal(err)
		}
		if res := runIn(t, home, "models", "list"); !strings.Contains(res.stderr, "the copy cannot be read") {
			t.Errorf("gobble models list: stderr %q; want the copy's D3 warning", res.stderr)
		}
	})
}

// M80 (internal/cli's half): the agent's ModelInfo and CachedLimits hooks
// hand the session's document to the lookups, and nil reads the current one
// (0072-MADR D19; "Hooks take the session's document (Recommended)").
func TestAgentHooksReadSessionData_0072(t *testing.T) {
	e, _ := modelEnv(t)
	keepModelData(t)
	doc := func(window string) *provider.ModelData {
		return cliParsed(t, []byte(`{"openai":{"models":{"hooked":{"name":"Hooked","limit":{"context":`+window+`,"output":1000}}}}}`),
			provider.ModelDataStamp{Source: cliDataURL, Fetched: time.Now()})
	}
	session, current := doc("50000"), doc("70000")
	provider.SetCurrentModelData(current)
	old := modelFacts
	modelFacts = provider.ModelInfoWith // the real lookup, with no key: it reads the layers alone
	t.Cleanup(func() { modelFacts = old })
	o := e.agentOptions()
	if m := o.ModelInfo(session, "openai", "hooked"); m.ContextWindow != 50000 {
		t.Errorf("ModelInfo with the session's document: window %d, want 50,000", m.ContextWindow)
	}
	if m := o.ModelInfo(nil, "openai", "hooked"); m.ContextWindow != 70000 {
		t.Errorf("ModelInfo with none: window %d, want the current 70,000", m.ContextWindow)
	}
	if l := o.CachedLimits(session, "openai", "hooked"); l.Window.Tokens != 50000 {
		t.Errorf("CachedLimits with the session's document: window %+v, want 50,000", l.Window)
	}
	// The Discover hook hands the session's document to discovery, as every
	// lookup for the session reads it (0072-MADR D17, its ninth addition).
	var discovered *provider.ModelData
	oldDiscover := discoverLimits
	discoverLimits = func(_ context.Context, o provider.Options, _, _ string, _ *catalog.ModelsJSON) error {
		discovered = o.ModelData
		return nil
	}
	t.Cleanup(func() { discoverLimits = oldDiscover })
	if err := o.Discover(t.Context(), session, "openai", "hooked"); err != nil || discovered != session {
		t.Errorf("Discover handed %p (%v); want the session's document %p", discovered, err, session)
	}
}

// M16's reason in the singular (the ninth addition, R5's class): a section
// down to one model is "1 model".
func TestModelDataGuardWords_0072(t *testing.T) {
	have := cliParsed(t, cliModelDoc(map[string]int{"openai": 4}), provider.ModelDataStamp{Source: cliDataURL, Fetched: time.Now()})
	got := cliParsed(t, cliModelDoc(map[string]int{"openai": 1}), provider.ModelDataStamp{Source: cliDataURL, Fetched: time.Now()})
	if err := guardModelData(got, have, false); err == nil || err.Error() != "its openai section has 1 model, under half the copy's 4" {
		t.Errorf("refusal %v; want \"its openai section has 1 model, under half the copy's 4\"", err)
	}
}

// M46 (the ninth addition): an interrupt that arrives after the guards and
// before the copy's write writes nothing, and the attempt is canceled.
func TestModelDataSaveAfterCancel_0072(t *testing.T) {
	r := newDataRun(t, triggerReply())
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	old := modelDataGuarded
	modelDataGuarded = cancel // the interrupt lands once the download passed the guards
	t.Cleanup(func() { modelDataGuarded = old })
	out := r.e.refreshModelData(ctx, modelDataCommand, false)
	if _, err := os.Stat(r.copy); out.Result != attemptCanceled || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("result %s, copy %v; want canceled and no copy written", out.Result, err)
	}
	if a := r.lastAttempt(t); a.Result != attemptCanceled {
		t.Errorf("attempt %+v; want canceled", a)
	}
	t.Run("refusable", func(t *testing.T) {
		// The interrupt ends the fetch, whose download lacks a section the
		// copy has: canceled, not refused, which would gate the hour.
		r := newDataRun(t)
		r.seed(t, cliModelDoc(map[string]int{"openai": 3, "xai": 2}), cliDataURL)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		fetchModelData = func(fctx context.Context, _ string, _ provider.Options) (*provider.ModelData, bool, error) {
			cancel()
			<-fctx.Done()
			return stampedAgo(t, map[string]int{"openai": 3}, 0), false, nil
		}
		if out := r.e.refreshModelData(ctx, modelDataCommand, false); out.Result != attemptCanceled {
			t.Errorf("result %s (%s); want canceled", out.Result, out.Reason)
		}
	})
}

// M60 (internal/cli's half): a model the current copy lacks is unknown, with
// --model's usual custom-id warning (0072-MADR D19).
func TestSessionKeepsModelData_0072(t *testing.T) {
	withCopy(t, stampedAgo(t, map[string]int{"openai": 2}, time.Hour)) // no copy-only-1
	r := runMain(t, nil, "acp", "--model", "openai/copy-only-1")
	if r.code != ExitOK || !strings.Contains(r.stderr, `Model "copy-only-1" not found for provider "openai". Using custom model id.`) {
		t.Errorf("exit %d, stderr %q; want the custom-id warning", r.code, r.stderr)
	}
}
