package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// commandTimeout bounds a models.json !command, as Pi bounds one.
const commandTimeout = 10 * time.Second

// modelsJSONs are the models.json files read in this process, by path: each
// is read, and its diagnostics logged, once (0005-PLAN F4-2b).
var modelsJSONs = struct {
	sync.Mutex
	byPath map[string]modelsJSONRead
}{byPath: map[string]modelsJSONRead{}}

// modelsJSONRead is one read of a models.json: the file, or nil, and the
// error that made it nil, if any.
type modelsJSONRead struct {
	mj  *catalog.ModelsJSON
	err error
}

// modelsJSON is <config>/models.json, read once per process. No file, or a
// config directory that cannot be resolved, is none; a file that cannot be
// read or parsed is none, with one warning; each provider or field it skips
// is a warning.
func (e *runEnv) modelsJSON() *catalog.ModelsJSON { return e.modelsJSONOnce().mj }

// modelsJSONError is why models.json is none although it exists: it could
// not be read or parsed. No file is no error (0045-MADR D5).
func (e *runEnv) modelsJSONError() error { return e.modelsJSONOnce().err }

func (e *runEnv) modelsJSONOnce() modelsJSONRead {
	dirs, _, err := systemDirs()
	if err != nil {
		return modelsJSONRead{}
	}
	path := filepath.Join(dirs.Config, "models.json")
	modelsJSONs.Lock()
	defer modelsJSONs.Unlock()
	if read, ok := modelsJSONs.byPath[path]; ok {
		return read
	}
	mj, err := e.loadModelsJSON(path)
	read := modelsJSONRead{mj: mj, err: err}
	modelsJSONs.byPath[path] = read
	return read
}

func (e *runEnv) readModelsJSON(path string) *catalog.ModelsJSON {
	mj, err := e.loadModelsJSON(path)
	if err != nil {
		return nil // warned by loadModelsJSON
	}
	return mj
}

// loadModelsJSON is readModelsJSON, with the error of a file that exists but
// cannot be read or parsed; the error is also warned.
func (e *runEnv) loadModelsJSON(path string) (*catalog.ModelsJSON, error) {
	data, err := fsx.ReadRegularFile(path, fsx.ConfigLimit)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		e.log().Warn("models.json", slog.String("message", "cannot be read, so it is ignored: "+err.Error()), slog.String("file", path))
		return nil, err
	}
	mj, diags, err := catalog.ParseModelsJSON(data, provider.SDKVersion())
	if err != nil {
		e.log().Warn("models.json", slog.String("message", err.Error()+"; the file is ignored"), slog.String("file", path))
		return nil, err
	}
	for _, d := range diags {
		e.log().Warn("models.json", slog.String("message", d), slog.String("file", path))
	}
	return mj, nil
}

// runCommand runs a models.json !command through the bash tool's shell,
// bounded by commandTimeout, with no stdin and its stderr discarded, and
// returns its output; a non-zero exit is an error.
func (e *runEnv) runCommand(ctx context.Context, command string) (string, error) {
	bash, err := builtin.BashPath(builtinOptions(e.processSettings("").Settings))
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bash, "-c", command) //nolint:gosec // G204: the user's own models.json command
	cmd.WaitDelay = time.Second
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return "", fmt.Errorf("timed out after %s", commandTimeout)
	}
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// headersFor is provider id's Headers option from models.json: each
// request's model's headers, resolved for that request by Pi's grammar; nil
// when models.json gives the provider none.
func (e *runEnv) headersFor(mj *catalog.ModelsJSON, id string) func(context.Context, string) (map[string]string, error) {
	if !mj.HasHeaders(id) {
		return nil
	}
	return func(ctx context.Context, model string) (map[string]string, error) {
		raw := mj.Headers(id, model)
		out := make(map[string]string, len(raw))
		for name, v := range raw {
			run := func(command string) (string, error) { return e.runCommand(ctx, command) }
			resolved, err := catalog.ResolveValue(v, os.LookupEnv, run)
			if err != nil {
				return nil, fmt.Errorf("models.json: provider %s: header %q: %w", id, name, err)
			}
			out[name] = resolved
		}
		return out, nil
	}
}

// providerOptions are o with provider id's base URL and headers from
// models.json, its stored workspace (0035-MADR D5), and the run's SDK cache
// (0046-MADR D7).
func (e *runEnv) providerOptions(id string, o provider.Options) provider.Options {
	mj := e.modelsJSON()
	o.BaseURL = mj.BaseURL(id)
	o.Headers = e.headersFor(mj, id)
	o.Workspace = e.knownWorkspace(id)
	o.Cache = &e.cache
	return o
}

// withModelsJSON is ids with the models models.json defines for provider id
// after them, each once.
func withModelsJSON(ids []string, mj *catalog.ModelsJSON, id string) []string {
	for _, m := range mj.Models(id) {
		if !slices.Contains(ids, m) {
			ids = append(ids, m)
		}
	}
	return ids
}
