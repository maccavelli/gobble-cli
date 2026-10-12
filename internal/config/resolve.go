package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// Options say which settings to resolve.
type Options struct {
	// ConfigDir is gobble's config directory; "" reads no global file.
	ConfigDir string
	// Cwd is the session's or the process's working directory; "" reads no
	// project file (gobble acp's process keys, 0018-MADR D16).
	Cwd string
	// Read selects session or process keys; zero is both.
	Read Read
	// Trusted reports whether a project root is trusted. Nil trusts none,
	// which is every project until 0005-PLAN F6 supplies it (D5).
	Trusted func(root string) bool
	// Overrides are --set's, the session layer.
	Overrides []Override
}

// DisabledLayer is a project file read but not applied, with why, and the
// keys it would have set.
type DisabledLayer struct {
	File   string
	Reason string
	Keys   []string
}

// Resolved is the effective settings, where each value came from, and the
// layers that were read but not applied.
type Resolved struct {
	Settings Settings
	Origins  map[string]Source
	Disabled []DisabledLayer
}

// GlobalFile is the global settings file in configDir, or "" with none.
func GlobalFile(configDir string) string {
	if configDir == "" {
		return ""
	}
	return filepath.Join(configDir, "settings.yaml")
}

// ProjectFile is the project settings file for cwd: .gobble/settings.yaml
// at the git top-level of cwd, or at cwd outside a repository (D2). ok is
// false for an empty cwd.
func ProjectFile(cwd string) (path string, ok bool) {
	if cwd == "" {
		return "", false
	}
	root := cwd
	if repo, _, inRepo := fsx.GitRepo(cwd); inRepo {
		root = repo
	}
	return filepath.Join(root, ".gobble", "settings.yaml"), true
}

// Resolve layers the defaults, the global file, the project file when its
// root is trusted, and the overrides, applying only the keys o.Read
// selects (0018-MADR D2, D4, D5, D16). The diagnostics are every layer's,
// a disabled project's included.
func Resolve(o Options) (Resolved, []Diagnostic) {
	r := Resolved{Settings: Defaults(), Origins: map[string]Source{}}
	for _, k := range Keys {
		r.Origins[k.Path] = Source{Layer: LayerDefault}
	}
	var ds []Diagnostic
	apply := func(ls map[string]leaf) {
		for path, l := range ls {
			if k, _ := KeyOf(path); o.Read != 0 && k.Read != o.Read {
				continue
			}
			if l.null {
				continue // a null clears to the layer below: that layer's value stays
			}
			if err := r.Settings.set(path, l.value); err == nil {
				r.Origins[path] = l.src
			}
		}
	}
	if f := GlobalFile(o.ConfigDir); f != "" {
		ls, fds := readLayer(f, LayerGlobal)
		ds = append(ds, fds...)
		apply(ls)
	}
	if f, ok := ProjectFile(o.Cwd); ok {
		ls, fds := readLayer(f, LayerProject)
		ds = append(ds, fds...)
		root := filepath.Dir(filepath.Dir(f))
		switch {
		case ls == nil:
		case o.Trusted != nil && o.Trusted(root):
			apply(ls)
		default:
			d := DisabledLayer{File: f, Reason: textUntrusted}
			for _, k := range Keys {
				if _, set := ls[k.Path]; set {
					d.Keys = append(d.Keys, k.Path)
				}
			}
			r.Disabled = append(r.Disabled, d)
			ds = append(ds, warningAt(f, 0, "", textUntrusted))
		}
	}
	ov := map[string]leaf{}
	for _, x := range o.Overrides {
		ov[x.Path] = leaf{value: x.value, null: x.null, src: Source{Layer: LayerSession, File: "--set"}}
	}
	apply(ov)
	return r, ds
}

// readLayer reads and checks one file. A missing file is no layer and no
// diagnostic; nil leaves mean the file was absent or unusable.
func readLayer(file string, layer Layer) (map[string]leaf, []Diagnostic) {
	data, err := os.ReadFile(file) //nolint:gosec // G304: GlobalFile's or ProjectFile's settings.yaml
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, []Diagnostic{errorAt(file, 0, "", err.Error())}
	}
	root, ds := Parse(file, data)
	if root == nil {
		return nil, ds
	}
	ls, lds := leaves(file, root, layer)
	return ls, append(ds, lds...)
}
