package mcpclient

import (
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// A server that advertises resources is listed, and the three methods
// read its resources, templates and contents; one that does not is not
// listed, and its name is refused (0005-PLAN F1d-2).
func TestResources(t *testing.T) {
	m := connectAll(t, Options{}, httpFixture("plain", mcptest.HTTPServer(t, "")), httpFixture("docs", mcptest.ResourceHTTPServer(t)))
	if got := m.ResourceServers(); !slices.Equal(got, []string{"docs"}) {
		t.Fatalf("resource servers %v, want [docs]", got)
	}
	rs, next, err := m.ListResources(t.Context(), "docs", "")
	if err != nil || next != "" || len(rs) != 1 || rs[0].URI != mcptest.ResourceURI || rs[0].Name != "readme" || rs[0].MIMEType != "text/markdown" {
		t.Fatalf("resources %+v, next %q, %v", rs, next, err)
	}
	ts, _, err := m.ListResourceTemplates(t.Context(), "docs", "")
	if err != nil || len(ts) != 1 || ts[0].URITemplate != mcptest.TemplateURI {
		t.Fatalf("templates %+v, %v", ts, err)
	}
	parts, err := m.ReadResource(t.Context(), "docs", mcptest.ResourceURI)
	if err != nil || len(parts) != 1 || parts[0].Text != mcptest.ResourceText {
		t.Fatalf("read %+v, %v", parts, err)
	}
	if _, _, err := m.ListResources(t.Context(), "plain", ""); err == nil || !strings.Contains(err.Error(), `"plain" is not connected, or offers no resources`) {
		t.Fatalf("a server with no resources: %v", err)
	}
	tools := m.ResourceTools()
	if got := names(tools); !slices.Equal(got, builtin.ResourceNames()) {
		t.Fatalf("resource tools %v", got)
	}
	r := call(t, byName(t, tools, builtin.ReadResourceName), `{"server":"docs","uri":"`+mcptest.ResourceURI+`"}`)
	if r.IsError || r.Text() != mcptest.ResourceText {
		t.Fatalf("read_mcp_resource = %+v", r)
	}
	r = call(t, byName(t, tools, builtin.ListResourcesName), `{}`)
	if r.IsError || !strings.HasPrefix(r.Text(), "resources of docs (1):\n- doc://readme: readme (text/markdown)") {
		t.Fatalf("list_mcp_resources = %q", r.Text())
	}
}

// With no server that advertises resources, there are no resource tools.
func TestNoResourceTools(t *testing.T) {
	m := connectAll(t, Options{}, httpFixture("plain", mcptest.HTTPServer(t, "")))
	if got := m.ResourceServers(); len(got) != 0 {
		t.Fatalf("resource servers %v", got)
	}
	if tools := m.ResourceTools(); tools != nil {
		t.Fatalf("resource tools %v", names(tools))
	}
}
