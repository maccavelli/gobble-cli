package tool

import (
	"context"
	"errors"
	"fmt"

	"encoding/json/jsontext"
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

// Env is the per-call context later phases will fill.
// Phase 1 carries no workspace, client, or credentials.
type Env struct{}

// Result is the tool's reply.
type Result struct {
	Output  jsontext.Value `json:"output,omitzero"`
	IsError bool           `json:"isError,omitzero"`
}

// Option configures New.
type Option func(*config)

type config struct {
	kind        Kind
	exposure    Exposure
	annotations Annotations
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

type generic[In, Out any] struct {
	name        string
	description string
	fn          func(context.Context, In, Env) (Out, error)
	cfg         config
}

func (g *generic[In, Out]) Spec() Spec {
	return Spec{
		Name:        g.name,
		Description: g.description,
		Annotations: g.cfg.annotations,
		Kind:        g.cfg.kind,
		Exposure:    g.cfg.exposure,
	}
}

func (g *generic[In, Out]) Run(context.Context, Call, Env) (Result, error) {
	state := "unset"
	if g.fn != nil {
		state = "set"
	}
	return Result{}, fmt.Errorf("tool: %s: %s: %w", g.name, state, errors.ErrUnsupported)
}

// New returns a tool whose input type is In and whose output type is Out.
// Phase 1 does not call fn and does not derive a JSON Schema.
func New[In, Out any](name, description string, fn func(context.Context, In, Env) (Out, error), opts ...Option) Tool {
	cfg := config{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &generic[In, Out]{
		name:        name,
		description: description,
		fn:          fn,
		cfg:         cfg,
	}
}
