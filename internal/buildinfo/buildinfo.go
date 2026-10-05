package buildinfo

import (
	"regexp"
	"runtime"
	"strings"
	"sync"

	lib "github.com/maccavelli/go-selfupdate-lib/buildinfo"
)

// devVersion is the version of any build that has no SemVer version of its
// own (0008-MADR D19 item 1).
const devVersion = "0.0.0-dev"

// semverRE is SemVer 2.0.0's regular expression (semver.org, section
// "Is there a suggested regular expression").
var semverRE = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// Info is the running binary's identity.
type Info struct {
	// Version always parses as SemVer: the release or module version
	// without its "v", else 0.0.0-dev[+<revision>[.dirty]].
	Version string
	// Commit is vcs.revision, or "" when the build has no VCS information.
	Commit string
	// Date is vcs.time, the commit time in RFC 3339, or "".
	Date string
	// Modified reports a build from a modified tree.
	Modified bool
	// Release reports a published release (go-selfupdate-lib's rule).
	Release bool
	// Go, OS and Arch are the toolchain and target.
	Go, OS, Arch string
	// Lib is go-selfupdate-lib's identity, for selfupdate/cli.
	Lib lib.Info
}

var identity = sync.OnceValue(func() Info {
	return fromLib(lib.Identity(), runtime.GOOS, runtime.GOARCH, runtime.Version())
})

// Identity returns the running binary's identity. It is computed once.
func Identity() Info { return identity() }

func fromLib(li lib.Info, goos, goarch, gover string) Info {
	return Info{
		Version:  semver(li),
		Commit:   li.Revision,
		Date:     li.Time,
		Modified: li.Modified,
		Release:  li.Kind == lib.KindRelease,
		Go:       strings.TrimPrefix(gover, "go"),
		OS:       goos,
		Arch:     goarch,
		Lib:      li,
	}
}

// pseudoRE matches the tail of a Go pseudo-version, which go build records as
// the module version of an unstamped build from a checkout (Go 1.24+), as
// in v0.0.0-20261005150633-55f35bac850a or v1.2.4-0.20261005150633-55f35bac850a.
var pseudoRE = regexp.MustCompile(`(^|[.-])\d{14}-[0-9a-f]{12}$`)

// semver keeps the stamped or module version only when it is a clean tag:
// valid SemVer, no build metadata, not a pseudo-version. A pseudo-version
// after a tag would claim the next patch release (0009-REPORT M7), so every
// other build is 0.0.0-dev+<revision>[.dirty] (0008-MADR D19 item 1).
func semver(li lib.Info) string {
	if v := strings.TrimPrefix(li.Current(), "v"); semverRE.MatchString(v) && !strings.Contains(v, "+") && !pseudoRE.MatchString(v) {
		return v
	}
	if li.Revision == "" {
		return devVersion
	}
	meta := li.Revision[:min(12, len(li.Revision))]
	if li.Modified {
		meta += ".dirty"
	}
	return devVersion + "+" + meta
}

// UserAgent is the HTTP User-Agent of 0003-MADR:
// gobble/<version> (<GOOS>; <GOARCH>) go/<go version>.
func (i Info) UserAgent() string {
	return "gobble/" + i.Version + " (" + i.OS + "; " + i.Arch + ") go/" + i.Go
}
