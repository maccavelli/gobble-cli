package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// hostIDFile is the file under the state directory that holds the host id,
// and hostIDLock the lock file under which it is placed.
const (
	hostIDFile = "host-id"
	hostIDLock = "host-id.lock"
)

// hostIDHooks are test seams, nil in production: afterRead runs when a
// placement has read no id, before it takes host-id.lock. Only
// internal/auth's own tests set it.
var hostIDHooks struct{ afterRead func() }

// HostID is this installation's host id, a random urn:uuid read from
// dir/host-id, or placed there when the file is absent or empty, owner-only.
// Of two callers racing to place it, the one that takes host-id.lock first
// writes its id and the other reads it under the lock (0052-MADR D4, as its
// note of 2026-11-21 amends it).
func HostID(dir string) (string, error) {
	if id, err := readHostID(filepath.Join(dir, hostIDFile)); err != nil || id != "" {
		return id, err
	}
	id, err := newHostID()
	if err != nil {
		return "", fmt.Errorf("host-id: %w", err)
	}
	return placeHostID(dir, id)
}

// placeHostID is the id dir/host-id holds, after writing id there when the
// file is absent or empty: under host-id.lock it reads the file again and,
// finding it still absent or empty, renames a temporary file holding id,
// owner-only before its first byte and synced, onto it. A file that holds
// anything but a urn:uuid is an error and is not replaced. The directory is
// made 0700. It waits for the lock with no context of a caller's, as HostID
// has none to give it.
//
//nolint:contextcheck // HostID's callers give it no context (0052-PLAN P5)
func placeHostID(dir, id string) (string, error) {
	got, _, err := placeHostIDAfter(context.Background(), dir, id, hostIDHooks.afterRead)
	return got, err
}

// placeHostIDAfter is placeHostID, waiting for the lock under ctx, calling
// afterRead, when not nil, after it reads no id and before it takes the
// lock, and reporting whether it wrote id.
func placeHostIDAfter(ctx context.Context, dir, id string, afterRead func()) (got string, wrote bool, err error) {
	path := filepath.Join(dir, hostIDFile)
	if got, err := readHostID(path); err != nil || got != "" {
		return got, false, err
	}
	if afterRead != nil {
		afterRead()
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", false, fmt.Errorf("host-id: %w", err)
	}
	release, err := filelock.Lock(ctx, filepath.Join(dir, hostIDLock), authPolicy)
	if err != nil {
		return "", false, fmt.Errorf("host-id: %w", err)
	}
	defer func() { err = errors.Join(err, release()) }()
	if got, err := readHostID(path); err != nil || got != "" {
		return got, false, err
	}
	if err := renameHostID(dir, id); err != nil {
		return "", false, err
	}
	return id, true, nil
}

// renameHostID writes id to a temporary file in dir and renames it onto
// dir/host-id; the temporary file is removed on any failure.
func renameHostID(dir, id string) error {
	tmp, err := os.CreateTemp(dir, ".host-id.*.tmp")
	if err != nil {
		return fmt.Errorf("host-id: %w", err)
	}
	name := tmp.Name()
	if err := writeTemp(tmp, []byte(id+"\n")); err != nil {
		return errors.Join(fmt.Errorf("host-id: %w", err), os.Remove(name))
	}
	if err := os.Rename(name, filepath.Join(dir, hostIDFile)); err != nil {
		return errors.Join(fmt.Errorf("host-id: %w", err), os.Remove(name))
	}
	return nil
}

// readHostID is the id in path, "" when there is no file or it is empty,
// and an error for a file that holds anything but an id.
func readHostID(path string) (string, error) {
	data, err := fsx.ReadRegularFile(path, fsx.ConfigLimit)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("host-id: %w", err)
	}
	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", nil
	}
	if !validHostID(id) {
		return "", fmt.Errorf("host-id: %s does not hold a urn:uuid", path)
	}
	return id, nil
}

// validHostID reports whether id is a urn:uuid, the rule readHostID applies
// to the file and the migration to a plan entry's hostId (0052-MADR D9, Q3).
func validHostID(id string) bool { return strings.HasPrefix(id, "urn:uuid:") }

// newHostID is a random (version 4) urn:uuid.
func newHostID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
