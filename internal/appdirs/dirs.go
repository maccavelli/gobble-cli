package appdirs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// product is the directory leaf every role ends in.
const product = "gobble"

// Dirs are the four role directories. Each is absolute and clean.
type Dirs struct {
	Config, Data, State, Cache string
}

// Diagnostic is a non-fatal resolution note, such as an ignored relative
// XDG value.
type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Base holds the platform directories the roles are built on: Home on Unix
// and macOS, the RoamingAppData and LocalAppData Known Folders on Windows.
type Base struct {
	Home, AppData, LocalAppData string
}

// roleEnv names the per-role override of each role.
var roleEnv = [...]struct {
	name string
	dir  func(*Dirs) *string
}{
	{"GOBBLE_CONFIG_DIR", func(d *Dirs) *string { return &d.Config }},
	{"GOBBLE_DATA_DIR", func(d *Dirs) *string { return &d.Data }},
	{"GOBBLE_STATE_DIR", func(d *Dirs) *string { return &d.State }},
	{"GOBBLE_CACHE_DIR", func(d *Dirs) *string { return &d.Cache }},
}

// Resolve applies the 0003-MADR table for goos. Each role is, highest first:
// its GOBBLE_<ROLE>_DIR; GOBBLE_HOME, which makes all four roles that one
// directory; the platform default. On Unix and macOS that is
// $XDG_<ROLE>_HOME/gobble; on Windows it is %APPDATA%\gobble for config and
// %LOCALAPPDATA%\gobble\{data,state,cache}, and XDG is ignored.
//
// base reads the platform base directories. Resolve calls it only when a
// role falls to the platform default: with no GOBBLE_HOME, and a role with no
// variable of its own. So GOBBLE_HOME, or all four role variables, resolve
// where the home directory or a Known Folder is unusable (0049-MADR D11);
// otherwise base's error is Resolve's.
//
// A relative GOBBLE_* value is an error. A relative XDG value is ignored
// with a diagnostic, as the XDG Base Directory specification requires.
func Resolve(env func(string) string, goos string, base func() (Base, error)) (Dirs, []Diagnostic, error) {
	var (
		d     Dirs
		diags []Diagnostic
	)
	switch home := env("GOBBLE_HOME"); {
	case home != "":
		h, err := absSetting("GOBBLE_HOME", home)
		if err != nil {
			return Dirs{}, nil, err
		}
		d = Dirs{Config: h, Data: h, State: h, Cache: h}
	case needsPlatform(env):
		b, err := base()
		if err != nil {
			return Dirs{}, nil, err
		}
		if d, diags, err = platform(env, goos, b); err != nil {
			return Dirs{}, nil, err
		}
	}
	for _, r := range roleEnv {
		v := env(r.name)
		if v == "" {
			continue
		}
		p, err := absSetting(r.name, v)
		if err != nil {
			return Dirs{}, nil, err
		}
		*r.dir(&d) = p
	}
	return d, diags, nil
}

// needsPlatform reports whether a role, with GOBBLE_HOME unset, has no
// variable of its own and so falls to the platform default.
func needsPlatform(env func(string) string) bool {
	for _, r := range roleEnv {
		if env(r.name) == "" {
			return true
		}
	}
	return false
}

// readBase reads the platform base directories; a test replaces it.
var readBase = systemBase

// System resolves the roles for this process: its environment and OS, and
// the platform base directories when a role needs them (Resolve).
func System() (Dirs, []Diagnostic, error) {
	return Resolve(os.Getenv, runtime.GOOS, readBase)
}

func platform(env func(string) string, goos string, base Base) (Dirs, []Diagnostic, error) {
	if goos == "windows" {
		roaming, err := absBase("RoamingAppData", base.AppData)
		if err != nil {
			return Dirs{}, nil, err
		}
		local, err := absBase("LocalAppData", base.LocalAppData)
		if err != nil {
			return Dirs{}, nil, err
		}
		leaf := filepath.Join(local, product)
		return Dirs{
			Config: filepath.Join(roaming, product),
			Data:   filepath.Join(leaf, "data"),
			State:  filepath.Join(leaf, "state"),
			Cache:  filepath.Join(leaf, "cache"),
		}, nil, nil
	}
	home, err := absBase("home directory", base.Home)
	if err != nil {
		return Dirs{}, nil, err
	}
	var diags []Diagnostic
	xdg := func(name string, def ...string) string {
		fallback := filepath.Join(append([]string{home}, def...)...)
		v := env(name)
		switch {
		case v == "":
			return filepath.Join(fallback, product)
		case !filepath.IsAbs(v):
			diags = append(diags, Diagnostic{
				Code:    "xdg_relative_ignored",
				Message: fmt.Sprintf("%s=%q is relative; using %s", name, v, fallback),
			})
			return filepath.Join(fallback, product)
		}
		return filepath.Join(filepath.Clean(v), product)
	}
	return Dirs{
		Config: xdg("XDG_CONFIG_HOME", ".config"),
		Data:   xdg("XDG_DATA_HOME", ".local", "share"),
		State:  xdg("XDG_STATE_HOME", ".local", "state"),
		Cache:  xdg("XDG_CACHE_HOME", ".cache"),
	}, diags, nil
}

func absSetting(name, v string) (string, error) {
	if !filepath.IsAbs(v) {
		return "", fmt.Errorf("appdirs: %s=%q must be an absolute path", name, v)
	}
	return filepath.Clean(v), nil
}

func absBase(what, v string) (string, error) {
	if v == "" || !filepath.IsAbs(v) {
		return "", fmt.Errorf("appdirs: %s %q is not an absolute path", what, v)
	}
	return filepath.Clean(v), nil
}
