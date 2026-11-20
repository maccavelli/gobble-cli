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

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// mcpTool is one server tool as a gobble tool.
type mcpTool struct {
	conn *conn
	tool *mcp.Tool
	spec tool.Spec
	// rewritten reports that the rule of 0060-MADR D2 changed the schema;
	// flag is the reason it left a shape alone that some providers refuse,
	// or "".
	rewritten bool
	flag      string
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
	schema, rewritten, flag := inputSchema(t.InputSchema)
	return &mcpTool{conn: c, tool: t, rewritten: rewritten, flag: flag, spec: tool.Spec{
		Name:        name,
		Description: description(c.server.Name, t),
		InputSchema: schema,
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

// inputSchema is the schema the model is sent for a tool, as JSON, made on
// a deep copy of s, the go-sdk's own: "type": "object" and "properties": {}
// are added when missing, as Pi adds them, since some providers reject an
// object schema without properties; a schema that is not a JSON object
// becomes an empty one; then the rule of 0060-MADR D2 gives each branch of a
// root anyOf or oneOf the root's object keywords (see rewriteRootUnion).
// rewritten reports that the rule changed the schema, and flag, when the
// schema is a root union the rule did not cover, that some providers
// refuse, why (D3, D4).
func inputSchema(s any) (spec jsontext.Value, rewritten bool, flag string) {
	m := map[string]any{}
	if src, ok := s.(map[string]any); ok && src != nil {
		m = cloneMap(src)
	}
	if m["type"] == nil {
		m["type"] = typeObject
	}
	if _, ok := m["properties"]; !ok {
		m["properties"] = map[string]any{}
	}
	m, rewritten, flag = rewriteRootUnion(m)
	b, err := jsonx.Marshal(m, json.Deterministic(true))
	if err != nil {
		return jsontext.Value(`{"type":"object","properties":{}}`), false, ""
	}
	return b, rewritten, flag
}

func (t *mcpTool) Spec() tool.Spec { return t.spec }

// Describe titles a call <server>/<tool>, Pi's label.
func (t *mcpTool) Describe(tool.Call, tool.Env) string {
	return t.conn.server.Name + "/" + t.tool.Name
}

// Run calls the tool within the server's timeout. The server's errors, and
// a timeout, are error results the model reads; only a done ctx is an
// error.
func (t *mcpTool) Run(ctx context.Context, call tool.Call, env tool.Env) (tool.Result, error) {
	label := t.Describe(call, tool.Env{})
	args := call.Args
	if len(args) == 0 {
		args = jsontext.Value("{}")
	}
	if args.Kind() != '{' {
		return tool.BadArgumentsResult(fmt.Sprintf("MCP tool %s: the arguments must be a JSON object", label)), nil
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
	out := preparedResult(res, env)
	if res.IsError && out.Text() == "" {
		out = tool.ErrorResult(fmt.Sprintf("MCP tool %s returned an error", label))
	}
	out.IsError = res.IsError
	return out, nil
}

// preparedResult is a tool result as the model reads it. An image is prepared
// the same way read prepares one, or replaced by the same note (0036-MADR D8).
func preparedResult(r *mcp.CallToolResult, env tool.Env) tool.Result {
	if len(r.Content) == 0 && r.StructuredContent != nil {
		return tool.Result{Output: jsonx.Quote(resultText(r))}
	}
	var parts []string
	var images []tool.Image
	for _, c := range r.Content {
		img, ok := c.(*mcp.ImageContent)
		if !ok {
			parts = append(parts, blockText(c))
			continue
		}
		if !env.Images {
			parts = append(parts, tool.ImageNotSent)
			continue
		}
		prepared, note, err := builtin.PrepareImage(img.Data, img.MIMEType, !env.NoAutoResize)
		if err != nil || len(prepared.Data) == 0 {
			if note == "" {
				note = tool.ImageOmitted
			}
			parts = append(parts, note)
			continue
		}
		images = append(images, prepared)
		if note != "" {
			parts = append(parts, note)
		}
	}
	return tool.Result{Output: jsonx.Quote(strings.Join(parts, "\n")), Images: images}
}

func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', -1, 64)
}

// resultText is a result's text, each block as Pi's toLlmContent and
// tools.ts convert it, joined by newlines. An image here is the text-only
// note; preparedResult carries the image when the model takes one
// (0036-MADR D8).
func resultText(r *mcp.CallToolResult) string {
	if len(r.Content) == 0 && r.StructuredContent != nil {
		b, err := jsonx.Marshal(r.StructuredContent, json.Deterministic(true), jsontext.WithIndent("  "))
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
		return tool.ImageNotSent
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
