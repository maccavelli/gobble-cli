// Package complete is gobble's native shell completion (0008-MADR D17): a
// walker over Kong's model that finds what the cursor is completing, the
// value sources behind complete:"<name>" tags, the completion-mode protocol,
// and the bash, zsh, fish and PowerShell registration scripts.
// Stability: internal
//
// Completion mode is read-only and network-free (0008-PLAN C6). It runs
// before configuration, logging and credentials, under a 150 ms deadline.
package complete
