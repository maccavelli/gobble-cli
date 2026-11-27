package provider

import (
	"strings"
	"testing"

	gcatalog "github.com/maccavelli/gobble-cli/llm/catalog"
)

// layerDoc is a document whose openai model "layered" has window w.
func layerDoc(t *testing.T, w string) *ModelData {
	t.Helper()
	d, err := ParseModelData(strings.NewReader(`{"openai":{"models":{"layered":{"name":"Layered","limit":{"context":` + w + `,"output":1000}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	return d.Stamped(ModelDataStamp{Source: "https://layer.invalid/api.json"})
}

// M51 (a guard), M80: models.json is the layer above the document a lookup
// reads, and that document is the session's, Options.ModelData, not the
// current one (0072-MADR D16, D19).
func TestModelsJSONAboveModelData_0072(t *testing.T) {
	keepModelData(t)
	session, current := layerDoc(t, "50000"), layerDoc(t, "70000")
	SetCurrentModelData(current)
	// The name is a fact the higher layer simply wins; a window is the
	// smallest any source states, whatever their order.
	mj, _, err := gcatalog.ParseModelsJSON([]byte(`{"providers":{"openai":{"modelOverrides":{"layered":{"name":"From models.json"}}}}}`),
		SDKVersion(), (*Set)(nil).IDs())
	if err != nil {
		t.Fatal(err)
	}
	o := Options{ModelData: session}
	if m := ModelInfoWith(o, "openai", "layered", mj); m.Name != "From models.json" || m.ContextWindow != 50000 {
		t.Errorf("with models.json: name %q, window %d; want models.json's name above the session's document, its window 50,000", m.Name, m.ContextWindow)
	}
	if m := ModelInfoWith(o, "openai", "layered", nil); m.ContextWindow != 50000 {
		t.Errorf("the session's document: window %d; want its 50,000, not the current one's", m.ContextWindow)
	}
	if l := CachedLimits(o, "openai", "layered", nil); l.Window.Tokens != 50000 {
		t.Errorf("CachedLimits: window %+v; want the session's document's 50,000", l.Window)
	}
	if m := ModelInfoWith(Options{}, "openai", "layered", nil); m.ContextWindow != 70000 {
		t.Errorf("no session document: window %d; want the current one's 70,000", m.ContextWindow)
	}
}
