package personality

import (
	"bufio"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Catalog is one personality's messages: each key's variants, in file
// order.
type Catalog struct {
	variants map[Key][]string
}

// Variants are key's variants, or nil when the catalog leaves it out.
func (c *Catalog) Variants(key Key) []string {
	if c == nil {
		return nil
	}
	return c.variants[key]
}

// Keys are the catalog's keys, sorted.
func (c *Catalog) Keys() []Key {
	out := make([]Key, 0, len(c.variants))
	for k := range c.variants {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// ParseCatalog reads a catalog file (the package documentation's shape). A
// key given twice, a key with no variant, or a variant before any key is
// an error naming its line.
func ParseCatalog(data []byte) (*Catalog, error) {
	c := &Catalog{variants: map[Key][]string{}}
	var (
		key       Key
		para      []string // the paragraph or bullet being read
		inBullet  bool
		inHeading bool // a key's heading is open, no variant yet
		line      int
		headingAt int
	)
	flush := func() {
		if len(para) > 0 {
			c.variants[key] = append(c.variants[key], strings.Join(para, " "))
			inHeading = false
		}
		para, inBullet = nil, false
	}
	body, err := withoutFrontMatter(data)
	if err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		line++
		raw := sc.Text()
		text := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(text, "## "):
			flush()
			if inHeading {
				return nil, fmt.Errorf("personality: line %d: key %s has no variant", headingAt, key)
			}
			key = Key(strings.TrimSpace(strings.TrimPrefix(text, "## ")))
			if _, dup := c.variants[key]; dup {
				return nil, fmt.Errorf("personality: line %d: key %s is given twice", line, key)
			}
			c.variants[key] = nil
			inHeading, headingAt = true, line
		case strings.HasPrefix(text, "#"):
			flush() // any other heading ends the key's section: what follows is documentation
			if inHeading {
				return nil, fmt.Errorf("personality: line %d: key %s has no variant", headingAt, key)
			}
			key = ""
		case text == "":
			flush()
		case key == "":
			// documentation before the first key, or after a lower heading
		case strings.HasPrefix(text, "- "):
			flush()
			para, inBullet = []string{strings.TrimSpace(strings.TrimPrefix(text, "- "))}, true
		case inBullet && raw != text:
			para = append(para, text) // an indented continuation of the bullet
		default:
			if inBullet {
				flush()
			}
			para = append(para, text)
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("personality: %w", err)
	}
	if inHeading {
		return nil, fmt.Errorf("personality: line %d: key %s has no variant", headingAt, key)
	}
	for k, vs := range c.variants {
		if len(vs) == 0 {
			delete(c.variants, k)
		}
	}
	return c, nil
}

// withoutFrontMatter is data with a leading "---" block removed; a block
// that never closes is an error.
func withoutFrontMatter(data []byte) (string, error) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return s, nil
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		if strings.HasSuffix(s, "\n---") {
			return "", nil
		}
		return "", errors.New("personality: the front matter does not close with ---")
	}
	return s[4+end+5:], nil
}

// Placeholders are the names in text's braces, in order, each once.
func Placeholders(text string) []string {
	var out []string
	for i := 0; i < len(text); i++ {
		switch {
		case strings.HasPrefix(text[i:], "{{"), strings.HasPrefix(text[i:], "}}"):
			i++
		case text[i] == '{':
			end := strings.IndexByte(text[i:], '}')
			if end < 0 {
				return out
			}
			if name := text[i+1 : i+end]; !slices.Contains(out, name) {
				out = append(out, name)
			}
			i += end
		}
	}
	return out
}

// CheckPlaceholders is the facts rule (0022-MADR D5): variant carries
// exactly the placeholders of boring's text, so a joke can add words but
// never drop the number, the name or the error.
func CheckPlaceholders(boring, variant string) error {
	want, got := Placeholders(boring), Placeholders(variant)
	for _, n := range want {
		if !slices.Contains(got, n) {
			return fmt.Errorf("personality: %q leaves out {%s}", variant, n)
		}
	}
	for _, n := range got {
		if !slices.Contains(want, n) {
			return fmt.Errorf("personality: %q has {%s}, which the boring text does not", variant, n)
		}
	}
	return nil
}

// fill substitutes args into text's placeholders. A placeholder with no
// argument is left as it is, so a missing value shows rather than
// vanishing.
func fill(text string, args Args) string {
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		switch {
		case strings.HasPrefix(text[i:], "{{"):
			b.WriteByte('{')
			i++
		case strings.HasPrefix(text[i:], "}}"):
			b.WriteByte('}')
			i++
		case text[i] == '{':
			end := strings.IndexByte(text[i:], '}')
			if end < 0 {
				b.WriteString(text[i:])
				return b.String()
			}
			name := text[i+1 : i+end]
			if v, ok := args[name]; ok {
				b.WriteString(v)
			} else {
				b.WriteString(text[i : i+end+1])
			}
			i += end
		default:
			b.WriteByte(text[i])
		}
	}
	return b.String()
}
