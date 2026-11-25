package llm

import "time"

// SentLimit is the output limit a call carried on the wire, as
// go-llmprovider-sdk reports it on the call's done response (its
// Response.OutputLimit, 0089-MADR D7): Tokens is the value the field
// carried, after a thinking budget raised it, 0 when Sent is false; Sent
// reports whether the request carried a limit at all; Basis is the SDK's
// reason, such as "request" for a caller's value or "none"; Source names the
// evidence of a value the SDK chose. A provider that does not report one
// leaves it zero (0071-MADR P14).
type SentLimit struct {
	Tokens int    `json:"tokens,omitzero"`
	Sent   bool   `json:"sent,omitzero"`
	Basis  string `json:"basis,omitzero"`
	Source string `json:"source,omitzero"`
}

// LimitObservation is an output cap a service stated when it refused a
// call's output limit, which go-llmprovider-sdk validated and kept for the
// process (its APIError.Observed, 0089-MADR DL78): the model refused, the
// provider's base URL and scope as the SDK names them (empty for the
// default, and for none), the cap, and when the refusal was read. It is zero
// when no cap was kept, as for a window form (0071-MADR P15).
type LimitObservation struct {
	Model   string
	BaseURL string
	Scope   string
	Tokens  int
	Seen    time.Time
}
