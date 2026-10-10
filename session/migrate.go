package session

import (
	"bytes"
	"encoding/json/jsontext"
	"fmt"
	"math"
	"strconv"
)

// firstKeptIndex is a v1 compaction's member naming its first kept entry by
// its line among the parsed lines, the header being 0.
const firstKeptIndex = "firstKeptEntryIndex"

// roleHookMessage is v2's name for the custom role.
const roleHookMessage = "hookMessage"

// Migrate brings the entries of a version's file to Version in memory, as
// Pi's migrateToCurrentVersion does (session-manager.ts:287-347). The file
// is never rewritten (0005-PLAN F2), so v1's ids are fixed rather than
// random: entry n, counting from 1, is fmt.Sprintf("%08x", n), and every
// load gives the same ids. entries is not changed.
//
//   - v1 to v2: each entry gets its id and the previous entry as its parent,
//     and a compaction's firstKeptEntryIndex becomes the firstKeptEntryId of
//     that entry. An index at the header, or past the end, leaves none.
//   - v2 to v3: a message with the role hookMessage gets the role custom.
//
// A version of 0 is 1, as a header without one is.
func Migrate(version int, entries []Entry) ([]Entry, error) {
	version = max(version, 1)
	if version > Version {
		return nil, fmt.Errorf("unsupported session version %d (gobble reads versions 1 to %d)", version, Version)
	}
	out := make([]Entry, len(entries))
	copy(out, entries)
	if version < 2 {
		if err := migrateV1(out); err != nil {
			return nil, err
		}
	}
	if version < 3 {
		migrateV2(out)
	}
	return out, nil
}

func v1ID(n int) string { return fmt.Sprintf("%08x", n) }

func migrateV1(entries []Entry) error {
	for i := range entries {
		e := &entries[i]
		e.ID, e.ParentID = v1ID(i+1), nil
		if i > 0 {
			e.ParentID = new(entries[i-1].ID)
		}
		if e.Type != TypeCompaction {
			continue
		}
		rest, index, err := cutMember(e.Unknown, firstKeptIndex)
		if err != nil {
			return fmt.Errorf("session: entry %d: %w", i+1, err)
		}
		e.Unknown = rest
		if n, ok := lineIndex(index); ok && n >= 1 && n <= len(entries) {
			e.FirstKeptEntryID = v1ID(n)
		}
	}
	return nil
}

// lineIndex is a JSON number that is a whole line index.
func lineIndex(v jsontext.Value) (int, bool) {
	if v.Kind() != '0' {
		return 0, false
	}
	f, err := strconv.ParseFloat(string(v), 64)
	if err != nil || f != math.Trunc(f) || f < 0 || f > math.MaxInt32 {
		return 0, false
	}
	return int(f), true
}

func migrateV2(entries []Entry) {
	for i := range entries {
		e := &entries[i]
		if e.Type == TypeMessage && e.Message != nil && e.Message.Role == roleHookMessage {
			m := *e.Message
			m.Role = RoleCustom
			e.Message = &m
		}
	}
}

// cutMember is the object v without its member name, and that member's
// value, or nil when it has none. An empty v is an empty object.
func cutMember(v jsontext.Value, name string) (rest, value jsontext.Value, err error) {
	if len(v) == 0 {
		return v, nil, nil
	}
	dec := jsontext.NewDecoder(bytes.NewReader(v))
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(&buf)
	if tok, err := dec.ReadToken(); err != nil || tok.Kind() != '{' {
		return nil, nil, fmt.Errorf("unknown members are not an object: %w", err)
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return nil, nil, err
	}
	kept := 0
	for dec.PeekKind() != '}' {
		key, err := dec.ReadToken()
		if err != nil {
			return nil, nil, err
		}
		member := key.String() // the token is void after the next read
		val, err := dec.ReadValue()
		if err != nil {
			return nil, nil, err
		}
		if member == name {
			value = val.Clone()
			continue
		}
		kept++
		if err := enc.WriteToken(jsontext.String(member)); err != nil {
			return nil, nil, err
		}
		if err := enc.WriteValue(val); err != nil {
			return nil, nil, err
		}
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return nil, nil, err
	}
	if kept == 0 {
		return nil, value, nil
	}
	return bytes.TrimSpace(buf.Bytes()), value, nil
}
