// Package session is the session log contract.
// Stability: stable
//
// Phase 1 declares the types only. It does not open, list, or append logs.
//
// Entry is the JSONL union. Members this version does not know are preserved
// by a jsontext.Value field tagged json:",embed". Go 1.27.1 ignores the
// older json:",unknown" option (it was removed), so embed is the fallback
// that still round-trips those members. Optional fields are tagged omitzero.
package session
