package config

import (
	"testing"
)

// --set parses PATH=VALUE: the value is YAML, checked as a file value is.
func TestParseOverride(t *testing.T) {
	good := []struct {
		arg  string
		want any
	}{
		{"steeringMode=all", "all"},
		{"mcp.stopGraceMs=250", int64(250)},
		{"tools.bash.useClientTerminal=true", true},
		{"shellCommandPrefix=set -e; export A=b", "set -e; export A=b"},
		{"shellPath=", ""},
		{`externalEditor="code --wait"`, "code --wait"},
		{"defaultThinkingLevel='low'", "low"},
	}
	for _, g := range good {
		o, err := ParseOverride(g.arg)
		if err != nil || o.value != g.want {
			t.Errorf("ParseOverride(%q) = %+v, %v; want %v", g.arg, o, err, g.want)
		}
	}
	bad := map[string]string{
		"steeringMode":                     "--set steeringMode: want PATH=VALUE",
		"=all":                             "--set =all: want PATH=VALUE",
		"theme=dark":                       `unknown setting "theme"`,
		"apiKey=x":                         "--set apiKey: credentials go in auth.json, not settings.yaml",
		"steeringMode=some":                "--set steeringMode: must be one of one-at-a-time, all",
		"mcp.stopGraceMs=soon":             "--set mcp.stopGraceMs: want an integer, have a string",
		"mcp.stopGraceMs=":                 "--set mcp.stopGraceMs: want an integer, have an empty value",
		"tools.bash.maxTimeoutMs=0":        "--set tools.bash.maxTimeoutMs: must be at least 1",
		"tools.bash.useClientTerminal=[a]": "--set tools.bash.useClientTerminal: want a boolean, have a sequence",
	}
	for arg, want := range bad {
		if _, err := ParseOverride(arg); err == nil || err.Error() != want {
			t.Errorf("ParseOverride(%q) = %v, want %q", arg, err, want)
		}
	}
	if o, err := ParseOverride("steeringMode=~"); err != nil || !o.null {
		t.Errorf("a null value: %+v %v", o, err)
	}
}
