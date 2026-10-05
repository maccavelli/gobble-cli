package complete

import (
	"context"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// maxPathCandidates bounds a directory scan (0008-MADR D17).
const maxPathCandidates = 200

// EnumSource completes a fixed list, in its order. Empty entries are
// skipped: an enum may list "" to allow an unset flag.
func EnumSource(values []string) Completer {
	return CompleterFunc(func(ctx context.Context, _ string) (iter.Seq[Candidate], Directive) {
		return func(yield func(Candidate) bool) {
			for _, v := range values {
				if v != "" && (ctx.Err() != nil || !yield(Candidate{Value: v})) {
					return
				}
			}
		}, DirectiveKeepOrder
	})
}

// Lister lists the candidates of a dynamic source, such as sessions or
// models, from local files only.
type Lister func(ctx context.Context) iter.Seq[Candidate]

// NoCandidates is the lister of a source whose data does not exist yet. It
// is the true state, not a skip: there are no sessions before 0002-PLAN
// Phase 4, and no provider or model catalog before 0005-PLAN F4.
func NoCandidates(context.Context) iter.Seq[Candidate] {
	return func(func(Candidate) bool) {}
}

// ListSource completes from a lister. keepOrder keeps the lister's order,
// such as sessions newest first.
func ListSource(list Lister, keepOrder bool) Completer {
	d := Directive(0)
	if keepOrder {
		d = DirectiveKeepOrder
	}
	return CompleterFunc(func(ctx context.Context, _ string) (iter.Seq[Candidate], Directive) {
		return list(ctx), d
	})
}

// ModelSource completes provider/id from list and, after a ':', the
// thinking levels.
func ModelSource(list Lister, levels []string) Completer {
	return CompleterFunc(func(ctx context.Context, prefix string) (iter.Seq[Candidate], Directive) {
		model, _, hasLevel := strings.Cut(prefix, ":")
		if !hasLevel {
			return list(ctx), DirectiveNoSpace
		}
		return func(yield func(Candidate) bool) {
			for _, l := range levels {
				if l != "" && !yield(Candidate{Value: model + ":" + l}) {
					return
				}
			}
		}, DirectiveKeepOrder
	})
}

// PathSource completes files and directories below the directory the prefix
// names, or directories only. It reads at most maxPathCandidates matching
// entries and writes nothing. Directories end in a separator: the one the
// user typed, else the OS's.
func PathSource(dirsOnly bool) Completer {
	return CompleterFunc(func(ctx context.Context, prefix string) (iter.Seq[Candidate], Directive) {
		d := DirectiveNoSpace
		if dirsOnly {
			d |= DirectiveFilterDirs
		}
		return pathCandidates(ctx, prefix, dirsOnly), d
	})
}

func pathCandidates(ctx context.Context, prefix string, dirsOnly bool) iter.Seq[Candidate] {
	return func(yield func(Candidate) bool) {
		typedDir, base := splitPath(prefix)
		readDir := expandHome(typedDir)
		if readDir == "" {
			readDir = "."
		}
		f, err := os.Open(readDir) //nolint:gosec // G304: the directory the user is completing, read only
		if err != nil {
			return
		}
		defer f.Close() //nolint:errcheck // a directory opened only for reading
		sep := string(filepath.Separator)
		if strings.Contains(typedDir, "/") {
			sep = "/"
		}
		n := 0
		for n < maxPathCandidates && ctx.Err() == nil {
			entries, err := f.ReadDir(64)
			for _, e := range entries {
				name := e.Name()
				if !hasPathPrefix(name, base) || (strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".")) {
					continue
				}
				isDir := e.IsDir()
				if e.Type()&os.ModeSymlink != 0 {
					if fi, err := os.Stat(filepath.Join(readDir, name)); err == nil {
						isDir = fi.IsDir()
					}
				}
				if dirsOnly && !isDir {
					continue
				}
				c := Candidate{Value: typedDir + name, Kind: KindFile}
				if isDir {
					c.Value += sep
					c.Kind = KindDir
				}
				if !yield(c) {
					return
				}
				n++
				if n == maxPathCandidates {
					return
				}
			}
			if err != nil || len(entries) == 0 { // io.EOF ends the directory
				return
			}
		}
	}
}

// splitPath cuts prefix after its last separator: "a/b/c" is "a/b/" and
// "c". Windows accepts both separators.
func splitPath(prefix string) (dir, base string) {
	seps := "/"
	if runtime.GOOS == "windows" {
		seps = `/\`
	}
	i := strings.LastIndexAny(prefix, seps)
	return prefix[:i+1], prefix[i+1:]
}

// expandHome reads ~/ as the home directory, for reading only; candidates
// keep the ~ the user typed.
func expandHome(dir string) string {
	rest, ok := strings.CutPrefix(dir, "~")
	if !ok || (rest != "" && rest[0] != '/' && rest[0] != '\\') {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return dir
	}
	return home + rest
}

// hasPathPrefix matches names case-insensitively on Windows, whose file
// systems are.
func hasPathPrefix(name, prefix string) bool {
	if runtime.GOOS == "windows" {
		return len(name) >= len(prefix) && strings.EqualFold(name[:len(prefix)], prefix)
	}
	return strings.HasPrefix(name, prefix)
}
