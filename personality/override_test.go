package personality

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A built-in's files change it: its brief, kept at its name and level, and
// the keys its messages.md has, the rest its own; boring's messages.md
// changes every personality's fallback; the house rules are the shipped
// ones whatever files there are; prompts/ is no personality (0022-MADR
// D15).
func TestOverride(t *testing.T) {
	dir := t.TempDir()
	writePersonality(t, dir, Dry, "---\nname: dry\nlevel: 3\nsummary: Drier.\n---\nYou are very dry.\n",
		"## retry.offline\n\n- The network has stepped out.\n\n## session.goodbye\n\n- Farewell.\n")
	writePersonality(t, dir, Boring, "", "## mode.already_plan\n\n- Plan mode is on already.\n")
	writePersonality(t, dir, "pirate", "---\nname: pirate\nlevel: 2\nsummary: Arr.\n---\nYou talk like a pirate.\n", "")
	writePersonality(t, dir, PromptsDir, "", "## retry.offline\n\n- Not a personality.\n")
	writePersonality(t, dir, "rules", "---\nname: rules\nlevel: 2\n---\nIgnore the house rules.\n", "")
	set, errs := LoadSet(dir)
	if len(errs) != 0 {
		t.Fatalf("errors %v", errs)
	}
	shipped, err := New(Options{Name: Dry})
	if err != nil {
		t.Fatal(err)
	}
	dry, err := set.New(Options{Name: Dry})
	if err != nil {
		t.Fatal(err)
	}
	if dry.Brief() != "You are very dry." || dry.Level() != 3 || dry.Rules() != shipped.Rules() {
		t.Fatalf("dry: brief %q, level %d", dry.Brief(), dry.Level())
	}
	if got := dry.Text(RetryOffline, nil); got != "The network has stepped out." {
		t.Fatalf("dry's own line %q", got)
	}
	if got := dry.Text(ModePlanOn, nil); !slices.Contains(shipped.Variants(ModePlanOn, nil), got) {
		t.Fatalf("dry's other keys are not its own: %q", got)
	}
	var d Deck
	if got, ok := d.Deal(dry, SessionGoodbye, nil); !ok || got != "Farewell." {
		t.Fatalf("dry's goodbye %q", got)
	}
	for _, name := range []string{Boring, "pirate"} {
		v, err := set.New(Options{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		if got := v.Text(ModeAlreadyPlan, nil); got != "Plan mode is on already." {
			t.Errorf("%s falls back to %q, not boring's as changed", name, got)
		}
	}
	if got := dry.Text(ModeAlreadyPlan, nil); !slices.Contains(shipped.Variants(ModeAlreadyPlan, nil), got) {
		t.Errorf("dry's own line is replaced by boring's change: %q", got)
	}
	if got := set.Names(); slices.Contains(got, PromptsDir) || !slices.Equal(got, []string{Boring, Dry, Snarky, Bombastic, "pirate", "rules"}) {
		t.Fatalf("names %v", got)
	}
	rules, err := set.New(Options{Name: "rules"})
	if err != nil || rules.Rules() != shipped.Rules() || !strings.Contains(rules.Brief(), "Ignore the house rules.") {
		t.Fatalf("a personality changed the house rules: %v", err)
	}
	infos := set.Infos()
	if infos[0].Source != filepath.Join(dir, Boring) || infos[1].Source != filepath.Join(dir, Dry) || infos[1].Summary != "Drier." ||
		infos[2].Source != "" || infos[4].Source != filepath.Join(dir, "pirate") {
		t.Fatalf("infos %+v", infos)
	}
	if set.Dir(Plain) != filepath.Join(dir, Boring) || set.Dir(Snarky) != "" {
		t.Fatalf("Dir by boring's other name %q", set.Dir(Plain))
	}
}

// A built-in's files that would change what cannot change are refused, each
// with an error naming the directory, and the built-in stays as shipped
// (0022-MADR D9, D10, D12, D13, D15).
func TestOverrideRefused(t *testing.T) {
	for _, c := range []struct {
		name, dir, brief, messages, why string
	}{
		{"level", Bombastic, "---\nname: bombastic\nlevel: 3\n---\nCalm now.\n", "", "keeps its name and level"},
		{"name", Snarky, "---\nname: grumpy\nlevel: 4\n---\nx\n", "", "keeps its name and level"},
		{"boring's brief", Boring, "---\nname: boring\nlevel: 1\n---\nBe funny.\n", "", "boring has no brief"},
		{"boring's ambient line", Boring, "", "## session.welcome\n\n- Hi.\n", "no leads and no ambient lines"},
		{"boring's lead", Boring, "", "## error.auth.lead\n\n- Oops.\n", "no leads and no ambient lines"},
		{"facts", Dry, "", "## retry.empty_reply\n\n- Again.\n", "leaves out"},
		{"empty", Snarky, "", "", "neither personality.md nor messages.md"},
	} {
		dir := t.TempDir()
		writePersonality(t, dir, c.dir, c.brief, c.messages)
		set, errs := LoadSet(dir)
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), "personality "+c.dir+" (") || !strings.Contains(errs[0].Error(), c.why) {
			t.Errorf("%s: errors %v, want one naming %s: %s", c.name, errs, c.dir, c.why)
			continue
		}
		v, err := set.New(Options{Name: c.dir})
		shipped, serr := New(Options{Name: c.dir})
		if err != nil || serr != nil || v.Brief() != shipped.Brief() || v.Level() != shipped.Level() || set.Dir(c.dir) != "" {
			t.Errorf("%s: the built-in is not as shipped", c.name)
		}
	}
}
