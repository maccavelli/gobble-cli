package editor

import (
	"bytes"
	"unicode/utf8"
)

// The keys after the key reader, besides the ones keys.go names.
const (
	keyCtrlU     = 0x15
	keyLF        = '\n'
	keyEscape    = 0x1b
	keyBackspace = 0x7f

	// maxLineLength is the most runes a typed line takes, as in x/term.
	maxLineLength = 4096
)

// The keys decoded from several bytes or given a meaning of their own, in
// the UTF-16 surrogate range, which no typed rune takes (as x/term's are).
const (
	keyUnknown = 0xd800 + iota
	keyUp
	keyDown
	keyLeft
	keyRight
	keyWordLeft
	keyWordRight
	keyHome
	keyEnd
	keyDeleteWord
	keyDeleteLine
	keyDelete
	keyClearScreen
	keyTranspose
	keyPasteStart
	keyPasteEnd
)

// controlKeys are the control bytes that are editing keys outside a paste.
var controlKeys = map[byte]rune{
	0x01: keyHome,        // Ctrl+A
	0x02: keyLeft,        // Ctrl+B
	0x05: keyEnd,         // Ctrl+E
	0x06: keyRight,       // Ctrl+F
	0x08: keyBackspace,   // Ctrl+H
	0x0b: keyDeleteLine,  // Ctrl+K
	0x0c: keyClearScreen, // Ctrl+L
	0x0e: keyDown,        // Ctrl+N
	0x10: keyUp,          // Ctrl+P
	0x14: keyTranspose,   // Ctrl+T
	0x17: keyDeleteWord,  // Ctrl+W
}

// escapeKeys are the escape sequences that are keys outside a paste: the
// ones x/term reads, and ESC b, ESC f and Ctrl+←/→ by word (0027-MADR D4,
// as amended).
var escapeKeys = []struct {
	seq string
	key rune
}{
	{"\x1b[A", keyUp},
	{"\x1b[B", keyDown},
	{"\x1b[C", keyRight},
	{"\x1b[D", keyLeft},
	{"\x1b[H", keyHome},
	{"\x1b[F", keyEnd},
	{"\x1b[3~", keyDelete},
	{"\x1b[1;3C", keyWordRight}, // Alt+→
	{"\x1b[1;3D", keyWordLeft},  // Alt+←
	{"\x1b[1;5C", keyWordRight}, // Ctrl+→
	{"\x1b[1;5D", keyWordLeft},  // Ctrl+←
	{"\x1bf", keyWordRight},     // Option+→ in macOS Terminal, Alt+F
	{"\x1bb", keyWordLeft},      // Option+← in macOS Terminal, Alt+B
	{"\x1b[200~", keyPasteStart},
}

// decodeKey reads the key at the front of b and returns it with the bytes
// after it. ok is false when b holds only the start of a key, which waits
// for more. Inside a paste every byte is text but the paste's end marker.
// An escape sequence that is no key is keyUnknown, up to its final letter
// or '~'; a byte that is not UTF-8 is utf8.RuneError.
func decodeKey(b []byte, pasting bool) (key rune, rest []byte, ok bool) {
	if len(b) == 0 {
		return 0, b, false
	}
	if !pasting {
		if k, found := controlKeys[b[0]]; found {
			return k, b[1:], true
		}
	}
	if b[0] != keyEscape {
		if !utf8.FullRune(b) {
			return 0, b, false
		}
		r, n := utf8.DecodeRune(b)
		return r, b[n:], true
	}
	if pasting {
		if bytes.HasPrefix(b, pasteEnd) {
			return keyPasteEnd, b[len(pasteEnd):], true
		}
	} else {
		for _, k := range escapeKeys {
			if bytes.HasPrefix(b, []byte(k.seq)) {
				return k.key, b[len(k.seq):], true
			}
		}
	}
	for i, c := range b[1:] {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '~' {
			return keyUnknown, b[i+2:], true
		}
	}
	return 0, b, false
}

// bufferEvent is what feeding the buffer came to.
type bufferEvent int

const (
	// needMore: every key given was used; the line goes on.
	needMore bufferEvent = iota
	// lineEnded: Enter ended the line; take it.
	lineEnded
	// endOfInput: Ctrl+D on an empty line, or Ctrl+C.
	endOfInput
	// clearScreen: Ctrl+L; clear the screen and draw the line again.
	clearScreen
)

// lineBuffer is the line being typed and its cursor, in runes, and what
// each key does to them (0027-MADR D1, D4): the editing of x/term's
// Terminal, with ↑/↓ by row and more keys by word. The row keys use the
// layout of prompt at columns; columns 0 is one row.
type lineBuffer struct {
	text []rune
	pos  int

	prompt  string
	columns int

	// complete is called, as x/term's AutoCompleteCallback is, for every
	// key that is not an editing key: the line, the cursor's byte offset
	// and the key. A line it returns replaces the buffer's.
	complete func(line string, pos int, key rune) (newLine string, newPos int, ok bool)

	history  History
	recalled int    // the entry shown, or -1 for the user's own line
	own      []rune // the user's own line while an entry is shown

	pasting bool // between the paste markers
	pasted  bool // the line began inside a paste and has had no typed key
}

func newLineBuffer(h History) *lineBuffer {
	return &lineBuffer{history: h, recalled: -1}
}

// feed applies the keys at the front of b until the line ends, the input
// ends, the screen is to be cleared, or b is used up, and returns the bytes
// it has not used: the start of a key, or the keys after the event.
func (l *lineBuffer) feed(b []byte) ([]byte, bufferEvent) {
	for {
		k, rest, ok := decodeKey(b, l.pasting)
		if !ok {
			return b, needMore
		}
		b = rest
		if !l.pasting {
			switch {
			case k == keyCtrlD && len(l.text) == 0, k == keyCtrlC:
				return b, endOfInput
			case k == keyPasteStart:
				l.pasting = true
				l.pasted = l.pasted || len(l.text) == 0
				continue
			}
			l.pasted = false
		} else if k == keyPasteEnd {
			l.pasting = false
			continue
		}
		if k == keyEnter && len(b) > 0 && b[0] == keyLF {
			b = b[1:]
		}
		if ev := l.key(k); ev != needMore {
			return b, ev
		}
	}
}

// take returns the line that ended and whether it was pasted, and starts
// the next, which is pasted while the paste goes on.
func (l *lineBuffer) take() (line string, pasted bool) {
	line, pasted = string(l.text), l.pasted
	l.text, l.pos = nil, 0
	l.recalled, l.own = -1, nil
	l.pasted = l.pasting
	return line, pasted
}

// key applies one key.
func (l *lineBuffer) key(k rune) bufferEvent {
	if l.pasting && k != keyEnter && k != keyLF {
		if k != keyUnknown && k != utf8.RuneError {
			l.insert(k)
		}
		return needMore
	}
	switch k {
	case keyEnter, keyLF:
		return lineEnded
	case keyClearScreen:
		return clearScreen
	case keyBackspace:
		l.erase(min(1, l.pos))
	case keyLeft:
		l.pos = max(l.pos-1, 0)
	case keyRight:
		l.pos = min(l.pos+1, len(l.text))
	case keyWordLeft:
		l.pos -= l.toLeftWord()
	case keyWordRight:
		l.pos += l.toRightWord()
	case keyHome:
		l.pos = 0
	case keyEnd:
		l.pos = len(l.text)
	case keyUp:
		l.up()
	case keyDown:
		l.down()
	case keyDeleteWord:
		l.erase(l.toLeftWord())
	case keyDeleteLine:
		l.text = l.text[:l.pos]
	case keyCtrlD, keyDelete:
		if l.pos < len(l.text) {
			l.pos++
			l.erase(1)
		}
	case keyCtrlU:
		l.erase(l.pos)
	case keyTranspose:
		l.transpose()
	default:
		l.other(k)
	}
	return needMore
}

// other offers k to the completion callback, then types it when it is
// printable and the line has room.
func (l *lineBuffer) other(k rune) {
	if l.complete != nil {
		line, pos, ok := l.complete(string(l.text), len(string(l.text[:l.pos])), k)
		if ok {
			l.text = []rune(line)
			l.pos = utf8.RuneCountInString(line[:pos])
			return
		}
	}
	if k < ' ' || k >= keyUnknown && k <= 0xdbff || k == utf8.RuneError || len(l.text) == maxLineLength {
		return
	}
	l.insert(k)
}

func (l *lineBuffer) insert(k rune) {
	l.text = append(l.text[:l.pos], append([]rune{k}, l.text[l.pos:]...)...)
	l.pos++
}

// erase removes the n runes before the cursor.
func (l *lineBuffer) erase(n int) {
	l.text = append(l.text[:l.pos-n], l.text[l.pos:]...)
	l.pos -= n
}

// transpose swaps the two runes around the cursor, or the last two at the
// line's end, and moves past them.
func (l *lineBuffer) transpose() {
	if len(l.text) < 2 || l.pos < 1 {
		return
	}
	swap := min(l.pos, len(l.text)-1)
	l.text[swap-1], l.text[swap] = l.text[swap], l.text[swap-1]
	l.pos = min(l.pos+1, len(l.text))
}

// toLeftWord is how far the cursor goes back to the start of the word
// before it, over the spaces after that word.
func (l *lineBuffer) toLeftWord() int {
	if l.pos == 0 {
		return 0
	}
	p := l.pos - 1
	for p > 0 && l.text[p] == ' ' {
		p--
	}
	for p > 0 {
		if l.text[p] == ' ' {
			p++
			break
		}
		p--
	}
	return l.pos - p
}

// toRightWord is how far the cursor goes forward to the start of the next
// word.
func (l *lineBuffer) toRightWord() int {
	p := l.pos
	for p < len(l.text) && l.text[p] != ' ' {
		p++
	}
	for p < len(l.text) && l.text[p] == ' ' {
		p++
	}
	return p - l.pos
}

// up moves to the row above, or recalls the entry before when the cursor
// is on the first row (Q1).
func (l *lineBuffer) up() {
	if places := placesOf(l.prompt, l.text, l.columns); places[l.pos].row > 0 {
		l.pos = nearest(places, places[l.pos].row-1, places[l.pos].column)
		return
	}
	entry, ok := l.entry(l.recalled + 1)
	if !ok {
		return
	}
	if l.recalled == -1 {
		l.own = l.text
	}
	l.recalled++
	l.show([]rune(entry))
}

// down moves to the row below, or recalls the entry after, and after the
// newest the user's own line, when the cursor is on the last row (Q1).
func (l *lineBuffer) down() {
	places := placesOf(l.prompt, l.text, l.columns)
	if row := places[l.pos].row; row < places[len(l.text)].row {
		l.pos = nearest(places, row+1, places[l.pos].column)
		return
	}
	switch l.recalled {
	case -1:
	case 0:
		l.recalled--
		l.show(l.own)
	default:
		if entry, ok := l.entry(l.recalled - 1); ok {
			l.recalled--
			l.show([]rune(entry))
		}
	}
}

func (l *lineBuffer) entry(idx int) (string, bool) {
	if l.history == nil || idx < 0 || idx >= l.history.Len() {
		return "", false
	}
	return l.history.At(idx), true
}

// show puts text in the line, the cursor at its end.
func (l *lineBuffer) show(text []rune) {
	l.text, l.pos = text, len(text)
}

// place is where the cursor stands on screen: its row in the line's rows
// and its column.
type place struct{ row, column int }

// placesOf is the place layout gives the cursor at each rune index of text,
// 0 to len(text), in one pass over the same rows. columns 0 is one row.
func placesOf(prompt string, text []rune, columns int) []place {
	indent := displayWidth(prompt)
	if columns <= 0 {
		columns = indent + runesWidth(text) + 1
	}
	room := max(columns-indent, 1)
	out := make([]place, 0, len(text)+1)
	for start, row := 0, 0; ; row++ {
		end, next := rowEnd(text, start, room)
		last := next >= len(text) && end == len(text)
		stop := next
		if last {
			stop = len(text) + 1
		}
		col := indent
		for p := start; p < stop; p++ {
			if p > start && p <= end {
				col += runeWidth(text[p-1])
			}
			if col >= columns {
				out = append(out, place{row + 1, indent})
			} else {
				out = append(out, place{row, col})
			}
		}
		if last {
			return out
		}
		start = next
	}
}

// nearest is the rune index on row whose column is the cursor's, or the
// last one before it, or the row's first.
func nearest(places []place, row, column int) int {
	best := -1
	for p, pl := range places {
		if pl.row != row {
			continue
		}
		if best == -1 || pl.column <= column {
			best = p
		}
	}
	return best
}
