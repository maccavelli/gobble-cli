package editor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	cases := []struct {
		in      string
		want    []string
		wantErr bool
	}{
		{in: "code --wait", want: []string{"code", "--wait"}},
		{in: `"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl" -w`,
			want: []string{"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl", "-w"}},
		{in: `C:\Windows\System32\notepad.exe`, want: []string{`C:\Windows\System32\notepad.exe`}},
		{in: `"C:\Program Files\Editor\ed.exe" --wait`, want: []string{`C:\Program Files\Editor\ed.exe`, "--wait"}},
		{in: `vim -c 'set tw=72'`, want: []string{"vim", "-c", "set tw=72"}},
		{in: `code --wait "unclosed`, wantErr: true},
		{in: `""`, wantErr: true},
	}
	for _, tc := range cases {
		got, err := splitCommand(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("splitCommand(%q) = %q, want an error", tc.in, got)
			}
			continue
		}
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("splitCommand(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestEditorCommand(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		goos string
		want []string
	}{
		{"GOBBLE_EDITOR first", map[string]string{"GOBBLE_EDITOR": "g", "VISUAL": "v", "EDITOR": "e"}, "linux", []string{"g"}},
		{"then VISUAL", map[string]string{"GOBBLE_EDITOR": " ", "VISUAL": "v", "EDITOR": "e"}, "linux", []string{"v"}},
		{"then EDITOR", map[string]string{"EDITOR": "e -x"}, "linux", []string{"e", "-x"}},
		{"vi by default", nil, "darwin", []string{"vi"}},
		{"notepad on Windows", nil, "windows", []string{"notepad"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := editorCommand(func(k string) string { return tc.env[k] }, tc.goos)
			if err != nil || !slices.Equal(got, tc.want) {
				t.Fatalf("editorCommand = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestCompose(t *testing.T) {
	dir := t.TempDir()
	env := func(k string) string {
		if k == "GOBBLE_EDITOR" {
			return "myeditor --wait"
		}
		return ""
	}
	var file string
	run := func(cmd *exec.Cmd) error {
		file = cmd.Args[len(cmd.Args)-1]
		if !slices.Equal(cmd.Args[:2], []string{"myeditor", "--wait"}) {
			t.Errorf("args = %q", cmd.Args)
		}
		if filepath.Dir(file) != dir {
			t.Errorf("temp file %s is not in %s", file, dir)
		}
		b, err := os.ReadFile(file)
		if err != nil || string(b) != "draft" {
			t.Errorf("editor saw %q, %v; want \"draft\"", b, err)
		}
		return os.WriteFile(file, []byte("edited\nprompt\n"), 0o600)
	}
	got, err := Compose(t.Context(), env, dir, "draft", run)
	if err != nil || got != "edited\nprompt" {
		t.Fatalf("Compose = %q, %v; want \"edited\\nprompt\"", got, err)
	}
	if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp file %s not removed: %v", file, err)
	}
}

func TestComposeEditorFails(t *testing.T) {
	dir := t.TempDir()
	boom := errors.New("exit status 1")
	_, err := Compose(t.Context(), func(string) string { return "" }, dir, "x", func(*exec.Cmd) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the editor's error", err)
	}
	if des, _ := os.ReadDir(dir); len(des) != 0 {
		t.Fatalf("temp file left behind: %v", des)
	}
}
