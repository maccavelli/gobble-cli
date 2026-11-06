package archtest

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/maccavelli/gobble-cli/jsonx"
)

const modulePath = "github.com/maccavelli/gobble-cli"

// acpextPath is the nested module of gobble's ACP extension contract. Its
// path is under modulePath, but it is a module of its own, which the root
// requires by tag (0042-MADR D1, A4).
const acpextPath = modulePath + "/acpext"

// Violation is one forbidden import edge, or, for rule 11, a forbidden
// field, which Field names.
type Violation struct {
	Rule  int
	From  string
	To    string
	Field string
}

func (v Violation) String() string {
	if v.Field != "" {
		return fmt.Sprintf("rule %d: %s has %s", v.Rule, v.From, v.Field)
	}
	return fmt.Sprintf("rule %d: %s imports %s", v.Rule, v.From, v.To)
}

type modPkg struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
	// Deps, for a go-tui-lib package, is its whole import closure, as go
	// list gives it (0032-MADR D1).
	Deps []string
	Err  string
	// Dir and GoFiles are a module package's directory and its non-test
	// Go files, for rule 11.
	Dir     string
	GoFiles []string
}

// tuiLib is go-tui-lib's module path, its /v2 major since v2.0.0, and
// tuiLibV1 the path before it, which no package may import (0035-MADR
// D13); xansi is the one Charm module the native CLI may import
// (0032-MADR amendment A1).
const (
	tuiLib   = "github.com/maccavelli/go-tui-lib/v2"
	tuiLibV1 = "github.com/maccavelli/go-tui-lib"
	xansi    = "github.com/charmbracelet/x/ansi"
)

type edge struct {
	from string
	to   string
	test bool
}

// loadGraph lists the module's packages and their dependencies, the tests'
// included, so that a go-tui-lib package only a test imports has its
// closure too (0035-PLAN P14, deviation 5). A test build lists the
// module's packages again as variants ("p [p.test]") and its test mains
// ("p.test"); those are skipped, since each package's own entry carries
// its test imports apart from its imports.
func loadGraph(ctx context.Context, dir string) ([]modPkg, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-e", "-deps", "-test", "-json", "./...")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list: %w\n%s", err, stderr.String())
	}
	dec := jsontext.NewDecoder(&stdout, jsonx.Options())
	var pkgs []modPkg
	for {
		var raw struct {
			ImportPath   string
			Deps         []string
			Imports      []string
			TestImports  []string
			XTestImports []string
			Dir          string
			GoFiles      []string
			Error        *struct {
				Err string
			}
		}
		err := jsonx.UnmarshalDecode(dec, &raw)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode go list: %w", err)
		}
		path, variant := strings.CutSuffix(raw.ImportPath, "]")
		if variant {
			path, _, _ = strings.Cut(path, " [")
		}
		if pathUnder(path, tuiLib) { // its closure, for rule 5
			pkgs = append(pkgs, modPkg{ImportPath: path, Deps: raw.Deps})
			continue
		}
		if _, ok := relOf(path); !ok || variant || strings.HasSuffix(path, ".test") {
			continue
		}
		item := modPkg{
			ImportPath:   raw.ImportPath,
			Imports:      raw.Imports,
			TestImports:  raw.TestImports,
			XTestImports: raw.XTestImports,
			Dir:          raw.Dir,
			GoFiles:      raw.GoFiles,
		}
		if raw.Error != nil {
			item.Err = raw.Error.Err
		}
		pkgs = append(pkgs, item)
	}
	return pkgs, nil
}

func relOf(importPath string) (string, bool) {
	if pathUnder(importPath, acpextPath) { // another module, not a root package (0042-MADR A4)
		return "", false
	}
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
	"personality", // 0022-MADR D16
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
	return pathUnder(imp, tuiLib+"/tuitest")
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
	case pathUnder(imp, "github.com/maccavelli/acp-go-sdk2"):
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
		if !pathUnder(e.to, "github.com/maccavelli/acp-go-sdk2") {
			continue
		}
		rel, ok := relOf(e.from)
		if !ok {
			continue
		}
		if under(rel, "acpserver") || under(rel, "acpclient") {
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

// checkRule5 keeps Charm in internal/tui, which may import anything of
// go-tui-lib. internal/cli may import x/ansi, and a go-tui-lib package
// whose whole closure holds no Charm but x/ansi, and never internal/tui;
// any other package, none of go-tui-lib but tuitest in a test (0004-MADR
// rule 5, as amended by 0032-MADR D1 and its amendment A1). No package,
// internal/tui included, imports go-tui-lib's v1 path (0035-MADR D13).
func checkRule5(pkgs []modPkg) []Violation {
	closure := map[string][]string{}
	for _, p := range pkgs {
		if pathUnder(p.ImportPath, tuiLib) {
			closure[p.ImportPath] = p.Deps
		}
	}
	var vs []Violation
	for _, e := range edges(pkgs) {
		rel, ok := relOf(e.from)
		if ok && pathUnder(e.to, tuiLibV1) && !pathUnder(e.to, tuiLib) {
			vs = append(vs, Violation{Rule: 5, From: e.from, To: e.to + " (go-tui-lib's v1 path; import " + tuiLib + ", 0035-MADR D13)"})
			continue
		}
		if !ok || under(rel, "internal/tui") {
			continue
		}
		to, _ := relOf(e.to)
		switch {
		case isCharm(e.to):
			if !under(rel, "internal/cli") || !pathUnder(e.to, xansi) {
				vs = add(vs, 5, e)
			}
		case under(rel, "internal/cli") && to != "" && under(to, "internal/tui"):
			vs = add(vs, 5, e)
		case !pathUnder(e.to, tuiLib), e.test && isTUITest(e.to):
		case !under(rel, "internal/cli"):
			vs = add(vs, 5, e)
		default:
			if why := charmIn(e.to, closure); why != "" {
				vs = append(vs, Violation{Rule: 5, From: e.from, To: e.to + " (" + why + ")"})
			}
		}
	}
	return vs
}

// charmIn says why go-tui-lib package imp may not reach internal/cli: the
// first Charm package in its closure, or that its closure is unknown; ""
// when it holds no Charm.
func charmIn(imp string, closure map[string][]string) string {
	deps, ok := closure[imp]
	if !ok {
		return "its closure is unknown"
	}
	for _, d := range deps {
		if isCharm(d) && !pathUnder(d, xansi) {
			return "through " + d
		}
	}
	return ""
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

// checkRule10: jsonx imports no package of this module, so that every
// package may import it (0039-MADR D1, D24). Its own tests may.
func checkRule10(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, pkg := range pkgs {
		if pkg.ImportPath != modulePath+"/jsonx" {
			continue
		}
		for _, imp := range pkg.Imports {
			if _, ok := relOf(imp); ok {
				vs = add(vs, 10, edge{from: pkg.ImportPath, to: imp})
			}
		}
	}
	return vs
}

// byteFieldsAllowed are the struct fields of type []byte with a json tag,
// as package.Type.Field: binary data, which encodes as base64; raw JSON is
// a jsontext.Value (0039-MADR D20, D24).
var byteFieldsAllowed = map[string]bool{
	"llm.Content.Data": true,
}

// checkRule11: a struct field of type []byte with a json tag is one of
// byteFieldsAllowed. Each package's non-test Go files are parsed.
func checkRule11(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, pkg := range pkgs {
		rel, ok := relOf(pkg.ImportPath)
		if !ok || pkg.Dir == "" {
			continue
		}
		for _, name := range pkg.GoFiles {
			fields, err := jsonByteFields(filepath.Join(pkg.Dir, name))
			if err != nil {
				vs = append(vs, Violation{Rule: 11, From: pkg.ImportPath, Field: "a file not parsed: " + err.Error()})
				continue
			}
			for _, f := range fields {
				if !byteFieldsAllowed[rel+"."+f] {
					vs = append(vs, Violation{Rule: 11, From: pkg.ImportPath, Field: "a []byte field with a json tag, " + f})
				}
			}
		}
	}
	return vs
}

// jsonByteFields are path's struct fields of type []byte with a json tag,
// each as Type.Field, the type being the declared type or function that
// holds the struct.
func jsonByteFields(path string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []string
	scan := func(owner string, n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				if !isByteSlice(field.Type) || field.Tag == nil {
					continue
				}
				tag, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					continue
				}
				if v, ok := reflect.StructTag(tag).Lookup("json"); !ok || v == "-" {
					continue
				}
				for _, name := range field.Names {
					out = append(out, owner+"."+name.Name)
				}
			}
			return true
		})
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					scan(ts.Name.Name, ts.Type)
				} else {
					scan("var", spec)
				}
			}
		case *ast.FuncDecl:
			scan(d.Name.Name, d)
		}
	}
	return out, nil
}

// isByteSlice reports whether t is []byte or []uint8.
func isByteSlice(t ast.Expr) bool {
	at, ok := t.(*ast.ArrayType)
	if !ok || at.Len != nil {
		return false
	}
	id, ok := at.Elt.(*ast.Ident)
	return ok && (id.Name == "byte" || id.Name == "uint8")
}

// checkRule12: only acpserver, acpclient and internal/cli import acpext,
// the ACP edge's contract; internal/cli to keep its provider list equal
// to the registry's (0042-MADR D4).
func checkRule12(pkgs []modPkg) []Violation {
	var vs []Violation
	for _, e := range edges(pkgs) {
		if !pathUnder(e.to, acpextPath) {
			continue
		}
		rel, ok := relOf(e.from)
		if !ok || under(rel, "acpserver") || under(rel, "acpclient") || under(rel, "internal/cli") {
			continue
		}
		vs = add(vs, 12, e)
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
	case 10:
		return checkRule10(pkgs)
	case 11:
		return checkRule11(pkgs)
	case 12:
		return checkRule12(pkgs)
	default:
		return []Violation{{Rule: rule, From: "archtest", To: "unknown rule"}}
	}
}
