// Package tools keeps the Phase 0 module requirement that no other
// package imports yet. 0004-PLAN step 3 requires
// github.com/coder/acp-go-sdk in go.mod; go mod tidy drops a
// requirement that nothing imports.
package tools

import (
	_ "github.com/coder/acp-go-sdk" // ACP pin (0004-PLAN phase 0); imported for real in a later phase
)
