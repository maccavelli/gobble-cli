package mcpclient

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
)

// member is one name and value of a JSON object, in document order.
type member struct {
	name  string
	value jsontext.Value
}

var errNotObject = errors.New("not a JSON object")

// members are a JSON object's members in order. A repeated name keeps its
// first place and its last value, as JSON.parse does.
func members(b []byte) ([]member, error) {
	d := jsontext.NewDecoder(bytes.NewReader(b), jsontext.AllowDuplicateNames(true))
	tok, err := d.ReadToken()
	if err != nil {
		return nil, err
	}
	if tok.Kind() != '{' {
		return nil, errNotObject
	}
	var out []member
	at := map[string]int{}
	for d.PeekKind() != '}' {
		tok, err := d.ReadToken()
		if err != nil {
			return nil, err
		}
		name := tok.String() // before the next read voids the token
		v, err := d.ReadValue()
		if err != nil {
			return nil, err
		}
		v = v.Clone() // the decoder reuses its buffer
		if i, ok := at[name]; ok {
			out[i].value = v
			continue
		}
		at[name] = len(out)
		out = append(out, member{name: name, value: v})
	}
	if _, err := d.ReadToken(); err != nil {
		return nil, err
	}
	if _, err := d.ReadValue(); !errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected data after the top-level object")
	}
	return out, nil
}

func find(ms []member, name string) (jsontext.Value, bool) {
	for _, m := range ms {
		if m.name == name {
			return m.value, true
		}
	}
	return nil, false
}

// object encodes members as one compact JSON object, in order.
func object(ms []member) (jsontext.Value, error) {
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(&buf)
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return nil, err
	}
	for _, m := range ms {
		if err := enc.WriteToken(jsontext.String(m.name)); err != nil {
			return nil, err
		}
		if err := enc.WriteValue(m.value); err != nil {
			return nil, err
		}
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

// Add adds a server entry to an mcp.json, creating the file when it is
// missing, as Pi's addMcpServerConfig does: an entry of the same name is
// replaced in place, and a new one goes last. The rest of the file is kept.
// replaced reports that an entry was replaced.
func Add(path, name string, entry jsontext.Value) (replaced bool, err error) {
	text, top, servers, err := readForEdit(path)
	if err != nil {
		return false, err
	}
	for i := range servers {
		if servers[i].name == name {
			servers[i].value, replaced = entry, true
		}
	}
	if !replaced {
		servers = append(servers, member{name: name, value: entry})
	}
	return replaced, writeEdit(path, text, top, servers)
}

// Remove removes a server entry from an mcp.json, keeping the rest of the
// file. It reports false, and writes nothing, when the file does not
// define the server.
func Remove(path, name string) (bool, error) {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	text, top, servers, err := readForEdit(path)
	if err != nil {
		return false, err
	}
	i := slices.IndexFunc(servers, func(m member) bool { return m.name == name })
	if i < 0 {
		return false, nil
	}
	return true, writeEdit(path, text, top, slices.Delete(servers, i, i+1))
}

// readForEdit reads an mcp.json for an edit; a missing file is empty.
func readForEdit(path string) (text []byte, top, servers []member, err error) {
	text, err = os.ReadFile(path) //nolint:gosec // G304: the user's own mcp.json
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}
	top, servers, err = parseFile(text)
	return text, top, servers, err
}

// indentation is Pi's: the first line's leading whitespace, else two
// spaces.
var indentation = regexp.MustCompile(`(?m)^([ \t]+)\S`)

// writeEdit writes the file back with servers as its mcpServers, in the
// file's own indentation and with a final newline. The new file is written
// beside the old one and renamed over it, so a failed write leaves the old
// one whole.
func writeEdit(path string, text []byte, top, servers []member) error {
	obj, err := object(servers)
	if err != nil {
		return err
	}
	if i := slices.IndexFunc(top, func(m member) bool { return m.name == "mcpServers" }); i >= 0 {
		top[i].value = obj
	} else {
		top = append(top, member{name: "mcpServers", value: obj})
	}
	doc, err := object(top)
	if err != nil {
		return err
	}
	indent := "  "
	if m := indentation.FindSubmatch(text); m != nil {
		indent = string(m[1])
	}
	if err := doc.Indent(jsontext.WithIndent(indent)); err != nil {
		return err
	}
	return writeFile(path, append(doc, '\n'))
}

func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".mcp.json.*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		return errors.Join(fmt.Errorf("write %s: %w", path, err), os.Remove(tmp))
	}
	return nil
}
