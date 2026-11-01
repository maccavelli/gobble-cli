// Package render turns tool events and gobble's own text into terminal
// output for the native CLI mode: bounded tool output, plans and diffs, a
// context status line, a spinner, gobble's messages, and Wrap for thinking
// text (0008-MADR D7, D8). A reply's Markdown is drawn by go-tui-lib's
// stream (0035-MADR D14 step 5a). With colour off, text passes through
// unchanged.
// Stability: internal
package render
