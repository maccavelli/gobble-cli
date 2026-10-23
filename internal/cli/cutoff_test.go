package cli

import (
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// A turn that fails while its reply is being drawn keeps what was drawn and
// says the reply was cut off, before the error; a failure with no reply
// drawn, or after a tool ran, says nothing of the kind (0031-MADR D3).
func TestCutOffReply(t *testing.T) {
	tool := `{"sessionUpdate":"tool_call","toolCallId":"t1","title":"read","status":"completed"}`
	for _, c := range []struct {
		name    string
		updates []string
		cut     bool
	}{
		{"mid-reply", []string{chunk("half a rep")}, true},
		{"before any reply", nil, false},
		{"after a tool ran", []string{chunk("let me look"), tool}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			for _, p := range provider.KeyProviders() {
				t.Setenv(provider.CredentialVar(p), "")
			}
			agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: c.updates, ErrCode: -32603,
				ErrMessage: "llm: unavailable: the stream broke"}}}
			h := newLineHarness(t, agent)
			log := &modeLog{h: h}
			h.e.out.out, h.e.out.err = log, log
			h.start(Prompt{})
			h.press("hi\r")
			waitUntil(t, func() bool { return strings.Contains(log.String(), "the stream broke") })
			h.press("\x04")
			if err := h.wait(); err != nil {
				t.Fatal(err)
			}
			out := log.String()
			at := strings.Index(out, "(the reply was cut off)")
			if c.cut != (at >= 0) {
				t.Fatalf("the cut-off line shown %v, want %v:\n%q", at >= 0, c.cut, out)
			}
			if c.cut && (strings.Index(out, "half a rep") > at || at > strings.Index(out, "the stream broke")) {
				t.Fatalf("the cut-off line is not between the reply and the error:\n%q", out)
			}
		})
	}
}
