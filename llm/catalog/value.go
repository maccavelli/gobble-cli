package catalog

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// envName is an environment variable's name, as Pi's grammar reads one.
var (
	envName       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	envNamePrefix = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*`)
)

// ResolveValue is a models.json value, by Pi's grammar
// (resolve-config-value.ts): "!command" is run's trimmed output; otherwise
// $NAME and ${NAME} are variables lookup finds, and $$ and $! are a literal
// $ and !. A missing variable, or a command that fails or prints nothing, is
// an error naming them, and no error holds a resolved value. It is not
// cached (0005-PLAN F4-2b).
func ResolveValue(v string, lookup func(string) (string, bool), run func(string) (string, error)) (string, error) {
	if command, ok := strings.CutPrefix(v, "!"); ok {
		out, err := run(command)
		if err != nil {
			return "", fmt.Errorf("the command %q failed: %w", command, err)
		}
		if out = strings.TrimSpace(out); out == "" {
			return "", fmt.Errorf("the command %q printed nothing", command)
		}
		return out, nil
	}
	var b strings.Builder
	var missing []string
	for i := 0; i < len(v); {
		j := strings.IndexByte(v[i:], '$')
		if j < 0 {
			b.WriteString(v[i:])
			break
		}
		b.WriteString(v[i : i+j])
		i += j
		rest := v[i+1:]
		switch {
		case strings.HasPrefix(rest, "$"), strings.HasPrefix(rest, "!"):
			b.WriteByte(rest[0])
			i += 2
		case strings.HasPrefix(rest, "{"):
			end := strings.IndexByte(rest, '}')
			if end < 0 {
				b.WriteByte('$')
				i++
				break
			}
			if name := rest[1:end]; envName.MatchString(name) {
				missing = appendValue(&b, lookup, name, missing)
			} else {
				b.WriteString(v[i : i+end+2])
			}
			i += end + 2
		default:
			if name := envNamePrefix.FindString(rest); name != "" {
				missing = appendValue(&b, lookup, name, missing)
				i += 1 + len(name)
				break
			}
			b.WriteByte('$')
			i++
		}
	}
	switch len(missing) {
	case 0:
		return b.String(), nil
	case 1:
		return "", fmt.Errorf("the environment variable %s is not set", missing[0])
	}
	return "", fmt.Errorf("the environment variables %s are not set", strings.Join(missing, ", "))
}

// QuoteValue is the value Pi's grammar gives back as s: each $ written $$,
// and a leading ! written $!, so ResolveValue returns s with no lookup and
// no command (0005-PLAN F4-3b).
func QuoteValue(s string) string {
	q := strings.ReplaceAll(s, "$", "$$")
	if strings.HasPrefix(q, "!") {
		q = "$" + q
	}
	return q
}

// appendValue writes variable name's value to b, or adds name to missing,
// once, when lookup has none; Pi treats an empty value as missing.
func appendValue(b *strings.Builder, lookup func(string) (string, bool), name string, missing []string) []string {
	if v, ok := lookup(name); ok && v != "" {
		b.WriteString(v)
		return missing
	}
	if !slices.Contains(missing, name) {
		missing = append(missing, name)
	}
	return missing
}
