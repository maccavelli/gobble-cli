package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/jsonx"
)

// ClientRecord is the OAuth client a provider issued this installation, and
// the host id it was registered with. gobble's own type: internal/auth
// imports no SDK package (0052-MADR D5, D7).
type ClientRecord struct {
	ClientID string `json:"clientId,omitzero"`
	HostID   string `json:"hostId,omitzero"`
}

// Clients is the issued clients as clients.json at Path, one record per
// provider, apart from auth.json: a session's Delete leaves its client, so
// a sign-out never forces a new registration (0052-MADR D5). Each change is
// under clients.json.lock and is written as auth.json is.
type Clients struct {
	Path string
}

// Load is id's record; the zero record when there is none.
func (c Clients) Load(_ context.Context, id string) (ClientRecord, error) {
	m, err := c.read()
	if err != nil {
		return ClientRecord{}, err
	}
	return m[id], nil
}

// Save stores r as id's record, keeping every other provider's; a zero r
// removes id's member.
func (c Clients) Save(ctx context.Context, id string, r ClientRecord) (err error) {
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o700); err != nil {
		return fmt.Errorf("clients.json: %w", err)
	}
	release, err := filelock.Lock(ctx, c.Path+".lock", authPolicy)
	if err != nil {
		return fmt.Errorf("clients.json: %w", err)
	}
	defer func() { err = errors.Join(err, release()) }()
	m, err := c.read()
	if err != nil {
		return err
	}
	if r == (ClientRecord{}) {
		if _, ok := m[id]; !ok {
			return nil
		}
		delete(m, id)
	} else {
		m[id] = r
	}
	var b bytes.Buffer
	b.WriteString("{")
	for i, k := range slices.Sorted(maps.Keys(m)) {
		v, err := jsonx.MarshalStrict(m[k])
		if err != nil {
			return fmt.Errorf("clients.json: %w", err)
		}
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "\n  %s: %s", jsonx.Quote(k), indented(v))
	}
	if len(m) > 0 {
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return write(c.Path, b.Bytes())
}

// read is the file's records; none for no file. A file that is not an
// object of records is an error naming it.
func (c Clients) read() (map[string]ClientRecord, error) {
	m := map[string]ClientRecord{}
	data, err := fsx.ReadRegularFile(c.Path, fsx.ConfigLimit)
	if errors.Is(err, fs.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("clients.json: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return m, nil
	}
	if err := jsonx.UnmarshalStrict(data, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", c.Path, err)
	}
	return m, nil
}
