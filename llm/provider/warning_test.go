package provider

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	sdktest "github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"

	"github.com/maccavelli/gobble-cli/llm"
)

// TestWarningsYielded: a done response's warnings become llm.Warning events,
// each with the SDK's kind, model and detail, before Usage and Done
// (0036-MADR D3). An unknown kind is passed through unchanged.
func TestWarningsYielded(t *testing.T) {
	f := sdktest.NewFake("fake", toolCaps).Reply(&llmprovider.Response{
		Output:       []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "ok"}},
		FinishReason: llmprovider.FinishStop,
		Warnings: []llmprovider.Warning{
			{Kind: llmprovider.WarningEffortDegraded, Model: "m", Detail: "sent low"},
			{Kind: "no_such_kind", Model: "m2", Detail: "unknown"},
		},
	})
	evs, err := drain(t, Wrap(f), &llm.Request{Messages: []llm.Message{text(llm.RoleUser, "hi")}})
	if err != nil {
		t.Fatal(err)
	}
	var warnings []llm.Warning
	var sawUsage, sawDone bool
	for _, ev := range evs {
		switch ev := ev.(type) {
		case llm.Warning:
			if sawUsage || sawDone {
				t.Fatalf("warning after usage or done: %+v", evs)
			}
			warnings = append(warnings, ev)
		case llm.Usage:
			sawUsage = true
		case llm.Done:
			sawDone = true
		}
	}
	if !sawUsage || !sawDone {
		t.Fatalf("events %+v", evs)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings %+v in %+v", warnings, evs)
	}
	if warnings[0] != (llm.Warning{Kind: llm.WarningEffortDegraded, Model: "m", Detail: "sent low"}) {
		t.Fatalf("first warning %+v", warnings[0])
	}
	if warnings[1] != (llm.Warning{Kind: "no_such_kind", Model: "m2", Detail: "unknown"}) {
		t.Fatalf("second warning %+v", warnings[1])
	}
}
