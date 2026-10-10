// Package skill parses and discovers Agent Skills.
// Stability: stable
//
// A skill is a SKILL.md, or a loose .md file with a description, whose YAML
// frontmatter follows agentskills.io. Load finds skills in a list of
// directories as Pi does: the first of two skills with one name wins, the
// other a collision diagnostic. Format is the system prompt's list of them,
// and Expand is Pi's /skill:<name> (0005-PLAN F3-2).
package skill
