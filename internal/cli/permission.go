package cli

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// permissionKeys are D16's keys, by ACP option kind, in the order shown.
var permissionKeys = []struct {
	key   byte
	kind  string
	label string
}{
	{'y', "allow_once", "allow once"},
	{'a', "allow_always", "always"},
	{'n', "reject_once", "deny"},
	{'N', "reject_always", "never"},
}

// ctrlC is Ctrl+C in raw mode.
const ctrlC = 0x03

// permissionLine is D16's one line: the call's title, then the keys of the
// options offered, then [c] cancel. keys maps each shown key to its
// option.
func permissionLine(r acpclient.PermissionRequest) (string, map[byte]string) {
	keys := map[byte]string{}
	var b strings.Builder
	title := strings.Join(strings.Fields(term.Sanitize(r.Tool.Title)), " ")
	if title == "" {
		title = "Run the tool call"
	}
	fmt.Fprintf(&b, "%s?", title)
	for _, k := range permissionKeys {
		for _, o := range r.Options {
			if o.Kind == k.kind {
				if _, seen := keys[k.key]; !seen {
					keys[k.key] = o.ID
					fmt.Fprintf(&b, "  [%c] %s", k.key, k.label)
				}
			}
		}
	}
	b.WriteString("  [c] cancel ")
	return b.String(), keys
}

// askPermission is the line session's answer to session/request_permission
// (0008-MADR D16): the line on stderr, then one key read in raw mode. A key
// that is not shown is ignored; c, Ctrl+C and the end of input cancel.
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
	line, keys := permissionLine(r)
	errw := ls.e.out.err
	if _, err := fmt.Fprint(errw, line); err != nil {
		return acpclient.PermissionAnswer{}
	}
	restore, err := ls.raw()
	if err != nil {
		return acpclient.PermissionAnswer{}
	}
	answer := acpclient.PermissionAnswer{}
	buf := make([]byte, 1)
	for {
		n, err := keysIn.ReadContext(ctx, buf)
		if err != nil || n == 0 {
			break
		}
		if buf[0] == 'c' || buf[0] == ctrlC {
			break
		}
		if id, ok := keys[buf[0]]; ok {
			answer.OptionID = id
			break
		}
	}
	rerr := restore()
	choice := "cancel"
	for _, k := range permissionKeys {
		if id, ok := keys[k.key]; ok && id == answer.OptionID && answer.OptionID != "" {
			choice = k.label
		}
	}
	if _, err := fmt.Fprintf(errw, "%s\n", choice); err != nil || rerr != nil {
		ls.e.log().WarnContext(ctx, "permission prompt", slog.String("write_error", fmt.Sprint(err)), slog.String("restore_error", fmt.Sprint(rerr)))
	}
	return answer
}
