package personality

import (
	"embed"
	"fmt"
	"math/rand/v2"
	"sync"
)

//go:embed builtin
var builtin embed.FS

// Boring is the canonical personality's name (0022-MADR D1); Plain and
// Normal are other names for it.
const (
	Boring = "boring"
	Plain  = "plain"
	Normal = "normal"
)

// Args are a message's placeholder values, by name.
type Args map[string]string

// Options choose a Voice: the personality's name, and the seed of its
// variety, 0 being a random one.
type Options struct {
	Name string
	Seed uint64
}

// Voice speaks one personality's catalog, falling back to boring's for a
// key it leaves out. It is safe for concurrent use. A nil *Voice is
// boring, and picks no variety.
type Voice struct {
	name   string
	own    *Catalog
	boring *Catalog

	mu   sync.Mutex
	rng  *rand.Rand
	last map[Key]int // each key's last variant, so it is not picked twice running
}

var (
	boringOnce    sync.Once
	boringCatalog *Catalog
	errBoring     error
)

// boringCat is the built-in boring catalog. It is part of the binary, so
// a failure to read it is a programming error.
func boringCat() *Catalog {
	boringOnce.Do(func() {
		data, err := builtin.ReadFile("builtin/boring/messages.md")
		if err == nil {
			boringCatalog, err = ParseCatalog(data)
		}
		errBoring = err
	})
	if errBoring != nil {
		panic(fmt.Sprintf("personality: the built-in boring catalog: %v", errBoring))
	}
	return boringCatalog
}

// New is the Voice of o.Name: a built-in personality, or boring when the
// name is empty, boring, plain or normal.
func New(o Options) (*Voice, error) {
	name := o.Name
	switch name {
	case "", Plain, Normal:
		name = Boring
	}
	if name != Boring {
		return nil, fmt.Errorf("personality: no personality named %q", o.Name)
	}
	seed := o.Seed
	if seed == 0 {
		seed = rand.Uint64() //nolint:gosec // variety in wording, not security
	}
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) //nolint:gosec // variety in wording, not security
	return &Voice{name: name, boring: boringCat(), rng: rng, last: map[Key]int{}}, nil
}

// Name is the personality's name; boring for a nil Voice.
func (v *Voice) Name() string {
	if v == nil {
		return Boring
	}
	return v.name
}

// Text is key's message with args in its placeholders: one of the
// personality's variants, never the one it gave last time while it has
// another, else boring's. An unknown key is shown as itself, which a test
// keeps from shipping.
func (v *Voice) Text(key Key, args Args) string {
	variants := v.variants(key)
	switch len(variants) {
	case 0:
		return string(key)
	case 1:
		return fill(variants[0], args)
	}
	v.mu.Lock()
	i := v.rng.IntN(len(variants) - 1)
	if last, ok := v.last[key]; ok && i >= last {
		i++
	} else if !ok {
		i = v.rng.IntN(len(variants))
	}
	v.last[key] = i
	v.mu.Unlock()
	return fill(variants[i], args)
}

func (v *Voice) variants(key Key) []string {
	if v == nil {
		return boringCat().Variants(key)
	}
	if vs := v.own.Variants(key); len(vs) > 0 {
		return vs
	}
	return v.boring.Variants(key)
}
