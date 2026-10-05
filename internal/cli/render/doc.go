// Package render turns assistant text and tool events into terminal output
// for the native CLI mode: a stream that releases only Markdown that can no
// longer change meaning, Markdown coloured in place, aligned tables,
// bounded tool output, a context status line and a spinner (0008-MADR D7,
// D8). With colour off, text passes through unchanged.
// Stability: internal
package render
