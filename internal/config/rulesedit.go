package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// AppendRule appends r to file's permissions.rules in place (0005-PLAN
// F6-2b decision E): a line after a block list's last item, at its
// indentation, so every comment stays; a flow list, the template's [],
// becomes a block list holding it; an absent key is uncommented from the
// template's block, or added. file must exist. The result must read back
// as the list with r after it, and every other key as it was.
func AppendRule(file string, r PermissionRule) error {
	return editRules(file, func(old []PermissionRule) ([]PermissionRule, error) {
		return append(old, r), nil
	}, func(l *lines, seq *yaml.Node) bool {
		item, err := ruleLine(seq.Column-1, r)
		if err != nil {
			return false
		}
		at := lastLine(l, nil, seq)
		l.apply([]op{{from: at, to: at, text: []string{item}}})
		return true
	})
}

// RemoveRule removes from file's permissions.rules the item on line, which
// must still be r, with its lines (decision H): a block list's item is
// deleted, every other line staying; a flow list is written again without
// it. The result must read back as the list without it.
func RemoveRule(file string, line int, r PermissionRule) error {
	var index int
	return editRules(file, func(old []PermissionRule) ([]PermissionRule, error) {
		index = -1
		for i, o := range old {
			if o.Source.Line == line && sameRule(o, r) {
				index = i
			}
		}
		if index < 0 {
			effect, text := r.Effect()
			return nil, fmt.Errorf("%s:%d: no longer holds %s: %s; the file is unchanged", file, line, effect, text)
		}
		return append(old[:index:index], old[index+1:]...), nil
	}, func(l *lines, seq *yaml.Node) bool {
		if len(seq.Content) < 2 {
			return false // the list's last item: written again as []
		}
		c := seq.Content[index]
		end := lastLine(l, &yaml.Node{Line: c.Line, Column: seq.Column}, c)
		l.apply([]op{{from: c.Line - 1, to: end, text: nil}})
		return true
	})
}

// sameRule reports whether a and b are one rule.
func sameRule(a, b PermissionRule) bool {
	ae, at := a.Effect()
	be, bt := b.Effect()
	return ae == be && at == bt
}

// ruleLine is r as a block list's item at indent.
func ruleLine(indent int, r PermissionRule) (string, error) {
	effect, text := r.Effect()
	v, err := renderScalar(&yaml.Node{Kind: yaml.ScalarNode, Tag: tagStr, Value: text})
	if err != nil {
		return "", err
	}
	return strings.Repeat(" ", indent) + "- " + effect + ": " + v, nil
}

// editRules locks file, reads its own rules, and writes want's list: by
// inPlace's line edit when the key is an active block list, otherwise by
// writing the key's value again. Either result is checked by readsBack.
func editRules(file string, want func(old []PermissionRule) ([]PermissionRule, error), inPlace func(l *lines, seq *yaml.Node) bool) (err error) {
	release, err := lockSettings(file)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, release()) }()
	data, err := os.ReadFile(file) //nolint:gosec // G304: the settings file the caller names
	if err != nil {
		return err
	}
	root, ds := Parse(file, data)
	if root == nil {
		return diagnosticsError(ds)
	}
	steps, err := lookup(root, pathPermissionRules)
	if err != nil {
		return err
	}
	var old []PermissionRule
	var seq *yaml.Node
	if len(steps) == 2 {
		seq = resolveAlias(steps[1].value)
		if seq.Kind == yaml.SequenceNode {
			var bad []Diagnostic
			if old, bad = convertRules(file, seq, LayerGlobal); len(bad) > 0 {
				return diagnosticsError(bad)
			}
		} else if !isNull(seq) {
			return fmt.Errorf("%s: %s: want a list of rules, have %s", file, pathPermissionRules, kindOf(seq))
		}
	}
	rules, err := want(old)
	if err != nil {
		return err
	}
	e := Edit{Path: pathPermissionRules, Value: rulesNode(rules, false)}
	var out []byte
	l := splitLines(data)
	if seq != nil && seq.Kind == yaml.SequenceNode && seq.Style&yaml.FlowStyle == 0 && len(seq.Content) > 0 && !steps[1].flow && inPlace(l, seq) {
		out = l.bytes()
		if !readsBack(file, root, out, e) {
			return unsafeEdit(file, pathPermissionRules)
		}
	} else if out, err = Apply(file, data, e); err != nil {
		return err
	}
	if bytes.Equal(out, data) {
		return nil
	}
	return fsx.WriteFileMode(file, out, 0o644)
}
