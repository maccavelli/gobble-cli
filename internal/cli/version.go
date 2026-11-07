package cli

import (
	"fmt"

	"github.com/alecthomas/kong"

	"github.com/maccavelli/gobble-cli/internal/buildinfo"
	"github.com/maccavelli/gobble-cli/jsonx"
)

// identity is the build identity; tests replace it.
var identity = buildinfo.Identity

// VersionCmd prints the build identity.
type VersionCmd struct {
	JSON bool `name:"json" xor:"version-format" help:"Print JSON."`
	// Identity is the line go-selfupdate-lib's build workflow checks on each
	// platform's runner: buildinfo.Identity(), "<tag> (release) <commit>"
	// (0004-PLAN Phase 4, amendment of 2026-10-07).
	Identity bool `name:"identity" xor:"version-format" help:"Print only the build identity line the release workflow checks."`
}

type versionReport struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

// versionOf is the report of build identity i, which the version command
// and the --version flag print.
func versionOf(i buildinfo.Info) versionReport {
	return versionReport{Name: "gobble", Version: i.Version, Commit: i.Commit, Date: i.Date, Go: i.Go, OS: i.OS, Arch: i.Arch}
}

// versionFlag is the root's --version: gobble's own flag type rather than
// kong.VersionFlag, so its text is the version command's, read from the
// build identity only when asked (0008-MADR D21).
type versionFlag bool

// BeforeReset writes what `gobble version` writes and ends the parse with
// exit 0, before a command's arguments are checked or anything runs. A
// failed write is the parse's error.
func (versionFlag) BeforeReset(app *kong.Kong) error {
	if _, err := fmt.Fprint(app.Stdout, versionText(versionOf(identity()))); err != nil {
		return fmt.Errorf("--version: %w", err)
	}
	app.Exit(ExitOK)
	return nil
}

// Run prints `gobble <version>` and the build details, one JSON object, or
// the identity line.
func (c *VersionCmd) Run(e *runEnv) error {
	i := identity()
	if c.Identity {
		return e.out.Result([]byte(i.Lib.String() + "\n"))
	}
	r := versionOf(i)
	if c.JSON {
		b, err := jsonx.Marshal(r)
		if err != nil {
			return failf("version: %v", err)
		}
		return e.out.Result(append(b, '\n'))
	}
	return e.out.Result([]byte(versionText(r)))
}

func versionText(r versionReport) string {
	s := r.Name + " " + r.Version + "\n"
	if r.Commit != "" {
		s += fmt.Sprintf("commit: %s\n", r.Commit)
	}
	if r.Date != "" {
		s += fmt.Sprintf("date:   %s\n", r.Date)
	}
	return s + fmt.Sprintf("go:     %s %s/%s\n", r.Go, r.OS, r.Arch)
}
