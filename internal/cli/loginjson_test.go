package cli

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// jsonLine is one line of gobble auth login --json as the contract's types
// decode it, with no member unknown to them: the line's own members, and
// those of the method or the step.
type jsonLine struct {
	Event    string               `json:"event"`
	Provider string               `json:"provider"`
	Method   *authcontract.Method `json:"method"`
	Step     *authcontract.Step   `json:"step"`
}

// jsonLines decodes stdout, one JSON object a line, strictly. A line that is
// neither a method line nor a step line, such as one of the five events
// 0051-MADR D5 retires, or one that holds a member the contract does not
// have, is an error.
func jsonLines(t *testing.T, stdout string) []jsonLine {
	t.Helper()
	if stdout == "" || !strings.HasSuffix(stdout, "\n") {
		t.Fatalf("stdout %q is not whole lines", stdout)
	}
	var out []jsonLine
	for raw := range strings.SplitSeq(strings.TrimSuffix(stdout, "\n"), "\n") {
		var l jsonLine
		if err := json.Unmarshal([]byte(raw), &l, json.RejectUnknownMembers(true)); err != nil {
			t.Errorf("the line %s does not decode into the contract's types: %v", raw, err)
			continue
		}
		if l.Event != "method" && l.Event != "step" || l.Event == "method" && l.Method == nil || l.Event == "step" && l.Step == nil {
			t.Errorf("the line %s is event %q, want a method line or a step line", raw, l.Event)
			continue
		}
		out = append(out, l)
	}
	return out
}

// wantSteps checks lines are a method line for method, then a step line of
// each kind in kinds, with seq rising from one, every line naming provider id,
// and every step this schema's; it returns the steps.
func wantSteps(t *testing.T, lines []jsonLine, id, method string, kinds ...authcontract.StepKind) []authcontract.Step {
	t.Helper()
	if len(lines) != 1+len(kinds) {
		t.Fatalf("%d lines, want a method line and %d steps: %+v", len(lines), len(kinds), lines)
	}
	for _, l := range lines {
		if l.Provider != id {
			t.Errorf("a line names provider %q, want %q", l.Provider, id)
		}
	}
	if m := lines[0]; m.Event != "method" || m.Method.ID != method || m.Method.SchemaVersion != authcontract.SchemaVersion {
		t.Errorf("the first line %+v, want the method line for %s", m, method)
	}
	var steps []authcontract.Step
	for i, k := range kinds {
		l := lines[1+i]
		if l.Event != "step" || l.Step.Kind != k || l.Step.Seq != i+1 || l.Step.SchemaVersion != authcontract.SchemaVersion {
			t.Errorf("line %d is %+v, want a step of kind %s with seq %d", 1+i, l, k, i+1)
			continue
		}
		steps = append(steps, *l.Step)
	}
	return steps
}

// issuerEnv is a config directory, an issuer and the real wizard run against
// it, with the calls to the wizard recorded as "provider method".
func issuerEnv(t *testing.T) (*cfgEnv, *loginIssuer, *[]string) {
	t.Helper()
	c := newCfgEnv(t)
	ti := newLoginIssuer(t)
	calls := new([]string)
	replace(t, &signInWith, signInAtIssuer(t, ti, calls))
	return c, ti, calls
}

// gobble auth login grok --device --json prints a method line, then the
// contract's steps for the SDK's own device sign-in against the issuer:
// show_code with the URL, the code and its expiry, waiting, and the done
// step gobble states once the session is stored. Each line decodes strictly
// into the contract's types, none is one of the five retired events, stderr is
// empty, and the session is in auth.json (0051-MADR D5).
func TestAuthLoginJSONSteps(t *testing.T) {
	c, ti, calls := issuerEnv(t)
	r := c.run("auth", "login", "grok", "--device", "--json")
	if r.code != ExitOK || r.stderr != "" {
		t.Fatalf("exit %d, stderr %q; want 0 and nothing", r.code, r.stderr)
	}
	lines := jsonLines(t, r.stdout)
	steps := wantSteps(t, lines, "grok", "device_code", authcontract.StepShowCode, authcontract.StepWaiting, authcontract.StepDone)
	if len(steps) != 3 {
		return
	}
	if m := lines[0].Method; m.Kind != authcontract.KindDeviceCode || m.Completion != authcontract.CompletionDevice || m.Yields != authcontract.YieldsOAuthSession {
		t.Errorf("the method %+v, want grok's device sign-in as the contract states it", m)
	}
	if s := steps[0]; s.URL != ti.URL+"/device" || s.Code != "ABCD-1234" || s.ExpiresAt.IsZero() {
		t.Errorf("show_code %+v, want the issuer's URL, code and an expiry", s)
	}
	if s := steps[2]; s.Yields != authcontract.YieldsOAuthSession || s.Error != nil {
		t.Errorf("done %+v, want it to yield the session", s)
	}
	if got := c.read(c.authFile()); !strings.Contains(got, `"access": "access-1"`) {
		t.Errorf("auth.json after the sign-in:\n%s", got)
	}
	if !slices.Equal(*calls, []string{"grok device"}) {
		t.Errorf("the wizard ran %q, want grok's device sign-in once", *calls)
	}
}

// Kilo's device sign-in ends with a done step that yields the device key, and
// auth.json holds the key as gobble quotes it, so Pi's grammar gives it back
// unchanged (0051-MADR D5).
func TestAuthLoginJSONKiloKey(t *testing.T) {
	c, ti, _ := issuerEnv(t)
	r := c.run("auth", "login", "kilo", "--json")
	if r.code != ExitOK || r.stderr != "" {
		t.Fatalf("exit %d, stderr %q; want 0 and nothing", r.code, r.stderr)
	}
	steps := wantSteps(t, jsonLines(t, r.stdout), "kilo", "device_code", authcontract.StepShowCode, authcontract.StepWaiting, authcontract.StepDone)
	if len(steps) != 3 {
		return
	}
	if steps[0].URL != ti.URL+"/kilo" || steps[0].Code != "KILO-1" || steps[2].Yields != authcontract.YieldsDeviceKey {
		t.Errorf("kilo's steps %+v, want its code at the issuer and a done that yields device_key", steps)
	}
	f, err := auth.Load(c.authFile())
	if err != nil {
		t.Fatal(err)
	}
	e, _ := f.Entry("kilo")
	back, rerr := catalog.ResolveValue(e.Key, func(string) (string, bool) { return "", false },
		func(string) (string, error) { return "", os.ErrInvalid })
	if e.Type != auth.TypeAPIKey || e.Key != "$!kilo$$token-0123456789" || back != ti.kiloToken || rerr != nil {
		t.Errorf("kilo's entry %+v resolves to %q, %v; want the key stored escaped as $!kilo$$token-0123456789 and read back as %q", e, back, rerr, ti.kiloToken)
	}
}

// --api-key - with --json prints the method line for api_key and a done step
// that yields api_key, exit 0, nothing on stderr; a missing key is a usage
// error, one failed step with no method line (0051-MADR D5).
func TestAuthLoginJSONAPIKey(t *testing.T) {
	c := newCfgEnv(t)
	r := c.runStdin(stdinFile(t, "sk-test-only\n"), "auth", "login", "anthropic", "--api-key", "-", "--json")
	if r.code != ExitOK || r.stderr != "" {
		t.Fatalf("exit %d, stderr %q; want 0 and nothing", r.code, r.stderr)
	}
	lines := jsonLines(t, r.stdout)
	steps := wantSteps(t, lines, "claude", "api_key", authcontract.StepDone)
	if len(steps) == 1 && (steps[0].Yields != authcontract.YieldsAPIKey || lines[0].Method.Kind != authcontract.KindAPIKey) {
		t.Errorf("the method %+v and the done step %+v, want api_key and a done that yields api_key", lines[0].Method, steps[0])
	}
	if got := c.read(c.authFile()); !strings.Contains(got, `"key": "sk-test-only"`) {
		t.Errorf("auth.json:\n%s", got)
	}

	// No key given is a usage error, which is one failed step and no method
	// line (0051-PLAN P5 step 2: "a usage error is one `failed` step").
	r = c.runStdin(stdinFile(t, ""), "auth", "login", "openai", "--api-key", "-", "--json")
	lines = jsonLines(t, r.stdout)
	if r.code != ExitUsage || r.stderr != "" || len(lines) != 1 || lines[0].Step == nil || lines[0].Step.Kind != authcontract.StepFailed ||
		lines[0].Step.Seq != 1 || lines[0].Step.Error.Kind != authcontract.FailureInvalidRequest {
		t.Errorf("no key given: exit %d, stdout %q, stderr %q; want exit 2 and one failed invalid_request step, no method line", r.code, r.stdout, r.stderr)
	}
}

// A usage error, an unknown provider or a provider with no login, is one
// failed step of kind invalid_request with seq 1 and no method line, exit 2,
// nothing on stderr (0051-MADR D5).
func TestAuthLoginJSONUsageError(t *testing.T) {
	c := newCfgEnv(t)
	for _, tc := range []struct {
		args    []string
		message string
	}{
		{[]string{"ollama"}, `unknown provider "ollama": one of `},
		{[]string{"anthropic"}, "claude has no login; use --api-key -"},
		{[]string{"grok", "--method", "chatgpt-plan"}, "grok signs in with --method "},
	} {
		r := c.run(append([]string{"auth", "login", "--json"}, tc.args...)...)
		lines := jsonLines(t, r.stdout)
		if r.code != ExitUsage || r.stderr != "" || len(lines) != 1 {
			t.Errorf("%q: exit %d, stdout %q, stderr %q; want exit 2, one line, nothing on stderr", tc.args, r.code, r.stdout, r.stderr)
			continue
		}
		s := lines[0].Step
		if lines[0].Event != "step" || lines[0].Provider != tc.args[0] && lines[0].Provider != "claude" || s.Kind != authcontract.StepFailed || s.Seq != 1 ||
			s.Error == nil || s.Error.Kind != authcontract.FailureInvalidRequest || s.Error.Retryable || !strings.HasPrefix(s.Error.Message, tc.message) {
			t.Errorf("%q: the line %+v, want one failed step, seq 1, invalid_request, a message that begins %q", tc.args, lines[0], tc.message)
		}
	}
}

// A sign-in that would ask a question is refused before it starts: the first
// ChatGPT plan sign-in asks its consent, so under --json it ends exit 2 with
// the method line and a failed step of kind unsupported that names a terminal
// and an ACP client, and the wizard is never run, no URL is shown, no
// question is asked and nothing is written. With the answer already stored,
// the check passes and the sign-in starts (0051-MADR D5; 0052-MADR D2).
func TestAuthLoginJSONRefusesAQuestion(t *testing.T) {
	c := newCfgEnv(t)
	run := &planRun{}
	started := 0
	replace(t, &signInWith, func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		started++
		// A sign-in that gets as far as its browser ends there, as planSignIn's
		// does; the steps are still printed first.
		steps := o.Steps
		o.Steps = func(s authcontract.Step) error {
			if err := steps(s); err != nil {
				return err
			}
			if s.Kind == authcontract.StepOpenURL {
				return errAtBrowser
			}
			return nil
		}
		return planSignIn(run)(ctx, id, p, o, store)
	})
	r := c.run("auth", "login", "openai", "--json")
	lines := jsonLines(t, r.stdout)
	if r.code != ExitUsage || r.stderr != "" || len(lines) != 2 {
		t.Fatalf("exit %d, stdout %q, stderr %q; want exit 2, the method line and a failed step, nothing on stderr", r.code, r.stdout, r.stderr)
	}
	s := lines[1].Step
	if lines[0].Event != "method" || lines[0].Method.ID != "chatgpt_plan" || s.Kind != authcontract.StepFailed || s.Seq != 1 || s.Error == nil ||
		s.Error.Kind != authcontract.FailureUnsupported || !strings.Contains(s.Error.Message, "terminal") || !strings.Contains(s.Error.Message, "ACP client") {
		t.Errorf("the lines %+v, want chatgpt_plan, then a failed step seq 1 of kind unsupported naming a terminal and an ACP client", lines)
	}
	if started != 0 || len(run.seen()) != 0 {
		t.Errorf("the sign-in started %d times and did %q, want it refused before it began", started, run.seen())
	}
	if _, err := os.Stat(c.authFile()); err == nil {
		t.Error("the refusal wrote auth.json")
	}

	c.write(c.authFile(), `{"openai": {"type": "oauth", "access": "a", "refresh": "r", "expires": 4102444800000, "issuer": "https://auth.openai.com",
  "clientId": "oaiapp_test", "resource": "https://api.openai.com/v1", "kind": "chatgpt_plan", "consents": "chatgpt_plan.background_use=declined"}}`)
	r = c.run("auth", "login", "openai", "--json")
	lines = jsonLines(t, r.stdout)
	if started != 1 || len(lines) < 3 {
		t.Fatalf("with the consent answered, the sign-in started %d times, stdout %q; want it started once", started, r.stdout)
	}
	if open, last := lines[1], lines[len(lines)-1]; open.Step == nil || open.Step.Kind != authcontract.StepOpenURL || last.Step == nil ||
		last.Step.Kind != authcontract.StepFailed || last.Step.Error.Kind == authcontract.FailureUnsupported {
		t.Errorf("with the consent answered, the lines %+v, want the browser's open_url step first, then the end the test gave it, not a refusal to answer", lines)
	}
}

// Under --json no browser opens on the host: a browser sign-in prints its
// authorize URL as an open_url step with no accepts and leaves the opening to
// whoever reads it; the openURL seam is never called (0051-MADR D5).
func TestAuthLoginJSONNoHostBrowser(t *testing.T) {
	c, _, calls := issuerEnv(t)
	var opened []string
	replace(t, &openURL, func(_ context.Context, u string) error { opened = append(opened, u); return nil })
	r := c.run("auth", "login", "grok", "--method", "browser", "--json")
	if r.code != ExitOK || r.stderr != "" {
		t.Fatalf("exit %d, stderr %q; want 0 and nothing", r.code, r.stderr)
	}
	lines := jsonLines(t, r.stdout)
	var shown []string
	for _, l := range lines {
		if l.Step != nil && l.Step.Kind == authcontract.StepOpenURL {
			shown = append(shown, l.Step.URL)
			if l.Step.Accepts != authcontract.PasteNone {
				t.Errorf("the open_url step accepts %q, want no paste under --json", l.Step.Accepts)
			}
		}
	}
	if len(shown) != 1 || !strings.Contains(shown[0], "/oauth2/authorize?") || len(opened) != 0 {
		t.Errorf("the URL was shown %q and opened %q; want it shown once as a step and never opened", shown, opened)
	}
	if last := lines[len(lines)-1]; last.Step == nil || last.Step.Kind != authcontract.StepDone || !slices.Equal(*calls, []string{"grok browser"}) {
		t.Errorf("the sign-in ended %+v after %q, want grok's browser sign-in, done", last, *calls)
	}
}

// A failure is the failed step with the kind provider.ErrorKind gives the
// error SignIn returned, which holds the SDK's own error inside a
// *provider.SignInError: a refused device sign-in is denied, and its message
// is what gobble prints without --json; the exit is 1, stderr is empty, and
// nothing is written (0051-MADR D5, D6).
func TestAuthLoginJSONFailureKind(t *testing.T) {
	c, ti, _ := issuerEnv(t)
	ti.denying()
	text := c.run("auth", "login", "grok", "--device")
	_, last, _ := strings.Cut(strings.TrimSuffix(text.stderr, "\n"), "\n")
	msg, ok := strings.CutPrefix(last, "Error: ")
	if text.code != ExitFailure || !ok {
		t.Fatalf("without --json: exit %d, stderr %q", text.code, text.stderr)
	}
	r := c.run("auth", "login", "grok", "--device", "--json")
	lines := jsonLines(t, r.stdout)
	if r.code != ExitFailure || r.stderr != "" || len(lines) != 4 {
		t.Fatalf("exit %d, stdout %q, stderr %q; want exit 1, the method line, show_code, waiting and a failed step", r.code, r.stdout, r.stderr)
	}
	s := lines[3].Step
	var got authcontract.Failure
	if s.Error != nil {
		got = *s.Error
	}
	if s.Kind != authcontract.StepFailed || s.Seq != 3 || s.Error == nil || got.Kind != authcontract.FailureDenied || got.Retryable || got.Message != msg {
		t.Errorf("the last step is %s, seq %d, error %+v; want failed, seq 3, kind denied, the message %q", s.Kind, s.Seq, got, msg)
	}
	if _, err := os.Stat(c.authFile()); err == nil {
		t.Error("a refused sign-in wrote auth.json")
	}
}

// Each step kind gobble prints has the members of the contract's golden vector
// of that kind and no other, and decodes into the same Go type; the vectors
// are read from the module's source at the pinned tag (go-llmprovider-sdk
// 0073-MADR-publish-an-auth-contract-module.md D5).
func TestAuthLoginJSONMatchesVectors(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}", "github.com/maccavelli/go-llmprovider-sdk/authcontract").Output() //nolint:gosec // G204: a fixed command
	if err != nil {
		t.Fatalf("find the contract's source: %v", err)
	}
	dir := filepath.Join(strings.TrimSpace(string(out)), "testdata", "vectors")
	vector := map[authcontract.StepKind]string{
		authcontract.StepOpenURL: "step-open-url.json", authcontract.StepShowCode: "step-show-code.json", authcontract.StepWaiting: "step-waiting.json",
		authcontract.StepDone: "step-done.json", authcontract.StepFailed: "step-failed.json",
	}
	seen := map[authcontract.StepKind]bool{}
	for _, args := range [][]string{{"grok", "--device"}, {"grok", "--method", "browser"}} {
		c, _, _ := issuerEnv(t)
		compareWithVectors(t, dir, vector, seen, c.run(append([]string{"auth", "login", "--json"}, args...)...).stdout)
	}
	c, ti, _ := issuerEnv(t)
	ti.denying()
	compareWithVectors(t, dir, vector, seen, c.run("auth", "login", "grok", "--device", "--json").stdout)
	for _, k := range slices.Sorted(maps.Keys(vector)) {
		if !seen[k] {
			t.Errorf("no run printed a %s step, so its vector %s was not compared", k, vector[k])
		}
	}
}

// compareWithVectors holds each step line of stdout to the vector of its kind
// in dir: the same Go type, the same members (memberDiff), and the members
// the contract's types require. It records the kinds it saw in seen.
func compareWithVectors(t *testing.T, dir string, vector map[authcontract.StepKind]string, seen map[authcontract.StepKind]bool, stdout string) {
	t.Helper()
	for raw := range strings.SplitSeq(strings.TrimSuffix(stdout, "\n"), "\n") {
		var l struct {
			Step map[string]any `json:"step"`
		}
		if err := json.Unmarshal([]byte(raw), &l); err != nil {
			t.Fatal(err)
		}
		if l.Step == nil {
			continue
		}
		kind := authcontract.StepKind(l.Step["kind"].(string)) //nolint:forcetypeassert // a line the test decoded as a step
		name, ok := vector[kind]
		if !ok {
			t.Errorf("the line %s is a step of kind %s, which has no vector", raw, kind)
			continue
		}
		seen[kind] = true
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		var golden authcontract.Step
		if err := json.Unmarshal(b, &golden, json.RejectUnknownMembers(true)); err != nil || golden.Kind != kind {
			t.Fatalf("%s: %v, kind %s", name, err, golden.Kind)
		}
		var want map[string]any
		if err := json.Unmarshal(b, &want); err != nil {
			t.Fatal(err)
		}
		extra, missing := memberDiff(l.Step, want, "")
		if len(extra) != 0 || len(missing) != 0 {
			t.Errorf("the %s step %s has members %v that %s does not, and lacks %v that it has", kind, raw, extra, name, missing)
		}
		requireMembers(t, raw, "", l.Step, reflect.TypeFor[authcontract.Step]())
		if f, ok := l.Step["error"].(map[string]any); ok {
			requireMembers(t, raw, "error.", f, reflect.TypeFor[authcontract.Failure]())
		}
	}
}

// requireMembers fails for each member the contract's type typ requires that
// got, the printed object at path, lacks.
func requireMembers(t *testing.T, raw, path string, got map[string]any, typ reflect.Type) {
	t.Helper()
	for _, m := range requiredMembers(typ) {
		if _, ok := got[m]; !ok {
			t.Errorf("the step %s lacks %s%s, which the contract's %s requires", raw, path, m, typ.Name())
		}
	}
}

// memberDiff compares the members of a printed step, got, with those of the
// contract's vector for its kind, want, by their path, at any depth, and
// returns those got has and want has not, and those want has and got has not.
// It forgives two things, each at the step's top level only: a "secret" whose
// value is false, which encoding/json/v2, which jsonx writes with, prints
// where the SDK's own encoding omits it ("P5 record (2026-11-15)", deviation
// 1), and a missing "text", which the SDK's steps from a real sign-in do not
// set, so gobble prints none. Nothing else is forgiven: the members the
// contract's types require (a name with neither omitempty nor omitzero in its
// tag: schema_version, seq and kind of a Step, kind, message and retryable of
// its Failure) are held by requiredMembers, and every other member of a
// vector is a fact of the step (url, code, expires_at, yields, error) that
// the sign-in states and gobble must carry.
func memberDiff(got, want map[string]any, path string) (extra, missing []string) {
	for k, v := range got {
		w, ok := want[k]
		switch {
		case !ok && path == "" && k == "secret" && v == false:
		case !ok:
			extra = append(extra, path+k)
		default:
			gm, gok := v.(map[string]any)
			wm, wok := w.(map[string]any)
			if gok && wok {
				e, m := memberDiff(gm, wm, path+k+".")
				extra, missing = append(extra, e...), append(missing, m...)
			}
		}
	}
	for k := range want {
		if _, ok := got[k]; !ok && (path != "" || k != "text") {
			missing = append(missing, path+k)
		}
	}
	return extra, missing
}

// requiredMembers are the JSON names of the fields of t that the contract's
// tag does not make optional, read from the type itself.
func requiredMembers(t reflect.Type) []string {
	var out []string
	for f := range t.Fields() {
		tag := f.Tag.Get("json")
		name, opts, _ := strings.Cut(tag, ",")
		if name != "" && name != "-" && !strings.Contains(opts, "omitempty") && !strings.Contains(opts, "omitzero") {
			out = append(out, name)
		}
	}
	return out
}

// A failed step's retryable is the SDK's for the same error: the API error's
// own Retryable, which honours the service's x-should-retry, when the error
// holds one, and otherwise true for rate_limited, offline and unavailable
// (go-llmprovider-sdk v1.31.0, signinflow.Failure: "whether the same sign-in
// can succeed later"; the PLAN and the contract's Failure say no more). A
// sign-in ends with an API error when the listing that checks the new
// credential refuses it with HTTP 401: the wizard wraps the listing's error in
// its refusal, so the real wizard against an issuer whose listing refuses, for
// Kilo's device sign-in and Grok's, gives a failed step of kind auth whose
// retryable is the reply's x-should-retry, true, false, or, when it states none,
// false: the kind auth is in no table.
func TestAuthLoginJSONRetryable(t *testing.T) {
	for _, id := range []string{"kilo", "grok"} {
		for _, tc := range []struct {
			name, header string
			want         bool
		}{{"no x-should-retry", "", false}, {"x-should-retry true", "true", true}, {"x-should-retry false", "false", false}} {
			if id == "kilo" && tc.header != "true" {
				continue // Kilo's device poll takes seconds; Grok holds the other two
			}
			t.Run(id+", "+tc.name, func(t *testing.T) {
				c, ti, _ := issuerEnv(t)
				ti.refusing(tc.header)
				args := []string{"auth", "login", id, "--json"}
				if id == "grok" {
					args = append(args, "--device")
				}
				r := c.run(args...)
				lines := jsonLines(t, r.stdout)
				if r.code != ExitFailure || r.stderr != "" || len(lines) != 4 {
					t.Fatalf("exit %d, stdout %q, stderr %q; want exit 1, the method line, show_code, waiting and a failed step", r.code, r.stdout, r.stderr)
				}
				last := lines[3].Step
				var got authcontract.Failure
				if last.Error != nil {
					got = *last.Error
				}
				if last.Kind != authcontract.StepFailed || got.Kind != authcontract.FailureAuth || got.Retryable != tc.want {
					t.Errorf("the last step is %s with error %+v, want a failed step of kind auth with retryable %t", last.Kind, got, tc.want)
				}
			})
		}
	}
}

// done follows the store: when auth.json cannot be written, a key's SetKey and
// a session's save fail, the last step is failed, and no done step was
// printed before it (0051-MADR D5). The lock file's name is a directory, so
// the lock cannot be taken.
func TestAuthLoginJSONDoneFollowsTheStore(t *testing.T) {
	for _, args := range [][]string{{"kilo"}, {"grok", "--device"}} {
		t.Run(args[0], func(t *testing.T) {
			c, _, _ := issuerEnv(t)
			if err := os.Mkdir(c.authFile()+".lock", 0o700); err != nil {
				t.Fatal(err)
			}
			r := c.run(append([]string{"auth", "login", "--json"}, args...)...)
			lines := jsonLines(t, r.stdout)
			if r.code != ExitFailure || r.stderr != "" || len(lines) < 2 {
				t.Fatalf("exit %d, stdout %q, stderr %q; want exit 1 and a failed last step", r.code, r.stdout, r.stderr)
			}
			for _, l := range lines {
				if l.Step != nil && l.Step.Kind == authcontract.StepDone {
					t.Errorf("a done step was printed though the store failed: %s", r.stdout)
				}
			}
			if last := lines[len(lines)-1]; last.Step == nil || last.Step.Kind != authcontract.StepFailed {
				t.Errorf("the last line %+v, want a failed step", last)
			}
		})
	}
}

// A sign-in ended by a signal prints a failed step of kind canceled, and the
// exit is 128 plus the signal, not 1: the path Main takes, watchSignals' context
// cancelled with the signal as its cause, run, then signalExit (0008-MADR D10).
func TestAuthLoginJSONEndedBySignal(t *testing.T) {
	c, _, _ := issuerEnv(t)
	notify, ch := captureNotify()
	var stdout, stderr bytes.Buffer
	var root Root
	k, err := newParser(&root, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	intr := &interrupts{}
	ctx, stop := watchSignals(t.Context(), []os.Signal{syscall.SIGTERM}, notify, intr)
	defer stop()
	inner := signInWith
	replace(t, &signInWith, func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		steps := o.Steps
		o.Steps = func(s authcontract.Step) error {
			if err := steps(s); err != nil {
				return err
			}
			if s.Kind == authcontract.StepWaiting {
				ch() <- syscall.SIGTERM
			}
			return nil
		}
		return inner(ctx, id, p, o, store)
	})
	out := &Output{out: &stdout, err: &stderr}
	code := run(ctx, k, &root, []string{"auth", "login", "--json", "grok", "--device"}, nil, out, intr)
	if code != ExitFailure {
		t.Errorf("the command's own code is %d, want %d", code, ExitFailure)
	}
	if sig, ok := signalExit(ctx); !ok || sig != 128+int(syscall.SIGTERM) {
		t.Errorf("the exit by the signal is %d, %v; want %d", sig, ok, 128+int(syscall.SIGTERM))
	}
	lines := jsonLines(t, stdout.String())
	steps := wantSteps(t, lines, "grok", "device_code", authcontract.StepShowCode, authcontract.StepWaiting, authcontract.StepFailed)
	if len(steps) == 3 && (steps[2].Error == nil || steps[2].Error.Kind != authcontract.FailureCanceled || steps[2].Error.Retryable) {
		t.Errorf("the last step %+v, want failed, canceled, not retryable", steps[2])
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr %q, want nothing", stderr.String())
	}
	if _, err := os.Stat(c.authFile()); err == nil {
		t.Error("a cancelled sign-in wrote auth.json")
	}
}

// done states what the sign-in yielded as the SDK's own done step said it,
// carried in Credential.Yields, in place of the method's fact, for a method
// whose yields the SDK states only at its end; with none carried, the method's
// (0051-PLAN P5 review, item 7). The stand-in is signInWith, which saves a
// login as a real one does and returns the credential.
func TestAuthLoginJSONDoneYieldsWhatTheSignInSays(t *testing.T) {
	for _, tc := range []struct {
		name  string
		given authcontract.Yields
		want  authcontract.Yields
	}{
		{"the SDK's done step states one", authcontract.YieldsAccessOnly, authcontract.YieldsAccessOnly},
		{"none carried", "", authcontract.YieldsOAuthSession},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCfgEnv(t)
			replace(t, &signInWith, func(ctx context.Context, id string, _ provider.Prompter, _ provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
				return provider.Credential{Yields: tc.given}, store.Save(ctx, id, &provider.Login{Access: "a", Refresh: "r"})
			})
			r := c.run("auth", "login", "grok", "--device", "--json")
			lines := jsonLines(t, r.stdout)
			if r.code != ExitOK || len(lines) < 2 {
				t.Fatalf("exit %d, stdout %q", r.code, r.stdout)
			}
			if last := lines[len(lines)-1]; last.Step == nil || last.Step.Kind != authcontract.StepDone || last.Step.Yields != tc.want {
				t.Errorf("the last line %+v, want a done step that yields %s", last, tc.want)
			}
		})
	}
}

// A first plan sign-in is refused by CheckNoQuestion before it starts; where
// the check passes and the SDK would still ask, because the stored session is
// a plan session with no client id, the driver's backstop refuses it, with the
// same lines and exit, the wizard having started (0051-PLAN P5 review, item
// 11). The other edge, that the SDK asks every consent and the check only the
// required ones, has no offered method to run: TestSignInStepsRefuseWhatAsks
// holds the driver for any step that asks.
func TestAuthLoginJSONBackstopMatchesTheCheck(t *testing.T) {
	c := newCfgEnv(t)
	run := &planRun{}
	started := 0
	replace(t, &signInWith, func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		started++
		// A sign-in that is not refused ends at its browser, as in
		// TestAuthLoginJSONRefusesAQuestion, so a missed refusal fails and
		// does not wait for a browser.
		steps := o.Steps
		o.Steps = func(s authcontract.Step) error {
			if err := steps(s); err != nil {
				return err
			}
			if s.Kind == authcontract.StepOpenURL {
				return errAtBrowser
			}
			return nil
		}
		return planSignIn(run)(ctx, id, p, o, store)
	})
	checked := c.run("auth", "login", "openai", "--json")
	if started != 0 || checked.code != ExitUsage {
		t.Fatalf("the check: started %d, exit %d, stdout %q; want it refused before the wizard", started, checked.code, checked.stdout)
	}
	c.write(c.authFile(), `{"openai": {"type": "oauth", "access": "a", "refresh": "r", "expires": 4102444800000, "issuer": "https://auth.openai.com",
  "resource": "https://api.openai.com/v1", "kind": "chatgpt_plan", "consents": "chatgpt_plan.background_use=granted"}}`)
	backstop := c.run("auth", "login", "openai", "--json")
	if started != 1 {
		t.Fatalf("the check refused a plan session with an answer and no client id, started %d; stdout %q", started, backstop.stdout)
	}
	if backstop.code != checked.code || backstop.stdout != checked.stdout || backstop.stderr != checked.stderr {
		t.Errorf("the backstop printed exit %d, stdout %q; the check printed exit %d, stdout %q; want the same", backstop.code, backstop.stdout, checked.code, checked.stdout)
	}
}

// A command line Kong refuses, with auth login selected and --json read as true
// before Kong stopped, is one failed step of kind invalid_request, seq 1, exit
// 2, and nothing on stderr (0051-MADR D5; the owner's ruling, relayed by the
// coordinator, "Emit the failed step from the parser's own context"). Through
// Main's real path. The line's provider is empty, the context not having it at a
// parse error.
func TestAuthLoginJSONParseFailure(t *testing.T) {
	c := newCfgEnv(t)
	for _, tc := range []struct {
		args    []string
		message string
	}{
		{[]string{"auth", "login", "--json"}, `expected "<provider>"`},
		{[]string{"auth", "login", "--json", "--bogus", "grok"}, "unknown flag --bogus"},
		{[]string{"auth", "login", "--json", "grok", "--method"}, "--method: expected string value"},
	} {
		r := c.run(tc.args...)
		lines := jsonLines(t, r.stdout)
		if r.code != ExitUsage || r.stderr != "" || len(lines) != 1 {
			t.Errorf("%q: exit %d, stdout %q, stderr %q; want exit 2, one line, nothing on stderr", tc.args, r.code, r.stdout, r.stderr)
			continue
		}
		s := lines[0].Step
		if lines[0].Event != "step" || lines[0].Provider != "" || s.Kind != authcontract.StepFailed || s.Seq != 1 || s.SchemaVersion != authcontract.SchemaVersion ||
			s.Error == nil || s.Error.Kind != authcontract.FailureInvalidRequest || s.Error.Retryable || !strings.HasPrefix(s.Error.Message, tc.message) {
			t.Errorf("%q: the line %+v, want one failed step, seq 1, invalid_request, a message that begins %q", tc.args, lines[0], tc.message)
		}
	}
}

// What the parser's context does not show stays Kong's text on stderr, exit 2,
// nothing on stdout: --json read as false; a --json after the argument Kong
// rejected, which is never read (the residue the guide states); and every other
// command that has a --json flag, whose parse errors this change leaves as they
// were. The commands with a --json flag are config get, config path, version,
// update, auth status, auth check, mcp list and providers, and auth login.
func TestAuthLoginJSONParseFailureLeavesTheRest(t *testing.T) {
	c := newCfgEnv(t)
	for _, tc := range []struct {
		name    string
		args    []string
		message string
	}{
		{"--json=false", []string{"auth", "login", "--json=false"}, `expected "<provider>"`},
		{"--json before the rejected flag is read, after it is not", []string{"auth", "login", "--bogus", "--json", "grok"}, "unknown flag --bogus"},
		{"--json after the rejected argument", []string{"auth", "login", "grok", "extra", "--json"}, "unexpected argument extra"},
		{"config get", []string{"config", "get", "--json", "--bogus"}, "unknown flag --bogus"},
		{"config path", []string{"config", "path", "--json", "--bogus"}, "unknown flag --bogus"},
		{"version", []string{"version", "--json", "--bogus"}, "unknown flag --bogus"},
		{"update", []string{"update", "--json", "--bogus"}, "unknown flag --bogus"},
		{"auth status", []string{"auth", "status", "--json", "--bogus"}, "unknown flag --bogus"},
		{"auth check", []string{"auth", "check", "--json", "--bogus"}, "unknown flag --bogus"},
		{"mcp list", []string{"mcp", "list", "--json", "--bogus"}, "unknown flag --bogus"},
		{"providers", []string{"providers", "--json", "--bogus"}, "unknown flag --bogus"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := c.run(tc.args...)
			if want := "Error: " + tc.message; r.code != ExitUsage || r.stdout != "" || !strings.HasPrefix(r.stderr, want) || !strings.HasSuffix(r.stderr, "run 'gobble --help' for usage\n") {
				t.Errorf("%q: exit %d, stdout %q, stderr %q; want exit 2, nothing on stdout, Kong's text beginning %q", tc.args, r.code, r.stdout, r.stderr, want)
			}
		})
	}
}

// A failure that is not a usage error can come before the method line: a stdin
// that cannot be read under --api-key - is one failed step, seq 1, exit 1, with
// no method line, the key being read first (0051-PLAN P5 step 2 gives the
// order of a success only: the method line, then the steps).
func TestAuthLoginJSONStdinReadError(t *testing.T) {
	c := newCfgEnv(t)
	dir, err := os.Open(c.dir) // a directory reads with an error
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dir.Close() })
	r := c.runStdin(dir, "auth", "login", "openai", "--api-key", "-", "--json")
	lines := jsonLines(t, r.stdout)
	if r.code != ExitFailure || r.stderr != "" || len(lines) != 1 || lines[0].Step == nil || lines[0].Step.Kind != authcontract.StepFailed || lines[0].Step.Seq != 1 {
		t.Errorf("exit %d, stdout %q, stderr %q; want exit 1 and one failed step, seq 1, no method line", r.code, r.stdout, r.stderr)
	}
}
