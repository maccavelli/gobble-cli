package mcpclient

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/gobble-cli/tool"
)

// mcpTool is one server tool as a gobble tool.
type mcpTool struct {
	conn *conn
	tool *mcp.Tool
	spec tool.Spec
}

var (
	_ tool.Tool      = (*mcpTool)(nil)
	_ tool.Describer = (*mcpTool)(nil)
)

func newTool(c *conn, t *mcp.Tool, name string) *mcpTool {
	var a tool.Annotations
	if an := t.Annotations; an != nil {
		a.ReadOnlyHint, a.IdempotentHint = an.ReadOnlyHint, an.IdempotentHint
		a.DestructiveHint = an.DestructiveHint != nil && *an.DestructiveHint
		a.OpenWorldHint = an.OpenWorldHint != nil && *an.OpenWorldHint
	}
	return &mcpTool{conn: c, tool: t, spec: tool.Spec{
		Name:        name,
		Description: description(c.server.Name, t),
		InputSchema: inputSchema(t.InputSchema),
		Annotations: a,
		Kind:        tool.KindOther,
		Exposure:    tool.ExposureDirect,
	}}
}

// description is Pi's: the trimmed description, else the title, else
// "MCP tool <tool> from server <server>".
func description(server string, t *mcp.Tool) string {
	if d := strings.TrimSpace(t.Description); d != "" {
		return d
	}
	if t.Title != "" {
		return t.Title
	}
	if t.Annotations != nil && t.Annotations.Title != "" {
		return t.Annotations.Title
	}
	return fmt.Sprintf("MCP tool %s from server %s", t.Name, server)
}

// inputSchema is the tool's schema with "type": "object" and
// "properties": {} added when missing, as Pi adds them: some providers
// reject an object schema without properties.
func inputSchema(s any) jsontext.Value {
	m, _ := s.(map[string]any) //nolint:errcheck // a non-object schema becomes an empty one
	if m == nil {
		m = map[string]any{}
	}
	if m["type"] == nil {
		m["type"] = "object"
	}
	if _, ok := m["properties"]; !ok {
		m["properties"] = map[string]any{}
	}
	b, err := json.Marshal(m, json.Deterministic(true))
	if err != nil {
		return jsontext.Value(`{"type":"object","properties":{}}`)
	}
	return b
}

func (t *mcpTool) Spec() tool.Spec { return t.spec }

// Describe titles a call <server>/<tool>, Pi's label.
func (t *mcpTool) Describe(tool.Call, tool.Env) string {
	return t.conn.server.Name + "/" + t.tool.Name
}

// Run calls the tool within the server's timeout. The server's errors, and
// a timeout, are error results the model reads; only a done ctx is an
// error.
func (t *mcpTool) Run(ctx context.Context, call tool.Call, _ tool.Env) (tool.Result, error) {
	label := t.Describe(call, tool.Env{})
	args := call.Args
	if len(args) == 0 {
		args = jsontext.Value("{}")
	}
	if args.Kind() != '{' {
		return tool.ErrorResult(fmt.Sprintf("MCP tool %s: the arguments must be a JSON object", label)), nil
	}
	cs := t.conn.session()
	if cs == nil {
		return tool.ErrorResult(fmt.Sprintf("MCP server %q is not connected", t.conn.server.Name)), nil
	}
	timeout := t.conn.server.Timeout
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res, err := cs.CallTool(cctx, &mcp.CallToolParams{Name: t.tool.Name, Arguments: args})
	switch {
	case ctx.Err() != nil:
		return tool.Result{}, context.Cause(ctx)
	case errors.Is(cctx.Err(), context.DeadlineExceeded):
		return tool.ErrorResult(fmt.Sprintf("MCP tool %s timed out after %s s", label, seconds(timeout))), nil
	case err != nil:
		return tool.ErrorResult(fmt.Sprintf("MCP tool %s failed: %v", label, err)), nil
	}
	text := resultText(res)
	if res.IsError && text == "" {
		text = fmt.Sprintf("MCP tool %s returned an error", label)
	}
	return tool.Result{Output: quote(text), IsError: res.IsError}, nil
}

func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', -1, 64)
}

func quote(s string) jsontext.Value {
	b, err := json.Marshal(s)
	if err != nil { // a string always encodes
		panic(err)
	}
	return b
}

// resultText is a result as the model reads it: each block as Pi's
// toLlmContent and tools.ts convert it, joined by newlines. gobble's tool
// results are text, so an image is a note until 0005-PLAN F1 carries
// images.
func resultText(r *mcp.CallToolResult) string {
	if len(r.Content) == 0 && r.StructuredContent != nil {
		b, err := json.Marshal(r.StructuredContent, json.Deterministic(true), jsontext.WithIndent("  "))
		if err == nil {
			return string(b)
		}
	}
	parts := make([]string, 0, len(r.Content))
	for _, c := range r.Content {
		parts = append(parts, blockText(c))
	}
	return strings.Join(parts, "\n")
}

func blockText(c mcp.Content) string {
	switch c := c.(type) {
	case *mcp.TextContent:
		return c.Text
	case *mcp.ImageContent:
		return fmt.Sprintf("[image %s omitted]", c.MIMEType)
	case *mcp.AudioContent:
		return fmt.Sprintf("[audio %s omitted]", c.MIMEType)
	case *mcp.ResourceLink:
		return resourceLink(c)
	case *mcp.EmbeddedResource:
		r := c.Resource
		switch {
		case r == nil:
			return "[unsupported MCP content resource]"
		case r.Blob == nil:
			return r.Text
		case textMIME(r.MIMEType):
			return string(r.Blob)
		}
		mime := r.MIMEType
		if mime == "" {
			mime = "unknown type"
		}
		return fmt.Sprintf("[binary resource %s (%s) omitted]", r.URI, mime)
	}
	return fmt.Sprintf("[unsupported MCP content %T]", c)
}

// resourceLink is Pi's: [Resource <uri> "<title or name>" (<mime>, <size>): <description>].
func resourceLink(l *mcp.ResourceLink) string {
	title := l.Title
	if title == "" {
		title = l.Name
	}
	var details []string
	if l.MIMEType != "" {
		details = append(details, l.MIMEType)
	}
	if l.Size != nil {
		details = append(details, formatSize(*l.Size))
	}
	s := "[Resource " + l.URI + ` "` + title + `"`
	if len(details) > 0 {
		s += " (" + strings.Join(details, ", ") + ")"
	}
	if l.Description != "" {
		s += ": " + l.Description
	}
	return s + "]"
}

// formatSize is Pi's (truncate.ts).
func formatSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(n)/(1024*1024))
}

// textMIME is Pi's isTextMimeType: blobs of these types are shown as text.
func textMIME(mime string) bool {
	t, _, _ := strings.Cut(mime, ";")
	t = strings.ToLower(strings.TrimSpace(t))
	return strings.HasPrefix(t, "text/") || t == "application/json" || strings.HasSuffix(t, "+json") || strings.HasSuffix(t, "+xml")
}
