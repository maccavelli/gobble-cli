package cli

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// ACP's option kinds, as acpclient gives them.
const (
	kindAllowOnce    = "allow_once"
	kindAllowAlways  = "allow_always"
	kindRejectOnce   = "reject_once"
	kindRejectAlways = "reject_always"
)

// permissionKeys are D16's keys, by ACP option kind, in the order shown.
var permissionKeys = []struct {
	key   byte
	kind  string
	label string
}{
	{'y', kindAllowOnce, "allow once"},
	{'a', kindAllowAlways, "always"},
	{'n', kindRejectOnce, "deny"},
	{'N', kindRejectAlways, "never"},
}

// ctrlC is Ctrl+C in raw mode; backspace and delete are the keys that
// erase while a reason is typed.
const (
	ctrlC     = 0x03
	backspace = 0x08
	del       = 0x7f
)

// sayWhy is the key of "deny and say why", and chooseWhy the choice it
// gives, which is no option's id (0005-PLAN F6-2b decision F).
const (
	sayWhy    = 'f'
	chooseWhy = "\x00why"
)

// maxWhy bounds the reason typed, in bytes.
const maxWhy = 2048

// permissionLine is D16's one line: the call's title, then the keys of the
// options offered, then [c] cancel. Where two options share a kind, as
// trust's answers do, each option is listed by its name, [1] to [9], in
// the order offered (0005-PLAN F6-1 decision 6); otherwise D16's keys, by
// kind. A tool call's question, which offers reject_once, ends with [f]
// deny and say why (F6-2b decision F). keys maps each shown key to its
// option, and labels each option to the word echoed when it is chosen.
func permissionLine(r acpclient.PermissionRequest) (line string, keys map[byte]string, labels map[string]string) {
	keys, labels = map[byte]string{}, map[string]string{}
	var b strings.Builder
	title := strings.Join(strings.Fields(term.Sanitize(r.Tool.Title)), " ")
	if title == "" {
		title = "Run the tool call"
	}
	fmt.Fprintf(&b, "%s?", title)
	if kindsRepeat(r.Options) {
		for i, o := range r.Options[:min(len(r.Options), 9)] {
			key := byte('1' + i)
			name := strings.Join(strings.Fields(term.Sanitize(o.Name)), " ")
			keys[key], labels[o.ID] = o.ID, name
			fmt.Fprintf(&b, "  [%c] %s", key, name)
		}
	} else {
		for _, k := range permissionKeys {
			for _, o := range r.Options {
				if o.Kind == k.kind {
					if _, seen := keys[k.key]; !seen {
						keys[k.key], labels[o.ID] = o.ID, k.label
						fmt.Fprintf(&b, "  [%c] %s", k.key, k.label)
					}
				}
			}
		}
	}
	if slices.ContainsFunc(r.Options, func(o acpclient.PermissionOption) bool { return o.ID == acpserver.RejectOnceID }) {
		keys[sayWhy], labels[chooseWhy] = chooseWhy, "deny and say why"
		b.WriteString("  [f] deny and say why")
	}
	b.WriteString("  [c] cancel ")
	return b.String(), keys, labels
}

// kindsRepeat reports whether two options share a kind.
func kindsRepeat(opts []acpclient.PermissionOption) bool {
	seen := map[string]bool{}
	for _, o := range opts {
		if seen[o.Kind] {
			return true
		}
		seen[o.Kind] = true
	}
	return false
}

// chooseKey reads keys with read until one of keys is pressed, and gives
// its option; c, Ctrl+C, the end of input and a read error give "", a
// cancel. A key that is not shown is ignored.
func chooseKey(keys map[byte]string, read func([]byte) (int, error)) string {
	buf := make([]byte, 1)
	for {
		n, err := read(buf)
		if err != nil || n == 0 || buf[0] == 'c' || buf[0] == ctrlC {
			return ""
		}
		if id, ok := keys[buf[0]]; ok {
			return id
		}
	}
}

// labelOf is the word echoed for the chosen option: its label, or cancel.
func labelOf(labels map[string]string, id string) string {
	if l, ok := labels[id]; ok && id != "" {
		return l
	}
	return "cancel"
}

// readWhy reads a reason in raw mode up to Enter, echoing each character
// with echo and erasing on backspace, and gives it trimmed and cut to
// maxWhy; Ctrl+C, the end of input and a read error give false.
func readWhy(read func([]byte) (int, error), echo func(string)) (string, bool) {
	var line []rune
	buf := make([]byte, 64)
	var pending []byte
	for {
		n, err := read(buf)
		if err != nil || n == 0 {
			return "", false
		}
		pending = append(pending, buf[:n]...)
		for len(pending) > 0 {
			c, size := utf8.DecodeRune(pending)
			if c == utf8.RuneError && !utf8.FullRune(pending) {
				break
			}
			pending = pending[size:]
			switch {
			case c == ctrlC:
				return "", false
			case c == '\r' || c == '\n':
				return cutWhy(strings.TrimSpace(string(line))), true
			case c == backspace || c == del:
				if len(line) > 0 {
					line = line[:len(line)-1]
					echo("\b \b")
				}
			case c >= ' ':
				line = append(line, c)
				echo(term.Sanitize(string(c)))
			}
		}
	}
}

// cutWhy is s cut to maxWhy bytes on a rune boundary.
func cutWhy(s string) string {
	if len(s) <= maxWhy {
		return s
	}
	cut := maxWhy
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// askPermission is the line session's answer to session/request_permission
// (0008-MADR D16): why a question offers "once" alone, when it says, then
// the line on stderr, then one key read in raw mode. A key that is not
// shown is ignored; c, Ctrl+C and the end of input cancel. [f] then reads
// the reason, echoed as it is typed, up to Enter, and rejects once with it;
// Ctrl+C or the end of input cancels instead (F6-2b decisions A and F).
func (ls *lineSession) askPermission(ctx context.Context, r acpclient.PermissionRequest) acpclient.PermissionAnswer {
	ls.mu.Lock()
	v, keysIn := ls.view, ls.keys
	ls.mu.Unlock()
	if v != nil {
		v.quiet()
	}
	if keysIn == nil {
		return acpclient.PermissionAnswer{}
	}
	line, keys, labels := permissionLine(r)
	errw := ls.e.out.err
	if r.OnceOnly != "" {
		line = "(" + strings.Join(strings.Fields(term.Sanitize(r.OnceOnly)), " ") + ")\n" + line
	}
	if _, err := fmt.Fprint(errw, line); err != nil {
		return acpclient.PermissionAnswer{}
	}
	restore, err := ls.raw()
	if err != nil {
		return acpclient.PermissionAnswer{}
	}
	read := func(b []byte) (int, error) { return keysIn.ReadContext(ctx, b) }
	answer := acpclient.PermissionAnswer{OptionID: chooseKey(keys, read)}
	var werr error
	shown := line
	if answer.OptionID == chooseWhy {
		_, werr = fmt.Fprintf(errw, "%s: ", labelOf(labels, chooseWhy))
		why, ok := readWhy(read, func(s string) { fmt.Fprint(errw, s) }) //nolint:errcheck // the echo is best effort
		shown += labelOf(labels, chooseWhy) + ": " + why
		answer = acpclient.PermissionAnswer{}
		if ok {
			answer = acpclient.PermissionAnswer{OptionID: acpserver.RejectOnceID, Feedback: why}
			labels[acpserver.RejectOnceID] = ""
		}
	}
	rerr := restore()
	if rec := ls.e.out.rec; rec != nil { // the question and its answer, as shown (0030-MADR D4)
		rec.fixed(shown + labelOf(labels, answer.OptionID) + "\n")
	}
	if _, err := fmt.Fprintf(errw, "%s\n", labelOf(labels, answer.OptionID)); err != nil || rerr != nil || werr != nil {
		ls.e.log().WarnContext(ctx, "permission prompt", slog.String("write_error", fmt.Sprint(err)), slog.String("restore_error", fmt.Sprint(rerr)))
	}
	return answer
}
