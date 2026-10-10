// Package prompt assembles the system prompt.
// Stability: beta
//
// The prompt is a list of named sections in Pi's order and shapes (Build),
// from the context files (ContextFiles, Nested) and the system files
// (SystemFiles). A session freezes its first sections as a baseline; a later
// change is a Patch (Diff, Apply), which reaches the model as an appended
// message (Update), so the cached prefix holds (0005-MADR, amendment of
// 2026-10-10; 0005-PLAN F3-1).
package prompt
