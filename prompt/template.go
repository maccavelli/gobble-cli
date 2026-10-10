package prompt

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/internal/frontmatter"
	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// Template is a prompt template: a Markdown file whose body is sent in
// place of /<name> [args] (Pi's PromptTemplate, prompt-templates.ts:12-19).
type Template struct {
	// Name is the file name without .md.
	Name string
	// Description is the frontmatter's, or the body's first non-empty line,
	// cut to 60 characters.
	Description string
	// ArgumentHint is the frontmatter's argument-hint, or "".
	ArgumentHint string
	// Content is the body, with its arguments still to substitute.
	Content  string
	FilePath string
}

// templateIgnoreFiles are the files whose rules a prompts directory
// honours, as Pi's collectAutoPromptEntries reads them
// (package-manager.ts:219, :489-516; 0005-PLAN F3-3 deviation 3).
var templateIgnoreFiles = []string{".gitignore", ".ignore", ".fdignore"}

// LoadTemplates loads the templates of dirs, the first directory first, as
// Pi loads its prompts directories (package-manager.ts:489-516,
// prompt-templates.ts:105-153, resource-loader.ts:1148-1170):
//   - each directory's direct .md children only, a link followed to a
//     regular file, without dotfiles or the files the directory's
//     .gitignore, .ignore and .fdignore exclude (Pi's node_modules rule
//     cannot match a .md file);
//   - a file that cannot be read, or whose frontmatter is not YAML, is
//     skipped with a warning;
//   - a name taken by an earlier template keeps the earlier one, with Pi's
//     collision diagnostic, `name "/<name>" collision`, naming the winner.
//
// A directory that does not exist gives nothing.
func LoadTemplates(dirs []string) ([]Template, []Diagnostic) {
	var all []Template
	var diags []Diagnostic
	for _, dir := range dirs {
		for _, path := range templateFiles(dir) {
			t, d, ok := loadTemplate(path)
			if ok {
				all = append(all, t)
			}
			diags = append(diags, d...)
		}
	}
	seen := map[string]string{}
	out := make([]Template, 0, len(all))
	for _, t := range all {
		if winner, ok := seen[t.Name]; ok {
			diags = append(diags, Diagnostic{Path: t.FilePath, Message: `name "/` + t.Name + `" collision`, Winner: winner})
			continue
		}
		seen[t.Name] = t.FilePath
		out = append(out, t)
	}
	return out, diags
}

// templateFiles are a prompts directory's template files, in name order.
func templateFiles(dir string) []string {
	var ig fsx.Ignore
	for _, name := range templateIgnoreFiles {
		if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil { //nolint:gosec // G304: a file of the prompts directory
			ig.Add("", 0, string(b))
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") || ig.Ignored(name, false) {
			continue
		}
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		out = append(out, path)
	}
	return out
}

// loadTemplate is Pi's loadTemplateFromFile (prompt-templates.ts:105-153).
func loadTemplate(path string) (Template, []Diagnostic, bool) {
	b, err := os.ReadFile(path) //nolint:gosec // G304: a template file the scan found
	if err != nil {
		return Template{}, []Diagnostic{{Path: path, Message: err.Error()}}, false
	}
	fields, body, err := frontmatter.Parse(string(b))
	if err != nil {
		return Template{}, []Diagnostic{{Path: path, Message: err.Error()}}, false
	}
	desc, _ := fields["description"].(string) //nolint:errcheck // a description that is not a string is none
	if desc == "" {
		for line := range strings.SplitSeq(body, "\n") {
			if frontmatter.Trim(line) != "" {
				desc = cutDescription(line)
				break
			}
		}
	}
	hint, _ := fields["argument-hint"].(string) //nolint:errcheck // a hint that is not a string is none
	return Template{
		Name:         strings.TrimSuffix(filepath.Base(path), ".md"),
		Description:  desc,
		ArgumentHint: hint,
		Content:      body,
		FilePath:     path,
	}, nil, true
}

// maxDescription is the length a body's first line is cut to, in
// JavaScript's UTF-16 units, as Pi's slice(0, 60) counts them.
const maxDescription = 60

// cutDescription is Pi's firstLine.slice(0, 60), with "..." when it cuts. A
// surrogate pair cut in half leaves U+FFFD, as JavaScript's lone surrogate
// becomes when written as UTF-8.
func cutDescription(line string) string {
	u := utf16.Encode([]rune(line))
	if len(u) <= maxDescription {
		return line
	}
	return string(utf16.Decode(u[:maxDescription])) + "..."
}

// ParseArgs is Pi's parseCommandArgs (prompt-templates.ts:25-55): words
// split on JavaScript's whitespace, a "…" or '…' quote kept as one word
// with its quotes removed, an empty quote giving nothing. A backslash has
// no meaning.
func ParseArgs(s string) []string {
	var args []string
	var cur strings.Builder
	var quote rune
	// A rune's own bytes are kept, so bytes that are not UTF-8 pass
	// through as they came.
	for i, w := 0, 0; i < len(s); i += w {
		var r rune
		r, w = utf8.DecodeRuneInString(s[i:])
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteString(s[i : i+w])
			}
		case r == '"' || r == '\'':
			quote = r
		case frontmatter.IsSpace(r):
			if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteString(s[i : i+w])
		}
	}
	if cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args
}

// placeholder is Pi's substituteArgs pattern (prompt-templates.ts:73).
var placeholder = regexp.MustCompile(`\$\{(\d+|ARGUMENTS|@):-([^}]*)\}|\$\{@:(\d+)(?::(\d+))?\}|\$(ARGUMENTS|@|\d+)`)

// Substitute is Pi's substituteArgs (prompt-templates.ts:70-103), in one
// pass over content, so an argument or a default is never substituted in
// turn:
//   - $1, $2, …, one argument, or "";
//   - $@ and $ARGUMENTS, every argument joined with spaces;
//   - ${N:-default}, ${@:-default} and ${ARGUMENTS:-default}, the same, or
//     the default when that is empty;
//   - ${@:N} and ${@:N:L}, the arguments from the Nth, L of them, ${@:0}
//     counting as ${@:1}.
func Substitute(content string, args []string) string {
	all := strings.Join(args, " ")
	nth := func(n int) string {
		if n < 1 || n > len(args) {
			return ""
		}
		return args[n-1]
	}
	var b strings.Builder
	last := 0
	for _, m := range placeholder.FindAllStringSubmatchIndex(content, -1) {
		b.WriteString(content[last:m[0]])
		last = m[1]
		group := func(i int) (string, bool) {
			if m[2*i] < 0 {
				return "", false
			}
			return content[m[2*i]:m[2*i+1]], true
		}
		if target, ok := group(1); ok {
			value := all
			if target != "@" && target != "ARGUMENTS" {
				value = nth(number(target))
			}
			if value == "" {
				value, _ = group(2)
			}
			b.WriteString(value)
			continue
		}
		if from, ok := group(3); ok {
			start := max(number(from)-1, 0)
			if start >= len(args) {
				continue
			}
			end := len(args)
			if l, ok := group(4); ok {
				if n := number(l); n < end-start {
					end = start + n
				}
			}
			b.WriteString(strings.Join(args[start:end], " "))
			continue
		}
		simple, _ := group(5)
		if simple == "@" || simple == "ARGUMENTS" {
			b.WriteString(all)
		} else {
			b.WriteString(nth(number(simple)))
		}
	}
	b.WriteString(content[last:])
	return b.String()
}

// number is a placeholder's decimal digits as JavaScript's parseInt reads
// them, a number too large for an int being larger than any argument list.
func number(digits string) int {
	n, err := strconv.Atoi(digits)
	if err != nil {
		return math.MaxInt
	}
	return n
}

// ExpandTemplate is Pi's expandPromptTemplate (prompt-templates.ts:304-322):
// text that is "/", a name up to JavaScript's whitespace, and its
// arguments after it, with a template of that name, becomes the
// template's content with the arguments substituted (ParseArgs,
// Substitute). Any other text is returned unchanged, with false.
func ExpandTemplate(text string, templates []Template) (string, bool) {
	rest, ok := strings.CutPrefix(text, "/")
	if !ok {
		return text, false
	}
	end := strings.IndexFunc(rest, frontmatter.IsSpace)
	if end < 0 {
		end = len(rest)
	}
	name := rest[:end]
	if name == "" {
		return text, false
	}
	for _, t := range templates {
		if t.Name == name {
			return Substitute(t.Content, ParseArgs(rest[end:])), true
		}
	}
	return text, false
}
