package provider

import (
	"context"
	"errors"
	"fmt"

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
}

// List lists the built-in provider id's models with apiKey, through the
// SDK's catalog.List with the capable profile, model metadata on and no
// limit. It sends no generation and is bounded by the SDK at 10 s. It
// passes no logger, so the SDK logs nothing of it. A listing that fails or
// recommends nothing gives the static models, Live false and the reason in
// Err; only an unknown id or a refused option is an error. o's APIKey,
// Model and Logger are ignored.
func List(ctx context.Context, id, apiKey string, o Options) (Listing, error) {
	opts := []llmprovider.Option{
		catalog.WithProfile(catalog.ProfileCapable),
		llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),
	}
	opts = append(opts, o.sdkOptions()...)
	if o.ModelMetadataURL != "" {
		opts = append(opts, llmprovider.WithModelMetadataURL(o.ModelMetadataURL))
	}
	cat, err := catalog.List(ctx, llmprovider.ProviderID(id), llmprovider.NewStaticToken(apiKey), opts...)
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
	return Listing{Models: cat.Recommended, Live: true}, nil
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
