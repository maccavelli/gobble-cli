package auth

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
)

var urnUUID = regexp.MustCompile(`^urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// The host id is made once and read ever after; callers racing to make it
// agree on one; the file is the user's alone (0052-PLAN P2-1).
func TestHostIDIsStable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	first, err := HostID(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !urnUUID.MatchString(first) {
		t.Fatalf("HostID = %q, want a version 4 urn:uuid", first)
	}
	if again, err := HostID(dir); err != nil || again != first {
		t.Errorf("second call: %q, %v; want %q", again, err, first)
	}
	path := filepath.Join(dir, "host-id")
	if ok, err := appdirs.FileIsOwnerOnly(path); err != nil || !ok {
		t.Errorf("owner-only %v, %v; want the user's alone", ok, err)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("%s: %v, %v; want 0600", path, fi, err)
		}
	}

	const callers = 8
	for round := range 25 {
		race := filepath.Join(t.TempDir(), "state")
		ids := make([]string, callers)
		errs := make([]error, callers)
		var ready, done sync.WaitGroup
		start := make(chan struct{})
		ready.Add(callers)
		done.Add(callers)
		for i := range callers {
			go func() {
				defer done.Done()
				ready.Done()
				<-start
				ids[i], errs[i] = HostID(race)
			}()
		}
		ready.Wait()
		close(start)
		done.Wait()
		for i := range callers {
			if errs[i] != nil {
				t.Fatalf("round %d caller %d: %v", round, i, errs[i])
			}
			if ids[i] != ids[0] {
				t.Fatalf("round %d: caller %d got %q, caller 0 got %q; want one id", round, i, ids[i], ids[0])
			}
		}
		if got, err := HostID(race); err != nil || got != ids[0] {
			t.Fatalf("round %d: read back %q, %v; want %q", round, got, err, ids[0])
		}
	}
}

// A file that holds something other than an id is an error, not replaced.
func TestHostIDRefusesOtherContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "host-id")
	if err := os.WriteFile(path, []byte("not an id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if id, err := HostID(dir); err == nil {
		t.Errorf("HostID = %q, nil; want an error", id)
	}
	if got, _ := os.ReadFile(path); string(got) != "not an id\n" {
		t.Errorf("the file became %q", got)
	}
}
