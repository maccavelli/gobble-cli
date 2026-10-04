package gobble

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/maccavelli/gobble-cli/hook"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/tool"
)

// Dirs names the four role directories.
// Phase 2 resolves them. Phase 1 only records that a value was supplied.
type Dirs struct {
	Config string
	Data   string
	State  string
	Cache  string
}

// Option configures New.
type Option func(*Runtime)

// Runtime is a constructed gobble instance.
// Its methods are stubs in this phase.
type Runtime struct {
	n int
}

// New builds a Runtime.
// A nil context is rejected. Options are counted and not invoked.
func New(ctx context.Context, opts ...Option) (*Runtime, error) {
	if ctx == nil {
		return nil, fmt.Errorf("gobble: nil context: %w", errors.ErrUnsupported)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rt := &Runtime{}
	for _, opt := range opts {
		if opt != nil {
			opt(rt)
		}
	}
	return rt, nil
}

// WithProviders records provider values. They are not called.
func WithProviders(providers ...llm.Provider) Option {
	return func(rt *Runtime) { rt.n += len(providers) }
}

// WithTools records tool values. They are not called.
func WithTools(tools ...tool.Tool) Option {
	return func(rt *Runtime) { rt.n += len(tools) }
}

// WithHooks records hook values. They are not called.
func WithHooks(hooks ...hook.Hook) Option {
	return func(rt *Runtime) { rt.n += len(hooks) }
}

// WithSessionStore records a session store. It is not opened.
func WithSessionStore(store session.Store) Option {
	return func(rt *Runtime) {
		if store != nil {
			rt.n++
		}
	}
}

// WithPolicy records a permission policy. It is not consulted.
func WithPolicy(policy permission.Policy) Option {
	return func(rt *Runtime) {
		if policy != nil {
			rt.n++
		}
	}
}

// WithLogger records that a logger was supplied. It is not used.
func WithLogger(logger *slog.Logger) Option {
	return func(rt *Runtime) {
		if logger != nil {
			rt.n++
		}
	}
}

// WithTracerProvider records that a tracer provider was supplied.
// The argument is any so this package does not import OpenTelemetry.
func WithTracerProvider(provider any) Option {
	return func(rt *Runtime) {
		if provider != nil {
			rt.n++
		}
	}
}

// WithDirs records directory paths. They are not created.
func WithDirs(dirs Dirs) Option {
	return func(rt *Runtime) {
		if dirs != (Dirs{}) {
			rt.n++
		}
	}
}

// ServeACP would serve an ACP agent on in and out.
// It returns an error wrapping errors.ErrUnsupported.
func (rt *Runtime) ServeACP(ctx context.Context, in io.Reader, out io.Writer) error {
	if rt == nil || ctx == nil {
		return fmt.Errorf("gobble: ServeACP: %w", errors.ErrUnsupported)
	}
	return fmt.Errorf("gobble: ServeACP: options %d: readers %d: %w", rt.n, countReaders(in, out), errors.ErrUnsupported)
}

// Connect would return an in-process ACP client.
// It returns an error wrapping errors.ErrUnsupported and no client.
func (rt *Runtime) Connect(ctx context.Context) error {
	if rt == nil || ctx == nil {
		return fmt.Errorf("gobble: Connect: %w", errors.ErrUnsupported)
	}
	return fmt.Errorf("gobble: Connect: options %d: %w", rt.n, errors.ErrUnsupported)
}

func countReaders(in io.Reader, out io.Writer) int {
	n := 0
	if in != nil {
		n++
	}
	if out != nil {
		n++
	}
	return n
}
