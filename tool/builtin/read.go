package builtin

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/tool"
)

// maxLineChars is where read cuts a long line (opencode's limit).
const maxLineChars = 2000

type readIn struct {
	Path   string `json:"path" jsonschema:"the file or directory to read, relative to the working directory or absolute"`
	Offset int    `json:"offset,omitzero" jsonschema:"the first line (or entry) to show, counting from 1"`
	Limit  int    `json:"limit,omitzero" jsonschema:"how many lines (or entries) to show"`
}

// imageNotSent is read's note for an image: the provider SDK carries text
// only (0005-PLAN F1, owner's decision 3 of 2026-10-07).
const imageNotSent = "[The image was not sent: gobble cannot send images to models yet.]"

// Read is the read tool with the default options.
func Read() tool.Tool { return readWith(Options{}.withDefaults()) }

// readWith is the read tool (0005-MADR amendment of 2026-10-07, choice 3):
// numbered lines from an offset, bounded, with a footer saying how to go
// on; a directory's entries; a note for an image; and a refusal for other
// binary files. It may also read o.OutputDir, where long command output is
// kept, without asking.
func readWith(o Options) tool.Tool {
	return tool.New("read", description("read", o),
		func(ctx context.Context, in readIn, env tool.Env) (tool.Result, error) {
			name := shown(env, in.Path)
			ws := readWorkspace(env, o)
			abs, info, err := statFile(env, ws, in.Path)
			if err != nil {
				return tool.Result{}, err
			}
			if info.IsDir() {
				return readDir(ws, abs, name, in)
			}
			io := textIO{ctx: ctx, ws: ws}
			if !imagePath(abs) {
				io = textIOFor(ctx, env, ws, abs)
			}
			b, err := io.read(abs)
			if err != nil {
				return tool.Result{}, fmt.Errorf("read %s: %w", name, unwrapPath(err))
			}
			if mime := imageType(b); mime != "" {
				r := tool.TextResult(fmt.Sprintf("Read image file [%s]\n%s", mime, imageNotSent))
				r.Summary = summary(fmt.Sprintf("read %s (%s, not sent)", name, mime))
				return r, nil
			}
			if binaryContent(b) {
				return tool.Result{}, fmt.Errorf("read %s: a binary file; read cannot show it", name)
			}
			text, lines, err := numbered(string(b), in, name)
			if err != nil {
				return tool.Result{}, err
			}
			r := tool.TextResult(text)
			r.Summary = summary(fmt.Sprintf("read %s (%s)", name, count(lines, "line", "lines")))
			return r, nil
		},
		tool.WithKind(tool.KindRead),
		tool.WithSchema(func(s *jsonschema.Schema) { bound(s, "offset", 1, 0, 1); bound(s, "limit", 1, maxLines, maxLines) }),
		tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true, IdempotentHint: true}),
		tool.WithOutside(func(c tool.Call, env tool.Env) []string { return outsideOf(readWorkspace(env, o), env, argPath(c)) }),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("read " + shown(env, titlePath(c))) }),
		tool.WithLocations(readLocations),
	)
}

// imageExts are the extensions read takes as images without reading the
// file, so an image stays local when the client offers files: the types
// imageType sniffs (0005-PLAN F1d-3 deviation 1).
var imageExts = []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"}

func imagePath(abs string) bool {
	return slices.Contains(imageExts, strings.ToLower(filepath.Ext(abs)))
}

// readLocations is a read's file, at its offset when one is given.
func readLocations(c tool.Call, env tool.Env) []tool.Location {
	p := argPath(c)
	if p == "" {
		return nil
	}
	var a struct {
		Offset int `json:"offset"`
	}
	_ = json.Unmarshal(c.Args, &a) //nolint:errcheck // no offset is line 0, none
	return []tool.Location{{Path: resolve(env, p), Line: max(a.Offset, 0)}}
}

// statFile resolves the path a model named and stats it. A path that does
// not exist is tried in the other spellings macOS gives pasted screenshot
// names (fsx.ReadVariants), and then gets an error naming similar entries
// of its directory.
func statFile(env tool.Env, ws fsx.Workspace, path string) (string, fs.FileInfo, error) {
	abs := resolve(env, path)
	info, err := ws.Stat(abs)
	if errors.Is(err, fs.ErrNotExist) {
		for _, v := range fsx.ReadVariants(abs) {
			if vi, verr := ws.Stat(v); verr == nil {
				return v, vi, nil
			}
		}
		return "", nil, fmt.Errorf("read %s: no such file%s", shown(env, path), didYouMean(env, ws, abs))
	}
	if err != nil {
		return "", nil, fmt.Errorf("read %s: %w", shown(env, path), unwrapPath(err))
	}
	return abs, info, nil
}

// numbered is a text file's lines from in.Offset, each as "N: text", with a
// footer. A BOM and the CR of a CRLF are not shown, and a final newline does
// not count as a line. It also returns the file's line count.
func numbered(s string, in readIn, name string) (string, int, error) {
	lines := countingLines(strings.TrimPrefix(s, "\uFEFF"))
	total := len(lines)
	if total == 0 {
		return "(empty file)", 0, nil
	}
	first := max(in.Offset, 1)
	if first > total {
		return "", total, fmt.Errorf("read %s: offset %d is past the end of the file (%s)", name, in.Offset, count(total, "line", "lines"))
	}
	limit := maxLines
	if in.Limit > 0 {
		limit = min(in.Limit, maxLines)
	}
	var b strings.Builder
	last, byBytes := first-1, false
	for n := first; n <= total && n < first+limit; n++ {
		text, _ := truncateLine(strings.TrimSuffix(lines[n-1], "\r"), maxLineChars)
		line := strconv.Itoa(n) + ": " + text + "\n"
		if b.Len()+len(line) > maxBytes && n > first {
			byBytes = true
			break
		}
		b.WriteString(line)
		last = n
	}
	b.WriteString(footer("lines", first, last, total, byBytes))
	return b.String(), total, nil
}

// footer is the last line of a read: where the output stopped and the
// offset to continue from, or that the end was reached.
func footer(what string, first, last, total int, byBytes bool) string {
	switch {
	case last < total && byBytes:
		return fmt.Sprintf("(%s %d-%d of %d, %d KB limit; continue with offset=%d)", what, first, last, total, maxBytes/1024, last+1)
	case last < total:
		return fmt.Sprintf("(%s %d-%d of %d; continue with offset=%d)", what, first, last, total, last+1)
	case what == "lines":
		return "(end of file, " + count(total, "line", "lines") + ")"
	default:
		return "(end of directory, " + count(total, "entry", "entries") + ")"
	}
}
