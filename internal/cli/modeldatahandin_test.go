package cli

import (
	"runtime"
	"slices"
	"testing"
	"time"
	"weak"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// M62 and M86 (internal/cli's half): the agent's three builders hand the
// document they are given to provider.Options.ModelData, which New hands
// the SDK; nil hands none (0072-MADR D17; "Builders take the document
// (Recommended)").
func TestProviderGetsSessionModelData_0072(t *testing.T) {
	e, _ := modelEnv(t)
	d := stampedAgo(t, map[string]int{"openai": 1}, time.Hour)
	var got []*provider.ModelData
	oldAmbient, oldSmall := ambientProvider, newSmallProvider
	ambientProvider = func(_ func(string) (string, bool), _ string, o provider.Options) (llm.Provider, provider.Selection, error) {
		got = append(got, o.ModelData)
		return llmtest.NewScript(), provider.Selection{}, nil
	}
	newSmallProvider = func(_ string, o provider.Options) (llm.Provider, error) {
		got = append(got, o.ModelData)
		return llmtest.NewScript(), nil
	}
	t.Cleanup(func() { ambientProvider, newSmallProvider = oldAmbient, oldSmall })
	o := e.agentOptions()
	for _, build := range []func() (llm.Provider, error){
		func() (llm.Provider, error) { return o.Provider(d, "s1", llm.DefaultRetry) },
		func() (llm.Provider, error) { return o.ProviderFor(d, "openai", "s1", llm.DefaultRetry) },
		func() (llm.Provider, error) {
			return smallProvider(e, "openai", "gpt-5-mini")(d, "s1", llm.DefaultRetry)
		},
		func() (llm.Provider, error) { return o.Provider(nil, "s2", llm.DefaultRetry) },
	} {
		if _, err := build(); err != nil {
			t.Fatal(err)
		}
	}
	if want := []*provider.ModelData{d, d, d, nil}; !slices.Equal(got, want) {
		t.Errorf("the builders handed New %p; want the session's document %p to each of the three, and none for nil", got, d)
	}
}

// M72 (internal/cli's half): across a refresh the replaced document is
// released once the last session holding it ends: neither the process's
// holder nor the agent keeps it (0072-MADR D23).
func TestModelDataRefreshCost_0072(t *testing.T) {
	r := newDataRun(t, triggerReply())
	realTrigger(t)
	withModel(t, llmtest.NewScript(llmtest.Text("one")))
	old := stampedAgo(t, map[string]int{"openai": 2}, 48*time.Hour) // due
	released := weak.Make(old)
	provider.SetCurrentModelData(old)
	s, err := agentConn(t, r.e).NewSession(t.Context(), t.TempDir(), nil) // takes it
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "hello"}); err != nil { // starts the refresh
		t.Fatal(err)
	}
	settle(r.e)
	if d := provider.CurrentModelData(); d == old || d.Sections()["openai"] != 3 {
		t.Fatalf("current %+v; want the download, which replaced the copy", d.Stamp())
	}
	old = nil //nolint:ineffassign,wastedassign // the test drops its own reference, so only the session holds it
	_ = old
	if err := s.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20 && released.Value() != nil; i++ {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	if released.Value() != nil {
		t.Error("the replaced document is still held after the last session holding it ended and a collection")
	}
}
