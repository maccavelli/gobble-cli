package builtin

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/tool"
)

// The file operations of 0005-PLAN's entry "F1c made executable", F1c-2:
// native, so the permission rules and undo see their exact paths, and the
// same on every OS (0011-REPORT, "The rule").

type moveIn struct {
	From      string `json:"from" jsonschema:"the file or directory to move, relative to the working directory or absolute"`
	To        string `json:"to" jsonschema:"where it goes: the new path, not a directory to move it into"`
	Overwrite bool   `json:"overwrite,omitzero" jsonschema:"replace what is already at to"`
}

type deleteIn struct {
	Path      string `json:"path" jsonschema:"the file or directory to delete, relative to the working directory or absolute"`
	Recursive bool   `json:"recursive,omitzero" jsonschema:"delete a directory that is not empty, with everything in it"`
}

type copyIn struct {
	From      string `json:"from" jsonschema:"the file or directory to copy, relative to the working directory or absolute"`
	To        string `json:"to" jsonschema:"the path of the copy, not a directory to copy it into"`
	Overwrite bool   `json:"overwrite,omitzero" jsonschema:"replace what is already at to"`
}

type mkdirIn struct {
	Path string `json:"path" jsonschema:"the directory to create, with any missing parents, relative to the working directory or absolute"`
}

// Move is the move tool: a rename within one root and device, and a copy
// then a delete otherwise.
func Move() tool.Tool {
	return tool.New("move", description("move", Options{}.withDefaults()), runMove,
		tool.WithKind(tool.KindMove),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true}),
		tool.WithSchema(func(s *jsonschema.Schema) {
			nonEmpty(s, "from")
			nonEmpty(s, "to")
			property(s, "overwrite").Default = []byte("false")
		}),
		tool.WithOutside(twoPaths),
		tool.WithLocations(argLocations("from", "to")),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string {
			return title("move " + shown(env, argString(c, "from")) + " to " + shown(env, argString(c, "to")))
		}),
	)
}

// Delete is the delete tool. Every call asks (0005-MADR, native-tools
// amendment; F1c's facts: acpserver's policy has no "always").
func Delete() tool.Tool {
	return tool.New("delete", description("delete", Options{}.withDefaults()), runDelete,
		tool.WithKind(tool.KindDelete),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true}),
		tool.WithSchema(func(s *jsonschema.Schema) {
			nonEmpty(s, "path")
			property(s, "recursive").Default = []byte("false")
		}),
		tool.WithOutside(outsidePath),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("delete " + shown(env, titlePath(c))) }),
		tool.WithLocations(argLocations("path")),
	)
}

// Copy is the copy tool.
func Copy() tool.Tool {
	return tool.New("copy", description("copy", Options{}.withDefaults()), runCopy,
		tool.WithKind(tool.KindEdit),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true}),
		tool.WithSchema(func(s *jsonschema.Schema) {
			nonEmpty(s, "from")
			nonEmpty(s, "to")
			property(s, "overwrite").Default = []byte("false")
		}),
		tool.WithOutside(twoPaths),
		tool.WithLocations(argLocations("from", "to")),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string {
			return title("copy " + shown(env, argString(c, "from")) + " to " + shown(env, argString(c, "to")))
		}),
	)
}

// Mkdir is the mkdir tool. It only adds, so it is not destructive (F1c-2
// deviation 1).
func Mkdir() tool.Tool {
	return tool.New("mkdir", description("mkdir", Options{}.withDefaults()), runMkdir,
		tool.WithKind(tool.KindEdit),
		tool.WithAnnotations(tool.Annotations{IdempotentHint: true}),
		tool.WithSchema(func(s *jsonschema.Schema) { nonEmpty(s, "path") }),
		tool.WithOutside(outsidePath),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("mkdir " + shown(env, titlePath(c))) }),
		tool.WithLocations(argLocations("path")),
	)
}

// twoPaths is the Confined answer of move and copy: from and to, each when
// it is outside the roots.
func twoPaths(c tool.Call, env tool.Env) []string {
	ws := workspace(env)
	return append(outsideOf(ws, env, argString(c, "from")), outsideOf(ws, env, argString(c, "to"))...)
}

func runMove(_ context.Context, in moveIn, env tool.Env) (tool.Result, error) {
	ws := workspace(env)
	from, to := resolve(env, in.From), resolve(env, in.To)
	name := shown(env, in.From) + " to " + shown(env, in.To)
	unlock := fsx.LockPair(from, to)
	defer unlock()
	info, err := ws.Lstat(from)
	if err != nil {
		return tool.Result{}, missing(env, ws, "move", in.From, from, err)
	}
	if fsx.Inside(from, to) {
		return tool.Result{}, fmt.Errorf("move %s: a directory cannot move into itself", name)
	}
	if err := makeRoom(ws, "move", env, in.To, to, in.Overwrite); err != nil {
		return tool.Result{}, err
	}
	err = ws.Rename(from, to)
	if errors.Is(err, fsx.ErrCrossDevice) {
		if _, err := moveCopy(ws, from, to, info); err != nil {
			ws.RemoveAll(to) //nolint:errcheck,gosec // the partial copy; the source is untouched
			return tool.Result{}, fmt.Errorf("move %s: the copy failed, and %s is as it was: %w", name, shown(env, in.From), unwrapPath(err))
		}
		err = ws.RemoveAll(from)
	}
	if err != nil {
		return tool.Result{}, fmt.Errorf("move %s: %w", name, unwrapPath(err))
	}
	return opResult("moved " + name), nil
}

func runDelete(_ context.Context, in deleteIn, env tool.Env) (tool.Result, error) {
	ws := workspace(env)
	abs := resolve(env, in.Path)
	name := shown(env, in.Path)
	unlock := fsx.Lock(abs)
	defer unlock()
	info, err := ws.Lstat(abs)
	if err != nil {
		return tool.Result{}, missing(env, ws, "delete", in.Path, abs, err)
	}
	if ws.IsRoot(abs) {
		return tool.Result{}, fmt.Errorf("delete %s: a workspace root cannot be deleted", name)
	}
	if !info.IsDir() {
		if err := ws.Remove(abs); err != nil {
			return tool.Result{}, fmt.Errorf("delete %s: %w", name, unwrapPath(err))
		}
		return opResult("deleted " + name), nil
	}
	n, err := countEntries(ws, abs)
	if err != nil {
		return tool.Result{}, fmt.Errorf("delete %s: %w", name, unwrapPath(err))
	}
	if n > 0 && !in.Recursive {
		return tool.Result{}, fmt.Errorf("delete %s: the directory is not empty (%s); set recursive to delete it with everything in it", name, count(n, "entry", "entries"))
	}
	if err := ws.RemoveAll(abs); err != nil {
		return tool.Result{}, fmt.Errorf("delete %s: %w", name, unwrapPath(err))
	}
	return opResult(fmt.Sprintf("deleted %s/ (%s)", trimSlash(name), count(n, "entry", "entries"))), nil
}

func runCopy(_ context.Context, in copyIn, env tool.Env) (tool.Result, error) {
	ws := workspace(env)
	from, to := resolve(env, in.From), resolve(env, in.To)
	name := shown(env, in.From) + " to " + shown(env, in.To)
	unlock := fsx.LockPair(from, to)
	defer unlock()
	info, err := ws.Lstat(from)
	if err != nil {
		return tool.Result{}, missing(env, ws, "copy", in.From, from, err)
	}
	if info.IsDir() && fsx.Inside(from, to) {
		return tool.Result{}, fmt.Errorf("copy %s: a directory cannot be copied into itself", name)
	}
	if err := makeRoom(ws, "copy", env, in.To, to, in.Overwrite); err != nil {
		return tool.Result{}, err
	}
	n, err := copyEntry(ws, from, to, info)
	if err != nil {
		return tool.Result{}, fmt.Errorf("copy %s: %w", name, unwrapPath(err))
	}
	msg := "copied " + name
	if info.IsDir() {
		msg += " (" + count(n, "entry", "entries") + ")"
	}
	return opResult(msg), nil
}

func runMkdir(_ context.Context, in mkdirIn, env tool.Env) (tool.Result, error) {
	ws := workspace(env)
	abs := resolve(env, in.Path)
	name := shown(env, in.Path)
	if info, err := ws.Lstat(abs); err == nil {
		if info.IsDir() {
			return opResult(name + " already exists"), nil
		}
		return tool.Result{}, fmt.Errorf("mkdir %s: a file of that name exists", name)
	}
	if err := ws.MkdirAll(abs); err != nil {
		return tool.Result{}, fmt.Errorf("mkdir %s: %w", name, unwrapPath(err))
	}
	return opResult("created " + name), nil
}

// makeRoom checks the destination of a move or a copy: refused when it
// exists and overwrite is not set, removed when it is, and its directory
// required to exist.
func makeRoom(ws fsx.Workspace, op string, env tool.Env, path, abs string, overwrite bool) error {
	name := shown(env, path)
	if _, err := ws.Lstat(abs); err == nil {
		if !overwrite {
			return fmt.Errorf("%s: %s already exists; set overwrite to replace it", op, name)
		}
		if ws.IsRoot(abs) {
			return fmt.Errorf("%s: %s is a workspace root, which cannot be replaced", op, name)
		}
		if err := ws.RemoveAll(abs); err != nil {
			return fmt.Errorf("%s: %s cannot be replaced: %w", op, name, unwrapPath(err))
		}
	}
	if dir, err := ws.Stat(filepath.Dir(abs)); err != nil || !dir.IsDir() {
		return fmt.Errorf("%s: the directory %s does not exist; create it with mkdir first", op, shown(env, filepath.Dir(path)))
	}
	return nil
}

// moveCopy is a cross-device move's copy. Only tests set it, to make the
// copy fail part way.
var moveCopy = copyEntry

// copyEntry copies from to to: a file atomically with its mode, a link as a
// link, a directory entry by entry. It returns the entries copied under a
// directory.
func copyEntry(ws fsx.Workspace, from, to string, info fs.FileInfo) (int, error) {
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		return 0, ws.CopyLink(from, to)
	case !info.IsDir():
		return 0, ws.CopyFile(from, to)
	}
	if err := ws.MkdirAll(to); err != nil {
		return 0, err
	}
	entries, err := ws.ReadDir(from)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		ei, err := ws.Lstat(filepath.Join(from, e.Name()))
		if err != nil {
			return n, err
		}
		sub, err := copyEntry(ws, filepath.Join(from, e.Name()), filepath.Join(to, e.Name()), ei)
		if err != nil {
			return n, err
		}
		n += 1 + sub
	}
	return n, nil
}

// countEntries is how many entries lie under the directory abs, at every
// depth.
func countEntries(ws fsx.Workspace, abs string) (int, error) {
	entries, err := ws.ReadDir(abs)
	if err != nil {
		return 0, err
	}
	n := len(entries)
	for _, e := range entries {
		if e.IsDir() {
			sub, err := countEntries(ws, filepath.Join(abs, e.Name()))
			if err != nil {
				return 0, err
			}
			n += sub
		}
	}
	return n, nil
}

// missing is the error of a source path that cannot be read: one that does
// not exist names similar entries, as read's does.
func missing(env tool.Env, ws fsx.Workspace, op, path, abs string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s %s: no such file or directory%s", op, shown(env, path), didYouMean(env, ws, abs))
	}
	return fmt.Errorf("%s %s: %w", op, shown(env, path), unwrapPath(err))
}

// opResult is a file operation's result: the line, and the same as the
// client's summary.
func opResult(line string) tool.Result {
	r := tool.TextResult(line)
	r.Summary = summary(line)
	return r
}

func trimSlash(s string) string {
	for len(s) > 1 && (s[len(s)-1] == '/' || s[len(s)-1] == '\\') {
		s = s[:len(s)-1]
	}
	return s
}
