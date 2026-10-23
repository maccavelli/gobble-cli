---
status: accepted
date: 2026-10-22
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Let the native CLI use go-tui-lib's Charm-free packages, and build gobble's complex terminal elements in go-tui-lib

## Context and Problem Statement

The owner asked on 2026-10-22, while 0029-PLAN P2 (the grouped permission prompt) was about to start: "Evaluate
integrating the go-tui-lib sdk now so we can build the more complex cli and tui elements in a canonical manner", having
stopped that phase first ("Stop p2"). go-tui-lib is the owner's terminal-interface library
(`github.com/maccavelli/go-tui-lib`, `v0.12.0`, one module, built on Charm v2). This record is also the one
[0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md) promised, "written next", on the overlap of
go-tui-lib's `command` and `termcap` with gobble's `command` package and `internal/cli/term`; it was never written.

Every fact below was read on 2026-10-22, from gobble at `a35f682` and from go-tui-lib's `v0.12.0` tag (its imports
parsed from each package's non-test files, nothing built).

### What gobble's records decide

- **The core TUI is go-tui-lib, and gobble waits rather than copies**
  ([0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md), "TUI library"; `AGENTS.md`).
- **Rule 5:** "Only `internal/tui` imports `charm.land/...` and `github.com/maccavelli/go-tui-lib`", enforced by
  `internal/archtest` (`checkRule5`), with `tuitest` allowed in tests. `internal/tui` is an empty placeholder.
- **The native CLI is free of Charm by decision**
  ([0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) F5, D2, and Option B, "Allow Charm in
  `internal/cli`", rejected): `internal/cli` may import no go-tui-lib package, not even `glyph`, which needs only the
  standard library, so that the default mode does not carry the TUI's dependency graph. D8 and D11 leave a door open:
  a Charm-free go-tui-lib package, once tagged, may be admitted by a later record.
- **The TUI mode** is a 1.0 feature, chosen explicitly, which gets the transcript, tool cards, permission dialog,
  pickers, editor and completion "when the library provides them"
  ([0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) F9).

### What go-tui-lib offers at `v0.12.0`

| Package | Use | Charm modules it pulls in |
| :--- | :--- | :--- |
| `glyph`, `when`, `layout` | glyph sets; when-clause expressions; a pane solver | none |
| `command` | a command registry: `ParseArgs`, `Complete`, `WriteResult`, `ParseSlash`, ACP and MCP export, gate verdicts in ACP's permission kinds | Bubble Tea, `x/ansi` |
| `termcap` | `EnvCaps` (the environment's facts), `Report` (a doctor report), and a probe that runs inside Bubble Tea | Bubble Tea, colorprofile, `x/ansi`, ultraviolet (an untagged pseudo-version) |
| `termsvc` | notifications, clipboard (OSC 52), links (OSC 8) and prompt marks (OSC 133), as bytes for a program to write | Bubble Tea, `x/ansi` |
| `keymap` | keys to commands, chords, `keybindings.json` in VS Code's format | Bubble Tea, Bubbles, `x/ansi` |
| `launch` | `--tui` flags for Kong, `Decide`, `Run` with a fallback to the CLI on failure or crash | Bubble Tea, colorprofile, `x/term`, ultraviolet |
| `theme`, `workspace` | styles; the pane host | Lip Gloss; everything |
| `tuitest` | golden checks across colour, glyphs and widths | colorprofile, `x/ansi` |

`fuzzy` came after the tag, importing `x/ansi`. The library writes to no stream, installs no signal handler and
reads no environment, by its own conformance test; it needs no cgo, and its Go 1.27.2 floor is below gobble's 1.27.3.
Its records name gobble as a consumer (`launch` was designed for gobble's `--tui`).

**It has no component of the kind the owner means.** There is no prompt, menu, question, picker, confirm, form,
input editor, wrapping or transcript, for a line-mode CLI or a TUI. Its own architecture lists the dialog, picker,
palette, Markdown panes and inline scrollback as not built; only the command palette, with a `Picker` and a fuzzy
matcher, has a plan in progress (for `v0.13.0`), and streaming Markdown one proposed.

### So

gobble's native CLI cannot use any of it today, since everything useful links Bubble Tea; and what gobble wants to
build next, the grouped permission prompt among the first, exists in neither gobble nor go-tui-lib. gobble's own
rule, wait rather than copy, and the owner's ownership of both repositories, point the same way: the canonical home
for these elements is go-tui-lib.

## Decision Drivers

* The owner's direction: the more complex CLI and TUI elements built once, canonically.
* 0004-MADR: gobble waits for the library rather than copying it.
* 0008-MADR F5's reason: the default mode does not carry the TUI's dependency graph, nor an untagged module.
* Both modes, the line-mode CLI and the TUI, served by one implementation of each element.

## Considered Options

* A. Charm-free go-tui-lib packages in the native CLI; the components built in go-tui-lib, gobble waiting.
* B. Admit Charm into the native CLI, and use go-tui-lib as it is.
* C. Keep rule 5 as it is: go-tui-lib in the TUI mode only; the native CLI's elements stay gobble's own.

## Decision Outcome

Chosen option: "A", the owner's answers of 2026-10-22: "Charm-free packages only", "In go-tui-lib; gobble waits", and
for 0029-PLAN P2 "Becomes go-tui-lib's first inline component". It keeps the default binary free of Charm, puts each
element in one place for both modes, and follows gobble's rule of waiting for the library.

- **D1. The boundary.** Rule 5 is amended: `internal/cli` and its packages may import a go-tui-lib package whose whole
  import closure holds no Charm module (`charm.land/...`, `github.com/charmbracelet/...`); everything else of
  go-tui-lib, and Charm itself, stays in `internal/tui`. `internal/cli` may not import `internal/tui`, which would
  bring Charm in through it. `archtest` checks the closure, not only the direct import.
- **D2. What go-tui-lib is asked for, Charm-free.** The headless parts gobble's native CLI would use, each in a package
  that links no Charm module: the environment's terminal facts and the doctor report (`termcap`'s `EnvCaps` and
  `Report`); the sequences for notifications, links, clipboard and prompt marks (`termsvc`'s byte functions); the
  command registry with its parsing, completion and result writing (`command`, without its Bubble Tea messages);
  `keybindings.json` loading (`keymap`'s VS Code loader); and fuzzy matching.
- **D3. The complex elements are built in go-tui-lib,** each for both a line-mode CLI (drawn inline, with no full
  screen, in the CLI's scrollback) and the TUI, and gobble adopts each when it is tagged: the inline question and
  menu, then the picker (models, sessions), the input editor, the transcript with streamed Markdown, and the permission
  dialog. Until one is tagged, gobble keeps what it has and builds no copy.
- **D4. The first: 0029's grouped permission prompt.** Its accepted layout
  ([0029-MADR-permission-prompt-layout.md](0029-MADR-permission-prompt-layout.md) D1, D2, D5, D6) is go-tui-lib's
  first inline component, a question with grouped rows of keyed options, wrapped, with nothing split. 0029-PLAN P2
  waits for its tag and then draws through it; 0029-PLAN P1, the agent's meta and the client's fields, stands.
- **D5. Requests, not copies.** gobble files each of D2 and D3 in go-tui-lib's tracker, with gobble's case and
  measurements, as it did for the two SDKs; the owner files them, or says to.
- **D6. The TUI mode is unchanged:** `internal/tui` may import any of go-tui-lib, as rule 5 has it, and 0005-PLAN F9
  builds it from the library's tags.
- **D7. Each adoption is its own plan step,** written when the tag exists: the pin (a module named by this record, in
  the commit that first imports it, as `AGENTS.md` requires), the gobble code it replaces, and that code removed.

### Consequences

* Good, because each element is built once, for both modes, in the library both are drawn with.
* Good, because the default binary stays free of Charm, and of ultraviolet's untagged pseudo-version.
* Good, because the overlap 0004-PLAN left open is decided.
* Bad, because gobble's permission prompt, pickers and editor wait for go-tui-lib's tags; the prompt stays one line
  until then.
* Bad, because go-tui-lib takes on line-mode (inline) drawing, which it does not do today.
* Neutral, because `archtest` grows a closure check.

### Confirmation

0032-PLAN's `archtest` change, seen failing first on a fixture that imports a go-tui-lib package with Charm in its
closure, and on one that imports `internal/tui` from `internal/cli`; the requests filed; each later adoption by its own
plan step, with the gates on macOS, the Linux machine and the Windows laptop.

## Pros and Cons of the Options

### A. Charm-free packages in the CLI; the components in go-tui-lib

* Good, because it keeps 0008-MADR F5's reason and opens the door its D8 and D11 left.
* Bad, because nothing is usable until go-tui-lib splits its packages and builds the components.

### B. Admit Charm into the native CLI

* Good, because `command`, `termcap`, `termsvc`, `keymap` and `launch` are usable at once.
* Bad, because Bubble Tea, colorprofile and an untagged ultraviolet link into every gobble binary, against 0008-MADR F5.
* Bad, because it gives no component either: the library has none to use.

### C. go-tui-lib in the TUI mode only

* Good, because it changes no rule.
* Bad, because the native CLI's elements stay gobble's own, built a second time for the TUI.

## More Information

- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): rule 5, and "wait rather than copy";
  [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md): the overlap record promised.
- [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md): F5, D2, D8, D11 and Option B.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md): F9, the TUI mode.
- [0029-MADR-permission-prompt-layout.md](0029-MADR-permission-prompt-layout.md): the first component.
- The plan: [0032-PLAN-go-tui-lib-integration-boundary.md](0032-PLAN-go-tui-lib-integration-boundary.md).
