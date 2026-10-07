package cli

import (
	"encoding/json/v2"
	"fmt"

	"github.com/maccavelli/gobble-cli/internal/buildinfo"
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

// Run prints `gobble <version>` and the build details, one JSON object, or
// the identity line.
func (c *VersionCmd) Run(e *runEnv) error {
	i := identity()
	if c.Identity {
		return e.out.Result([]byte(i.Lib.String() + "\n"))
	}
	r := versionReport{Name: "gobble", Version: i.Version, Commit: i.Commit, Date: i.Date, Go: i.Go, OS: i.OS, Arch: i.Arch}
	if c.JSON {
		b, err := json.Marshal(r)
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
