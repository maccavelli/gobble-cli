package acpserver

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// The user message's files (0005-PLAN F3-4): what a prompt's resource
// blocks carry, and what the @path words of its text name, each written as
// Pi's CLI writes a file, ahead of the text.

// mentionPunct may follow an @path in prose, and is dropped when the name
// without it is a file and the name with it is not.
const mentionPunct = ".,;:!?)]}"

// bom is the UTF-8 byte-order mark, dropped from a file as Pi's stripBom
// drops it.
var bom = []byte{0xEF, 0xBB, 0xBF}

// fileBlock is a file as Pi's CLI puts it in a message
// (cli/file-processor.ts:78).
func fileBlock(name, text string) string {
	return `<file name="` + name + `">` + "\n" + text + "\n</file>\n"
}

// mention is one @path word: the name after the @, and whether it was
// quoted, which keeps any trailing punctuation.
type mention struct {
	name   string
	quoted bool
}

// mentionsIn are text's @path words: an @ at the start or after
// whitespace, followed by a "…" quote or by characters up to the next
// whitespace. An unterminated or empty quote, and an @ inside a word, are
// none.
func mentionsIn(text string) []mention {
	var out []mention
	afterSpace := true
	for i := 0; i < len(text); {
		r, w := utf8.DecodeRuneInString(text[i:])
		if r != '@' || !afterSpace {
			afterSpace = unicode.IsSpace(r)
			i += w
			continue
		}
		afterSpace = false
		rest := text[i+1:]
		if quoted, ok := strings.CutPrefix(rest, `"`); ok {
			end := strings.IndexByte(quoted, '"')
			if end > 0 {
				out = append(out, mention{name: quoted[:end], quoted: true})
				i += 1 + 1 + end + 1
				continue
			}
			i += w
			continue
		}
		end := strings.IndexFunc(rest, unicode.IsSpace)
		if end < 0 {
			end = len(rest)
		}
		if end > 0 {
			out = append(out, mention{name: rest[:end]})
		}
		i += w + end
	}
	return out
}

// mentionFiles are the blocks of the files text's @path words name, in
// the order named, each once across seen. A name is resolved against cwd
// and confined to ws's roots; it is expanded only when it names a regular
// file of valid UTF-8 with no NUL byte, and an empty file gives nothing,
// as Pi's CLI skips one. Anything else is left as typed.
func mentionFiles(text, cwd string, ws fsx.Workspace, seen map[string]bool) string {
	var b strings.Builder
	for _, m := range mentionsIn(text) {
		abs, ok := mentionPath(m, cwd, ws)
		if !ok || seen[abs] {
			continue
		}
		data, err := ws.ReadFile(abs)
		if err != nil || len(data) == 0 || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			continue
		}
		seen[abs] = true
		b.WriteString(fileBlock(abs, string(bytes.TrimPrefix(data, bom))))
	}
	return b.String()
}

// mentionPath is the regular file a mention names inside ws's roots,
// dropping trailing punctuation from an unquoted name while the name with
// it is no such file.
func mentionPath(m mention, cwd string, ws fsx.Workspace) (string, bool) {
	name := m.name
	for {
		abs := fsx.Resolve(cwd, name)
		if info, err := ws.Stat(abs); err == nil && info.Mode().IsRegular() {
			return abs, true
		}
		if m.quoted || name == "" || !strings.ContainsRune(mentionPunct, rune(name[len(name)-1])) {
			return "", false
		}
		name = name[:len(name)-1]
	}
}

// unscanned reports whether a text block is marked _meta.gobble.mentions
// false: input the client has already handled, such as the CLI's stdin and
// attached files (0005-PLAN F3-4 deviation 1).
func unscanned(b *acp.ContentBlockText) bool {
	g, _ := b.Meta[gobble].(map[string]any) //nolint:errcheck // absent or not an object is none
	v, ok := g["mentions"].(bool)
	return ok && !v
}

// promptFiles are a prompt's files, ahead of its text: each embedded
// resource and resource link in block order, then the files the @path
// words of its unmarked text blocks name. An image or audio block is left
// out, as gobble advertises neither.
func promptFiles(blocks []acp.ContentBlock, cwd string, ws fsx.Workspace) string {
	var b strings.Builder
	for _, c := range blocks {
		switch {
		case c.Resource != nil:
			b.WriteString(resourceText(c.Resource.Resource))
		case c.ResourceLink != nil:
			b.WriteString(c.ResourceLink.Name + ": " + c.ResourceLink.Uri + "\n")
		}
	}
	seen := map[string]bool{}
	for _, c := range blocks {
		if c.Text != nil && !unscanned(c.Text) {
			b.WriteString(mentionFiles(c.Text.Text, cwd, ws, seen))
		}
	}
	return b.String()
}

// resourceText is an embedded resource in the message: a text resource as
// a file block named by its URI, a blob as Pi names an omitted one
// (packages/mcp/src/protocol/content.ts:91-97).
func resourceText(r acp.EmbeddedResourceResource) string {
	switch {
	case r.TextResourceContents != nil:
		return fileBlock(r.TextResourceContents.Uri, r.TextResourceContents.Text)
	case r.BlobResourceContents != nil:
		mime := "unknown type"
		if m := r.BlobResourceContents.MimeType; m != nil && *m != "" {
			mime = *m
		}
		return "[binary resource " + r.BlobResourceContents.Uri + " (" + mime + ") omitted]\n"
	}
	return ""
}
