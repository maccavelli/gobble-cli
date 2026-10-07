package command

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in, name, args string
		ok             bool
	}{
		{"/compact", "compact", "", true},
		{"  /compact keep the API notes \n", "compact", "keep the API notes", true},
		{"/name\tmy session", "name", "my session", true},
		{"/deep-research_2 x", "deep-research_2", "x", true},
		{"/usr/local/bin is wrong", "", "", false},
		{"/skill:review", "", "", false},
		{"/", "", "", false},
		{"/-x", "", "", false},
		{"compact", "", "", false},
		{"please /compact", "", "", false},
	}
	for _, c := range cases {
		name, args, ok := Parse(c.in)
		if name != c.name || args != c.args || ok != c.ok {
			t.Errorf("Parse(%q) = %q, %q, %v; want %q, %q, %v", c.in, name, args, ok, c.name, c.args, c.ok)
		}
	}
}
