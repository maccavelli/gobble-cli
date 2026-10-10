package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Pi's substituteArgs cases with literal arguments, each row its test's
// name, Pi's expected value, and its line in Pi's
// packages/coding-agent/test/prompt-templates.test.ts.
func TestSubstitute(t *testing.T) {
	for _, c := range []struct {
		name, content string
		args          []string
		want          string
	}{
		{"should replace $ARGUMENTS with all args joined", "Test: $ARGUMENTS", []string{"a", "b", "c"}, "Test: a b c"},                                                            // :29
		{"should replace $@ with all args joined", "Test: $@", []string{"a", "b", "c"}, "Test: a b c"},                                                                            // :33
		{"should NOT recursively substitute patterns in argument values", "$ARGUMENTS", []string{"$1", "$ARGUMENTS"}, "$1 $ARGUMENTS"},                                            // :43
		{"should NOT recursively substitute patterns in argument values", "$@", []string{"$100", "$1"}, "$100 $1"},                                                                // :44
		{"should NOT recursively substitute patterns in argument values", "$ARGUMENTS", []string{"$100", "$1"}, "$100 $1"},                                                        // :45
		{"should support mixed $1, $2, and $ARGUMENTS", "$1: $ARGUMENTS", []string{"prefix", "a", "b"}, "prefix: prefix a b"},                                                     // :49
		{"should support mixed $1, $2, and $@", "$1: $@", []string{"prefix", "a", "b"}, "prefix: prefix a b"},                                                                     // :53
		{"should handle empty arguments array with $ARGUMENTS", "Test: $ARGUMENTS", []string{}, "Test: "},                                                                         // :57
		{"should handle empty arguments array with $@", "Test: $@", []string{}, "Test: "},                                                                                         // :61
		{"should handle empty arguments array with $1", "Test: $1", []string{}, "Test: "},                                                                                         // :65
		{"should handle multiple occurrences of $ARGUMENTS", "$ARGUMENTS and $ARGUMENTS", []string{"a", "b"}, "a b and a b"},                                                      // :69
		{"should handle multiple occurrences of $@", "$@ and $@", []string{"a", "b"}, "a b and a b"},                                                                              // :73
		{"should handle mixed occurrences of $@ and $ARGUMENTS", "$@ and $ARGUMENTS", []string{"a", "b"}, "a b and a b"},                                                          // :77
		{"should handle special characters in arguments", "$1 $2: $ARGUMENTS", []string{"arg100", "@user"}, "arg100 @user: arg100 @user"},                                         // :82
		{"should handle out-of-range numbered placeholders", "$1 $2 $3 $4 $5", []string{"a", "b"}, "a b   "},                                                                      // :87
		{"should handle unicode characters", "$ARGUMENTS", []string{"\u65e5\u672c\u8a9e", "\U0001f389", "caf\u00e9"}, "\u65e5\u672c\u8a9e \U0001f389 caf\u00e9"},                  // :91
		{"should preserve newlines and tabs in argument values", "$1 $2", []string{"line1\nline2", "tab\tthere"}, "line1\nline2 tab\tthere"},                                      // :95
		{"should handle consecutive dollar patterns", "$1$2", []string{"a", "b"}, "ab"},                                                                                           // :99
		{"should handle quoted arguments with spaces", "$ARGUMENTS", []string{"first arg", "second arg"}, "first arg second arg"},                                                 // :103
		{"should handle single argument with $ARGUMENTS", "Test: $ARGUMENTS", []string{"only"}, "Test: only"},                                                                     // :107
		{"should handle single argument with $@", "Test: $@", []string{"only"}, "Test: only"},                                                                                     // :111
		{"should handle $0 (zero index)", "$0", []string{"a", "b"}, ""},                                                                                                           // :115
		{"should handle decimal number in pattern (only integer part matches)", "$1.5", []string{"a"}, "a.5"},                                                                     // :119
		{"should handle $ARGUMENTS as part of word", "pre$ARGUMENTS", []string{"a", "b"}, "prea b"},                                                                               // :123
		{"should handle $@ as part of word", "pre$@", []string{"a", "b"}, "prea b"},                                                                                               // :127
		{"should handle empty arguments in middle of list", "$ARGUMENTS", []string{"a", "", "c"}, "a  c"},                                                                         // :131
		{"should handle trailing and leading spaces in arguments", "$ARGUMENTS", []string{"  leading  ", "trailing  "}, "  leading   trailing  "},                                 // :135
		{"should handle argument containing pattern partially", "Prefix $ARGUMENTS suffix", []string{"ARGUMENTS"}, "Prefix ARGUMENTS suffix"},                                     // :139
		{"should handle non-matching patterns", "$A $$ $ $ARGS", []string{"a"}, "$A $$ $ $ARGS"},                                                                                  // :143
		{"should handle case variations (case-sensitive)", "$arguments $Arguments $ARGUMENTS", []string{"a", "b"}, "$arguments $Arguments a b"},                                   // :147
		{"should handle numbered placeholders with single digit", "$1 $2 $3", []string{"a", "b", "c"}, "a b c"},                                                                   // :165
		{"should handle escaped dollar signs (literal backslash preserved)", "Price: \\$100", []string{}, "Price: \\"},                                                            // :175
		{"should handle mixed numbered and wildcard placeholders", "$1: $@ ($ARGUMENTS)", []string{"first", "second", "third"}, "first: first second third (first second third)"}, // :179
		{"should handle command with no placeholders", "Just plain text", []string{"a", "b"}, "Just plain text"},                                                                  // :185
		{"should handle command with only placeholders", "$1 $2 $@", []string{"a", "b", "c"}, "a b a b c"},                                                                        // :189
		{"should use default when positional arg is missing", "List exactly ${1:-7} next steps", []string{}, "List exactly 7 next steps"},                                         // :199
		{"should use positional arg when present", "List exactly ${1:-7} next steps", []string{"3"}, "List exactly 3 next steps"},                                                 // :212
		{"should use default when positional arg is empty", "Mode: ${1:-brief}", []string{""}, "Mode: brief"},                                                                     // :216
		{"should support multiple positional defaults", "${1:-7} ${2:-brief}", []string{}, "7 brief"},                                                                             // :220
		{"should support multiple positional defaults", "${1:-7} ${2:-brief}", []string{"3"}, "3 brief"},                                                                          // :221
		{"should support multiple positional defaults", "${1:-7} ${2:-brief}", []string{"3", "verbose"}, "3 verbose"},                                                             // :222
		{"should not recursively substitute patterns in arg values", "${1:-7}", []string{"$ARGUMENTS"}, "$ARGUMENTS"},                                                             // :226
		{"should not recursively substitute patterns in arg values", "${1:-7}", []string{"$1"}, "$1"},                                                                             // :227
		{"should not recursively substitute patterns in default values", "${1:-$ARGUMENTS}", []string{"a", "b"}, "a"},                                                             // :231
		{"should not recursively substitute patterns in default values", "${3:-$ARGUMENTS}", []string{"a", "b"}, "$ARGUMENTS"},                                                    // :232
		{"should support defaults with spaces", "${1:-seven steps}", []string{}, "seven steps"},                                                                                   // :236
		{"should support out-of-range positional defaults", "${3:-fallback}", []string{"a", "b"}, "fallback"},                                                                     // :240
		{"should mix positional defaults with existing placeholders", "$1 ${2:-x} $ARGUMENTS", []string{"a"}, "a x a"},                                                            // :244
		{"should slice from index (${@:N})", "${@:2}", []string{"a", "b", "c", "d"}, "b c d"},                                                                                     // :254
		{"should slice from index (${@:N})", "${@:1}", []string{"a", "b", "c"}, "a b c"},                                                                                          // :255
		{"should slice from index (${@:N})", "${@:3}", []string{"a", "b", "c", "d"}, "c d"},                                                                                       // :256
		{"should slice with length (${@:N:L})", "${@:2:2}", []string{"a", "b", "c", "d"}, "b c"},                                                                                  // :260
		{"should slice with length (${@:N:L})", "${@:1:1}", []string{"a", "b", "c"}, "a"},                                                                                         // :261
		{"should slice with length (${@:N:L})", "${@:3:1}", []string{"a", "b", "c", "d"}, "c"},                                                                                    // :262
		{"should slice with length (${@:N:L})", "${@:2:3}", []string{"a", "b", "c", "d", "e"}, "b c d"},                                                                           // :263
		{"should handle out of range slices", "${@:99}", []string{"a", "b"}, ""},                                                                                                  // :267
		{"should handle out of range slices", "${@:5}", []string{"a", "b"}, ""},                                                                                                   // :268
		{"should handle out of range slices", "${@:10:5}", []string{"a", "b"}, ""},                                                                                                // :269
		{"should handle zero-length slices", "${@:2:0}", []string{"a", "b", "c"}, ""},                                                                                             // :273
		{"should handle zero-length slices", "${@:1:0}", []string{"a", "b"}, ""},                                                                                                  // :274
		{"should handle length exceeding array", "${@:2:99}", []string{"a", "b", "c"}, "b c"},                                                                                     // :278
		{"should handle length exceeding array", "${@:1:10}", []string{"a", "b"}, "a b"},                                                                                          // :279
		{"should process slice before simple $@", "${@:2} vs $@", []string{"a", "b", "c"}, "b c vs a b c"},                                                                        // :283
		{"should process slice before simple $@", "First: ${@:1:1}, All: $@", []string{"x", "y", "z"}, "First: x, All: x y z"},                                                    // :284
		{"should not recursively substitute slice patterns in args", "${@:1}", []string{"${@:2}", "test"}, "${@:2} test"},                                                         // :288
		{"should not recursively substitute slice patterns in args", "${@:2}", []string{"a", "${@:3}", "c"}, "${@:3} c"},                                                          // :289
		{"should handle mixed usage with positional args", "$1: ${@:2}", []string{"cmd", "arg1", "arg2"}, "cmd: arg1 arg2"},                                                       // :293
		{"should handle mixed usage with positional args", "$1 $2 ${@:3}", []string{"a", "b", "c", "d"}, "a b c d"},                                                               // :294
		{"should treat ${@:0} as all args", "${@:0}", []string{"a", "b", "c"}, "a b c"},                                                                                           // :298
		{"should handle empty args array", "${@:2}", []string{}, ""},                                                                                                              // :302
		{"should handle empty args array", "${@:1}", []string{}, ""},                                                                                                              // :303
		{"should handle single arg array", "${@:1}", []string{"only"}, "only"},                                                                                                    // :307
		{"should handle single arg array", "${@:2}", []string{"only"}, ""},                                                                                                        // :308
		{"should handle slice in middle of text", "Process ${@:2} with $1", []string{"tool", "file1", "file2"}, "Process file1 file2 with tool"},                                  // :312
		{"should handle multiple slices in one template", "${@:1:1} and ${@:2}", []string{"a", "b", "c"}, "a and b c"},                                                            // :318
		{"should handle multiple slices in one template", "${@:1:2} vs ${@:3:2}", []string{"a", "b", "c", "d", "e"}, "a b vs c d"},                                                // :319
		{"should handle quoted arguments in slices", "${@:2}", []string{"cmd", "first arg", "second arg"}, "first arg second arg"},                                                // :323
		{"should handle special characters in sliced args", "${@:2}", []string{"cmd", "$100", "@user", "#tag"}, "$100 @user #tag"},                                                // :327
		{"should handle unicode in sliced args", "${@:1}", []string{"\u65e5\u672c\u8a9e", "\U0001f389", "caf\u00e9"}, "\u65e5\u672c\u8a9e \U0001f389 caf\u00e9"},                  // :331
		{"should handle slice with no spacing", "prefix${@:2}suffix", []string{"a", "b", "c"}, "prefixb csuffix"},                                                                 // :343
	} {
		if got := Substitute(c.content, c.args); got != c.want {
			t.Errorf("%s: Substitute(%q, %q) = %q, want %q", c.name, c.content, c.args, got, c.want)
		}
	}
}

// numbered is n arguments named prefix and their index from first.
func numbered(prefix string, n, first int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%d", prefix, i+first)
	}
	return out
}

// Pi's substituteArgs cases whose arguments are computed, ported by hand.
func TestSubstituteComputed(t *testing.T) {
	// should replace $@ and $ARGUMENTS identically (:36-39)
	args := []string{"foo", "bar", "baz"}
	if a, b := Substitute("Test: $@", args), Substitute("Test: $ARGUMENTS", args); a != b {
		t.Errorf("$@ %q, $ARGUMENTS %q", a, b)
	}
	// should handle both syntaxes in same command with same result (:150-156)
	args = []string{"x", "y", "z"}
	r1, r2 := Substitute("$@ and $ARGUMENTS", args), Substitute("$ARGUMENTS and $@", args)
	if r1 != r2 || r1 != "x y z and x y z" {
		t.Errorf("both syntaxes: %q, %q", r1, r2)
	}
	// should handle very long argument lists (:158-162)
	args = numbered("arg", 100, 0)
	if got := Substitute("$ARGUMENTS", args); got != strings.Join(args, " ") {
		t.Errorf("100 args: %q", got)
	}
	// should handle numbered placeholders with multiple digits (:168-171)
	if got := Substitute("$10 $12 $15", numbered("val", 15, 0)); got != "val9 val11 val14" {
		t.Errorf("multiple digits: %q", got)
	}
	// should support defaults for all arguments (:202-209)
	template := "${@:-default}\n${ARGUMENTS:-default}"
	if got := Substitute(template, nil); got != "default\ndefault" {
		t.Errorf("all defaults, none: %q", got)
	}
	if got := Substitute(template, []string{"This", "would", "be", "the", "arguments"}); got != "This would be the arguments\nThis would be the arguments" {
		t.Errorf("all defaults, given: %q", got)
	}
	// should combine positional, slice, and wildcard placeholders (:334-340)
	got := Substitute("Run $1 on ${@:2:2}, then process $@", []string{"eslint", "file1.ts", "file2.ts", "file3.ts"})
	if got != "Run eslint on file1.ts file2.ts, then process eslint file1.ts file2.ts file3.ts" {
		t.Errorf("combined: %q", got)
	}
	// should handle large slice lengths gracefully (:346-349)
	if got := Substitute("${@:5:100}", numbered("arg", 10, 1)); got != "arg5 arg6 arg7 arg8 arg9 arg10" {
		t.Errorf("large slice length: %q", got)
	}
}

// Numbers past an int, which JavaScript's parseInt reads as a number past
// any argument list: not Pi's cases, but its outcomes.
func TestSubstituteLargeNumbers(t *testing.T) {
	args := []string{"a", "b"}
	for content, want := range map[string]string{
		"$99999999999999999999":             "",
		"${99999999999999999999:-d}":        "d",
		"${@:99999999999999999999}":         "",
		"${@:2:99999999999999999999}":       "b",
		"${@:0:99999999999999999999}":       "a b",
		"${@:000000000000000000002}":        "b",
		"$000000000000000000001 and ${@:1}": "a and a b",
	} {
		if got := Substitute(content, args); got != want {
			t.Errorf("Substitute(%q) = %q, want %q", content, got, want)
		}
	}
}

// Pi's parseCommandArgs cases with literal arguments.
func TestParseArgs(t *testing.T) {
	for _, c := range []struct {
		name, in string
		want     []string
	}{
		{"should parse simple space-separated arguments", "a b c", []string{"a", "b", "c"}},                                                                          // :358
		{"should parse quoted arguments with spaces", "\"first arg\" second", []string{"first arg", "second"}},                                                       // :362
		{"should parse single-quoted arguments", "'first arg' second", []string{"first arg", "second"}},                                                              // :366
		{"should parse mixed quote styles", "\"double\" 'single' \"double again\"", []string{"double", "single", "double again"}},                                    // :370
		{"should handle empty string", "", []string{}},                                                                                                               // :374
		{"should handle extra spaces", "a  b   c", []string{"a", "b", "c"}},                                                                                          // :378
		{"should handle tabs as separators", "a\tb\tc", []string{"a", "b", "c"}},                                                                                     // :382
		{"should handle quoted empty string", "\"\" \" \"", []string{" "}},                                                                                           // :387
		{"should handle arguments with special characters", "$100 @user #tag", []string{"$100", "@user", "#tag"}},                                                    // :391
		{"should handle unicode characters", "\u65e5\u672c\u8a9e \U0001f389 caf\u00e9", []string{"\u65e5\u672c\u8a9e", "\U0001f389", "caf\u00e9"}},                   // :395
		{"should handle newlines in quoted arguments", "\"line1\nline2\" second", []string{"line1\nline2", "second"}},                                                // :399
		{"should treat unquoted newlines as separators", "label-2\n\nHere is some description #2.", []string{"label-2", "Here", "is", "some", "description", "#2."}}, // :403
		{"should collapse mixed unquoted whitespace", "a\n\n\tb  c", []string{"a", "b", "c"}},                                                                        // :414
		{"should handle escaped quotes inside quoted strings", "\"quoted \\\"text\\\"\"", []string{"quoted \\text\\"}},                                               // :419
		{"should handle trailing spaces", "a b c   ", []string{"a", "b", "c"}},                                                                                       // :423
		{"should handle leading spaces", "   a b c", []string{"a", "b", "c"}},                                                                                        // :427
	} {
		if got := ParseArgs(c.in); !slices.Equal(got, c.want) {
			t.Errorf("%s: ParseArgs(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// JavaScript's \s separates arguments, so U+3000 and U+FEFF do and U+0085
// does not (0005-PLAN F3-3 deviation 4); bytes that are not UTF-8 are
// kept.
func TestParseArgsWhitespace(t *testing.T) {
	bad := string([]byte{'a', 0xff, 'b'})
	for in, want := range map[string][]string{
		"a\u3000b\ufeffc":        {"a", "b", "c"},
		"a\u0085b":               {"a\u0085b"},
		"\u2028a\u00a0b\u205f":   {"a", "b"},
		bad + " c":               {bad, "c"},
		"\"" + bad + " q\" next": {bad + " q", "next"},
	} {
		if got := ParseArgs(in); !slices.Equal(got, want) {
			t.Errorf("ParseArgs(%q) = %q, want %q", in, got, want)
		}
	}
}

// Pi's parseCommandArgs + substituteArgs integration (:469-494).
func TestParseAndSubstitute(t *testing.T) {
	args := ParseArgs(`Button "onClick handler" "disabled support"`)
	if got := Substitute("Create component $1 with features: $ARGUMENTS", args); got != "Create component Button with features: Button onClick handler disabled support" {
		t.Errorf("parse and substitute: %q", got)
	}
	if got := Substitute("Create a React component named $1 with features: $ARGUMENTS", args); got != "Create a React component named Button with features: Button onClick handler disabled support" {
		t.Errorf("README example: %q", got)
	}
	args = ParseArgs("feature1 feature2 feature3")
	if a, b := Substitute("Implement: $@", args), Substitute("Implement: $ARGUMENTS", args); a != b {
		t.Errorf("$@ %q, $ARGUMENTS %q", a, b)
	}
}

// Pi's expandPromptTemplate cases (:435-463), and text that is not a
// template.
func TestExpandTemplate(t *testing.T) {
	argTest := func(content string) []Template {
		return []Template{{Name: "arg-test", Description: "test", Content: content, FilePath: "/tmp/arg-test.md"}}
	}
	for _, c := range []struct {
		name, text, content, want string
	}{
		{"should split template arguments on unquoted newlines", "/arg-test label-2\n\nHere is some description #2.", "- arg1: $1\n- rest: ${@:2}", "- arg1: label-2\n- rest: Here is some description #2."},
		{"should support template command separated from args by newline", "/arg-test\nlabel-2", "arg1: $1", "arg1: label-2"},
		{"a name ends at U+3000", "/arg-test\u3000x", "arg1: $1", "arg1: x"},
	} {
		if got, ok := ExpandTemplate(c.text, argTest(c.content)); !ok || got != c.want {
			t.Errorf("%s: %q, %v", c.name, got, ok)
		}
	}
	for _, text := range []string{"/other x", "arg-test x", " /arg-test x", "/", "/ arg-test", "/arg-test\u0085x", "/arg-tests"} {
		if got, ok := ExpandTemplate(text, argTest("$1")); ok || got != text {
			t.Errorf("%q expanded to %q", text, got)
		}
	}
	// Pi's name is one character or more, so a template with no name
	// never matches.
	if got, ok := ExpandTemplate("/ x", []Template{{Content: "none"}}); ok || got != "/ x" {
		t.Errorf("a template with no name expanded to %q", got)
	}
}

// Pi's documentation example (packages/coding-agent/docs/prompt-templates.md),
// loaded from testdata, as F3's accept line asks: quoting and defaults.
func TestDocsExample(t *testing.T) {
	templates, diags := LoadTemplates([]string{filepath.Join("testdata", "prompts")})
	if len(diags) != 0 || len(templates) != 1 {
		t.Fatalf("templates %v, diagnostics %v", templates, diags)
	}
	review := templates[0]
	if review.Name != "review" || review.Description != "Review staged git changes" || review.ArgumentHint != "[focus]" {
		t.Fatalf("review: %+v", review)
	}
	for text, want := range map[string]string{
		"/review":                     "Review the staged changes. Focus on correctness, security, and error handling.",
		"/review concurrency":         "Review the staged changes. Focus on concurrency.",
		`/review "API compatibility"`: "Review the staged changes. Focus on API compatibility.",
	} {
		if got, ok := ExpandTemplate(text, templates); !ok || got != want {
			t.Errorf("%s: %q, %v", text, got, ok)
		}
	}
}

func writeTemplate(t *testing.T, dir, name, text string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func find(templates []Template, name string) (Template, bool) {
	i := slices.IndexFunc(templates, func(t Template) bool { return t.Name == name })
	if i < 0 {
		return Template{}, false
	}
	return templates[i], true
}

// Pi's argument-hint cases (:500-624).
func TestArgumentHint(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "pr.md", "---\ndescription: Review PRs from URLs with structured issue and code analysis\nargument-hint: \"<PR-URL>\"\n---\nYou are given one or more GitHub PR URLs: $@")
	writeTemplate(t, dir, "wr.md", "---\ndescription: Finish the current task end-to-end with changelog, commit, and push\nargument-hint: \"[instructions]\"\n---\nWrap it. Additional instructions: $ARGUMENTS")
	writeTemplate(t, dir, "cl.md", "---\ndescription: Audit changelog entries before release\n---\nAudit changelog entries for all commits since the last release.")
	writeTemplate(t, dir, "empty-hint.md", "---\ndescription: A command with empty hint\nargument-hint: \"\"\n---\nDo something")
	writeTemplate(t, dir, "is.md", "---\ndescription: Analyze GitHub issues (bugs or feature requests)\nargument-hint: \"<issue>\"\n---\nAnalyze GitHub issue(s): $ARGUMENTS")
	templates, diags := LoadTemplates([]string{dir})
	if len(diags) != 0 {
		t.Fatalf("diagnostics %v", diags)
	}
	for _, c := range []struct{ name, hint, desc string }{
		{"pr", "<PR-URL>", "Review PRs from URLs with structured issue and code analysis"},
		{"wr", "[instructions]", "Finish the current task end-to-end with changelog, commit, and push"},
		{"cl", "", "Audit changelog entries before release"},
		{"empty-hint", "", "A command with empty hint"},
		{"is", "<issue>", "Analyze GitHub issues (bugs or feature requests)"},
	} {
		tmpl, ok := find(templates, c.name)
		if !ok || tmpl.ArgumentHint != c.hint || tmpl.Description != c.desc {
			t.Errorf("%s: %+v, %v", c.name, tmpl, ok)
		}
	}
}

// Pi's diagnostics case (:626-653): invalid YAML is a warning with the
// file's path, the file is skipped, and its valid sibling loads. The
// message is yaml.v3's, where Pi's library gives a line and a column
// (0005-PLAN F3-3 deviation 5).
func TestInvalidFrontmatter(t *testing.T) {
	dir := t.TempDir()
	invalid := writeTemplate(t, dir, "invalid.md", "---\ndescription: Broken: unquoted colon\n---\nDo something.\n")
	writeTemplate(t, dir, "valid.md", "Valid prompt content.")
	templates, diags := LoadTemplates([]string{dir})
	if len(templates) != 1 || templates[0].Name != "valid" {
		t.Fatalf("templates %v", templates)
	}
	want := []Diagnostic{{Path: invalid, Message: "yaml: mapping values are not allowed in this context"}}
	if !slices.Equal(diags, want) {
		t.Fatalf("diagnostics %v", diags)
	}
}

// The description falls back to the body's first non-empty line, cut to 60
// of JavaScript's UTF-16 units with "...", as Pi cuts it; a pair cut in
// half leaves U+FFFD.
func TestDescriptionFallback(t *testing.T) {
	dir := t.TempDir()
	sixty := strings.Repeat("x", 60)
	writeTemplate(t, dir, "blank.md", "\n  \n\t\nFirst line\nSecond")
	writeTemplate(t, dir, "exact.md", sixty+"\nmore")
	writeTemplate(t, dir, "long.md", sixty+"y")
	writeTemplate(t, dir, "pair.md", strings.Repeat("x", 59)+"\U0001F389 party")
	writeTemplate(t, dir, "front.md", "---\nargument-hint: \"[x]\"\n---\n\n  Indented line  \n")
	writeTemplate(t, dir, "notstring.md", "---\ndescription: 7\n---\nBody line")
	writeTemplate(t, dir, "nel.md", "\u0085\nNext")
	templates, _ := LoadTemplates([]string{dir})
	for name, want := range map[string]string{
		"blank":     "First line",
		"exact":     sixty,
		"long":      sixty + "...",
		"pair":      strings.Repeat("x", 59) + "\ufffd...",
		"front":     "Indented line",
		"notstring": "Body line",
		"nel":       "\u0085",
	} {
		if tmpl, ok := find(templates, name); !ok || tmpl.Description != want {
			t.Errorf("%s: %q, %v", name, tmpl.Description, ok)
		}
	}
}

// The scan is Pi's collectAutoPromptEntries: a directory's direct .md
// children, without dotfiles, node_modules or what its .gitignore, .ignore
// and .fdignore exclude (0005-PLAN F3-3 deviation 3), a link followed to a
// regular file; a missing directory gives nothing.
func TestTemplateScan(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"keep.md", ".hidden.md", "notes.txt", "sub/deep.md", "node_modules/mod.md",
		"by-git.md", "by-ignore.md", "by-fd.md", "README.MD"} {
		writeTemplate(t, dir, name, "Body of "+name)
	}
	writeTemplate(t, dir, ".gitignore", "by-git.md\n")
	writeTemplate(t, dir, ".ignore", "by-ignore.md\n")
	writeTemplate(t, dir, ".fdignore", "by-fd.md\n")
	if err := os.Mkdir(filepath.Join(dir, "folder.md"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "keep.md"), filepath.Join(dir, "alias.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing.md"), filepath.Join(dir, "broken.md")); err != nil {
		t.Fatal(err)
	}
	templates, diags := LoadTemplates([]string{dir, filepath.Join(dir, "absent")})
	var got []string
	for _, tmpl := range templates {
		got = append(got, tmpl.Name)
	}
	if !slices.Equal(got, []string{"alias", "keep"}) || len(diags) != 0 {
		t.Fatalf("templates %v, diagnostics %v", got, diags)
	}
}

// A name in two directories keeps the first, with Pi's collision
// diagnostic naming the winner (resource-loader.ts:1148-1170).
func TestTemplateCollision(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	winner := writeTemplate(t, first, "deploy.md", "First")
	loser := writeTemplate(t, second, "deploy.md", "Second")
	writeTemplate(t, second, "other.md", "Other")
	templates, diags := LoadTemplates([]string{first, second})
	if len(templates) != 2 || templates[0].Content != "First" || templates[1].Name != "other" {
		t.Fatalf("templates %v", templates)
	}
	want := []Diagnostic{{Path: loser, Message: `name "/deploy" collision`, Winner: winner}}
	if !slices.Equal(diags, want) {
		t.Fatalf("diagnostics %v", diags)
	}
}
