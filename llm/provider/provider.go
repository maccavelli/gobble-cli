package provider

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"

	"github.com/maccavelli/gobble-cli/llm"
)

// retryPolicy is Pi's 3 retries with a 60 s cap (0004-MADR, F4 notes).
var retryPolicy = llmprovider.RetryPolicy{MaxAttempts: 4, BaseDelay: 2 * time.Second, MaxDelay: 60 * time.Second}

// Options configure New.
type Options struct {
	// APIKey is the credential. Empty leaves the provider to fail with an
	// authentication error.
	APIKey string
	// Model is the model id. Empty keeps the provider's own.
	Model string
	// ClientName and ClientVersion lead the User-Agent (0003-MADR).
	ClientName, ClientVersion string
	Logger                    *slog.Logger
	// HTTPClient replaces the provider's client, for tests.
	HTTPClient *http.Client
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
	if o.Model != "" {
		opts = append(opts, llmprovider.WithModel(o.Model))
	}
	if o.Logger != nil {
		opts = append(opts, llmprovider.WithLogger(o.Logger))
	}
	if o.HTTPClient != nil {
		opts = append(opts, llmprovider.WithHTTPClient(o.HTTPClient))
	}
	p, err := providers.New(llmprovider.ProviderID(id), opts...)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", id, mapError(err))
	}
	// The retry wrapper streams through Generate; ID and capabilities stay
	// the provider's own.
	return &adapter{p: llmprovider.WithRetry(p, retryPolicy), id: p.ID(), caps: p.Capabilities()}, nil
}

// Wrap adapts an SDK provider to llm.Provider, with no retry of its own.
func Wrap(p llmprovider.Provider) llm.Provider {
	return &adapter{p: p, id: p.ID(), caps: p.Capabilities()}
}

type adapter struct {
	p    llmprovider.Provider
	id   llmprovider.ProviderID
	caps llmprovider.Capabilities
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
		for ev, err := range llmprovider.Stream(ctx, a.p, sreq) {
			if err != nil {
				yield(nil, mapError(err))
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
		if u.InputTokens != 0 || u.OutputTokens != 0 || u.ReasoningTokens != 0 || u.CachedTokens != 0 {
			return llm.Usage{
				InputTokens: int64(u.InputTokens), OutputTokens: int64(u.OutputTokens),
				ReasoningTokens: int64(u.ReasoningTokens), CacheReadTokens: int64(u.CachedTokens),
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
	out := &llmprovider.Request{Model: req.Model, Reasoning: reasoningFor(req.Thinking)}
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
				out.Input = append(out.Input, llmprovider.FunctionCallOutputItem{CallID: c.ToolCallID, Output: c.Text})
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
			out.Tools = append(out.Tools, llmprovider.Tool{Name: s.Name, Description: s.Description, Schema: s.InputSchema})
		}
	}
	return out, nil
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

// mapError wraps an SDK error so llm's sentinel matches and the SDK's still
// does. An APIError becomes an *llm.APIError with its status and RetryAfter.
func mapError(err error) error {
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
	if ae, ok := errors.AsType[*llmprovider.APIError](err); ok {
		return &llm.APIError{Status: ae.Status, RetryAfter: ae.RetryAfter, Err: wrapped}
	}
	return wrapped
}
