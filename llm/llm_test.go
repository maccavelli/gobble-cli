package llm

import (
	"bytes"
	"testing"

	"github.com/maccavelli/gobble-cli/jsonx"
)

// Content.Data is binary, and encodes as base64, not as raw JSON
// (0039-MADR D20).
func TestContentDataBase64(t *testing.T) {
	c := Content{Type: ContentImage, MediaType: "image/png", Data: []byte{0, 1, 2, 0xff}}
	b, err := jsonx.Marshal(c)
	if want := `{"type":"image","mediaType":"image/png","data":"AAEC/w=="}`; err != nil || string(b) != want {
		t.Fatalf("wrote %s, %v; want %s", b, err, want)
	}
	var back Content
	if err := jsonx.Unmarshal(b, &back); err != nil || !bytes.Equal(back.Data, c.Data) {
		t.Fatalf("read back %v, %v; want %v", back.Data, err, c.Data)
	}
}
