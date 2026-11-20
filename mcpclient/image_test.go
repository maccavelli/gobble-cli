package mcpclient

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/gobble-cli/tool"
)

// An image result is prepared the same way read prepares one (0036-PLAN P7).
func TestMCPImageResult(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	res := preparedResult(&mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: "see"},
		&mcp.ImageContent{MIMEType: "image/png", Data: b.Bytes()},
	}}, tool.Env{Images: true})
	if res.IsError || res.Text() == "" || len(res.Images) != 1 {
		t.Fatalf("result error %v, %d images, text %q", res.IsError, len(res.Images), res.Text())
	}
	if res.Images[0].MediaType != "image/png" && res.Images[0].MediaType != "image/jpeg" {
		t.Fatalf("media %s, want png or jpeg", res.Images[0].MediaType)
	}
	if _, _, err := image.Decode(bytes.NewReader(res.Images[0].Data)); err != nil {
		t.Fatalf("prepared image: %v", err)
	}
	if !bytes.Contains([]byte(res.Text()), []byte("see")) {
		t.Fatalf("text %q lost the other block", res.Text())
	}
	note := preparedResult(&mcp.CallToolResult{Content: []mcp.Content{
		&mcp.ImageContent{MIMEType: "image/png", Data: b.Bytes()},
	}}, tool.Env{})
	if len(note.Images) != 0 || !bytes.Contains([]byte(note.Text()), []byte(tool.ImageNotSent)) {
		t.Fatalf("text-only images %d, text %q", len(note.Images), note.Text())
	}
}
