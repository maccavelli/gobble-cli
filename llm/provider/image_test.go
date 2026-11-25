package provider

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	sdktest "github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"

	"github.com/maccavelli/gobble-cli/llm"
)

// textOnlyLayer states that every model reads text and nothing else
// (0036-MADR D7).
type textOnlyLayer struct{}

func (textOnlyLayer) Name() string { return "test" }

func (textOnlyLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	return catalog.ModelInfo{ID: id, Input: []catalog.Modality{catalog.ModalityText}}.Known(catalog.FieldInput), true
}

func imagePNG(img llmprovider.ImageItem, png []byte) bool {
	return img.Role == llmprovider.RoleUser && img.MediaType == "image/png" && img.URL == "" && bytes.Equal(img.Data, png)
}

func indexText(in []llmprovider.Item, role llmprovider.Role, text string) int {
	for i, it := range in {
		if m, ok := it.(llmprovider.MessageItem); ok && m.Role == role && m.Text == text {
			return i
		}
	}
	return -1
}

func indexOutput(in []llmprovider.Item, callID string) int {
	for i, it := range in {
		if o, ok := it.(llmprovider.FunctionCallOutputItem); ok && o.CallID == callID {
			return i
		}
	}
	return -1
}

// A user image stays where it was, and a tool message's image is the item
// after that call's output, with the load after the image (0036-MADR D7;
// 0036-PLAN P6).
func TestImagesToRequest(t *testing.T) {
	png := []byte("PNG")
	f := sdktest.NewFake("fake", llmprovider.Capabilities{Images: llmprovider.Supported, Tools: llmprovider.Supported}).ReplyText("seen")
	req := &llm.Request{
		Tools:    specsOf(t, deferSearch),
		Deferred: specsOf(t, deferPerms),
		Search:   deferSearch.Name,
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: []llm.Content{
				{Type: llm.ContentText, Text: "look"},
				{Type: llm.ContentImage, MediaType: "image/png", Data: png},
				{Type: llm.ContentText, Text: "please"},
			}},
			{Role: llm.RoleAssistant, Content: []llm.Content{{
				Type: llm.ContentToolCall, ToolCallID: "s1", ToolName: deferSearch.Name, ToolArgs: `{"query":"permissions"}`,
			}}},
			{Role: llm.RoleTool, Content: []llm.Content{
				{Type: llm.ContentToolResult, ToolCallID: "s1", ToolName: deferSearch.Name, Text: "Loaded request_permissions.", Loads: []string{deferPerms.Name}},
				{Type: llm.ContentImage, MediaType: "image/png", Data: png},
			}},
		},
	}
	if _, err := drain(t, Wrap(f), req); err != nil {
		t.Fatal(err)
	}
	recs := f.Requests()
	if len(recs) != 1 {
		t.Fatalf("%d requests, want the one the fake recorded", len(recs))
	}
	in := recs[0].Input
	look := indexText(in, llmprovider.RoleUser, "look")
	please := indexText(in, llmprovider.RoleUser, "please")
	if look < 0 || please != look+2 {
		t.Fatalf("user items around the image: %+v", in)
	}
	if img, ok := in[look+1].(llmprovider.ImageItem); !ok || !imagePNG(img, png) {
		t.Fatalf("user image %#v, want an ImageItem in place", in[look+1])
	}
	out := indexOutput(in, "s1")
	if out < 0 || out+2 >= len(in) {
		t.Fatalf("input %+v, want the image and its load after s1", in)
	}
	if img, ok := in[out+1].(llmprovider.ImageItem); !ok || !imagePNG(img, png) {
		t.Fatalf("after s1's result: %#v, want its ImageItem", in[out+1])
	}
	if load, ok := in[out+2].(llmprovider.ToolLoadItem); !ok || load != (llmprovider.ToolLoadItem{CallID: "s1", Name: deferPerms.Name}) {
		t.Fatalf("after the image: %#v, want the ToolLoadItem", in[out+2])
	}
}

// imageProbeRequest is a tool result with an image after it: the shape the
// probe table recorded (0036-MADR).
func imageProbeRequest() *llm.Request {
	return &llm.Request{Messages: []llm.Message{
		{Role: llm.RoleAssistant, Content: []llm.Content{{Type: llm.ContentToolCall, ToolCallID: "c1", ToolName: "read", ToolArgs: "{}"}}},
		{Role: llm.RoleTool, Content: []llm.Content{
			{Type: llm.ContentToolResult, ToolCallID: "c1", ToolName: "read", Text: "a picture"},
			{Type: llm.ContentImage, MediaType: "image/png", Data: []byte("PNG")},
		}},
	}}
}

// bodyFor is the one request whose path ends with suffix.
func bodyFor(t *testing.T, c *cannedWire, suffix string) string {
	t.Helper()
	paths, bodies := c.requests()
	var found string
	n := 0
	for i, p := range paths {
		if strings.HasSuffix(p, suffix) {
			n++
			found = bodies[i]
		}
	}
	if n != 1 {
		t.Fatalf("%s requests %d (%q), want 1", suffix, n, paths)
	}
	return found
}

// imageWhere fails unless body holds png's base64 after before and before after.
func imageWhere(t *testing.T, body, before, after string) {
	t.Helper()
	if !strings.Contains(body, "UE5H") {
		t.Fatalf("body has no image bytes:\n%s", body)
	}
	bi, ai := strings.Index(body, before), strings.Index(body, after)
	if bi < 0 || ai < 0 || bi > ai {
		t.Fatalf("%s at %d, %s at %d; want the image after the tool output:\n%s", before, bi, after, ai, body)
	}
}

// Each wire carries the image where the probe table says (0036-PLAN P6).
func TestImageWires(t *testing.T) {
	off := &llm.Retry{Off: true}
	for _, w := range []struct{ id, model, path, before, after string }{
		{"claude", "claude-haiku-5-5", "/messages", `"type":"tool_result"`, `"type":"image"`},
		{"openai", "gpt-6-luna", "/responses", `"type":"function_call_output"`, `"type":"input_image"`},
		{"gemini", "gemini-3.5-flash-lite", "/interactions", `"type":"function_result"`, `"type":"image"`},
	} {
		t.Run(w.id, func(t *testing.T) {
			c := &cannedWire{}
			p, err := New(w.id, Options{APIKey: "k", Model: w.model, HTTPClient: &http.Client{Transport: c}, Retry: off})
			if err != nil {
				t.Fatal(err)
			}
			req := assumedLimit(imageProbeRequest())
			req.Model = w.model
			if _, err := drain(t, p, req); err != nil {
				t.Fatal(err)
			}
			imageWhere(t, bodyFor(t, c, w.path), w.before, w.after)
		})
	}
	t.Run("chat", func(t *testing.T) {
		c := &cannedWire{}
		set, _ := chatSet(t, `{"providers":{"local":{"name":"Local","baseUrl":"http://127.0.0.1:1/v1"}}}`)
		p, err := New("local", Options{Set: set, Model: "m", BaseURL: "http://127.0.0.1:1/v1",
			HTTPClient: &http.Client{Transport: c}, Retry: off})
		if err != nil {
			t.Fatal(err)
		}
		req := imageProbeRequest()
		req.Model = "m"
		if _, err := drain(t, p, req); err != nil {
			t.Fatal(err)
		}
		imageWhere(t, bodyFor(t, c, "/chat/completions"), `"role":"tool"`, `"image_url"`)
	})
}

// ImageInput follows Capabilities.Images: anything but Unsupported (0036-MADR D7).
func TestCapabilitiesImageInput(t *testing.T) {
	for _, c := range []struct {
		name   string
		images llmprovider.Support
		want   bool
	}{
		{"supported", llmprovider.Supported, true},
		{"best effort", llmprovider.BestEffort, true},
		{"unsupported", llmprovider.Unsupported, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := Wrap(sdktest.NewFake("fake", llmprovider.Capabilities{Images: c.images}))
			if got := p.Capabilities().ImageInput; got != c.want {
				t.Fatalf("ImageInput %v, want %v", got, c.want)
			}
		})
	}
}

// A model whose facts say text only is refused before the network (0036-MADR D7).
func TestTextOnlyModelImage(t *testing.T) {
	c := &cannedWire{}
	raw, err := NewSet().registry().New(llmprovider.ProviderClaude,
		llmprovider.WithAPIKey("k"),
		llmprovider.WithModel("text-only"),
		llmprovider.WithModelProbes(false),
		llmprovider.WithHTTPClient(&http.Client{Transport: c}),
		catalog.WithModelLayer(textOnlyLayer{}),
	)
	if err != nil {
		t.Fatal(err)
	}
	req := &llm.Request{Model: "text-only", Messages: []llm.Message{{
		Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentImage, MediaType: "image/png", Data: []byte("PNG")}},
	}}}
	_, err = drain(t, Wrap(raw), assumedLimit(req))
	if !errors.Is(err, llm.ErrUnsupported) {
		t.Fatalf("err %v, want llm.ErrUnsupported", err)
	}
	if paths, _ := c.requests(); len(paths) != 0 {
		t.Fatalf("sent %q, want nothing", paths)
	}
}
