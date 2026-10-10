List the resources that connected MCP servers offer, such as documents, records or files they expose.

## Use when

- You need data an MCP server holds, and want to know which resources it has.
- Read one with read_mcp_resource, using the server and uri this tool gives.
- Not for files in the workspace: use read, find or tree.

## Returns

- For each server, a block headed by its name, then one line per resource: its uri, its name, its MIME type and its description, when the server gives them.
- A `[more: …]` line with the cursor for the server's next page, when there is one.

## Rules

- Without `server`, each server's first page; with it, that server's page from `cursor`.
- A cursor works only with the server it came from.
