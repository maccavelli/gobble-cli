// Package catalog is the model catalog.
// Stability: beta
//
// It holds Pi's rules over a model's facts: its thinking levels (Levels,
// Clamp) and a call's cost (Price); models.json (ParseModelsJSON); and Pi's
// model resolution. It holds no model data: gobble downloads models.dev's
// document at run time, and llm/provider parses it through
// go-llmprovider-sdk, which only it may import, and fills llm.Model from it
// (0005-PLAN F4-2a; 0072-MADR D4, D14).
package catalog
