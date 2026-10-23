package provider

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
)

// streamIDs are a provider on each of the four wires.
var streamIDs = []struct{ id, model string }{
	{"openai", "gpt-6-luna"}, {"claude", "claude-haiku-5-5"}, {"gemini", "gemini-3.5-flash-lite"},
	{"together", "zai-org/GLM-5.3-Flash"},
}

// fastRetry is a retry policy whose waits take milliseconds.
var fastRetry = llm.Retry{MaxRetries: 2, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}

// liveWire answers with the canned text stream: its events up to the first
// text delta's, then, once the test says that delta arrived or 2 s pass,
// the rest. It records whether the request asked to stream, and whether the
// delta arrived before the rest was sent.
type liveWire struct {
	first    chan struct{}
	early    atomic.Bool
	streamed atomic.Bool
}

func (w *liveWire) RoundTrip(r *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	w.streamed.Store(strings.Contains(string(body), `"stream":true`))
	s, _ := wireStreamFor(r.URL.Path, false)
	at := strings.Index(s, "hello")
	cut := at + strings.Index(s[at:], "\n\n") + 2
	pr, pw := io.Pipe()
	go func() {
		_, _ = io.WriteString(pw, s[:cut]) //nolint:errcheck // the reader's end reports a failed write
		select {
		case <-w.first:
			w.early.Store(true)
		case <-time.After(2 * time.Second):
		}
		_, _ = io.WriteString(pw, s[cut:]) //nolint:errcheck // as above
		_ = pw.Close()                     //nolint:errcheck // a pipe's writer closes cleanly
	}()
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: pr,
		Request: r}, nil
}

// New streams live: a reply's first text delta reaches the caller before
// the service has sent the rest of the reply (0031-MADR D2).
func TestStreamIsLive(t *testing.T) {
	for _, c := range streamIDs {
		t.Run(c.id, func(t *testing.T) {
			w := &liveWire{first: make(chan struct{})}
			p, err := New(c.id, Options{APIKey: "k", Model: c.model, HTTPClient: &http.Client{Transport: w}, Retry: &fastRetry})
			if err != nil {
				t.Fatal(err)
			}
			req := wireRequest()
			req.Model = c.model
			var once sync.Once
			var text strings.Builder
			for ev, err := range p.Stream(t.Context(), req) {
				if err != nil {
					t.Fatalf("stream: %v (streamed %v)", err, w.streamed.Load())
				}
				if d, ok := ev.(llm.TextDelta); ok {
					text.WriteString(d.Text)
					once.Do(func() { close(w.first) })
				}
			}
			if !w.streamed.Load() {
				t.Fatal("the request did not ask to stream")
			}
			if !w.early.Load() || text.String() != "hello" {
				t.Fatalf("text %q, its first delta before the rest was sent: %v; want hello, delivered live", text.String(), w.early.Load())
			}
		})
	}
}

// brokenWire answers its first request with the text stream cut after the
// first text delta, and counts the requests.
type brokenWire struct{ requests atomic.Int32 }

func (w *brokenWire) RoundTrip(r *http.Request) (*http.Response, error) {
	w.requests.Add(1)
	if r.Body != nil {
		_ = r.Body.Close() //nolint:errcheck // a canned reply reads nothing of the request
	}
	s, _ := wireStreamFor(r.URL.Path, false)
	at := strings.Index(s, "hello")
	cut := at + strings.Index(s[at:], "\n\n") + 2
	pr, pw := io.Pipe()
	go func() {
		_, _ = io.WriteString(pw, s[:cut])     //nolint:errcheck // the reader's end reports a failed write
		pw.CloseWithError(io.ErrUnexpectedEOF) //nolint:errcheck,gosec // the failure is the point
	}()
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: pr,
		Request: r}, nil
}

// A stream that fails after its first text delta is not sent again: the
// delta arrives once, then the failure, after one request (0031-MADR D3).
func TestFailureAfterFirstDeltaNotRetried(t *testing.T) {
	for _, c := range streamIDs {
		t.Run(c.id, func(t *testing.T) {
			w := &brokenWire{}
			p, err := New(c.id, Options{APIKey: "k", Model: c.model, HTTPClient: &http.Client{Transport: w}, Retry: &fastRetry})
			if err != nil {
				t.Fatal(err)
			}
			req := wireRequest()
			req.Model = c.model
			var text strings.Builder
			var failure error
			for ev, err := range p.Stream(t.Context(), req) {
				if err != nil {
					failure = err
					break
				}
				if d, ok := ev.(llm.TextDelta); ok {
					text.WriteString(d.Text)
				}
			}
			if failure == nil || text.String() != "hello" || w.requests.Load() != 1 {
				t.Fatalf("text %q, err %v, after %d requests; want hello once, then the failure, after 1", text.String(), failure,
					w.requests.Load())
			}
		})
	}
}

// flakyWire answers its first request with a 503 and the rest with the
// canned text stream, counting the requests.
type flakyWire struct{ requests atomic.Int32 }

func (w *flakyWire) RoundTrip(r *http.Request) (*http.Response, error) {
	n := w.requests.Add(1)
	if r.Body != nil {
		_ = r.Body.Close() //nolint:errcheck // a canned reply reads nothing of the request
	}
	if n == 1 {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"error":{"message":"overloaded","type":"server_error"}}`)), Request: r}, nil
	}
	s, _ := wireStreamFor(r.URL.Path, false)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(s)), Request: r}, nil
}

// A failure before the first event is retried, and the retry is reported
// before the reply, which arrives once: the try that failed, of how many,
// and the failure's class (0031-MADR D4).
func TestProviderRetryReported(t *testing.T) {
	for _, c := range streamIDs {
		t.Run(c.id, func(t *testing.T) {
			w := &flakyWire{}
			p, err := New(c.id, Options{APIKey: "k", Model: c.model, HTTPClient: &http.Client{Transport: w}, Retry: &fastRetry})
			if err != nil {
				t.Fatal(err)
			}
			req := wireRequest()
			req.Model = c.model
			var text strings.Builder
			var retries []llm.ProviderRetry
			for ev, err := range p.Stream(t.Context(), req) {
				if err != nil {
					t.Fatal(err)
				}
				switch ev := ev.(type) {
				case llm.ProviderRetry:
					if text.Len() > 0 {
						t.Fatal("a retry reported after the reply began")
					}
					retries = append(retries, ev)
				case llm.TextDelta:
					text.WriteString(ev.Text)
				}
			}
			if text.String() != "hello" || w.requests.Load() != 2 {
				t.Fatalf("text %q after %d requests, want hello once after 2", text.String(), w.requests.Load())
			}
			if len(retries) != 1 || retries[0].Attempt != 1 || retries[0].MaxAttempts != 3 || retries[0].Kind != "unavailable" ||
				retries[0].Wait <= 0 {
				t.Fatalf("retries %+v, want one: attempt 1 of 3, unavailable, with its wait", retries)
			}
		})
	}
}
