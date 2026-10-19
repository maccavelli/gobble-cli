package personality

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

//go:embed builtin
var builtin embed.FS

// The built-in personalities' names (0022-MADR D1). Boring is the
// canonical one; Plain and Normal are other names for it.
const (
	Boring    = "boring"
	Plain     = "plain"
	Normal    = "normal"
	Dry       = "dry"
	Snarky    = "snarky"
	Bombastic = "bombastic"
)

// Builtins are the built-in personalities, by level.
var Builtins = []string{Boring, Dry, Snarky, Bombastic}

// BoringSummary is boring's line in /personality's list; boring has no
// brief to give one.
const BoringSummary = "gobble's plain text, with no humor."

// InsultLevel is the level from which a personality may mock or insult
// the user, and must be accepted before it is first taken (0022-MADR D1,
// D10).
const InsultLevel = 5

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
	level  int
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
func Canonical(name string) string { return (*Set)(nil).Canonical(name) }

// New is the Voice of o.Name, a built-in personality or one of boring's
// names; empty is boring, the zero value code relies on (0022-MADR D17).
func New(o Options) (*Voice, error) { return (*Set)(nil).New(o) }

// Set is the personalities a process offers: the built-ins, with the
// user's files over them, and the user's own, from their config directory
// (0022-PLAN P3, P5). A nil Set is the built-ins alone.
type Set struct {
	user  map[string]builtinData
	order []string
	// over are the built-ins the user's files change, each laid over its
	// built-in; dirs are where each user-made or changed one came from.
	over map[string]builtinData
	dirs map[string]string
}

// Info is a personality as /personality and gobble personality list it.
// Source is the directory of a user-made personality, or of the files that
// change a built-in; "" for a built-in as shipped.
type Info struct {
	Name    string
	Level   int
	Summary string
	Builtin bool
	Source  string
}

// LoadSet reads the personalities under dir, one directory each, named
// for the personality, holding personality.md and messages.md: a
// built-in's name changes that built-in (loadOverride), any other makes a
// personality of the user's. The prompts directory is not a personality
// (PromptsDir). A directory that does not make a personality is left out
// with an error naming it. A missing dir is no personalities.
func LoadSet(dir string) (*Set, []error) {
	set := &Set{user: map[string]builtinData{}, over: map[string]builtinData{}, dirs: map[string]string{}}
	if dir == "" {
		return set, nil
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return set, nil
	}
	if err != nil {
		return set, []error{fmt.Errorf("personality: %w", err)}
	}
	var errs []error
	for _, e := range entries {
		if !e.IsDir() || e.Name() == PromptsDir {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if slices.Contains(Builtins, e.Name()) {
			d, err := loadOverride(path, e.Name())
			if err != nil {
				errs = append(errs, err)
				continue
			}
			set.over[e.Name()], set.dirs[e.Name()] = d, path
			continue
		}
		d, err := loadUser(path, e.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		set.user[e.Name()], set.dirs[e.Name()] = d, path
		set.order = append(set.order, e.Name())
	}
	return set, errs
}

// loadUser reads one user-made personality and checks it: its brief names
// it, it is not a built-in's name, and its catalog keeps boring's keys and
// placeholders (0022-MADR D5).
func loadUser(dir, name string) (builtinData, error) {
	fail := func(err error) (builtinData, error) {
		return builtinData{}, fmt.Errorf("personality %s (%s): %w", name, dir, err)
	}
	if (*Set)(nil).Canonical(name) != "" {
		return fail(errors.New("a built-in personality has this name"))
	}
	data, err := os.ReadFile(filepath.Join(dir, "personality.md")) //nolint:gosec // the user's own config directory
	if err != nil {
		return fail(err)
	}
	brief, err := ParseBrief(data)
	if err != nil {
		return fail(err)
	}
	if brief.Name != name {
		return fail(fmt.Errorf("its brief is named %s", brief.Name))
	}
	cat := &Catalog{variants: map[Key][]string{}}
	data, err = os.ReadFile(filepath.Join(dir, "messages.md")) //nolint:gosec // the user's own config directory
	switch {
	case errors.Is(err, fs.ErrNotExist): // every message falls back to boring's
	case err != nil:
		return fail(err)
	default:
		if cat, err = ParseCatalog(data); err != nil {
			return fail(err)
		}
	}
	if err := CheckCatalog(cat); err != nil {
		return fail(err)
	}
	return builtinData{catalog: cat, brief: brief}, nil
}

// CheckCatalog checks a catalog against boring's: every key is boring's or
// a lead, every variant keeps boring's placeholders, and a lead has none
// (0022-MADR D5, D13).
func CheckCatalog(c *Catalog) error {
	for _, k := range c.Keys() {
		switch {
		case slices.Contains(LeadKeys, k):
			for _, v := range c.Variants(k) {
				if ps := Placeholders(v); len(ps) > 0 {
					return fmt.Errorf("%s: a lead has no placeholders, and %q has %v", k, v, ps)
				}
			}
		case slices.Contains(AmbientKeys, k):
			for _, v := range c.Variants(k) {
				if err := CheckPlaceholders(ambientFacts[k], v); err != nil {
					return fmt.Errorf("%s: %w", k, err)
				}
			}
		case slices.Contains(Keys, k):
			for _, v := range c.Variants(k) {
				if err := CheckPlaceholders(boringCat().Variants(k)[0], v); err != nil {
					return fmt.Errorf("%s: %w", k, err)
				}
			}
		default:
			return fmt.Errorf("no message is called %s", k)
		}
	}
	return nil
}

// Canonical is name as the Set knows it: boring for boring's other names,
// a built-in's or a user-made personality's own, or "" for none.
func (set *Set) Canonical(name string) string {
	switch name {
	case Plain, Normal:
		return Boring
	}
	if slices.Contains(Builtins, name) {
		return name
	}
	if set != nil {
		if _, ok := set.user[name]; ok {
			return name
		}
	}
	return ""
}

// Names are the personalities' names: the built-ins by level, then the
// user's, as their directories sort.
func (set *Set) Names() []string {
	out := slices.Clone(Builtins)
	if set != nil {
		out = append(out, set.order...)
	}
	return out
}

// Level is the named personality's level, 0 for none (0022-MADR D1).
func (set *Set) Level(name string) int {
	d, ok := set.data(set.Canonical(name))
	if !ok {
		return 0
	}
	return max(d.brief.Level, 1) // boring has no brief, and is level 1
}

// Infos are the personalities, in Names' order, for /personality's list.
func (set *Set) Infos() []Info {
	var out []Info
	for _, n := range set.Names() {
		d, _ := set.data(n) //nolint:errcheck // Names are all known
		summary := d.brief.Summary
		if n == Boring {
			summary = BoringSummary
		}
		out = append(out, Info{Name: n, Level: set.Level(n), Summary: summary, Builtin: slices.Contains(Builtins, n), Source: set.Dir(n)})
	}
	return out
}

// Dir is where the named personality's files are, as Info.Source.
func (set *Set) Dir(name string) string {
	if set == nil {
		return ""
	}
	return set.dirs[set.Canonical(name)]
}

func (set *Set) data(name string) (builtinData, bool) {
	if set != nil {
		if d, ok := set.over[name]; ok {
			return d, true
		}
	}
	if d, ok := builtins()[name]; ok {
		return d, true
	}
	if set != nil {
		d, ok := set.user[name]
		return d, ok
	}
	return builtinData{}, false
}

// boring is boring's catalog, with the user's changes over it: every
// personality's fallback.
func (set *Set) boring() *Catalog {
	d, _ := set.data(Boring) //nolint:errcheck // boring is always known
	return d.catalog
}

// New is the Voice of o.Name, any personality the Set knows or one of
// boring's names; empty is boring.
func (set *Set) New(o Options) (*Voice, error) {
	name := set.Canonical(o.Name)
	if o.Name == "" {
		name = Boring
	}
	if name == "" {
		return nil, fmt.Errorf("personality: no personality named %q (the personalities are %s)", o.Name, strings.Join(set.Names(), ", "))
	}
	d, _ := set.data(name) //nolint:errcheck // Canonical found it
	seed := o.Seed
	if seed == 0 {
		seed = rand.Uint64() //nolint:gosec // variety in wording, not security
	}
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) //nolint:gosec // variety in wording, not security
	v := &Voice{name: name, level: set.Level(name), boring: set.boring(), rng: rng, last: map[Key]int{}, brief: d.brief}
	if name != Boring {
		v.own = d.catalog
	}
	return v, nil
}

// Level is the voice's level: 1 for boring and a nil Voice.
func (v *Voice) Level() int {
	if v == nil {
		return 1
	}
	return v.level
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

// Deck deals a session's ambient lines (0022-MADR D12's budget): each of
// a key's variants in a voice once, in a random order, before any comes
// round again. The zero Deck is ready.
type Deck struct {
	mu    sync.Mutex
	dealt map[string][]bool
}

// Deal is one of v's own variants of key, with args in its placeholders,
// that d has not dealt for v's personality since it last dealt them all.
// ok is false when the personality has none, as boring has none.
func (d *Deck) Deal(v *Voice, key Key, args Args) (string, bool) {
	if v == nil || v.own == nil {
		return "", false
	}
	variants := v.own.Variants(key)
	if len(variants) == 0 {
		return "", false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.dealt == nil {
		d.dealt = map[string][]bool{}
	}
	id := v.name + "\x00" + string(key)
	used := d.dealt[id]
	if len(used) != len(variants) || !slices.Contains(used, false) {
		used = make([]bool, len(variants))
	}
	var free []int
	for i, u := range used {
		if !u {
			free = append(free, i)
		}
	}
	v.mu.Lock()
	i := free[v.rng.IntN(len(free))]
	v.mu.Unlock()
	used[i] = true
	d.dealt[id] = used
	return fill(variants[i], args), true
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
