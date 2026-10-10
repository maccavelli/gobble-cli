package builtin

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

// stubSource is two servers: docs, with two pages of resources, a
// template, and a text, a two-part and a binary resource; and empty, with
// nothing.
type stubSource struct{}

func (stubSource) ResourceServers() []string { return []string{"docs", "empty"} }

func (stubSource) ListResources(_ context.Context, server, cursor string) ([]Resource, string, error) {
	switch {
	case server == "docs" && cursor == "":
		return []Resource{{URI: "doc://readme", Name: "readme", MIMEType: "text/markdown", Description: "The   project's\nreadme."}}, "page2", nil
	case server == "docs" && cursor == "page2":
		return []Resource{{URI: "doc://notes", Title: "Notes"}}, "", nil
	case server == "docs":
		return nil, "", errors.New("bad cursor")
	}
	return nil, "", nil
}

func (stubSource) ListResourceTemplates(_ context.Context, server, _ string) ([]ResourceTemplate, string, error) {
	if server == "docs" {
		return []ResourceTemplate{{URITemplate: "doc://page/{n}", Name: "page"}}, "", nil
	}
	return nil, "", nil
}

func (stubSource) ReadResource(_ context.Context, server, uri string) ([]ResourceContent, error) {
	switch uri {
	case "doc://readme":
		return []ResourceContent{{URI: uri, MIMEType: "text/markdown", Text: "# Read me"}}, nil
	case "doc://two":
		return []ResourceContent{{URI: "doc://two#1", Text: "one"}, {URI: "doc://two#2", Blob: []byte{1, 2, 3}, MIMEType: "image/png"}}, nil
	case "doc://big":
		return []ResourceContent{{URI: uri, Text: strings.Repeat("é", maxBytes)}}, nil
	}
	return nil, errors.New("not found: " + uri)
}

func resourceTool(name string) tool.Tool {
	for _, tl := range ResourceTools(stubSource{}) {
		if tl.Spec().Name == name {
			return tl
		}
	}
	panic("no resource tool " + name)
}

// The three tools are deferred, read-only and in one family.
func TestResourceToolsSpecs(t *testing.T) {
	var names []string
	for _, tl := range ResourceTools(stubSource{}) {
		s := tl.Spec()
		names = append(names, s.Name)
		if s.Exposure != tool.ExposureDeferred || s.Family != ResourceFamily || !s.Annotations.ReadOnlyHint || s.Kind != tool.KindRead {
			t.Errorf("%s spec %+v; want deferred, read-only, kind read, in family %s", s.Name, s, ResourceFamily)
		}
	}
	if got := strings.Join(names, ","); got != strings.Join(ResourceNames(), ",") {
		t.Fatalf("names %s, want %v", got, ResourceNames())
	}
}

// Listing gives every server's first page, or one server's page from a
// cursor, with the next cursor; a cursor without a server, and an unknown
// server, are refused in the tool's own words.
func TestListResources(t *testing.T) {
	r := call(t, resourceTool(ListResourcesName), tool.Env{}, map[string]string{})
	want := "resources of docs (1):\n- doc://readme: readme (text/markdown) - The project's readme.\n" +
		`[more: call list_mcp_resources with server "docs" and cursor "page2"]` + "\n\nresources of empty: none"
	if r.IsError || r.Text() != want {
		t.Fatalf("list all = %q\nwant %q", r.Text(), want)
	}
	r = call(t, resourceTool(ListResourcesName), tool.Env{}, map[string]string{"server": "docs", "cursor": "page2"})
	if r.IsError || r.Text() != "resources of docs (1):\n- doc://notes: Notes" {
		t.Fatalf("page 2 = %q", r.Text())
	}
	for args, msg := range map[[2]string]string{
		{"", "page2"}:   "list_mcp_resources: a cursor needs the server it came from; send server too",
		{"nope", ""}:    `list_mcp_resources: no connected MCP server named "nope" offers resources; the servers are docs, empty`,
		{"docs", "zzz"}: "list_mcp_resources docs: bad cursor",
	} {
		r := call(t, resourceTool(ListResourcesName), tool.Env{}, map[string]string{"server": args[0], "cursor": args[1]})
		if !r.IsError || r.Text() != msg {
			t.Errorf("list %v = %q, want the error %q", args, r.Text(), msg)
		}
	}
	r = call(t, resourceTool(ListResourceTemplatesName), tool.Env{}, map[string]string{"server": "docs"})
	if r.IsError || r.Text() != "resource templates of docs (1):\n- doc://page/{n}: page" {
		t.Fatalf("templates = %q", r.Text())
	}
}

// Reading gives the text, each part of a several-part resource under its
// uri with a note for binary content, a cut at maxBytes, and refusals in
// the tool's own words.
func TestReadResource(t *testing.T) {
	read := func(server, uri string) tool.Result {
		return call(t, resourceTool(ReadResourceName), tool.Env{}, map[string]string{"server": server, "uri": uri})
	}
	if r := read("docs", "doc://readme"); r.IsError || r.Text() != "# Read me" {
		t.Fatalf("readme = %+v", r)
	}
	if r := read("docs", "doc://two"); r.IsError || r.Text() != "doc://two#1:\none\n\ndoc://two#2:\n[binary content, 3 bytes, image/png: not shown]" {
		t.Fatalf("two parts = %q", r.Text())
	}
	r := read("docs", "doc://big")
	if body, note, _ := strings.Cut(r.Text(), "\n[cut at 50 KB of"); r.IsError || len(body) > maxBytes || !strings.HasPrefix(note, " ") || !strings.HasSuffix(body, "é") {
		t.Fatalf("big = %d bytes, error %v, ends %q", len(r.Text()), r.IsError, r.Text()[len(r.Text())-30:])
	}
	for _, c := range []struct{ server, uri, want string }{
		{"docs", "doc://gone", "read_mcp_resource docs doc://gone: not found: doc://gone"},
		{"nope", "doc://readme", `read_mcp_resource: no connected MCP server named "nope" offers resources; the servers are docs, empty`},
		{" ", "doc://readme", "read_mcp_resource: send both server and uri, as list_mcp_resources gives them"},
	} {
		if r := read(c.server, c.uri); !r.IsError || r.Text() != c.want {
			t.Errorf("read %s %s = %q, want the error %q", c.server, c.uri, r.Text(), c.want)
		}
	}
}
