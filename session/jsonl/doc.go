// Package jsonl stores sessions as JSONL files, as Pi writes them.
// Stability: stable
//
// A store's directory holds one subdirectory per cwd, --<cwd>--, of files
// named <timestamp>_<id>.jsonl; a flat store keeps them all in one
// directory, filtered by the header's cwd (Pi's session-manager.ts at
// maccavelli/pi 312184edb). A file is created with the session's first user
// or assistant message, and each entry after it is one appended line. A
// writer holds an exclusive lock on a sidecar <file>.lock that holds its
// pid; readers take none. A malformed line is skipped with a warning; a file
// whose header is unreadable, or whose version is not 3, fails closed with
// session.ErrCorrupt. A file is never rewritten.
package jsonl
