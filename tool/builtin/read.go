package builtin

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/tool"
)

const (
	// maxLineChars is where read cuts a long line (opencode's limit).
	maxLineChars = 2000
	// lineKeep is how much of a shown line read holds: maxLineChars
	// characters of the longest UTF-8 encoding, and one more, so
	// truncateLine still sees that the line is longer.
	lineKeep = (maxLineChars + 1) * utf8.UTFMax
	// readChunk is a streamed read's buffer, and the size of the chunks
	// the lines after the last one shown are counted in (0037-MADR D7).
	readChunk = 64 << 10
	// maxImageBytes is the most read loads of a file named as an image.
	maxImageBytes = 20 << 20
	// utf8BOM is the byte-order mark read does not show.
	utf8BOM = "\uFEFF"
)

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
			src, done, err := openText(ctx, env, ws, abs)
			if err != nil {
				return tool.Result{}, fmt.Errorf("read %s: %w", name, unwrapPath(err))
			}
			defer done() //nolint:errcheck // read-only
			head, err := src.Peek(sniffBytes)
			if err != nil && !errors.Is(err, io.EOF) {
				return tool.Result{}, fmt.Errorf("read %s: %w", name, unwrapPath(err))
			}
			if mime := imageType(head); mime != "" {
				r := tool.TextResult(fmt.Sprintf("Read image file [%s]\n%s", mime, imageNotSent))
				r.Summary = summary(fmt.Sprintf("read %s (%s, not sent)", name, mime))
				return r, nil
			}
			if binaryContent(head) {
				return tool.Result{}, fmt.Errorf("read %s: a binary file; read cannot show it", name)
			}
			text, lines, err := numbered(src, in, name)
			if err != nil {
				return tool.Result{}, err
			}
			// The context files below cwd this read reaches, and the
			// session has not had (0005-PLAN F3-1).
			var added []string
			if env.Instructions != nil {
				var b strings.Builder
				b.WriteString(text)
				for _, ins := range env.Instructions(abs) {
					b.WriteString("\n\n" + ins.Text)
					added = append(added, ins.Path)
				}
				text = b.String()
			}
			r := tool.TextResult(text)
			r.Summary = summary(fmt.Sprintf("read %s (%s)", name, count(lines, "line", "lines")))
			r.Instructions = added
			return r, nil
		},
		tool.WithKind(tool.KindRead),
		tool.WithSchema(func(s *jsonschema.Schema) { bound(s, "offset", 1, 0, 1); bound(s, "limit", 1, maxLines, maxLines) }),
		tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true, IdempotentHint: true}),
		tool.WithOutside(func(c tool.Call, env tool.Env) []string { return outsideOf(readWorkspace(env, o), env, argPath(c)) }),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("read " + shown(env, titlePath(c))) }),
		tool.WithLocations(readLocations),
		tool.WithSubject(pathSubject("path")),
	)
}

// imageExts are the extensions read takes as images without reading the
// file, so an image stays local when the client offers files: the types
// imageType sniffs (0005-PLAN F1d-3 deviation 1).
var imageExts = []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"}

func imagePath(abs string) bool {
	return slices.Contains(imageExts, strings.ToLower(filepath.Ext(abs)))
}

// openText opens the file abs for read, as a regular file (0037-MADR D7):
// an image whole, up to maxImageBytes; a file the client offers through it,
// whole, as the client returns it; and any other file streamed from disk.
// done releases it.
func openText(ctx context.Context, env tool.Env, ws fsx.Workspace, abs string) (src *bufio.Reader, done func() error, err error) {
	whole := func(b []byte, err error) (*bufio.Reader, func() error, error) {
		if err != nil {
			return nil, nil, err
		}
		return bufio.NewReader(bytes.NewReader(b)), func() error { return nil }, nil
	}
	if imagePath(abs) {
		return whole(ws.ReadRegular(abs, maxImageBytes))
	}
	if t := textIOFor(ctx, env, ws, abs); t.files != nil {
		return whole(t.read(abs))
	}
	f, err := ws.OpenRegular(abs)
	if err != nil {
		return nil, nil, err
	}
	return bufio.NewReaderSize(f, readChunk), f.Close, nil
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
	_ = jsonx.Unmarshal(c.Args, &a) //nolint:errcheck // no offset is line 0, none
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
// not count as a line. It also returns the file's line count. Only the
// lines shown are held; the rest are counted as they stream past
// (0037-MADR D7).
func numbered(src *bufio.Reader, in readIn, name string) (string, int, error) {
	if bom, err := src.Peek(len(utf8BOM)); err == nil && string(bom) == utf8BOM {
		src.Discard(len(utf8BOM)) //nolint:errcheck,gosec // peeked
	}
	first := max(in.Offset, 1)
	limit := maxLines
	if in.Limit > 0 {
		limit = min(in.Limit, maxLines)
	}
	var b strings.Builder
	n, last, byBytes := 0, first-1, false
	for n < first-1+limit {
		line, ok, err := nextLine(src, n+1 >= first)
		if err != nil {
			return "", 0, fmt.Errorf("read %s: %w", name, unwrapPath(err))
		}
		if !ok {
			break
		}
		n++
		if n < first {
			continue
		}
		text, _ := truncateLine(strings.TrimSuffix(line, "\r"), maxLineChars)
		out := strconv.Itoa(n) + ": " + text + "\n"
		if b.Len()+len(out) > maxBytes && n > first {
			byBytes = true
			break
		}
		b.WriteString(out)
		last = n
	}
	rest, err := countRest(src)
	if err != nil {
		return "", 0, fmt.Errorf("read %s: %w", name, unwrapPath(err))
	}
	total := n + rest
	if total == 0 {
		return "(empty file)", 0, nil
	}
	if first > total {
		return "", total, fmt.Errorf("read %s: offset %d is past the end of the file (%s)", name, in.Offset, count(total, "line", "lines"))
	}
	b.WriteString(footer("lines", first, last, total, byBytes))
	return b.String(), total, nil
}

// nextLine reads src's next line, without its newline, holding at most
// lineKeep bytes of it when keep is set and none otherwise. ok is false at
// the end, when there is no line left.
func nextLine(src *bufio.Reader, keep bool) (line string, ok bool, err error) {
	var held []byte
	for {
		frag, err := src.ReadSlice('\n')
		ok = ok || len(frag) > 0
		if keep && len(held) < lineKeep {
			held = append(held, frag[:min(len(frag), lineKeep-len(held))]...)
		}
		switch {
		case err == nil:
			return string(bytes.TrimSuffix(held, []byte("\n"))), true, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			return string(held), ok, nil
		default:
			return "", false, err
		}
	}
}

// countRest is the number of lines left in src: its newlines, and one more
// for a last line without one. It reads in chunks of readChunk bytes.
func countRest(src io.Reader) (int, error) {
	buf := make([]byte, readChunk)
	n, last := 0, byte('\n')
	for {
		k, err := src.Read(buf)
		if chunk := buf[:k]; len(chunk) > 0 {
			n += bytes.Count(chunk, []byte{'\n'})
			last = chunk[len(chunk)-1]
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	if last != '\n' {
		n++
	}
	return n, nil
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
