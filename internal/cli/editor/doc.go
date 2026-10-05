// Package editor is the line editor of the native CLI mode: golang.org/x/term's
// Terminal behind gobble's adapter, with two-stage Ctrl+C, bracketed paste
// coalesced into one submission, a file-backed history, backslash
// continuation and an external editor (0008-MADR D6).
// Stability: internal
//
// One Reader owns stdin for the life of the process. Nothing in this package
// closes it: on Windows, closing a console input handle while a read is
// pending hangs the process (0008-MADR F20, measurement 10).
package editor
