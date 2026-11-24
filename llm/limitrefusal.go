package llm

import (
	"errors"
	"regexp"
	"strconv"
)

// LimitRefusal is a refusal of a call's output limit read from its message
// (0071-MADR D8 (a)): a window form's room, the window it states less the
// input it states, or a cap form's stated maximum.
type LimitRefusal struct {
	// Tokens is the room or the cap; a room may be 0 or less, when the input
	// alone fills the window.
	Tokens int
	// Window reports a window form.
	Window bool
}

// limitForm is one refusal form: its pattern, and how its numbers give the
// room or the cap, sent being the limit the call carried.
type limitForm struct {
	pattern *regexp.Regexp
	window  bool
	read    func(n []int, sent int) int
}

// The seven forms, in go-llmprovider-sdk's records' words (0071-MADR F17):
// the cap forms first, then the window forms. Each pattern's groups are its
// numbers in the order the text states them.
var limitForms = []limitForm{
	// Anthropic: "max_tokens: N > M, which is the maximum allowed number of
	// output tokens".
	{regexp.MustCompile(`max_tokens: (\d+) > (\d+), which is the maximum allowed number of output tokens`), false,
		func(n []int, _ int) int { return n[1] }},
	// OpenAI's Chat Completions: "supports at most M completion tokens,
	// whereas you provided N".
	{regexp.MustCompile(`supports at most (\d+) completion tokens, whereas you provided (\d+)`), false,
		func(n []int, _ int) int { return n[0] }},
	// generateContent: "value of N but the supported range is from 1
	// (inclusive) to M (exclusive)": the most is M - 1.
	{regexp.MustCompile(`value of (\d+) but the supported range is from 1 \(inclusive\) to (\d+) \(exclusive\)`), false,
		func(n []int, _ int) int { return n[1] - 1 }},
	// Anthropic: "input length and `max_tokens` exceed context limit: I + N >
	// W".
	{regexp.MustCompile("input length and `?max_tokens`? exceed context limit: (\\d+) \\+ (\\d+) > (\\d+)"), true,
		func(n []int, _ int) int { return n[2] - n[0] }},
	// OpenAI's split, which Together sends too: "maximum context length is W
	// tokens ... (I in the messages, C in the completion)".
	{regexp.MustCompile(`maximum context length is (\d+) tokens.*?\((\d+) in the messages, (\d+) in the completion\)`), true,
		func(n []int, _ int) int { return n[0] - n[1] }},
	// OpenRouter's, as Kilo passes it: "maximum context length is W tokens.
	// However, you requested about T tokens", T the request's total.
	{regexp.MustCompile(`maximum context length is (\d+) tokens\. However, you requested about (\d+) tokens`), true,
		func(n []int, sent int) int { return n[0] - (n[1] - sent) }},
	// Kilo's: "exceeds the model's context window of W tokens. Your request
	// contains approximately T tokens", T the request's total.
	{regexp.MustCompile(`exceeds the model's context window of (\d+) tokens\. Your request contains approximately (\d+) tokens`), true,
		func(n []int, sent int) int { return n[0] - (n[1] - sent) }},
}

// ReadLimitRefusal reads err's message, an APIError's Message when it has
// one and otherwise err's text, as a refusal of the output limit sent, in
// one of the forms of 0071-MADR F17. It is false when no form matches.
func ReadLimitRefusal(err error, sent int) (LimitRefusal, bool) {
	if err == nil {
		return LimitRefusal{}, false
	}
	msg := err.Error()
	if ae, ok := errors.AsType[*APIError](err); ok && ae.Message != "" {
		msg = ae.Message
	}
	for _, f := range limitForms {
		m := f.pattern.FindStringSubmatch(msg)
		if m == nil {
			continue
		}
		n := make([]int, len(m)-1)
		for i, s := range m[1:] {
			v, err := strconv.Atoi(s)
			if err != nil {
				return LimitRefusal{}, false
			}
			n[i] = v
		}
		return LimitRefusal{Tokens: f.read(n, sent), Window: f.window}, true
	}
	return LimitRefusal{}, false
}
