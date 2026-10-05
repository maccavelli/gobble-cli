package complete

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/kong"
)

// Target is what the cursor is completing, found by Resolve.
type Target struct {
	// Node is the command the words reached.
	Node *kong.Node
	// Flag, when set, is the flag whose value is being completed.
	Flag *kong.Flag
	// Arg, when set, is the positional argument being completed.
	Arg *kong.Value
	// Names reports that command names (Commands) and flag names (Flags)
	// may be offered.
	Commands, Flags bool
	// Prefix is the typed part of what is being completed.
	Prefix string
	// Insert is kept in front of every candidate value, so a candidate
	// replaces the whole current token: "--model=" or "-t".
	Insert string
}

// Resolve finds what the last word, the one at the cursor, completes. words
// are the command line after the program name, cut at the cursor; the last
// may be empty. It follows Kong's parsing over the model: children and
// aliases, the withargs default command (Kong context.go:399-403 and
// :569-576), long and short flags, aliases and negations, --name=value,
// short clusters, "--" and passthrough. It never calls Node.AllFlags or
// kong.Trace, which change the model as they run.
func Resolve(app *kong.Application, words []string) Target {
	if len(words) == 0 {
		words = []string{""}
	}
	w := walker{node: app.Node}
	for _, word := range words[:len(words)-1] {
		w.step(word)
	}
	return w.target(words[len(words)-1])
}

type walker struct {
	node       *kong.Node
	positional int
	pending    *kong.Flag // a flag that still needs its value
	rest       bool       // after "--" or a passthrough: only positionals
}

func (w *walker) step(word string) {
	switch {
	case w.pending != nil:
		w.pending = nil
	case w.rest:
		w.positional++
	case word == "--":
		w.rest = true
	case strings.HasPrefix(word, "--"):
		name, _, inline := strings.Cut(word[2:], "=")
		if f := longFlag(w.node, name); f != nil && takesValue(f) && !inline {
			w.pending = f
		}
	case len(word) > 1 && word[0] == '-':
		f, rest := shortCluster(w.node, word[1:])
		if f != nil && rest == "" {
			w.pending = f
		}
	default:
		w.argument(word)
	}
}

// argument consumes a word that is not a flag: a child command, the
// withargs default command's first argument, or a positional.
func (w *walker) argument(word string) {
	if w.positional == 0 {
		if c := child(w.node, word); c != nil {
			w.enter(c)
			return
		}
		if d := withArgs(w.node); d != nil && len(w.node.Positional) == 0 {
			w.enter(d)
		}
	}
	if p := w.positionalValue(); p != nil && p.PassthroughMode != kong.PassThroughModeNone {
		w.rest = true
	}
	w.positional++
}

func (w *walker) enter(n *kong.Node) {
	w.node, w.positional = n, 0
	if n.Passthrough {
		w.rest = true
	}
}

// positionalValue is the positional the next argument fills, or nil.
func (w *walker) positionalValue() *kong.Value {
	ps := w.node.Positional
	switch {
	case len(ps) == 0:
		return nil
	case w.positional < len(ps):
		return ps[w.positional]
	case ps[len(ps)-1].IsCumulative():
		return ps[len(ps)-1]
	}
	return nil
}

func (w *walker) target(cur string) Target {
	t := Target{Node: w.node, Prefix: cur}
	switch {
	case w.pending != nil:
		t.Flag = w.pending
		return t
	case w.rest:
		t.Arg = w.argForCompletion()
		return t
	case strings.HasPrefix(cur, "--"):
		if name, value, inline := strings.Cut(cur[2:], "="); inline {
			if f := longFlag(w.node, name); f != nil && takesValue(f) {
				t.Flag, t.Prefix, t.Insert = f, value, "--"+name+"="
			}
			return t
		}
		t.Flags = true
		return t
	case len(cur) > 1 && cur[0] == '-':
		// A short cluster with a value already started: -tbash.
		for i, r := range cur[1:] {
			f := shortFlag(w.node, r)
			if f == nil {
				break
			}
			if takesValue(f) {
				if at := 1 + i + utf8.RuneLen(r); at < len(cur) {
					t.Flag, t.Prefix, t.Insert = f, cur[at:], cur[:at]
					return t
				}
				break
			}
		}
		t.Flags = true
		return t
	case cur == "-":
		t.Flags = true
		return t
	}
	t.Commands = w.positional == 0 && len(w.node.Positional) == 0
	t.Arg = w.argForCompletion()
	return t
}

// argForCompletion is the positional at the cursor, looking through a
// withargs default command at the root of a command with no positionals.
func (w *walker) argForCompletion() *kong.Value {
	if p := w.positionalValue(); p != nil {
		return p
	}
	if d := withArgs(w.node); d != nil && w.positional == 0 && len(d.Positional) > 0 {
		return d.Positional[0]
	}
	return nil
}

// withArgs is node's default command when it accepts arguments and flags
// without being named (Kong's default:"withargs").
func withArgs(node *kong.Node) *kong.Node {
	if d := node.DefaultCmd; d != nil && d.Tag != nil && d.Tag.Default == "withargs" {
		return d
	}
	return nil
}

func child(node *kong.Node, name string) *kong.Node {
	for _, c := range node.Children {
		if c.Type != kong.CommandNode {
			continue
		}
		if c.Name == name || slices.Contains(c.Aliases, name) {
			return c
		}
	}
	return nil
}

// flagsFor lists the flags valid at node: its own, its ancestors', and, as
// Kong allows, its withargs default command's.
func flagsFor(node *kong.Node) []*kong.Flag {
	var out []*kong.Flag
	if d := withArgs(node); d != nil {
		out = append(out, d.Flags...)
	}
	for n := node; n != nil; n = n.Parent {
		out = append(out, n.Flags...)
	}
	return out
}

func longFlag(node *kong.Node, name string) *kong.Flag {
	for _, f := range flagsFor(node) {
		if f.Name == name || negatedName(f) == "--"+name {
			return f
		}
		for _, a := range f.Aliases {
			if utf8.RuneCountInString(a) > 1 && a == name {
				return f
			}
		}
	}
	return nil
}

func shortFlag(node *kong.Node, r rune) *kong.Flag {
	for _, f := range flagsFor(node) {
		if f.Short == r || slices.Contains(f.Aliases, string(r)) {
			return f
		}
	}
	return nil
}

// shortCluster walks -abc: it returns the first flag that takes a value,
// and what follows it in the cluster (its inline value, or "").
func shortCluster(node *kong.Node, cluster string) (*kong.Flag, string) {
	for i, r := range cluster {
		f := shortFlag(node, r)
		if f == nil {
			return nil, ""
		}
		if takesValue(f) {
			return f, cluster[i+utf8.RuneLen(r):]
		}
	}
	return nil, ""
}

func takesValue(f *kong.Flag) bool { return !f.IsBool() && !f.IsCounter() }

// negatedName is the flag's negation, or "". It is copied from Kong's
// unexported negatableFlagName (negatable.go, v1.16.1, MIT): "_" is Kong's
// placeholder for the default --no-<name>.
func negatedName(f *kong.Flag) string {
	if f.Tag == nil {
		return ""
	}
	switch f.Tag.Negatable {
	case "":
		return ""
	case "_":
		return "--no-" + f.Name
	default:
		return "--" + f.Tag.Negatable
	}
}
