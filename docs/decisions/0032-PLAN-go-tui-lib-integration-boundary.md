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

## Verification

- `archtest`'s new cases, each seen failing first; a negative per case on a scratch copy.
- `make preflight` and the tests on macOS, the Linux machine and the Windows laptop; `check-records`; `markdownlint`.
- The issues' text scanned for identifiers before filing.

## Rollout and Rollback

P1 is one commit: reverting it restores rule 5 as it was, and nothing imports go-tui-lib. P2 changes no file of
gobble's but its records. Each adoption is its own commit.
