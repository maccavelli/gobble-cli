package archtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const modulePath = "github.com/maccavelli/gobble-cli"

// Violation is one forbidden import edge.
type Violation struct {
	Rule int
	From string
	To   string
}

func (v Violation) String() string {
	return fmt.Sprintf("rule %d: %s imports %s", v.Rule, v.From, v.To)
}

type modPkg struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
	Err          string
}

type edge struct {
	from string
	to   string
	test bool
}

func loadGraph(ctx context.Context, dir string) ([]modPkg, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-e", "-deps", "-json", "./...")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list: %w\n%s", err, stderr.String())
	}
	dec := json.NewDecoder(&stdout)
	var pkgs []modPkg
	for {
		var raw struct {
			ImportPath   string
			Imports      []string
			TestImports  []string
			XTestImports []string
			Error        *struct {
				Err string
			}
		}
		err := dec.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode go list: %w", err)
		}
		if _, ok := relOf(raw.ImportPath); !ok {
			continue
		}
		item := modPkg{
			ImportPath:   raw.ImportPath,
			Imports:      raw.Imports,
			TestImports:  raw.TestImports,
			XTestImports: raw.XTestImports,
		}
		if raw.Error != nil {
			item.Err = raw.Error.Err
		}
		pkgs = append(pkgs, item)
	}
	return pkgs, nil
}

func relOf(importPath string) (string, bool) {
	if importPath == modulePath {
		return ".", true
	}
	prefix := modulePath + "/"
	if after, ok := strings.CutPrefix(importPath, prefix); ok {
		return after, true
	}
	return "", false
}

func under(rel, root string) bool {
	return rel == root || strings.HasPrefix(rel, root+"/")
}

func pathUnder(imp, prefix string) bool {
	return imp == prefix || strings.HasPrefix(imp, prefix+"/")
}

func edges(pkgs []modPkg) []edge {
	var out []edge
	for _, pkg := range pkgs {
		for _, imp := range pkg.Imports {
			out = append(out, edge{from: pkg.ImportPath, to: imp})
		}
		for _, imp := range pkg.TestImports {
			out = append(out, edge{from: pkg.ImportPath, to: imp, test: true})
		}
		for _, imp := range pkg.XTestImports {
			out = append(out, edge{from: pkg.ImportPath, to: imp, test: true})
		}
	}
	return out
}

func add(vs []Violation, rule int, e edge) []Violation {
	return append(vs, Violation{Rule: rule, From: e.from, To: e.to})
}

var rule1Roots = []string{
	"agent", "llm", "tool", "session", "skill", "hook",
	"compaction", "prompt", "command", "permission", "checkpoint",
}

func inRule1(rel string) bool {
	for _, root := range rule1Roots {
		if under(rel, root) {
			return true
		}
	}
	return false
}

func isVendorSDK(imp string) bool {
	return pathUnder(imp, "github.com/anthropics/anthropic-sdk-go") ||
		pathUnder(imp, "github.com/openai/openai-go") ||
		pathUnder(imp, "google.golang.org/genai")
}

func isCoreLib(imp string) bool {
	return pathUnder(imp, "github.com/maccavelli/go-core-lib") ||
		pathUnder(imp, "github.com/maccavelli/go-selfupdate-lib")
}

// libBuildinfo is the one go-selfupdate-lib package internal/buildinfo may
// import.
const libBuildinfo = "github.com/maccavelli/go-selfupdate-lib/buildinfo"

func isSelfUpdateTest(imp string) bool {
	return imp == "github.com/maccavelli/go-selfupdate-lib/selfupdate/selfupdatetest" ||
		imp == "github.com/maccavelli/go-core-lib/selfupdate/selfupdatetest"
}

func isTUITest(imp string) bool {
	return pathUnder(imp, "github.com/maccavelli/go-tui-lib/tuitest")
}

func isCharm(imp string) bool {
	return strings.HasPrefix(imp, "charm.land/") || strings.HasPrefix(imp, "github.com/charmbracelet/")
}

func checkRule1(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		rel, ok := relOf(e.from)
		if !ok {
			continue
		}
		if inRule1(rel) && rule1Forbidden(rel, e.to) {
			vs = add(vs, 1, e)
			continue
		}
		if pathUnder(e.to, "go.opentelemetry.io/otel") && !under(rel, "telemetry") {
			vs = add(vs, 1, e)
		}
	}
	return vs
}

func rule1Forbidden(rel, imp string) bool {
	switch {
	case pathUnder(imp, "github.com/coder/acp-go-sdk"):
		return true
	case pathUnder(imp, "github.com/modelcontextprotocol/go-sdk"):
		return true
	case pathUnder(imp, "github.com/maccavelli/go-llmprovider-sdk"):
		return !under(rel, "llm/provider")
	case isCoreLib(imp):
		return true
	case isVendorSDK(imp):
		return true
	case isCharm(imp):
		return true
	case pathUnder(imp, "go.opentelemetry.io/otel"):
		return true
	default:
		return false
	}
}

func checkRule2(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		if !pathUnder(e.to, "github.com/coder/acp-go-sdk") {
			continue
		}
		rel, ok := relOf(e.from)
		if !ok {
			continue
		}
		// The module root blank-imports the SDK so go mod tidy keeps the
		// Phase 0 require (0004-PLAN, 2026-10-04). No other package may.
		if rel == "." || under(rel, "acpserver") || under(rel, "acpclient") {
			continue
		}
		vs = add(vs, 2, e)
	}
	return vs
}

func checkRule3(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		if !pathUnder(e.to, "github.com/modelcontextprotocol/go-sdk") {
			continue
		}
		rel, ok := relOf(e.from)
		if !ok || under(rel, "mcpclient") {
			continue
		}
		vs = add(vs, 3, e)
	}
	return vs
}

func checkRule4(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		rel, ok := relOf(e.from)
		if !ok {
			continue
		}
		if isVendorSDK(e.to) {
			vs = add(vs, 4, e)
			continue
		}
		if pathUnder(e.to, "github.com/maccavelli/go-llmprovider-sdk") && !under(rel, "llm/provider") {
			vs = add(vs, 4, e)
		}
	}
	return vs
}

func checkRule5(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		rel, ok := relOf(e.from)
		if !ok {
			continue
		}
		switch {
		case isCharm(e.to):
			if !under(rel, "internal/tui") {
				vs = add(vs, 5, e)
			}
		case pathUnder(e.to, "github.com/maccavelli/go-tui-lib"):
			if e.test && isTUITest(e.to) {
				continue
			}
			if !under(rel, "internal/tui") {
				vs = add(vs, 5, e)
			}
		}
	}
	return vs
}

func checkRule6(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		rel, ok := relOf(e.from)
		if !ok {
			continue
		}
		if (under(rel, "internal/cli") || under(rel, "internal/tui")) && pathUnder(e.to, modulePath+"/agent") {
			vs = add(vs, 6, e)
		}
	}
	return vs
}

func checkRule7(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		if !pathUnder(e.to, modulePath+"/exp") {
			continue
		}
		rel, ok := relOf(e.from)
		if !ok || under(rel, "cmd/gobble") {
			continue
		}
		vs = add(vs, 7, e)
	}
	return vs
}

func checkRule8(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		if !isCoreLib(e.to) {
			continue
		}
		if e.test && isSelfUpdateTest(e.to) {
			continue
		}
		rel, ok := relOf(e.from)
		if !ok || under(rel, "internal/cli") {
			continue
		}
		// internal/buildinfo wraps the library's stamps, and only those
		// (0004-MADR amendment of 2026-10-05).
		if rel == "internal/buildinfo" && e.to == libBuildinfo {
			continue
		}
		vs = add(vs, 8, e)
	}
	return vs
}

// checkRule9: only internal/cli/... may import Kong (0008-MADR D14).
func checkRule9(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		rel, ok := relOf(e.from)
		if !ok || !pathUnder(e.to, "github.com/alecthomas/kong") {
			continue
		}
		if !under(rel, "internal/cli") {
			vs = add(vs, 9, e)
		}
	}
	return vs
}

func checkRule(rule int, pkgs []modPkg) []Violation {
	switch rule {
	case 1:
		return checkRule1(pkgs)
	case 2:
		return checkRule2(pkgs)
	case 3:
		return checkRule3(pkgs)
	case 4:
		return checkRule4(pkgs)
	case 5:
		return checkRule5(pkgs)
	case 6:
		return checkRule6(pkgs)
	case 7:
		return checkRule7(pkgs)
	case 8:
		return checkRule8(pkgs)
	case 9:
		return checkRule9(pkgs)
	default:
		return []Violation{{Rule: rule, From: "archtest", To: "unknown rule"}}
	}
}
