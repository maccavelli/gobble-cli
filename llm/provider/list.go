package provider

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"

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
	// catalog is the SDK's listing when this one is live, so a search can
	// read its facts. held says it was stored. Both are empty when the
	// listing is not live (0036-MADR D6).
	catalog catalog.Catalog
	held    bool
}

// publicListings are the providers whose listing needs no key
// (go-llmprovider-sdk v1.17.0, llmprovider/catalog/discovery.go: OpenCode's
// /models at 1012, Hugging Face's at 1045, Kilo's at 1251).
var publicListings = []string{"opencode-zen", "opencode-go", "huggingface", "kilo"}

// PublicListing reports whether provider id's listing needs no key, so a
// listing cannot reject one: OpenCode Zen and Go, Hugging Face and Kilo
// (0025-MADR D4).
func PublicListing(id string) bool { return slices.Contains(publicListings, id) }

// keysCheckedAtSignIn are the public-listing providers whose key a sign-in
// checks anyway: Kilo's, at Kilo's profile (go-llmprovider-sdk v1.32.0,
// 0080-MADR; 0068-MADR D2 b).
var keysCheckedAtSignIn = []string{"kilo"}

// KeyCheckedAtSignIn reports whether a sign-in to provider id checks a typed
// key although its listing needs none, so the key is not "unchecked": Kilo's
// (0068-MADR D2 b). PublicListing does not change.
func KeyCheckedAtSignIn(id string) bool { return slices.Contains(keysCheckedAtSignIn, id) }

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

// SearchListing is the ids of l's models that match query, best first, by
// the listing's facts: its name, its family, and the features those facts
// state (0036-MADR D6). A listing with no catalog matches nothing, as does
// an empty query.
func SearchListing(l Listing, query string) []string {
	if !l.held {
		return nil
	}
	var out []string
	for _, m := range l.catalog.Search(query) {
		out = append(out, m.ID)
	}
	return out
}

// HasCatalog reports whether l keeps a live listing's catalog, so a search
// can read its facts (0036-MADR D6). A listing a test builds with no
// catalog does not.
func HasCatalog(l Listing) bool { return l.held }

// List lists the built-in provider id's models with apiKey, through the
// SDK's catalog.List with the capable profile, o.ModelData's document,
// if any, and no limit. It sends no generation and fetches no document,
// and is bounded by the SDK at 10 s. It
// passes no logger, so the SDK logs nothing of it. A listing that fails or
// recommends nothing gives the static models, Live false and the reason in
// Err; only an unknown id or a refused option is an error. o's APIKey,
// Model and Logger are ignored. With no apiKey, a provider with an OAuth
// login lists with o.Logins' stored login (0005-PLAN F4-3b).
func List(ctx context.Context, id, apiKey string, o Options) (Listing, error) {
	if _, ok := o.Set.chatBase(id); ok {
		return listChat(ctx, id, apiKey, o)
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
	return listWith(ctx, id, src, o)
}

// listWith is List with the token source it signs with: o's key and logins
// are not consulted.
func listWith(ctx context.Context, id string, src llmprovider.TokenSource, o Options) (Listing, error) {
	opts := []llmprovider.Option{
		catalog.WithProfile(catalog.ProfileCapable),
		llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),
	}
	opts = append(opts, o.sdkOptions()...)
	opts = append(opts, o.ModelData.options()...)
	if len(o.LastGood) > 0 {
		opts = append(opts, catalog.WithLastGood(o.LastGood, o.LastGoodAt))
	}
	if o.Workspace != "" {
		opts = append(opts, llmprovider.For(llmprovider.ProviderClaude, catalog.WithClaudeWorkspace(o.Workspace)))
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
	return Listing{Models: cat.Recommended, Live: true, Source: string(cat.Source), FetchedAt: cat.FetchedAt, Usable: cat.Usable, catalog: cat, held: true}, nil
}

// listChat lists a Chat Completions server through catalog.List, with its id
// and base URL. WithProfile and WithLimit are not sent: the SDK refuses both
// for an id that is not built in (0036-PLAN P4).
func listChat(ctx context.Context, id, apiKey string, o Options) (Listing, error) {
	opts := []llmprovider.Option{llmprovider.WithClientInfo(o.ClientName, o.ClientVersion)}
	opts = append(opts, o.sdkOptions()...)
	opts = append(opts, o.chatOptions(id)...)
	opts = append(opts, o.ModelData.options()...)
	cat, err := catalog.List(ctx, llmprovider.ProviderID(id), llmprovider.NewStaticToken(apiKey), opts...)
	if err != nil {
		return Listing{}, fmt.Errorf("provider %s: list models: %w", id, mapError(err))
	}
	if !cat.Live || len(cat.Recommended) == 0 {
		reason := mapError(cat.Err)
		if reason == nil {
			reason = errNothingListed
		}
		return Listing{Err: reason}, nil
	}
	return Listing{Models: cat.Recommended, Live: true, Source: string(cat.Source), FetchedAt: cat.FetchedAt, Usable: cat.Usable, catalog: cat, held: true}, nil
}

// ErrorKind names err's class in the words of the SDK's auth contract
// (0051-MADR D6): the SDK's signin.ErrorKindOf for an error of the SDK's
// sign-ins, else llm.Kind's. A sign-in's error is a *SignInError, which
// SignIn returns for every failure of the SDK's, or an error that holds the
// SDK's denied or expired marker (auth.ErrSignInDenied, ErrSignInExpired).
// Any other error, a provider's reply included, is llm.Kind's, so a quota
// the service sends with HTTP 401 is still "quota" (0049-MADR D8). It never
// carries err's text, which a provider may fill with part of the credential
// (0020-MADR F5). A nil error is "other".
func ErrorKind(err error) string {
	if isSignInError(err) {
		if k := signin.ErrorKindOf(err); k != "" {
			return string(k)
		}
	}
	return llm.Kind(err)
}

// isSignInError reports whether err is an error of the SDK's sign-ins.
func isSignInError(err error) bool {
	var se *SignInError
	return errors.As(err, &se) || errors.Is(err, auth.ErrSignInDenied) || errors.Is(err, auth.ErrSignInExpired)
}
