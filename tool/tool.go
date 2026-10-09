package tool

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// Tool is one callable operation.
type Tool interface {
	Spec() Spec
	Run(ctx context.Context, call Call, env Env) (Result, error)
}

// Spec is the description a model sees.
type Spec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitzero"`
	InputSchema jsontext.Value `json:"inputSchema,omitzero"`
	Annotations Annotations    `json:"annotations,omitzero"`
	Kind        Kind           `json:"kind,omitzero"`
	Exposure    Exposure       `json:"exposure,omitzero"`
}

// Annotations carries MCP tool-behavior hints.
type Annotations struct {
	ReadOnlyHint    bool `json:"readOnlyHint,omitzero"`
	DestructiveHint bool `json:"destructiveHint,omitzero"`
	IdempotentHint  bool `json:"idempotentHint,omitzero"`
	OpenWorldHint   bool `json:"openWorldHint,omitzero"`
}

// Kind is the ACP tool kind.
type Kind string

// Tool kinds use the ACP tool-kind names.
const (
	KindRead    Kind = "read"
	KindEdit    Kind = "edit"
	KindDelete  Kind = "delete"
	KindMove    Kind = "move"
	KindSearch  Kind = "search"
	KindExecute Kind = "execute"
	KindThink   Kind = "think"
	KindFetch   Kind = "fetch"
	KindOther   Kind = "other"
)

// Exposure says whether the tool is offered immediately.
type Exposure string

const (
	// ExposureDirect offers the tool on every turn.
	ExposureDirect Exposure = "direct"
	// ExposureDeferred keeps the tool behind search.
	ExposureDeferred Exposure = "deferred"
)

// Call is one invocation.
type Call struct {
	ID   string         `json:"id,omitzero"`
	Name string         `json:"name"`
	Args jsontext.Value `json:"args,omitzero"`
}

// Env is the per-call context. Cwd is the session's working directory, which
// relative paths resolve against. Roots are the directories file tools may
// use without asking: Cwd first, then the session's additional directories,
// all absolute. AllowOutside is set on a call whose paths outside Roots were
// approved (Confined). Environ is KEY=VALUE pairs a tool's child processes
// get on top of gobble's own environment, such as the session markers
// (0003-MADR).
type Env struct {
	Cwd          string   `json:"cwd,omitzero"`
	Roots        []string `json:"roots,omitzero"`
	AllowOutside bool     `json:"allowOutside,omitzero"`
	Environ      []string `json:"environ,omitzero"`
}

// Result is the tool's reply. Output is what the model sees: a JSON string
// is given to it as text. Title, Summary and Diffs are for the client's
// display: a short phrase, a self-contained summary of at most 400 bytes,
// and the files the call changed (0008-MADR D19 item 3). Plan, when not
// nil, is the task's checklist as the call left it, for the client to show
// as its plan; an empty, non-nil Plan clears it (0005-PLAN F1d-1).
type Result struct {
	Output  jsontext.Value `json:"output,omitzero"`
	IsError bool           `json:"isError,omitzero"`
	Title   string         `json:"title,omitzero"`
	Summary string         `json:"summary,omitzero"`
	Diffs   []Diff         `json:"diffs,omitzero"`
	Plan    []PlanItem     `json:"plan,omitzero"`
}

// PlanItem is one step of a plan. Status is pending, in_progress or
// completed; Priority is high, medium or low. Both use ACP's plan names.
type PlanItem struct {
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

// Plan statuses and priorities.
const (
	PlanPending    = "pending"
	PlanInProgress = "in_progress"
	PlanCompleted  = "completed"
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// Diff is one file a call changed. OldText is nil for a new file.
type Diff struct {
	Path    string  `json:"path"`
	OldText *string `json:"oldText,omitzero"`
	NewText string  `json:"newText"`
}

// Text is the output as the model reads it: a JSON string's value, or the
// JSON itself.
func (r Result) Text() string {
	var s string
	if err := json.Unmarshal(r.Output, &s); err == nil {
		return s
	}
	return string(r.Output)
}

// TextResult is a successful result whose output is text.
func TextResult(text string) Result {
	return Result{Output: quote(text)}
}

// ErrorResult is a failed result the model reads as text.
func ErrorResult(text string) Result {
	return Result{Output: quote(text), IsError: true}
}

func quote(s string) jsontext.Value {
	b, err := json.Marshal(s)
	if err != nil { // a string always encodes
		panic(err)
	}
	return b
}

// Describer is implemented by a tool that can title a call before it runs,
// such as "read internal/cli/term.go". Clients show the title.
type Describer interface {
	Describe(call Call, env Env) string
}

// Confined is implemented by a tool whose calls name files. Outside returns
// the paths a call names that lie outside env.Roots. Such a call needs a
// permission decision even when the tool is read-only, and runs with
// env.AllowOutside set only when it is allowed (0005-MADR "Confinement").
type Confined interface {
	Outside(call Call, env Env) []string
}

// Option configures New.
type Option func(*config)

type config struct {
	kind        Kind
	exposure    Exposure
	annotations Annotations
	describe    func(Call, Env) string
	outside     func(Call, Env) []string
	prepare     func(jsontext.Value) (jsontext.Value, error)
	schema      func(*jsonschema.Schema)
}

// WithOutside sets how a call's paths outside the roots are found
// (Confined). Without it a tool names no paths.
func WithOutside(outside func(call Call, env Env) []string) Option {
	return func(cfg *config) { cfg.outside = outside }
}

// WithPrepare sets a rewrite of a call's arguments that runs before they
// are validated, for shapes a model sends that the schema does not accept,
// such as Pi's edit repairs. An error's text is what the model reads.
func WithPrepare(prepare func(args jsontext.Value) (jsontext.Value, error)) Option {
	return func(cfg *config) { cfg.prepare = prepare }
}

// WithDescribe sets how a call is titled before it runs (Describer). Without
// it the title is the tool's name.
func WithDescribe(describe func(call Call, env Env) string) Option {
	return func(cfg *config) { cfg.describe = describe }
}

// WithKind sets the ACP kind.
func WithKind(kind Kind) Option {
	return func(cfg *config) { cfg.kind = kind }
}

// WithExposure sets direct or deferred exposure.
func WithExposure(exposure Exposure) Option {
	return func(cfg *config) { cfg.exposure = exposure }
}

// WithAnnotations sets the MCP hints.
func WithAnnotations(annotations Annotations) Option {
	return func(cfg *config) { cfg.annotations = annotations }
}

// WithSchema adds what struct tags cannot say to the schema a model sees:
// bounds, defaults, enums and minItems (0012-MADR D1, D2). Run does not
// validate against them: the tool's own code enforces what WithSchema
// publishes, in its own words (0012-MADR D2 rule 8).
func WithSchema(adjust func(s *jsonschema.Schema)) Option {
	return func(cfg *config) { cfg.schema = adjust }
}

type generic[In, Out any] struct {
	name        string
	description string
	fn          func(context.Context, In, Env) (Out, error)
	cfg         config
	schema      jsontext.Value
	resolved    *jsonschema.Resolved
}

func (g *generic[In, Out]) Spec() Spec {
	return Spec{
		Name:        g.name,
		Description: g.description,
		InputSchema: g.schema,
		Annotations: g.cfg.annotations,
		Kind:        g.cfg.kind,
		Exposure:    g.cfg.exposure,
	}
}

// Describe titles a call: WithDescribe's function, or the tool's name.
func (g *generic[In, Out]) Describe(call Call, env Env) string {
	if g.cfg.describe != nil {
		return g.cfg.describe(call, env)
	}
	return g.name
}

// Outside is WithOutside's function, or no paths.
func (g *generic[In, Out]) Outside(call Call, env Env) []string {
	if g.cfg.outside != nil {
		return g.cfg.outside(call, env)
	}
	return nil
}

// Run validates the arguments against the input schema, decodes them into
// In and calls fn. Arguments the model got wrong, and an error fn returns,
// are an error result the model reads; only a done ctx is an error.
func (g *generic[In, Out]) Run(ctx context.Context, call Call, env Env) (Result, error) {
	args := call.Args
	if len(args) == 0 {
		args = jsontext.Value("{}")
	}
	if g.cfg.prepare != nil {
		prepared, err := g.cfg.prepare(args)
		if err != nil {
			return ErrorResult(err.Error()), nil
		}
		args = prepared
	}
	var instance any
	if err := json.Unmarshal(args, &instance); err != nil {
		return ErrorResult(fmt.Sprintf("%s: the arguments are not JSON: %v", g.name, err)), nil
	}
	if err := g.resolved.Validate(instance); err != nil {
		return ErrorResult(fmt.Sprintf("%s: invalid arguments: %v", g.name, err)), nil
	}
	var in In
	if err := json.Unmarshal(args, &in); err != nil {
		return ErrorResult(fmt.Sprintf("%s: invalid arguments: %v", g.name, err)), nil
	}
	out, err := g.fn(ctx, in, env)
	if ctx.Err() != nil {
		return Result{}, context.Cause(ctx)
	}
	if err != nil {
		return ErrorResult(err.Error()), nil
	}
	return resultOf(out)
}

// resultOf makes the result of a successful call: a Result is used as it
// is, a string is text, and anything else is its JSON.
func resultOf(out any) (Result, error) {
	switch v := out.(type) {
	case Result:
		return v, nil
	case string:
		return TextResult(v), nil
	}
	b, err := json.Marshal(out)
	if err != nil {
		return Result{}, fmt.Errorf("tool: encode output: %w", err)
	}
	return Result{Output: b}, nil
}

// New returns a tool whose input type is In and whose output type is Out.
// The input schema is derived from In with github.com/google/jsonschema-go
// (0004-MADR), so fields tagged omitzero are optional and a jsonschema tag
// is a description. An Out of type Result is returned as is, and a string
// is text. New panics if In has no JSON Schema, which is a programming
// error found by the first test that builds the tool. The schema a model
// sees (Spec.InputSchema) has no null types and carries WithSchema's
// additions; Run validates against the schema derived from In alone, so
// those additions are the tool's own code to enforce.
func New[In, Out any](name, description string, fn func(context.Context, In, Env) (Out, error), opts ...Option) Tool {
	cfg := config{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	schema, err := jsonschema.For[In](nil)
	if err != nil {
		panic(fmt.Sprintf("tool: %s: input schema: %v", name, err))
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		panic(fmt.Sprintf("tool: %s: resolve input schema: %v", name, err))
	}
	published, err := jsonschema.For[In](nil)
	if err != nil {
		panic(fmt.Sprintf("tool: %s: input schema: %v", name, err))
	}
	nonNull(published)
	if cfg.schema != nil {
		cfg.schema(published)
	}
	if _, err := published.Resolve(nil); err != nil {
		panic(fmt.Sprintf("tool: %s: resolve published schema: %v", name, err))
	}
	raw, err := json.Marshal(published)
	if err != nil {
		panic(fmt.Sprintf("tool: %s: encode input schema: %v", name, err))
	}
	return &generic[In, Out]{
		name:        name,
		description: description,
		fn:          fn,
		cfg:         cfg,
		schema:      raw,
		resolved:    resolved,
	}
}

// nonNull drops the "null" that jsonschema-go adds to the type of a slice
// or a pointer, here and in every nested property and item: a model is
// never told an argument may be null (0012-MADR D2 rule 5).
func nonNull(s *jsonschema.Schema) {
	if s == nil {
		return
	}
	if len(s.Types) == 2 && s.Types[0] == "null" {
		s.Type, s.Types = s.Types[1], nil
	}
	for _, p := range s.Properties {
		nonNull(p)
	}
	nonNull(s.Items)
}
