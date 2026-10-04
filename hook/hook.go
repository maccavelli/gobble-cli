package hook

import (
	"context"

	"encoding/json/jsontext"
)

// Hook handles one lifecycle event.
type Hook interface {
	Handle(ctx context.Context, ev Event) (Outcome, error)
}

// Event is the JSON object passed to a hook.
type Event struct {
	Name    string         `json:"name"`
	Payload jsontext.Value `json:"payload,omitzero"`
}

// Action is what the host should do after a hook.
type Action string

const (
	// ActionContinue lets the host proceed.
	ActionContinue Action = "continue"
	// ActionBlock stops the host action.
	ActionBlock Action = "block"
	// ActionModify lets the host proceed with a replacement payload.
	ActionModify Action = "modify"
)

// Outcome is a hook's decision.
type Outcome struct {
	Action  Action         `json:"action"`
	Reason  string         `json:"reason,omitzero"`
	Payload jsontext.Value `json:"payload,omitzero"`
}
