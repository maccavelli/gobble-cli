package render

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Stream buffers streamed Markdown and releases only text that later chunks
// cannot re-interpret: nothing inside an open fence, table, heading, inline
// code span, emphasis, link or image.
//
// It reimplements the behaviour of goose's MarkdownBuffer
// (crates/goose-cli/src/session/streaming_buffer.rs at 591edd4, Apache-2.0),
// including its resumable scan checkpoint, so each Push scans only the new
// tail. goose's code-block truncation is not carried over; tool output is
// bounded separately (ToolOutput).
type Stream struct {
	buf string
	cp  checkpoint
}

// checkpoint is scan progress carried across Push calls: pos is a line-start
// offset in buf, and state and lastSafe are as of pos. It is only kept at
// line starts on or before stableLimit, past which a line-start decision can
// still be re-interpreted as the buffer grows.
type checkpoint struct {
	pos      int
	state    parseState
	lastSafe int
}

type parseState struct {
	inCodeBlock     bool
	fenceChar       byte
	fenceLen        int
	inTable         bool
	pendingHeading  bool
	inInlineCode    bool
	inlineCodeLen   int
	inBold          bool
	inItalic        bool
	inStrikethrough bool
	inLinkText      bool
	inLinkURL       bool
	inImageAlt      bool
}

func (s parseState) clean() bool {
	return !s.inCodeBlock && !s.inTable && !s.pendingHeading && !s.inInlineCode && !s.inBold &&
		!s.inItalic && !s.inStrikethrough && !s.inLinkText && !s.inLinkURL && !s.inImageAlt
}

// Push adds chunk and returns the text that is now safe to render, or "".
func (m *Stream) Push(chunk string) string {
	m.buf += chunk
	end := m.safeEnd()
	if end == 0 {
		return ""
	}
	out := m.buf[:end]
	m.buf = m.buf[end:]
	// The remainder's first byte is now a line start, so the next scan must
	// begin from scratch, exactly as a full rescan of the remainder would.
	m.cp = checkpoint{}
	return out
}

// Flush returns everything still buffered, open constructs included. Call it
// at the end of a stream.
func (m *Stream) Flush() string {
	out := m.buf
	*m = Stream{}
	return out
}

// stableLimit is where the buffer's mutable tail begins: the trailing
// incomplete line, plus any complete lines above it holding only whitespace
// and fence characters.
func (m *Stream) stableLimit() int {
	limit := strings.LastIndexByte(m.buf, '\n') + 1
	for limit > 0 {
		prev := strings.LastIndexByte(m.buf[:limit-1], '\n') + 1
		if strings.Trim(m.buf[prev:limit], "`~ \t\r\n") != "" {
			break
		}
		limit = prev
	}
	return limit
}

// safeEnd returns the last offset at which the parse state is clean,
// resuming from the checkpoint.
func (m *Stream) safeEnd() int {
	state, lastSafe, pos := m.cp.state, m.cp.lastSafe, m.cp.pos
	stable := m.stableLimit()
	n := len(m.buf)
	for pos < n {
		atLineStart := pos == 0 || m.buf[pos-1] == '\n'
		if atLineStart && pos <= stable {
			m.cp = checkpoint{pos: pos, state: state, lastSafe: lastSafe}
		}
		if atLineStart {
			if next, ok := m.lineStart(&state, pos); ok {
				pos = next
				if state.clean() {
					lastSafe = pos
				}
				continue
			}
		}
		if state.inCodeBlock {
			if i := strings.IndexByte(m.buf[pos:], '\n'); i >= 0 {
				pos += i + 1
			} else {
				pos = n
			}
			continue
		}
		lineEnd := n
		if i := strings.IndexByte(m.buf[pos:], '\n'); i >= 0 {
			lineEnd = pos + i + 1
		}
		for off := pos; off < lineEnd; {
			tok := nextToken(m.buf[off:lineEnd])
			off += len(tok)
			inlineToken(&state, tok)
			if state.clean() {
				lastSafe = off
			}
		}
		if lineEnd > pos && m.buf[lineEnd-1] == '\n' {
			state.pendingHeading = false
			if state.clean() {
				lastSafe = lineEnd
			}
		}
		pos = lineEnd
	}
	return lastSafe
}

// lineStart handles block constructs at a line start. ok reports that the
// construct consumed input up to next.
func (m *Stream) lineStart(state *parseState, pos int) (next int, ok bool) {
	rest := m.buf[pos:]
	state.pendingHeading = false
	if adv, ok := codeFence(rest, state); ok {
		return pos + adv, true
	}
	if state.inCodeBlock {
		return 0, false
	}
	if hashes := len(rest) - len(strings.TrimLeft(rest, "#")); hashes > 0 && hashes <= 6 {
		after := rest[hashes:]
		if after == "" || after[0] == ' ' || after[0] == '\n' {
			state.pendingHeading = true
			return 0, false
		}
	}
	if strings.HasPrefix(rest, "|") {
		state.inTable = true
		return 0, false
	}
	if (rest == "" || rest[0] == '\n') && state.inTable {
		state.inTable = false
		return pos + 1, true
	}
	state.inTable = false
	return 0, false
}

// codeFence opens or closes a fence at the start of rest. Like goose, it
// trims leading whitespace, blank lines included, before looking for the
// fence, and advances past the first line of rest.
func codeFence(rest string, state *parseState) (adv int, ok bool) {
	trimmed := strings.TrimLeftFunc(rest, unicode.IsSpace)
	if trimmed == "" || (trimmed[0] != '`' && trimmed[0] != '~') {
		return 0, false
	}
	ch := trimmed[0]
	fenceLen := len(trimmed) - len(strings.TrimLeft(trimmed, string(ch)))
	if fenceLen < 3 {
		return 0, false
	}
	after := trimmed[fenceLen:]
	if state.inCodeBlock {
		closes := ch == state.fenceChar && fenceLen >= state.fenceLen &&
			(after == "" || after[0] == '\n' || strings.TrimSpace(after) == "")
		if !closes {
			return 0, false
		}
		state.inCodeBlock, state.fenceChar, state.fenceLen = false, 0, 0
	} else {
		state.inCodeBlock, state.fenceChar, state.fenceLen = true, ch, fenceLen
	}
	if i := strings.IndexByte(rest, '\n'); i >= 0 {
		return i + 1, true
	}
	return len(rest), true
}

// special are the bytes that start an inline token.
const special = "\\*_`~[]!()"

// nextToken returns the inline token at the start of s, in goose's order:
// an escape, a backtick run, ***, **, *, ___, __, _, ~~, ![, ](, [, ], ),
// a run of plain text, or any other single character.
func nextToken(s string) string {
	if s[0] == '\\' && len(s) > 1 {
		if s[1] == '\n' {
			return s[:1]
		}
		_, size := utf8.DecodeRuneInString(s[1:])
		return s[:1+size]
	}
	if s[0] == '`' {
		return s[:len(s)-len(strings.TrimLeft(s, "`"))]
	}
	for _, t := range [...]string{"***", "**", "*", "___", "__", "_", "~~", "![", "](", "[", "]", ")"} {
		if strings.HasPrefix(s, t) {
			return t
		}
	}
	if i := strings.IndexAny(s, special); i != 0 {
		if i < 0 {
			return s
		}
		return s[:i]
	}
	_, size := utf8.DecodeRuneInString(s)
	return s[:size]
}

func inlineToken(state *parseState, tok string) {
	if tok[0] == '\\' && len(tok) == 2 {
		return
	}
	if tok[0] == '`' {
		switch {
		case !state.inInlineCode:
			state.inInlineCode, state.inlineCodeLen = true, len(tok)
		case len(tok) == state.inlineCodeLen:
			state.inInlineCode, state.inlineCodeLen = false, 0
		}
		return
	}
	if state.inInlineCode {
		return
	}
	switch tok {
	case "***", "___":
		switch {
		case state.inBold && state.inItalic:
			state.inBold, state.inItalic = false, false
		case state.inBold:
			state.inItalic = !state.inItalic
		case state.inItalic:
			state.inBold = !state.inBold
		default:
			state.inBold, state.inItalic = true, true
		}
	case "**", "__":
		state.inBold = !state.inBold
	case "*", "_":
		state.inItalic = !state.inItalic
	case "~~":
		state.inStrikethrough = !state.inStrikethrough
	case "![":
		state.inImageAlt = true
	case "[":
		if !state.inLinkText && !state.inImageAlt {
			state.inLinkText = true
		}
	case "](":
		switch {
		case state.inLinkText:
			state.inLinkText, state.inLinkURL = false, true
		case state.inImageAlt:
			state.inImageAlt, state.inLinkURL = false, true
		}
	case ")":
		state.inLinkURL = false
	}
}
