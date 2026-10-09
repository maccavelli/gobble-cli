package builtin

import (
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
	moveCopy = func(ws fsx.Workspace, from, to string, info fs.FileInfo) (int, error) {
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
