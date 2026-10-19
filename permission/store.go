package permission

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// PermissionsFile is the project store's file in gobble's config
// directory: each project's "always" answers (0005-PLAN F6-2 decision 10).
const PermissionsFile = "permissions.json"

// StoredRule is one rule as a store or a settings file writes it: one of
// Allow, Ask and Deny holds "<tool>" or "<tool>(<pattern>)".
type StoredRule struct {
	Allow string `json:"allow,omitzero"`
	Ask   string `json:"ask,omitzero"`
	Deny  string `json:"deny,omitzero"`
}

// StoredOf is s as a store writes it.
func StoredOf(s Spec) StoredRule {
	switch s.Effect {
	case ModeAllow:
		return StoredRule{Allow: s.Rule()}
	case ModeAsk:
		return StoredRule{Ask: s.Rule()}
	}
	return StoredRule{Deny: s.Rule()}
}

// Spec is r read by ParseSpec.
func (r StoredRule) Spec() (Spec, error) {
	switch {
	case r.Allow != "" && r.Ask == "" && r.Deny == "":
		return ParseSpec(ModeAllow, r.Allow)
	case r.Ask != "" && r.Allow == "" && r.Deny == "":
		return ParseSpec(ModeAsk, r.Ask)
	case r.Deny != "" && r.Allow == "" && r.Ask == "":
		return ParseSpec(ModeDeny, r.Deny)
	}
	return Spec{}, fmt.Errorf("%w: want one of allow, ask or deny", ErrBadRule)
}

// RuleStore is permissions.json: each project's saved rules, by its
// root's real path, in the order saved (decisions 10 and E). A root's
// rules are its own: no parent's apply.
type RuleStore struct {
	// Path is the file, PermissionsFile in gobble's config directory.
	Path string
}

// Load is every project's saved rules; a missing file is none.
func (s RuleStore) Load() (map[string][]StoredRule, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string][]StoredRule{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", PermissionsFile, err)
	}
	m := map[string][]StoredRule{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", PermissionsFile, err)
	}
	return m, nil
}

// Rules are the rules saved for root's real path, each with the store as
// its source. One that does not parse is left out, and named in the
// error, which the rest come with.
func (s RuleStore) Rules(root string) ([]Spec, error) {
	m, err := s.Load()
	if err != nil {
		return nil, err
	}
	p, err := realPath(root)
	if err != nil {
		return nil, err
	}
	var out []Spec
	var errs []error
	for k, rules := range m {
		if !samePath(k, p) {
			continue
		}
		for _, r := range rules {
			spec, err := r.Spec()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %s: %w", PermissionsFile, k, err))
				continue
			}
			spec.Source = RuleSource{Layer: LayerStore, File: s.Path}
			out = append(out, spec)
		}
	}
	return out, errors.Join(errs...)
}

// Add appends rules to those saved for root's real path, which it returns;
// a rule already saved there is not saved twice.
func (s RuleStore) Add(ctx context.Context, root string, rules ...Spec) (string, error) {
	p, err := realPath(root)
	if err != nil {
		return "", err
	}
	return p, s.modify(ctx, func(m map[string][]StoredRule) bool {
		key := keyOf(m, p)
		changed := false
		for _, r := range rules {
			if sr := StoredOf(r); !slices.Contains(m[key], sr) {
				m[key] = append(m[key], sr)
				changed = true
			}
		}
		return changed
	})
}

// Remove removes the first rule saved for root's real path that is rule,
// reporting whether there was one; a root left with none is dropped.
func (s RuleStore) Remove(ctx context.Context, root string, rule Spec) (bool, error) {
	p, err := realPath(root)
	if err != nil {
		return false, err
	}
	removed := false
	err = s.modify(ctx, func(m map[string][]StoredRule) bool {
		key := keyOf(m, p)
		if i := slices.Index(m[key], StoredOf(rule)); i >= 0 {
			m[key] = slices.Delete(m[key], i, i+1)
			if len(m[key]) == 0 {
				delete(m, key)
			}
			removed = true
		}
		return removed
	})
	return removed, err
}

// keyOf is m's key for the real path p: one already there for it, or p.
func keyOf(m map[string][]StoredRule, p string) string {
	for k := range m {
		if samePath(k, p) {
			return k
		}
	}
	return p
}

// modify changes the saved rules under the file's lock, and writes them,
// keys sorted and indented two spaces, owner-only, when change reports a
// change.
func (s RuleStore) modify(ctx context.Context, change func(map[string][]StoredRule) bool) (err error) {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("%s: %w", PermissionsFile, err)
	}
	release, err := filelock.Lock(ctx, s.Path+".lock", filelock.EditPolicy)
	if err != nil {
		return fmt.Errorf("%s: %w", PermissionsFile, err)
	}
	defer func() { err = errors.Join(err, release()) }()
	m, err := s.Load()
	if err != nil {
		return err
	}
	if !change(m) {
		return nil
	}
	data, err := json.Marshal(m, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("%s: %w", PermissionsFile, err)
	}
	return fsx.WriteFileMode(s.Path, append(data, '\n'), 0o600)
}
