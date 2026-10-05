package complete

import (
	"context"
	"iter"
)

// Kind is what a candidate is. PowerShell receives it as the third field of
// a line and maps it to a CompletionResultType (0008-PLAN P5 deviation 2).
type Kind uint8

// Candidate kinds.
const (
	KindValue Kind = iota
	KindCommand
	KindFlag
	KindFile
	KindDir
)

func (k Kind) String() string {
	switch k {
	case KindCommand:
		return "command"
	case KindFlag:
		return "flag"
	case KindFile:
		return "file"
	case KindDir:
		return "dir"
	}
	return "value"
}

// Candidate is one completion: the value that replaces the current token,
// an optional one-line description, and its kind.
type Candidate struct {
	Value, Description string
	Kind               Kind
}

// Directive tells the shell script how to present candidates. The bits are
// Cobra's, so their meaning is familiar.
type Directive uint8

// Directive bits.
const (
	DirectiveError      Directive = 1
	DirectiveNoSpace    Directive = 2
	DirectiveNoFileComp Directive = 4
	DirectiveFilterDirs Directive = 16
	DirectiveKeepOrder  Directive = 32
)

// Completer produces the values for one source. prefix is the part of the
// value already typed. It must stop when ctx is done.
type Completer interface {
	Complete(ctx context.Context, prefix string) (iter.Seq[Candidate], Directive)
}

// CompleterFunc adapts a function to Completer.
type CompleterFunc func(ctx context.Context, prefix string) (iter.Seq[Candidate], Directive)

// Complete calls f.
func (f CompleterFunc) Complete(ctx context.Context, prefix string) (iter.Seq[Candidate], Directive) {
	return f(ctx, prefix)
}

// Registry maps a complete:"<name>" tag to its source. "none" needs no
// entry: it means no values.
type Registry map[string]Completer
