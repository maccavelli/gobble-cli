package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// hostIDFile is the file under the state directory that holds the host id.
const hostIDFile = "host-id"

// HostID is this installation's host id, a random urn:uuid read from
// dir/host-id, or created there when the file is absent, owner-only. Of two
// callers racing to create it, one wins and the other reads the winner's
// (0052-MADR D4).
func HostID(dir string) (string, error) {
	path := filepath.Join(dir, hostIDFile)
	if id, err := readHostID(path); err != nil || id != "" {
		return id, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("host-id: %w", err)
	}
	id, err := newHostID()
	if err != nil {
		return "", fmt.Errorf("host-id: %w", err)
	}
	f, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return awaitHostID(path)
	}
	if err != nil {
		return "", fmt.Errorf("host-id: %w", err)
	}
	if err := securePrivate(path); err != nil {
		return "", errors.Join(fmt.Errorf("host-id: %w", err), f.Close(), os.Remove(path))
	}
	if _, err := f.WriteString(id + "\n"); err != nil {
		return "", errors.Join(fmt.Errorf("host-id: %w", err), f.Close(), os.Remove(path))
	}
	if err := f.Sync(); err != nil {
		return "", errors.Join(fmt.Errorf("host-id: %w", err), f.Close(), os.Remove(path))
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("host-id: %w", err)
	}
	return id, nil
}

// readHostID is the id in path, "" when there is no file or it is still
// empty, and an error for a file that holds anything but an id.
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
	if !strings.HasPrefix(id, "urn:uuid:") {
		return "", fmt.Errorf("host-id: %s does not hold a urn:uuid", path)
	}
	return id, nil
}

// awaitHostID reads the id the other caller is writing: the file exists
// before its first byte does.
func awaitHostID(path string) (string, error) {
	deadline := time.Now().Add(5 * time.Second)
	for {
		id, err := readHostID(path)
		if err != nil || id != "" {
			return id, err
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("host-id: %s stayed empty", path)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

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
