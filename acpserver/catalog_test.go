package acpserver

import (
	"math"
	"slices"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// catalogModels are testModels as a catalog knows them: m1 reasons with
// none, low and high, and has a window and a price; m2 does not reason, and
// has neither.
var catalogModels = map[string]llm.Model{
	"m1": {ContextWindow: 1000, ReasoningKnown: true, Reasoning: true, Efforts: []string{"none", "low", "high"},
		Priced: true, Cost: llm.ModelCost{Input: 2, Output: 10, CacheRead: 0.5, CacheWrite: 2.5}},
	"m2": {ReasoningKnown: true},
}

// catalogAgent is an agent on p whose catalog is catalogModels, joined to
// client.
func catalogAgent(t *testing.T, p llm.Provider, client *acptest.Client) *acptest.Conn {
	t.Helper()
	ag := New(Options{
		Version:  "1.2.3",
		Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Models:   func() ModelChoice { return testModels },
		ModelInfo: func(provider, model string) llm.Model {
			m := catalogModels[model]
			m.Provider, m.ID = provider, model
			return m
		},
		Store: jsonl.New(t.TempDir(), jsonl.Options{}),
	})
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c
}

// thoughtLevel is the thought_level option of opts: its values and current.
func thoughtLevel(t *testing.T, opts []acp.SessionConfigOption) ([]string, string) {
	t.Helper()
	for _, o := range opts {
		if o.Select == nil || string(o.Select.Id) != configThinking {
			continue
		}
		var values []string
		for _, v := range *o.Select.Options.Ungrouped {
			values = append(values, string(v.Value))
		}
		return values, string(o.Select.CurrentValue)
	}
	t.Fatal("no thought_level option")
	return nil, ""
}

func setOption(t *testing.T, c *acptest.Conn, id acp.SessionId, option, value string) ([]acp.SessionConfigOption, error) {
	t.Helper()
	resp, err := c.Client.SetSessionConfigOption(t.Context(), acp.SetSessionConfigOptionRequest{ValueId: &acp.SetSessionConfigOptionValueId{
		SessionId: id, ConfigId: acp.SessionConfigId(option), Value: acp.SessionConfigValueId(value)}})
	if err != nil {
		return nil, err
	}
	return resp.ConfigOptions, nil
}

// entryTypes are the types of the session's entries, in order.
func entryTypes(t *testing.T, c *acptest.Conn, id acp.SessionId) []string {
	t.Helper()
	var types []string
	for _, e := range mustExt(t, c, "_gobble/get_entries", map[string]any{"sessionId": id})["entries"].([]any) {
		types = append(types, e.(map[string]any)["type"].(string))
	}
	return types
}

// thought_level offers the session model's levels; a level it lacks is
// refused; a model change clamps the level, recording the change only when
// there is one (0005-PLAN F4-2a).
func TestCatalogThinkingLevels(t *testing.T) {
	c := catalogAgent(t, llmtest.NewScript(), &acptest.Client{})
	resp := newSession(t, c, t.TempDir(), nil)
	id := resp.SessionId
	if values, current := thoughtLevel(t, resp.ConfigOptions); !slices.Equal(values, []string{"off", "low", "high"}) || current != "off" {
		t.Fatalf("m1: thought_level %q, current %q; want off, low, high and off", values, current)
	}
	if _, err := setOption(t, c, id, configThinking, "medium"); err == nil {
		t.Error("m1 does not offer medium, but setting it succeeded")
	}
	if _, err := setOption(t, c, id, configThinking, "high"); err != nil {
		t.Fatal(err)
	}
	before := len(entryTypes(t, c, id))
	opts, err := setOption(t, c, id, configModel, "m2")
	if err != nil {
		t.Fatal(err)
	}
	if values, current := thoughtLevel(t, opts); !slices.Equal(values, []string{"off"}) || current != "off" {
		t.Errorf("m2: thought_level %q, current %q; want off alone", values, current)
	}
	if got := entryTypes(t, c, id)[before:]; !slices.Equal(got, []string{"model_change", "thinking_level_change"}) {
		t.Errorf("m2: entries %q, want a model_change and the clamp's thinking_level_change", got)
	}
	before = len(entryTypes(t, c, id))
	if _, err := setOption(t, c, id, configModel, "m1"); err != nil {
		t.Fatal(err)
	}
	if got := entryTypes(t, c, id)[before:]; !slices.Equal(got, []string{"model_change"}) {
		t.Errorf("m1 again: entries %q, want the model_change alone: off is offered", got)
	}
}

// lastUsage is the last usage_update client received.
func lastUsage(t *testing.T, client *acptest.Client) *acp.SessionUsageUpdate {
	t.Helper()
	var last *acp.SessionUsageUpdate
	for _, n := range client.Updates() {
		if n.Update.UsageUpdate != nil {
			last = n.Update.UsageUpdate
		}
	}
	if last == nil {
		t.Fatal("no usage_update")
	}
	return last
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

// A priced model with a window: the turn's message carries its cost, and
// usage_update, the stats, /usage and /context report the cost and window.
// llmtest.Text reports 10 tokens in and, for "hi", 2 out: $0.00002 each.
func TestCatalogPriced(t *testing.T) {
	client := &acptest.Client{}
	c := catalogAgent(t, llmtest.NewScript(llmtest.Text("hi")), client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "hello"); err != nil {
		t.Fatal(err)
	}
	const cost = 0.00004
	if u := lastUsage(t, client); u.Used != 12 || u.Size != 1000 || u.Cost == nil || !near(u.Cost.Amount, cost) || u.Cost.Currency != "USD" {
		t.Errorf("usage_update %+v (cost %+v), want 12 of 1000 at $%v", u, u.Cost, cost)
	}
	var total float64
	for _, m := range mustExt(t, c, "_gobble/get_messages", map[string]any{"sessionId": id})["messages"].([]any) {
		if msg := m.(map[string]any); msg["role"] == "assistant" {
			total = msg["usage"].(map[string]any)["cost"].(map[string]any)["total"].(float64)
		}
	}
	if !near(total, cost) {
		t.Errorf("the assistant message's usage.cost.total is %v, want %v", total, cost)
	}
	stats := mustExt(t, c, "_gobble/get_session_stats", map[string]any{"sessionId": id})
	cu, _ := stats["contextUsage"].(map[string]any)
	if !near(stats["cost"].(float64), cost) || cu["tokens"] != float64(12) || cu["contextWindow"] != float64(1000) || !near(cu["percent"].(float64), 1.2) {
		t.Errorf("stats cost %v contextUsage %v, want %v and 12 of 1000 (1.2%%)", stats["cost"], cu, cost)
	}
	if usage, _ := slash(t, c, client, id, "/usage"); !strings.Contains(usage, "Cost: $0.000\n") {
		t.Errorf("/usage %q lacks the cost", usage)
	}
	if info, _ := slash(t, c, client, id, "/context"); !strings.Contains(info, "Context: 12 of 1,000 tokens (1.2%)\n") {
		t.Errorf("/context %q lacks the window", info)
	}
}

// A model with no window and no price: usage_update has no size or cost,
// the stats no contextUsage, and /usage and /context say so.
func TestCatalogUnknown(t *testing.T) {
	client := &acptest.Client{}
	c := catalogAgent(t, llmtest.NewScript(llmtest.Text("hi")), client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := setOption(t, c, id, configModel, "m2"); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, id, "hello"); err != nil {
		t.Fatal(err)
	}
	if u := lastUsage(t, client); u.Size != 0 || u.Cost != nil {
		t.Errorf("usage_update %+v (cost %+v), want no size and no cost", u, u.Cost)
	}
	if stats := mustExt(t, c, "_gobble/get_session_stats", map[string]any{"sessionId": id}); stats["contextUsage"] != nil || stats["cost"] != float64(0) {
		t.Errorf("stats contextUsage %v cost %v, want none and 0", stats["contextUsage"], stats["cost"])
	}
	if usage, _ := slash(t, c, client, id, "/usage"); !strings.Contains(usage, "Cost: unknown (no price for fake/m2 in the catalog)\n") {
		t.Errorf("/usage %q", usage)
	}
	if info, _ := slash(t, c, client, id, "/context"); !strings.Contains(info, "tokens (estimated) of an unknown window\n") {
		t.Errorf("/context %q", info)
	}
}
