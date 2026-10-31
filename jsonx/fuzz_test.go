package jsonx_test

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/session"
)

// Quote equals Marshal byte for byte, and is valid JSON, for any bytes
// (0039-MADR Confirmation 5).
func FuzzQuote(f *testing.F) {
	for _, s := range []string{"", "plain", "a<b&c>d", "tab\tnl\nnul\x00del\x7f", "x\xff\xfe\xfdy", "caf\xe9", "  ", "\xed\xa0\x80"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		q := jsonx.Quote(s)
		m, err := jsonx.Marshal(s)
		if err != nil {
			t.Fatalf("Marshal(%q): %v", s, err)
		}
		if string(q) != string(m) {
			t.Fatalf("Quote(%q) = %s, Marshal = %s", s, q, m)
		}
		if !q.IsValid() {
			t.Fatalf("Quote(%q) = %s, not valid JSON", s, q)
		}
	})
}

// reasoning is the four-string shape Fields serves (0039-PLAN deviation 1).
type reasoning struct {
	Text      string `json:"text,omitzero"`
	Signature string `json:"signature,omitzero"`
	Encrypted string `json:"encrypted,omitzero"`
	Format    string `json:"format,omitzero"`
}

// Fields equals Marshal of a struct of omitzero string fields in the same
// order, for any values.
func FuzzFields(f *testing.F) {
	f.Add("", "", "", "")
	f.Add("hm", "sig", "", "messages")
	f.Add("caf\xe9", "", "x\xff", "<&>")
	f.Fuzz(func(t *testing.T, text, sig, enc, format string) {
		got := jsonx.Fields(jsonx.Field{Name: "text", Value: text}, jsonx.Field{Name: "signature", Value: sig},
			jsonx.Field{Name: "encrypted", Value: enc}, jsonx.Field{Name: "format", Value: format})
		want, err := jsonx.Marshal(reasoning{Text: text, Signature: sig, Encrypted: enc, Format: format})
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("Fields = %s, Marshal = %s", got, want)
		}
		if !got.IsValid() {
			t.Fatalf("Fields = %s, not valid JSON", got)
		}
	})
}

// gobble writes no repeated name: a session entry whose embedded unknown
// members repeat a known one either fails to encode, or encodes to JSON that
// a strict reader accepts (0039-MADR D3, Q6; Confirmation 22).
func FuzzNoDuplicateWritten(f *testing.F) {
	f.Add("message", "id1", `{"type":"other"}`)
	f.Add("custom", "id2", `{"x":1,"x":2}`)
	f.Add("message", "", `{"extra":true}`)
	f.Fuzz(func(t *testing.T, typ, id, unknown string) {
		e := session.Entry{Type: typ, ID: id, Unknown: jsontext.Value(unknown)}
		out, err := jsonx.Marshal(e)
		if err != nil {
			return
		}
		if !jsontext.Value(out).IsValid() {
			t.Fatalf("encoded %q, which a strict reader refuses", out)
		}
	})
}
