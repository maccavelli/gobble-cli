package provider

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	gcatalog "github.com/maccavelli/gobble-cli/llm/catalog"
)

// chatStream is a Chat Completions event stream whose text is hello.
const chatStream = "data: {\"id\":\"cmpl\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hello\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"cmpl\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
	"data: [DONE]\n\n"

// chatWire answers a Chat Completions server: a model list, and a streamed
// completion. It records each request as "METHOD path auth".
type chatWire struct {
	mu   sync.Mutex
	seen []string
	list string
}

func (c *chatWire) RoundTrip(r *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.seen = append(c.seen, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
	c.mu.Unlock()
	body := "{}"
	kind := "application/json"
	if strings.HasSuffix(r.URL.Path, "/chat/completions") {
		body, kind = chatStream, "text/event-stream"
	} else if strings.HasSuffix(r.URL.Path, "/models") {
		body = c.list
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {kind}},
		Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func (c *chatWire) posts() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []string
	for _, s := range c.seen {
		if strings.HasPrefix(s, "POST ") && strings.Contains(s, "chat/completions") {
			out = append(out, s)
		}
	}
	return out
}

// chatSet is a run's set holding the Chat Completions servers doc names.
func chatSet(t *testing.T, doc string) (*Set, *gcatalog.ModelsJSON) {
	t.Helper()
	mj, _, err := gcatalog.ParseModelsJSON([]byte(doc), "v1.31.0", gcatalog.ProviderIDs())
	if err != nil {
		t.Fatal(err)
	}
	set := NewSet()
	if err := set.AddChatServers(mj); err != nil {
		t.Fatal(err)
	}
	return set, mj
}

// A set holding local streams its text. With no key the completion has no
// Authorization header; with the key, it is the bearer token (0036-PLAN P4).
func TestChatServerStreams(t *testing.T) {
	off := &llm.Retry{Off: true}
	for _, key := range []string{"", "sk-test"} {
		wire := &chatWire{list: `{"data":[{"id":"m"}]}`}
		set, _ := chatSet(t, `{"providers":{"local":{"name":"Local","baseUrl":"http://127.0.0.1:1/v1"}}}`)
		p, err := New("local", Options{Set: set, APIKey: key, Model: "m", BaseURL: "http://127.0.0.1:1/v1",
			HTTPClient: &http.Client{Transport: wire}, Retry: off})
		if err != nil {
			t.Fatalf("key %q: %v", key, err)
		}
		var text strings.Builder
		req := wireRequest()
		req.Model = "m"
		for ev, err := range p.Stream(t.Context(), req) {
			if err != nil {
				t.Fatalf("key %q: stream: %v", key, err)
			}
			if d, ok := ev.(llm.TextDelta); ok {
				text.WriteString(d.Text)
			}
		}
		if text.String() != "hello" {
			t.Errorf("key %q: text %q, want hello", key, text.String())
		}
		posts := wire.posts()
		if len(posts) != 1 {
			t.Fatalf("key %q: completions %q, want one", key, posts)
		}
		switch key {
		case "":
			if strings.TrimPrefix(posts[0], "POST ") != "/v1/chat/completions " && !strings.HasSuffix(posts[0], " ") {
				t.Errorf("no key: %q, want no Authorization header", posts[0])
			}
			if strings.Contains(posts[0], "Bearer") {
				t.Errorf("no key: %q, want no Authorization header", posts[0])
			}
		default:
			if !strings.HasSuffix(posts[0], "Bearer "+key) {
				t.Errorf("key: %q, want Bearer %s", posts[0], key)
			}
		}
	}
}

// List returns the server's /v1/models ids, and models.json's models are
// among the known ones (0036-PLAN P4).
func TestChatServerLists(t *testing.T) {
	wire := &chatWire{list: `{"data":[{"id":"a"},{"id":"b"}]}`}
	set, mj := chatSet(t, `{"providers":{"local":{"baseUrl":"http://127.0.0.1:1/v1","models":[{"id":"mine"}]}}}`)
	l, err := List(t.Context(), "local", "", Options{Set: set, BaseURL: "http://127.0.0.1:1/v1",
		HTTPClient: &http.Client{Transport: wire}, ModelMetadataURL: "http://127.0.0.1:1/v1/meta.json"})
	if err != nil {
		t.Fatal(err)
	}
	if !l.Live || len(l.Models) != 2 || l.Models[0] != "a" || l.Models[1] != "b" {
		t.Fatalf("listing %+v, want live [a b]", l)
	}
	if len(mj.Models("local")) != 1 || mj.Models("local")[0] != "mine" {
		t.Fatalf("known %q, want models.json's mine among them", mj.Models("local"))
	}
}

// ChooseWith never picks a Chat Completions server on its own. Named as
// defaultProvider, it does, and --model local/m resolves (0036-PLAN P4).
func TestChatServerNotAmbient(t *testing.T) {
	set, _ := chatSet(t, `{"providers":{"local":{"name":"Local","baseUrl":"http://127.0.0.1:1/v1","models":[{"id":"m"}]}}}`)
	none := func(string) bool { return false }
	if sel, err := ChooseWith(set, none, ""); err == nil || sel.ID == "local" {
		t.Fatalf("no defaultProvider: %+v, %v; want it never to pick local", sel, err)
	}
	sel, err := ChooseWith(set, none, "local")
	if err != nil || sel.ID != "local" {
		t.Fatalf("defaultProvider local: %+v, %v", sel, err)
	}
	res, err := gcatalog.ResolveModel("", "local/m", "", []llm.Model{{Provider: "local", ID: "m"}}, none)
	if err != nil || res.Model.Provider != "local" || res.Model.ID != "m" {
		t.Fatalf("--model local/m: %+v, %v", res, err)
	}
}
