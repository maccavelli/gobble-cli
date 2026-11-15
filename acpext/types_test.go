package acpext_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"slices"
	"strconv"
	"testing"
)

// contractKinds are the eighteen values of go-llmprovider-sdk's
// authcontract.ErrorKind, copied from 0073-MADR-publish-an-auth-contract-module.md
// D3's table in that repository, in its order. acpext imports no SDK
// module, so the list is a copy: a word added there reaches acpext only by
// gobble's next record (0051-MADR D6).
var contractKinds = []string{
	"canceled", "timeout", "denied", "expired", "auth", "quota", "rate_limited", "not_permitted", "workspace",
	"context_overflow", "empty_answer", "incomplete", "invalid_request", "invalid_provider", "offline", "unavailable",
	"unsupported", "other",
}

var snakeCase = regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)

// constantsOf returns the values of the string constants of types.go that
// are declared with the named type, read from the source so that a constant
// left out of a list in this test is still found.
func constantsOf(t *testing.T, typeName string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "types.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var vals []string
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, sp := range gd.Specs {
			vs := sp.(*ast.ValueSpec)
			id, ok := vs.Type.(*ast.Ident)
			if !ok || id.Name != typeName {
				continue
			}
			for _, v := range vs.Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Fatalf("%s constant %v is not a string literal", typeName, vs.Names)
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatal(err)
				}
				vals = append(vals, s)
			}
		}
	}
	return vals
}

// acpext's ErrorKind and RetryCause values are the auth contract's words
// (0051-MADR D6, QB): the ErrorKind constants are exactly the contract's
// eighteen, and every RetryCause is snake_case and one of them.
func TestKindsAreTheContractsSpelling(t *testing.T) {
	kinds := constantsOf(t, "ErrorKind")
	for _, k := range kinds {
		if !snakeCase.MatchString(k) {
			t.Errorf("kind %q is not snake_case", k)
		}
	}
	got, want := slices.Sorted(slices.Values(kinds)), slices.Sorted(slices.Values(contractKinds))
	if !slices.Equal(got, want) {
		t.Errorf("the ErrorKind constants are %q, want the contract's %q", got, want)
	}
	if len(want) != 18 {
		t.Errorf("the contract's list holds %d kinds, want 18", len(want))
	}
	causes := constantsOf(t, "RetryCause")
	if len(causes) == 0 {
		t.Fatal("no RetryCause constants found in types.go")
	}
	for _, c := range causes {
		if !snakeCase.MatchString(c) {
			t.Errorf("retry cause %q is not snake_case", c)
		}
		if !slices.Contains(contractKinds, c) {
			t.Errorf("retry cause %q is not one of the contract's kinds", c)
		}
	}
}
