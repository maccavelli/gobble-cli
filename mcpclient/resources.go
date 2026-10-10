package mcpclient

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// The Manager is the MCP resource tools' source (0005-PLAN F1d-2,
// deviation 2): builtin.ResourceTools builds the tools over it.
var _ builtin.ResourceSource = (*Manager)(nil)

// ResourceServers are the connected servers that advertised resources at
// initialize, in server order.
func (m *Manager) ResourceServers() []string {
	var out []string
	for _, c := range m.conns {
		if c.offersResources() {
			out = append(out, c.server.Name)
		}
	}
	return out
}

// ResourceTools are the MCP resource tools over m, or none while no
// connected server offers resources.
func (m *Manager) ResourceTools() []tool.Tool {
	if len(m.ResourceServers()) == 0 {
		return nil
	}
	return builtin.ResourceTools(m)
}

// ListResources is one page of server's resources (resources/list).
func (m *Manager) ListResources(ctx context.Context, server, cursor string) ([]builtin.Resource, string, error) {
	cs, ctx, cancel, err := m.resourceSession(ctx, server)
	if err != nil {
		return nil, "", err
	}
	defer cancel()
	res, err := cs.ListResources(ctx, &mcp.ListResourcesParams{Cursor: cursor})
	if err != nil {
		return nil, "", err
	}
	out := make([]builtin.Resource, 0, len(res.Resources))
	for _, r := range res.Resources {
		if r != nil {
			out = append(out, builtin.Resource{URI: r.URI, Name: r.Name, Title: r.Title, Description: r.Description, MIMEType: r.MIMEType, Size: r.Size})
		}
	}
	return out, res.NextCursor, nil
}

// ListResourceTemplates is one page of server's resource templates
// (resources/templates/list).
func (m *Manager) ListResourceTemplates(ctx context.Context, server, cursor string) ([]builtin.ResourceTemplate, string, error) {
	cs, ctx, cancel, err := m.resourceSession(ctx, server)
	if err != nil {
		return nil, "", err
	}
	defer cancel()
	res, err := cs.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{Cursor: cursor})
	if err != nil {
		return nil, "", err
	}
	out := make([]builtin.ResourceTemplate, 0, len(res.ResourceTemplates))
	for _, t := range res.ResourceTemplates {
		if t != nil {
			out = append(out, builtin.ResourceTemplate{URITemplate: t.URITemplate, Name: t.Name, Title: t.Title, Description: t.Description, MIMEType: t.MIMEType})
		}
	}
	return out, res.NextCursor, nil
}

// ReadResource is a resource's contents (resources/read).
func (m *Manager) ReadResource(ctx context.Context, server, uri string) ([]builtin.ResourceContent, error) {
	cs, ctx, cancel, err := m.resourceSession(ctx, server)
	if err != nil {
		return nil, err
	}
	defer cancel()
	res, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		return nil, err
	}
	out := make([]builtin.ResourceContent, 0, len(res.Contents))
	for _, c := range res.Contents {
		if c != nil {
			out = append(out, builtin.ResourceContent{URI: c.URI, MIMEType: c.MIMEType, Text: c.Text, Blob: c.Blob})
		}
	}
	return out, nil
}

// resourceSession is server's open session, if it offers resources, and a
// context bounded by the server's timeout, as a tool call is.
func (m *Manager) resourceSession(ctx context.Context, server string) (*mcp.ClientSession, context.Context, context.CancelFunc, error) {
	for _, c := range m.conns {
		if c.server.Name != server {
			continue
		}
		cs := c.session()
		if cs == nil || !c.offersResources() {
			break
		}
		cctx, cancel := context.WithTimeout(ctx, c.server.Timeout)
		return cs, cctx, cancel, nil
	}
	return nil, nil, nil, fmt.Errorf("MCP server %q is not connected, or offers no resources", server)
}
