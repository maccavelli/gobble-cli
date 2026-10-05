package appdirs

// Provenance: magic-cli-remote internal/appdirs/owneronly_test.go and
// converge_test.go at 9778cbc1 (Apache-2.0). The converge test's
// "already owner-only" skip is not carried over: gobble's suites do not skip
// on host state (0008-PLAN C5), so only the deterministic cases are kept.

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestFileIsOwnerOnly: "private to me", asked once and answered per platform.
func TestFileIsOwnerOnly(t *testing.T) {
	// The file must live in a directory this package made private. A bare
	// t.TempDir() inherits whatever the platform's temp ACL grants, so a file
	// created there may genuinely NOT be owner-only on Windows.
	dir := filepath.Join(t.TempDir(), "private")
	if err := EnsurePrivateDir(dir); err != nil {
		t.Fatal(err)
	}

	tight := filepath.Join(dir, "tight")
	if err := os.WriteFile(tight, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	ok, err := FileIsOwnerOnly(tight)
	if err != nil {
		t.Fatalf("FileIsOwnerOnly(0600): %v", err)
	}
	if !ok {
		t.Error("a 0600 file reported as not owner-only")
	}

	// The negative case is Unix-only: on Windows the mode argument is ignored
	// and privacy comes from the DACL, so a "0666" file is still owner-only
	// until its ACL says otherwise.
	if runtime.GOOS != "windows" {
		loose := filepath.Join(dir, "loose")
		if err := os.WriteFile(loose, []byte("secret"), 0o666); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(loose, 0o666); err != nil { // past the umask
			t.Fatal(err)
		}
		ok, err := FileIsOwnerOnly(loose)
		if err != nil {
			t.Fatalf("FileIsOwnerOnly(0666): %v", err)
		}
		if ok {
			t.Error("a 0666 file reported as owner-only")
		}
	}
}

func TestFileIsOwnerOnlyRejectsBadInput(t *testing.T) {
	if _, err := FileIsOwnerOnly(""); err == nil {
		t.Error("accepted an empty path")
	}
	if _, err := FileIsOwnerOnly("relative"); err == nil {
		t.Error("accepted a relative path")
	}
	if _, err := FileIsOwnerOnly(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("accepted a missing path")
	}
}

// A second call must be a no-op, and a file created inside a converged
// directory must come out owner-only on both platforms.
func TestEnsurePrivateDirIsIdempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cfg")
	if err := EnsurePrivateDir(dir); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := EnsurePrivateDir(dir); err != nil {
		t.Fatalf("second call: %v", err)
	}
	file := filepath.Join(dir, "probe")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	ok, err := FileIsOwnerOnly(file)
	if err != nil {
		t.Fatalf("FileIsOwnerOnly: %v", err)
	}
	if !ok {
		t.Error("a file created inside a converged directory is not owner-only")
	}
}

func TestEnsurePrivateDirRejectsBadInput(t *testing.T) {
	if err := EnsurePrivateDir(""); err == nil {
		t.Error("accepted an empty path")
	}
	if err := EnsurePrivateDir("relative"); err == nil {
		t.Error("accepted a relative path")
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrivateDir(file); err == nil {
		t.Error("accepted a regular file")
	}
}
