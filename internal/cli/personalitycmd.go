package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/personality"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// PersonalityCmd is gobble personality: the personalities and the user's
// files that change gobble's text (0022-MADR D15).
type PersonalityCmd struct {
	List  PersonalityListCmd  `cmd:"" help:"List the personalities, their levels and where they come from."`
	Show  PersonalityShowCmd  `cmd:"" help:"Print a personality's brief, or one of its messages."`
	Check PersonalityCheckCmd `cmd:"" help:"Check every file that changes gobble's text."`
	Path  PersonalityPathCmd  `cmd:"" help:"Print the directory the personalities are in."`
	Reset PersonalityResetCmd `cmd:"" help:"Remove your files for a personality, or for one prompt, after asking."`
}

// PersonalityListCmd is gobble personality list.
type PersonalityListCmd struct{}

// PersonalityShowCmd is gobble personality show NAME [KEY].
type PersonalityShowCmd struct {
	Name string `arg:"" complete:"personality" help:"The personality."`
	Key  string `arg:"" optional:"" complete:"none" help:"A message's key, such as retry.offline."`
}

// PersonalityCheckCmd is gobble personality check.
type PersonalityCheckCmd struct{}

// PersonalityPathCmd is gobble personality path.
type PersonalityPathCmd struct{}

// PersonalityResetCmd is gobble personality reset NAME|prompts/ID.
type PersonalityResetCmd struct {
	Target string `arg:"" complete:"none" placeholder:"NAME|prompts/ID" help:"A personality, or prompts/ and a prompt's id."`
}

// personalitiesDir is <config>/personalities.
func personalitiesDir() (string, error) {
	dirs, _, err := systemDirs()
	if err != nil {
		return "", failf("%v", err)
	}
	return filepath.Join(dirs.Config, "personalities"), nil
}

// The prompt ids that are not tool descriptions: the system prompt's
// sections the user may replace (0022-PLAN P5 step 2).
const (
	promptTools = "tools"
	promptRules = "rules"
	toolPrefix  = "tool."
)

// promptIDs are every prompt a file under personalities/prompts may
// replace: the tools and rules sections, compaction's prompts (0005-PLAN
// F5-4 decision 8), and each built-in tool's description as tool.<name>,
// powershell's on every host.
func promptIDs() []string {
	ids := append([]string{promptTools, promptRules}, compaction.PromptIDs()...)
	names := builtin.Names()
	if !slices.Contains(names, "powershell") {
		names = append(names, "powershell")
	}
	for _, n := range names {
		ids = append(ids, toolPrefix+n)
	}
	return ids
}

// promptOverrides are the user's model-facing texts, by prompt id, each
// checked: a section by prompt.CheckSection, a compaction prompt by
// compaction.CheckPrompt, a description by builtin.CheckDescription with
// o's limits.
type promptOverrides struct {
	sections     map[string]string
	descriptions map[string]string // by tool name, the template
	compaction   compaction.Prompts
	count        int // the files taken
}

// loadPrompts reads dir's <id>.md files. A file with an id gobble does not
// have, or one that fails its check, is left out with an error naming the
// file and the rule (0022-MADR D15). A missing dir is no overrides.
func loadPrompts(dir string, o builtin.Options) (promptOverrides, []error) {
	out := promptOverrides{sections: map[string]string{}, descriptions: map[string]string{}}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, []error{fmt.Errorf("personality prompts: %w", err)}
	}
	ids := promptIDs()
	var errs []error
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		id, ok := strings.CutSuffix(e.Name(), ".md")
		switch {
		case e.IsDir():
			continue
		case !ok || !slices.Contains(ids, id):
			errs = append(errs, fmt.Errorf("prompt %s: no prompt has this name; the prompts are %s", path, strings.Join(ids, ", ")))
			continue
		}
		data, err := os.ReadFile(path) //nolint:gosec // the user's own config directory
		if err != nil {
			errs = append(errs, fmt.Errorf("prompt %s: %w", path, err))
			continue
		}
		text := string(data)
		if name, ok := strings.CutPrefix(id, toolPrefix); ok {
			if _, err := builtin.CheckDescription(name, text, o); err != nil {
				errs = append(errs, fmt.Errorf("prompt %s: 0012-MADR D5: %w", path, err))
				continue
			}
			out.descriptions[name] = text
			out.count++
			continue
		}
		if slices.Contains(compaction.PromptIDs(), id) {
			if err := out.compaction.Set(id, text); err != nil {
				errs = append(errs, fmt.Errorf("prompt %s: %w", path, err))
				continue
			}
			out.count++
			continue
		}
		if err := sysprompt.CheckSection(id, text); err != nil {
			errs = append(errs, fmt.Errorf("prompt %s: %w", path, err))
			continue
		}
		out.sections[id] = strings.TrimSpace(text)
		out.count++
	}
	return out, errs
}

// prompts are the user's prompt overrides, read once, each problem a
// warning; the descriptions are checked with the process settings' limits,
// and again where the tools are built.
func (e *runEnv) prompts() promptOverrides {
	e.promptsOnce.Do(func() {
		e.promptSet = promptOverrides{sections: map[string]string{}, descriptions: map[string]string{}}
		dir, err := personalitiesDir()
		if err != nil {
			return
		}
		var errs []error
		e.promptSet, errs = loadPrompts(filepath.Join(dir, personality.PromptsDir), builtinOptions(e.processSettings("").Settings))
		for _, err := range errs {
			e.log().Warn("personality", slog.String("error", err.Error()))
		}
	})
	return e.promptSet
}

// sourceOf is where a personality comes from, as list shows it.
func sourceOf(info personality.Info) string {
	switch {
	case info.Source == "":
		return "built-in"
	case info.Builtin:
		return "built-in, changed by " + info.Source
	}
	return info.Source
}

// Run lists the personalities.
func (*PersonalityListCmd) Run(e *runEnv) error {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "NAME\tLEVEL\tSOURCE\tSUMMARY"); err != nil {
		return failf("%v", err)
	}
	for _, info := range e.personalities().Infos() {
		if _, err := fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", info.Name, info.Level, sourceOf(info), info.Summary); err != nil {
			return failf("%v", err)
		}
	}
	if err := w.Flush(); err != nil {
		return failf("%v", err)
	}
	return e.out.Result([]byte(b.String()))
}

// Run prints the personality's brief, or one key's lines as it says them,
// with their placeholders.
func (c *PersonalityShowCmd) Run(e *runEnv) error {
	set := e.personalities()
	name := set.Canonical(c.Name)
	if name == "" {
		return usageErrorf("no personality is called %s; the personalities are %s", c.Name, strings.Join(set.Names(), ", "))
	}
	v, err := set.New(personality.Options{Name: name})
	if err != nil {
		return failf("%v", err)
	}
	var info personality.Info
	for _, i := range set.Infos() {
		if i.Name == name {
			info = i
		}
	}
	if c.Key == "" {
		brief := v.Brief()
		if brief == "" {
			brief = "boring has no brief: gobble's plain text, with no humor."
		}
		return e.out.Resultf("%s, level %d: %s\nsource: %s\n\n%s\n", name, info.Level, info.Summary, sourceOf(info), brief)
	}
	key := personality.Key(c.Key)
	if !slices.Contains(personality.Keys, key) && !slices.Contains(personality.LeadKeys, key) && !slices.Contains(personality.AmbientKeys, key) {
		return usageErrorf("no message is called %s", c.Key)
	}
	lines := v.Variants(key, nil)
	if len(lines) == 0 {
		return e.out.Resultf("%s has no %s\n", name, key)
	}
	var b strings.Builder
	for _, l := range lines {
		b.WriteString("- " + l + "\n")
	}
	return e.out.Result([]byte(b.String()))
}

// Run checks the personalities and the prompts, prints each problem, and
// exits 1 when there is one. A trusted project's personality files are
// checked too; an untrusted project's are reported as held back, which is
// no problem (0005-PLAN F6-1 decision 8).
func (*PersonalityCheckCmd) Run(e *runEnv) error {
	dir, err := personalitiesDir()
	if err != nil {
		return err
	}
	set, errs := personality.LoadSet(dir)
	o := builtinOptions(e.processSettings("").Settings)
	prompts, perrs := loadPrompts(filepath.Join(dir, personality.PromptsDir), o)
	errs = append(errs, perrs...)
	var held string
	if cwd, err := os.Getwd(); err == nil {
		if p := filepath.Join(cwd, ".gobble", "personalities"); isDir(p) {
			if _, d := e.projectTrust(cwd); d.Trusted {
				_, perrs := set.WithProject(p)
				project, qerrs := loadPrompts(filepath.Join(p, personality.PromptsDir), o)
				errs = append(append(errs, perrs...), qerrs...)
				prompts.count += project.count
			} else {
				held = fmt.Sprintf("%s: %s\n", p, sysprompt.TrustNote)
			}
		}
	}
	var b strings.Builder
	for _, err := range errs {
		b.WriteString(err.Error() + "\n")
	}
	b.WriteString(held)
	changed := 0
	for _, info := range set.Infos() {
		if info.Source != "" {
			changed++
		}
	}
	fmt.Fprintf(&b, "%d problems; %d personalities yours or changed, %d prompts\n", len(errs), changed, prompts.count)
	if err := e.out.Result([]byte(b.String())); err != nil {
		return err
	}
	if len(errs) > 0 {
		return &exitError{code: ExitFailure}
	}
	return nil
}

// Run prints the personalities' directory.
func (*PersonalityPathCmd) Run(e *runEnv) error {
	dir, err := personalitiesDir()
	if err != nil {
		return err
	}
	return e.out.Resultf("%s\n", dir)
}

// Run removes the user's files for a personality, its personality.md and
// messages.md and then its directory when nothing else is in it, or one
// prompt's file, once the user says yes (0022-MADR D15, goose's reset).
func (c *PersonalityResetCmd) Run(e *runEnv) error {
	dir, err := personalitiesDir()
	if err != nil {
		return err
	}
	var files []string
	var holder string
	if id, ok := strings.CutPrefix(c.Target, personality.PromptsDir+"/"); ok {
		if !slices.Contains(promptIDs(), id) {
			return usageErrorf("no prompt is called %s; the prompts are %s", id, strings.Join(promptIDs(), ", "))
		}
		files = existing(filepath.Join(dir, personality.PromptsDir, id+".md"))
	} else {
		name := c.Target
		if canon := personality.Canonical(name); canon != "" {
			name = canon
		}
		if name == personality.PromptsDir || name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
			return usageErrorf("reset takes a personality's name, or prompts/ and a prompt's id")
		}
		holder = filepath.Join(dir, name)
		files = existing(filepath.Join(holder, "personality.md"), filepath.Join(holder, "messages.md"))
	}
	if len(files) == 0 {
		return failf("%s: none of your files is there; nothing to reset", c.Target)
	}
	ok, err := confirm(e, "Remove "+strings.Join(files, " and ")+"?")
	if err != nil {
		return failf("%v", err)
	}
	if !ok {
		return e.out.Resultf("Nothing removed.\n")
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			return failf("%v", err)
		}
	}
	if holder != "" {
		if left, err := os.ReadDir(holder); err == nil && len(left) == 0 {
			if err := os.Remove(holder); err != nil {
				return failf("%v", err)
			}
		}
	}
	return e.out.Resultf("Removed %s.\n", strings.Join(files, " and "))
}

func isDir(p string) bool { info, err := os.Stat(p); return err == nil && info.IsDir() }

// existing are those of paths that are files.
func existing(paths ...string) []string {
	var out []string
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.Mode().IsRegular() {
			out = append(out, p)
		}
	}
	return out
}
