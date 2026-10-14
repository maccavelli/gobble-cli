// Package catalog is the model catalog.
// Stability: beta
//
// It holds the embedded models.dev snapshot (Snapshot, Source), made by
// scripts/gen_catalog.py, and Pi's rules over a model's facts: its thinking
// levels (Levels, Clamp) and a call's cost (Price). llm/provider parses the
// snapshot through go-llmprovider-sdk, which only it may import, and fills
// llm.Model from it (0005-PLAN F4-2a). models.json and refresh come later.
package catalog
