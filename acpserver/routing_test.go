package acpserver

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// fsAndTerminal are the client capabilities the routing tests advertise.
var fsAndTerminal = acp.ClientCapabilities{Fs: acp.FileSystemCapabilities{ReadTextFile: true, WriteTextFile: true}, Terminal: true}

// routingSession is a session over tools, on p, with client advertising
// caps at initialize, in a new working directory it returns.
func routingSession(t *testing.T, p llm.Provider, client *acptest.Client, caps acp.ClientCapabilities, tools []tool.Tool) (*acptest.Conn, acp.SessionId, string) {
	t.Helper()
	ag := New(Options{Version: "1.2.3", Provider: func() (llm.Provider, error) { return p, nil }, Tools: tools})
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := c.Client.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber, ClientCapabilities: caps,
		ClientInfo: &acp.Implementation{Name: "test", Version: "0.0.0"}}); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	return c, newSession(t, c, cwd, nil).SessionId, cwd
}

func writeDisk(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// resultText is the tool result the model's last request carried.
func resultText(t *testing.T, s *llmtest.Script) string {
	t.Helper()
	reqs := s.Requests()
	last := reqs[len(reqs)-1].Messages
	return last[len(last)-1].Content[0].Text
}

// A read through a client that offers files asks fs/read_text_file and
// shows the client's text, not the disk's; an image stays local; a path
// outside the roots, approved, is read here and the client is not asked
// (0005-PLAN F1d-3, and F1's acceptance line).
func TestReadThroughClient(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside.txt")
	writeDisk(t, outside, "outside\n")
	client := &acptest.Client{Allow: true, Files: map[string]string{}}
	model := llmtest.NewScript(
		llmtest.ToolCall("r1", "read", `{"path":"a.txt"}`), llmtest.Text("ok"),
		llmtest.ToolCall("r2", "read", `{"path":"logo.png"}`), llmtest.Text("ok"),
		llmtest.ToolCall("r3", "read", args(t, map[string]string{"path": outside})), llmtest.Text("ok"),
	)
	c, id, cwd := routingSession(t, model, client, fsAndTerminal, []tool.Tool{builtin.Read()})
	a := filepath.Join(cwd, "a.txt")
	writeDisk(t, a, "on disk\n")
	client.Files[a] = "in the editor\n"
	if _, err := prompt(t, c, id, "read a"); err != nil {
		t.Fatal(err)
	}
	if got := resultText(t, model); !strings.Contains(got, "1: in the editor") || strings.Contains(got, "on disk") {
		t.Fatalf("read = %q; want the client's text", got)
	}
	png := filepath.Join(cwd, "logo.png")
	if err := os.WriteFile(png, []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, id, "read the logo"); err != nil {
		t.Fatal(err)
	}
	if got := resultText(t, model); !strings.Contains(got, "Read image file [image/png]") {
		t.Fatalf("read png = %q; want the local image note", got)
	}
	if _, err := prompt(t, c, id, "read outside"); err != nil {
		t.Fatal(err)
	}
	if got := resultText(t, model); !strings.Contains(got, "1: outside") {
		t.Fatalf("read outside = %q", got)
	}
	if got := client.FileCalls(); !slices.Equal(got, []string{"read " + a}) {
		t.Fatalf("file calls %v; want only the read of a.txt", got)
	}
}

// A write and an edit through a client that offers files ask
// fs/write_text_file (the edit reads twice first, for its compare-and-swap),
// and leave the disk as it was.
func TestWriteAndEditThroughClient(t *testing.T) {
	client := &acptest.Client{Allow: true, Files: map[string]string{}}
	model := llmtest.NewScript(
		llmtest.ToolCall("w1", "write", `{"path":"new.txt","content":"fresh\n"}`), llmtest.Text("ok"),
		llmtest.ToolCall("e1", "edit", `{"path":"a.txt","edits":[{"oldText":"editor","newText":"agent"}]}`), llmtest.Text("ok"),
	)
	c, id, cwd := routingSession(t, model, client, fsAndTerminal, []tool.Tool{builtin.Write(), builtin.Edit()})
	a, fresh := filepath.Join(cwd, "a.txt"), filepath.Join(cwd, "new.txt")
	writeDisk(t, a, "on disk\n")
	client.Files[a] = "in the editor\n"
	for _, p := range []string{"write", "edit"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	if got := client.FileCalls(); !slices.Equal(got, []string{"write " + fresh, "read " + a, "read " + a, "write " + a}) {
		t.Fatalf("file calls %v", got)
	}
	if client.Files[fresh] != "fresh\n" || client.Files[a] != "in the agent\n" {
		t.Fatalf("client files %v", client.Files)
	}
	if b, err := os.ReadFile(a); err != nil || string(b) != "on disk\n" {
		t.Fatalf("a.txt on disk = %q, %v; want it unchanged", b, err)
	}
	if _, err := os.Stat(fresh); !os.IsNotExist(err) {
		t.Fatalf("new.txt on disk: %v; want none", err)
	}
}

// bash, with UseClientTerminal and a client that offers terminals, runs in
// the client's terminal: create, wait for exit, output, release; the tool
// call carries the terminal; and the result is a local run's shape.
func TestBashInClientTerminal(t *testing.T) {
	client := &acptest.Client{Allow: true, Terminals: &acptest.Terminals{Output: "hello\n", ExitCode: 0}}
	model := llmtest.NewScript(llmtest.ToolCall("b1", "bash", `{"command":"echo hello"}`), llmtest.Text("ok"))
	c, id, cwd := routingSession(t, model, client, fsAndTerminal, clientTerminalBash(t))
	if _, err := prompt(t, c, id, "say hello"); err != nil {
		t.Fatal(err)
	}
	if got := client.Terminals.Calls(); !slices.Equal(got, []string{"create", "wait_for_exit", "output", "release"}) {
		t.Fatalf("terminal calls %v", got)
	}
	reqs := client.Terminals.Requests()
	if len(reqs) != 1 || !slices.Contains(reqs[0].Args, "echo hello") || reqs[0].Cwd == nil || *reqs[0].Cwd != cwd {
		t.Fatalf("terminal/create %+v", reqs)
	}
	if got := resultText(t, model); got != "hello\n[exit status 0]" {
		t.Fatalf("bash = %q", got)
	}
	if !carriesTerminal(client, "b1", "term-1") {
		t.Fatal("no tool_call_update carries the terminal")
	}
}

// A command in the client's terminal that outlives its timeout is stopped
// with terminal/kill, and the result says it was stopped.
func TestBashClientTerminalTimeout(t *testing.T) {
	client := &acptest.Client{Allow: true, Terminals: &acptest.Terminals{Hang: true}}
	model := llmtest.NewScript(llmtest.ToolCall("b1", "bash", `{"command":"sleep 100","timeout":1}`), llmtest.Text("ok"))
	c, id, _ := routingSession(t, model, client, fsAndTerminal, clientTerminalBash(t))
	if _, err := prompt(t, c, id, "wait"); err != nil {
		t.Fatal(err)
	}
	if got := client.Terminals.Calls(); !slices.Equal(got, []string{"create", "wait_for_exit", "kill", "output", "release"}) {
		t.Fatalf("terminal calls %v", got)
	}
	if got := resultText(t, model); !strings.Contains(got, "[stopped after 1 s") {
		t.Fatalf("bash = %q", got)
	}
}

// clientTerminalBash is bash with UseClientTerminal set, alone.
func clientTerminalBash(t *testing.T) []tool.Tool {
	t.Helper()
	for _, tl := range builtin.ToolsWith(builtin.Options{UseClientTerminal: true}) {
		if tl.Spec().Name == "bash" {
			return []tool.Tool{tl}
		}
	}
	t.Fatal("no bash")
	return nil
}

func carriesTerminal(client *acptest.Client, call acp.ToolCallId, term string) bool {
	for _, n := range client.Updates() {
		if u := n.Update.ToolCallUpdate; u != nil && u.ToolCallId == call {
			for _, ct := range u.Content {
				if ct.Terminal != nil && ct.Terminal.TerminalId == term {
					return true
				}
			}
		}
	}
	return false
}

// read and edit name their file at tool_call, and grep its matches' files
// and lines at tool_call_update (0005-PLAN F1d-3).
func TestLocations(t *testing.T) {
	client := &acptest.Client{Allow: true}
	model := llmtest.NewScript(
		llmtest.ToolCall("r1", "read", `{"path":"a.txt","offset":2}`),
		llmtest.ToolCall("e1", "edit", `{"path":"a.txt","edits":[{"oldText":"two","newText":"2"}]}`),
		llmtest.ToolCall("g1", "grep", `{"pattern":"one|three"}`),
		llmtest.Text("ok"),
	)
	c, id, cwd := routingSession(t, model, client, acp.ClientCapabilities{}, []tool.Tool{builtin.Read(), builtin.Edit(), builtin.Grep()})
	a := filepath.Join(cwd, "a.txt")
	writeDisk(t, a, "one\ntwo\nthree\n")
	if _, err := prompt(t, c, id, "look"); err != nil {
		t.Fatal(err)
	}
	starts, ends := map[acp.ToolCallId][]acp.ToolCallLocation{}, map[acp.ToolCallId][]acp.ToolCallLocation{}
	for _, n := range client.Updates() {
		if u := n.Update.ToolCall; u != nil {
			starts[u.ToolCallId] = u.Locations
		}
		if u := n.Update.ToolCallUpdate; u != nil && len(u.Locations) > 0 {
			ends[u.ToolCallId] = u.Locations
		}
	}
	line := func(l acp.ToolCallLocation) int {
		if l.Line == nil {
			return 0
		}
		return *l.Line
	}
	if l := starts["r1"]; len(l) != 1 || l[0].Path != a || line(l[0]) != 2 {
		t.Errorf("read's locations %+v", l)
	}
	if l := starts["e1"]; len(l) != 1 || l[0].Path != a || line(l[0]) != 0 {
		t.Errorf("edit's locations %+v", l)
	}
	if l := ends["g1"]; len(l) != 2 || l[0].Path != a || line(l[0]) != 1 || line(l[1]) != 3 {
		t.Errorf("grep's locations %+v", l)
	}
}

func args(t *testing.T, v map[string]string) string {
	t.Helper()
	return toolArgs(t, v)
}
