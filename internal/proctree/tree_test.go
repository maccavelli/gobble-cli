package proctree_test

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/proctree"
)

// helperEnv selects what the test binary does when it is run as a helper:
// "parent" starts a "child" of itself, prints the child's pid, and waits;
// "child" sleeps.
const helperEnv = "GOBBLE_PROCTREE_HELPER"

func TestMain(m *testing.M) {
	switch os.Getenv(helperEnv) {
	case "parent":
		child := exec.CommandContext(context.Background(), os.Args[0]) //nolint:gosec // G204: the test binary itself
		child.Env = append(os.Environ(), helperEnv+"=child")
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		os.Stdout.WriteString(strconv.Itoa(child.Process.Pid) + "\n") //nolint:errcheck // the test reads it
		_ = child.Wait()                                              //nolint:errcheck // the tree is stopped from outside
		os.Exit(0)
	case "child":
		time.Sleep(60 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// startTree starts the parent helper as a tree and returns it with the pid
// of the grandchild it started.
func startTree(t *testing.T) (*exec.Cmd, *proctree.Tree, int) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), os.Args[0]) //nolint:gosec // G204: the test binary itself
	cmd.Env = append(os.Environ(), helperEnv+"=parent")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	proctree.Prepare(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	tree, err := proctree.Attach(cmd.Process)
	if err != nil {
		_ = cmd.Process.Kill() //nolint:errcheck // the test fails with err
		t.Fatal(err)
	}
	t.Cleanup(tree.Release)
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatal(err)
	}
	return cmd, tree, pid
}

// gone waits up to 5 s for pid to exit.
func gone(pid int) bool {
	for range 50 {
		if !alive(pid) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// Stopping the tree stops the grandchild too, which stopping only the
// process it started would not.
func TestTerminateReachesGrandchild(t *testing.T) {
	cmd, tree, child := startTree(t)
	if !alive(child) {
		t.Fatal("the grandchild did not start")
	}
	tree.Terminate()
	_ = cmd.Wait() //nolint:errcheck // a terminated process exits with an error
	if !gone(child) {
		t.Fatalf("grandchild %d outlived Terminate", child)
	}
}

func TestKillReachesGrandchild(t *testing.T) {
	cmd, tree, child := startTree(t)
	tree.Kill()
	_ = cmd.Wait() //nolint:errcheck // a killed process exits with an error
	if !gone(child) {
		t.Fatalf("grandchild %d outlived Kill", child)
	}
}
