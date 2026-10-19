package personality

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writePersonality(t *testing.T, dir, name, brief, messages string) {
	t.Helper()
	d := filepath.Join(dir, name)
	if err := os.MkdirAll(d, 0o700); err != nil {
		t.Fatal(err)
	}
	if brief != "" {
		if err := os.WriteFile(filepath.Join(d, "personality.md"), []byte(brief), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if messages != "" {
		if err := os.WriteFile(filepath.Join(d, "messages.md"), []byte(messages), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// A user's personality loads beside the built-ins, with its level and
// summary, and speaks its own messages, boring's for the rest; one that
// does not make a personality is left out with an error naming it
// (0022-PLAN P3 step 6).
func TestLoadSet(t *testing.T) {
	dir := t.TempDir()
	writePersonality(t, dir, "pirate", "---\nname: pirate\nlevel: 2\nsummary: Arr.\n---\nYou talk like a pirate.\n",
		"## retry.offline\n\n- The sea is calm, and so is the network.\n")
	writePersonality(t, dir, "quiet", "---\nname: quiet\nlevel: 1\nsummary: Hush.\n---\nYou say little.\n", "")
	bad := map[string][2]string{
		"misnamed": {"---\nname: other\nlevel: 2\n---\nx\n", ""},
		"dry":      {"---\nname: dry\nlevel: 3\n---\nx\n", ""},
		"nokey":    {"---\nname: nokey\nlevel: 2\n---\nx\n", "## no.such.key\n\n- x\n"},
		"facts":    {"---\nname: facts\nlevel: 2\n---\nx\n", "## retry.empty_reply\n\n- Again, {attempt} of many.\n"},
		"nobrief":  {"", "## retry.offline\n\n- x\n"},
	}
	for name, files := range bad {
		writePersonality(t, dir, name, files[0], files[1])
	}
	set, errs := LoadSet(dir)
	if len(errs) != len(bad) {
		t.Fatalf("%d errors %v, want %d", len(errs), errs, len(bad))
	}
	for name := range bad {
		if !slices.ContainsFunc(errs, func(e error) bool { return strings.Contains(e.Error(), "personality "+name+" (") }) {
			t.Errorf("no error names %s: %v", name, errs)
		}
	}
	if got := set.Names(); !slices.Equal(got, []string{Boring, Dry, Snarky, Bombastic, "pirate", "quiet"}) {
		t.Fatalf("names %v", got)
	}
	if set.Level("pirate") != 2 || set.Level("plain") != 1 || set.Level("bombastic") != 5 || set.Level("nobody") != 0 {
		t.Fatal("levels")
	}
	v, err := set.New(Options{Name: "pirate"})
	if err != nil || v.Level() != 2 || v.Brief() != "You talk like a pirate." || v.Rules() == "" {
		t.Fatalf("pirate: %+v, %v", v, err)
	}
	if got := v.Text(RetryOffline, nil); got != "The sea is calm, and so is the network." {
		t.Fatalf("pirate's own message %q", got)
	}
	if got := v.Text(ModeAlreadyPlan, nil); got != "Already in plan mode." {
		t.Fatalf("pirate's fallback %q", got)
	}
	infos := set.Infos()
	if len(infos) != 6 || infos[0] != (Info{Name: Boring, Level: 1, Summary: BoringSummary, Builtin: true}) ||
		infos[4] != (Info{Name: "pirate", Level: 2, Summary: "Arr."}) {
		t.Fatalf("infos %+v", infos)
	}
	if _, err := (*Set)(nil).New(Options{Name: "pirate"}); err == nil {
		t.Fatal("a nil set knew a user's personality")
	}
	if set, errs := LoadSet(filepath.Join(dir, "none")); len(errs) != 0 || len(set.Names()) != len(Builtins) {
		t.Fatal("a missing directory is no personalities")
	}
}
