package acpserver

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/personality"
	"github.com/maccavelli/gobble-cli/tool"
)

// keyLimit is the _meta.gobble key of a question one of gobble's limits
// asked: a client offers no "always" for it, and --yes does not approve it
// (0005-PLAN F6-2 decisions 5 and 8).
const keyLimit = "limit"

// keyRepeated is the _meta.gobble key of a repeated call's question, the
// run's length: --yes does not approve it (0005-PLAN F6-2a's deviation 2).
const keyRepeated = "repeated"

// keyOnceOnly is the _meta.gobble key of a question that offers "once"
// alone because no rule can match the call alone, with why (0005-PLAN
// F6-2b decision A); keyRule is the question's deciding rule, {rule,
// from} (decision G); keyFeedback is an answer's reason for a reject
// (decision F).
const (
	keyOnceOnly = "onceOnly"
	keyRule     = "rule"
	keyFrom     = "from"
	keyFeedback = "feedback"
)

// AllowOnceID is the option id of a tool call's "allow once", which only
// the questions about a tool call offer: print mode's --yes chooses it
// (0005-PLAN F6-2 decision 8).
const AllowOnceID = string(allowOnce)

// RejectOnceID is the option id of a tool call's "reject once", which a
// client answers with a reason in _meta.gobble.feedback (0005-PLAN F6-2b
// decision F).
const RejectOnceID = string(rejectOnce)

// textOnceOnly is why a question offers "once" alone (decision A).
const textOnceOnly = "no rule can match this call alone: its command or a path holds * ? [ or {"

// maxFeedback bounds an answer's reason, in bytes (decision F).
const maxFeedback = 2048

// PermissionOptions are where the questions' "always" answers and
// /permissions save rules (0005-PLAN F6-2b decision E).
type PermissionOptions struct {
	// Store is the project store, permissions.json; a zero Path keeps none.
	Store permission.RuleStore
	// AddGlobal appends rule to the global settings file, and RemoveGlobal
	// removes the rule at its source there; each returns the file's rules
	// after, each with its file and line. Nil saves nothing there.
	AddGlobal    func(rule permission.Spec) ([]permission.Spec, error)
	RemoveGlobal func(rule permission.Spec) ([]permission.Spec, error)
}

// The session engine's layers, in the order they are joined (decision E;
// F6-3 decision 10's defaults first).
const (
	layerDefault = iota
	layerGlobal
	layerProject
	layerSet
	layerStore
	layerSession
	layerCount
)

// layerSetName is --set's layer, as internal/config names it.
const layerSetName = "--set"

// defaultRules are the rules before every file's: gobble update asks,
// from a shell, though a broader allow follows (0005-MADR's self-update
// row; 0005-PLAN F6-3 decision 10).
var defaultRules = []permission.Spec{
	{Effect: permission.ModeAsk, Tool: "bash", Pattern: "gobble update*", Source: permission.RuleSource{Layer: permission.LayerDefault}},
	{Effect: permission.ModeAsk, Tool: "powershell", Pattern: "gobble update*", Source: permission.RuleSource{Layer: permission.LayerDefault}},
}

// engineOf is a session's rule engine: the default rules, then its
// settings' rules split by their source into the global, project and --set
// layers, then stored, the project store's, and the session's, none yet;
// relative to its project root, or to cwd when it has none, with gobble's
// protected paths placed by its home and config directories (0005-PLAN
// F6-2 decisions 3 to 5).
func engineOf(ss SessionSettings, cwd string, stored []permission.Spec) permission.Engine {
	layers := make([][]permission.Spec, layerCount)
	layers[layerDefault] = defaultRules
	for _, r := range ss.Rules {
		i := layerGlobal
		switch r.Source.Layer {
		case permission.LayerProject:
			i = layerProject
		case layerSetName:
			i = layerSet
		}
		layers[i] = append(layers[i], r)
	}
	layers[layerStore] = stored
	return permission.Engine{Layers: layers, Root: cmp.Or(ss.ProjectRoot, cwd), Home: ss.Home, ConfigDir: ss.ConfigDir}
}

// storedRules are the project store's rules for a session at root, or
// none, with a log line for a store that does not read.
func (a *Agent) storedRules(root string) []permission.Spec {
	if a.perms.Store.Path == "" {
		return nil
	}
	rules, err := a.perms.Store.Rules(root)
	if err != nil {
		a.logger.Warn("permissions: the project store", slog.String("error", err.Error()))
	}
	return rules
}

// rulesOf is s's rule engine as it is now. Its layers are replaced, never
// changed in place, so a copy stays whole.
func (a *Agent) rulesOf(s *liveSession) permission.Engine {
	a.mu.Lock()
	defer a.mu.Unlock()
	return s.rules
}

// setLayer replaces s's layer i with rules.
func (a *Agent) setLayer(s *liveSession, i int, rules []permission.Spec) {
	a.mu.Lock()
	defer a.mu.Unlock()
	layers := slices.Clone(s.rules.Layers)
	layers[i] = rules
	s.rules.Layers = layers
}

// addToLayer appends rules to s's layer i, each with source.
func (a *Agent) addToLayer(s *liveSession, i int, source permission.RuleSource, rules []permission.Spec) {
	a.mu.Lock()
	defer a.mu.Unlock()
	add := make([]permission.Spec, len(rules))
	for j, r := range rules {
		r.Source = source
		add[j] = r
	}
	layers := slices.Clone(s.rules.Layers)
	layers[i] = slices.Concat(layers[i], add)
	s.rules.Layers = layers
}

// withDefaults is e with the defaults of decision 4: allow for a built-in
// read-only tool, ask for every other tool and for each of mcp's, whatever
// it declares.
func withDefaults(e permission.Engine, mcp []tool.Tool) permission.Engine {
	names := make(map[string]bool, len(mcp))
	for _, t := range mcp {
		names[t.Spec().Name] = true
	}
	e.Default = func(r permission.Rule) permission.Mode {
		if r.ReadOnly && !names[r.Tool] {
			return permission.ModeAllow
		}
		return permission.ModeAsk
	}
	return e
}

// visible are tools without those the rules hide: a tool whose last
// matching rule is a deny with no pattern, or "*", is neither sent nor
// found by tool_search (decision 6).
func visible(e permission.Engine, tools []tool.Tool) []tool.Tool {
	return slices.DeleteFunc(slices.Clone(tools), func(t tool.Tool) bool { return e.Hidden(t.Spec().Name) })
}

// refusalOf is the text the model reads for a call the rules or a limit
// deny.
func refusalOf(r permission.Rule, v permission.Verdict) string {
	if v.Limit != "" {
		return fmt.Sprintf("%s did not run: %s.", r.Tool, v.Limit)
	}
	return fmt.Sprintf("%s did not run: the permission rule %q (%s) denies it.", r.Tool, v.By.String(), v.By.Source)
}

// logDecision logs a call's verdict, its tool, subjects, effect and the
// rule that decided, with its source, "from" (decision 12).
func (a *Agent) logDecision(ctx context.Context, s *liveSession, r permission.Rule, v permission.Verdict) {
	attrs := []any{slog.String("session", string(s.id)), slog.String("tool", r.Tool), slog.Any("subjects", r.Subjects),
		slog.String("effect", string(v.Effect)), slog.String("rule", v.By.String()), slog.String("from", v.By.Source.String())}
	if len(r.Outside) > 0 {
		attrs = append(attrs, slog.Any("outside", r.Outside))
	}
	if v.Limit != "" {
		attrs = append(attrs, slog.String("limit", v.Limit))
	}
	a.logger.InfoContext(ctx, "permission: decided", attrs...)
}

// The scopes an "always" answer saves to, and /permissions adds to
// (decisions E and H).
const (
	scopeSession = "session"
	scopeProject = "project"
	scopeGlobal  = "global"
)

// remember saves rules for s in scope, and reports where they went: the
// session's layer; the project store, and its layer; or the global file,
// whose rules become the global layer. A save that fails keeps the rules
// in the session's layer, and the error says why (decision E).
func (a *Agent) remember(ctx context.Context, s *liveSession, scope string, rules []permission.Spec) (permission.RuleSource, error) {
	session := permission.RuleSource{Layer: permission.LayerSession}
	keep := func(err error) (permission.RuleSource, error) {
		a.addToLayer(s, layerSession, session, rules)
		return session, err
	}
	switch scope {
	case scopeProject:
		if a.perms.Store.Path == "" {
			return keep(errNoStore)
		}
		if _, err := a.perms.Store.Add(ctx, cmp.Or(s.root, s.cwd), rules...); err != nil {
			return keep(err)
		}
		src := permission.RuleSource{Layer: permission.LayerStore, File: a.perms.Store.Path}
		a.addToLayer(s, layerStore, src, rules)
		return src, nil
	case scopeGlobal:
		if a.perms.AddGlobal == nil {
			return keep(errNoGlobal)
		}
		var global []permission.Spec
		for _, r := range rules {
			got, err := a.perms.AddGlobal(r)
			if err != nil {
				if global != nil {
					a.setLayer(s, layerGlobal, global)
				}
				return keep(err)
			}
			global = got
		}
		a.setLayer(s, layerGlobal, global)
		if len(global) > 0 {
			return global[len(global)-1].Source, nil
		}
		return permission.RuleSource{Layer: permission.LayerGlobal}, nil
	}
	return keep(nil)
}

// errNoStore and errNoGlobal are remember's errors when the agent was given
// no project store, or no way to edit the global file.
var (
	errNoStore  = errors.New("this agent keeps no project store")
	errNoGlobal = errors.New("this agent cannot edit the global settings file")
)

// noteSaveFailed tells the person a rule was not saved where they chose,
// sent as a command's reply is (decision E).
func (a *Agent) noteSaveFailed(ctx context.Context, conn *acp.AgentSideConnection, s *liveSession, rules []permission.Spec, err error) {
	a.logger.WarnContext(ctx, "permissions: a rule was not saved", slog.String("error", err.Error()))
	if conn == nil {
		return
	}
	a.mu.Lock()
	v := s.voice
	a.mu.Unlock()
	for _, r := range rules {
		newUpdates(ctx, conn, s.id).send(acp.UpdateAgentMessageText(
			v.Text(personality.PermissionsFailed, personality.Args{argRule: r.String(), argError: err.Error()}) + "\n\n"))
	}
}
