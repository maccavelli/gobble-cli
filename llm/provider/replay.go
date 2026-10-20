package provider

import (
	"errors"
	"strings"

	"github.com/maccavelli/gobble-cli/llm"
)

// noResult is the result a call left unanswered is given, Pi's words.
const noResult = "No result provided"

// replay is msgs as a request to provider's model sends them, by Pi's
// transformMessages rules (packages/ai/src/api/transform-messages.ts), keyed
// by model (0025-MADR D2):
//   - a message from another model, of another provider or the same one,
//     loses its thinking, and its text and tool calls their signatures; a
//     model's wire is fixed by its provider and id, so those two are
//     Pi's provider, API and id;
//   - an errored or aborted turn is not replayed, nor the results of its
//     calls, which would reach the provider with no call;
//   - a call left with no result gets "No result provided", as an error,
//     before the next assistant or user message, and at the end.
//
// A message whose origin is unknown, written before 0025, is sent as it
// was. msgs is not changed. Call ids a wire refuses are the SDK's to remap
// (RemapCallIDs).
func replay(msgs []llm.Message, provider, model string) []llm.Message {
	out := make([]llm.Message, 0, len(msgs))
	skipped := map[string]bool{} // the calls of turns not replayed
	var pending []llm.Content    // the last assistant message's calls
	answered := map[string]bool{}
	closePending := func() {
		var results []llm.Content
		for _, c := range pending {
			if !answered[c.ToolCallID] {
				results = append(results, llm.Content{Type: llm.ContentToolResult, ToolCallID: c.ToolCallID, ToolName: c.ToolName, Text: noResult, IsError: true})
			}
		}
		if len(results) > 0 {
			out = append(out, llm.Message{Role: llm.RoleTool, Content: results})
		}
		pending, answered = nil, map[string]bool{}
	}
	for _, m := range msgs {
		switch m.Role {
		case llm.RoleAssistant:
			closePending()
			if m.Origin.Failed {
				for _, c := range m.Content {
					if c.Type == llm.ContentToolCall {
						skipped[c.ToolCallID] = true
					}
				}
				continue
			}
			if m.Origin != (llm.Origin{}) && (m.Origin.Provider != provider || m.Origin.Model != model) {
				m = foreign(m)
			}
			for _, c := range m.Content {
				if c.Type == llm.ContentToolCall {
					pending = append(pending, c)
				}
			}
			out = append(out, m)
		case llm.RoleTool:
			var kept []llm.Content
			for _, c := range m.Content {
				if c.Type == llm.ContentToolResult && skipped[c.ToolCallID] {
					continue
				}
				answered[c.ToolCallID] = true
				kept = append(kept, c)
			}
			if len(kept) > 0 {
				m.Content = kept
				out = append(out, m)
			}
		case llm.RoleUser:
			closePending()
			out = append(out, m)
		default:
			out = append(out, m)
		}
	}
	closePending()
	return out
}

// foreign is m as another model receives it: no thinking, and no
// signature on its text or tool calls.
func foreign(m llm.Message) llm.Message {
	var content []llm.Content
	for _, c := range m.Content {
		if c.Type == llm.ContentThinking {
			continue
		}
		c.ProviderData = nil
		content = append(content, c)
	}
	m.Content = content
	return m
}

// stripThinking is msgs with every thinking block, and every signature,
// removed, for the one retry of a request refused over a signature
// (0025-MADR Q4).
func stripThinking(msgs []llm.Message) []llm.Message {
	out := make([]llm.Message, len(msgs))
	for i, m := range msgs {
		if m.Role == llm.RoleAssistant {
			m = foreign(m)
		}
		out[i] = m
	}
	return out
}

// signatureMarks are the words the providers refuse a thinking signature or
// encrypted reasoning with: Anthropic's "Invalid `signature` in `thinking`
// block", OpenAI's "encrypted content … could not be verified", Gemini's
// "Thought signature is not valid".
var signatureMarks = []string{"signature", "encrypted content", "encrypted_content"}

// signatureRefused reports whether err is a provider refusing a request,
// HTTP 400, over a thinking signature or encrypted reasoning it cannot
// verify (0025-MADR Q4). The error's text is read here only, never shown.
func signatureRefused(err error) bool {
	ae, ok := errors.AsType[*llm.APIError](err)
	if !ok || ae.Status != 400 || errors.Is(err, llm.ErrAuth) {
		return false
	}
	text := strings.ToLower(err.Error())
	for _, mark := range signatureMarks {
		if strings.Contains(text, mark) {
			return true
		}
	}
	return false
}
