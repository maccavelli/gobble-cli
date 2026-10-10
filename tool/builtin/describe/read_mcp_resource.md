Read one resource from a connected MCP server, by the server's name and the resource's uri.

## Use when

- list_mcp_resources or list_mcp_resource_templates gave you the server and uri you need.
- The data is held by an MCP server rather than in the workspace.
- Not for files in the workspace: use read.

## Returns

- The resource's text. A resource in several parts gives each part under its own uri.
- A note in place of binary content, with its size and MIME type.

## Rules

- At most {{.MaxKB}} KB; a longer resource is cut, with a note saying so.
- Send `server` and `uri` exactly as a list gave them.
