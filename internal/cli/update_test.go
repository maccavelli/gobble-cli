package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	lib "github.com/maccavelli/go-selfupdate-lib/buildinfo"
	"github.com/maccavelli/go-selfupdate-lib/selfupdate"
	"github.com/maccavelli/go-selfupdate-lib/selfupdate/selfupdatetest"

	"github.com/maccavelli/gobble-cli/internal/buildinfo"
	"github.com/maccavelli/gobble-cli/jsonx"
)

// testBinary is this test binary's bytes: an executable image for the
// running platform, which the image verifier accepts.
var testBinary = sync.OnceValues(func() ([]byte, error) {
	p, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
})

// updateFixture serves gobble's releases on loopback and points the
// updater at them: its API, a target file under a scratch home, and a
// probe that records what it ran (0048-MADR D7).
type updateFixture struct {
	srv    *selfupdatetest.GitHubServer
	gobble string // GOBBLE_HOME
	target string // the file an install replaces
	old    []byte // target's bytes before the run
	// probeVersion is what the probe prints as "version".
	probeVersion string

	mu     sync.Mutex
	probed []string
}

// runningPlatform reports whether the running platform is a release target.
func runningPlatform(t *testing.T) bool {
	t.Helper()
	spec, err := releaseSpec()
	if err != nil {
		t.Fatal(err)
	}
	return slices.Contains(spec.Targets(), selfupdate.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
}

// gobbleRelease is release tag of every target, the running platform's
// asset being body, or a few bytes of text when body is nil.
func gobbleRelease(t *testing.T, tag string, body []byte) selfupdatetest.ReleaseSpec {
	t.Helper()
	spec, err := releaseSpec()
	if err != nil {
		t.Fatal(err)
	}
	return selfupdatetest.NewRelease(updateProduct, tag, spec.Targets(), func(p selfupdate.Platform) []byte {
		if body != nil && p.OS == runtime.GOOS && p.Arch == runtime.GOARCH {
			return body
		}
		return []byte("gobble " + tag + " " + p.OS + "/" + p.Arch + "\n")
	})
}

// releaseIdentity makes the running identity a release build of version v.
func releaseIdentity(t *testing.T, v string) {
	t.Helper()
	old := identity
	t.Cleanup(func() { identity = old })
	identity = func() buildinfo.Info {
		return buildinfo.Info{Version: strings.TrimPrefix(v, "v"), Release: true, Go: "1.27.3",
			OS: runtime.GOOS, Arch: runtime.GOARCH, Lib: lib.Info{Version: v, Kind: lib.KindRelease}}
	}
}

// newUpdateFixture serves releases, runs as release current, and installs
// to a file under a scratch home, outside the network guard's reach.
func newUpdateFixture(t *testing.T, current string, releases ...selfupdatetest.ReleaseSpec) *updateFixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("AI_AGENT", "")
	releaseIdentity(t, current)
	name := "gobble"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	f := &updateFixture{
		srv:    selfupdatetest.NewGitHubServer(t, updateOwner, updateRepo, releases...),
		gobble: t.TempDir(),
		target: filepath.Join(home, ".local", "bin", name),
		old:    []byte("gobble " + current + "\n"),
	}
	if err := os.MkdirAll(filepath.Dir(f.target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.target, f.old, 0o755); err != nil { //nolint:gosec // an executable's mode
		t.Fatal(err)
	}
	old := updaterDeps
	t.Cleanup(func() { updaterDeps = old })
	updaterDeps = updateDeps{
		apiBase:  f.srv.APIBase,
		client:   f.srv.Client,
		target:   selfupdate.TargetPolicy{ExecutablePath: f.target},
		probeRun: f.probe,
	}
	return f
}

func (f *updateFixture) probe(_ context.Context, path string, _ []string, stdout io.Writer) error {
	f.mu.Lock()
	f.probed = append(f.probed, path)
	v := f.probeVersion
	f.mu.Unlock()
	_, err := io.WriteString(stdout, `{"name":"gobble","version":"`+v+`"}`+"\n")
	return err
}

func (f *updateFixture) run(t *testing.T, args ...string) result {
	t.Helper()
	return runIn(t, f.gobble, append([]string{"update"}, args...)...)
}

// unchanged fails the test when the target's bytes changed.
func (f *updateFixture) unchanged(t *testing.T) {
	t.Helper()
	b, err := os.ReadFile(f.target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, f.old) {
		t.Fatalf("target changed: %d bytes", len(b))
	}
}

func TestUpdateHelp(t *testing.T) {
	r := runMain(t, nil, "update", "--help")
	if r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	for _, want := range []string{"--check", "--yes", "-y", "--force", "--dry-run", "--json", "--release=vX.Y.Z", "Exit status:"} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("help lacks %q:\n%s", want, r.stdout)
		}
	}
	if strings.Contains(r.stdout, "--channel") {
		t.Errorf("help offers --channel:\n%s", r.stdout)
	}
	k, err := newParser(&Root{}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range k.Model.Children {
		if c.Name == "update" {
			for _, fl := range c.Flags {
				names = append(names, fl.Name)
			}
		}
	}
	slices.Sort(names)
	if want := []string{"check", "dry-run", "force", "json", "release", "yes"}; !slices.Equal(names, want) {
		t.Errorf("update's own flags %v, want %v", names, want)
	}
}

func TestUpdateCheck(t *testing.T) {
	f := newUpdateFixture(t, "v0.3.1", gobbleRelease(t, "v0.3.1", nil), gobbleRelease(t, "v0.4.0", nil))
	r := f.run(t, "--check")
	if r.code != ExitUpdateAvailable || r.stdout != "" || !strings.Contains(r.stderr, "gobble: update available: v0.3.1 -> v0.4.0\n") {
		t.Fatalf("exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	f.unchanged(t)

	f = newUpdateFixture(t, "v0.4.0", gobbleRelease(t, "v0.3.1", nil), gobbleRelease(t, "v0.4.0", nil))
	r = f.run(t, "--check")
	if r.code != ExitOK || r.stdout != "" || !strings.Contains(r.stderr, "gobble: up to date (v0.4.0)\n") {
		t.Fatalf("exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
}

func TestUpdateJSON(t *testing.T) {
	f := newUpdateFixture(t, "v0.3.1", gobbleRelease(t, "v0.4.0", nil))
	r := f.run(t, "--check", "--json")
	lines := strings.Split(strings.TrimSuffix(r.stdout, "\n"), "\n")
	var last struct {
		Kind     string `json:"kind"`
		ExitCode int    `json:"exit_code"`
	}
	for _, l := range lines {
		var v map[string]any
		if err := jsonx.Unmarshal([]byte(l), &v); err != nil {
			t.Fatalf("not a JSON line: %q: %v", l, err)
		}
	}
	if err := jsonx.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		t.Fatal(err)
	}
	if r.code != ExitUpdateAvailable || last.Kind != "result" || last.ExitCode != r.code {
		t.Fatalf("exit %d, last line %+v, stdout %q", r.code, last, r.stdout)
	}
}

func TestUpdateContradiction(t *testing.T) {
	f := newUpdateFixture(t, "v0.3.1", gobbleRelease(t, "v0.4.0", nil))
	for _, extra := range []string{"--yes", "--force", "--dry-run"} {
		r := f.run(t, "--check", extra)
		if r.code != ExitUsage || !strings.Contains(r.stderr, "contradictory") {
			t.Errorf("--check %s: exit %d, stderr %q", extra, r.code, r.stderr)
		}
	}
	if n := len(f.srv.Requests()); n != 0 {
		t.Errorf("the server saw %d requests", n)
	}
}

func TestUpdateRelease(t *testing.T) {
	f := newUpdateFixture(t, "v0.1.0", gobbleRelease(t, "v0.2.0", nil), gobbleRelease(t, "v0.3.0", nil))
	if r := f.run(t, "--check"); r.code != ExitUpdateAvailable || !strings.Contains(r.stderr, "v0.1.0 -> v0.3.0\n") {
		t.Fatalf("latest: exit %d, stderr %q", r.code, r.stderr)
	}
	if r := f.run(t, "--check", "--release", "v0.2.0"); r.code != ExitUpdateAvailable || !strings.Contains(r.stderr, "v0.1.0 -> v0.2.0\n") {
		t.Fatalf("--release v0.2.0: exit %d, stderr %q", r.code, r.stderr)
	}
}

func TestUpdateApply(t *testing.T) {
	if !runningPlatform(t) {
		t.Skipf("%s/%s is not a release target", runtime.GOOS, runtime.GOARCH)
	}
	bin, err := testBinary()
	if err != nil {
		t.Fatal(err)
	}
	f := newUpdateFixture(t, "v0.3.1", gobbleRelease(t, "v0.4.0", bin))
	f.probeVersion = "0.4.0"
	r := f.run(t, "--yes")
	if r.code != ExitOK || !strings.Contains(r.stderr, "gobble: updated v0.3.1 -> v0.4.0\n") {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	got, err := os.ReadFile(f.target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bin) {
		t.Fatalf("target is %d bytes, not the release's %d", len(got), len(bin))
	}
	installed, err := filepath.EvalSymlinks(f.target)
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	probed := slices.Clone(f.probed)
	f.mu.Unlock()
	if len(probed) < 2 || probed[0] == installed || probed[len(probed)-1] != installed {
		t.Fatalf("probed %q; want the staged file, then %q", probed, installed)
	}
}

func TestUpdateRefusesBadImage(t *testing.T) {
	if !runningPlatform(t) {
		t.Skipf("%s/%s is not a release target", runtime.GOOS, runtime.GOARCH)
	}
	f := newUpdateFixture(t, "v0.3.1", gobbleRelease(t, "v0.4.0", []byte("#!/bin/sh\necho not gobble\n")))
	f.probeVersion = "0.4.0"
	r := f.run(t, "--yes")
	if r.code != ExitFailure || !strings.Contains(r.stderr, "update failed: ") || !strings.Contains(r.stderr, "integrity") {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	f.unchanged(t)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestUpdateIgnoresArchive(t *testing.T) {
	name := updateProduct + "-" + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"
	body := []byte("not a tarball")
	rel := selfupdatetest.ReleaseSpec{Tag: "v0.4.0", Immutable: true, Assets: []selfupdatetest.AssetSpec{
		{Name: name, Body: body},
		{Name: "SHA256SUMS", Body: []byte(sha256Hex(body) + "  " + name + "\n")},
	}}
	f := newUpdateFixture(t, "v0.3.1", rel)
	r := f.run(t, "--yes")
	if r.code != ExitFailure || !strings.Contains(r.stderr, "update failed: ") {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	f.unchanged(t)
}

func TestUpdateAgentGuard(t *testing.T) {
	f := newUpdateFixture(t, "v0.3.1", gobbleRelease(t, "v0.4.0", nil))
	t.Setenv("AI_AGENT", "gobble")
	for _, args := range [][]string{nil, {"--yes"}, {"--dry-run"}, {"--yes", "--force"}} {
		r := f.run(t, args...)
		if r.code != ExitFailure || r.stderr != "Error: "+agentRefusal+"\n" {
			t.Errorf("%v: exit %d, stderr %q", args, r.code, r.stderr)
		}
	}
	if n := len(f.srv.Requests()); n != 0 {
		t.Fatalf("the server saw %d requests", n)
	}
	f.unchanged(t)
	if r := f.run(t, "--check"); r.code != ExitUpdateAvailable {
		t.Fatalf("--check: exit %d, stderr %q", r.code, r.stderr)
	}
}

func TestUpdateSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a symlink needs privileges on Windows")
	}
	if !runningPlatform(t) {
		t.Skipf("%s/%s is not a release target", runtime.GOOS, runtime.GOARCH)
	}
	bin, err := testBinary()
	if err != nil {
		t.Fatal(err)
	}
	f := newUpdateFixture(t, "v0.3.1", gobbleRelease(t, "v0.4.0", bin))
	f.probeVersion = "0.4.0"
	link := filepath.Join(filepath.Dir(f.target), "gobble-link")
	if err := os.Symlink(f.target, link); err != nil {
		t.Fatal(err)
	}
	updaterDeps.target.ExecutablePath = link
	r := f.run(t, "--yes")
	if r.code != ExitFailure || !strings.Contains(r.stderr, "symlink") || !strings.Contains(r.stderr, "hint: ") ||
		!strings.Contains(r.stderr, link+" is not one") {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	f.unchanged(t)
}

// The probe looks for what `gobble version --json` prints (0048-MADR Q5).
func TestUpdateProbeWant(t *testing.T) {
	if got, want := probeWant("v0.4.0"), `"version":"0.4.0"`; got != want {
		t.Fatalf("probeWant(v0.4.0) = %q, want %q", got, want)
	}
	releaseIdentity(t, "v0.4.0")
	r := runMain(t, nil, "version", "--json")
	if r.code != ExitOK || !strings.Contains(r.stdout, probeWant("v0.4.0")) {
		t.Fatalf("exit %d, stdout %q lacks %q", r.code, r.stdout, probeWant("v0.4.0"))
	}
}
