package builtin

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/tool"
)

// maxSuggestions is how many similar names a missing file's error offers.
const maxSuggestions = 3

// readDir is read of a directory: its entries sorted without regard to
// case, a "/" after each directory (a symlink counts as what it points
// at), paged by offset and limit like a file's lines.
func readDir(ws fsx.Workspace, abs, name string, in readIn) (tool.Result, error) {
	entries, err := ws.ReadDir(abs)
	if err != nil {
		return tool.Result{}, fmt.Errorf("read %s: %w", name, unwrapPath(err))
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		n := e.Name()
		dir := e.IsDir()
		if e.Type()&fs.ModeSymlink != 0 {
			if info, err := ws.Stat(filepath.Join(abs, n)); err == nil {
				dir = info.IsDir()
			}
		}
		if dir {
			n += "/"
		}
		names = append(names, n)
	}
	slices.SortFunc(names, func(a, b string) int {
		if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
			return c
		}
		return strings.Compare(a, b)
	})
	total := len(names)
	if total == 0 {
		r := tool.TextResult("(empty directory)")
		r.Summary = summary("listed " + name + " (empty)")
		return r, nil
	}
	first := max(in.Offset, 1)
	if first > total {
		return tool.Result{}, fmt.Errorf("read %s: offset %d is past the end of the directory (%s)", name, in.Offset, count(total, "entry", "entries"))
	}
	limit := maxEntries
	if in.Limit > 0 {
		limit = min(in.Limit, maxEntries)
	}
	last := min(total, first+limit-1)
	var b strings.Builder
	for _, n := range names[first-1 : last] {
		b.WriteString(n + "\n")
	}
	b.WriteString(footer("entries", first, last, total, false))
	r := tool.TextResult(b.String())
	r.Summary = summary(fmt.Sprintf("listed %s (%s)", name, count(total, "entry", "entries")))
	return r, nil
}

// maxEntries is a directory listing's page, as ls's default (0005-MADR).
const maxEntries = 500

// didYouMean is "; did you mean …?" naming up to maxSuggestions entries of
// abs's directory whose names contain the name asked for, or are contained
// in it, without regard to case; or "" when there are none.
func didYouMean(env tool.Env, ws fsx.Workspace, abs string) string {
	dir, want := filepath.Dir(abs), strings.ToLower(filepath.Base(abs))
	entries, err := ws.ReadDir(dir)
	if err != nil {
		return ""
	}
	var found []string
	for _, e := range entries {
		got := strings.ToLower(e.Name())
		if strings.Contains(got, want) || strings.Contains(want, got) {
			found = append(found, shown(env, filepath.Join(dir, e.Name())))
		}
		if len(found) == maxSuggestions {
			break
		}
	}
	if len(found) == 0 {
		return ""
	}
	return "; did you mean " + strings.Join(found, " or ") + "?"
}
