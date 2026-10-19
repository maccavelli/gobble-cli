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
// is asked about, so a client can show what would run. Repeated, when not
// 0, is the run of identical calls the call ends: the question is whether
// the turn may go on, which approves no write (0005-PLAN F5-3 decision 1).
type Rule struct {
	Tool     string         `json:"tool"`
	Mode     Mode           `json:"mode,omitzero"`
	CallID   string         `json:"callId,omitzero"`
	Title    string         `json:"title,omitzero"`
	Input    jsontext.Value `json:"input,omitzero"`
	Repeated int            `json:"repeated,omitzero"`
	// Kind and Subjects are what the rules' patterns match: the absolute
	// paths a file tool names, or a shell tool's command; Outside are its
	// paths outside the workspace's roots (0005-PLAN F6-2 decisions 3, 7).
	Kind     SubjectKind `json:"kind,omitzero"`
	Subjects []string    `json:"subjects,omitzero"`
	Outside  []string    `json:"outside,omitzero"`
	// Saves are a command's subjects' "always" patterns, beside Subjects,
	// "" for a subject's exact text (0005-PLAN F6-3 decisions 4 and 11).
	Saves []string `json:"saves,omitzero"`
	// ReadOnly says the tool changes nothing, for the defaults.
	ReadOnly bool `json:"readOnly,omitzero"`
}

// Decision is a policy answer. Refusal, when set, is a refusal made
// without asking anyone, such as plan mode's: it is the whole text the
// model reads, in place of a declined call's.
// By, when set, is the rule or limit that decided (0005-PLAN F6-2
// decision 12).
type Decision struct {
	Allow   bool   `json:"allow"`
	Reason  string `json:"reason,omitzero"`
	Refusal string `json:"refusal,omitzero"`
	By      Spec   `json:"by,omitzero"`
}
