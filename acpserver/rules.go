package acpserver

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
)

// keyLimit is the _meta.gobble key of a question one of gobble's limits
// asked: a client offers no "always" for it, and --yes does not approve it
// (0005-PLAN F6-2 decisions 5 and 8).
const keyLimit = "limit"

// keyRepeated is the _meta.gobble key of a repeated call's question, the
// run's length: --yes does not approve it (0005-PLAN F6-2a's deviation 2).
const keyRepeated = "repeated"

// AllowOnceID is the option id of a tool call's "allow once", which only
// the questions about a tool call offer: print mode's --yes chooses it
// (0005-PLAN F6-2 decision 8).
const AllowOnceID = string(allowOnce)

// engineOf is a session's rule engine: its settings' rules, relative to
// its project root, or to cwd when it has none, with gobble's protected
// paths placed by its home and config directories (0005-PLAN F6-2
// decisions 3 to 5). F6-2b adds the project store's and the session's
// layers.
func engineOf(ss SessionSettings, cwd string) permission.Engine {
	return permission.Engine{Layers: [][]permission.Spec{ss.Rules}, Root: cmp.Or(ss.ProjectRoot, cwd),
		Home: ss.Home, ConfigDir: ss.ConfigDir}
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
