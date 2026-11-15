package llm

// ConsentError is a request refused because a consent is not granted: Consent
// is the consent's id, the SDK's, such as chatgpt_plan.background_use, and
// Err the refusal, which matches ErrNotPermitted so that Kind gives
// not_permitted. It is set only when allowing that consent would let the
// request through (0052-MADR D2).
type ConsentError struct {
	Consent string
	Err     error
}

// Error is Err's text.
func (e *ConsentError) Error() string {
	if e == nil || e.Err == nil {
		return "llm: consent not granted"
	}
	return e.Err.Error()
}

// Unwrap gives Err.
func (e *ConsentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
