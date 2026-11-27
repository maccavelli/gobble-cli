package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The run's side of gobble's model data (0072-MADR D7, D8, D12, D13, D19,
// D20): the copy loaded at the run's start, the automatic refresh before a
// model call and in the models commands, the run's end, the warnings and
// the line /context and gobble models info show.

// modelDataExitWait bounds how long a headless run and the models commands
// wait at their end for a refresh under way, its write included, before
// they cancel it (0072-MADR D20). Tests shorten it.
var modelDataExitWait = 5 * time.Second

// modelDataGate is how long any process's attempt holds back every
// process's automatic refresh, whatever its result (0072-MADR D8, D18; "Any
// attempt gates an hour (Recommended)").
const modelDataGate = time.Hour

// unfinishedAttemptGate is how long an attempt left started, with no
// finished time, holds back every process's automatic refresh: the longest a
// download runs (the command's 120 s deadline, which covers the
// background's 30 s), so a process that died mid-download holds no one back
// for the hour ("Unfinished attempts gate only briefly (Recommended)",
// 0072-MADR M87).
const unfinishedAttemptGate = modelDataCommandWait

// modelDataOwner holds the run's refresh: its goroutine, the context it runs
// under, whether the run's end waits for it, and what keeps a second one
// from starting. mu guards the rest; once ended is set, group gains no
// goroutine (0072-MADR D18, D20; 0046-MADR D2).
type modelDataOwner struct {
	mu       sync.Mutex
	group    sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	ended    bool
	waits    bool      // the run's end waits up to modelDataExitWait
	inFlight bool      // one refresh per process at a time (M25)
	gate     time.Time // no automatic refresh before it (M26)
	// noneWarned is D13's warning, given at most once per run.
	noneWarned bool
	// settings are the two modelData keys, resolved once (D11).
	settingsOnce sync.Once
	settings     config.ModelData
}

// goModelData runs f on the run's refresh owner, with the owner's context,
// and reports false, running nothing, once the run has ended it.
func (e *runEnv) goModelData(f func(ctx context.Context)) bool {
	o := &e.modelData
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.ended {
		return false
	}
	if o.ctx == nil {
		o.ctx, o.cancel = context.WithCancel(e.ctxOrBackground()) // a signal ends it too
	}
	ctx := o.ctx
	o.group.Go(func() { f(ctx) })
	return true
}

// ctxOrBackground is the run's context, or Background for a runEnv a test
// built without one.
func (e *runEnv) ctxOrBackground() context.Context {
	if e.ctx == nil {
		return context.Background()
	}
	return e.ctx
}

// waitModelDataAtEnd makes the run's end wait for a refresh under way, up to
// modelDataExitWait: a headless run's and the models commands' (0072-MADR
// D20).
func (e *runEnv) waitModelDataAtEnd() {
	e.modelData.mu.Lock()
	e.modelData.waits = true
	e.modelData.mu.Unlock()
}

// endModelData ends the run's refresh: none starts after it; a headless run
// and the models commands wait for one under way up to modelDataExitWait,
// its write and fsx's Windows retries included; then it is cancelled.
// gobble acp and the interactive CLI cancel it at once and do not wait for
// a write under way: at worst a temporary file is left, which the next
// accepted write sweeps (0072-MADR D20, D2; "acp/interactive cancel at once
// (Recommended)").
func (e *runEnv) endModelData() {
	o := &e.modelData
	o.mu.Lock()
	if o.ended {
		o.mu.Unlock()
		return
	}
	o.ended = true
	waits, cancel := o.waits, o.cancel
	o.mu.Unlock()
	if waits && cancel != nil {
		done := make(chan struct{})
		go func() {
			o.group.Wait()
			close(done)
		}()
		t := time.NewTimer(modelDataExitWait)
		select {
		case <-done:
		case <-t.C:
		}
		t.Stop()
	}
	if cancel != nil {
		cancel()
	}
}

// modelDataSettings are the two modelData keys, global only, so the same
// for every working directory; resolved once, their diagnostics left to the
// process settings' report (0072-MADR D11).
func (e *runEnv) modelDataSettings() config.ModelData {
	o := &e.modelData
	o.settingsOnce.Do(func() {
		o.settings = e.resolveSettings("", config.ReadProcess, func(string, ...any) {}).Settings.ModelData
	})
	return o.settings
}

// loadModelDataAtStart reads the copy once, at the run's start, before any
// read of model facts, whatever the command, and makes it current; it never
// fetches (0072-MADR D7, M20; "Load the copy first (Recommended)").
func (e *runEnv) loadModelDataAtStart(ctx context.Context) {
	dirs, _, err := systemDirs()
	if err != nil {
		return
	}
	copyPath, _ := modelDataPaths(dirs)
	if d := e.loadModelDataCopy(ctx, copyPath); d != nil {
		provider.SetCurrentModelData(d)
	}
}

// loadCompletionModelData is completion mode's load: the copy, read once
// and made current, never fetched, nothing written or logged (0072-MADR D7,
// D8, M29).
func loadCompletionModelData() {
	dirs, _, err := systemDirs()
	if err != nil {
		return
	}
	copyPath, _ := modelDataPaths(dirs)
	if d, err := loadModelData(copyPath); err == nil && d != nil {
		provider.SetCurrentModelData(d)
	}
}

// modelDataDue reports whether a copy stamped s needs downloading at now
// with refreshHours: none (the zero stamp), one from another source, one
// fetched after now by this clock, or one as old as refreshHours (0072-MADR
// D8, M31).
func modelDataDue(s provider.ModelDataStamp, now time.Time, refreshHours int64) bool {
	switch {
	case s.Source != modelDataSource:
		return true
	case s.Fetched.After(now):
		return true
	}
	return now.Sub(s.Fetched) >= time.Duration(refreshHours)*time.Hour
}

// startAutoModelData is the automatic refresh's trigger. TestMain replaces
// it with one that starts nothing, so a test's model call starts no
// refresh whose writes outlive its run; the trigger's own tests restore it
// (0072-PLAN's Phase 2 entry; 0072-MADR D10).
var startAutoModelData = (*runEnv).autoRefreshModelData

// refreshModelDataIfDue starts the automatic refresh when it is due: before
// a model call, and in gobble models list and info (0072-MADR D8, M24;
// "Only model calls and models cmds (Recommended)").
func (e *runEnv) refreshModelDataIfDue() { startAutoModelData(e) }

// autoRefreshModelData starts one refresh in the background when the
// current copy is due and refreshHours is not 0, no refresh of this process
// is under way, and no attempt holds back this process; it returns at once
// (0072-MADR D8, D18, M25, M30, M65).
func (e *runEnv) autoRefreshModelData() {
	hours := e.modelDataSettings().RefreshHours
	now := modelDataNow()
	if hours == 0 || !modelDataDue(provider.CurrentModelData().Stamp(), now, hours) {
		return // pinned, or fresh: no request at all
	}
	o := &e.modelData
	o.mu.Lock()
	if o.inFlight || now.Before(o.gate) {
		o.mu.Unlock()
		return
	}
	o.inFlight = true
	o.mu.Unlock()
	started := e.goModelData(func(ctx context.Context) {
		defer e.modelDataIdle()
		e.autoRefresh(ctx, hours)
	})
	if !started {
		e.modelDataIdle()
	}
}

// modelDataIdle marks the process's refresh ended.
func (e *runEnv) modelDataIdle() {
	e.modelData.mu.Lock()
	e.modelData.inFlight = false
	e.modelData.mu.Unlock()
}

// holdModelData holds back this process's automatic refresh until t.
func (e *runEnv) holdModelData(t time.Time) {
	e.modelData.mu.Lock()
	if t.After(e.modelData.gate) {
		e.modelData.gate = t
	}
	e.modelData.mu.Unlock()
}

// autoRefresh is the automatic refresh's body (0072-MADR D8, D18): the copy
// read again, and a copy another process wrote that is not due made current
// with nothing fetched (M27); then attempt.json read, and an attempt any
// process started within the last hour skips it (M26), read before this
// refresh writes its own, a cancelled one not counted (M84), an unfinished
// one only for 120 s (M87); then the
// refresh, which gates the hour whatever its result (M67). A failed one with no data in use is D13's warning, once
// per run (M63).
func (e *runEnv) autoRefresh(ctx context.Context, hours int64) {
	dirs, _, err := systemDirs()
	if err != nil {
		e.warnModelData(ctx, "dirs", "model data: no cache directory, so it was not refreshed", slog.String("error", err.Error()))
		return
	}
	copyPath, attemptPath := modelDataPaths(dirs)
	have := e.loadModelDataCopy(ctx, copyPath)
	now := modelDataNow()
	if have != nil && !modelDataDue(have.Stamp(), now, hours) {
		provider.SetCurrentModelData(have)
		return
	}
	if a, ok := e.readAttempt(ctx, attemptPath); ok && a.Result != attemptCanceled { // a cancel gates nothing (M84)
		window := modelDataGate
		if a.Finished == "" { // unfinished: a download under way, or a process that died (M87)
			window = unfinishedAttemptGate
		}
		if st, err := time.Parse(time.RFC3339, a.Started); err == nil && !st.After(now) && now.Sub(st) < window {
			e.holdModelData(st.Add(window))
			return
		}
	}
	e.holdModelData(now.Add(modelDataGate))
	out := e.runModelDataRefresh(ctx, modelDataBackground, false, copyPath, attemptPath, have)
	if out.Result == attemptFailed && provider.CurrentModelData().Stamp().Source == "" {
		if text, ok := e.noneWarning(out.Reason); ok {
			e.log().LogAttrs(ctx, slog.LevelWarn, "model data", slog.String("warning", text))
		}
	}
}

// readAttempt is attempt.json at path; an unreadable one is no attempt,
// logged once (0072-MADR D18, M66).
func (e *runEnv) readAttempt(ctx context.Context, path string) (modelDataAttempt, bool) {
	a, ok, err := readModelDataAttempt(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		e.warnModelData(ctx, "attempt-unreadable", "model data: attempt.json cannot be read, so it is not counted",
			slog.String("path", path), slog.String("reason", err.Error()))
	}
	return a, ok
}

// lastAttempt is attempt.json as it is now, for the warnings, or false.
func (e *runEnv) lastAttempt(ctx context.Context) (modelDataAttempt, bool) {
	dirs, _, err := systemDirs()
	if err != nil {
		return modelDataAttempt{}, false
	}
	_, attemptPath := modelDataPaths(dirs)
	return e.readAttempt(ctx, attemptPath)
}

// The warnings' words (0072-MADR D12, D13).
const (
	noModelDataText = "no model data: models are unknown; gobble models refresh downloads them"
	lastFailedText  = "; the last download failed: "
)

// noneWarning is D13's warning with the failure's kind, and true, when it
// was not yet given this run, which it now is.
func (e *runEnv) noneWarning(kind string) (string, bool) {
	o := &e.modelData
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.noneWarned {
		return "", false
	}
	o.noneWarned = true
	if kind == "" {
		return noModelDataText, true
	}
	return noModelDataText + lastFailedText + kind, true
}

// modelDataWarning is the warning a run gives at its start, or "": with a
// copy past warnDays (and not ahead of this clock), its age, with the last
// attempt's refusal and, for one --force passes, --force, or its failure's
// kind, a cancelled attempt named neither way (M84); with none (before
// 0072-PLAN's Phase 3, the built-in copy in use), D13's, when the run is
// pinned or the last attempt failed, and none before a first attempt
// (0072-MADR D12, D13, M39, M40, M63).
func (e *runEnv) modelDataWarning(ctx context.Context, now time.Time) string {
	s := e.modelDataSettings()
	st := provider.CurrentModelData().Stamp()
	a, tried := e.lastAttempt(ctx)
	if st.Source == "" {
		failed := tried && a.Result == attemptFailed
		if s.RefreshHours != 0 && !failed {
			return ""
		}
		kind := ""
		if failed {
			kind = a.Reason
		}
		text, _ := e.noneWarning(kind)
		return text
	}
	if s.WarnDays == 0 || now.Sub(st.Fetched) < days(s.WarnDays) { // a copy ahead of this clock is younger than any
		return ""
	}
	text := fmt.Sprintf("model data was fetched %s ago from %s; gobble models refresh fetches it again",
		plural(int64(now.Sub(st.Fetched)/(24*time.Hour)), "day"), sourceName(st.Source))
	switch {
	case tried && a.Result == attemptRefused:
		text += "; the last download was refused: " + a.Reason
		if a.Reason != errNoModels.Error() { // --force never saves a document with no models (M68)
			text += "; gobble models refresh --force accepts it"
		}
	case tried && a.Result == attemptFailed:
		text += lastFailedText + a.Reason
	}
	return text
}

// warnModelDataAtStart gives the start's warning once: on stderr, or, under
// gobble acp, whose stdio is the protocol, at warn in the log (0072-MADR
// D12, D13; M39's acp_log_once).
func (e *runEnv) warnModelDataAtStart(acp bool) {
	text := e.modelDataWarning(e.ctxOrBackground(), modelDataNow())
	switch {
	case text == "":
	case acp:
		e.log().LogAttrs(e.ctxOrBackground(), slog.LevelWarn, "model data", slog.String("warning", text))
	default:
		e.out.Warnf("%s", text)
	}
}

// days is n days.
func days(n int64) time.Duration { return time.Duration(n) * 24 * time.Hour }

// sourceName is the host of a document's source, as the lines name it:
// models.dev.
func sourceName(source string) string {
	if u, err := url.Parse(source); err == nil && u.Host != "" {
		return u.Host
	}
	return source
}

// modelDataTime is a stamp's time as the lines write it, in UTC to the
// minute.
func modelDataTime(t time.Time) string { return t.UTC().Format("2006-01-02 15:04") }

// modelDataLine is the line for d, the document a session or a command
// reads, at now: its source, when it was fetched and how long ago, with
// D12's advice once it is warnDays old; ahead of this clock; or none
// (before 0072-PLAN's Phase 3, the built-in copy in use) (0072-MADR D12,
// M37).
func modelDataLine(d *provider.ModelData, now time.Time, warnDays int64) string {
	st := d.Stamp()
	if st.Source == "" {
		return "Model data: none; models are unknown until gobble models refresh downloads them"
	}
	when := modelDataTime(st.Fetched) + " UTC"
	if st.Fetched.After(now) {
		return "Model data: " + sourceName(st.Source) + ", fetched " + when + " (ahead of this clock)"
	}
	age := now.Sub(st.Fetched)
	line := "Model data: " + sourceName(st.Source) + ", fetched " + when + " (" + ageWords(age) + " ago)"
	if warnDays > 0 && age >= days(warnDays) {
		line += "; gobble models refresh fetches it again"
	}
	return line
}

// ageWords is an age as the line says it: minutes under an hour, hours
// under two days, else days.
func ageWords(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "under a minute"
	case d < time.Hour:
		return plural(int64(d/time.Minute), "minute")
	case d < 48*time.Hour:
		return plural(int64(d/time.Hour), "hour")
	}
	return plural(int64(d/(24*time.Hour)), "day")
}

// plural is n units, "1 unit" in the singular.
func plural(n int64, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// currentModelDataLine is the line for the current document.
func (e *runEnv) currentModelDataLine() string {
	return modelDataLine(provider.CurrentModelData(), modelDataNow(), e.modelDataSettings().WarnDays)
}

// currentCopy is the current document when it is a copy, nil while there is
// none (before 0072-PLAN's Phase 3, the built-in copy, whose stamp is the
// zero one): what a new ACP session takes (0072-MADR D19, M60, M61).
func currentCopy() *provider.ModelData {
	if d := provider.CurrentModelData(); d.Stamp().Source != "" {
		return d
	}
	return nil
}

// refreshing is p, whose model calls first start the automatic refresh
// when it is due; a Prober stays one (0072-MADR D8, M24).
func (e *runEnv) refreshing(p llm.Provider) llm.Provider {
	r := refreshingProvider{Provider: p, e: e}
	if pr, ok := p.(llm.Prober); ok {
		return refreshingProber{refreshingProvider: r, pr: pr}
	}
	return r
}

// refreshingBuild is build, its providers refreshing.
func (e *runEnv) refreshingBuild(build func(*provider.ModelData, string, llm.Retry) (llm.Provider, error)) func(*provider.ModelData, string, llm.Retry) (llm.Provider, error) {
	if build == nil {
		return nil
	}
	return func(data *provider.ModelData, sessionID string, retry llm.Retry) (llm.Provider, error) {
		p, err := build(data, sessionID, retry)
		if err != nil || p == nil {
			return p, err
		}
		return e.refreshing(p), nil
	}
}

// refreshingFor is build, a switched session's builder, its providers
// refreshing.
func (e *runEnv) refreshingFor(build func(*provider.ModelData, string, string, llm.Retry) (llm.Provider, error)) func(*provider.ModelData, string, string, llm.Retry) (llm.Provider, error) {
	if build == nil {
		return nil
	}
	return func(data *provider.ModelData, id, sessionID string, retry llm.Retry) (llm.Provider, error) {
		return e.refreshingBuild(func(d *provider.ModelData, sid string, r llm.Retry) (llm.Provider, error) {
			return build(d, id, sid, r)
		})(data, sessionID, retry)
	}
}

// refreshingProvider starts the automatic refresh, when due, before each
// model call it makes.
type refreshingProvider struct {
	llm.Provider
	e *runEnv
}

func (p refreshingProvider) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	p.e.refreshModelDataIfDue()
	return p.Provider.Stream(ctx, req)
}

// refreshingProber is a refreshingProvider over a Prober.
type refreshingProber struct {
	refreshingProvider
	pr llm.Prober
}

func (p refreshingProber) Probe(ctx context.Context) error { return p.pr.Probe(ctx) }
