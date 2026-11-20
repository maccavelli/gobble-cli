package provider

import (
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The built-ins' registry is built once per process. A lookup that built one
// on every call allocated about 85 times (0036-PLAN P1).
func TestRegistryBuiltOnce(t *testing.T) {
	for _, c := range []struct {
		name string
		fn   func()
	}{
		{"Label", func() { Label("together") }},
		{"CredentialVar", func() { CredentialVar("openai") }},
	} {
		if n := testing.AllocsPerRun(100, c.fn); n > 2 {
			t.Errorf("%s allocates %.0f times a call, want at most 2", c.name, n)
		}
	}
}

// A nil set is the built-ins. The run's set, before anything is registered
// on it, has the same ids, labels and sign-in methods (0036-PLAN P1).
func TestSetHoldsBuiltIns(t *testing.T) {
	var none *Set
	built := NewSet()
	if ids := none.IDs(); len(ids) == 0 || !slices.Equal(ids, built.IDs()) {
		t.Fatalf("ids nil %q, built-ins %q", ids, built.IDs())
	}
	for _, id := range none.IDs() {
		a, oka := none.Descriptor(id)
		b, okb := built.Descriptor(id)
		if !oka || !okb {
			t.Errorf("%s: descriptor on the nil set %v, on the built-ins %v", id, oka, okb)
			continue
		}
		if a.Label != b.Label {
			t.Errorf("%s label %q, built-ins %q", id, a.Label, b.Label)
		}
		if !slices.Equal(a.AuthMethods, b.AuthMethods) {
			t.Errorf("%s methods differ", id)
		}
	}
}

// A registration stays on the run's set. The built-ins take none, so one
// run's provider does not appear in another's (0036-MADR D2).
func TestRegisterStaysOnTheRun(t *testing.T) {
	d := llmprovider.Descriptor{ID: "extra", Label: "Extra"}
	f := func(...llmprovider.Option) (llmprovider.Provider, error) { return nil, nil }
	var none *Set
	if err := none.register(d, f); err == nil {
		t.Fatal("register on the built-ins succeeded")
	}
	run := NewSet()
	if err := run.register(d, f); err != nil {
		t.Fatal(err)
	}
	got, ok := run.Descriptor("extra")
	if !ok || got.Label != "Extra" || !slices.Contains(run.IDs(), "extra") {
		t.Fatalf("run set: ok %v, %+v, ids %q", ok, got, run.IDs())
	}
	if _, ok := none.Descriptor("extra"); ok || slices.Contains(none.IDs(), "extra") {
		t.Fatal("the built-ins gained extra")
	}
}
