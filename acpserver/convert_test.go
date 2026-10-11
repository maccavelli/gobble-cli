package acpserver

import (
	stdjson "encoding/json"
	"math"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// A tool call with no input sends no rawInput key, and one with input
// carries it verbatim, as the SDK's encoding/json writes the update
// (0014-PLAN P1 deviation 2).
func TestRawInputOnTheWire(t *testing.T) {
	for in, want := range map[string]string{"": "", `{"a":1}`: `"rawInput":{"a":1}`} {
		b, err := stdjson.Marshal(acp.StartToolCall("c1", "t", acp.WithStartRawInput(jsonRaw([]byte(in)))))
		if err != nil {
			t.Fatal(err)
		}
		got := string(b)
		if want == "" && strings.Contains(got, "rawInput") || want != "" && !strings.Contains(got, want) {
			t.Errorf("input %q: %s", in, got)
		}
	}
}

// The boundary conversions to ACP's unsigned fields clamp or leave out a
// value out of range, and never wrap it (0014-PLAN C4).
func TestConversions(t *testing.T) {
	for in, want := range map[int64]uint64{0: 0, 7: 7, -1: 0, math.MinInt64: 0, math.MaxInt64: math.MaxInt64} {
		if got := count(in); got != want {
			t.Errorf("count(%d) = %d, want %d", in, got, want)
		}
	}
	for in, want := range map[int]uint32{1: 1, 42: 42, math.MaxUint32: math.MaxUint32} {
		if got := line(in); got == nil || *got != want {
			t.Errorf("line(%d) = %v, want %d", in, got, want)
		}
	}
	for _, in := range []int{0, -1, math.MinInt, math.MaxUint32 + 1, math.MaxInt} {
		if got := line(in); got != nil {
			t.Errorf("line(%d) = %d, want none", in, *got)
		}
	}
	if got := exitCode(nil); got != nil {
		t.Errorf("exitCode(nil) = %d", *got)
	}
	for _, in := range []uint32{0, 1, math.MaxUint32} {
		if got := exitCode(new(in)); got == nil || *got != int(in) {
			t.Errorf("exitCode(%d) = %v", in, got)
		}
	}
}
