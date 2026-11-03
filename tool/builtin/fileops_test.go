package builtin

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/tool"
)

// runEnv runs tl with args under env.
func runEnv(t *testing.T, tl tool.Tool, env tool.Env, args string) tool.Result {
	t.Helper()
	r, err := tl.Run(t.Context(), tool.Call{ID: "c1", Name: tl.Spec().Name, Args: jsontext.Value(args)}, env)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// move: within a root, across roots (a copy, then a delete), overwrite, and
// a failed copy that leaves the source as it was.
func TestMove(t *testing.T) {
	dir := project(t, map[string]string{"a.txt": "a", "d/x": "x", "taken": "t"})
	wantText(t, runIn(t, Move(), dir, `{"from":"a.txt","to":"b.txt"}`), "moved a.txt to b.txt")
	if exists(filepath.Join(dir, "a.txt")) || !exists(filepath.Join(dir, "b.txt")) {
		t.Fatal("the rename did not happen")
	}
	wantError(t, runIn(t, Move(), dir, `{"from":"b.txt","to":"taken"}`), "move: taken already exists; set overwrite to replace it")
	wantText(t, runIn(t, Move(), dir, `{"from":"b.txt","to":"taken","overwrite":true}`), "moved b.txt to taken")
	wantError(t, runIn(t, Move(), dir, `{"from":"d","to":"d/inner"}`), "a directory cannot move into itself")
	wantError(t, runIn(t, Move(), dir, `{"from":"taken","to":"no/such/dir/x"}`), "the directory no")
	wantError(t, runIn(t, Move(), dir, `{"from":"nope","to":"x"}`), "move nope: no such file or directory")

	other := t.TempDir()
	env := tool.Env{Cwd: dir, Roots: []string{dir, other}}
	r := runEnv(t, Move(), env, fmt.Sprintf(`{"from":"d","to":%q}`, filepath.Join(other, "d")))
	if r.IsError || exists(filepath.Join(dir, "d")) || !exists(filepath.Join(other, "d", "x")) {
		t.Fatalf("a move across roots gave %q; the source exists %v, the copy %v", r.Text(), exists(filepath.Join(dir, "d")), exists(filepath.Join(other, "d", "x")))
	}

	saved := moveCopy
	defer func() { moveCopy = saved }()
	moveCopy = func(_ context.Context, ws fsx.Workspace, from, to string, info fs.FileInfo) (int, error) {
		_ = ws.MkdirAll(to) //nolint:errcheck // a partial copy, for the move to remove
		return 0, errors.New("disk full")
	}
	r = runEnv(t, Move(), env, fmt.Sprintf(`{"from":%q,"to":"back"}`, filepath.Join(other, "d")))
	if !r.IsError || !strings.Contains(r.Text(), "the copy failed") || !exists(filepath.Join(other, "d", "x")) || exists(filepath.Join(dir, "back")) {
		t.Fatalf("a failed cross-device copy gave %q; the source exists %v, the partial copy %v", r.Text(), exists(filepath.Join(other, "d", "x")), exists(filepath.Join(dir, "back")))
	}
}

// delete: a file, an empty directory, a full one with and without
// recursive, and the root and its ancestors refused.
func TestDelete(t *testing.T) {
	dir := project(t, map[string]string{"f": "", "empty/": "", "full/a/b": "", "full/c": ""})
	wantText(t, runIn(t, Delete(), dir, `{"path":"f"}`), "deleted f")
	wantText(t, runIn(t, Delete(), dir, `{"path":"empty"}`), "deleted empty/ (0 entries)")
	wantError(t, runIn(t, Delete(), dir, `{"path":"full"}`), "delete full: the directory is not empty (3 entries); set recursive")
	wantText(t, runIn(t, Delete(), dir, `{"path":"full","recursive":true}`), "deleted full/ (3 entries)")
	for _, p := range []string{"f", "empty", "full"} {
		if exists(filepath.Join(dir, p)) {
			t.Errorf("%s is still there", p)
		}
	}
	wantError(t, runIn(t, Delete(), dir, `{"path":"."}`), "a workspace root cannot be deleted")
	env := tool.Env{Cwd: dir, Roots: []string{dir}, AllowOutside: true}
	wantError(t, runEnv(t, Delete(), env, fmt.Sprintf(`{"path":%q}`, filepath.Dir(dir))), "a workspace root cannot be deleted")
	wantError(t, runIn(t, Delete(), dir, `{"path":"gone"}`), "delete gone: no such file or directory")
}

// copy: a file and its mode, a directory, a link as a link, overwrite.
func TestCopy(t *testing.T) {
	dir := project(t, map[string]string{"run.sh": "#!/bin/sh\n", "tree/a": "a", "tree/sub/b": "b"})
	if err := os.Chmod(filepath.Join(dir, "run.sh"), 0o750); err != nil {
		t.Fatal(err)
	}
	wantText(t, runIn(t, Copy(), dir, `{"from":"run.sh","to":"run2.sh"}`), "copied run.sh to run2.sh")
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(filepath.Join(dir, "run2.sh")); err != nil || info.Mode().Perm() != 0o750 {
			t.Errorf("the copy's mode is %v, %v; want 0750", info.Mode().Perm(), err)
		}
	}
	wantText(t, runIn(t, Copy(), dir, `{"from":"tree","to":"tree2"}`), "copied tree to tree2 (3 entries)")
	if r := runEnv(t, Read(), tool.Env{Cwd: dir, Roots: []string{dir}}, `{"path":"tree2/sub/b"}`); r.IsError || !strings.HasPrefix(r.Text(), "1: b\n") {
		t.Fatalf("the copied tree's file reads %q", r.Text())
	}
	wantError(t, runIn(t, Copy(), dir, `{"from":"run.sh","to":"run2.sh"}`), "copy: run2.sh already exists; set overwrite to replace it")
	wantText(t, runIn(t, Copy(), dir, `{"from":"run.sh","to":"run2.sh","overwrite":true}`), "copied run.sh to run2.sh")
	wantError(t, runIn(t, Copy(), dir, `{"from":"tree","to":"tree/inner"}`), "a directory cannot be copied into itself")
	if err := os.Symlink("run.sh", filepath.Join(dir, "link")); err != nil {
		t.Logf("no symlink test here: %v", err)
		return
	}
	wantText(t, runIn(t, Copy(), dir, `{"from":"link","to":"link2"}`), "copied link to link2")
	if target, err := os.Readlink(filepath.Join(dir, "link2")); err != nil || target != "run.sh" {
		t.Fatalf("the copied link points at %q, %v", target, err)
	}
}

// mkdir: a new path with parents, an existing directory, an existing file.
func TestMkdir(t *testing.T) {
	dir := project(t, map[string]string{"file": ""})
	wantText(t, runIn(t, Mkdir(), dir, `{"path":"docs/guides"}`), "created docs/guides")
	if info, err := os.Stat(filepath.Join(dir, "docs", "guides")); err != nil || !info.IsDir() {
		t.Fatalf("docs/guides is %v, %v", info, err)
	}
	wantText(t, runIn(t, Mkdir(), dir, `{"path":"docs"}`), "docs already exists")
	wantError(t, runIn(t, Mkdir(), dir, `{"path":"file"}`), "mkdir file: a file of that name exists")
}

// Moves in opposite directions at once all finish: the lock order holds.
func TestMovesDoNotDeadlock(t *testing.T) {
	dir := project(t, map[string]string{})
	var wg sync.WaitGroup
	for i := range 40 {
		a, b := fmt.Sprintf("a%d", i), fmt.Sprintf("b%d", i)
		if err := os.WriteFile(filepath.Join(dir, a), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		for _, args := range []string{fmt.Sprintf(`{"from":%q,"to":%q}`, a, b), fmt.Sprintf(`{"from":%q,"to":%q}`, b, a)} {
			wg.Go(func() {
				_, _ = Move().Run(t.Context(), tool.Call{ID: "c", Name: "move", Args: jsontext.Value(args)}, tool.Env{Cwd: dir, Roots: []string{dir}}) //nolint:errcheck // only finishing matters
			})
		}
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("opposite moves deadlocked")
	}
}

// move and copy report both paths when they are outside the roots.
func TestFileOpsOutside(t *testing.T) {
	root, other := t.TempDir(), t.TempDir()
	env := tool.Env{Cwd: root, Roots: []string{root}}
	args := jsontext.Value(fmt.Sprintf(`{"from":%q,"to":%q}`, filepath.Join(other, "a"), filepath.Join(other, "b")))
	for _, tl := range []tool.Tool{Move(), Copy()} {
		if got := tl.(tool.Confined).Outside(tool.Call{Name: tl.Spec().Name, Args: args}, env); len(got) != 2 {
			t.Errorf("%s: Outside = %v, want both paths", tl.Spec().Name, got)
		}
	}
}

// An intercepted mv or cp into an existing directory puts its source inside
// it, as the shell does, and leaves what the directory holds (0037-MADR D1).
func TestInterceptIntoDirectory(t *testing.T) {
	for _, tc := range []struct {
		cmd, want, result string
	}{
		{"mv -f f.txt d", "d/f.txt", "moved f.txt to d/f.txt"},
		{"mv f.txt d", "d/f.txt", "moved f.txt to d/f.txt"},
		{"cp -f f.txt d", "d/f.txt", "copied f.txt to d/f.txt"},
		{"cp -r src d", "d/src/x", "copied src to d/src (1 entry)"},
	} {
		t.Run(tc.cmd, func(t *testing.T) {
			dir := project(t, map[string]string{"f.txt": "f", "src/x": "x", "d/precious.txt": "keep"})
			call, ok := interceptShell(tool.Call{ID: "c1", Name: nameBash, Args: shellArgs(t, tc.cmd, "")})
			if !ok {
				t.Fatalf("%q was not intercepted", tc.cmd)
			}
			tl := map[string]tool.Tool{"move": Move(), "copy": Copy()}[call.Name]
			r, err := tl.Run(t.Context(), call, tool.Env{Cwd: dir, Roots: []string{dir}, Rerouted: nameBash})
			if err != nil {
				t.Fatal(err)
			}
			wantText(t, r, tc.result)
			if b, err := os.ReadFile(filepath.Join(dir, "d", "precious.txt")); err != nil || string(b) != "keep" {
				t.Fatalf("d/precious.txt: %q, %v", b, err)
			}
			if !exists(filepath.Join(dir, filepath.FromSlash(tc.want))) {
				t.Fatalf("%s is missing", tc.want)
			}
		})
	}
}

// The model's own move or copy, overwrite or not, never removes a directory
// to put a file in its place (0037-MADR D1).
func TestMakeRoomKeepsDirectory(t *testing.T) {
	for _, tl := range []tool.Tool{Move(), Copy()} {
		for _, overwrite := range []bool{false, true} {
			dir := project(t, map[string]string{"f.txt": "f", "d/precious.txt": "keep"})
			args := fmt.Sprintf(`{"from":"f.txt","to":"d","overwrite":%v}`, overwrite)
			wantError(t, runIn(t, tl, dir, args), tl.Spec().Name+": d is a directory; a file cannot replace it")
			if !exists(filepath.Join(dir, "d", "precious.txt")) || !exists(filepath.Join(dir, "f.txt")) {
				t.Fatalf("%s with overwrite %v changed the files", tl.Spec().Name, overwrite)
			}
		}
	}
}

// runCtx runs tl with args under env and ctx.
func runCtx(ctx context.Context, tl tool.Tool, env tool.Env, args string) (tool.Result, error) {
	return tl.Run(ctx, tool.Call{ID: "c1", Name: tl.Spec().Name, Args: jsontext.Value(args)}, env)
}

// A copy of ws/a to the real path of ws, then a/b, is a copy into itself,
// refused, not a tree that grows until the deadline (0037-MADR F11, D11).
func TestCopyIntoItselfRealPath(t *testing.T) {
	base := t.TempDir()
	real, ws := filepath.Join(base, "real"), filepath.Join(base, "ws")
	if err := os.MkdirAll(filepath.Join(real, "a"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "a", "x"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, ws); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	to := filepath.Join(fsx.RealPath(ws), "a", "b")
	r, err := runCtx(ctx, Copy(), tool.Env{Cwd: ws, Roots: []string{ws}}, fmt.Sprintf(`{"from":"a","to":%q}`, to))
	if err != nil {
		t.Fatalf("copy gave %v, want its refusal", err)
	}
	wantError(t, r, "a directory cannot be copied into itself")
	if exists(to) {
		t.Fatalf("%s was made", to)
	}
}

// A symlink to a directory moves into it, and a symlink to the root is
// deleted, as before: each acts on the link, not on what it names
// (0037-PLAN-permission-and-file-tool-hardening.md, deviation 12).
func TestRealPathKeepsLinks(t *testing.T) {
	dir := project(t, map[string]string{"d/x": "x"})
	if err := os.Symlink("d", filepath.Join(dir, "l")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	if err := os.Symlink(dir, filepath.Join(dir, "rootlink")); err != nil {
		t.Fatal(err)
	}
	wantText(t, runIn(t, Move(), dir, `{"from":"l","to":"d/l2"}`), "moved l to d/l2")
	if info, err := os.Lstat(filepath.Join(dir, "d", "l2")); err != nil || info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("d/l2 is not the moved link: %v", err)
	}
	wantText(t, runIn(t, Delete(), dir, `{"path":"rootlink"}`), "deleted rootlink")
	if exists(filepath.Join(dir, "rootlink")) || !exists(filepath.Join(dir, "d", "x")) {
		t.Fatal("the delete did not remove the link alone")
	}
}

// A cancelled copy of a 2,000-entry tree stops and copies fewer entries,
// and a cancelled recursive delete removes none (0037-MADR D11).
func TestCopyHonoursCancel(t *testing.T) {
	files := map[string]string{}
	for d := range 40 {
		for f := range 49 {
			files[fmt.Sprintf("tree/d%02d/f%02d", d, f)] = "x"
		}
	}
	dir := project(t, files)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	env := tool.Env{Cwd: dir, Roots: []string{dir}}
	if _, err := runCtx(ctx, Copy(), env, `{"from":"tree","to":"copy"}`); !errors.Is(err, context.Canceled) {
		t.Fatalf("copy gave %v, want context.Canceled", err)
	}
	if n := entriesUnder(t, filepath.Join(dir, "copy")); n >= 2000 {
		t.Fatalf("the cancelled copy made %d entries, want fewer than 2000", n)
	}
	if _, err := runCtx(ctx, Delete(), env, `{"path":"tree","recursive":true}`); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete gave %v, want context.Canceled", err)
	}
	if n := entriesUnder(t, filepath.Join(dir, "tree")); n != 2000 {
		t.Fatalf("the cancelled delete left %d entries, want 2000", n)
	}
}

// entriesUnder is how many entries lie under dir, 0 when it does not exist.
func entriesUnder(t *testing.T, dir string) int {
	t.Helper()
	n := -1
	err := filepath.WalkDir(dir, func(_ string, _ fs.DirEntry, err error) error {
		n++
		return err
	})
	if errors.Is(err, fs.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return n
}
