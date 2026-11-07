---
status: accepted
date: 2026-11-07
decision-makers: repository owner
consulted: "this repository at 90b015a (main, 2026-11-07); magic-cli-remote's tools/vendor-icons/sync.sh, its 67 bundled vendor icons and its 0179-MADR-gobble-native-acp-provider.md (the 2026-11-06 body, D11 and Q4); the icon drawn and approved in a magic-cli-remote session on 2026-11-06 and 2026-11-07"
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Publish gobble's icon, a cartoon turkey, as `assets/gobble.svg`, with its generator in `scripts/gen_icon.py`

## Context and Problem Statement

magic-cli-remote ("Magic") is registering gobble as a provider (magic-cli-remote
`0179-MADR-gobble-native-acp-provider.md`). Its phone app shows one small monochrome icon for each provider, and gobble
has none: this repository holds no image file at all (`git ls-files -- '*.svg' '*.png' '*.ico'` is empty at
`90b015a`).

On 2026-11-06, in a magic-cli-remote session, the owner asked for one: "Gobble gobble. Turkey. Bird. Cartoonish. An
icon styled and optimized cartoon turkey is the icon. Make it." It was drawn there, by a Python script, as a cartoon
turkey in profile. On 2026-11-07 the owner approved it as drawn ("No tweaks required"), and asked where it should live:

> would it make more sense to move it into the gobble-cli sources with the generate script there so if i wanted to
> change it, that would happen there then magic-cli-remote would just import the svg from gobble-cli sources rather than
> have to regenerate it every every run?

The question for this record is where gobble's icon and its generator live in this repository, what a consumer may rely
on, and what keeps the two in step.

### Facts

* **The icon,** as approved: 2,462 bytes, sha256 `8e136fa1585b183ffacd8d2cfe9bf9a0db2974c801d8f1ebc41078c900a6308b`.
  *Amended 2026-11-07: those are the bytes macOS and Windows draw; Linux draws two points differently. The owner
  re-approved the icon as sha256 `5c7e6c823f2d6ce65cb3d318d85c74378c40485b5fac6b7a6d81124103154685`, also 2,462 bytes,
  the same on all three (see "Amendment (2026-11-07)").*
  * One `<svg>` with seven attributes, in this order: `fill="currentColor"`, `fill-rule="evenodd"`, `height="1em"`,
    `style="flex:none;line-height:1"`, `viewBox="0 0 24 24"`, `width="1em"` and
    `xmlns="http://www.w3.org/2000/svg"`. Then `<title>gobble</title>`, eight `<path>` elements, and nothing else.
  * The file is one line of ASCII, with no line break and no newline at the end.
  * The paths are the tail fan with a stripe cut out, the body with a wing cut out, the head with an eye cut out, the
    neck, the beak, the wattle and two legs.
  * Its detail is negative space, so it survives a single-colour tint.
* **How Magic draws icons.** Magic's phone tints each icon with `ColorFilter.mode(onSurface, BlendMode.srcIn)` at 24,
  28 and 32 dp. Its 67 bundled icons come from Lobe Icons (`@lobehub/icons-static-svg`, MIT), monochrome variants only.
  In 66 of the 67, the root element has the same attributes, in the same order, as the turkey's, apart from the
  title. The odd one out is `nova.svg`, with `fill="none"` and a 33 by 32 `viewBox`.
* **How Magic imports icons.** `tools/vendor-icons/sync.sh` downloads the Lobe Icons npm tarball at a pinned version
  (`LOBE_ICONS_VERSION=1.94.0`), then deletes and rewrites `apps/mobile/assets/vendor_icons/`. It also writes a
  generated Dart manifest, and the owner commits the result. Nothing in Magic draws an icon.
* **The generator,** as it ran in that session:
  * one Python file, standard library only for the SVG itself;
  * it computes the fan, its stripe and its gaps from a few constants (the body ellipse, the head circle, the fan's
    centre, span, lobe count, radius and scallop depth, and the gap);
  * it checks the geometry with `assert`s: the fan is never thinner than 0.6 units, and nothing covers the eye or the
    wing;
  * it simplifies each curve with Ramer-Douglas-Peucker at 0.03 units and writes paths rounded to 0.01;
  * it writes PNG previews with `rsvg-convert` and ImageMagick. Those two are optional tools, and only the previews
    need them.
* **This repository's scripts.** `scripts/` already holds Python generators and checks: `gen_catalog.py`,
  `check_records.py`, `check_libs.py`, `verify_build_metadata.py`. `.gitignore` already ignores `__pycache__/` and
  `*.py[cod]` (`docs/decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md` D4). The Makefile passes
  `$(PYTHON)` (`python3`, or `python` on Windows CI).
* **CI.** `ci.yml` runs `make preflight` on `ubuntu-24.04`, `macos-15` and `windows-2025`. Its `paths-ignore` lists
  only Markdown and the agent rules, so a change to `assets/` or `scripts/` runs the full CI.
* **Tags.** `v*` tags carry immutable releases, and `acpext/v*` tags have a ruleset with no bypass
  (`docs/decisions/0043-MADR-protect-main-and-acpext-tags-without-bypass.md`). A file at a released `v*` tag does not
  change.
* **acpext.** `acpext/` holds "names, types and fixed lists, and no behaviour"
  (`docs/decisions/0042-MADR-acpext-extension-contract-module.md`; `acpext/doc.go`), and is tagged on its own.

## Decision Drivers

* **One source for the mark.** gobble's icon belongs to gobble. A change to it is made here, once, and reaches every
  consumer through a release.
* **Consumers import a file, not a script.** Magic copies `gobble.svg` as it copies every Lobe icon. Nothing outside
  this repository runs the generator.
* **Regenerating is deliberate.** The generator runs only when someone changes the turkey. Its output is committed, and
  no build, release or sync runs it.
* **The two cannot drift.** A hand edit to the SVG that the script would not produce is caught before it is committed.
  Otherwise the next regeneration would silently lose it.
* **Nothing else moves.** No Go code, module, `go.mod` or `acpext` change. The binary does not embed the icon.

## Considered Options

* **A.** `assets/gobble.svg` and `scripts/gen_icon.py`, with a `--check` mode run by `make preflight`.
* **B.** As A, with no check.
* **C.** Both inside `acpext/`, so Magic's acpext requirement also pins the icon.
* **D.** Both under `docs/brand/`.
* **E.** The SVG alone, hand-edited from now on, with no generator.

## Decision Outcome

Chosen option: **A**, because it is the only option that keeps the generator here, keeps it in step with the file, and
leaves acpext and the build alone. The owner chose the location and Magic's import on 2026-11-07 (see "Owner answers").

### D1. The file

* `assets/gobble.svg`, byte for byte the ~~approved icon (sha256 above)~~ re-approved icon, sha256 `5c7e6c82…`
  (amendment of 2026-11-07).
* Its root element keeps the Lobe Icons monochrome form: the seven attributes above, in that order, then a `<title>`.
  Its detail stays as cut-outs.
* It stays one line of ASCII with no newline at the end. `.gitattributes`' `eol=lf` then has nothing to convert, so a
  Windows checkout holds the same bytes.
* It is covered by the repository's `LICENSE`, like every other file. No separate notice is added.
* No Go file reads it, and `cmd/gobble` does not embed it.

### D2. The generator

* `scripts/gen_icon.py`, the session's script ported to this repository's script form: a module docstring with usage,
  `argparse`, `pathlib`, and the standard library only.
* With no flag, it writes `assets/gobble.svg`, as UTF-8 bytes with no newline at the end (`write_bytes`, as
  `gen_catalog.py` writes its output). The bytes then depend on neither the platform's default encoding nor its
  newline.
* `--check` writes nothing. It compares bytes, and exits 1, naming the file, when the committed file differs from what
  the script would write, and 0 when they match.
* `--preview DIR` also writes PNGs at 512, 72, 48 and 24 px, and pixel-enlarged 48 and 24 px copies, into `DIR`. It
  needs `rsvg-convert` and ImageMagick's `magick`. When either is missing it exits 2 and names it, having written the
  SVG first. Without `--preview`, neither tool is needed.
* The geometry `assert`s stay, so a change that breaks the fan or covers the eye or the wing fails loudly.
* The constants keep their names and values, so the ~~approved~~ re-approved bytes come out unchanged.
* *Amended 2026-11-07:* Ramer-Douglas-Peucker keeps the first of the distances within 1e-9 of the largest, rather than
  whichever the platform's maths library makes a bit larger. This is the one change to the session's geometry.

### D3. The gate

* `make check-icon` runs `$(PYTHON) scripts/gen_icon.py --check`. `make preflight` runs it, between `check-records`
  and `markdownlint`.
* Because preflight runs on Linux, macOS and Windows, the output must be the same bytes on all three. The script uses
  `math.sin` and `math.cos`, and a float power for the feathers' shape (`lobe ** 0.45`). All three come from each
  platform's maths library, which may differ in the last bit. (`math.sqrt` is correctly rounded everywhere, and Python
  formats floats itself.) Such a difference changes the output only where it moves a value across an edge: a 0.01
  rounding boundary, Ramer-Douglas-Peucker's keep-or-drop test at 0.03, or the stripe's bounds test. That is unlikely
  but not proven. So the PLAN runs the script on all three before the gate is added, and CI then runs it on every push.
  *Amended 2026-11-07: the run on all three found a fourth edge, a tie between two candidates in Ramer-Douglas-Peucker,
  and Linux drew differently. D2's tie rule removes it (see "Amendment (2026-11-07)").*
* `docs/guides/developing.md` lists the gate, and says how to change the icon: edit the constants, run with
  `--preview`, look at the PNGs, and commit the script and the SVG together.

### D4. What a consumer may rely on

* At any `v*` tag that contains it, `assets/gobble.svg` keeps D1's form. Moving or renaming it is a breaking change
  for consumers, made only by a record that says so.
* Magic fetches the file from GitHub's raw content at a pinned `v*` tag, the way it pins Lobe Icons' version
  (magic-cli-remote `0179-MADR-gobble-native-acp-provider.md`, D11, as amended on 2026-11-07). Until a tag carries the
  file, Magic shows its generic monogram for gobble.
* The icon ships in the next `v*` release. No tag is made for it alone; tagging stays the owner's, under 0043.

### Consequences

* Good, because the turkey has one home, beside the product it marks, and changing it is one commit here and one pin
  bump in each consumer.
* Good, because the gate makes the SVG and its generator one thing. The generator's own `assert`s keep the geometry
  honest.
* Good, because acpext, the module graph and the binary are untouched.
* Neutral, because the full CI runs for an icon change. `paths-ignore` lists only Markdown, and an icon change is rare.
* Bad, because preflight gains a gate whose byte-identical output across three platforms rests on floating-point
  maths. The PLAN proves it on all three before adding the gate. If CI ever disagrees, the fix is in the script (for
  example, rounding before the simplification), never in loosening the check.
* Bad, because the icon reaches Magic only at the next release, so gobble shows a monogram there until then.

### Confirmation

1. `assets/gobble.svg`'s sha256 is the ~~approved one above~~ re-approved one, `5c7e6c82…` (amendment of 2026-11-07).
2. `python3 scripts/gen_icon.py --check` exits 0 on the tree. On a scratch copy, it exits 1 with the SVG changed by one
   byte, and again with one constant changed.
3. `scripts/gen_icon.py` writes the same bytes on macOS, Linux and Windows, run on each before the gate is added.
4. `make preflight` runs `check-icon` and passes; CI passes on all three runners.
5. `scripts/gen_icon.py --preview DIR` writes the six PNGs, and exits 2 naming the tool when one is missing (tried with
   a `PATH` that lacks it).
6. `git ls-files` lists no Go change and no `acpext/` change in the commit.

## Pros and Cons of the Options

### A. `assets/` and `scripts/`, with a check

* Good, because the generator sits beside this repository's other generators, and the file sits in a directory named for
  what it holds.
* Good, because the check catches drift on every preflight and every CI run.
* Bad, because preflight gains a gate, and its cross-platform byte identity has to be proven once (D3).

### B. As A, with no check

* Good, because it adds no gate.
* Bad, because a hand edit to the SVG, or a script change with no regeneration, goes unnoticed until the next person
  runs the script and silently loses or adds a change.

### C. Inside `acpext/`

* Good, because Magic's existing `acpext` requirement would pin the icon too, with one version for both.
* Bad, because acpext holds names, types and fixed lists, and no behaviour (0042-MADR). A brand asset and a Python
  script are outside that.
* Bad, because an icon change would need an `acpext/v*` tag, and Magic would have to bump a Go dependency to change a
  picture.

### D. `docs/brand/`

* Good, because it keeps the icon out of anything that looks like build input.
* Bad, because `docs/` holds records, guides and two index files, and nothing else (the fleet's documentation layout).
* Bad, because it separates the generator from `scripts/`, where every other generator is.

### E. The SVG alone

* Good, because it is the smallest change: one file.
* Bad, because the turkey is computed geometry: the fan's edges follow the body and the head with a fixed gap. A hand
  edit of 2,462 bytes of path data cannot change a proportion without redrawing it.
* Bad, because it is the opposite of what the owner asked for.

## More Information

### Amendment (2026-11-07): ties in the simplification

Found at 0044-PLAN P1 step 4, its deviation 1, which holds the evidence. The session's script, ported unchanged,
wrote the approved bytes on macOS and Windows, and different bytes on Linux under the same Python (3.14.10). The
fan's scalloped edge is symmetric, so Ramer-Douglas-Peucker meets ties between candidate points, equal or within
about 1e-14. `max` kept whichever the platform's `sin` and `cos` made a bit larger, and Linux's maths library chose
two differently.

No deterministic rule tried gave back the approved bytes. The owner chose the smallest change ("take option 1,
continue"): keep the first of the distances within 1e-9 of the largest (D2). One point of the fan's upper edge moves,
`6.85 8.31` to `6.82 8.33`. Rendered at 24 px the two icons are pixel-identical, and at 48 px one pixel differs by
1/255. The icon is now sha256 `5c7e6c823f2d6ce65cb3d318d85c74378c40485b5fac6b7a6d81124103154685`, 2,462 bytes, the
same on macOS, Linux and Windows. D3's cross-platform check stands unchanged.

### Owner answers, 2026-11-07

Asked in the magic-cli-remote session that drew the icon. Each question had three options and a recommendation.

* **Where in gobble-cli:** `assets/` and `scripts/` (recommended). Not chosen: inside `acpext`; `docs/brand/`.
* **How Magic imports it:** the raw file at a pinned tag (recommended). Not chosen: the Go module proxy; the raw file at
  a commit SHA.
* **Who makes the change here:** "Authorize me there". The agent writes this pair and, after approval, the files.
  Recommended had been an issue with the files attached. Not chosen: the owner copies them.
* **What Magic shows until a tag carries it:** the monogram, then the icon (recommended). Not chosen: hold Magic's
  registration for the tag; pin to the commit on `main`.

### Not decided here

* A colour variant, a wordmark, or a favicon for a website. None exists, and nothing asks for one.
* Whether `gobble` itself ever shows the icon (for example in a terminal that renders images). Nothing does today.

### Relationship to other records

* [0044-PLAN-publish-gobbles-icon-and-its-generator.md](0044-PLAN-publish-gobbles-icon-and-its-generator.md): this
  record's plan.
* [0003-MADR-gobble-product-identity.md](0003-MADR-gobble-product-identity.md): gobble's name and identity. It names no
  mark; this record adds one.
* [0007-MADR-repository-scaffolding-to-fleet-standard.md](0007-MADR-repository-scaffolding-to-fleet-standard.md): the
  Python ignore rules and the gates the new one joins.
* [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md): why the icon is not in
  acpext.
* [0043-MADR-protect-main-and-acpext-tags-without-bypass.md](0043-MADR-protect-main-and-acpext-tags-without-bypass.md):
  why a file at a released `v*` tag does not change.
* magic-cli-remote `0179-MADR-gobble-native-acp-provider.md`: the consumer. Its D11 and Q4 say how the icon is fetched.
