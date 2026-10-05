package term

import (
	"os"
	"testing"
)

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestColorOK(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		tty  bool
		want bool
	}{
		{"tty", nil, true, true},
		{"not a tty", nil, false, false},
		{"NO_COLOR", map[string]string{"NO_COLOR": "1"}, true, false},
		{"TERM=dumb", map[string]string{"TERM": "dumb"}, true, false},
		{"CLICOLOR_FORCE on a pipe", map[string]string{"CLICOLOR_FORCE": "1"}, false, true},
		{"CLICOLOR_FORCE=0 does not force", map[string]string{"CLICOLOR_FORCE": "0"}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := colorOK(envOf(tc.env), tc.tty); got != tc.want {
				t.Fatalf("colorOK = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUnicodeOK(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		goos string
		want bool
	}{
		{"windows", nil, "windows", true},
		{"windows TERM=dumb", map[string]string{"TERM": "dumb"}, "windows", false},
		{"linux UTF-8 LANG", map[string]string{"LANG": "en_US.UTF-8"}, "linux", true},
		{"linux utf8 LC_CTYPE", map[string]string{"LC_CTYPE": "C.utf8"}, "linux", true},
		{"linux C locale", map[string]string{"LANG": "C"}, "linux", false},
		{"linux no locale", nil, "linux", false},
		{"darwin TERM=dumb", map[string]string{"TERM": "dumb", "LANG": "en_US.UTF-8"}, "darwin", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := unicodeOK(envOf(tc.env), tc.goos); got != tc.want {
				t.Fatalf("unicodeOK = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetectPipes(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})

	caps := Detect(envOf(map[string]string{"COLUMNS": "132"}), r, w, nil)
	if caps.In || caps.Out.TTY || caps.Err.TTY {
		t.Fatalf("pipes reported as terminals: %+v", caps)
	}
	if caps.Out.Color {
		t.Fatal("colour on a pipe without CLICOLOR_FORCE")
	}
	if caps.Out.Width != 132 {
		t.Fatalf("Out.Width = %d, want COLUMNS=132", caps.Out.Width)
	}

	for _, cols := range []string{"", "abc", "-3"} {
		caps = Detect(envOf(map[string]string{"COLUMNS": cols}), nil, w, nil)
		if caps.Out.Width != defaultWidth {
			t.Fatalf("COLUMNS=%q: Width = %d, want %d", cols, caps.Out.Width, defaultWidth)
		}
	}
}
