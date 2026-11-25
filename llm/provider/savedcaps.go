package provider

import (
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

	"github.com/maccavelli/gobble-cli/llm"
)

// savedCapAge is how long a cap a service stated on an earlier run is
// handed back: a day, the SDK's window for a remembered cap (0066-MADR D7;
// go-llmprovider-sdk 0089-MADR D5).
const savedCapAge = 24 * time.Hour

// savedCapSkew is how far after now a saved cap's Seen may be and still be
// fresh: a clock step between its write and this read. A cap seen later is
// not stated (0071-PLAN's review deviation of 2026-11-25, R2).
const savedCapSkew = 5 * time.Minute

// savedCaps is the caps services stated on earlier runs for one provider, as
// a caller's layer named "service-stated" that states each model's output
// alone (0071-MADR P21, P24, the owner's "Hand them in as a caller layer
// (Recommended)"): a saved cap whose model matches in lower case, whose base
// URL and scope are the options', and whose Seen is within savedCapAge and
// not more than savedCapSkew after now. It is
// handed below models.json's layer, so models.json outranks it and it
// outranks the listing and the document; the SDK ranks it.
type savedCaps struct {
	obs   []llm.LimitObservation
	base  string
	scope string
	now   time.Time
}

// savedLayer is o's saved caps for provider id as savedCaps, or nil when o
// has none. The scope is a Claude key's workspace; gobble passes no Kilo
// organization, so every other scope is empty (0071-MADR F25).
func savedLayer(o Options, id string) catalog.Layer {
	if len(o.Observations) == 0 {
		return nil
	}
	scope := ""
	if id == string(llmprovider.ProviderClaude) {
		scope = o.Workspace
	}
	return savedCaps{obs: o.Observations, base: trimBase(o.BaseURL), scope: strings.TrimSpace(scope), now: time.Now()}
}

func (savedCaps) Name() string { return sourceServiceStated }

// Model states the first saved cap of id that applies: the newest, as the
// store keeps them.
func (l savedCaps) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	for _, o := range l.obs {
		if !strings.EqualFold(strings.TrimSpace(o.Model), strings.TrimSpace(id)) || trimBase(o.BaseURL) != l.base ||
			strings.TrimSpace(o.Scope) != l.scope || o.Tokens < 1 || o.Seen.IsZero() || l.now.Sub(o.Seen) >= savedCapAge ||
			o.Seen.Sub(l.now) > savedCapSkew {
			continue
		}
		return catalog.ModelInfo{MaxOutputTokens: o.Tokens}.Known(catalog.FieldMaxOutputTokens), true
	}
	return catalog.ModelInfo{}, false
}

// trimBase is a base URL as an observation names it: without spaces or
// trailing slashes; empty is the provider's default.
func trimBase(base string) string { return strings.TrimRight(strings.TrimSpace(base), "/") }
