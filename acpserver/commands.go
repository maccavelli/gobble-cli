package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/command"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/session"
)

// commands are the slash commands Prompt executes, frozen by 0002-PLAN
// Phase 6 step 6. Every one is advertised, and only these are: a test
// sends each and asserts it is handled (advertisement ⊆ handlers). TUI
// chrome is never here.
var commands = []command.Command{
	{Name: "help", Description: "List the commands"},
	{Name: "compact", Description: "Compact the session context, optionally focused by instructions", Hint: "[instructions]"},
	{Name: "usage", Description: "Show the session's tokens and cost"},
	{Name: "context", Description: "Show context-window use and session information"},
	{Name: "session", Description: "Show session information and statistics"},
	{Name: "model", Description: "Show or set the model", Hint: "[model]"},
	{Name: "thinking", Description: "Show or set the thinking level", Hint: "[level]"},
	{Name: "mode", Description: "Show or set the session mode", Hint: "[default|plan]"},
	{Name: "plan", Description: "Enter plan mode: read-only until a plan is approved"},
	{Name: "name", Description: "Show or set the session name", Hint: "[name]"},
	{Name: "fork", Description: "Start a new session from before an earlier message", Hint: "[entry]"},
	{Name: "clone", Description: "Start a new session from a copy of this one"},
}

// availableCommands is the commands as ACP lists them.
func availableCommands() []acp.AvailableCommand {
	out := make([]acp.AvailableCommand, len(commands))
	for i, c := range commands {
		out[i] = acp.AvailableCommand{Name: c.Name, Description: c.Description}
		if c.Hint != "" {
			out[i].Input = &acp.AvailableCommandInput{Unstructured: &acp.UnstructuredCommandInput{Hint: c.Hint}}
		}
	}
	return out
}

// cmdTurn is one command's answer: its updates, and the response's _meta.
type cmdTurn struct {
	s    *liveSession
	conn *acp.AgentSideConnection
	out  *updates
	meta map[string]any
}

func (t *cmdTurn) say(text string) { t.out.send(acp.UpdateAgentMessageText(text)) }

// failed tells the user what failed. A command's failure is its answer,
// not a JSON-RPC error: the turn ends normally, as Pi shows the failure
// and goes on.
func (t *cmdTurn) failed(what string, err error) error {
	t.say(what + err.Error())
	return nil
}

type handler func(ctx context.Context, t *cmdTurn, args string) error

func (a *Agent) handlers() map[string]handler {
	return map[string]handler{
		"help":     a.cmdHelp,
		"compact":  a.cmdCompact,
		"usage":    a.cmdUsage,
		"context":  a.cmdContext,
		"session":  a.cmdContext,
		"model":    a.cmdModel,
		"thinking": a.cmdThinking,
		"mode":     a.cmdMode,
		"plan":     func(ctx context.Context, t *cmdTurn, _ string) error { return a.cmdMode(ctx, t, modePlan) },
		"name":     a.cmdName,
		"fork":     a.cmdFork,
		"clone":    a.cmdClone,
	}
}

// runCommand executes a slash command in place of a model turn: nothing
// is written for the prompt itself, and only /compact calls the model. It
// holds the session's turn, so session/cancel stops it.
func (a *Agent) runCommand(ctx context.Context, s *liveSession, conn *acp.AgentSideConnection, name, args string) (acp.PromptResponse, error) {
	turnCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	a.mu.Lock()
	s.cancel = cancel
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		s.cancel = nil
		a.mu.Unlock()
	}()
	t := &cmdTurn{s: s, conn: conn, out: newUpdates(ctx, conn, s.id)}
	h, ok := a.handlers()[name]
	if !ok {
		t.say(fmt.Sprintf("Unknown command /%s. /help lists the commands.", name))
	} else if err := h(turnCtx, t, args); err != nil {
		if turnCtx.Err() != nil {
			return acp.PromptResponse{StopReason: acp.StopReasonCancelled}, nil
		}
		return acp.PromptResponse{}, err
	}
	if t.out.err != nil {
		return acp.PromptResponse{}, t.out.err
	}
	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn, Meta: t.meta}, nil
}

func (*Agent) cmdHelp(_ context.Context, t *cmdTurn, _ string) error {
	var b strings.Builder
	b.WriteString("Commands:\n")
	for _, c := range commands {
		name := "/" + c.Name
		if c.Hint != "" {
			name += " " + c.Hint
		}
		fmt.Fprintf(&b, "%s — %s\n", name, c.Description)
	}
	t.say(b.String())
	return nil
}

// totals are a session's message counts and usage, over every entry, as
// Pi's getSessionStats counts them: compacted history included.
type totals struct {
	messages, user, assistant, toolCalls, toolResults int
	input, output, cacheRead, cacheWrite              int64
	estimated                                         bool
}

func sessionTotals(entries []session.Entry) totals {
	var t totals
	addUsage := func(u *session.Usage) {
		if u == nil {
			return
		}
		t.input += u.Input
		t.output += u.Output
		t.cacheRead += u.CacheRead
		t.cacheWrite += u.CacheWrite
		var x struct {
			Estimated bool `json:"estimated"`
		}
		if len(u.Unknown) > 0 && json.Unmarshal(u.Unknown, &x) == nil && x.Estimated {
			t.estimated = true
		}
	}
	for _, e := range entries {
		if e.Type == session.TypeCompaction {
			addUsage(e.Usage)
		}
		if e.Type != session.TypeMessage || e.Message == nil {
			continue
		}
		t.messages++
		switch e.Message.Role {
		case session.RoleUser:
			t.user++
		case session.RoleToolResult:
			t.toolResults++
		case session.RoleAssistant:
			t.assistant++
			addUsage(e.Message.Usage)
			if blocks, err := e.Message.Blocks(); err == nil {
				for _, b := range blocks {
					if b.Type == session.BlockToolCall {
						t.toolCalls++
					}
				}
			}
		}
	}
	return t
}

// thousands is n with comma separators, as Pi's toLocaleString writes it.
func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func (t totals) tokenLines(b *strings.Builder) {
	fmt.Fprintf(b, "Input: %s\n", thousands(t.input+t.cacheRead+t.cacheWrite))
	if t.cacheRead > 0 || t.cacheWrite > 0 {
		fmt.Fprintf(b, "  Cached: %s\n  Uncached: %s\n", thousands(t.cacheRead), thousands(t.input+t.cacheWrite))
	}
	fmt.Fprintf(b, "Output: %s\n", thousands(t.output))
	fmt.Fprintf(b, "Total: %s\n", thousands(t.input+t.output+t.cacheRead+t.cacheWrite))
}

func (a *Agent) cmdUsage(_ context.Context, t *cmdTurn, _ string) error {
	tot := sessionTotals(t.s.log.Entries())
	var b strings.Builder
	b.WriteString("Session usage\n\n")
	tot.tokenLines(&b)
	b.WriteString("Cost: $0.000 (prices arrive with the model catalog, 0005-PLAN F4)\n")
	if tot.estimated {
		b.WriteString("\nThese counts are estimated: the provider did not report usage.\n")
	}
	provider := a.models().Provider
	if provider == "" {
		provider = "the provider"
	}
	fmt.Fprintf(&b, "Account and rate-limit figures are not reported by %s.\n", provider)
	t.say(b.String())
	return nil
}

// cmdContext is /context and /session: Pi's Session Info, with the mode,
// the model and the context's use, the same figure as the last
// usage_update.
func (a *Agent) cmdContext(_ context.Context, t *cmdTurn, _ string) error {
	s := t.s
	st := replayState(s.log.Entries())
	tot := sessionTotals(s.log.Entries())
	var b strings.Builder
	b.WriteString("Session Info\n\n")
	if st.name != "" {
		fmt.Fprintf(&b, "Name: %s\n", st.name)
	}
	file := "In-memory"
	if pf, ok := a.store.(interface{ PathOf(session.ID) string }); ok {
		if p := pf.PathOf(session.ID(s.id)); p != "" {
			file = p
		}
	}
	model := s.model
	if model == "" {
		model = "none"
	} else if p := a.models().Provider; p != "" {
		model = p + "/" + model
	}
	fmt.Fprintf(&b, "File: %s\nID: %s\nMode: %s\nModel: %s\n\n", file, s.id, a.modeOf(s), model)
	fmt.Fprintf(&b, "Messages\nTotal: %d\nUser: %d\nAssistant: %d\nTools: %d calls, %d results\n\n",
		tot.messages, tot.user, tot.assistant, tot.toolCalls, tot.toolResults)
	b.WriteString("Tokens\n")
	tot.tokenLines(&b)
	fmt.Fprintf(&b, "\nContext: %s tokens (estimated) of an unknown window (the model catalog arrives with 0005-PLAN F4)\n", thousands(int64(a.usedOf(s))))
	t.say(b.String())
	return nil
}

func (a *Agent) cmdModel(ctx context.Context, t *cmdTurn, arg string) error {
	choice := a.models()
	if len(choice.Models) == 0 {
		t.say("No models are offered: no model provider is configured.")
		return nil
	}
	if arg == "" {
		t.say(fmt.Sprintf("Model: %s\nModels: %s", t.s.model, strings.Join(choice.Models, ", ")))
		return nil
	}
	if !slices.Contains(choice.Models, arg) {
		t.say(fmt.Sprintf("Unknown model %q. Available models: %s.", arg, strings.Join(choice.Models, ", ")))
		return nil
	}
	if err := a.setConfigUpdate(ctx, t, configModel, arg); err != nil {
		return err
	}
	t.say("Model: " + arg)
	return nil
}

func (a *Agent) cmdThinking(ctx context.Context, t *cmdTurn, arg string) error {
	if arg == "" {
		t.say(fmt.Sprintf("Thinking level: %s\nLevels: %s", t.s.think, strings.Join(thinkingLevels, ", ")))
		return nil
	}
	level := strings.ToLower(arg)
	if !validThinking(level) {
		t.say(fmt.Sprintf("Unknown thinking level %q. Available levels: %s.", arg, strings.Join(thinkingLevels, ", ")))
		return nil
	}
	if err := a.setConfigUpdate(ctx, t, configThinking, level); err != nil {
		return err
	}
	t.say("Thinking level: " + level)
	return nil
}

// setConfigUpdate is setConfig, reported with config_option_update: a
// command has no response that carries the options.
func (a *Agent) setConfigUpdate(ctx context.Context, t *cmdTurn, id, value string) error {
	opts, err := a.setConfig(ctx, t.s, id, value)
	if err != nil {
		return err
	}
	t.out.send(acp.SessionUpdate{ConfigOptionUpdate: &acp.SessionConfigOptionUpdate{SessionUpdate: "config_option_update", ConfigOptions: opts}})
	return nil
}

func (a *Agent) cmdMode(ctx context.Context, t *cmdTurn, arg string) error {
	current := a.modeOf(t.s)
	switch {
	case arg == "":
		t.say(fmt.Sprintf("Mode: %s\nModes: %s, %s", current, modeDefault, modePlan))
		return nil
	case !validMode(arg):
		t.say(fmt.Sprintf("Unknown mode %q. The modes are %s and %s.", arg, modeDefault, modePlan))
		return nil
	case arg == modePlan && current == modePlan:
		t.say("Already in plan mode.")
		return nil
	}
	if err := a.setMode(ctx, t.s, arg); err != nil {
		return err
	}
	if arg == modePlan {
		t.say("Mode: plan. Files cannot change and commands cannot run until the user approves a plan.")
		return nil
	}
	t.say("Mode: " + arg)
	return nil
}

// cmdName is Pi's /name: show the name, or set it with a session_info
// entry, reported as session_info_update.
func (a *Agent) cmdName(ctx context.Context, t *cmdTurn, arg string) error {
	if arg == "" {
		if name := replayState(t.s.log.Entries()).name; name != "" {
			t.say("Session name: " + name)
		} else {
			t.say("Usage: /name <name>")
		}
		return nil
	}
	name := oneLine(arg)
	if err := t.s.append(ctx, a.now(), session.Entry{Type: session.TypeSessionInfo, Name: new(name)}); err != nil {
		return acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	t.out.send(acp.SessionUpdate{SessionInfoUpdate: &acp.SessionSessionInfoUpdate{SessionUpdate: "session_info_update", Title: new(name)}})
	if name != arg {
		t.say(fmt.Sprintf("Session name was normalized from %q to %q.\n", arg, name))
	}
	t.say("Session name set: " + name)
	return nil
}

// cmdFork is /fork: with no argument, the user messages to fork before;
// with one, a new session holding the path up to that message's parent.
func (a *Agent) cmdFork(ctx context.Context, t *cmdTurn, arg string) error {
	entries := t.s.log.Entries()
	if arg == "" {
		var b strings.Builder
		for _, e := range entries {
			if e.Type == session.TypeMessage && e.Message != nil && e.Message.Role == session.RoleUser {
				if text := e.Message.Text(); text != "" {
					fmt.Fprintf(&b, "  %s  %s\n", e.ID, cutTitle(firstLine(text)))
				}
			}
		}
		if b.Len() == 0 {
			t.say("No messages to fork from yet.")
			return nil
		}
		t.say("Send /fork <entry> to start a new session before one of these messages:\n" + b.String())
		return nil
	}
	var target *session.Entry
	for i := range entries {
		e := &entries[i]
		if e.ID == arg && e.Type == session.TypeMessage && e.Message != nil && e.Message.Role == session.RoleUser {
			target = e
		}
	}
	switch {
	case target == nil:
		t.say(fmt.Sprintf("No user message %q in this session. /fork lists them.", arg))
		return nil
	case target.ParentID == nil:
		t.say("Nothing comes before that message; start a new session instead.")
		return nil
	}
	return a.copyTo(ctx, t, pathTo(entries, *target.ParentID), "Forked")
}

// cmdClone is /clone: a new session holding the whole current path.
func (a *Agent) cmdClone(ctx context.Context, t *cmdTurn, _ string) error {
	entries := t.s.log.Entries()
	if len(entries) == 0 {
		t.say("Nothing to clone yet")
		return nil
	}
	return a.copyTo(ctx, t, session.Path(entries), "Cloned")
}

// copyTo writes path into a new stored session whose parent is this one,
// closes it so its lock is free, and names it: the response's
// _meta.gobble.switchTo lets gobble's CLI resume it, as Pi's terminal
// switches to a fork (owner's decision of 2026-10-06).
func (a *Agent) copyTo(ctx context.Context, t *cmdTurn, path []session.Entry, verb string) error {
	id := session.ID(a.newID())
	h := session.Header{Type: session.TypeHeader, Version: session.Version, ID: id, Timestamp: session.Timestamp(a.now()), Cwd: t.s.cwd,
		ParentSession: string(t.s.id)}
	if pf, ok := a.store.(interface{ PathOf(session.ID) string }); ok {
		h.ParentSession = pf.PathOf(session.ID(t.s.id))
	}
	log, err := a.store.Create(ctx, h)
	if err != nil {
		return acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	for _, e := range path {
		if err := log.Append(ctx, e); err != nil {
			return acp.NewInternalError(map[string]any{keyReason: errors.Join(err, log.Close()).Error()})
		}
	}
	if err := log.Close(); err != nil {
		return acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	t.meta = map[string]any{gobble: map[string]any{"switchTo": string(id)}}
	t.say(fmt.Sprintf("%s to session %s.", verb, id))
	return nil
}

// pathTo is the path from the root to the entry id, in order.
func pathTo(entries []session.Entry, id string) []session.Entry {
	byID := make(map[string]session.Entry, len(entries))
	for _, e := range entries {
		byID[e.ID] = e
	}
	var rev []session.Entry
	for cur, ok := byID[id]; ok; {
		rev = append(rev, cur)
		if cur.ParentID == nil {
			break
		}
		cur, ok = byID[*cur.ParentID]
	}
	out := make([]session.Entry, len(rev))
	for i, e := range rev {
		out[len(rev)-1-i] = e
	}
	return out
}

// startFrames are what a client needs before its first prompt, sent before
// a session response (2026-10-01 amendment): the commands, the mode, and
// the context's estimated use, the system prompt and the context.
func (a *Agent) startFrames(ctx context.Context, conn *acp.AgentSideConnection, s *liveSession) error {
	used := compaction.EstimateText(systemPrompt(s.cwd, a.tools)) + compaction.EstimateEntries(session.Path(s.log.Entries()))
	a.setUsed(s, used)
	out := newUpdates(ctx, conn, s.id)
	out.send(acp.SessionUpdate{AvailableCommandsUpdate: &acp.SessionAvailableCommandsUpdate{SessionUpdate: "available_commands_update", AvailableCommands: availableCommands()}})
	out.send(modeUpdate(a.modeOf(s)))
	out.send(usedUpdate(used))
	return out.err
}

func (a *Agent) setUsed(s *liveSession, used int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s.used = used
}

func (a *Agent) usedOf(s *liveSession) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return s.used
}
