package jsonx_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/jsonx"
)

// backslash is made in code, so that no JSON escape in a test input is
// decoded before the file is written (0039-PLAN C4).
var backslash = string(rune(92))

func esc(s string) string { return strings.ReplaceAll(s, "~", backslash) }

// From keeps empty absent and stores anything else as it is (#9's U1).
func TestFrom(t *testing.T) {
	if v := jsonx.From(nil); v != nil || !jsonx.IsAbsent(v) {
		t.Fatalf("From(nil) = %q, want absent", v)
	}
	if v := jsonx.From([]byte{}); v != nil || !jsonx.IsAbsent(v) {
		t.Fatalf("From([]byte{}) = %q, want nil", v)
	}
	if v := jsonx.From([]byte("null")); jsonx.IsAbsent(v) || !jsonx.IsNull(v) {
		t.Fatalf("From(null) = %q, want present and null", v)
	}
	type holder struct {
		V jsontext.Value `json:"v,omitzero"`
	}
	b, err := jsonx.Marshal(holder{V: jsonx.From([]byte("{}"))})
	if err != nil || string(b) != `{"v":{}}` {
		t.Fatalf("{} = %s, %v; want {\"v\":{}}", b, err)
	}
	b, err = jsonx.Marshal(holder{V: jsonx.From([]byte{})})
	if err != nil || string(b) != `{}` {
		t.Fatalf("empty = %s, %v; want the field left out", b, err)
	}
	if v := jsonx.From([]byte("{")); string(v) != "{" {
		t.Fatalf("From({) = %q, want it stored", v)
	}
	if _, err := jsonx.Marshal(holder{V: jsonx.From([]byte("{"))}); err == nil {
		t.Fatal("a parent of bad JSON encoded")
	}
}

// Value is never empty non-nil, and repairs invalid UTF-8.
func TestValue(t *testing.T) {
	v, err := jsonx.Value(struct {
		S string `json:"s"`
	}{"caf\xe9"})
	if err != nil || string(v) != "{\"s\":\"caf�\"}" {
		t.Fatalf("Value = %q, %v", v, err)
	}
	if v, err := jsonx.Value(nil); err != nil || string(v) != "null" {
		t.Fatalf("Value(nil) = %q, %v; want null", v, err)
	}
}

// Splice passes JSON through, leaves empty out, and turns anything else
// into a JSON string of its text (0039-MADR D9).
func TestSplice(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"valid", `{"path":"a.txt"}`, `{"path":"a.txt"}`},
		{"cut off", `{"path":"a.tx`, esc(`"{~"path~":~"a.tx"`)},
		{"invalid byte", "{\"path\":\"a\xffb\"}", "{\"path\":\"a\xffb\"}"},
		{"lone surrogate", esc(`{"path":"a~ud83d"}`), esc(`{"path":"a~ud83d"}`)},
	} {
		if got := jsonx.Splice(jsontext.Value(tc.in)); string(got) != tc.want {
			t.Errorf("%s: Splice(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
	if got := jsonx.Splice(jsontext.Value{}); got != nil {
		t.Fatalf("Splice(empty) = %q, want nil", got)
	}
}

func TestIsNull(t *testing.T) {
	for in, want := range map[string]bool{"null": true, " null\n": true, "": false, "{}": false, `"null"`: false} {
		if got := jsonx.IsNull(jsontext.Value(in)); got != want {
			t.Errorf("IsNull(%q) = %v, want %v", in, got, want)
		}
	}
}

// The data policy repairs; the configuration policy refuses (0039-MADR D2,
// D3).
func TestPolicies(t *testing.T) {
	bad := struct {
		S string `json:"s"`
	}{"a\xffb"}
	if _, err := jsonx.Marshal(bad); err != nil {
		t.Fatalf("Marshal of invalid UTF-8: %v", err)
	}
	if _, err := jsonx.MarshalStrict(bad); err == nil {
		t.Fatal("MarshalStrict accepted invalid UTF-8")
	}
	var s struct {
		S string `json:"s"`
	}
	lone := []byte(esc(`{"s":"a~ud83d"}`))
	if err := jsonx.Unmarshal(lone, &s); err != nil || s.S != "a�" {
		t.Fatalf("Unmarshal of a lone surrogate = %q, %v", s.S, err)
	}
	if err := jsonx.UnmarshalStrict(lone, &s); err == nil {
		t.Fatal("UnmarshalStrict accepted a lone surrogate")
	}
	if err := jsonx.UnmarshalStrict([]byte(`{"s":"a","s":"b"}`), &s); err == nil {
		t.Fatal("UnmarshalStrict accepted a repeated name")
	}
	m := map[string]int{"b": 2, "a": 1, "c": 3}
	for range 20 {
		b, err := jsonx.Marshal(m)
		if err != nil || string(b) != `{"a":1,"b":2,"c":3}` {
			t.Fatalf("Marshal(map) = %s, %v; want its keys sorted", b, err)
		}
	}
	if _, err := json.Marshal(m); err != nil {
		t.Fatal(err)
	}
}
