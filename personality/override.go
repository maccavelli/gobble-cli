package personality

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// PromptsDir is the directory beside the personalities that holds the
// model-facing text a user overrides, which is no personality (0022-MADR
// D15).
const PromptsDir = "prompts"

// loadOverride reads the user's files for a built-in personality, in dir,
// and lays them over the built-in: a personality.md replaces its brief and
// keeps its name and level, so the house rules and Bombastic's question
// stay as they are (D9, D10); a messages.md replaces the keys it has, the
// rest falling back to the built-in's. boring has no brief, no leads and no
// ambient lines, and a file cannot give it any (D12, D13, D15).
func loadOverride(dir, name string) (builtinData, error) {
	fail := func(err error) (builtinData, error) {
		return builtinData{}, fmt.Errorf("personality %s (%s): %w", name, dir, err)
	}
	base := builtins()[name]
	d := base
	found := false
	data, err := os.ReadFile(filepath.Join(dir, "personality.md")) //nolint:gosec // the user's own config directory
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return fail(err)
	case name == Boring:
		return fail(errors.New("boring has no brief, and a personality.md cannot give it one"))
	default:
		brief, err := ParseBrief(data)
		if err != nil {
			return fail(err)
		}
		if brief.Name != name || brief.Level != base.brief.Level {
			return fail(fmt.Errorf("a built-in's brief keeps its name and level, %s and %d; this one has %s and %d",
				name, base.brief.Level, brief.Name, brief.Level))
		}
		d.brief, found = brief, true
	}
	cat, err := readMessages(dir, name)
	if err != nil {
		return fail(err)
	}
	if cat != nil {
		d.catalog, found = overlay(cat, base.catalog), true
	}
	if !found {
		return fail(errors.New("neither personality.md nor messages.md is here"))
	}
	return d, nil
}

// readMessages is dir's messages.md for the named personality, checked as
// boring's (0022-MADR D5): boring's may give no leads and no ambient
// lines (D12, D13). It is nil when dir has none.
func readMessages(dir, name string) (*Catalog, error) {
	data, err := os.ReadFile(filepath.Join(dir, "messages.md")) //nolint:gosec // the user's own config directory, or a trusted project's
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	cat, err := ParseCatalog(data)
	if err != nil {
		return nil, err
	}
	if err := CheckCatalog(cat); err != nil {
		return nil, err
	}
	if name == Boring {
		for _, k := range cat.Keys() {
			if slices.Contains(LeadKeys, k) || slices.Contains(AmbientKeys, k) {
				return nil, fmt.Errorf("%s: boring has no leads and no ambient lines", k)
			}
		}
	}
	return cat, nil
}

// WithProject is the Set with a trusted project's personality files, under
// dir, over it, for one session (0005-PLAN F6-1 decision 8): each
// <name>/messages.md lays its keys over that personality's messages, the
// project's winning. A project only changes what a personality says: its
// personality.md is refused, since a brief sets who speaks (0022-MADR D15),
// and a directory naming no personality the Set knows is left out. Each is
// an error naming the file; the rest still load. A missing dir is the Set
// as it is.
func (set *Set) WithProject(dir string) (*Set, []error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return set, nil
	}
	if err != nil {
		return set, []error{fmt.Errorf("personality: %w", err)}
	}
	out := &Set{user: map[string]builtinData{}, over: map[string]builtinData{}, dirs: map[string]string{}}
	if set != nil {
		maps.Copy(out.user, set.user)
		maps.Copy(out.over, set.over)
		maps.Copy(out.dirs, set.dirs)
		out.order = slices.Clone(set.order)
	}
	var errs []error
	for _, e := range entries {
		if !e.IsDir() || e.Name() == PromptsDir {
			continue
		}
		path := filepath.Join(dir, e.Name())
		name := out.Canonical(e.Name())
		if name == "" || name != e.Name() {
			errs = append(errs, fmt.Errorf("personality %s (%s): a project changes only a personality you have; the personalities are %s",
				e.Name(), path, strings.Join(out.Names(), ", ")))
			continue
		}
		if _, err := os.Stat(filepath.Join(path, "personality.md")); err == nil {
			errs = append(errs, fmt.Errorf("personality %s (%s): a project may not change a personality's brief, personality.md (0022-MADR D15); its messages.md still loads",
				name, path))
		}
		cat, err := readMessages(path, name)
		if err != nil {
			errs = append(errs, fmt.Errorf("personality %s (%s): %w", name, path, err))
			continue
		}
		if cat == nil {
			continue
		}
		d, _ := out.data(name) //nolint:errcheck // Canonical found it
		d.catalog = overlay(cat, d.catalog)
		if _, user := out.user[name]; user {
			out.user[name] = d
		} else {
			out.over[name] = d
		}
		out.dirs[name] = path
	}
	return out, errs
}

// overlay is a catalog with top's keys and, for the keys top leaves out,
// under's.
func overlay(top, under *Catalog) *Catalog {
	out := &Catalog{variants: map[Key][]string{}}
	if under != nil {
		maps.Copy(out.variants, under.variants)
	}
	maps.Copy(out.variants, top.variants)
	return out
}
