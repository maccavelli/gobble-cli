package provider

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	gcatalog "github.com/maccavelli/gobble-cli/llm/catalog"
)

// modelDoc is a models.dev document with each section's models, each model
// named after its id and taking tools.
func modelDoc(sections map[string][]string) []byte {
	var b strings.Builder
	b.WriteString("{")
	for i, key := range slices.Sorted(maps.Keys(sections)) {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "%q:{\"models\":{", key)
		for j, id := range sections[key] {
			if j > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, "%q:{\"name\":%q,\"tool_call\":true,\"limit\":{\"context\":1000,\"output\":100}}", id, "Name of "+id)
		}
		b.WriteString("}}")
	}
	b.WriteString("}\n")
	return []byte(b.String())
}

// served is one canned reply.
type served struct {
	status int
	header http.Header
	body   io.Reader
	err    error
}

// dataWire serves model data documents in memory, by URL, and records every
// request; a URL it does not serve is an error. It never dials.
type dataWire struct {
	mu      sync.Mutex
	replies map[string]served
	got     []*http.Request
}

func (w *dataWire) RoundTrip(r *http.Request) (*http.Response, error) {
	w.mu.Lock()
	w.got = append(w.got, r)
	s, ok := w.replies[r.URL.String()]
	w.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no reply for %s", r.URL)
	}
	if s.err != nil {
		return nil, s.err
	}
	body := s.body
	if body == nil {
		body = http.NoBody
	}
	return &http.Response{StatusCode: s.status, Header: cmpHeader(s.header), Body: io.NopCloser(body), Request: r}, nil
}

func (w *dataWire) requests() []*http.Request {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.got)
}

func cmpHeader(h http.Header) http.Header {
	if h == nil {
		return http.Header{}
	}
	return h
}

const testDataURL = "https://models-dev.invalid/api.json"

func parsed(t *testing.T, doc []byte) *ModelData {
	t.Helper()
	d, err := ParseModelData(bytes.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// M77: the stamp, a document stamped anew, the copy's bytes and the URL
// (0072-MADR M77, "Add the four names (Recommended)").
func TestModelDataStamp_0072(t *testing.T) {
	if ModelDataURL != "https://models.dev/api.json" {
		t.Errorf("ModelDataURL %q", ModelDataURL)
	}
	pretty := []byte("{\n  \"openai\": {\"models\": {\"m1\": {\"name\": \"One\"}}},\n  \"xai\": {\"models\": {\"g1\": {}, \"g2\": {}}}\n}\n")
	d := parsed(t, pretty)
	if d.Stamp() != (ModelDataStamp{}) {
		t.Errorf("a parsed document's stamp %+v, want the zero stamp", d.Stamp())
	}
	s := ModelDataStamp{Source: testDataURL, Fetched: time.Date(2026, 11, 25, 1, 2, 3, 0, time.UTC), ETag: `"e1"`, LastModified: "Wed, 25 Nov 2026 01:00:00 GMT"}
	st := d.Stamped(s)
	if st.Stamp() != s || d.Stamp() != (ModelDataStamp{}) {
		t.Errorf("stamped %+v, original %+v; want the new stamp on the copy alone", st.Stamp(), d.Stamp())
	}
	if !bytes.Equal(st.Document(), d.Document()) || st.doc != d.doc {
		t.Error("a stamped document does not share its parse")
	}
	doc := d.Document()
	if bytes.ContainsAny(doc, "\n ") {
		t.Errorf("Document %q is not compact", doc)
	}
	back, err := jsonx.MarshalStrict(struct{ D jsontext.Value }{D: doc})
	if err != nil || !bytes.Contains(back, doc) {
		t.Errorf("the strict form of Document differs: %s (%v)", back, err)
	}
	doc[1] = 'X'
	if bytes.Equal(d.Document(), doc) {
		t.Error("Document is not the caller's own copy")
	}
	if got, want := d.Sections(), map[string]int{"openai": 1, "xai": 2}; !maps.Equal(got, want) {
		t.Errorf("Sections %v, want %v", got, want)
	}
	var nilData *ModelData
	if nilData.Stamp() != (ModelDataStamp{}) {
		t.Error("nil's stamp is not zero")
	}
}

// M8, M23: gobble's own GET of the document, no cache (0072-MADR D4).
func TestFetchModelData_0072(t *testing.T) {
	if ModelDataURL != "https://models.dev/api.json" {
		t.Fatalf("ModelDataURL %q, want models.dev's", ModelDataURL)
	}
	doc := modelDoc(map[string][]string{"openai": {"gpt-a", "gpt-b"}, "anthropic": {"claude-a"}})
	w := &dataWire{replies: map[string]served{}}
	o := Options{HTTPClient: &http.Client{Transport: w}, ClientName: "gobble", ClientVersion: "1.2.3"}
	for i := range 2 {
		w.mu.Lock()
		w.replies[testDataURL] = served{status: http.StatusOK, header: http.Header{"Etag": {`"v1"`}}, body: bytes.NewReader(doc)}
		w.mu.Unlock()
		d, nm, err := FetchModelData(t.Context(), testDataURL, o)
		if err != nil || nm {
			t.Fatalf("fetch %d: %v, not modified %v", i, err, nm)
		}
		if got := d.Document(); !bytes.Equal(got, parsed(t, doc).Document()) {
			t.Errorf("fetch %d: document %s", i, got)
		}
		if s := d.Stamp(); s.Source != testDataURL || s.ETag != `"v1"` || s.Fetched.IsZero() || s.Fetched.Location() != time.UTC {
			t.Errorf("fetch %d: stamp %+v", i, s)
		}
	}
	reqs := w.requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests, want two: one per call", len(reqs))
	}
	for _, r := range reqs {
		if r.Method != http.MethodGet || r.URL.String() != testDataURL || r.Header.Get("User-Agent") != "gobble/1.2.3" ||
			r.Header.Get("If-None-Match") != "" {
			t.Errorf("request %s %s, user agent %q, If-None-Match %q", r.Method, r.URL, r.Header.Get("User-Agent"), r.Header.Get("If-None-Match"))
		}
	}
}

// M9, M77: validators from a copy of the same source; a 304 renews the
// stamp alone (0072-MADR D4).
func TestFetchModelDataConditional_0072(t *testing.T) {
	doc := modelDoc(map[string][]string{"openai": {"gpt-a"}})
	old := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	prevOf := func(etag, lm, source string) *ModelData {
		return parsed(t, doc).Stamped(ModelDataStamp{Source: source, Fetched: old, ETag: etag, LastModified: lm})
	}
	fetch := func(t *testing.T, prev *ModelData, reply served) (*ModelData, bool, *http.Request, error) {
		t.Helper()
		w := &dataWire{replies: map[string]served{testDataURL: reply}}
		d, nm, err := FetchModelData(t.Context(), testDataURL, Options{HTTPClient: &http.Client{Transport: w}, ModelData: prev})
		reqs := w.requests()
		if len(reqs) != 1 {
			t.Fatalf("%d requests", len(reqs))
		}
		return d, nm, reqs[0], err
	}
	t.Run("etag", func(t *testing.T) {
		_, _, r, _ := fetch(t, prevOf(`"e1"`, "", testDataURL), served{status: http.StatusNotModified})
		if r.Header.Get("If-None-Match") != `"e1"` || r.Header.Get("If-Modified-Since") != "" {
			t.Errorf("If-None-Match %q, If-Modified-Since %q", r.Header.Get("If-None-Match"), r.Header.Get("If-Modified-Since"))
		}
	})
	t.Run("last_modified", func(t *testing.T) {
		lm := "Sun, 01 Nov 2026 00:00:00 GMT"
		_, _, r, _ := fetch(t, prevOf("", lm, testDataURL), served{status: http.StatusNotModified})
		if r.Header.Get("If-Modified-Since") != lm || r.Header.Get("If-None-Match") != "" {
			t.Errorf("If-None-Match %q, If-Modified-Since %q", r.Header.Get("If-None-Match"), r.Header.Get("If-Modified-Since"))
		}
	})
	t.Run("other_source", func(t *testing.T) {
		_, _, r, _ := fetch(t, prevOf(`"e1"`, "x", "https://other.invalid/api.json"), served{status: http.StatusOK, body: bytes.NewReader(doc)})
		if r.Header.Get("If-None-Match") != "" || r.Header.Get("If-Modified-Since") != "" {
			t.Error("a copy of another source sent its validators")
		}
	})
	t.Run("304_renews", func(t *testing.T) {
		prev := prevOf(`"e1"`, "", testDataURL)
		d, nm, _, err := fetch(t, prev, served{status: http.StatusNotModified})
		if err != nil || !nm {
			t.Fatalf("err %v, not modified %v", err, nm)
		}
		s := d.Stamp()
		if !bytes.Equal(d.Document(), prev.Document()) || s.Source != testDataURL || s.ETag != `"e1"` || !s.Fetched.After(old) ||
			prev.Stamp().Fetched != old {
			t.Errorf("renewed %+v; want the copy's document, its ETag, a new time, the copy unchanged", s)
		}
	})
	t.Run("304_unasked", func(t *testing.T) {
		d, nm, _, err := fetch(t, nil, served{status: http.StatusNotModified})
		if _, ok := errors.AsType[*llm.APIError](err); !ok || d != nil || nm {
			t.Errorf("a 304 to a request with no validator: %v, %v, %v; want a status failure", d, nm, err)
		}
	})
}

// M10: the run's client carries the request.
func TestFetchModelDataUsesClient_0072(t *testing.T) {
	w := &dataWire{replies: map[string]served{testDataURL: {status: http.StatusOK, body: bytes.NewReader(modelDoc(map[string][]string{"xai": {"g"}}))}}}
	if _, _, err := FetchModelData(t.Context(), testDataURL, Options{HTTPClient: &http.Client{Transport: w}}); err != nil {
		t.Fatal(err)
	}
	if n := len(w.requests()); n != 1 {
		t.Fatalf("the run's client carried %d requests, want the one", n)
	}
}

// M58: a redirect to another host is followed; the stamp is the URL asked
// for.
//
// M82: a redirect from https to http fails the download before the http
// request (the owner's "Follow any, but refuse https→http (Recommended)").
func TestFetchModelDataRedirect_0072(t *testing.T) {
	t.Run("other_host", func(t *testing.T) {
		mirror := "https://mirror.invalid/doc.json"
		w := &dataWire{replies: map[string]served{
			testDataURL: {status: http.StatusFound, header: http.Header{"Location": {mirror}}},
			mirror:      {status: http.StatusOK, body: bytes.NewReader(modelDoc(map[string][]string{"xai": {"g"}}))},
		}}
		d, _, err := FetchModelData(t.Context(), testDataURL, Options{HTTPClient: &http.Client{Transport: w}})
		if err != nil {
			t.Fatal(err)
		}
		if reqs := w.requests(); len(reqs) != 2 || reqs[1].URL.String() != mirror {
			t.Fatalf("requests %d, want the asked URL then the mirror", len(reqs))
		}
		if s := d.Stamp(); s.Source != testDataURL {
			t.Errorf("stamped %q, want the URL asked for", s.Source)
		}
	})
	t.Run("https_to_http", func(t *testing.T) {
		plain := "http://mirror.invalid/doc.json"
		w := &dataWire{replies: map[string]served{
			testDataURL: {status: http.StatusFound, header: http.Header{"Location": {plain}}},
			plain:       {status: http.StatusOK, body: bytes.NewReader(modelDoc(map[string][]string{"xai": {"g"}}))},
		}}
		d, nm, err := FetchModelData(t.Context(), testDataURL, Options{HTTPClient: &http.Client{Transport: w}})
		if d != nil || nm || !errors.Is(err, llm.ErrNotPermitted) || errors.Is(err, llm.ErrOffline) {
			t.Errorf("%v, %v, %v; want the download refused, not permitted", d, nm, err)
		}
		if reqs := w.requests(); len(reqs) != 1 {
			t.Errorf("%d requests; want the https one alone, no http request sent", len(reqs))
		}
	})
}

// blockingWire answers nothing until its request's context ends.
type blockingWire struct{}

func (blockingWire) RoundTrip(r *http.Request) (*http.Response, error) {
	<-r.Context().Done()
	return nil, r.Context().Err()
}

// M59: the fetch ends at its caller's deadline, as a timeout.
func TestFetchModelDataDeadline_0072(t *testing.T) {
	for _, wait := range []time.Duration{30 * time.Second, 120 * time.Second} {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), wait)
			defer cancel()
			start := time.Now()
			_, _, err := FetchModelData(ctx, testDataURL, Options{HTTPClient: &http.Client{Transport: blockingWire{}}})
			if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, llm.ErrOffline) || time.Since(start) != wait {
				t.Errorf("after %v: %v; want a timeout at %v", time.Since(start), err, wait)
			}
		})
	}
}

// M11 to M13, M78, as llm/provider reports them: offline, a status, a body
// over the limit or not a document, a section the SDK cannot decode.
func TestFetchModelDataFailures_0072(t *testing.T) {
	// The padding is in a key the SDK does not keep, so only the size can
	// refuse it.
	big := io.MultiReader(strings.NewReader(`{"xai":{"models":{"g":{}}},"pad":"`), io.LimitReader(repeatReader('a'), 33<<20), strings.NewReader(`"}`))
	for name, c := range map[string]struct {
		reply   served
		offline bool
		status  bool
		says    string
	}{
		"offline":             {reply: served{err: errors.New("dial tcp: connection refused")}, offline: true},
		"status":              {reply: served{status: http.StatusBadGateway}, status: true},
		"oversize":            {reply: served{status: http.StatusOK, body: big}, says: "the reply is over 32 MiB"},
		"not_document":        {reply: served{status: http.StatusOK, body: strings.NewReader("<html>")}},
		"section_undecodable": {reply: served{status: http.StatusOK, body: strings.NewReader(`{"openai":{"models":5}}`)}},
	} {
		t.Run(name, func(t *testing.T) {
			w := &dataWire{replies: map[string]served{testDataURL: c.reply}}
			d, nm, err := FetchModelData(t.Context(), testDataURL, Options{HTTPClient: &http.Client{Transport: w}})
			_, isStatus := errors.AsType[*llm.APIError](err)
			if err == nil || d != nil || nm || errors.Is(err, llm.ErrOffline) != c.offline || isStatus != c.status {
				t.Errorf("%v, %v, %v; want a failure, offline %v, status %v", d, nm, err, c.offline, c.status)
			}
			if c.says != "" && (err == nil || !strings.Contains(err.Error(), c.says)) {
				t.Errorf("%v; want gobble's own limit, %q", err, c.says)
			}
		})
	}
}

type repeatReader byte

func (r repeatReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(r)
	}
	return len(p), nil
}

// M19, M74 (the lookups' half): the current document is the copy, else the
// built-in copy until Phase 3, never both.
func TestModelDataCurrent_0072(t *testing.T) {
	t.Cleanup(func() { SetCurrentModelData(nil) })
	const builtInOnly = "claude-opus-4-6"
	if slices.Contains(Models("claude"), builtInOnly) {
		t.Fatalf("%s is curated, so the test shows nothing", builtInOnly)
	}
	t.Run("built_in", func(t *testing.T) {
		SetCurrentModelData(nil)
		d := CurrentModelData()
		built, err := builtIn()
		if err != nil || d != built || d.Stamp() != (ModelDataStamp{}) {
			t.Fatalf("current %p, built-in %p (%v), stamp %+v", d, built, err, d.Stamp())
		}
		_, fetched := gcatalog.Source()
		if fetched.IsZero() || d.Sections()["anthropic"] != 17 {
			t.Errorf("sections %v; want the built-in copy's", d.Sections())
		}
		if m := ModelInfoContext(t.Context(), "claude", "", builtInOnly, nil); m.Name == "" {
			t.Errorf("%s unknown with the built-in copy current", builtInOnly)
		}
	})
	t.Run("copy", func(t *testing.T) {
		copyData := parsed(t, modelDoc(map[string][]string{"anthropic": {"claude-copy-only-1"}})).Stamped(ModelDataStamp{Source: testDataURL})
		SetCurrentModelData(copyData)
		if CurrentModelData() != copyData {
			t.Fatal("the set document is not current")
		}
		if !slices.Contains(KnownModels("claude"), "claude-copy-only-1") {
			t.Errorf("KnownModels %v lacks the copy's model", KnownModels("claude"))
		}
		if m := ModelInfoContext(t.Context(), "claude", "", "claude-copy-only-1", nil); m.Name != "Name of claude-copy-only-1" {
			t.Errorf("the copy's model reads %q", m.Name)
		}
	})
	t.Run("never_both", func(t *testing.T) {
		SetCurrentModelData(parsed(t, modelDoc(map[string][]string{"anthropic": {"claude-copy-only-1"}})))
		if slices.Contains(KnownModels("claude"), builtInOnly) {
			t.Errorf("KnownModels holds %s, the built-in copy's, beside the current document", builtInOnly)
		}
		if m := ModelInfoContext(t.Context(), "claude", "", builtInOnly, nil); m.Name != "" || m.ContextWindow != 0 {
			t.Errorf("%s reads %q, window %d, from the built-in copy beside the current document", builtInOnly, m.Name, m.ContextWindow)
		}
	})
}
