// Package filelock is an exclusive lock between processes, held on a lock
// file that records its holder's pid: flock on Unix, LockFileEx on Windows.
// Stability: internal
//
// session/jsonl takes it for a session file's writer, and mcpclient and
// internal/config for an edit of mcp.json or settings.yaml (0018-MADR D18).
// The lock file is left in place when released, so no two processes race to
// create and delete it.
package filelock
