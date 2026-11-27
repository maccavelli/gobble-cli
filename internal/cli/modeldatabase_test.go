package cli

import (
	"bytes"
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// These tests use only names the tree had before 0072-PLAN's Phase 2, so
// each is red on its assertion there (M20, M38).

// copyOnlyDoc is a models.dev document whose openai section holds a model
// no built-in list names, which takes tools.
const copyOnlyDoc = `{"openai":{"models":{"copy-only-1":{"name":"Copy Only One","tool_call":true,` +
	`"limit":{"context":64000,"output":4000}}}}}`

// withCopy makes the copy reader return d, counting its reads, and leaves
// no current document behind; the fetch counts its calls and fails.
func withCopy(t *testing.T, d *provider.ModelData) (loads, fetches *atomic.Int64) {
	t.Helper()
	loads, fetches = &atomic.Int64{}, &atomic.Int64{}
	oldLoad, oldFetch := loadModelData, fetchModelData
	loadModelData = func(string) (*provider.ModelData, error) {
		loads.Add(1)
		if d == nil {
			return noModelDataCopy("")
		}
		return d, nil
	}
	fetchModelData = func(ctx context.Context, url string, o provider.Options) (*provider.ModelData, bool, error) {
		fetches.Add(1)
		return offlineModelData(ctx, url, o)
	}
	t.Cleanup(func() {
		loadModelData, fetchModelData = oldLoad, oldFetch
		provider.SetCurrentModelData(nil)
	})
	return loads, fetches
}

// M20: the copy is loaded once, at the run's start, before any read of
// model facts: --model naming a model only the copy knows resolves with no
// custom-id warning, and completion offers it; nothing is fetched
// (0072-MADR D7; "Load the copy first (Recommended)").
func TestModelDataLoadedFirst_0072(t *testing.T) {
	d := cliParsed(t, []byte(copyOnlyDoc), provider.ModelDataStamp{Source: provider.ModelDataURL, Fetched: time.Now().UTC().Truncate(time.Second)})
	t.Run("model_flag", func(t *testing.T) {
		loads, fetches := withCopy(t, d)
		r := runMain(t, nil, "acp", "--model", "openai/copy-only-1")
		if r.code != ExitOK || strings.Contains(r.stderr, "Using custom model id.") {
			t.Errorf("exit %d, stderr %q; want --model to resolve from the copy, with no custom-id warning", r.code, r.stderr)
		}
		if loads.Load() != 1 || fetches.Load() != 0 {
			t.Errorf("%d loads and %d fetches; want the copy loaded once and nothing fetched", loads.Load(), fetches.Load())
		}
	})
	t.Run("models_info", func(t *testing.T) {
		withCopy(t, d)
		r := runMain(t, nil, "models", "info", "openai/copy-only-1")
		if r.code != ExitOK || strings.Contains(r.stderr, "Using custom model id.") || !strings.Contains(r.stdout, "name      Copy Only One\n") {
			t.Errorf("exit %d, stdout %q, stderr %q; want the copy's facts", r.code, r.stdout, r.stderr)
		}
	})
	t.Run("completion", func(t *testing.T) {
		loads, fetches := withCopy(t, d)
		t.Setenv("GOBBLE_HOME", t.TempDir())
		t.Setenv(complete.EnvShell, "fish")
		t.Setenv(complete.EnvProtocol, complete.Protocol)
		var out, errw bytes.Buffer
		code := Main(t.Context(), []string{"--", "--model", "openai/copy-o"}, nil, &out, &errw)
		if code != ExitOK || !strings.Contains(out.String(), "openai/copy-only-1") {
			t.Errorf("exit %d, stdout %q; want completion to offer the copy's model", code, out.String())
		}
		if loads.Load() != 1 || fetches.Load() != 0 {
			t.Errorf("%d loads and %d fetches; want the copy loaded once and nothing fetched", loads.Load(), fetches.Load())
		}
	})
}

// M38: gobble models info's last line is the model data's, in /context's
// words: a copy's source, time and age, or none (0072-MADR D12).
func TestModelsInfoData_0072(t *testing.T) {
	fetched := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Second)
	d := cliParsed(t, []byte(copyOnlyDoc), provider.ModelDataStamp{Source: provider.ModelDataURL, Fetched: fetched})
	for name, tc := range map[string]struct {
		copy *provider.ModelData
		want string
	}{
		"copy": {d, "data      models.dev, fetched " + fetched.Format("2006-01-02 15:04") + " UTC (3 hours ago)\n"},
		"none": {nil, "data      none; models are unknown until gobble models refresh downloads them\n"},
	} {
		t.Run(name, func(t *testing.T) {
			withCopy(t, tc.copy)
			r := runMain(t, nil, "models", "info", "openai/copy-only-1")
			if r.code != ExitOK || !strings.HasSuffix(r.stdout, tc.want) {
				t.Errorf("exit %d, stdout %q; want it to end %q", r.code, r.stdout, tc.want)
			}
		})
	}
}
