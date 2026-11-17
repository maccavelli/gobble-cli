---
status: in-progress
date: 2026-11-17
associated-madr: "0060-MADR-normalise-root-level-unions-in-mcp-tool-schemas.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement giving each branch of an MCP tool's root-level union the root's object schema at load

Associated MADR: [0060-MADR-normalise-root-level-unions-in-mcp-tool-schemas.md](0060-MADR-normalise-root-level-unions-in-mcp-tool-schemas.md)

## Goal

A session with an MCP server whose tool has a root-level union, such as magictools' `execute_pipeline`, works on
`grok-4.6`: the MADR's D2 rewrites the schema at load, the rewritten schema accepts exactly the arguments the server's
schema accepts, a schema D2 cannot rewrite is sent as given and flagged once (D3, D4), and the docs say so. The plan is
complete when the MADR's Confirmation 1 to 5 hold.

Approved 2026-11-17 through P3, the owner's live step, with the label "Approve through my live step too, then ask about
`main`", relayed by the coordinator; the owner answered the MADR's Q1 to Q7 with the recommended option each, the
answers the phases were written for. After P3's commands pass the plan closes itself, and the coordinator asks the owner
only for the `main` fast-forward.

## Scope

**Base.** The branch `p3-acpext-v0.5.0`, tip `2004980` when drafted. On 2026-11-17 `main` was fast-forwarded to
`2004980` and the branch's earlier pull request was merged. By the owner's ruling on Q6 as re-put, this plan runs next,
before 0052-PLAN P5 (then 0052 P6 and 0051-PLAN P8 and P8a), on the same branch, with a fresh draft pull request to
`main` so CI runs on every push. `main` moves only on the owner's word at a phase boundary; the work ships in `v0.7.0`.

**In scope:**

* this pair and its two `docs/README.md` rows (P0);
* `mcpclient`: the rule (a new `mcpclient/schema.go`), `inputSchema` calling it on a deep copy (`mcpclient/tool.go`),
  and the flagged tools a `Manager` reports (P1);
* `acpserver/mcp.go`: one warning and notice per session per flagged tool, beside the existing MCP diagnostics (P1);
* `mcpclient/mcptest`: a fixture server whose tool carries the real schema, registered through the go-sdk's low-level
  `Server.AddTool` so the schema is sent as written (P1);
* tests in `mcpclient`, `acpserver` and `llm/provider`, and a `live_grok` test (P1);
* docs in the same commit (P1): `docs/architecture.md`'s `mcpclient` row, `mcpclient/doc.go` and the doc comments of
  every changed function, a new `docs/guides/mcp-servers.md` (no MCP guide exists; the docs tree's guide list in
  `docs/README.md` gains it), and a `docs/README.md` "I want to…" row;
* the independent review (P2) and the owner's live step and close-out (P3).

**Out of scope:** go-llmprovider-sdk (its draft's D5); `mcp-server-magictools` (Q5); the typed-error follow-up (Q4 (a),
a record of its own); the `signature` misfire (Q7 (a), that record); any pin change. 0042-MADR stands: `acpext` by tag,
no `replace`, no committed `go.work`.

## Conventions

* **K1. Start state.** Each phase records `git status`, `HEAD`, the branch tip, `origin/p3-acpext-v0.5.0`, and `go
  version`. A difference from this plan is a deviation.
* **K2. Scratch only for plants.** Every planted break is made in a copy under `<scratch>`: `git archive HEAD` extracted
  there, with the phase's uncommitted diff applied by `git diff | git -C <copy> apply` (or the files copied), never in
  the working tree. Each planted file is `gofmt` clean.
* **K3. Tests first.** Each new test is run against the code before the change and seen to fail, its output kept; then
  the change; then green.
* **K4. Evidence.** Full output to a `<scratch>` file and the exit status taken before any filter; counts by a script
  that reads every file in full.
* **K5. Heavy runs.** `make preflight`, `make race` and any `-race` run go through
  the fleet's `run-heavy -- <command>` on this Mac or its `run-on-host` elsewhere,
  one at a time per host; `run-on-host --race` for a race run, on a host that is not a WSL distro
  (0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md); a multi-round race run carries an explicit
  `-timeout`.
* **K6. Deviations stop the phase,** recorded here with a date, and in the MADR when a decision or a fact changes.
* **K7. Identifiers.** No hostname, account name or real-machine path in a committed file; the copied schema names its
  source by repository, path and commit.
* **K8. No real credential** reaches any agent: the live step is the owner's, in the owner's shell.

## Implementation Steps

### P0 — the records

Model class and effort: lightweight, low.

Files: the MADR, this file, `docs/README.md`.

1. At a phase boundary (K1), confirm `python3 scripts/check_records.py --next` printed `0060` when the pair was drafted
   and that no other `0060` record exists; place the pair if drafted elsewhere, with its two README rows after the 0059
   rows.
2. Gates: `python3 scripts/check_records.py --check-all`, `make check-records markdownlint`, an identifier scan of the
   diff. The commit holds only the pair and `docs/README.md` (AGENTS.md's bootstrap exception); it is docs-only, so
   `docs.yml` checks it (0026-MADR-skip-full-ci-for-docs-only-pushes.md).
3. On the owner's acceptance: the MADR `accepted`, dated, each of Q1 to Q7 answered with the owner's label quoted; this
   plan `in-progress`; any phase an answer changes amended before it runs; the README rows' status columns to match.
   *Done 2026-11-17, before the first commit:* the owner accepted the pair the day it was drafted, so it is committed
   once, already `accepted`, with the answers. Every answer is the option the phases were written for, so no phase
   changed; P1 step 10 and Rollout now name the fresh draft pull request (Q6 as re-put).
4. Commit, by the coordinator, with `git commit --no-edit`, the pair and `docs/README.md` staged by path. The push to
   `p3-acpext-v0.5.0` and the fresh draft pull request to `main` follow the owner's word on the branch.

**Acceptance.** The gates exit 0; the pair is on the branch, `accepted`, with the answers.

### P1 — the rule, its tests first, the warning, and the docs

Model class and effort: mainstream, high.

Files: `mcpclient/schema.go` (new), `mcpclient/schema_test.go` (new), `mcpclient/testdata/` (new: the real schema and
the expected rewrite), `mcpclient/tool.go`, `mcpclient/manager.go`, `mcpclient/manager_test.go`,
`mcpclient/mcptest/mcptest.go`, `mcpclient/doc.go`, `acpserver/mcp.go`, `acpserver/mcp_test.go`,
`llm/provider/wire_test.go` (or a new `_test.go` beside it), `llm/provider/live_grok_union_test.go` (new,
`//go:build live_grok`), `docs/architecture.md`, `docs/guides/mcp-servers.md` (new), `docs/README.md`, this file.

1. **Start state** (K1).
2. **The data.** Copy `execute_pipeline`'s `inputSchema` verbatim from `mcp-server-magictools`
   `internal/handler/inventory.go:289-336` at `368ac6a` into `mcpclient/testdata/magictools-execute-pipeline.json`,
   with a sibling note naming the repository, path and commit. Write the expected rewrite
   `magictools-execute-pipeline.rewritten.json` by hand from the MADR's D2 (each branch: `"type": "object"`, the seven
   properties without descriptions, `additionalProperties: false`, its own `required`). Add S2 to S12 from the MADR's
   table as Go test cases, each with its expected result: rewritten (with its expected JSON), or refused (with its
   reason).
3. **Tests first (K3).** Written in the tree and run with `mcpclient/schema.go` holding only a stub of the rule that
   returns its input, so each test compiles and fails for its own reason, the output kept:
   * `mcpclient/schema_test.go`:
     * **the rule:** for each case, the rule's output equals the expected JSON, or the case is refused with its reason
       and the schema is returned unchanged;
     * **input preservation:** for each rewritten case, a corpus generated in the test (every subset of the property
       names plus one unknown name with a typed value; 2,000 objects from a fixed seed; nine non-objects) gets the same
       verdict from `github.com/google/jsonschema-go` against the original and the rewritten schema;
     * **the precondition Grok states:** every branch of every rewritten schema has `"type": "object"`, and each branch
       that had no object keywords now carries the root's;
     * **the root is untouched:** the rewritten root's keywords other than the union equal the original's,
       descriptions included;
     * **idempotence:** the rule applied twice equals the rule applied once;
     * **no mutation:** the go-sdk `mcp.Tool`'s `InputSchema` map is deep-equal before and after `newTool`.
   * `mcpclient/manager_test.go`: a `Manager` connected to the new fixture reports the flagged tools, each once, with
     server, tool and reason, and none for a rewritten tool.
   * `acpserver/mcp_test.go`, the end-to-end test: an in-process MCP server from the new `mcptest` fixture offering the
     real schema and one schema outside D2 (S8's shape), an `llm.Provider` wrapper over `llmtest.Script` that refuses
     any request whose tools carry a root union with a branch not typed `object`, with the error the SDK makes of xAI's
     recorded body (`llmprovider.ClassifyHTTPError("grok", …)` on a status-400 response whose body is
     `{"code":"invalid-argument","error":"Failed to start sampling: [invalid_client_tool_schema] <tool>: tool parameter
     root must be an object type (root schema is an anyOf/oneOf union with a non-object branch)"}`, mapped as gobble's
     provider maps it), and no outside request or real credential. With the S8 tool excluded by the session's tool
     filter, the prompt fails before the change with the owner's paste's text (the MADR's P1) and, after it, completes; the request's schema for
     `mcp__fx__execute_pipeline` equals the expected rewrite; a scripted call of that tool reaches the fixture server
     with its arguments intact. With the S8 tool included, two prompts give exactly one warning record and one notice
     naming it.
   * `llm/provider`, the wire: gobble's `grok` provider, through `New`, against a fake transport that answers 400 in
     xAI's recorded body when `tools[].parameters` holds a root union with a branch not typed `object`, and the
     existing canned text reply otherwise: the original schema is refused with the MADR's P1 text after one request; the
     expected rewrite is sent byte-identical to the spec and answered.
   * `internal/cli`: if its MCP test harness can start a session with a stdio fixture (read `internal/cli/mcp_test.go`
     first), assert the flagged tool's `Warning:` line on stderr; if it cannot, record why here, and the warning is
     covered by the end-to-end test's log check.
4. **The change.** `mcpclient/schema.go` holds the rule exactly as the MADR's D2 states it, on a deep copy, returning the
   schema and, when refused in a shape D3 flags, a reason. `inputSchema` applies gobble's existing additions on the copy,
   then the rule. The `Manager` keeps each flagged tool's server, tool and reason; `acpserver`'s MCP diagnostics
   report each once per session (`l.reported`, keyed by tool name), as a warning record and a notice with the MADR's
   D4 title and text; a rewritten tool gets one debug record. The fixture server is added to `mcptest` without changing
   `NewServer`'s tools.
5. **Green.** Every test of step 3 passes; `go test ./mcpclient/... ./acpserver/ ./llm/provider/` on this Mac.
6. **Planted breaks (K2),** each on its own scratch copy, each expected to turn the named tests red and seen to:
   1. the rule returns its input: the rule's cases, the wire's rewrite case, and the end-to-end prompt;
   2. branches typed but the root's keywords not copied: the rule's cases and the precondition test;
   3. the union dropped: the preservation test (S0 and S2 at least);
   4. a branch's own `properties` overwritten: the preservation test (S7);
   5. the rule applied to a root typed `["object","null"]`: the S8 case (refused expected) and its preservation;
   6. the go-sdk's map written in place: the no-mutation test;
   7. the warning sent on every prompt: the end-to-end test's once-per-session count;
   8. descriptions removed from the root's properties too: the root-untouched test.
   Record each break's failing test name and its first failure line.
7. **The live test (owner-run in P3).** `llm/provider/live_grok_union_test.go`, `//go:build live_grok`, reading
   `XAI_API_KEY` through `llmtest.LiveCredential` and redacting it from every failure as `liveTurn` does (G18 of the
   MADR): `TestLiveGrokUnionSchema` sends one short prompt to `grok-4.6` twice, once with the original schema as the
   only tool and once with the expected rewrite. It fails when the rewrite is refused; it logs, without failing, whether
   the original was refused, and prints statuses only. It is built by `go vet -tags live_grok ./llm/provider/` here,
   never run by an agent.
8. **Docs, in this commit.** Each sentence checked against the code before it is kept:
   * `docs/guides/mcp-servers.md`: adding, listing and removing servers (`gobble mcp add|remove|list`), tool names
     (`mcp__<server>__<tool>`, the 64-character cut), choosing tools (`-t`, `--exclude-tools`, a deny rule), what gobble
     changes in a tool's schema (the existing additions and D2, with the magictools example), and what the warning
     means and what to do;
   * `docs/README.md`: the guide in the guides list; an "I want to…" row, "know why gobble changes an MCP tool's schema,
     or warns about one" → the guide and this MADR;
   * `docs/architecture.md`, the `mcpclient` row: one sentence on the rule and the flag;
   * `mcpclient/doc.go`, and the doc comments of `inputSchema` and every new or changed function.
   `AGENTS.md`: read; it does not describe MCP schemas, so no change is expected; record that.
9. **Gates,** after the last file change:
   * `make pre-add-check FILES="<every changed .go file>"`;
   * `make preflight` on this Mac through `run-heavy`: exit 0, "preflight passed";
   * `make preflight` on the Linux machine through `run-on-host`: exit 0;
   * `make race` through `run-on-host --race` on a host that is not a WSL distro: exit 0;
   * the end-to-end test repeated under the race detector, through `run-on-host --race`:
     `go test -race -count=20 -timeout 15m -run 'TestMCPUnionSchema' ./acpserver/`: exit 0;
   * `make check-test-isolation` through `run-on-host`: exit 0;
   * `python3 scripts/check_records.py --check-all`, `make check-records markdownlint`: exit 0;
   * an identifier scan of the diff: nothing.
10. **Commit.** The hooks path is the global hooks directory or a wrapper that chains to it; one commit holding the
    rule, its tests, the warning and the docs, made by the coordinator with `git commit --no-edit` on
    `p3-acpext-v0.5.0`. The push runs CI on the branch's fresh draft pull request to `main` (Q6 as re-put); `main` does
    not move.

**Acceptance.** Step 3's tests seen red before the change and green after; the eight plants each red as named; every
gate exits 0; the commit on the branch; CI on the fresh draft pull request green on the three `validate` runners.

### P2 — independent review of what the model is shown

Model class and effort: strongest, high.

The reviewer gets the MADR's D2, the diff of P1, the two testdata files and the shapes, and not P1's conclusions. It
checks: that the code implements D2 and nothing more; the JSON Schema argument of the MADR's Context, step by step,
against the code; that no shape outside D2 is rewritten; that the expected rewrite is what xAI's guide and Grok's own
client show (the MADR's X1 and K4); that the warning names the remedy correctly; and that no test can pass with the rule
disabled. Findings are fixed in the tree under a dated deviation entry and the P1 gates rerun, or put to the owner.

**Acceptance.** The review's findings recorded here, each fixed or answered by the owner.

### P3 — the owner's live step, and close-out

Model class and effort: lightweight, low (the record); the commands are the owner's (K8).

At the commit of P1 (and P2's fixes), in the owner's shell, with `XAI_API_KEY` set from the owner's own store:

1. **Build:** `make bin/gobble`. Expect exit 0.
2. **The server is there:** `bin/gobble mcp list`. Expect the `magictools` entry, enabled.
3. **The API:** `go test -tags live_grok -run '^TestLiveGrokUnionSchema$' -count=1 -v ./llm/provider/`. Expect `PASS`,
   with a log line saying whether the original schema was refused (expected: refused, 400, naming the tool) and the
   rewrite answered.
4. **The session:** `bin/gobble -p --model grok/grok-4.6 "Do not call any tool. List, one per line, the names of your
   tools that start with mcp__magictools__."` Expect exit 0, a list that includes
   `mcp__magictools__execute_pipeline`, and no `invalid_client_tool_schema` anywhere.
5. **The tool still works:** `bin/gobble --model grok/grok-4.6`, then the prompt `Call
   mcp__magictools__execute_pipeline once with dry_run true, target set to the current directory, intent "preview only"
   and session_id "probe-0060". Then tell me exactly what it returned.` Approve the call once. Expect one call shown as
   `magictools/execute_pipeline`, its result text (the pipeline's preview, or the server's own refusal message), and a
   normal end of turn.
6. **Optional, the other providers the owner uses:** step 4 with `--model` set to each (for example a `claude/`,
   `openai/` and `gemini/` model). Expect exit 0 each. A failure is compared with the same command on a build of
   `2004980` (`git worktree add <dir> 2004980`, then `make bin/gobble` there).

What each outcome decides:

* Steps 3 to 5 pass: the MADR's Confirmation 5 is met; this plan `complete`, the MADR's Confirmation marked, both
  README rows updated, gates as P0 step 2 (the owner's approval covers this close-out). The coordinator then asks the
  owner only for the `main` fast-forward.
* **Stop condition, step 3's rewrite refused** (a 400 naming the tool): Grok does not accept D2's form. The plan stops;
  the MADR is amended with the refusal's text; the owner chooses among a revert of P1 (`git revert`, a new commit), a
  Grok-only degrade of such schemas to an open object with the schema in the description (option G, for that route), or
  Q1 (c)'s typed-only form, which the live test is then extended to try first. The choice among the three is the
  owner's, made then; the approval of 2026-11-17 does not make it. Nothing is released until then.
* Step 3 passes and step 4 fails: the failure's text is recorded. A refusal naming another tool is another schema; if
  gobble flagged it, the warning is working and the server is the owner's; if not, a deviation and a new question.
* Step 5 has no call or no result: a deviation, read before anything is rerun.
* Step 6 fails on a provider whose step 4 passes at `2004980`'s build: that route refuses D2's form; stop, amend the MADR,
  and Q1 comes back to the owner, as the owner's ruling on Q1 provides.

**Acceptance.** The owner's outputs recorded here (statuses and the lines named above, no key); the plan `complete`, or
stopped with the owner's choice recorded.

## Verification

| Check | Command or step | Expect | MADR Confirmation |
| :--- | :--- | :--- | :--- |
| the rule and its cases | `go test ./mcpclient/` | pass; red before the change | 1 |
| input preservation | the corpus test in `mcpclient` | equal verdicts on every rewritten case | 1 |
| the end-to-end prompt | `go test ./acpserver/ -run TestMCPUnionSchema` | fails before, passes after; one warning, one notice | 2 |
| the wire | the `llm/provider` test | original refused, rewrite sent unchanged and answered | 2 |
| plants | P1 step 6 | each named test red | 3 |
| race | `make race`; the repeated end-to-end run | exit 0 | — |
| the gates | `make preflight` on two hosts | exit 0 | — |
| review | P2 | findings fixed or answered | 4 |
| live | P3 steps 3 to 5 | pass | 5 |

## Rollout and Rollback

**Rollout.** P0's records commit and P1's commit land on `p3-acpext-v0.5.0`, next, before 0052-PLAN P5 (Q6 as
re-put). CI runs on the branch's fresh draft pull request to `main` with every push. `main` moves only on the owner's
word at a phase boundary; after P3 passes, the coordinator asks the owner for that fast-forward. The work ships in
`v0.7.0` (0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md). No tag or release follows from this
plan alone.

**Rollback.** `git revert` of P1's commit, as a new commit, restores today's schema path; the MADR gets a dated
amendment and this plan a deviation entry. Reasons: a route that accepted the original refuses the rewrite (P3 step 6),
or Grok refuses it (P3's stop condition).

## Open questions for the owner

### Decided (2026-11-17)

All seven answered with the recommended option, the labels quoted in the MADR's "Decided" list. For this plan:

* Q1 "Every provider, when the tool is loaded (Recommended)", Q2 "Send it as given and flag it once (Recommended)", Q3
  "Rewrites in the debug log; one warning per session for each flagged tool (Recommended)": P1 as written.
* Q4 "A record of its own, drafted when the SDK release is tagged (Recommended)" and Q7 "In the follow-up record, by
  checking the SDK's error kind first (Recommended)": no phase here; the `signature` misfire stays out of P1.
* Q5 "Yes, both, by that repository's own record (Recommended)": nothing in `mcp-server-magictools` here.
* Q6, re-put: "Next, before 0052 P5, on the same branch with a fresh draft pull request (Recommended)": Scope's base, P0
  step 4, P1 step 10 and Rollout.
* The plan: "Approve through my live step too, then ask about `main`": approved through P3.

As drafted, any other answer would have moved P1's change (Q1 (b)), added a degrade or an exclusion (Q2 (b), (c)), a
gated P4 (Q4 (b)) or a text exclusion (Q7 (b)); none applies.

## Execution record

### Drafting (2026-11-17)

* Drafted read-only against `p3-acpext-v0.5.0` at `2004980` (level with `main` and `origin/main`, tree clean);
  `python3 scripts/check_records.py --next` printed `0060`, exit 0.
* The experiments (the MADR's R1 to R4) ran in a `git archive 2004980` copy and a folder beside it under `<scratch>`;
  nothing ran in the working tree. Their scripts and result tables are kept with the session's records, not committed.
* Found while drafting, for the owner: the magictools handler requires all of `target`, `intent` and `session_id`
  while its schema and description offer `session_id` alone (the MADR's M5 and M8; Q5); gobble's `inputSchema` writes
  into the go-sdk's own schema map (the MADR's G2; P1 ends it); the `signature` text match resends once on a tool named
  `…signature…` (the MADR's R3; Q7).

### Acceptance and P0 (2026-11-17)

* The owner answered Q1 to Q7 and approved this plan through P3, relayed by the coordinator (the MADR's "Decided"). The
  MADR is `accepted`, this plan `in-progress`, the two README rows to match; P0 step 3 done before the first commit.
* Start state: `p3-acpext-v0.5.0` at `2004980`, with `main` and `origin/main` at `2004980` (fast-forwarded that day);
  only this pair and `docs/README.md` changed.
* Gates after the last edit, each exit status taken before any filter: `make check-records` 0; `make markdownlint` 0
  (it excludes records by its globs); a citation check of every `path:line` in the pair, 59 citations against 56 lines
  read at their commits, 0, and 1 on a copy with one citation moved to a false line; markdownlint on renamed copies of
  the pair, MD004 (`*` list markers) only, the class the accepted 0059 records give; an identifier scan of the pair and
  the README diff, no hits.
* The pair and `docs/README.md` staged by path for the coordinator's commit (P0 step 4).

### P1 (2026-11-17)

* **Provenance.** Built by a mainstream-class model at high effort, as this plan prices P1, on the coordinator's brief; the
  coordinator reads the diff and runs its own gate. Base `0592367` on `p3-acpext-v0.5.0` (level with
  `origin/p3-acpext-v0.5.0`, tree clean, `go1.27.4`; K1). The hooks path is the global hooks directory. Nothing is
  committed, pushed or tagged.
* **Built.** `mcpclient/schema.go`: the rule `rewriteRootUnion` (`:49`) with its conditions in D2's order (`ruleApplies`,
  `:77`), each branch's kind (`objectBranch`, `:113`), the typed copy with the root's keywords (`objectify`, `:147`), the
  flag's condition and text (`hasUntypedBranch`, `:171`; `flagText`, `:189`), and a schema-aware removal of descriptions
  (`stripDescriptions`, `:220`; a property named `description` and a `description` inside `enum`, `const`, `default` or
  `examples` are kept). `mcpclient/tool.go:78`: `inputSchema` works on a deep copy of the SDK's map, adds `type` and
  `properties`, then applies the rule; `newTool` keeps whether it rewrote and the flag (`:43`). `mcpclient/manager.go`:
  `SchemaNote` (`:106`), `FlaggedTools` (`:115`) and `RewrittenTools` (`:121`) over `listed` (`:71`), which `Tools` now
  uses, each limited by the tool filter. `acpserver/mcp.go`: `reportSchemas` (`:198`, called at `:178`), once per
  session for each tool: a debug record for a rewritten tool, and for a flagged one a warning record and a notice, title
  `mcp: tool schema sent as the server gave it`, naming `<server>/<tool>`, the reason and the remedy
  (`--exclude-tools mcp__<server>__<tool>`, or a permission rule). `mcpclient/mcptest/mcptest.go`: `NewUnionServer`
  (`:208`), `UnionHTTPServer` (`:246`), `ExecutePipelineSchema` (`:129`) and `NullableUnionSchema` (`:184`), without
  changing `NewServer`'s tools. Data: `mcpclient/testdata/magictools-execute-pipeline.json` (the object of
  `mcp-server-magictools` `internal/handler/inventory.go:289-336` at `368ac6a`, read there with `git show`, that
  repository untouched), `.rewritten.json` (by hand from D2), `.source.txt` (the note). The live test
  `llm/provider/live_grok_union_test.go` (`//go:build live_grok`), compiled by `go vet -tags live_grok ./llm/provider/`
  (exit 0) and never run. Docs: `docs/guides/mcp-servers.md` (new), `docs/README.md` (guides list and two "I want to…"
  rows), `docs/architecture.md` (the `mcpclient`, `mcpclient/mcptest` and `acpserver` rows), `mcpclient/doc.go`,
  `mcpclient/mcptest/doc.go`.
* **Tests.** `mcpclient/schema_test.go`: `TestRewriteRootUnion` (S0 to S12, S6b and X1 to X16: expected JSON, or the schema
  as given with its flag), `TestRewritePreservesInputs` (every rewritten case: the same verdict from jsonschema-go on the
  corpus of the plan, 2,265 instances for the real schema with 91 accepted by both, 30,783 over all fifteen rewritten
  cases, each corpus checked to accept some and refuse some), `TestRewrittenBranchesAreObjects`, `TestRootIsUntouched`,
  `TestRewriteIsIdempotent`, `TestRewriteSharesNothing`, `TestNewToolDoesNotWriteTheSDKsMap`,
  `TestInputSchemaAdditionsThenRule`, `TestRefusedShapesAreUntouched`, `TestFlagReasonWording`,
  `TestFixtureSchemaIsTheCopy`, `TestFixtureShapes`. `mcpclient/manager_test.go`: `TestSchemaNotes`,
  `TestRewrittenToolIsCalledAsGiven`. `acpserver/mcp_test.go`: `TestMCPUnionSchemaProviderRefusal` (the test double
  refuses the original with the owner's text and passes the rewrite), `TestMCPUnionSchemaRewritten` (the end-to-end
  prompt: completes, the request's schema equals the rewrite, the call reaches the server with its arguments intact, one
  debug record over two prompts), `TestMCPUnionSchemaFlaggedOnce` (S8's shape served by the fixture, two prompts, one
  warning record and one notice, and the provider's refusal is of that tool), `TestMCPUnionSchemaRemedies` (the filter
  and a deny rule: nothing warned, the prompt completes). `llm/provider/wire_union_test.go`: `TestWireRootUnionSchema`.
  `internal/cli/mcp_test.go`: `TestChatWarnsOfAFlaggedToolSchema`.
* **Red first (K3).** Run against a `schema.go` that returned its input, with `FlaggedTools` and `RewrittenTools` stubs
  returning nothing and no `acpserver` code: `go test ./mcpclient/` failed `TestSchemaNotes` (`manager_test.go:138`:
  flagged `[]`), `TestRewriteRootUnion` (`schema_test.go:233`: the real schema, `rewritten false`),
  `TestFlagReasonWording`, `TestRewritePreservesInputs` ("not rewritten"), `TestRewrittenBranchesAreObjects`
  (`:451`: `anyOf[0]` not typed), `TestRewriteSharesNothing`, `TestNewToolDoesNotWriteTheSDKsMap` (two subcases, the SDK's
  map written), `TestInputSchemaAdditionsThenRule` and `TestFixtureShapes`; `go test ./acpserver/ -run TestMCPUnionSchema`
  failed the end-to-end, flagged-once and remedies tests with the owner's refusal text (the provider-double test passed:
  it tests the double); `TestWireRootUnionSchema` failed with the same 400 on the loaded tool; and
  `TestChatWarnsOfAFlaggedToolSchema` failed with 0 warnings on stderr. Tests that pass against the stub by their
  nature (`TestRootIsUntouched`, `TestRefusedShapesAreUntouched`, `TestRewriteIsIdempotent`, `TestFixtureSchemaIsTheCopy`)
  are held by plants 8, 15, 3 and 9 below. The outputs are kept with the session's records.
* **Green (step 5).** `go test ./mcpclient/... ./acpserver/ ./llm/provider/` exit 0.
* **Planted breaks (K2),** on one scratch copy of `git archive HEAD` plus the uncommitted files copied, each file
  asserted to occur once, `gofmt` clean, the copy built with its tests (so a red is a test failure, never a build failure),
  run again on the final code. Failing line is the first the test prints:

  | # | Plant | Red tests | First failing line |
  | :--- | :--- | :--- | :--- |
  | 1 | the rule returns its input | `TestRewriteRootUnion/S0`; `TestWireRootUnionSchema`; `TestMCPUnionSchemaRewritten` | `schema_test.go:233` (rewritten false); `wire_union_test.go:178` (the 400); `mcp_test.go:535` (the prompt fails with the 400) |
  | 2 | branches typed, root's keywords not copied | `TestRewriteRootUnion/S0`, `TestRewrittenBranchesAreObjects` | `schema_test.go:233`; `schema_test.go:460` (`anyOf[0]` has no object keyword) |
  | 3 | the union dropped | `TestRewritePreservesInputs/S0` and `/S2` | `schema_test.go:404`: instance 9 `map[]`, the original false, the rewrite true |
  | 4 | a branch's own properties overwritten | `TestRewritePreservesInputs/S7` | `schema_test.go:404`: instance 10 `map[mode:x]`, the original false, the rewrite true |
  | 5 | the rule applied to a root typed `["object","null"]` | `TestRewriteRootUnion/S8`, `TestRewritePreservesInputs/S8` | `schema_test.go:221` (rewritten true); `schema_test.go:404`: instance 5 `null`, the original true, the rewrite false |
  | 6 | the SDK's map written in place | `TestNewToolDoesNotWriteTheSDKsMap` (two subcases) | `schema_test.go:607` |
  | 7 | the warning on every prompt | `TestMCPUnionSchemaFlaggedOnce` | `mcp_test.go:584`: 2 warning records over two prompts, want 1 |
  | 8 | descriptions removed from the root's properties too | `TestRootIsUntouched`, `TestRewriteRootUnion/S0` | `schema_test.go:524` |

  Beyond the eight, seven more, all red: 9 the fixture's copy of the schema drifts (`TestFixtureSchemaIsTheCopy`); 10 the
  provider double never refuses (`TestMCPUnionSchemaProviderRefusal`, `TestMCPUnionSchemaFlaggedOnce`); 11 descriptions
  removed by name (`TestRewriteRootUnion/X1`); 12 a denied tool still warned of (`TestMCPUnionSchemaRemedies/a_deny_rule`);
  13 the rewrite noted on every prompt (`TestMCPUnionSchemaRewritten`); 14 the filter ignored when noting tools
  (`TestSchemaNotes`); 15 a `true` branch removed (`TestRewritePreservesInputs/S9`). No plant survived.
* **Gates** (each exit status taken before any filter; times are the host clock, queue waits included):

  | Gate | Host | Start | End | Exit |
  | :--- | :--- | :--- | :--- | :--- |
  | `make pre-add-check` over the 13 changed `.go` files | session host | 13:09:49 | 13:10:05 | 0 |
  | `make preflight`, first run | session host, through `run-heavy` | 13:10:09 | 13:18:56 | 2: `markdownlint` MD004, `*` list markers in the new guide; every earlier step passed. The guide's markers were changed to `-`, nothing else |
  | `make preflight`, second run | session host, through `run-heavy` | 13:19:35 | 13:29:28 | 0, "preflight passed" |
  | `make race` | the Linux machine, `run-on-host --race --need linux` | 13:10:19 | 13:28:01 | 0 |
  | `make race` | the Windows laptop, `run-on-host --race --need windows -- sh -c "make race"` | 13:10:25 | 13:17:14 | 0 |
  | `go test -race -count=20 -timeout 15m -run TestMCPUnionSchema ./acpserver/` | session host, through `run-heavy` | 13:10:44 | 13:19:40 | 0 |
  | `make check-test-isolation` | session host | 13:18:00 | 13:19:25 | 0 |
  | `make preflight` | a Linux target (the picker chose a WSL distro of the Windows laptop), `run-on-host --need linux` | 13:29:39 | 13:39:41 | 0, "preflight passed" |

  After the last record edit, each exit status taken before any filter: `make check-records` 0 and `make markdownlint` 0
  (59 files, 0 issues; it excludes records by its globs), and an identifier scan of the staged diff for the local account
  name, the hosts' names and real-machine absolute paths, no hits.
* **Notes on the plan as written, none stopping the phase** (each keeps the step's intent; the owner may disagree):
  1. S8's shape cannot be registered with the go-sdk's `Server.AddTool`, which panics unless a root's `type` is exactly
     `"object"` (`mcp/server.go`, `AddTool`, v1.8.0). The fixture registers `pick_one` with an object root and a
     receiving middleware swaps S8's schema in when it lists its tools, so the client is sent S8 as written.
  2. The end-to-end provider double cannot call `llmprovider.ClassifyHTTPError`: `internal/archtest` rule 4 lets only
     `llm/provider` import the SDK, tests included. The double sends a request through gobble's own `grok` provider over an
     in-process transport that answers xAI's recorded 400, so the SDK's classification and gobble's mapping both run, and
     the text asserted is the owner's paste with the SDK's own words in its ellipsis: `llm: api status 400: llmprovider:
     invalid request: grok HTTP 400 invalid-argument: Failed to start sampling: [invalid_client_tool_schema] <tool>: …`.
  3. "Sent byte-identical to the spec" holds up to key order: the SDK writes a schema again with its keys sorted, so the
     wire test compares the schema on the wire with the spec's and with the rewrite file after RFC 8785 canonicalising both.
  4. The wire test takes its spec from the tool as `mcpclient` loads it from the union fixture (the test imports
     `mcpclient` and `mcptest`, which no archtest rule forbids), so that plant 1 turns it red as the plan expects; the
     original schema is handed to the provider as the server sends it.
  5. `internal/cli`: the harness starts a session and shows the record, but its stderr is not a terminal, so it is
     slog text (`level=WARN msg="mcp: tool schema sent as the server gave it" server=fx tool=pick_one …`); the `Warning:` line
     is 0024-MADR D6's handler on a terminal, which `Main` cannot be given here. The test asserts the record once, naming the
     tool and the remedy. The MADR's "terminal shows a `Warning:` line" stays not verified for this record.
  6. D4's remedy "a permission rule that denies it" would not end the warning if a hidden tool were still reported, so
     `reportSchemas` reports only tools the session's rules leave visible (`visible(withDefaults(…))`, as the prompt does);
     `TestMCPUnionSchemaRemedies` holds it, plant 12 breaks it. `l.reported` keys are `tool schema rewritten: <name>` and
     `tool schema flagged: <name>`, so they cannot meet a server's name.
  7. S6 as the MADR writes it has `$defs` at the root, so D2's order refuses it for the root keyword, not the `$ref`;
     S6b has a `$ref` branch alone.
  8. A refused shape is flagged only when its root union has a branch not typed `object` (D3); one whose branches are all typed
     objects (X7) is sent as given and not flagged.
* **Not done, and why.** No outside request, no real credential, no real port beyond the in-process `httptest`
  servers the existing fixtures use; the live test was compiled, not run (P3). The typed-error fallback, the `signature`
  misfire and the `mcp-server-magictools` findings are other records. `AGENTS.md` read: it does not describe MCP
  schemas, no change. No `//nolint`, no skipped test, no `replace`, no `go.work`, no pin change.
* **Not verified.** Whether Grok or any other route accepts the rewritten form (P3); that a terminal shows the warning as a
  `Warning:` line (note 5); `$dynamicRef` or `$id` inside a branch or inside the properties the rule copies, which D2 does
  not name and the corpus does not cover; `unevaluatedProperties` and dependent keywords inside a branch.
* **Docs rows.** `docs/guides/mcp-servers.md:1` fixed (new); `docs/README.md:10-14` fixed (guides list); `docs/README.md`
  "I want to…" two rows fixed; `docs/architecture.md:128`, `:131`, `:132` fixed; `mcpclient/doc.go` and
  `mcpclient/mcptest/doc.go` fixed; `AGENTS.md` unchanged (no MCP-schema text). docs: searched `README.md`, `docs/`,
  `AGENTS.md`, `mcpclient/`, `acpserver/` for `inputSchema`, input schema, tool schema, `exclude-tools` and `mcp__`; nothing
  else it makes false.
* **Staged, not committed.** The changed and new files of P1 and this record staged by path for the coordinator.

### P2 (2026-11-17): the independent review of `cde93be`

* **Who and what.** An independent review by the strongest-class model, as this plan prices P2, which was given D2, the
  diff of P1, the two testdata files and the shapes, and not P1's conclusions; the coordinator committed and pushed P1
  as `cde93be` first, after its own preflight and ten race rounds.
* **Verdict.** The commit does what 0060 decided: the notices, the remedies, the aliasing (the SDK's map is not written
  to) and the call path work. The rule is not input-preserving for three kinds of schema it rewrites, which D2's
  argument and its conditions did not cover (B1, in the MADR's amendment of this date): `unevaluatedProperties` in a
  union branch (180 objects newly accepted), a `$ref` anywhere but a branch's top level, including one into the union
  (up to 1,948 newly refused), and `$id` or `$anchor` in the properties it copies (duplicates make the rewritten schema
  invalid for one of two validators). The real magictools schema has none and is preserved exactly. Smaller findings:
  S1, a surviving plant ("copy each missing keyword instead of none-of-the-three" survived, since no case had a branch
  with its own `properties` under a root with `additionalProperties`); S2, the live test should log the refusal's text;
  S3, a test for a tool hidden and then shown; N1, the wire test should compare the spec's bytes to the wire's raw; N2,
  the guide should show what a terminal prints and the notice should say a deny rule "with no pattern"; N3,
  `NewUnionServer`'s doc should say how `pick_one` is registered; N4, the guide should say a root with no type is typed
  first.
* **The eight deviations of P1.** P1's entry lists eight notes on the plan as written, headed "none stopping the phase".
  They were built without stopping: the rule is that a deviation from the plan stops the step and is put to the
  coordinator first, and P1 built each one way and reported it afterwards. The review found each sound, so none is
  undone; the entry above is kept as it stands, and this is its correction.
* **The owner's rulings of 2026-11-17,** relayed by the coordinator, by label: R-1, "Refuse the rule whenever any of
  those keywords appears anywhere in the union or the copied properties (Recommended)", which amends D2 (the MADR's
  amendment of this date); R-2, "Keep it, and add a test for hidden-then-shown (Recommended)", for the deny-rule
  behaviour P1 added (note 6 of P1's entry).

### P1 fixes (2026-11-17)

* **Provenance.** A mainstream-class model at high effort, on the coordinator's brief after P2; base `cde93be` (tree
  clean at the start); nothing committed, pushed or tagged. The review's probes and scratch fix were read, not copied:
  the walk below is not the scratch fix's, which looked at member names anywhere and so would refuse a property called
  `$ref`.
* **Built.**
  1. **B1, the walk.** `mcpclient/schema.go:52` `scopeKeywords` (the nine); `:121`, in `ruleApplies`, searches the union
     and the three copied keywords with `scopeKeywordAt` (`:134`) and `scopeKeywordIn` (`:165`), and refuses with
     `the union or the keywords it would copy hold "<keyword>", which the rule does not cover`. It looks only where a
     schema is, with the lists `stripSchema` already uses: a schema's own keys are tested against the nine; a key that
     holds a schema (`items`, `additionalItems`, `contains`, `propertyNames`, `not`, `if`, `then`, `else`,
     `additionalProperties`, `unevaluatedProperties`, `unevaluatedItems`, `contentSchema`), a list of schemas (`allOf`,
     `anyOf`, `oneOf`, `prefixItems`) or a map of names to schemas (`properties`, `patternProperties`, `$defs`,
     `definitions`, `dependentSchemas`, `dependencies`) is walked. In a map of names the keys are never tested, only
     their values, which is how a property called `$ref` is told from the keyword; `enum`, `const`, `default`,
     `examples`, `required` and `type` are not walked, so data holding a `$ref` key is not a keyword either. A keyword
     the 2020-12 applicator vocabulary (and the older drafts' `definitions`, `dependencies` and `additionalItems`) does
     not list is not walked.
  2. **S1.** Cases and a plant only; the rule was right.
  3. **S2.** `llm/provider/live_grok_union_test.go` logs the redacted error text when the original is refused (still
     compiled only).
  4. **S3 and R-2.** `acpserver/mcp_test.go` `TestMCPUnionSchemaHiddenThenShown`; the deny-rule behaviour is kept.
  5. **Notes.** N1: `TestWireRootUnionSchema` compares the schema on the wire to the spec raw, byte for byte (they are
     identical, the SDK and mcpclient both write sorted keys), and canonicalises only the hand-indented file. N2: the
     notice's and the log's remedy says "a permission rule that denies it with no pattern" (`acpserver/mcp.go:219`), and
     the guide quotes what a terminal prints, run through the terminal handler (`internal/logging`'s `Show` and `Wrap`,
     at 80 columns) with this record's fields. N3: `NewUnionServer`'s doc says how `pick_one` is registered and swapped.
     N4: the guide says a root with no type is typed first.
  6. **Docs and records.** `docs/guides/mcp-servers.md`, `docs/architecture.md` (the `mcpclient` row),
     `mcpclient/doc.go`; the MADR's dated amendment, with pointers on its Context step 6 and D2's list.
* **Tests, red first on `cde93be`** (the outputs are kept with the session's records):
  * `mcpclient/schema_scope_test.go` `TestScopeKeywordsRefusedEverywhere`: each of the nine keywords in each of 38
    positions of the union or of the copied keywords (including three deep, and a `oneOf` root), each position also with
    a harmless title as its control, which must be rewritten. 341 of its 342 subtests failed (`schema_scope_test.go:116`,
    "rewritten true"); the one that passed is a `$ref` at a branch's top level, which `cde93be` already refused.
  * `TestRewriteRootUnion/R1` to `R7` failed (`schema_test.go:282`, "rewritten true"): R1 `unevaluatedProperties` in a
    branch, R2 a root property `$ref` into the union, R3 a nested `$ref` shifted by a removed branch, R4 `$anchor`, R5
    `$id`, R6 `$dynamicAnchor`, R7 the known cost `"$ref": "#"`, now flagged. `TestRewritePreservesInputs/R1`, `/R2`,
    `/R3` failed at `schema_test.go:465` with the review's instance (`map[a:x]`: R1 the original false, the rewrite
    true; R2 and R3 the original true, the rewrite false); `/R4` to `/R7` failed at `:474`, "the rewrite holds
    "$anchor"" and so on, the check that no rewritten schema carries any of the nine names, searched as member names
    anywhere with a list of its own. `TestRefusedShapesAreUntouched` failed at `:495` (R1 rewritten).
  * Already green on `cde93be`, so held by plants alone: `TestScopeKeywordNamesAreNotKeywords` (a property, a name in a
    branch, deep, in `required`, and data in `default`, `enum`, `const` and `examples`, called each of the nine; rewritten
    and preserved), cases `N1` to `N3`, `X17` (S1), `TestCounterExamplesAreReal` (each counter-example's hand-written
    pre-amendment rewrite changes the verdict on the review's instance), `TestMCPUnionSchemaHiddenThenShown`, and the
    raw wire comparison.
  * `TestMCPUnionSchemaFlaggedOnce` failed on `cde93be` for the new remedy wording (`mcp_test.go:605` the log, `:614` the
    notice lack "a permission rule that denies it with no pattern").
* **Plants (K2),** on a scratch copy of `git archive HEAD` plus the uncommitted files copied, each file asserted to occur
  once, gofmt clean, the copy built with its tests; each red as a test failure:

  | # | Plant | Red | First failing line |
  | :--- | :--- | :--- | :--- |
  | F1 | the walk skips nested positions | `TestScopeKeywordsRefusedEverywhere` (e.g. `root_property_items`) | `schema_scope_test.go:116` |
  | F2 | `$dynamicAnchor` dropped from the list | the matrix for it, `TestRewriteRootUnion/R6` | `schema_scope_test.go:116` |
  | F3 | a property named like a keyword taken for it | `TestScopeKeywordNamesAreNotKeywords` (all nine) | `schema_scope_test.go:138` |
  | F4 | copy each missing keyword instead of none-of-the-three | `TestRewriteRootUnion/X17`, `TestRewritePreservesInputs/X17` | `schema_test.go:294`; `:465`: instance 14 `map[a:x c:true]`, the original true, the rewrite false |
  | F5 | the reported-once key recorded while hidden | `TestMCPUnionSchemaHiddenThenShown` | `mcp_test.go:645`: after prompt "shown", 0 warning records, want 1 |
  | F6 | the spec written indented | `TestWireRootUnionSchema`, `TestInputSchemaAdditionsThenRule` | `wire_union_test.go:186`: not the spec's, byte for byte |
  | F7 | the remedy without "with no pattern" | `TestMCPUnionSchemaFlaggedOnce` | `mcp_test.go:605` |
  | F8 | the union's own keyword not searched | the matrix, the branch positions | `schema_scope_test.go:116` |
  | F9 | data in `default`, `enum`, `const` walked as schemas | `TestScopeKeywordNamesAreNotKeywords` | `schema_scope_test.go:138` |

  No plant survived. The three the brief names are F1, F2 and F3; F4 is the S1 plant; F5 the S3 plant.
* **Input preservation.** The plan's corpus (seed `20261117`; every subset of the names plus `zz`, 2,000 random objects,
  nine non-objects) with jsonschema-go `v0.4.3`: 0 differences over every rewritten case, 38,891 instances in all in
  19 cases (the 15 of P1, X17 and N1 to N3), the real schema 2,265 instances with 91 accepted by both, as before; and
  0 differences in each of the nine generated cases of `TestScopeKeywordNamesAreNotKeywords`. The counter-examples R1 to
  R7 are not rewritten, so have none to preserve. One validator: `santhosh-tekuri/jsonschema`, which the review used, is not in `go.mod`, and
  `AGENTS.md` requires a MADR naming a module before it is required, so it is not added here; the amendment says so. The
  corpus cannot show a duplicate `$id` or `$anchor` invalid (jsonschema-go accepts the copies), so those are held by
  the search of every rewritten schema for the nine names, and by the matrix.
* **Gates** (each exit status taken before any filter; host clock, queue waits included):

  | Gate | Host | Start | End | Exit |
  | :--- | :--- | :--- | :--- | :--- |
  | `make pre-add-check` over the 9 changed `.go` files | session host | 15:34:34 | 15:34:46 | 0 |
  | `make preflight` | session host, `run-heavy` | 15:34:56 | 15:43:26 | 0, "preflight passed" |
  | `make race` | the Linux machine, `run-on-host --race --need linux` | 15:34:56 | 15:39:35 | 0 |
  | `make race` | the Windows laptop, `run-on-host --race --need windows -- sh -c "make race"` | 15:34:56 | 15:41:15 | 0 |
  | `go test -race -count=20 -timeout 30m ./mcpclient/...` | session host, `run-heavy` | 15:34:56 (queued) | 15:46:29 | 0 (180 s of run) |
  | `make check-test-isolation` | session host, `run-heavy` | 15:35:02 (queued) | 15:47:55 | 0 |
  | `go vet -tags live_grok ./llm/provider/` | session host | 15:35 | 15:35 | 0 |

  After the last record edit, each exit status taken before any filter: `make check-records` 0 and `make markdownlint` 0 (59 files, 0 issues); and an identifier scan of the staged diff for the local account name, the hosts' names and real-machine paths, no hits.
* **Docs rows.** `docs/guides/mcp-servers.md` (the rule's conditions, the terminal and notice forms, hidden-then-shown):
  fixed. `docs/architecture.md:131` fixed. `mcpclient/doc.go` fixed. `mcpclient/mcptest/mcptest.go` doc fixed.
  `AGENTS.md` unchanged. docs: searched `README.md`, `docs/`, `AGENTS.md`, `mcpclient/`, `acpserver/` for the rule's
  conditions, `accepts exactly`, `$ref`, `with no pattern` and `Warning:`; nothing else it makes false.
* **Removed lines in the staged record diff:** none. The MADR gains two pointer lines (Context step 6, D2's list) and
  an appended amendment; this plan gains the three entries above; no committed record line is rewritten or removed.
* **Not verified.** That any provider route accepts the rewritten form (P3); the second validator's agreement on this
  pass's cases (not in the tree, above); a keyword an MCP server invents that holds a schema, which the walk does not
  look into, since no validator would; the terminal form of the warning for this
  record as printed by `gobble` itself, rather than by its handler with this record's fields.
* **Staged, not committed.** The fix pass's files staged by path for the coordinator.
