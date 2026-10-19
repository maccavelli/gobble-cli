package builtin

import (
	"encoding/json/v2"
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

// Every form 0012-MADR D10 lists runs as its native tool, with its call's
// id; every near miss runs as bash (0005-PLAN F6-3 decision 9; 0012-PLAN's
// gate).
func TestShellIntercept(t *testing.T) {
	for _, c := range []struct{ cmd, name, args string }{
		{"rm a.txt", "delete", `{"path":"a.txt"}`},
		{"rm -r build", "delete", `{"path":"build","recursive":true}`},
		{"rm -R build", "delete", `{"path":"build","recursive":true}`},
		{"rmdir empty", "delete", `{"path":"empty"}`},
		{"mv a b", "move", `{"from":"a","to":"b"}`},
		{"mv -f a b", "move", `{"from":"a","overwrite":true,"to":"b"}`},
		{"cp a b", "copy", `{"from":"a","to":"b"}`},
		{"cp -r d e", "copy", `{"from":"d","to":"e"}`},
		{"cp -R d e", "copy", `{"from":"d","to":"e"}`},
		{"cp -f a b", "copy", `{"from":"a","overwrite":true,"to":"b"}`},
		{"mkdir x", "mkdir", `{"path":"x"}`},
		{"mkdir -p x/y", "mkdir", `{"path":"x/y"}`},
		{`rm "a file.txt"`, "delete", `{"path":"a file.txt"}`},
		{"  rm a.txt  ", "delete", `{"path":"a.txt"}`},
	} {
		got, ok := interceptShell(tool.Call{ID: "c1", Name: "bash", Args: shellArgs(t, c.cmd, "")})
		if !ok || got.ID != "c1" || got.Name != c.name || !sameJSON(t, got.Args, c.args) {
			t.Errorf("%q: %v %s %s, want %s %s", c.cmd, ok, got.Name, got.Args, c.name, c.args)
		}
	}
	for _, cmd := range []string{
		"rm -f a", "rm -rf a", "rm -r -f a", "rm a b", "rm", "rm a -r", "rm -- a",
		"rmdir -p a", "rmdir a b",
		"mv a", "mv a b c", "mv -i a b", "mv -n a b",
		"cp -rf a b", "cp -r -f a b", "cp -a a b", "cp a", "cp a b c",
		"mkdir a b", "mkdir -m 700 a", "mkdir -p -v a",
		"rm *.txt", "rm a?", "rm [ab]", "rm {a,b}", "rm ~/x", "rm $F", "rm $(ls)",
		"rm a | cat", "rm a && ls", "rm a; ls", "rm a > log", "X=1 rm a", "! rm a", "rm a &", "(rm a)",
		"ls a", "sudo rm a", "/bin/rm a", `bash -c "rm a"`,
	} {
		if got, ok := interceptShell(tool.Call{ID: "c1", Name: "bash", Args: shellArgs(t, cmd, "")}); ok {
			t.Errorf("%q: intercepted as %s %s", cmd, got.Name, got.Args)
		}
	}
	if got, ok := interceptShell(tool.Call{ID: "c1", Name: "bash", Args: shellArgs(t, "rm a", "sub")}); ok {
		t.Errorf("a call with a workdir: intercepted as %s", got.Name)
	}
}

func shellArgs(t *testing.T, command, workdir string) []byte {
	t.Helper()
	b, err := json.Marshal(shellIn{Command: command, Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// sameJSON reports whether a and b are one JSON value.
func sameJSON(t *testing.T, a []byte, b string) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &y); err != nil {
		t.Fatal(err)
	}
	ax, _ := json.Marshal(x, json.Deterministic(true))
	by, _ := json.Marshal(y, json.Deterministic(true))
	return string(ax) == string(by)
}
