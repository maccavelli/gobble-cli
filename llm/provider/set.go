package provider

import (
	"errors"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"

	gcatalog "github.com/maccavelli/gobble-cli/llm/catalog"
)

// builtins is the process's built-in registry, built once (0036-MADR D2).
var builtins = sync.OnceValue(providers.Default)

// builtinDescs is that registry's descriptors, copied once. A lookup reads
// them and does not build a registry.
var builtinDescs = sync.OnceValue(func() []llmprovider.Descriptor {
	return builtins().Descriptors()
})

// chatServer is a Chat Completions server registered on a run's set.
type chatServer struct {
	base string
}

// Set is the providers one run can build. A nil *Set is the built-ins. The
// SDK registry it wraps stays in this package (0036-MADR D2).
type Set struct {
	reg *llmprovider.Registry
	// chat are the Chat Completions servers registered on this set, by id
	// (0036-MADR D5).
	chat map[string]chatServer
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

// AddChatServers registers mj's Chat Completions servers on this run's set.
// Each is built with chatcompat.New, its id and its base URL (0036-MADR D5).
// A server already on the set is left as it is. The built-in set takes none.
func (s *Set) AddChatServers(mj *gcatalog.ModelsJSON) error {
	if mj == nil {
		return nil
	}
	for _, srv := range mj.ChatServers() {
		if _, ok := s.chatBase(srv.ID); ok {
			continue
		}
		id, base, label := srv.ID, srv.BaseURL, srv.Label
		err := s.register(chatcompat.Descriptor(llmprovider.ProviderID(id), label),
			func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
				all := make([]llmprovider.Option, 0, 2+len(opts))
				all = append(all, chatcompat.WithID(llmprovider.ProviderID(id)), llmprovider.WithBaseURL(base))
				all = append(all, opts...)
				return chatcompat.New(all...)
			})
		if err != nil {
			return err
		}
		if s.chat == nil {
			s.chat = map[string]chatServer{}
		}
		s.chat[id] = chatServer{base: base}
	}
	return nil
}

// chatBase is the base URL of a Chat Completions server on this set.
func (s *Set) chatBase(id string) (string, bool) {
	if s == nil {
		return "", false
	}
	c, ok := s.chat[id]
	return c.base, ok
}

// chatOptions are the SDK options that name id as a Chat Completions server
// on this set: WithID, and WithBaseURL when o does not set one. Nil when id
// is not one (0036-MADR D5).
func (o Options) chatOptions(id string) []llmprovider.Option {
	base, ok := o.Set.chatBase(id)
	if !ok {
		return nil
	}
	opts := []llmprovider.Option{chatcompat.WithID(llmprovider.ProviderID(id))}
	if o.BaseURL == "" && base != "" {
		opts = append(opts, llmprovider.WithBaseURL(base))
	}
	return opts
}

// chatDescriptor is the descriptor of a Chat Completions server on this set.
func (s *Set) chatDescriptor(id string) (llmprovider.Descriptor, bool) {
	if _, ok := s.chatBase(id); !ok {
		return llmprovider.Descriptor{}, false
	}
	return s.Descriptor(id)
}
