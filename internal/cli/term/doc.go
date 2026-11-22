// Package term is the terminal layer of the native CLI mode: capability
// detection, styles and glyphs, escape-sequence sanitising, Windows console
// preparation, the flush of a terminal's input queue (FlushInput), a console's
// input with the rule that reads again after an interrupt gobble took
// (ConsoleInput), the cancellable read of a POSIX terminal (OpenGated, Gated),
// and the per-OS shutdown signals (0008-MADR D2, D8; 0063-MADR D18, D19,
// D22).
// Stability: internal
//
// It imports only the standard library, golang.org/x/term and
// golang.org/x/sys. It never writes to a stream itself.
package term
