//go:build windows && conhost

package editor

// The live Console Host suite (0008-PLAN P3 step 7): measurement 10 rebuilt
// as a test. TestConhost starts this test binary again inside a new
// conhost.exe window. The child runs a real Editor on CONIN$/CONOUT$. The
// parent pastes three lines through conhost's own Paste command
// (WM_COMMAND 0xFFF1), so nothing is typed into whichever window has focus.
// The child then injects Enter, a typed line, Ctrl+C and another line into
// its own console input buffer. The clipboard is saved and restored.
//
// Run it with `make probe-conhost`. It opens a console window for a few
// seconds and fails, rather than skips, where no Console Host is available
// (0008-PLAN C5).

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

const (
	childEnv       = "GOBBLE_CONHOST_DIR"
	wmCommand      = 0x0111
	idPaste        = 0xFFF1
	cfUnicodeText  = 13
	gmemMoveable   = 0x0002
	leftCtrl       = 0x0008
	pasteText      = "line1\r\nline2\r\nline3"
	consoleClass   = "ConsoleWindowClass"
	childDeadline  = 25 * time.Second
	parentDeadline = 40 * time.Second
)

var (
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	user32                         = windows.NewLazySystemDLL("user32.dll")
	procGetConsoleWindow           = kernel32.NewProc("GetConsoleWindow")
	procWriteConsoleInputW         = kernel32.NewProc("WriteConsoleInputW")
	procGlobalAlloc                = kernel32.NewProc("GlobalAlloc")
	procGlobalFree                 = kernel32.NewProc("GlobalFree")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
	procGlobalSize                 = kernel32.NewProc("GlobalSize")
	procRtlMoveMemory              = kernel32.NewProc("RtlMoveMemory")
	procSendMessageW               = user32.NewProc("SendMessageW")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procEmptyClipboard             = user32.NewProc("EmptyClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procSetClipboardData           = user32.NewProc("SetClipboardData")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
)

// TestMain runs the child half when the parent started this binary.
func TestMain(m *testing.M) {
	if dir := os.Getenv(childEnv); dir != "" {
		os.Exit(child(dir))
	}
	os.Exit(m.Run())
}

func TestConhost(t *testing.T) {
	dir := t.TempDir()
	saved, err := saveClipboard()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := restoreClipboard(saved); err != nil {
			t.Errorf("restore clipboard: %v", err)
		}
	})
	if err := setClipboardText(pasteText); err != nil {
		t.Fatal(err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), parentDeadline)
	defer cancel()
	t.Setenv(childEnv, dir) // inherited by conhost.exe and the child
	waited, err := startConhost(ctx, exe, "-test.run=^$")
	if err != nil {
		t.Fatal(err)
	}

	ready, ok := waitFile(ctx, filepath.Join(dir, "ready"), waited)
	if !ok {
		result, _ := os.ReadFile(filepath.Join(dir, "result"))
		t.Fatalf("the child never reported its console window; its result: %q", result)
	}
	hwndText, class, _ := strings.Cut(ready, " ")
	t.Logf("console window class %q (hwnd %s)", class, hwndText)
	if class != consoleClass {
		t.Fatalf("console window class = %q, want %q: the child is not in Console Host", class, consoleClass)
	}
	hwnd, err := strconv.ParseUint(hwndText, 10, 64)
	if err != nil || hwnd == 0 {
		t.Fatalf("console window handle %q: %v", hwndText, err)
	}
	procSendMessageW.Call(uintptr(hwnd), wmCommand, idPaste, 0)
	if err := os.WriteFile(filepath.Join(dir, "pasted"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-waited:
		if err != nil {
			t.Errorf("child: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("child did not exit")
	}
	result, err := os.ReadFile(filepath.Join(dir, "result"))
	if err != nil {
		t.Fatalf("child wrote no result: %v", err)
	}
	var subs []string
	for line := range strings.Lines(string(result)) {
		s, err := strconv.Unquote(strings.TrimSpace(line))
		if err != nil {
			t.Fatalf("result line %q: %v", line, err)
		}
		subs = append(subs, s)
	}
	want := []string{"line1\nline2\nline3", "xy", "<nil>"}
	if fmt.Sprint(subs) != fmt.Sprint(want) {
		t.Fatalf("child submissions = %q, want %q (one pasted submission; Ctrl+C cleared \"abc\")", subs, want)
	}
}

// startConhost runs conhost.exe with argv in a new console window and
// returns a channel that receives its exit. It calls CreateProcess itself:
// os/exec always sets STARTF_USESTDHANDLES, and conhost.exe started that way
// exits at once without running its child (measured 2026-10-05). The
// process is terminated when ctx is done.
func startConhost(ctx context.Context, argv ...string) (<-chan error, error) {
	cl, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{"conhost.exe"}, argv...)))
	if err != nil {
		return nil, err
	}
	si := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(nil, cl, nil, nil, false, windows.CREATE_NEW_CONSOLE, nil, nil, &si, &pi); err != nil {
		return nil, fmt.Errorf("start conhost.exe: %w", err)
	}
	if err := windows.CloseHandle(pi.Thread); err != nil {
		return nil, err
	}
	exited := make(chan error, 1)
	go func() {
		defer windows.CloseHandle(pi.Process)
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				windows.TerminateProcess(pi.Process, 1)
			case <-done:
			}
		}()
		_, err := windows.WaitForSingleObject(pi.Process, windows.INFINITE)
		close(done)
		var code uint32
		if err == nil {
			err = windows.GetExitCodeProcess(pi.Process, &code)
		}
		if err == nil && code != 0 {
			err = fmt.Errorf("conhost.exe exited with %d", code)
		}
		exited <- err
	}()
	return exited, nil
}

// waitFile polls for path until ctx is done or the child exits.
func waitFile(ctx context.Context, path string, exited <-chan error) (string, bool) {
	for {
		if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
			return string(b), true
		}
		select {
		case <-ctx.Done():
			return "", false
		case <-exited:
			return "", false
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// child runs inside conhost.exe. It reports the console window, reads two
// submissions with a real Editor, and writes them to dir/result.
func child(dir string) int {
	ctx, cancel := context.WithTimeout(context.Background(), childDeadline)
	defer cancel()
	subs, err := childRun(ctx, dir)
	var b strings.Builder
	for _, s := range subs {
		b.WriteString(strconv.Quote(s) + "\n")
	}
	b.WriteString(strconv.Quote(fmt.Sprint(err)) + "\n")
	if werr := os.WriteFile(filepath.Join(dir, "result"), []byte(b.String()), 0o600); werr != nil {
		return 1
	}
	return 0
}

func childRun(ctx context.Context, dir string) (subs []string, err error) {
	in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	restoreOut, err := term.PrepareConsole(out, out)
	if err != nil {
		return nil, err
	}
	defer restoreOut()
	restoreIn, err := Raw(in)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, restoreIn()) }()

	r := &Reader{}
	r.Start(ctx, in)
	e := New(r, out, out, Options{})
	defer e.Restore()

	if err := os.WriteFile(filepath.Join(dir, "ready"), []byte(consoleWindow()), 0o600); err != nil {
		return nil, err
	}
	injectErr := make(chan error, 1)
	go func() { injectErr <- injectAfterPaste(ctx, dir, in) }()

	for range 2 {
		s, rerr := e.ReadPrompt(ctx)
		if rerr != nil {
			return subs, errors.Join(rerr, <-injectErr)
		}
		subs = append(subs, s)
	}
	return subs, <-injectErr
}

// injectAfterPaste waits for the parent's paste, then writes key events
// behind it in the console input buffer: Enter ends the pasted
// submission; "abc", Ctrl+C, "xy", Enter is the second.
func injectAfterPaste(ctx context.Context, dir string, in *os.File) error {
	for {
		if _, err := os.Stat(filepath.Join(dir, "pasted")); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	var recs []inputRecord
	recs = append(recs, key(windows.VK_RETURN, '\r', 0)...)
	for _, c := range "abc" {
		recs = append(recs, key(uint16(c-'a'+'A'), uint16(c), 0)...)
	}
	recs = append(recs, key('C', 0x03, leftCtrl)...)
	for _, c := range "xy" {
		recs = append(recs, key(uint16(c-'a'+'A'), uint16(c), 0)...)
	}
	recs = append(recs, key(windows.VK_RETURN, '\r', 0)...)
	var written uint32
	ok, _, err := procWriteConsoleInputW.Call(in.Fd(), uintptr(unsafe.Pointer(&recs[0])), uintptr(len(recs)), uintptr(unsafe.Pointer(&written)))
	if ok == 0 {
		return fmt.Errorf("WriteConsoleInputW: %w", err)
	}
	if int(written) != len(recs) {
		return fmt.Errorf("WriteConsoleInputW wrote %d of %d records", written, len(recs))
	}
	return nil
}

// inputRecord is INPUT_RECORD holding a KEY_EVENT_RECORD.
type inputRecord struct {
	eventType uint16
	_         uint16
	keyDown   int32
	repeat    uint16
	vk        uint16
	scan      uint16
	char      uint16
	ctrlState uint32
}

func key(vk, char uint16, ctrl uint32) []inputRecord {
	down := inputRecord{eventType: windows.KEY_EVENT, keyDown: 1, repeat: 1, vk: vk, char: char, ctrlState: ctrl}
	up := down
	up.keyDown = 0
	return []inputRecord{down, up}
}

func consoleWindow() string {
	h, _, _ := procGetConsoleWindow.Call()
	if h == 0 {
		return "0 "
	}
	buf := make([]uint16, 256)
	n, err := windows.GetClassName(windows.HWND(h), &buf[0], int32(len(buf)))
	if err != nil {
		return fmt.Sprintf("%d ", h)
	}
	return fmt.Sprintf("%d %s", h, windows.UTF16ToString(buf[:n]))
}

// savedClipboard is the clipboard's text, if it held text.
type savedClipboard struct {
	text    string
	hasText bool
}

func withClipboard(fn func() error) error {
	var err error
	for range 20 {
		var ok uintptr
		if ok, _, err = procOpenClipboard.Call(0); ok != 0 {
			defer procCloseClipboard.Call()
			return fn()
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("OpenClipboard: %w", err)
}

func saveClipboard() (savedClipboard, error) {
	var s savedClipboard
	err := withClipboard(func() error {
		for _, f := range []uintptr{2, 8, 11, 12, 15, 17} { // bitmap, DIB, RIFF, wave, file drop, DIBv5
			if ok, _, _ := procIsClipboardFormatAvailable.Call(f); ok != 0 {
				return errors.New("the clipboard holds non-text data; not overwriting it")
			}
		}
		if ok, _, _ := procIsClipboardFormatAvailable.Call(cfUnicodeText); ok == 0 {
			return nil
		}
		h, _, err := procGetClipboardData.Call(cfUnicodeText)
		if h == 0 {
			return fmt.Errorf("GetClipboardData: %w", err)
		}
		text, err := readGlobal(h)
		s = savedClipboard{text: text, hasText: true}
		return err
	})
	return s, err
}

func restoreClipboard(s savedClipboard) error {
	if s.hasText {
		return setClipboardText(s.text)
	}
	return withClipboard(func() error {
		if ok, _, err := procEmptyClipboard.Call(); ok == 0 {
			return fmt.Errorf("EmptyClipboard: %w", err)
		}
		return nil
	})
}

func setClipboardText(text string) error {
	u, err := windows.UTF16FromString(text)
	if err != nil {
		return err
	}
	size := uintptr(len(u) * 2)
	h, _, err := procGlobalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return fmt.Errorf("GlobalAlloc: %w", err)
	}
	p, _, err := procGlobalLock.Call(h)
	if p == 0 {
		procGlobalFree.Call(h)
		return fmt.Errorf("GlobalLock: %w", err)
	}
	procRtlMoveMemory.Call(p, uintptr(unsafe.Pointer(&u[0])), size)
	procGlobalUnlock.Call(h)
	return withClipboard(func() error {
		if ok, _, err := procEmptyClipboard.Call(); ok == 0 {
			procGlobalFree.Call(h)
			return fmt.Errorf("EmptyClipboard: %w", err)
		}
		if r, _, err := procSetClipboardData.Call(cfUnicodeText, h); r == 0 {
			procGlobalFree.Call(h)
			return fmt.Errorf("SetClipboardData: %w", err)
		}
		return nil // the clipboard owns h now
	})
}

func readGlobal(h uintptr) (string, error) {
	size, _, err := procGlobalSize.Call(h)
	if size == 0 {
		return "", fmt.Errorf("GlobalSize: %w", err)
	}
	p, _, err := procGlobalLock.Call(h)
	if p == 0 {
		return "", fmt.Errorf("GlobalLock: %w", err)
	}
	defer procGlobalUnlock.Call(h)
	buf := make([]uint16, size/2)
	procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&buf[0])), p, size)
	return windows.UTF16ToString(buf), nil
}
