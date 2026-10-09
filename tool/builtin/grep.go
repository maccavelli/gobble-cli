package builtin

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"regexp/syntax"
	"strings"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/tool"
)

const (
	defaultGrepLimit = 100
	maxGrepLimit     = 1000
	maxGrepContext   = 10
	maxGrepLineChars = 500     // Pi's grep line cap (0005-MADR, grep row)
	maxGrepFileBytes = 8 << 20 // larger files are skipped, and counted
	binarySniffBytes = 8000    // git's binary test reads this much
)

type grepIn struct {
	Pattern    string `json:"pattern" jsonschema:"an RE2 regular expression, or plain text when literal is set"`
	Path       string `json:"path,omitzero" jsonschema:"the directory or the file to search, relative to the working directory or absolute; the working directory when not given"`
	Glob       string `json:"glob,omitzero" jsonschema:"only the files this glob matches, such as *.go"`
	IgnoreCase bool   `json:"ignoreCase,omitzero" jsonschema:"match without regard to case"`
	Literal    bool   `json:"literal,omitzero" jsonschema:"treat pattern as plain text"`
	Context    int    `json:"context,omitzero" jsonschema:"lines to show before and after each match"`
	Limit      int    `json:"limit,omitzero" jsonschema:"the most matches to show"`
}

// Grep is the grep tool: a regular expression over the files under a
// directory, or in one file, with the repository's ignore rules (0005-PLAN,
// entry "F1c made executable").
func Grep() tool.Tool {
	return tool.New("grep", description("grep", Options{}.withDefaults()),
		runGrep,
		tool.WithKind(tool.KindSearch),
		tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true, IdempotentHint: true}),
		tool.WithSchema(func(s *jsonschema.Schema) {
			nonEmpty(s, "pattern")
			bound(s, "context", 0, maxGrepContext, 0)
			bound(s, "limit", 1, maxGrepLimit, defaultGrepLimit)
			property(s, "ignoreCase").Default = []byte("false")
			property(s, "literal").Default = []byte("false")
		}),
		tool.WithOutside(outsidePath),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string {
			return title("grep " + argString(c, "pattern") + " in " + shown(env, orDot(argPath(c))))
		}),
	)
}

func runGrep(ctx context.Context, in grepIn, env tool.Env) (tool.Result, error) {
	re, err := grepPattern(in)
	if err != nil {
		return tool.Result{}, err
	}
	var glob *fsx.Glob
	if in.Glob != "" {
		g, err := fsx.CompileGlob(in.Glob)
		if err != nil {
			return tool.Result{}, fmt.Errorf("grep: the glob %q is malformed; check its [ ] classes and { } groups", in.Glob)
		}
		glob = &g
	}
	s, err := openSearch(env, "grep", in.Path)
	if err != nil {
		return tool.Result{}, err
	}
	defer s.close()
	g := grepper{re: re, context: min(max(in.Context, 0), maxGrepContext), limit: clampInt(in.Limit, 1, maxGrepLimit, defaultGrepLimit)}
	if s.walker == nil {
		if s.size > maxGrepFileBytes {
			g.big++
		} else if b, err := s.ws.ReadFile(s.abs); err == nil && !hasNUL(b) {
			g.file(s.shown, b)
		}
	} else {
		err = s.walker.Walk(func(rel string, d fs.DirEntry) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() || (glob != nil && !glob.Match(rel, false)) {
				return nil
			}
			b, tooBig := candidate(s.walker, rel, d)
			g.big += boolCount(tooBig)
			if b != nil {
				g.file(rel, b)
			}
			if g.full() {
				return fs.SkipAll
			}
			return nil
		})
		if err != nil {
			return tool.Result{}, err
		}
	}
	return g.result(s, in.Pattern), nil
}

// grepPattern is the call's pattern as a regular expression.
func grepPattern(in grepIn) (*regexp.Regexp, error) {
	p := in.Pattern
	if in.Literal {
		p = regexp.QuoteMeta(p)
	}
	if in.IgnoreCase {
		p = "(?i)" + p
	}
	re, err := regexp.Compile(p)
	if err != nil {
		reason := err.Error()
		if se, ok := errors.AsType[*syntax.Error](err); ok {
			reason = string(se.Code)
		}
		return nil, fmt.Errorf("grep: the pattern is not valid RE2 (%s); escape it, or set literal", reason)
	}
	return re, nil
}

// grepper gathers a search's output lines.
type grepper struct {
	re       *regexp.Regexp
	context  int
	limit    int
	lines    []string
	bytes    int
	matches  int
	files    int
	big      int
	more     bool // a match past the limit was found
	cutBytes bool
	cutLines bool
}

// full reports whether the search can stop: the limit is passed, or the
// output is.
func (g *grepper) full() bool { return g.more || g.cutBytes }

// file adds rel's matches, with their context, each line once.
func (g *grepper) file(rel string, b []byte) {
	lines := strings.Split(string(b), "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	hits := map[int]bool{}
	var order []int
	for i, ln := range lines {
		if !g.re.MatchString(strings.TrimSuffix(ln, "\r")) {
			continue
		}
		if g.matches == g.limit {
			g.more = true
			break
		}
		hits[i] = true
		order = append(order, i)
		g.matches++
	}
	if len(order) == 0 {
		return
	}
	g.files++
	next := 0 // the first line not yet shown
	for _, h := range order {
		from, to := max(h-g.context, next), min(h+g.context, len(lines)-1)
		for j := from; j <= to; j++ {
			sep := "-"
			if hits[j] {
				sep = ":"
			}
			if !g.emit(fmt.Sprintf("%s%s%d%s %s", rel, sep, j+1, sep, g.cut(strings.TrimSuffix(lines[j], "\r")))) {
				return
			}
		}
		next = to + 1
	}
}

// cut is a line of at most maxGrepLineChars characters, ending in "…" when
// it was cut.
func (g *grepper) cut(line string) string {
	if utf8.RuneCountInString(line) <= maxGrepLineChars {
		return line
	}
	g.cutLines = true
	n := 0
	for i := range line {
		if n == maxGrepLineChars {
			return line[:i] + "…"
		}
		n++
	}
	return line
}

// emit adds a line, unless it would pass the output cap.
func (g *grepper) emit(line string) bool {
	if g.cutBytes || g.bytes+len(line) >= maxBytes {
		g.cutBytes = true
		return false
	}
	g.lines = append(g.lines, line)
	g.bytes += len(line) + 1
	return true
}

func (g *grepper) result(s search, pattern string) tool.Result {
	var notes []string
	if g.more {
		notes = append(notes, limitNote(g.limit, maxGrepLimit, "matches", "narrow the pattern or path"))
	}
	if g.cutBytes {
		notes = append(notes, fmt.Sprintf("[output cut at %d KB; narrow the pattern or path]", maxBytes/1024))
	}
	if g.cutLines {
		notes = append(notes, fmt.Sprintf("[some lines cut at %d characters; read the file for the whole line]", maxGrepLineChars))
	}
	if g.big > 0 {
		verb := "were"
		if g.big == 1 {
			verb = "was"
		}
		notes = append(notes, fmt.Sprintf("[%s over %d MB %s not searched]", count(g.big, "file", "files"), maxGrepFileBytes>>20, verb))
	}
	body := strings.Join(g.lines, "\n")
	if g.matches == 0 {
		body = fmt.Sprintf("grep %s: no matches for %s", s.shown, pattern)
	}
	r := tool.TextResult(joinNotes(body, notes))
	r.Summary = summary(fmt.Sprintf("grep %s: %s in %s", s.shown, count(g.matches, "match", "matches"), count(g.files, "file", "files")))
	return r
}

// search is a search tool's root: a directory, opened for walking, or one
// file.
type search struct {
	ws     fsx.Workspace
	abs    string
	shown  string
	size   int64
	walker *fsx.Walker // nil for a file
}

func (s search) close() {
	if s.walker != nil {
		s.walker.Close() //nolint:errcheck,gosec // read-only
	}
}

// openSearch resolves a search tool's path, the working directory when it
// is empty. A missing path is an error naming similar entries, as read's is.
func openSearch(env tool.Env, name, path string) (search, error) {
	path = orDot(path)
	ws := workspace(env)
	abs := resolve(env, path)
	s := search{ws: ws, abs: abs, shown: shown(env, path)}
	info, err := ws.Stat(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return search{}, fmt.Errorf("%s %s: no such file or directory%s", name, s.shown, didYouMean(env, ws, abs))
	}
	if err != nil {
		return search{}, fmt.Errorf("%s %s: %w", name, s.shown, unwrapPath(err))
	}
	if !info.IsDir() {
		s.size = info.Size()
		return s, nil
	}
	if s.walker, err = ws.Walker(abs); err != nil {
		return search{}, fmt.Errorf("%s %s: %w", name, s.shown, unwrapPath(err))
	}
	return s, nil
}

// limitNote is the notice of a search that reached its limit: what was
// shown, and how to see more, while a larger limit is possible.
func limitNote(limit, most int, what, narrow string) string {
	if limit < most {
		return fmt.Sprintf("[%d %s shown, the limit; use limit=%d for more, or %s]", limit, what, min(2*limit, most), narrow)
	}
	return fmt.Sprintf("[%d %s shown, the most there can be; %s]", limit, what, narrow)
}

// joinNotes is body, then a blank line and the notices, if any.
func joinNotes(body string, notes []string) string {
	if len(notes) == 0 {
		return body
	}
	return body + "\n\n" + strings.Join(notes, "\n")
}

// candidate is a file's bytes for searching, or nil when it is skipped:
// too big (which tooBig reports, so the result can say so), or unreadable
// or binary, which are skipped silently, as ripgrep skips them.
func candidate(k *fsx.Walker, rel string, d fs.DirEntry) (b []byte, tooBig bool) {
	info, err := d.Info()
	if err != nil {
		return nil, false
	}
	if info.Size() > maxGrepFileBytes {
		return nil, true
	}
	if b, err = k.ReadFile(rel); err != nil || hasNUL(b) {
		return nil, false
	}
	return b, false
}

// hasNUL reports a NUL in a file's first bytes, git's test for a binary
// file.
func hasNUL(b []byte) bool { return bytes.IndexByte(b[:min(len(b), binarySniffBytes)], 0) >= 0 }

// clampInt is v within [lo, hi], or def when v is 0: a published bound
// served within it (0012-MADR D2 rule 8).
func clampInt(v, lo, hi, def int) int {
	if v == 0 {
		return def
	}
	return min(max(v, lo), hi)
}

func boolCount(b bool) int {
	if b {
		return 1
	}
	return 0
}

// orDot is path, or "." when it is empty.
func orDot(path string) string {
	if path == "" {
		return "."
	}
	return path
}

// argString is a string argument of a call, for its title, or "".
func argString(c tool.Call, key string) string {
	var args map[string]any
	if json.Unmarshal(c.Args, &args) != nil {
		return ""
	}
	if s, ok := args[key].(string); ok {
		return s
	}
	return ""
}
