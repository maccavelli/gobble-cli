package permission

import (
	"context"
	"encoding/json/jsontext"
)

// Policy decides whether a rule allows an action.
type Policy interface {
	Decide(ctx context.Context, rule Rule) (Decision, error)
}

// Mode is the permission posture for a rule.
type Mode string

const (
	// ModeAsk prompts before the action.
	ModeAsk Mode = "ask"
	// ModeAllow permits the action.
	ModeAllow Mode = "allow"
	// ModeDeny refuses the action.
	ModeDeny Mode = "deny"
)

// Rule is one permission question: the tool, the posture, and the call it
// is asked about, so a client can show what would run.
type Rule struct {
	Tool   string         `json:"tool"`
	Mode   Mode           `json:"mode,omitzero"`
	CallID string         `json:"callId,omitzero"`
	Title  string         `json:"title,omitzero"`
	Input  jsontext.Value `json:"input,omitzero"`
}

// Decision is a policy answer.
type Decision struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason,omitzero"`
}
