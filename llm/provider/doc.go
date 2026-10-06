// Package provider adapts the llm facade to go-llmprovider-sdk.
// Stability: beta
//
// New builds an SDK provider by id, wrapped in the SDK's retry, as an
// llm.Provider; Wrap adapts one built elsewhere. Ambient picks the provider
// from the standard credential variables, in the SDK's registry order, with
// its first curated model, until 0005-PLAN F4 adds flags and a credential
// store. This is the only package that imports the SDK (0004-MADR rule 4).
package provider
