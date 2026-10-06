package provider

import (
	"errors"
	"fmt"
	"slices"
	"strings"

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

// Choose is Ambient's choice without building the provider: the first
// built-in provider, in the SDK's registry order, that needs a key and has
// its standard variable set, with its first curated model (0002-PLAN Phase 3,
// owner's decision of 2026-10-05). env reads the environment.
func Choose(env func(string) string) (Selection, error) {
	for _, d := range providers.Default().Descriptors() {
		if !d.RequiresAPIKey || d.EnvVar == "" || env(d.EnvVar) == "" {
			continue
		}
		s := Selection{ID: string(d.ID), EnvVar: d.EnvVar}
		if len(d.StaticModels) > 0 {
			s.Model = d.StaticModels[0]
		}
		return s, nil
	}
	return Selection{}, fmt.Errorf("%w: set one of %s", ErrNoCredential, strings.Join(CredentialVars(), ", "))
}

// Ambient builds the provider Choose picks, with the credential its variable
// holds. o's APIKey and Model are ignored.
func Ambient(env func(string) string, o Options) (llm.Provider, Selection, error) {
	s, err := Choose(env)
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
