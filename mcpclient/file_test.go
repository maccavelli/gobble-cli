package mcpclient

import (
	"bytes"
	"encoding/json/jsontext"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const keptFile = `{
    "autoEnableCodemode": false,
    "mcpServers": {
        "a": {"command": "old-a"},
        "b": {"url": "https://example.com/mcp", "headers": {"Authorization": "Bearer ${TOKEN}"}}
    },
    "zzz": {"keep": [1, 2]}
}
`

const keptWant = `{
    "autoEnableCodemode": false,
    "mcpServers": {
        "a": {
            "command": "new-a"
        },
        "c": {
            "command": "c",
            "args": [
                "-y"
            ]
        }
    },
    "zzz": {
        "keep": [
            1,
            2
        ]
    }
}
`

// Add and Remove keep every other member, the servers' order and the
// file's indentation, as Pi's editMcpServers does.
func TestAddRemoveKeepTheFile(t *testing.T) {
	p := writeConfig(t, keptFile)
	replaced, err := Add(p, "a", jsontext.Value(`{"command":"new-a"}`))
	if err != nil || !replaced {
		t.Fatalf("Add a = %v, %v; want replaced", replaced, err)
	}
	replaced, err = Add(p, "c", jsontext.Value(`{"command":"c","args":["-y"]}`))
	if err != nil || replaced {
		t.Fatalf("Add c = %v, %v; want added", replaced, err)
	}
	removed, err := Remove(p, "b")
	if err != nil || !removed {
		t.Fatalf("Remove b = %v, %v", removed, err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != keptWant {
		t.Fatalf("file\n%s\nwant\n%s", got, keptWant)
	}
	before := got
	if removed, err := Remove(p, "nope"); err != nil || removed {
		t.Fatalf("Remove nope = %v, %v", removed, err)
	}
	if after, _ := os.ReadFile(p); !bytes.Equal(after, before) { //nolint:errcheck // compared below
		t.Fatal("Remove of an unknown server rewrote the file")
	}
}

func TestAddCreatesTheFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config", "mcp.json")
	if removed, err := Remove(p, "x"); err != nil || removed {
		t.Fatalf("Remove on a missing file = %v, %v", removed, err)
	}
	if _, err := Add(p, "x", jsontext.Value(`{"url":"https://example.com/mcp"}`)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"mcpServers\": {\n    \"x\": {\n      \"url\": \"https://example.com/mcp\"\n    }\n  }\n}\n"
	if string(got) != want {
		t.Fatalf("file %q, want %q", got, want)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("mode %v, %v", fi.Mode(), err)
		}
	}
}

func TestAddRefusesAMalformedFile(t *testing.T) {
	p := writeConfig(t, `{"mcpServers": []}`)
	if _, err := Add(p, "x", jsontext.Value(`{"command":"x"}`)); err == nil || err.Error() != `expected an object with an "mcpServers" object` {
		t.Fatalf("Add = %v", err)
	}
	if got, _ := os.ReadFile(p); string(got) != `{"mcpServers": []}` { //nolint:errcheck // compared
		t.Fatalf("file changed: %s", got)
	}
}
