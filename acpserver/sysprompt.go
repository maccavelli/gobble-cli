package acpserver

import (
	"cmp"
	"encoding/json/v2"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/skill"
	"github.com/maccavelli/gobble-cli/tool"
)

// PromptOptions configure the system prompt (0005-PLAN F3-1).
type PromptOptions struct {
	// ConfigDir holds the user's AGENTS.md, SYSTEM.md and
	// APPEND_SYSTEM.md; "" reads none of them.
	ConfigDir string
	// System and Append, when set, replace SYSTEM.md's and
	// APPEND_SYSTEM.md's texts, as Pi's --system-prompt and
	// --append-system-prompt do (resource-loader.ts:645-655).
	System, Append string
	// NoContextFiles reads no context file, nested ones included.
	NoContextFiles bool
	// GOOS is the system the prompt says commands run on; "" is the host's
	// (0005-PLAN F3-1 deviation 5).
	GOOS string
	// Home holds ~/.agents/skills; "" reads none there. NoSkills reads no
	// skills at all (0005-PLAN F3-2).
	Home     string
	NoSkills bool
	// cwd, when set, is the cwd the prompt names in place of the
	// session's, so a golden's size is the same on every host; tests only
	// (0019-PLAN P4 deviation 4).
	cwd string
}

// sections are the session's system prompt sections as its files make
// them now: the agent's own tools, not the MCP servers', so they never wait
// for one (0005-PLAN F3-1 deviation 2). Each diagnostic is logged once per
// session.
func (a *Agent) sections(s *liveSession) []sysprompt.Section {
	o := a.prompt
	system, appended, waiting := sysprompt.SystemFiles(o.ConfigDir, s.cwd)
	if o.System != "" {
		system = o.System
	}
	if o.Append != "" {
		appended = o.Append
	}
	var files []sysprompt.File
	diags := waiting
	if !o.NoContextFiles {
		f, d := sysprompt.ContextFiles(o.ConfigDir, s.cwd)
		files, diags = f, append(diags, d...)
	}
	a.note(s, diags)
	loaded := loadedTools(s.tree.Path())
	var names, families []string
	for _, t := range a.tools {
		sp := t.Spec()
		switch {
		case sp.Exposure == tool.ExposureHidden:
		case sp.Exposure == tool.ExposureDeferred && !slices.Contains(loaded, sp.Name):
			if sp.Family != "" && !slices.Contains(families, sp.Family) {
				families = append(families, sp.Family)
			}
		default:
			names = append(names, sp.Name)
		}
	}
	slices.Sort(families)
	var skills string
	if slices.Contains(names, "read") {
		skills = skill.Format(a.skills(s))
	}
	goos := o.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	return sysprompt.Build(sysprompt.Options{Custom: system, Append: appended, Tools: names, Deferred: families,
		ContextFiles: files, Skills: skills, Cwd: cmp.Or(o.cwd, s.cwd), GOOS: goos})
}

// skills are the session's skills, found in 0005-MADR's order (Skill
// discovery order (changed)): the config directory's skills/, then
// ~/.agents/skills; the project's .gobble/skills and its .agents/skills,
// from the cwd up to the repository root, wait for trust (0005-PLAN F6),
// and the Pi bridge for F10. Each diagnostic is logged once per session.
func (a *Agent) skills(s *liveSession) []skill.Skill {
	o := a.prompt
	if o.NoSkills {
		return nil
	}
	var sources []skill.Source
	if o.ConfigDir != "" {
		sources = append(sources, skill.Source{Dir: filepath.Join(o.ConfigDir, "skills")})
	}
	if o.Home != "" {
		sources = append(sources, skill.Source{Dir: filepath.Join(o.Home, ".agents", "skills"), Agents: true})
	}
	found, diags := skill.Load(sources)
	notes := make([]sysprompt.Diagnostic, 0, len(diags))
	for _, d := range diags {
		notes = append(notes, sysprompt.Diagnostic{Path: d.Path, Message: d.Message, Winner: d.Winner})
	}
	// Home's own .agents/skills is the user's, never the project's, as Pi
	// excludes it by the real home directory, whether or not it is read.
	home := o.Home
	if home == "" {
		home, _ = os.UserHomeDir() //nolint:errcheck // with no home, none is excluded
	}
	for _, dir := range projectSkillDirs(s.cwd, home) {
		notes = append(notes, sysprompt.Diagnostic{Path: dir, Message: sysprompt.TrustNote})
	}
	a.note(s, notes)
	return found
}

// templates are the session's prompt templates (0005-PLAN F3-3): the config
// directory's prompts/; the project's .gobble/prompts waits for trust
// (0005-PLAN F6), and the Pi bridge for F10. A template that has a
// built-in command's name is left out, logged in Pi's collision wording
// with the built-in as the winner (F3-3 deviation 1). Each diagnostic is
// logged once per session.
func (a *Agent) templates(s *liveSession) []sysprompt.Template {
	o := a.prompt
	var dirs []string
	if o.ConfigDir != "" {
		dirs = append(dirs, filepath.Join(o.ConfigDir, "prompts"))
	}
	found, diags := sysprompt.LoadTemplates(dirs)
	builtins := a.handlers()
	out := make([]sysprompt.Template, 0, len(found))
	for _, t := range found {
		if _, ok := builtins[t.Name]; ok {
			diags = append(diags, sysprompt.Diagnostic{Path: t.FilePath, Message: `name "/` + t.Name + `" collision`, Winner: "the built-in command /" + t.Name})
			continue
		}
		out = append(out, t)
	}
	if p := filepath.Join(s.cwd, ".gobble", "prompts"); isDir(p) {
		diags = append(diags, sysprompt.Diagnostic{Path: p, Message: sysprompt.TrustNote})
	}
	a.note(s, diags)
	return out
}

// templatesOf are the templates found when s started.
func (a *Agent) templatesOf(s *liveSession) []sysprompt.Template {
	a.mu.Lock()
	defer a.mu.Unlock()
	return s.templates
}

// expand is Pi's expansion of a prompt's text (agent-session.ts:1922-1924):
// /skill:<name> into Pi's <skill> block, then a template of the name after
// "/" into its content (0005-PLAN F3-2, F3-3). ok is false when neither
// matched.
func (a *Agent) expand(s *liveSession, text string) (string, bool) {
	text, skilled := skill.Expand(text, a.skills(s))
	text, templated := sysprompt.ExpandTemplate(text, a.templatesOf(s))
	return text, skilled || templated
}

func isDir(p string) bool { info, err := os.Stat(p); return err == nil && info.IsDir() }

// projectSkillDirs are the project's own skill directories that exist: its
// .gobble/skills, and each .agents/skills from cwd up to the repository
// root, or the filesystem root outside one, but not home's, which is the
// user's (Pi's collectAncestorAgentsSkillDirs, package-manager.ts:468-487).
func projectSkillDirs(cwd, home string) []string {
	var out []string
	if p := filepath.Join(cwd, ".gobble", "skills"); isDir(p) {
		out = append(out, p)
	}
	stop := ""
	for d := filepath.Clean(cwd); ; {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			stop = d
			break
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	for d := filepath.Clean(cwd); ; {
		if p := filepath.Join(d, ".agents", "skills"); isDir(p) && (home == "" || filepath.Clean(d) != filepath.Clean(home)) {
			out = append(out, p)
		}
		parent := filepath.Dir(d)
		if d == stop || parent == d {
			break
		}
		d = parent
	}
	return out
}

// note logs each diagnostic once per session.
func (a *Agent) note(s *liveSession, diags []sysprompt.Diagnostic) {
	for _, d := range diags {
		if d.Winner != "" {
			d.Message += " (" + d.Winner + " is loaded)"
		}
		key := d.Path + "\x00" + d.Message
		a.mu.Lock()
		seen := s.noted[key]
		if s.noted == nil {
			s.noted = map[string]bool{}
		}
		s.noted[key] = true
		a.mu.Unlock()
		if !seen {
			a.logger.Warn("prompt: a file was not used", slog.String("file", d.Path), slog.String("reason", d.Message))
		}
	}
}

// promptState is the system prompt a path holds: its latest baseline, and
// the sections with that baseline's later patches applied. ok is false when
// the path has no baseline.
func promptState(path []session.Entry) (base, current []sysprompt.Section, ok bool) {
	for _, e := range path {
		if e.Type != session.TypeCustom || e.CustomType != session.CustomPrompt {
			continue
		}
		b, patch, err := sysprompt.ParseData(e.Data)
		switch {
		case err != nil:
		case b != nil:
			base, current, ok = b, b, true
		case ok:
			current = sysprompt.Apply(current, patch)
		}
	}
	return base, current, ok
}

func promptEntry(data []byte) session.Entry {
	return session.Entry{Type: session.TypeCustom, CustomType: session.CustomPrompt, Data: data}
}

// writePrompt records the system prompt before a turn's user message: a
// baseline when the path has none, else a patch when the sections changed.
// It returns the system prompt to send, the latest baseline's text, and the
// patch's update message, which joins the history at its place.
func (a *Agent) writePrompt(w *writer, s *liveSession) (string, *llm.Message) {
	cur := a.sections(s)
	base, current, ok := promptState(s.tree.Path())
	if !ok {
		w.add(promptEntry(sysprompt.BaselineData(cur)))
		return sysprompt.Text(cur), nil
	}
	patch := sysprompt.Diff(current, cur)
	if patch == nil {
		return sysprompt.Text(base), nil
	}
	w.add(promptEntry(sysprompt.PatchData(patch)))
	return sysprompt.Text(base), &llm.Message{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: sysprompt.Update(patch)}}}
}

// systemText is the system prompt the session's next request sends: its
// latest baseline's, or, before its first prompt, the one that prompt would
// write. Estimates count it.
func (a *Agent) systemText(s *liveSession) string {
	if base, _, ok := promptState(s.tree.Path()); ok {
		return sysprompt.Text(base)
	}
	return sysprompt.Text(a.sections(s))
}

// rebase writes a new baseline for a compaction, folding the patches in,
// just before the compaction entry (0005-MADR, amendment of 2026-10-10,
// item 1; 0005-PLAN F3-1 deviation 3).
func (a *Agent) rebase(s *liveSession, w func(session.Entry) error) error {
	return w(promptEntry(sysprompt.BaselineData(a.sections(s))))
}

// instructions is the read tool's Env.Instructions for s: the nested
// context files a read brings in that the session's context does not hold.
// A file is held when a gobble.context entry among the path's context
// entries names it, so a compaction that drops the entry drops the file
// (0005-MADR, amendment of 2026-10-10, item 5).
func (a *Agent) instructions(s *liveSession) func(string) []tool.Instruction {
	if a.prompt.NoContextFiles {
		return nil
	}
	return func(path string) []tool.Instruction {
		held := map[string]bool{}
		for _, e := range compaction.Context(s.tree.Path()) {
			if e.Type != session.TypeCustom || e.CustomType != session.CustomContext {
				continue
			}
			var d contextData
			if json.Unmarshal(e.Data, &d) == nil {
				for _, p := range d.Paths {
					held[p] = true
				}
			}
		}
		var out []tool.Instruction
		for _, f := range sysprompt.Nested(s.cwd, path) {
			if !held[f.Path] {
				out = append(out, tool.Instruction{Path: f.Path, Text: sysprompt.Instruction(f)})
			}
		}
		return out
	}
}

// contextData is a gobble.context entry's data.
type contextData struct {
	Paths []string `json:"paths"`
}

func contextEntry(paths []string) session.Entry {
	return session.Entry{Type: session.TypeCustom, CustomType: session.CustomContext, Data: mustJSON(contextData{Paths: paths})}
}
