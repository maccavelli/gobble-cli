package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"

	"github.com/maccavelli/gobble-cli/internal/frontmatter"
	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// Pi's limits (skills.ts:10-14).
const (
	maxName        = 64
	maxDescription = 1024
)

// ignoreFiles are the files whose rules the scan honours in each
// directory, as Pi's does (skills.ts:16).
var ignoreFiles = []string{".gitignore", ".ignore", ".fdignore"}

var namePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Skill is one Agent Skill: its name and description, its file, and the
// directory its relative references resolve against.
type Skill struct {
	Name, Description      string
	FilePath, BaseDir      string
	DisableModelInvocation bool
}

// Diagnostic is a warning about one file, in Pi's words. A collision
// names the skill that won in Winner.
type Diagnostic struct {
	Path, Message, Winner string
}

// Source is one directory skills are found in. Agents is the .agents
// layout, where a loose .md file is a skill only below the directory's top;
// otherwise, as in Pi's own directories, only at the top (Pi's
// collectSkillEntries modes "agents" and "pi", package-manager.ts:369-448).
type Source struct {
	Dir    string
	Agents bool
}

// Load is the skills of sources, in order, as Pi's loadSkills resolves
// them (skills.ts:409-509): a path already loaded through a link is
// skipped, and of two skills with one name the first wins, the second
// a collision diagnostic, listed after every other.
func Load(sources []Source) ([]Skill, []Diagnostic) {
	var skills []Skill
	var diags, collisions []Diagnostic
	byName := map[string]int{}
	loaded := map[string]bool{}
	for _, src := range sources {
		for _, file := range scan(src) {
			s, d, ok := loadFile(file)
			diags = append(diags, d...)
			if !ok {
				continue
			}
			rp := canonical(s.FilePath)
			if loaded[rp] {
				continue
			}
			if at, dup := byName[s.Name]; dup {
				collisions = append(collisions, Diagnostic{Path: s.FilePath, Message: `name "` + s.Name + `" collision`, Winner: skills[at].FilePath})
				continue
			}
			byName[s.Name], loaded[rp] = len(skills), true
			skills = append(skills, s)
		}
	}
	return skills, append(diags, collisions...)
}

func canonical(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// scan is a source's candidate files, as Pi's collectSkillEntries finds
// them: a directory holding SKILL.md is a skill, and is not searched
// further; dotfiles and node_modules are skipped; the ignore files'
// rules apply; a directory reached twice through links is searched once.
func scan(src Source) []string {
	if info, err := os.Stat(src.Dir); err != nil || !info.IsDir() {
		return nil
	}
	w := walker{root: src.Dir, agents: src.Agents, ig: &fsx.Ignore{}, seen: map[string]bool{}}
	w.dir(src.Dir, 0)
	return w.files
}

type walker struct {
	root   string
	agents bool
	ig     *fsx.Ignore
	seen   map[string]bool
	files  []string
}

func (w *walker) rel(p string) string {
	r, err := filepath.Rel(w.root, p)
	if err != nil || r == "." {
		return ""
	}
	return filepath.ToSlash(r)
}

func (w *walker) dir(dir string, depth int) {
	rp := canonical(dir)
	if w.seen[rp] {
		return
	}
	w.seen[rp] = true
	base := w.rel(dir)
	for _, name := range ignoreFiles {
		if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil { //nolint:gosec // G304: a file of the skills directory
			w.ig.Add(base, depth, string(b))
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	skillMD := filepath.Join(dir, "SKILL.md")
	if info, err := os.Stat(skillMD); err == nil && info.Mode().IsRegular() && !w.ig.Ignored(w.rel(skillMD), false) {
		w.files = append(w.files, skillMD)
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" {
			continue
		}
		full := filepath.Join(dir, name)
		info, err := os.Stat(full) // follows a link, as Pi's statSync does
		if err != nil {
			continue
		}
		rel := w.rel(full)
		top := dir == w.root
		if info.Mode().IsRegular() && strings.HasSuffix(name, ".md") && !w.ig.Ignored(rel, false) && top != w.agents {
			w.files = append(w.files, full)
			continue
		}
		if info.IsDir() && !w.ig.Ignored(rel, true) {
			w.dir(full, depth+1)
		}
	}
}

// jsLen is a string's length as JavaScript counts it, so the limits are
// Pi's.
func jsLen(s string) int { return len(utf16.Encode([]rune(s))) }

// loadFile reads one candidate, as Pi's loadSkillFromFile does
// (skills.ts:277-345): a SKILL.md whose frontmatter does not parse is a
// warning; any other file without a description is skipped silently; a
// skill keeps its name and description warnings, and loads unless its
// description is missing.
func loadFile(path string) (Skill, []Diagnostic, bool) {
	declared := filepath.Base(path) == "SKILL.md"
	warn := func(msg string) Diagnostic { return Diagnostic{Path: path, Message: msg} }
	b, err := os.ReadFile(path) //nolint:gosec // G304: a skill file the scan found
	if err != nil {
		return Skill{}, []Diagnostic{warn(err.Error())}, false
	}
	fields, _, err := frontmatter.Parse(string(b))
	if err != nil {
		if declared {
			return Skill{}, []Diagnostic{warn(err.Error())}, false
		}
		return Skill{}, nil, false
	}
	desc, isString := fields["description"].(string)
	has := isString && frontmatter.Trim(desc) != ""
	if !declared && !has {
		return Skill{}, nil, false
	}
	var diags []Diagnostic
	switch {
	case !has:
		diags = append(diags, warn("description is required"))
	case jsLen(desc) > maxDescription:
		diags = append(diags, warn(fmt.Sprintf("description exceeds %d characters (%d)", maxDescription, jsLen(desc))))
	}
	dir := filepath.Dir(path)
	name, _ := fields["name"].(string) //nolint:errcheck // a name that is not a string is none
	if name == "" {
		name = filepath.Base(dir)
	}
	if n := jsLen(name); n > maxName {
		diags = append(diags, warn(fmt.Sprintf("name exceeds %d characters (%d)", maxName, n)))
	}
	if !namePattern.MatchString(name) {
		diags = append(diags, warn("name contains invalid characters (must be lowercase a-z, 0-9, hyphens only)"))
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		diags = append(diags, warn("name must not start or end with a hyphen"))
	}
	if strings.Contains(name, "--") {
		diags = append(diags, warn("name must not contain consecutive hyphens"))
	}
	if !has {
		return Skill{}, diags, false
	}
	hidden, _ := fields["disable-model-invocation"].(bool) //nolint:errcheck // only true hides a skill
	return Skill{Name: name, Description: desc, FilePath: path, BaseDir: dir, DisableModelInvocation: hidden}, diags, true
}

// Format is the skills section's text, Pi's formatSkillsForPrompt with the
// read tool (skills.ts:355-383), trimmed as Pi's prompt trims it; "" when
// every skill is hidden from the model.
func Format(skills []Skill) string {
	var lines []string
	for _, s := range skills {
		if s.DisableModelInvocation {
			continue
		}
		lines = append(lines, "  <skill>",
			"    <name>"+escape(s.Name)+"</name>",
			"    <description>"+escape(s.Description)+"</description>",
			"    <location>"+escape(s.FilePath)+"</location>",
			"  </skill>")
	}
	if len(lines) == 0 {
		return ""
	}
	head := []string{
		"The following skills provide specialized instructions for specific tasks.",
		"Use the read tool to load a skill's file when the task matches its description.",
		"When a skill file references a relative path, resolve it against the skill directory (parent of SKILL.md / dirname of the path) and use that absolute path in tool commands.",
		"",
		"<available_skills>",
	}
	return strings.Join(append(append(head, lines...), "</available_skills>"), "\n")
}

var escaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")

func escape(s string) string { return escaper.Replace(s) }

// Expand is Pi's /skill:<name> [args] (agent-session.ts:2062-2086): the
// name ends at the first space, and the skill's body goes in Pi's <skill>
// block, the arguments after it. A text that is not a skill command, an
// unknown name, or a file that cannot be read gives the text unchanged and
// false. A hidden skill still expands.
func Expand(text string, skills []Skill) (string, bool) {
	rest, ok := strings.CutPrefix(text, "/skill:")
	if !ok {
		return text, false
	}
	name, args, _ := strings.Cut(rest, " ")
	args = frontmatter.Trim(args)
	for _, s := range skills {
		if s.Name != name {
			continue
		}
		b, err := os.ReadFile(s.FilePath)
		if err != nil {
			return text, false
		}
		_, body, err := frontmatter.Parse(string(b))
		if err != nil {
			return text, false
		}
		block := `<skill name="` + s.Name + `" location="` + s.FilePath + `">` + "\nReferences are relative to " + s.BaseDir +
			".\n\n" + frontmatter.Trim(body) + "\n</skill>"
		if args != "" {
			block += "\n\n" + args
		}
		return block, true
	}
	return text, false
}
