package acpclient_test

import (
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// Result.OutputLimit reads the response's _meta.gobble.outputLimit; a
// response without one, or with a mistyped one, gives none (0071-MADR D10,
// L27).
func TestResultOutputLimit_0071(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{
		{Meta: `{"gobble":{"outputLimit":{"tokens":64000,"basis":"ceiling"}}}`},
		{},
		{Meta: `{"gobble":{"outputLimit":"64000"}}`},
		{Meta: `{"gobble":{"cleared":{"followUp":[],"steering":["s"]},"outputLimit":{"tokens":1024,"basis":"window"}}}`, StopReason: "cancelled"},
	}}
	c := start(t, agent, acpclient.Options{Name: "test", Version: "0.0.0"})
	s, err := c.NewSession(t.Context(), acptest.Abs("/w"), func(acpclient.Update) {})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []*acpclient.OutputLimit{{Tokens: 64000, Basis: "ceiling"}, nil, nil, {Tokens: 1024, Basis: "window"}} {
		r, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "go"})
		if err != nil {
			t.Fatal(err)
		}
		if (r.OutputLimit == nil) != (want == nil) || (want != nil && *r.OutputLimit != *want) {
			t.Errorf("turn %d: OutputLimit %+v, want %+v", i+1, r.OutputLimit, want)
		}
	}
}
