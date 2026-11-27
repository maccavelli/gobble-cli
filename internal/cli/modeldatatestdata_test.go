package cli

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// testModelDataSections are the models per section of the 2026-10-14
// models.dev document, as the built-in copy's source.json counted them.
var testModelDataSections = map[string]int{"anthropic": 17, "google": 40, "huggingface": 81, "kilo": 390, "openai": 53,
	"opencode": 122, "opencode-go": 37, "togetherai": 29, "xai": 13}

// installTestModelData makes llm/provider's test data, the 2026-10-14
// models.dev document, the current document, with the zero stamp the
// built-in copy had before 0072-PLAN's Phase 3 removed it, so every test
// reads the facts it read before and no expected value changes (0072-MADR
// D14, M48). TestMain calls it. The file is found from this source file, so
// a child this binary starts in another directory finds it too.
func installTestModelData() error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("no source path for the test data")
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "llm", "provider", "testdata", "models-dev-2026-10-14.json"))
	if err != nil {
		return err
	}
	d, err := provider.ParseModelData(bytes.NewReader(b))
	if err != nil {
		return err
	}
	provider.SetCurrentModelData(d)
	return nil
}

// keepModelData restores, when t ends, the current document t found.
func keepModelData(t *testing.T) {
	t.Helper()
	found := provider.CurrentModelData()
	t.Cleanup(func() { provider.SetCurrentModelData(found) })
}

// M48: TestMain installed the test data: it is this binary's current
// document, unstamped, with the 2026-10-14 document's sections (0072-MADR
// D14).
func TestModelDataTestdataInstalled_0072(t *testing.T) {
	d := provider.CurrentModelData()
	if d == nil {
		t.Fatal("no current document: TestMain did not install the test data")
	}
	if got := d.Sections(); d.Stamp() != (provider.ModelDataStamp{}) || !maps.Equal(got, testModelDataSections) {
		t.Fatalf("current document stamped %+v with sections %v; want the test data's, unstamped: %v", d.Stamp(), got, testModelDataSections)
	}
}
