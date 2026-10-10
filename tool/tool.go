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
	// Family groups deferred tools that load together: loading one loads
	// every tool of its family (0012-MADR D8).
	Family string `json:"family,omitzero"`
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

// Exposure says whether the tool is offered immediately. The empty
// Exposure is direct (0012-MADR D7).
type Exposure string

const (
	// ExposureDirect offers the tool on every turn.
	ExposureDirect Exposure = "direct"
	// ExposureDeferred keeps the tool behind search, and out of the
	// requests, until a call loads it.
	ExposureDeferred Exposure = "deferred"
	// ExposureHidden neither offers the tool nor lets search find it.
	ExposureHidden Exposure = "hidden"
)

// TextFiles is a client's text files, through ACP's fs/read_text_file and
// fs/write_text_file, so a call reads and writes what the client's editor
// holds, unsaved changes included. A path is absolute, and is resolved and
// confined before it is asked for (0005-PLAN F1d-3).
type TextFiles interface {
	ReadTextFile(ctx context.Context, path string) (string, error)
	WriteTextFile(ctx context.Context, path, content string) error
}

// Terminal is a client's terminals, through ACP's terminal/* methods, so a
// command runs where the user can watch it (0005-PLAN F1d-3).
type Terminal interface {
	// Create starts a command and returns its terminal's id.
	Create(ctx context.Context, r TerminalRequest) (string, error)
	// Output is the terminal's output so far, and its exit once it has one.
	Output(ctx context.Context, id string) (TerminalOutput, error)
	// WaitForExit returns when the command exits.
	WaitForExit(ctx context.Context, id string) (TerminalExit, error)
	// Kill stops the command and keeps the terminal.
	Kill(ctx context.Context, id string) error
	// Release frees the terminal.
	Release(ctx context.Context, id string) error
}

// TerminalRequest is a command for a client's terminal. Env is KEY=VALUE
// pairs; OutputByteLimit bounds the output the client keeps, 0 for none.
type TerminalRequest struct {
	Command         string
	Args            []string
	Cwd             string
	Env             []string
	OutputByteLimit int
}

// TerminalOutput is a terminal's output, whether the client cut its start,
// and its exit, nil while the command runs.
type TerminalOutput struct {
	Output    string
	Truncated bool
	Exit      *TerminalExit
}

// TerminalExit is how a command ended: its exit code, or the signal that
// stopped it.
type TerminalExit struct {
	Code   *int
	Signal string
}

// Location is a file a call reads or changes, with a line when one is
// known (1-based; 0 for none), for the client to follow (ACP
// ToolCallLocation).
type Location struct {
	Path string `json:"path"`
	Line int    `json:"line,omitzero"`
}

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
// (0003-MADR). Deferred is the turn's deferred tools that are not loaded
// yet, for tool_search to rank (0005-PLAN F1d-2). Files and Terminal are
// the client's, when it offers them; nil otherwise (0005-PLAN F1d-3).
type Env struct {
	Cwd          string    `json:"cwd,omitzero"`
	Roots        []string  `json:"roots,omitzero"`
	AllowOutside bool      `json:"allowOutside,omitzero"`
	Environ      []string  `json:"environ,omitzero"`
	Deferred     []Spec    `json:"deferred,omitzero"`
	Files        TextFiles `json:"-"`
	Terminal     Terminal  `json:"-"`
}

// Result is the tool's reply. Output is what the model sees: a JSON string
// is given to it as text. Title, Summary and Diffs are for the client's
// display: a short phrase, a self-contained summary of at most 400 bytes,
// and the files the call changed (0008-MADR D19 item 3). Plan, when not
// nil, is the task's checklist as the call left it, for the client to show
// as its plan; an empty, non-nil Plan clears it (0005-PLAN F1d-1). Load
// names deferred tools the agent loads before its next model call, with
// their families (0005-PLAN F1d-2). Locations are the files the call
// touched, with lines where it found some (0005-PLAN F1d-3).
type Result struct {
	Output    jsontext.Value `json:"output,omitzero"`
	IsError   bool           `json:"isError,omitzero"`
	Title     string         `json:"title,omitzero"`
	Summary   string         `json:"summary,omitzero"`
	Diffs     []Diff         `json:"diffs,omitzero"`
	Plan      []PlanItem     `json:"plan,omitzero"`
	Load      []string       `json:"load,omitzero"`
	Locations []Location     `json:"locations,omitzero"`
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

// Locator is implemented by a tool that can name a call's locations
// before it runs, as Describer names its title; the client shows them
// with the call (0005-PLAN F1d-3).
type Locator interface {
	Locations(call Call, env Env) []Location
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
	family      string
	annotations Annotations
	describe    func(Call, Env) string
	outside     func(Call, Env) []string
	locate      func(Call, Env) []Location
	prepare     func(jsontext.Value) (jsontext.Value, error)
	schema      func(*jsonschema.Schema)
}

// WithOutside sets how a call's paths outside the roots are found
// (Confined). Without it a tool names no paths.
func WithOutside(outside func(call Call, env Env) []string) Option {
	return func(cfg *config) { cfg.outside = outside }
}

// WithLocations sets how a call's locations are named before it runs
// (Locator). Without it a tool names none.
func WithLocations(locate func(call Call, env Env) []Location) Option {
	return func(cfg *config) { cfg.locate = locate }
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

// WithExposure sets direct, deferred or hidden exposure.
func WithExposure(exposure Exposure) Option {
	return func(cfg *config) { cfg.exposure = exposure }
}

// WithFamily sets the family a deferred tool loads with (0012-MADR D8).
func WithFamily(family string) Option {
	return func(cfg *config) { cfg.family = family }
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
		Family:      g.cfg.family,
	}
}

// Describe titles a call: WithDescribe's function, or the tool's name.
func (g *generic[In, Out]) Describe(call Call, env Env) string {
	if g.cfg.describe != nil {
		return g.cfg.describe(call, env)
	}
	return g.name
}

// Locations is WithLocations' function, or none.
func (g *generic[In, Out]) Locations(call Call, env Env) []Location {
	if g.cfg.locate != nil {
		return g.cfg.locate(call, env)
	}
	return nil
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
