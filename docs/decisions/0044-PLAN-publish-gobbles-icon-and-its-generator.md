---
status: in-progress
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
   * *Deviation 1 (2026-11-07):* one change to RDP. Of the distances within 1e-9 of the largest, the first is kept,
     so the platform's maths library no longer picks among ties (MADR D2, as amended).
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
   changed the geometry or the bytes. *Deviation 1 (2026-11-07): the sha256 is the re-approved `5c7e6c82…`.*
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

### Deviation 1 (2026-11-07): the approved bytes are not the same on Linux

**Found at P1 step 4.** Steps 1 to 3 had passed on macOS: the port wrote the approved bytes (sha256 `8e136fa1…`,
2,462 bytes, `cmp` identical to the session's output), and every planted input failed as step 3 lists. On the Linux
dev host, the same script, under the same Python (3.14.10), wrote sha256
`2aaa88180480228575b7b606087ef47618f7f974d5dfee20a51e868690bff35b`, also 2,462 bytes. The Windows dev host wrote the
approved bytes.

**Cause.** The fan's scalloped outer edge is symmetric, so Ramer-Douglas-Peucker meets ties: 33 splits whose two
largest distances are equal (difference `0.0`) or differ by about 1e-14. `max` keeps whichever is a bit larger, which
the last bit of the platform's `sin` and `cos` decides, and Linux's maths library decides two of them differently. Two
of the fan's points moved: `4.46 10.98` to `4.48 10.95`, and `6.85 8.31` to `6.82 8.33`. This is the risk MADR D3
named; its edges did not list a tie between two candidates.

**Tried, on all three platforms** (scratch scripts, not committed). Each gave one hash on all three, and none gave the
approved one:

| Variant | sha256 | Bytes |
| :--- | :--- | :--- |
| the first of the distances within 1e-9 of the largest | `5c7e6c82…` | 2,462 |
| the middle one of those | `5c7e6c82…` | 2,462 |
| the last one of those | `e7654139…` | 2,465 |
| each point rounded to 1e-6 before simplifying | `e80f41ad…` | 2,463 |

**Options put to the owner.** (1, recommended) the first of tied distances; (2) rounding to 1e-6, a bigger change for
no gain; (3) `sin`, `cos` and `pow` written in Python, which still changes the bytes and leaves the ties. Not offered,
as workarounds: running the check on fewer platforms, a tolerance, or no check (MADR option B).

**Decision (owner, 2026-11-07): option 1,** "take option 1, continue". Against the approved icon, one point moves,
`6.85 8.31` to `6.82 8.33` (0.036 units), on the fan's upper edge. Rendered at 24 px the two are pixel-identical; at
48 px one pixel differs by 1/255. The icon is now sha256
`5c7e6c823f2d6ce65cb3d318d85c74378c40485b5fac6b7a6d81124103154685`, 2,462 bytes, and the MADR is amended (Facts, D1,
D2, D3, Confirmation 1). P1 step 1 gains the rule; steps 2 and 4 compare with the new sha256. No file joins the scope.

### P1 record (2026-11-07)

On the owner's "committed, proceed" after P0 (`6c6c7a7`). Start state (C1): `main` at `6c6c7a7`, `origin/main` at
`90b015a`, the tree clean; the session's script present; `rsvg-convert` and `magick` on `PATH`.

**Step 1, the port.** `scripts/gen_icon.py`, mode 100644 like the other Python scripts, in `gen_catalog.py`'s form. The
geometry is the session's, moved into `fan_paths()` and `build()` with the same expressions in the same order. The SVG
is written with `write_bytes`, and `--check` compares `read_bytes`. `--check` and `--preview` are an argparse mutually
exclusive group. Deviation 1 then added `TIE` and its rule to `rdp`.

**Step 2, the icon.** Before deviation 1: sha256 `8e136fa1…`, 2462 bytes, last byte `>`, `cmp` identical to the
session's output. After it: `5c7e6c823f2d6ce65cb3d318d85c74378c40485b5fac6b7a6d81124103154685`, 2462 bytes, last
byte `>`, and `cmp` identical to the deviation's trial output. `--check` exits 0.

**Step 3, seen failing** (C2), by `scratchpad/icon_seenfail.py` (not committed), each case on a fresh copy of the tree,
run before deviation 1 and again after it, with the same results:

| Planted | Exit | stderr |
| :--- | :--- | :--- |
| one byte of the SVG (`gobble` to `gObble` in the title) | 1 | `gen_icon: assets/gobble.svg differs from the generator; run scripts/gen_icon.py` |
| `GAP = 1.0` to `0.9` | 1 | the same |
| the SVG deleted | 1 | the same |
| a newline appended to the SVG | 1 | the same |
| the tie rule taken out (after deviation 1 only) | 1 | the same |
| `--preview`, the absolute interpreter, `PATH` an empty directory | 2 | `gen_icon: --preview needs rsvg-convert, which is not on PATH`, after `gobble.svg: 2462 bytes` on stdout |
| nothing | 0 | empty |

**Step 4, three platforms.** Each dev host got a scratch directory holding only `scripts/gen_icon.py` and an empty
`assets/`. Python 3.14.10 on all three (macOS; Linux x86_64; Windows 11 under Git Bash, where `python3` and `python`
are the same 3.14.10). The first run found deviation 1. After it, all three wrote
`5c7e6c823f2d6ce65cb3d318d85c74378c40485b5fac6b7a6d81124103154685`, and `--check` exited 0 on each. CI's runners have
Pythons of their own; P2 step 1 checks them.

**Step 5, the gate.** `Makefile`: `check-icon` (`$(PYTHON) scripts/gen_icon.py --check`, its comment naming 0044-MADR
D3), in `.PHONY`, and in `preflight` between `check-records` and `markdownlint`.

**Step 6, the guide.** `docs/guides/developing.md`: `check-icon` is item 9 of "The gates", `markdownlint` 10; a new
section, "Change the icon", before "Decision records".

**Step 7, gates.**

* `make check-icon`: exit 0.
* `python3 scripts/gen_icon.py --preview DIR`: exit 0, and `DIR` holds `gobble-512.png`, `gobble-72.png`,
  `gobble-48.png`, `gobble-24.png`, `gobble-48-x.png` and `gobble-24-x.png` (Confirmation 5). The 512 px render was
  looked at: the turkey as approved.
* `make preflight`: "preflight passed", with `==> check-icon` and `python3 scripts/gen_icon.py --check` between
  `check-records` and `markdownlint`; markdownlint 0 issues (Confirmation 4, locally).
* The identifier scan of every changed file found nothing.
* The change is `Makefile`, `docs/guides/developing.md`, `docs/README.md`, this pair, and the new `assets/gobble.svg`
  and `scripts/gen_icon.py`: no Go file and nothing under `acpext/` (Confirmation 6).

**Step 8** is this record and the `docs/README.md` rows. Staged; the owner commits. P2 waits for the push.
