package render

import (
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// The spinner asks for a new label every four seconds, not every frame
// (0022-MADR D12).
func TestSpinnerLabelRotates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var w syncBuffer
		n := 0
		sp := StartSpinner(t.Context(), &w, colorOn, false, func() string {
			n++
			return "label " + strconv.Itoa(n)
		})
		time.Sleep(8*time.Second + 50*time.Millisecond)
		if err := sp.Stop(); err != nil {
			t.Fatal(err)
		}
		if n != 3 {
			t.Fatalf("asked for %d labels in 8 s, want 3 (at 0, 4 and 8 s)", n)
		}
		out := w.String()
		for _, l := range []string{"label 1", "label 2", "label 3"} {
			if !strings.Contains(out, " "+l+"\x1b[K") {
				t.Errorf("%q never drawn", l)
			}
		}
		if i, j := strings.LastIndex(out, "label 1"), strings.Index(out, "label 2"); i > j {
			t.Fatal("label 1 drawn after label 2")
		}
	})
}
