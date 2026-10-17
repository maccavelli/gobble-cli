package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/maccavelli/gobble-cli/internal/filelock"
)

// fixture reads testdata/edit/name.
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "edit", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// setEdit is an edit of path to text, parsed as Set parses it.
func setEdit(t *testing.T, path, text string) Edit {
	t.Helper()
	n, err := valueNode(path, text)
	if err != nil {
		t.Fatal(err)
	}
	return Edit{Path: path, Value: n}
}

// applyAll applies edits in turn.
func applyAll(t *testing.T, data []byte, edits ...Edit) []byte {
	t.Helper()
	for _, e := range edits {
		out, err := Apply("s.yaml", data, e)
		if err != nil {
			t.Fatalf("Apply %s: %v", e.Path, err)
		}
		data = out
	}
	return data
}

// wantBytes fails unless got is want, byte for byte.
func wantBytes(t *testing.T, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

type editCase struct {
	name, in, want string
	edits          []Edit
}

func runCases(t *testing.T, cases []editCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantBytes(t, applyAll(t, fixture(t, c.in), c.edits...), fixture(t, c.want))
		})
	}
}

// Case 1 replaces the value's own bytes: an inline comment, the spaces
// before it, a quoted value's quotes and an anchor are handled in place.
func TestEditActiveScalar(t *testing.T) {
	runCases(t, []editCase{
		{"top", "active.in.yaml", "active-top.want.yaml", []Edit{setEdit(t, "steeringMode", "one-at-a-time")}},
		{"quoted", "active.in.yaml", "active-quoted.want.yaml", []Edit{setEdit(t, "externalEditor", "code --wait")}},
		{"nested", "active.in.yaml", "active-nested.want.yaml", []Edit{setEdit(t, "mcp.stopGraceMs", "250")}},
		{"anchor", "active.in.yaml", "active-anchor.want.yaml", []Edit{setEdit(t, "mcp.stopTimeoutMs", "3000")}},
		// "key:" with nothing after it gains the value before its comment,
		// and an empty parent gains a child two spaces deeper (case 3).
		{"empty", "empty.in.yaml", "empty.want.yaml", []Edit{setEdit(t, "steeringMode", "all"), setEdit(t, "mcp.stopGraceMs", "5")}},
	})
}

// Case 2 uncomments the key's line, and its commented parents, and edits
// it as case 1; prose and sibling comments stay.
func TestEditCommentedKey(t *testing.T) {
	runCases(t, []editCase{
		{"top", "commented.in.yaml", "commented-top.want.yaml", []Edit{setEdit(t, "steeringMode", "all")}},
		{"nested", "commented.in.yaml", "commented-nested.want.yaml", []Edit{setEdit(t, "mcp.stopGraceMs", "250")}},
		{"under an active parent", "commented-active.in.yaml", "commented-active.want.yaml", []Edit{setEdit(t, "mcp.stopGraceMs", "7")}},
	})
}

// Every key of the shipped template is set through case 2, and only its
// lines change.
func TestEditEveryTemplateKey(t *testing.T) {
	for _, k := range Keys {
		out := applyAll(t, Template(), setEdit(t, k.Path, sample(k)))
		old, got := strings.Split(string(Template()), "\n"), strings.Split(string(out), "\n")
		if len(old) != len(got) {
			t.Fatalf("%s: %d lines, want %d", k.Path, len(got), len(old))
		}
		changed := 0
		for i := range old {
			if old[i] != got[i] {
				changed++
			}
		}
		if want := strings.Count(k.Path, ".") + 1; changed != want {
			t.Errorf("%s: %d lines changed, want %d", k.Path, changed, want)
		}
		r, ds := resolveText(t, out)
		if v, _ := r.Settings.Value(k.Path); len(ds) != 0 || r.Origins[k.Path].Layer != LayerGlobal || v == defaultValue(k) {
			t.Errorf("%s: read back %v from %v, %v", k.Path, v, r.Origins[k.Path], ds)
		}
	}
}

// sample is a valid value for k other than its default.
func sample(k Key) string {
	switch k.Kind {
	case KindBool:
		if k.Default == "true" {
			return "false"
		}
		return "true"
	case KindInt:
		return "4242"
	case KindEnum:
		return k.Enum[len(k.Enum)-1]
	case KindPath:
		return "~/sessions"
	}
	return "sample text"
}

// Case 3 inserts the key, and any missing parents, after the parent's last
// line at its children's indentation.
func TestEditAbsentKey(t *testing.T) {
	runCases(t, []editCase{
		{"top", "absent.in.yaml", "absent-top.want.yaml", []Edit{setEdit(t, "followUpMode", "all")}},
		{"parents", "absent.in.yaml", "absent-parents.want.yaml", []Edit{setEdit(t, "tools.bash.maxTimeoutMs", "1000")}},
		{"child", "absent.in.yaml", "absent-child.want.yaml", []Edit{setEdit(t, "mcp.stopTimeoutMs", "9")}},
	})
}

// Case 4 replaces a sequence's lines with the scalar, and writes a
// sequence value in block style.
func TestEditSequence(t *testing.T) {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{scalarNode("x"), scalarNode("y z")}}
	runCases(t, []editCase{
		{"to a scalar", "sequence.in.yaml", "sequence-scalar.want.yaml", []Edit{setEdit(t, "tools.bash.useClientTerminal", "true")}},
		{"to a block sequence", "sequence.in.yaml", "sequence-block.want.yaml", []Edit{{Path: "other", Value: seq}}},
	})
}

// Case 4 replaces a block scalar's lines.
func TestEditBlockScalar(t *testing.T) {
	runCases(t, []editCase{{"literal", "block.in.yaml", "block.want.yaml", []Edit{setEdit(t, "shellCommandPrefix", "set -euo pipefail")}}})
}

// Unsetting a template key restores its commented form at its
// indentation, and setting it again restores the file.
func TestUnsetTemplateKey(t *testing.T) {
	runCases(t, []editCase{
		{"top", "unset.in.yaml", "unset-top.want.yaml", []Edit{{Path: "steeringMode"}}},
		{"nested", "unset.in.yaml", "unset-nested.want.yaml", []Edit{{Path: "mcp.stopGraceMs"}}},
	})
	round := applyAll(t, fixture(t, "unset.in.yaml"), Edit{Path: "mcp.stopGraceMs"}, setEdit(t, "mcp.stopGraceMs", "100"),
		Edit{Path: "steeringMode"}, setEdit(t, "steeringMode", "all"))
	wantBytes(t, round, fixture(t, "unset.in.yaml"))
}

// A parent left with no active key is commented too, in the template's
// form, so a later set finds the key through case 2 and restores the file
// (0018-PLAN P3 deviation 7).
func TestUnsetLastChildCommentsParent(t *testing.T) {
	runCases(t, []editCase{
		{"mcp", "lastchild.in.yaml", "lastchild-mcp.want.yaml", []Edit{{Path: "mcp.stopGraceMs"}}},
		{"tools.bash", "lastchild.in.yaml", "lastchild-tools.want.yaml", []Edit{{Path: "tools.bash.maxTimeoutMs"}}},
	})
	in := fixture(t, "lastchild.in.yaml")
	wantBytes(t, applyAll(t, in, Edit{Path: "mcp.stopGraceMs"}, setEdit(t, "mcp.stopGraceMs", "100")), in)
	wantBytes(t, applyAll(t, in, Edit{Path: "tools.bash.maxTimeoutMs"}, setEdit(t, "tools.bash.maxTimeoutMs", "5")), in)
}

// An unknown key's lines are deleted, with a parent left empty: commented
// when it is a template group, deleted when not. An absent key is no edit.
func TestUnsetUnknownKey(t *testing.T) {
	runCases(t, []editCase{
		{"top", "unknown.in.yaml", "unknown-top.want.yaml", []Edit{{Path: "theme"}}},
		{"under a group", "unknown.in.yaml", "unknown-group.want.yaml", []Edit{{Path: "mcp.retries"}}},
		{"under a non-group", "unknown.in.yaml", "unknown-parent.want.yaml", []Edit{{Path: "custom.a"}}},
		{"absent", "unknown.in.yaml", "unknown.in.yaml", []Edit{{Path: "nothing.here"}}},
	})
}

// A string that reads back as another type is quoted: 1.10 is a number and
// yes a YAML 1.1 boolean.
func TestEditQuotesAmbiguous(t *testing.T) {
	runCases(t, []editCase{{"strings", "ambiguous.in.yaml", "ambiguous.want.yaml",
		[]Edit{setEdit(t, "externalEditor", "1.10"), setEdit(t, "shellCommandPrefix", "yes")}}})
	for text, want := range map[string]string{"all": "all", "": `""`, "a: b": `"a: b"`, "x #y": `"x #y"`, "on": `"on"`, "~": `"~"`, "-1": `"-1"`, "007": `"007"`} {
		if got, _ := renderScalar(scalarNode(text)); got != want {
			t.Errorf("renderScalar(%q) = %s, want %s", text, got, want)
		}
	}
}

// crlf is s with every LF made CRLF.
func crlf(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\n"), []byte("\r\n")) }

// A CRLF file keeps CRLF on every line, edited and inserted alike. The
// fixtures are LF on disk (.gitattributes), so the test makes them CRLF
// (0018-PLAN P3 deviation 6).
func TestEditCRLFKept(t *testing.T) {
	cases := []editCase{
		{"active", "active.in.yaml", "active-nested.want.yaml", []Edit{setEdit(t, "mcp.stopGraceMs", "250")}},
		{"commented", "commented.in.yaml", "commented-nested.want.yaml", []Edit{setEdit(t, "mcp.stopGraceMs", "250")}},
		{"inserted", "absent.in.yaml", "absent-parents.want.yaml", []Edit{setEdit(t, "tools.bash.maxTimeoutMs", "1000")}},
		{"unset", "lastchild.in.yaml", "lastchild-tools.want.yaml", []Edit{{Path: "tools.bash.maxTimeoutMs"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := crlf(fixture(t, c.in))
			if bytes.Count(in, []byte("\r\n")) == 0 {
				t.Fatal("the input is not CRLF")
			}
			wantBytes(t, applyAll(t, in, c.edits...), crlf(fixture(t, c.want)))
		})
	}
	// A file without a final newline gains one before an insertion.
	wantBytes(t, applyAll(t, []byte("a: 1\r\nb: 2"), setEdit(t, "steeringMode", "all")), []byte("a: 1\r\nb: 2\r\nsteeringMode: all\r\n"))
}

// Check refuses, in D7's words, what config set must exit 2 on; a
// global-only key only for a project's file.
func TestCheck(t *testing.T) {
	bad := []struct {
		path, value string
		layer       Layer
		want        string
	}{
		{"theme", "dark", LayerGlobal, `unknown setting "theme"`},
		{"apiKey", "x", LayerGlobal, "apiKey: credentials go in auth.json, not settings.yaml"},
		{"steeringMode", "some", LayerGlobal, "steeringMode: must be one of one-at-a-time, all"},
		{"mcp.stopGraceMs", "-1", LayerProject, "mcp.stopGraceMs: must be at least 1"},
		// zero would mean "the default" downstream (0018-PLAN P5 deviation 19)
		{"mcp.startupWaitMs", "0", LayerGlobal, "mcp.startupWaitMs: must be at least 1"},
		{"shellPath", "/bin/bash", LayerProject, "shellPath: only the global settings file can set this"},
		{"sessionDir", "rel", LayerGlobal, "sessionDir: must be an absolute path, or start with ~"},
	}
	for _, b := range bad {
		if err := Check(b.path, b.value, b.layer); err == nil || err.Error() != b.want {
			t.Errorf("Check(%s, %s, %v) = %v, want %q", b.path, b.value, b.layer, err, b.want)
		}
	}
	for _, g := range []struct {
		path, value string
		layer       Layer
	}{{"shellPath", "/bin/bash", LayerGlobal}, {"steeringMode", "all", LayerProject}, {"externalEditor", "", LayerProject}} {
		if err := Check(g.path, g.value, g.layer); err != nil {
			t.Errorf("Check(%s, %s, %v) = %v", g.path, g.value, g.layer, err)
		}
	}
}

// A parent the user set to an explicit null, which clears the layer below
// (D4), is not turned into a mapping (0018-PLAN P3 deviation 8).
func TestEditExplicitNullParentRefused(t *testing.T) {
	for _, src := range []string{"mcp: ~\n", "mcp: null\n", "mcp: !!null\n"} {
		_, err := Apply("s.yaml", []byte(src), setEdit(t, "mcp.stopGraceMs", "5"))
		if want := "s.yaml: the edit to mcp.stopGraceMs could not be applied safely; the file is unchanged"; err == nil || err.Error() != want {
			t.Errorf("%q: %v", src, err)
		}
	}
}

// An aliased value, or a path through an alias, is refused in D7's words.
func TestEditAliasRefused(t *testing.T) {
	for _, src := range []string{"base: &b 1\nmcp:\n  stopGraceMs: *b\n", "x: &m\n  stopGraceMs: 1\nmcp: *m\n"} {
		_, err := Apply("s.yaml", []byte(src), setEdit(t, "mcp.stopGraceMs", "2"))
		if err == nil || err.Error() != "mcp.stopGraceMs: its value is a YAML alias; edit the anchor" {
			t.Errorf("%q: %v", src, err)
		}
	}
}

// writeSettings writes data to a fresh settings file, dated an hour ago so
// a write shows in its modification time.
func writeSettings(t *testing.T, data []byte) (string, time.Time) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "settings.yaml")
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(file, past, past); err != nil {
		t.Fatal(err)
	}
	return file, past
}

// unchanged fails unless file still holds data with its old time.
func unchanged(t *testing.T, file string, data []byte, mtime time.Time) {
	t.Helper()
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) || !info.ModTime().Equal(mtime) {
		t.Fatalf("the file changed: %q at %v", got, info.ModTime())
	}
}

// A file that does not parse is refused with its diagnostic and never
// written.
func TestEditUnparsableRefused(t *testing.T) {
	data := []byte("mcp:\n\tstopGraceMs: 1\n")
	file, mtime := writeSettings(t, data)
	err := Set(file, "mcp.stopGraceMs", "2")
	if err == nil || !strings.Contains(err.Error(), file+":2: ") {
		t.Fatalf("Set = %v, want the diagnostic at line 2", err)
	}
	unchanged(t, file, data, mtime)
}

// A splice that changes another path fails the check, and nothing is
// written.
func TestEditCheckRefusesBadSplice(t *testing.T) {
	spliceHook = func(b []byte) []byte {
		return bytes.Replace(b, []byte("stopTimeoutMs: 9"), []byte("stopTimeoutMs: 8"), 1)
	}
	t.Cleanup(func() { spliceHook = nil })
	data := fixture(t, "unset.in.yaml")
	file, mtime := writeSettings(t, data)
	err := Set(file, "steeringMode", "one-at-a-time")
	if want := file + ": the edit to steeringMode could not be applied safely; the file is unchanged"; err == nil || err.Error() != want {
		t.Fatalf("Set = %v, want %q", err, want)
	}
	unchanged(t, file, data, mtime)
}

// Set and Unset wait for the lock, write the file in place, keep its mode,
// and refuse a missing file and a bad value.
func TestSetLocksAndWrites(t *testing.T) {
	data := fixture(t, "unset.in.yaml")
	file, mtime := writeSettings(t, data)
	release, err := filelock.TryLock(file + ".lock")
	if err != nil {
		t.Fatal(err)
	}
	var held *filelock.HeldError
	if err := Set(file, "steeringMode", "one-at-a-time"); !errors.As(err, &held) {
		t.Fatalf("Set while locked = %v, want the holder's error", err)
	}
	unchanged(t, file, data, mtime)
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if err := Set(file, "steeringMode", "one-at-a-time"); err != nil {
		t.Fatal(err)
	}
	if err := Unset(file, "mcp.stopGraceMs"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes(t, got, applyAll(t, data, setEdit(t, "steeringMode", "one-at-a-time"), Edit{Path: "mcp.stopGraceMs"}))
	if info, err := os.Stat(file); err != nil {
		t.Fatal(err)
	} else if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v: want the file's own 0600 kept", info.Mode())
	}
	if err := Set(filepath.Join(t.TempDir(), "none.yaml"), "steeringMode", "all"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Set on a missing file = %v", err)
	}
	for path, want := range map[string]string{
		"steeringMode=some":  "steeringMode: must be one of one-at-a-time, all",
		"theme=dark":         `unknown setting "theme"`,
		"apiKey=x":           "apiKey: credentials go in auth.json, not settings.yaml",
		"mcp.stopGraceMs=":   "mcp.stopGraceMs: want an integer, have an empty value",
		"mcp.stopGraceMs=-1": "mcp.stopGraceMs: must be at least 1",
	} {
		p, v, _ := strings.Cut(path, "=")
		if err := Set(file, p, v); err == nil || err.Error() != want {
			t.Errorf("Set %s = %v, want %q", path, err, want)
		}
	}
}
