package provider

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
)

// retryPolicy is Pi's 3 retries with a 60 s cap (0004-MADR, F4 notes), a
// session's when its settings change nothing (0005-PLAN F5-1).
var retryPolicy = policyOf(llm.DefaultRetry)

// policyOf is r as the SDK's retry takes it: MaxAttempts counts the first
// try.
func policyOf(r llm.Retry) llmprovider.RetryPolicy {
	return llmprovider.RetryPolicy{MaxAttempts: r.MaxRetries + 1, BaseDelay: r.BaseDelay, MaxDelay: r.MaxDelay}
}

// Options configure New.
type Options struct {
	// APIKey is the credential. Empty leaves the provider to fail with an
	// authentication error.
	APIKey string
	// CredentialSource is where APIKey came from, such as "auth.json" or
	// "OPENAI_API_KEY". With it, a key the provider's response refuses is
	// an error naming the provider and the source (0005-PLAN F4-3a).
	CredentialSource string
	// Model is the model id. Empty keeps the provider's own.
	Model string
	// ClientName and ClientVersion lead the User-Agent (0003-MADR).
	ClientName, ClientVersion string
	Logger                    *slog.Logger
	// HTTPClient replaces the provider's client, for tests. Headers, when
	// set, wraps it.
	HTTPClient *http.Client
	// ModelMetadataURL replaces List's model metadata document, for
	// tests; New never fetches it.
	ModelMetadataURL string
	// SessionID names the conversation: the routing key a cached request
	// carries, and OpenCode's and Kilo's session headers (0021-MADR D4).
	// Empty keeps the SDK's random id, fixed for the provider.
	SessionID string
	// BaseURL replaces the provider's base URL, as the SDK's WithBaseURL
	// takes it; empty keeps the provider's (0005-PLAN F4-2b).
	BaseURL string
	// Headers, when set, gives the headers each request adds, resolved for
	// that request and its model, "" for a listing; an error ends the
	// request unsent (0005-PLAN F4-2b, deviation 1). They ride on the SDK's
	// own client, or on HTTPClient when set, so a provider with headers keeps
	// the SDK's transport and its limits (0041-MADR D1).
	Headers func(ctx context.Context, model string) (map[string]string, error)
	// Logins keeps OAuth logins. With no APIKey, a provider with an OAuth
	// login (HasLogin) uses its stored login, and saves each rotation here
	// (0005-PLAN F4-3b).
	Logins LoginStore
	// Retry is the session's retry settings; nil is llm.DefaultRetry. Off
	// builds the provider without the SDK's retry (0005-PLAN F5-1).
	Retry *llm.Retry
	// LastGood and LastGoodAt are a listing kept from an earlier run, its
	// Usable ids and its FetchedAt, which List serves as a stale listing
	// after a transient failure, within 24 hours (0025-MADR D4).
	LastGood   []string
	LastGoodAt time.Time
	// Workspace is the Anthropic workspace a Claude key not scoped to one
	// runs in; empty sends none. Other providers ignore it (0035-MADR D5).
	Workspace string
	// Cache, when set, is the run's SDK cache: the provider, its listings,
	// its model lookups and the sign-in's discovery use it in place of the
	// process's, and the run closes it at its end (0046-MADR D7).
	Cache *Cache
	// Sessions, when set, holds the run's OAuth sessions: a stored login is
	// loaded once per provider and shared, and the run waits for their
	// refreshes at its end (0046-MADR D8).
	Sessions *Sessions
}

// login is provider id's stored login, as the SDK's session, when o has no
// key and id has an OAuth login; nil otherwise. With o.Sessions, it is the
// run's session for id, loaded on first use.
func (o Options) login(ctx context.Context, id string) (*auth.OAuthSession, error) {
	if o.APIKey != "" || o.Logins == nil || !HasLogin(id) {
		return nil, nil
	}
	load := func() (*auth.OAuthSession, error) {
		l, err := o.Logins.Load(ctx, id)
		if err != nil || l == nil {
			return nil, err
		}
		return l.session(id, o.Logins, nil, o.Logger), nil
	}
	if o.Sessions != nil {
		return o.Sessions.get(id, load)
	}
	return load()
}

// modelKey is the context key of a request's model, which Stream sets for
// the header transport.
type modelKey struct{}

// requestModel is the model Stream put in ctx, or "".
func requestModel(ctx context.Context) string {
	m, _ := ctx.Value(modelKey{}).(string) //nolint:errcheck // absent is a listing's request, which has no model
	return m
}

// sdkOptions are o's transport options for the SDK: its client, wrapped to
// add o.Headers, its base URL, and its cache.
func (o Options) sdkOptions() []llmprovider.Option {
	var opts []llmprovider.Option
	client := o.HTTPClient
	if o.Headers != nil {
		client = withHeaders(client, o.Headers)
	}
	if client != nil {
		opts = append(opts, llmprovider.WithHTTPClient(client))
	}
	if o.BaseURL != "" {
		opts = append(opts, llmprovider.WithBaseURL(o.BaseURL))
	}
	if o.Cache != nil {
		opts = append(opts, o.Cache.option())
	}
	return opts
}

// headerTransport adds each request's headers, resolved for it, before base
// sends it.
type headerTransport struct {
	base    http.RoundTripper
	headers func(ctx context.Context, model string) (map[string]string, error)
}

func (t headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	h, err := t.headers(r.Context(), requestModel(r.Context()))
	if err != nil {
		if r.Body != nil {
			_ = r.Body.Close() //nolint:errcheck // the request is not sent; its error is the one returned
		}
		// A header that cannot be resolved is a request that cannot be built
		// as given: the SDK's retry does not resend it.
		return nil, fmt.Errorf("%w: %w", llmprovider.ErrInvalidRequest, err)
	}
	r = r.Clone(r.Context())
	for k, v := range h {
		r.Header.Set(k, v)
	}
	return t.base.RoundTrip(r)
}

// withHeaders is a copy of c, or of the SDK's own client when c is nil,
// whose transport adds headers. The SDK's client keeps its shared, bounded
// transport, so a provider with headers has the limits of one without; a nil
// Transport in c is net/http's default, as http.Client reads it (0041-MADR
// D1, D2).
func withHeaders(c *http.Client, headers func(ctx context.Context, model string) (map[string]string, error)) *http.Client {
	if c == nil {
		c = llmprovider.NewHTTPClient()
	}
	out := *c
	base := out.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	out.Transport = headerTransport{base: base, headers: headers}
	return &out
}

// New builds the SDK provider id with o, wrapped in the SDK's retry, as an
// llm.Provider. Construction makes no network call: model probes and the
// model metadata fetch are off (0005-MADR).
func New(id string, o Options) (llm.Provider, error) {
	opts := []llmprovider.Option{
		llmprovider.WithModelProbes(false),
		llmprovider.WithoutModelMetadata(),
		llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),
	}
	if o.APIKey != "" {
		opts = append(opts, llmprovider.WithAPIKey(o.APIKey))
	}
	login, err := o.login(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", id, err)
	}
	if login != nil {
		opts = append(opts, llmprovider.WithTokenSource(login))
	}
	if o.Model != "" {
		opts = append(opts, llmprovider.WithModel(o.Model))
	}
	if o.Logger != nil {
		opts = append(opts, llmprovider.WithLogger(o.Logger))
	}
	opts = append(opts, o.sdkOptions()...)
	if o.Workspace != "" {
		opts = append(opts, llmprovider.For(llmprovider.ProviderClaude, claude.WithWorkspace(o.Workspace)))
	}
	if o.SessionID != "" {
		opts = append(opts, llmprovider.WithSessionID(o.SessionID))
	}
	p, err := providers.New(llmprovider.ProviderID(id), opts...)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", id, mapError(err))
	}
	// The retry wrapper passes the stream on as it arrives, retrying only a
	// failure before its first event, and keeps the provider's capabilities,
	// native streaming included (0031-MADR D2).
	r := llm.DefaultRetry
	if o.Retry != nil {
		r = *o.Retry
	}
	w := p
	if !r.Off {
		w = llmprovider.WithStreamRetry(p, policyOf(r), notice)
	}
	return &adapter{p: w, id: p.ID(), caps: w.Capabilities(), source: o.CredentialSource, login: login != nil}, nil
}

// noticeKey carries a call's hook for the retry notices of its stream.
type noticeKey struct{}

// notice hands n to the hook of the call whose stream it comes from. The
// SDK calls it on the goroutine ranging that stream, before its wait.
func notice(ctx context.Context, n llmprovider.RetryNotice) {
	if hook, ok := ctx.Value(noticeKey{}).(func(llmprovider.RetryNotice)); ok {
		hook(n)
	}
}

// Wrap adapts an SDK provider to llm.Provider, with no retry of its own.
func Wrap(p llmprovider.Provider) llm.Provider {
	return &adapter{p: p, id: p.ID(), caps: p.Capabilities()}
}

type adapter struct {
	p    llmprovider.Provider
	id   llmprovider.ProviderID
	caps llmprovider.Capabilities
	// source is where the key came from, or "" (0005-PLAN F4-3a).
	source string
	// login reports that the credential is a stored OAuth login (0005-PLAN
	// F4-3b).
	login bool
	// mu guards offlineURL: the URL of the last request the network could
	// not carry, whose host Probe dials (0005-PLAN F5-1, deviation 1).
	mu         sync.Mutex
	offlineURL string
}

// providerRetry is n as llm's event: its try, the wait taken, and the
// failure's class, with the wait the provider asked for, whether it asked,
// and the failure's code and status (0031-MADR D4; 0049-MADR D6).
func (a *adapter) providerRetry(n llmprovider.RetryNotice) llm.ProviderRetry {
	mapped := mapErrorAt(a.id, n.Err, time.Now())
	pr := llm.ProviderRetry{Attempt: n.Attempt, MaxAttempts: n.MaxAttempts, Wait: n.Wait, Kind: FailureKind(mapped),
		ServerDirected: n.ServerDirected}
	if ae, ok := errors.AsType[*llm.APIError](mapped); ok {
		pr.RetryAfter, pr.Code, pr.Status = ae.RetryAfter, ae.Code, ae.Status
	}
	return pr
}

// rejected is mapped, err's llm form, led by "<provider> rejected the API
// key from <source>" when the provider's response refused the key: an
// APIError of ErrAuthFailure that is not ErrNotPermitted, as the SDK's
// wizard tells a rejected key (wizard/configure.go). For a login, any
// ErrAuthFailure, a refused refresh too, is led by LoginRejected's sentence
// (0005-PLAN F4-3b decision 8). Anything else, or no source, is mapped
// alone.
func (a *adapter) rejected(mapped, err error) error {
	// A quota some services send with HTTP 401, such as OpenCode's
	// MonthlyLimitError, also unwraps to ErrAuthFailure; it is a quota, not a
	// refused credential (0049-MADR D8).
	if errors.Is(err, llmprovider.ErrQuotaExhausted) {
		return mapped
	}
	if a.login && errors.Is(err, llmprovider.ErrAuthFailure) && !errors.Is(err, llmprovider.ErrNotPermitted) {
		return fmt.Errorf("%s: %w", LoginRejected(string(a.id)), mapped)
	}
	var ae *llmprovider.APIError
	response := errors.As(err, &ae)
	if a.source == "" || !response || !errors.Is(err, llmprovider.ErrAuthFailure) || errors.Is(err, llmprovider.ErrNotPermitted) {
		return mapped
	}
	return fmt.Errorf("%s rejected the API key from %s: %w", a.id, a.source, mapped)
}

func (a *adapter) ID() string { return string(a.id) }

func (a *adapter) Capabilities() llm.Capabilities {
	return llm.Capabilities{NativeStreaming: a.caps.NativeStreaming == llmprovider.Supported}
}

// Stream maps the SDK's stream onto llm's sealed events, as llm's package
// documentation lists. Another model's messages go without its data
// (0025-MADR D2); a request refused over a signature despite that, before
// any event, is sent once more with no thinking at all (0025-MADR Q4).
func (a *adapter) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	return func(yield func(llm.Event, error) bool) {
		msgs := replay(req.Messages, string(a.id), req.Model)
		if a.attempt(ctx, req, msgs, true, yield) {
			a.attempt(ctx, req, stripThinking(msgs), false, yield)
		}
	}
}

// attempt streams req with msgs, and reports whether it was refused over a
// signature before any event, when canRetry, instead of yielding that.
func (a *adapter) attempt(ctx context.Context, req *llm.Request, msgs []llm.Message, canRetry bool, yield func(llm.Event, error) bool) (retry bool) {
	r := *req
	r.Messages = msgs
	sreq, err := toRequest(&r)
	if err != nil {
		yield(nil, err)
		return false
	}
	var out strings.Builder
	calls, sent, stopped := false, false, false
	// tried is the last retry notice's try, so a failure after retries says
	// how many tries were made (0049-MADR D6).
	tried := 0
	ctx = context.WithValue(ctx, modelKey{}, req.Model) // for the header transport (0005-PLAN F4-2b)
	// The provider's retries are yielded as they are made; a caller that
	// stops at one ends the wait at once (0031-MADR D4).
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = context.WithValue(ctx, noticeKey{}, func(n llmprovider.RetryNotice) {
		tried = n.Attempt
		if !stopped && !yield(a.providerRetry(n), nil) {
			stopped = true
			cancel()
		}
	})
	for ev, err := range llmprovider.Stream(ctx, a.p, sreq) {
		if stopped {
			return false
		}
		if err != nil {
			if canRetry && !sent && signatureRefused(mapError(err)) {
				return true
			}
			if ue, ok := errors.AsType[*url.Error](err); ok && errors.Is(err, llmprovider.ErrOffline) {
				a.mu.Lock()
				a.offlineURL = ue.URL
				a.mu.Unlock()
			}
			if refusedAnswer(err) { // a refusal, as one with text is (0005-PLAN F5-3 decision 5)
				yield(llm.Done{StopReason: llm.StopRefusal}, nil)
				return false
			}
			mapped := mapErrorAt(a.id, err, time.Now())
			if ae, ok := errors.AsType[*llm.APIError](mapped); ok && tried > 0 {
				ae.Attempts = tried + 1
			}
			yield(nil, a.rejected(mapped, err))
			return false
		}
		sent = true
		var e llm.Event
		switch ev.Type {
		case llmprovider.EventTextDelta:
			out.WriteString(ev.Text)
			e = llm.TextDelta{Text: ev.Text}
		case llmprovider.EventReasoningDelta:
			e = llm.ThinkingDelta{Text: ev.Text}
		case llmprovider.EventItem:
			switch it := ev.Item.(type) {
			case llmprovider.FunctionCallItem:
				calls = true
				e = llm.ToolCallDone{ID: it.CallID, Name: it.Name, Args: it.Arguments, ProviderData: signatureData(it.Signature)}
			case llmprovider.ReasoningItem:
				e = llm.ThinkingDelta{ProviderData: reasoningData(it)}
			}
		case llmprovider.EventDone:
			if !yield(usageOf(ev.Response, sreq, out.String()), nil) {
				return false
			}
			e = llm.Done{StopReason: stopOf(ev.Response, calls)}
		}
		if e != nil && !yield(e, nil) {
			return false
		}
	}
	return false
}

// reasoning is a ReasoningItem as ProviderData carries it for replay.
type reasoning struct {
	Text      string `json:"text,omitzero"`
	Signature string `json:"signature,omitzero"`
	Encrypted string `json:"encrypted,omitzero"`
	Format    string `json:"format,omitzero"`
}

// reasoningData is it as reasoning's JSON. jsonx.Fields writes what
// Marshal writes for reasoning's omitzero strings, and cannot fail
// (0039-PLAN deviation 1).
func reasoningData(it llmprovider.ReasoningItem) jsontext.Value {
	return jsonx.Fields(jsonx.Field{Name: "text", Value: it.Text}, jsonx.Field{Name: "signature", Value: it.Signature},
		jsonx.Field{Name: "encrypted", Value: it.Encrypted}, jsonx.Field{Name: "format", Value: it.Format})
}

func signatureData(sig string) jsontext.Value {
	if sig == "" {
		return nil
	}
	return reasoningData(llmprovider.ReasoningItem{Signature: sig})
}

func stopOf(r *llmprovider.Response, calls bool) llm.StopReason {
	switch {
	case calls:
		return llm.StopToolUse
	case r != nil && r.FinishReason == llmprovider.FinishLength:
		return llm.StopMaxTokens
	case r != nil && r.FinishReason == llmprovider.FinishContentFilter:
		return llm.StopRefusal // every wire's safety and refusal stops (0005-PLAN F5-3 decision 5)
	}
	return llm.StopEndTurn
}

// usageOf is the response's token count. An all-zero report is replaced by
// an estimate of four bytes per token, marked Estimated (llm's doc).
func usageOf(r *llmprovider.Response, req *llmprovider.Request, output string) llm.Usage {
	if r != nil {
		u := r.Usage
		if u.InputTokens != 0 || u.OutputTokens != 0 || u.ReasoningTokens != 0 || u.CachedTokens != 0 || u.CacheWriteTokens != 0 {
			return llm.Usage{
				InputTokens: int64(u.InputTokens), OutputTokens: int64(u.OutputTokens),
				ReasoningTokens: int64(u.ReasoningTokens), CacheReadTokens: int64(u.CachedTokens),
				CacheWriteTokens: int64(u.CacheWriteTokens),
			}
		}
	}
	in := len(req.Instructions)
	for _, it := range req.Input {
		switch it := it.(type) {
		case llmprovider.MessageItem:
			in += len(it.Text)
		case llmprovider.FunctionCallItem:
			in += len(it.Arguments)
		case llmprovider.FunctionCallOutputItem:
			in += len(it.Output)
		case llmprovider.ReasoningItem:
			in += len(it.Text)
		}
	}
	return llm.Usage{InputTokens: int64((in + 3) / 4), OutputTokens: int64((len(output) + 3) / 4), Estimated: true}
}

// toolSpec is the part of a tool.Spec the SDK sends.
type toolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// toolSpecs decodes a request's encoded specs; what names says is where
// they came from.
func toolSpecs(v jsontext.Value, what string) ([]toolSpec, error) {
	if len(v) == 0 {
		return nil, nil
	}
	var specs []toolSpec
	if err := jsonx.Unmarshal(v, &specs); err != nil {
		return nil, fmt.Errorf("provider: request %s: %w", what, err)
	}
	return specs, nil
}

// toRequest maps llm's request onto the SDK's. Deferred tools go as the
// SDK's deferred tools, after the others, the Search tool as its search,
// and each load a result names as a ToolLoadItem after that result; a
// loaded tool whose load no result here carries stays an ordinary tool; a
// request that defers nothing is sent as Tools has it (0012-MADR D16, D17).
func toRequest(req *llm.Request) (*llmprovider.Request, error) {
	out := &llmprovider.Request{Model: req.Model, Reasoning: reasoningFor(req.Thinking), Cache: llmprovider.CacheRetention(req.Cache),
		MaxOutputTokens: req.MaxTokens}
	deferred, err := toolSpecs(req.Deferred, "deferred tools")
	if err != nil {
		return nil, err
	}
	isDeferred, replayed := map[string]bool{}, map[string]bool{} // replayed: loaded by an item this request sends
	for _, s := range deferred {
		isDeferred[s.Name] = true
	}
	var system []string
	for _, m := range req.Messages {
		for _, c := range m.Content {
			switch {
			case m.Role == llm.RoleSystem:
				system = append(system, c.Text)
			case c.Type == llm.ContentText:
				role := llmprovider.RoleUser
				if m.Role == llm.RoleAssistant {
					role = llmprovider.RoleAssistant
				}
				out.Input = append(out.Input, llmprovider.MessageItem{Role: role, Text: c.Text})
			case c.Type == llm.ContentThinking:
				it := llmprovider.ReasoningItem{Text: c.Text}
				var r reasoning
				if len(c.ProviderData) > 0 && jsonx.Unmarshal(c.ProviderData, &r) == nil {
					it = llmprovider.ReasoningItem{Text: r.Text, Signature: r.Signature, Encrypted: r.Encrypted, Format: r.Format}
				}
				out.Input = append(out.Input, it)
			case c.Type == llm.ContentToolCall:
				var r reasoning
				if len(c.ProviderData) > 0 {
					_ = jsonx.Unmarshal(c.ProviderData, &r) //nolint:errcheck // an unreadable signature is sent as none
				}
				out.Input = append(out.Input, llmprovider.FunctionCallItem{CallID: c.ToolCallID, Name: c.ToolName, Arguments: c.ToolArgs, Signature: r.Signature})
			case c.Type == llm.ContentToolResult:
				out.Input = append(out.Input, llmprovider.FunctionCallOutputItem{CallID: c.ToolCallID, Output: c.Text, IsError: c.IsError})
				// A load the request does not defer was sent as an
				// ordinary tool, and has nothing to load (0012-MADR D17).
				for _, n := range c.Loads {
					if isDeferred[n] {
						out.Input = append(out.Input, llmprovider.ToolLoadItem{CallID: c.ToolCallID, Name: n})
						replayed[n] = true
					}
				}
			}
		}
	}
	out.Instructions = strings.Join(system, "\n\n")
	specs, err := toolSpecs(req.Tools, "tools")
	if err != nil {
		return nil, err
	}
	// A loaded tool, in Tools and Deferred, goes as deferred only when a
	// result this request sends carries its load. Otherwise its load went
	// with a compaction, a turn replay does not send, or a session from
	// before loads named their call, and it stays an ordinary tool in its
	// place (0012-MADR D17).
	ordinary := map[string]bool{}
	for _, s := range specs {
		if isDeferred[s.Name] && replayed[s.Name] {
			continue // sent below, as deferred
		}
		ordinary[s.Name] = true
		out.Tools = append(out.Tools, llmprovider.Tool{Name: s.Name, Description: s.Description, Schema: s.InputSchema,
			Strict: strictFor(s.InputSchema), Search: s.Name == req.Search})
	}
	sent := false
	for _, s := range deferred {
		if ordinary[s.Name] {
			continue
		}
		sent = true
		out.Tools = append(out.Tools, llmprovider.Tool{Name: s.Name, Description: s.Description, Schema: s.InputSchema,
			Strict: strictFor(s.InputSchema), Deferred: true})
	}
	if !sent {
		// Nothing is deferred, so there is no search: today's request.
		for i := range out.Tools {
			out.Tools[i].Search = false
		}
	}
	return out, nil
}

// strictFor is a tool's strict flag (0012-MADR D3): false when its schema
// has a property a call may leave out, at any depth, so that strict mode
// does not make the model fill it in; unset otherwise, the service's own
// default. It is never true (0019-MADR D11).
func strictFor(schema any) *bool {
	if hasOptional(schema) {
		return new(false)
	}
	return nil
}

// hasOptional reports whether schema, a decoded JSON Schema, or an object
// schema in its properties or items, has a property outside its required.
func hasOptional(schema any) bool {
	s, ok := schema.(map[string]any)
	if !ok {
		return false
	}
	if props, ok := s["properties"].(map[string]any); ok {
		required, listed := s["required"].([]any)
		for name, sub := range props {
			if !listed || !slices.Contains(required, any(name)) || hasOptional(sub) {
				return true
			}
		}
	}
	return hasOptional(s["items"])
}

// reasoningFor maps gobble's thinking levels onto the SDK's efforts (llm's
// doc); "" and "off" ask for none.
func reasoningFor(level string) *llmprovider.Reasoning {
	effort := map[string]llmprovider.Effort{
		"minimal": llmprovider.EffortLow, "low": llmprovider.EffortLow, "medium": llmprovider.EffortMedium,
		"high": llmprovider.EffortHigh, "xhigh": llmprovider.EffortXHigh, "max": llmprovider.EffortMax,
	}[level]
	if effort == "" {
		return nil
	}
	return &llmprovider.Reasoning{Effort: effort}
}

// sentinels maps the SDK's kinds onto llm's, most specific first, because
// the SDK's overlap (llm's doc).
var sentinels = []struct{ sdk, llm error }{
	{llmprovider.ErrWorkspaceRequired, llm.ErrWorkspace},
	{llmprovider.ErrWorkspaceNotFound, llm.ErrWorkspace},
	{llmprovider.ErrQuotaExhausted, llm.ErrQuota},
	{llmprovider.ErrRateLimited, llm.ErrRateLimited},
	{llmprovider.ErrContextOverflow, llm.ErrContextOverflow},
	{llmprovider.ErrIncomplete, llm.ErrIncomplete},
	{llmprovider.ErrAuthFailure, llm.ErrAuth},
	{llmprovider.ErrNotPermitted, llm.ErrNotPermitted},
	// Offline also matches ErrProviderUnavailable since the SDK's v1.23.0,
	// so it comes first: the agent waits for the network on llm.ErrOffline
	// (0035-MADR D3).
	{llmprovider.ErrOffline, llm.ErrOffline},
	{llmprovider.ErrProviderUnavailable, llm.ErrUnavailable},
	{llmprovider.ErrUnsupported, llm.ErrUnsupported},
}

// emptyAnswer reports whether err is the SDK's answer with nothing in it
// that is not a refusal or a safety stop, which the SDK puts in Reason:
// ErrEmptyAnswer with no reason, or stop (0005-PLAN F5-1 decision 4;
// 0035-MADR D4).
func emptyAnswer(err error) bool {
	ae, ok := errors.AsType[*llmprovider.APIError](err)
	return ok && errors.Is(ae.Kind, llmprovider.ErrEmptyAnswer) && (ae.Reason == "" || ae.Reason == string(llmprovider.FinishStop))
}

// refusedAnswer reports whether err is the SDK's answer with nothing in it
// that the service stopped for safety, or the model refused: its Reason is
// content_filter (0005-PLAN F5-3 decision 5).
func refusedAnswer(err error) bool {
	ae, ok := errors.AsType[*llmprovider.APIError](err)
	return ok && errors.Is(ae.Kind, llmprovider.ErrEmptyAnswer) && ae.Reason == string(llmprovider.FinishContentFilter)
}

// probeTimeout bounds one Probe's dial (Kilo's PROBE_MS, 0005-PLAN F5-1
// decision 3).
const probeTimeout = 5 * time.Second

// dialProbe is Probe's dial. Tests replace it.
var dialProbe = (&net.Dialer{}).DialContext

// Probe dials the host the last request the network could not carry went
// to: the proxy the environment names for its URL, or else its own host.
// It sends nothing (0005-PLAN F5-1 decision 3, deviation 1).
func (a *adapter) Probe(ctx context.Context) error {
	a.mu.Lock()
	target := a.offlineURL
	a.mu.Unlock()
	if target == "" {
		return fmt.Errorf("provider %s: no failed request to probe for", a.id)
	}
	u, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("provider %s: %w", a.id, err)
	}
	if proxy, err := http.ProxyFromEnvironment(&http.Request{URL: u}); err == nil && proxy != nil {
		u = proxy
	}
	port := u.Port()
	if port == "" {
		port = "443"
		if u.Scheme == "http" {
			port = "80"
		}
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	c, err := dialProbe(ctx, "tcp", net.JoinHostPort(u.Hostname(), port))
	if err != nil {
		return err
	}
	return c.Close()
}

// mapError is mapErrorAt for an error no provider of this package made,
// such as providers.New's.
func mapError(err error) error {
	return mapErrorAt("", err, time.Now())
}

// mapErrorAt wraps an SDK error so llm's sentinel matches and the SDK's
// still does. An APIError becomes an *llm.APIError with its status and
// RetryAfter, and what the service said: provider id's name, its code and
// message, whether a retry can succeed, the limit its code states, and, at
// now plus RetryAfter, when the limit lifts (0049-MADR D4 to D6).
func mapErrorAt(id llmprovider.ProviderID, err error, now time.Time) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	wrapped := err
	for _, s := range sentinels {
		if errors.Is(err, s.sdk) {
			wrapped = fmt.Errorf("%w: %w", s.llm, err)
			break
		}
	}
	if emptyAnswer(err) {
		wrapped = fmt.Errorf("%w: %w", llm.ErrEmptyReply, wrapped)
	}
	if ae, ok := errors.AsType[*llmprovider.APIError](err); ok {
		retryable := ae.Retryable()
		out := &llm.APIError{Status: ae.Status, RetryAfter: ae.RetryAfter, Err: wrapped, Provider: string(id),
			Code: ae.Code, Message: ae.Message, Retryable: &retryable, Limit: limitOf(id, ae)}
		if ae.RetryAfter > 0 {
			out.ResetAt = now.Add(ae.RetryAfter)
		}
		return out
	}
	return wrapped
}

// The provider codes that state which limit a quota is (0049-MADR D4;
// go-llmprovider-sdk v1.28.0's classifyAPIError names them a quota).
var (
	freeTierCodes = []string{"FreeUsageLimitError", "PROMOTION_MODEL_LIMIT_REACHED"}
	monthlyCodes  = []string{"MonthlyLimitError"}
	creditsCodes  = []string{"CreditsError", "insufficient_quota", "insufficient_balance"}
	planCodes     = []string{"GoUsageLimitError", "BlackUsageLimitError", "usage_limit_reached",
		"subscription_sharing_usage_limit_exceeded"}
)

// limitOf is the limit ae states: from the provider's code, HTTP 402 for
// credits, Gemini's exhausted quota on 429 for a day (the SDK makes a
// Gemini 429 a quota only when it names a per-day one), and a rate limit for
// any other ErrRateLimited. It is "" when nothing states the limit, and it
// never reads RetryAfter: a long wait is not a window. id is the provider,
// or "" when not known, when the SDK's own name for it is read.
func limitOf(id llmprovider.ProviderID, ae *llmprovider.APIError) string {
	if !errors.Is(ae, llmprovider.ErrQuotaExhausted) {
		if errors.Is(ae, llmprovider.ErrRateLimited) {
			return llm.LimitRate
		}
		return ""
	}
	service := string(id)
	if service == "" {
		service, _, _ = strings.Cut(ae.Provider, "/")
	}
	switch {
	case slices.Contains(freeTierCodes, ae.Code):
		return llm.LimitFreeTier
	case slices.Contains(monthlyCodes, ae.Code):
		return llm.LimitMonthly
	case slices.Contains(creditsCodes, ae.Code), ae.Status == http.StatusPaymentRequired:
		return llm.LimitCredits
	case slices.Contains(planCodes, ae.Code):
		return llm.LimitPlan
	case service == string(llmprovider.ProviderGemini) && ae.Status == http.StatusTooManyRequests:
		return llm.LimitDaily
	}
	return ""
}
