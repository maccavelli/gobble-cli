package builtin

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/tool"
)

// grep over a directory holding a FIFO skips it, and returns the other
// file's match at once (0037-MADR D7).
func TestGrepSkipsFIFO(t *testing.T) {
	dir := project(t, map[string]string{"a.txt": "hit\n"})
	fifo(t, filepath.Join(dir, "pipe"))
	o := runBounded(t, Grep(), tool.Env{Cwd: dir, Roots: []string{dir}}, `{"pattern":"hit"}`, time.Second)
	if o.err != nil {
		t.Fatal(o.err)
	}
	wantText(t, o.r, "a.txt:1: hit")
}

// grep naming a FIFO is an error, not a skip that reads as no matches
// (0037-PLAN deviation 6).
func TestGrepFileRefusesFIFO(t *testing.T) {
	dir := project(t, map[string]string{"a.txt": "hit\n"})
	fifo(t, filepath.Join(dir, "pipe"))
	o := runBounded(t, Grep(), tool.Env{Cwd: dir, Roots: []string{dir}}, `{"pattern":"hit","path":"pipe"}`, time.Second)
	if o.err != nil || !o.r.IsError || o.r.Text() != "grep pipe: not a regular file" {
		t.Fatalf("grep of a FIFO = %q (isError %v), %v; want the error \"grep pipe: not a regular file\"", o.r.Text(), o.r.IsError, o.err)
	}
}
