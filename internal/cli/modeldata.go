package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// gobble keeps its own copy of models.dev's document, fetched by the engine
// below and read back at the next start (0072-MADR D1 to D6, D18).

const (
	// modelDataVersion is the copy's and attempt.json's format.
	modelDataVersion = 1
	// modelDataLimit bounds the copy's file, as the SDK bounds a document.
	modelDataLimit = 32 << 20
	// staleTempAge is how old a temporary file a killed process left must be
	// before an accepted write removes it (D2).
	staleTempAge = time.Hour
	// The fetch's own deadlines: in the background, and for gobble models
	// refresh (D4; the owner's "Background 30 s, refresh cmd 120 s
	// (Recommended)").
	modelDataBackgroundWait = 30 * time.Second
	modelDataCommandWait    = 120 * time.Second
)

// modelDataPaths are the copy's file and attempt.json, in <cache>/model-data,
// a directory of its own, so no models.json server id names the same file
// (0072-MADR D1).
func modelDataPaths(dirs appdirs.Dirs) (copyPath, attemptPath string) {
	dir := filepath.Join(dirs.Cache, "model-data")
	return filepath.Join(dir, "models-dev.json"), filepath.Join(dir, "attempt.json")
}

// modelDataFile is the copy as the file holds it (0072-MADR D2): document is
// the SDK's Encode of the nine sections in their compact form, and sha256
// is over those bytes.
type modelDataFile struct {
	Version      int            `json:"version"`
	Source       string         `json:"source"`
	Fetched      string         `json:"fetched"`
	ETag         string         `json:"etag,omitzero"`
	LastModified string         `json:"lastModified,omitzero"`
	SHA256       string         `json:"sha256"`
	Sections     map[string]int `json:"sections"`
	Document     jsontext.Value `json:"document"`
}

// encodeModelData is d as the copy's file: one compact JSON object.
func encodeModelData(d *provider.ModelData) ([]byte, error) {
	doc := d.Document()
	sum := sha256.Sum256(doc)
	s := d.Stamp()
	f := modelDataFile{Version: modelDataVersion, Source: s.Source, Fetched: s.Fetched.UTC().Format(time.RFC3339),
		ETag: s.ETag, LastModified: s.LastModified, SHA256: hex.EncodeToString(sum[:]), Sections: d.Sections(), Document: doc}
	b, err := jsonx.MarshalStrict(f)
	if err != nil {
		return nil, fmt.Errorf("model data: encode the copy: %w", err)
	}
	return b, nil
}

// decodeModelData reads the copy's file: anything but a whole copy of this
// version, whose document matches its sha256 and is a document the SDK
// reads, is no copy, and the error says why (0072-MADR D3).
func decodeModelData(b []byte) (*provider.ModelData, error) {
	var f modelDataFile
	if err := jsonx.UnmarshalStrict(b, &f); err != nil {
		return nil, fmt.Errorf("not a copy: %w", err)
	}
	if f.Version != modelDataVersion {
		return nil, fmt.Errorf("version %d, not %d", f.Version, modelDataVersion)
	}
	if sum := sha256.Sum256(f.Document); hex.EncodeToString(sum[:]) != f.SHA256 {
		return nil, errors.New("its document does not match its sha256")
	}
	fetched, err := time.Parse(time.RFC3339, f.Fetched)
	if err != nil {
		return nil, fmt.Errorf("its fetched time: %w", err)
	}
	d, err := provider.ParseModelData(bytes.NewReader(f.Document))
	if err != nil {
		return nil, err
	}
	return d.Stamped(provider.ModelDataStamp{Source: f.Source, Fetched: fetched.UTC(), ETag: f.ETag, LastModified: f.LastModified}), nil
}

// readModelDataCopy is the copy at path: nil and an error that is
// fs.ErrNotExist when there is none, or nil and why the file is no copy
// (0072-MADR D3). Only a regular file is read, up to modelDataLimit, through
// fsx's Windows retries (0067-MADR D1).
func readModelDataCopy(path string) (*provider.ModelData, error) {
	b, err := fsx.ReadRegularFile(path, modelDataLimit)
	if err != nil {
		return nil, err
	}
	return decodeModelData(b)
}

// loadModelData reads the copy; TestMain replaces it with one that reads no
// file, so no test reads the host's cache (0072-MADR D10).
var loadModelData = readModelDataCopy

// fetchModelData fetches the document; TestMain replaces it with one that
// fails offline, so no test reaches models.dev (0072-MADR D10).
var fetchModelData = provider.FetchModelData

// modelDataSource is the URL the engine fetches; tests point it under
// .invalid (0072-MADR D10).
var modelDataSource = provider.ModelDataURL

// modelDataNow is the engine's clock; tests set it.
var modelDataNow = time.Now

// writeModelDataCopy writes d as the copy at path, through writeReplacing,
// so a reader reads the old copy or the new one whole, and the last writer
// wins; a failure leaves the old copy as it was. An accepted write then
// removes the temporary files a killed process left, an hour old or older
// (0072-MADR D2).
func writeModelDataCopy(path string, d *provider.ModelData) error {
	b, err := encodeModelData(d)
	if err != nil {
		return err
	}
	if err := writeReplacing(path, b); err != nil {
		return err
	}
	sweepModelDataTemps(filepath.Dir(path), modelDataNow())
	return nil
}

// writeReplacing writes data to path, its directory made private first,
// with internal/fsx's atomic writer: a temporary file beside it
// (.<name>.gobble-<hex>.tmp), synced, renamed over it with Windows'
// retries, and the directory synced; a new file is owner-only; a failure
// removes the temporary file (0072-MADR D2, its fifth addition).
func writeReplacing(path string, data []byte) error {
	if err := appdirs.EnsurePrivateDir(filepath.Dir(path)); err != nil {
		return err
	}
	return fsx.WriteFileMode(path, data, 0o600)
}

// modelDataTempPrefixes name the temporary files of the copy and of
// attempt.json, each .<name>.<anything>.tmp.
var modelDataTempPrefixes = []string{".models-dev.json.", ".attempt.json."}

// sweepModelDataTemps removes the temporary files of the copy and of
// attempt.json in dir last changed an hour or more before now; younger
// ones may be another process's write under way. The directory is read
// whole, so no character of its path is a pattern's.
func sweepModelDataTemps(dir string, now time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".tmp") || !slices.ContainsFunc(modelDataTempPrefixes, func(p string) bool { return strings.HasPrefix(name, p) }) {
			continue
		}
		if fi, err := e.Info(); err == nil && fi.Mode().IsRegular() && now.Sub(fi.ModTime()) >= staleTempAge {
			_ = os.Remove(filepath.Join(dir, name)) //nolint:errcheck // another process may have removed it; the next sweep tries again
		}
	}
}

// guardModelData is why got may not replace have, the copy, or nil
// (0072-MADR D6): a document with no section holding a model is refused
// always; against a copy from the same source, unless force, a section the
// copy has that got lacks, or in which got has fewer than half the copy's
// models. A section the SDK reads as absent (no models, or null) is lacking
// (M15, M78).
func guardModelData(got, have *provider.ModelData, force bool) error {
	sections := got.Sections()
	if len(sections) == 0 {
		return errors.New("it holds no models")
	}
	if force || have == nil || have.Stamp().Source != got.Stamp().Source {
		return nil
	}
	for key, n := range have.Sections() {
		switch m := sections[key]; {
		case m == 0:
			return fmt.Errorf("its %s section is missing", key)
		case 2*m < n:
			return fmt.Errorf("its %s section has %d models, under half the copy's %d", key, m, n)
		}
	}
	return nil
}

// modelDataAttempt is attempt.json: the last refresh any process started,
// written when it starts and when it ends, by rename, the last writer
// winning (0072-MADR D18).
type modelDataAttempt struct {
	Version  int    `json:"version"`
	Source   string `json:"source"`
	Started  string `json:"started"`
	Finished string `json:"finished,omitzero"`
	Result   string `json:"result"`
	Reason   string `json:"reason,omitzero"`
}

// The results attempt.json and a refresh report (0072-MADR D18).
const (
	attemptStarted     = "started"
	attemptOK          = "ok"
	attemptNotModified = "not_modified"
	attemptFailed      = "failed"
	attemptRefused     = "refused"
	attemptNotSaved    = "not_saved"
)

// writeModelDataAttempt writes a as attempt.json at path.
func writeModelDataAttempt(path string, a modelDataAttempt) error {
	a.Version = modelDataVersion
	b, err := jsonx.MarshalStrict(a)
	if err != nil {
		return err
	}
	return writeReplacing(path, b)
}

// readModelDataAttempt is attempt.json at path: false with fs.ErrNotExist
// when there is none, false and why when it is unreadable, which counts as
// no attempt (0072-MADR D18).
func readModelDataAttempt(path string) (modelDataAttempt, bool, error) {
	b, err := fsx.ReadRegularFile(path, 1<<16)
	if err != nil {
		return modelDataAttempt{}, false, err
	}
	var a modelDataAttempt
	if err := jsonx.UnmarshalStrict(b, &a); err != nil {
		return modelDataAttempt{}, false, fmt.Errorf("not an attempt: %w", err)
	}
	if a.Version != modelDataVersion || a.Started == "" || a.Result == "" {
		return modelDataAttempt{}, false, errors.New("not an attempt of this version")
	}
	return a, true, nil
}

// modelDataMode is how a refresh runs: in the background, or for gobble
// models refresh, which differ in the fetch's deadline (0072-MADR D4, M59).
type modelDataMode int

const (
	modelDataBackground modelDataMode = iota
	modelDataCommand
)

// wait is the mode's deadline for the fetch.
func (m modelDataMode) wait() time.Duration {
	if m == modelDataCommand {
		return modelDataCommandWait
	}
	return modelDataBackgroundWait
}

// modelDataOutcome is what one refresh did: its result, as attempt.json
// names it, the reason of a failure (its kind), a refusal or a failed save,
// the document it made current, and the copy it read first.
type modelDataOutcome struct {
	Result string
	Reason string
	Data   *provider.ModelData
	Copy   *provider.ModelData
}

// startModelDataRefresh starts one refresh in its own goroutine, on the
// run's listing group, so the run owns it and ends it (0046-MADR D2), and
// returns at once: no prompt, tool call or start-up waits for it, and
// readers take the current document without a lock (0072-MADR D18, M65). It
// reports false, starting nothing, once the run has ended its listings.
func (e *runEnv) startModelDataRefresh(mode modelDataMode, force bool) bool {
	ctx := e.listingCtx()
	return e.goListing(func() { e.refreshModelData(ctx, mode, force) })
}

// refreshModelData is one refresh, the engine's body (0072-MADR D4 to D6,
// D18): attempt.json marked started; the copy read again; one fetch under
// the mode's deadline, sending the copy's validators; the guards; the save;
// the document made current, saved or not; attempt.json marked with the
// result. A failure changes nothing on disk or in memory.
func (e *runEnv) refreshModelData(ctx context.Context, mode modelDataMode, force bool) modelDataOutcome {
	dirs, _, err := systemDirs()
	if err != nil {
		e.warnModelData(ctx, "dirs", "model data: no cache directory, so it was not refreshed", slog.String("error", err.Error()))
		return modelDataOutcome{Result: attemptFailed, Reason: "no_cache_dir"}
	}
	copyPath, attemptPath := modelDataPaths(dirs)
	source := modelDataSource
	started := modelDataNow().UTC().Format(time.RFC3339)
	e.recordModelDataAttempt(ctx, attemptPath, modelDataAttempt{Source: source, Started: started, Result: attemptStarted})
	out := e.fetchAndKeepModelData(ctx, mode, force, copyPath, source)
	e.recordModelDataAttempt(ctx, attemptPath, modelDataAttempt{Source: source, Started: started,
		Finished: modelDataNow().UTC().Format(time.RFC3339), Result: out.Result, Reason: out.Reason})
	return out
}

// fetchAndKeepModelData is refreshModelData between its two attempt.json
// writes.
func (e *runEnv) fetchAndKeepModelData(ctx context.Context, mode modelDataMode, force bool, copyPath, source string) modelDataOutcome {
	have := e.loadModelDataCopy(ctx, copyPath)
	out := modelDataOutcome{Copy: have}
	fetchCtx, cancel := context.WithTimeout(ctx, mode.wait())
	defer cancel()
	got, notModified, err := fetchModelData(fetchCtx, source,
		provider.Options{ClientName: appName, ClientVersion: identity().Version, ModelData: have})
	if err != nil {
		out.Result, out.Reason = attemptFailed, modelDataFailure(err)
		return out
	}
	out.Result = attemptOK
	if notModified {
		out.Result = attemptNotModified
	} else if err := guardModelData(got, have, force); err != nil {
		e.warnModelData(ctx, "refused:"+err.Error(), "model data: a download was refused, and the copy kept",
			slog.String("url", source), slog.String("reason", err.Error()))
		out.Result, out.Reason = attemptRefused, err.Error()
		return out
	}
	out.Data = got
	provider.SetCurrentModelData(got) // current for this run, saved or not (M69)
	if err := writeModelDataCopy(copyPath, got); err != nil {
		e.warnModelData(ctx, "not_saved", "model data: a download could not be saved; it is used for this run",
			slog.String("path", copyPath), slog.String("error", err.Error()))
		out.Result, out.Reason = attemptNotSaved, err.Error()
	}
	return out
}

// loadModelDataCopy is the copy at path, or nil: none, or a file that is no
// copy, which is logged once per run with the path and the reason, and left
// for the next accepted download to replace (0072-MADR D3).
func (e *runEnv) loadModelDataCopy(ctx context.Context, path string) *provider.ModelData {
	d, err := loadModelData(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		e.warnModelData(ctx, "unreadable:"+path, "model data: the copy cannot be read, so it is not used",
			slog.String("path", path), slog.String("reason", err.Error()))
		return nil
	}
	return d
}

// recordModelDataAttempt writes attempt.json; a failure is logged once, and
// the refresh goes on (0072-MADR D18, M69).
func (e *runEnv) recordModelDataAttempt(ctx context.Context, path string, a modelDataAttempt) {
	if err := writeModelDataAttempt(path, a); err != nil {
		e.warnModelData(ctx, "attempt", "model data: attempt.json cannot be written",
			slog.String("path", path), slog.String("error", err.Error()))
	}
}

// warnModelData logs text at warn once per run for key, as "model data"
// with text as its warning.
func (e *runEnv) warnModelData(ctx context.Context, key, text string, attrs ...slog.Attr) {
	e.modelDataMu.Lock()
	if e.modelDataWarned[key] {
		e.modelDataMu.Unlock()
		return
	}
	if e.modelDataWarned == nil {
		e.modelDataWarned = map[string]bool{}
	}
	e.modelDataWarned[key] = true
	e.modelDataMu.Unlock()
	e.log().LogAttrs(ctx, slog.LevelWarn, "model data", append([]slog.Attr{slog.String("warning", text)}, attrs...)...)
}

// modelDataFailure is a failed fetch's kind, as attempt.json's reason names
// it (0072-MADR D5, M11 to M13, M78, M82): timeout, canceled,
// redirect_downgrade (a redirect from https to http), offline, status, or
// not_document for a body over the limit or not models.dev's document.
func modelDataFailure(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, llm.ErrNotPermitted):
		return "redirect_downgrade"
	case errors.Is(err, llm.ErrOffline):
		return "offline"
	}
	if ae, ok := errors.AsType[*llm.APIError](err); ok && ae != nil {
		return "status"
	}
	return "not_document"
}
