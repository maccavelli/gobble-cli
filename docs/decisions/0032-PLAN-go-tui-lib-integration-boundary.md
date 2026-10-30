---
status: in-progress
date: 2026-10-22
associated-madr: "0032-MADR-go-tui-lib-integration-boundary.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement the go-tui-lib boundary, and ask go-tui-lib for the elements gobble needs

Associated MADR: [0032-MADR-go-tui-lib-integration-boundary.md](0032-MADR-go-tui-lib-integration-boundary.md)

## Goal

`archtest` lets the native CLI import a go-tui-lib package whose import closure holds no Charm module, and nothing
else of go-tui-lib or Charm; the requests for the Charm-free packages and the components are in go-tui-lib's tracker;
and each adoption has a place to be written when its tag exists (MADR D1–D7).

## Scope

- **In:** `internal/archtest` (rule 5's check) and its tests; the records (0004-MADR rule 5 and its dated note,
  0008-MADR F5, 0029-PLAN P2's wait, `docs/architecture.md`, `docs/README.md`); the requests in go-tui-lib's tracker.
- **Out:** any go-tui-lib import or pin (D7: in the commit that first imports a package); the TUI mode (0005-PLAN F9);
  go-tui-lib's own work, which its repository decides and builds.

## Implementation Steps

### P1 — the boundary (D1)

1. **Tests first,** on `archtest`'s fixture packages (no go-tui-lib requirement in `go.mod`): rule 5 is broken by an
   `internal/cli` package importing a go-tui-lib package that imports Charm, directly or through another go-tui-lib
   package, and by an `internal/cli` package importing `internal/tui`; it is kept by one importing a go-tui-lib package
   with no Charm in its closure, by `internal/tui` importing anything of go-tui-lib or Charm, and by a test importing
   `tuitest`.
2. **The check:** `checkRule5` looks at each go-tui-lib package a gobble package outside `internal/tui` imports, and at
   that package's whole closure as `go list -deps` reports it, and refuses Charm anywhere in it; `internal/cli` →
   `internal/tui` is refused.
3. **Records:** 0004-MADR rule 5 amended, with its text and the check changed together (`AGENTS.md`); a dated note
   in 0008-MADR F5 that D8's and D11's door is opened by this record; `docs/architecture.md`'s rule list.
4. Gates. Commit.

### P2 — the requests (D2–D5)

Each written as an issue in `github.com/maccavelli/go-tui-lib`, with gobble's case, its measurements and the records
it rests on; filed by the owner, or at the owner's word:

1. **Charm-free headless packages:** `termcap`'s environment facts and doctor report; `termsvc`'s sequences;
   `command`'s registry, parsing, completion and result writing; `keymap`'s `keybindings.json` loading; `fuzzy`.
2. **An inline question with grouped rows,** drawn in a line-mode CLI's scrollback and in the TUI, from 0029-MADR D1,
   D2, D5 and D6: a wrapped title, a note, rows of keyed options never split, the keys and the echoed answer as the
   caller gives them, sanitised, laid out to a width, testable without a terminal.
3. **The later elements, as candidates:** the picker (models, sessions), the input editor (what gobble's
   `internal/cli/editor` does today: whole words, history, paste, resize), the transcript with streamed Markdown and
   resize, and the permission dialog for the TUI.

Then: 0029-PLAN gets a dated note that P2 waits for item 2's tag; this PLAN's record lists the issues.

### P3 and on — adoptions (D7)

One step per tag, written as an amendment to this PLAN when the tag exists: pin it, draw through it, remove what it
replaces in gobble, and gate it on the three hosts. 0029-PLAN P2 is the first.

*(2026-10-29: carried out by [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md) P13–P18, which pins go-tui-lib/v2 `v2.6.2`. This PLAN stays `in-progress` until
that PLAN's P18, whose commit makes it `complete`.)*

### P1a — `x/ansi` admitted (MADR amendment A1)

*Approved by the owner, 2026-10-22: "Post. File. Approved", with step 4's comments and the sanitiser issue.*

1. **Tests first,** in `TestImportRules`: `internal/cli` importing `github.com/charmbracelet/x/ansi` directly passes;
   `internal/cli` importing a go-tui-lib package whose closure holds `x/ansi` and its text modules passes; one whose
   closure also holds `charm.land/bubbletea/v2`, `colorprofile` or `ultraviolet` is refused, naming that module; a core
   package (rule 1's roots) importing `x/ansi` is refused, by rule 1 and rule 5; `internal/tui` passes as before.
2. **The check:** rule 5's closure test, and its direct-import test for `internal/cli`, exempt
   `github.com/charmbracelet/x/ansi` and its sub-packages, and nothing else.
3. **Records:** 0004-MADR rule 5's amendment note, 0008-MADR F5's dated note and `docs/architecture.md`'s rule 5 name
   `x/ansi`.
4. **go-tui-lib's issues 2 to 10:** a comment on each that `x/ansi` may stay, and what gobble's CLI still needs it to
   drop; at the owner's word, as before. If the sanitiser is wanted from the CLI, an issue asking for it exported.
5. **Gates. Commit.** The negatives: the exemption widened to all of `github.com/charmbracelet/x/...`; the exemption
   given to a core package; the closure's Bubble Tea no longer named.

The first adoption that brings `x/ansi` into `go.mod` (`fuzzy` at its tag, or gobble's width measured with it) is a
P3 step, measured against 0030-MADR D5's budget before it lands.

## Verification

- `archtest`'s new cases, each seen failing first; a negative per case on a scratch copy.
- `make preflight` and the tests on macOS, the Linux machine and the Windows laptop; `check-records`; `markdownlint`.
- The issues' text scanned for identifiers before filing.

## Rollout and Rollback

P1 is one commit: reverting it restores rule 5 as it was, and nothing imports go-tui-lib. P2 changes no file of
gobble's but its records. Each adoption is its own commit.

## Execution record

Approved by the owner's "yes proceed to adopt from the SDK as much sa possible. open any issues needed.", 2026-10-22.
The records were committed first (`6ee2014`).

### P1 execution record (2026-10-22)

1. **Tests first,** eight new cases in `TestImportRules` (`internal/archtest/check_test.go`), on fixture packages, with
   `modPkg.Deps` and the `tuiLib` constant added as stubs that changed nothing:
   - `rule 5 cli may import a charm-free go-tui-lib package` failed on the old check: `unexpected edge`;
   - `rule 5 cli go-tui-lib with charm further down` failed: the old refusal named no Charm module
     (`missing "github.com/charmbracelet/x/ansi"`);
   - `rule 5 cli may not import internal/tui` failed: the old check did not see the edge;
   - `with charm in its closure`, `whose closure is unknown`, `a core package may not import a charm-free go-tui-lib
     package`, `tui may import any of go-tui-lib` and `a test may import tuitest` passed on the old check, which
     refused every go-tui-lib import outside `internal/tui`; they stay as guards, each shown failing below.
2. **The check:** the loader keeps each go-tui-lib package `go list -deps` reports, with its whole closure (`Deps`).
   `checkRule5` passes anything from `internal/tui`; refuses Charm from any other package; refuses `internal/cli` →
   `internal/tui`; refuses any go-tui-lib package outside `internal/cli`, `tuitest` in a test excepted; and lets
   `internal/cli` import a go-tui-lib package only when its closure is known and holds no Charm, naming the first
   Charm package when it does not (`… (through charm.land/bubbletea/v2)`).
3. **Negatives,** one at a time on a scratch copy: the closure ignored; an unknown closure allowed; `internal/cli` →
   `internal/tui` allowed; any package let import a Charm-free go-tui-lib package; `internal/tui` no longer exempt;
   `tuitest` refused in tests. Each failed its cases (`… with charm in its closure`, `… further down`, `… whose
   closure is unknown`, `… may not import internal/tui`, `… a core package may not import …`, `… tui may import charm`
   and `… tui may import any of go-tui-lib`, `… a test may import tuitest`); the restored copy passed. Two first drafts
   of these mutations did not build (a variable left unused) and were rewritten to disable the check alone.
4. **Records:** 0004-MADR rule 5 amended in place, with the check (`AGENTS.md`); a dated note on 0008-MADR F5;
   `docs/architecture.md`'s rule 5.

The real module passes: nothing in gobble imports go-tui-lib yet, so the loader's new path is first exercised by the
first adoption.

**Gates:** the pre-add check on the two changed `.go` files, `2 file(s) clean (gofmt, golint, govulncheck)`; macOS,
`make preflight` `preflight passed`, the tests and the race detector 36 packages ok each; the Linux machine and the
Windows laptop, on a copy of the tree, the tests and the race detector 36 packages ok each, `make preflight` to
`verify-build-metadata` (golangci-lint `0 issues.`, govulncheck "No vulnerabilities found."), as the copies are not git
checkouts; `make check-records markdownlint` exit 0 on each host; `make probe-conhost` on the laptop, `--- PASS:
TestConhost`; `make perf` within three times each baseline on all three (macOS ReplyRender 34.6 µs, Reprint 10.1 ms,
LineSessionStart 1.10 ms, KeyDraw 1.60 µs; the Linux machine 139 µs, 34.9 ms, 4.00 ms, 5.75 µs; the Windows laptop
28.7 µs, 7.69 ms, 4.85 ms, 1.08 µs).

**P1 complete, 2026-10-22.**

### P2 execution record (2026-10-22)

Filed in `github.com/maccavelli/go-tui-lib` at the owner's word, each body scanned for identifiers first, each
labelled `enhancement`:

| Issue | Request | MADR |
|---|---|---|
| 2 | `termcap`: the environment's facts and the doctor report without Charm | D2 |
| 3 | `termsvc`: the notification, link, clipboard and prompt-mark sequences without Charm | D2 |
| 4 | `command`: the registry, parsing, completion and result writing without Bubble Tea | D2 |
| 5 | `keymap`: `keybindings.json` loading, its schema, key labels and a byte decoder without Bubble Tea | D2 |
| 6 | `fuzzy`: the matcher without `x/ansi` | D2 |
| 7 | an inline question with grouped rows of keyed options, for line-mode CLIs and the TUI (0029's layout) | D3, D4 |
| 8 | an inline picker for line-mode CLIs: models and sessions | D3 |
| 9 | a line-mode input editor (composer) | D3 |
| 10 | a line-mode transcript: streamed Markdown, wrapped, reprinted on resize | D3 |
| 11 | `glyph`: status and progress glyphs, so a line-mode CLI can use the set whole | D2 |

Issue 11 came from step P3's first candidate. `glyph` is Charm-free at `v0.12.0`, but it covers three of gobble's
seven glyphs (`Bullet`; `Focus` for gobble's `Arrow`; `Ellipsis`, whose ASCII twin is `~` where gobble's is `...`)
and none of `Check`, `Cross`, `Bar` and `BarEmpty`. Adopting it now would keep gobble's set beside it and change
gobble's ASCII output, so it waits, as the rest do. `when` and `layout`, the other Charm-free packages, have no use in
the native CLI. Nothing of go-tui-lib is adopted yet; P3 begins with the first of these issues tagged.

0029-PLAN's P2 has a dated note that it waits for issue 7.

### P1a execution record (2026-10-22)

The owner, on the pros and cons of changing `x/ansi` in go-tui-lib, asked for the amendment ("Draft it"), then approved
it with this phase ("Post. File. Approved").

1. **Tests first,** in `TestImportRules`, with an `xansi` constant added as a stub:
   - `rule 5 cli may import x/ansi` and `rule 5 cli go-tui-lib whose charm is x/ansi alone` failed on the check before
     it, `unexpected edge`;
   - `rule 5 cli go-tui-lib with bubbletea beyond x/ansi` and `… with ultraviolet beyond x/ansi` failed, the old
     refusal naming `x/ansi` (`missing "charm.land/bubbletea/v2"`, `missing "github.com/charmbracelet/ultraviolet"`);
   - `rule 5 cli may not import another charm x module` (`x/term`), `rule 5 a core package may not import x/ansi` and
     `rule 1 a core package may not import x/ansi` passed, and stay as guards;
   - P1's `rule 5 cli go-tui-lib with charm further down` had used `x/ansi` as its Charm module; it now uses
     `colorprofile`, as A1 decides.
2. **The check:** `internal/cli`'s direct import of `x/ansi` (or a sub-package) passes; the closure test skips
   `x/ansi` and refuses the first other Charm module it meets. Nothing else changed.
3. **Records:** 0004-MADR rule 5's note, 0008-MADR F5's note and `docs/architecture.md`'s rule 5 name `x/ansi`.
4. **go-tui-lib:** a comment on issues 2 to 10, each saying `x/ansi` and `internal/sanitize` may stay and what that
   issue's packages must still drop; issue 6 (`fuzzy`) needs only a tag. Issue 12 filed: the sanitiser exported, with
   invalid UTF-8 made harmless and the bidirectional controls made visible, so gobble can drop its own (which has no
   fuzz target; go-tui-lib's has). Each text was scanned for identifiers first.
5. **Negatives,** one at a time on a scratch copy: the exemption widened to all of `github.com/charmbracelet/x/...`
   (fails `… another charm x module`); given to every package (fails `… a core package may not import x/ansi`); the
   closure test exempting nothing (fails three cases); the closure's Bubble Tea no longer named past `x/ansi` (fails
   three cases). The restored copy passed.

**Gates:** the pre-add check on the two changed `.go` files, `2 file(s) clean (gofmt, golint, govulncheck)`; macOS,
`make preflight` `preflight passed`, the tests and the race detector 36 packages ok each; the Linux machine and the
Windows laptop, on a copy of the tree, the tests and the race detector 36 packages ok each, `make preflight` to
`verify-build-metadata` (golangci-lint `0 issues.`, govulncheck "No vulnerabilities found."); `make check-records
markdownlint` exit 0 on each host; `make probe-conhost`, `--- PASS: TestConhost`; `make perf` within three times each
baseline (macOS ReplyRender 32.7 µs, Reprint 9.55 ms, LineSessionStart 826 µs, KeyDraw 1.50 µs; the Linux machine,
slowed as in 0031-PLAN P5's gates, 195 µs, 47.2 ms, 5.29 ms, 7.31 µs; the Windows laptop 28.4 µs, 7.30 ms, 5.01 ms,
1.08 µs).

**P1a complete, 2026-10-22.** Nothing of go-tui-lib or `x/ansi` is in `go.mod` yet: each comes with the adoption that
first imports it, P3's steps, beginning with whichever of issues 2 to 12 is tagged first.
