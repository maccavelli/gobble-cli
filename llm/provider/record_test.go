package provider

import (
	"context"
	"encoding/json/jsontext"
	"io"
	"iter"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	sdktest "github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"

	"github.com/maccavelli/gobble-cli/llm"
)

// captured is an slog.Handler that keeps each record it takes, at level and
// above.
type captured struct {
	level slog.Level
	mu    sync.Mutex
	recs  []slog.Record
}

func (c *captured) Enabled(_ context.Context, l slog.Level) bool { return l >= c.level }
func (c *captured) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recs = append(c.recs, r.Clone())
	return nil
}
func (c *captured) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *captured) WithGroup(string) slog.Handler      { return c }

// callRecord is one "model call" record: its level and its fields.
type callRecord struct {
	level  slog.Level
	fields map[string]any
}

// calls are the "model call" records c took, in order.
func (c *captured) calls() []callRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []callRecord
	for _, r := range c.recs {
		if r.Message != "model call" {
			continue
		}
		f := map[string]any{}
		r.Attrs(func(a slog.Attr) bool {
			f[a.Key] = a.Value.Resolve().Any()
			return true
		})
		out = append(out, callRecord{r.Level, f})
	}
	return out
}

// reply is one scripted HTTP answer: a status, headers and a body, or, with
// body nil, the stream writes.
type reply struct {
	status int
	header http.Header
	body   string
	pipe   *io.PipeReader
}

// seqWire answers each request with the next reply, the last one again
// once they run out.
type seqWire struct {
	mu      sync.Mutex
	replies []reply
	n       int
}

func (s *seqWire) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, r.Body) //nolint:errcheck // the body is not read here
	}
	s.mu.Lock()
	rp := s.replies[min(s.n, len(s.replies)-1)]
	s.n++
	s.mu.Unlock()
	h := http.Header{"Content-Type": {"text/event-stream"}}
	if rp.status != http.StatusOK {
		h = http.Header{"Content-Type": {"application/json"}}
	}
	maps.Copy(h, rp.header)
	var body io.Reader = strings.NewReader(rp.body)
	if rp.pipe != nil {
		// As a transport does, a cancelled request ends its body's reads.
		body = rp.pipe
		ctx := r.Context()
		go func() {
			<-ctx.Done()
			rp.pipe.CloseWithError(ctx.Err())
		}()
	}
	return &http.Response{StatusCode: rp.status, Header: h, Body: io.NopCloser(body), Request: r}, nil
}

// streamed is the Responses wire's canned "hello" stream.
var streamed = reply{status: http.StatusOK, body: wireStreams["/responses"][0]}

// recordProvider is provider openai over w, logging to c, for session s1.
func recordProvider(t *testing.T, w http.RoundTripper, c *captured, retry *llm.Retry) *adapter {
	t.Helper()
	if retry == nil {
		retry = &llm.Retry{Off: true}
	}
	p, err := New("openai", Options{APIKey: "k", Model: "gpt-6-luna", HTTPClient: &http.Client{Transport: w}, Logger: slog.New(c),
		SessionID: "s1", Retry: retry})
	if err != nil {
		t.Fatal(err)
	}
	return p.(*adapter)
}

// callReq is a request of model gpt-6-luna with output limit 4096.
func callReq() *llm.Request {
	r := wireRequest()
	r.Model, r.MaxTokens = "gpt-6-luna", 4096
	return r
}

// consume streams req through p and returns its error.
func consume(ctx context.Context, p llm.Provider, req *llm.Request) error {
	for _, err := range p.Stream(ctx, req) {
		if err != nil {
			return err
		}
	}
	return nil
}

// oneCall fails unless c took exactly one record, and returns it.
func oneCall(t *testing.T, c *captured) callRecord {
	t.Helper()
	calls := c.calls()
	if len(calls) != 1 {
		t.Fatalf("%d model call records, want 1: %+v", len(calls), calls)
	}
	return calls[0]
}

// Each model call writes one "model call" record: the call's session,
// provider, model and limit, how it went, and its usage; a background call
// says so; one the request refused before the SDK has its error, no
// delivery and its duration only; a call sent again after a cut carries its
// place (0070-MADR D15, D17, D21; T1, T3, T13, T20, T22, C23).
func TestModelCallRecord(t *testing.T) {
	c := &captured{level: slog.LevelDebug}
	if err := consume(t.Context(), recordProvider(t, &seqWire{replies: []reply{streamed}}, c, nil), callReq()); err != nil {
		t.Fatal(err)
	}
	f := oneCall(t, c).fields
	want := map[string]any{"session": "s1", "provider": "openai", "model": "gpt-6-luna", "background": false, "buffered": false,
		"limit": int64(4096), "cache": "", "outcome": "done", "stop": "end_turn", "delivery": "shown", "input": int64(12), "output": int64(3),
		"retries": int64(0)}
	for k, v := range want {
		if f[k] != v {
			t.Errorf("%s = %v (%T), want %v", k, f[k], f[k], v)
		}
	}
	t.Run("background", func(t *testing.T) {
		c := &captured{level: slog.LevelDebug}
		if err := consume(llm.WithBackground(t.Context()), recordProvider(t, &seqWire{replies: []reply{streamed}}, c, nil), callReq()); err != nil {
			t.Fatal(err)
		}
		if f := oneCall(t, c).fields; f["background"] != true {
			t.Fatalf("background %v, want true", f["background"])
		}
	})
	t.Run("warnings", func(t *testing.T) {
		c := &captured{level: slog.LevelDebug}
		w := llmprovider.Warning{Kind: llmprovider.WarningDeferralEmulated, Model: "m"}
		f := &streamer{caps: llmprovider.Capabilities{NativeStreaming: llmprovider.Supported}, events: []llmprovider.Event{
			{Type: llmprovider.EventTextDelta, Text: "hi"},
			{Type: llmprovider.EventDone, Response: &llmprovider.Response{Warnings: []llmprovider.Warning{w}}}}}
		a := &adapter{p: f, id: "openai", logger: slog.New(c), session: "s1"}
		if err := consume(t.Context(), a, callReq()); err != nil {
			t.Fatal(err)
		}
		if got, _ := oneCall(t, c).fields["warnings"].([]string); !slices.Equal(got, []string{string(w.Kind)}) {
			t.Fatalf("warnings %v, want [%s]", oneCall(t, c).fields["warnings"], w.Kind)
		}
	})
	t.Run("request_refused", func(t *testing.T) {
		c := &captured{level: slog.LevelDebug}
		req := callReq()
		req.Tools = jsontext.Value(`not json`)
		if err := consume(t.Context(), recordProvider(t, &seqWire{replies: []reply{streamed}}, c, nil), req); err == nil {
			t.Fatal("the request was not refused")
		}
		f := oneCall(t, c).fields
		_, timed := f["duration_ms"]
		if f["outcome"] != "error" || f["delivery"] != "none" || !timed {
			t.Fatalf("fields %v, want outcome error, delivery none, a duration", f)
		}
		for _, k := range []string{"ttft_ms", "events", "gap_max_ms", "retries"} {
			if _, ok := f[k]; ok {
				t.Errorf("%s present on a request refused before the SDK", k)
			}
		}
	})
	t.Run("cut_retry", func(t *testing.T) {
		c := &captured{level: slog.LevelDebug}
		if err := consume(llm.WithCutRetry(t.Context(), 2), recordProvider(t, &seqWire{replies: []reply{streamed}}, c, nil), callReq()); err != nil {
			t.Fatal(err)
		}
		if r := oneCall(t, c); r.fields["cut_retry"] != int64(2) || r.level != slog.LevelInfo {
			t.Fatalf("cut_retry %v at %v, want 2 at Info", r.fields["cut_retry"], r.level)
		}
	})
}

// streamer is an SDK provider that streams its events, or, with gen set,
// only generates (not a Streamer).
type streamer struct {
	caps   llmprovider.Capabilities
	events []llmprovider.Event
}

func (s *streamer) ID() llmprovider.ProviderID             { return "openai" }
func (s *streamer) Capabilities() llmprovider.Capabilities { return s.caps }
func (s *streamer) Generate(context.Context, *llmprovider.Request) (*llmprovider.Response, error) {
	return &llmprovider.Response{}, nil
}
func (s *streamer) Stream(_ context.Context, _ *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		for _, ev := range s.events {
			if !yield(ev, nil) {
				return
			}
		}
	}
}

// Above Info the adapter reads no clock and writes no record (0070-MADR
// D16; T2).
func TestModelCallRecordOffAboveInfo(t *testing.T) {
	c := &captured{level: slog.LevelWarn}
	a := recordProvider(t, &seqWire{replies: []reply{streamed}}, c, nil)
	reads := 0
	a.now = func() time.Time { reads++; return time.Now() }
	if err := consume(t.Context(), a, callReq()); err != nil {
		t.Fatal(err)
	}
	if reads != 0 || len(c.calls()) != 0 {
		t.Fatalf("%d clock reads, %d records at warn; want none", reads, len(c.calls()))
	}
}

// manyDeltas is a streaming SDK provider of 100 text deltas and a done.
func manyDeltas() *streamer {
	evs := make([]llmprovider.Event, 0, 101)
	for range 100 {
		evs = append(evs, llmprovider.Event{Type: llmprovider.EventTextDelta, Text: "x"})
	}
	evs = append(evs, llmprovider.Event{Type: llmprovider.EventDone, Response: &llmprovider.Response{}})
	return &streamer{caps: llmprovider.Capabilities{NativeStreaming: llmprovider.Supported}, events: evs}
}

// Above Info, a call allocates no more than with no logger (0070-MADR D16;
// T19).
func TestModelCallRecordNoAllocsAboveInfo(t *testing.T) {
	run := func(logger *slog.Logger) float64 {
		a := &adapter{p: manyDeltas(), id: "openai", logger: logger}
		req := callReq()
		return testing.AllocsPerRun(50, func() {
			for range a.Stream(context.Background(), req) {
			}
		})
	}
	none, warn := run(nil), run(slog.New(&captured{level: slog.LevelWarn}))
	if warn != none {
		t.Fatalf("%v allocations at warn, %v with no logger; want them equal", warn, none)
	}
}

// BenchmarkAdapterStream is a call of 100 deltas at info, at warn and with
// no logger (0070-MADR D16; T19).
func BenchmarkAdapterStream(b *testing.B) {
	for _, c := range []struct {
		name   string
		logger *slog.Logger
	}{
		{"info", slog.New(&captured{level: slog.LevelInfo})},
		{"warn", slog.New(&captured{level: slog.LevelWarn})},
		{"none", nil},
	} {
		b.Run(c.name, func(b *testing.B) {
			a := &adapter{p: manyDeltas(), id: "openai", logger: c.logger}
			req := callReq()
			b.ReportAllocs()
			for b.Loop() {
				for range a.Stream(context.Background(), req) {
				}
			}
		})
	}
}

// A call's timing: the first event's wait, the longest wait for an event
// once the consumer returned, the longest the consumer held one, all on the
// stream as it arrives (0070-MADR D17; T4). The first delta comes after
// 1 s; the second 3 s after the consumer returned; the consumer holds the
// second 5 s; the rest is written only once the hold ends.
func TestModelCallTiming(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pr, pw := io.Pipe()
		held := make(chan struct{})
		go func() {
			time.Sleep(time.Second)
			_, _ = io.WriteString(pw, sse("response.output_text.delta", `{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"hel"}`))
			time.Sleep(3 * time.Second)
			_, _ = io.WriteString(pw, sse("response.output_text.delta", `{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"lo"}`))
			<-held
			_, _ = io.WriteString(pw, sse("response.output_item.done", `{"type":"response.output_item.done","output_index":0,"item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}}`))
			_, _ = io.WriteString(pw, sse("response.completed", `{"type":"response.completed","response":`+wireResponsesText+`}`))
			_ = pw.Close()
		}()
		c := &captured{level: slog.LevelDebug}
		a := recordProvider(t, &seqWire{replies: []reply{{status: http.StatusOK, pipe: pr}}}, c, nil)
		n := 0
		for ev, err := range a.Stream(t.Context(), callReq()) {
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := ev.(llm.TextDelta); ok {
				if n++; n == 2 {
					time.Sleep(5 * time.Second)
					close(held)
				}
			}
		}
		f := oneCall(t, c).fields
		if f["ttft_ms"] != int64(1000) || f["gap_max_ms"] != int64(3000) || f["handling_max_ms"] != int64(5000) {
			t.Fatalf("ttft %v, gap %v, handling %v; want 1000, 3000, 5000", f["ttft_ms"], f["gap_max_ms"], f["handling_max_ms"])
		}
	})
}

// The SDK's retries before the first event: how many, their kinds in
// order, their waits and how many the service directed (0070-MADR D17;
// T5).
func TestModelCallRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		down := reply{status: http.StatusServiceUnavailable, body: `{"error":{"message":"overloaded","type":"server_error"}}`}
		slow := reply{status: http.StatusTooManyRequests, header: http.Header{"Retry-After": {"1"}}, body: wireRateLimited}
		c := &captured{level: slog.LevelDebug}
		a := recordProvider(t, &seqWire{replies: []reply{down, slow, streamed}}, c,
			&llm.Retry{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Second})
		if err := consume(t.Context(), a, callReq()); err != nil {
			t.Fatal(err)
		}
		r := oneCall(t, c)
		kinds, _ := r.fields["retry_kinds"].([]string)
		wait, _ := r.fields["retry_wait_ms"].(int64)
		if r.fields["retries"] != int64(2) || !slices.Equal(kinds, []string{"unavailable", "rate_limited"}) || r.fields["server_directed"] != int64(1) ||
			wait < 1000 || r.level != slog.LevelInfo {
			t.Fatalf("fields %v at %v; want 2 retries, unavailable then rate_limited, one directed, at least 1 s waited, at Info", r.fields, r.level)
		}
	})
}

// A call sent again with no thinking after a signature refusal is one
// record, marked (0070-MADR D17; T6).
func TestModelCallSignatureResend(t *testing.T) {
	rt := &refuseFirst{next: &cannedWire{}, body: `{"type":"error","error":{"type":"invalid_request_error","message":"messages.1.content.0: Invalid ` +
		"`signature`" + ` in ` + "`thinking`" + ` block"}}`}
	c := &captured{level: slog.LevelDebug}
	p, err := New("claude", Options{APIKey: "k", Model: "claude-haiku-5-5", HTTPClient: &http.Client{Transport: rt}, Logger: slog.New(c),
		Retry: &llm.Retry{Off: true}})
	if err != nil {
		t.Fatal(err)
	}
	req := signed("claude", "claude-haiku-5-5")
	req.Model = "claude-haiku-5-5"
	if err := consume(t.Context(), p, req); err != nil {
		t.Fatal(err)
	}
	if r := oneCall(t, c); r.fields["resent_without_thinking"] != true || r.fields["outcome"] != "done" {
		t.Fatalf("fields %v, want one done record, resent without thinking", r.fields)
	}
}

// toolStream is the Responses wire's canned get_weather stream: events, and
// nothing shown.
var toolStream = reply{status: http.StatusOK, body: wireStreams["/responses"][1]}

// refused400 is a 400 with a code.
var refused400 = reply{status: http.StatusBadRequest, body: `{"error":{"message":"bad","type":"invalid_request_error","code":"bad_thing"}}`}

// Delivery: none before any event, events when nothing was shown, shown
// after text (0070-MADR D1, D17; T7).
func TestModelCallDelivery(t *testing.T) {
	for _, c := range []struct {
		name  string
		reply reply
		want  string
	}{{"none", refused400, "none"}, {"events", toolStream, "events"}, {"shown", streamed, "shown"}} {
		t.Run(c.name, func(t *testing.T) {
			cp := &captured{level: slog.LevelDebug}
			_ = consume(t.Context(), recordProvider(t, &seqWire{replies: []reply{c.reply}}, cp, nil), callReq()) //nolint:errcheck // the record is the subject
			if got := oneCall(t, cp).fields["delivery"]; got != c.want {
				t.Fatalf("delivery %v, want %s", got, c.want)
			}
		})
	}
}

// Outcome: done with its stop; error with its kind, status and code, and no
// usage; cancelled; stopped by the consumer (0070-MADR D17; T8, T10).
func TestModelCallOutcome(t *testing.T) {
	t.Run("done", func(t *testing.T) {
		cp := &captured{level: slog.LevelDebug}
		_ = consume(t.Context(), recordProvider(t, &seqWire{replies: []reply{toolStream}}, cp, nil), callReq()) //nolint:errcheck // as above
		if f := oneCall(t, cp).fields; f["outcome"] != "done" || f["stop"] != "tool_use" {
			t.Fatalf("fields %v, want done, tool_use", f)
		}
	})
	t.Run("error", func(t *testing.T) {
		cp := &captured{level: slog.LevelDebug}
		_ = consume(t.Context(), recordProvider(t, &seqWire{replies: []reply{refused400}}, cp, nil), callReq()) //nolint:errcheck // as above
		f := oneCall(t, cp).fields
		if f["outcome"] != "error" || f["kind"] != "other" || f["status"] != int64(400) || f["code"] != "bad_thing" {
			t.Fatalf("fields %v, want error, other, 400, bad_thing", f)
		}
		for _, k := range []string{"input", "output", "reasoning", "cache_read", "cache_write", "estimated", "cache_miss", "stop"} {
			if _, ok := f[k]; ok {
				t.Errorf("%s present on a call with no usage", k)
			}
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		pr, pw := io.Pipe()
		defer pw.Close()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go func() {
			_, _ = io.WriteString(pw, sse("response.output_text.delta", `{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"hel"}`))
		}()
		cp := &captured{level: slog.LevelDebug}
		for _, err := range recordProvider(t, &seqWire{replies: []reply{{status: http.StatusOK, pipe: pr}}}, cp, nil).Stream(ctx, callReq()) {
			if err != nil {
				break
			}
			cancel()
		}
		if r := oneCall(t, cp); r.fields["outcome"] != "cancelled" || r.level != slog.LevelInfo {
			t.Fatalf("fields %v at %v, want cancelled at Info", r.fields, r.level)
		}
	})
	t.Run("stopped", func(t *testing.T) {
		cp := &captured{level: slog.LevelDebug}
		for range recordProvider(t, &seqWire{replies: []reply{streamed}}, cp, nil).Stream(t.Context(), callReq()) {
			break
		}
		if f := oneCall(t, cp).fields; f["outcome"] != "stopped" {
			t.Fatalf("fields %v, want stopped", f)
		}
	})
}

// Usage and the cache: cache_miss is the input less the cache reads, when
// caching was asked (0070-MADR D17; T9).
func TestModelCallCache(t *testing.T) {
	stream := sse("message_start", `{"type":"message_start","message":{"id":"m","type":"message","role":"assistant","content":[],"usage":{"input_tokens":10,"cache_read_input_tokens":4,"cache_creation_input_tokens":2,"output_tokens":0}}}`) +
		sse("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`) +
		sse("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`) +
		sse("content_block_stop", `{"type":"content_block_stop","index":0}`) +
		sse("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`) +
		sse("message_stop", `{"type":"message_stop"}`)
	cp := &captured{level: slog.LevelDebug}
	p, err := New("claude", Options{APIKey: "k", Model: "claude-haiku-5-5", Logger: slog.New(cp), Retry: &llm.Retry{Off: true},
		HTTPClient: &http.Client{Transport: &seqWire{replies: []reply{{status: http.StatusOK, body: stream}}}}})
	if err != nil {
		t.Fatal(err)
	}
	req := callReq()
	req.Model, req.Cache = "claude-haiku-5-5", llm.CacheShort
	if err := consume(t.Context(), p, req); err != nil {
		t.Fatal(err)
	}
	f := oneCall(t, cp).fields
	in, _ := f["input"].(int64)
	read, _ := f["cache_read"].(int64)
	write, _ := f["cache_write"].(int64)
	if read != 4 || write != 2 || f["cache"] != "short" || f["cache_miss"] != in-read {
		t.Fatalf("fields %v, want cache_read 4, cache_write 2, cache short, cache_miss input less cache_read", f)
	}
}

// d17 are the record's keys, all of them, less out_limit* until 0066 is
// built (0070-MADR D17, D18).
var d17 = []string{"session", "provider", "model", "background", "buffered", "cut_retry", "limit", "cache", "ttft_ms", "gap_max_ms",
	"handling_max_ms", "duration_ms", "events", "retries", "retry_kinds", "retry_wait_ms", "server_directed", "resent_without_thinking",
	"warnings", "delivery", "outcome", "kind", "status", "code", "stop", "input", "output", "reasoning", "cache_read", "cache_write",
	"estimated", "cache_miss"}

// The record holds D17's keys and nothing else: no text, no error text, no
// header; no out_limit field yet (0070-MADR D18; T11, T14).
func TestModelCallRecordHoldsNoContent(t *testing.T) {
	cp := &captured{level: slog.LevelDebug}
	_ = consume(t.Context(), recordProvider(t, &seqWire{replies: []reply{refused400}}, cp, nil), callReq())                    //nolint:errcheck // the record is the subject
	_ = consume(llm.WithCutRetry(t.Context(), 1), recordProvider(t, &seqWire{replies: []reply{streamed}}, cp, nil), callReq()) //nolint:errcheck // as above
	for _, r := range cp.calls() {
		for k, v := range r.fields {
			if !slices.Contains(d17, k) {
				t.Errorf("key %q = %v is not one of D17's", k, v)
			}
			if s, ok := v.(string); ok && (strings.Contains(s, "bad") || strings.Contains(s, "hello")) && k != "code" {
				t.Errorf("%s = %q carries content", k, s)
			}
		}
	}
	t.Run("no_out_limit", func(t *testing.T) {
		for _, r := range cp.calls() {
			for k := range r.fields {
				if strings.HasPrefix(k, "out_limit") {
					t.Errorf("%s present before 0066 is built", k)
				}
			}
		}
	})
}

// buffered says the SDK provider does not stream: a provider that only
// generates is buffered, a Streamer reporting BestEffort is not (0070-MADR
// D17; T21).
func TestModelCallRecordBuffered(t *testing.T) {
	gen := sdktest.NewFake("openai", llmprovider.Capabilities{}).ReplyText("ok")
	best := &streamer{caps: llmprovider.Capabilities{NativeStreaming: llmprovider.BestEffort}, events: []llmprovider.Event{
		{Type: llmprovider.EventTextDelta, Text: "ok"}, {Type: llmprovider.EventDone, Response: &llmprovider.Response{}}}}
	for _, c := range []struct {
		name string
		p    llmprovider.Provider
		want bool
	}{{"generates", gen, true}, {"best_effort_streamer", best, false}} {
		t.Run(c.name, func(t *testing.T) {
			cp := &captured{level: slog.LevelDebug}
			if err := consume(t.Context(), &adapter{p: c.p, id: "openai", logger: slog.New(cp)}, callReq()); err != nil {
				t.Fatal(err)
			}
			if got := oneCall(t, cp).fields["buffered"]; got != c.want {
				t.Fatalf("buffered %v, want %v", got, c.want)
			}
		})
	}
}

// The record's level: Info when the call failed, was cancelled or was
// retried, Debug when it went well (0070-MADR D15, Q5; T23).
func TestModelCallRecordLevel(t *testing.T) {
	down := reply{status: http.StatusServiceUnavailable, body: `{"error":{"message":"overloaded","type":"server_error"}}`}
	for _, c := range []struct {
		name  string
		ctx   func(context.Context) context.Context
		reply []reply
		retry *llm.Retry
		want  slog.Level
	}{
		{"failed", nil, []reply{refused400}, nil, slog.LevelInfo},
		{"retried", nil, []reply{down, streamed}, &llm.Retry{MaxRetries: 2, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}, slog.LevelInfo},
		{"cut_retry", func(ctx context.Context) context.Context { return llm.WithCutRetry(ctx, 1) }, []reply{streamed}, nil, slog.LevelInfo},
		{"healthy", nil, []reply{streamed}, nil, slog.LevelDebug},
	} {
		t.Run(c.name, func(t *testing.T) {
			cp := &captured{level: slog.LevelDebug}
			ctx := t.Context()
			if c.ctx != nil {
				ctx = c.ctx(ctx)
			}
			_ = consume(ctx, recordProvider(t, &seqWire{replies: c.reply}, cp, c.retry), callReq()) //nolint:errcheck // the record is the subject
			if got := oneCall(t, cp).level; got != c.want {
				t.Fatalf("level %v, want %v", got, c.want)
			}
		})
	}
	t.Run("healthy_at_info", func(t *testing.T) {
		cp := &captured{level: slog.LevelInfo}
		_ = consume(t.Context(), recordProvider(t, &seqWire{replies: []reply{streamed}}, cp, nil), callReq()) //nolint:errcheck // as above
		if n := len(cp.calls()); n != 0 {
			t.Fatalf("%d records at info for a healthy call, want none", n)
		}
	})
}
