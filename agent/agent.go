package agent

import (
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
)

// DefaultMaxCalls is how many model calls one prompt may make before the
// turn ends with StopMaxTurnRequests.
const DefaultMaxCalls = 100

// Config configures an Agent.
type Config struct {
	Provider llm.Provider
	Tools    []tool.Tool
	// System is the system prompt.
	System string
	// Model overrides the provider's model; empty keeps it.
	Model string
	// Thinking is the thinking level each request asks for (llm's levels);
	// empty asks for none.
	Thinking string
	// Cache asks the service to cache each model call's prompt
	// (0021-MADR D2); CacheNone asks for nothing.
	Cache llm.CacheRetention
	// Policy approves calls to tools that are not read-only. Nil allows
	// every call.
	Policy permission.Policy
	// MaxCalls bounds the model calls of one prompt; zero is
	// DefaultMaxCalls.
	MaxCalls int
	// Queue gives the turn the messages queued while it runs. Nil queues
	// nothing.
	Queue Queue
	// Search is tool_search, offered as a direct tool only while a tool is
	// deferred (0012-MADR D9). Nil offers none.
	Search tool.Tool
	// Loaded names deferred tools an earlier prompt loaded, in load order.
	// They are sent as direct tools, after the others (0012-MADR D9).
	Loaded []string
	// Retry is the session's retry settings, of which the turn reads Off
	// and OfflineWait: the provider's own retries are the provider's. Nil
	// takes no wait (0005-PLAN F5-1).
	Retry *llm.Retry
	// RetryAbort, once closed, ends a wait the turn is taking, and the turn
	// with the error that caused it (_gobble/abort_retry). Nil never does.
	RetryAbort <-chan struct{}
	// Compact is asked before each model call whether to compact the
	// session, and, with overflow true, once when a call fails with
	// llm.ErrContextOverflow (0005-PLAN F5-2 decisions 1 to 3). When it
	// compacted, it returns the session after the compaction, and the turn
	// goes on from it; the call after an overflow is sent again once. It
	// returns nil when it did not compact; an overflow it did not compact,
	// with no error, ends the turn with the overflow. Its error ends the
	// turn; a cancelled compaction returns ctx's. Nil never compacts.
	Compact func(ctx context.Context, overflow bool) (*Compacted, error)
}

// Compacted is the session after a compaction (Config.Compact): its
// history, which holds the turn so far, and the system prompt a compaction
// rebuilds, which the turn sends from then on (0005-PLAN F5-2 decision 3,
// deviation 1).
type Compacted struct {
	History []llm.Message
	System  string
}

// Queue is Pi's steering and follow-up queues (agent.ts). Each call drains
// by the queue's own mode: the first message, or all of them.
type Queue interface {
	// Steering is polled after the prompt, and after each model reply and
	// its tool results. Its messages enter before the next model call.
	Steering() []llm.Message
	// FollowUps is polled only when the turn would otherwise end.
	FollowUps() []llm.Message
}

// Agent runs turns: model calls and the tool calls they ask for, until the
// model ends its reply. It is safe to run turns of different sessions at
// once; one session's turns run one after another.
//
// Tools are offered by exposure (0012-MADR D7): direct ones are sent with
// every request, deferred ones wait behind tool_search until a call loads
// them, and hidden ones are neither sent nor found. A load is append-only
// and lasts the turn; the caller records it, and passes it back as Loaded.
type Agent struct {
	cfg      Config
	tools    map[string]tool.Tool // the tools a call may name
	direct   []tool.Spec          // the specs sent, in order
	deferred []tool.Tool          // the tools behind tool_search, in Config order
}

// New returns an Agent. It panics if two tools share a name, which is a
// programming error.
func New(cfg Config) *Agent {
	if cfg.MaxCalls <= 0 {
		cfg.MaxCalls = DefaultMaxCalls
	}
	a := &Agent{cfg: cfg, tools: map[string]tool.Tool{}}
	byName := map[string]tool.Tool{}
	for _, t := range cfg.Tools {
		name := t.Spec().Name
		if _, dup := byName[name]; dup {
			panic("agent: two tools are named " + name)
		}
		byName[name] = t
	}
	loaded := map[string]bool{}
	for _, n := range cfg.Loaded {
		loaded[n] = true
	}
	for _, t := range cfg.Tools {
		switch s := t.Spec(); s.Exposure {
		case tool.ExposureHidden:
		case tool.ExposureDeferred:
			if !loaded[s.Name] {
				a.deferred = append(a.deferred, t)
			}
		default:
			a.offer(t)
		}
	}
	for _, n := range cfg.Loaded {
		if t, ok := byName[n]; ok && t.Spec().Exposure == tool.ExposureDeferred && a.tools[n] == nil {
			a.offer(t)
		}
	}
	if len(a.deferred) > 0 && cfg.Search != nil {
		if _, dup := byName[cfg.Search.Spec().Name]; dup {
			panic("agent: two tools are named " + cfg.Search.Spec().Name)
		}
		a.offer(cfg.Search)
	}
	return a
}

// Offered names the tools a turn's first request sends, in order: the
// direct ones, then those Loaded names, then tool_search while a tool is
// deferred. A caller names them in its system prompt.
func (a *Agent) Offered() []string {
	out := make([]string, len(a.direct))
	for i, s := range a.direct {
		out[i] = s.Name
	}
	return out
}

// offer makes t a tool the requests send.
func (a *Agent) offer(t tool.Tool) {
	s := t.Spec()
	a.tools[s.Name] = t
	a.direct = append(a.direct, s)
}

// Run runs one prompt over history and yields the turn's events, ending
// with End. A done ctx ends the turn with StopCancelled, not an error; a
// provider failure is yielded as an error and ends the stream.
func (a *Agent) Run(ctx context.Context, env tool.Env, history []llm.Message, prompt llm.Message) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		t := &turn{a: a, ctx: ctx, env: env, yield: yield, history: history, added: []llm.Message{prompt}, system: a.cfg.System,
			tools: maps.Clone(a.tools), sent: slices.Clone(a.direct), deferred: slices.Clone(a.deferred)}
		t.specs = encodeSpecs(t.sent)
		t.run()
	}
}

// turn is one Run's state. Its tools start as the Agent's, and grow by the
// loads its calls make.
type turn struct {
	a        *Agent
	ctx      context.Context
	env      tool.Env
	yield    func(Event, error) bool
	history  []llm.Message
	added    []llm.Message
	system   string // Config.System, until a compaction rebuilds it
	usage    llm.Usage
	stopped  bool // the consumer stopped iterating
	tools    map[string]tool.Tool
	sent     []tool.Spec
	specs    jsontext.Value
	deferred []tool.Tool
	// repeat is the run of identical calls the turn's last call ended, and
	// last that call's key (0005-PLAN F5-3 decision 1); bad counts the
	// turn's consecutive results whose arguments were refused, badTool the
	// last one's tool (decision 3); halted says a call was stopped, which
	// ends the turn.
	repeat  int
	last    string
	bad     int
	badTool string
	halted  bool
}

// encodeSpecs is specs as a request carries them, or nil for none.
func encodeSpecs(specs []tool.Spec) jsontext.Value {
	if len(specs) == 0 {
		return nil
	}
	b, err := json.Marshal(specs)
	if err != nil {
		panic(fmt.Sprintf("agent: encode tool specs: %v", err))
	}
	return b
}

// load moves the named deferred tools, each with its whole family, into
// the requests, after the tools already sent, and encodes the specs again
// for the next model call. It returns the names loaded, in that order; a
// name that is not deferred is ignored.
func (t *turn) load(names []string) []string {
	var out []string
	for _, n := range names {
		i := slices.IndexFunc(t.deferred, func(d tool.Tool) bool { return d.Spec().Name == n })
		if i < 0 {
			continue
		}
		family := t.deferred[i].Spec().Family
		var keep []tool.Tool
		for _, d := range t.deferred {
			s := d.Spec()
			if s.Name != n && (family == "" || s.Family != family) {
				keep = append(keep, d)
				continue
			}
			t.tools[s.Name] = d
			t.sent = append(t.sent, s)
			out = append(out, s.Name)
		}
		t.deferred = keep
	}
	if len(out) > 0 {
		t.specs = encodeSpecs(t.sent)
	}
	return out
}

// deferredSpecs are the turn's deferred tools not loaded yet, for
// tool_search (tool.Env.Deferred).
func (t *turn) deferredSpecs() []tool.Spec {
	if len(t.deferred) == 0 {
		return nil
	}
	out := make([]tool.Spec, len(t.deferred))
	for i, d := range t.deferred {
		out[i] = d.Spec()
	}
	return out
}

func (t *turn) emit(ev Event) bool {
	if t.stopped {
		return false
	}
	if !t.yield(ev, nil) {
		t.stopped = true
	}
	return !t.stopped
}

func (t *turn) end(reason StopReason) {
	t.emit(End{StopReason: reason, Messages: t.added, Usage: t.usage})
}

// run is Pi's run loop (agent-loop.ts:173-308): steering is polled after
// the prompt and after each reply and its tool results, and the turn goes
// on while the reply asked for tools or steering came; when it would end,
// follow-ups are polled, and any keep it going.
func (t *turn) run() {
	if !t.inject(t.steering()) {
		return
	}
	for calls := 0; ; calls++ {
		if calls == t.a.cfg.MaxCalls {
			t.end(StopMaxTurnRequests)
			return
		}
		if _, err := t.compact(false); err != nil {
			t.fail(err)
			return
		}
		reply, ok := t.next()
		if !ok {
			return
		}
		msg := reply.message()
		t.added = append(t.added, msg)
		if !t.emit(Message{Message: msg, Usage: reply.usage, StopReason: reply.stop}) {
			return
		}
		if t.ctx.Err() != nil {
			t.added = append(t.added, cancelledResults(reply.calls)...)
			t.end(StopCancelled)
			return
		}
		if len(reply.calls) > 0 {
			results, ok := t.runTools(reply.calls)
			if !ok {
				return
			}
			t.added = append(t.added, results)
			switch {
			case t.ctx.Err() != nil:
				t.end(StopCancelled)
				return
			case t.halted:
				t.end(StopEndTurn)
				return
			case t.bad >= badLimit:
				t.fail(fmt.Errorf("the model sent invalid tool arguments %d times in a row, the last for %s", t.bad, t.badTool))
				return
			}
		}
		pending := t.steering()
		if len(reply.calls) == 0 && len(pending) == 0 {
			pending = t.followUps()
		}
		if len(reply.calls) == 0 && len(pending) == 0 {
			switch reply.stop {
			case llm.StopMaxTokens:
				t.end(StopMaxTokens)
			case llm.StopRefusal:
				t.end(StopRefusal)
			default:
				t.end(StopEndTurn)
			}
			return
		}
		if !t.inject(pending) {
			return
		}
	}
}

func (t *turn) steering() []llm.Message {
	if t.a.cfg.Queue == nil || t.ctx.Err() != nil {
		return nil
	}
	return t.a.cfg.Queue.Steering()
}

func (t *turn) followUps() []llm.Message {
	if t.a.cfg.Queue == nil || t.ctx.Err() != nil {
		return nil
	}
	return t.a.cfg.Queue.FollowUps()
}

// inject adds queued messages to the turn, each reported before the model
// call that reads it.
func (t *turn) inject(msgs []llm.Message) bool {
	for _, m := range msgs {
		t.added = append(t.added, m)
		if !t.emit(Queued{Message: m}) {
			return false
		}
	}
	return true
}

// reply is one model call's answer.
type reply struct {
	text     strings.Builder
	thinking []llm.Content
	pending  strings.Builder // thinking text not yet in a block
	calls    []llm.ToolCallDone
	stop     llm.StopReason
	usage    llm.Usage
}

func (r *reply) message() llm.Message {
	m := llm.Message{Role: llm.RoleAssistant}
	m.Content = append(m.Content, r.thinking...)
	if r.pending.Len() > 0 {
		m.Content = append(m.Content, llm.Content{Type: llm.ContentThinking, Text: r.pending.String()})
	}
	if r.text.Len() > 0 {
		m.Content = append(m.Content, llm.Content{Type: llm.ContentText, Text: r.text.String()})
	}
	for _, c := range r.calls {
		m.Content = append(m.Content, llm.Content{Type: llm.ContentToolCall, ToolCallID: c.ID, ToolName: c.Name, ToolArgs: c.Args, ProviderData: c.ProviderData})
	}
	return m
}

// The turn's own waits (0005-PLAN F5-1): an empty reply runs again after
// each of emptyDelays, as Kilo's does; the network is probed every
// probeEvery, as Kilo's is.
var (
	emptyDelays = []time.Duration{2 * time.Second, 4 * time.Second}
	probeEvery  = 3 * time.Second
)

var (
	// errEmpty ends a turn whose empty reply's wait was cut short.
	errEmpty = fmt.Errorf("the provider gave an empty reply: %w", llm.ErrEmptyReply)
	// errNoUsableReply ends a turn whose replies were all empty.
	errNoUsableReply = fmt.Errorf("the provider gave no usable reply %d times: %w", len(emptyDelays)+1, llm.ErrEmptyReply)
)

// What a wait's End says when it was cut short.
const (
	retryCancelled = "retry cancelled" // _gobble/abort_retry
	turnCancelled  = "cancelled"       // the turn
)

// waited is how a wait ended.
type waited int

const (
	waitDone waited = iota
	waitCancelled
	waitAborted
	waitExpired
)

// compact asks Config.Compact to compact the session (0005-PLAN F5-2
// decisions 1 to 3). When it did, the turn goes on from the history and
// the system prompt it returns; the history holds the turn so far, so
// added is emptied, and End's Messages are what came after the compaction.
func (t *turn) compact(overflow bool) (bool, error) {
	if t.a.cfg.Compact == nil {
		return false, nil
	}
	c, err := t.a.cfg.Compact(t.ctx, overflow)
	if err != nil || c == nil {
		return false, err
	}
	t.history, t.system, t.added = c.History, c.System, nil
	return true, nil
}

// fail ends the turn on err: as cancelled when the turn was, or with err
// yielded.
func (t *turn) fail(err error) {
	if t.ctx.Err() != nil {
		t.end(StopCancelled)
		return
	}
	t.yield(nil, err)
	t.stopped = true
}

// next is call with the waits of 0005-PLAN F5-1 decisions 3 and 4: an
// offline call waits for the provider's host and is sent again, and an
// empty reply runs again twice, neither reaching the history. Each wait is
// reported as Retry events. A call over the model's window is compacted
// and sent once more (0005-PLAN F5-2 decision 2). ok is false when the
// turn is over: a failure yielded, the consumer stopped, or the turn
// cancelled in a wait or a compaction.
func (t *turn) next() (*reply, bool) {
	rt := t.a.cfg.Retry
	var cause RetryCause // the wait under way, or none
	var deadline time.Time
	empties := 0
	resent := false // the call was sent again after an overflow
	for {
		r, err := t.call()
		switch {
		case t.stopped:
			return r, false
		case err == nil && t.ctx.Err() != nil:
			t.endWait(cause, turnCancelled)
			return r, !t.stopped
		case errors.Is(err, llm.ErrContextOverflow) && !resent && t.a.cfg.Compact != nil:
			resent = true
			done, cerr := t.compact(true)
			if cerr != nil && t.ctx.Err() != nil {
				t.endWait(cause, turnCancelled)
				t.end(StopCancelled)
				return nil, false
			}
			if done {
				continue
			}
			if cerr != nil {
				err = cerr
			}
		case rt == nil || rt.Off:
		case errors.Is(err, llm.ErrOffline) && t.prober() != nil && rt.OfflineWait > 0:
			if cause != RetryOffline {
				t.endWait(cause, "")
				cause, deadline = RetryOffline, time.Now().Add(rt.OfflineWait)
				if !t.emit(Retry{Phase: RetryStart, Cause: RetryOffline}) {
					return r, false
				}
			}
			w := t.waitOnline(deadline)
			if w == waitDone {
				continue
			}
			if !t.cutShort(w, &cause) {
				return nil, false
			}
		case t.empty(r, err) && empties < len(emptyDelays):
			if cause == RetryOffline {
				t.endWait(cause, "") // the network is back
			}
			cause = RetryEmptyReply
			d := emptyDelays[empties]
			empties++
			if !t.emit(Retry{Phase: RetryStart, Cause: RetryEmptyReply, Attempt: empties, MaxAttempts: len(emptyDelays), Delay: d}) {
				return r, false
			}
			w := t.sleep(d)
			if w == waitDone {
				continue
			}
			if !t.cutShort(w, &cause) {
				return nil, false
			}
			if err == nil {
				err = errEmpty
			}
		case t.empty(r, err):
			err = errNoUsableReply
		}
		if err != nil {
			t.endWait(cause, err.Error())
			t.yield(nil, err)
			t.stopped = true
			return r, false
		}
		t.endWait(cause, "")
		return r, true
	}
}

// cutShort ends a wait that w cut short. A cancel ends the turn as
// cancelled, and it returns false. An abort reports the wait's end as
// cancelled and clears *cause; an expiry leaves the end for the caller to
// report with the wait's error. Either returns true for the caller to end
// the turn with that error.
func (t *turn) cutShort(w waited, cause *RetryCause) bool {
	switch w {
	case waitCancelled:
		t.endWait(*cause, turnCancelled)
		t.end(StopCancelled)
		return false
	case waitAborted:
		t.endWait(*cause, retryCancelled)
		*cause = ""
	}
	return true
}

// endWait reports the end of the wait under way, if any: a success when
// why is "".
func (t *turn) endWait(cause RetryCause, why string) {
	if cause != "" {
		t.emit(Retry{Phase: RetryEnd, Cause: cause, Err: why})
	}
}

// empty reports whether a call's answer had nothing usable: the SDK's
// empty answer, or a reply with no text and no tool calls, reasoning or
// not, that did not stop at max tokens (0005-PLAN F5-1 decision 4), nor as
// a refusal (F5-3 decision 5).
func (t *turn) empty(r *reply, err error) bool {
	if err != nil {
		return errors.Is(err, llm.ErrEmptyReply)
	}
	return r.text.Len() == 0 && len(r.calls) == 0 && r.stop != llm.StopMaxTokens && r.stop != llm.StopRefusal
}

// prober is the provider as an llm.Prober, or nil.
func (t *turn) prober() llm.Prober {
	if p, ok := t.a.cfg.Provider.(llm.Prober); ok {
		return p
	}
	return nil
}

// waitOnline probes the provider's host every probeEvery until it
// answers, the turn is cancelled, the wait is aborted, or deadline passes;
// a probe that would come after deadline is not made.
func (t *turn) waitOnline(deadline time.Time) waited {
	for {
		left := time.Until(deadline)
		if left < probeEvery {
			if w := t.sleep(left); w != waitDone {
				return w
			}
			return waitExpired
		}
		if w := t.sleep(probeEvery); w != waitDone {
			return w
		}
		if probe(t.ctx, t.prober(), t.a.cfg.RetryAbort) == nil {
			return waitDone
		}
	}
}

// probe is one of p's probes, cut short when ctx is done or abort closes.
func probe(ctx context.Context, p llm.Prober, abort <-chan struct{}) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-abort:
			cancel()
		case <-ctx.Done():
		}
	}()
	return p.Probe(ctx)
}

// sleep waits d, or until the turn is cancelled or the wait aborted.
func (t *turn) sleep(d time.Duration) waited {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-t.ctx.Done():
		return waitCancelled
	case <-t.a.cfg.RetryAbort:
		return waitAborted
	case <-timer.C:
		return waitDone
	}
}

// call makes one model call, yielding its deltas. err is the stream's
// failure, not yet yielded, or nil when the turn was cancelled; a consumer
// that stopped sets t.stopped.
func (t *turn) call() (*reply, error) {
	msgs := make([]llm.Message, 0, len(t.history)+len(t.added)+1)
	if t.system != "" {
		msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: []llm.Content{{Type: llm.ContentText, Text: t.system}}})
	}
	msgs = append(append(msgs, t.history...), t.added...)
	req := &llm.Request{Model: t.a.cfg.Model, Messages: msgs, Tools: t.specs, Thinking: t.a.cfg.Thinking, Cache: t.a.cfg.Cache}
	r := &reply{}
	for ev, err := range t.a.cfg.Provider.Stream(t.ctx, req) {
		if err != nil {
			if t.ctx.Err() != nil {
				return r, nil
			}
			return r, err
		}
		switch ev := ev.(type) {
		case llm.TextDelta:
			r.text.WriteString(ev.Text)
			if ev.Text != "" && !t.emit(TextDelta{Text: ev.Text}) {
				return r, nil
			}
		case llm.ThinkingDelta:
			r.pending.WriteString(ev.Text)
			if len(ev.ProviderData) > 0 {
				// A delta carrying provider data closes a reasoning block, which
				// is replayed to the provider as it was given.
				r.thinking = append(r.thinking, llm.Content{Type: llm.ContentThinking, Text: r.pending.String(), ProviderData: ev.ProviderData})
				r.pending.Reset()
			}
			if ev.Text != "" && !t.emit(ThinkingDelta{Text: ev.Text}) {
				return r, nil
			}
		case llm.ToolCallDone:
			ev.Name = t.repair(ev.Name)
			r.calls = append(r.calls, ev)
		case llm.Usage:
			t.usage = addUsage(t.usage, ev)
			r.usage = addUsage(r.usage, ev)
			if !t.emit(Usage{Usage: ev}) {
				return r, nil
			}
		case llm.Done:
			r.stop = ev.StopReason
		}
	}
	return r, nil
}

// runTools runs the calls one after another and returns their results as
// one tool message. After a cancel, the calls not run get a cancelled
// result, so every call in the history has one.
func (t *turn) runTools(calls []llm.ToolCallDone) (llm.Message, bool) {
	m := llm.Message{Role: llm.RoleTool}
	for i, c := range calls {
		if t.ctx.Err() != nil {
			m.Content = append(m.Content, cancelledResults(calls[i:])[0].Content...)
			break
		}
		var (
			res tool.Result
			ok  bool
		)
		if t.repeated(c) {
			res, ok = t.stopRepeat(c)
		} else {
			res, ok = t.runTool(c)
		}
		if !ok {
			return m, false
		}
		if res.BadArguments {
			t.bad, t.badTool = t.bad+1, c.Name
		} else {
			t.bad = 0
		}
		if t.halted {
			m.Content = append(m.Content, llm.Content{Type: llm.ContentToolResult, ToolCallID: c.ID, ToolName: c.Name, Text: res.Text(), IsError: true})
			if rest := cancelledResults(calls[i+1:]); len(rest) > 0 {
				m.Content = append(m.Content, rest[0].Content...)
			}
			break
		}
		m.Content = append(m.Content, llm.Content{Type: llm.ContentToolResult, ToolCallID: c.ID, ToolName: c.Name, Text: res.Text(), IsError: res.IsError})
		if len(res.Load) > 0 && !res.IsError {
			if names := t.load(res.Load); len(names) > 0 && !t.emit(Loaded{Names: names}) {
				return m, false
			}
		}
	}
	return m, true
}

func (t *turn) runTool(c llm.ToolCallDone) (tool.Result, bool) {
	call := tool.Call{ID: c.ID, Name: c.Name, Args: jsontext.Value(c.Args)}
	tl, known := t.tools[c.Name]
	if !known {
		res := tool.ErrorResult(fmt.Sprintf("there is no tool named %q", c.Name))
		return res, t.emit(ToolStart{Call: call, Title: c.Name}) && t.emit(ToolEnd{Call: call, Result: res})
	}
	spec := tl.Spec()
	title := spec.Name
	if d, ok := tl.(tool.Describer); ok {
		title = d.Describe(call, t.env)
	}
	if !t.emit(ToolStart{Call: call, Spec: spec, Title: title}) {
		return tool.Result{}, false
	}
	env := t.env
	env.Deferred = t.deferredSpecs()
	var outside []string
	if c, ok := tl.(tool.Confined); ok {
		outside = c.Outside(call, env)
	}
	if res, denied := t.approve(call, spec, title, outside); denied {
		return res, t.emit(ToolEnd{Call: call, Result: res})
	}
	env.AllowOutside = len(outside) > 0
	if !t.emit(ToolRun{Call: call}) {
		return tool.Result{}, false
	}
	res, err := tl.Run(t.ctx, call, env)
	if err != nil {
		res = tool.ErrorResult(cancelledText)
		if t.ctx.Err() == nil {
			res = tool.ErrorResult(fmt.Sprintf("%s failed: %v", spec.Name, err))
		}
	}
	return res, t.emit(ToolEnd{Call: call, Result: res})
}

// approve asks the policy about a call to a tool that is not read-only,
// and about any call naming paths outside the workspace roots. It returns
// the result to give the model when the call does not run. With no policy
// an outside call is refused: there is no one to ask.
func (t *turn) approve(call tool.Call, spec tool.Spec, title string, outside []string) (tool.Result, bool) {
	if len(outside) > 0 {
		if t.a.cfg.Policy == nil {
			return tool.ErrorResult(outside[0] + " is outside the workspace (the working directory and the additional directories), and no permission policy can approve it."), true
		}
		title += " (outside the workspace)"
	} else if t.a.cfg.Policy == nil || spec.Annotations.ReadOnlyHint {
		return tool.Result{}, false
	}
	d, err := t.a.cfg.Policy.Decide(t.ctx, permission.Rule{Tool: spec.Name, Mode: permission.ModeAsk, CallID: call.ID, Title: title, Input: call.Args})
	switch {
	case t.ctx.Err() != nil:
		return tool.ErrorResult(cancelledText), true
	case err != nil:
		return tool.ErrorResult(fmt.Sprintf("the permission request failed, so %s did not run: %v", spec.Name, err)), true
	case !d.Allow && d.Refusal != "":
		return tool.ErrorResult(d.Refusal), true
	case !d.Allow:
		msg := "The user declined this call, so it did not run."
		if d.Reason != "" {
			msg += " " + d.Reason
		}
		return tool.ErrorResult(msg), true
	}
	return tool.Result{}, false
}

const cancelledText = "The user cancelled the turn, so this call did not finish."

// The loop's limits (0005-PLAN F5-3): the run of identical calls that asks
// whether the turn may go on (opencode's DOOM_LOOP_THRESHOLD), and the
// consecutive refused arguments that end it (Kilo's
// REPEATED_TOOL_FAILURE_LIMIT).
const (
	repeatLimit = 3
	badLimit    = 3
)

// stoppedText is the result of a repeated call the user stopped.
const stoppedText = "The user stopped this call: the model made the same call 3 times in a row."

// repair is name, or the one tool the turn offers whose name it is but for
// case and surrounding space (Kilo's repair; 0005-PLAN F5-3 decision 2). A
// name that is a tool, or that two tools or none match, is kept.
func (t *turn) repair(name string) string {
	if _, ok := t.tools[name]; ok {
		return name
	}
	want := strings.TrimSpace(name)
	found := ""
	for n := range t.tools {
		if !strings.EqualFold(n, want) {
			continue
		}
		if found != "" {
			return name
		}
		found = n
	}
	return cmp.Or(found, name)
}

// repeated counts c into the turn's run of identical calls, its name and
// arguments, compared as canonical JSON or else as text, equal to the one
// before it; it reports whether the run reached repeatLimit (0005-PLAN
// F5-3 decision 1).
func (t *turn) repeated(c llm.ToolCallDone) bool {
	args := jsontext.Value(c.Args).Clone()
	if args.Canonicalize() != nil {
		args = jsontext.Value(c.Args)
	}
	key := c.Name + "\x00" + string(args)
	if key == t.last {
		t.repeat++
	} else {
		t.last, t.repeat = key, 1
	}
	return t.repeat >= repeatLimit
}

// stopRepeat asks the policy whether a call that ends a run of repeatLimit
// identical calls may go on. Allowed, or with no policy to ask, it runs as
// any call does, and the run starts again. Refused, or when the question
// fails, it does not run, and the turn ends after it (halted).
func (t *turn) stopRepeat(c llm.ToolCallDone) (tool.Result, bool) {
	n := t.repeat
	t.repeat = 0
	if t.a.cfg.Policy == nil {
		return t.runTool(c)
	}
	call := tool.Call{ID: c.ID, Name: c.Name, Args: jsontext.Value(c.Args)}
	title := c.Name
	if tl, ok := t.tools[c.Name]; ok {
		if d, ok := tl.(tool.Describer); ok {
			title = d.Describe(call, t.env)
		}
	}
	d, err := t.a.cfg.Policy.Decide(t.ctx, permission.Rule{Tool: c.Name, Mode: permission.ModeAsk, CallID: c.ID, Title: title,
		Input: call.Args, Repeated: n})
	switch {
	case t.ctx.Err() != nil:
		res := tool.ErrorResult(cancelledText)
		return res, t.emit(ToolStart{Call: call, Title: title}) && t.emit(ToolEnd{Call: call, Result: res})
	case err == nil && d.Allow:
		return t.runTool(c)
	}
	t.halted = true
	res := tool.ErrorResult(stoppedText)
	return res, t.emit(ToolStart{Call: call, Title: title}) && t.emit(ToolEnd{Call: call, Result: res})
}

// cancelledResults is a tool message answering each call with the
// cancelled result, or nothing when there are no calls.
func cancelledResults(calls []llm.ToolCallDone) []llm.Message {
	if len(calls) == 0 {
		return nil
	}
	m := llm.Message{Role: llm.RoleTool}
	for _, c := range calls {
		m.Content = append(m.Content, llm.Content{Type: llm.ContentToolResult, ToolCallID: c.ID, ToolName: c.Name, Text: cancelledText, IsError: true})
	}
	return []llm.Message{m}
}

func addUsage(a, b llm.Usage) llm.Usage {
	return llm.Usage{
		InputTokens:      a.InputTokens + b.InputTokens,
		OutputTokens:     a.OutputTokens + b.OutputTokens,
		ReasoningTokens:  a.ReasoningTokens + b.ReasoningTokens,
		CacheReadTokens:  a.CacheReadTokens + b.CacheReadTokens,
		CacheWriteTokens: a.CacheWriteTokens + b.CacheWriteTokens,
		Estimated:        a.Estimated || b.Estimated,
	}
}
