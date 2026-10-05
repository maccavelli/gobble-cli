package complete

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alecthomas/kong"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// Protocol is the version the scripts and the binary must agree on.
const Protocol = "1"

const (
	deadline        = 150 * time.Millisecond
	maxDescription  = 80
	wordSep         = "\x1f" // PowerShell joins words with U+001F
	shellPowerShell = "powershell"
)

// The completion-mode environment (0008-MADR D17).
const (
	EnvShell    = "GOBBLE_COMPLETE"
	EnvProtocol = "GOBBLE_COMPLETE_PROTOCOL"
	EnvLine     = "GOBBLE_COMPLETE_LINE"
	EnvPoint    = "GOBBLE_COMPLETE_POINT"
	EnvCur      = "GOBBLE_COMPLETE_CUR"
	EnvWords    = "GOBBLE_COMPLETE_WORDS"
	EnvDebug    = "GOBBLE_COMPLETE_DEBUG"
)

// EnvNames lists every completion variable, so cli.Main can remove them
// before any child process starts.
var EnvNames = []string{EnvShell, EnvProtocol, EnvLine, EnvPoint, EnvCur, EnvWords, EnvDebug}

// Run answers one completion request and returns the exit status, which is
// always 0. args are the command line after the program name. Output is
// built in a buffer and written once: candidate lines, then :<directive>.
// A protocol mismatch or an unknown shell writes only ":1".
func Run(ctx context.Context, app *kong.Application, reg Registry, env func(string) string, args []string, out io.Writer) int {
	start := time.Now()
	shell := env(EnvShell)
	var b bytes.Buffer
	words, strip, ok := wordsFor(shell, env, args)
	if env(EnvProtocol) != Protocol || !ok {
		b.WriteString(":1\n")
		finish(out, &b, env, fmt.Sprintf("refused shell=%q protocol=%q", shell, env(EnvProtocol)))
		return 0
	}
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	t := Resolve(app, words)
	seq, d := Candidates(ctx, t, reg)
	var list []Candidate
	for c := range seq {
		if ctx.Err() != nil {
			break
		}
		list = append(list, c)
	}
	if ctx.Err() != nil {
		d |= DirectiveKeepOrder // the deadline passed: return what there is, as found
	}
	// Paths want no space after a directory, but a space after a file. When
	// one file is all that is left, the shell may add its space.
	if len(list) == 1 && list[0].Kind == KindFile {
		d &^= DirectiveNoSpace
	}
	if d&DirectiveKeepOrder == 0 {
		slices.SortStableFunc(list, func(a, b Candidate) int {
			return cmp.Or(cmp.Compare(a.Kind, b.Kind), strings.Compare(a.Value, b.Value))
		})
	}
	for _, c := range list {
		writeLine(&b, shell, c, strip)
	}
	fmt.Fprintf(&b, ":%d\n", d)
	finish(out, &b, env, fmt.Sprintf("shell=%s words=%q node=%s flag=%t arg=%t n=%d directive=%d elapsed=%s",
		shell, words, t.Node.Name, t.Flag != nil, t.Arg != nil, len(list), d, time.Since(start)))
	return 0
}

// wordsFor returns the words after the program name, the last one at the
// cursor. strip is the part of that word bash already shows before its
// own word break (= or :), removed from every candidate.
func wordsFor(shell string, env func(string) string, args []string) (words []string, strip string, ok bool) {
	switch shell {
	case "zsh", "fish":
		if i := slices.Index(args, "--"); i >= 0 {
			args = args[i+1:]
		}
		words = slices.Clone(args)
	case "bash":
		point, err := strconv.Atoi(env(EnvPoint))
		if err != nil {
			point = -1
		}
		var cur string
		words, cur = SplitLine(env(EnvLine), point)
		if len(words) > 0 {
			words = words[1:] // the program name
		}
		words = append(words, cur)
		if bc := env(EnvCur); bc != cur && strings.HasSuffix(cur, bc) {
			strip = cur[:len(cur)-len(bc)]
		}
	case shellPowerShell:
		words = strings.Split(env(EnvWords), wordSep)
	default:
		return nil, "", false
	}
	if len(words) == 0 {
		words = []string{""}
	}
	return words, strip, true
}

func writeLine(b *bytes.Buffer, shell string, c Candidate, strip string) {
	v := term.Sanitize(c.Value)
	if v == "" || strings.ContainsAny(v, "\t\n") {
		return
	}
	if strip != "" {
		var ok bool
		if v, ok = strings.CutPrefix(v, strip); !ok {
			return
		}
	}
	desc := oneLine(term.Sanitize(c.Description))
	switch {
	case shell == shellPowerShell:
		b.WriteString(v + "\t" + desc + "\t" + c.Kind.String() + "\n")
	case desc != "":
		b.WriteString(v + "\t" + desc + "\n")
	default:
		b.WriteString(v + "\n")
	}
}

// oneLine folds whitespace and caps the result at maxDescription runes.
// Values are never cut: a cut value completes to a token that does not
// exist (0008-PLAN P5 deviation 3).
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= maxDescription {
		return s
	}
	r := []rune(s)
	return string(r[:maxDescription-1]) + "…"
}

// finish writes the buffer once, then the debug line when
// GOBBLE_COMPLETE_DEBUG names a file. stderr is never written.
func finish(out io.Writer, b *bytes.Buffer, env func(string) string, debug string) {
	_, werr := out.Write(b.Bytes())
	path := env(EnvDebug)
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec // G304: the user names the debug file
	if err != nil {
		return
	}
	defer f.Close()                                                                          //nolint:errcheck // best-effort debug output
	fmt.Fprintf(f, "%s %s write_err=%v\n", time.Now().Format(time.RFC3339Nano), debug, werr) //nolint:errcheck // best-effort debug output
}

// Candidates lists what t completes, and the directive for the shell.
// Every directive carries NoFileComp: gobble completes paths itself, and the
// scripts never fall back to the shell's file completion.
func Candidates(ctx context.Context, t Target, reg Registry) (iter.Seq[Candidate], Directive) {
	d := DirectiveNoFileComp
	var seqs []iter.Seq[Candidate]
	if t.Commands {
		seqs = append(seqs, commandNames(t.Node))
	}
	if t.Flags {
		seqs = append(seqs, flagNames(t.Node))
	}
	insert, prefix := t.Insert, t.Prefix
	v := t.Arg
	if t.Flag != nil {
		v = t.Flag.Value
	}
	if v != nil {
		if sep := listSep(v); sep != 0 {
			if i := strings.LastIndex(prefix, string(sep)); i >= 0 {
				insert, prefix = insert+prefix[:i+1], prefix[i+1:]
			}
			d |= DirectiveNoSpace
		}
		if src := sourceFor(v, reg); src != nil {
			s, sd := src.Complete(ctx, prefix)
			d |= sd
			seqs = append(seqs, prefixed(s, insert))
		}
	}
	want := insert + prefix
	if v == nil {
		want = t.Prefix
	}
	return func(yield func(Candidate) bool) {
		for _, s := range seqs {
			for c := range s {
				// Path sources match names themselves (case-insensitively
				// on Windows), so only other kinds are filtered here.
				if c.Kind != KindFile && c.Kind != KindDir && !strings.HasPrefix(c.Value, want) {
					continue
				}
				if !yield(c) {
					return
				}
			}
		}
	}, d
}

func prefixed(s iter.Seq[Candidate], insert string) iter.Seq[Candidate] {
	if insert == "" {
		return s
	}
	return func(yield func(Candidate) bool) {
		for c := range s {
			c.Value = insert + c.Value
			if !yield(c) {
				return
			}
		}
	}
}

// sourceFor is the value's completer: its complete:"<name>" tag, else its
// enum. complete:"none", and a value with neither, complete nothing.
func sourceFor(v *kong.Value, reg Registry) Completer {
	if v.Tag != nil {
		switch name := v.Tag.Get("complete"); name {
		case "none":
			return nil
		case "":
		default:
			return reg[name]
		}
	}
	if v.Enum != "" {
		return EnumSource(v.EnumSlice())
	}
	return nil
}

// listSep is the separator of a list flag such as --tools a,b, or 0. Kong
// gives every slice a default separator, but splits only flag values, so a
// positional is never a list.
func listSep(v *kong.Value) rune {
	if v.Flag != nil && v.IsSlice() && v.Tag != nil && v.Tag.Sep > 0 {
		return v.Tag.Sep
	}
	return 0
}

func commandNames(node *kong.Node) iter.Seq[Candidate] {
	return func(yield func(Candidate) bool) {
		for _, c := range node.Children {
			if c.Type == kong.CommandNode && !c.Hidden && !yield(Candidate{Value: c.Name, Description: c.Help, Kind: KindCommand}) {
				return
			}
		}
	}
}

func flagNames(node *kong.Node) iter.Seq[Candidate] {
	return func(yield func(Candidate) bool) {
		seen := map[string]bool{}
		emit := func(v, help string) bool {
			if seen[v] {
				return true
			}
			seen[v] = true
			return yield(Candidate{Value: v, Description: help, Kind: KindFlag})
		}
		for _, f := range flagsFor(node) {
			if f.Hidden {
				continue
			}
			names := []string{"--" + f.Name}
			if n := negatedName(f); n != "" {
				names = append(names, n)
			}
			for _, a := range f.Aliases {
				if utf8.RuneCountInString(a) > 1 {
					names = append(names, "--"+a)
				} else {
					names = append(names, "-"+a)
				}
			}
			if f.Short != 0 {
				names = append(names, "-"+string(f.Short))
			}
			for _, n := range names {
				if !emit(n, f.Help) {
					return
				}
			}
		}
	}
}
