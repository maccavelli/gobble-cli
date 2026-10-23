package acpserver

import (
	"context"
	"errors"
	"strings"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/permission"
)

// A tool call's options beyond allow_once and reject_once: an "always" for
// this session, this project and everywhere, each way (0005-PLAN F6-2b
// decision D).
const (
	allowSession  acp.PermissionOptionId = "allow_session"
	allowProject  acp.PermissionOptionId = "allow_project"
	allowAlways   acp.PermissionOptionId = "allow_always"
	rejectSession acp.PermissionOptionId = "reject_session"
	rejectProject acp.PermissionOptionId = "reject_project"
	rejectAlways  acp.PermissionOptionId = "reject_always"
)

// scopes are the "always" options' scopes.
var scopes = map[acp.PermissionOptionId]string{
	allowSession: scopeSession, allowProject: scopeProject, allowAlways: scopeGlobal,
	rejectSession: scopeSession, rejectProject: scopeProject, rejectAlways: scopeGlobal,
}

// question is what a tool call's question carries beyond the call: a
// limit's reason, a repeated call's run, why "always" is not offered, the
// rule that asked, and whether the six "always" options are offered and a
// "never" would hide the tool (decisions A, C, D and G).
type question struct {
	limit, onceOnly string
	repeated        int
	by              *permission.Spec
	always, hides   bool
	// saves, for a shell call, are the patterns an "always" saves, which
	// its options' names show (F6-3 decision 4a).
	saves []string
	// title is the call's whole description, for a client that shows it
	// (0029-MADR D3).
	title string
}

// answer is the option chosen, none for a cancel, and a reject's reason.
type answer struct {
	option   acp.PermissionOptionId
	feedback string
}

// options are q's options, in decision D's order.
func (q question) options() []acp.PermissionOption {
	opt := func(id acp.PermissionOptionId, name string, kind acp.PermissionOptionKind) acp.PermissionOption {
		return acp.PermissionOption{OptionId: id, Name: name, Kind: kind}
	}
	never := func(name string) string {
		if q.hides {
			return name + ", hiding the tool"
		}
		return name
	}
	what := ""
	if len(q.saves) > 0 {
		what = " `" + strings.Join(q.saves, "`, `") + "`"
	}
	out := []acp.PermissionOption{opt(allowOnce, "Allow once", acp.PermissionOptionKindAllowOnce)}
	if q.always {
		out = append(out,
			opt(allowSession, "Allow"+what+" for this session", acp.PermissionOptionKindAllowAlways),
			opt(allowProject, "Allow"+what+" for this project", acp.PermissionOptionKindAllowAlways),
			opt(allowAlways, "Allow"+what+" everywhere", acp.PermissionOptionKindAllowAlways))
	}
	out = append(out, opt(rejectOnce, "Reject", acp.PermissionOptionKindRejectOnce))
	if q.always {
		out = append(out,
			opt(rejectSession, never("Reject"+what+" for this session"), acp.PermissionOptionKindRejectAlways),
			opt(rejectProject, never("Reject"+what+" for this project"), acp.PermissionOptionKindRejectAlways),
			opt(rejectAlways, never("Reject"+what+" everywhere"), acp.PermissionOptionKindRejectAlways))
	}
	return out
}

// The _meta.gobble keys of a tool call's question for a client that lays
// it out itself: the call's whole description, cut to maxFullTitle runes;
// the patterns an "always" saves; and whether a "never" hides the tool
// (0029-MADR D3).
const (
	keyTitle     = "title"
	keySaves     = "saves"
	keyHides     = "hides"
	maxFullTitle = 2000
)

// meta is q's _meta.gobble: its marks, its rule and its description, or
// none.
func (q question) meta() map[string]any {
	g := map[string]any{}
	switch {
	case q.limit != "":
		g[keyLimit] = q.limit
	case q.repeated != 0:
		g[keyRepeated] = q.repeated
	case q.onceOnly != "":
		g[keyOnceOnly] = q.onceOnly
	}
	if q.by != nil {
		g[keyRule] = map[string]any{keyRule: q.by.String(), keyFrom: q.by.Source.String()}
	}
	if q.title != "" {
		g[keyTitle] = cutTo(q.title, maxFullTitle)
	}
	if len(q.saves) > 0 { // only an "always" has patterns to save
		g[keySaves] = q.saves
	}
	if q.always && q.hides { // as options shows it: only beside an "always"
		g[keyHides] = true
	}
	if len(g) == 0 {
		return nil
	}
	return map[string]any{gobble: g}
}

// acpPolicy asks the client about a call (session/request_permission). A
// cancelled request is a refusal.
type acpPolicy struct {
	conn    *acp.AgentSideConnection
	session acp.SessionId
}

// Decide asks a repeated call's question, whose two options approve no
// write (0005-PLAN F5-3 decision 1).
func (p *acpPolicy) Decide(ctx context.Context, r permission.Rule) (permission.Decision, error) {
	a, err := p.ask(ctx, r, question{repeated: r.Repeated})
	return permission.Decision{Allow: err == nil && a.option == allowOnce}, err
}

// ask asks the client about r with q's options and marks, and gives the
// option chosen with a reject's reason, cut to maxFeedback.
func (p *acpPolicy) ask(ctx context.Context, r permission.Rule, q question) (answer, error) {
	if p.conn == nil {
		return answer{}, errors.New("no client connection")
	}
	q.title = r.Title
	resp, err := p.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
		Meta:      q.meta(),
		SessionId: p.session,
		ToolCall:  acp.ToolCallUpdate{ToolCallId: acp.ToolCallId(r.CallID), Title: new(ruleTitle(r)), RawInput: jsonRaw(r.Input)},
		Options:   q.options(),
	})
	if err != nil {
		return answer{}, err
	}
	var a answer
	if s := resp.Outcome.Selected; s != nil {
		a.option = s.OptionId
	}
	if g, ok := resp.Meta[gobble].(map[string]any); ok {
		if f, ok := g[keyFeedback].(string); ok {
			a.feedback = cutBytes(f, maxFeedback)
		}
	}
	return a, nil
}
