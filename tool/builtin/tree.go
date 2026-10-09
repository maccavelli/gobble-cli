package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/tool"
)

const (
	defaultTreeDepth = 2
	maxTreeDepth     = 10
	defaultTreeLimit = 500
	maxTreeLimit     = 2000
)

type treeIn struct {
	Path  string `json:"path,omitzero" jsonschema:"the directory to show, relative to the working directory or absolute; the working directory when not given"`
	Depth int    `json:"depth,omitzero" jsonschema:"how many levels below the directory to show"`
	Limit int    `json:"limit,omitzero" jsonschema:"the most entries to show"`
}

// Tree is the tree tool: a directory's layout, a few levels deep, with the
// repository's ignore rules. Its entry budget is spent breadth-first, so
// every entry of a level is listed before any of the next (0005-PLAN, entry
// "F1c made executable"; grok-build's list_dir, 0011-REPORT K2).
func Tree() tool.Tool {
	return tool.New("tree", description("tree", Options{}.withDefaults()),
		runTree,
		tool.WithKind(tool.KindRead),
		tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true, IdempotentHint: true}),
		tool.WithSchema(func(s *jsonschema.Schema) {
			bound(s, "depth", 1, maxTreeDepth, defaultTreeDepth)
			bound(s, "limit", 1, maxTreeLimit, defaultTreeLimit)
		}),
		tool.WithOutside(outsidePath),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("tree " + shown(env, orDot(argPath(c)))) }),
	)
}

// node is one entry of the tree.
type node struct {
	rel      string
	name     string
	dir      bool
	size     int64
	children []*node
	listed   bool // a directory whose entries were read
	more     int  // entries of a directory left out by the budget
}

func runTree(ctx context.Context, in treeIn, env tool.Env) (tool.Result, error) {
	s, err := openSearch(env, "tree", in.Path)
	if err != nil {
		return tool.Result{}, err
	}
	defer s.close()
	if s.walker == nil {
		return tool.Result{}, fmt.Errorf("tree %s: not a directory; read shows a file", s.shown)
	}
	depth := clampInt(in.Depth, 1, maxTreeDepth, defaultTreeDepth)
	limit := clampInt(in.Limit, 1, maxTreeLimit, defaultTreeLimit)
	root := &node{rel: ".", dir: true}
	budget, shownEntries, unlisted := limit, 0, 0
	level := []*node{root}
	for d := 1; d <= depth && len(level) > 0; d++ {
		var next []*node
		for _, n := range level {
			if err := ctx.Err(); err != nil {
				return tool.Result{}, err
			}
			if budget == 0 {
				unlisted++
				continue
			}
			entries, err := s.walker.ReadDir(n.rel)
			if err != nil {
				continue
			}
			n.listed = true
			for _, e := range sortedEntries(entries) {
				if budget == 0 {
					n.more++
					continue
				}
				c := &node{rel: path.Join(n.rel, e.Name()), name: e.Name(), dir: e.IsDir()}
				if info, err := e.Info(); err == nil && !c.dir {
					c.size = info.Size()
				}
				n.children = append(n.children, c)
				budget--
				shownEntries++
				if c.dir {
					next = append(next, c)
				}
			}
		}
		level = next
	}
	var b strings.Builder
	b.WriteString(strings.TrimSuffix(s.shown, "/") + "/\n")
	render(&b, root, 1, depth)
	var notes []string
	if budget == 0 && (unlisted > 0 || anyMore(root)) {
		notes = append(notes, limitNote(limit, maxTreeLimit, "entries", "tree a subdirectory"))
	}
	if unlisted > 0 {
		notes = append(notes, fmt.Sprintf("[%s not expanded; use a larger limit, or tree a subdirectory]", count(unlisted, "directory", "directories")))
	}
	r := tool.TextResult(joinNotes(strings.TrimSuffix(b.String(), "\n"), notes))
	r.Summary = summary(fmt.Sprintf("tree %s: %s", s.shown, count(shownEntries, "entry", "entries")))
	return r, nil
}

// sortedEntries is directories first, then files, each by name.
func sortedEntries(entries []fs.DirEntry) []fs.DirEntry {
	out := slices.Clone(entries)
	slices.SortStableFunc(out, func(a, b fs.DirEntry) int {
		switch {
		case a.IsDir() == b.IsDir():
			return strings.Compare(a.Name(), b.Name())
		case a.IsDir():
			return -1
		default:
			return 1
		}
	})
	return out
}

// render writes n's children, indented two spaces a level. A directory
// within the depth whose entries the budget left unread ends in "…".
func render(b *strings.Builder, n *node, level, depth int) {
	indent := strings.Repeat("  ", level)
	for _, c := range n.children {
		switch {
		case !c.dir:
			fmt.Fprintf(b, "%s%s (%s)\n", indent, c.name, fileSize(c.size))
		case !c.listed && level < depth:
			fmt.Fprintf(b, "%s%s/ …\n", indent, c.name)
		default:
			fmt.Fprintf(b, "%s%s/\n", indent, c.name)
			render(b, c, level+1, depth)
		}
	}
	if n.more > 0 {
		fmt.Fprintf(b, "%s… %d more\n", indent, n.more)
	}
}

// anyMore reports a directory with entries the budget left out.
func anyMore(n *node) bool {
	if n.more > 0 {
		return true
	}
	return slices.ContainsFunc(n.children, anyMore)
}

// fileSize is a file size for people: 12 B, 3.4 KB, 1.2 MB, 2.0 GB.
func fileSize(n int64) string {
	const k = 1024
	switch {
	case n < k:
		return fmt.Sprintf("%d B", n)
	case n < k*k:
		return fmt.Sprintf("%.1f KB", float64(n)/k)
	case n < k*k*k:
		return fmt.Sprintf("%.1f MB", float64(n)/(k*k))
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/(k*k*k))
	}
}
