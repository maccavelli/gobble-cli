package cli

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// ModelsCmd lists the models gobble knows, or shows one (0005-MADR's CLI
// surface; 0005-PLAN F4-2c).
type ModelsCmd struct {
	List ModelsListCmd `cmd:"" help:"List the models of the providers with a credential."`
	Info ModelsInfoCmd `cmd:"" help:"Show what gobble knows of one model."`
}

// ModelsListCmd is gobble models list, and --list-models.
type ModelsListCmd struct {
	Search   []string `arg:"" optional:"" complete:"none" help:"Search words, each matched against the provider and id."`
	Provider string   `complete:"provider" help:"Only this provider's models, its known ones when it has no credential."`
	All      bool     `help:"Every provider's models, the known ones of those with no credential."`
}

// Run prints the list.
func (c *ModelsListCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	if c.Provider != "" && !slices.Contains(provider.Providers(), c.Provider) {
		return usageErrorf("unknown provider %q; gobble providers lists them", c.Provider)
	}
	return e.printModels(strings.Join(c.Search, " "), c.Provider, c.All)
}

// ModelsInfoCmd is gobble models info.
type ModelsInfoCmd struct {
	Model string `arg:"" complete:"model" help:"The model, as --model takes it: provider/id, an id, or part of one."`
}

// Run resolves the model as --model resolves one and prints its facts.
func (c *ModelsInfoCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	mj := e.modelsJSON()
	res, err := catalog.ResolveModel("", c.Model, "", offlineModels(e.ctx, mj), e.hasCredential)
	if err != nil {
		return usageErrorf("%v", err)
	}
	if res.Warning != "" {
		e.out.Warnf("%s", res.Warning)
	}
	return e.out.Result(modelInfo(provider.ModelInfoContext(e.ctx, res.Model.Provider, "", res.Model.ID, mj)))
}

// printModels prints Pi's --list-models table (cli/list-models.ts) for the
// providers with a credential: each one's offline set and live listing,
// each model once, its facts from the catalog with no request; search's
// words filter it as Pi's fuzzyFilter does. only keeps one provider; all,
// and only for a provider with no credential, add the known models of those
// with none, named in a note after the table; both take Ollama, which takes
// no key (0025-MADR D5).
func (e *runEnv) printModels(search, only string, all bool) error {
	var sels []provider.Selection
	var unkeyed []string
	ids := provider.KeyProviders()
	if only != "" || all {
		ids = provider.Providers()
	}
	for _, p := range ids {
		if only != "" && p != only {
			continue
		}
		if provider.CredentialVar(p) == "" {
			sels = append(sels, provider.Selection{ID: p})
			continue
		}
		if !e.hasCredential(p) {
			if all || only != "" {
				unkeyed = append(unkeyed, p)
			}
			continue
		}
		if sel, err := provider.ChooseWith(e.hasCredential, p); err == nil {
			sels = append(sels, sel)
		}
	}
	if len(sels) == 0 && len(unkeyed) == 0 {
		return e.out.Resultf("No models available. Set one of %s.\n", strings.Join(provider.CredentialVars(), ", "))
	}
	listings := make([]*modelListing, len(sels))
	for i, sel := range sels { // every listing starts before the first is read
		listings[i] = e.modelListing(sel)
		listings[i].start()
	}
	mj := e.modelsJSON()
	var rows []llm.Model
	for i, sel := range sels {
		ids := withModelsJSON(provider.KnownModels(sel.ID), mj, sel.ID)
		if l := listings[i].get(); l.Live {
			for _, m := range l.Models {
				if !slices.Contains(ids, m) {
					ids = append(ids, m)
				}
			}
		}
		for _, id := range ids {
			rows = append(rows, provider.ModelInfoContext(e.ctx, sel.ID, "", id, mj))
		}
	}
	for _, p := range unkeyed {
		for _, id := range withModelsJSON(provider.KnownModels(p), mj, p) {
			rows = append(rows, provider.ModelInfoContext(e.ctx, p, "", id, mj))
		}
	}
	rows = searchModels(rows, search)
	if len(rows) == 0 {
		return e.out.Resultf("No models matching %q\n", search)
	}
	slices.SortFunc(rows, func(a, b llm.Model) int {
		return cmp.Or(catalog.CompareIDs(a.Provider, b.Provider), catalog.CompareIDs(a.ID, b.ID))
	})
	out := modelTable(rows)
	switch {
	case only != "" && len(unkeyed) == 1:
		out = fmt.Appendf(out, "\n%s has no credential: these are the models gobble knows of it. gobble auth login %s sets one.\n", only, only)
	case len(unkeyed) > 0:
		out = fmt.Appendf(out, "\nNo credential, the models gobble knows: %s.\n", strings.Join(unkeyed, ", "))
	}
	return e.out.Result(out)
}

// modelTable is Pi's columns, two spaces apart, each as wide as its widest
// cell; a fact the catalog does not know is "-", and no line ends in spaces.
func modelTable(rows []llm.Model) []byte {
	cells := [][]string{{labelProvider, labelModel, "context", "max-out", "thinking", "images"}}
	for _, m := range rows {
		cells = append(cells, []string{m.Provider, m.ID, tokenCount(m.ContextWindow), tokenCount(m.MaxTokens),
			yesNo(m.ReasoningKnown, m.Reasoning), yesNo(m.Input != nil, slices.Contains(m.Input, "image"))})
	}
	widths := make([]int, len(cells[0]))
	for _, row := range cells {
		for i, c := range row {
			widths[i] = max(widths[i], utf8.RuneCountInString(c))
		}
	}
	var b strings.Builder
	for _, row := range cells {
		var line strings.Builder
		for i, c := range row {
			if i > 0 {
				line.WriteString("  ")
			}
			fmt.Fprintf(&line, "%-*s", widths[i], c)
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// tokenCount is Pi's formatTokenCount: 200K, 1.5M; "-" when unknown.
func tokenCount(n int) string {
	switch {
	case n <= 0:
		return "-"
	case n >= 1_000_000:
		return scaledCount(n, 1_000_000, "M")
	case n >= 1_000:
		return scaledCount(n, 1_000, "K")
	}
	return strconv.Itoa(n)
}

func scaledCount(n, unit int, suffix string) string {
	if n%unit == 0 {
		return strconv.Itoa(n/unit) + suffix
	}
	return strconv.FormatFloat(float64(n)/float64(unit), 'f', 1, 64) + suffix
}

func yesNo(known, v bool) string {
	switch {
	case !known:
		return "-"
	case v:
		return "yes"
	}
	return "no"
}

// searchSplit splits a search into its words, as Pi's fuzzyFilter does.
var searchSplit = regexp.MustCompile(`[\s/]+`)

// searchModels are the rows whose "<provider> <id>" matches every word of
// search, by Pi's fuzzyMatch (tui/src/fuzzy.ts); every row for no words.
func searchModels(rows []llm.Model, search string) []llm.Model {
	words := slices.DeleteFunc(searchSplit.Split(strings.TrimSpace(search), -1), func(w string) bool { return w == "" })
	if len(words) == 0 {
		return rows
	}
	return slices.DeleteFunc(slices.Clone(rows), func(m llm.Model) bool {
		text := m.Provider + " " + m.ID
		return slices.ContainsFunc(words, func(w string) bool { return !fuzzyMatch(w, text) })
	})
}

// The two shapes whose letters and digits Pi's fuzzyMatch also tries
// swapped.
var (
	lettersDigits = regexp.MustCompile(`^([a-z]+)(\d+)$`)
	digitsLetters = regexp.MustCompile(`^(\d+)([a-z]+)$`)
)

// fuzzyMatch reports whether query's characters appear in text in order,
// without case, or do with its letters and digits swapped, as Pi's
// fuzzyMatch decides a match.
func fuzzyMatch(query, text string) bool {
	q, t := strings.ToLower(query), strings.ToLower(text)
	if subsequence(q, t) {
		return true
	}
	for _, re := range []*regexp.Regexp{lettersDigits, digitsLetters} {
		if m := re.FindStringSubmatch(q); m != nil {
			return subsequence(m[2]+m[1], t)
		}
	}
	return false
}

func subsequence(q, t string) bool {
	rest := q
	for _, r := range t {
		if rest == "" {
			break
		}
		if first, size := utf8.DecodeRuneInString(rest); r == first {
			rest = rest[size:]
		}
	}
	return rest == ""
}

// modelInfo is a model's facts, one aligned line each, "unknown" for what
// the catalog does not know.
func modelInfo(m llm.Model) []byte {
	lines := [][2]string{
		{labelProvider, m.Provider}, {labelModel, m.ID}, {"name", cmp.Or(m.Name, unknownFact)},
		{"context", countOrUnknown(m.ContextWindow)}, {"max-out", countOrUnknown(m.MaxTokens)},
		{"input", listOrUnknown(m.Input)}, {"thinking", strings.Join(catalog.Levels(m), ", ")},
		{"cost", unknownFact},
	}
	if m.Priced {
		c := m.Cost
		lines[len(lines)-1][1] = rates(c.Input, c.Output, c.CacheRead, c.CacheWrite) + ", per million tokens"
		for _, t := range c.Tiers {
			lines = append(lines, [2]string{"tier", fmt.Sprintf("above %d: %s", t.Above, rates(t.Input, t.Output, t.CacheRead, t.CacheWrite))})
		}
	}
	var b strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&b, "%-8s  %s\n", l[0], l[1])
	}
	return []byte(b.String())
}

// The labels of the provider and model, as the list's columns and info's
// lines name them.
const (
	labelProvider = "provider"
	labelModel    = "model"
)

// unknownFact is what models info prints for a fact the catalog lacks.
const unknownFact = "unknown"

func countOrUnknown(n int) string {
	if n <= 0 {
		return unknownFact
	}
	return strconv.Itoa(n)
}

func listOrUnknown(v []string) string {
	if v == nil {
		return unknownFact
	}
	return strings.Join(v, ", ")
}

func rates(input, output, cacheRead, cacheWrite float64) string {
	usd := func(v float64) string { return "$" + strconv.FormatFloat(v, 'f', -1, 64) }
	return "input " + usd(input) + ", output " + usd(output) + ", cache read " + usd(cacheRead) + ", cache write " + usd(cacheWrite)
}
