package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// M7: on Windows, the copy's replace waits out a reader that holds the
// file, and its read waits out a replace under way, as fsx retries a
// sharing violation (0067-MADR D1, D4; 0072-MADR D2).
func TestModelDataHeldWindows_0072(t *testing.T) {
	path := filepath.Join(t.TempDir(), "models-dev.json")
	first := cliParsed(t, cliModelDoc(map[string]int{"openai": 1}), provider.ModelDataStamp{Source: "first"})
	if err := writeModelDataCopy(path, first); err != nil {
		t.Fatal(err)
	}
	t.Run("replace", func(t *testing.T) {
		reader, err := os.Open(path) //nolint:gosec // G304: the test's temp file
		if err != nil {
			t.Fatal(err)
		}
		time.AfterFunc(200*time.Millisecond, func() { _ = reader.Close() })
		next := cliParsed(t, cliModelDoc(map[string]int{"openai": 2}), provider.ModelDataStamp{Source: "next"})
		if err := writeModelDataCopy(path, next); err != nil {
			t.Fatalf("the replace under a reader held for 200 ms: %v", err)
		}
		if got, err := readModelDataCopy(path); err != nil || got.Stamp().Source != "next" {
			t.Fatalf("read %v (%v), want the new copy", got, err)
		}
	})
	t.Run("read", func(t *testing.T) {
		name, err := windows.UTF16PtrFromString(path)
		if err != nil {
			t.Fatal(err)
		}
		h, err := windows.CreateFile(name, windows.DELETE,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
			nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
		if err != nil {
			t.Fatal(err)
		}
		time.AfterFunc(200*time.Millisecond, func() { _ = windows.CloseHandle(h) })
		if got, err := readModelDataCopy(path); err != nil || got.Stamp().Source != "next" {
			t.Fatalf("the read under a replace held for 200 ms: %v (%v)", got, err)
		}
	})
}
