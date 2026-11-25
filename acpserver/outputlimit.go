package acpserver

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/personality"
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

// compactionModel is provider id's model as s's summary's: its output cap,
// the cap's source and its window, from the catalog (0071-MADR D6). Its
// provider is p watched: a cap a refusal's Observed states, which the
// ladder sends again at, is learned, and a completed request's output
// reported to the saved caps (0071-MADR P25, P34).
func (a *Agent) compactionModel(s *liveSession, p llm.Provider, providerID, model, think string) compaction.Model {
	info := a.modelOf(providerID, model)
	c, from, _ := outputCap(info)
	watched := watchCalls(p, a.outputSeen(providerID), a.observedSaved(s, providerID))
	return compaction.Model{Provider: watched, Model: model, Thinking: think, MaxTokens: c, MaxTokensFrom: from, Window: windowOf(info)}
}

// sourceModelsJSON is the SDK's name of models.json's layer.
const sourceModelsJSON = "models.json"

// outputCap is a streamed call's cap of info and its source (0071-MADR P17):
// Limits.StreamedOutput when the catalog's limits know it, otherwise the
// merged MaxTokens. held reports models.json's maxTokens held below its value
// by a cap a service stated (P28), and asked is that maxTokens.
func outputCap(info llm.Model) (c int, from llm.CapSource, held int) {
	f := info.Limits.StreamedOutput
	if !f.Known {
		return info.MaxTokens, info.MaxTokensFrom, 0
	}
	if info.MaxTokensFrom == llm.CapUser && f.Source != sourceModelsJSON {
		for _, o := range f.Others {
			if o.Source == sourceModelsJSON && o.Tokens > f.Tokens {
				held = o.Tokens
			}
		}
	}
	return f.Tokens, info.MaxTokensFrom, held
}

// windowOf is info's window as compaction and the fit read it (0066-MADR
// D9): Limits.Window, else Limits.Input where a source states only that,
// else the merged ContextWindow; 0 for unknown.
func windowOf(info llm.Model) int {
	switch {
	case info.Limits.Window.Known:
		return info.Limits.Window.Tokens
	case info.Limits.Input.Known:
		return info.Limits.Input.Tokens
	}
	return info.ContextWindow
}

// turnLimits is one prompt's agent.Config.OutputLimit (0071-MADR D1, D2,
// D4): each call's limit from the catalog's cap for the turn's provider and
// model, read for each call, fitted to the window over contextEstimate, and
// recorded on the session; last is the turn's last call's, as asked, and
// shown what that call carried, failed whether it failed (0071-MADR P22,
// P32). note is P28's line, once per session and model, for the prompt's
// loop to send.
type turnLimits struct {
	a        *Agent
	s        *liveSession
	provider string
	model    string
	last     llm.OutputLimit
	shown    llm.OutputLimit
	failed   bool

	mu      sync.Mutex
	note    string
	noteKey string
}

// limit is the call's output limit: the refused limit's stated room or cap
// when req already names one (agent.Config.OutputLimit), otherwise
// llm.FitOutput's over outputCap and windowOf. A models.json maxTokens a
// service's stated cap holds is sent as that cap, basis service, and said
// once per session and model (0071-MADR P28).
func (l *turnLimits) limit(req *llm.Request) llm.OutputLimit {
	lim := llm.OutputLimit{Tokens: req.MaxTokens, Basis: llm.BasisRefusal}
	if req.MaxTokens <= 0 {
		info := l.a.modelOf(l.provider, l.model)
		c, from, held := outputCap(info)
		lim = llm.FitOutput(c, from, windowOf(info), l.a.contextEstimate(l.s, req.Tools))
		if held > 0 && lim.Basis == llm.BasisUser {
			lim.Basis = llm.BasisService
			l.userOver(held, c)
		}
	}
	l.last, l.shown, l.failed = lim, lim, false
	l.a.mu.Lock()
	l.s.outLimit, l.s.raisedFrom, l.s.outFailed = lim, 0, false
	l.a.mu.Unlock()
	return lim
}

// sent records what the call asked for at limit carried, from its done
// event's report (0071-MADR P22, P31, the owner's "gobble's basis, wire
// value (Recommended)"): the wire's tokens with gobble's basis when the SDK
// sent gobble's value, raised from it for a thinking budget when greater
// (0071-PLAN's review deviation of 2026-11-25, R4); basis
// none for a call that carried none; another SDK basis raw, as sdk:<basis>,
// logged once per session and model. A provider that reports nothing
// leaves the value asked.
func (l *turnLimits) sent(_ *llm.Request, d llm.Done, _ llm.Usage) {
	r := d.OutputLimit
	shown, raised := l.last, 0
	switch {
	case r.Basis == "":
	case !r.Sent || r.Basis == string(llm.BasisNone):
		shown = llm.OutputLimit{Basis: llm.BasisNone}
	case r.Basis == sdkBasisRequest:
		if r.Tokens > shown.Tokens {
			raised = shown.Tokens
		}
		shown.Tokens = r.Tokens
	default:
		shown = llm.OutputLimit{Tokens: r.Tokens, Basis: llm.LimitBasis(sdkBasisPrefix + r.Basis)}
		l.a.warnSDKBasis(l.s, l.provider, l.model, r.Basis)
	}
	l.shown, l.failed = shown, false
	l.a.mu.Lock()
	l.s.outLimit, l.s.raisedFrom, l.s.outFailed = shown, raised, false
	l.a.mu.Unlock()
}

// callFailed marks the last call's limit as asked by a call that failed
// (0071-MADR P32, the owner's "Show what was asked, marked (Recommended)").
func (l *turnLimits) callFailed(*llm.Request, error) {
	l.failed = true
	l.a.mu.Lock()
	l.s.outFailed = true
	l.a.mu.Unlock()
}

// The SDK's basis for a caller's value, and the prefix of any other basis it
// reports, as gobble shows it (0071-MADR P31).
const (
	sdkBasisRequest = "request"
	sdkBasisPrefix  = "sdk:"
)

// userOver keeps P28's line for the turn's loop while the session has not
// yet said it for the model: a service's cap sent over models.json's asked.
func (l *turnLimits) userOver(asked, stated int) {
	key := l.provider + "/" + l.model
	l.a.mu.Lock()
	said := l.s.userOver[key]
	l.a.mu.Unlock()
	if said {
		return
	}
	text := l.a.voice(l.s).Text(personality.ModelLimitUserOver, personality.Args{argModel: l.model,
		"asked": thousands(int64(asked)), "stated": thousands(int64(stated))})
	l.mu.Lock()
	l.note, l.noteKey = text, key
	l.mu.Unlock()
}

// sayNote sends the line userOver kept, once, to out, and marks it said for
// the session and model, so a call that failed before saying it leaves it
// for the next: the call just made sends a service's cap below
// models.json's maxTokens, said before its reply, as an agent message,
// live only (0071-MADR P28; 0071-PLAN Phase 2b's review).
func (l *turnLimits) sayNote(out *updates) {
	l.mu.Lock()
	note, key := l.note, l.noteKey
	l.note = ""
	l.mu.Unlock()
	if note == "" {
		return
	}
	l.a.mu.Lock()
	if l.s.userOver == nil {
		l.s.userOver = map[string]bool{}
	}
	l.s.userOver[key] = true
	l.a.mu.Unlock()
	out.flush()
	out.send(acp.UpdateAgentMessageText(note + "\n\n"))
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
// outputLimitNote (creditNote), and the error text is left as it is
// (0071-MADR, amendment of 2026-11-24).
func limitLine(err error, lim llm.OutputLimit, model string) error {
	if lim.Tokens <= 0 || !limitRefusal(err, lim) {
		return err
	}
	return fmt.Errorf("%w; gobble asked for at most %s output tokens (%s); to send less, set maxTokens for %s under modelOverrides in models.json",
		err, thousands(int64(lim.Tokens)), lim.Basis, model)
}

// kiloProvider is gobble's id of Kilo, whose gateway answers an upstream's
// 402 with a 503 (0065-MADR R34).
const kiloProvider = "kilo"

// creditNote is D8 (d)'s line for a failed turn whose last call asked for
// the limit lim: a 402 or quota failure, a 429 or rate-limit failure that
// outlasted the SDK's retries, or Kilo's 503 for an upstream's 402, each
// held by some services against the number asked for. It is the member
// acpext.ErrorData.OutputLimitNote and never part of the error's text. A turn
// whose calls carried no limit, and any other failure, gets "" (0071-MADR D8
// (d) as amended 2026-11-24; L15).
func creditNote(err error, lim llm.OutputLimit, model string) string {
	if lim.Tokens <= 0 || !creditOrRate(err) {
		return ""
	}
	return fmt.Sprintf("gobble asked for at most %s output tokens (%s); some services hold credit or rate against that number; "+
		"set maxTokens for %s under modelOverrides in models.json to ask for less", thousands(int64(lim.Tokens)), lim.Basis, model)
}

// creditOrRate reports a quota or rate-limit failure, or Kilo's 503 for an
// upstream's 402.
func creditOrRate(err error) bool {
	if errors.Is(err, llm.ErrQuota) || errors.Is(err, llm.ErrRateLimited) {
		return true
	}
	ae, ok := errors.AsType[*llm.APIError](err)
	return ok && ae.Status == http.StatusServiceUnavailable && ae.Provider == kiloProvider
}

// limitRefusal reports a refusal of the limit lim (D8 (c)): an invalid
// request whose message names an output-limit field, the SDK's "fills the
// context window" reason, or a refusal of a call already sent again at a
// refused limit's room or cap: in a form llm.ReadLimitRefusal reads, or one
// only the SDK reads, which states its cap as the error's Observed, such as
// Kilo's range form (0071-MADR P34).
func limitRefusal(err error, lim llm.OutputLimit) bool {
	text := strings.ToLower(err.Error())
	if strings.Contains(text, fillsWindow) {
		return true
	}
	if _, ok := llm.ReadLimitRefusal(err, lim.Tokens); ok && lim.Basis == llm.BasisRefusal {
		return true
	}
	if ae, ok := errors.AsType[*llm.APIError](err); ok && ae.Observed.Tokens > 0 && lim.Basis == llm.BasisRefusal {
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
	llm.BasisService: "the cap the service stated, below your models.json maxTokens",
}

// wordsOf is basis in /context's words; a basis gobble has no words for,
// the SDK's raw sdk:<basis>, as it is (0071-MADR P31).
func wordsOf(basis llm.LimitBasis) string {
	if w, ok := basisWords[basis]; ok {
		return w
	}
	return string(basis)
}

// sourceWords are each cap source in /context's words.
var sourceWords = map[llm.CapSource]string{
	llm.CapCatalog: "the catalog",
	llm.CapUser:    "your models.json",
	llm.CapAssumed: "assumed",
}

// outputLimitLine is /context's and /session's line after the context's
// (0071-MADR D10): the last call's limit and its basis, or, before the
// session's first call in this process, a loaded or forked one's included,
// the model's cap and its source. The limit is what the call carried: the
// wire's tokens, with the value gobble asked when a thinking budget raised
// it; none, when it carried none; asked, marked, when the call failed
// (0071-MADR P22, P31, P32).
func (a *Agent) outputLimitLine(s *liveSession, provider, model string) string {
	a.mu.Lock()
	lim, raised, failed := s.outLimit, s.raisedFrom, s.outFailed
	a.mu.Unlock()
	switch {
	case lim.Basis == llm.BasisNone:
		return "Output limit: none sent on the last call; the service's default applies\n"
	case lim.Tokens > 0 && failed:
		return fmt.Sprintf("Output limit: %s tokens asked on the last call, which failed (%s)\n", thousands(int64(lim.Tokens)), wordsOf(lim.Basis))
	case lim.Tokens > 0 && raised > 0:
		return fmt.Sprintf("Output limit: %s tokens on the last call (%s); raised from %s for the thinking budget\n",
			thousands(int64(lim.Tokens)), wordsOf(lim.Basis), thousands(int64(raised)))
	case lim.Tokens > 0:
		return fmt.Sprintf("Output limit: %s tokens on the last call (%s)\n", thousands(int64(lim.Tokens)), wordsOf(lim.Basis))
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
