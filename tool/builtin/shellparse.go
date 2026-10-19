package builtin

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// A bash command as the permission rules see it (0005-PLAN F6-3): one
// subject per simple command, at any depth, with its "always" pattern, and
// the paths its cd, its file programs and its redirections name.

// maxInline is how deep `bash -c '<script>'` is parsed (decision 6;
// grok-build's MAX_INLINE_SHELL_DEPTH).
const maxInline = 8

// shellCommand is a bash command's subjects, each with the pattern an
// "always" saves, "" for its exact text, and the literal paths it names,
// unresolved.
type shellCommand struct {
	subjects, saves []string
	paths           []string
}

// parseShell is command's subjects (decisions 2 to 7). A command that does
// not parse, holds no simple command, or runs a program that is not a plain
// word is one raw subject, its whole text, which only a rule matching that
// text decides (decision 3).
func parseShell(command string) shellCommand {
	var sc shellCommand
	if !sc.add(command, 0) {
		return shellCommand{subjects: []string{strings.TrimSpace(command)}, saves: []string{""}}
	}
	return sc
}

// add adds src's simple commands, at depth inline shells deep; false when
// the whole command must be raw.
func (sc *shellCommand) add(src string, depth int) bool {
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(src), "")
	if err != nil {
		return false
	}
	stmts := map[*syntax.CallExpr]*syntax.Stmt{}
	var calls []*syntax.CallExpr
	syntax.Walk(f, func(n syntax.Node) bool {
		switch x := n.(type) {
		case *syntax.Stmt:
			if c, ok := x.Cmd.(*syntax.CallExpr); ok {
				stmts[c] = x
			}
			for _, r := range x.Redirs {
				if p, ok := redirectPath(r); ok {
					sc.paths = append(sc.paths, p)
				}
			}
		case *syntax.CallExpr:
			if len(x.Args) > 0 {
				calls = append(calls, x)
			}
		}
		return true
	})
	if len(calls) == 0 {
		return false
	}
	for _, c := range calls {
		if !sc.call(src, c.Assigns, c.Args, stmts[c], depth) {
			return false
		}
	}
	return true
}

// call adds one simple command, its program args[0]: cd's operands as
// paths; otherwise its subject, a second one as written when an assignment
// changes what runs, its file operands, the command a wrapper runs, and an
// inline shell's script.
func (sc *shellCommand) call(src string, assigns []*syntax.Assign, args []*syntax.Word, st *syntax.Stmt, depth int) bool {
	words, lits := make([]string, len(args)), make([]bool, len(args))
	for i, w := range args {
		words[i], lits[i] = wordValue(w)
	}
	if !lits[0] || strings.ContainsAny(words[0], "*?[") {
		return false // not a plain word: an expansion, or a glob
	}
	if slices.Contains([]string{"cd", "pushd", "chdir"}, words[0]) {
		for i := 1; i < len(words); i++ {
			if !lits[i] {
				return false
			}
			if !strings.HasPrefix(words[i], "-") {
				sc.paths = append(sc.paths, words[i])
			}
		}
		return true
	}
	sc.subjects = append(sc.subjects, subjectText(src, args[0].Pos(), args, st))
	sc.saves = append(sc.saves, arityPattern(words, lits))
	for _, a := range assigns {
		if a.Name != nil && execKey(a.Name.Value) {
			sc.subjects = append(sc.subjects, subjectText(src, assigns[0].Pos(), args, st))
			sc.saves = append(sc.saves, "")
			break
		}
	}
	sc.paths = append(sc.paths, fileOperands(words, lits)...)
	if i, ok := unwrap(words, lits); ok {
		if !sc.call(src, nil, args[i:], st, depth) {
			return false
		}
	} else if i > 0 {
		return false // the program it runs is not a plain word (decision 3)
	}
	if script, inline, literal := inlineScript(words, lits); inline {
		if !literal || depth >= maxInline || !sc.add(script, depth+1) {
			return false
		}
	}
	return true
}

// wordValue is w's value when it is literal: plain text, single quotes
// (not $”), and double quotes holding only text, with backslash escapes
// taken out; ok is false for any expansion.
func wordValue(w *syntax.Word) (string, bool) {
	var b strings.Builder
	for _, part := range w.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			b.WriteString(unescape(p.Value))
		case *syntax.SglQuoted:
			if p.Dollar {
				return "", false
			}
			b.WriteString(p.Value)
		case *syntax.DblQuoted:
			if p.Dollar {
				return "", false
			}
			for _, q := range p.Parts {
				l, ok := q.(*syntax.Lit)
				if !ok {
					return "", false
				}
				b.WriteString(unescape(l.Value))
			}
		default:
			return "", false
		}
	}
	return b.String(), true
}

// unescape takes the backslash out of each escaped character.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// subjectText is a simple command's text from start, a byte offset in src,
// to its last word, with its statement's redirections after start, masked
// as Kilo masks it (decision 5).
func subjectText(src string, start syntax.Pos, args []*syntax.Word, st *syntax.Stmt) string {
	from, to := start.Offset(), args[len(args)-1].End().Offset()
	var inert [][2]uint
	for _, w := range args {
		inert = append(inert, literalSpans(w)...)
	}
	if st != nil {
		for _, r := range st.Redirs {
			if r.Pos().Offset() < from {
				continue
			}
			end := r.End().Offset()
			if r.Hdoc != nil {
				// A heredoc's body follows the line; its newline, body and
				// closing word are inert.
				if body := r.Hdoc.Pos().Offset(); body > 0 && src[body-1] == '\n' {
					inert = append(inert, [2]uint{body - 1, body})
				}
				end = max(end, heredocEnd(src, r.Hdoc.End().Offset()))
				inert = append(inert, [2]uint{r.Hdoc.Pos().Offset(), end})
			} else if inertRedirect(r) {
				inert = append(inert, [2]uint{r.Pos().Offset(), r.End().Offset()})
			} else if r.Word != nil {
				inert = append(inert, literalSpans(r.Word)...)
			}
			to = max(to, end)
		}
	}
	b := []byte(src[from:min(to, uint(len(src)))])
	for _, s := range inert {
		for i := max(s[0], from); i < min(s[1], to); i++ {
			if strings.IndexByte("<>|&;$`\n", b[i-from]) >= 0 {
				b[i-from] = '_'
			}
		}
	}
	return strings.TrimSpace(string(b))
}

// heredocEnd is the end of a heredoc's closing line, which follows its
// body at body.
func heredocEnd(src string, body uint) uint {
	if body >= uint(len(src)) {
		return uint(len(src))
	}
	if i := strings.IndexByte(src[body:], '\n'); i >= 0 {
		return body + uint(i)
	}
	return uint(len(src))
}

// literalSpans are the byte ranges of w's literal parts: plain text and
// quoted strings, where an operator character is inert.
func literalSpans(w *syntax.Word) [][2]uint {
	var out [][2]uint
	for _, part := range w.Parts {
		switch p := part.(type) {
		case *syntax.Lit, *syntax.SglQuoted:
			out = append(out, [2]uint{p.Pos().Offset(), p.End().Offset()})
		case *syntax.DblQuoted:
			if p.Dollar {
				out = append(out, [2]uint{p.Pos().Offset(), p.Pos().Offset() + 1})
			}
			for _, q := range p.Parts {
				if l, ok := q.(*syntax.Lit); ok {
					out = append(out, [2]uint{l.Pos().Offset(), l.End().Offset()})
				}
			}
		}
	}
	return out
}

// sinks are the redirection targets that write nowhere a rule cares about
// (decision 5).
var sinks = []string{"/dev/null", "/dev/stdout", "/dev/stderr"}

// inertRedirect reports whether r sends to or reads from a sink, or
// duplicates or closes a file descriptor.
func inertRedirect(r *syntax.Redirect) bool {
	if r.Word == nil {
		return false
	}
	target, ok := wordValue(r.Word)
	if !ok {
		return false
	}
	switch r.Op {
	case syntax.RdrOut, syntax.AppOut, syntax.RdrClob, syntax.RdrAll, syntax.AppAll, syntax.RdrIn:
		return slices.Contains(sinks, target)
	case syntax.DplIn, syntax.DplOut:
		return target == "-" || strings.Trim(target, "0123456789") == ""
	}
	return false
}

// redirectPath is the literal file a redirection reads or writes, other
// than a sink (decision 7).
func redirectPath(r *syntax.Redirect) (string, bool) {
	switch r.Op {
	case syntax.RdrOut, syntax.AppOut, syntax.RdrClob, syntax.RdrAll, syntax.AppAll, syntax.RdrIn, syntax.RdrInOut:
	default:
		return "", false
	}
	p, ok := wordValue(r.Word)
	if !ok || p == "" || slices.Contains(sinks, p) {
		return "", false
	}
	return p, true
}

// fileProgram are the programs whose operands are files (decision 7):
// grok-build's readers and writers (shell_access.rs:636-650), and the file
// commands.
var fileProgram = map[string]bool{}

func init() {
	for p := range strings.FieldsSeq(`cat tac nl head tail grep egrep fgrep rg sed awk less more bat strings xxd od hexdump
		base64 base32 cut sort uniq wc diff comm rev jq yq ag ack zcat zless zmore zgrep zegrep zfgrep bzcat bzgrep xzcat
		xzgrep zstdcat lz4cat tee truncate rm rmdir mv cp mkdir touch ln chmod chown`) {
		fileProgram[p] = true
	}
}

// fileOperands are a file program's literal operands: every word after the
// program that is not an option, and every word after "--".
func fileOperands(words []string, lits []bool) []string {
	if !fileProgram[words[0]] {
		return nil
	}
	var out []string
	options := true
	for i := 1; i < len(words); i++ {
		switch {
		case options && words[i] == "--":
			options = false
		case options && strings.HasPrefix(words[i], "-"):
		case lits[i] && words[i] != "":
			out = append(out, words[i])
		}
	}
	return out
}

// execKey reports whether an assignment to name changes what a command
// runs (decision 3; grok-build's env_risk.rs:4-38).
func execKey(name string) bool {
	switch name {
	case "LD_PRELOAD", "LD_AUDIT", "BASH_ENV", "ENV", "IFS", "PATH", "PROMPT_COMMAND", "GIT_EXTERNAL_DIFF", "GIT_PROXY_COMMAND":
		return true
	}
	return strings.HasPrefix(name, "DYLD_") || strings.HasPrefix(name, "GIT_CONFIG")
}

// unwrap is the index of the program a wrapper runs, when the wrapper's
// own options can be told apart (decision 6). When that word is there and
// is not literal, it gives its index and false: the call is raw.
func unwrap(words []string, lits []bool) (int, bool) {
	n := len(words)
	operand := func(i int) (int, bool) {
		switch {
		case i < n && !lits[i]:
			return i, false
		case i < n && !strings.HasPrefix(words[i], "-"):
			return i, true
		}
		return 0, false
	}
	switch words[0] {
	case "command", "exec", "builtin", "nohup", "sudo", "doas":
		i := 1
		if i < n && words[i] == "--" {
			i++
		}
		return operand(i)
	case "time":
		i := 1
		if i < n && words[i] == "-p" {
			i++
		}
		return operand(i)
	case "nice":
		i := 1
		if i+1 < n && words[i] == "-n" {
			i += 2
		}
		return operand(i)
	case "timeout":
		if i, ok := operand(1); !ok {
			return i, false
		}
		return operand(2)
	case "env":
		for i := 1; i < n; i++ {
			switch w := words[i]; {
			case !lits[i]:
				return i, false
			case w == "-i" || w == "--":
			case strings.HasPrefix(w, "-"):
				return 0, false
			case strings.Contains(w, "="):
			default:
				return i, true
			}
		}
	}
	return 0, false
}

// inlineScript is the script of `bash|sh|zsh [options] -c SCRIPT`: inline
// is true for such a call, literal when the script is a plain word.
func inlineScript(words []string, lits []bool) (script string, inline, literal bool) {
	if !slices.Contains([]string{nameBash, "sh", "zsh"}, words[0]) {
		return "", false, false
	}
	for i := 1; i < len(words); i++ {
		w := words[i]
		if !strings.HasPrefix(w, "-") || strings.HasPrefix(w, "--") {
			if strings.HasPrefix(w, "--") {
				continue
			}
			return "", false, false
		}
		if strings.Contains(w, "c") {
			if i+1 >= len(words) {
				return "", true, false
			}
			return words[i+1], true, lits[i+1]
		}
	}
	return "", false, false
}

// arityPattern is the pattern an "always" saves for a simple command
// (decision 4a): its program and as many words after it as opencode's
// arity table gives its longest known prefix, then " *"; "" when a word of
// that prefix is not literal, so the exact text is saved.
func arityPattern(words []string, lits []bool) string {
	n := 1
	for length := min(len(words), maxArityWords); length > 0; length-- {
		if !slices.Equal(lits[:length], slices.Repeat([]bool{true}, length)) {
			continue
		}
		if a, ok := arity[strings.Join(words[:length], " ")]; ok {
			n = a
			break
		}
	}
	n = min(n, len(words))
	for i := range n {
		if !lits[i] {
			return ""
		}
	}
	return strings.Join(words[:n], " ") + " *"
}
