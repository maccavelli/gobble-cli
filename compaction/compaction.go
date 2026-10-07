package compaction

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/session"
)

// Settings are Pi's compaction settings. ReserveTokens leaves room for the
// reply in the automatic trigger (0005-PLAN F5); KeepRecentTokens is the
// recent context a compaction keeps.
type Settings struct {
	ReserveTokens    int
	KeepRecentTokens int
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

// Preparation is what a compaction summarises and keeps.
type Preparation struct {
	// FirstKeptEntryID is where the kept context starts.
	FirstKeptEntryID string
	toSummarize      []msg
	turnPrefix       []msg
	// SplitTurn reports a cut inside one turn, whose start is summarised
	// separately.
	SplitTurn bool
	// TokensBefore is the context's estimate before compaction.
	TokensBefore    int64
	previousSummary string
	files           fileOps
}

// Prepare finds what a compaction of path summarises (Pi's
// prepareCompaction): the context from the previous compaction's kept
// range, or the start, up to a cut point that keeps KeepRecentTokens.
func Prepare(path []session.Entry, s Settings) (*Preparation, error) {
	if n := len(path); n > 0 && path[n-1].Type == session.TypeCompaction {
		return nil, ErrAlreadyCompacted
	}
	proj := project(Context(path))
	start, previous := 0, ""
	prev := -1
	if len(proj) > 0 && proj[0].entry.Type == session.TypeCompaction && len(proj[0].msgs) > 0 {
		prev, start, previous = 0, 1, proj[0].entry.Summary
	}
	cut := findCut(proj, start, len(proj), s.KeepRecentTokens)
	if cut.first >= len(proj) || proj[cut.first].entry.ID == "" {
		return nil, ErrNothingToCompact
	}
	historyEnd := cut.first
	if cut.split {
		historyEnd = cut.turnStart
	}
	p := &Preparation{
		FirstKeptEntryID: proj[cut.first].entry.ID,
		toSummarize:      messages(proj[start:historyEnd]),
		SplitTurn:        cut.split,
		TokensBefore:     int64(EstimateEntries(path)),
		previousSummary:  previous,
		files:            newFileOps(),
	}
	if cut.split {
		p.turnPrefix = messages(proj[cut.turnStart:cut.first])
	}
	if len(p.toSummarize) == 0 && len(p.turnPrefix) == 0 {
		return nil, ErrNothingToCompact
	}
	if prev >= 0 {
		p.files.carry(proj[prev].entry)
	}
	for _, x := range p.toSummarize {
		p.files.add(x)
	}
	for _, x := range p.turnPrefix {
		p.files.add(x)
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
func messages(ps []projected) []msg {
	var out []msg
	for _, p := range ps {
		if p.entry.Type == session.TypeCompaction {
			continue
		}
		out = append(out, p.msgs...)
	}
	return out
}

type cutResult struct {
	first, turnStart int
	split            bool
}

func isCutEntry(p projected) bool {
	if p.entry.Type == session.TypeCompaction {
		return false
	}
	for _, x := range p.msgs {
		if x.cutPoint() {
			return true
		}
	}
	return false
}

func isTurnStartEntry(p projected) bool {
	if p.entry.Type == session.TypeCompaction {
		return false
	}
	for _, x := range p.msgs {
		if x.turnStart() {
			return true
		}
	}
	return false
}

// findCut is Pi's findProjectedCutPoint: walk back from the newest entry
// until keep tokens are reached, then cut at the closest valid point at or
// after it, never at a tool result; step back over context-invisible
// entries; and say whether the cut splits a turn.
func findCut(proj []projected, start, end, keep int) cutResult {
	var points []int
	for i := start; i < end; i++ {
		if isCutEntry(proj[i]) {
			points = append(points, i)
		}
	}
	if len(points) == 0 {
		return cutResult{first: start, turnStart: -1}
	}
	cut, acc := points[0], 0
	for i := end - 1; i >= start; i-- {
		t := entryTokens(proj[i])
		if t == 0 {
			continue
		}
		acc += t
		if acc >= keep {
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
	for cut > start {
		p := proj[cut-1]
		if p.entry.Type == session.TypeCompaction || len(p.msgs) > 0 {
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

// Model is the model a summary is asked of: the session's own.
type Model struct {
	Provider llm.Provider
	Model    string
	Thinking string
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
			text, u, err := summarise(ctx, m, historyPrompt(p.toSummarize, instructions, p.previousSummary), "Summarization")
			if err != nil {
				return Result{}, err
			}
			history, usage = text, u
		}
		prefixPrompt := "# Conversation\n" + serialize(p.turnPrefix) + "\n\n# Instructions\n" + turnPrefixPrompt
		text, u, err := summarise(ctx, m, prefixPrompt, "Turn prefix summarization")
		if err != nil {
			return Result{}, err
		}
		summary = history + "\n\n---\n\n**Turn Context (split turn):**\n\n" + text
		usage = addUsage(usage, u)
	} else {
		text, u, err := summarise(ctx, m, historyPrompt(p.toSummarize, instructions, p.previousSummary), "Summarization")
		if err != nil {
			return Result{}, err
		}
		summary, usage = text, u
	}
	d := p.files.lists()
	return Result{Summary: summary + d.format(), FirstKeptEntryID: p.FirstKeptEntryID, TokensBefore: p.TokensBefore, Usage: usage, Details: d}, nil
}

// historyPrompt is Pi's summary request: the conversation, the previous
// summary when there is one, then the first or the update prompt, and the
// user's focus.
func historyPrompt(msgs []msg, instructions, previous string) string {
	base := summarizationPrompt
	if previous != "" {
		base = updatePrompt
	}
	if instructions != "" {
		base += "\n\nAdditional focus: " + instructions
	}
	var b strings.Builder
	b.WriteString("<conversation>\n" + serialize(msgs) + "\n</conversation>\n\n")
	if previous != "" {
		b.WriteString("<previous-summary>\n" + previous + "\n</previous-summary>\n\n")
	}
	b.WriteString(base)
	return b.String()
}

// summarise sends one summary request and reads the whole reply. A reply
// cut at the output limit, or one that calls a tool, is not a summary.
func summarise(ctx context.Context, m Model, prompt, label string) (string, llm.Usage, error) {
	req := &llm.Request{Model: m.Model, Thinking: m.Thinking, Messages: []llm.Message{
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
		return "", usage, fmt.Errorf("%s failed: generation hit the token cap and the summary is incomplete", label)
	case calls:
		return "", usage, fmt.Errorf("%s attempted to call a tool", label)
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
