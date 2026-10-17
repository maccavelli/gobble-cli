package provider

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"

	"github.com/maccavelli/gobble-cli/llm"
)

// direct is a client that takes no proxy from the environment.
func direct() *http.Client { return &http.Client{Transport: &http.Transport{}} }

// A request the network cannot carry, to a refused port or a host whose
// name does not resolve, is llm.ErrOffline, and Probe dials that request's
// host and port, and no other (0005-PLAN F5-1 decision 3, deviation 1).
func TestWireOffline(t *testing.T) {
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	refused := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ name, base, host string }{
		{"refused", "http://" + refused + "/v1", refused},
		{"unknown host", "http://gobble-offline.invalid/v1", "gobble-offline.invalid:80"},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, err := New("openai", Options{APIKey: "k", Model: "gpt-6-luna", BaseURL: c.base, HTTPClient: direct(), Retry: &llm.Retry{Off: true}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := drain(t, p, wireRequest()); !errors.Is(err, llm.ErrOffline) {
				t.Fatalf("err %v, want llm.ErrOffline", err)
			}
			var mu sync.Mutex
			var dialed []string
			old := dialProbe
			dialProbe = func(_ context.Context, network, address string) (net.Conn, error) {
				mu.Lock()
				defer mu.Unlock()
				dialed = append(dialed, network+" "+address)
				return nil, errors.New("still down")
			}
			t.Cleanup(func() { dialProbe = old })
			if err := p.(llm.Prober).Probe(t.Context()); err == nil {
				t.Fatal("Probe succeeded with the host down")
			}
			if len(dialed) != 1 || dialed[0] != "tcp "+c.host {
				t.Fatalf("Probe dialed %q, want tcp %s alone", dialed, c.host)
			}
		})
	}
	// A provider whose requests have all been carried has nothing to probe,
	// and dials nothing.
	p, err := New("openai", Options{APIKey: "k", HTTPClient: &http.Client{Transport: &cannedWire{}}})
	if err != nil {
		t.Fatal(err)
	}
	old := dialProbe
	dialProbe = func(context.Context, string, string) (net.Conn, error) {
		t.Error("Probe dialed with no failed request")
		return nil, errors.New("no")
	}
	t.Cleanup(func() { dialProbe = old })
	if err := p.(llm.Prober).Probe(t.Context()); err == nil {
		t.Fatal("Probe succeeded with nothing to probe")
	}
	// A service that answers, even with an error, is not offline.
	refusing := &cannedWire{status: http.StatusBadRequest, body: `{"error":{"message":"bad request","type":"invalid_request_error"}}`}
	if o := wireStream(t, "openai", "gpt-6-luna", refusing, wireRequest()); o.err == nil || errors.Is(o.err, llm.ErrOffline) {
		t.Fatalf("err %v, want an error that is not llm.ErrOffline", o.err)
	}
}

// Probe dials the proxy the environment names for the failed request's
// URL, or else its host, with the scheme's port when the URL gives none.
func TestProbePort(t *testing.T) {
	for _, c := range []struct{ url, want string }{
		{"https://api.example.test/v1/responses", "api.example.test:443"},
		{"http://api.example.test/v1/responses", "api.example.test:80"},
		{"http://localhost:8080/v1/chat/completions", "localhost:8080"},
	} {
		u, err := url.Parse(c.url)
		if err != nil {
			t.Fatal(err)
		}
		if proxy, err := http.ProxyFromEnvironment(&http.Request{URL: u}); err != nil || proxy != nil {
			t.Skipf("the environment names a proxy for %s (%v, %v), which Probe dials", c.url, proxy, err)
		}
		var dialed []string
		old := dialProbe
		dialProbe = func(_ context.Context, _, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			return nil, errors.New("down")
		}
		a := &adapter{id: "openai", offlineURL: c.url}
		err = a.Probe(t.Context())
		dialProbe = old
		if err == nil {
			t.Fatal("Probe succeeded with the host down")
		}
		if len(dialed) != 1 || dialed[0] != c.want {
			t.Errorf("%s: dialed %q, want %q alone", c.url, dialed, c.want)
		}
	}
}

// Each of the four wires' answer with nothing in it is llm.ErrEmptyReply,
// still llm.ErrIncomplete, after one request; a refusal with nothing in it
// is not llm.ErrEmptyReply. These pin the SDK's message (sdkEmptyAnswer)
// (0005-PLAN F5-1 decision 4).
func TestWireEmptyAnswer(t *testing.T) {
	for _, c := range []struct{ wire, id, model, body string }{
		{"responses", "openai", "gpt-6-luna", `{"id":"r","status":"completed","output":[],"usage":{"input_tokens":12,"output_tokens":0,"total_tokens":12}}`},
		{"messages", "claude", "claude-haiku-5-5", `{"id":"m","type":"message","role":"assistant","stop_reason":"end_turn","content":[],"usage":{"input_tokens":12,"output_tokens":0}}`},
		{"interactions", "gemini", "gemini-3.5-flash-lite", `{"id":"i","status":"completed","steps":[],"usage":{"total_input_tokens":12,"total_output_tokens":0,"total_tokens":12}}`},
		{"chat completions", "together", "zai-org/GLM-5.3-Flash", `{"id":"c","choices":[],"usage":{"prompt_tokens":12,"completion_tokens":0,"total_tokens":12}}`},
	} {
		t.Run(c.wire, func(t *testing.T) {
			w := &cannedWire{status: http.StatusOK, body: c.body}
			o := wireStream(t, c.id, c.model, w, wireRequest())
			if !errors.Is(o.err, llm.ErrEmptyReply) || !errors.Is(o.err, llm.ErrIncomplete) {
				t.Fatalf("err %v, want llm.ErrEmptyReply and llm.ErrIncomplete", o.err)
			}
			oneRequest(t, w)
		})
	}
	refusal := &cannedWire{status: http.StatusOK,
		body: `{"id":"m","type":"message","role":"assistant","stop_reason":"refusal","content":[],"usage":{"input_tokens":12,"output_tokens":0}}`}
	o := wireStream(t, "claude", "claude-haiku-5-5", refusal, wireRequest())
	if !errors.Is(o.err, llm.ErrIncomplete) || errors.Is(o.err, llm.ErrEmptyReply) {
		t.Fatalf("refusal: err %v, want llm.ErrIncomplete that is not llm.ErrEmptyReply", o.err)
	}
}

// The retry settings reach the SDK's policy: maxRetries more tries, none
// when off, a server's ask above maxDelay returned at once, and a first
// wait of at least half of baseDelay (0005-PLAN F5-1 decision 2).
func TestRetrySettings(t *testing.T) {
	if got, want := policyOf(llm.Retry{MaxRetries: 5, BaseDelay: 3 * time.Millisecond, MaxDelay: 9 * time.Millisecond}),
		(llmprovider.RetryPolicy{MaxAttempts: 6, BaseDelay: 3 * time.Millisecond, MaxDelay: 9 * time.Millisecond}); got != want {
		t.Fatalf("policyOf = %+v, want %+v", got, want)
	}
	if want := (llmprovider.RetryPolicy{MaxAttempts: 4, BaseDelay: 2 * time.Second, MaxDelay: 60 * time.Second}); retryPolicy != want {
		t.Fatalf("the default policy %+v, want Pi's %+v", retryPolicy, want)
	}
	fast := llm.Retry{MaxRetries: 2, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}
	off := fast
	off.Off = true
	slow := llm.Retry{MaxRetries: 1, BaseDelay: 400 * time.Millisecond, MaxDelay: time.Second}
	for _, c := range []struct {
		name     string
		retry    llm.Retry
		header   http.Header
		requests int
		atLeast  time.Duration
	}{
		{"maxRetries", fast, nil, 3, 0},
		{"off", off, nil, 1, 0},
		{"maxDelay", fast, http.Header{"Retry-After": {"1"}}, 1, 0},
		{"baseDelay", slow, nil, 2, 200 * time.Millisecond},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := &cannedWire{status: http.StatusTooManyRequests, header: c.header, body: wireRateLimited}
			p, err := New("openai", Options{APIKey: "k", Model: "gpt-6-luna", HTTPClient: &http.Client{Transport: w}, Retry: &c.retry})
			if err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			if _, err := drain(t, p, wireRequest()); !errors.Is(err, llm.ErrRateLimited) {
				t.Fatalf("err %v, want llm.ErrRateLimited", err)
			}
			if paths, _ := w.requests(); len(paths) != c.requests {
				t.Fatalf("%d requests, want %d", len(paths), c.requests)
			}
			if took := time.Since(start); took < c.atLeast {
				t.Fatalf("took %s, want at least %s", took, c.atLeast)
			}
		})
	}
}
