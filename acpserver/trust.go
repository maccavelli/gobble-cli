package acpserver

import (
	"cmp"
	"context"
	"log/slog"
	"path/filepath"
	"slices"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/personality"
	"github.com/maccavelli/gobble-cli/tool"
)

// ProjectOverrides are a trusted project's own overrides for one session,
// each checked as the user's are (0005-PLAN F6-1 decision 8). Zero is
// none.
type ProjectOverrides struct {
	// Personalities are Options.Personalities with the project's messages
	// over them; nil is Options.Personalities.
	Personalities *personality.Set
	// ToolsSection and RulesSection replace PromptOptions'; "" keeps them.
	ToolsSection, RulesSection string
	// CompactionPrompts replace Options.CompactionPrompts', each empty one
	// keeping the user's.
	CompactionPrompts compaction.Prompts
	// Descriptions are built-in tools' descriptions, rendered, by name.
	Descriptions map[string]string
}

// projectOf is a session's overrides: *p, or none for nil.
func projectOf(p *ProjectOverrides) ProjectOverrides {
	if p == nil {
		return ProjectOverrides{}
	}
	return *p
}

// TrustOptions are /trust's store (0005-PLAN F6-1 decision 7).
type TrustOptions struct {
	// Save records trusted for dir, returning the path saved, as
	// permission.TrustStore.Save does. Nil saves nothing.
	Save func(ctx context.Context, dir string, trusted bool) (string, error)
	// Home is the home directory /trust never offers; "" is the user's.
	Home string
	// Quiet sends no notice for an untrusted project: the client tells
	// the user itself, as gobble's own modes do (decision 9).
	Quiet bool
}

// TrustCallID is the tool-call id of /trust's question, which no model
// call has.
const TrustCallID = "gobble-trust"

// /trust's answers.
const (
	trustYes    = "trust"
	trustParent = "trust-parent"
	trustNo     = "untrust"
)

// The placeholders /trust's replies fill.
const (
	argRoot     = "root"
	argDecision = "decision"
	argPath     = "path"
	argState    = "state"
)

// noteTrust tells the client, once a session, that its project is not
// trusted: a line before the first reply, sent as a command's reply is, so
// neither the session file nor the model has it (decision 9). A session
// whose settings decided nothing is told nothing.
func (a *Agent) noteTrust(ctx context.Context, conn *acp.AgentSideConnection, s *liveSession) {
	a.mu.Lock()
	note := !s.trust.Trusted && s.trust.Source != 0 && !s.trustNoted && !a.trust.Quiet
	s.trustNoted = true
	v := s.voice
	a.mu.Unlock()
	if note {
		newUpdates(ctx, conn, s.id).send(acp.UpdateAgentMessageText(v.Text(personality.TrustUntrusted, nil) + "\n\n"))
	}
}

// cmdTrust is /trust: the session's decision and where it came from, then
// the question, whose answer is saved for the project's next session
// (decision 7): a session's settings, prompt and tools are fixed when it
// starts.
func (a *Agent) cmdTrust(ctx context.Context, t *cmdTurn, _ string) error {
	root := cmp.Or(t.s.root, t.s.cwd)
	t.say(t.text(personality.TrustCurrent, personality.Args{argRoot: root, argDecision: t.s.trust.String()}) + "\n")
	if a.trust.Save == nil || t.conn == nil {
		t.say(t.text(personality.TrustUnavailable, nil))
		return nil
	}
	opts := []acp.PermissionOption{{OptionId: trustYes, Name: "Trust", Kind: acp.PermissionOptionKindAllowAlways}}
	parent := filepath.Dir(root)
	if resolved, err := filepath.EvalSymlinks(parent); err == nil && !permission.TooBroad(resolved, a.trust.Home) {
		opts = append(opts, acp.PermissionOption{OptionId: trustParent, Name: "Trust the parent folder (" + parent + ")", Kind: acp.PermissionOptionKindAllowAlways})
	}
	opts = append(opts, acp.PermissionOption{OptionId: trustNo, Name: "Do not trust", Kind: acp.PermissionOptionKindRejectAlways})
	resp, err := t.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
		SessionId: t.s.id,
		ToolCall:  acp.ToolCallUpdate{ToolCallId: TrustCallID, Title: new("Trust " + root)},
		Options:   opts,
	})
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if err != nil {
		a.logger.WarnContext(ctx, "trust: the question was not answered", slog.String("error", err.Error()))
	}
	var chosen string
	if err == nil && resp.Outcome.Selected != nil {
		chosen = string(resp.Outcome.Selected.OptionId)
	}
	dir, trusted := root, true
	switch chosen {
	case trustYes:
	case trustParent:
		dir = parent
	case trustNo:
		trusted = false
	default:
		t.say(t.text(personality.TrustNotSaved, nil))
		return nil
	}
	at, err := a.trust.Save(ctx, dir, trusted)
	if err != nil {
		t.say(t.text(personality.TrustFailed, personality.Args{argError: err.Error()}))
		return nil //nolint:nilerr // the failed save is the command's answer (above)
	}
	state := "not trusted"
	if trusted {
		state = "trusted"
	}
	t.say(t.text(personality.TrustSaved, personality.Args{argPath: at, argState: state}))
	return nil
}

// toolsOf are the agent's tools for s, a trusted project's descriptions in
// place of their own (decision 8).
func (a *Agent) toolsOf(s *liveSession) []tool.Tool {
	if len(s.project.Descriptions) == 0 {
		return slices.Clip(a.tools)
	}
	out := slices.Clone(a.tools)
	for i, t := range out {
		if d, ok := s.project.Descriptions[t.Spec().Name]; ok {
			out[i] = tool.Redescribed(t, d)
		}
	}
	return slices.Clip(out)
}

// compactionPromptsOf are s's compaction prompts: its trusted project's,
// each over the user's (decision 8).
func (a *Agent) compactionPromptsOf(s *liveSession) compaction.Prompts {
	p, u := s.project.CompactionPrompts, a.compactionPrompts
	return compaction.Prompts{System: cmp.Or(p.System, u.System), Summary: cmp.Or(p.Summary, u.Summary),
		Update: cmp.Or(p.Update, u.Update), TurnPrefix: cmp.Or(p.TurnPrefix, u.TurnPrefix)}
}
