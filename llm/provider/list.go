package provider

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

	"github.com/maccavelli/gobble-cli/llm"
)

// errNothingListed is a listing that answered but recommended no model.
var errNothingListed = errors.New("provider: the listing recommended no model")

// Listing is one model listing of a provider (0020-MADR D2).
type Listing struct {
	// Models are the provider's full curated catalog, ranked by the
	// capable profile, when Live; otherwise its static models.
	Models []string
	// Live reports whether Models came from the provider's listing.
	Live bool
	// Err is why the listing was not live; nil when it was.
	Err error
	// Source is where a live listing came from: "live", read now or within
	// the SDK's ten minutes, or "stale", the last good listing after a
	// transient failure; "" when not live (0025-MADR D4).
	Source string
	// FetchedAt is when a live listing was read from the provider.
	FetchedAt time.Time
	// Usable are every usable id of a live listing, uncapped, as the SDK's
	// WithLastGood takes them back.
	Usable []string
}

// publicListings are the providers whose listing needs no key
// (go-llmprovider-sdk v1.17.0, llmprovider/catalog/discovery.go: OpenCode's
// /models at 1012, Hugging Face's at 1045, Kilo's at 1251).
var publicListings = []string{"opencode-zen", "opencode-go", "huggingface", "kilo"}

// PublicListing reports whether provider id's listing needs no key, so a
// listing cannot reject one: OpenCode Zen and Go, Hugging Face and Kilo
// (0025-MADR D4).
func PublicListing(id string) bool { return slices.Contains(publicListings, id) }

// Search is the ids of models that match query, best first, by the SDK's
// matcher: the id exactly, as a prefix or substring, then its label, then a
// subsequence (0025-MADR D5).
func Search(id string, models []string, query string) []string {
	var out []string
	for _, m := range catalog.Search(llmprovider.ProviderID(id), models, query) {
		out = append(out, m.ID)
	}
	return out
}

// List lists the built-in provider id's models with apiKey, through the
// SDK's catalog.List with the capable profile, model metadata on and no
// limit. It sends no generation and is bounded by the SDK at 10 s. It
// passes no logger, so the SDK logs nothing of it. A listing that fails or
// recommends nothing gives the static models, Live false and the reason in
// Err; only an unknown id or a refused option is an error. o's APIKey,
// Model and Logger are ignored. With no apiKey, a provider with an OAuth
// login lists with o.Logins' stored login (0005-PLAN F4-3b).
func List(ctx context.Context, id, apiKey string, o Options) (Listing, error) {
	opts := []llmprovider.Option{
		catalog.WithProfile(catalog.ProfileCapable),
		llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),
	}
	opts = append(opts, o.sdkOptions()...)
	if o.ModelMetadataURL != "" {
		opts = append(opts, llmprovider.WithModelMetadataURL(o.ModelMetadataURL))
	}
	if len(o.LastGood) > 0 {
		opts = append(opts, catalog.WithLastGood(o.LastGood, o.LastGoodAt))
	}
	var src llmprovider.TokenSource = llmprovider.NewStaticToken(apiKey)
	o.APIKey, o.Logger = apiKey, nil
	login, err := o.login(ctx, id) // with no key, id's stored login (0005-PLAN F4-3b decision 9)
	if err != nil {
		return Listing{}, fmt.Errorf("provider %s: list models: %w", id, err)
	}
	if login != nil {
		src = login
	}
	cat, err := catalog.List(ctx, llmprovider.ProviderID(id), src, opts...)
	if err != nil {
		return Listing{}, fmt.Errorf("provider %s: list models: %w", id, mapError(err))
	}
	if !cat.Live || len(cat.Recommended) == 0 {
		reason := mapError(cat.Err)
		if reason == nil {
			reason = errNothingListed
		}
		return Listing{Models: Models(id), Err: reason}, nil
	}
	return Listing{Models: cat.Recommended, Live: true, Source: string(cat.Source), FetchedAt: cat.FetchedAt, Usable: cat.Usable}, nil
}

// failureKinds name the errors FailureKind knows, most specific first.
var failureKinds = []struct {
	err  error
	kind string
}{
	{llm.ErrQuota, "quota"},
	{llm.ErrRateLimited, "rate-limited"},
	{llm.ErrAuth, "auth"},
	{llm.ErrNotPermitted, "not-permitted"},
	{llm.ErrUnavailable, "unavailable"},
	{context.DeadlineExceeded, "timeout"},
	{context.Canceled, "canceled"},
	{errNothingListed, "empty"},
}

// FailureKind names err's class for a log line, and never carries its
// text, which a provider may fill with part of the credential
// (0020-MADR F5): quota, rate-limited, auth, not-permitted, unavailable,
// timeout, canceled, empty, or other.
func FailureKind(err error) string {
	for _, k := range failureKinds {
		if errors.Is(err, k.err) {
			return k.kind
		}
	}
	return "other"
}
