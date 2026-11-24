package llm_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// refusal is a 400 carrying msg as the provider's message, as gobble's
// adapter maps one, with the SDK's text in its chain.
func refusal(msg string) error {
	return &llm.APIError{Status: 400, Message: msg, Err: fmt.Errorf("invalid request: %s", msg)}
}

// ReadLimitRefusal reads the seven forms of 0071-MADR F17, in the texts of
// go-llmprovider-sdk's records (window_0088_record_test.go:148-162 and
// internal/wire/limit_refusal.go:102-115 at 33e6f09): a window form's room,
// a cap form's cap; a room at or below 0 is read, for the overflow path;
// any other text is not (L39).
func TestReadLimitRefusal(t *testing.T) {
	for _, c := range []struct {
		name string
		err  error
		sent int
		want llm.LimitRefusal
		ok   bool
	}{
		{"anthropic_window", refusal("input length and `max_tokens` exceed context limit: 184000 + 32000 > 200000 (req_REDACTED, wrkspc_REDACTED, 00000000-0000-0000-0000-000000000000)"),
			32000, llm.LimitRefusal{Tokens: 16000, Window: true}, true},
		{"openrouter_window", refusal("This endpoint's maximum context length is 131072 tokens. However, you requested about 195072 tokens (131072 of text input, 64000 in the output). Please reduce the length of either one."),
			64000, llm.LimitRefusal{Tokens: 0, Window: true}, true},
		{"openrouter_window_room", refusal("This endpoint's maximum context length is 131072 tokens. However, you requested about 164000 tokens (100000 of text input, 64000 in the output)."),
			64000, llm.LimitRefusal{Tokens: 31072, Window: true}, true},
		{"kilo_window", refusal("Your request exceeds the model's context window of 200000 tokens. Your request contains approximately 230000 tokens."),
			64000, llm.LimitRefusal{Tokens: 34000, Window: true}, true},
		{"split_window", refusal("This model's maximum context length is 128000 tokens. However, you requested 160000 tokens (96000 in the messages, 64000 in the completion). Please reduce the length of the messages or completion."),
			64000, llm.LimitRefusal{Tokens: 32000, Window: true}, true},
		{"anthropic_cap", refusal("max_tokens: 64000 > 32000, which is the maximum allowed number of output tokens for claude-opus-4-1-20250805"),
			64000, llm.LimitRefusal{Tokens: 32000}, true},
		{"openai_cap", refusal("max_tokens is too large: 64000. This model supports at most 16384 completion tokens, whereas you provided 64000."),
			64000, llm.LimitRefusal{Tokens: 16384}, true},
		{"gemini_range", refusal("Unable to submit request because it has a maxOutputTokens value of 64000 but the supported range is from 1 (inclusive) to 65537 (exclusive). Update the value and try again."),
			64000, llm.LimitRefusal{Tokens: 65536}, true},
		{"room_below_zero", refusal("input length and `max_tokens` exceed context limit: 210000 + 32000 > 200000"),
			32000, llm.LimitRefusal{Tokens: -10000, Window: true}, true},
		{"from_the_text", fmt.Errorf("Summarization failed: %w", errors.New("max_tokens: 64000 > 8192, which is the maximum allowed number of output tokens")),
			64000, llm.LimitRefusal{Tokens: 8192}, true},
		{"not_a_form", refusal("max_tokens must be an integer"), 64000, llm.LimitRefusal{}, false},
		{"other_error", llm.ErrContextOverflow, 64000, llm.LimitRefusal{}, false},
		{"nil", nil, 64000, llm.LimitRefusal{}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := llm.ReadLimitRefusal(c.err, c.sent)
			if got != c.want || ok != c.ok {
				t.Errorf("ReadLimitRefusal = %+v, %v; want %+v, %v", got, ok, c.want, c.ok)
			}
		})
	}
}
