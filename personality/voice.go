package personality

import (
	"embed"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
)

//go:embed builtin
var builtin embed.FS

// The built-in personalities' names (0022-MADR D1). Boring is the
// canonical one; Plain and Normal are other names for it.
const (
	Boring = "boring"
	Plain  = "plain"
	Normal = "normal"
	Dry    = "dry"
)

// Builtins are the built-in personalities, by level.
var Builtins = []string{Boring, Dry}

// Default is the personality a session takes when nothing chooses one
// (0022-MADR D1, the owner's answer 4).
const Default = Dry

// Args are a message's placeholder values, by name.
type Args map[string]string

// Options choose a Voice: the personality's name, and the seed of its
// variety, 0 being a random one.
type Options struct {
	Name string
	Seed uint64
}

// Voice speaks one personality: its catalog, falling back to boring's for
// a key it leaves out, and its brief for the system prompt. It is safe for
// concurrent use. A nil *Voice is boring, and picks no variety.
type Voice struct {
	name   string
	own    *Catalog
	boring *Catalog
	brief  Brief

	mu   sync.Mutex
	rng  *rand.Rand
	last map[Key]int // each key's last variant, so it is not picked twice running
}

// builtinData is a built-in personality's files, parsed once. They are
// part of the binary, so a failure to read one is a programming error.
type builtinData struct {
	catalog *Catalog
	brief   Brief
}

var (
	loadOnce sync.Once
	loaded   map[string]builtinData
	rules    string
)

func builtins() map[string]builtinData {
	loadOnce.Do(func() {
		loaded = map[string]builtinData{}
		for _, name := range Builtins {
			var d builtinData
			data, err := builtin.ReadFile("builtin/" + name + "/messages.md")
			if err == nil {
				d.catalog, err = ParseCatalog(data)
			}
			if err == nil && name != Boring {
				if data, err = builtin.ReadFile("builtin/" + name + "/personality.md"); err == nil {
					d.brief, err = ParseBrief(data)
				}
			}
			if err != nil {
				panic(fmt.Sprintf("personality: the built-in %s: %v", name, err))
			}
			loaded[name] = d
		}
		data, err := builtin.ReadFile("builtin/rules.md")
		if err != nil {
			panic(fmt.Sprintf("personality: the built-in rules: %v", err))
		}
		rules = strings.TrimSpace(string(data))
	})
	return loaded
}

func boringCat() *Catalog { return builtins()[Boring].catalog }

// Canonical is name with boring's other names as boring, or "" when no
// built-in personality has it.
func Canonical(name string) string {
	switch name {
	case Plain, Normal:
		return Boring
	}
	if slices.Contains(Builtins, name) {
		return name
	}
	return ""
}

// New is the Voice of o.Name, a built-in personality or one of boring's
// names; empty is boring, the zero value code relies on (0022-MADR D17).
func New(o Options) (*Voice, error) {
	name := Canonical(o.Name)
	if o.Name == "" {
		name = Boring
	}
	if name == "" {
		return nil, fmt.Errorf("personality: no personality named %q (the personalities are %s)", o.Name, strings.Join(Builtins, ", "))
	}
	d := builtins()[name]
	seed := o.Seed
	if seed == 0 {
		seed = rand.Uint64() //nolint:gosec // variety in wording, not security
	}
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) //nolint:gosec // variety in wording, not security
	v := &Voice{name: name, boring: boringCat(), rng: rng, last: map[Key]int{}, brief: d.brief}
	if name != Boring {
		v.own = d.catalog
	}
	return v, nil
}

// Name is the personality's name; boring for a nil Voice.
func (v *Voice) Name() string {
	if v == nil {
		return Boring
	}
	return v.name
}

// Brief is the personality's brief, the system prompt's personality
// section, or "" for boring, which has none (0022-MADR D4).
func (v *Voice) Brief() string {
	if v == nil {
		return ""
	}
	return v.brief.Body
}

// Rules are the house rules, the system prompt's personality_rules
// section, under every personality but boring (0022-MADR D3). No
// personality changes them.
func (v *Voice) Rules() string {
	if v == nil || v.name == Boring {
		return ""
	}
	builtins()
	return rules
}

// Text is key's message with args in its placeholders: one of the
// personality's variants, never the one it gave last time while it has
// another, else boring's. An unknown key is shown as itself, which a test
// keeps from shipping.
func (v *Voice) Text(key Key, args Args) string {
	variants := v.choices(key)
	if len(variants) == 0 {
		return string(key)
	}
	return fill(v.pick(key, variants), args)
}

// Variants are key's messages as this voice may give them, each with args
// in its placeholders, in catalog order.
func (v *Voice) Variants(key Key, args Args) []string {
	vs := v.choices(key)
	out := make([]string, len(vs))
	for i, s := range vs {
		out[i] = fill(s, args)
	}
	return out
}

// Lead is the line a personality puts before an error or a warning, and
// whether it has one; boring has none (0022-MADR D13).
func (v *Voice) Lead(key Key) (string, bool) {
	if v == nil || v.own == nil {
		return "", false
	}
	variants := v.own.Variants(key)
	if len(variants) == 0 {
		return "", false
	}
	return v.pick(key, variants), true
}

// pick is one of variants, never the one key gave last while there are
// others.
func (v *Voice) pick(key Key, variants []string) string {
	if v == nil || len(variants) == 1 {
		return variants[0]
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	last, seen := v.last[key]
	var i int
	if seen {
		i = v.rng.IntN(len(variants) - 1)
		if i >= last {
			i++
		}
	} else {
		i = v.rng.IntN(len(variants))
	}
	v.last[key] = i
	return variants[i]
}

func (v *Voice) choices(key Key) []string {
	if v == nil {
		return boringCat().Variants(key)
	}
	if vs := v.own.Variants(key); len(vs) > 0 {
		return vs
	}
	return v.boring.Variants(key)
}
