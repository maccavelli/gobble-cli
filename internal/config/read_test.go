package config

import (
	"strings"
	"testing"
)

// parseOne parses src and returns the one diagnostic it gives, failing
// unless the file is refused with exactly one.
func parseOne(t *testing.T, src string) Diagnostic {
	t.Helper()
	root, ds := Parse("s.yaml", []byte(src))
	if root != nil || len(ds) != 1 {
		t.Fatalf("Parse(%q) = %v, %v; want nil and one diagnostic", src, root, ds)
	}
	return ds[0]
}

// An empty file, and one of comments only, are an empty mapping.
func TestReadEmptyAndComments(t *testing.T) {
	for _, src := range []string{"", "\n\n", "# a comment\n# another\n", "# only\n\n# comments\n"} {
		root, ds := Parse("s.yaml", []byte(src))
		if root == nil || len(ds) != 0 || len(root.Content) != 0 {
			t.Fatalf("Parse(%q) = %v, %v; want an empty mapping", src, root, ds)
		}
	}
}

// A second document is refused at its line: yaml.v3's Unmarshal would read
// the first and drop it silently (0018-MADR D3).
func TestReadSecondDocument(t *testing.T) {
	d := parseOne(t, "steeringMode: all\n---\nfollowUpMode: all\n")
	if got := d.String(); got != "s.yaml:2: settings are one YAML document; a second one starts here" {
		t.Fatalf("diagnostic %q", got)
	}
}

// A repeated key is refused in yaml.v3's words, at the repeat, at any
// depth: decoding into a Node does not check it.
func TestReadDuplicateKey(t *testing.T) {
	d := parseOne(t, "mcp:\n  stopGraceMs: 1\n  stopGraceMs: 2\n")
	if got := d.String(); got != `s.yaml:3: mapping key "stopGraceMs" already defined at line 2` {
		t.Fatalf("diagnostic %q", got)
	}
}

// A tab in the indentation is a syntax error at its line.
func TestReadTab(t *testing.T) {
	d := parseOne(t, "mcp:\n\tstopGraceMs: 1\n")
	if d.Line != 2 || !strings.Contains(d.Text, "found character that cannot start any token") {
		t.Fatalf("diagnostic %q", d.String())
	}
}

// A syntax error is yaml.v3's message at its line, without yaml.v3's own
// "yaml: line N:" prefix.
func TestReadSyntax(t *testing.T) {
	d := parseOne(t, "a: 1\n  b: [\n")
	if d.Line == 0 || strings.Contains(d.Text, "yaml:") || strings.Contains(d.Text, "line ") {
		t.Fatalf("diagnostic %q: want the line in the location only", d.String())
	}
}

// A root that is not a mapping is refused at its line.
func TestReadNonMapping(t *testing.T) {
	for _, src := range []string{"- a\n- b\n", "just text\n", "~\n"} {
		d := parseOne(t, src)
		if d.Text != textNotMapping || d.Line != 1 {
			t.Fatalf("Parse(%q): %q", src, d.String())
		}
	}
}

// CRLF line endings read as LF.
func TestReadCRLF(t *testing.T) {
	src := strings.ReplaceAll("# c\nmcp:\n  stopGraceMs: 7\n", "\n", "\r\n")
	root, ds := Parse("s.yaml", []byte(src))
	if root == nil || len(ds) != 0 {
		t.Fatalf("Parse: %v %v", root, ds)
	}
	ls, lds := leaves("s.yaml", root, LayerGlobal)
	if len(lds) != 0 || ls["mcp.stopGraceMs"].value != int64(7) {
		t.Fatalf("leaves %v %v", ls, lds)
	}
}
