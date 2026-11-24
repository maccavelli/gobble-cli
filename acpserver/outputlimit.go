package acpserver

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
)

// contextEstimate is the context's estimate before a call that carries
// tools, the automatic trigger's and the output limit's alike: the path's
// count anchored on its last valid usage, which holds the system prompt and
// the tools; otherwise the system prompt, the tools and every message
// (0071-MADR D3, pi's estimate.ts).
func (a *Agent) contextEstimate(s *liveSession, tools jsontext.Value) int {
	n, anchored := compaction.EstimateContext(s.tree.Path())
	if anchored {
		return n
	}
	return n + compaction.EstimateText(a.systemText(s)) + compaction.EstimateText(string(tools))
}

// compactionModel is provider id's model as a summary's: its output cap,
// the cap's source and its window, from the catalog (0071-MADR D6).
func (a *Agent) compactionModel(p llm.Provider, providerID, model, think string) compaction.Model {
	info := a.modelOf(providerID, model)
	return compaction.Model{Provider: p, Model: model, Thinking: think, MaxTokens: info.MaxTokens, MaxTokensFrom: info.MaxTokensFrom,
		Window: info.ContextWindow}
}

// turnLimits is one prompt's agent.Config.OutputLimit (0071-MADR D1, D2,
// D4): each call's limit from the catalog's cap for the turn's provider and
// model, read for each call, fitted to the window over contextEstimate, and
// recorded on the session; last is the turn's last call's.
type turnLimits struct {
	a        *Agent
	s        *liveSession
	provider string
	model    string
	last     llm.OutputLimit
}

// limit is the call's output limit: the refused limit's stated room or cap
// when req already names one (agent.Config.OutputLimit), otherwise
// llm.FitOutput's.
func (l *turnLimits) limit(req *llm.Request) llm.OutputLimit {
	lim := llm.OutputLimit{Tokens: req.MaxTokens, Basis: llm.BasisRefusal}
	if req.MaxTokens <= 0 {
		info := l.a.modelOf(l.provider, l.model)
		lim = llm.FitOutput(info.MaxTokens, info.MaxTokensFrom, info.ContextWindow, l.a.contextEstimate(l.s, req.Tools))
	}
	l.last = lim
	l.a.mu.Lock()
	l.s.outLimit = lim
	l.a.mu.Unlock()
	return lim
}

// limitFields are the request fields that carry an output limit, as a
// refusal names them, in lower case (0071-MADR D8 (c)).
var limitFields = []string{"max_tokens", "max_completion_tokens", "max_output_tokens", "maxoutputtokens"}

// fillsWindow is go-llmprovider-sdk's reason for an invalid request whose
// requested completion alone fills the context window.
const fillsWindow = "fills the context window"

// limitLine is a failed turn's error with the line that names the limit
// its last call asked for, when the failure is a refusal of that limit
// (0071-MADR D8 (c)). A turn whose calls carried no limit gains nothing. A
// credit or rate failure's line, D8 (d)'s, rides acpext.ErrorData's
// outputLimitNote from acpext's next minor tag, and the error text is left
// as it is (0071-MADR, amendment of 2026-11-24).
func limitLine(err error, lim llm.OutputLimit, model string) error {
	if lim.Tokens <= 0 || !limitRefusal(err, lim) {
		return err
	}
	return fmt.Errorf("%w; gobble asked for at most %s output tokens (%s); to send less, set maxTokens for %s under modelOverrides in models.json",
		err, thousands(int64(lim.Tokens)), lim.Basis, model)
}

// limitRefusal reports a refusal of the limit lim (D8 (c)): an invalid
// request whose message names an output-limit field, the SDK's "fills the
// context window" reason, or a refusal in a form llm.ReadLimitRefusal reads
// of a call already sent again at a refused limit's room or cap.
func limitRefusal(err error, lim llm.OutputLimit) bool {
	text := strings.ToLower(err.Error())
	if strings.Contains(text, fillsWindow) {
		return true
	}
	if _, ok := llm.ReadLimitRefusal(err, lim.Tokens); ok && lim.Basis == llm.BasisRefusal {
		return true
	}
	ae, ok := errors.AsType[*llm.APIError](err)
	if !ok || (ae.Status != http.StatusBadRequest && ae.Status != http.StatusUnprocessableEntity) || errors.Is(err, llm.ErrContextOverflow) {
		return false
	}
	text += " " + strings.ToLower(ae.Message)
	for _, f := range limitFields {
		if strings.Contains(text, f) {
			return true
		}
	}
	return false
}

// basisWords are each basis in /context's words (0071-MADR D10).
var basisWords = map[llm.LimitBasis]string{
	llm.BasisStated:  "the model's stated limit",
	llm.BasisCeiling: "gobble's 64,000 ceiling; set maxTokens in models.json to raise it",
	llm.BasisUser:    "your models.json maxTokens",
	llm.BasisAssumed: "assumed: no source states this model's limit",
	llm.BasisWindow:  "held to the room the window leaves",
	llm.BasisFull:    "the window is full by gobble's estimate",
	llm.BasisRefusal: "the cap the service stated when it refused the first",
}

// sourceWords are each cap source in /context's words.
var sourceWords = map[llm.CapSource]string{
	llm.CapCatalog: "the catalog",
	llm.CapUser:    "your models.json",
	llm.CapAssumed: "assumed",
}

// outputLimitLine is /context's and /session's line after the context's
// (0071-MADR D10): the last call's limit and its basis, or, before the
// session's first call, the model's cap and its source.
func (a *Agent) outputLimitLine(s *liveSession, provider, model string) string {
	a.mu.Lock()
	lim := s.outLimit
	a.mu.Unlock()
	if lim.Tokens > 0 {
		return fmt.Sprintf("Output limit: %s tokens on the last call (%s)\n", thousands(int64(lim.Tokens)), basisWords[lim.Basis])
	}
	info := a.modelOf(provider, model)
	if info.MaxTokens <= 0 {
		return "Output limit: none sent yet; the model's limit is unknown\n"
	}
	from := info.MaxTokensFrom
	if from == "" {
		from = llm.CapCatalog
	}
	return fmt.Sprintf("Output limit: none sent yet; the model's limit is %s (%s)\n", thousands(int64(info.MaxTokens)), sourceWords[from])
}
