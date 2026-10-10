package builtin

import (
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

// tool_search ranks the deferred tools it is given and loads the best, in
// the result's Load; with no match it says so and loads nothing.
func TestToolSearch(t *testing.T) {
	env := tool.Env{Deferred: []tool.Spec{
		{Name: "list_mcp_resources", Description: "List the resources that connected MCP servers offer.\n\n## Use when"},
		{Name: "read_mcp_resource", Description: "Read one resource from a connected MCP server.\n\n## Use when"},
		{Name: "notebook", Description: "Edit a Jupyter notebook's cells."},
	}}
	r := call(t, ToolSearch(), env, map[string]any{"query": "read mcp resource", "limit": 1})
	want := "loaded 1 tool, callable from your next call:\n- read_mcp_resource: Read one resource from a connected MCP server."
	if r.IsError || r.Text() != want || len(r.Load) != 1 || r.Load[0] != "read_mcp_resource" {
		t.Fatalf("search = %+v, text %q", r, r.Text())
	}
	r = call(t, ToolSearch(), env, map[string]any{"query": "resources"})
	if r.IsError || len(r.Load) != 1 || r.Load[0] != "list_mcp_resources" {
		t.Fatalf("search resources = %+v", r)
	}
	r = call(t, ToolSearch(), env, map[string]any{"query": "kubernetes"})
	if r.IsError || r.Text() != "no deferred tool matches kubernetes" || r.Load != nil {
		t.Fatalf("no match = %+v, text %q", r, r.Text())
	}
	if r := call(t, ToolSearch(), env, map[string]any{"query": "  "}); !r.IsError || r.Text() != "tool_search: the query is empty; say what the tool you need does" {
		t.Fatalf("empty query = %q", r.Text())
	}
	if s := ToolSearch().Spec(); s.Name != ToolSearchName || !s.Annotations.ReadOnlyHint || s.Exposure != "" {
		t.Fatalf("spec %+v", s)
	}
}
