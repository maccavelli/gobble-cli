// Package provider adapts the llm facade to go-llmprovider-sdk.
// Stability: beta
//
// New builds an SDK provider by id as an llm.Provider, wrapped in the SDK's
// retry with Options.Retry, or not wrapped when retry is off; Wrap adapts
// one built elsewhere. A request the network could not carry is
// llm.ErrOffline, and the SDK's empty answer llm.ErrEmptyReply. ChooseWith
// and AmbientWith pick the provider: defaultProvider's, or the first, in
// the SDK's registry order, with a credential from any source. The Grok and Kilo logins and a Pi
// login's import are here too. This is the only package that imports the
// SDK (0004-MADR rule 4).
package provider
