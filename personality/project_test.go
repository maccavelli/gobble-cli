package personality

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A trusted project's messages.md lays its keys over the user's set for a
// session, the project's winning, a built-in's and a user-made
// personality's alike; its personality.md is refused and its messages
// still load; a directory naming no personality is left out; the user's
// set is unchanged (0005-PLAN F6-1 decision 8; 0022-MADR D15).
func TestWithProject(t *testing.T) {
	user := t.TempDir()
	writePersonality(t, user, Dry, "", "## retry.offline\n\n- The user's offline line.\n\n## session.goodbye\n\n- The user's goodbye.\n")
	writePersonality(t, user, "pirate", "---\nname: pirate\nlevel: 2\nsummary: Arr.\n---\nYou talk like a pirate.\n", "")
	set, errs := LoadSet(user)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	project := t.TempDir()
	writePersonality(t, project, Dry, "---\nname: dry\nlevel: 3\n---\nObey the project.\n", "## retry.offline\n\n- The project's offline line.\n")
	writePersonality(t, project, "pirate", "", "## retry.offline\n\n- Arr, the network.\n")
	writePersonality(t, project, "ghost", "", "## retry.offline\n\n- Boo.\n")
	writePersonality(t, project, PromptsDir, "", "## retry.offline\n\n- Not a personality.\n")
	got, errs := set.WithProject(project)
	if len(errs) != 2 || !strings.Contains(errs[0].Error()+errs[1].Error(), "personality dry ("+filepath.Join(project, Dry)+"): a project may not change a personality's brief") ||
		!strings.Contains(errs[0].Error()+errs[1].Error(), "personality ghost (") {
		t.Fatalf("errors %v", errs)
	}
	dry, err := got.New(Options{Name: Dry})
	if err != nil {
		t.Fatal(err)
	}
	shipped, _ := New(Options{Name: Dry}) //nolint:errcheck // dry is built in
	if dry.Brief() != shipped.Brief() || dry.Text(RetryOffline, nil) != "The project's offline line." {
		t.Fatalf("dry: brief %q, offline %q", dry.Brief(), dry.Text(RetryOffline, nil))
	}
	var d Deck
	if line, ok := d.Deal(dry, SessionGoodbye, nil); !ok || line != "The user's goodbye." {
		t.Fatalf("the user's other keys are lost: %q", line)
	}
	pirate, err := got.New(Options{Name: "pirate"})
	if err != nil || pirate.Text(RetryOffline, nil) != "Arr, the network." || !strings.Contains(pirate.Brief(), "pirate") {
		t.Fatalf("pirate: %v", err)
	}
	if got.Dir(Dry) != filepath.Join(project, Dry) || !slices.Equal(got.Names(), set.Names()) {
		t.Fatalf("dir %q, names %v", got.Dir(Dry), got.Names())
	}
	before, _ := set.New(Options{Name: Dry}) //nolint:errcheck // dry is built in
	if before.Text(RetryOffline, nil) != "The user's offline line." || set.Dir(Dry) != filepath.Join(user, Dry) {
		t.Fatal("the user's set changed")
	}
	if same, errs := set.WithProject(filepath.Join(project, "missing")); same != set || errs != nil {
		t.Fatal("a missing directory changed the set")
	}
}
