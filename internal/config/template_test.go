package config

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// keyLine is the shape of a commented-out key, as D8's case 2 finds one:
// "#", one space, the nesting indentation, the key, a colon and a value.
var keyLine = regexp.MustCompile(`^#( +)([a-z][A-Za-z]*):( .*)?$`)

// proseKey is a prose line that reads as "name: value", which D9 forbids.
var proseKey = regexp.MustCompile(`^#\s*[A-Za-z][A-Za-z0-9_.-]*:( |$)`)

// templateKey is one key line of the template.
type templateKey struct {
	line  int    // 1-based
	path  string // dotted, with its parents
	value string // after ": ", or "" for a group
}

// templateKeys reads the template's key lines, building each one's path
// from the key lines above it by their indentation.
func templateKeys(t *testing.T, text []byte) []templateKey {
	t.Helper()
	var keys []templateKey
	var stack []string
	for i, l := range strings.Split(string(text), "\n") {
		m := keyLine.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		indent := len(m[1]) - 1
		if indent%2 != 0 || indent/2 > len(stack) {
			t.Fatalf("template line %d: indentation %d does not nest: %q", i+1, indent, l)
		}
		stack = append(stack[:indent/2], m[2])
		keys = append(keys, templateKey{line: i + 1, path: strings.Join(stack, "."), value: strings.TrimPrefix(m[3], " ")})
	}
	return keys
}

// uncommented is the template with every key line made active.
func uncommented(text []byte) []byte {
	lines := strings.Split(string(text), "\n")
	for i, l := range lines {
		if keyLine.MatchString(l) {
			lines[i] = strings.TrimPrefix(l, "# ")
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// resolveText resolves text as the global file.
func resolveText(t *testing.T, text []byte) (Resolved, []Diagnostic) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.yaml"), text, 0o600); err != nil {
		t.Fatal(err)
	}
	return Resolve(Options{ConfigDir: dir})
}

// As shipped, the template is comments only: it sets nothing.
func TestTemplateParsesAsComments(t *testing.T) {
	root, ds := Parse("settings.yaml", Template())
	if root == nil || len(ds) != 0 || len(root.Content) != 0 {
		t.Fatalf("Parse(template) = %v, %v; want an empty mapping", root, ds)
	}
	if bytes.Contains(Template(), []byte("\r")) || bytes.Contains(Template(), []byte("\t")) {
		t.Fatal("the template holds a CR or a tab")
	}
}

// Every key has one block, in Keys' order, and each group line holds keys.
func TestTemplateHasEveryKeyOnce(t *testing.T) {
	var leaves []string
	for _, k := range templateKeys(t, Template()) {
		if k.value == "" {
			if !isGroup(k.path) {
				t.Errorf("template line %d: %s has no value and is not a group", k.line, k.path)
			}
			continue
		}
		leaves = append(leaves, k.path)
	}
	var want []string
	for _, k := range Keys {
		want = append(want, k.Path)
	}
	if !slices.Equal(leaves, want) {
		t.Fatalf("template keys %q, want Keys' %q", leaves, want)
	}
}

// Each key line shows its key's default, and "" for an empty string.
func TestTemplateDefaultsMatch(t *testing.T) {
	for _, tk := range templateKeys(t, Template()) {
		k, ok := KeyOf(tk.path)
		if !ok {
			continue
		}
		want := k.Default
		if want == "" {
			want = `""`
		}
		if tk.value != want {
			t.Errorf("template line %d: %s: %s, want the default %s", tk.line, tk.path, tk.value, want)
		}
	}
}

// With every key line uncommented, the file sets every key, to its default.
func TestTemplateUncommentedIsDefaults(t *testing.T) {
	r, _ := resolveText(t, uncommented(Template()))
	if !reflect.DeepEqual(r.Settings, Defaults()) {
		t.Fatalf("uncommented template = %+v, want Defaults()", r.Settings)
	}
	for _, k := range Keys {
		if o := r.Origins[k.Path]; o.Layer != LayerGlobal {
			t.Errorf("%s: origin %v, want the file", k.Path, o)
		}
	}
}

// Uncommented, the template reads without a diagnostic: no unknown key,
// no wrong type.
func TestTemplateHasNoUnknownKey(t *testing.T) {
	if _, ds := resolveText(t, uncommented(Template())); len(ds) != 0 {
		t.Fatalf("uncommented template: %v", ds)
	}
}

// Only key lines read as "name: value". A prose line of that form is one
// D8's case 2 could take for a key.
func TestTemplateProseIsNotAKey(t *testing.T) {
	for i, l := range strings.Split(string(Template()), "\n") {
		if m := keyLine.FindStringSubmatch(l); m != nil {
			continue
		}
		if proseKey.MatchString(l) {
			t.Errorf("template line %d reads as a key: %q", i+1, l)
		}
	}
	for _, k := range templateKeys(t, Template()) {
		if _, ok := KeyOf(k.path); !ok && !isGroup(k.path) {
			t.Errorf("template line %d: %s is neither a key nor a group", k.line, k.path)
		}
	}
}
