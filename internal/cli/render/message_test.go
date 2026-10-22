package render

import "testing"

// A message of gobble's own wraps at spaces, its continuation rows
// indented two columns past its own indentation; a word wider than its row
// is left whole; a line that fits is left exactly as it is (0028-MADR D6).
func TestMessage(t *testing.T) {
	for _, tc := range []struct {
		name, in string
		width    int
		want     string
	}{
		{"broken at spaces", "Warning: no model credential: set one of GEMINI_API_KEY, OPENAI_API_KEY", 40,
			"Warning: no model credential: set one of\n  GEMINI_API_KEY, OPENAI_API_KEY"},
		{"a long word whole", "Error: session/prompt: llmprovider:kilo-failed-after-4-attempts-with-http-500 retry", 30,
			"Error: session/prompt:\n  llmprovider:kilo-failed-after-4-attempts-with-http-500\n  retry"},
		{"each line on its own", "first line that is long enough to wrap\n  an indented hint that also needs a wrap", 24,
			"first line that is long\n  enough to wrap\n  an indented hint that\n    also needs a wrap"},
		{"a line that fits is kept, spaces and all", "a  b   c", 40, "a  b   c"},
		{"wide characters take two columns", "注意 漢字漢字漢字 漢字漢字漢字", 17, "注意 漢字漢字漢字\n  漢字漢字漢字"},
		{"escape sequences take no columns", "\x1b[2mdim words here\x1b[0m and more", 16,
			"\x1b[2mdim words here\x1b[0m\n  and more"},
		{"no width, no wrapping", "Warning: no model credential: set one of GEMINI_API_KEY", 0,
			"Warning: no model credential: set one of GEMINI_API_KEY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Message(tc.in, tc.width); got != tc.want {
				t.Errorf("got\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}
