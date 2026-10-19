package config

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/maccavelli/gobble-cli/permission"
)

// pathPermissionRules is the permission rules' key (0005-PLAN F6-2).
const pathPermissionRules = "permissions.rules"

// PermissionRule is one rule as a settings file writes it: one of Allow,
// Ask and Deny holds the rule, "<tool>" or "<tool>(<pattern>)", and Source
// is where it was read (0005-PLAN F6-2 decisions 2 and 12).
type PermissionRule struct {
	Allow  string `yaml:"allow,omitempty" json:"allow,omitempty"`
	Ask    string `yaml:"ask,omitempty" json:"ask,omitempty"`
	Deny   string `yaml:"deny,omitempty" json:"deny,omitempty"`
	Source Source `yaml:"-" json:"-"`
}

// Effect and Text are the rule's effect and its text.
func (r PermissionRule) Effect() (effect, text string) {
	switch {
	case r.Allow != "":
		return string(permission.ModeAllow), r.Allow
	case r.Ask != "":
		return string(permission.ModeAsk), r.Ask
	}
	return string(permission.ModeDeny), r.Deny
}

// tagStr is YAML's string tag.
const tagStr = "!!str"

// textList is a list key's refusal where only a single value is taken.
const textList = "is a list of rules: write it in settings.yaml, or give a run's with --set PATH='[{allow: read}]'"

// convertRules reads a permissions.rules node: a sequence of one-key
// mappings, each allow, ask or deny of a rule permission.ParseSpec takes.
// A rule that is not one is left out with an error naming its line, and
// the layer applies without it (decision 2); a node that is not a sequence
// is one error, and the layer gives no rules.
func convertRules(file string, n *yaml.Node, layer Layer) ([]PermissionRule, []Diagnostic) {
	if n.Kind != yaml.SequenceNode {
		return nil, []Diagnostic{errorAt(file, n.Line, pathPermissionRules, "want a list of rules, have "+kindOf(n))}
	}
	var out []PermissionRule
	var ds []Diagnostic
	for _, item := range n.Content {
		item = resolveAlias(item)
		r, why := convertRule(item)
		if why != "" {
			ds = append(ds, errorAt(file, item.Line, pathPermissionRules, why))
			continue
		}
		r.Source = Source{Layer: layer, File: file, Line: item.Line}
		out = append(out, r)
	}
	return out, ds
}

// convertRule is one item's rule, or why it is not one.
func convertRule(n *yaml.Node) (PermissionRule, string) {
	if n.Kind != yaml.MappingNode || len(n.Content) != 2 {
		return PermissionRule{}, "want a rule, one of allow, ask or deny with its rule, such as \"allow: read\""
	}
	k, v := n.Content[0], resolveAlias(n.Content[1])
	if v.Kind != yaml.ScalarNode || isNull(v) {
		return PermissionRule{}, fmt.Sprintf("%s: want a rule, have %s", k.Value, kindOf(v))
	}
	if _, err := permission.ParseSpec(permission.Mode(k.Value), v.Value); err != nil {
		return PermissionRule{}, strings.TrimPrefix(err.Error(), permission.ErrBadRule.Error()+": ")
	}
	var r PermissionRule
	switch permission.Mode(k.Value) {
	case permission.ModeAllow:
		r.Allow = v.Value
	case permission.ModeAsk:
		r.Ask = v.Value
	default:
		r.Deny = v.Value
	}
	return r, ""
}

// rulesNode is rules as a YAML sequence, each with its origin when origin
// is set.
func rulesNode(rules []PermissionRule, origin bool) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode}
	if len(rules) == 0 {
		n.Style = yaml.FlowStyle
	}
	for _, r := range rules {
		effect, text := r.Effect()
		v := &yaml.Node{Kind: yaml.ScalarNode, Tag: tagStr, Value: text}
		if origin {
			v.LineComment = Origin(r.Source)
		}
		m := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: effect}, v}}
		n.Content = append(n.Content, m)
	}
	return n
}
