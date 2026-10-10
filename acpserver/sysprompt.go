package acpserver

import (
	"encoding/json/v2"
	"log/slog"
	"runtime"
	"slices"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/session"
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
	goos := o.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	return sysprompt.Build(sysprompt.Options{Custom: system, Append: appended, Tools: names, Deferred: families,
		ContextFiles: files, Cwd: s.cwd, GOOS: goos})
}

// note logs each diagnostic once per session.
func (a *Agent) note(s *liveSession, diags []sysprompt.Diagnostic) {
	for _, d := range diags {
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
