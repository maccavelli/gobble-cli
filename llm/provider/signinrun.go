package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
)

// startSignIn is the SDK's sign-in engine; a test replaces it to record the
// options it was given (0051-PLAN P6 step 1).
var startSignIn = signin.Start

// SignInResult is what a finished StartSignIn sign-in left: the key for the
// caller to store, with Yields device_key, or no key, with Yields
// oauth_session, the session saved through the store (0051-MADR D3).
type SignInResult struct {
	Key    string
	Yields authcontract.Yields
}

// SignInRun is a sign-in in progress, the SDK's x/signin handle as gobble
// drives it: Steps and Answer carry the contract's steps and answers, Wait
// stores the result, Cancel ends it (0051-MADR D3). Its methods are safe for
// concurrent use.
type SignInRun struct {
	id     string
	run    *signin.SignIn
	method authcontract.Method
	store  LoginStore
	o      SignInOptions

	once sync.Once
	res  SignInResult
	err  error
}

// NewSignInError is err as a sign-in of provider id's failure: the
// *SignInError that SignIn and SignInRun.Wait return, so that ErrorKind
// reads the SDK's word for it (0051-MADR D6).
func NewSignInError(id string, err error) error {
	return &SignInError{ID: id, err: mapError(err)}
}

// MethodCommand is the host command that signs in to provider id with the
// method whose contract id is methodID: gobble auth login <id> --method
// <word>, or the bare gobble auth login <id> for a method --method has no
// word for (0051-MADR D4 step 1).
func MethodCommand(id, methodID string) string {
	cmd := "gobble auth login " + id
	if w := wordOf(methodID); w != "" {
		cmd += " --method " + w
	}
	return cmd
}

// SignedInMethod is the offered method that stored login l of provider id was
// signed in by, as its kind names it (0051-MADR D4 step 2).
func SignedInMethod(id string, l *Login) (authcontract.Method, bool) { return storedMethod(id, l) }

// SessionKindOf is the kind of stored login l of provider id, as the SDK's
// (*OAuthSession).SessionKind gives it: l's own Kind when set, returned as
// stored even when it is not a kind the SDK knows, else the kind the SDK
// infers, so a session saved before kind existed still has one; "" for l nil
// or a login the SDK infers no kind for. It is the one place gobble asks the
// SDK for a session's kind: storedMethod, and through it Accepts and the
// resolver, the adapter's planConsentOf and StartSignIn's Previous all read
// it here, and gobble copies no inference rule; auth status --json's
// sessionKind is this value (0052-MADR D8).
func SessionKindOf(id string, l *Login) string {
	if l == nil {
		return ""
	}
	return kindOfSession(l.session(id, nil, nil, nil))
}

// kindOfSession is the SDK's kind of live session s, read under the session's
// own lock, as the SDK's SessionKind does: a run's Sessions share one session
// among its providers while a refresh rewrites its fields. It is the one call
// into the SDK's SessionKind, and what SessionKindOf and the adapter's
// planConsentOf both read.
func kindOfSession(s *auth.OAuthSession) string { return string(s.SessionKind()) }

// StartSignIn begins provider id's sign-in by the method whose contract id
// is method (one of Offered), for a caller that relays its steps, such as an
// ACP client: signin.Start with the options MADR D3 lists. It opens no
// browser on the host and asks nothing itself. Every error is a
// *SignInError, so ErrorKind reads the SDK's word for it, an offline failure
// included (0051-MADR D6). A stored session of the kind the method signs in
// to is Previous, so a re-sign-in with an answer asks no consent again.
func StartSignIn(ctx context.Context, id, method string, o SignInOptions, store LoginStore) (*SignInRun, error) {
	m, ok := offeredByID(id, method)
	if !ok {
		return nil, NewSignInError(id, fmt.Errorf("%w: provider %s offers no sign-in %q", llmprovider.ErrInvalidRequest, id, method))
	}
	so := signin.Options{
		HTTPClient:  o.HTTPClient,
		Identity:    identityOf(o.Options),
		AgentName:   AgentName,
		HostID:      o.HostID,
		ConsentText: ConsentText,
		NoLoopback:  o.Remote,
	}
	ts := tokenStore(store)
	if cs, ok := ts.(auth.ClientStore); ok {
		so.ClientStore = cs
	}
	l, err := store.Load(ctx, id)
	if err != nil {
		return nil, NewSignInError(id, fmt.Errorf("the stored sign-in: %w", err))
	}
	if l != nil {
		if kind := SessionKindOf(id, l); kind != "" && kind == string(sessionKinds[sessionKey{id, m.ClientIdentity}]) {
			so.Previous = l.session(id, nil, o.HTTPClient, nil)
		}
	}
	run, err := startSignIn(ctx, llmprovider.ProviderID(id), llmprovider.AuthMethodID(method), so)
	if err != nil {
		return nil, NewSignInError(id, err)
	}
	return &SignInRun{id: id, run: run, method: run.Method(), store: store, o: o}, nil
}

// offeredByID is provider id's offered method whose contract id is mid.
func offeredByID(id, mid string) (authcontract.Method, bool) {
	for _, m := range Offered(id) {
		if m.ID == mid {
			return m, true
		}
	}
	return authcontract.Method{}, false
}

// Method is the sign-in's method, as the contract describes it.
func (r *SignInRun) Method() authcontract.Method { return r.method }

// Steps are the sign-in's steps, in order, the last done or failed; the
// channel closes after it.
func (r *SignInRun) Steps() <-chan authcontract.Step { return r.run.Steps() }

// Answer gives the answer to the step whose Seq it names.
func (r *SignInRun) Answer(ctx context.Context, a authcontract.Answer) error {
	return r.run.Answer(ctx, a)
}

// Cancel ends the sign-in; Wait then returns an error of kind canceled, unless
// the sign-in had already finished and saved.
func (r *SignInRun) Cancel() { r.run.Cancel() }

// Wait returns the sign-in's result once it ends. The new credential is tried
// against the provider's listing first, the check the wizard runs before it
// saves: a session is listed through itself, so a renewal the listing causes
// rotates the session that is saved, and a key is listed as the listing's key.
// Only a credential the listing does not refuse is saved through the store, in
// one write, or returned as a key for the caller to store. So a sign-in that
// fails, is declined, is cancelled or is refused leaves what was stored exactly
// as it was, and under fresh a working credential is never replaced by a
// refused one (0051-MADR D4 step 3). A refusal is the wizard's: an
// authentication failure that is not a refusal to permit something
// (wizard/configure.go, refusedCredential), whether List returns it or the
// listing carries it, or a renewal the issuer refuses (wizard/auth.go, save);
// any other listing failure, a 403, a 503 or a failed renewal included, does
// not fail the sign-in, as in the wizard. A context that is done before the
// save fails the sign-in with it, as the wizard does not save then. Wait does
// its work once; later calls return the same result.
func (r *SignInRun) Wait(ctx context.Context) (SignInResult, error) {
	r.once.Do(func() { r.res, r.err = r.finish(ctx) })
	return r.res, r.err
}

func (r *SignInRun) finish(ctx context.Context) (SignInResult, error) {
	cred, err := r.run.Wait(ctx)
	if err != nil {
		return SignInResult{}, NewSignInError(r.id, err)
	}
	res := SignInResult{Yields: cred.Yields()}
	var key string
	var sess *auth.OAuthSession
	switch c := cred.(type) {
	case signin.Session:
		sess = c.OAuth
	case signin.AccessOnly:
		sess = c.OAuth
	case signin.DeviceKey:
		res.Key, key = c.Key, c.Key
	default:
		return SignInResult{}, NewSignInError(r.id, fmt.Errorf("%w: gobble stores no %s credential", llmprovider.ErrUnsupported, cred.Yields()))
	}
	if err := checkListing(ctx, r.id, key, sess, r.o.Options); err != nil {
		return SignInResult{}, NewSignInError(r.id, err)
	}
	if sess != nil {
		// As the wizard's save does: wait for a refresh the listing left
		// running, so that the store gets the rotated tokens, and save nothing
		// the issuer refuses or a context that is done.
		if _, err := sess.Token(ctx); err != nil && (errors.Is(err, llmprovider.ErrAuthFailure) || ctx.Err() != nil) {
			return SignInResult{}, NewSignInError(r.id, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return SignInResult{}, NewSignInError(r.id, err)
	}
	if sess != nil {
		if err := tokenStore(r.store).Save(ctx, llmprovider.ProviderID(r.id), sess); err != nil {
			return SignInResult{}, NewSignInError(r.id, err)
		}
	}
	return res, nil
}

// refusedCredential is the wizard's predicate for a listing's failure that is
// the service refusing the credential: an authentication failure that is not
// a refusal to permit something (wizard/configure.go at go-llmprovider-sdk
// v1.32.0). Its other refusal, of a workspace, is Anthropic's and a prompt's,
// and cannot arise here: no Claude sign-in is one gobble drives.
func refusedCredential(err error) bool {
	return errors.Is(err, llmprovider.ErrAuthFailure) && !errors.Is(err, llmprovider.ErrNotPermitted)
}

// checkListing is the check the wizard's Discover makes after a sign-in: it
// lists provider id's models with sess, the new session itself, or with key, and
// fails only when the provider refused the credential, in List's error or in the
// listing's (refusedCredential). An outage, a refusal to permit and an empty
// listing do not fail a sign-in, and a session of a provider whose listing
// needs no key cannot be refused by it. Kilo's listing is public, but the SDK
// sends a key it is given, so the device key is listed with. Kilo's listing
// answers every key, so a Kilo key is first checked at Kilo's profile with the
// run's own options and no organization, as the wizard's Discover checks it
// (signin.VerifyKilo, go-llmprovider-sdk v1.32.0, 0080-MADR D10; 0068-MADR D3
// a): its refusal is returned, a profile that cannot say passes, and the
// caller's ended context is returned. Options the check rejects fail the
// sign-in, since passing them would skip the check unseen (L9).
func checkListing(ctx context.Context, id, key string, sess *auth.OAuthSession, o Options) error {
	if id == string(llmprovider.ProviderKilo) && key != "" {
		if err := signin.VerifyKilo(ctx, key, "", o.sdkOptions()...); err != nil {
			return err // a refusal, ctx's end, or options it rejects (ErrInvalidRequest)
		}
	}
	if PublicListing(id) && key == "" {
		return nil
	}
	var l Listing
	var err error
	if sess != nil {
		l, err = listWith(ctx, id, sess, o)
	} else {
		l, err = List(ctx, id, key, o)
	}
	if refusedCredential(err) {
		return err
	}
	if refusedCredential(l.Err) {
		return l.Err
	}
	return nil
}
