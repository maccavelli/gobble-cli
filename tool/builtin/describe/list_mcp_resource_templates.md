List the resource templates connected MCP servers offer: uri patterns with parameters, for resources not listed one by one.

## Use when

- list_mcp_resources does not list what you need, and the server may build it from a uri pattern.
- Fill in a template's parameters, then read the result with read_mcp_resource.
- Not for files in the workspace: use read, find or tree.

## Returns

- For each server, a block headed by its name, then one line per template: its uri pattern, its name, its MIME type and its description, when the server gives them.
- A `[more: …]` line with the cursor for the server's next page, when there is one.

## Rules

- Without `server`, each server's first page; with it, that server's page from `cursor`.
- A cursor works only with the server it came from.
