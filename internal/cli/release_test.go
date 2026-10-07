package cli

import (
	"os"
	"slices"
	"testing"

	"github.com/maccavelli/go-selfupdate-lib/selfupdate"
	"github.com/maccavelli/go-selfupdate-lib/selfupdate/releasespec"
)

// The release spec the build workflow reads (0004-PLAN Phase 4, amendment
// of 2026-10-07) parses, and says what was decided: one product with its
// identity command, five raw-binary targets with no darwin/amd64, the SBOM
// as the one extra from the extras artifact, and stable tags only. 0005-PLAN
// F10 embeds the same file for the updater.
func TestReleaseSpec(t *testing.T) {
	b, err := os.ReadFile("selfupdate-release.json")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := releasespec.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	p, err := spec.Product("gobble")
	if err != nil {
		t.Fatal(err)
	}
	if p.Package != "./cmd/gobble" || !slices.Equal(p.IdentityArgs, []string{"version", "--identity"}) || len(p.Tags) != 0 || len(spec.Products) != 1 {
		t.Fatalf("products %+v", spec.Products)
	}
	want := []selfupdate.Platform{
		{OS: "darwin", Arch: "arm64"},
		{OS: "linux", Arch: "amd64"}, {OS: "linux", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"}, {OS: "windows", Arch: "arm64"},
	}
	if got := spec.Targets(); !slices.Equal(got, want) {
		t.Fatalf("targets %v, want %v", got, want)
	}
	if spec.Packaging != releasespec.PackagingBinary {
		t.Fatalf("packaging %q, want binary", spec.Packaging)
	}
	if len(spec.Extras) != 1 || spec.Extras[0].Name != "gobble.spdx.json" || spec.Extras[0].Path != "" {
		t.Fatalf("extras %+v, want gobble.spdx.json from the extras artifact", spec.Extras)
	}
	if len(spec.PrereleaseChannels) != 0 {
		t.Fatalf("prerelease channels %v; gobble update selects stable tags only (0005-MADR)", spec.PrereleaseChannels)
	}
}
