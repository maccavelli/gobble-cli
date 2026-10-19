// Package compaction is Pi's context compaction over gobble's session
// entries: where to cut, what to summarise, and the summary itself.
// Stability: beta
//
// Context is the model context of a session path with a compaction in it.
// Prepare finds the cut point that keeps KeepRecentTokens of recent
// context, never at a tool result, and splits a turn when one turn is too
// big, by Pi's rules at maccavelli/pi 312184edb (core/compaction/
// compaction.ts and utils.ts). Over says when to compact before a model call
// (0005-PLAN F5-2). Compact asks the model for the summary on opencode's
// template, updating a previous one, with its reply capped and an empty one
// refused (0005-PLAN F5-4); Prompts replaces its prompts with the user's.
package compaction
