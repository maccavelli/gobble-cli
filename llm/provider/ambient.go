package provider

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"

	"github.com/maccavelli/gobble-cli/llm"
)

// ErrNoCredential is Ambient's error when no credential variable is set. It
// matches llm.ErrAuth.
var ErrNoCredential = fmt.Errorf("%w: no model credential", llm.ErrAuth)

// Selection is the provider Ambient chose.
type Selection struct {
	ID, Model, EnvVar string
}

// Choose is Ambient's choice without building the provider. With preferred,
// the defaultProvider setting, empty: the first built-in provider, in the
// SDK's registry order, that needs a key and has its standard variable set
// (0002-PLAN Phase 3, owner's decision of 2026-10-05). With preferred set:
// that provider when its variable is set, and otherwise ErrNoCredential
// naming that variable, never another provider (0019-MADR D15). Either way
// with its first curated model. env reads the environment.
func Choose(env func(string) string, preferred string) (Selection, error) {
	if preferred != "" {
		d, ok := keyProvider(preferred)
		switch {
		case !ok:
			return Selection{}, fmt.Errorf("provider: defaultProvider %q is not a built-in provider that takes a key", preferred)
		case env(d.EnvVar) == "":
			return Selection{}, fmt.Errorf("%w: defaultProvider is %s; set %s", ErrNoCredential, preferred, d.EnvVar)
		}
		return selection(d), nil
	}
	for _, d := range providers.Default().Descriptors() {
		if takesKey(d) && env(d.EnvVar) != "" {
			return selection(d), nil
		}
	}
	return Selection{}, fmt.Errorf("%w: set one of %s", ErrNoCredential, strings.Join(CredentialVars(), ", "))
}

func takesKey(d llmprovider.Descriptor) bool { return d.RequiresAPIKey && d.EnvVar != "" }

// keyProvider is the descriptor of the built-in provider id that takes a key.
func keyProvider(id string) (llmprovider.Descriptor, bool) {
	for _, d := range providers.Default().Descriptors() {
		if string(d.ID) == id && takesKey(d) {
			return d, true
		}
	}
	return llmprovider.Descriptor{}, false
}

func selection(d llmprovider.Descriptor) Selection {
	s := Selection{ID: string(d.ID), EnvVar: d.EnvVar}
	if len(d.StaticModels) > 0 {
		s.Model = d.StaticModels[0]
	}
	return s
}

// KeyProviders are the ids of the built-in providers that take a key, in
// registry order: the values defaultProvider may name (0019-MADR D15).
func KeyProviders() []string {
	var ids []string
	for _, d := range providers.Default().Descriptors() {
		if takesKey(d) {
			ids = append(ids, string(d.ID))
		}
	}
	return ids
}

// CredentialVar is the standard credential variable of the built-in
// provider id, or "" for one that takes no key or is unknown.
func CredentialVar(id string) string {
	d, _ := keyProvider(id)
	return d.EnvVar
}

// Ambient builds the provider Choose picks for preferred, with the
// credential its variable holds. o's APIKey and Model are ignored.
func Ambient(env func(string) string, preferred string, o Options) (llm.Provider, Selection, error) {
	s, err := Choose(env, preferred)
	if err != nil {
		return nil, Selection{}, err
	}
	o.APIKey, o.Model = env(s.EnvVar), s.Model
	p, err := New(s.ID, o)
	if err != nil {
		return nil, s, err
	}
	return p, s, nil
}

// CredentialVars are the standard credential variables of the built-in
// providers, in registry order, each once. Tests clear them so no test
// reaches a real provider.
func CredentialVars() []string {
	var vars []string
	for _, d := range providers.Default().Descriptors() {
		if d.EnvVar != "" && !slices.Contains(vars, d.EnvVar) {
			vars = append(vars, d.EnvVar)
		}
	}
	return vars
}

// Models are the curated models of the built-in provider id, in the SDK's
// order, or none for an unknown id.
func Models(id string) []string {
	for _, d := range providers.Default().Descriptors() {
		if string(d.ID) == id {
			return slices.Clone(d.StaticModels)
		}
	}
	return nil
}

// IsNoCredential reports whether err is Ambient's missing-credential error.
func IsNoCredential(err error) bool { return errors.Is(err, ErrNoCredential) }
