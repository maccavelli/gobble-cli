package acpserver

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/personality"
)

// The placeholders /permissions' replies fill.
const (
	argRule   = "rule"
	argNumber = "number"
)

// cmdPermissions is /permissions (0005-PLAN F6-2b decision H): with no
// arguments the rules in force, numbered in evaluation order, each with
// its source, between the defaults and gobble's limits; "remove N" removes
// one; "allow|ask|deny RULE [session|project|global]" adds one.
func (a *Agent) cmdPermissions(ctx context.Context, t *cmdTurn, args string) error {
	fields := strings.Fields(args)
	switch {
	case len(fields) == 0:
		t.say(a.listRules(t))
	case fields[0] == "remove" && len(fields) == 2:
		t.say(a.removeRule(ctx, t, fields[1]) + "\n")
	case slices.Contains([]string{string(permission.ModeAllow), string(permission.ModeAsk), string(permission.ModeDeny)}, fields[0]) && len(fields) > 1:
		t.say(a.addRule(ctx, t, fields) + "\n")
	default:
		t.say(t.text(personality.PermissionsUsage, nil) + "\n")
	}
	return nil
}

// numbered is one rule in force: its layer, its place there, and itself.
type numbered struct {
	layer, index int
	rule         permission.Spec
}

// inForce are s's rules in evaluation order.
func (a *Agent) inForce(s *liveSession) []numbered {
	var out []numbered
	for i, layer := range a.rulesOf(s).Layers {
		for j, r := range layer {
			out = append(out, numbered{layer: i, index: j, rule: r})
		}
	}
	return out
}

func (a *Agent) listRules(t *cmdTurn) string {
	var b strings.Builder
	b.WriteString(t.text(personality.PermissionsDefaults, nil) + "\n\n")
	rules := a.inForce(t.s)
	if len(rules) == 0 {
		b.WriteString(t.text(personality.PermissionsNone, nil) + "\n")
	}
	for i, r := range rules {
		b.WriteString(t.text(personality.PermissionsRule, personality.Args{
			argNumber: strconv.Itoa(i + 1), argRule: r.rule.String(), argFrom: r.rule.Source.String()}) + "\n")
	}
	// The session's grants follow its rules (0005-PLAN F6-5 decision 8).
	for i, g := range a.grantsOf(t.s) {
		b.WriteString(t.text(personality.PermissionsRule, personality.Args{
			argNumber: strconv.Itoa(len(rules) + i + 1), argRule: g.String(), argFrom: g.scope}) + "\n")
	}
	b.WriteString("\n" + t.text(personality.PermissionsLimits, nil) + "\n")
	return b.String()
}

// removeRule removes rule n: a session rule, a store rule, or a global
// file's rule, whose file is edited in place; a trusted project's, --set's
// and the defaults are changed where they are set.
func (a *Agent) removeRule(ctx context.Context, t *cmdTurn, n string) string {
	rules, grants := a.inForce(t.s), a.grantsOf(t.s)
	i, err := strconv.Atoi(n)
	if err != nil || i < 1 || i > len(rules)+len(grants) {
		return t.text(personality.PermissionsNoRule, personality.Args{argNumber: n})
	}
	if i > len(rules) {
		g := grants[i-len(rules)-1]
		args := personality.Args{argNumber: n, argRule: g.String(), argFrom: g.scope}
		if !a.dropGrant(t.s, g) {
			return t.text(personality.PermissionsNoRule, args)
		}
		return t.text(personality.PermissionsRemoved, args)
	}
	r := rules[i-1]
	args := personality.Args{argNumber: n, argRule: r.rule.String(), argFrom: r.rule.Source.String()}
	switch r.layer {
	case layerGlobal:
		if a.perms.RemoveGlobal == nil {
			return t.text(personality.PermissionsFixed, args)
		}
		global, err := a.perms.RemoveGlobal(r.rule)
		if err != nil {
			return t.text(personality.PermissionsBad, personality.Args{argError: err.Error()})
		}
		a.setLayer(t.s, layerGlobal, global)
	case layerStore:
		if _, err := a.perms.Store.Remove(ctx, cmp.Or(t.s.root, t.s.cwd), r.rule); err != nil {
			return t.text(personality.PermissionsBad, personality.Args{argError: err.Error()})
		}
		a.dropFromLayer(t.s, r)
	case layerSession:
		a.dropFromLayer(t.s, r)
	default:
		return t.text(personality.PermissionsFixed, args)
	}
	return t.text(personality.PermissionsRemoved, args)
}

// dropFromLayer removes r from its layer, when it is still there.
func (a *Agent) dropFromLayer(s *liveSession, r numbered) {
	a.mu.Lock()
	defer a.mu.Unlock()
	layer := s.rules.Layers[r.layer]
	if r.index >= len(layer) || layer[r.index] != r.rule {
		return
	}
	layers := slices.Clone(s.rules.Layers)
	layers[r.layer] = slices.Concat(layer[:r.index:r.index], layer[r.index+1:])
	s.rules.Layers = layers
}

// addRule adds "EFFECT RULE [SCOPE]" to the scope, the session's by
// default, after permission.ParseSpec has checked it.
func (a *Agent) addRule(ctx context.Context, t *cmdTurn, fields []string) string {
	scope := scopeSession
	if last := fields[len(fields)-1]; len(fields) > 2 && slices.Contains([]string{scopeSession, scopeProject, scopeGlobal}, last) {
		scope, fields = last, fields[:len(fields)-1]
	}
	spec, err := permission.ParseSpec(permission.Mode(fields[0]), strings.Join(fields[1:], " "))
	if err != nil {
		return t.text(personality.PermissionsBad, personality.Args{argError: err.Error()})
	}
	src, err := a.remember(ctx, t.s, scope, []permission.Spec{spec})
	if err != nil {
		return t.text(personality.PermissionsFailed, personality.Args{argRule: spec.String(), argError: err.Error()})
	}
	return t.text(personality.PermissionsAdded, personality.Args{argRule: spec.String(), argFrom: src.String()})
}
