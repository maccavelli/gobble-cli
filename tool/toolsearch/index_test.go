package toolsearch

import (
	"encoding/json/jsontext"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

func names(specs []tool.Spec) []string {
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Name
	}
	return out
}

// The same word counts most in the name, then in the first sentence, then
// in the rest of the description or a property (0012-MADR D11).
func TestFieldWeights(t *testing.T) {
	specs := []tool.Spec{
		{Name: "rest_tool", Description: "Does something. Mentions widget later."},
		{Name: "first_tool", Description: "Handles a widget. Then more."},
		{Name: "widget_tool", Description: "Does something. Then more."},
		{Name: "prop_tool", Description: "Does something.", InputSchema: jsontext.Value(`{"type":"object","properties":{"widget":{"type":"string","description":"x"}}}`)},
	}
	got := names(New(specs).Search("widget", 10))
	if got[0] != "widget_tool" || got[1] != "first_tool" || !slices.Contains(got[2:], "rest_tool") || !slices.Contains(got[2:], "prop_tool") || len(got) != 4 {
		t.Fatalf("ranking %v; want widget_tool, first_tool, then rest_tool and prop_tool", got)
	}
}

// A spec named by the whole query comes first, even against a spec that
// matches its words more often; stop words match nothing; hidden specs
// are never found.
func TestExactNameStopWordsHidden(t *testing.T) {
	specs := []tool.Spec{
		{Name: "list_files", Description: "List files, files, files, in a directory of files."},
		{Name: "files", Description: "Something else."},
		{Name: "secret_files", Description: "Files.", Exposure: tool.ExposureHidden},
	}
	ix := New(specs)
	if got := names(ix.Search("files", 10)); len(got) != 2 || got[0] != "files" {
		t.Fatalf("search files = %v; want the exact name first and the hidden spec absent", got)
	}
	if got := ix.Search("the of and", 10); len(got) != 0 {
		t.Fatalf("a query of stop words found %v", names(got))
	}
	if got := ix.Search("secret", 10); len(got) != 0 {
		t.Fatalf("a hidden spec was found: %v", names(got))
	}
	if got := ix.Search("files", 1); len(got) != 1 || got[0].Name != "files" {
		t.Fatalf("limit 1 = %v", names(got))
	}
	if got := Tokens("Read_file, then grep-for 2 items; the END."); !slices.Equal(got, []string{"read", "file", "then", "grep", "2", "items", "end"}) {
		t.Fatalf("tokens %v", got)
	}
}

// k1 saturates repeated terms, and b normalizes for length, as BM25 says:
// with b 0 a long document scores as a short one with the same count, and
// with b 0.75 it scores less; a larger k1 lets a repeated term count more.
func TestK1AndB(t *testing.T) {
	short := tool.Spec{Name: "a", Description: "zebra"}
	long := tool.Spec{Name: "b", Description: "zebra one two three four five six seven eight"}
	other := tool.Spec{Name: "c", Description: "other"}
	terms := []string{"zebra"}
	flat := newIndex([]tool.Spec{short, long, other}, K1, 0)
	if s, l := flat.score(0, terms), flat.score(1, terms); s != l {
		t.Fatalf("with b 0, short %v and long %v differ", s, l)
	}
	norm := newIndex([]tool.Spec{short, long, other}, K1, B)
	if s, l := norm.score(0, terms), norm.score(1, terms); s <= l {
		t.Fatalf("with b %v, short %v does not beat long %v", B, s, l)
	}
	once := tool.Spec{Name: "d", Description: "yak"}
	twice := tool.Spec{Name: "e", Description: "yak yak"}
	ratio := func(k1 float64) float64 {
		ix := newIndex([]tool.Spec{once, twice, other}, k1, 0)
		return ix.score(1, []string{"yak"}) / ix.score(0, []string{"yak"})
	}
	if lo, hi := ratio(0.5), ratio(K1); lo >= hi || hi >= 2 {
		t.Fatalf("twice/once ratio %v at k1 0.5 and %v at k1 %v; want it to grow with k1 and stay under 2", lo, hi, K1)
	}
}
