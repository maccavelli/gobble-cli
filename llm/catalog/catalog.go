package catalog

import (
	"bytes"
	_ "embed" // the snapshot
	"encoding/json/v2"
	"slices"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
)

// snapshot is the nine models.dev sections go-llmprovider-sdk reads, made by
// scripts/gen_catalog.py; source is where and when they came from.
var (
	//go:embed data/models.json
	snapshot []byte
	//go:embed data/source.json
	source []byte
)

// Snapshot is a copy of the embedded models.dev document.
func Snapshot() []byte { return bytes.Clone(snapshot) }

// Source is where the snapshot was fetched from, "file" for a document
// given to the generator, and when.
func Source() (url string, fetched time.Time) {
	var s struct {
		Source  string `json:"source"`
		Fetched string `json:"fetched"`
	}
	if json.Unmarshal(source, &s) != nil {
		return "", time.Time{}
	}
	t, err := time.Parse("2006-01-02T15:04Z", s.Fetched)
	if err != nil {
		return s.Source, time.Time{}
	}
	return s.Source, t
}

// levelOff is the level that asks for no reasoning.
const levelOff = "off"

// levelOrder is Pi's order of thinking levels, which Clamp walks.
var levelOrder = []string{levelOff, "minimal", "low", "medium", "high", "xhigh", "max"}

// unknownLevels are what a model offers when nothing is known of its
// reasoning: the levels gobble offered before the catalog.
var unknownLevels = levelOrder[:6]

// Levels are the thinking levels m offers, by Pi's rule: off alone when it
// does not reason; off to high when it reasons with no effort list; with
// one, off when "none" is in it and each other level it lists. A model whose
// reasoning is unknown offers off to xhigh.
func Levels(m llm.Model) []string {
	switch {
	case !m.ReasoningKnown:
		return slices.Clone(unknownLevels)
	case !m.Reasoning:
		return []string{levelOff}
	}
	listed := func(level string) bool { return slices.Contains(m.Efforts, level) }
	if !listed("none") && !slices.ContainsFunc(levelOrder[1:], listed) {
		return slices.Clone(levelOrder[:5])
	}
	var out []string
	if listed("none") {
		out = append(out, levelOff)
	}
	for _, level := range levelOrder[1:] {
		if listed(level) {
			out = append(out, level)
		}
	}
	return out
}

// Clamp is level when levels offers it, else Pi's clamp: the next higher
// level offered, else the next lower; a level Pi does not name is the first
// offered.
func Clamp(levels []string, level string) string {
	if slices.Contains(levels, level) {
		return level
	}
	first := levelOff
	if len(levels) > 0 {
		first = levels[0]
	}
	i := slices.Index(levelOrder, level)
	if i < 0 {
		return first
	}
	for _, l := range levelOrder[i:] {
		if slices.Contains(levels, l) {
			return l
		}
	}
	for j := i - 1; j >= 0; j-- {
		if slices.Contains(levels, levelOrder[j]) {
			return levelOrder[j]
		}
	}
	return first
}

// Charge is a turn's cost in USD, by Pi's categories.
type Charge struct {
	Input, Output, CacheRead, CacheWrite, Total float64
}

// Price is the cost of a model call by Pi's rule: each category's tokens at
// its rate per million, the rates of the highest tier whose Above the input,
// cache reads and writes included, exceeds. input is the uncached input. It
// is zero when m has no price.
func Price(m llm.Model, input, output, cacheRead, cacheWrite int64) Charge {
	if !m.Priced {
		return Charge{}
	}
	c := m.Cost
	in, out, read, write := c.Input, c.Output, c.CacheRead, c.CacheWrite
	for _, t := range c.Tiers {
		if input+cacheRead+cacheWrite > int64(t.Above) {
			in, out, read, write = t.Input, t.Output, t.CacheRead, t.CacheWrite
		}
	}
	ch := Charge{Input: in * float64(input) / 1e6, Output: out * float64(output) / 1e6,
		CacheRead: read * float64(cacheRead) / 1e6, CacheWrite: write * float64(cacheWrite) / 1e6}
	ch.Total = ch.Input + ch.Output + ch.CacheRead + ch.CacheWrite
	return ch
}
