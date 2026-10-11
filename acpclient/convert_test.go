package acpclient

import (
	"errors"
	"math"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// A scripted exit code outside ACP's uint32 is an internal error from the
// test client, never a wrapped code (0014-PLAN C4).
func TestScriptedExitCode(t *testing.T) {
	for code, valid := range map[int]bool{0: true, 3: true, math.MaxUint32: true, -1: false, math.MaxUint32 + 1: false} {
		c := &acptest.Client{Terminals: &acptest.Terminals{ExitCode: code}}
		w, werr := c.WaitForTerminalExit(t.Context(), acp.WaitForTerminalExitRequest{})
		o, oerr := c.TerminalOutput(t.Context(), acp.TerminalOutputRequest{})
		if !valid {
			for _, err := range []error{werr, oerr} {
				if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32603 {
					t.Errorf("exit code %d: %v, want an internal error", code, err)
				}
			}
			continue
		}
		if werr != nil || oerr != nil || w.ExitCode == nil || int(*w.ExitCode) != code ||
			o.ExitStatus == nil || o.ExitStatus.ExitCode == nil || int(*o.ExitStatus.ExitCode) != code {
			t.Errorf("exit code %d: %+v, %v; %+v, %v", code, w, werr, o, oerr)
		}
	}
}

// An ACP count past an int's range is its maximum, never wrapped
// (0014-PLAN C4).
func TestToInt(t *testing.T) {
	for in, want := range map[uint64]int{0: 0, 7: 7, math.MaxInt: math.MaxInt, math.MaxInt + 1: math.MaxInt, math.MaxUint64: math.MaxInt} {
		if got := toInt(in); got != want {
			t.Errorf("toInt(%d) = %d, want %d", in, got, want)
		}
	}
	for in, want := range map[uint64]int64{0: 0, math.MaxInt64: math.MaxInt64, math.MaxInt64 + 1: math.MaxInt64, math.MaxUint64: math.MaxInt64} {
		if got := toInt64(in); got != want {
			t.Errorf("toInt64(%d) = %d, want %d", in, got, want)
		}
	}
	if got := toIntPtr(nil); got != nil {
		t.Errorf("toIntPtr(nil) = %d", *got)
	}
	if got := toIntPtr(new(uint64(math.MaxUint64))); got == nil || *got != math.MaxInt {
		t.Errorf("toIntPtr(max) = %v", got)
	}
}
