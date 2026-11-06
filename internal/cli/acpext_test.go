package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/jsonx"
)

// gobble serves what acpext publishes (0042-MADR D4): the extensions
// initialize advertises, the table's keys, are acpext.Requests().
func TestAcpextRequestsAreServed(t *testing.T) {
	caps, err := acpserver.Capabilities()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := jsonx.Marshal(caps.Meta[acpext.MetaKey])
	if err != nil {
		t.Fatal(err)
	}
	var meta acpext.CapabilitiesMeta
	if err := jsonx.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("_meta.%s %s: %v", acpext.MetaKey, raw, err)
	}
	if !slices.Equal(meta.Extensions, acpext.Requests()) {
		t.Errorf("acpserver serves %q,\nacpext publishes %q", meta.Extensions, acpext.Requests())
	}
}

// Every notification acpserver sends is one acpext publishes, and each one
// acpext publishes is sent. acpserver sends them through its updates'
// notify alone; each call names its method by an acpext constant, read
// here from acpserver's source.
func TestAcpextNotificationsAreSent(t *testing.T) {
	published := map[string]string{
		"MethodRetry": acpext.MethodRetry, "MethodCompaction": acpext.MethodCompaction, "MethodPersonality": acpext.MethodPersonality,
	}
	files, err := filepath.Glob(filepath.Join("..", "..", "acpserver", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	sent := map[string]bool{}
	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok && fn.Name.Name == "notify" {
				return false // the funnel itself passes its method on
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch sel.Sel.Name {
			case "NotifyExtension":
				t.Errorf("%s: NotifyExtension outside updates.notify", fset.Position(call.Pos()))
			case "notify":
				if len(call.Args) != 2 {
					return true
				}
				m, ok := call.Args[0].(*ast.SelectorExpr)
				var pkg *ast.Ident
				if ok {
					pkg, ok = m.X.(*ast.Ident)
				}
				if !ok || pkg.Name != "acpext" || published[m.Sel.Name] == "" {
					t.Errorf("%s: a notification not named by an acpext notification constant", fset.Position(call.Pos()))
					return true
				}
				sent[published[m.Sel.Name]] = true
			}
			return true
		})
	}
	for _, m := range acpext.Notifications() {
		if !sent[m] {
			t.Errorf("acpext publishes %s, which acpserver never sends", m)
		}
	}
	if len(published) != len(acpext.Notifications()) {
		t.Errorf("this test names %d notifications, acpext publishes %d", len(published), len(acpext.Notifications()))
	}
}

// acpext's static provider list is the one gobble builds from the
// registry, so a client may show it before it connects (0042-MADR D3).
func TestAcpextAuthProvidersAreTheRegistry(t *testing.T) {
	if got, want := acpext.AuthProviders(), authProviders(); !slices.Equal(got, want) {
		t.Errorf("acpext.AuthProviders() = %+v,\nthe registry gives %+v", got, want)
	}
}
