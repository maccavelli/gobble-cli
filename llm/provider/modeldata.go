package provider

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
)

// ModelDataURL is models.dev's document, the one source of gobble's model
// data, which gobble fetches itself (0072-MADR D4, M23).
const ModelDataURL = "https://models.dev/api.json"

// ModelDataStamp is where and when a document came from: the URL asked for
// (not the one a redirect reached), the time of the reply, in UTC to the
// second, and the reply's ETag and Last-Modified, which a later fetch from
// the same URL sends back (0072-MADR D2, D4, M77). The zero stamp is a
// document that was never fetched, and no current document has one.
type ModelDataStamp struct {
	Source       string
	Fetched      time.Time
	ETag         string
	LastModified string
}

// ModelData is a parsed models.dev document and its stamp, immutable once
// made: what a listing ranks by and reads its menu's facts from, handed in
// through Options.ModelData, and what the lookups read as their models.dev
// layer (0071-MADR P3; 0072-MADR D7, D21). go-llmprovider-sdk fetches no
// document from v1.33.0. A ModelData is safe for concurrent use.
type ModelData struct {
	doc   *parsedModelData
	stamp ModelDataStamp
}

// parsedModelData is one parse, shared by every stamp of it: the SDK's
// Metadata, and, made once when first asked for, the document's compact
// bytes and its models per section.
type parsedModelData struct {
	meta     catalog.Metadata
	once     sync.Once
	compact  []byte
	sections map[string]int
}

// modelDataSections are the document's nine sections the SDK reads, by
// their key in the document, with the provider each serves.
var modelDataSections = map[string]llmprovider.ProviderID{
	"opencode": llmprovider.ProviderOpencodeZen, "opencode-go": llmprovider.ProviderOpencodeGo,
	"huggingface": llmprovider.ProviderHuggingFace, "togetherai": llmprovider.ProviderTogether,
	"openai": llmprovider.ProviderOpenAI, "anthropic": llmprovider.ProviderClaude, "google": llmprovider.ProviderGemini,
	"xai": llmprovider.ProviderGrok, string(llmprovider.ProviderKilo): llmprovider.ProviderKilo,
}

// ParseModelData reads a models.dev document from r with the SDK's parser,
// which keeps the nine sections it reads, decodes each now, and refuses a
// document over 32 MiB, one that is not a document, or one with a section
// it cannot decode (catalog.ParseMetadata; 0072-MADR F5, M78). The result
// has the zero stamp; Stamped gives it one.
func ParseModelData(r io.Reader) (*ModelData, error) {
	meta, err := catalog.ParseMetadata(r)
	if err != nil {
		return nil, fmt.Errorf("provider: model data: %w", err)
	}
	return &ModelData{doc: &parsedModelData{meta: meta}}, nil
}

// Stamp is where and when d came from; the zero stamp for nil.
func (d *ModelData) Stamp() ModelDataStamp {
	if d == nil {
		return ModelDataStamp{}
	}
	return d.stamp
}

// Stamped is d's document under stamp s: the same parse, read by both, and
// d unchanged (0072-MADR M77).
func (d *ModelData) Stamped(s ModelDataStamp) *ModelData {
	return &ModelData{doc: d.doc, stamp: s}
}

// Document is the SDK's Encode of d, the nine sections' bytes as read, in
// the compact form gobble's strict JSON writes and reads back byte for byte
// (0072-MADR D2, the PLAN's Phase 0 step 2), as the caller's own copy. A
// document the strict form refuses (a repeated section, invalid UTF-8) is
// Encode's bytes as they are, which then cannot be saved (M69).
func (d *ModelData) Document() []byte {
	d.doc.made()
	return bytes.Clone(d.doc.compact)
}

// Sections are d's models per section, by the section's key in the
// document, for each section holding at least one model (0072-MADR D2, D6).
// The map is the caller's own.
func (d *ModelData) Sections() map[string]int {
	d.doc.made()
	return maps.Clone(d.doc.sections)
}

// made fills compact and sections, once.
func (p *parsedModelData) made() {
	p.once.Do(func() {
		var buf bytes.Buffer
		_ = p.meta.Encode(&buf) //nolint:errcheck // a bytes.Buffer's write does not fail
		p.compact = buf.Bytes()
		if v, err := jsonx.MarshalStrict(jsontext.Value(p.compact)); err == nil {
			p.compact = v
		}
		p.sections = map[string]int{}
		for key, id := range modelDataSections {
			if n := len(p.meta.Models(id)); n > 0 {
				p.sections[key] = n
			}
		}
	})
}

// options are the SDK options that hand d to a listing; nil hands none.
func (d *ModelData) options() []llmprovider.Option {
	if d == nil {
		return nil
	}
	return []llmprovider.Option{catalog.WithMetadata(d.doc.meta)}
}

// current is the process's current document, set by SetCurrentModelData
// (0072-MADR D7).
var current atomic.Pointer[ModelData]

// CurrentModelData is the process's current document: the one
// SetCurrentModelData set last, else nil, and with nil every model is
// unknown: gobble carries no built-in copy (0072-MADR D7, D13, M19). Every
// lookup reads it as its one models.dev document, never two. It takes no
// lock.
func CurrentModelData() *ModelData { return current.Load() }

// SetCurrentModelData makes d the process's current document, for the
// lookups, the listings and the sessions that start after it; nil leaves
// none (0072-MADR D7, D19).
func SetCurrentModelData(d *ModelData) { current.Store(d) }

// FetchModelData GETs url once with o's client (o.HTTPClient, else the SDK's
// llmprovider.NewHTTPClient, which reads the proxy variables) and gobble's
// name and version as its User-Agent, and parses the reply with
// ParseModelData; it keeps nothing, so two calls are two requests
// (0072-MADR D4, M8). When o.ModelData came from url, its ETag and
// Last-Modified are sent as If-None-Match and If-Modified-Since, and a 304
// returns o.ModelData's document stamped anew, with true (M9, M77). A
// redirect is followed to any host, up to ten as Go's client follows them,
// but one from https to http fails the fetch before it is sent; the stamp's
// Source is url, whatever a redirect reached (M58, M82). The body is read up
// to 32 MiB (D4). ctx bounds it: the caller sets the deadline (M59).
//
// Its error is ctx's when ctx ended it; wraps llm.ErrNotPermitted for a
// redirect from https to http; wraps llm.ErrOffline when the request or its
// body could not be carried; is an *llm.APIError for a status other than
// 200 or 304, or a 304 to a request that sent no validator; and is
// otherwise the reply not being a document, one over 32 MiB included (M11 to
// M13, M78).
func FetchModelData(ctx context.Context, url string, o Options) (*ModelData, bool, error) {
	client := o.HTTPClient
	if client == nil {
		client = llmprovider.NewHTTPClient()
	}
	client = refusingDowngrade(client)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, false, fmt.Errorf("provider: model data: %w", err)
	}
	if o.ClientName != "" {
		req.Header.Set("User-Agent", o.ClientName+"/"+o.ClientVersion)
	}
	prev := o.ModelData
	sent := false
	if prev != nil && prev.stamp.Source == url {
		if prev.stamp.ETag != "" {
			req.Header.Set("If-None-Match", prev.stamp.ETag)
			sent = true
		}
		if prev.stamp.LastModified != "" {
			req.Header.Set("If-Modified-Since", prev.stamp.LastModified)
			sent = true
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, false, fetchError(ctx, url, err)
	}
	defer resp.Body.Close() //nolint:errcheck // a read-only body
	stamp := ModelDataStamp{Source: url, Fetched: time.Now().UTC().Truncate(time.Second),
		ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified")}
	switch {
	case resp.StatusCode == http.StatusNotModified && sent:
		stamp.ETag = cmp.Or(stamp.ETag, prev.stamp.ETag)
		stamp.LastModified = cmp.Or(stamp.LastModified, prev.stamp.LastModified)
		return prev.Stamped(stamp), true, nil
	case resp.StatusCode == http.StatusNotModified:
		return nil, false, &llm.APIError{Status: resp.StatusCode, Provider: "models.dev",
			Message: "a 304 to a request that sent no validator"}
	case resp.StatusCode != http.StatusOK:
		return nil, false, &llm.APIError{Status: resp.StatusCode, Provider: "models.dev"}
	}
	body := &readErrors{r: resp.Body, left: modelDataBodyLimit}
	d, err := ParseModelData(body)
	if err != nil {
		if body.over {
			return nil, false, fmt.Errorf("provider: model data from %s: the reply is over %d MiB", url, modelDataBodyLimit>>20)
		}
		if body.err != nil {
			return nil, false, fetchError(ctx, url, body.err)
		}
		return nil, false, fmt.Errorf("provider: model data from %s is not a models.dev document: %w", url, err)
	}
	return d.Stamped(stamp), false, nil
}

// fetchError is err from a fetch of url: ctx's error when ctx ended it,
// else the network's, as llm.ErrOffline.
func fetchError(ctx context.Context, url string, err error) error {
	if ce := ctx.Err(); ce != nil {
		return fmt.Errorf("provider: model data: fetch %s: %w", url, ce)
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, llm.ErrNotPermitted) {
		return fmt.Errorf("provider: model data: fetch %s: %w", url, err)
	}
	return fmt.Errorf("provider: model data: fetch %s: %w: %w", url, llm.ErrOffline, err)
}

// modelDataBodyLimit bounds the body FetchModelData reads (0072-MADR D4).
const modelDataBodyLimit = 32 << 20

// errModelDataOver ends a read past modelDataBodyLimit.
var errModelDataOver = errors.New("the reply is over the limit")

// errRedirectDowngrade refuses a redirect from https to http (0072-MADR
// M82, the owner's "Follow any, but refuse https→http (Recommended)").
var errRedirectDowngrade = fmt.Errorf("a redirect from https to http: %w", llm.ErrNotPermitted)

// refusingDowngrade is a copy of c whose redirects stop at one from https
// to http; any other goes to c's own CheckRedirect, or, with none, is
// followed as Go's client follows it, up to ten.
func refusingDowngrade(c *http.Client) *http.Client {
	out := *c
	next := c.CheckRedirect
	out.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 && via[len(via)-1].URL.Scheme == "https" && req.URL.Scheme == "http" {
			return errRedirectDowngrade
		}
		if next != nil {
			return next(req, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return &out
}

// readErrors reads r, at most left bytes, and keeps the first error a read
// returns but io.EOF, so a body the network cut is told from one that is
// not a document; over reports a body past the limit.
type readErrors struct {
	r    io.Reader
	left int64
	over bool
	err  error
}

func (b *readErrors) Read(p []byte) (int, error) {
	if b.left <= 0 {
		var one [1]byte
		n, err := b.r.Read(one[:])
		switch {
		case n > 0:
			b.over = true
			return 0, errModelDataOver
		case err != nil && !errors.Is(err, io.EOF):
			if b.err == nil {
				b.err = err
			}
			return 0, err
		}
		return 0, io.EOF
	}
	if int64(len(p)) > b.left {
		p = p[:b.left]
	}
	n, err := b.r.Read(p)
	b.left -= int64(n)
	if err != nil && !errors.Is(err, io.EOF) && b.err == nil {
		b.err = err
	}
	return n, err
}
