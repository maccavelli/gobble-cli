package jsonl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/session"
)

// writerEnv makes the test binary a session writer for
// TestKilledWriterReopens, in mode "loop" or "torn"; dirEnv is its store's
// directory.
const (
	writerEnv = "GOBBLE_JSONL_WRITER"
	dirEnv    = "GOBBLE_JSONL_DIR"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(writerEnv); mode != "" {
		os.Exit(runWriter(mode, os.Getenv(dirEnv)))
	}
	os.Exit(m.Run())
}

// bigText is a large entry's text, so that a kill can land inside a write.
var bigText = strings.Repeat("x", 256<<10)

// maxLoop bounds the loop writer, should nothing kill it.
const maxLoop = 200

func entryID(i int) string { return fmt.Sprintf("e%07d", i) }

// runWriter creates session s1 in dir and appends to it, printing "ok <id>"
// once each Append returns nil. In mode "loop" it appends large entries
// until it is killed. In mode "torn" it appends three, writes half of a
// fourth entry's line past the store, as a crash mid-write leaves it, prints
// "torn", and waits to be killed.
func runWriter(mode, dir string) int {
	ctx := context.Background()
	l, err := New(dir, Options{}).Create(ctx, header("s1", "/w"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		return 2
	}
	last := 3
	if mode == "loop" {
		last = maxLoop
	}
	for i := 0; i <= last; i++ {
		parent, text := "", "first"
		if i > 0 {
			parent, text = entryID(i-1), bigText
		}
		if err := l.Append(ctx, userEntry(entryID(i), parent, text)); err != nil {
			fmt.Fprintln(os.Stderr, "append:", err)
			return 2
		}
		fmt.Println("ok " + entryID(i))
	}
	if mode == "loop" {
		fmt.Fprintln(os.Stderr, "loop writer was not killed")
		return 3
	}
	line, err := json.Marshal(userEntry(entryID(last+1), entryID(last), bigText))
	if err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		return 2
	}
	f, err := os.OpenFile(l.(*fileLog).path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		return 2
	}
	if _, err := f.Write(line[:len(line)/2]); err != nil {
		fmt.Fprintln(os.Stderr, "torn write:", err)
		return 2
	}
	fmt.Println("torn")
	time.Sleep(time.Minute)
	return 0
}

// A writer killed mid-append, or with half a line on disk, leaves a session
// that reopens with every entry it acknowledged, its torn tail cut and
// logged, and that takes the next append whole (0037-MADR D40; gobble-cli#7
// U3).
func TestKilledWriterReopens(t *testing.T) {
	for _, r := range []struct {
		mode string
		acks int
	}{{"loop", 3}, {"loop", 20}, {"torn", 4}} {
		t.Run(r.mode+"-"+strconv.Itoa(r.acks), func(t *testing.T) { killedWriter(t, r.mode, r.acks) })
	}
}

func killedWriter(t *testing.T, mode string, acks int) {
	dir := t.TempDir()
	ctx := t.Context()
	cmd := exec.CommandContext(ctx, os.Args[0]) //nolint:gosec // G204: the test binary itself
	cmd.Env = append(os.Environ(), writerEnv+"="+mode, dirEnv+"="+dir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var acked []string
	torn := false
	sc := bufio.NewScanner(out)
	read := func() {
		for sc.Scan() {
			if id, ok := strings.CutPrefix(sc.Text(), "ok "); ok {
				acked = append(acked, id)
			}
			torn = torn || sc.Text() == "torn"
			if torn || (mode == "loop" && len(acked) >= acks) {
				return
			}
		}
	}
	read()
	if !torn && len(acked) < acks {
		_ = cmd.Wait() //nolint:errcheck // the writer's stderr says why it stopped
		t.Fatalf("writer stopped after %d acknowledgements: %s", len(acked), stderr.String())
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	// What the writer printed before it died is acknowledged too: read it
	// all, then wait (os/exec: no Wait before the pipe's reads are done).
	for sc.Scan() {
		if id, ok := strings.CutPrefix(sc.Text(), "ok "); ok {
			acked = append(acked, id)
		}
	}
	_ = cmd.Wait() //nolint:errcheck // killed: the error is the kill

	var log bytes.Buffer
	s := New(dir, Options{Logger: slog.New(slog.NewTextHandler(&log, nil))})
	path, err := s.find("s1")
	if err != nil || path == "" {
		t.Fatalf("session file: %q, %v", path, err)
	}
	before := len(readBytes(t, path))
	l := openUnlocked(t, s, "s1")
	b := readBytes(t, path)

	entries := l.Entries()
	if len(entries) < len(acked) || len(entries) > len(acked)+1 {
		t.Errorf("%d entries after %d acknowledged appends, want %d or %d", len(entries), len(acked), len(acked), len(acked)+1)
	}
	// The file after Open, read as a reader reads it, holds what the log
	// holds: every acknowledged entry is on disk, not only in memory.
	_, onDisk, skipped, err := s.Read(ctx, "s1")
	if err != nil || skipped != 0 || len(onDisk) != len(entries) {
		t.Errorf("file after Open: %d entries, %d skipped, %v; the log holds %d", len(onDisk), skipped, err, len(entries))
	}
	for i, id := range acked {
		if i >= len(onDisk) || onDisk[i].ID != id {
			t.Errorf("acknowledged entry %s missing at %d in the file", id, i)
			break
		}
	}
	if len(entries) == len(acked)+1 && entries[len(acked)].ID != entryID(len(acked)) {
		t.Errorf("entry past the acknowledged ones is %s, want %s", entries[len(acked)].ID, entryID(len(acked)))
	}
	if !bytes.HasSuffix(b, []byte("\n")) {
		t.Errorf("file does not end in a newline after Open: ...%q", b[max(0, len(b)-40):])
	}
	cut := strings.Contains(log.String(), `msg="session: torn last line cut"`)
	switch {
	case mode == "torn" && !cut:
		t.Errorf("torn writer: no cut logged: %q", log.String())
	case cut && !strings.Contains(log.String(), "bytes="+strconv.Itoa(before-len(b))):
		t.Errorf("log %q, want bytes=%d", log.String(), before-len(b))
	case !cut && before != len(b):
		t.Errorf("file went from %d to %d bytes with no cut logged", before, len(b))
	}
	t.Logf("%s: %d acknowledged, %d entries, torn tail cut: %v", mode, len(acked), len(entries), cut)

	next := entryID(len(entries))
	if err := l.Append(ctx, userEntry(next, entries[len(entries)-1].ID, "after")); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	_, got, skipped, err := s.Read(ctx, "s1")
	if err != nil || skipped != 0 || len(got) != len(entries)+1 || got[len(got)-1].ID != next {
		t.Fatalf("read after the next append: %d entries, %d skipped, %v; want %d and 0", len(got), skipped, err, len(entries)+1)
	}
}

// openUnlocked opens a session whose writer was killed. Unix releases a dead
// holder's flock at once; Windows releases LockFileEx's lock after the
// process ends, with no stated bound. So ErrLocked alone is retried, for at
// most 5 s (0037-PLAN-session-store-durability.md C10).
func openUnlocked(t *testing.T, s *Store, id session.ID) session.Log {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		l, err := s.Open(t.Context(), id)
		if err == nil {
			return l
		}
		if _, locked := errors.AsType[*session.ErrLocked](err); !locked || time.Now().After(deadline) {
			t.Fatalf("open after the kill: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
