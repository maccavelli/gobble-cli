// Package session is the session log contract, in Pi's JSONL v3 shapes.
// Stability: stable
//
// A session is a header and a tree of entries linked by ParentID; the leaf
// is the last entry in file order, and Path walks from it to the root. The
// members of Pi's entry types are typed. Every other member, and every
// other entry type, round-trips through a jsontext.Value field tagged
// json:",embed" (Go 1.27.1 removed the older json:",unknown"). Pi's format
// is read from maccavelli/pi 312184edb (0002-PLAN Phase 4); v1 and v2 files
// are migrated in memory when read, and never rewritten (Migrate,
// 0005-PLAN F2).
//
// session/jsonl stores sessions as files; NewMemoryStore keeps them in
// memory, for --no-session and tests.
//
// # Raw JSON and binary
//
// A member kept as written is a jsontext.Value, and binary data is a
// []byte (0039-MADR D20, D21):
//
//	the member     the Go value            written as
//	absent         nil, len 0              left out, by omitzero
//	null           jsontext.Value("null")  null
//	a JSON value   its bytes               those bytes
//	binary         []byte                  a base64 string
//
// An empty non-nil jsontext.Value is not JSON, and a field holding one fails
// the marshal. A raw value enters a field through jsonx.From, jsonx.Value or
// jsonx.Quote, which never give one. Tests and diagnostics compare len(v)
// and string(v), never v.String(), which gives "null" for an absent value.
package session
