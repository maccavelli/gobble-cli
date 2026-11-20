package provider

import (
	"errors"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// builtins is the process's built-in registry, built once (0036-MADR D2).
var builtins = sync.OnceValue(providers.Default)

// builtinDescs is that registry's descriptors, copied once. A lookup reads
// them and does not build a registry.
var builtinDescs = sync.OnceValue(func() []llmprovider.Descriptor {
	return builtins().Descriptors()
})

// Set is the providers one run can build. A nil *Set is the built-ins. The
// SDK registry it wraps stays in this package (0036-MADR D2).
type Set struct {
	reg *llmprovider.Registry
}

// NewSet is a run's set: the built-ins, in a registry of their own, so a
// provider this run registers later is not on the process's built-ins.
func NewSet() *Set {
	return &Set{reg: providers.Default()}
}

// registry is the set's SDK registry. A nil set is the process's built-ins.
func (s *Set) registry() *llmprovider.Registry {
	if s == nil || s.reg == nil {
		return builtins()
	}
	return s.reg
}

// own reports whether s is a run's registry, not the process's built-ins.
func (s *Set) own() bool {
	return s != nil && s.reg != nil && s.reg != builtins()
}

// descriptors are the set's descriptors. The built-ins' are the copy made once.
func (s *Set) descriptors() []llmprovider.Descriptor {
	if !s.own() {
		return builtinDescs()
	}
	return s.reg.Descriptors()
}

// IDs are the set's provider ids, in registry order.
func (s *Set) IDs() []string {
	var ids []string
	for _, d := range s.descriptors() {
		ids = append(ids, string(d.ID))
	}
	return ids
}

// Descriptor is the set's descriptor for id.
func (s *Set) Descriptor(id string) (llmprovider.Descriptor, bool) {
	if !s.own() {
		for _, d := range builtinDescs() {
			if string(d.ID) == id {
				return d, true
			}
		}
		return llmprovider.Descriptor{}, false
	}
	return s.reg.Descriptor(llmprovider.ProviderID(id))
}

// register adds a provider to this run's set. The built-ins take none: a
// registration there would show up in every run. P4 registers a Chat
// Completions server here (0036-MADR D2, D5).
func (s *Set) register(d llmprovider.Descriptor, f llmprovider.Factory) error {
	if !s.own() {
		return errors.New("provider: the built-in set cannot take a registration")
	}
	return s.reg.Register(d, f)
}
