---
status: proposed
date: 2026-11-07
associated-madr: "0044-MADR-publish-gobbles-icon-and-its-generator.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement publishing gobble's icon, a cartoon turkey, as `assets/gobble.svg`, with its generator in `scripts/gen_icon.py`

Associated MADR: [0044-MADR-publish-gobbles-icon-and-its-generator.md](0044-MADR-publish-gobbles-icon-and-its-generator.md)

## Goal

`assets/gobble.svg` holds the approved turkey, byte for byte (MADR D1). `scripts/gen_icon.py` writes it, checks it and
previews it (D2). `make preflight` runs the check, proven byte-identical on macOS, Linux and Windows first (D3).
`docs/guides/developing.md` says how to change the icon. The plan is complete when the MADR's Confirmation 1 to 6 is
met.

## Scope

**Base.** `main` at `90b015a`, equal to `origin/main`, clean.

**Source of the port.** The script that drew the approved icon, in the magic-cli-remote session's scratchpad. It is
not committed anywhere, so P1 step 1 carries it into this repository and the MADR's sha256 is the check that nothing
changed on the way. The scratchpad is under the system's temporary directory, which a reboot clears, so P1 should run
before the host next restarts. If the script is lost first, the sha256 cannot be reproduced and P1 stops as a
deviation (C4).

**In scope:**

* `assets/gobble.svg`, new;
* `scripts/gen_icon.py`, new;
* `Makefile`: a `check-icon` target, its `.PHONY` entry, and one line in `preflight`;
* `docs/guides/developing.md`: the gate in "The gates", and a new section, "Change the icon";
* `docs/README.md`: the two 0044 rows, and one "I want to…" row;
* this pair.

**Out of scope:** any Go file, `go.mod`, `acpext/`, the workflows, and a release tag. Magic's side is magic-cli-remote
`0179-PLAN-gobble-native-acp-provider.md`'s.

## Conventions every phase follows

* **C1. Start state.** Each phase first re-reads `git status`, `HEAD` and `origin/main`. A difference from this plan is
  a deviation.
* **C2. Seen failing.** Every new check (`--check`, the `--preview` tool check, the sha256 comparison) is seen failing
  on a planted input in a scratch copy, never in the working tree.
* **C3. Evidence.** Commands write to a scratch log, with their exit status taken before any filter. The phase record
  quotes the lines it relies on.
* **C4. Deviations stop the phase,** and are recorded here, and in the MADR when a decision or fact changes, before
  work goes on.
* **C5. Records and commits.** `make check-records markdownlint` passes, and the identifier scan and the disclosure
  guard's redact preview find nothing, before staging. The agent stages; the owner commits with
  `git commit --no-edit`.
* **C6. Identifiers.** No hostname, account name or real-machine path in a committed file. The dev hosts in P1 step 4
  are named in this record by platform only.

## Implementation Steps

### P0 — the records

Files: this pair; `docs/README.md`.

1. `docs/README.md` gains the two 0044 rows, `proposed`, and an "I want to…" row: "change gobble's icon, or know where
   consumers get it".
2. `make check-records markdownlint` passes.
3. Staged for the owner's review. The owner commits.

### P1 — the icon, its generator and its gate

**Gate:** P0 committed, and the owner's approval of this plan.

Files: `scripts/gen_icon.py`, `assets/gobble.svg`, `Makefile`, `docs/guides/developing.md`, `docs/README.md`, this
plan.

1. **The port.** `scripts/gen_icon.py`, in `gen_catalog.py`'s form: a module docstring naming this pair, its outputs
   and its usage; `from __future__ import annotations`; `argparse`; `pathlib`; functions with one-line docstrings.
   * The geometry is copied unchanged: every constant, every `assert`, the 480-step fan, RDP at 0.03, two-decimal
     rounding, and the path order.
   * The output path is `assets/gobble.svg`, found from the script's own location, as `gen_catalog.py` finds `DATA`.
   * The SVG is written with `write_bytes(svg.encode("utf-8"))`: one line, with no newline at the end (MADR D1, D2).
     The session's `write_text` is not kept, since its encoding and newline depend on the platform.
   * `--check`: build the SVG in memory and compare its bytes with the file's (`read_bytes`). On a difference, print
     `gen_icon: assets/gobble.svg differs from the generator; run scripts/gen_icon.py` to stderr and exit 1. A missing
     file is a difference.
   * `--preview DIR`: after writing the SVG, write `gobble-512.png`, `gobble-72.png`, `gobble-48.png`,
     `gobble-24.png`, `gobble-48-x.png` and `gobble-24-x.png` into `DIR`. Look up `rsvg-convert` and `magick` with
     `shutil.which` first; when one is missing, print its name and exit 2.
   * `--check` and `--preview` are mutually exclusive.
2. **The icon.** Run `python3 scripts/gen_icon.py`. `shasum -a 256 assets/gobble.svg` must print the MADR's sha256
   (Confirmation 1), `wc -c` must print 2462, and the last byte must be `>`. A different hash stops the phase: the port
   changed the geometry or the bytes.
3. **Seen failing** (C2), on a scratch copy of the repository with the new files:
   * one byte of `assets/gobble.svg` changed: `--check` exits 1 and names the file;
   * `GAP = 1.0` changed to `0.9`: `--check` exits 1;
   * `assets/gobble.svg` deleted: `--check` exits 1;
   * `--preview`, run by the interpreter's absolute path (`sys.executable`) with `PATH` set to an empty directory:
     exits 2 and names `rsvg-convert`. Nothing but the interpreter is then reachable, with no link to make.
   * a newline appended to `assets/gobble.svg`: `--check` exits 1;
   * the unchanged copy: `--check` exits 0.
4. **The same bytes on three platforms** (D3, Confirmation 3). On the Linux dev host and on the Windows dev host
   (under Git Bash's Python), make a scratch directory holding `scripts/gen_icon.py` and an empty `assets/`. Run the
   script there, and compare the sha256 of the `assets/gobble.svg` it writes with step 2's. The first time each host is used, check its Python
   version and record it. A different hash stops the phase. The fix goes in the script, and the MADR is amended; the
   check is never loosened.
5. **The gate.** In the `Makefile`:
   * a `check-icon` target, `$(PYTHON) scripts/gen_icon.py --check`, with a comment naming 0044-MADR D3;
   * `check-icon` in `.PHONY`;
   * `@echo "==> check-icon"; $(MAKE) --no-print-directory check-icon` in `preflight`, between `check-records` and
     `markdownlint`.
6. **The guide.** In `docs/guides/developing.md`:
   * "The gates" gains `check-icon`, the icon matches its generator, as item 9; `markdownlint` becomes 10;
   * a new section, "Change the icon", before "Decision records": the icon is generated, never hand-edited; edit the
     constants in `scripts/gen_icon.py`; run it with `--preview DIR` and look at the 24 and 48 px enlargements; commit
     the script and `assets/gobble.svg` together; consumers pick it up at the next `v*` release (MADR D4).
7. **Gates.** `make preflight` passes, with `==> check-icon` in its output (Confirmation 4, locally). The
   `--preview` run writes six PNGs (Confirmation 5). `git diff --cached --stat` lists only P1's files (Confirmation 6).
8. **Records.** This plan's execution record. The `docs/README.md` rows are updated. Staged; the owner commits.

### P2 — acceptance

**Gate:** P1 committed and pushed on the owner's word.

1. CI's three `make preflight` jobs pass on the pushed commit, each with `==> check-icon` in its log (Confirmation 4).
2. The plan's status becomes `complete`, and the MADR's `accepted`, with the `docs/README.md` rows to match.
3. magic-cli-remote is told which release carries the file, once the owner tags one. Its 0179-PLAN pins that tag.

## Verification

| Confirmation | Where |
| :--- | :--- |
| 1. The sha256 | P1 step 2 |
| 2. `--check`, passing and seen failing | P1 step 3 |
| 3. The same bytes on three platforms | P1 step 4 |
| 4. Preflight runs `check-icon`; CI passes | P1 step 7, P2 step 1 |
| 5. `--preview`, and its missing-tool exit | P1 steps 3 and 7 |
| 6. No Go or `acpext/` change | P1 step 7 |

## Rollout and Rollback

* **Rollout.** The file and script land on `main` in P1. The icon reaches consumers at the next `v*` release, which
  this plan does not make.
* **Rollback.** Before a release: revert P1's commit. After a release, the file stays at that tag (MADR D4), and a
  change is a new commit and a new release. If the gate disagrees across CI's platforms after P1, the fix is the
  script's (C4); removing the gate is not a rollback this plan offers.

## Execution record

Not started.
