package acpserver

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"strings"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/tool"
)

// sdkAPI is the api an assistant message names: the SDK gobble speaks
// through (0002-PLAN Phase 4 amendment).
const sdkAPI = "go-llmprovider-sdk"

// reasoning is llm/provider's provider data for a reasoning block or a tool
// call's signature.
type reasoning struct {
	Text      string `json:"text,omitzero"`
	Signature string `json:"signature,omitzero"`
	Encrypted string `json:"encrypted,omitzero"`
	Format    string `json:"format,omitzero"`
}

// gobbleMember is a block's extra member holding gobble's own data, which Pi
// ignores.
type gobbleMember struct {
	Gobble jsontext.Value `json:"gobble,omitzero"`
}

func extra(v any) jsontext.Value {
	b, err := json.Marshal(gobbleMember{Gobble: mustJSON(v)})
	if err != nil {
		panic(err)
	}
	return b
}

func mustJSON(v any) jsontext.Value {
	b, err := json.Marshal(v)
	if err != nil { // values of plain fields always encode
		panic(err)
	}
	return b
}

func ms(ts string) int64 {
	return parseTimestamp(ts).UnixMilli()
}

// userEntryMessage is a prompt as Pi's user message: one text is a string.
func userEntryMessage(m llm.Message, ts string) *session.Message {
	var texts []string
	for _, c := range m.Content {
		if c.Type == llm.ContentText {
			texts = append(texts, c.Text)
		}
	}
	return &session.Message{Role: session.RoleUser, Content: mustJSON(strings.Join(texts, "\n\n")), Timestamp: ms(ts)}
}

// assistantEntryMessage is an assistant message in Pi's shape.
func assistantEntryMessage(ev agent.Message, provider, model, ts string) *session.Message {
	var blocks []session.Block
	for _, c := range ev.Message.Content {
		switch c.Type {
		case llm.ContentThinking:
			b := session.Block{Type: session.BlockThinking, Thinking: c.Text}
			var r reasoning
			if len(c.ProviderData) > 0 && json.Unmarshal(c.ProviderData, &r) == nil {
				b.ThinkingSignature = r.Signature
				if r.Encrypted != "" && r.Signature == "" {
					b.ThinkingSignature, b.Redacted = r.Encrypted, r.Text == ""
				}
				b.Unknown = extra(c.ProviderData)
			}
			blocks = append(blocks, b)
		case llm.ContentText:
			blocks = append(blocks, session.Block{Type: session.BlockText, Text: c.Text})
		case llm.ContentToolCall:
			b := session.Block{Type: session.BlockToolCall, ID: c.ToolCallID, Name: c.ToolName}
			var obj map[string]any
			if json.Unmarshal([]byte(c.ToolArgs), &obj) == nil && obj != nil {
				b.Arguments = jsontext.Value(c.ToolArgs)
			} else {
				b.Arguments = jsontext.Value("{}")
				b.Unknown = extra(map[string]string{"rawArguments": c.ToolArgs})
			}
			var r reasoning
			if len(c.ProviderData) > 0 && json.Unmarshal(c.ProviderData, &r) == nil {
				b.ThoughtSignature = r.Signature
			}
			blocks = append(blocks, b)
		}
	}
	m := &session.Message{Role: session.RoleAssistant, API: sdkAPI, Provider: provider, Model: model,
		Usage: piUsage(ev.Usage), StopReason: piStop(ev.StopReason), Timestamp: ms(ts)}
	_ = m.SetBlocks(blocks) //nolint:errcheck // blocks of plain fields always encode
	return m
}

// toolResultMessage is one tool call's result in Pi's shape.
func toolResultMessage(id, name, text string, isError bool, ts string) *session.Message {
	m := &session.Message{Role: session.RoleToolResult, ToolCallID: id, ToolName: name, IsError: new(isError), Timestamp: ms(ts)}
	_ = m.SetBlocks([]session.Block{{Type: session.BlockText, Text: text}}) //nolint:errcheck // as above
	return m
}

func piUsage(u llm.Usage) *session.Usage {
	out := &session.Usage{Input: u.InputTokens, Output: u.OutputTokens, CacheRead: u.CacheReadTokens, CacheWrite: u.CacheWriteTokens,
		Reasoning: u.ReasoningTokens, TotalTokens: u.InputTokens + u.OutputTokens}
	if u.Estimated {
		out.Unknown = jsontext.Value(`{"estimated":true}`)
	}
	return out
}

// piStop is llm's stop reason as Pi names it.
func piStop(s llm.StopReason) string {
	switch s {
	case llm.StopToolUse:
		return "toolUse"
	case llm.StopMaxTokens:
		return "length"
	case "":
		return "aborted"
	}
	return "stop"
}

// state is what a session's entries say: the context messages, the model
// and thinking chosen, the name, and the last call's usage.
type state struct {
	history         []llm.Message
	provider, model string
	thinking, name  string
	lastUsed        int64
	leaf            string
	ids             map[string]bool
	results         map[string]*session.Message // tool call id → its result
	messages        []session.Entry             // the path's message entries
	context         []session.Entry             // the entries the model sees
	path            []session.Entry
}

// replayState reads entries: model_change, thinking_level_change,
// session_info and the assistant messages along the active path set the
// choices, as Pi's buildSessionContext reads them (session-manager.ts
// :418-433), and the history is the path's model context, which starts at
// the newest compaction's summary when there is one (compaction.Context).
// Entry types gobble does not act on yet take no part in context
// (0005-PLAN F2).
func replayState(entries []session.Entry) state {
	st := state{ids: map[string]bool{}, results: map[string]*session.Message{}, thinking: thinkingOff}
	for _, e := range entries {
		st.ids[e.ID] = true
		if e.Type == session.TypeSessionInfo && e.Name != nil {
			st.name = *e.Name
		}
	}
	if len(entries) > 0 {
		st.leaf = entries[len(entries)-1].ID
	}
	st.path = session.Path(entries)
	for _, e := range st.path {
		switch e.Type {
		case session.TypeModelChange:
			st.provider, st.model = e.Provider, e.ModelID
		case session.TypeThinkingLevelChange:
			st.thinking = e.ThinkingLevel
		case session.TypeMessage:
			if e.Message == nil {
				continue
			}
			st.messages = append(st.messages, e)
			if e.Message.Role == session.RoleAssistant && e.Message.Provider != "" {
				st.provider, st.model = e.Message.Provider, e.Message.Model
			}
		}
	}
	st.context = compaction.Context(st.path)
	for i, e := range st.context {
		switch {
		case e.Type == session.TypeCompaction && i == 0:
			st.history = append(st.history, llm.Message{Role: llm.RoleUser,
				Content: []llm.Content{{Type: llm.ContentText, Text: compaction.SummaryText(e.Summary)}}})
		case e.Type == session.TypeMessage && e.Message != nil:
			st.addMessage(e.Message)
		}
	}
	return st
}

func (st *state) addMessage(m *session.Message) {
	blocks, err := m.Blocks()
	if err != nil {
		return
	}
	switch m.Role {
	case session.RoleUser:
		var c []llm.Content
		for _, b := range blocks {
			if b.Type == session.BlockText {
				c = append(c, llm.Content{Type: llm.ContentText, Text: b.Text})
			}
		}
		st.history = append(st.history, llm.Message{Role: llm.RoleUser, Content: c})
	case session.RoleAssistant:
		if m.Usage != nil {
			st.lastUsed = m.Usage.Input + m.Usage.Output
		}
		st.history = append(st.history, llm.Message{Role: llm.RoleAssistant, Content: assistantContent(blocks)})
	case session.RoleToolResult:
		st.results[m.ToolCallID] = m
		c := llm.Content{Type: llm.ContentToolResult, ToolCallID: m.ToolCallID, ToolName: m.ToolName, Text: m.Text(), IsError: m.IsError != nil && *m.IsError}
		// Consecutive results are one tool message, as the agent sends them.
		if n := len(st.history); n > 0 && st.history[n-1].Role == llm.RoleTool {
			st.history[n-1].Content = append(st.history[n-1].Content, c)
			return
		}
		st.history = append(st.history, llm.Message{Role: llm.RoleTool, Content: []llm.Content{c}})
	}
}

func assistantContent(blocks []session.Block) []llm.Content {
	var out []llm.Content
	for _, b := range blocks {
		var g gobbleMember
		if len(b.Unknown) > 0 {
			_ = json.Unmarshal(b.Unknown, &g) //nolint:errcheck // no gobble member is none
		}
		switch b.Type {
		case session.BlockThinking:
			data := g.Gobble
			if len(data) == 0 && (b.ThinkingSignature != "" || b.Redacted) {
				r := reasoning{Text: b.Thinking, Signature: b.ThinkingSignature}
				if b.Redacted {
					r = reasoning{Encrypted: b.ThinkingSignature}
				}
				data = mustJSON(r)
			}
			out = append(out, llm.Content{Type: llm.ContentThinking, Text: b.Thinking, ProviderData: data})
		case session.BlockText:
			out = append(out, llm.Content{Type: llm.ContentText, Text: b.Text})
		case session.BlockToolCall:
			args := string(b.Arguments)
			var raw struct {
				RawArguments string `json:"rawArguments"`
			}
			if len(g.Gobble) > 0 && json.Unmarshal(g.Gobble, &raw) == nil && raw.RawArguments != "" {
				args = raw.RawArguments
			}
			c := llm.Content{Type: llm.ContentToolCall, ToolCallID: b.ID, ToolName: b.Name, ToolArgs: args}
			if b.ThoughtSignature != "" {
				c.ProviderData = mustJSON(reasoning{Signature: b.ThoughtSignature})
			}
			out = append(out, c)
		}
	}
	return out
}

// replayUpdates is session/load's compact replay (0008-MADR D19 item 5) of
// what the model sees: a compaction as one agent_message_chunk with its
// summary, one user_message_chunk per user message, one
// agent_message_chunk per assistant message with its whole text, and one
// tool_call per call in its final state. The session-start frames follow
// them, sent by the caller.
func replayUpdates(st state, tools map[string]tool.Tool, cwd string) []acp.SessionUpdate {
	var out []acp.SessionUpdate
	for i, e := range st.context {
		if e.Type == session.TypeCompaction && i == 0 {
			out = append(out, acp.UpdateAgentMessageText(compactedText(e)))
			continue
		}
		if e.Type != session.TypeMessage || e.Message == nil {
			continue
		}
		m := e.Message
		blocks, err := m.Blocks()
		if err != nil {
			continue
		}
		switch m.Role {
		case session.RoleUser:
			if t := m.Text(); t != "" {
				out = append(out, acp.UpdateUserMessageText(t))
			}
		case session.RoleAssistant:
			if t := assistantText(blocks); t != "" {
				out = append(out, acp.UpdateAgentMessageText(t))
			}
			for _, b := range blocks {
				if b.Type == session.BlockToolCall {
					out = append(out, replayToolCall(b, st.results[b.ID], tools, cwd))
				}
			}
		}
	}
	return out
}

// compactedText is a compaction as the client shows it.
func compactedText(e session.Entry) string {
	var before int64
	if e.TokensBefore != nil {
		before = *e.TokensBefore
	}
	return fmt.Sprintf("Compacted from %d tokens:\n\n%s", before, e.Summary)
}

func assistantText(blocks []session.Block) string {
	var parts []string
	for _, b := range blocks {
		if b.Type == session.BlockText && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "")
}

// replayToolCall is a call in its final state: completed or failed by its
// result, or failed when it has none (the turn was cut short).
func replayToolCall(b session.Block, result *session.Message, tools map[string]tool.Tool, cwd string) acp.SessionUpdate {
	call := tool.Call{ID: b.ID, Name: b.Name, Args: b.Arguments}
	title, kind := b.Name, acp.ToolKind(tool.KindOther)
	if t, ok := tools[b.Name]; ok {
		kind = toolKind(t.Spec().Kind)
		if d, ok := t.(tool.Describer); ok {
			title = d.Describe(call, tool.Env{Cwd: cwd})
		}
	}
	status, text := acp.ToolCallStatusFailed, "no result was recorded"
	if result != nil {
		text = result.Text()
		if result.IsError == nil || !*result.IsError {
			status = acp.ToolCallStatusCompleted
		}
	}
	return acp.StartToolCall(acp.ToolCallId(b.ID), cutTitle(title), acp.WithStartKind(kind), acp.WithStartStatus(status),
		acp.WithStartRawInput(jsonRaw(b.Arguments)),
		acp.WithStartContent([]acp.ToolCallContent{acp.ToolContent(acp.TextBlock(cutBytes(firstLine(text), maxSummary)))}))
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
