package compaction

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/session"
)

// Settings are Pi's compaction settings, and the two thresholds gobble
// adds (0005-PLAN F5-2 decision 1). ReserveTokens leaves room for the
// reply in the automatic trigger, and sets a summary's output cap;
// KeepRecentTokens is the recent context a compaction keeps;
// ThresholdPercent, a percentage of the window, and ThresholdTokens, a
// count of tokens, compact sooner, each 0 being off.
type Settings struct {
	ReserveTokens    int
	KeepRecentTokens int
	ThresholdPercent int
	ThresholdTokens  int
}

// Over reports whether a context of used tokens is to be compacted before
// a model call (0005-PLAN F5-2 decision 1): over the window less
// ReserveTokens (Pi), at ThresholdPercent of the window (Kilo), or at
// ThresholdTokens (goose, Codex). A window of 0, unknown, triggers only
// ThresholdTokens.
func Over(used, window int, s Settings) bool {
	if window > 0 {
		if used > window-s.ReserveTokens {
			return true
		}
		if s.ThresholdPercent > 0 && used >= window*s.ThresholdPercent/100 {
			return true
		}
	}
	return s.ThresholdTokens > 0 && used >= s.ThresholdTokens
}

// DefaultSettings are Pi's defaults (compaction.ts DEFAULT_COMPACTION_SETTINGS).
var DefaultSettings = Settings{ReserveTokens: 16384, KeepRecentTokens: 20000}

// The errors of Prepare are Pi's messages (agent-session.ts compact).
var (
	//lint:ignore ST1005 Pi's message, word for word (0002-PLAN Phase 5, deviation 3, applied again in Phase 6)
	ErrAlreadyCompacted = errors.New("Already compacted") //nolint:staticcheck // as the lint:ignore above
	//lint:ignore ST1005 Pi's message, word for word (0002-PLAN Phase 5, deviation 3, applied again in Phase 6)
	ErrNothingToCompact = errors.New("Nothing to compact (session too small)") //nolint:staticcheck // as the lint:ignore above
)

// ErrSummaryRefused is wrapped by every summary the model gave that is not
// taken: cut at the output limit, calling a tool, empty, or too short.
// Asking again with the same history and model cannot be expected to do
// better (0005-PLAN F5-2 decision 11).
var ErrSummaryRefused = errors.New("compaction: summary refused")

// refused is a summary's refusal: its message, wrapping ErrSummaryRefused.
type refused string

func (r refused) Error() string { return string(r) }
func (refused) Unwrap() error   { return ErrSummaryRefused }

// The summary floor (0005-PLAN F5-2 decision 6): grok-build's
// MIN_SUMMARY_SEED_CHARS, applied only to a summary of floorSource
// characters or more, so a short history's short summary stands.
const (
	floorChars  = 500
	floorSource = 5000
)

// Preparation is what a compaction summarises and keeps.
type Preparation struct {
	// FirstKeptEntryID is where the kept context starts.
	FirstKeptEntryID string
	toSummarize      []*session.Message
	turnPrefix       []*session.Message
	// SplitTurn reports a cut inside one turn, whose start is summarised
	// separately.
	SplitTurn bool
	// TokensBefore is the context's estimate before compaction.
	TokensBefore    int64
	previousSummary string
	files           fileOps
	reserve         int
}

// Prepare finds what a compaction of path summarises (Pi's
// prepareCompaction): the context from the previous compaction's kept
// range, or the start, up to a cut point that keeps KeepRecentTokens.
func Prepare(path []session.Entry, s Settings) (*Preparation, error) {
	if n := len(path); n > 0 && path[n-1].Type == session.TypeCompaction {
		return nil, ErrAlreadyCompacted
	}
	proj := Project(Context(path))
	start, previous := 0, ""
	prev := -1
	if len(proj) > 0 && proj[0].Entry.Type == session.TypeCompaction && len(proj[0].Messages) > 0 {
		prev, start, previous = 0, 1, proj[0].Entry.Summary
	}
	cut := findCut(proj, start, len(proj), s.KeepRecentTokens)
	if cut.first >= len(proj) || proj[cut.first].Entry.ID == "" {
		return nil, ErrNothingToCompact
	}
	historyEnd := cut.first
	if cut.split {
		historyEnd = cut.turnStart
	}
	p := &Preparation{
		FirstKeptEntryID: proj[cut.first].Entry.ID,
		toSummarize:      messages(proj[start:historyEnd]),
		SplitTurn:        cut.split,
		TokensBefore:     int64(EstimateEntries(path)),
		previousSummary:  previous,
		files:            newFileOps(),
		reserve:          s.ReserveTokens,
	}
	if cut.split {
		p.turnPrefix = messages(proj[cut.turnStart:cut.first])
	}
	if len(p.toSummarize) == 0 && len(p.turnPrefix) == 0 {
		return nil, ErrNothingToCompact
	}
	if prev >= 0 {
		p.files.carry(proj[prev].Entry)
	}
	for _, m := range p.toSummarize {
		p.files.add(m)
	}
	for _, m := range p.turnPrefix {
		p.files.add(m)
	}
	return p, nil
}

// carry adds a previous gobble- or Pi-generated compaction's file lists.
func (f fileOps) carry(c session.Entry) {
	if (c.FromHook != nil && *c.FromHook) || len(c.Details) == 0 {
		return
	}
	var d Details
	if json.Unmarshal(c.Details, &d) != nil {
		return
	}
	for _, p := range d.ReadFiles {
		f.read[p] = true
	}
	for _, p := range d.ModifiedFiles {
		f.edited[p] = true
	}
}

// messages are the entries' messages, without compaction summaries.
func messages(ps []Projected) []*session.Message {
	var out []*session.Message
	for _, p := range ps {
		if p.Entry.Type == session.TypeCompaction {
			continue
		}
		out = append(out, p.Messages...)
	}
	return out
}

type cutResult struct {
	first, turnStart int
	split            bool
}

func isCutEntry(p Projected) bool {
	return p.Entry.Type != session.TypeCompaction && slices.ContainsFunc(p.Messages, cutPoint)
}

func isTurnStartEntry(p Projected) bool {
	return p.Entry.Type != session.TypeCompaction && slices.ContainsFunc(p.Messages, turnStart)
}

// findCut is Pi's findProjectedCutPoint: walk back from the newest entry
// until keep tokens are reached, then cut at the closest valid point at or
// after it, never at a tool result; move past an omitted recovery suffix;
// step back over context-invisible entries; and say whether the cut splits
// a turn.
func findCut(proj []Projected, start, end, keep int) cutResult {
	var points []int
	for i := start; i < end; i++ {
		if isCutEntry(proj[i]) {
			points = append(points, i)
		}
	}
	if len(points) == 0 {
		return cutResult{first: start, turnStart: -1}
	}
	cut, acc, exceeded := points[0], 0, false
	for i := end - 1; i >= start; i-- {
		t := entryTokens(proj[i])
		if t == 0 {
			continue
		}
		acc += t
		if acc >= keep {
			exceeded = true
			cut = points[len(points)-1]
			for _, c := range points {
				if c >= i {
					cut = c
					break
				}
			}
			break
		}
	}
	if exceeded && recoverySuffix(proj[cut+1:end]) {
		cut++
	}
	for cut > start {
		p := proj[cut-1]
		if p.Entry.Type == session.TypeCompaction || len(p.Messages) > 0 {
			break
		}
		cut--
	}
	if isTurnStartEntry(proj[cut]) {
		return cutResult{first: cut, turnStart: -1}
	}
	for i := cut; i >= start; i-- {
		if isTurnStartEntry(proj[i]) {
			return cutResult{first: cut, turnStart: i, split: true}
		}
	}
	return cutResult{first: cut, turnStart: -1}
}

// recoverySuffix reports the entries after a cut as a recovery attempt and
// its omission edits, which are context-invisible after the last visible
// input (Pi's isRecoveryOmissionSuffix, compaction.ts:832-850): an omitted
// assistant message, no replacement of an entry that is not omitted, no
// compaction, and every visible entry omitted. The cut then moves past it,
// and never past input not yet sent.
func recoverySuffix(suffix []Projected) bool {
	visible := func(p Projected) bool {
		return p.Entry.Type != session.TypeContextEdit && len(entryMessages(p.Entry)) > 0
	}
	omitted := func(p Projected) bool { return visible(p) && len(p.Messages) == 0 }
	gone := map[string]bool{}
	for _, p := range suffix {
		if omitted(p) {
			gone[p.Entry.ID] = true
		}
	}
	assistant := false
	for _, p := range suffix {
		e := p.Entry
		switch {
		case e.Type == session.TypeContextEdit && !missing(e.Replacement) && !gone[e.TargetID]:
			return false
		case e.Type == session.TypeCompaction, visible(p) && !omitted(p):
			return false
		case e.Type == session.TypeMessage && e.Message != nil && e.Message.Role == session.RoleAssistant && omitted(p):
			assistant = true
		}
	}
	return assistant
}

// Model is the model a summary is asked of: the session's own, or the one
// before a model switch (0005-PLAN F5-2 decision 12). MaxTokens is its
// output limit, 0 unknown.
type Model struct {
	Provider  llm.Provider
	Model     string
	Thinking  string
	MaxTokens int
}

// Result is a compaction to record.
type Result struct {
	Summary          string
	FirstKeptEntryID string
	TokensBefore     int64
	Usage            llm.Usage
	Details          Details
}

// Compact asks the model for the summary (Pi's compact): one summary of
// the history, updating the previous one; for a split turn, a second
// summary of the turn's start, merged; then the file lists.
func Compact(ctx context.Context, p *Preparation, m Model, instructions string) (Result, error) {
	var (
		summary string
		usage   llm.Usage
	)
	if p.SplitTurn && len(p.turnPrefix) > 0 {
		history := p.previousSummary
		if history == "" {
			history = "No prior history."
		}
		if len(p.toSummarize) > 0 {
			conv := serialize(p.toSummarize)
			text, u, err := summarise(ctx, m, historyPrompt(conv, instructions, p.previousSummary), "Summarization",
				replyCap(p.reserve*4/5, m.MaxTokens), utf8.RuneCountInString(conv))
			if err != nil {
				return Result{}, err
			}
			history, usage = text, u
		}
		prefixPrompt := "# Conversation\n" + serialize(p.turnPrefix) + "\n\n# Instructions\n" + turnPrefixPrompt
		text, u, err := summarise(ctx, m, prefixPrompt, "Turn prefix summarization", replyCap(p.reserve/2, m.MaxTokens), 0)
		if err != nil {
			return Result{}, err
		}
		summary = history + "\n\n---\n\n**Turn Context (split turn):**\n\n" + text
		usage = addUsage(usage, u)
	} else {
		conv := serialize(p.toSummarize)
		text, u, err := summarise(ctx, m, historyPrompt(conv, instructions, p.previousSummary), "Summarization",
			replyCap(p.reserve*4/5, m.MaxTokens), utf8.RuneCountInString(conv))
		if err != nil {
			return Result{}, err
		}
		summary, usage = text, u
	}
	d := p.files.lists()
	return Result{Summary: summary + d.format(), FirstKeptEntryID: p.FirstKeptEntryID, TokensBefore: p.TokensBefore, Usage: usage, Details: d}, nil
}

// historyPrompt is Pi's summary request: the serialised conversation, the
// previous summary when there is one, then the first or the update prompt,
// and the user's focus.
func historyPrompt(conv, instructions, previous string) string {
	base := summarizationPrompt
	if previous != "" {
		base = updatePrompt
	}
	if instructions != "" {
		base += "\n\nAdditional focus: " + instructions
	}
	var b strings.Builder
	b.WriteString("<conversation>\n" + conv + "\n</conversation>\n\n")
	if previous != "" {
		b.WriteString("<previous-summary>\n" + previous + "\n</previous-summary>\n\n")
	}
	b.WriteString(base)
	return b.String()
}

// replyCap is a summary's output cap (Pi's, 0005-PLAN F5-2 decision 7):
// share, a part of the reserve, no more than the model's maxTokens. An
// unknown maxTokens, or no reserve, caps nothing.
func replyCap(share, maxTokens int) int {
	if maxTokens <= 0 || share <= 0 {
		return 0
	}
	return min(share, maxTokens)
}

// summarise asks for a summary, capped at maxTokens, of source characters
// of history, and reads the whole reply. A reply cut at the output limit,
// or one that calls a tool, is not a summary. An empty one, or, of
// floorSource characters or more, one shorter than floorChars, is asked for
// once more, then refused (0005-PLAN F5-2 decision 6). The usage is every
// attempt's.
func summarise(ctx context.Context, m Model, prompt, label string, maxTokens, source int) (string, llm.Usage, error) {
	var usage llm.Usage
	for attempt := 1; ; attempt++ {
		text, u, err := summariseOnce(ctx, m, prompt, label, maxTokens)
		usage = addUsage(usage, u)
		if err != nil {
			return "", usage, err
		}
		n := utf8.RuneCountInString(strings.TrimSpace(text))
		switch {
		case n == 0 && attempt == 2:
			return "", usage, refused(label + " failed: the summary is empty")
		case n > 0 && n < floorChars && source >= floorSource && attempt == 2:
			return "", usage, refused(fmt.Sprintf("%s failed: the summary is too short (%d characters)", label, n))
		case n == 0, n < floorChars && source >= floorSource:
			continue
		}
		return text, usage, nil
	}
}

// summariseOnce sends one summary request and reads the whole reply.
func summariseOnce(ctx context.Context, m Model, prompt, label string, maxTokens int) (string, llm.Usage, error) {
	req := &llm.Request{Model: m.Model, Thinking: m.Thinking, MaxTokens: maxTokens, Messages: []llm.Message{
		{Role: llm.RoleSystem, Content: []llm.Content{{Type: llm.ContentText, Text: systemPrompt}}},
		{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: prompt}}},
	}}
	var (
		text  strings.Builder
		usage llm.Usage
		stop  llm.StopReason
		calls bool
	)
	for ev, err := range m.Provider.Stream(ctx, req) {
		if err != nil {
			return "", llm.Usage{}, fmt.Errorf("%s failed: %w", label, err)
		}
		switch ev := ev.(type) {
		case llm.TextDelta:
			text.WriteString(ev.Text)
		case llm.ToolCallStart, llm.ToolCallDone:
			calls = true
		case llm.Usage:
			usage = ev
		case llm.Done:
			stop = ev.StopReason
		}
	}
	switch {
	case stop == llm.StopMaxTokens:
		return "", usage, refused(label + " failed: generation hit the token cap and the summary is incomplete")
	case calls:
		return "", usage, refused(label + " attempted to call a tool")
	}
	return text.String(), usage, nil
}

func addUsage(a, b llm.Usage) llm.Usage {
	return llm.Usage{
		InputTokens: a.InputTokens + b.InputTokens, OutputTokens: a.OutputTokens + b.OutputTokens,
		ReasoningTokens: a.ReasoningTokens + b.ReasoningTokens, CacheReadTokens: a.CacheReadTokens + b.CacheReadTokens,
		CacheWriteTokens: a.CacheWriteTokens + b.CacheWriteTokens, Estimated: a.Estimated || b.Estimated,
	}
}
