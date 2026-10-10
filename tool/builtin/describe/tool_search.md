Find and load tools that are not offered yet, by describing what you need.

## Use when

- You need a capability none of your tools has, such as an MCP server's resources.
- You know a tool's exact name and it is not in your tools: search for the name.
- Not for searching files or their contents: use grep or find.

## Returns

- A line such as `loaded 2 tools, callable from your next call:`, then one line per tool, with its name and first sentence.
- `no deferred tool matches` and your query, when nothing matches: try other words.

## Rules

- The tools it loads are callable from your next call, not in this one.
- A tool loads with its family, so related tools arrive together.
