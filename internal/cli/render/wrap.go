package render

import "strings"

// Wrap breaks streamed Markdown prose at spaces so no row is wider than a
// width, and passes code, tables and headings through unchanged (0028-MADR
// D3–D5, D7). It takes what Style made of the text a Stream released:
// escape sequences count as zero width, stay with the word they are in and
// are never dropped at a break, and a line's kind and a word's opening a
// block are read with them taken out (D7, as amended).
//
// Prose is a paragraph, a list item or a block quote: its continuation rows
// are indented under a list item's text and repeat a quote's markers, and
// none starts with a word that would open a block, so the text still means
// what it meant when copied. A word wider than the row is left whole on a
// row of its own. A fenced or indented code line, a table row (a line that
// starts with |, as Tables has it) and a heading pass through.
//
// Across pushes it keeps the row's column and the line's kind, and holds
// back the word still arriving and the last finished word on the row, which
// the next word may move to a new row (D5, as amended). Feeding text in any
// chunks gives what feeding it whole gives.
type Wrap struct {
	out   strings.Builder // this push's output
	width int
	esc   string // an escape sequence still arriving

	known     bool   // the current line's kind is known
	head      string // the line's start, while its kind is not known
	headPlain string // head without its escapes
	pass      bool   // the current line passes through
	opening   bool   // the current line opened the fence
	line      string // the current passed line, without escapes, to find the fence's close
	fence     string // the open fence's marker run, or ""

	cont      string // a prose line's continuation prefix
	col       int    // the columns used on the row
	held      string // the last finished word on the row, not yet written
	heldPlain string // held without its escapes
	heldGap   string // the spaces before held
	heldFirst bool   // held is the row's first word
	gap       string // the spaces after held, before word
	word      string // the word still arriving
	wordPlain string // word without its escapes
}

// Push wraps text to width columns and returns what it can release; a width
// of 0 or less wraps nothing.
func (w *Wrap) Push(text string, width int) string {
	w.width = width
	for _, r := range text {
		w.take(r)
	}
	out := w.out.String()
	w.out.Reset()
	return out
}

// Flush returns everything held and starts afresh, as at a line start with
// no fence open. Call it with the Stream's Flush.
func (w *Wrap) Flush() string {
	if w.esc != "" {
		esc := w.esc
		w.esc = ""
		w.token(esc, true)
	}
	if !w.known && w.head != "" {
		w.classify(true)
	}
	if w.known && !w.pass {
		w.endWord()
		w.emitHeld()
		w.out.WriteString(w.gap)
	}
	out := w.out.String()
	*w = Wrap{}
	return out
}

// take reads a rune, gathering an escape sequence until it is whole.
func (w *Wrap) take(r rune) {
	if w.esc == "" && r != 0x1b {
		w.token(string(r), false)
		return
	}
	w.esc += string(r)
	if len(w.esc) == 2 && r != '[' || len(w.esc) > 2 && r >= 0x40 && r <= 0x7e {
		esc := w.esc
		w.esc = ""
		w.token(esc, true)
	}
}

// token takes one rune, or a whole escape sequence (esc), which has no
// width.
func (w *Wrap) token(t string, esc bool) {
	if !w.known {
		if w.fence != "" {
			w.known, w.pass = true, true
		} else {
			if t == "\n" {
				w.classify(true)
				w.token(t, false)
				return
			}
			w.head += t
			if !esc {
				w.headPlain += t
				w.classify(false)
			}
			return
		}
	}
	if w.pass {
		w.out.WriteString(t)
		switch {
		case esc:
		case t != "\n":
			w.line += t
		default:
			if w.fence != "" && !w.opening && isFenceClose(w.line, w.fence) {
				w.fence = ""
			}
			w.startLine()
		}
		return
	}
	switch {
	case esc:
		w.word += t
	case t == "\n":
		w.endWord()
		w.emitHeld()
		if w.fits(after(w.col, w.gap)) {
			w.out.WriteString(w.gap)
		}
		w.out.WriteByte('\n')
		w.startLine()
	case t == " " || t == "\t":
		w.endWord()
		w.gap += t
	default:
		w.word += t
		w.wordPlain += t
	}
}

// startLine forgets the line just ended; an open fence stays open.
func (w *Wrap) startLine() {
	w.known, w.head, w.headPlain, w.pass, w.opening, w.line = false, "", "", false, false, ""
	w.cont, w.col, w.gap, w.word, w.wordPlain = "", 0, "", "", ""
	w.held, w.heldPlain, w.heldGap, w.heldFirst = "", "", "", false
}

// classify decides the line's kind from its start, once it can, or at the
// line's end; a prose line's prefix is written and the rest of its start
// taken as prose.
func (w *Wrap) classify(atEnd bool) {
	kind, n, run, ok := lineKind(w.headPlain, atEnd)
	if !ok {
		return
	}
	head, plain := w.head, w.headPlain
	w.head, w.headPlain, w.known = "", "", true
	if kind != kindProse {
		w.pass, w.line = true, plain
		if kind == kindFence {
			w.fence, w.opening = run, true
		}
		w.out.WriteString(head)
		return
	}
	prefix, rest := cutPlain(head, n)
	w.out.WriteString(prefix)
	w.cont = strings.Map(func(r rune) rune {
		if r == '>' || r == '\t' {
			return r
		}
		return ' '
	}, plain[:n])
	w.col = after(0, plain[:n])
	for _, r := range rest {
		w.take(r)
	}
}

// cutPlain cuts s after its first n bytes that are not in an escape
// sequence, the escapes among them kept with the head.
func cutPlain(s string, n int) (head, rest string) {
	i := 0
	for plain := 0; i < len(s) && plain < n; {
		if s[i] == 0x1b {
			i += escapeLen(s[i:])
			continue
		}
		i++
		plain++
	}
	return s[:i], s[i:]
}

// escapeLen is the length of the escape sequence s starts with: a control
// sequence through its final byte, or ESC and one byte.
func escapeLen(s string) int {
	if len(s) < 2 || s[1] != '[' {
		return min(2, len(s))
	}
	j := 2
	for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
		j++
	}
	return min(j+1, len(s))
}

// endWord places the word just finished: after the held word if it fits;
// otherwise on a new row, unless it would open a block there, when the held
// word moves down with it, or, when it cannot, the word stays on this row.
func (w *Wrap) endWord() {
	if w.word == "" {
		return
	}
	word, plain, gap := w.word, w.wordPlain, w.gap
	w.word, w.wordPlain, w.gap = "", "", ""
	switch {
	case w.held == "":
		w.held, w.heldPlain, w.heldGap, w.heldFirst = word, plain, gap, true
		return
	case w.fits(after(after(after(w.col, w.heldGap+w.heldPlain), gap), plain)):
	case !opensBlock(plain):
		w.emitHeld()
		w.newRow()
		w.held, w.heldPlain, w.heldGap, w.heldFirst = word, plain, "", true
		return
	case !w.heldFirst && !opensBlock(w.heldPlain):
		w.newRow()
		w.heldGap = ""
	}
	w.emitHeld()
	w.held, w.heldPlain, w.heldGap, w.heldFirst = word, plain, gap, false
}

func (w *Wrap) fits(col int) bool { return w.width <= 0 || col <= w.width }

func (w *Wrap) emitHeld() {
	if w.held == "" {
		return
	}
	w.out.WriteString(w.heldGap + w.held)
	w.col = after(w.col, w.heldGap+w.heldPlain)
	w.held, w.heldPlain, w.heldGap = "", "", ""
}

func (w *Wrap) newRow() {
	w.out.WriteString("\n" + w.cont)
	w.col = after(0, w.cont)
}

// after is the column after s is written from col: a tab to the next
// multiple of 8, other runes by their display width.
func after(col int, s string) int {
	for _, r := range s {
		if r == '\t' {
			col = (col/8 + 1) * 8
			continue
		}
		col += displayWidth(string(r))
	}
	return col
}

// The kinds of line Wrap tells apart.
const (
	kindProse = iota
	kindPass
	kindFence
)

// lineKind reads a line's start s: prose with the length of its prefix
// (indentation, quote markers, a list marker and the spaces after it), a
// line that passes through, or a fence's opening line and its marker run.
// ok is false while more of the line is needed; atEnd says the line ended.
func lineKind(s string, atEnd bool) (kind, prefix int, run string, ok bool) {
	i, cols := 0, 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		cols++
		if s[i] == '\t' {
			cols += 3 // a tab reaches code's indent alone
		}
		i++
	}
	if cols >= 4 {
		return kindPass, 0, "", true
	}
	quoted := false
	for i < len(s) && s[i] == '>' {
		quoted = true
		i++
		for i < len(s) && s[i] == ' ' {
			i++
		}
	}
	if i == len(s) {
		return kindProse, i, "", atEnd
	}
	switch c := s[i]; {
	case c == '|':
		return kindPass, 0, "", true
	case c == '#':
		return headingKind(s, i, atEnd)
	case c == '`' || c == '~':
		return fenceKind(s, i, quoted, atEnd)
	case c == '-' || c == '*' || c == '+':
		return bulletKind(s, i, atEnd)
	case c >= '0' && c <= '9':
		return orderedKind(s, i, atEnd)
	}
	return kindProse, i, "", true
}

// headingKind is lineKind for hashes at i.
func headingKind(s string, i int, atEnd bool) (kind, prefix int, run string, ok bool) {
	h := len(s[i:]) - len(strings.TrimLeft(s[i:], "#"))
	if i+h == len(s) && !atEnd {
		return 0, 0, "", false
	}
	if isHeading(s[i:]) {
		return kindPass, 0, "", true
	}
	return kindProse, i, "", true
}

// fenceKind is lineKind for backticks or tildes at i: a fence is known only
// at its line's end. A fence inside a quote passes its own line only.
func fenceKind(s string, i int, quoted, atEnd bool) (kind, prefix int, run string, ok bool) {
	n := len(s[i:]) - len(strings.TrimLeft(s[i:], s[i:i+1]))
	if !atEnd && (n >= 3 || i+n == len(s)) {
		return 0, 0, "", false
	}
	switch r := fenceRun(s[i:]); {
	case r == "":
		return kindProse, i, "", true
	case quoted:
		return kindPass, 0, "", true
	default:
		return kindFence, 0, r, true
	}
}

// bulletKind is lineKind for -, * or + at i.
func bulletKind(s string, i int, atEnd bool) (kind, prefix int, run string, ok bool) {
	if i+1 == len(s) && !atEnd {
		return 0, 0, "", false
	}
	if i+1 < len(s) && s[i+1] == ' ' {
		return afterMarker(s, i+1, atEnd)
	}
	return kindProse, i, "", true
}

// orderedKind is lineKind for digits at i: an ordered marker is 1 to 9
// digits, then . or ), then a space.
func orderedKind(s string, i int, atEnd bool) (kind, prefix int, run string, ok bool) {
	d := len(s[i:]) - len(strings.TrimLeft(s[i:], "0123456789"))
	switch {
	case i+d+1 >= len(s) && !atEnd:
		return 0, 0, "", false
	case d <= 9 && i+d+1 < len(s) && (s[i+d] == '.' || s[i+d] == ')') && s[i+d+1] == ' ':
		return afterMarker(s, i+d+1, atEnd)
	}
	return kindProse, i, "", true
}

// afterMarker is lineKind's prose prefix ending after a list marker at j and
// the spaces that follow it.
func afterMarker(s string, j int, atEnd bool) (kind, prefix int, run string, ok bool) {
	for j < len(s) && s[j] == ' ' {
		j++
	}
	if j == len(s) && !atEnd {
		return 0, 0, "", false
	}
	return kindProse, j, "", true
}

// opensBlock reports whether word, at a row's start, would open a block in
// Markdown: a list or quote marker, a heading's hashes, a table's pipe, a
// fence, a thematic break or a setext underline (0028-MADR D4).
func opensBlock(word string) bool {
	switch {
	case word == "":
		return false
	case word[0] == '>' || word[0] == '|':
		return true
	case word[0] == '#':
		return isHeading(word)
	case strings.Trim(word, "-*_=+") == "":
		return true
	case fenceRun(word) != "":
		return true
	case word[0] >= '0' && word[0] <= '9':
		return markerLen(word+" ") > 0
	}
	return false
}

// Idle reports whether w is at a line's start holding nothing back: what
// it wrote ends a line, and nothing waits to be written, not even part of
// an escape sequence (0030-MADR D3). The words a line holds are empty at
// its start.
func (w *Wrap) Idle() bool {
	return !w.known && w.head == "" && w.esc == ""
}
