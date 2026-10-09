package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/tool"
)

const (
	defaultFindLimit = 1000
	maxFindLimit     = 5000
)

type findIn struct {
	Pattern string `json:"pattern" jsonschema:"the glob to match, such as *.go, *_test.go or docs/**/*.md; without a / it matches names at any depth"`
	Path    string `json:"path,omitzero" jsonschema:"the directory to search under, relative to the working directory or absolute; the working directory when not given"`
	Limit   int    `json:"limit,omitzero" jsonschema:"the most paths to show"`
}

// Find is the find tool: the files and directories under a directory whose
// paths match a glob, with the repository's ignore rules (0005-PLAN, entry
// "F1c made executable").
func Find() tool.Tool {
	return tool.New("find", description("find", Options{}.withDefaults()),
		runFind,
		tool.WithKind(tool.KindSearch),
		tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true, IdempotentHint: true}),
		tool.WithSchema(func(s *jsonschema.Schema) {
			nonEmpty(s, "pattern")
			bound(s, "limit", 1, maxFindLimit, defaultFindLimit)
		}),
		tool.WithOutside(outsidePath),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string {
			return title("find " + argString(c, "pattern") + " in " + shown(env, orDot(argPath(c))))
		}),
	)
}

func runFind(ctx context.Context, in findIn, env tool.Env) (tool.Result, error) {
	glob, err := fsx.CompileGlob(in.Pattern)
	if err != nil {
		return tool.Result{}, fmt.Errorf("find: the glob %q is malformed; check its [ ] classes and { } groups", in.Pattern)
	}
	s, err := openSearch(env, "find", in.Path)
	if err != nil {
		return tool.Result{}, err
	}
	defer s.close()
	if s.walker == nil {
		return tool.Result{}, fmt.Errorf("find %s: not a directory; find searches under a directory, and read shows a file", s.shown)
	}
	limit := clampInt(in.Limit, 1, maxFindLimit, defaultFindLimit)
	var lines []string
	size, more, cut := 0, false, false
	err = s.walker.Walk(func(rel string, d fs.DirEntry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !glob.Match(rel, d.IsDir()) {
			return nil
		}
		if len(lines) == limit {
			more = true
			return fs.SkipAll
		}
		if d.IsDir() {
			rel += "/"
		}
		if size+len(rel)+1 > maxBytes {
			cut = true
			return fs.SkipAll
		}
		lines = append(lines, rel)
		size += len(rel) + 1
		return nil
	})
	if err != nil {
		return tool.Result{}, err
	}
	var notes []string
	if more {
		notes = append(notes, limitNote(limit, maxFindLimit, "paths", "narrow the pattern or path"))
	}
	if cut {
		notes = append(notes, fmt.Sprintf("[output cut at %d KB; narrow the pattern or path]", maxBytes/1024))
	}
	body := strings.Join(lines, "\n")
	if len(lines) == 0 {
		body = fmt.Sprintf("find %s: nothing matches %s", s.shown, in.Pattern)
	}
	r := tool.TextResult(joinNotes(body, notes))
	r.Summary = summary(fmt.Sprintf("find %s: %s", s.shown, count(len(lines), "path", "paths")))
	return r, nil
}
