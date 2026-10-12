package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"

	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// Edit is one change to a settings file (0018-MADR D8): a dotted path and
// its new value, or a removal when Value is nil.
type Edit struct {
	Path  string
	Value *yaml.Node
}

// Edits lock <file>.lock as mcp.json's do: up to 10 tries 20 ms apart
// while another process holds it (0018-MADR D18).
const (
	lockAttempts = 10
	lockWait     = 20 * time.Millisecond
)

// spliceHook, when set by a test, changes a splice before it is checked.
var spliceHook func([]byte) []byte

// Apply returns data with e applied, or an error naming why it could not be
// applied safely. It never touches a file. The edit is a splice of the
// file's own lines, which keep their endings and every comment (D8), and
// the result is parsed again: the edited path must read back as the value,
// or as absent, and every other path as it was.
func Apply(file string, data []byte, e Edit) ([]byte, error) {
	root, ds := Parse(file, data)
	if root == nil {
		return nil, diagnosticsError(ds)
	}
	out, err := splice(file, data, root, e, true)
	if err != nil {
		return nil, err
	}
	if spliceHook != nil {
		out = spliceHook(out)
	}
	if !readsBack(file, root, out, e) {
		return nil, unsafeEdit(file, e.Path)
	}
	return out, nil
}

// Set sets path to value in file, which must exist. The value is YAML,
// checked as a file value is (D7).
func Set(file, path, value string) error {
	n, err := valueNode(path, value)
	if err != nil {
		return err
	}
	return edit(file, Edit{Path: path, Value: n})
}

// Unset removes path from file, which must exist: a template key goes back
// to its commented-out form, any other key is deleted (D8 case 5).
func Unset(file, path string) error { return edit(file, Edit{Path: path}) }

// edit locks file, reads it, applies e and writes the result atomically.
func edit(file string, e Edit) (err error) {
	release, err := lockSettings(file)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, release()) }()
	data, err := os.ReadFile(file) //nolint:gosec // G304: the settings file the caller names
	if err != nil {
		return err
	}
	out, err := Apply(file, data, e)
	if err != nil || bytes.Equal(out, data) {
		return err
	}
	return fsx.WriteFileMode(file, out, 0o644)
}

// lockSettings takes file's edit lock, creating its directory first.
func lockSettings(file string) (release func() error, err error) {
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		return nil, err
	}
	release, err = filelock.Lock(context.Background(), file+".lock", lockAttempts, lockWait)
	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", file, err)
	}
	return release, nil
}

func unsafeEdit(file, path string) error {
	return fmt.Errorf("%s: the edit to %s could not be applied safely; the file is unchanged", file, path)
}

func aliasEdit(path string) error {
	return fmt.Errorf("%s: its value is a YAML alias; edit the anchor", path)
}

func diagnosticsError(ds []Diagnostic) error {
	errs := make([]error, len(ds))
	for i, d := range ds {
		errs[i] = errors.New(d.String())
	}
	return errors.Join(errs...)
}

// valueNode parses and checks a value for path, as --set's are, and returns
// it as a scalar node of the key's type.
func valueNode(path, text string) (*yaml.Node, error) {
	if isCredentialPath(path) {
		return nil, fmt.Errorf("%s: %s", path, textCredential)
	}
	k, known := KeyOf(path)
	if !known {
		return nil, fmt.Errorf("unknown setting %q", path)
	}
	if text == "" {
		if k.Kind == KindInt || k.Kind == KindBool {
			return nil, fmt.Errorf("%s: want %s, have an empty value", path, want(k))
		}
		return scalarNode(""), nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil || len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s: the value is not YAML", path)
	}
	n := doc.Content[0]
	if isNull(n) {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "null"}, nil
	}
	v, d := convert(k, n)
	if d != "" {
		return nil, fmt.Errorf("%s: %s", path, d)
	}
	return scalarNode(v), nil
}

// scalarNode is v, a key's Go value, as a scalar. A boolean or an integer
// resolves to its own tag; a string is tagged, since "5" would resolve to
// an integer.
func scalarNode(v any) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode}
	switch v := v.(type) {
	case bool:
		n.Value = strconv.FormatBool(v)
	case int64:
		n.Value = strconv.FormatInt(v, 10)
	case string:
		n.Tag, n.Value = "!!str", v
	}
	return n
}

// isEmptyValue reports whether n is "key:" with nothing after it: a plain
// null with no text, anchor or tag.
func isEmptyValue(n *yaml.Node) bool {
	return isNull(n) && n.Value == "" && n.Style == 0 && n.Anchor == ""
}

// lines is a file split into its lines, each kept with its own ending.
type lines struct {
	text []string // without endings
	eol  []string // "\n", "\r\n", or "" for a last line without one
	nl   string   // the ending new lines take: the first line's, else "\n"
}

func splitLines(data []byte) *lines {
	l := &lines{nl: "\n"}
	for s := string(data); s != ""; {
		line, rest, found := strings.Cut(s, "\n")
		eol := ""
		if found {
			eol = "\n"
			if strings.HasSuffix(line, "\r") {
				line, eol = line[:len(line)-1], "\r\n"
			}
		}
		l.text, l.eol, s = append(l.text, line), append(l.eol, eol), rest
	}
	if len(l.eol) > 0 && l.eol[0] == "\r\n" {
		l.nl = "\r\n"
	}
	return l
}

func (l *lines) bytes() []byte {
	var b bytes.Buffer
	for i, t := range l.text {
		b.WriteString(t)
		b.WriteString(l.eol[i])
	}
	return b.Bytes()
}

// op replaces lines [from, to) with text, each line ending as the file's do.
type op struct {
	from, to int
	text     []string
}

// apply applies non-overlapping ops, last first so earlier indexes hold.
func (l *lines) apply(ops []op) {
	slices.SortFunc(ops, func(a, b op) int { return b.from - a.from })
	for _, o := range ops {
		if o.from == len(l.text) && o.from > 0 && l.eol[o.from-1] == "" {
			l.eol[o.from-1] = l.nl // a last line without an ending gains one
		}
		eols := make([]string, len(o.text))
		for i := range eols {
			eols[i] = l.nl
		}
		if o.to > o.from && len(o.text) > 0 {
			eols[len(eols)-1] = l.eol[o.to-1] // a replaced range keeps its last ending
		}
		l.text = slices.Concat(l.text[:o.from], o.text, l.text[o.to:])
		l.eol = slices.Concat(l.eol[:o.from], eols, l.eol[o.to:])
	}
}

// step is one segment of a path found active in the file.
type step struct {
	key, value *yaml.Node
	flow       bool // the key sits in a flow mapping, or under one
}

// lookup walks path through the file's active mappings and returns the
// steps it found: all of them, or the leading ones. A path through an alias
// is refused (D7).
func lookup(root *yaml.Node, path string) ([]step, error) {
	var steps []step
	m, flow := root, root.Style&yaml.FlowStyle != 0
	for seg := range strings.SplitSeq(path, ".") {
		if m == nil || m.Kind != yaml.MappingNode {
			break
		}
		var found *step
		for i := 0; i+1 < len(m.Content); i += 2 {
			if k := m.Content[i]; k.Value == seg {
				found = &step{key: k, value: m.Content[i+1], flow: flow}
			}
		}
		if found == nil {
			break
		}
		if found.value.Kind == yaml.AliasNode {
			return nil, aliasEdit(path)
		}
		steps = append(steps, *found)
		m = found.value
		flow = flow || m.Style&yaml.FlowStyle != 0
	}
	return steps, nil
}

// splice applies e to data's lines; withComments allows D8's case 2.
func splice(file string, data []byte, root *yaml.Node, e Edit, withComments bool) ([]byte, error) {
	steps, err := lookup(root, e.Path)
	if err != nil {
		return nil, err
	}
	segs := strings.Split(e.Path, ".")
	l := splitLines(data)
	switch {
	case e.Value == nil && len(steps) == len(segs):
		return removal(file, l, steps, e.Path)
	case e.Value == nil:
		return data, nil // already absent
	case len(steps) == len(segs):
		return setActive(file, l, steps[len(steps)-1], e)
	case withComments:
		if out, ok := uncomment(l, steps, segs); ok {
			root, _ := Parse(file, out)
			if root == nil {
				return nil, unsafeEdit(file, e.Path)
			}
			return splice(file, out, root, e, false)
		}
	}
	return insert(file, l, root, steps, segs, e)
}

// setActive is D8's cases 1 and 4: the key is active.
func setActive(file string, l *lines, s step, e Edit) ([]byte, error) {
	if s.flow {
		return nil, unsafeEdit(file, e.Path)
	}
	v := s.value
	if v.Kind == yaml.ScalarNode && e.Value.Kind == yaml.ScalarNode && v.Style&(yaml.LiteralStyle|yaml.FoldedStyle) == 0 {
		text, err := render(e.Value)
		if err != nil {
			return nil, err
		}
		if replaceScalar(l, v, text) {
			return l.bytes(), nil
		}
	}
	head, ok := keyHead(l, s.key)
	if !ok {
		return nil, unsafeEdit(file, e.Path)
	}
	body, err := valueLines(head, e.Value, keyIndent(s.key))
	if err != nil {
		return nil, err
	}
	l.apply([]op{{from: s.key.Line - 1, to: lastLine(l, s.key, v), text: body}})
	return l.bytes(), nil
}

// replaceScalar is case 1: the value's own bytes on its line are replaced,
// keeping an anchor; the key, an inline comment and the spaces before it
// stay. It reports false for a scalar that runs over more than one line.
func replaceScalar(l *lines, v *yaml.Node, text string) bool {
	i := v.Line - 1
	line := l.text[i]
	start := byteOffset(line, v.Column)
	if isEmptyValue(v) {
		l.text[i] = line[:start] + " " + text + line[start:] // "key:" with no value
		return true
	}
	pos := skipProperties(line, start)
	end, ok := scalarEnd(line, pos, v.Style)
	if !ok || (v.Style&(yaml.SingleQuotedStyle|yaml.DoubleQuotedStyle) == 0 && line[pos:end] != v.Value) {
		return false
	}
	if v.Anchor != "" {
		text = "&" + v.Anchor + " " + text
	}
	l.text[i] = line[:start] + text + line[end:]
	return true
}

// skipProperties skips an anchor and a tag, and the spaces after each.
func skipProperties(line string, pos int) int {
	for pos < len(line) && (line[pos] == '&' || line[pos] == '!') {
		for pos < len(line) && line[pos] != ' ' && line[pos] != '\t' {
			pos++
		}
		for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
			pos++
		}
	}
	return pos
}

// scalarEnd is where a scalar starting at pos ends on its line: after its
// closing quote, or for a plain one before an inline comment or the line's
// end, without trailing spaces. It reports false for a quote not closed on
// the line.
func scalarEnd(line string, pos int, style yaml.Style) (int, bool) {
	switch {
	case style&yaml.DoubleQuotedStyle != 0:
		for j := pos + 1; j < len(line); j++ {
			switch line[j] {
			case '\\':
				j++
			case '"':
				return j + 1, true
			}
		}
		return 0, false
	case style&yaml.SingleQuotedStyle != 0:
		for j := pos + 1; j < len(line); j++ {
			if line[j] == '\'' {
				if j+1 < len(line) && line[j+1] == '\'' {
					j++
					continue
				}
				return j + 1, true
			}
		}
		return 0, false
	}
	end := len(line)
	for j := pos + 1; j < len(line); j++ {
		if line[j] == '#' && (line[j-1] == ' ' || line[j-1] == '\t') {
			end = j
			break
		}
	}
	return pos + len(strings.TrimRight(line[pos:end], " \t")), true
}

// keyHead is the key's line up to and including its colon.
func keyHead(l *lines, key *yaml.Node) (string, bool) {
	line := l.text[key.Line-1]
	start := byteOffset(line, key.Column)
	end := start + len(key.Value)
	if key.Style&(yaml.SingleQuotedStyle|yaml.DoubleQuotedStyle) != 0 {
		var ok bool
		if end, ok = scalarEnd(line, start, key.Style); !ok {
			return "", false
		}
	}
	colon := strings.IndexByte(line[end:], ':')
	if colon < 0 || strings.TrimSpace(line[end:end+colon]) != "" {
		return "", false
	}
	return line[:end+colon+1], true
}

// valueLines are "head value", or for a mapping or sequence, head and the
// value in block style two spaces deeper than the key (D8 case 4).
func valueLines(head string, v *yaml.Node, indent int) ([]string, error) {
	if v.Kind == yaml.ScalarNode {
		text, err := render(v)
		if err != nil {
			return nil, err
		}
		return []string{head + " " + text}, nil
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	out := []string{head}
	pad := strings.Repeat(" ", indent+2)
	for t := range strings.SplitSeq(strings.TrimRight(b.String(), "\n"), "\n") {
		out = append(out, pad+t)
	}
	return out, nil
}

// insert is case 3: the key and any missing parents go after the last line
// of the nearest parent the file has, at its children's indentation, or two
// spaces deeper than the parent.
func insert(file string, l *lines, root *yaml.Node, steps []step, segs []string, e Edit) ([]byte, error) {
	at, indent := len(l.text), 0
	if len(steps) == 0 && len(root.Content) > 0 {
		at = lastLine(l, nil, root)
	}
	if len(steps) > 0 {
		p := steps[len(steps)-1]
		switch {
		case p.flow:
			return nil, unsafeEdit(file, e.Path)
		case p.value.Kind == yaml.MappingNode && len(p.value.Content) > 0:
			at, indent = lastLine(l, p.key, p.value), keyIndent(p.value.Content[0])
		case isEmptyValue(p.value):
			at, indent = p.key.Line, keyIndent(p.key)+2
		default:
			return nil, unsafeEdit(file, e.Path)
		}
	}
	var text []string
	for i, seg := range segs[len(steps):] {
		pad := strings.Repeat(" ", indent+2*i)
		if len(steps)+i < len(segs)-1 {
			text = append(text, pad+seg+":")
			continue
		}
		body, err := valueLines(pad+seg+":", e.Value, indent+2*i)
		if err != nil {
			return nil, err
		}
		text = append(text, body...)
	}
	l.apply([]op{{from: at, to: at, text: text}})
	return l.bytes(), nil
}

// commentedKey matches a commented-out key as the template writes one: "#",
// one space, the key at its nesting indentation, a colon and a value.
func commentedKey(line string) (indent int, key, rest string, ok bool) {
	body, found := strings.CutPrefix(line, "# ")
	if !found {
		return 0, "", "", false
	}
	trimmed := strings.TrimLeft(body, " ")
	indent = len(body) - len(trimmed)
	key, rest, found = strings.Cut(trimmed, ":")
	if !found || key == "" || !isKeyName(key) || (rest != "" && rest[0] != ' ') {
		return 0, "", "", false
	}
	return indent, key, rest, true
}

func isKeyName(s string) bool {
	if s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return true
}

// uncomment is case 2: it finds the key's commented line, with each missing
// parent commented directly above it in the same comment block, and the
// shallowest inside the nearest active parent; it removes "# " from those
// lines and returns the file, which case 1 then edits.
func uncomment(l *lines, steps []step, segs []string) ([]byte, bool) {
	depth := len(segs) - 1
	for i, line := range l.text {
		ind, key, _, ok := commentedKey(line)
		if !ok || ind != 2*depth || key != segs[depth] {
			continue
		}
		marked := []int{i}
		at := i
		for j := depth - 1; j >= len(steps) && at >= 0; j-- {
			at = commentedParent(l, at, j, segs[j])
			marked = append(marked, at)
		}
		if at < 0 || !insideActive(l, steps, at) {
			continue
		}
		for _, m := range marked {
			l.text[m] = strings.TrimPrefix(l.text[m], "# ")
		}
		return l.bytes(), true
	}
	return nil, false
}

// commentedParent is the line of "# <indent><seg>:" above line at, within
// the same run of comment lines, or -1.
func commentedParent(l *lines, at, depth int, seg string) int {
	for k := at - 1; k >= 0 && strings.HasPrefix(l.text[k], "#"); k-- {
		ind, key, rest, ok := commentedKey(l.text[k])
		if !ok {
			continue
		}
		if ind == 2*depth && key == seg && strings.TrimSpace(rest) == "" {
			return k
		}
		if ind < 2*depth {
			return -1
		}
	}
	return -1
}

// insideActive reports whether line at lies in the block of the deepest
// active parent: below its key, with no active line between them at or
// left of the parent's indentation, and the parent a block mapping or
// empty.
func insideActive(l *lines, steps []step, at int) bool {
	if len(steps) == 0 {
		return true
	}
	p := steps[len(steps)-1]
	if p.flow || (p.value.Kind != yaml.MappingNode && !isEmptyValue(p.value)) {
		return false
	}
	ind := keyIndent(p.key)
	if at < p.key.Line {
		return false
	}
	for k := p.key.Line; k < at; k++ {
		t := l.text[k]
		if strings.TrimSpace(t) == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if len(t)-len(strings.TrimLeft(t, " ")) <= ind {
			return false
		}
	}
	return true
}

// removal is case 5. A template key goes back to its commented-out form, at
// its indentation; any other key's lines are deleted; the comments above
// it stay. A parent left with no active key is then commented too, or
// deleted when it is not a template group, and so on upward (0018-PLAN P3
// deviation 7).
func removal(file string, l *lines, steps []step, path string) ([]byte, error) {
	s := steps[len(steps)-1]
	if s.flow {
		return nil, unsafeEdit(file, path)
	}
	var text []string
	if k, ok := KeyOf(path); ok {
		text = []string{commentedForm(keyIndent(s.key), s.key.Value, templateValue(k))}
	}
	ops := []op{{from: s.key.Line - 1, to: lastLine(l, s.key, s.value), text: text}}
	segs := strings.Split(path, ".")
	for j := len(steps) - 2; j >= 0 && len(steps[j].value.Content) == 2; j-- {
		p := steps[j]
		var line []string
		if isGroup(strings.Join(segs[:j+1], ".")) {
			line = []string{commentedForm(keyIndent(p.key), p.key.Value, "")}
		}
		ops = append(ops, op{from: p.key.Line - 1, to: p.key.Line, text: line})
	}
	l.apply(ops)
	return l.bytes(), nil
}

// commentedForm is the template's form of a key line: "# ", the key's
// indentation, the key, a colon and its value when it has one.
func commentedForm(indent int, key, value string) string {
	s := "# " + strings.Repeat(" ", indent) + key + ":"
	if value != "" {
		s += " " + value
	}
	return s
}

// templateValue is the value k's line shows in the template: its default,
// or "" for an empty string.
func templateValue(k Key) string {
	if k.Default == "" {
		return `""`
	}
	return k.Default
}

func keyIndent(key *yaml.Node) int { return key.Column - 1 }

// lastLine is the 1-based last line of a key's value: the largest line
// among the value and its descendants, past any scalar that runs over more
// than one line. A nil key is the document's root.
func lastLine(l *lines, key, v *yaml.Node) int {
	last := v.Line
	if key != nil {
		last = max(last, key.Line)
	}
	switch v.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(v.Content); i += 2 {
			last = max(last, lastLine(l, v.Content[i], v.Content[i+1]))
		}
	case yaml.SequenceNode:
		for _, c := range v.Content {
			last = max(last, lastLine(l, &yaml.Node{Line: c.Line, Column: v.Column}, c))
		}
	case yaml.ScalarNode:
		indent := 0
		if key != nil {
			indent = keyIndent(key)
		}
		last = max(last, scalarLastLine(l, v, indent))
	}
	return last
}

// scalarLastLine is the last line of a scalar whose key is at indent: a
// block scalar's last content line, a quoted scalar's closing quote, or a
// plain scalar's last continuation line.
func scalarLastLine(l *lines, v *yaml.Node, indent int) int {
	i := v.Line - 1
	line := l.text[i]
	pos := skipProperties(line, byteOffset(line, v.Column))
	block := v.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0
	if !block {
		if end, ok := scalarEnd(line, pos, v.Style); ok && (v.Style&(yaml.SingleQuotedStyle|yaml.DoubleQuotedStyle) != 0 || line[pos:end] == v.Value) {
			return v.Line
		}
	}
	last := v.Line
	for k := i + 1; k < len(l.text); k++ {
		t := l.text[k]
		if strings.TrimSpace(t) == "" {
			continue
		}
		if len(t)-len(strings.TrimLeft(t, " ")) <= indent || (!block && strings.HasPrefix(strings.TrimSpace(t), "#")) {
			break
		}
		last = k + 1
	}
	return last
}

// byteOffset is the byte offset of yaml.v3's 1-based column, which counts
// runes.
func byteOffset(line string, column int) int {
	off := 0
	for range column - 1 {
		if off >= len(line) {
			break
		}
		_, n := utf8.DecodeRuneInString(line[off:])
		off += n
	}
	return off
}

// yaml11Words are the plain words a YAML 1.1 reader takes for a boolean;
// yaml.v3 tags them as strings but decodes them into a bool (0018-PLAN P2's
// probe), so a string with one of these values is quoted.
var yaml11Words = strings.Fields("y n yes no on off")

// render writes a scalar plain when yaml.v3 reads the plain text back as
// the same tag and value, and double-quoted otherwise (D8): a string 1.10
// is written "1.10".
func render(v *yaml.Node) (string, error) {
	if v.Kind != yaml.ScalarNode {
		return "", errors.New("config: render: not a scalar")
	}
	if plainReadsBack(v.Value, v.ShortTag()) {
		return v.Value, nil
	}
	return strconv.Quote(v.Value), nil
}

func plainReadsBack(text, tag string) bool {
	if text == "" || strings.ContainsAny(text, "\r\n") ||
		(tag == "!!str" && slices.Contains(yaml11Words, strings.ToLower(text))) {
		return false
	}
	var doc yaml.Node
	if yaml.Unmarshal([]byte("k: "+text+"\n"), &doc) != nil || len(doc.Content) == 0 {
		return false
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode || len(m.Content) != 2 {
		return false
	}
	n := m.Content[1]
	return n.Kind == yaml.ScalarNode && n.Style == 0 && n.ShortTag() == tag && n.Value == text
}

// readsBack is D8's check: out parses as one mapping, e's path reads back
// as its value (or as absent), and every other path reads as it did.
func readsBack(file string, old *yaml.Node, out []byte, e Edit) bool {
	root, _ := Parse(file, out)
	if root == nil {
		return false
	}
	before, after := flatten(old), flatten(root)
	under := func(p string) bool {
		return p == e.Path || strings.HasPrefix(p, e.Path+".") || strings.HasPrefix(p, e.Path+"[")
	}
	want := map[string]string{}
	if e.Value != nil {
		flattenInto(want, e.Path, e.Value)
	}
	for p, v := range after {
		if under(p) {
			if want[p] != v {
				return false
			}
		} else if before[p] != v {
			return false
		}
	}
	// An ancestor written "key:" with nothing after it becomes the mapping
	// that holds the path (0018-PLAN P3 deviation 8); an explicit ~ or null
	// does not.
	emptied := func(p string) bool {
		if e.Value == nil || !strings.HasPrefix(e.Path, p+".") {
			return false
		}
		steps, err := lookup(old, p)
		return err == nil && len(steps) == strings.Count(p, ".")+1 && isEmptyValue(steps[len(steps)-1].value)
	}
	for p := range before {
		if _, ok := after[p]; !ok && !under(p) && !emptied(p) {
			return false
		}
	}
	for p := range want {
		if _, ok := after[p]; !ok {
			return false
		}
	}
	return true
}

// flatten is every scalar of a document by path, as its tag and value; an
// empty mapping or sequence is a value of its own.
func flatten(root *yaml.Node) map[string]string {
	out := map[string]string{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		flattenInto(out, root.Content[i].Value, root.Content[i+1])
	}
	return out
}

func flattenInto(out map[string]string, path string, n *yaml.Node) {
	n = resolveAlias(n)
	switch n.Kind {
	case yaml.MappingNode:
		if len(n.Content) == 0 {
			out[path] = "{}"
		}
		for i := 0; i+1 < len(n.Content); i += 2 {
			flattenInto(out, path+"."+n.Content[i].Value, n.Content[i+1])
		}
	case yaml.SequenceNode:
		if len(n.Content) == 0 {
			out[path] = "[]"
		}
		for i, c := range n.Content {
			flattenInto(out, path+"["+strconv.Itoa(i)+"]", c)
		}
	default:
		out[path] = n.ShortTag() + "\x00" + n.Value
	}
}
