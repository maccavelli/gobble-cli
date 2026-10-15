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
// its tokens, when its access token expires, in milliseconds since the
// epoch, as Pi writes it, and the SDK session's own fields (0005-PLAN F4-3b
// decision 3).
type Entry struct {
	Type, Key       string
	Expires         int64
	Access, Refresh string
	// Issuer, ClientID, AccountID, TokenURL and FedRAMP are an oauth
	// login's issuer, client id, account, token endpoint, and
	// whether it sends X-OpenAI-Fedramp (0015-REPORT F5).
	Issuer, ClientID, AccountID, TokenURL string
	FedRAMP                               bool
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
			Type      *string `json:"type"`
			Key       string  `json:"key"`
			Expires   int64   `json:"expires"`
			Access    string  `json:"access"`
			Refresh   string  `json:"refresh"`
			Issuer    string  `json:"issuer"`
			ClientID  string  `json:"clientId"`
			AccountID string  `json:"accountId"`
			TokenURL  string  `json:"tokenUrl"`
			FedRAMP   bool    `json:"fedramp"`
		}
		if value.Kind() != '{' || json.Unmarshal(value, &e) != nil || e.Type == nil {
			return nil, fmt.Errorf("invalid auth.json credential for provider %q", provider)
		}
		entry := Entry{Type: *e.Type, Key: e.Key, Expires: e.Expires, Access: e.Access, Refresh: e.Refresh,
			Issuer: e.Issuer, ClientID: e.ClientID, AccountID: e.AccountID, TokenURL: e.TokenURL, FedRAMP: e.FedRAMP}
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

// SetOAuth stores e as provider's oauth entry: under auth.json's lock it
// re-reads the file and sets an oauth login's fields in the provider's
// entry, keeping every other field as written, or adds the entry. fedramp
// is written only when true, and an SDK field only when set (0005-PLAN
// F4-3b decision 3).
func SetOAuth(ctx context.Context, path, provider string, e Entry) error {
	e.Type = TypeOAuth
	set, remove, err := oauthFields(e)
	if err != nil {
		return fmt.Errorf("auth.json: %w", err)
	}
	var mergeErr error
	err = modify(ctx, path, func(f *File) bool {
		var old jsontext.Value
		i := f.index(provider)
		if i >= 0 {
			old = f.members[i].value
		}
		value, err := merge(old, set, remove)
		if err != nil {
			mergeErr = fmt.Errorf("auth.json: provider %s: %w", provider, err)
			return false
		}
		m := member{provider: provider, value: value, entry: e}
		if i >= 0 {
			f.members[i] = m
		} else {
			f.members = append(f.members, m)
		}
		return true
	})
	return errors.Join(err, mergeErr)
}

// field is one member of an entry's object.
type field struct {
	name  string
	value jsontext.Value
}

// oauthFields are the fields SetOAuth writes for e, in Pi's order then the
// SDK session's, and the fields it removes: an SDK field e leaves unset, and
// fedramp when false.
func oauthFields(e Entry) (set []field, remove []string, err error) {
	add := func(name string, v any) {
		b, merr := json.Marshal(v)
		err = errors.Join(err, merr)
		set = append(set, field{name, b})
	}
	add("type", e.Type)
	add("access", e.Access)
	add("refresh", e.Refresh)
	add("expires", e.Expires)
	for _, f := range []struct{ name, v string }{
		{"issuer", e.Issuer}, {"clientId", e.ClientID}, {"accountId", e.AccountID}, {"tokenUrl", e.TokenURL},
	} {
		if f.v == "" {
			remove = append(remove, f.name)
		} else {
			add(f.name, f.v)
		}
	}
	if e.FedRAMP {
		add("fedramp", true)
	} else {
		remove = append(remove, "fedramp")
	}
	return set, remove, err
}

// merge is the object old with set's fields replacing or following its own,
// and remove's left out; every other field keeps its value. A nil old is an
// empty object. The result is compact, for encode to lay out.
func merge(old jsontext.Value, set []field, remove []string) (jsontext.Value, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	put := func(name string, v jsontext.Value) {
		if b.Len() > 1 {
			b.WriteByte(',')
		}
		q, _ := jsontext.AppendQuote(nil, name) //nolint:errcheck // a name read from JSON, or given by gobble, quotes
		b.Write(q)
		b.WriteByte(':')
		b.Write(v)
	}
	done := map[string]bool{}
	if len(old) > 0 {
		d := jsontext.NewDecoder(bytes.NewReader(old))
		if _, err := d.ReadToken(); err != nil {
			return nil, err
		}
		for d.PeekKind() != '}' {
			tok, err := d.ReadToken()
			if err != nil {
				return nil, err
			}
			name := tok.String()
			v, err := d.ReadValue()
			if err != nil {
				return nil, err
			}
			switch i := slices.IndexFunc(set, func(f field) bool { return f.name == name }); {
			case i >= 0:
				put(name, set[i].value)
				done[name] = true
			case !slices.Contains(remove, name):
				put(name, v)
			}
		}
	}
	for _, f := range set {
		if !done[f.name] {
			put(f.name, f.value)
		}
	}
	b.WriteByte('}')
	out := jsontext.Value(b.Bytes())
	if err := out.Compact(); err != nil {
		return nil, err
	}
	return out, nil
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
