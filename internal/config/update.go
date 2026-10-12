package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// Init writes Template() to file, creating its directory; an existing file
// is an error (0018-MADR D9).
func Init(file string) (err error) {
	release, err := lockSettings(file)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, release()) }()
	switch _, err := os.Stat(file); {
	case err == nil:
		return fmt.Errorf("%s already exists; gobble config update adds the settings it lacks", file)
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return fsx.WriteFileMode(file, Template(), 0o644)
}

// Update adds the template's blocks for keys the file lacks, active or
// commented-out, without changing an existing line (D9). It returns the
// keys it added and a "+"-prefixed listing of the lines; with dryRun it
// writes nothing.
func Update(file string, dryRun bool) (added []string, preview []byte, err error) {
	release, err := lockSettings(file)
	if err != nil {
		return nil, nil, err
	}
	defer func() { err = errors.Join(err, release()) }()
	data, err := os.ReadFile(file) //nolint:gosec // G304: the settings file the caller names
	if err != nil {
		return nil, nil, err
	}
	out, added, preview, err := grow(file, data)
	if err != nil || len(added) == 0 || dryRun {
		return added, preview, err
	}
	return added, preview, fsx.WriteFileMode(file, out, 0o644)
}

// ruler is a section's ruler line in the template.
var ruler = regexp.MustCompile(`^# -+$`)

// tmplBlock is a run of the template's non-blank lines holding keys: its
// prose and its commented key lines.
type tmplBlock struct {
	section int
	text    []string
	keys    []tmplKey // leaf keys, in order
}

type tmplKey struct {
	path string
	line string // its commented line, as the template writes it
}

// templateLayout splits the template into its section headings (ruler,
// title, ruler) and its key blocks.
func templateLayout() (headings [][]string, blocks []tmplBlock) {
	var run []string
	var stack []string
	flush := func() {
		switch {
		case len(run) == 3 && ruler.MatchString(run[0]) && ruler.MatchString(run[2]):
			headings = append(headings, run)
		case len(run) > 0:
			b := tmplBlock{section: len(headings) - 1, text: run}
			for _, line := range run {
				ind, key, rest, ok := commentedKey(line)
				if !ok || ind%2 != 0 || ind/2 > len(stack) {
					continue
				}
				stack = append(stack[:ind/2], key)
				if strings.TrimSpace(rest) != "" {
					b.keys = append(b.keys, tmplKey{path: strings.Join(stack, "."), line: line})
				}
			}
			if len(b.keys) > 0 {
				blocks = append(blocks, b)
			}
		}
		run = nil
	}
	for line := range strings.SplitSeq(strings.TrimRight(string(Template()), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		run = append(run, line)
	}
	flush()
	return headings, blocks
}

// activeKey is an uncommented key line, as far as its name and indentation.
var activeKey = regexp.MustCompile(`^( *)([A-Za-z][A-Za-z0-9_-]*):`)

// keyLines is the 0-based last line each key occupies in the file, active
// or commented-out, by path. Paths are built from the key lines above by
// indentation, two spaces a level.
func keyLines(l *lines) map[string]int {
	out := map[string]int{}
	var stack []string
	for i, line := range l.text {
		ind, key, ok := 0, "", false
		if m := activeKey.FindStringSubmatch(line); m != nil {
			ind, key, ok = len(m[1]), m[2], true
		} else if cind, ckey, _, cok := commentedKey(line); cok {
			ind, key, ok = cind, ckey, true
		}
		if !ok || ind%2 != 0 || ind/2 > len(stack) {
			continue
		}
		stack = append(stack[:ind/2], key)
		out[strings.Join(stack, ".")] = i
	}
	return out
}

// insertion is lines to put before the file's line at; a block is set off
// by blank lines, a single key line is not.
type insertion struct {
	at, order int
	text      []string
	block     bool
}

// grow computes Update's result: the new file, the keys added and the
// preview, and checks that every old line is kept, in order, and that the
// settings read as before.
func grow(file string, data []byte) ([]byte, []string, []byte, error) {
	root, ds := Parse(file, data)
	if root == nil {
		return nil, nil, nil, diagnosticsError(ds)
	}
	l := splitLines(data)
	present := keyLines(l)
	for _, k := range Keys {
		if steps, err := lookup(root, k.Path); err == nil && len(steps) == len(strings.Split(k.Path, ".")) {
			s := steps[len(steps)-1]
			present[k.Path] = lastLine(l, s.key, s.value) - 1
		}
	}
	headings, blocks := templateLayout()
	var ins []insertion
	var added []string
	pos := map[string]int{} // a key's insertion point, once it has one
	after := func(path string) (int, bool) {
		if at, ok := pos[path]; ok {
			return at, true
		}
		if line, ok := present[path]; ok {
			return line + 1, true
		}
		return 0, false
	}
	add := func(i insertion) {
		i.order = len(ins)
		ins = append(ins, i)
	}
	headed := map[int]bool{}
	for bi, b := range blocks {
		var missing []tmplKey
		for _, k := range b.keys {
			if _, ok := present[k.path]; !ok {
				missing = append(missing, k)
			}
		}
		if len(missing) == 0 {
			continue
		}
		for _, k := range missing {
			added = append(added, k.path)
		}
		if len(missing) < len(b.keys) {
			for ki, k := range b.keys {
				if !slices.Contains(missing, k) {
					continue
				}
				at, ok := siblingPoint(b.keys, ki, after, present)
				if !ok {
					return nil, nil, nil, unsafeEdit(file, k.path)
				}
				add(insertion{at: at, text: []string{k.line}})
				pos[k.path] = at
			}
			continue
		}
		at, ok := blockPoint(blocks, bi, after)
		if !ok {
			at, ok = headingPoint(l, headings[b.section])
		}
		if !ok {
			at = len(l.text)
			if !headed[b.section] {
				add(insertion{at: at, text: headings[b.section], block: true})
				headed[b.section] = true
			}
		}
		add(insertion{at: at, text: b.text, block: true})
		for _, k := range b.keys {
			pos[k.path] = at
		}
	}
	if len(ins) == 0 {
		return data, nil, nil, nil
	}
	out, preview := place(l, ins)
	if !keepsEveryLine(data, out) || !readsSame(file, root, out) {
		return nil, nil, nil, fmt.Errorf("%s: the update could not be applied safely; the file is unchanged", file)
	}
	return out, added, preview, nil
}

// siblingPoint is where a missing key of a partly present block goes: after
// its nearest earlier sibling, or else before its nearest later one.
func siblingPoint(keys []tmplKey, ki int, after func(string) (int, bool), present map[string]int) (int, bool) {
	for j := ki - 1; j >= 0; j-- {
		if at, ok := after(keys[j].path); ok {
			return at, true
		}
	}
	for j := ki + 1; j < len(keys); j++ {
		if line, ok := present[keys[j].path]; ok {
			return line, true
		}
	}
	return 0, false
}

// blockPoint is after the nearest earlier block of the same section that
// the file has.
func blockPoint(blocks []tmplBlock, bi int, after func(string) (int, bool)) (int, bool) {
	for j := bi - 1; j >= 0 && blocks[j].section == blocks[bi].section; j-- {
		best, found := 0, false
		for _, k := range blocks[j].keys {
			if at, ok := after(k.path); ok {
				best, found = max(best, at), true
			}
		}
		if found {
			return best, true
		}
	}
	return 0, false
}

// headingPoint is after the section's heading in the file: its title line
// with a ruler below it.
func headingPoint(l *lines, heading []string) (int, bool) {
	for i := 0; i+1 < len(l.text); i++ {
		if l.text[i] == heading[1] && ruler.MatchString(l.text[i+1]) {
			return i + 2, true
		}
	}
	return 0, false
}

// place inserts each group of insertions before its line, blocks set off
// by one blank line each side, and returns the file and the preview.
func place(l *lines, ins []insertion) ([]byte, []byte) {
	groups := map[int][]insertion{}
	for _, i := range ins {
		groups[i.at] = append(groups[i.at], i)
	}
	var ops []op
	var preview bytes.Buffer
	for _, at := range slices.Sorted(maps.Keys(groups)) {
		var text []string
		block := false
		for _, i := range groups[at] {
			if i.block {
				text = append(text, "")
				block = true
			}
			text = append(text, i.text...)
		}
		if block && at < len(l.text) && strings.TrimSpace(l.text[at]) != "" {
			text = append(text, "")
		}
		for _, t := range text {
			preview.WriteString("+" + t + "\n")
		}
		ops = append(ops, op{from: at, to: at, text: text})
	}
	l.apply(ops)
	return l.bytes(), preview.Bytes()
}

// keepsEveryLine reports whether every line of old is in out, in order.
func keepsEveryLine(old, out []byte) bool {
	a, b := splitLines(old), splitLines(out)
	j := 0
	for i, t := range a.text {
		for j < len(b.text) && (b.text[j] != t || (b.eol[j] != a.eol[i] && i != len(a.text)-1)) {
			j++
		}
		if j == len(b.text) {
			return false
		}
		j++
	}
	return true
}

// readsSame reports whether out holds the same settings as root.
func readsSame(file string, root *yaml.Node, out []byte) bool {
	r, _ := Parse(file, out)
	return r != nil && maps.Equal(flatten(root), flatten(r))
}
