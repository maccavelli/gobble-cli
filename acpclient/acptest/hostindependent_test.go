package acptest

import (
	"testing"

	sdkacptest "github.com/maccavelli/acp-go-sdk2/acptest"
)

// A path under the host's root is written in slash form, and nothing else in
// the frame moves; with "/" as the root, frames are unchanged (0019-MADR D4,
// 0016-MADR D6). The root is a parameter here, so the Windows case runs on
// every host.
func TestHostIndependent(t *testing.T) {
	const first = `{"cwd":"C:\\work\\a.go","name":"C:\\not-a-path-of-ours","other":"D:\\elsewhere"}`
	in := []sdkacptest.Record{
		{Direction: sdkacptest.Sent, Frame: []byte(first)},
		{Direction: sdkacptest.Received, Frame: []byte(`{"text":"no path here"}`)},
	}
	got := slashRecords(in, `C:\`)
	want := []string{`{"cwd":"/work/a.go","name":"/not-a-path-of-ours","other":"D:\\elsewhere"}`, `{"text":"no path here"}`}
	for i := range want {
		if got[i].Direction != in[i].Direction || string(got[i].Frame) != want[i] {
			t.Errorf("record %d = %s %s, want %s %s", i, got[i].Direction, got[i].Frame, in[i].Direction, want[i])
		}
	}
	if string(in[0].Frame) != first {
		t.Errorf("the input record changed to %s; want it left as it was", in[0].Frame)
	}
	for i, r := range slashRecords(in, "/") {
		if string(r.Frame) != string(in[i].Frame) {
			t.Errorf("with root \"/\", record %d = %s, want it unchanged", i, r.Frame)
		}
	}
}
