package builtin

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/tool"
)

type writeIn struct {
	Path    string `json:"path" jsonschema:"the file to write, relative to the working directory or absolute"`
	Content string `json:"content" jsonschema:"the whole new content of the file"`
}

// Write is the write tool: it replaces or creates a file, and its parent
// directories. The write is atomic, and holds the file's mutation lock.
func Write() tool.Tool {
	return tool.New("write", description("write", Options{}.withDefaults()),
		func(ctx context.Context, in writeIn, env tool.Env) (tool.Result, error) {
			abs := resolve(env, in.Path)
			unlock := fsx.Lock(abs)
			defer unlock()
			io := textIOFor(ctx, env, workspace(env), abs)
			old, err := readOld(io, abs)
			if err != nil {
				return tool.Result{}, fmt.Errorf("write %s: %w", shown(env, in.Path), unwrapPath(err))
			}
			if err := io.write(abs, []byte(in.Content)); err != nil {
				return tool.Result{}, fmt.Errorf("write %s: %w", shown(env, in.Path), unwrapPath(err))
			}
			msg := fmt.Sprintf("wrote %s (%s)", shown(env, in.Path), count(countLines(in.Content), "line", "lines"))
			r := tool.TextResult(msg)
			r.Summary = summary(msg)
			r.Diffs = []tool.Diff{{Path: abs, OldText: old, NewText: in.Content}}
			return r, nil
		},
		tool.WithKind(tool.KindEdit),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true}),
		tool.WithOutside(outsidePath),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("write " + shown(env, titlePath(c))) }),
		tool.WithLocations(argLocations("path")),
	)
}

// readOld is a file's content before a write, or nil when it does not exist.
func readOld(io textIO, abs string) (*string, error) {
	b, err := io.read(abs)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	}
	s := string(b)
	return &s, nil
}
