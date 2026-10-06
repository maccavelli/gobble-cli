package mcpclient

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// DefaultTimeout is a server's per-request timeout when its entry sets none,
// as in Pi.
const DefaultTimeout = 60 * time.Second

// exposures are Pi's exposure values, in Pi's order (mcp-servers.ts).
var exposures = []string{"codemode", "codemode-deferred", "deferred", "direct", "hidden"}

var serverName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Pair is one name and value of an env or headers map, kept in file order.
type Pair struct {
	Name, Value string
}

// Server is one valid server: an mcp.json entry, or one a session supplied.
// A stdio server has Command; an HTTP server has URL.
type Server struct {
	Name    string
	Command string
	Args    []string
	Env     []Pair
	Cwd     string
	URL     string
	Headers []Pair
	Timeout time.Duration
	Enabled bool
	// Exposure, ToolExposure and OAuth are checked and kept; they are
	// applied from 0005-PLAN F8.
	Exposure     string
	ToolExposure []Pair
	OAuth        bool
	// Literal servers use Env and Headers as given, without $NAME
	// interpolation: the values of an ACP client, which resolved them.
	Literal bool
}

// IsHTTP reports whether the server is reached over streamable HTTP.
func (s Server) IsHTTP() bool { return s.URL != "" }

// Transport is how the server is reached: its URL, or its command line.
func (s Server) Transport() string {
	if s.IsHTTP() {
		return s.URL
	}
	return strings.Join(append([]string{s.Command}, s.Args...), " ")
}

// Config is an mcp.json file's valid servers, in file order, and the
// errors of its invalid entries, each "<path>: <message>".
type Config struct {
	Path    string
	Servers []Server
	Errors  []string
}

// errShape is Pi's message for a file that is not an object with an
// mcpServers object.
var errShape = errors.New(`expected an object with an "mcpServers" object`)

// Load reads an mcp.json as Pi reads one (config.ts readConfigFile): a
// missing file has no servers, an unreadable one is a single error, and an
// invalid entry is skipped with an error while the others load.
func Load(path string) Config {
	c := Config{Path: path}
	b, err := os.ReadFile(path) //nolint:gosec // G304: the user's own mcp.json
	if errors.Is(err, fs.ErrNotExist) {
		return c
	}
	if err != nil {
		c.Errors = append(c.Errors, fmt.Sprintf("%s: %v", path, err))
		return c
	}
	top, servers, err := parseFile(b)
	if err != nil {
		c.Errors = append(c.Errors, fmt.Sprintf("%s: %v", path, err))
		return c
	}
	if v, ok := find(top, "autoEnableCodemode"); ok && v.Kind() != 't' && v.Kind() != 'f' {
		c.Errors = append(c.Errors, path+": autoEnableCodemode must be a boolean")
	}
	for _, m := range servers {
		s, err := Validate(m.name, m.value)
		if err != nil {
			c.Errors = append(c.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		c.Servers = append(c.Servers, s)
	}
	return c
}

// parseFile splits an mcp.json into its top-level members and the members
// of its mcpServers object, which may be absent.
func parseFile(b []byte) (top, servers []member, err error) {
	top, err = members(b)
	if errors.Is(err, errNotObject) {
		return nil, nil, errShape
	}
	if err != nil {
		return nil, nil, err
	}
	v, ok := find(top, "mcpServers")
	if !ok {
		return top, nil, nil
	}
	if v.Kind() != '{' {
		return nil, nil, errShape
	}
	servers, err = members(v)
	return top, servers, err
}

// Validate checks one entry as Pi checks it (mcp-servers.ts
// validateMcpServerConfig), with Pi's messages.
func Validate(name string, raw jsontext.Value) (Server, error) {
	if err := validName(name); err != nil {
		return Server{}, err
	}
	fields, err := members(raw)
	if err != nil {
		return Server{}, fmt.Errorf("server %q must be an object", name)
	}
	e := entry{name: name, fields: fields}
	s := Server{Name: name, Timeout: DefaultTimeout, Enabled: true}
	if err := e.common(&s); err != nil {
		return Server{}, err
	}
	typ, typed := e.str("type")
	_, hasType := find(fields, "type")
	if typed && typ == "sse" {
		return Server{}, e.errorf("legacy SSE transport is not supported; use the streamable HTTP URL")
	}
	untyped := !hasType
	if u, ok := e.str("url"); ok && (untyped || typed && (typ == "http" || typ == "streamable-http")) {
		return e.http(s, u)
	}
	if cmd, ok := e.str("command"); ok && (untyped || typed && typ == "stdio") {
		return e.stdio(s, cmd)
	}
	return Server{}, fmt.Errorf(`server %q needs either "command" (stdio) or "url" (streamable HTTP)`, name)
}

// SessionStdio is a stdio server a session supplied, such as an ACP
// client's: Pi's name rule applies, and its values are used as given.
func SessionStdio(name, command string, args []string, env []Pair) (Server, error) {
	if err := validName(name); err != nil {
		return Server{}, err
	}
	return Server{Name: name, Command: command, Args: args, Env: env, Timeout: DefaultTimeout, Enabled: true, Literal: true}, nil
}

// SessionHTTP is a streamable HTTP server a session supplied: Pi's name
// and URL rules apply, and its headers are used as given.
func SessionHTTP(name, rawURL string, headers []Pair) (Server, error) {
	if err := validName(name); err != nil {
		return Server{}, err
	}
	if !httpURL(rawURL) {
		return Server{}, fmt.Errorf("server %q: url must be an http or https URL", name)
	}
	return Server{Name: name, URL: rawURL, Headers: headers, Timeout: DefaultTimeout, Enabled: true, Literal: true}, nil
}

func validName(name string) error {
	if !serverName.MatchString(name) {
		return fmt.Errorf(`invalid server name %q (use letters, digits, "_" and "-")`, name)
	}
	return nil
}

// entry is one server's members, read with Pi's type checks.
type entry struct {
	name   string
	fields []member
}

func (e entry) errorf(format string, args ...any) error {
	return fmt.Errorf("server %q: %s", e.name, fmt.Sprintf(format, args...))
}

func (e entry) value(key string) (jsontext.Value, bool) { return find(e.fields, key) }

// str is a member's string value; ok is false when it is absent or not a
// string.
func (e entry) str(key string) (string, bool) {
	v, ok := e.value(key)
	if !ok || v.Kind() != '"' {
		return "", false
	}
	var s string
	if json.Unmarshal(v, &s) != nil {
		return "", false
	}
	return s, true
}

// pairs is a member's string map, in order; ok is false when it is not an
// object of strings.
func (e entry) pairs(key string) ([]Pair, bool) {
	v, _ := e.value(key)
	ms, err := members(v)
	if err != nil {
		return nil, false
	}
	out := make([]Pair, 0, len(ms))
	for _, m := range ms {
		var s string
		if m.value.Kind() != '"' || json.Unmarshal(m.value, &s) != nil {
			return nil, false
		}
		out = append(out, Pair{Name: m.name, Value: s})
	}
	return out, true
}

// common checks the members every server may carry: exposure,
// toolExposure, enabled and timeout.
func (e entry) common(s *Server) error {
	quoted := make([]string, len(exposures))
	for i, x := range exposures {
		quoted[i] = `"` + x + `"`
	}
	choices := strings.Join(quoted, ", ")
	if _, ok := e.value("exposure"); ok {
		x, isStr := e.str("exposure")
		if !isStr || !slices.Contains(exposures, x) {
			return e.errorf("exposure must be one of %s", choices)
		}
		s.Exposure = x
	}
	if v, ok := e.value("toolExposure"); ok {
		ms, err := members(v)
		if err != nil {
			return e.errorf("toolExposure must map tool names to exposures")
		}
		for _, m := range ms {
			var x string
			if m.value.Kind() != '"' || json.Unmarshal(m.value, &x) != nil || !slices.Contains(exposures, x) {
				return e.errorf("toolExposure %q must be one of %s", m.name, choices)
			}
			s.ToolExposure = append(s.ToolExposure, Pair{Name: m.name, Value: x})
		}
	}
	if v, ok := e.value("enabled"); ok {
		switch v.Kind() {
		case 't':
		case 'f':
			s.Enabled = false
		default:
			return e.errorf("enabled must be a boolean")
		}
	}
	if v, ok := e.value("timeout"); ok {
		var secs float64
		if v.Kind() != '0' || json.Unmarshal(v, &secs) != nil || !(secs > 0) || math.IsInf(secs, 0) {
			return e.errorf("timeout must be a positive number of seconds")
		}
		s.Timeout = time.Duration(secs * float64(time.Second))
	}
	return nil
}

func (e entry) http(s Server, raw string) (Server, error) {
	if !httpURL(raw) {
		return Server{}, e.errorf("url must be an http or https URL")
	}
	s.URL = raw
	if _, ok := e.value("headers"); ok {
		h, ok := e.pairs("headers")
		if !ok {
			return Server{}, e.errorf("headers must map names to strings")
		}
		s.Headers = h
	}
	if v, ok := e.value("oauth"); ok {
		if err := validateOAuth(v); err != nil {
			return Server{}, e.errorf("%v", err)
		}
		s.OAuth = true
	}
	return s, nil
}

func (e entry) stdio(s Server, cmd string) (Server, error) {
	s.Command = cmd
	if v, ok := e.value("args"); ok {
		var args []string
		if v.Kind() != '[' || json.Unmarshal(v, &args) != nil {
			return Server{}, e.errorf("args must be an array of strings")
		}
		s.Args = args
	}
	if _, ok := e.value("env"); ok {
		env, ok := e.pairs("env")
		if !ok {
			return Server{}, e.errorf("env must map names to strings")
		}
		s.Env = env
	}
	if _, ok := e.value("cwd"); ok {
		cwd, ok := e.str("cwd")
		if !ok {
			return Server{}, e.errorf("cwd must be a string")
		}
		s.Cwd = cwd
	}
	return s, nil
}

// httpURL reports whether raw parses as an http or https URL with a host.
func httpURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return (scheme == "http" || scheme == "https") && u.Host != ""
}

// validateOAuth is Pi's validateOAuth (mcp-servers.ts), message for
// message.
func validateOAuth(v jsontext.Value) error {
	fields, err := members(v)
	if err != nil {
		return errors.New("oauth must be an object")
	}
	e := entry{fields: fields}
	for _, key := range []string{"clientId", "clientSecret"} {
		if _, ok := e.value(key); ok {
			if _, isStr := e.str(key); !isStr {
				return fmt.Errorf("oauth.%s must be a string", key)
			}
		}
	}
	port, hasPort := 0.0, false
	if pv, ok := e.value("callbackPort"); ok {
		if pv.Kind() != '0' || json.Unmarshal(pv, &port) != nil || port != math.Trunc(port) || port < 1 || port > 65535 {
			return errors.New("oauth.callbackPort must be a port number")
		}
		hasPort = true
	}
	if _, ok := e.value("callbackUrl"); ok {
		cb, isStr := e.str("callbackUrl")
		u, parsed := loopbackRedirect(cb)
		if !isStr || !parsed {
			return errors.New("oauth.callbackUrl must be an http URI on localhost, 127.0.0.1, or [::1] without query or fragment")
		}
		if p := u.Port(); p != "" && hasPort && p != strconv.Itoa(int(port)) {
			return errors.New("oauth.callbackUrl and oauth.callbackPort name different ports")
		}
	}
	if _, ok := e.value("scope"); ok {
		if _, isStr := e.str("scope"); !isStr {
			return errors.New("oauth.scope must be a string")
		}
	}
	return nil
}

// loopbackRedirect is Pi's isLoopbackRedirectUri: an http URI on
// localhost, 127.0.0.1 or [::1], with no query or fragment.
func loopbackRedirect(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "http") {
		return nil, false
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1":
	default:
		return nil, false
	}
	return u, u.RawQuery == "" && u.Fragment == ""
}
