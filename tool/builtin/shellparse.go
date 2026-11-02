package builtin

import (
	"path"
	"runtime"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// A bash command as the permission rules see it (0005-PLAN F6-3): one
// subject per simple command, at any depth, with its "always" pattern, the
// paths its cd, its file programs and its redirections name, and the files
// it writes, as far as its text names them (0037-MADR D3).

// maxInline is how deep `bash -c '<script>'` is parsed (decision 6;
// grok-build's MAX_INLINE_SHELL_DEPTH).
const maxInline = 8

// shellCommand is a bash command's subjects, each with the pattern an
// "always" saves, "" for its exact text, the literal paths it names, and
// the literal files it writes, unresolved.
type shellCommand struct {
	subjects, saves []string
	paths           []string
	writes          []string
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
					if writesTo(r) {
						sc.writes = append(sc.writes, p)
					}
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
	raw, save := subjectText(src, args[0].Pos(), args, st), arityPattern(words, lits)
	sc.subjects = append(sc.subjects, raw)
	sc.saves = append(sc.saves, save)
	// The command as the shell runs it, so that quoting, a tab, a path or
	// case does not get round a deny rule; with the raw subject's save, so
	// that an "always" saves what it saves for the raw one (0037-MADR D4;
	// 0037-PLAN deviation 3).
	if n := normalised(src, args, words, lits); save != "" && n != raw {
		sc.subjects = append(sc.subjects, n)
		sc.saves = append(sc.saves, save)
	}
	for _, a := range assigns {
		if a.Name != nil && execKey(a.Name.Value) {
			sc.subjects = append(sc.subjects, subjectText(src, assigns[0].Pos(), args, st))
			sc.saves = append(sc.saves, "")
			break
		}
	}
	sc.paths = append(sc.paths, fileOperands(words, lits)...)
	sc.writes = append(sc.writes, writeOperands(words, lits)...)
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

// normalised is a simple command's words as the shell runs them: each
// literal word unquoted, its escapes taken out, any other word as written,
// joined by single spaces; the program by its base name when it is a path,
// and in lower case on darwin and windows, whose volumes find a program
// whatever its case (0037-MADR D4).
func normalised(src string, args []*syntax.Word, words []string, lits []bool) string {
	out := make([]string, len(words))
	for i := range words {
		if lits[i] {
			out[i] = words[i]
		} else {
			out[i] = src[args[i].Pos().Offset():args[i].End().Offset()]
		}
	}
	if strings.Contains(out[0], "/") {
		out[0] = path.Base(out[0])
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		out[0] = strings.ToLower(out[0])
	}
	return strings.Join(out, " ")
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

// writesTo reports whether r writes its file: >, >>, >|, &>, &>> and <>
// (0037-MADR D3).
func writesTo(r *syntax.Redirect) bool {
	switch r.Op {
	case syntax.RdrOut, syntax.AppOut, syntax.RdrClob, syntax.RdrAll, syntax.AppAll, syntax.RdrInOut:
		return true
	}
	return false
}

// writeProgram are the file programs that write their operands
// (0037-MADR D3).
var writeProgram = map[string]bool{}

// inPlaceProgram are the programs that write their file operands when an
// in-place flag is given: -i, -i<suffix> or --in-place (0037-MADR D3).
var inPlaceProgram = map[string]bool{sed: true, "perl": true, "ruby": true}

// sed is the stream editor, whose -f takes a script file and whose -E is
// no script.
const sed = "sed"

// writeOperands are the literal files a simple command writes: a writing
// program's operands, and an in-place editor's file operands, its script
// left out.
func writeOperands(words []string, lits []bool) []string {
	switch {
	case writeProgram[words[0]]:
		return fileOperands(words, lits)
	case inPlaceProgram[words[0]]:
		return inPlaceFiles(words, lits)
	}
	return nil
}

// inPlaceFiles are sed's, perl's or ruby's literal file operands when an
// in-place flag is among its options. The script given by -e (or a bundle
// ending in e or E), --expression, -f or --file is not a file it writes;
// with none of those, its first operand is the script, and is left out.
func inPlaceFiles(words []string, lits []bool) []string {
	inPlace, scripted := false, false
	var operands []int
	options := true
	for i := 1; i < len(words); i++ {
		w := words[i]
		switch {
		case options && w == "--":
			options = false
		case options && lits[i] && (w == "-i" || strings.HasPrefix(w, "-i") || w == "--in-place" || strings.HasPrefix(w, "--in-place=")):
			inPlace = true
		case options && lits[i] && scriptOption(words[0], w):
			scripted = true
			i++ // the script, or its file
		case options && lits[i] && strings.HasPrefix(w, "-"):
			if strings.HasPrefix(w, "--expression=") || strings.HasPrefix(w, "--file=") {
				scripted = true
			}
		default:
			operands = append(operands, i)
		}
	}
	if !inPlace {
		return nil
	}
	if !scripted && len(operands) > 0 {
		operands = operands[1:]
	}
	var out []string
	for _, i := range operands {
		if lits[i] && words[i] != "" {
			out = append(out, words[i])
		}
	}
	return out
}

// scriptOption reports whether option w of program takes the script, or
// its file, as the next word.
func scriptOption(program, w string) bool {
	switch w {
	case "-e", "--expression":
		return true
	case "-f", "--file":
		return program == sed
	}
	if program != sed && !strings.HasPrefix(w, "--") && len(w) > 2 {
		last := w[len(w)-1]
		return last == 'e' || last == 'E'
	}
	return program != sed && w == "-E"
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
	for p := range strings.FieldsSeq(`tee truncate rm rmdir mv cp mkdir touch ln chmod chown`) {
		writeProgram[p] = true
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
