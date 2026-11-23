package auth

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// holdForReplace opens p as a replacing rename does, with DELETE access
// and every share, so os.Open is refused with a sharing violation while it
// is held (0067-MADR probe i).
func holdForReplace(t *testing.T, p string) windows.Handle {
	t.Helper()
	name, err := windows.UTF16PtrFromString(p)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(name, windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// auth.json, clients.json and host-id are read through fsx's retry: each
// reader, refused once by a replace in the middle of its move, reads the
// file once the holder lets go. The open's first refusal closes the holder
// (0067-MADR C1 to C3, C8; D6).
func TestAuthReadersSurviveAHeldHandle(t *testing.T) {
	const hostID = "urn:uuid:00000000-0000-4000-8000-000000000000"
	cases := []struct {
		name, file, content string
		read                func(path string) error
	}{
		{"auth.Load", "auth.json", `{"claude": {"type": "api_key", "key": "sk-test-HELD"}}`, func(path string) error {
			f, err := Load(path)
			if err != nil {
				return err
			}
			if e, ok := f.Entry("claude"); !ok || e.Key != "sk-test-HELD" {
				return errors.New("auth.Load read the file but not its entry")
			}
			return nil
		}},
		{"Clients.Load", "clients.json", `{"openai": {"clientId": "oaiapp_held"}}`, func(path string) error {
			r, err := Clients{Path: path}.Load(t.Context(), "openai")
			if err != nil {
				return err
			}
			if r.ClientID != "oaiapp_held" {
				return errors.New("Clients.Load read the file but not its record")
			}
			return nil
		}},
		{"readHostID", "host-id", hostID + "\n", func(path string) error {
			id, err := readHostID(path)
			if err != nil {
				return err
			}
			if id != hostID {
				return errors.New("readHostID read the file but not its id")
			}
			return nil
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), c.file)
			if err := os.WriteFile(p, []byte(c.content), 0o600); err != nil {
				t.Fatal(err)
			}
			h := holdForReplace(t, p)
			released := false
			t.Cleanup(func() {
				if !released {
					windows.CloseHandle(h) //nolint:errcheck,gosec // the test's own handle
				}
			})
			var refusal error
			restore := fsx.SetOpenFile(func(name string, flag int, perm fs.FileMode) (*os.File, error) {
				f, err := os.OpenFile(name, flag, perm) //nolint:gosec // G304: the test's temp file
				if err != nil && !released {
					refusal = err
					released = true
					if cerr := windows.CloseHandle(h); cerr != nil {
						t.Errorf("close the holder: %v", cerr)
					}
				}
				return f, err
			})
			t.Cleanup(restore)
			err := c.read(p)
			if !released || !errors.Is(refusal, windows.ERROR_SHARING_VIOLATION) {
				t.Fatalf("%s did not open through fsx's seam, or was not refused with a sharing violation (%v)", c.name, refusal)
			}
			if err != nil {
				t.Fatalf("%s under a held handle: %v; want the file once the holder lets go", c.name, err)
			}
		})
	}
}
