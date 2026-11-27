package acpserver

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// D17 (0072-MADR's ninth addition): a session's discovery reads its own
// document: after a refresh, an open session's model switch discovers with
// the document the session began with, and a session started after the
// refresh with the new one.
func TestDiscoverReadsSessionModelData_0072(t *testing.T) {
	d1, d2 := dataDoc(t, "https://one.invalid/api.json", time.Now()), dataDoc(t, "https://two.invalid/api.json", time.Now())
	var cur atomic.Pointer[provider.ModelData]
	cur.Store(d1)
	var mu sync.Mutex
	var got []string
	h := newLimitHarnessWith(t, SessionSettings{}, map[string]llm.Model{}, func(o *Options) {
		o.ModelData = cur.Load
		o.Discover = func(_ context.Context, d *provider.ModelData, id, model string) error {
			src := "none"
			if d != nil {
				src = d.Stamp().Source
			}
			mu.Lock()
			got = append(got, id+"/"+model+" "+src)
			mu.Unlock()
			return nil
		}
	})
	seen := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(got)
	}
	open := h.session()
	waitFor(t, func() bool { return len(seen()) == 1 })
	cur.Store(d2) // a refresh lands while the session is open
	if _, err := setOption(t, h.c, open, configModel, "m2"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return len(seen()) == 2 })
	h.session()
	waitFor(t, func() bool { return len(seen()) == 3 })
	want := []string{"fake/m1 https://one.invalid/api.json", "fake/m2 https://one.invalid/api.json", "fake/m1 https://two.invalid/api.json"}
	if g := seen(); !slices.Equal(g, want) {
		t.Errorf("discoveries %q; want %q: the open session's own document, then the new one for a new session", g, want)
	}
}
