package permission

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// SubjectKind is what a rule's pattern matches for a tool (0005-PLAN F6-2
// decision 3).
type SubjectKind int

// The subject kinds. A tool with none matches by its name alone.
const (
	SubjectNone    SubjectKind = iota
	SubjectPath                // each path the call names
	SubjectCommand             // the whole command line
)

// ExternalDirectory is the rule tool of a path outside the workspace's
// roots (decision 7).
const ExternalDirectory = "external_directory"

// The layers a rule comes from, in the order they are joined (decision 4),
// and gobble's limits, applied after them (decision 5).
const (
	LayerDefault = "default"
	LayerGlobal  = "global"
	LayerProject = "project"
	LayerStore   = "project store"
	LayerSession = "session"
	LayerLimit   = "limit"
)

// RuleSource is where a rule came from: its layer, and its file and line
// when it has them (decision 12).
type RuleSource struct {
	Layer string `json:"layer"`
	File  string `json:"file,omitzero"`
	Line  int    `json:"line,omitzero"`
}

func (s RuleSource) String() string {
	switch {
	case s.File == "":
		return s.Layer
	case s.Line == 0:
		return fmt.Sprintf("%s (%s)", s.File, s.Layer)
	}
	return fmt.Sprintf("%s:%d (%s)", s.File, s.Line, s.Layer)
}

// Spec is one rule: its effect, the tools it names by opencode's wildcard,
// and the pattern their subjects must match, "" being any (decision 2).
type Spec struct {
	Effect  Mode       `json:"effect"`
	Tool    string     `json:"tool"`
	Pattern string     `json:"pattern,omitzero"`
	Source  RuleSource `json:"source"`
}

// String is the rule as a settings file writes it: "allow: bash(git *)".
func (s Spec) String() string { return string(s.Effect) + ": " + s.Rule() }

// Rule is the rule without its effect: "bash(git *)".
func (s Spec) Rule() string {
	if s.Pattern == "" {
		return s.Tool
	}
	return s.Tool + "(" + s.Pattern + ")"
}

// ErrBadRule is ParseSpec's error.
var ErrBadRule = errors.New("not a rule")

// ParseSpec reads text, "<tool>" or "<tool>(<pattern>)", as a rule of
// effect allow, ask or deny. The tool part is a name or a wildcard over
// names; a pattern that is a path is checked as a glob.
func ParseSpec(effect Mode, text string) (Spec, error) {
	switch effect {
	case ModeAllow, ModeAsk, ModeDeny:
	default:
		return Spec{}, fmt.Errorf("%w: %q is not allow, ask or deny", ErrBadRule, effect)
	}
	text = strings.TrimSpace(text)
	tool, pattern := text, ""
	if i := strings.IndexByte(text, '('); i >= 0 {
		if !strings.HasSuffix(text, ")") {
			return Spec{}, fmt.Errorf("%w: %q has an opening ( and no closing ) at its end", ErrBadRule, text)
		}
		tool, pattern = strings.TrimSpace(text[:i]), text[i+1:len(text)-1]
	}
	if tool == "" || strings.ContainsAny(tool, " ()") {
		return Spec{}, fmt.Errorf("%w: %q names no tool; write <tool> or <tool>(<pattern>)", ErrBadRule, text)
	}
	if pattern != "" {
		if _, err := fsx.CompileGlob(filepath.ToSlash(pattern)); err != nil {
			return Spec{}, fmt.Errorf("%w: %q: %w", ErrBadRule, text, err)
		}
	}
	return Spec{Effect: effect, Tool: tool, Pattern: pattern}, nil
}

// wildcard is opencode's matcher (packages/core/src/util/wildcard.ts):
// "*" is any text, "?" one character, a trailing " *" also matches nothing
// after the word, and on darwin and windows, whose volumes find a program
// whatever its case, case is ignored (0037-MADR D4).
func wildcard(input, pattern string) bool {
	var b strings.Builder
	b.WriteString("(?s")
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		b.WriteString("i")
	}
	b.WriteString(")^")
	p := strings.ReplaceAll(pattern, `\`, "/")
	tail := ""
	if cut, ok := strings.CutSuffix(p, " *"); ok {
		p, tail = cut, "( .*)?"
	}
	for _, r := range p {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString(tail + "$")
	return regexp.MustCompile(b.String()).MatchString(strings.ReplaceAll(input, `\`, "/"))
}

// Verdict is a call's answer from the rules and the limits: the effect, the
// rule that decided it, and, when one of gobble's limits decided, why.
type Verdict struct {
	Effect Mode   `json:"effect"`
	By     Spec   `json:"by"`
	Limit  string `json:"limit,omitzero"`
}

// Engine evaluates a call against the rules (decisions 2 to 7).
type Engine struct {
	// Layers are the rules of the global file, a trusted project, the
	// project's store and the session, joined in that order; the last rule
	// that matches decides.
	Layers [][]Spec
	// Default is a call's default effect, the layer before Layers; nil asks.
	// external_directory's is always ask.
	Default func(r Rule) Mode
	// Root is the project root, which path patterns are relative to; Home
	// and ConfigDir place the protected paths.
	Root, Home, ConfigDir string
}

// writers are the tools whose paths the protected paths keep (decision 5).
var writers = []string{"write", "edit", "move", "delete", "copy", "mkdir"}

// Evaluate is r's verdict: the rules for its subjects, the strictest of them,
// then external_directory for each outside path, which decides a tie, so
// the rule that let an outside path through is the one named; then
// gobble's limits.
func (e Engine) Evaluate(r Rule) Verdict {
	subjects := r.Subjects
	if r.Kind == SubjectNone || len(subjects) == 0 {
		subjects = []string{""}
	}
	def := ModeAsk
	if e.Default != nil {
		def = e.Default(r)
	}
	var v Verdict
	for i, s := range subjects {
		got := e.matchReadings(r.Tool, r.Kind, s, def)
		if i == 0 || stricter(got.Effect, v.Effect) {
			v = got
		}
	}
	for _, p := range r.Outside {
		if got := e.match(ExternalDirectory, SubjectPath, p, ModeAsk); !stricter(v.Effect, got.Effect) {
			v = got
		}
	}
	return e.limits(r, v)
}

// readingsOf is a subject's readings: a path as given, and as the file
// system resolves it when that differs, so that a symlink cannot carry a
// path past a rule (0037-MADR D2, D5); any other subject as it is.
func readingsOf(kind SubjectKind, s string) []string {
	if kind != SubjectPath || s == "" {
		return []string{s}
	}
	return readings(s)
}

// matchReadings is subject's verdict over its readings: the strictest of
// those a rule matched, or the default's when no rule matched any. A
// reading only the default decides does not outrank one a rule decides, so
// an allow written for either spelling of a path still allows, and a deny
// for either still denies (0037-MADR D5, as amended by 0037-PLAN P2's
// deviation 1).
func (e Engine) matchReadings(tool string, kind SubjectKind, subject string, def Mode) Verdict {
	var v, fallback Verdict
	found := false
	for i, reading := range readingsOf(kind, subject) {
		got := e.match(tool, kind, reading, def)
		if got.By.Source.Layer == LayerDefault {
			if i == 0 {
				fallback = got
			}
			continue
		}
		if !found || stricter(got.Effect, v.Effect) {
			v, found = got, true
		}
	}
	if !found {
		return fallback
	}
	return v
}

// readings are p cleaned, and its real path when that differs.
func readings(p string) []string {
	p = filepath.Clean(p)
	if r := fsx.RealPath(p); r != p {
		return []string{p, r}
	}
	return []string{p}
}

// match is the last rule over the layers that names tool and whose pattern
// matches subject, or def.
func (e Engine) match(tool string, kind SubjectKind, subject string, def Mode) Verdict {
	shown := subject
	if kind == SubjectPath {
		shown = e.relative(subject)
	}
	var found *Spec
	for _, layer := range e.Layers {
		for i := range layer {
			s := &layer[i]
			if wildcard(tool, s.Tool) && patternMatches(kind, s.Pattern, shown) {
				found = s
			}
		}
	}
	if found != nil {
		return Verdict{Effect: found.Effect, By: *found}
	}
	return Verdict{Effect: def, By: Spec{Effect: def, Tool: tool, Source: RuleSource{Layer: LayerDefault}}}
}

// patternMatches reports whether pattern takes subject: any for "" or "*";
// a path by gobble's globs, a leading "/" anchoring the pattern at the root
// as gitignore's does (F6-2b decision C); a command by opencode's wildcard;
// a tool with no subject only by "" or "*".
func patternMatches(kind SubjectKind, pattern, subject string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	switch kind {
	case SubjectPath:
		g, err := fsx.CompileGlob(folded(pattern))
		return err == nil && g.Match(strings.TrimPrefix(folded(subject), "/"), false)
	case SubjectCommand:
		return wildcard(subject, pattern)
	}
	return false
}

// folded is a pattern or a path with "/", as the volume compares names
// (fsx.Fold): a path rule then holds for every spelling the volume takes
// for the same file (0037-MADR D5).
func folded(p string) string {
	return filepath.ToSlash(fsx.Fold(filepath.FromSlash(p)))
}

// relative is p relative to the root with "/", when it is inside the root
// as given or as resolved, and otherwise absolute with "/".
func (e Engine) relative(p string) string {
	if e.Root != "" {
		for _, root := range readings(e.Root) {
			if rel, err := filepath.Rel(root, p); err == nil && filepath.IsLocal(rel) {
				return filepath.ToSlash(rel)
			}
		}
	}
	return filepath.ToSlash(p)
}

// rank orders the effects, deny strictest (decision 3).
func rank(m Mode) int {
	switch m {
	case ModeDeny:
		return 2
	case ModeAsk:
		return 1
	}
	return 0
}

func stricter(a, b Mode) bool { return rank(a) > rank(b) }

// The limits' reasons, which the model reads in a refusal.
const (
	limitProtected = "gobble never changes files under .git, under ~/.ssh, or its own credential and trust files"
	limitConfig    = "a change to gobble's or the project's configuration, AGENTS.md or the project's skills always asks"
	limitDelete    = "a delete always asks"
)

// limits applies gobble's limits to v, which no rule lifts (decision 5): a
// write to a protected path is denied; a write to a configuration path,
// and a delete, are the limit's question whenever the rules do not deny
// them, so the question is marked, and --yes does not approve it, when
// the rules would only have asked (decision 8). A shell command's Writes
// meet the same two limits, whatever its tool (0037-MADR D3).
func (e Engine) limits(r Rule, v Verdict) Verdict {
	limit := func(effect Mode, why string) Verdict {
		return Verdict{Effect: effect, By: Spec{Effect: effect, Tool: r.Tool, Source: RuleSource{Layer: LayerLimit}}, Limit: why}
	}
	if slices.ContainsFunc(r.Writes, e.protected) {
		return limit(ModeDeny, limitProtected)
	}
	if v.Effect != ModeDeny && slices.ContainsFunc(r.Writes, e.configPath) {
		return limit(ModeAsk, limitConfig)
	}
	if !slices.Contains(writers, r.Tool) || r.Kind != SubjectPath {
		return v
	}
	if slices.ContainsFunc(r.Subjects, e.protected) {
		return limit(ModeDeny, limitProtected)
	}
	if v.Effect == ModeDeny {
		return v
	}
	if slices.ContainsFunc(r.Subjects, e.configPath) {
		return limit(ModeAsk, limitConfig)
	}
	if r.Tool == "delete" {
		return limit(ModeAsk, limitDelete)
	}
	return v
}

// protected reports whether p is under a .git directory, under ~/.ssh, or
// one of gobble's credential, trust and permission files: as given or as
// resolved, and in any spelling the volume takes for it (0037-MADR D2).
func (e Engine) protected(p string) bool {
	for _, q := range readings(p) {
		if slices.Contains(strings.Split(folded(q), "/"), ".git") {
			return true
		}
		if e.Home != "" && under(q, filepath.Join(e.Home, ".ssh")) {
			return true
		}
		if e.ConfigDir != "" {
			for _, name := range []string{"auth.json", TrustFile, "permissions.json"} {
				if samePlace(q, filepath.Join(e.ConfigDir, name)) {
					return true
				}
			}
		}
	}
	return false
}

// configPath reports whether p is in gobble's config directory, under the
// project's .gobble or .agents/skills, or an AGENTS.md, as protected reads
// it.
func (e Engine) configPath(p string) bool {
	for _, q := range readings(p) {
		switch {
		case fsx.Fold(filepath.Base(q)) == fsx.Fold("AGENTS.md"):
			return true
		case e.ConfigDir != "" && under(q, e.ConfigDir):
			return true
		case e.Root != "" && (under(q, filepath.Join(e.Root, ".gobble")) || under(q, filepath.Join(e.Root, ".agents", "skills"))):
			return true
		}
	}
	return false
}

// under reports whether p is dir or inside it, dir as given or as resolved,
// compared as the volume compares names.
func under(p, dir string) bool {
	for _, d := range readings(dir) {
		if fsx.UnderDir(p, d) {
			return true
		}
	}
	return false
}

// samePlace reports whether p is the file named, as given or as resolved,
// compared as the volume compares names.
func samePlace(p, file string) bool {
	for _, f := range readings(file) {
		if fsx.SamePath(p, f) {
			return true
		}
	}
	return false
}

// pathMeta and commandMeta are the characters that make a subject a
// pattern rather than itself: gobble's globs' and opencode's wildcard's.
const (
	pathMeta    = `*?[{\`
	commandMeta = `*?`
	// prefixMeta are the characters that keep an arity prefix from being
	// saved (F6-3 decision 4).
	prefixMeta = `*?[{`
)

// Remember is the rules an "always" answer of effect saves for r (F6-2b
// decisions A and C, F6-3 decision 4a): a path from the root with a
// leading "/", or absolute outside it; a command's arity pattern from
// r.Saves, or its exact text when it has none; a tool with no subject by
// its name; each saved once. An allow
// also saves an external_directory rule for each path outside the roots;
// a deny saves the tool's alone. ok is false when a subject holds a
// pattern's character, so no rule matches it alone: only "once" can answer.
func (e Engine) Remember(effect Mode, r Rule) (rules []Spec, ok bool) {
	spec := func(tool, pattern string) Spec { return Spec{Effect: effect, Tool: tool, Pattern: pattern} }
	if r.Kind == SubjectNone || len(r.Subjects) == 0 {
		rules = append(rules, spec(r.Tool, ""))
	}
	for i, subj := range r.Subjects {
		switch r.Kind {
		case SubjectPath:
			p, exact := e.exactPath(subj)
			if !exact {
				return nil, false
			}
			rules = append(rules, spec(r.Tool, p))
		case SubjectCommand:
			save := ""
			if i < len(r.Saves) {
				save = r.Saves[i]
			}
			switch {
			case save != "" && strings.ContainsAny(strings.TrimSuffix(save, " *"), prefixMeta):
				return nil, false
			case save != "":
				subj = save
			case strings.ContainsAny(subj, commandMeta):
				return nil, false
			}
			if s := spec(r.Tool, subj); !slices.Contains(rules, s) {
				rules = append(rules, s)
			}
		}
	}
	if effect == ModeAllow {
		for _, out := range r.Outside {
			p, exact := e.exactPath(out)
			if !exact {
				return nil, false
			}
			rules = append(rules, spec(ExternalDirectory, p))
		}
	}
	return rules, true
}

// exactPath is p as a pattern that matches it alone: from the root with a
// leading "/" inside it, absolute outside it; false when p holds a glob's
// character.
func (e Engine) exactPath(p string) (string, bool) {
	shown := e.relative(p)
	if strings.ContainsAny(shown, pathMeta) {
		return "", false
	}
	if !strings.HasPrefix(shown, "/") && !filepath.IsAbs(filepath.FromSlash(shown)) {
		shown = "/" + shown
	}
	return shown, true
}

// Hidden reports whether tool is left out of the model's tools: the last
// rule that names it is a deny for every subject (decision 6).
func (e Engine) Hidden(tool string) bool {
	var last *Spec
	for _, layer := range e.Layers {
		for i := range layer {
			if wildcard(tool, layer[i].Tool) {
				last = &layer[i]
			}
		}
	}
	return last != nil && last.Effect == ModeDeny && (last.Pattern == "" || last.Pattern == "*")
}
