package builtin

import (
	"encoding/json/v2"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/tool"
)

// resolve makes a path the model gave absolute, as Pi resolves it, against
// the session's working directory.
func resolve(env tool.Env, path string) string {
	return fsx.Resolve(env.Cwd, path)
}

// roots are the call's workspace roots: Env.Roots, or the working directory
// alone when the caller set no roots.
func roots(env tool.Env) []string {
	if len(env.Roots) > 0 {
		return env.Roots
	}
	if env.Cwd != "" {
		return []string{env.Cwd}
	}
	return nil
}

// workspace is the call's view of the file system.
func workspace(env tool.Env) fsx.Workspace {
	return fsx.Workspace{Roots: roots(env), AllowOutside: env.AllowOutside}
}

// outsidePath is the Confined answer of a tool whose one path is its
// "path" argument: that path, resolved, when it is outside the roots.
func outsidePath(c tool.Call, env tool.Env) []string {
	p := argPath(c)
	if p == "" {
		return nil
	}
	abs := resolve(env, p)
	if workspace(env).Outside(abs) {
		return []string{abs}
	}
	return nil
}

// shown is path as a title shows it: relative to cwd when it lies under it.
func shown(env tool.Env, path string) string {
	abs := resolve(env, path)
	if env.Cwd != "" {
		if rel, err := filepath.Rel(env.Cwd, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(abs)
}

// argPath is a call's "path" argument, or "" when it has none.
func argPath(c tool.Call) string {
	var a struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(c.Args, &a); err != nil {
		return ""
	}
	return a.Path
}

// titlePath is argPath for a title: "?" when the call names no path.
func titlePath(c tool.Call) string {
	if p := argPath(c); p != "" {
		return p
	}
	return "?"
}

// unwrapPath drops the absolute path an *fs.PathError repeats, since the
// message already names the file as the model gave it.
func unwrapPath(err error) error {
	if pe, ok := errors.AsType[*fs.PathError](err); ok {
		return pe.Err
	}
	return err
}
