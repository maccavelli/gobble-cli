package builtin

import (
	"encoding/json/v2"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/maccavelli/gobble-cli/tool"
)

// interceptShell is 0012-MADR D10: a bash call whose command is exactly one
// simple command, rm, rmdir, mv, cp or mkdir with only the flags D10 maps,
// is the native tool's call, which the agent runs in its place, keeping its
// id. No pipe, list, redirection, assignment, substitution, glob, brace or
// tilde: each word is plain. A call with a workdir is not intercepted: the
// native tools resolve its paths against the working directory (0005-PLAN
// F6-3 decision 9).
func interceptShell(c tool.Call) (tool.Call, bool) {
	var a shellIn
	if err := json.Unmarshal(c.Args, &a); err != nil || a.Workdir != "" {
		return tool.Call{}, false
	}
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(a.Command), "")
	if err != nil || len(f.Stmts) != 1 {
		return tool.Call{}, false
	}
	st := f.Stmts[0]
	call, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || st.Negated || st.Background || st.Coprocess || len(st.Redirs) > 0 || len(call.Assigns) > 0 {
		return tool.Call{}, false
	}
	words := make([]string, len(call.Args))
	for i, w := range call.Args {
		v, lit := wordValue(w)
		if !lit || v == "" || !plainWord(w) {
			return tool.Call{}, false
		}
		words[i] = v
	}
	var flags, operands []string
	for _, w := range words[1:] {
		switch {
		case len(operands) == 0 && strings.HasPrefix(w, "-"):
			flags = append(flags, w)
		case strings.HasPrefix(w, "-"):
			return tool.Call{}, false // a flag after an operand
		default:
			operands = append(operands, w)
		}
	}
	native, args, ok := nativeForm(words[0], flags, operands)
	if !ok {
		return tool.Call{}, false
	}
	data, err := json.Marshal(args)
	if err != nil {
		return tool.Call{}, false
	}
	return tool.Call{ID: c.ID, Name: native, Args: data}, true
}

// plainWord reports whether w expands to nothing: no unquoted glob, brace
// or tilde in its plain text.
func plainWord(w *syntax.Word) bool {
	for _, part := range w.Parts {
		if l, ok := part.(*syntax.Lit); ok && strings.ContainsAny(l.Value, "*?[{~") {
			return false
		}
	}
	return true
}

// nativeForm is D10's table: the native tool and its arguments for prog
// with exactly these flags and operands, or false.
func nativeForm(prog string, flags, operands []string) (string, map[string]any, bool) {
	one := func(allowed ...string) (string, bool) {
		if len(flags) > 1 || len(flags) == 1 && !slices.Contains(allowed, flags[0]) {
			return "", false
		}
		if len(flags) == 1 {
			return flags[0], true
		}
		return "", true
	}
	switch prog {
	case "rm":
		if f, ok := one("-r", "-R"); ok && len(operands) == 1 {
			args := map[string]any{keyPath: operands[0]}
			if f != "" {
				args["recursive"] = true
			}
			return "delete", args, true
		}
	case "rmdir":
		if len(flags) == 0 && len(operands) == 1 {
			return "delete", map[string]any{keyPath: operands[0]}, true
		}
	case "mv":
		if f, ok := one("-f"); ok && len(operands) == 2 {
			args := map[string]any{"from": operands[0], "to": operands[1]}
			if f != "" {
				args["overwrite"] = true
			}
			return "move", args, true
		}
	case "cp":
		if f, ok := one("-r", "-R", "-f"); ok && len(operands) == 2 {
			args := map[string]any{"from": operands[0], "to": operands[1]}
			if f == "-f" {
				args["overwrite"] = true
			}
			return "copy", args, true
		}
	case nameMkdir:
		if _, ok := one("-p"); ok && len(operands) == 1 {
			return nameMkdir, map[string]any{keyPath: operands[0]}, true
		}
	}
	return "", nil, false
}
