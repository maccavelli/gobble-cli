package provider

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// Login is a stored OAuth login: a subscription session's tokens, when its
// access token expires, and the SDK session's own fields (0005-PLAN F4-3b).
// Format a *Login: its String and LogValue never show a token.
type Login struct {
	Access, Refresh                       string
	Expires                               time.Time
	Issuer, ClientID, AccountID, TokenURL string
	FedRAMP                               bool
}

// String renders the login without its tokens.
func (l *Login) String() string {
	if l == nil {
		return "Login(nil)"
	}
	return fmt.Sprintf("Login{Issuer:%s ClientID:%s AccountID:%s TokenURL:%s FedRAMP:%t Expires:%s}",
		l.Issuer, l.ClientID, l.AccountID, l.TokenURL, l.FedRAMP, l.Expires.Format(time.RFC3339))
}

// LogValue renders the login for slog, without its tokens.
func (l *Login) LogValue() slog.Value { return slog.StringValue(l.String()) }

// LoginStore keeps OAuth logins, one per provider: Load gives nil and no
// error for a provider with none. LockRefresh serialises one provider's
// refreshes across processes, and must not be the lock Save takes: a
// refresh saves while it holds it (0005-PLAN F4-3b decision 4).
type LoginStore interface {
	Load(ctx context.Context, id string) (*Login, error)
	Save(ctx context.Context, id string, l *Login) error
	Delete(ctx context.Context, id string) error
	LockRefresh(ctx context.Context, id string) (unlock func(), err error)
}

// LoginOptions configure a login. Origin replaces the vendor's issuer, and
// HTTPClient the SDK's client, for tests. OpenURL is given the browser
// login's URL; InputCode, when set, may return the redirect URL the user
// pastes instead of the browser reaching the callback.
type LoginOptions struct {
	HTTPClient *http.Client
	Origin     string
	OpenURL    func(string) error
	InputCode  func(context.Context) (string, error)
}

func (o LoginOptions) flow() auth.OAuthFlowOptions {
	return auth.OAuthFlowOptions{HTTPClient: o.HTTPClient, Issuer: o.Origin, OpenURL: o.OpenURL, InputCode: o.InputCode}
}

// DeviceCode is what a device login shows: where to go, the code to enter
// there, and when the code expires.
type DeviceCode struct {
	URL, Code string
	Expires   time.Time
}

// HasLogin reports whether provider id has an OAuth login gobble keeps and
// refreshes: Grok's, through the SDK's flows (0005-PLAN F4-3b decision 1).
func HasLogin(id string) bool { return id == string(llmprovider.ProviderGrok) }

// HasKeyLogin reports whether provider id has a device login that gives an
// API key: Kilo's (0005-PLAN F4-3b).
func HasKeyLogin(id string) bool { return id == string(llmprovider.ProviderKilo) }

// PiLogin is Pi's stored login for provider id as gobble's. Pi keeps no
// issuer or client id, so Grok's takes the SDK's, which are the ones Pi
// signs in with; a refresh sends them (0005-PLAN F4-3c decision 7). Any
// other id is false: gobble imports no other login.
func PiLogin(id, access, refresh string, expires time.Time) (*Login, bool) {
	if !HasLogin(id) {
		return nil, false
	}
	return &Login{
		Access: access, Refresh: refresh, Expires: expires,
		Issuer: auth.DefaultGrokOAuthIssuer, ClientID: auth.DefaultGrokOAuthClientID,
	}, true
}

// LoginRejected is what gobble says when provider id refuses its stored
// login, or the login cannot be refreshed: nothing of the provider's text
// (0005-PLAN F4-3b decision 8).
func LoginRejected(id string) string {
	return id + " rejected the OAuth login in auth.json; sign in again with gobble auth login " + id
}

// BrowserLogin signs in to provider id in the browser and saves the login
// in store.
func BrowserLogin(ctx context.Context, id string, o LoginOptions, store LoginStore) error {
	if !HasLogin(id) {
		return fmt.Errorf("provider %s has no OAuth login", id)
	}
	s, err := auth.LoginBrowserOAuth(ctx, llmprovider.ProviderID(id), o.flow())
	if err != nil {
		return fmt.Errorf("%s login: %w", id, mapError(err))
	}
	return store.Save(ctx, id, loginOf(s))
}

// DeviceLogin signs in to provider id with a device code, given to show,
// and saves the login in store. An error from show cancels the login.
func DeviceLogin(ctx context.Context, id string, o LoginOptions, show func(DeviceCode) error, store LoginStore) error {
	if !HasLogin(id) {
		return fmt.Errorf("provider %s has no OAuth login", id)
	}
	s, err := device(ctx, id, o, show)
	if err != nil {
		return fmt.Errorf("%s login: %w", id, mapError(err))
	}
	return store.Save(ctx, id, loginOf(s))
}

// KeyLogin signs in to provider id with a device code, given to show, and
// returns the API key the login gives. An error from show cancels it.
func KeyLogin(ctx context.Context, id string, o LoginOptions, show func(DeviceCode) error) (string, error) {
	if !HasKeyLogin(id) {
		return "", fmt.Errorf("provider %s has no device login that gives a key", id)
	}
	s, err := device(ctx, id, o, show)
	if err != nil {
		return "", fmt.Errorf("%s login: %w", id, mapError(err))
	}
	return s.Access, nil
}

func device(ctx context.Context, id string, o LoginOptions, show func(DeviceCode) error) (*auth.OAuthSession, error) {
	d, err := auth.StartDeviceOAuth(ctx, llmprovider.ProviderID(id), o.flow())
	if err != nil {
		return nil, err
	}
	if err := show(DeviceCode{URL: d.VerificationURI, Code: d.UserCode, Expires: d.Expiry}); err != nil {
		d.Cancel()
		return nil, err
	}
	return d.Wait(ctx)
}

// Logout revokes provider id's stored login, then deletes it from store,
// whether or not revocation succeeded: the SDK leaves the stored copy to its
// caller (0005-PLAN F4-3b decision 7). revoke is revocation's error; err is
// the store's.
func Logout(ctx context.Context, id string, o LoginOptions, store LoginStore) (revoke, err error) {
	l, err := store.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	if l != nil && HasLogin(id) {
		if rerr := auth.RevokeOAuthSession(ctx, l.session(id, nil, o.HTTPClient, nil)); rerr != nil {
			revoke = mapError(rerr)
		}
	}
	return revoke, store.Delete(ctx, id)
}

// session is l as the SDK's session for provider id, saving each rotation
// to store when it is not nil.
func (l *Login) session(id string, store LoginStore, client *http.Client, logger *slog.Logger) *auth.OAuthSession {
	s := &auth.OAuthSession{
		Provider: llmprovider.ProviderID(id), Access: l.Access, Refresh: l.Refresh, Expiry: l.Expires,
		Issuer: l.Issuer, ClientID: l.ClientID, AccountID: l.AccountID, FedRAMP: l.FedRAMP, TokenURL: l.TokenURL,
		HTTPClient: client, Logger: logger,
	}
	if store != nil {
		s.Store = sdkStore{store}
	}
	return s
}

// loginOf is the SDK session s as a Login.
func loginOf(s *auth.OAuthSession) *Login {
	return &Login{
		Access: s.Access, Refresh: s.Refresh, Expires: s.Expiry,
		Issuer: s.Issuer, ClientID: s.ClientID, AccountID: s.AccountID, TokenURL: s.TokenURL, FedRAMP: s.FedRAMP,
	}
}

// sdkStore is a LoginStore as the SDK's TokenStore and RefreshLocker
// (0005-PLAN F4-3b decision 4).
type sdkStore struct{ s LoginStore }

func (a sdkStore) Load(ctx context.Context, p llmprovider.ProviderID) (*auth.OAuthSession, error) {
	l, err := a.s.Load(ctx, string(p))
	if err != nil || l == nil {
		return nil, err
	}
	return l.session(string(p), a.s, nil, nil), nil
}

func (a sdkStore) Save(ctx context.Context, p llmprovider.ProviderID, s *auth.OAuthSession) error {
	return a.s.Save(ctx, string(p), loginOf(s))
}

func (a sdkStore) Delete(ctx context.Context, p llmprovider.ProviderID) error {
	return a.s.Delete(ctx, string(p))
}

func (a sdkStore) LockRefresh(ctx context.Context, p llmprovider.ProviderID) (func(), error) {
	return a.s.LockRefresh(ctx, string(p))
}

var (
	_ auth.TokenStore    = sdkStore{}
	_ auth.RefreshLocker = sdkStore{}
)
