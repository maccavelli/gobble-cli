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

// A response whose call carried no limit, {0, "none"}, gives that limit,
// not none (0071-MADR P35).
func TestPromptResponseOutputLimit_0071(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Meta: `{"gobble":{"outputLimit":{"tokens":0,"basis":"none"}}}`}}}
		c := start(t, agent, acpclient.Options{Name: "test", Version: "0.0.0"})
		s, err := c.NewSession(t.Context(), acptest.Abs("/w"), func(acpclient.Update) {})
		if err != nil {
			t.Fatal(err)
		}
		r, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "go"})
		if err != nil {
			t.Fatal(err)
		}
		if want := (acpclient.OutputLimit{Tokens: 0, Basis: "none"}); r.OutputLimit == nil || *r.OutputLimit != want {
			t.Errorf("OutputLimit %+v, want %+v", r.OutputLimit, want)
		}
	})
	// A failed call's limit is what it asked for, marked (0071-MADR P32).
	t.Run("asked", func(t *testing.T) {
		agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Meta: `{"gobble":{"outputLimit":{"tokens":32000,"basis":"stated","asked":true}}}`}}}
		c := start(t, agent, acpclient.Options{Name: "test", Version: "0.0.0"})
		s, err := c.NewSession(t.Context(), acptest.Abs("/w"), func(acpclient.Update) {})
		if err != nil {
			t.Fatal(err)
		}
		r, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "go"})
		if err != nil {
			t.Fatal(err)
		}
		// 0071 2c new names: begin
		if want := (acpclient.OutputLimit{Tokens: 32000, Basis: "stated", Asked: true}); r.OutputLimit == nil || *r.OutputLimit != want {
			t.Errorf("OutputLimit %+v, want %+v", r.OutputLimit, want)
		}
		// 0071 2c new names: end
	})
}
