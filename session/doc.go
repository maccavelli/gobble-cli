// Package session is the session log contract, in Pi's JSONL v3 shapes.
// Stability: stable
//
// A session is a header and a tree of entries linked by ParentID; the leaf
// is the last entry in file order, and Path walks from it to the root. The
// members gobble acts on are typed. Every other member, and every entry
// type gobble does not act on yet, round-trips through a jsontext.Value field
// tagged json:",embed" (Go 1.27.1 removed the older json:",unknown"). Pi's
// format is read from maccavelli/pi 312184edb (0002-PLAN Phase 4).
//
// session/jsonl stores sessions as files; NewMemoryStore keeps them in
// memory, for --no-session and tests.
package session
