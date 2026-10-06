package builtin

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/maccavelli/gobble-cli/tool"
)

type readIn struct {
	Path string `json:"path" jsonschema:"the file to read, relative to the working directory or absolute"`
}

// Read is the read tool: a text file's first 2,000 lines or 50 KB.
func Read() tool.Tool {
	return tool.New("read", "Read a text file. Output is cut at 2000 lines or 50 KB, with a note.",
		func(_ context.Context, in readIn, env tool.Env) (tool.Result, error) {
			path := resolve(env, in.Path)
			b, err := os.ReadFile(path) //nolint:gosec // G304: the model names the file; 0005-PLAN F1 adds confinement
			if err != nil {
				return tool.Result{}, fmt.Errorf("read %s: %w", shown(env, in.Path), unwrapPath(err))
			}
			r := tool.TextResult(head(string(b)))
			r.Summary = summary(fmt.Sprintf("read %s (%d lines)", shown(env, in.Path), countLines(string(b))))
			return r, nil
		},
		tool.WithKind(tool.KindRead),
		tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true, IdempotentHint: true}),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("read " + shown(env, argPath(c))) }),
	)
}

type writeIn struct {
	Path    string `json:"path" jsonschema:"the file to write"`
	Content string `json:"content" jsonschema:"the file's whole new content"`
}

// Write is the write tool: it replaces or creates a file, and its parent
// directories.
func Write() tool.Tool {
	return tool.New("write", "Create or overwrite a text file with the given content.",
		func(_ context.Context, in writeIn, env tool.Env) (tool.Result, error) {
			path := resolve(env, in.Path)
			old, err := readOld(path)
			if err != nil {
				return tool.Result{}, fmt.Errorf("write %s: %w", shown(env, in.Path), unwrapPath(err))
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				return tool.Result{}, fmt.Errorf("write %s: %w", shown(env, in.Path), unwrapPath(err))
			}
			if err := os.WriteFile(path, []byte(in.Content), 0o600); err != nil {
				return tool.Result{}, fmt.Errorf("write %s: %w", shown(env, in.Path), unwrapPath(err))
			}
			msg := fmt.Sprintf("wrote %s (%d lines)", shown(env, in.Path), countLines(in.Content))
			r := tool.TextResult(msg)
			r.Summary = summary(msg)
			r.Diffs = []tool.Diff{{Path: path, OldText: old, NewText: in.Content}}
			return r, nil
		},
		tool.WithKind(tool.KindEdit),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true}),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("write " + shown(env, argPath(c))) }),
	)
}

type editIn struct {
	Path    string `json:"path" jsonschema:"the file to edit"`
	OldText string `json:"oldText" jsonschema:"text that occurs exactly once in the file"`
	NewText string `json:"newText" jsonschema:"the text to put in its place"`
}

// Edit is the edit tool: it replaces text that occurs exactly once.
func Edit() tool.Tool {
	return tool.New("edit", "Replace oldText, which must occur exactly once in the file, with newText.",
		func(_ context.Context, in editIn, env tool.Env) (tool.Result, error) {
			path := resolve(env, in.Path)
			b, err := os.ReadFile(path) //nolint:gosec // G304: as read
			if err != nil {
				return tool.Result{}, fmt.Errorf("edit %s: %w", shown(env, in.Path), unwrapPath(err))
			}
			old := string(b)
			switch n := strings.Count(old, in.OldText); {
			case in.OldText == "":
				return tool.Result{}, errors.New("edit: oldText is empty")
			case n == 0:
				return tool.Result{}, fmt.Errorf("edit %s: oldText was not found", shown(env, in.Path))
			case n > 1:
				return tool.Result{}, fmt.Errorf("edit %s: oldText occurs %d times; give more context so it occurs once", shown(env, in.Path), n)
			}
			updated := strings.Replace(old, in.OldText, in.NewText, 1)
			info, err := os.Stat(path)
			if err != nil {
				return tool.Result{}, fmt.Errorf("edit %s: %w", shown(env, in.Path), unwrapPath(err))
			}
			if err := os.WriteFile(path, []byte(updated), info.Mode().Perm()); err != nil { //nolint:gosec // G703: the model names the file, behind the session's approval; 0005-PLAN F1 adds confinement
				return tool.Result{}, fmt.Errorf("edit %s: %w", shown(env, in.Path), unwrapPath(err))
			}
			msg := fmt.Sprintf("edited %s: replaced %d lines with %d", shown(env, in.Path), countLines(in.OldText), countLines(in.NewText))
			r := tool.TextResult(msg)
			r.Summary = summary(msg)
			r.Diffs = []tool.Diff{{Path: path, OldText: &old, NewText: updated}}
			return r, nil
		},
		tool.WithKind(tool.KindEdit),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true}),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("edit " + shown(env, argPath(c))) }),
	)
}

// readOld is a file's content before a write, or nil when it does not exist.
func readOld(path string) (*string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // G304: as read
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	}
	s := string(b)
	return &s, nil
}

// unwrapPath drops the absolute path an *fs.PathError repeats, since the
// message already names the file as the model gave it.
func unwrapPath(err error) error {
	if pe, ok := errors.AsType[*fs.PathError](err); ok {
		return pe.Err
	}
	return err
}

// argPath is a call's "path" argument, for its title before it runs.
func argPath(c tool.Call) string {
	var a struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(c.Args, &a); err != nil || a.Path == "" {
		return "?"
	}
	return a.Path
}
