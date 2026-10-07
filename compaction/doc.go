// Package compaction is Pi's context compaction over gobble's session
// entries: where to cut, what to summarise, and the summary itself.
// Stability: beta
//
// Context is the model context of a session path with a compaction in it.
// Prepare finds the cut point that keeps KeepRecentTokens of recent
// context, never at a tool result, and splits a turn when one turn is too
// big; Compact asks the model for the summary in Pi's format, updating a
// previous one. The rules, prompts and serialisation are Pi's at
// maccavelli/pi 312184edb (core/compaction/compaction.ts and utils.ts),
// ported for /compact (0002-PLAN Phase 6). The automatic trigger and
// overflow recovery arrive with 0005-PLAN F5.
package compaction
