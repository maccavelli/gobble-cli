package acpserver

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// An @path word is an @ at the start or after whitespace, then a quote or
// the characters up to the next whitespace; an @ inside a word, a lone @,
// and an empty or unterminated quote are none (0005-PLAN F3-4).
func TestMentionsIn(t *testing.T) {
	for text, want := range map[string][]mention{
		"@a.txt":               {{name: "a.txt"}},
		"see @a.txt, then":     {{name: "a.txt,"}},
		`open @"a b.txt" now`:  {{name: "a b.txt", quoted: true}},
		"x@a.txt":              nil,
		"@ alone":              nil,
		`@"" and @"open`:       nil,
		"one\n@b c\t@d":        {{name: "b"}, {name: "d"}},
		"mail me@x.org @e.txt": {{name: "e.txt"}},
	} {
		if got := mentionsIn(text); !slices.Equal(got, want) {
			t.Errorf("mentionsIn(%q) = %+v, want %+v", text, got, want)
		}
	}
}

// mentionDir is a cwd with the files the mention tests name.
func mentionDir(t *testing.T) string {
	t.Helper()
	cwd := t.TempDir()
	for name, data := range map[string][]byte{
		"a.txt":     []byte("A\n"),
		"a b.txt":   []byte("AB"),
		"end.txt":   []byte("E"),
		"empty.txt": {},
		"bin.dat":   {'a', 0, 'b'},
		"bad.txt":   {0xff, 'x'},
		"bom.txt":   append(slices.Clone(bom), 'h', 'i'),
	} {
		if err := os.WriteFile(filepath.Join(cwd, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(cwd, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	return cwd
}

// A named file is expanded once, in the order named, as Pi's file block,
// a byte-order mark dropped; trailing punctuation goes when the name
// without it is a file. A word inside a word, a missing file, a directory,
// a binary file, one not UTF-8, an empty file and a file outside the roots
// are left as typed.
func TestMentionFiles(t *testing.T) {
	cwd := mentionDir(t)
	outside := filepath.Join(t.TempDir(), "o.txt")
	if err := os.WriteFile(outside, []byte("O"), 0o600); err != nil {
		t.Fatal(err)
	}
	ws := fsx.Workspace{Roots: []string{cwd}}
	text := `read @a.txt and @"a b.txt" then @end.txt). Again @a.txt, x@bom.txt @missing.txt @sub @bin.dat @bad.txt @empty.txt @bom.txt @` + outside
	want := fileBlock(filepath.Join(cwd, "a.txt"), "A\n") + fileBlock(filepath.Join(cwd, "a b.txt"), "AB") +
		fileBlock(filepath.Join(cwd, "end.txt"), "E") + fileBlock(filepath.Join(cwd, "bom.txt"), "hi")
	if got := mentionFiles(text, cwd, ws, map[string]bool{}); got != want {
		t.Fatalf("mentionFiles:\n%s\nwant:\n%s", got, want)
	}
	if got := mentionFiles(`@"end.txt)"`, cwd, ws, map[string]bool{}); got != "" {
		t.Errorf("a quoted name kept its punctuation and still expanded: %q", got)
	}
	if got := fileBlock("/p", "t"); got != "<file name=\"/p\">\nt\n</file>\n" {
		t.Errorf("fileBlock = %q", got)
	}
}

// A prompt's files are its resource blocks and links in block order, then
// the files its unmarked text blocks name; a text block marked
// _meta.gobble.mentions false is not scanned (F3-4 deviation 1).
func TestPromptFiles(t *testing.T) {
	cwd := mentionDir(t)
	marked := acp.TextBlock("@end.txt")
	marked.Text.Meta = map[string]any{"gobble": map[string]any{"mentions": false}}
	blocks := []acp.ContentBlock{
		acp.TextBlock("look at @a.txt"),
		acp.ResourceBlock(acp.EmbeddedResourceResource{TextResourceContents: &acp.TextResourceContents{Uri: "file:///x.go", Text: "package x"}}),
		acp.ResourceBlock(acp.EmbeddedResourceResource{BlobResourceContents: &acp.BlobResourceContents{Uri: "file:///p.png", Blob: "AA==", MimeType: new("image/png")}}),
		acp.ResourceBlock(acp.EmbeddedResourceResource{BlobResourceContents: &acp.BlobResourceContents{Uri: "file:///b", Blob: "AA=="}}),
		acp.ResourceLinkBlock("spec", "https://example.com/spec"),
		marked,
	}
	want := fileBlock("file:///x.go", "package x") +
		"[binary resource file:///p.png (image/png) omitted]\n" +
		"[binary resource file:///b (unknown type) omitted]\n" +
		"spec: https://example.com/spec\n" +
		fileBlock(filepath.Join(cwd, "a.txt"), "A\n")
	if got := promptFiles(blocks, cwd, fsx.Workspace{Roots: []string{cwd}}); got != want {
		t.Fatalf("promptFiles:\n%s\nwant:\n%s", got, want)
	}
}

// Through the agent: the model gets the files first, then the text with
// its @path as typed, and the session's user entry holds both, so a
// resumed session gives the model what it had (0005-PLAN F3-4).
func TestPromptCarriesFiles(t *testing.T) {
	cwd, dir := mentionDir(t), t.TempDir()
	model := llmtest.NewScript(llmtest.Text("ok"))
	var logs bytes.Buffer
	c := promptAgent(t, dir, model, PromptOptions{}, &logs)
	id := newSession(t, c, cwd, nil).SessionId
	_, err := c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{
		acp.TextBlock("explain @a.txt"), acp.ResourceLinkBlock("spec", "https://example.com/spec")}})
	if err != nil {
		t.Fatal(err)
	}
	files := "spec: https://example.com/spec\n" + fileBlock(filepath.Join(cwd, "a.txt"), "A\n")
	if got := userTextsOf(model.Requests()[0]); !slices.Equal(got, []string{files, "explain @a.txt"}) {
		t.Fatalf("the user message %q", got)
	}
	var user string
	for _, m := range fileMessages(t, dir, id) {
		if strings.HasPrefix(m, "user: ") {
			user = m
		}
	}
	if !strings.Contains(user, files) || !strings.Contains(user, "explain @a.txt") {
		t.Fatalf("the session's user entry %q", user)
	}
}
