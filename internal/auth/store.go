package auth

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/maccavelli/gobble-cli/internal/filelock"
)

// The entry types of Pi's auth.json (ai/src/auth/types.ts).
const (
	TypeAPIKey = "api_key"
	TypeOAuth  = "oauth"
)

// authPolicy is how long a change waits for auth.json's lock: Pi's 30 s
// (core/auth-storage.ts), backing off from 20 ms to 2 s.
var authPolicy = filelock.Policy{First: 20 * time.Millisecond, Max: 2 * time.Second, Deadline: 30 * time.Second}

// Entry is one provider's credential, in Pi's shape: its type; for an
// api_key, its key as written, in Pi's value grammar; for an oauth login,
// when its access token expires, in milliseconds since the epoch, as Pi
// writes it.
type Entry struct {
	Type, Key string
	Expires   int64
}

// member is one provider's entry as the file holds it: its bytes are kept,
// so a write gives back every field gobble does not read.
type member struct {
	provider string
	value    jsontext.Value
	entry    Entry
}

// File is auth.json's entries, in the file's order (0005-PLAN F4-3a).
type File struct {
	members []member
}

// Load reads auth.json at path. No file is an empty File. A file that is
// not an object of entries, each an object with a string "type", is an
// error naming it.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: the config directory's own auth.json
	if errors.Is(err, fs.ErrNotExist) {
		return &File{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth.json: %w", err)
	}
	f, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

func parse(data []byte) (*File, error) {
	f := &File{}
	if len(bytes.TrimSpace(data)) == 0 {
		return f, nil
	}
	d := jsontext.NewDecoder(bytes.NewReader(data))
	if tok, err := d.ReadToken(); err != nil || tok.Kind() != '{' {
		return nil, errors.New("invalid auth.json: expected an object")
	}
	for d.PeekKind() != '}' {
		tok, err := d.ReadToken()
		if err != nil {
			return nil, fmt.Errorf("invalid auth.json: %w", err)
		}
		provider := tok.String()
		value, err := d.ReadValue()
		if err != nil {
			return nil, fmt.Errorf("invalid auth.json: %w", err)
		}
		var e struct {
			Type    *string `json:"type"`
			Key     string  `json:"key"`
			Expires int64   `json:"expires"`
		}
		if value.Kind() != '{' || json.Unmarshal(value, &e) != nil || e.Type == nil {
			return nil, fmt.Errorf("invalid auth.json credential for provider %q", provider)
		}
		entry := Entry{Type: *e.Type, Key: e.Key, Expires: e.Expires}
		f.members = append(f.members, member{provider: provider, value: value.Clone(), entry: entry})
	}
	if _, err := d.ReadToken(); err != nil {
		return nil, fmt.Errorf("invalid auth.json: %w", err)
	}
	return f, nil
}

// Entry is provider's entry, if the file has one.
func (f *File) Entry(provider string) (Entry, bool) {
	if i := f.index(provider); i >= 0 {
		return f.members[i].entry, true
	}
	return Entry{}, false
}

// Providers are the providers with an entry, in the file's order.
func (f *File) Providers() []string {
	out := make([]string, len(f.members))
	for i, m := range f.members {
		out[i] = m.provider
	}
	return out
}

func (f *File) index(provider string) int {
	return slices.IndexFunc(f.members, func(m member) bool { return m.provider == provider })
}

// Set stores e as provider's entry: under auth.json's lock it re-reads the
// file, replaces or adds the one entry, and writes the file whole, as Pi's
// modify does. A file it cannot parse is an error, and is not overwritten.
func Set(ctx context.Context, path, provider string, e Entry) error {
	value, err := json.Marshal(struct {
		Type string `json:"type"`
		Key  string `json:"key,omitempty"`
	}{e.Type, e.Key})
	if err != nil {
		return fmt.Errorf("auth.json: %w", err)
	}
	return modify(ctx, path, func(f *File) bool {
		m := member{provider: provider, value: value, entry: e}
		if i := f.index(provider); i >= 0 {
			f.members[i] = m
		} else {
			f.members = append(f.members, m)
		}
		return true
	})
}

// Delete removes provider's entry, reporting whether there was one.
func Delete(ctx context.Context, path, provider string) (bool, error) {
	removed := false
	err := modify(ctx, path, func(f *File) bool {
		if i := f.index(provider); i >= 0 {
			f.members = slices.Delete(f.members, i, i+1)
			removed = true
		}
		return removed
	})
	return removed, err
}

// modify applies change to the file under its lock, and writes it when
// change reports a change.
func modify(ctx context.Context, path string, change func(*File) bool) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("auth.json: %w", err)
	}
	release, err := filelock.Lock(ctx, path+".lock", authPolicy)
	if err != nil {
		return fmt.Errorf("auth.json: %w", err)
	}
	defer func() { err = errors.Join(err, release()) }()
	f, err := Load(path)
	if err != nil {
		return err
	}
	if !change(f) {
		return nil
	}
	return write(path, f.encode())
}

// encode is the file in Pi's layout, two spaces deep, each kept entry's
// bytes as read and a new one indented as Pi indents it.
func (f *File) encode() []byte {
	var b bytes.Buffer
	b.WriteString("{")
	for i, m := range f.members {
		if i > 0 {
			b.WriteString(",")
		}
		name, _ := jsontext.AppendQuote(nil, m.provider) //nolint:errcheck // a name read from JSON, or given by gobble, quotes
		value := m.value
		if !bytes.Contains(value, []byte("\n")) {
			value = indented(value)
		}
		fmt.Fprintf(&b, "\n  %s: %s", name, value)
	}
	if len(f.members) > 0 {
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.Bytes()
}

// indented is a one-line value as Pi's JSON.stringify(…, null, 2) lays it
// out under a top-level key.
func indented(v jsontext.Value) jsontext.Value {
	out := v.Clone()
	if out.Indent(jsontext.WithIndentPrefix("  "), jsontext.WithIndent("  ")) != nil {
		return v
	}
	return out
}

// write replaces path with data through a temporary file in its directory,
// owner-only before its first byte (0005-PLAN F4-3a decision 4).
func write(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".auth.json.*.tmp")
	if err != nil {
		return fmt.Errorf("auth.json: %w", err)
	}
	name := tmp.Name()
	if err := writeTemp(tmp, data); err != nil {
		return errors.Join(fmt.Errorf("auth.json: %w", err), os.Remove(name))
	}
	if err := os.Rename(name, path); err != nil {
		return errors.Join(fmt.Errorf("auth.json: %w", err), os.Remove(name))
	}
	return nil
}

func writeTemp(tmp *os.File, data []byte) error {
	if err := securePrivate(tmp.Name()); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if _, err := tmp.Write(data); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, tmp.Close())
	}
	return tmp.Close()
}
