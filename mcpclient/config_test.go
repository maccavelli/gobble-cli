package mcpclient

import (
	"encoding/json/jsontext"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// Validate gives Pi's checks with Pi's messages (mcp-servers.ts).
func TestValidateMessages(t *testing.T) {
	cases := []struct {
		name, raw, want string
	}{
		{"bad name", `{"command":"x"}`, `invalid server name "bad name" (use letters, digits, "_" and "-")`},
		{"s", `[]`, `server "s" must be an object`},
		{"s", `{"command":"x","exposure":"loud"}`, `server "s": exposure must be one of "codemode", "codemode-deferred", "deferred", "direct", "hidden"`},
		{"s", `{"command":"x","toolExposure":[]}`, `server "s": toolExposure must map tool names to exposures`},
		{"s", `{"command":"x","toolExposure":{"a":"direct","b":"nope"}}`, `server "s": toolExposure "b" must be one of "codemode", "codemode-deferred", "deferred", "direct", "hidden"`},
		{"s", `{"command":"x","enabled":"yes"}`, `server "s": enabled must be a boolean`},
		{"s", `{"command":"x","timeout":0}`, `server "s": timeout must be a positive number of seconds`},
		{"s", `{"command":"x","timeout":"5"}`, `server "s": timeout must be a positive number of seconds`},
		{"old", `{"type":"sse","url":"https://example.com/sse"}`, `server "old": legacy SSE transport is not supported; use the streamable HTTP URL`},
		{"s", `{"url":"ftp://example.com"}`, `server "s": url must be an http or https URL`},
		{"s", `{"url":"https://example.com","headers":{"A":1}}`, `server "s": headers must map names to strings`},
		{"s", `{"url":"https://example.com","oauth":[]}`, `server "s": oauth must be an object`},
		{"s", `{"url":"https://example.com","oauth":{"callbackPort":70000}}`, `server "s": oauth.callbackPort must be a port number`},
		{"s", `{"url":"https://example.com","oauth":{"callbackUrl":"https://localhost/cb"}}`, `server "s": oauth.callbackUrl must be an http URI on localhost, 127.0.0.1, or [::1] without query or fragment`},
		{"s", `{"url":"https://example.com","oauth":{"callbackUrl":"http://localhost:8080/cb","callbackPort":8765}}`, `server "s": oauth.callbackUrl and oauth.callbackPort name different ports`},
		{"s", `{"command":"x","args":"-y"}`, `server "s": args must be an array of strings`},
		{"s", `{"command":"x","env":{"A":true}}`, `server "s": env must map names to strings`},
		{"s", `{"command":"x","cwd":3}`, `server "s": cwd must be a string`},
		{"s", `{"type":"stdio","url":"https://example.com"}`, `server "s" needs either "command" (stdio) or "url" (streamable HTTP)`},
		{"s", `{"type":null,"command":"x"}`, `server "s" needs either "command" (stdio) or "url" (streamable HTTP)`},
		{"s", `{}`, `server "s" needs either "command" (stdio) or "url" (streamable HTTP)`},
	}
	for _, c := range cases {
		_, err := Validate(c.name, jsontext.Value(c.raw))
		if err == nil || err.Error() != c.want {
			t.Errorf("Validate(%q, %s) = %v, want %q", c.name, c.raw, err, c.want)
		}
	}
}

func TestValidateServers(t *testing.T) {
	s, err := Validate("fs", jsontext.Value(`{"command":"npx","args":["-y","srv"],"env":{"B":"2","A":"1"},"cwd":"sub","timeout":1.5,"exposure":"direct"}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.IsHTTP() || s.Command != "npx" || !slices.Equal(s.Args, []string{"-y", "srv"}) || s.Cwd != "sub" ||
		s.Timeout != 1500*time.Millisecond || !s.Enabled || s.Exposure != "direct" || s.Transport() != "npx -y srv" {
		t.Fatalf("stdio server %+v", s)
	}
	if !slices.Equal(s.Env, []Pair{{"B", "2"}, {"A", "1"}}) {
		t.Fatalf("env %v, want file order", s.Env)
	}
	h, err := Validate("web", jsontext.Value(`{"type":"streamable-http","url":"https://example.com/mcp","headers":{"X":"y"},"enabled":false,"oauth":{"clientId":"c","callbackUrl":"http://[::1]/cb"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !h.IsHTTP() || h.Enabled || !h.OAuth || h.Timeout != DefaultTimeout || h.Transport() != "https://example.com/mcp" {
		t.Fatalf("http server %+v", h)
	}
}

func writeConfig(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// Load skips invalid entries with an error each, as Pi does, and keeps the
// others in file order.
func TestLoad(t *testing.T) {
	p := writeConfig(t, `{"autoEnableCodemode": 1, "mcpServers": {
		"b": {"command": "b"},
		"old": {"type": "sse", "url": "https://example.com/sse"},
		"a": {"url": "https://example.com/mcp"}
	}}`)
	c := Load(p)
	if len(c.Servers) != 2 || c.Servers[0].Name != "b" || c.Servers[1].Name != "a" {
		t.Fatalf("servers %+v", c.Servers)
	}
	want := []string{
		p + ": autoEnableCodemode must be a boolean",
		p + `: server "old": legacy SSE transport is not supported; use the streamable HTTP URL`,
	}
	if !slices.Equal(c.Errors, want) {
		t.Fatalf("errors\n%q\nwant\n%q", c.Errors, want)
	}
	for _, text := range []string{`[]`, `{"mcpServers": []}`} {
		if c := Load(writeConfig(t, text)); len(c.Errors) != 1 || c.Errors[0] != c.Path+`: expected an object with an "mcpServers" object` {
			t.Errorf("%s: errors %q", text, c.Errors)
		}
	}
	if c := Load(writeConfig(t, `{"mcpServers": {`)); len(c.Errors) != 1 || len(c.Servers) != 0 {
		t.Errorf("malformed: %+v", c)
	}
	if c := Load(filepath.Join(t.TempDir(), "none.json")); len(c.Errors) != 0 || len(c.Servers) != 0 {
		t.Errorf("missing file: %+v", c)
	}
}
