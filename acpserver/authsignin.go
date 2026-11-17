package acpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The names of the fields of the forms authenticate sends.
const (
	fieldCallback = "callback"
	fieldAllow    = "allow"
	fieldMethod   = "method"
)

// signInMeta is _meta.gobble of an elicitation of a sign-in: the step it
// relays, as the contract states it, so a client that reads the contract
// renders the step itself and learns from its secret member that the answer is
// one (0051-MADR D4).
type signInMeta map[string]authcontract.Step

// elicitScope is the scope of an elicitation sent during authenticate: the
// authenticate request's own id, since no session exists yet (0051-MADR D4,
// U8).
func elicitScope(ctx context.Context) (acp.ElicitationRequestScope, error) {
	raw, ok := acp.RequestIDFromContext(ctx)
	if !ok {
		return acp.ElicitationRequestScope{}, errors.New("authenticate: no request id")
	}
	var id acp.RequestId
	if err := jsonx.Unmarshal(raw, &id); err != nil {
		return acp.ElicitationRequestScope{}, err
	}
	return acp.ElicitationRequestScope{RequestId: id}, nil
}

// signInConn is the agent's connection, or an internal error when it has
// none.
func (a *Agent) signInConn() (*acp.AgentSideConnection, error) {
	conn := a.connection()
	if conn == nil {
		return nil, acp.NewInternalError(errData(acpext.ErrorData{Reason: "no connection to ask the client on"}))
	}
	return conn, nil
}

// askMethod asks the client which of cands to sign in by: a form whose one
// field is an enum of their labels (0051-MADR D4 step 3). A decline or cancel
// ends authenticate as a cancelled sign-in.
func (a *Agent) askMethod(ctx context.Context, ap acpext.AuthProvider, cands []authcontract.Method) (authcontract.Method, error) {
	conn, err := a.signInConn()
	if err != nil {
		return authcontract.Method{}, err
	}
	scope, err := elicitScope(ctx)
	if err != nil {
		return authcontract.Method{}, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	labels := make([]string, len(cands))
	for i, m := range cands {
		labels[i] = m.Label
	}
	prop := acp.NewElicitationPropertySchemaString()
	prop.String.Enum = labels
	prop.String.Title = new("Sign-in method")
	prop.String.Default = &labels[0]
	schema := acp.ElicitationSchema{Properties: map[string]acp.ElicitationPropertySchema{fieldMethod: prop}, Required: []string{fieldMethod}}
	req := acp.NewCreateElicitationRequestForm("How do you want to sign in to "+ap.Name+"?", schema)
	req.Form.Request = &scope
	resp, err := conn.CreateElicitation(ctx, req)
	if err != nil {
		if ctx.Err() != nil {
			return authcontract.Method{}, ctx.Err()
		}
		return authcontract.Method{}, a.elicitFailed(ap, err)
	}
	if resp.Accept != nil {
		if v, ok := resp.Accept.Content[fieldMethod]; ok && v.String != nil {
			if i := slices.Index(labels, string(*v.String)); i >= 0 {
				return cands[i], nil
			}
		}
	}
	return authcontract.Method{}, failure(ap, canceled(ap))
}

// elicitFailed is authenticate's answer to an elicitation the client refused
// or that failed on the wire: auth_required of kind unsupported when the SDK
// found the client lacks the capability, else other. The text is fixed: an
// SDK error about the client's reply could name what it held.
func (a *Agent) elicitFailed(ap acpext.AuthProvider, err error) error {
	kind := authcontract.FailureOther
	if ce, ok := errors.AsType[*acp.CapabilityError](err); ok && ce != nil {
		kind = authcontract.FailureUnsupported
	}
	a.logger.Warn("auth: the client's reply to a sign-in elicitation was refused", slog.String("provider", ap.ID), slog.String("kind", string(kind)))
	return acp.NewAuthRequired(errData(acpext.ErrorData{
		Provider: ap.ID, Kind: acpext.ErrorKind(kind), Command: ap.Command,
		Reason: "the client's reply to the sign-in request was refused",
	}))
}

// failure is auth_required for a sign-in that ended with err: Kind the
// contract's word for it, Retryable what the SDK says of it, Reason the
// error's own text, which holds nothing of the service's (0051-MADR D4, D6).
func failure(ap acpext.AuthProvider, err error) error {
	retry := provider.Retryable(err)
	return acp.NewAuthRequired(errData(acpext.ErrorData{
		Provider: ap.ID, Kind: acpext.ErrorKind(provider.ErrorKind(err)), Reason: err.Error(), Command: ap.Command, Retryable: &retry,
	}))
}

// reserve registers a sign-in to provider ap, one per provider at a time
// within this agent process, before the client is asked which method to sign in
// by; a second request while one runs is -32600 naming the running one. The
// release func ends the reservation, on every path (0051-MADR D4). It holds
// a.mu only to register or remove the entry, never across a request to the
// client.
func (a *Agent) reserve(ap acpext.AuthProvider) (release func(), err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if running, busy := a.signing[ap.ID]; busy {
		return nil, acp.NewInvalidRequest(errData(acpext.ErrorData{
			Provider: ap.ID, Reason: "a sign-in to " + ap.ID + " is already running (" + running + ")",
		}))
	}
	if a.signing == nil {
		a.signing = map[string]string{}
	}
	a.signing[ap.ID] = "asking which method"
	return func() {
		a.mu.Lock()
		delete(a.signing, ap.ID)
		a.mu.Unlock()
	}, nil
}

// reserved names method m as the one the reserved sign-in to ap runs.
func (a *Agent) reserved(ap acpext.AuthProvider, m authcontract.Method) {
	a.mu.Lock()
	a.signing[ap.ID] = m.ID
	a.mu.Unlock()
}

// signIn signs in to provider ap by method m for the client that sent
// authenticate, under the reservation authenticate holds (0051-MADR D4). It
// answers nil once the credential is saved and the run's credentials are told.
func (a *Agent) signIn(ctx context.Context, ap acpext.AuthProvider, m authcontract.Method, remote bool, caps acp.ElicitationCapabilities) error {
	conn, err := a.signInConn()
	if err != nil {
		return err
	}
	scope, err := elicitScope(ctx)
	if err != nil {
		return acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	run, err := a.auth.SignIn(ctx, ap.ID, m.ID, remote)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return failure(ap, err)
	}
	r := &relay{a: a, ap: ap, run: run, ctx: ctx, conn: conn, scope: scope, url: caps.Url != nil}
	err = r.serve(ctx)
	if err == nil && a.auth.Signed != nil {
		a.auth.Signed(ap.ID)
	}
	return err
}

// relay is one sign-in's relay of its steps to the client as elicitations.
type relay struct {
	a     *Agent
	ap    acpext.AuthProvider
	run   SignInRun
	ctx   context.Context
	conn  *acp.AgentSideConnection
	scope acp.ElicitationRequestScope
	url   bool // the client offers elicitation.url

	pastes int
	urlIDs []acp.ElicitationId // the URL-mode elicitations sent
}

// elicitation is one elicitation in flight for step: its request runs in a
// goroutine that sends its outcome once, and cancel ends the request, which
// sends $/cancel_request.
type elicitation struct {
	step   authcontract.Step
	cancel context.CancelFunc
	res    chan elicited
}

// elicited is a client's reply to an elicitation, or the error that ended it.
type elicited struct {
	resp acp.CreateElicitationResponse
	err  error
}

// ended is how the sign-in finished.
type ended struct {
	res provider.SignInResult
	err error
}

// relay reads the run's steps and sends each that shows or asks as an
// elicitation, one at a time, in order, until the sign-in ends. No lock is
// held while the client is awaited, and every goroutine and request it started
// has ended when it returns (0051-MADR D4).
func (r *relay) serve(ctx context.Context) error {
	steps := r.run.Steps()
	done := make(chan ended, 1)
	go func() {
		res, err := r.run.Wait(ctx)
		done <- ended{res, err}
	}()
	var (
		queue []authcontract.Step
		cur   *elicitation
		curC  <-chan elicited
	)
	next := func() {
		for cur == nil && len(queue) > 0 {
			s := queue[0]
			queue = queue[1:]
			if cur = r.start(ctx, s); cur != nil {
				curC = cur.res
			}
		}
	}
	stop := func() { // ends the open elicitation, and waits for its goroutine
		if cur != nil {
			cur.cancel()
			<-cur.res
			cur, curC = nil, nil
		}
	}
	defer r.run.Cancel()
	for {
		select {
		case s, ok := <-steps:
			if !ok {
				steps = nil
				continue
			}
			switch s.Kind {
			case authcontract.StepOpenURL, authcontract.StepShowCode, authcontract.StepPaste, authcontract.StepConfirm, authcontract.StepForm:
				queue = append(queue, s)
				next()
			}
			// waiting shows nothing; done and failed end the sign-in through
			// Wait; a kind this gobble does not know is not shown.
		case res := <-curC:
			e := cur
			cur, curC = nil, nil
			e.cancel()
			if err := r.reply(e.step, res); err != nil {
				stop()
				r.run.Cancel()
				end := <-done
				r.completeURLs()
				if end.err == nil {
					return nil // the sign-in finished first, and saved, whatever the context says now
				}
				if r.ctx.Err() != nil {
					return r.ctx.Err()
				}
				if !errors.Is(end.err, context.Canceled) {
					return failure(r.ap, end.err)
				}
				return err
			}
			next()
		case end := <-done:
			stop()
			r.completeURLs()
			if end.err != nil {
				if r.ctx.Err() != nil {
					return r.ctx.Err()
				}
				return failure(r.ap, end.err)
			}
			return nil
		case <-r.ctx.Done():
			stop()
			r.run.Cancel()
			if end := <-done; end.err == nil {
				// The credential was saved before the cancel arrived: it is
				// stored, so the agent is told, not left holding the old one
				// (0051-MADR D4, "After a sign-in").
				return nil
			}
			return r.ctx.Err()
		}
	}
}

// completeURLs sends elicitation/complete for each URL-mode elicitation
// sent, now that the sign-in ended without an answer to it (0051-MADR D4).
// A connection that is already gone is not told.
func (r *relay) completeURLs() {
	if r.ctx.Err() != nil {
		return
	}
	for _, id := range r.urlIDs {
		if err := r.conn.CompleteElicitation(r.ctx, acp.CompleteElicitationNotification{ElicitationId: id}); err != nil {
			r.a.logger.Warn("auth: elicitation/complete was not sent", slog.String("provider", r.ap.ID), slog.String("error", err.Error()))
		}
	}
	r.urlIDs = nil
}

// asks reports whether a step of kind k takes an answer.
func asks(k authcontract.StepKind) bool {
	return k == authcontract.StepPaste || k == authcontract.StepConfirm || k == authcontract.StepForm
}

// reply acts on the client's reply to step s: an accept of a step that only
// shows something does nothing, and an accept of one that asks answers it; a
// decline or cancel, or a reply that is not an accept, ends the sign-in as
// cancelled. A non-nil error is the answer authenticate gives instead of
// waiting for the sign-in to end.
func (r *relay) reply(s authcontract.Step, res elicited) error {
	if res.err != nil {
		if r.ctx.Err() != nil {
			return r.ctx.Err()
		}
		return r.a.elicitFailed(r.ap, res.err)
	}
	if res.resp.Accept == nil {
		return failure(r.ap, canceled(r.ap))
	}
	if !asks(s.Kind) {
		return nil // a shown step is acknowledged, never answered
	}
	ans, ok := answerOf(s, res.resp.Accept.Content)
	if !ok {
		return failure(r.ap, canceled(r.ap))
	}
	if err := r.run.Answer(r.ctx, ans); err != nil {
		// The sign-in ended while the answer was on its way; Wait says how.
		r.a.logger.Debug("auth: an answer was not taken", slog.String("provider", r.ap.ID), slog.Int("seq", s.Seq))
	}
	return nil
}

// answerOf is the contract's answer to step s from an accepted form's
// content, with Secret taken from the step the relay sent for that seq, so
// the contract's own redaction masks it (0051-MADR D4, "Secrets").
func answerOf(s authcontract.Step, content map[string]acp.ElicitationContentValue) (authcontract.Answer, bool) {
	secret := s.Secret
	a := authcontract.Answer{SchemaVersion: authcontract.SchemaVersion, Seq: s.Seq, Secret: &secret}
	switch s.Kind {
	case authcontract.StepPaste:
		v, ok := content[fieldCallback]
		if !ok || v.String == nil {
			return a, false
		}
		a.Paste = string(*v.String)
	case authcontract.StepConfirm:
		v, ok := content[fieldAllow]
		if !ok || v.Boolean == nil {
			return a, false
		}
		allow := bool(*v.Boolean)
		a.Confirm = &allow
	case authcontract.StepForm:
		a.Fields = map[string]string{}
		for _, f := range s.Fields {
			if v, ok := content[f.Key]; ok && v.String != nil {
				a.Fields[f.Key] = string(*v.String)
			}
		}
	}
	return a, true
}

// start sends the elicitation for step s in a goroutine of its own, so the
// relay goes on reading the run, and returns it; nil when the step cannot be
// built (which ends the sign-in through the run's cancel by the caller's
// next read).
func (r *relay) start(parent context.Context, s authcontract.Step) *elicitation {
	req, err := r.request(s)
	if err != nil {
		r.a.logger.WarnContext(parent, "auth: a step was not sent", slog.String("provider", r.ap.ID), slog.String("kind", string(s.Kind)), slog.String("error", err.Error()))
		r.run.Cancel()
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	e := &elicitation{step: s, cancel: cancel, res: make(chan elicited, 1)}
	go func() {
		resp, err := r.conn.CreateElicitation(ctx, req)
		e.res <- elicited{resp, err}
	}()
	return e
}

// request is the elicitation of step s by MADR D4's table: URL mode for a
// shown step when the client offers elicitation.url, else a form, and the
// form the step's kind asks, every one request-scoped and carrying the step
// in _meta.gobble.step.
func (r *relay) request(s authcontract.Step) (acp.CreateElicitationRequest, error) {
	name := r.ap.Name
	var req acp.CreateElicitationRequest
	switch s.Kind {
	case authcontract.StepOpenURL:
		msg := "Sign in to " + name + ": open the address below in a browser on the machine gobble runs on, and sign in; the sign-in finishes by itself."
		if s.Accepts != "" {
			msg = "Sign in to " + name + ": open the address below in a browser and sign in. Accept here when you have, and you will be asked to paste what the browser ends on."
		}
		req = r.shown(s, msg, s.URL)
	case authcontract.StepShowCode:
		msg := "Sign in to " + name + ": open the address below and enter the code " + s.Code + "."
		if !s.ExpiresAt.IsZero() {
			msg += " It expires at " + s.ExpiresAt.UTC().Format("15:04 MST") + "."
		}
		req = r.shown(s, msg, s.URL)
	case authcontract.StepPaste:
		r.pastes++
		text := s.Text
		if text == "" {
			text = "Paste what the browser ended on"
		}
		prop := acp.NewElicitationPropertySchemaString()
		prop.String.Title = &text
		schema := acp.ElicitationSchema{Properties: map[string]acp.ElicitationPropertySchema{fieldCallback: prop}, Required: []string{fieldCallback}}
		msg := text
		if r.pastes > 1 {
			msg = "That did not complete the sign-in. " + text
		}
		req = acp.NewCreateElicitationRequestForm(msg, schema)
	case authcontract.StepConfirm:
		prop := acp.NewElicitationPropertySchemaBoolean()
		prop.Boolean.Default = new(false) // consent must be express
		prop.Boolean.Title = new("Allow")
		schema := acp.ElicitationSchema{Properties: map[string]acp.ElicitationPropertySchema{fieldAllow: prop}, Required: []string{fieldAllow}}
		var msg strings.Builder
		msg.WriteString(s.Text)
		for _, l := range s.Links {
			msg.WriteString("\n" + l.URL)
		}
		req = acp.NewCreateElicitationRequestForm(msg.String(), schema)
	case authcontract.StepForm:
		props := map[string]acp.ElicitationPropertySchema{}
		var required []string
		for _, f := range s.Fields {
			p := acp.NewElicitationPropertySchemaString()
			p.String.Title = &f.Label
			if f.Type == authcontract.FieldSelect {
				p.String.Enum = f.Options
			}
			if f.Default != "" {
				p.String.Default = &f.Default
			}
			props[f.Key] = p
			if f.Required {
				required = append(required, f.Key)
			}
		}
		req = acp.NewCreateElicitationRequestForm(s.Text, acp.ElicitationSchema{Properties: props, Required: required})
	default:
		return req, fmt.Errorf("a %q step is not sent", s.Kind)
	}
	meta, err := stepMeta(s)
	if err != nil {
		return req, err
	}
	switch {
	case req.Form != nil:
		req.Form.Request, req.Form.Meta = &r.scope, meta
	case req.Url != nil:
		req.Url.Request, req.Url.Meta = &r.scope, meta
	}
	return req, nil
}

// shown is the request that asks the client to show step s: URL mode to u for
// a client that offers elicitation.url, else a form with no fields whose
// message holds u.
func (r *relay) shown(s authcontract.Step, msg, u string) acp.CreateElicitationRequest {
	if r.url {
		id := acp.ElicitationId("gobble-signin-" + r.ap.ID + "-" + strconv.Itoa(s.Seq))
		r.urlIDs = append(r.urlIDs, id)
		return acp.NewCreateElicitationRequestUrl(msg, id, u)
	}
	return acp.NewCreateElicitationRequestForm(msg+"\n"+u, acp.ElicitationSchema{Properties: map[string]acp.ElicitationPropertySchema{}})
}

// stepMeta is _meta of an elicitation: gobble's member holding step s.
func stepMeta(s authcontract.Step) (map[string]any, error) {
	return jsonx.SetMember(nil, gobble, signInMeta{acpext.MetaStep: s})
}

// canceled is the error of a sign-in the client ended, as the SDK's sign-in
// states one: provider.SignInRun.Wait's, kind canceled.
func canceled(ap acpext.AuthProvider) error { return provider.NewSignInError(ap.ID, context.Canceled) }
