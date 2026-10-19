package personality

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
	data, err = os.ReadFile(filepath.Join(dir, "messages.md")) //nolint:gosec // the user's own config directory
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return fail(err)
	default:
		cat, err := ParseCatalog(data)
		if err != nil {
			return fail(err)
		}
		if err := CheckCatalog(cat); err != nil {
			return fail(err)
		}
		if name == Boring {
			for _, k := range cat.Keys() {
				if slices.Contains(LeadKeys, k) || slices.Contains(AmbientKeys, k) {
					return fail(fmt.Errorf("%s: boring has no leads and no ambient lines", k))
				}
			}
		}
		d.catalog, found = overlay(cat, base.catalog), true
	}
	if !found {
		return fail(errors.New("neither personality.md nor messages.md is here"))
	}
	return d, nil
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
