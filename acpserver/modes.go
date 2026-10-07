package acpserver

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
)

// The session modes (0002-PLAN Phase 6). 0005-PLAN F6 adds accept-edits
// and bypass.
const (
	modeDefault = "default"
	modePlan    = "plan"
)

// exitPlanTool is the tool that asks the user to leave plan mode.
const exitPlanTool = "exit_plan_mode"

const (
	startImplementing acp.PermissionOptionId = "start_implementing"
	keepPlanning      acp.PermissionOptionId = "keep_planning"
)

var modes = []acp.SessionMode{
	{Id: modeDefault, Name: "Default", Description: new("Ask before changing files or running commands.")},
	{Id: modePlan, Name: "Plan", Description: new("Read-only: explore and plan, then leave plan mode with an approved plan.")},
}

func validMode(id string) bool {
	for _, m := range modes {
		if string(m.Id) == id {
			return true
		}
	}
	return false
}

// modeState is the modes a session response advertises.
func modeState(current string) *acp.SessionModeState {
	return &acp.SessionModeState{AvailableModes: modes, CurrentModeId: acp.SessionModeId(current)}
}

func modeUpdate(current string) acp.SessionUpdate {
	return acp.SessionUpdate{CurrentModeUpdate: &acp.SessionCurrentModeUpdate{SessionUpdate: "current_mode_update", CurrentModeId: acp.SessionModeId(current)}}
}

// modeOf is a session's mode, read under the agent's lock: set_mode may
// arrive while a turn runs.
func (a *Agent) modeOf(s *liveSession) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return s.mode
}

// setMode changes a session's mode and confirms it with
// current_mode_update. The mode is not persisted: Pi has no mode entry.
func (a *Agent) setMode(ctx context.Context, s *liveSession, id string) error {
	if !validMode(id) {
		return invalidParams(s.id, fmt.Sprintf("unknown mode %q (the modes are default and plan)", id))
	}
	a.mu.Lock()
	s.mode = id
	conn := a.conn
	a.mu.Unlock()
	out := newUpdates(ctx, conn, s.id)
	out.send(modeUpdate(id))
	return out.err
}

// SetSessionMode switches between default and plan.
func (a *Agent) SetSessionMode(ctx context.Context, p acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	a.mu.Lock()
	s, ok := a.sessions[p.SessionId]
	a.mu.Unlock()
	if !ok {
		return acp.SetSessionModeResponse{}, invalidParams(p.SessionId, "unknown session")
	}
	if err := a.setMode(ctx, s, string(p.ModeId)); err != nil {
		return acp.SetSessionModeResponse{}, err
	}
	return acp.SetSessionModeResponse{}, nil
}

// planPolicy refuses every call that asks for approval while the session
// is in plan mode, without asking: plan mode is read-only. Otherwise the
// inner policy decides.
type planPolicy struct {
	a     *Agent
	s     *liveSession
	inner permission.Policy
}

func (p planPolicy) Decide(ctx context.Context, r permission.Rule) (permission.Decision, error) {
	if p.a.modeOf(p.s) == modePlan {
		return permission.Decision{Refusal: fmt.Sprintf(
			"Plan mode is read-only, so %s did not run. Call %s with the plan to ask the user to leave plan mode.", r.Tool, exitPlanTool)}, nil
	}
	return p.inner.Decide(ctx, r)
}

type exitPlanInput struct {
	Plan string `json:"plan" jsonschema:"The plan to carry out, in Markdown: the steps, in order."`
}

// exitPlanSpec is exit_plan_mode's spec. It asks for itself, so it is
// marked read-only and the agent does not ask again.
var exitPlanSpec = tool.New(exitPlanTool,
	"Ask the user to approve the plan and leave plan mode. Call it once the plan is complete; until the user approves, files cannot be changed and commands cannot run.",
	func(context.Context, exitPlanInput, tool.Env) (string, error) { return "", nil },
	tool.WithKind(tool.KindThink), tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true})).Spec()

// exitPlan is the tool plan mode offers. It publishes the plan as an ACP
// plan update and asks to leave plan mode through
// session/request_permission for its own call, carrying the plan
// (0005-MADR); on allow, the session returns to the default mode.
type exitPlan struct {
	a    *Agent
	conn *acp.AgentSideConnection
	s    *liveSession
}

var (
	_ tool.Tool      = exitPlan{}
	_ tool.Describer = exitPlan{}
)

func (exitPlan) Spec() tool.Spec { return exitPlanSpec }

func (exitPlan) Describe(tool.Call, tool.Env) string { return "leave plan mode" }

func (e exitPlan) Run(ctx context.Context, call tool.Call, _ tool.Env) (tool.Result, error) {
	var in exitPlanInput
	if err := json.Unmarshal(call.Args, &in); err != nil {
		return tool.ErrorResult(fmt.Sprintf("%s: invalid arguments: %v", exitPlanTool, err)), nil
	}
	if strings.TrimSpace(in.Plan) == "" {
		return tool.ErrorResult(exitPlanTool + " needs the plan, as {\"plan\": \"…\"}"), nil
	}
	out := newUpdates(ctx, e.conn, e.s.id)
	out.send(acp.UpdatePlan(planEntries(in.Plan)...))
	if out.err != nil {
		return tool.Result{}, out.err
	}
	if e.conn == nil {
		return tool.ErrorResult("there is no client to ask"), nil
	}
	resp, err := e.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
		SessionId: e.s.id,
		ToolCall: acp.ToolCallUpdate{ToolCallId: acp.ToolCallId(call.ID), Title: new("Leave plan mode and carry out the plan"),
			Content: []acp.ToolCallContent{acp.ToolContent(acp.TextBlock(cutBytes(in.Plan, maxSummary)))}},
		Options: []acp.PermissionOption{
			{OptionId: startImplementing, Name: "Start implementing", Kind: acp.PermissionOptionKindAllowOnce},
			{OptionId: keepPlanning, Name: "Keep planning", Kind: acp.PermissionOptionKindRejectOnce},
		},
	})
	if ctx.Err() != nil {
		return tool.Result{}, context.Cause(ctx)
	}
	if err != nil {
		return tool.ErrorResult(fmt.Sprintf("the permission request failed: %v", err)), nil
	}
	if sel := resp.Outcome.Selected; sel == nil || sel.OptionId != startImplementing {
		return tool.TextResult("The user wants to keep planning. Plan mode stays on: refine the plan and call " + exitPlanTool + " again."), nil
	}
	if err := e.a.setMode(ctx, e.s, modeDefault); err != nil {
		return tool.Result{}, err
	}
	return tool.TextResult("The user approved the plan. Plan mode is off: carry out the plan."), nil
}

// planEntries are a plan's lines as ACP plan entries, list markers
// removed, each pending and of medium priority.
func planEntries(plan string) []acp.PlanEntry {
	var out []acp.PlanEntry
	for line := range strings.Lines(plan) {
		line = strings.TrimSpace(line)
		for _, m := range []string{"- [ ] ", "- [x] ", "* [ ] ", "- ", "* ", "+ "} {
			line = strings.TrimPrefix(line, m)
		}
		if i := strings.Index(line, ". "); i > 0 && i <= 3 && isDigits(line[:i]) {
			line = line[i+2:]
		}
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, acp.PlanEntry{Content: line, Priority: acp.PlanEntryPriorityMedium, Status: acp.PlanEntryStatusPending})
		}
	}
	return out
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
