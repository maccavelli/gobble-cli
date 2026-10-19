package tool_test

import (
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

// Redescribed changes the description alone, and leaves the tool it was
// given as it was; a tool New did not build is returned as it is
// (0022-MADR D15).
func TestRedescribed(t *testing.T) {
	var seen readIn
	orig := readTool(&seen)
	re := tool.Redescribed(orig, "Read a file, as the user says it.")
	a, b := orig.Spec(), re.Spec()
	if b.Description != "Read a file, as the user says it." || a.Description != "read a file" {
		t.Fatalf("descriptions %q, %q", a.Description, b.Description)
	}
	if a.Name != b.Name || string(a.InputSchema) != string(b.InputSchema) || a.Kind != b.Kind {
		t.Fatalf("more than the description changed: %+v, %+v", a, b)
	}
	if _, ok := re.(tool.Describer); !ok {
		t.Fatal("the redescribed tool lost its Describer")
	}
	var other tool.Tool = plain{}
	if got := tool.Redescribed(other, "x"); got != other {
		t.Fatal("a tool New did not build was changed")
	}
}

type plain struct{ tool.Tool }
