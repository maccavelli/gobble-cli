package prompt

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// data is a gobble.prompt entry's data: a baseline or a patch, as arrays
// so the order holds, which a JSON object's would not.
type data struct {
	Baseline []dataSection `json:"baseline,omitzero"`
	Patch    []dataSection `json:"patch,omitzero"`
}

type dataSection struct {
	Name string  `json:"name"`
	Text *string `json:"text"`
}

// BaselineData is a baseline's entry data.
func BaselineData(sections []Section) jsontext.Value {
	d := data{Baseline: make([]dataSection, len(sections))}
	for i, s := range sections {
		d.Baseline[i] = dataSection{Name: s.Name, Text: new(s.Text)}
	}
	return mustJSON(d)
}

// PatchData is a patch's entry data.
func PatchData(patch []Patch) jsontext.Value {
	d := data{Patch: make([]dataSection, len(patch))}
	for i, p := range patch {
		d.Patch[i] = dataSection(p)
	}
	return mustJSON(d)
}

func mustJSON(v any) jsontext.Value {
	b, err := json.Marshal(v)
	if err != nil { // plain strings always encode
		panic(err)
	}
	return b
}

// ParseData reads a gobble.prompt entry's data: a baseline, or else a
// patch.
func ParseData(v jsontext.Value) (baseline []Section, patch []Patch, err error) {
	var d data
	if err := json.Unmarshal(v, &d); err != nil {
		return nil, nil, err
	}
	switch {
	case d.Baseline != nil:
		for _, s := range d.Baseline {
			if s.Text == nil {
				return nil, nil, errors.New("prompt: a baseline section with no text")
			}
			baseline = append(baseline, Section{Name: s.Name, Text: *s.Text})
		}
		return baseline, nil, nil
	case d.Patch != nil:
		for _, s := range d.Patch {
			patch = append(patch, Patch(s))
		}
		return nil, patch, nil
	}
	return nil, nil, errors.New("prompt: neither a baseline nor a patch")
}

// Section is one named part of the system prompt, as Pi stores it: the
// preamble bare, every other section wrapped in a tag of its name,
// "<name>\n…\n</name>", so a later update can be matched to it (Pi's
// system-prompt.ts:45-50, :175-178).
type Section struct {
	Name, Text string
}

// Patch changes one section: Text replaces it in its place, or adds it at
// the end; a nil Text removes it.
type Patch struct {
	Name string
	Text *string
}

// Preamble is gobble's opening sentence, which a custom prompt replaces.
const Preamble = "You are gobble, a coding agent. You help the user with software engineering tasks in their project."

// Options are what a system prompt is built from.
type Options struct {
	// Custom replaces the preamble, and leaves out tools and rules, as Pi's
	// SYSTEM.md does.
	Custom string
	// Personality is the session's personality brief, and PersonalityRules
	// the house rules that outrank it; both empty under boring (0022-MADR
	// D3, D4).
	Personality, PersonalityRules string
	// Append is the addendum, Pi's APPEND_SYSTEM.md.
	Append string
	// Tools are the agent's own tools' names, which the rules depend on.
	Tools []string
	// Deferred are the families of deferred tools tool_search can find.
	Deferred []string
	// ContextFiles are the project's instructions, in order.
	ContextFiles []File
	// Skills is the skills section's text, or "" for none.
	Skills string
	// Cwd is the working directory; GOOS the system commands run on.
	Cwd, GOOS string
	// ToolsSection and RulesSection, when set, are the user's text for the
	// tools and rules sections, in place of gobble's (0022-MADR D15). The
	// tools section still needs tools, and neither is under a custom
	// prompt.
	ToolsSection, RulesSection string
}

// CheckSection checks the user's text for the section name, tools or
// rules: it has words, and does not close its own tag (0022-MADR D15).
func CheckSection(name, text string) error {
	switch {
	case name != "tools" && name != "rules":
		return fmt.Errorf("no section %s is the user's to replace", name)
	case strings.TrimSpace(text) == "":
		return errors.New("it is empty")
	case strings.Contains(text, "</"+name+">"):
		return fmt.Errorf("it closes the section's tag, </%s>", name)
	}
	return nil
}

// Build is the system prompt's sections in Pi's order, without docs, which
// gobble has none of yet: preamble, tools, rules, addendum,
// project_context, skills, cwd (Pi's buildSystemPromptSections,
// system-prompt.ts:121-180; 0005-PLAN F3-1). The personality and its house
// rules follow the preamble, a custom one included (0022-MADR D4).
func Build(o Options) []Section {
	out := []Section{{Name: "preamble", Text: Preamble}}
	if o.Custom != "" {
		out[0].Text = o.Custom
	}
	if o.Personality != "" {
		out = append(out, tagged("personality", o.Personality))
	}
	if o.PersonalityRules != "" {
		out = append(out, tagged("personality_rules", o.PersonalityRules))
	}
	if o.Custom == "" {
		if t := toolsText(o.Tools, o.Deferred); t != "" {
			out = append(out, tagged("tools", cmp.Or(strings.TrimSpace(o.ToolsSection), t)))
		}
		out = append(out, tagged("rules", cmp.Or(strings.TrimSpace(o.RulesSection), rules(o.Tools, o.GOOS))))
	}
	if o.Append != "" {
		out = append(out, tagged("addendum", o.Append))
	}
	if len(o.ContextFiles) > 0 {
		out = append(out, tagged("project_context", renderContext(o.ContextFiles)))
	}
	if o.Skills != "" {
		out = append(out, tagged("skills", o.Skills))
	}
	return append(out, tagged("cwd", strings.ReplaceAll(o.Cwd, `\`, "/")))
}

func tagged(name, text string) Section {
	return Section{Name: name, Text: "<" + name + ">\n" + text + "\n</" + name + ">"}
}

// toolsText is 0012-MADR D6's cross-tool guidance, for the tools there are,
// the families tool_search can find, and a closing line; "" with no tools.
func toolsText(tools, deferred []string) string {
	if len(tools) == 0 {
		return ""
	}
	has := func(n string) bool { return slices.Contains(tools, n) }
	var lines []string
	var ops []string
	for _, n := range []string{"move", "delete", "copy", "mkdir"} {
		if has(n) {
			ops = append(ops, n)
		}
	}
	if len(ops) > 0 && (has("bash") || has("powershell")) {
		lines = append(lines, "- Prefer "+list(ops)+" to shell commands for moving, deleting, copying and creating directories.")
	}
	if has("read") && has("edit") {
		lines = append(lines, "- Read a file before you edit it.")
	}
	if has("delete") {
		lines = append(lines, "- A delete always asks the user first.")
	}
	if len(deferred) > 0 {
		lines = append(lines, "- tool_search can find more tools for: "+strings.Join(deferred, ", ")+".")
	}
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, "Each request describes the tools you can call; you may also have tools from MCP servers and the project.")
	return strings.Join(lines, "\n")
}

// list is names as English: "a", "a and b", "a, b and c".
func list(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// rules is Pi's buildRules (system-prompt.ts:81-118) with gobble's own:
// the shell rule when no search tool is offered, then the standing rules,
// each once, as "- rule" lines.
func rules(tools []string, goos string) string {
	has := func(n string) bool { return slices.Contains(tools, n) }
	var out []string
	add := func(r string) {
		if r = strings.TrimSpace(r); r != "" && !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	if (has("bash") || has("powershell")) && !has("grep") && !has("find") && !has("tree") {
		switch {
		case has("bash") && has("powershell"):
			add("Use bash or PowerShell for file operations like listing, searching, and finding files")
		case has("powershell"):
			add("Use PowerShell for file operations like listing, searching, and finding files")
		default:
			add("Use bash for file operations like ls, rg, find")
		}
	}
	add("A call to a tool that changes something may be declined by the user; then do not retry it, and say what you would have done.")
	add("Commands run on " + goos + ".")
	add("Be concise in your responses")
	add("Show file paths clearly when working with files")
	add("Answer in Markdown.")
	lines := make([]string, len(out))
	for i, r := range out {
		lines[i] = "- " + r
	}
	return strings.Join(lines, "\n")
}

// Text is the prompt the sections make: their texts joined by blank lines,
// as Pi's getSystemMessageText joins them.
func Text(sections []Section) string {
	var parts []string
	for _, s := range sections {
		if s.Text != "" {
			parts = append(parts, s.Text)
		}
	}
	return strings.Join(parts, "\n\n")
}

// Diff is the patch from prev to cur, Pi's diffSystemPromptSections
// (system-prompt.ts:204-216): each changed or new section in cur's order,
// then each removed one in prev's; nil when nothing changed.
func Diff(prev, cur []Section) []Patch {
	var out []Patch
	for _, c := range cur {
		at := slices.IndexFunc(prev, func(p Section) bool { return p.Name == c.Name })
		if at < 0 || prev[at].Text != c.Text {
			out = append(out, Patch{Name: c.Name, Text: new(c.Text)})
		}
	}
	for _, p := range prev {
		if !slices.ContainsFunc(cur, func(c Section) bool { return c.Name == p.Name }) {
			out = append(out, Patch{Name: p.Name})
		}
	}
	return out
}

// Apply is base with a patch applied, as Pi replays one
// (ai/src/utils/transcript.ts:73-96): a replaced section keeps its place, a
// new one goes last, a nil one is removed. base is not changed.
func Apply(base []Section, patch []Patch) []Section {
	out := slices.Clone(base)
	for _, p := range patch {
		at := slices.IndexFunc(out, func(s Section) bool { return s.Name == p.Name })
		switch {
		case p.Text == nil && at >= 0:
			out = slices.Delete(out, at, at+1)
		case p.Text == nil:
		case at >= 0:
			out[at].Text = *p.Text
		default:
			out = append(out, Section{Name: p.Name, Text: *p.Text})
		}
	}
	return out
}

// Update is a patch as the model reads it mid-conversation, framed by name
// as Pi frames it (ai/src/utils/text.ts:28-40).
func Update(patch []Patch) string {
	parts := make([]string, len(patch))
	for i, p := range patch {
		if p.Text == nil {
			parts[i] = `Removed system prompt section "` + p.Name + `".`
		} else {
			parts[i] = `Updated system prompt section "` + p.Name + `":` + "\n\n" + *p.Text
		}
	}
	return strings.Join(parts, "\n\n")
}
