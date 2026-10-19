// Package personality is gobble's voice (0022-MADR): the catalog of every
// message a person reads, written in natural-language Markdown, one file
// per personality, and the personality briefs the system prompt carries.
// Stability: beta
//
// A catalog file is Markdown. Each message is a level-two heading naming
// its key, followed by its variants: one per bullet, or one paragraph.
// Placeholders are names in braces, {name}; "{{" and "}}" are literal
// braces. Text before the first heading is documentation and is ignored.
//
//	## retry.offline
//	- waiting for the network
//
// boring is the canonical personality: one variant per key, the text gobble
// has always shown. Every other personality's variant of a key carries
// exactly the placeholders of boring's (the facts rule, 0022-MADR D5), and
// a key a personality leaves out falls back to boring's.
//
// A nil *Voice is boring, so code that is given none shows the plain text.
package personality
