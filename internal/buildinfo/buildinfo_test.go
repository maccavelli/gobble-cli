package buildinfo

import (
	"runtime"
	"strings"
	"testing"

	lib "github.com/maccavelli/go-selfupdate-lib/buildinfo"
)

const rev = "4e25c8a0123456789abcdef0123456789abcdef0"

func TestFromLib(t *testing.T) {
	cases := []struct {
		name    string
		li      lib.Info
		version string
		release bool
	}{
		{"release stamp", lib.Info{Version: "v1.2.3", Kind: lib.KindRelease, Revision: rev}, "1.2.3", true},
		{"prerelease stamp", lib.Info{Version: "v1.3.0-rc.2", Kind: lib.KindRelease}, "1.3.0-rc.2", true},
		{"go install at a tag", lib.Info{Kind: lib.KindLocal, ModuleVersion: "v0.4.1"}, "0.4.1", false},
		{"pseudo-version before any tag", lib.Info{Kind: lib.KindLocal, ModuleVersion: "v0.0.0-20261005131429-e825cdafd332", Revision: rev},
			"0.0.0-dev+4e25c8a01234", false},
		{"pseudo-version after a tag", lib.Info{Kind: lib.KindLocal, ModuleVersion: "v1.2.4-0.20261005131429-e825cdafd332", Revision: rev},
			"0.0.0-dev+4e25c8a01234", false},
		{"pseudo-version after a prerelease", lib.Info{Kind: lib.KindLocal, ModuleVersion: "v1.3.0-rc.1.0.20261005131429-e825cdafd332", Revision: rev},
			"0.0.0-dev+4e25c8a01234", false},
		{"dirty VCS module version", lib.Info{Kind: lib.KindLocal, ModuleVersion: "v0.0.0-20261005150633-55f35bac850a+dirty", Revision: rev, Modified: true},
			"0.0.0-dev+4e25c8a01234.dirty", false},
		{"stamped version with build metadata", lib.Info{Version: "v1.2.3+meta", Kind: lib.KindLocal, Revision: rev},
			"0.0.0-dev+4e25c8a01234", false},
		{"git describe without a tag", lib.Info{Version: "4e25c8a-dirty", Kind: lib.KindLocal, Revision: rev, Modified: true},
			"0.0.0-dev+4e25c8a01234.dirty", false},
		{"devel with a revision", lib.Info{Kind: lib.KindLocal, ModuleVersion: "(devel)", Revision: rev},
			"0.0.0-dev+4e25c8a01234", false},
		{"devel without a revision", lib.Info{Kind: lib.KindLocal, ModuleVersion: "(devel)"}, "0.0.0-dev", false},
		{"short revision", lib.Info{Kind: lib.KindLocal, Revision: "abc"}, "0.0.0-dev+abc", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fromLib(tc.li, "linux", "amd64", "go1.27.1")
			if got.Version != tc.version {
				t.Fatalf("Version = %q, want %q", got.Version, tc.version)
			}
			if !semverRE.MatchString(got.Version) {
				t.Fatalf("Version %q does not parse as SemVer", got.Version)
			}
			if got.Release != tc.release {
				t.Fatalf("Release = %v, want %v", got.Release, tc.release)
			}
			if got.Commit != tc.li.Revision || got.Modified != tc.li.Modified {
				t.Fatalf("Commit/Modified = %q/%v, want %q/%v", got.Commit, got.Modified, tc.li.Revision, tc.li.Modified)
			}
		})
	}
}

func TestUserAgent(t *testing.T) {
	i := fromLib(lib.Info{Version: "v1.2.3", Kind: lib.KindRelease}, "windows", "arm64", "go1.27.1")
	if got, want := i.UserAgent(), "gobble/1.2.3 (windows; arm64) go/1.27.1"; got != want {
		t.Fatalf("UserAgent = %q, want %q", got, want)
	}
}

// Under go test there are no stamps; the version must still be SemVer.
func TestIdentityIsSemVer(t *testing.T) {
	i := Identity()
	if !semverRE.MatchString(i.Version) {
		t.Fatalf("Identity().Version = %q is not SemVer", i.Version)
	}
	if i.OS != runtime.GOOS || i.Arch != runtime.GOARCH || strings.HasPrefix(i.Go, "go") {
		t.Fatalf("platform fields = %q %q %q", i.OS, i.Arch, i.Go)
	}
	if Identity() != i {
		t.Fatal("Identity is not stable across calls")
	}
}
