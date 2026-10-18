package mcpclient

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/gobble-cli/tool"
)

// An MCP tool's arguments that are not a JSON object are refused before
// any request, marked BadArguments; a tool whose server is gone is an
// error of another kind (0005-PLAN F5-3 decision 3).
func TestBadArguments(t *testing.T) {
	tl := newTool(&conn{server: Server{Name: "s"}}, &mcp.Tool{Name: "t"}, "mcp__s__t")
	r, err := tl.Run(t.Context(), tool.Call{Name: "mcp__s__t", Args: jsontext.Value(`[1]`)}, tool.Env{})
	if err != nil || !r.IsError || !r.BadArguments {
		t.Fatalf("an array: %+v (%q), %v; want an error result marked BadArguments", r, r.Text(), err)
	}
	r, err = tl.Run(t.Context(), tool.Call{Name: "mcp__s__t", Args: jsontext.Value(`{}`)}, tool.Env{})
	if err != nil || !r.IsError || r.BadArguments {
		t.Fatalf("no server: %+v (%q), %v; want an error result not marked BadArguments", r, r.Text(), err)
	}
}
