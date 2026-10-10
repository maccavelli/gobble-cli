package builtin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/toolsearch"
)

// ToolSearchName is tool_search's name.
const ToolSearchName = "tool_search"

const (
	defaultSearchLimit = 5
	maxSearchLimit     = 20
)

type searchIn struct {
	Query string `json:"query" jsonschema:"words for the tool you need, such as read mcp resource, or a tool's exact name"`
	Limit int    `json:"limit,omitzero" jsonschema:"the most tools to load"`
}

// ToolSearch is tool_search: it ranks the deferred tools the agent gives it
// (tool.Env.Deferred) with toolsearch's BM25, and loads the best, which are
// callable from the next model call (0005-PLAN F1d-2). The agent offers it
// only while a tool is deferred.
func ToolSearch() tool.Tool {
	return tool.New(ToolSearchName, description(ToolSearchName, Options{}.withDefaults()),
		runToolSearch,
		tool.WithKind(tool.KindSearch),
		tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true, IdempotentHint: true}),
		tool.WithSchema(func(s *jsonschema.Schema) {
			nonEmpty(s, "query")
			bound(s, "limit", 1, maxSearchLimit, defaultSearchLimit)
		}),
		tool.WithDescribe(func(c tool.Call, _ tool.Env) string {
			return title("search tools for " + argString(c, "query"))
		}),
	)
}

func runToolSearch(_ context.Context, in searchIn, env tool.Env) (tool.Result, error) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return tool.Result{}, errors.New("tool_search: the query is empty; say what the tool you need does")
	}
	found := toolsearch.New(env.Deferred).Search(query, clampInt(in.Limit, 1, maxSearchLimit, defaultSearchLimit))
	if len(found) == 0 {
		return tool.TextResult("no deferred tool matches " + query), nil
	}
	noun := "tools"
	if len(found) == 1 {
		noun = "tool"
	}
	lines := []string{fmt.Sprintf("loaded %d %s, callable from your next call:", len(found), noun)}
	names := make([]string, len(found))
	for i, s := range found {
		names[i] = s.Name
		first, _, _ := strings.Cut(strings.TrimSpace(s.Description), "\n")
		lines = append(lines, "- "+s.Name+": "+first)
	}
	r := tool.TextResult(strings.Join(lines, "\n"))
	r.Load = names
	return r, nil
}
