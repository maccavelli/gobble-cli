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
// session.ErrCorrupt.
//
// A file is never rewritten. Under the writer lock gobble only ever cuts off
// bytes it wrote itself: an append that failed partway, and, when Open finds
// one in a file of the current version, a last line that has no newline and
// is not an entry, which a crash mid-write leaves. An entry joins a log's
// Entries only once it is wholly on disk. Entries have no size cap; each is
// bounded where it is made. Appends are not synced: a crash of the process
// keeps every entry Append returned for, and power loss can lose the last
// ones (0037-MADR D37 to D39).
package jsonl
