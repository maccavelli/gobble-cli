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
	if s.File == "" {
		return s.Layer
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
func (s Spec) String() string {
	if s.Pattern == "" {
		return string(s.Effect) + ": " + s.Tool
	}
	return fmt.Sprintf("%s: %s(%s)", s.Effect, s.Tool, s.Pattern)
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
		if _, err := fsx.CompileGlob(strings.TrimPrefix(filepath.ToSlash(pattern), "/")); err != nil {
			return Spec{}, fmt.Errorf("%w: %q: %w", ErrBadRule, text, err)
		}
	}
	return Spec{Effect: effect, Tool: tool, Pattern: pattern}, nil
}

// wildcard is opencode's matcher (packages/core/src/util/wildcard.ts):
// "*" is any text, "?" one character, a trailing " *" also matches nothing
// after the word, and on Windows case is ignored.
func wildcard(input, pattern string) bool {
	var b strings.Builder
	b.WriteString("(?s")
	if runtime.GOOS == "windows" {
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
		got := e.match(r.Tool, r.Kind, s, def)
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
// a path by gobble's globs; a command by opencode's wildcard; a tool with no
// subject only by "" or "*".
func patternMatches(kind SubjectKind, pattern, subject string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	switch kind {
	case SubjectPath:
		g, err := fsx.CompileGlob(strings.TrimPrefix(filepath.ToSlash(pattern), "/"))
		return err == nil && g.Match(strings.TrimPrefix(subject, "/"), false)
	case SubjectCommand:
		return wildcard(subject, pattern)
	}
	return false
}

// relative is p relative to the root with "/", when it is inside it, and
// otherwise absolute with "/".
func (e Engine) relative(p string) string {
	if e.Root != "" {
		if rel, err := filepath.Rel(e.Root, p); err == nil && filepath.IsLocal(rel) {
			return filepath.ToSlash(rel)
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
// the rules would only have asked (decision 8).
func (e Engine) limits(r Rule, v Verdict) Verdict {
	if !slices.Contains(writers, r.Tool) || r.Kind != SubjectPath {
		return v
	}
	limit := func(effect Mode, why string) Verdict {
		return Verdict{Effect: effect, By: Spec{Effect: effect, Tool: r.Tool, Source: RuleSource{Layer: LayerLimit}}, Limit: why}
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
// one of gobble's credential, trust and permission files.
func (e Engine) protected(p string) bool {
	p = filepath.Clean(p)
	if slices.Contains(strings.Split(filepath.ToSlash(p), "/"), ".git") {
		return true
	}
	if e.Home != "" && under(p, filepath.Join(e.Home, ".ssh")) {
		return true
	}
	if e.ConfigDir != "" {
		for _, name := range []string{"auth.json", TrustFile, "permissions.json"} {
			if samePath(p, filepath.Join(e.ConfigDir, name)) {
				return true
			}
		}
	}
	return false
}

// configPath reports whether p is in gobble's config directory, under the
// project's .gobble or .agents/skills, or an AGENTS.md.
func (e Engine) configPath(p string) bool {
	p = filepath.Clean(p)
	switch {
	case filepath.Base(p) == "AGENTS.md":
		return true
	case e.ConfigDir != "" && under(p, e.ConfigDir):
		return true
	case e.Root != "" && (under(p, filepath.Join(e.Root, ".gobble")) || under(p, filepath.Join(e.Root, ".agents", "skills"))):
		return true
	}
	return false
}

// under reports whether p is dir or inside it.
func under(p, dir string) bool {
	rel, err := filepath.Rel(filepath.Clean(dir), p)
	return err == nil && filepath.IsLocal(rel) || samePath(p, dir)
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
