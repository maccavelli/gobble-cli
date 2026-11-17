# MCP servers

gobble can use the tools of MCP servers, over stdio or streamable HTTP. This page says how to add, list and remove
servers, what each tool is called, how to choose which of their tools a run offers, what gobble changes in a tool's
schema before the model sees it, and what the warning about a schema means. gobble reads Pi's `mcp.json`, so a Pi
server entry works unchanged.

## Adding, listing and removing servers

The servers are the `mcpServers` of `mcp.json` in the config directory, beside `settings.yaml`
(`gobble config path`). `gobble mcp` edits the file and keeps the rest of it:

```bash
gobble mcp add fs -- npx -y @modelcontextprotocol/server-filesystem .          # a stdio server: its command, after --
gobble mcp add docs --url https://example.com/mcp --bearer-token-env-var DOCS_TOKEN   # a streamable HTTP server
gobble mcp list                                                                # connect to each, and show its state and tools
gobble mcp remove fs
```

A stdio server takes `--env KEY=VALUE` and `--cwd DIR`; an HTTP server takes `--header KEY=VALUE`. A value may name a
variable as `$NAME`. A server name is letters, digits, `_` and `-`. `gobble mcp list` exits 1 while a server has
failed or needs sign-in, and `--json` prints the same as JSON.

A session connects its servers in the background when it starts. The first prompt waits for them up to
`mcp.startupWaitMs` (10 seconds); a server still connecting is logged, and its tools join a later prompt.

## What a tool is called

A tool is named `mcp__<server>__<tool>`. A character outside letters, digits, `_` and `-` becomes `_`, so a server's
`dotted.name` is `mcp__fx__dotted_name`. A name over 64 characters, or one another tool already has, is cut and ends in
`_` and eight hex digits of a hash. `gobble mcp list` shows a tool by the name its server gives it; the model, the
tool flags and the permission rules use the `mcp__` name.

## Choosing which tools a run offers

- `-t NAME,…` offers only these tools, built-in and MCP alike. When none of the names is an MCP tool's, no server
  connects.
- `--exclude-tools NAME,…` leaves these out; `--no-tools` leaves out every tool.
- A permission rule that denies a tool with no pattern, such as `deny: mcp__fx__execute_pipeline` or `deny: mcp__fx__*`
  under `permissions.rules` in `settings.yaml`, hides it: it is neither sent to the model nor found by `tool_search`.
- With no rule that allows it, a call of an MCP tool asks first, even a read-only one.

## What gobble changes in a tool's schema

The model is sent each tool's `inputSchema`, with these changes, made on a copy (the server's own is never written to):

1. A schema without `"type": "object"` or without `properties` gets them, `"properties": {}` when it has none, since
   some providers refuse an object schema without properties. A schema that is not a JSON object becomes an empty
   object schema.
2. **A root-level union is given object branches.** Some providers, Grok among them, refuse a tool whose schema is a
   union at its root (`anyOf` or `oneOf`) unless every branch is itself an object schema, and Grok's own client gives
   each such branch its own `properties`; a branch that only says which properties are required may leave its
   arguments unconstrained there. gobble gives each branch `"type": "object"` and the root's `properties`,
   `patternProperties` and `additionalProperties`, when the branch has none of its own, with the descriptions left
   out of the copies. A branch that is `false`, or whose type cannot be an object, is removed. When the rule applies,
   the rewritten schema accepts exactly the arguments the original accepts
   ([0060-MADR](../decisions/0060-MADR-normalise-root-level-unions-in-mcp-tool-schemas.md), D2 and its amendment of
   2026-11-17): it states the same contract in the form Grok takes, and it is larger, since each branch repeats the
   properties.

The rule applies only when the root is typed exactly `object` (a root with no type is given `object` first, by the
first change above, so it is covered), has one union, holds besides it only `type`, `properties`, `required`,
`additionalProperties`, `patternProperties` and annotations such as `description`, and neither the union nor the
properties it copies hold, at any depth, a reference (`$ref`, `$dynamicRef`, `$recursiveRef`), an id or an anchor
(`$id`, `$anchor`, `$dynamicAnchor`, `$recursiveAnchor`), `unevaluatedProperties` or `unevaluatedItems`: copying
properties into a branch, or removing a branch, can change what they mean. A property that is merely called `$ref` or
`$id` is not one of them. A schema with one of them is sent as given and warned of, as below, even when it is
harmless, such as a `$ref` to `#`. A root's own keywords, descriptions included, are left as the server wrote them,
and a nested union inside a branch is not touched.

An example is the `execute_pipeline` tool of the magictools server, whose schema is an object with seven properties and

```json
"anyOf": [{"required": ["target", "intent"]}, {"required": ["session_id"]}]
```

gobble sends the same root, with each of the two branches typed `"object"` and carrying the seven properties, `"additionalProperties":
false` and its own `required`. A call that gave `session_id` alone, or `target` and `intent`, is as valid as before, and a call that
gave neither is as invalid. With `--log-level debug`, the log file (`logs/gobble.log` in the state directory) holds one
record, `mcp: tool schema rewritten`, naming the server and the tool, for each tool rewritten in a session.

## The warning: a schema gobble could not rewrite

A schema outside the rule is sent as the server gave it, and nothing is dropped. When its root union has a branch that
is not an object schema typed `object`, some providers refuse the request, and then every prompt on that model fails
until the tool is left out, since gobble sends every tool each time. gobble says so once per session for each such tool,
as a warning in the log, which a terminal prints as a `Warning:` line (here wrapped at 80 columns, as gobble's terminal
handler does), and as a notice to an ACP client that offered notices. A terminal prints:

```text
Warning: mcp: tool schema sent as the server gave it (server: fx; tool:
  pick_one; reason: its root anyOf has a branch that is not an object, which
  some providers refuse; left as given because the root type is not exactly
  "object"; remedy: leave the tool out with --exclude-tools mcp__fx__pick_one,
  or add a permission rule that denies it with no pattern)
```

and the notice, titled `mcp: tool schema sent as the server gave it`, says:

```text
fx/pick_one: its root anyOf has a branch that is not an object, which some providers refuse; left as given because the root type is not exactly "object". To avoid a refusal, leave the tool out with --exclude-tools mcp__fx__pick_one, or add a permission rule that denies it with no pattern.
```

The text after "because" says why the rule did not apply: the root type is not exactly `object`, the root has both
`anyOf` and `oneOf`, the root has a keyword the rule does not cover (named), the union or the properties it would copy
hold a reference, an id, an anchor or an `unevaluated` keyword (named), the union is not a list, a branch uses `$ref`,
a branch is not a schema, a branch's type is not a type name or a list of them, or no branch can match an object.
There are three ways to stop the warning and the refusal:

- Leave the tool out of the run with `--exclude-tools mcp__<server>__<tool>`, or `-t` without it.
- Deny it with a permission rule (`deny: mcp__<server>__<tool>` under `permissions.rules`). A tool a rule hides is not
  sent, so it is not warned of either; if the rule is lifted during the session, the warning comes once, with the
  first prompt that sends the tool.
- Ask the server's author to type each branch of the union `"type": "object"`.

A provider that does not refuse the shape takes the tool as the server gave it, and the warning is then only
information.
