package acpclient

import (
	"errors"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/jsonx"
)

// PromptError is a turn the agent refused with a JSON-RPC error whose data
// is gobble's acpext.ErrorData: its message, and what the provider said
// when the turn failed on a limit (0049-MADR Q3 (a)). Prompt returns it
// wrapped, under ErrAuthRequired for auth_required, so errors.As finds it
// and errors.Is(err, ErrAuthRequired) holds as before.
type PromptError struct {
	// Code is the JSON-RPC error code: -32000 or -32603.
	Code int
	// Data is the error's data.
	Data acpext.ErrorData

	err *acp.RequestError
}

// Error is the data's message, or the request error's text when the data
// has none.
func (e *PromptError) Error() string {
	if e.Data.Error != "" {
		return e.Data.Error
	}
	return e.err.Error()
}

// Unwrap is the *acp.RequestError the agent answered.
func (e *PromptError) Unwrap() error { return e.err }

// promptError is err as a *PromptError when it is a request error whose
// data decodes as an ErrorData that says something; otherwise err. Members
// this client does not know are ignored, so a newer agent's data still
// decodes; data that is no ErrorData, such as another agent's, leaves err
// as it is.
func promptError(err error) error {
	re, ok := errors.AsType[*acp.RequestError](err)
	if !ok || re.Data == nil {
		return err
	}
	b, merr := jsonx.Marshal(re.Data)
	if merr != nil {
		return err
	}
	var d acpext.ErrorData
	if jsonx.Unmarshal(b, &d) != nil || d == (acpext.ErrorData{}) {
		return err
	}
	return &PromptError{Code: re.Code, Data: d, err: re}
}
