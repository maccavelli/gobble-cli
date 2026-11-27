package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// cliDataURL is the engine's source in these tests, under .invalid
// (0072-MADR D10).
const cliDataURL = "https://models-dev.invalid/api.json"

// cliModelDoc is a models.dev document whose sections hold n models each,
// named m0 to m<n-1> under the section's key.
func cliModelDoc(sections map[string]int) []byte {
	var b strings.Builder
	b.WriteString("{")
	for i, key := range slices.Sorted(maps.Keys(sections)) {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "%q:{\"models\":{", key)
		for j := range sections[key] {
			if j > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, "\"%s-m%d\":{\"name\":\"%s model %d\",\"tool_call\":true}", key, j, key, j)
		}
		b.WriteString("}}")
	}
	b.WriteString("}\n")
	return []byte(b.String())
}

func cliParsed(t *testing.T, doc []byte, s provider.ModelDataStamp) *provider.ModelData {
	t.Helper()
	d, err := provider.ParseModelData(bytes.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	return d.Stamped(s)
}

// dataReply is one canned reply of dataTransport.
type dataReply struct {
	status int
	header http.Header
	body   []byte
	err    error
}

// dataTransport serves the engine's fetch in memory: each request takes the
// next reply, the last repeating; it records each request and never dials.
type dataTransport struct {
	mu      sync.Mutex
	replies []dataReply
	got     []*http.Request
}

func (d *dataTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	d.mu.Lock()
	d.got = append(d.got, r)
	rep := d.replies[min(len(d.got), len(d.replies))-1]
	d.mu.Unlock()
	if rep.err != nil {
		return nil, rep.err
	}
	h := rep.header
	if h == nil {
		h = http.Header{}
	}
	return &http.Response{StatusCode: rep.status, Header: h, Body: io.NopCloser(bytes.NewReader(rep.body)), Request: r}, nil
}

func (d *dataTransport) requests() []*http.Request {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.got)
}

// dataRun is a run whose engine fetches through an in-memory transport from
// cliDataURL, and reads its copy from GOBBLE_HOME; the current document is
// the one t found again when t ends.
type dataRun struct {
	e             *runEnv
	warnings      *syncBuffer
	wire          *dataTransport
	copy, attempt string
}

func newDataRun(t *testing.T, replies ...dataReply) *dataRun {
	t.Helper()
	e, warnings := modelEnv(t)
	wire := &dataTransport{replies: replies}
	oldFetch, oldLoad, oldSource := fetchModelData, loadModelData, modelDataSource
	fetchModelData = func(ctx context.Context, url string, o provider.Options) (*provider.ModelData, bool, error) {
		o.HTTPClient = &http.Client{Transport: wire}
		return provider.FetchModelData(ctx, url, o)
	}
	loadModelData, modelDataSource = readModelDataCopy, cliDataURL
	keepModelData(t)
	t.Cleanup(func() {
		fetchModelData, loadModelData, modelDataSource = oldFetch, oldLoad, oldSource
	})
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	r := &dataRun{e: e, warnings: warnings, wire: wire}
	r.copy, r.attempt = modelDataPaths(dirs)
	return r
}

// seed writes doc as the copy, from source, fetched an hour ago.
func (r *dataRun) seed(t *testing.T, doc []byte, source string) *provider.ModelData {
	t.Helper()
	d := cliParsed(t, doc, provider.ModelDataStamp{Source: source, Fetched: time.Now().UTC().Add(-time.Hour).Truncate(time.Second), ETag: `"seed"`})
	if err := writeModelDataCopy(r.copy, d); err != nil {
		t.Fatal(err)
	}
	return d
}

func (r *dataRun) refresh(t *testing.T, mode modelDataMode, force bool) modelDataOutcome {
	t.Helper()
	return r.e.refreshModelData(t.Context(), mode, force)
}

func (r *dataRun) lastAttempt(t *testing.T) modelDataAttempt {
	t.Helper()
	a, ok, err := readModelDataAttempt(r.attempt)
	if !ok {
		t.Fatalf("attempt.json: %v", err)
	}
	return a
}

func fileBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func tempsIn(t *testing.T, dir string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(dir, ".*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// M1: <cache>/model-data/, the cache role resolved as gobble resolves it
// (0072-MADR D1).
func TestModelDataPath_0072(t *testing.T) {
	root := t.TempDir()
	at := func(parts ...string) string { return filepath.Join(append([]string{root}, parts...)...) }
	for _, c := range []struct {
		name, goos string
		env        map[string]string
		want       string
	}{
		{"GOBBLE_CACHE_DIR", "linux", map[string]string{"GOBBLE_CACHE_DIR": at("c"), "GOBBLE_HOME": at("h")}, at("c")},
		{"GOBBLE_HOME", "linux", map[string]string{"GOBBLE_HOME": at("h")}, at("h")},
		{"XDG", "linux", map[string]string{"XDG_CACHE_HOME": at("x")}, at("x", "gobble")},
		{"default", "darwin", nil, at("home", ".cache", "gobble")},
		{"windows", "windows", nil, at("local", "gobble", "cache")},
	} {
		t.Run(c.name, func(t *testing.T) {
			env := func(k string) string { return c.env[k] }
			base := func() (appdirs.Base, error) {
				return appdirs.Base{Home: at("home"), AppData: at("roaming"), LocalAppData: at("local")}, nil
			}
			dirs, _, err := appdirs.Resolve(env, c.goos, base)
			if err != nil {
				t.Fatal(err)
			}
			cp, ap := modelDataPaths(dirs)
			if want := filepath.Join(c.want, "model-data", "models-dev.json"); cp != want {
				t.Errorf("copy %s, want %s", cp, want)
			}
			if want := filepath.Join(c.want, "model-data", "attempt.json"); ap != want {
				t.Errorf("attempt %s, want %s", ap, want)
			}
		})
	}
}

// M2: the copy's object, written then read: the document byte for byte,
// its sha256 matching, the stamp kept (0072-MADR D2, Q3).
func TestModelDataRoundTrip_0072(t *testing.T) {
	dir := t.TempDir()
	built := provider.CurrentModelData()
	if built == nil {
		t.Fatal("no current document to write: TestMain did not install the test data (0072-MADR M48)")
	}
	for name, s := range map[string]provider.ModelDataStamp{
		"validators":    {Source: cliDataURL, Fetched: time.Date(2026, 11, 25, 1, 2, 3, 0, time.UTC), ETag: `W/"abc"`, LastModified: "Wed, 25 Nov 2026 01:00:00 GMT"},
		"no_validators": {Source: cliDataURL, Fetched: time.Date(2026, 11, 25, 1, 2, 3, 0, time.UTC)},
	} {
		t.Run(name, func(t *testing.T) {
			d := built.Stamped(s)
			path := filepath.Join(dir, name, "models-dev.json")
			if err := writeModelDataCopy(path, d); err != nil {
				t.Fatal(err)
			}
			raw := fileBytes(t, path)
			var f modelDataFile
			if err := jsonx.UnmarshalStrict(raw, &f); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(f.Document)
			if !bytes.Equal(f.Document, d.Document()) || f.SHA256 != hex.EncodeToString(sum[:]) || f.Version != 1 ||
				f.Source != s.Source || f.Fetched != "2026-11-25T01:02:03Z" || !maps.Equal(f.Sections, d.Sections()) {
				t.Errorf("the file holds version %d, source %q, fetched %q, sections %v; document equal %v, sha256 %s",
					f.Version, f.Source, f.Fetched, f.Sections, bytes.Equal(f.Document, d.Document()), f.SHA256)
			}
			if (s.ETag == "") != !bytes.Contains(raw, []byte(`"etag"`)) || (s.LastModified == "") != !bytes.Contains(raw, []byte(`"lastModified"`)) {
				t.Errorf("etag and lastModified written %s, want only when sent", raw[:min(len(raw), 200)])
			}
			back, err := readModelDataCopy(path)
			if err != nil {
				t.Fatal(err)
			}
			if back.Stamp() != s || !bytes.Equal(back.Document(), d.Document()) {
				t.Errorf("read back %+v, document equal %v", back.Stamp(), bytes.Equal(back.Document(), d.Document()))
			}
		})
	}
}

// M3, R6: a write that fails leaves the old copy byte for byte and no
// temporary file; the copy and attempt.json are written by internal/fsx's
// atomic writer, which syncs (0072-MADR D2, its fifth addition).
func TestModelDataWriteKeepsOld_0072(t *testing.T) {
	path := filepath.Join(t.TempDir(), "models-dev.json")
	old := cliParsed(t, cliModelDoc(map[string]int{"openai": 2}), provider.ModelDataStamp{Source: cliDataURL})
	if err := writeModelDataCopy(path, old); err != nil {
		t.Fatal(err)
	}
	before := fileBytes(t, path)
	next := cliParsed(t, cliModelDoc(map[string]int{"openai": 3}), provider.ModelDataStamp{Source: cliDataURL})
	boom := errors.New("planted failure")
	for name, rename := range map[string]func(tmp, name string) error{
		// The temporary file is gone before its rename, as when a disk
		// error or another process took it.
		"temporary_file": func(tmp, name string) error {
			_ = os.Remove(tmp)
			return os.Rename(tmp, name)
		},
		"rename": func(string, string) error { return boom },
	} {
		t.Run(name, func(t *testing.T) {
			renameVia(t, rename)
			if err := writeModelDataCopy(path, next); err == nil {
				t.Fatal("the write succeeded")
			}
			if !bytes.Equal(fileBytes(t, path), before) {
				t.Error("the old copy changed")
			}
			if left := tempsIn(t, filepath.Dir(path)); len(left) > 0 {
				t.Errorf("temporary files left: %v", left)
			}
		})
	}
	t.Run("house_writer", func(t *testing.T) {
		var tmps []string
		renameVia(t, func(tmp, name string) error {
			tmps = append(tmps, filepath.Base(tmp))
			return os.Rename(tmp, name)
		})
		if err := writeModelDataCopy(path, next); err != nil {
			t.Fatal(err)
		}
		if err := writeModelDataAttempt(filepath.Join(filepath.Dir(path), "attempt.json"), modelDataAttempt{Source: "s", Started: "0", Result: attemptOK}); err != nil {
			t.Fatal(err)
		}
		if len(tmps) != 2 || !strings.HasPrefix(tmps[0], ".models-dev.json.gobble-") || !strings.HasPrefix(tmps[1], ".attempt.json.gobble-") {
			t.Errorf("renamed %q; want the copy and attempt.json each through internal/fsx's atomic writer", tmps)
		}
	})
}

// M4: a file that is not a copy is no copy, logged once, and kept until an
// accepted download replaces it (0072-MADR D3).
func TestModelDataUnreadable_0072(t *testing.T) {
	good, err := encodeModelData(cliParsed(t, cliModelDoc(map[string]int{"openai": 1}), provider.ModelDataStamp{Source: cliDataURL, Fetched: time.Now()}))
	if err != nil {
		t.Fatal(err)
	}
	var f modelDataFile
	if err := jsonx.UnmarshalStrict(good, &f); err != nil {
		t.Fatal(err)
	}
	with := func(edit func(*modelDataFile)) []byte {
		g := f
		edit(&g)
		b, err := jsonx.MarshalStrict(g)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	refused := []byte(`{"openai":"not a section"}`)
	refusedSum := sha256.Sum256(refused)
	for name, content := range map[string][]byte{
		"not_json":  []byte("not json"),
		"truncated": good[:len(good)/2],
		"version":   with(func(g *modelDataFile) { g.Version = 2 }),
		"sha256":    with(func(g *modelDataFile) { g.SHA256 = strings.Repeat("0", 64) }),
		"refused": with(func(g *modelDataFile) {
			g.Document, g.SHA256 = refused, hex.EncodeToString(refusedSum[:])
		}),
		"oversize":  append(bytes.Repeat([]byte(" "), modelDataLimit), good...),
		"directory": nil,
	} {
		t.Run(name, func(t *testing.T) {
			r := newDataRun(t)
			if content == nil {
				if err := os.MkdirAll(r.copy, 0o700); err != nil {
					t.Fatal(err)
				}
			} else if err := writeReplacing(r.copy, content); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				if d := r.e.loadModelDataCopy(t.Context(), r.copy); d != nil {
					t.Fatalf("read %+v as a copy", d.Stamp())
				}
			}
			if n := strings.Count(r.warnings.String(), "the copy cannot be read"); n != 1 || !strings.Contains(r.warnings.String(), r.copy) {
				t.Errorf("%d warnings, want one naming the path: %q", n, r.warnings.String())
			}
			if content != nil && !bytes.Equal(fileBytes(t, r.copy), content) {
				t.Error("the file was changed")
			}
		})
	}
}

// M5: two writers at once; a reader reads one whole file or the other,
// never a mix (0072-MADR D2).
func TestModelDataConcurrentWriters_0072(t *testing.T) {
	path := filepath.Join(t.TempDir(), "models-dev.json")
	a := cliParsed(t, cliModelDoc(map[string]int{"openai": 2}), provider.ModelDataStamp{Source: "a"})
	b := cliParsed(t, cliModelDoc(map[string]int{"openai": 5}), provider.ModelDataStamp{Source: "b"})
	paused, resume := pauseFirstRename(t)
	doneA := make(chan error, 1)
	go func() { doneA <- writeModelDataCopy(path, a) }()
	select {
	case <-paused:
	case <-time.After(10 * time.Second):
		t.Fatal("writer A never paused between its temporary file and its rename")
	}
	if err := writeModelDataCopy(path, b); err != nil {
		t.Fatal(err)
	}
	if got, err := readModelDataCopy(path); err != nil || got.Stamp().Source != "b" {
		t.Fatalf("with A paused, the reader read %v (%v); want B's whole copy", got, err)
	}
	close(resume)
	if err := <-doneA; err != nil {
		t.Fatalf("writer A: %v", err)
	}
	if got, err := readModelDataCopy(path); err != nil || got.Stamp().Source != "a" || !bytes.Equal(got.Document(), a.Document()) {
		t.Fatalf("after A's rename the reader read %v (%v); want A's whole copy", got, err)
	}
}

// M6, R4, R5: temporary files a killed process left, the copy's and
// attempt.json's, are removed by an accepted write when an hour old;
// younger ones are kept; a cache path holding a glob's metacharacters is
// swept as any other (0072-MADR D2).
func TestModelDataStaleTemps_0072(t *testing.T) {
	// A glob's class, legal in a path on every platform.
	dir := filepath.Join(t.TempDir(), "cache [1]")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	oldNow := modelDataNow
	t.Cleanup(func() { modelDataNow = oldNow })
	modelDataNow = func() time.Time { return now }
	for name, age := range map[string]time.Duration{
		".models-dev.json.stale.tmp": 2 * time.Hour, ".models-dev.json.hour.tmp": time.Hour,
		".models-dev.json.young.tmp": 59 * time.Minute, ".models-dev.json.fresh.tmp": time.Second,
		".attempt.json.stale.tmp": 2 * time.Hour, ".attempt.json.young.tmp": time.Minute,
		".other.json.stale.tmp": 2 * time.Hour,
	} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, now.Add(-age), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeModelDataCopy(filepath.Join(dir, "models-dev.json"), cliParsed(t, cliModelDoc(map[string]int{"xai": 1}), provider.ModelDataStamp{})); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			left = append(left, e.Name())
		}
	}
	want := []string{".attempt.json.young.tmp", ".models-dev.json.fresh.tmp", ".models-dev.json.young.tmp", ".other.json.stale.tmp"}
	if !slices.Equal(left, want) {
		t.Errorf("left %v, want %v", left, want)
	}
}

// M9 (the engine's half): the copy's validators are sent; a 304 renews the
// copy's time and keeps its document (0072-MADR D4).
func TestFetchModelDataConditional_0072(t *testing.T) {
	r := newDataRun(t, dataReply{status: http.StatusNotModified})
	seeded := r.seed(t, cliModelDoc(map[string]int{"openai": 3}), cliDataURL)
	out := r.refresh(t, modelDataBackground, false)
	reqs := r.wire.requests()
	if len(reqs) != 1 || reqs[0].Header.Get("If-None-Match") != `"seed"` {
		t.Fatalf("requests %d; want one sending the copy's ETag", len(reqs))
	}
	got, err := readModelDataCopy(r.copy)
	if err != nil {
		t.Fatal(err)
	}
	if out.Result != attemptNotModified || !bytes.Equal(got.Document(), seeded.Document()) || !got.Stamp().Fetched.After(seeded.Stamp().Fetched) ||
		got.Stamp().ETag != `"seed"` || provider.CurrentModelData() != out.Data {
		t.Errorf("result %s, copy %+v; want not_modified, the document kept, a new time, current", out.Result, got.Stamp())
	}
	if a := r.lastAttempt(t); a.Result != attemptNotModified {
		t.Errorf("attempt %+v", a)
	}
}

// M11 to M13, M78, M82: a failed fetch changes nothing on disk or in memory
// and is recorded (0072-MADR D5; "As a failed download (Recommended)";
// "Follow any, but refuse https→http (Recommended)").
func TestModelDataRefreshFails_0072(t *testing.T) {
	pad := append(append([]byte(`{"xai":{"models":{"g":{}}},"pad":"`), bytes.Repeat([]byte("a"), 33<<20)...), `"}`...)
	for name, c := range map[string]struct {
		replies  []dataReply
		reason   string
		says     string
		requests int
	}{
		"offline":             {[]dataReply{{err: errors.New("dial tcp: connection refused")}}, "offline", "", 1},
		"status":              {[]dataReply{{status: http.StatusServiceUnavailable}}, "status", "", 1},
		"oversize":            {[]dataReply{{status: http.StatusOK, body: pad}}, "not_document", "the reply is over 32 MiB", 1},
		"not_document":        {[]dataReply{{status: http.StatusOK, body: []byte("<html>")}}, "not_document", "", 1},
		"section_undecodable": {[]dataReply{{status: http.StatusOK, body: []byte(`{"openai":{"models":5},"xai":{"models":{"g":{}}}}`)}}, "not_document", "", 1},
		"redirect_downgrade": {[]dataReply{
			{status: http.StatusFound, header: http.Header{"Location": {"http://models-dev.invalid/api.json"}}},
			{status: http.StatusOK, body: cliModelDoc(map[string]int{"openai": 2})},
		}, "redirect_downgrade", "", 1},
	} {
		t.Run(name, func(t *testing.T) {
			r := newDataRun(t, c.replies...)
			var fetchErr error
			inner := fetchModelData
			fetchModelData = func(ctx context.Context, url string, o provider.Options) (*provider.ModelData, bool, error) {
				d, nm, err := inner(ctx, url, o)
				fetchErr = err
				return d, nm, err
			}
			r.seed(t, cliModelDoc(map[string]int{"openai": 2}), cliDataURL)
			before := fileBytes(t, r.copy)
			current := cliParsed(t, cliModelDoc(map[string]int{"xai": 1}), provider.ModelDataStamp{Source: "current"})
			provider.SetCurrentModelData(current)
			out := r.refresh(t, modelDataBackground, false)
			if out.Result != attemptFailed || out.Reason != c.reason {
				t.Errorf("result %s %s, want failed %s", out.Result, out.Reason, c.reason)
			}
			if c.says != "" && (fetchErr == nil || !strings.Contains(fetchErr.Error(), c.says)) {
				t.Errorf("the fetch's error %v, want one saying %q", fetchErr, c.says)
			}
			if n := len(r.wire.requests()); n != c.requests {
				t.Errorf("%d requests, want %d", n, c.requests)
			}
			if !bytes.Equal(fileBytes(t, r.copy), before) || provider.CurrentModelData() != current {
				t.Error("a failed fetch changed the copy or the current document")
			}
			if a := r.lastAttempt(t); a.Result != attemptFailed || a.Reason != c.reason || a.Finished == "" || a.Source != cliDataURL {
				t.Errorf("attempt %+v", a)
			}
		})
	}
}

// M14 to M17: the guards; a refused download changes neither the copy nor
// the current document (0072-MADR D6).
func TestModelDataGuards_0072(t *testing.T) {
	have := map[string]int{"openai": 4, "anthropic": 4}
	for name, c := range map[string]struct {
		source string
		copy   map[string]int
		got    []byte
		force  bool
		ok     bool
	}{
		"empty":             {cliDataURL, have, []byte(`{"openai":{"models":{}},"other":{"models":{"x":{}}}}`), false, false},
		"empty_force":       {cliDataURL, have, []byte(`{"openai":{"models":{}}}`), true, false},
		"section_missing":   {cliDataURL, have, cliModelDoc(map[string]int{"anthropic": 8}), false, false},
		"section_malformed": {cliDataURL, have, []byte(`{"openai":{},"anthropic":{"models":{"a":{},"b":{},"c":{},"d":{},"e":{},"f":{},"g":{},"h":{}}}}`), false, false},
		"shrink_below_half": {cliDataURL, have, cliModelDoc(map[string]int{"openai": 1, "anthropic": 4}), false, false},
		"shrink_half":       {cliDataURL, have, cliModelDoc(map[string]int{"openai": 2, "anthropic": 4}), false, true},
		"first":             {"", nil, cliModelDoc(map[string]int{"xai": 1}), false, true},
		"other_source":      {"https://other.invalid/api.json", map[string]int{"openai": 10}, cliModelDoc(map[string]int{"openai": 1}), false, true},
		"force":             {cliDataURL, have, cliModelDoc(map[string]int{"xai": 1}), true, true},
	} {
		t.Run(name, func(t *testing.T) {
			r := newDataRun(t, dataReply{status: http.StatusOK, body: c.got})
			var before []byte
			if c.copy != nil {
				r.seed(t, cliModelDoc(c.copy), c.source)
				before = fileBytes(t, r.copy)
			}
			current := cliParsed(t, cliModelDoc(map[string]int{"google": 1}), provider.ModelDataStamp{Source: "current"})
			provider.SetCurrentModelData(current)
			out := r.refresh(t, modelDataCommand, c.force)
			switch {
			case c.ok && (out.Result != attemptOK || provider.CurrentModelData() != out.Data):
				t.Errorf("result %s (%s), want ok and the download current", out.Result, out.Reason)
			case !c.ok && (out.Result != attemptRefused || before != nil && !bytes.Equal(fileBytes(t, r.copy), before)):
				t.Errorf("result %s (%s), want refused, the copy kept", out.Result, out.Reason)
			case !c.ok && provider.CurrentModelData() != current:
				t.Error("a refused download changed the current document")
			case !c.ok && r.lastAttempt(t).Result != attemptRefused:
				t.Errorf("attempt %+v", r.lastAttempt(t))
			}
		})
	}
}

// M59 (the engine's half): 30 s in the background, 120 s for the command
// (0072-MADR D4; "Background 30 s, refresh cmd 120 s (Recommended)").
func TestFetchModelDataDeadline_0072(t *testing.T) {
	for mode, want := range map[modelDataMode]time.Duration{modelDataBackground: 30 * time.Second, modelDataCommand: 120 * time.Second} {
		synctest.Test(t, func(t *testing.T) {
			r := newDataRun(t)
			fetchModelData = func(ctx context.Context, _ string, _ provider.Options) (*provider.ModelData, bool, error) {
				<-ctx.Done()
				return nil, false, ctx.Err()
			}
			start := time.Now()
			out := r.refresh(t, mode, false)
			if out.Reason != "timeout" || time.Since(start) != want {
				t.Errorf("mode %d: %s after %v; want a timeout at %v", mode, out.Reason, time.Since(start), want)
			}
		})
	}
}

// M65, R7a: the engine runs in its own goroutine: starting it returns at
// once, and while its fetch, FetchModelData itself, blocks mid-request, a
// prompt whose turn reads the model facts (the current document) completes
// with a tool call, and a reader takes the current document without a lock
// (0072-MADR D18; case 7).
func TestModelDataEngineNeverBlocks_0072(t *testing.T) {
	r := newDataRun(t)
	block := blockedFetch{holding: make(chan struct{}, 1), release: make(chan struct{})}
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(block.release) }) })
	fetchModelData = func(ctx context.Context, url string, o provider.Options) (*provider.ModelData, bool, error) {
		o.HTTPClient = &http.Client{Transport: block}
		return provider.FetchModelData(ctx, url, o)
	}
	started := make(chan bool, 1)
	go func() { started <- r.e.startModelDataRefresh(modelDataBackground, false) }()
	select {
	case ok := <-started:
		if !ok {
			t.Fatal("the engine did not start")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("starting the engine waited for its fetch")
	}
	select {
	case <-block.holding:
	case <-time.After(10 * time.Second):
		t.Fatal("the engine's fetch never reached its request")
	}
	if err := os.WriteFile("f.txt", []byte("file text"), 0o600); err != nil {
		t.Fatal(err)
	}
	withModel(t, llmtest.NewScript(llmtest.ToolCall("r1", "read", `{"path":"f.txt"}`), llmtest.Text("done")))
	type ran struct {
		code        int
		out, stderr string
	}
	done := make(chan ran, 1)
	go func() {
		var out, errw bytes.Buffer
		code := Main(t.Context(), []string{"--no-session", "-p", "read f.txt"}, nil, &out, &errw)
		done <- ran{code, out.String(), errw.String()}
	}()
	select {
	case got := <-done:
		if got.code != ExitOK || got.out != "done\n" {
			t.Fatalf("with the fetch blocked: exit %d, stdout %q, stderr %q", got.code, got.out, got.stderr)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the prompt waited for the engine's fetch")
	}
	read := make(chan *provider.ModelData, 1)
	go func() { read <- provider.CurrentModelData() }()
	select {
	case d := <-read:
		if d == nil {
			t.Error("no current document while the fetch blocks")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reading the current document waited for the engine's fetch")
	}
	once.Do(func() { close(block.release) })
}

// M66: attempt.json's shape, an unreadable one, two writers (0072-MADR D18).
func TestModelDataAttemptFile_0072(t *testing.T) {
	t.Run("shape", func(t *testing.T) {
		r := newDataRun(t, dataReply{status: http.StatusOK, body: cliModelDoc(map[string]int{"xai": 2})})
		seen := make(chan modelDataAttempt, 1)
		old := fetchModelData
		fetchModelData = func(ctx context.Context, url string, o provider.Options) (*provider.ModelData, bool, error) {
			a, _, _ := readModelDataAttempt(r.attempt)
			seen <- a
			return old(ctx, url, o)
		}
		r.refresh(t, modelDataBackground, false)
		at := <-seen
		if at.Version != 1 || at.Source != cliDataURL || at.Result != attemptStarted || at.Started == "" || at.Finished != "" {
			t.Errorf("at the fetch: %+v; want started", at)
		}
		end := r.lastAttempt(t)
		if end.Result != attemptOK || end.Started != at.Started || end.Finished == "" {
			t.Errorf("at the end: %+v", end)
		}
		var keys map[string]any
		if err := jsonx.Unmarshal(fileBytes(t, r.attempt), &keys); err != nil {
			t.Fatal(err)
		}
		if got := slices.Sorted(maps.Keys(keys)); !slices.Equal(got, []string{"finished", "result", "source", "started", "version"}) {
			t.Errorf("keys %v", got)
		}
	})
	t.Run("unreadable", func(t *testing.T) {
		r := newDataRun(t, dataReply{status: http.StatusServiceUnavailable})
		if err := writeReplacing(r.attempt, []byte("{not json")); err != nil {
			t.Fatal(err)
		}
		if _, ok, err := readModelDataAttempt(r.attempt); ok || err == nil || errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("an unreadable attempt read as one (%v)", err)
		}
		r.refresh(t, modelDataBackground, false)
		if a := r.lastAttempt(t); a.Result != attemptFailed {
			t.Errorf("not replaced: %+v", a)
		}
	})
	t.Run("two_writers", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "attempt.json")
		if err := writeModelDataAttempt(path, modelDataAttempt{Source: "s", Started: "0", Result: attemptOK}); err != nil {
			t.Fatal(err)
		}
		paused, resume := pauseFirstRename(t)
		done := make(chan error, 1)
		go func() {
			done <- writeModelDataAttempt(path, modelDataAttempt{Source: "s", Started: "1", Result: attemptStarted})
		}()
		select {
		case <-paused:
		case <-time.After(10 * time.Second):
			t.Fatal("the first writer never paused before its rename")
		}
		if a, ok, err := readModelDataAttempt(path); !ok || a.Started != "0" {
			t.Fatalf("mid-write the reader read %+v (%v); want the old whole file", a, err)
		}
		if err := writeModelDataAttempt(path, modelDataAttempt{Source: "s", Started: "2", Result: attemptFailed}); err != nil {
			t.Fatal(err)
		}
		close(resume)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if a, ok, err := readModelDataAttempt(path); !ok || a.Started != "1" {
			t.Errorf("last writer: %+v (%v); want the last rename's, whole", a, err)
		}
	})
}

// M69: a download that cannot be saved is current for the run, logged
// once, and recorded not_saved (0072-MADR D18; "Use it this run, log, wait
// (Recommended)").
func TestModelDataNotSaved_0072(t *testing.T) {
	r := newDataRun(t, dataReply{status: http.StatusOK, body: cliModelDoc(map[string]int{"openai": 3})})
	renameVia(t, func(tmp, name string) error {
		if filepath.Base(name) == "models-dev.json" {
			return errors.New("planted: the copy cannot be saved")
		}
		return os.Rename(tmp, name)
	})
	for range 2 {
		out := r.refresh(t, modelDataBackground, false)
		if out.Result != attemptNotSaved || out.Data == nil || provider.CurrentModelData() != out.Data {
			t.Fatalf("result %s, current %p, data %p; want not_saved and the download current", out.Result, provider.CurrentModelData(), out.Data)
		}
	}
	if _, err := os.Stat(r.copy); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a copy was written (%v)", err)
	}
	if n := strings.Count(r.warnings.String(), "could not be saved"); n != 1 {
		t.Errorf("%d warnings, want one: %q", n, r.warnings.String())
	}
	if a := r.lastAttempt(t); a.Result != attemptNotSaved {
		t.Errorf("attempt %+v", a)
	}
}

// M19, M74 (the listings' half): a listing's Options carry the current
// document, the copy once one is set, and the listing ranks by it; the
// options New and the sign-in wizard are given carry none (0071-MADR P6;
// 0072-MADR D7, M74, which re-points 0071's
// TestListingRanksByBuiltInCopy_0071 here).
func TestModelDataCurrent_0072(t *testing.T) {
	t.Run("listing", func(t *testing.T) {
		keepModelData(t)
		// The copy states o3-mini takes images and gpt-5-mini does not.
		doc := []byte(`{"openai":{"models":{"o3-mini":{"name":"o3 mini","modalities":{"input":["text","image"],"output":["text"]}},` +
			`"gpt-5-mini":{"name":"GPT-5 mini","modalities":{"input":["text"],"output":["text"]}}}}}`)
		copyData := cliParsed(t, doc, provider.ModelDataStamp{Source: cliDataURL})
		provider.SetCurrentModelData(copyData)
		e, _ := modelEnv(t)
		t.Setenv("OPENAI_API_KEY", "test-only")
		listed := make(chan provider.Options, 1)
		withLister(t, func(_ context.Context, _, _ string, o provider.Options) (provider.Listing, error) {
			listed <- o
			return provider.Listing{}, errors.New("recorded")
		})
		l := e.modelListing(provider.Selection{ID: "openai"})
		l.start()
		o := <-listed
		if o.ModelData != copyData {
			t.Fatalf("the listing's Options carry %p, want the current document %p", o.ModelData, copyData)
		}
		lo := o
		lo.HTTPClient, lo.Cache, lo.Sessions, lo.LastGood = &http.Client{Transport: builtInListing{}}, nil, nil, nil
		got, err := provider.List(t.Context(), "openai", fmt.Sprintf("sk-copy-%d-%d", os.Getpid(), time.Now().UnixNano()), lo)
		if err != nil || !got.Live {
			t.Fatalf("listing %+v (%v), want live", got, err)
		}
		if vision := provider.SearchListing(got, "vision"); !slices.Equal(vision, []string{"o3-mini"}) {
			t.Errorf("vision %q, want the copy's [o3-mini]", vision)
		}
		if po := e.providerOptions("openai", provider.Options{}); po.ModelData != nil {
			t.Error("the options New and the sign-in wizard are given carry a document")
		}
	})
}

// renameVia makes fsx's writes outside a workspace rename through rename
// for t (fsx.SetRename).
func renameVia(t *testing.T, rename func(tmp, name string) error) {
	t.Helper()
	t.Cleanup(fsx.SetRename(rename))
}

// pauseFirstRename pauses the first rename fsx makes for t until resume is
// closed, closing paused when it starts to wait.
func pauseFirstRename(t *testing.T) (paused, resume chan struct{}) {
	t.Helper()
	paused, resume = make(chan struct{}), make(chan struct{})
	var once sync.Once
	renameVia(t, func(tmp, name string) error {
		first := false
		once.Do(func() { first = true })
		if first {
			close(paused)
			<-resume
		}
		return os.Rename(tmp, name)
	})
	return paused, resume
}

// blockedFetch answers the engine's request only when release is closed or
// the request's context ends; it signals each request it holds.
type blockedFetch struct {
	holding chan struct{}
	release chan struct{}
}

func (b blockedFetch) RoundTrip(r *http.Request) (*http.Response, error) {
	b.holding <- struct{}{}
	select {
	case <-b.release:
	case <-r.Context().Done():
	}
	return nil, errors.New("released")
}
