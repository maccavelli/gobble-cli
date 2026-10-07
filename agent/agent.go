package agent

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"iter"
	"strings"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
)

// DefaultMaxCalls is how many model calls one prompt may make before the
// turn ends with StopMaxTurnRequests.
const DefaultMaxCalls = 100

// Config configures an Agent.
type Config struct {
	Provider llm.Provider
	Tools    []tool.Tool
	// System is the system prompt.
	System string
	// Model overrides the provider's model; empty keeps it.
	Model string
	// Thinking is the thinking level each request asks for (llm's levels);
	// empty asks for none.
	Thinking string
	// Policy approves calls to tools that are not read-only. Nil allows
	// every call.
	Policy permission.Policy
	// MaxCalls bounds the model calls of one prompt; zero is
	// DefaultMaxCalls.
	MaxCalls int
}

// Agent runs turns: model calls and the tool calls they ask for, until the
// model ends its reply. It is safe to run turns of different sessions at
// once; one session's turns run one after another.
type Agent struct {
	cfg   Config
	tools map[string]tool.Tool
	specs jsontext.Value
}

// New returns an Agent. It panics if two tools share a name, which is a
// programming error.
func New(cfg Config) *Agent {
	if cfg.MaxCalls <= 0 {
		cfg.MaxCalls = DefaultMaxCalls
	}
	a := &Agent{cfg: cfg, tools: map[string]tool.Tool{}}
	specs := make([]tool.Spec, 0, len(cfg.Tools))
	for _, t := range cfg.Tools {
		s := t.Spec()
		if _, dup := a.tools[s.Name]; dup {
			panic("agent: two tools are named " + s.Name)
		}
		a.tools[s.Name] = t
		specs = append(specs, s)
	}
	if len(specs) > 0 {
		b, err := json.Marshal(specs)
		if err != nil {
			panic(fmt.Sprintf("agent: encode tool specs: %v", err))
		}
		a.specs = b
	}
	return a
}

// Run runs one prompt over history and yields the turn's events, ending
// with End. A done ctx ends the turn with StopCancelled, not an error; a
// provider failure is yielded as an error and ends the stream.
func (a *Agent) Run(ctx context.Context, env tool.Env, history []llm.Message, prompt llm.Message) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		t := &turn{a: a, ctx: ctx, env: env, yield: yield, history: history, added: []llm.Message{prompt}}
		t.run()
	}
}

// turn is one Run's state.
type turn struct {
	a       *Agent
	ctx     context.Context
	env     tool.Env
	yield   func(Event, error) bool
	history []llm.Message
	added   []llm.Message
	usage   llm.Usage
	stopped bool // the consumer stopped iterating
}

func (t *turn) emit(ev Event) bool {
	if t.stopped {
		return false
	}
	if !t.yield(ev, nil) {
		t.stopped = true
	}
	return !t.stopped
}

func (t *turn) end(reason StopReason) {
	t.emit(End{StopReason: reason, Messages: t.added, Usage: t.usage})
}

func (t *turn) run() {
	for calls := 0; ; calls++ {
		if calls == t.a.cfg.MaxCalls {
			t.end(StopMaxTurnRequests)
			return
		}
		reply, ok := t.call()
		if !ok {
			return
		}
		msg := reply.message()
		t.added = append(t.added, msg)
		if !t.emit(Message{Message: msg, Usage: reply.usage, StopReason: reply.stop}) {
			return
		}
		if t.ctx.Err() != nil {
			t.added = append(t.added, cancelledResults(reply.calls)...)
			t.end(StopCancelled)
			return
		}
		if len(reply.calls) == 0 {
			if reply.stop == llm.StopMaxTokens {
				t.end(StopMaxTokens)
			} else {
				t.end(StopEndTurn)
			}
			return
		}
		results, ok := t.runTools(reply.calls)
		if !ok {
			return
		}
		t.added = append(t.added, results)
		if t.ctx.Err() != nil {
			t.end(StopCancelled)
			return
		}
	}
}

// reply is one model call's answer.
type reply struct {
	text     strings.Builder
	thinking []llm.Content
	pending  strings.Builder // thinking text not yet in a block
	calls    []llm.ToolCallDone
	stop     llm.StopReason
	usage    llm.Usage
}

func (r *reply) message() llm.Message {
	m := llm.Message{Role: llm.RoleAssistant}
	m.Content = append(m.Content, r.thinking...)
	if r.pending.Len() > 0 {
		m.Content = append(m.Content, llm.Content{Type: llm.ContentThinking, Text: r.pending.String()})
	}
	if r.text.Len() > 0 {
		m.Content = append(m.Content, llm.Content{Type: llm.ContentText, Text: r.text.String()})
	}
	for _, c := range r.calls {
		m.Content = append(m.Content, llm.Content{Type: llm.ContentToolCall, ToolCallID: c.ID, ToolName: c.Name, ToolArgs: c.Args, ProviderData: c.ProviderData})
	}
	return m
}

// call makes one model call, yielding its deltas. ok is false when the
// stream failed or the consumer stopped; a failure has been yielded.
func (t *turn) call() (*reply, bool) {
	msgs := make([]llm.Message, 0, len(t.history)+len(t.added)+1)
	if t.a.cfg.System != "" {
		msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: []llm.Content{{Type: llm.ContentText, Text: t.a.cfg.System}}})
	}
	msgs = append(append(msgs, t.history...), t.added...)
	req := &llm.Request{Model: t.a.cfg.Model, Messages: msgs, Tools: t.a.specs, Thinking: t.a.cfg.Thinking}
	r := &reply{}
	for ev, err := range t.a.cfg.Provider.Stream(t.ctx, req) {
		if err != nil {
			if t.ctx.Err() != nil {
				return r, !t.stopped
			}
			if !t.stopped {
				t.yield(nil, err)
				t.stopped = true
			}
			return r, false
		}
		switch ev := ev.(type) {
		case llm.TextDelta:
			r.text.WriteString(ev.Text)
			if ev.Text != "" && !t.emit(TextDelta{Text: ev.Text}) {
				return r, false
			}
		case llm.ThinkingDelta:
			r.pending.WriteString(ev.Text)
			if len(ev.ProviderData) > 0 {
				// A delta carrying provider data closes a reasoning block, which
				// is replayed to the provider as it was given.
				r.thinking = append(r.thinking, llm.Content{Type: llm.ContentThinking, Text: r.pending.String(), ProviderData: ev.ProviderData})
				r.pending.Reset()
			}
			if ev.Text != "" && !t.emit(ThinkingDelta{Text: ev.Text}) {
				return r, false
			}
		case llm.ToolCallDone:
			r.calls = append(r.calls, ev)
		case llm.Usage:
			t.usage = addUsage(t.usage, ev)
			r.usage = addUsage(r.usage, ev)
			if !t.emit(Usage{Usage: ev}) {
				return r, false
			}
		case llm.Done:
			r.stop = ev.StopReason
		}
	}
	return r, !t.stopped
}

// runTools runs the calls one after another and returns their results as
// one tool message. After a cancel, the calls not run get a cancelled
// result, so every call in the history has one.
func (t *turn) runTools(calls []llm.ToolCallDone) (llm.Message, bool) {
	m := llm.Message{Role: llm.RoleTool}
	for i, c := range calls {
		if t.ctx.Err() != nil {
			m.Content = append(m.Content, cancelledResults(calls[i:])[0].Content...)
			break
		}
		res, ok := t.runTool(c)
		if !ok {
			return m, false
		}
		m.Content = append(m.Content, llm.Content{Type: llm.ContentToolResult, ToolCallID: c.ID, ToolName: c.Name, Text: res.Text(), IsError: res.IsError})
	}
	return m, true
}

func (t *turn) runTool(c llm.ToolCallDone) (tool.Result, bool) {
	call := tool.Call{ID: c.ID, Name: c.Name, Args: jsontext.Value(c.Args)}
	tl, known := t.a.tools[c.Name]
	if !known {
		res := tool.ErrorResult(fmt.Sprintf("there is no tool named %q", c.Name))
		return res, t.emit(ToolStart{Call: call, Title: c.Name}) && t.emit(ToolEnd{Call: call, Result: res})
	}
	spec := tl.Spec()
	title := spec.Name
	if d, ok := tl.(tool.Describer); ok {
		title = d.Describe(call, t.env)
	}
	if !t.emit(ToolStart{Call: call, Spec: spec, Title: title}) {
		return tool.Result{}, false
	}
	if res, denied := t.approve(call, spec, title); denied {
		return res, t.emit(ToolEnd{Call: call, Result: res})
	}
	if !t.emit(ToolRun{Call: call}) {
		return tool.Result{}, false
	}
	res, err := tl.Run(t.ctx, call, t.env)
	if err != nil {
		res = tool.ErrorResult(cancelledText)
		if t.ctx.Err() == nil {
			res = tool.ErrorResult(fmt.Sprintf("%s failed: %v", spec.Name, err))
		}
	}
	return res, t.emit(ToolEnd{Call: call, Result: res})
}

// approve asks the policy about a call to a tool that is not read-only. It
// returns the result to give the model when the call does not run.
func (t *turn) approve(call tool.Call, spec tool.Spec, title string) (tool.Result, bool) {
	if t.a.cfg.Policy == nil || spec.Annotations.ReadOnlyHint {
		return tool.Result{}, false
	}
	d, err := t.a.cfg.Policy.Decide(t.ctx, permission.Rule{Tool: spec.Name, Mode: permission.ModeAsk, CallID: call.ID, Title: title, Input: call.Args})
	switch {
	case t.ctx.Err() != nil:
		return tool.ErrorResult(cancelledText), true
	case err != nil:
		return tool.ErrorResult(fmt.Sprintf("the permission request failed, so %s did not run: %v", spec.Name, err)), true
	case !d.Allow && d.Refusal != "":
		return tool.ErrorResult(d.Refusal), true
	case !d.Allow:
		msg := "The user declined this call, so it did not run."
		if d.Reason != "" {
			msg += " " + d.Reason
		}
		return tool.ErrorResult(msg), true
	}
	return tool.Result{}, false
}

const cancelledText = "The user cancelled the turn, so this call did not finish."

// cancelledResults is a tool message answering each call with the
// cancelled result, or nothing when there are no calls.
func cancelledResults(calls []llm.ToolCallDone) []llm.Message {
	if len(calls) == 0 {
		return nil
	}
	m := llm.Message{Role: llm.RoleTool}
	for _, c := range calls {
		m.Content = append(m.Content, llm.Content{Type: llm.ContentToolResult, ToolCallID: c.ID, ToolName: c.Name, Text: cancelledText, IsError: true})
	}
	return []llm.Message{m}
}

func addUsage(a, b llm.Usage) llm.Usage {
	return llm.Usage{
		InputTokens:      a.InputTokens + b.InputTokens,
		OutputTokens:     a.OutputTokens + b.OutputTokens,
		ReasoningTokens:  a.ReasoningTokens + b.ReasoningTokens,
		CacheReadTokens:  a.CacheReadTokens + b.CacheReadTokens,
		CacheWriteTokens: a.CacheWriteTokens + b.CacheWriteTokens,
		Estimated:        a.Estimated || b.Estimated,
	}
}
