package acpclient_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/jsonx"
)

// A refused prompt whose data is gobble's ErrorData is a *PromptError: its
// code and data, its text the data's message, and the request error under
// it; auth_required is still ErrAuthRequired. An error with no data, or
// with data that is not gobble's, is as before (0049-MADR Q3 (a)).
func TestPromptError(t *testing.T) {
	quota, err := os.ReadFile(filepath.Join("..", "acpext", "testdata", "prompt-quota.error.json"))
	if err != nil {
		t.Fatal(err)
	}
	var want acpext.ErrorData
	if err := jsonx.Unmarshal(quota, &want); err != nil {
		t.Fatal(err)
	}
	const auth = `{"error":"llm: authentication failed: key refused","kind":"auth","provider":"claude"}`
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{
		{ErrCode: -32603, ErrMessage: "Internal error", ErrData: string(quota)},
		{ErrCode: -32000, ErrMessage: "Authentication required", ErrData: auth},
		{ErrCode: -32603, ErrMessage: "boom"},
		{ErrCode: -32603, ErrMessage: "Internal error", ErrData: `{"details":"another agent's"}`},
	}}
	c := start(t, agent, acpclient.Options{})
	s, err := c.NewSession(t.Context(), acptest.Abs("/w"), func(acpclient.Update) {})
	if err != nil {
		t.Fatal(err)
	}
	prompt := func() error {
		_, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "x"})
		return err
	}

	encode := func(d acpext.ErrorData) string {
		b, err := jsonx.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	err = prompt()
	pe, ok := errors.AsType[*acpclient.PromptError](err)
	if !ok || pe.Code != -32603 || encode(pe.Data) != encode(want) || pe.Error() != want.Error ||
		err.Error() != "session/prompt: "+want.Error {
		t.Fatalf("quota: err %q, %+v; want a PromptError of the frame's data", err, pe)
	}
	if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32603 {
		t.Fatalf("quota: no request error under %v", err)
	}

	err = prompt()
	if pe, ok := errors.AsType[*acpclient.PromptError](err); !errors.Is(err, acpclient.ErrAuthRequired) || !ok ||
		pe.Code != -32000 || pe.Data.Kind != acpext.KindAuth || pe.Data.Provider != "claude" {
		t.Fatalf("auth: err %v, %+v; want ErrAuthRequired and a PromptError", err, pe)
	}

	for _, what := range []string{"no data", "another agent's data"} {
		err = prompt()
		if _, ok := errors.AsType[*acpclient.PromptError](err); ok || !strings.HasPrefix(err.Error(), `session/prompt: {"code":-32603`) {
			t.Fatalf("%s: err %q, want the request error as before", what, err)
		}
	}
}
