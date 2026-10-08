package builtin

import (
	"encoding/json/jsontext"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

type editCase struct {
	name, content string
	edits         []edit
	want          string // the file afterwards, or the error text when wantErr
	wantErr       bool
}

func runEdit(t *testing.T, content string, args any) (tool.Result, string) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	r := call(t, Edit(), tool.Env{Cwd: dir}, args)
	b, err := os.ReadFile(p) //nolint:gosec // test file
	if err != nil {
		t.Fatal(err)
	}
	return r, string(b)
}

const notFoundHint = " was not found; it must match the file exactly, or nearly (whitespace, quotes and indentation are forgiven)"

func checkEdit(t *testing.T, tc editCase) {
	t.Helper()
	r, file := runEdit(t, tc.content, map[string]any{"path": "f.txt", "edits": tc.edits})
	if tc.wantErr {
		if !r.IsError || r.Text() != tc.want {
			t.Fatalf("result = %q (isError %v), want the error %q", r.Text(), r.IsError, tc.want)
		}
		if file != tc.content {
			t.Fatalf("a failed edit changed the file to %q", file)
		}
		return
	}
	if r.IsError {
		t.Fatalf("edit failed: %s", r.Text())
	}
	if file != tc.want {
		t.Fatalf("file = %q, want %q", file, tc.want)
	}
}

// The behaviour cases first written from Pi's suite (packages/coding-agent/
// test/tools.test.ts), with gobble's messages.
func TestEditBehaviour(t *testing.T) {
	cases := []editCase{
		{name: "replace", content: "Hello, world!", edits: []edit{{"world", "testing"}}, want: "Hello, testing!"},
		{name: "not found", content: "Hello, world!", edits: []edit{{"nonexistent", "testing"}}, wantErr: true,
			want: "edit f.txt: oldText" + notFoundHint},
		{name: "duplicate", content: "foo foo foo", edits: []edit{{"foo", "bar"}}, wantErr: true,
			want: "edit f.txt: oldText occurs 3 times; include more surrounding lines so it occurs once"},
		{name: "disjoint", content: "alpha\nbeta\ngamma\ndelta\n", edits: []edit{{"alpha\n", "ALPHA\n"}, {"gamma\n", "GAMMA\n"}},
			want: "ALPHA\nbeta\nGAMMA\ndelta\n"},
		{name: "against the original", content: "foo\nbar\n", edits: []edit{{"foo\n", "bar\n"}, {"bar\n", "baz\n"}},
			want: "bar\nbaz\n"},
		{name: "overlap", content: "one\ntwo\nthree\n", edits: []edit{{"one\ntwo\n", "ONE\nTWO\n"}, {"two\nthree\n", "TWO\nTHREE\n"}}, wantErr: true,
			want: "edit f.txt: edits[0] and edits[1] overlap; merge them into one edit or choose separate regions"},
		{name: "nothing applied when one fails", content: "foo\nbar\n", edits: []edit{{"foo\n", "FOO\n"}, {"missing\n", "x\n"}}, wantErr: true,
			want: "edit f.txt: edits[1].oldText" + notFoundHint},
		{name: "empty oldText of several", content: "a", edits: []edit{{"a", "b"}, {"", "c"}}, wantErr: true,
			want: "edit f.txt: edits[1].oldText is empty"},
		{name: "no change", content: "same", edits: []edit{{"same", "same"}}, wantErr: true, want: "edit f.txt: the edit changes nothing"},
		{name: "no change, several", content: "a b", edits: []edit{{"a", "a"}, {"b", "b"}}, wantErr: true, want: "edit f.txt: the edits change nothing"},

		{name: "trailing whitespace", content: "line one   \nline two  \nline three\n", edits: []edit{{"line one\nline two\n", "replaced\n"}},
			want: "replaced\nline three\n"},
		{name: "fullwidth punctuation", content: "\u4F60\u597D\uFF0C\u4E16\u754C\n\u4F60\u597D\uFF08\u4E16\u754C\uFF09\n",
			edits: []edit{{"\u4F60\u597D,\u4E16\u754C\n\u4F60\u597D(\u4E16\u754C)\n", "\u4F60\u597D\uFF0Cpi\n\u4F60\u597D(pi)\n"}},
			want:  "\u4F60\u597D\uFF0Cpi\n\u4F60\u597D(pi)\n"},
		{name: "compatibility forms", content: "\uFF21\uFF22\uFF23\uFF11\uFF12\uFF13\ncafe\u0301\n", edits: []edit{{"ABC123\ncaf\u00E9\n", "XYZ789\ncoffee\n"}},
			want: "XYZ789\ncoffee\n"},
		{name: "smart single quotes", content: "console.log(\u2018hello\u2019);\n", edits: []edit{{"console.log('hello');", "console.log('world');"}},
			want: "console.log('world');\n"},
		{name: "smart double quotes", content: "const msg = \u201CHello World\u201D;\n", edits: []edit{{`const msg = "Hello World";`, `const msg = "Goodbye";`}},
			want: "const msg = \"Goodbye\";\n"},
		{name: "dashes", content: "range: 1\u20135\nbreak\u2014here\n", edits: []edit{{"range: 1-5\nbreak-here", "range: 10-50\nbreak--here"}},
			want: "range: 10-50\nbreak--here\n"},
		{name: "no-break space", content: "hello\u00A0world\n", edits: []edit{{"hello world", "hello universe"}},
			want: "hello universe\n"},
		{name: "exact before fuzzy", content: "const x = 'exact';\nconst y = 'other';\n", edits: []edit{{"const x = 'exact';", "const x = 'changed';"}},
			want: "const x = 'changed';\nconst y = 'other';\n"},
		{name: "duplicate after whitespace", content: "hello world   \nhello world\n", edits: []edit{{"hello world", "replaced"}}, wantErr: true,
			want: "edit f.txt: oldText occurs 2 times; include more surrounding lines so it occurs once"},
		{name: "fuzzy, several", content: "console.log(\u2018hello\u2019);\nhello\u00A0world\n",
			edits: []edit{{"console.log('hello');\n", "console.log('world');\n"}, {"hello world\n", "hello universe\n"}},
			want:  "console.log('world');\nhello universe\n"},
		{name: "fuzzy keeps the right occurrence", content: "replace me   \nafter   \n", edits: []edit{{"replace me\n", "after\n"}},
			want: "after\nafter   \n"},
		{name: "fuzzy keeps untouched lines",
			content: "keep before  \nfirst target  \nfirst after\nkeep middle   \nsecond target  \nsecond after\nkeep after  \n",
			edits:   []edit{{"first target\nfirst after", "FIRST\nFIRST2"}, {"second target\nsecond after", "SECOND\nSECOND2"}},
			want:    "keep before  \nFIRST\nFIRST2\nkeep middle   \nSECOND\nSECOND2\nkeep after  \n"},

		{name: "LF oldText in a CRLF file", content: "line one\r\nline two\r\nline three\r\n", edits: []edit{{"line two\n", "replaced line\n"}},
			want: "line one\r\nreplaced line\r\nline three\r\n"},
		{name: "CRLF kept", content: "first\r\nsecond\r\nthird\r\n", edits: []edit{{"second\n", "REPLACED\n"}},
			want: "first\r\nREPLACED\r\nthird\r\n"},
		{name: "LF kept", content: "first\nsecond\nthird\n", edits: []edit{{"second\n", "REPLACED\n"}},
			want: "first\nREPLACED\nthird\n"},
		{name: "duplicates across CRLF and LF", content: "hello\r\nworld\r\n---\r\nhello\nworld\n", edits: []edit{{"hello\nworld\n", "replaced\n"}}, wantErr: true,
			want: "edit f.txt: oldText occurs 2 times; include more surrounding lines so it occurs once"},
		{name: "BOM kept", content: "\uFEFFfirst\r\nsecond\r\nthird\r\n", edits: []edit{{"second\n", "REPLACED\n"}},
			want: "\uFEFFfirst\r\nREPLACED\r\nthird\r\n"},
		{name: "BOM and CRLF, several", content: "\uFEFFfirst\r\nsecond\r\nthird\r\nfourth\r\n", edits: []edit{{"second\n", "SECOND\n"}, {"fourth\n", "FOURTH\n"}},
			want: "\uFEFFfirst\r\nSECOND\r\nthird\r\nFOURTH\r\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { checkEdit(t, tc) })
	}
}

// opencode's forgiving strategies, each where the layers before it find
// nothing, and the guard against a forgiving match far larger than oldText.
func TestEditStrategies(t *testing.T) {
	cases := []editCase{
		{name: "line-trimmed: spaces for a tab", content: "func f() {\n\treturn 1\n}\n",
			edits: []edit{{"func f() {\n    return 1\n}\n", "func f() {\n\treturn 2\n}\n"}},
			want:  "func f() {\n\treturn 2\n}\n"},
		{name: "indentation-flexible: chooses between trimmed look-alikes",
			content: "\tif x {\n\t\ty()\n\t}\nif x {\ny()\n}\n",
			edits:   []edit{{"if x {\n\ty()\n}", "if x {\n\tz()\n}"}},
			want:    "if x {\n\tz()\n}\nif x {\ny()\n}\n"},
		{name: "escape-normalised: an escaped newline", content: "line one\nline two\n",
			edits: []edit{{`line one\nline two`, "lines one and two"}},
			want:  "lines one and two\n"},
		{name: "oversize guard", content: strings.Repeat(" ", 600) + "a\n" + strings.Repeat(" ", 600) + "b\n",
			edits: []edit{{"a\nb", "c"}}, wantErr: true,
			want: "edit f.txt: oldText matched a region much larger than itself; read the file again and resend the edit"},
		{name: "ambiguous at every layer", content: "  x\n    x\n", edits: []edit{{"\tx", "y"}}, wantErr: true,
			want: "edit f.txt: oldText occurs 2 times; include more surrounding lines so it occurs once"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { checkEdit(t, tc) })
	}
}

// A change to the file between the edit's read and its write fails the
// edit, and the change is kept.
func TestEditCompareAndSwap(t *testing.T) {
	t.Cleanup(func() { beforeWrite = nil })
	beforeWrite = func(abs string) {
		if err := os.WriteFile(abs, []byte("changed elsewhere\n"), 0o600); err != nil {
			t.Error(err)
		}
	}
	r, file := runEdit(t, "original\n", map[string]any{"path": "f.txt", "edits": []edit{{"original", "edited"}}})
	if want := "edit f.txt: the file changed while the edit was being made; read it again and resend the edit"; !r.IsError || r.Text() != want {
		t.Fatalf("result %q, want %q", r.Text(), want)
	}
	if file != "changed elsewhere\n" {
		t.Fatalf("file = %q; the other change was overwritten", file)
	}
}

// The three repairs of a malformed edit call, and the empty list.
func TestEditArgumentRepairs(t *testing.T) {
	cases := map[string]string{
		"edits as a JSON string":     `{"path":"f.txt","edits":"[{\"oldText\":\"a\",\"newText\":\"b\"}]"}`,
		"one edit as a JSON string":  `{"path":"f.txt","edits":"{\"oldText\":\"a\",\"newText\":\"b\"}"}`,
		"one edit object":            `{"path":"f.txt","edits":{"oldText":"a","newText":"b"}}`,
		"top-level oldText, newText": `{"path":"f.txt","oldText":"a","newText":"b"}`,
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			r, file := runEdit(t, "a\n", jsontext.Value(args))
			if r.IsError || file != "b\n" || r.Text() != "edited f.txt: 1 replacement" {
				t.Fatalf("result %q, file %q; want the edit applied", r.Text(), file)
			}
		})
	}
	r, _ := runEdit(t, "a\n", map[string]any{"path": "f.txt", "edits": []edit{}})
	if !r.IsError || r.Text() != "edit: edits needs at least one replacement" {
		t.Fatalf("empty edits = %q", r.Text())
	}
}

func TestEditMissingFile(t *testing.T) {
	r := call(t, Edit(), tool.Env{Cwd: t.TempDir()}, map[string]any{"path": "nope.txt", "edits": []edit{{"a", "b"}}})
	if !r.IsError || !strings.HasPrefix(r.Text(), "edit nope.txt: ") {
		t.Fatalf("result = %q", r.Text())
	}
}
