// Package term is the terminal layer of the native CLI mode: capability
// detection, styles and glyphs, escape-sequence sanitising, Windows console
// preparation, and the per-OS shutdown signals (0008-MADR D2, D8).
// Stability: internal
//
// It imports only the standard library, golang.org/x/term and
// golang.org/x/sys. It never writes to a stream itself.
package term
