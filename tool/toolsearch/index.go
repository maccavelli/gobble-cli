package toolsearch

import (
	"cmp"
	"encoding/json/v2"
	"maps"
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/maccavelli/gobble-cli/tool"
)

// BM25's parameters, as Pi's ranker sets them (0012-MADR D11).
const (
	K1 = 1.2
	B  = 0.75
)

// The weight each field's words count with (0012-MADR D11).
const (
	weightName     = 3 // the name, split on "_"
	weightFirst    = 2 // the description's first sentence
	weightRest     = 1 // the rest of the description
	weightProperty = 1 // the schema's property names and descriptions
)

// stopWords are dropped from documents and queries: Pi's list.
var stopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true, "by": true,
	"for": true, "from": true, "in": true, "is": true, "it": true, "of": true, "on": true, "or": true,
	"that": true, "the": true, "this": true, "to": true, "with": true,
}

// Index ranks tool specs for a query with BM25 over weighted fields. A
// hidden spec is not indexed.
type Index struct {
	specs  []tool.Spec
	docs   []doc
	avgLen float64
	k1, b  float64
}

// doc is one spec's weighted term counts and length.
type doc struct {
	tf  map[string]float64
	len float64
}

// New indexes specs, in order, with K1 and B.
func New(specs []tool.Spec) *Index {
	return newIndex(specs, K1, B)
}

func newIndex(specs []tool.Spec, k1, b float64) *Index {
	ix := &Index{k1: k1, b: b}
	total := 0.0
	for _, s := range specs {
		if s.Exposure == tool.ExposureHidden {
			continue
		}
		d := document(s)
		ix.specs = append(ix.specs, s)
		ix.docs = append(ix.docs, d)
		total += d.len
	}
	if len(ix.docs) > 0 {
		ix.avgLen = total / float64(len(ix.docs))
	}
	if ix.avgLen == 0 {
		ix.avgLen = 1
	}
	return ix
}

// Search is up to limit specs whose fields hold a word of query, best
// first; ties keep index order. A spec whose name is the whole query comes
// first, matched or not.
func (ix *Index) Search(query string, limit int) []tool.Spec {
	if limit <= 0 {
		return nil
	}
	exact := strings.TrimSpace(query)
	terms := slices.Compact(slices.Sorted(slices.Values(Tokens(query))))
	type hit struct {
		i     int
		score float64
	}
	var hits []hit
	for i := range ix.docs {
		score := ix.score(i, terms)
		if strings.EqualFold(ix.specs[i].Name, exact) {
			score = math.Inf(1)
		}
		if score > 0 {
			hits = append(hits, hit{i, score})
		}
	}
	slices.SortStableFunc(hits, func(x, y hit) int { return cmp.Compare(y.score, x.score) })
	out := make([]tool.Spec, 0, min(limit, len(hits)))
	for _, h := range hits[:min(limit, len(hits))] {
		out = append(out, ix.specs[h.i])
	}
	return out
}

// score is doc i's BM25 score for terms.
func (ix *Index) score(i int, terms []string) float64 {
	d := ix.docs[i]
	n := float64(len(ix.docs))
	score := 0.0
	for _, t := range terms {
		tf := d.tf[t]
		if tf == 0 {
			continue
		}
		df := 0.0
		for _, o := range ix.docs {
			if o.tf[t] > 0 {
				df++
			}
		}
		idf := math.Log(1 + (n-df+0.5)/(df+0.5))
		norm := ix.k1 * (1 - ix.b + ix.b*d.len/ix.avgLen)
		score += idf * tf * (ix.k1 + 1) / (tf + norm)
	}
	return score
}

// document is a spec's weighted terms: each word counts with its field's
// weight, and the length is the sum of those weights.
func document(s tool.Spec) doc {
	d := doc{tf: map[string]float64{}}
	add := func(text string, w float64) {
		for _, t := range Tokens(text) {
			d.tf[t] += w
			d.len += w
		}
	}
	add(strings.ReplaceAll(s.Name, "_", " "), weightName)
	first, rest := firstSentence(s.Description)
	add(first, weightFirst)
	add(rest, weightRest)
	for _, p := range properties(s.InputSchema) {
		add(p, weightProperty)
	}
	return d
}

// firstSentence splits a description after its first sentence: the text up
// to the first ". " or line break, or all of it.
func firstSentence(desc string) (first, rest string) {
	desc = strings.TrimSpace(desc)
	end := len(desc)
	if i := strings.Index(desc, ". "); i >= 0 {
		end = i + 1
	}
	if i := strings.IndexByte(desc, '\n'); i >= 0 && i < end {
		end = i
	}
	return desc[:end], desc[end:]
}

// properties are a schema's property names and descriptions, at every
// depth of properties and items.
func properties(schema []byte) []string {
	var root map[string]any
	if len(schema) == 0 || json.Unmarshal(schema, &root) != nil {
		return nil
	}
	var out []string
	var walk func(node map[string]any, top bool)
	walk = func(node map[string]any, top bool) {
		if d, ok := node["description"].(string); ok && !top {
			out = append(out, d)
		}
		if props, ok := node["properties"].(map[string]any); ok {
			for _, name := range slices.Sorted(maps.Keys(props)) {
				out = append(out, name)
				if p, ok := props[name].(map[string]any); ok {
					walk(p, false)
				}
			}
		}
		if items, ok := node["items"].(map[string]any); ok {
			walk(items, false)
		}
	}
	walk(root, true)
	return out
}

// Tokens are text's lower-cased words, split on Unicode letter and digit
// boundaries, without stop words.
func Tokens(text string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if !stopWords[w] {
			out = append(out, w)
		}
	}
	return out
}
