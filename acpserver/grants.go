package acpserver

import (
	"cmp"
	"context"
	stdjson "encoding/json"
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
)

// RequestPermissionsTool is the tool that asks the user for a grant
// (0005-MADR's native-tools item 11; 0012-MADR's row; 0005-PLAN F6-5),
// which the tool flags select as they select every tool.
const RequestPermissionsTool = "request_permissions"

const requestPermissionsTool = RequestPermissionsTool

// A grant's scopes (decision 4) and its question's answers (decision 1),
// which --yes does not answer.
const (
	grantTurn    = "turn"
	grantSession = "session"

	grantYes acp.PermissionOptionId = "grant"
	grantNo  acp.PermissionOptionId = "refuse"
)

// grant is a path the user granted: reads under it, and writes too when
// write is set, for the turn or the session (decisions 2 and 4).
type grant struct {
	path  string
	write bool
	scope string
}

// String is the grant as /permissions lists it.
func (g grant) String() string {
	if g.write {
		return "grant write: " + g.path
	}
	return "grant read: " + g.path
}

// readTools are the tools a read grant covers; a write grant covers these
// and editTools (decision 2).
var readTools = map[string]bool{"read": true, "grep": true, "find": true, "tree": true}

type requestPermissionsInput struct {
	Permissions struct {
		Read    []string `json:"read,omitzero" jsonschema:"paths to read, relative to the working directory or absolute; each covers everything below it"`
		Write   []string `json:"write,omitzero" jsonschema:"paths to change, which can be read too; each covers everything below it"`
		Network bool     `json:"network,omitzero" jsonschema:"network access, which nothing in gobble restricts"`
	} `json:"permissions" jsonschema:"the access asked for"`
	Reason string `json:"reason" jsonschema:"why the access is needed, which the user reads"`
	Scope  string `json:"scope,omitzero" jsonschema:"how long the grant lasts: turn, this prompt's turn, or session"`
}

// requestPermissionsSpec is the tool's spec: 0012-MADR's schema, deferred
// in the permissions family; it asks for itself, so it is read-only.
var requestPermissionsSpec = tool.New(requestPermissionsTool,
	"Ask the user for read or write access to paths, for this turn or this session, saying why. Call it after the user declined a call you need, or before a run of calls that would each ask. A granted path covers everything below it; deny rules and gobble's own limits still apply.",
	func(context.Context, requestPermissionsInput, tool.Env) (string, error) { return "", nil },
	tool.WithKind(tool.KindOther), tool.WithExposure(tool.ExposureDeferred), tool.WithFamily(tool.FamilyPermissions),
	tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true}),
	tool.WithSchema(func(s *jsonschema.Schema) {
		s.Properties["permissions"].MinProperties = new(1)
		s.Properties["reason"].MinLength = new(1)
		scope := s.Properties["scope"]
		scope.Enum = []any{grantTurn, grantSession}
		scope.Default = stdjson.RawMessage(`"` + grantTurn + `"`)
	})).Spec()

// requestPermissions is the tool, for one session's turn.
type requestPermissions struct {
	a    *Agent
	conn *acp.AgentSideConnection
	s    *liveSession
}

var (
	_ tool.Tool      = requestPermissions{}
	_ tool.Describer = requestPermissions{}
)

func (requestPermissions) Spec() tool.Spec { return requestPermissionsSpec }

func (requestPermissions) Describe(tool.Call, tool.Env) string { return "ask for permissions" }

// Run asks the user for the grant and records it (decisions 1, 3, 5 and 7).
func (r requestPermissions) Run(ctx context.Context, call tool.Call, env tool.Env) (tool.Result, error) {
	var in requestPermissionsInput
	if err := json.Unmarshal(call.Args, &in); err != nil {
		return tool.ErrorResult(fmt.Sprintf("%s: invalid arguments: %v", requestPermissionsTool, err)), nil
	}
	scope := cmp.Or(in.Scope, grantTurn)
	read, write := resolveAll(env.Cwd, in.Permissions.Read), resolveAll(env.Cwd, in.Permissions.Write)
	switch {
	case scope != grantTurn && scope != grantSession:
		return tool.ErrorResult(requestPermissionsTool + ": scope is turn or session"), nil
	case strings.TrimSpace(in.Reason) == "":
		return tool.ErrorResult(requestPermissionsTool + " needs a reason, which the user reads"), nil
	case len(read)+len(write) == 0 && in.Permissions.Network:
		return tool.TextResult(textNetwork), nil
	case len(read)+len(write) == 0:
		return tool.ErrorResult(requestPermissionsTool + " needs at least one path to read or write"), nil
	case len(write) > 0 && r.a.modeOf(r.s) == modePlan:
		return tool.ErrorResult("Plan mode is read-only, so write access cannot be granted. Call " + exitPlanTool + " with the plan to leave plan mode first."), nil
	case r.conn == nil:
		return tool.ErrorResult("there is no client to ask"), nil
	}
	what := grantText(read, write)
	resp, err := r.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
		SessionId: r.s.id,
		ToolCall:  acp.ToolCallUpdate{ToolCallId: acp.ToolCallId(call.ID), Title: new(cutTitle("Grant " + what + " for this " + scope + ": " + in.Reason))},
		Options: []acp.PermissionOption{
			{OptionId: grantYes, Name: "Grant", Kind: acp.PermissionOptionKindAllowOnce},
			{OptionId: grantNo, Name: "Refuse", Kind: acp.PermissionOptionKindRejectOnce},
		},
	})
	if ctx.Err() != nil {
		return tool.Result{}, context.Cause(ctx)
	}
	if err != nil {
		return tool.ErrorResult(fmt.Sprintf("the permission request failed: %v", err)), nil
	}
	if sel := resp.Outcome.Selected; sel == nil || sel.OptionId != grantYes {
		text := "The user refused the request."
		if g, ok := resp.Meta[gobble].(map[string]any); ok {
			if f, ok := g[keyFeedback].(string); ok && f != "" {
				text += " The user said: " + cutBytes(f, maxFeedback)
			}
		}
		return tool.ErrorResult(text), nil
	}
	r.a.addGrants(r.s, read, write, scope)
	text := "The user granted " + what + " for this " + scope + ": calls under these paths run without asking, unless a deny rule or one of gobble's limits refuses them."
	if in.Permissions.Network {
		text += " " + textNetwork
	}
	return tool.TextResult(text), nil
}

// textNetwork is the answer to a request for network access (decision 3a).
const textNetwork = "Nothing in gobble restricts network access, so there is nothing to grant for it: network calls run as they did."

// resolveAll is paths resolved against cwd, each once.
func resolveAll(cwd string, paths []string) []string {
	var out []string
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if abs := fsx.Resolve(cwd, p); !slices.Contains(out, abs) {
			out = append(out, abs)
		}
	}
	return out
}

// grantText names what a grant covers: "read access to a, b and write
// access to c".
func grantText(read, write []string) string {
	var parts []string
	if len(read) > 0 {
		parts = append(parts, "read access to "+strings.Join(read, ", "))
	}
	if len(write) > 0 {
		parts = append(parts, "write access to "+strings.Join(write, ", "))
	}
	return strings.Join(parts, " and ")
}

// addGrants adds the session's grants, each once.
func (a *Agent) addGrants(s *liveSession, read, write []string, scope string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	add := func(p string, w bool) {
		g := grant{path: p, write: w, scope: scope}
		if !slices.Contains(s.grants, g) {
			s.grants = append(slices.Clip(s.grants), g)
		}
	}
	for _, p := range read {
		add(p, false)
	}
	for _, p := range write {
		add(p, true)
	}
}

// grantsOf are the session's grants now.
func (a *Agent) grantsOf(s *liveSession) []grant {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(s.grants)
}

// endTurnGrants drops the session's turn grants, at its turn's end.
func (a *Agent) endTurnGrants(s *liveSession) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s.grants = slices.DeleteFunc(slices.Clone(s.grants), func(g grant) bool { return g.scope == grantTurn })
}

// dropGrant removes g, when it is still there.
func (a *Agent) dropGrant(s *liveSession, g grant) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	i := slices.Index(s.grants, g)
	if i < 0 {
		return false
	}
	s.grants = slices.Delete(slices.Clone(s.grants), i, i+1)
	return true
}

// grantable reports whether a grant could cover r: a file tool's call that
// no limit asked about (decision 6).
func grantable(r permission.Rule, v permission.Verdict) bool {
	return v.Limit == "" && r.Kind == permission.SubjectPath && len(r.Subjects) > 0 && (readTools[r.Tool] || editTools[r.Tool])
}

// byGrant is v as the session's grants leave it: a question no limit asked,
// about a call whose every path a grant covers, is allowed, the grant's
// (decisions 2a and 8). A deny stays a deny.
func byGrant(v permission.Verdict, r permission.Rule, grants []grant) permission.Verdict {
	if v.Effect != permission.ModeAsk || len(grants) == 0 || !grantable(r, v) {
		return v
	}
	write := editTools[r.Tool]
	var by *grant
	for _, p := range slices.Concat(r.Subjects, r.Outside) {
		i := slices.IndexFunc(grants, func(g grant) bool { return (g.write || !write) && within(p, g.path) })
		if i < 0 {
			return v
		}
		if by == nil {
			by = &grants[i]
		}
	}
	return permission.Verdict{Effect: permission.ModeAllow,
		By: permission.Spec{Effect: permission.ModeAllow, Tool: r.Tool, Source: permission.RuleSource{Layer: "grant " + by.scope}}}
}

// within reports whether p is dir or below it.
func within(p, dir string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && (rel == "." || filepath.IsLocal(rel))
}
