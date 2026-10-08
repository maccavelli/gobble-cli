package builtin

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Edit matching (0005-MADR amendment of 2026-10-07, choice 2). Every edit is
// matched against the original content, not after the edits before it, as
// Pi's edit tool does. An edit's oldText is found by layers, in order:
//
//  1. exactly;
//  2. after Pi's normalisation (NFKC, trailing whitespace, quotes, dashes,
//     special spaces). When any edit needs it, the whole call moves into
//     that normalised space, and only the lines the edits touch are
//     rewritten from it: the other lines keep their bytes;
//  3. by opencode's line-trimmed, indentation-flexible and
//     escape-normalised strategies (editstrategy.go), against the same base.
//
// Each layer needs exactly one candidate. A match found by a layer other
// than the first must not be much larger than its oldText.

// edit is one replacement.
type edit struct {
	OldText string `json:"oldText" jsonschema:"The text to replace. It must occur once in the file as it was before this call, and must not overlap another edit's oldText."`
	NewText string `json:"newText" jsonschema:"The text to put in its place."`
}

// lineEnding is a file's line ending: CRLF when its first newline is one.
func lineEnding(s string) string {
	lf := strings.IndexByte(s, '\n')
	crlf := strings.Index(s, "\r\n")
	if lf < 0 || crlf < 0 || crlf >= lf {
		return "\n"
	}
	return "\r\n"
}

// toLF makes every line ending a \n: CRLF first, then a lone CR.
func toLF(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// restoreEnding gives LF text the file's line ending back.
func restoreEnding(s, ending string) string {
	if ending == "\r\n" {
		return strings.ReplaceAll(s, "\n", "\r\n")
	}
	return s
}

var (
	singleQuotes  = regexp.MustCompile(`[\x{2018}\x{2019}\x{201A}\x{201B}]`)
	doubleQuotes  = regexp.MustCompile(`[\x{201C}\x{201D}\x{201E}\x{201F}]`)
	dashes        = regexp.MustCompile(`[\x{2010}-\x{2015}\x{2212}]`)
	specialSpaces = regexp.MustCompile(`[\x{00A0}\x{2002}-\x{200A}\x{202F}\x{205F}\x{3000}]`)
)

// fuzzy is the normalisation of layer 2, in Pi's order: NFKC, trailing
// whitespace off each line, smart single and double quotes to ASCII, dashes
// to a hyphen, special spaces to a space. It keeps the number of lines.
func fuzzy(s string) string {
	s = norm.NFKC.String(s)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRightFunc(l, trailingSpace)
	}
	s = strings.Join(lines, "\n")
	s = singleQuotes.ReplaceAllString(s, "'")
	s = doubleQuotes.ReplaceAllString(s, `"`)
	s = dashes.ReplaceAllString(s, "-")
	return specialSpaces.ReplaceAllString(s, " ")
}

// trailingSpace is the whitespace trimmed from a line's end: Unicode spaces,
// tabs, and the byte-order mark some editors leave behind.
func trailingSpace(r rune) bool {
	return r == '\uFEFF' || unicode.IsSpace(r)
}

// matched is an edit found in the base content.
type matched struct {
	index   int // the edit's position in the call
	at, len int // the match, in the base content
	newText string
}

// editError builds the edit tool's errors, which name the file as shown
// and the edit the way the call does: "oldText" alone, or "edits[i].oldText"
// when the call has several.
type editError struct {
	path  string
	total int
}

func (e editError) label(i int) string {
	if e.total == 1 {
		return "oldText"
	}
	return fmt.Sprintf("edits[%d].oldText", i)
}

func (e editError) empty(i int) error {
	return fmt.Errorf("edit %s: %s is empty", e.path, e.label(i))
}

func (e editError) notFound(i int) error {
	return fmt.Errorf("edit %s: %s was not found; it must match the file exactly, or nearly (whitespace, quotes and indentation are forgiven)", e.path, e.label(i))
}

func (e editError) duplicate(i, n int) error {
	return fmt.Errorf("edit %s: %s occurs %d times; include more surrounding lines so it occurs once", e.path, e.label(i), n)
}

func (e editError) oversize(i int) error {
	return fmt.Errorf("edit %s: %s matched a region much larger than itself; read the file again and resend the edit", e.path, e.label(i))
}

func (e editError) overlap(a, b int) error {
	return fmt.Errorf("edit %s: edits[%d] and edits[%d] overlap; merge them into one edit or choose separate regions", e.path, a, b)
}

func (e editError) noChange() error {
	if e.total == 1 {
		return fmt.Errorf("edit %s: the edit changes nothing", e.path)
	}
	return fmt.Errorf("edit %s: the edits change nothing", e.path)
}

// applyEdits applies edits to LF content and returns the new content. path
// is the file as the model is shown it, for the errors.
func applyEdits(content string, edits []edit, path string) (string, error) {
	ee := editError{path: path, total: len(edits)}
	norms := make([]edit, len(edits))
	for i, e := range edits {
		norms[i] = edit{OldText: toLF(e.OldText), NewText: toLF(e.NewText)}
		if norms[i].OldText == "" {
			return "", ee.empty(i)
		}
	}
	useFuzzy := slices.ContainsFunc(norms, func(e edit) bool {
		return !strings.Contains(content, e.OldText) && strings.Contains(fuzzy(content), fuzzy(e.OldText))
	})
	base := content
	if useFuzzy {
		base = fuzzy(content)
	}
	ms := make([]matched, 0, len(norms))
	for i, e := range norms {
		at, n, err := locate(base, e.OldText, useFuzzy, i, ee)
		if err != nil {
			return "", err
		}
		ms = append(ms, matched{index: i, at: at, len: n, newText: e.NewText})
	}
	slices.SortFunc(ms, func(a, b matched) int { return a.at - b.at })
	for i := 1; i < len(ms); i++ {
		if prev, cur := ms[i-1], ms[i]; prev.at+prev.len > cur.at {
			return "", ee.overlap(prev.index, cur.index)
		}
	}
	var out string
	if useFuzzy {
		kept, err := replaceKeepingLines(content, base, ms)
		if err != nil {
			return "", fmt.Errorf("edit %s: %w", path, err)
		}
		out = kept
	} else {
		out = replaceAll(base, ms, 0)
	}
	if out == content {
		return "", ee.noChange()
	}
	return out, nil
}

// locate finds one edit's oldText in base by the layers, and returns where
// and how long the match is.
func locate(base, old string, inFuzzy bool, i int, ee editError) (int, int, error) {
	if n := strings.Count(base, old); n == 1 {
		return strings.Index(base, old), len(old), nil
	} else if n > 1 {
		return 0, 0, ee.duplicate(i, n)
	}
	find := old
	if inFuzzy {
		find = fuzzy(old)
		if n := strings.Count(base, find); n == 1 {
			return strings.Index(base, find), len(find), nil
		} else if n > 1 {
			return 0, 0, ee.duplicate(i, n)
		}
	}
	most := 0
	for _, strategy := range strategies {
		spans := strategy(base, find)
		if len(spans) != 1 {
			most = max(most, len(spans))
			continue
		}
		s := spans[0]
		if oversized(base[s.start:s.end], find) {
			return 0, 0, ee.oversize(i)
		}
		return s.start, s.end - s.start, nil
	}
	if most > 1 {
		return 0, 0, ee.duplicate(i, most)
	}
	return 0, 0, ee.notFound(i)
}

// oversized is opencode's guard: a forgiving match whose region is far
// larger than the text asked for is more likely the wrong place than the
// right one.
func oversized(region, find string) bool {
	rl, fl := strings.Count(region, "\n")+1, strings.Count(find, "\n")+1
	return rl >= max(fl+3, 2*fl) || len(region) > max(len(find)+500, 4*len(find))
}

// replaceAll applies sorted, disjoint replacements to s, last first so the
// offsets hold. offset is where s starts in the content the offsets count.
func replaceAll(s string, ms []matched, offset int) string {
	for _, m := range slices.Backward(ms) {
		at := m.at - offset
		s = s[:at] + m.newText + s[at+m.len:]
	}
	return s
}

// linesWithEndings splits s into lines that keep their \n.
func linesWithEndings(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.SplitAfter(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type span struct{ start, end int }

// spans are the byte ranges of s's lines, each with its \n.
func spans(s string) []span {
	var out []span
	off := 0
	for _, l := range linesWithEndings(s) {
		out = append(out, span{off, off + len(l)})
		off += len(l)
	}
	return out
}

var errOutsideBase = errors.New("a replacement fell outside the file's lines")

// lineRange is the lines, [start, end), that a replacement touches.
func lineRange(lines []span, m matched) (int, int, error) {
	start := -1
	for i, l := range lines {
		if m.at >= l.start && m.at < l.end {
			start = i
			break
		}
	}
	if start < 0 {
		return 0, 0, errOutsideBase
	}
	end := start
	for end < len(lines) && lines[end].end < m.at+m.len {
		end++
	}
	if end >= len(lines) {
		return 0, 0, errOutsideBase
	}
	return start, end + 1, nil
}

// replaceKeepingLines applies replacements matched in base, the normalised
// view of original, to original. Each replacement is widened to the lines it
// touches; those lines come from base with the replacement made, and every
// other line is copied from original.
func replaceKeepingLines(original, base string, ms []matched) (string, error) {
	origLines := linesWithEndings(original)
	baseLines := spans(base)
	if len(origLines) != len(baseLines) {
		return "", errors.New("normalising the file changed its number of lines; resend the edit with oldText copied exactly")
	}
	type group struct {
		start, end int
		ms         []matched
	}
	var groups []group
	for _, m := range ms {
		s, e, err := lineRange(baseLines, m)
		if err != nil {
			return "", err
		}
		if n := len(groups); n > 0 && s < groups[n-1].end {
			groups[n-1].end = max(groups[n-1].end, e)
			groups[n-1].ms = append(groups[n-1].ms, m)
			continue
		}
		groups = append(groups, group{start: s, end: e, ms: []matched{m}})
	}
	var b strings.Builder
	next := 0
	for _, g := range groups {
		b.WriteString(strings.Join(origLines[next:g.start], ""))
		from, to := baseLines[g.start].start, baseLines[g.end-1].end
		b.WriteString(replaceAll(base[from:to], g.ms, from))
		next = g.end
	}
	b.WriteString(strings.Join(origLines[next:], ""))
	return b.String(), nil
}
