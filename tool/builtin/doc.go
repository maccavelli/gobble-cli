// Package builtin holds the built-in tools.
// Stability: beta
//
// Tools returns read, write, edit and bash, at 0002-PLAN Phase 3's minimum:
// paths resolve against the session's working directory, output is bounded
// at 2,000 lines or 50 KB, and edit needs one exact match. 0005-PLAN F1 brings
// them to Pi's parity and adds grep, find and ls.
package builtin
