package personality

import (
	"slices"
	"strings"
	"testing"
)

// A catalog is keys under level-two headings, with variants as bullets or
// paragraphs; text before the first key, or after another heading, is
// documentation (0022-MADR D5).
func TestParseCatalog(t *testing.T) {
	c, err := ParseCatalog([]byte(`---
name: x
---
# Title

Documentation, not a variant.

## one

- first
- second, which
  goes on
- third

## two

A paragraph
across lines.

# A section

Also documentation.

## three

only
`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[Key][]string{
		"one":   {"first", "second, which goes on", "third"},
		"two":   {"A paragraph across lines."},
		"three": {"only"},
	}
	if got := c.Keys(); !slices.Equal(got, []Key{"one", "three", "two"}) {
		t.Fatalf("keys %v", got)
	}
	for k, vs := range want {
		if got := c.Variants(k); !slices.Equal(got, vs) {
			t.Errorf("%s: %q, want %q", k, got, vs)
		}
	}
	if got := (*Catalog)(nil).Variants("one"); got != nil {
		t.Fatalf("a nil catalog's variants %q", got)
	}
}

// A malformed catalog is an error naming the line.
func TestParseCatalogErrors(t *testing.T) {
	for _, c := range []struct{ name, text, want string }{
		{"a key twice", "## a\n\nx\n\n## a\n\ny\n", "line 5: key a is given twice"},
		{"no variant before the next key", "## a\n\n## b\n\ny\n", "line 1: key a has no variant"},
		{"no variant before a heading", "## a\n\n# doc\n", "line 1: key a has no variant"},
		{"no variant at the end", "## a\n\nx\n\n## b\n\ny\n\n## c\n", "line 9: key c has no variant"},
		{"front matter not closed", "---\nname: x\n\n## a\n\nx\n", "does not close"},
	} {
		if _, err := ParseCatalog([]byte(c.text)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}

// The facts rule: a variant carries exactly the boring text's
// placeholders; doubled braces are literal.
func TestCheckPlaceholders(t *testing.T) {
	boring := "retrying: {cause} (attempt {attempt} of {max})"
	for _, c := range []struct {
		variant string
		ok      bool
	}{
		{"Asking again, {cause}, {attempt} of {max}.", true},
		{"{max} at most; this is {attempt}, for {cause}.", true},
		{"Asking again ({attempt} of {max}).", false},
		{"Asking again ({attempt} of {max}), {cause}, {when}.", false},
		{"{{cause}} {attempt} of {max}", false},
	} {
		if err := CheckPlaceholders(boring, c.variant); (err == nil) != c.ok {
			t.Errorf("%q: %v, want ok %v", c.variant, err, c.ok)
		}
	}
	if got := Placeholders("a {x} {{y}} {x} {z}"); !slices.Equal(got, []string{"x", "z"}) {
		t.Fatalf("placeholders %q", got)
	}
}

// fill puts the values in; a missing one shows as its placeholder;
// doubled braces are single.
func TestFill(t *testing.T) {
	if got := fill("{a} and {b}, {{literal}}", Args{"a": "1"}); got != "1 and {b}, {literal}" {
		t.Fatalf("fill %q", got)
	}
}

// The built-in boring catalog has every key, and nothing else, one
// variant each (0022-PLAN P1).
func TestBoringCatalog(t *testing.T) {
	c := boringCat()
	sorted := slices.Sorted(slices.Values(Keys))
	if got := c.Keys(); !slices.Equal(got, sorted) {
		var missing, extra []Key
		for _, k := range sorted {
			if !slices.Contains(got, k) {
				missing = append(missing, k)
			}
		}
		for _, k := range got {
			if !slices.Contains(sorted, k) {
				extra = append(extra, k)
			}
		}
		t.Fatalf("boring catalog: missing %v, extra %v", missing, extra)
	}
	for _, k := range Keys {
		if vs := c.Variants(k); len(vs) != 1 {
			t.Errorf("%s: %d variants, want 1", k, len(vs))
		}
	}
}

// boring has three names; a nil Voice is boring; an unknown name is
// refused.
func TestNew(t *testing.T) {
	for _, name := range []string{"", Boring, Plain, Normal} {
		v, err := New(Options{Name: name})
		if err != nil || v.Name() != Boring || v.Text(RetryOffline, nil) != "waiting for the network" {
			t.Errorf("%q: %v, %v", name, v.Name(), err)
		}
	}
	var none *Voice
	if none.Name() != Boring || none.Text(ModelSet, Args{"model": "m1"}) != "Model: m1" {
		t.Fatalf("nil voice: %q", none.Text(ModelSet, Args{"model": "m1"}))
	}
	if _, err := New(Options{Name: "zany"}); err == nil {
		t.Fatal("an unknown personality was taken")
	}
	if got := none.Text("no.such.key", nil); got != "no.such.key" {
		t.Fatalf("an unknown key %q", got)
	}
}

// A key with several variants never gives the same one twice running;
// one seed gives one sequence; a key the personality leaves out falls
// back to boring's.
func TestVariety(t *testing.T) {
	own, err := ParseCatalog([]byte("## retry.offline\n\n- a {x}\n- b {x}\n- c {x}\n"))
	if err != nil {
		t.Fatal(err)
	}
	pick := func(seed uint64, n int) []string {
		v, err := New(Options{Seed: seed})
		if err != nil {
			t.Fatal(err)
		}
		v.own = own
		var out []string
		for range n {
			out = append(out, v.Text(RetryOffline, Args{"x": "!"}))
		}
		return out
	}
	got := pick(7, 300)
	seen := map[string]bool{}
	for i, s := range got {
		seen[s] = true
		if i > 0 && s == got[i-1] {
			t.Fatalf("pick %d repeats %q", i, s)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("variants used %v, want all three", seen)
	}
	if !slices.Equal(got, pick(7, 300)) || slices.Equal(got, pick(8, 300)) {
		t.Fatal("a seed did not fix the sequence, or two seeds gave one")
	}
	v, _ := New(Options{Seed: 1}) //nolint:errcheck // boring always builds
	v.own = own
	if got := v.Text(ModeAlreadyPlan, nil); got != "Already in plan mode." {
		t.Fatalf("fallback %q", got)
	}
}

// A brief is front matter with a name, a level of 1 to 5 and a summary,
// then prose (0022-MADR D4).
func TestParseBrief(t *testing.T) {
	b, err := ParseBrief([]byte("---\nname: dry\nlevel: 3\nsummary: Polite.\n---\n# Dry\n\nYou are courteous.\n"))
	if err != nil || b.Name != "dry" || b.Level != 3 || b.Summary != "Polite." || b.Body != "# Dry\n\nYou are courteous." {
		t.Fatalf("%+v, %v", b, err)
	}
	for _, c := range []struct{ name, text, want string }{
		{"no front matter", "# Dry\n", "no front matter"},
		{"not closed", "---\nname: x\nlevel: 3\n", "does not close"},
		{"no name", "---\nlevel: 3\n---\nx\n", "no name"},
		{"level 0", "---\nname: x\nlevel: 0\n---\nx\n", "level 0 is not 1 to 5"},
		{"level 6", "---\nname: x\nlevel: 6\n---\nx\n", "level 6 is not 1 to 5"},
		{"empty", "---\nname: x\nlevel: 3\n---\n\n", "the brief is empty"},
	} {
		if _, err := ParseBrief([]byte(c.text)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}
