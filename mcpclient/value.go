package mcpclient

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var (
	envName       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	envNamePrefix = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*`)
)

// Resolve resolves an env or header value as Pi resolves configuration
// values (resolve-config-value.ts): $NAME and ${NAME} name environment
// variables, $$ and $! are a literal $ and !, and the rest is literal. An
// unset or empty variable is an error naming it. A value that starts with !
// is a command, which arrives with 0005-PLAN F8. what names the value in an
// error, such as `MCP server "web" header "Authorization"`.
func Resolve(value, what string, getenv func(string) string) (string, error) {
	if strings.HasPrefix(value, "!") {
		return "", fmt.Errorf("%s: command values (!…) arrive with 0005-PLAN F8", what)
	}
	var (
		b       strings.Builder
		missing []string
	)
	lookup := func(name string) {
		v := getenv(name)
		if v == "" && !slices.Contains(missing, name) {
			missing = append(missing, name)
		}
		b.WriteString(v)
	}
	for i := 0; i < len(value); {
		j := strings.IndexByte(value[i:], '$')
		if j < 0 {
			b.WriteString(value[i:])
			break
		}
		j += i
		b.WriteString(value[i:j])
		rest := value[j+1:]
		switch {
		case strings.HasPrefix(rest, "$"), strings.HasPrefix(rest, "!"):
			b.WriteByte(rest[0])
			i = j + 2
		case strings.HasPrefix(rest, "{"):
			end := strings.IndexByte(rest[1:], '}')
			if end < 0 {
				b.WriteByte('$')
				i = j + 1
				continue
			}
			if name := rest[1 : 1+end]; envName.MatchString(name) {
				lookup(name)
			} else {
				b.WriteString(value[j : j+2+end+1])
			}
			i = j + 1 + 1 + end + 1
		default:
			if name := envNamePrefix.FindString(rest); name != "" {
				lookup(name)
				i = j + 1 + len(name)
				continue
			}
			b.WriteByte('$')
			i = j + 1
		}
	}
	switch len(missing) {
	case 0:
		return b.String(), nil
	case 1:
		//lint:ignore ST1005 Pi's message, word for word (0002-PLAN Phase 5, deviation 3)
		return "", fmt.Errorf("Failed to resolve %s from environment variable: %s", what, missing[0]) //nolint:staticcheck // as the lint:ignore above
	default:
		//lint:ignore ST1005 Pi's message, word for word (0002-PLAN Phase 5, deviation 3)
		return "", fmt.Errorf("Failed to resolve %s from environment variables: %s", what, strings.Join(missing, ", ")) //nolint:staticcheck // as the lint:ignore above
	}
}
