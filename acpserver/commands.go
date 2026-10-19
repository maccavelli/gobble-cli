package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/command"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/personality"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/session"
)

// commands are the slash commands Prompt executes: 0002-PLAN Phase 6 step
// 6's frozen set, then the ones 0005-MADR's command table adds as their
// phases land (/todos, 0005-PLAN F1d-1). Every one is advertised, and
// only these are: a test
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
	{Name: "todos", Description: "Show the todo list"},
	{Name: cmdPersonality, Description: "Show or set the personality", Hint: "[boring|dry|snarky|bombastic]"},
	{Name: "trust", Description: "Show whether this project is trusted, and save a decision"},
	{Name: "permissions", Description: "List, remove or add permission rules", Hint: "[remove N | allow|ask|deny RULE [session|project|global]]"},
}

// availableCommands is the commands as ACP lists them, then each template
// whose name is a command name, with its description and its
// argument-hint (0005-PLAN F3-3). A template a built-in shadows is not
// among templates (Agent.templates).
func availableCommands(templates []sysprompt.Template) []acp.AvailableCommand {
	all := slices.Clone(commands)
	for _, t := range templates {
		if command.ValidName(t.Name) {
			all = append(all, command.Command{Name: t.Name, Description: t.Description, Hint: t.ArgumentHint})
		}
	}
	out := make([]acp.AvailableCommand, len(all))
	for i, c := range all {
		out[i] = acp.AvailableCommand{Name: c.Name, Description: c.Description}
		if c.Hint != "" {
			out[i].Input = &acp.AvailableCommandInput{Unstructured: &acp.UnstructuredCommandInput{Hint: c.Hint}}
		}
	}
	return out
}

// cmdTurn is one command's answer: its updates, and the response's _meta.
// voice is the session's as the command runs, and after a switch it makes
// (0022-PLAN P3).
type cmdTurn struct {
	s     *liveSession
	conn  *acp.AgentSideConnection
	out   *updates
	meta  map[string]any
	voice *personality.Voice
}

func (t *cmdTurn) say(text string) { t.out.send(acp.UpdateAgentMessageText(text)) }

// The placeholders the replies fill, by the boring catalog's names
// (0022-PLAN P1).
const (
	argModel    = "model"
	argModels   = "models"
	argLevel    = "level"
	argLevels   = "levels"
	argMode     = "mode"
	argModes    = "modes"
	argName     = "name"
	argFrom     = "from"
	argTo       = "to"
	argID       = "id"
	argError    = "error"
	argTokens   = "tokens"
	argProvider = "provider"
	argNames    = "names"
)

// text is key's message in the session's voice (0022-PLAN P1).
func (t *cmdTurn) text(key personality.Key, args personality.Args) string {
	return t.voice.Text(key, args)
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
		"todos":    a.cmdTodos,
		// /humor is hidden: not in commands (0022-PLAN P3).
		cmdPersonality: a.cmdPersonality,
		cmdHumor:       a.cmdHumor,
		"trust":        a.cmdTrust,
		"permissions":  a.cmdPermissions,
	}
}

// runCommand executes a slash command in place of a model turn: nothing
// is written for the prompt itself, and only /compact calls the model. It
// runs in the turn Prompt claimed, so session/cancel stops it through
// turnCtx.
func (a *Agent) runCommand(turnCtx context.Context, s *liveSession, conn *acp.AgentSideConnection, name, args string) (acp.PromptResponse, error) {
	t := &cmdTurn{s: s, conn: conn, out: newUpdates(turnCtx, conn, s.id), voice: a.voice(s)}
	h, ok := a.handlers()[name]
	if !ok {
		t.say(t.text(personality.CommandUnknown, personality.Args{argName: name}))
	} else if err := h(turnCtx, t, args); err != nil {
		if turnCtx.Err() != nil {
			return cancelled(s)
		}
		return acp.PromptResponse{}, err
	}
	if t.out.err != nil {
		return acp.PromptResponse{}, t.out.err
	}
	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn, Meta: t.meta}, nil
}

// cancelled is a cancelled command turn's answer: not an error, with the
// session's queues emptied into _meta (clearedMeta).
func cancelled(s *liveSession) (acp.PromptResponse, error) {
	return acp.PromptResponse{StopReason: acp.StopReasonCancelled, Meta: clearedMeta(s)}, nil
}

func (*Agent) cmdHelp(_ context.Context, t *cmdTurn, _ string) error {
	var b strings.Builder
	b.WriteString(t.text(personality.HelpHeading, nil) + "\n")
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
// Pi's getSessionStats counts them (agent-session.ts:4093-4116): compacted
// history included, with the usage of usage entries, compactions, branch
// summaries, tool results and assistant messages.
type totals struct {
	messages, user, assistant, toolCalls, toolResults int
	input, output, cacheRead, cacheWrite              int64
	estimated                                         bool
	// cost is the sum of each usage's cost, in USD (0005-PLAN F4-2a).
	cost float64
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
		t.cost += u.Cost.Total
		var x struct {
			Estimated bool `json:"estimated"`
		}
		if len(u.Unknown) > 0 && json.Unmarshal(u.Unknown, &x) == nil && x.Estimated {
			t.estimated = true
		}
	}
	for _, e := range entries {
		switch e.Type {
		case session.TypeUsage, session.TypeCompaction, session.TypeBranchSummary:
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
			addUsage(e.Message.Usage)
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
	t.say(a.usageText(t.s, t.voice, t.s.log.Entries()))
	return nil
}

// usageText is /usage's answer, in voice, and _gobble/usage's text, in
// boring's: the cost is the session's when its model is priced (0005-PLAN
// F4-2a).
func (a *Agent) usageText(s *liveSession, voice *personality.Voice, entries []session.Entry) string {
	tot := sessionTotals(entries)
	var b strings.Builder
	b.WriteString(voice.Text(personality.UsageHeading, nil) + "\n\n")
	tot.tokenLines(&b)
	model, _ := a.choices(s)
	if info := a.modelOf(model); info.Priced {
		fmt.Fprintf(&b, "Cost: $%.3f\n", tot.cost)
	} else {
		name := "no model"
		if model != "" {
			name = a.qualified(model)
		}
		b.WriteString(voice.Text(personality.UsageCostUnknown, personality.Args{argModel: name}) + "\n")
	}
	if tot.estimated {
		b.WriteString("\n" + voice.Text(personality.UsageEstimated, nil) + "\n")
	}
	provider := a.models().Provider
	if provider == "" {
		provider = "the provider"
	}
	b.WriteString(voice.Text(personality.UsageNoAccount, personality.Args{argProvider: provider}) + "\n")
	return b.String()
}

// cmdContext is /context and /session: Pi's Session Info, with the mode,
// the model and the context's use, the same figure as the last
// usage_update.
func (a *Agent) cmdContext(_ context.Context, t *cmdTurn, _ string) error {
	s := t.s
	st := s.state()
	tot := sessionTotals(s.log.Entries())
	var b strings.Builder
	b.WriteString(t.text(personality.SessionHeading, nil) + "\n\n")
	if st.name != "" {
		fmt.Fprintf(&b, "Name: %s\n", st.name)
	}
	file := "In-memory"
	if pf, ok := a.store.(interface{ PathOf(session.ID) string }); ok {
		if p := pf.PathOf(session.ID(s.id)); p != "" {
			file = p
		}
	}
	model, _ := a.choices(s)
	shown := "none"
	if model != "" {
		shown = a.qualified(model)
	}
	fmt.Fprintf(&b, "File: %s\nID: %s\nMode: %s\nModel: %s\n\n", file, s.id, a.modeOf(s), shown)
	fmt.Fprintf(&b, "Messages\nTotal: %d\nUser: %d\nAssistant: %d\nTools: %d calls, %d results\n\n",
		tot.messages, tot.user, tot.assistant, tot.toolCalls, tot.toolResults)
	b.WriteString("Tokens\n")
	tot.tokenLines(&b)
	used := a.usedOf(s)
	if window := a.modelOf(model).ContextWindow; window > 0 {
		fmt.Fprintf(&b, "\nContext: %s of %s tokens (%.1f%%)\n", thousands(int64(used)), thousands(int64(window)), percentOf(used, window))
	} else {
		fmt.Fprintf(&b, "\nContext: %s tokens (estimated) of an unknown window\n", thousands(int64(used)))
	}
	t.say(b.String())
	return nil
}

// qualified is model with the session provider's id before it, as
// provider/model, or model alone when there is no provider.
func (a *Agent) qualified(model string) string {
	if p := a.models().Provider; p != "" {
		return p + "/" + model
	}
	return model
}

// percentOf is used as a percentage of window.
func percentOf(used, window int) float64 { return float64(used) / float64(window) * 100 }

func (a *Agent) cmdModel(ctx context.Context, t *cmdTurn, arg string) error {
	choice := a.models()
	if len(choice.Models) == 0 {
		t.say(t.text(personality.ModelNone, nil))
		return nil
	}
	if arg == "" {
		model, _ := a.choices(t.s)
		t.say(t.text(personality.ModelCurrent, personality.Args{argModel: model}) + "\n" +
			t.text(personality.ModelList, personality.Args{argModels: strings.Join(choice.Models, ", ")}))
		return nil
	}
	if !slices.Contains(choice.Models, arg) {
		t.say(t.text(personality.ModelUnknown, personality.Args{argModel: strconv.Quote(arg), argModels: strings.Join(choice.Models, ", ")}))
		return nil
	}
	if err := a.setConfigUpdate(ctx, t, configModel, arg); err != nil {
		return err
	}
	t.say(t.text(personality.ModelSet, personality.Args{argModel: arg}))
	return nil
}

// cmdThinking is /thinking: the session model's levels, as thought_level
// offers them (0005-PLAN F4-2a).
func (a *Agent) cmdThinking(ctx context.Context, t *cmdTurn, arg string) error {
	model, think := a.choices(t.s)
	levels := a.levelsOf(model)
	if arg == "" {
		t.say(t.text(personality.ThinkingCurrent, personality.Args{argLevel: catalog.Clamp(levels, think)}) + "\n" +
			t.text(personality.ThinkingList, personality.Args{argLevels: strings.Join(levels, ", ")}))
		return nil
	}
	level := strings.ToLower(arg)
	if !slices.Contains(levels, level) {
		t.say(t.text(personality.ThinkingUnknown, personality.Args{argLevel: strconv.Quote(arg), argLevels: strings.Join(levels, ", ")}))
		return nil
	}
	if err := a.setConfigUpdate(ctx, t, configThinking, level); err != nil {
		return err
	}
	t.say(t.text(personality.ThinkingSet, personality.Args{argLevel: level}))
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
		t.say(t.text(personality.ModeCurrent, personality.Args{argMode: current}) + "\n" +
			t.text(personality.ModeList, personality.Args{argModes: modeNames(t.s.allowBypass, ", ")}))
		return nil
	case !validMode(arg, t.s.allowBypass):
		t.say(t.text(personality.ModeUnknown, personality.Args{argMode: strconv.Quote(arg), argModes: modeNames(t.s.allowBypass, ", ")}))
		return nil
	case arg == modePlan && current == modePlan:
		t.say(t.text(personality.ModeAlreadyPlan, nil))
		return nil
	}
	if err := a.setMode(ctx, t.s, arg); err != nil {
		return err
	}
	if arg == modePlan {
		t.say(t.text(personality.ModePlanOn, nil))
		return nil
	}
	t.say(t.text(personality.ModeSet, personality.Args{argMode: arg}))
	return nil
}

// cmdName is Pi's /name: show the name, or set it with a session_info
// entry, reported as session_info_update.
func (a *Agent) cmdName(ctx context.Context, t *cmdTurn, arg string) error {
	if arg == "" {
		if name := t.s.state().name; name != "" {
			t.say(t.text(personality.NameCurrent, personality.Args{argName: name}))
		} else {
			t.say(t.text(personality.NameUsage, nil))
		}
		return nil
	}
	name, err := a.setName(ctx, t.s, t.out, arg)
	if err != nil {
		return err
	}
	if name != arg {
		t.say(t.text(personality.NameNormalized, personality.Args{argFrom: strconv.Quote(arg), argTo: strconv.Quote(name)}) + "\n")
	}
	t.say(t.text(personality.NameSet, personality.Args{argName: name}))
	return nil
}

// setName is /name's and _gobble/set_session_name's one path: the name on
// one line, written as session_info, then reported as session_info_update.
func (a *Agent) setName(ctx context.Context, s *liveSession, out *updates, arg string) (string, error) {
	s.nameMu.Lock()
	defer s.nameMu.Unlock()
	return a.setNameLocked(ctx, s, out, arg)
}

// setNameLocked is setName with s.nameMu held, for the title, which
// checks the name first (0005-PLAN F5-5 decision 7).
func (a *Agent) setNameLocked(ctx context.Context, s *liveSession, out *updates, arg string) (string, error) {
	name := oneLine(arg)
	if err := s.append(ctx, a.now(), session.Entry{Type: session.TypeSessionInfo, Name: new(name)}); err != nil {
		return "", appendFailed(err)
	}
	out.send(acp.SessionUpdate{SessionInfoUpdate: &acp.SessionSessionInfoUpdate{SessionUpdate: "session_info_update", Title: acp.PatchValue(name)}})
	return name, out.err
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
			t.say(t.text(personality.ForkNone, nil))
			return nil
		}
		t.say(t.text(personality.ForkList, nil) + "\n" + b.String())
		return nil
	}
	target, refusal := forkTarget(entries, arg)
	switch {
	case refusal == nothingBefore:
		t.say(t.text(personality.ForkNothingBefore, nil))
		return nil
	case refusal != "":
		t.say(t.text(personality.ForkNoMessage, personality.Args{argID: strconv.Quote(arg)}))
		return nil
	}
	return a.copyTo(ctx, t, t.s.tree.Branched(*target.ParentID), personality.ForkDone)
}

// nothingBefore refuses a fork before a session's first message: files are
// created lazily, so an empty session cannot be switched to.
const nothingBefore = "Nothing comes before that message; start a new session instead."

// forkTarget is the user message /fork <entry> and _gobble/fork fork
// before, or why there is none.
func forkTarget(entries []session.Entry, id string) (*session.Entry, string) {
	var target *session.Entry
	for i := range entries {
		e := &entries[i]
		if e.ID == id && e.Type == session.TypeMessage && e.Message != nil && e.Message.Role == session.RoleUser {
			target = e
		}
	}
	// Nothing comes before it when the path to its parent holds no user or
	// assistant message: the copy would have no file to switch to. Entries
	// such as a prompt baseline are not a conversation (0005-PLAN F3-1
	// deviation 3).
	switch {
	case target == nil:
		return nil, fmt.Sprintf("No user message %q in this session.", id)
	case target.ParentID == nil, !session.HasConversation(session.NewTree(entries).Branched(*target.ParentID)):
		return nil, nothingBefore
	}
	return target, ""
}

// cmdTodos is /todos: the active path's last todo list.
func (a *Agent) cmdTodos(_ context.Context, t *cmdTurn, _ string) error {
	t.say(todosText(t.s.state().todos, t.text(personality.TodosNone, nil)))
	return nil
}

// cmdClone is /clone: a new session holding the whole current path, as
// Pi's createBranchedSession writes it (session.Tree.Branched).
func (a *Agent) cmdClone(ctx context.Context, t *cmdTurn, _ string) error {
	leaf := t.s.tree.Leaf()
	if leaf == "" {
		t.say(t.text(personality.CloneNone, nil))
		return nil
	}
	return a.copyTo(ctx, t, t.s.tree.Branched(leaf), personality.CloneDone)
}

// copyTo copies path into a new session and names it: the response's
// _meta.gobble.switchTo lets gobble's CLI resume it, as Pi's terminal
// switches to a fork (owner's decision of 2026-10-06).
func (a *Agent) copyTo(ctx context.Context, t *cmdTurn, path []session.Entry, done personality.Key) error {
	id, err := a.copySession(ctx, t.s, path)
	if err != nil {
		return err
	}
	t.meta = map[string]any{gobble: map[string]any{"switchTo": string(id)}}
	t.say(t.text(done, personality.Args{argID: string(id)}))
	return nil
}

// copySession writes path into a new stored session whose parent is s, and
// closes it so its lock is free. It is the one path of /fork, /clone,
// _gobble/fork and _gobble/clone.
func (a *Agent) copySession(ctx context.Context, s *liveSession, path []session.Entry) (session.ID, error) {
	id := session.ID(a.newID())
	h := session.Header{Type: session.TypeHeader, Version: session.Version, ID: id, Timestamp: session.Timestamp(a.now()), Cwd: s.cwd,
		ParentSession: string(s.id)}
	if pf, ok := a.store.(interface{ PathOf(session.ID) string }); ok {
		h.ParentSession = pf.PathOf(session.ID(s.id))
	}
	log, err := a.store.Create(ctx, h)
	if err != nil {
		return "", acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	for _, e := range path {
		if err := log.Append(ctx, e); err != nil {
			return "", acp.NewInternalError(map[string]any{keyReason: errors.Join(err, log.Close()).Error()})
		}
	}
	if err := log.Close(); err != nil {
		return "", acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	return id, nil
}

// startFrames are what a client needs before its first prompt, sent before
// a session response (2026-10-01 amendment): the commands, the mode, the
// context's estimated use, the system prompt and the context, and the
// personality (0022-MADR D6). The
// session's templates are found here, once (0005-PLAN F3-3).
func (a *Agent) startFrames(ctx context.Context, conn *acp.AgentSideConnection, s *liveSession) error {
	templates := a.templates(s)
	a.mu.Lock()
	s.templates = templates
	a.mu.Unlock()
	used := compaction.EstimateText(a.systemText(s)) + compaction.EstimateEntries(s.tree.Path())
	a.setUsed(s, used)
	out := newUpdates(ctx, conn, s.id)
	out.send(acp.SessionUpdate{AvailableCommandsUpdate: &acp.SessionAvailableCommandsUpdate{SessionUpdate: "available_commands_update", AvailableCommands: availableCommands(templates)}})
	out.send(modeUpdate(a.modeOf(s)))
	out.send(a.contextUpdate(s, used))
	out.notify(methodPersonality, a.personalityNotice(s))
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
