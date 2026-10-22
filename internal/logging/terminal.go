package logging

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/text/width"
)

// terminalHandler writes a record to a terminal as gobble's own
// diagnostic, as 0008-MADR D3 has every diagnostic: "Warning:" or "Error:",
// the message, then the attributes in parentheses as "key: value",
// separated by "; ", wrapped at spaces to the width less one column, the
// continuation lines indented two (0024-MADR D6). A path under home is
// shown with ~. The redaction is the text handler's. With show, the
// diagnostic goes to it unwrapped instead (0030-MADR D4, as amended).
type terminalHandler struct {
	w     *lockedWriter
	show  func(diagnostic string)
	level slog.Leveler
	width int
	home  string
	attrs []string // the logger's own, "key: value", already redacted
	group string   // the open group's prefix, "a.b."
}

// lockedWriter serializes whole diagnostics, as slog's handlers do.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newTerminalHandler(w io.Writer, show func(string), level slog.Leveler, cols int, home string) *terminalHandler {
	return &terminalHandler{w: &lockedWriter{w: w}, show: show, level: level, width: cols, home: home}
}

func (h *terminalHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *terminalHandler) Handle(_ context.Context, r slog.Record) error {
	prefix := "Warning: "
	if r.Level >= slog.LevelError {
		prefix = "Error: "
	}
	parts := append([]string(nil), h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		parts = h.appendAttr(parts, h.group, a)
		return true
	})
	text := prefix + r.Message
	if len(parts) > 0 {
		text += " (" + strings.Join(parts, "; ") + ")"
	}
	h.w.mu.Lock()
	defer h.w.mu.Unlock()
	if h.show != nil {
		h.show(text)
		return nil
	}
	_, err := io.WriteString(h.w.w, Wrap(text, h.width))
	return err
}

func (h *terminalHandler) WithAttrs(as []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append([]string(nil), h.attrs...)
	for _, a := range as {
		c.attrs = h.appendAttr(c.attrs, h.group, a)
	}
	return &c
}

func (h *terminalHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	c := *h
	c.group = h.group + name + "."
	return &c
}

// appendAttr adds a, redacted, flattening a group into prefixed keys.
func (h *terminalHandler) appendAttr(parts []string, prefix string, a slog.Attr) []string {
	a.Value = a.Value.Resolve()
	if a.Value.Kind() == slog.KindGroup {
		inner := prefix
		if a.Key != "" {
			inner += a.Key + "."
		}
		for _, g := range a.Value.Group() {
			parts = h.appendAttr(parts, inner, g)
		}
		return parts
	}
	if a.Equal(slog.Attr{}) {
		return parts
	}
	a = redact(nil, a)
	return append(parts, prefix+a.Key+": "+h.tilde(a.Value.String()))
}

// tilde shows a path under home with ~ in its place.
func (h *terminalHandler) tilde(v string) string {
	if h.home == "" {
		return v
	}
	home := filepath.Clean(h.home)
	if v == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(v, home+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rest
	}
	return v
}

// Wrap draws a diagnostic as the terminal handler writes it on a terminal
// cols columns wide; with no width known (under 2), on one line.
func Wrap(diagnostic string, cols int) string {
	if cols < 2 {
		return strings.Join(strings.Fields(diagnostic), " ") + "\n"
	}
	var b strings.Builder
	for _, line := range wrapWords(strings.Fields(diagnostic), cols-1, 2) {
		b.WriteString(line + "\n")
	}
	return b.String()
}

// wrapWords fills lines of at most room columns with words, every line
// after the first indented indent columns. A word wider than its line is
// cut.
func wrapWords(words []string, room, indent int) []string {
	var lines []string
	var line strings.Builder
	pad, used := "", 0
	flush := func() {
		lines = append(lines, pad+line.String())
		line.Reset()
		pad, used = strings.Repeat(" ", indent), 0
	}
	for _, w := range words {
		ww := columns(w)
		if used > 0 && len(pad)+used+1+ww > room {
			flush()
		}
		for used == 0 && len(pad)+ww > room && room-len(pad) > 0 {
			head, tail := cutColumns(w, room-len(pad))
			line.WriteString(head)
			used = columns(head)
			flush()
			w, ww = tail, columns(tail)
		}
		if used > 0 {
			line.WriteByte(' ')
			used++
		}
		line.WriteString(w)
		used += ww
	}
	if used > 0 {
		flush()
	}
	return lines
}

// columns is the terminal columns s takes: two for an East Asian wide or
// fullwidth rune, one otherwise.
func columns(s string) int {
	n := 0
	for _, r := range s {
		if k := width.LookupRune(r).Kind(); k == width.EastAsianWide || k == width.EastAsianFullwidth {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// cutColumns splits s after at most n columns, keeping at least one rune in
// head.
func cutColumns(s string, n int) (head, tail string) {
	used := 0
	for i, r := range s {
		c := columns(string(r))
		if used+c > n && i > 0 {
			return s[:i], s[i:]
		}
		used += c
	}
	return s, ""
}
