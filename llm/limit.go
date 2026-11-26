package llm

// The numbers of a call's output limit (0071-MADR D2, "Every number"):
// ContextSafetyTokens is pi's margin left free in the window
// (CONTEXT_SAFETY_TOKENS); OutputCeiling bounds a cap the user did not set
// (the owner's Q3, qwen-code's OUTPUT_TOKEN_CEILING); MinOutputTokens is
// kilocode's floor (MIN_OUTPUT); AssumedMaxTokens is the cap of a model no
// source states, Pi's default for a model models.json defines (the owner's
// Q2).
const (
	ContextSafetyTokens = 4096
	OutputCeiling       = 64000
	MinOutputTokens     = 1024
	AssumedMaxTokens    = 16384
)

// CapSource is where a model's output cap came from (0071-MADR D1).
type CapSource string

// The sources. A zero CapSource is CapCatalog when the cap is known, and
// CapAssumed when it is not.
const (
	// CapCatalog is the models.dev document gobble reads (0072-MADR D7) or
	// the provider's listing.
	CapCatalog CapSource = "catalog"
	// CapUser is models.json's maxTokens for the model, an entry's or an
	// override's.
	CapUser CapSource = "user"
	// CapAssumed is AssumedMaxTokens: no source states the cap.
	CapAssumed CapSource = "assumed"
)

// LimitBasis is why a call asked for the output limit it did (0071-MADR D2,
// D8, D10).
type LimitBasis string

// The bases.
const (
	// BasisStated is the model's cap as the catalog states it.
	BasisStated LimitBasis = "stated"
	// BasisCeiling is OutputCeiling, which cut a larger cap.
	BasisCeiling LimitBasis = "ceiling"
	// BasisUser is models.json's maxTokens.
	BasisUser LimitBasis = "user"
	// BasisAssumed is AssumedMaxTokens.
	BasisAssumed LimitBasis = "assumed"
	// BasisWindow is the room the window leaves, or the floor.
	BasisWindow LimitBasis = "window"
	// BasisFull is the cap, sent when the window is full by the estimate, so
	// the service's refusal starts the overflow path.
	BasisFull LimitBasis = "full"
	// BasisRefusal is the room or cap a service stated when it refused the
	// call's first limit.
	BasisRefusal LimitBasis = "refusal"
	// BasisNone is a call that carried no output limit, Tokens 0, such as
	// one on a ChatGPT route (0071-MADR P22, P35).
	BasisNone LimitBasis = "none"
	// BasisService is a cap a service stated, sent below models.json's
	// maxTokens for the model, which it holds (0071-MADR P28).
	BasisService LimitBasis = "service"
)

// OutputLimit is the output limit a call asks for, reasoning included, and
// why.
type OutputLimit struct {
	Tokens int        `json:"tokens"`
	Basis  LimitBasis `json:"basis"`
}

// FitOutput is a call's output limit (0071-MADR D2): the model's cap, from
// its source, under OutputCeiling unless the user set it, then fitted to
// the room the window leaves, window less the context's estimate less
// ContextSafetyTokens. A cap of 0 or less is unknown: AssumedMaxTokens,
// basis assumed. A window of 0 or less is unknown, and the cap is sent as it
// is. Room of at least MinOutputTokens but below the cap is sent; room below
// MinOutputTokens sends the cap held to MinOutputTokens; no room sends the
// cap, basis full, for the service to refuse as an overflow.
func FitOutput(capTokens int, from CapSource, window, estimate int) OutputLimit {
	if capTokens <= 0 {
		capTokens, from = AssumedMaxTokens, CapAssumed
	}
	stated := OutputLimit{Tokens: capTokens, Basis: BasisStated}
	switch {
	case from == CapUser:
		stated.Basis = BasisUser
	case capTokens > OutputCeiling:
		stated = OutputLimit{Tokens: OutputCeiling, Basis: BasisCeiling}
	case from == CapAssumed:
		stated.Basis = BasisAssumed
	}
	if window <= 0 {
		return stated
	}
	room := window - estimate - ContextSafetyTokens
	switch {
	case room >= stated.Tokens:
		return stated
	case room >= MinOutputTokens:
		return OutputLimit{Tokens: room, Basis: BasisWindow}
	case room > 0:
		return OutputLimit{Tokens: min(stated.Tokens, MinOutputTokens), Basis: BasisWindow}
	}
	return OutputLimit{Tokens: stated.Tokens, Basis: BasisFull}
}
