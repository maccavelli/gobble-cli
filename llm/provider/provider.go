package provider

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"

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
	// HTTPClient replaces the provider's client, for tests.
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
	// request unsent (0005-PLAN F4-2b, deviation 1).
	Headers func(ctx context.Context, model string) (map[string]string, error)
	// Logins keeps OAuth logins. With no APIKey, a provider with an OAuth
	// login (HasLogin) uses its stored login, and saves each rotation here
	// (0005-PLAN F4-3b).
	Logins LoginStore
	// Retry is the session's retry settings; nil is llm.DefaultRetry. Off
	// builds the provider without the SDK's retry (0005-PLAN F5-1).
	Retry *llm.Retry
}

// login is provider id's stored login, as the SDK's session, when o has no
// key and id has an OAuth login; nil otherwise.
func (o Options) login(ctx context.Context, id string) (*auth.OAuthSession, error) {
	if o.APIKey != "" || o.Logins == nil || !HasLogin(id) {
		return nil, nil
	}
	l, err := o.Logins.Load(ctx, id)
	if err != nil || l == nil {
		return nil, err
	}
	return l.session(id, o.Logins, nil, o.Logger), nil
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
// add o.Headers, and its base URL.
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

// withHeaders is a copy of c, or of a default client, whose transport adds
// headers.
func withHeaders(c *http.Client, headers func(ctx context.Context, model string) (map[string]string, error)) *http.Client {
	out := &http.Client{}
	base := http.DefaultTransport
	if c != nil {
		*out = *c
		if c.Transport != nil {
			base = c.Transport
		}
	}
	out.Transport = headerTransport{base: base, headers: headers}
	return out
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
	if o.SessionID != "" {
		opts = append(opts, llmprovider.WithSessionID(o.SessionID))
	}
	p, err := providers.New(llmprovider.ProviderID(id), opts...)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", id, mapError(err))
	}
	// The retry wrapper buffers the stream, so its capabilities, not the
	// inner provider's, are what gobble can deliver (0019-MADR D2).
	r := llm.DefaultRetry
	if o.Retry != nil {
		r = *o.Retry
	}
	w := p
	if !r.Off {
		w = llmprovider.WithRetry(p, policyOf(r))
	}
	return &adapter{p: w, id: p.ID(), caps: w.Capabilities(), source: o.CredentialSource, login: login != nil}, nil
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

// rejected is mapped, err's llm form, led by "<provider> rejected the API
// key from <source>" when the provider's response refused the key: an
// APIError of ErrAuthFailure that is not ErrNotPermitted, as the SDK's
// wizard tells a rejected key (wizard/configure.go). For a login, any
// ErrAuthFailure, a refused refresh too, is led by LoginRejected's sentence
// (0005-PLAN F4-3b decision 8). Anything else, or no source, is mapped
// alone.
func (a *adapter) rejected(mapped, err error) error {
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
// documentation lists.
func (a *adapter) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	return func(yield func(llm.Event, error) bool) {
		sreq, err := toRequest(req)
		if err != nil {
			yield(nil, err)
			return
		}
		var out strings.Builder
		calls := false
		ctx := context.WithValue(ctx, modelKey{}, req.Model) // for the header transport (0005-PLAN F4-2b)
		for ev, err := range llmprovider.Stream(ctx, a.p, sreq) {
			if err != nil {
				if ue, ok := errors.AsType[*url.Error](err); ok && offline(err) {
					a.mu.Lock()
					a.offlineURL = ue.URL
					a.mu.Unlock()
				}
				yield(nil, a.rejected(mapError(err), err))
				return
			}
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
					return
				}
				e = llm.Done{StopReason: stopOf(ev.Response, calls)}
			}
			if e != nil && !yield(e, nil) {
				return
			}
		}
	}
}

// reasoning is a ReasoningItem as ProviderData carries it for replay.
type reasoning struct {
	Text      string `json:"text,omitzero"`
	Signature string `json:"signature,omitzero"`
	Encrypted string `json:"encrypted,omitzero"`
	Format    string `json:"format,omitzero"`
}

func reasoningData(it llmprovider.ReasoningItem) jsontext.Value {
	b, err := json.Marshal(reasoning{Text: it.Text, Signature: it.Signature, Encrypted: it.Encrypted, Format: it.Format})
	if err != nil { // a struct of strings always encodes
		panic(err)
	}
	return b
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

// toRequest maps llm's request onto the SDK's.
func toRequest(req *llm.Request) (*llmprovider.Request, error) {
	out := &llmprovider.Request{Model: req.Model, Reasoning: reasoningFor(req.Thinking), Cache: llmprovider.CacheRetention(req.Cache)}
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
				if len(c.ProviderData) > 0 && json.Unmarshal(c.ProviderData, &r) == nil {
					it = llmprovider.ReasoningItem{Text: r.Text, Signature: r.Signature, Encrypted: r.Encrypted, Format: r.Format}
				}
				out.Input = append(out.Input, it)
			case c.Type == llm.ContentToolCall:
				var r reasoning
				if len(c.ProviderData) > 0 {
					_ = json.Unmarshal(c.ProviderData, &r) //nolint:errcheck // an unreadable signature is sent as none
				}
				out.Input = append(out.Input, llmprovider.FunctionCallItem{CallID: c.ToolCallID, Name: c.ToolName, Arguments: c.ToolArgs, Signature: r.Signature})
			case c.Type == llm.ContentToolResult:
				out.Input = append(out.Input, llmprovider.FunctionCallOutputItem{CallID: c.ToolCallID, Output: c.Text, IsError: c.IsError})
			}
		}
	}
	out.Instructions = strings.Join(system, "\n\n")
	if len(req.Tools) > 0 {
		var specs []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema any    `json:"inputSchema"`
		}
		if err := json.Unmarshal(req.Tools, &specs); err != nil {
			return nil, fmt.Errorf("provider: request tools: %w", err)
		}
		for _, s := range specs {
			out.Tools = append(out.Tools, llmprovider.Tool{Name: s.Name, Description: s.Description, Schema: s.InputSchema, Strict: strictFor(s.InputSchema)})
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
		"high": llmprovider.EffortHigh, "xhigh": llmprovider.EffortXHigh, "max": llmprovider.EffortXHigh,
	}[level]
	if effort == "" {
		return nil
	}
	return &llmprovider.Reasoning{Effort: effort}
}

// sentinels maps the SDK's kinds onto llm's, most specific first, because
// the SDK's overlap (llm's doc).
var sentinels = []struct{ sdk, llm error }{
	{llmprovider.ErrQuotaExhausted, llm.ErrQuota},
	{llmprovider.ErrRateLimited, llm.ErrRateLimited},
	{llmprovider.ErrContextOverflow, llm.ErrContextOverflow},
	{llmprovider.ErrIncomplete, llm.ErrIncomplete},
	{llmprovider.ErrAuthFailure, llm.ErrAuth},
	{llmprovider.ErrNotPermitted, llm.ErrNotPermitted},
	{llmprovider.ErrProviderUnavailable, llm.ErrUnavailable},
	{llmprovider.ErrUnsupported, llm.ErrUnsupported},
}

// sdkEmptyAnswer ends the message of the SDK's answer with no content
// (go-llmprovider-sdk internal/wire/finish.go, EmptyAnswer). The wire tests
// pin it, so an SDK that words it otherwise fails them.
const sdkEmptyAnswer = ": the answer has no content"

// emptyAnswer reports whether err is the SDK's answer with nothing in it
// that is not a refusal or a safety stop, which the SDK puts in Reason: an
// APIError of ErrIncomplete with the SDK's message and no reason, or stop
// (0005-PLAN F5-1 decision 4).
func emptyAnswer(err error) bool {
	ae, ok := errors.AsType[*llmprovider.APIError](err)
	return ok && errors.Is(ae.Kind, llmprovider.ErrIncomplete) && strings.HasSuffix(ae.Message, sdkEmptyAnswer) &&
		(ae.Reason == "" || ae.Reason == string(llmprovider.FinishStop))
}

// offline reports whether err is a request that could not be sent for want
// of a network: a *url.Error whose host's name did not resolve, or whose
// dial timed out or found the network or host unreachable or the port
// refused (0005-PLAN F5-1 decision 3).
func offline(err error) bool {
	if _, ok := errors.AsType[*url.Error](err); !ok { //nolint:errcheck // only whether the chain holds one matters
		return false
	}
	if _, ok := errors.AsType[*net.DNSError](err); ok { //nolint:errcheck // only whether the chain holds one matters
		return true
	}
	op, ok := errors.AsType[*net.OpError](err)
	if !ok || op.Op != "dial" {
		return false
	}
	if op.Timeout() {
		return true
	}
	return slices.ContainsFunc(unreachable(), func(e syscall.Errno) bool { return errors.Is(err, e) })
}

// unreachable are the dial errors offline counts: refused, network down
// or unreachable, host unreachable. Windows reports them as Winsock's
// codes, which syscall names for no other system.
func unreachable() []syscall.Errno {
	if runtime.GOOS == "windows" {
		const wsaENETDOWN, wsaENETUNREACH, wsaECONNREFUSED, wsaEHOSTUNREACH = 10050, 10051, 10061, 10065
		return []syscall.Errno{wsaECONNREFUSED, wsaENETDOWN, wsaENETUNREACH, wsaEHOSTUNREACH}
	}
	return []syscall.Errno{syscall.ECONNREFUSED, syscall.ENETDOWN, syscall.ENETUNREACH, syscall.EHOSTUNREACH}
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

// mapError wraps an SDK error so llm's sentinel matches and the SDK's still
// does. An APIError becomes an *llm.APIError with its status and RetryAfter.
func mapError(err error) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	wrapped, kind := err, false
	for _, s := range sentinels {
		if errors.Is(err, s.sdk) {
			wrapped, kind = fmt.Errorf("%w: %w", s.llm, err), true
			break
		}
	}
	switch {
	case emptyAnswer(err):
		wrapped = fmt.Errorf("%w: %w", llm.ErrEmptyReply, wrapped)
	case !kind && offline(err):
		wrapped = fmt.Errorf("%w: %w", llm.ErrOffline, err)
	}
	if ae, ok := errors.AsType[*llmprovider.APIError](err); ok {
		return &llm.APIError{Status: ae.Status, RetryAfter: ae.RetryAfter, Err: wrapped}
	}
	return wrapped
}
