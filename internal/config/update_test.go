package config

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// templateRuns are the template's runs of non-blank lines, found here
// independently of update.go, and whether each holds a key line.
func templateRuns(t *testing.T) (runs []string, keyed []bool) {
	t.Helper()
	text := strings.TrimSuffix(string(Template()), "\n")
	runs = strings.Split(text, "\n\n")
	if strings.Join(runs, "\n\n")+"\n" != string(Template()) || strings.Contains(text, "\n\n\n") {
		t.Fatal("the template's runs are not separated by single blank lines")
	}
	for _, r := range runs {
		has := false
		for l := range strings.SplitSeq(r, "\n") {
			if m := keyLine.FindStringSubmatch(l); m != nil && m[3] != "" {
				has = true
			}
		}
		keyed = append(keyed, has)
	}
	return runs, keyed
}

// runUpdate runs Update on data in a fresh file and returns the new file.
func runUpdate(t *testing.T, data []byte) ([]byte, []string) {
	t.Helper()
	file, _ := writeSettings(t, data)
	added, _, err := Update(file, false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return out, added
}

// The template with any one key block deleted is the template again after
// Update: the block goes back after the nearest earlier block, or after its
// section's heading, set off by blank lines.
func TestUpdateAddsMissingBlocks(t *testing.T) {
	runs, keyed := templateRuns(t)
	for i, r := range runs {
		if !keyed[i] {
			continue
		}
		without := strings.Join(slices.Delete(slices.Clone(runs), i, i+1), "\n\n") + "\n"
		out, added := runUpdate(t, []byte(without))
		if !bytes.Equal(out, Template()) {
			t.Errorf("block %d (%.40q): got\n%s", i, r, out)
		}
		if len(added) == 0 {
			t.Errorf("block %d: no key reported added", i)
		}
	}
}

// A block missing one of several keys gets that key's line back beside its
// siblings, without its prose; under an active parent too.
func TestUpdatePartialBlock(t *testing.T) {
	lines := strings.Split(string(Template()), "\n")
	runs, _ := templateRuns(t)
	leavesIn := func(line string) int {
		for _, r := range runs {
			if strings.Contains(r+"\n", line+"\n") {
				n := 0
				for l := range strings.SplitSeq(r, "\n") {
					if m := keyLine.FindStringSubmatch(l); m != nil && m[3] != "" {
						n++
					}
				}
				return n
			}
		}
		return 0
	}
	tried := 0
	for i, l := range lines {
		if m := keyLine.FindStringSubmatch(l); m == nil || m[3] == "" || leavesIn(l) < 2 {
			continue
		}
		tried++
		without := strings.Join(slices.Delete(slices.Clone(lines), i, i+1), "\n")
		out, added := runUpdate(t, []byte(without))
		if !bytes.Equal(out, Template()) || len(added) != 1 {
			t.Errorf("line %d %q: added %v, got\n%s", i+1, l, added, out)
		}
	}
	if tried < 5 {
		t.Fatalf("only %d partial cases", tried)
	}
	out, _ := runUpdate(t, []byte("mcp:\n  stopTimeoutMs: 3\n"))
	if !strings.HasPrefix(string(out), "mcp:\n#   startupWaitMs: 10000\n#   stopGraceMs: 500\n  stopTimeoutMs: 3\n") {
		t.Fatalf("under an active parent:\n%s", out)
	}
}

// A file without the template's sections gets each heading once, with its
// blocks, at the end; every key is then present, the settings read the
// same, and a second Update adds nothing.
func TestUpdateMissingSection(t *testing.T) {
	out, added := runUpdate(t, []byte("steeringMode: all\n"))
	if len(added) != len(Keys)-1 || slices.Contains(added, "steeringMode") {
		t.Fatalf("added %v", added)
	}
	for _, h := range []string{"# Sessions", "# Compaction", "# Tools", "# Editor"} {
		if n := strings.Count(string(out), "\n"+h+"\n"); n != 1 {
			t.Errorf("heading %q %d times", h, n)
		}
	}
	if !strings.HasPrefix(string(out), "steeringMode: all\n\n# ---") {
		t.Fatalf("the old line moved:\n%s", out)
	}
	r, ds := resolveText(t, out)
	want := Defaults()
	want.SteeringMode = "all"
	if len(ds) != 0 || !reflect.DeepEqual(r.Settings, want) {
		t.Fatalf("%+v %v", r.Settings, ds)
	}
	again, more := runUpdate(t, out)
	if len(more) != 0 || !bytes.Equal(again, out) {
		t.Fatalf("a second Update added %v", more)
	}
}

// A dry run reports the lines it would add and writes nothing.
func TestUpdateDryRunWritesNothing(t *testing.T) {
	data := []byte("steeringMode: all\n")
	file, mtime := writeSettings(t, data)
	added, preview, err := Update(file, true)
	if err != nil || len(added) == 0 {
		t.Fatalf("Update = %v, %v", added, err)
	}
	unchanged(t, file, data, mtime)
	pl := strings.Split(strings.TrimSuffix(string(preview), "\n"), "\n")
	if len(pl) < len(Keys) {
		t.Fatalf("preview %d lines", len(pl))
	}
	for _, l := range pl {
		if !strings.HasPrefix(l, "+") {
			t.Fatalf("preview line %q", l)
		}
	}
	out, _ := runUpdate(t, data)
	if want := string(out[len(data):]); strings.ReplaceAll("\n"+string(preview), "\n+", "\n")[1:] != want {
		t.Fatalf("the preview is not what Update adds:\n%s\nvs\n%s", preview, want)
	}
}

// Every line of a user's file is kept, in order and with its ending.
func TestUpdateKeepsEveryLine(t *testing.T) {
	data := []byte("# my settings\r\ntheme: dark   # not ours\r\nmcp:\r\n  # mine\r\n  stopGraceMs: 7\r\n\r\ncustom:\r\n  a: [1, 2]\r\n")
	out, added := runUpdate(t, data)
	if len(added) == 0 {
		t.Fatal("nothing added")
	}
	old := strings.SplitAfter(string(data), "\r\n")
	got := strings.SplitAfter(string(out), "\n")
	j := 0
	for _, o := range old {
		for j < len(got) && got[j] != o {
			j++
		}
		if j == len(got) {
			t.Fatalf("line %q is gone, or out of order:\n%s", o, out)
		}
		j++
	}
	if bytes.Count(out, []byte("\n")) != bytes.Count(out, []byte("\r\n")) {
		t.Fatal("an added line does not end in CRLF")
	}
	r, ds := resolveText(t, out)
	if r.Settings.MCP.StopGraceMs != 7 || len(ds) != 2 {
		t.Fatalf("%+v %v: want stopGraceMs 7 and the two unknown keys reported", r.Settings, ds)
	}
}

// Init writes the template, creating its directory, and refuses a file
// that exists.
func TestInitRefusesExisting(t *testing.T) {
	file := filepath.Join(t.TempDir(), "proj", ".gobble", "settings.yaml")
	if err := Init(file); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(got, Template()) {
		t.Fatalf("Init wrote %d bytes, %v", len(got), err)
	}
	existing, mtime := writeSettings(t, []byte("steeringMode: all\n"))
	if err := Init(existing); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Init over a file = %v", err)
	}
	unchanged(t, existing, []byte("steeringMode: all\n"), mtime)
}
