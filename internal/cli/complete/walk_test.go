package complete

import (
	"context"
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// testRoot exercises every rule of the walker in one grammar shaped like
// gobble's: a withargs default command, aliases, hidden items, negations,
// list flags and a passthrough command.
type testRoot struct {
	Chat struct {
		Print    bool     `short:"p"`
		Format   string   `name:"output-format" enum:"text,json,stream-json" default:"text"`
		Model    string   `complete:"model"`
		Continue bool     `short:"c"`
		Name     string   `short:"n" complete:"none"`
		Approve  bool     `negatable:""`
		Quiet    bool     `negatable:"loud"`
		Tools    []string `short:"t" sep:"," complete:"tools"`
		Secret   string   `hidden:"" complete:"none"`
		Words    []string `arg:"" optional:"" complete:"none"`
	} `cmd:"" default:"withargs"`
	Version struct {
		JSON bool `name:"json"`
	} `cmd:"" aliases:"ver"`
	Hidden struct{} `cmd:"" hidden:""`
	Run    struct {
		Args []string `arg:"" optional:"" passthrough:""`
	} `cmd:""`
	Shell struct {
		Kind string `arg:"" enum:"bash,zsh"`
	} `cmd:""`
	Level string `name:"log-level" enum:"debug,info" default:"info"`
}

func newTestParser(t *testing.T) *kong.Kong {
	t.Helper()
	var root testRoot
	k, err := kong.New(&root, kong.Name("gobble"), kong.Exit(func(int) {}))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func listOf(values ...string) Lister {
	return func(context.Context) iter.Seq[Candidate] {
		return func(yield func(Candidate) bool) {
			for _, v := range values {
				if !yield(Candidate{Value: v}) {
					return
				}
			}
		}
	}
}

var testRegistry = Registry{
	"model": ModelSource(listOf("anthropic/claude", "openai/gpt"), []string{"low", "high"}),
	"tools": ListSource(listOf("bash", "read", "write"), false),
}

func values(t *testing.T, k *kong.Kong, words []string) ([]string, Directive) {
	t.Helper()
	seq, d := Candidates(t.Context(), Resolve(k.Model, words), testRegistry)
	var out []string
	for c := range seq {
		out = append(out, c.Value)
	}
	return out, d
}

func TestEngine(t *testing.T) {
	cases := []struct {
		name  string
		words []string
		want  []string
		dir   Directive // bits that must be set
		// oracle is the command kong.Trace reaches for the words before
		// the cursor; "" skips the oracle (an incomplete parse).
		oracle string
	}{
		{"root lists visible commands", []string{""}, []string{"chat", "version", "run", "shell"}, DirectiveNoFileComp, "chat"},
		{"root flags include the withargs command's (Kong rule 1)", []string{"--out"}, []string{"--output-format"}, 0, "chat"},
		{"--name=value replaces the whole token", []string{"--model=an"}, []string{"--model=anthropic/claude"}, 0, "chat"},
		{"model thinking level after a colon", []string{"--model", "openai/gpt:h"}, []string{"openai/gpt:high"}, DirectiveKeepOrder, "gobble"},
		{"short cluster ending in a value flag", []string{"-cn", ""}, nil, 0, ""},
		{"short cluster with an inline value", []string{"-tre"}, []string{"-tread"}, 0, "chat"},
		{"default negation", []string{"--no-ap"}, []string{"--no-approve"}, 0, "chat"},
		{"custom negation", []string{"--lo"}, []string{"--log-level", "--loud"}, 0, "chat"},
		{"after -- only positionals", []string{"--", ""}, nil, DirectiveNoFileComp, ""},
		{"alias reaches its command", []string{"ver", "--"}, []string{"--help", "--json", "--log-level"}, 0, "version"},
		{"hidden flag is not offered", []string{"--sec"}, nil, 0, "chat"},
		{"hidden flag still takes its value", []string{"--secret", "x", "--out"}, []string{"--output-format"}, 0, ""},
		{"hidden command is not offered", []string{"hid"}, nil, 0, "chat"},
		{"cut at the cursor", []string{"--output-format", "js"}, []string{"json"}, DirectiveKeepOrder, ""},
		{"list flag completes after the last comma", []string{"--tools", "bash,w"}, []string{"bash,write"}, DirectiveNoSpace, ""},
		{"enum positional", []string{"shell", ""}, []string{"bash", "zsh"}, DirectiveKeepOrder, "shell"},
		{"a word enters the withargs command (Kong rule 2)", []string{"hello", "--out"}, []string{"--output-format"}, 0, "chat"},
		{"passthrough takes every word as an argument", []string{"run", "--output-format", ""}, nil, 0, "run"},
		{"value flag's word is consumed", []string{"--model", "x", ""}, []string{"chat", "version", "run", "shell"}, 0, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k := newTestParser(t)
			got, d := values(t, k, tc.words)
			if !sameSet(got, tc.want) {
				t.Fatalf("candidates = %q, want %q", got, tc.want)
			}
			if d&tc.dir != tc.dir {
				t.Fatalf("directive %d lacks bits %d", d, tc.dir)
			}
			if tc.oracle == "" {
				return
			}
			oracle := newTestParser(t)
			kctx, err := kong.Trace(oracle, tc.words[:len(tc.words)-1])
			if err != nil {
				t.Fatalf("kong.Trace: %v", err)
			}
			want := "gobble"
			if sel := kctx.Selected(); sel != nil {
				want = sel.Name
			}
			if want != tc.oracle {
				t.Fatalf("test oracle is wrong: kong.Trace reached %q", want)
			}
			// With nothing named, Kong selects the default command at the
			// end of the parse (maybeSelectDefault); Resolve stays at the
			// root, where commands can still be offered. The two agree only
			// when every word before the cursor is a flag: a plain word must
			// enter the default command (Kong rule 2).
			node := Resolve(k.Model, tc.words).Node
			onlyFlags := !slices.ContainsFunc(tc.words[:len(tc.words)-1], func(w string) bool { return !strings.HasPrefix(w, "-") })
			if node.Name != want && (!onlyFlags || node != k.Model.Node || node.DefaultCmd == nil || node.DefaultCmd.Name != want) {
				t.Fatalf("Resolve reached %q, kong.Trace reached %q", node.Name, want)
			}
		})
	}
}

func sameSet(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}
