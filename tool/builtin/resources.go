package builtin

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/tool"
)

// The MCP resource tools' names and family (0012-MADR D7, D8).
const (
	ListResourcesName         = "list_mcp_resources"
	ListResourceTemplatesName = "list_mcp_resource_templates"
	ReadResourceName          = "read_mcp_resource"
	ResourceFamily            = "mcp-resources"
)

// ResourceNames are the resource tools' names, in the order ResourceTools
// builds them.
func ResourceNames() []string {
	return []string{ListResourcesName, ListResourceTemplatesName, ReadResourceName}
}

// ResourceSource is a session's MCP servers' resources, for the resource
// tools (0005-PLAN F1d-2, deviation 2). mcpclient's Manager implements it.
type ResourceSource interface {
	// ResourceServers are the connected servers that offer resources, in
	// server order.
	ResourceServers() []string
	// ListResources is a page of a server's resources, from cursor ("" for
	// the first), and the next page's cursor ("" when there is none).
	ListResources(ctx context.Context, server, cursor string) ([]Resource, string, error)
	// ListResourceTemplates is a page of a server's resource templates.
	ListResourceTemplates(ctx context.Context, server, cursor string) ([]ResourceTemplate, string, error)
	// ReadResource is a resource's contents.
	ReadResource(ctx context.Context, server, uri string) ([]ResourceContent, error)
}

// Resource is one resource a server lists.
type Resource struct {
	URI, Name, Title, Description, MIMEType string
	Size                                    int64
}

// ResourceTemplate is one parameterized resource a server lists.
type ResourceTemplate struct {
	URITemplate, Name, Title, Description, MIMEType string
}

// ResourceContent is one part of a resource as read: text, or a blob.
type ResourceContent struct {
	URI, MIMEType, Text string
	Blob                []byte
}

type resourceListIn struct {
	Server string `json:"server,omitzero" jsonschema:"the MCP server to list, by its name; every server's first page when not given"`
	Cursor string `json:"cursor,omitzero" jsonschema:"the cursor a previous call returned, for the server's next page; needs server"`
}

type resourceReadIn struct {
	Server string `json:"server" jsonschema:"the MCP server that lists the resource, by its name"`
	URI    string `json:"uri" jsonschema:"the resource's uri, as list_mcp_resources gives it"`
}

// ResourceTools are the three MCP resource tools over src, deferred in
// family ResourceFamily. A session offers them only while a connected
// server offers resources.
func ResourceTools(src ResourceSource) []tool.Tool {
	common := []tool.Option{
		tool.WithKind(tool.KindRead), tool.WithExposure(tool.ExposureDeferred), tool.WithFamily(ResourceFamily),
		tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: true}),
	}
	o := Options{}.withDefaults()
	listResources := func(ctx context.Context, in resourceListIn, _ tool.Env) (string, error) {
		return listPages(ctx, src, ListResourcesName, "resources", in, func(ctx context.Context, server, cursor string) ([]string, string, error) {
			rs, next, err := src.ListResources(ctx, server, cursor)
			lines := make([]string, len(rs))
			for i, r := range rs {
				lines[i] = entryLine(r.URI, r.Name, r.Title, r.MIMEType, r.Description)
			}
			return lines, next, err
		})
	}
	listTemplates := func(ctx context.Context, in resourceListIn, _ tool.Env) (string, error) {
		return listPages(ctx, src, ListResourceTemplatesName, "resource templates", in, func(ctx context.Context, server, cursor string) ([]string, string, error) {
			ts, next, err := src.ListResourceTemplates(ctx, server, cursor)
			lines := make([]string, len(ts))
			for i, rt := range ts {
				lines[i] = entryLine(rt.URITemplate, rt.Name, rt.Title, rt.MIMEType, rt.Description)
			}
			return lines, next, err
		})
	}
	read := func(ctx context.Context, in resourceReadIn, _ tool.Env) (string, error) {
		return readResource(ctx, src, in)
	}
	describeList := func(c tool.Call, _ tool.Env) string {
		if s := argString(c, "server"); s != "" {
			return title("list the resources of " + s)
		}
		return "list the MCP servers' resources"
	}
	return []tool.Tool{
		tool.New(ListResourcesName, description(ListResourcesName, o), listResources,
			append(slices.Clone(common), tool.WithDescribe(describeList))...),
		tool.New(ListResourceTemplatesName, description(ListResourceTemplatesName, o), listTemplates,
			append(slices.Clone(common), tool.WithDescribe(describeList))...),
		tool.New(ReadResourceName, description(ReadResourceName, o), read,
			append(slices.Clone(common),
				tool.WithSchema(func(s *jsonschema.Schema) {
					nonEmpty(s, "server")
					nonEmpty(s, "uri")
				}),
				tool.WithDescribe(func(c tool.Call, _ tool.Env) string { return title("read " + argString(c, "uri")) }))...),
	}
}

// listPages lists one server's page, or every server's first page when no
// server is named, as "<what> of <server>:" blocks.
func listPages(ctx context.Context, src ResourceSource, toolName, what string, in resourceListIn,
	page func(ctx context.Context, server, cursor string) ([]string, string, error)) (string, error) {
	servers := src.ResourceServers()
	server := strings.TrimSpace(in.Server)
	if server == "" && in.Cursor != "" {
		return "", fmt.Errorf("%s: a cursor needs the server it came from; send server too", toolName)
	}
	if server != "" {
		if !slices.Contains(servers, server) {
			return "", fmt.Errorf("%s: no connected MCP server named %q offers resources; the servers are %s", toolName, server, serverList(servers))
		}
		servers = []string{server}
	}
	var blocks []string
	for _, s := range servers {
		lines, next, err := page(ctx, s, in.Cursor)
		if err != nil {
			return "", fmt.Errorf("%s %s: %w", toolName, s, err)
		}
		var block strings.Builder
		if len(lines) == 0 {
			fmt.Fprintf(&block, "%s of %s: none", what, s)
		} else {
			fmt.Fprintf(&block, "%s of %s (%d):", what, s, len(lines))
		}
		for _, ln := range lines {
			block.WriteString("\n- " + ln)
		}
		if next != "" {
			fmt.Fprintf(&block, "\n[more: call %s with server %q and cursor %q]", toolName, s, next)
		}
		blocks = append(blocks, block.String())
	}
	if len(blocks) == 0 {
		return "no connected MCP server offers resources", nil
	}
	return strings.Join(blocks, "\n\n"), nil
}

// entryLine is one listed resource or template: its uri, then its name or
// title, its MIME type, and its description, when set.
func entryLine(uri, name, title, mime, desc string) string {
	line := uri
	if label := cmp.Or(title, name); label != "" && label != uri {
		line += ": " + label
	}
	if mime != "" {
		line += " (" + mime + ")"
	}
	if desc = strings.Join(strings.Fields(desc), " "); desc != "" {
		line += " - " + desc
	}
	return line
}

// readResource is a resource's text parts, each headed by its uri when
// there are several, and a note for each blob; at most maxBytes in all.
func readResource(ctx context.Context, src ResourceSource, in resourceReadIn) (string, error) {
	server, uri := strings.TrimSpace(in.Server), strings.TrimSpace(in.URI)
	if server == "" || uri == "" {
		return "", errors.New("read_mcp_resource: send both server and uri, as list_mcp_resources gives them")
	}
	if !slices.Contains(src.ResourceServers(), server) {
		return "", fmt.Errorf("read_mcp_resource: no connected MCP server named %q offers resources; the servers are %s", server, serverList(src.ResourceServers()))
	}
	parts, err := src.ReadResource(ctx, server, uri)
	if err != nil {
		return "", fmt.Errorf("read_mcp_resource %s %s: %w", server, uri, err)
	}
	if len(parts) == 0 {
		return uri + " has no contents", nil
	}
	var b strings.Builder
	for i, p := range parts {
		if len(parts) > 1 {
			if i > 0 {
				b.WriteString("\n\n")
			}
			fmt.Fprintf(&b, "%s:\n", cmp.Or(p.URI, uri))
		}
		if len(p.Blob) > 0 {
			fmt.Fprintf(&b, "[binary content, %d bytes, %s: not shown]", len(p.Blob), cmp.Or(p.MIMEType, "of unknown type"))
			continue
		}
		b.WriteString(p.Text)
	}
	out := b.String()
	if len(out) > maxBytes {
		cut := maxBytes
		for cut > 0 && !utf8.RuneStart(out[cut]) {
			cut--
		}
		out = out[:cut] + fmt.Sprintf("\n[cut at %d KB of %d bytes]", maxBytes/1024, len(out))
	}
	return out, nil
}

func serverList(servers []string) string {
	if len(servers) == 0 {
		return "none"
	}
	return strings.Join(servers, ", ")
}
