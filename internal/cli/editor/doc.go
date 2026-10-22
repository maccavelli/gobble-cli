// Package editor is the line editor of the native CLI mode: gobble's own,
// with the line in a buffer, laid out in whole words to the terminal's width
// (to 120 columns) and drawn again after each key, two-stage Ctrl+C,
// bracketed paste coalesced into one submission, a file-backed history,
// backslash continuation and an external editor (0008-MADR D6;
// 0027-MADR). golang.org/x/term gives it raw mode only.
// Stability: internal
//
// One Reader owns stdin for the life of the process. It reads only when
// asked, so each read starts in the terminal mode of whoever asked: a
// Windows console read keeps the mode it started in (0027-MADR D5, as
// amended). Nothing in this package closes stdin: on Windows, closing a
// console input handle while a read is pending hangs the process (0008-MADR
// F20, measurement 10).
package editor
