package appdirs

import (
	"maps"
	"path/filepath"
	"strings"
	"testing"
)

// The 0003-MADR table, for each GOOS, on fake base directories (0004-PLAN
// Phase 2 Accept: "passes for three GOOS values").
func TestResolveTable(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	roaming := filepath.Join(tmp, "Roaming")
	local := filepath.Join(tmp, "Local")
	base := Base{Home: home, AppData: roaming, LocalAppData: local}
	xdg := filepath.Join(tmp, "xdg")
	gh := filepath.Join(tmp, "portable")
	only := filepath.Join(tmp, "state-only")

	unixDefault := Dirs{
		Config: filepath.Join(home, ".config", "gobble"),
		Data:   filepath.Join(home, ".local", "share", "gobble"),
		State:  filepath.Join(home, ".local", "state", "gobble"),
		Cache:  filepath.Join(home, ".cache", "gobble"),
	}
	windowsDefault := Dirs{
		Config: filepath.Join(roaming, "gobble"),
		Data:   filepath.Join(local, "gobble", "data"),
		State:  filepath.Join(local, "gobble", "state"),
		Cache:  filepath.Join(local, "gobble", "cache"),
	}
	xdgEnv := map[string]string{
		"XDG_CONFIG_HOME": filepath.Join(xdg, "c"), "XDG_DATA_HOME": filepath.Join(xdg, "d"),
		"XDG_STATE_HOME": filepath.Join(xdg, "s"), "XDG_CACHE_HOME": filepath.Join(xdg, "k"),
	}
	xdgDirs := Dirs{
		Config: filepath.Join(xdg, "c", "gobble"), Data: filepath.Join(xdg, "d", "gobble"),
		State: filepath.Join(xdg, "s", "gobble"), Cache: filepath.Join(xdg, "k", "gobble"),
	}
	portable := Dirs{Config: gh, Data: gh, State: gh, Cache: gh}

	cases := []struct {
		name string
		goos string
		env  map[string]string
		want Dirs
	}{
		{"linux defaults", "linux", nil, unixDefault},
		{"darwin defaults are XDG too", "darwin", nil, unixDefault},
		{"windows defaults", "windows", nil, windowsDefault},
		{"linux XDG", "linux", xdgEnv, xdgDirs},
		{"darwin XDG", "darwin", xdgEnv, xdgDirs},
		{"windows ignores XDG", "windows", xdgEnv, windowsDefault},
		{"GOBBLE_HOME on linux", "linux", map[string]string{"GOBBLE_HOME": gh}, portable},
		{"GOBBLE_HOME on windows", "windows", map[string]string{"GOBBLE_HOME": gh}, portable},
		{"GOBBLE_HOME beats XDG", "darwin", merge(xdgEnv, map[string]string{"GOBBLE_HOME": gh}), portable},
		{"a role override beats GOBBLE_HOME", "linux",
			map[string]string{"GOBBLE_HOME": gh, "GOBBLE_STATE_DIR": only},
			Dirs{Config: gh, Data: gh, State: only, Cache: gh}},
		{"each role override on the platform default", "windows",
			map[string]string{
				"GOBBLE_CONFIG_DIR": filepath.Join(tmp, "c"), "GOBBLE_DATA_DIR": filepath.Join(tmp, "d"),
				"GOBBLE_STATE_DIR": filepath.Join(tmp, "s"), "GOBBLE_CACHE_DIR": filepath.Join(tmp, "k"),
			},
			Dirs{Config: filepath.Join(tmp, "c"), Data: filepath.Join(tmp, "d"),
				State: filepath.Join(tmp, "s"), Cache: filepath.Join(tmp, "k")}},
		{"overrides are cleaned", "linux", map[string]string{"GOBBLE_HOME": gh + string(filepath.Separator) + "." + string(filepath.Separator)}, portable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags, err := Resolve(envOf(tc.env), tc.goos, base)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("Resolve =\n  %+v\nwant\n  %+v", got, tc.want)
			}
			if len(diags) != 0 {
				t.Fatalf("unexpected diagnostics %+v", diags)
			}
		})
	}
}

func TestResolveRelativeXDGIsIgnored(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	got, diags, err := Resolve(envOf(map[string]string{"XDG_STATE_HOME": "relative/state"}), "linux", Base{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "state", "gobble"); got.State != want {
		t.Fatalf("State = %q, want the default %q", got.State, want)
	}
	if len(diags) != 1 || diags[0].Code != "xdg_relative_ignored" || !strings.Contains(diags[0].Message, "XDG_STATE_HOME") {
		t.Fatalf("diagnostics = %+v", diags)
	}
}

func TestResolveErrors(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "home")
	cases := []struct {
		name string
		goos string
		env  map[string]string
		base Base
		want string
	}{
		{"relative GOBBLE_HOME", "linux", map[string]string{"GOBBLE_HOME": "portable"}, Base{Home: abs}, "GOBBLE_HOME"},
		{"relative role override", "windows", map[string]string{"GOBBLE_DATA_DIR": `data`}, Base{AppData: abs, LocalAppData: abs}, "GOBBLE_DATA_DIR"},
		{"no home on unix", "linux", nil, Base{}, "home directory"},
		{"no AppData on windows", "windows", nil, Base{Home: abs, LocalAppData: abs}, "RoamingAppData"},
		{"no LocalAppData on windows", "windows", nil, Base{Home: abs, AppData: abs}, "LocalAppData"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Resolve(envOf(tc.env), tc.goos, tc.base)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want one naming %s", err, tc.want)
			}
		})
	}
}

// GOBBLE_HOME needs no platform base, so it works where HOME is unusable.
func TestResolveGobbleHomeNeedsNoBase(t *testing.T) {
	gh := filepath.Join(t.TempDir(), "p")
	got, _, err := Resolve(envOf(map[string]string{"GOBBLE_HOME": gh}), "linux", Base{})
	if err != nil || got.State != gh {
		t.Fatalf("Resolve = %+v, %v", got, err)
	}
}

func TestSystem(t *testing.T) {
	gh := t.TempDir()
	t.Setenv("GOBBLE_HOME", gh)
	got, _, err := System()
	if err != nil {
		t.Fatal(err)
	}
	if got != (Dirs{Config: gh, Data: gh, State: gh, Cache: gh}) {
		t.Fatalf("System with GOBBLE_HOME = %+v", got)
	}

	t.Setenv("GOBBLE_HOME", "")
	got, _, err = System()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{got.Config, got.Data, got.State, got.Cache} {
		if !filepath.IsAbs(p) || !strings.Contains(p, "gobble") {
			t.Fatalf("System role %q is not an absolute gobble path (%+v)", p, got)
		}
	}
}

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func merge(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	maps.Copy(out, a)
	maps.Copy(out, b)
	return out
}
