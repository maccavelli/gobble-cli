---
status: in-progress
date: 2026-10-29
associated-madr: "0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement pinning every sibling library's current tag, adopting what it provides, and removing gobble's copies

Associated MADR: [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md)
(accepted 2026-10-29; its D1–D18 and the owner's answers to Q1–Q6 are this plan's requirements).

## Goal

gobble pins these four tags, and drops the code each of them now replaces:

- acp-go-sdk2 `v1.26.0`;
- go-llmprovider-sdk `v1.24.1`;
- go-selfupdate-lib ~~`v1.22.3`~~ `v1.23.0` (deviation 1);
- `github.com/maccavelli/go-tui-lib/v2` ~~`v2.6.1`~~ `v2.6.2` (deviation 1).

The outcome:

- every gate passes on macOS, the Linux machine and the Windows laptop;
- `acpserver/testdata/session.golden` is byte-identical;
- the CLI's goldens change only at P18, and there only as named.

When it is complete, these are answered and can be closed by the owner:

- [gobble-cli#4](https://github.com/maccavelli/gobble-cli/issues/4)'s shipped items;
- [#5](https://github.com/maccavelli/gobble-cli/issues/5);
- [#6](https://github.com/maccavelli/gobble-cli/issues/6) items 1 and 2.

#6 item 3 and [#3](https://github.com/maccavelli/gobble-cli/issues/3), 0031-PLAN P4b, are not this plan's.

## Scope

- **In:** the files each phase names, and nothing else. A file a phase needs and does not name is a deviation (below).
- **Out:** everything in MADR D18, and:
  - every decision of [0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md](0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md), which starts after P5;
  - 0031-PLAN P4b;
  - 0033-PLAN;
  - 0016-MADR D8–D10;
  - adopting `tuitest`;
  - the SDK's `acptest`;
  - pushing and tagging. Pushing needs the owner's ask in the turn it happens (`AGENTS.md`).

## Conventions every phase follows

- **C1. Start state.**
  - The tree is clean.
  - `HEAD` is the previous phase's commit, or `ccd8aa8`'s descendant at P0.
  - `git rev-parse --path-format=absolute --git-path hooks` is the global hooks directory (checked once, at P0).
  - `go version` is `go1.27.3`.
- **C2. Tag drift.** Before a pin phase (P3, P4, P7, P14), `git ls-remote --tags https://github.com/maccavelli/<repo>`
  lists that library's tags. A tag newer than MADR D1's, of the same major, stops the phase for the owner (MADR D1).
- **C3. Scratch work.**
  - Every probe, and every deliberately broken copy, lives in the session's scratch directory, never in the tree.
  - A broken copy is made as follows:
    1. Write the tree's file list: `git ls-files -co --exclude-standard`.
    2. `tar` those files into `<scratch>/break-<phase>-<n>/`.
    3. Apply the named break there with an exact, asserted replacement: a Python script, old text present exactly
       once.
    4. Run the named command there. The copy's `go.mod` is the tree's.
  - The execution record keeps the command, its exit status and the failing lines.
- **C4. Long output.** Every gate writes its whole output to `<scratch>/<phase>-<gate>.log`, and its exit status is
  taken before any filter.
- **C5. Local gates (G-local),** run in this order, each passing:
  1. `make pre-add-check FILES="<every staged .go file>"`;
  2. `make preflight`;
  3. `make race`;
  4. for a phase that touches `internal/cli/...`, also `make perf`.
- **C6. Host gates (G-hosts).** On the Linux machine and the Windows laptop, on a copy of the tree, never their own
  checkouts:
  - `go test -count=1 ./...`;
  - `CGO_ENABLED=1 go test -race -count=1 ./...`;
  - `make preflight`, up to `verify-build-metadata`, which needs a commit and is therefore recorded as macOS-only;
  - `make check-records markdownlint`.

  A phase that touches `internal/cli/...` also runs `make perf` on both, and `make probe-conhost` on the Windows
  laptop. The copy recipe is 0031-PLAN's (its execution record), with the hostnames never written.
- **C7. Commit.** At the end of each phase, after G-local and G-hosts pass: `git add` of exactly the phase's files,
  then `git commit --no-edit`. Then the phase's execution record and this plan's `docs/README.md` row are updated, in
  the same commit or the next, as the phase says. Before the owner pushes, run the disclosure guard over the outgoing
  commits (`AGENTS.md`, "Identifiers").
- **C8. Deviations.** Any failure the phase does not name as expected, a file outside its list, or an API that is
  not as written here stops the phase. The owner is given:
  - the evidence: file, line, exact failure, and whether it is pre-existing;
  - resolutions, recommended first, each with its cost;
  - what doing nothing means.

  No workaround is offered. The chosen resolution is recorded as a dated deviation entry here, and in the MADR when a
  decision or fact changes, before work continues.
- **C9. Seen failing.** Every new test, case or gate is first run against a broken copy (C3) and seen to fail with
  the expected message. The execution record quotes that failure.
- **C10. Identifiers.** No hostname, account name, or absolute path of a real machine goes into a committed file, log
  excerpt included. Paths are quoted relative to the repository.

**The owner's answers, 2026-10-29** (three options offered per question, a recommendation first; each the
recommended one):

- **PQ1.** "Add --workspace (Recommended)": (a).
- **PQ2.** "Don't read it (Recommended)": (a).
- **PQ4.** "Take ask's behaviour (Recommended)": (a).
- **PQ5.** "Terminal paths (Recommended)": (a).
- **PQ6.** "Not adopted (Recommended)": (a).
- **PQ7.** "Take lineedit's (Recommended)": (a).

So the steps below stand as written, and no phase waits on a question.

## Order

*Deviation 1 (2026-10-29): the pin phases run first, in the order P0, P2, P3, P4, P7, P13, P14. Then P1, P5, P6,
P8–P12 and P15–P19 follow, in number order. See the execution record.*

*Deviation 11 (2026-10-30): P17 is held for go-tui-lib's `lineedit` key cost, [go-tui-lib#13](https://github.com/maccavelli/go-tui-lib/issues/13), and P18 and P19 wait
with it. See the execution record.*

P0 to P19, in order, by the owner's answer to Q6 ("Risk-first sequence"). A phase starts only on the owner's word for
it, or for a named range ("do phase N", "do P3–P6", "execute the plan").

- P6, P14, P15, P16 and P17 were written to PQ1, PQ2 and PQ4–PQ7's recommended answers, which the owner chose
  (above).
- P18 is written to PQ3's measurement.

## Questions this plan adds

- **PQ1. The key-only paths and the workspace (P6).** `gobble auth login claude --api-key -` and `gobble acp --login`
  write a key with no wizard (MADR D5, facts of 2026-10-29). The options:
  - **(a) Recommended.** A `--workspace <id>` flag on `gobble auth login`, accepted for the Claude provider only, with
    `gobble acp --login claude` asking for it after the key, empty for none. Both store it beside the key.
  - **(b)** No flag: a workspace comes only from `/login` and `gobble setup`, the wizard's paths.
  - **(c)** As (a), and the Pi import also copies a workspace, if Pi stores one. That is unknown; it would be
    checked first.
- **PQ2. `ANTHROPIC_WORKSPACE_ID` (P6).**
  - **(a) Recommended.** Not read: gobble sets neither `AllowEnv` nor `LookupEnv`, as today, so the wizard's workspace
    prompt has no default. This is a correction to Q5's answer as written, which assumed the default came free.
  - **(b)** Set `AllowEnv` and `LookupEnv`: the variable becomes the prompt's default. The wizard then also offers the
    environment's API key as a credential, which changes `/login`'s menu.
  - **(c)** gobble reads the variable itself, as the default of PQ1's flag and prompt only, and leaves the wizard as
    it is.
- **PQ3. Piped output (P18).** Answered by measurement in P18 step 1, under the owner's Q3 rule: piped output stays
  as it is today. No owner choice is needed unless the measurement finds a path the rule does not cover. Such a path
  is a deviation (C8).

  *The measurement, made 2026-10-29 while writing this plan (MADR D14's facts):*
  - print mode does not use `render`;
  - a piped line view is raw source, with no style, no layout and no wrap;
  - `CLICOLOR_FORCE=1` with a pipe is styled and laid out, but unwrapped.

  *P18 is written to these facts.*
- **PQ4. `ask` against 0029-MADR D1 (P14).** The four differences are listed in the MADR's facts under D14.
  - **(a) Recommended**, by the owner's Q3 direction to canonicalise. Take `ask`'s behaviour, and amend 0029-MADR D1 in
    P14's commit:
    - the reject row's "(hides the tool)" goes into that answer's label;
    - over-wide words break at the width;
    - the prompt glyph is `glyph`'s for the terminal's tier;
    - the reason prompt, echo and cancel texts come from gobble's voice catalogue through `WithCatalog` (0022-MADR).
  - **(b)** Keep 0029 D1 exactly, with a custom glyph set and labels, and file go-tui-lib requests for a row suffix
    and whole-word layout. P14 then waits on their tag.
  - **(c)** As (a), but keep `> ` through a custom glyph set.
- **PQ5. `safetext` (P15).**
  - **(a) Recommended.** `safetext` replaces `term.Sanitize` on every terminal path. Print mode and the piped line view
    keep `term.Sanitize`, so that their bytes do not change (Q3's guard). `sanitize.go` shrinks to that one function
    and `EscapeBidi` goes.
  - **(b)** `safetext` everywhere, print mode included. `sanitize.go` is deleted. Print mode's output changes for
    zero-width and tag characters, invalid UTF-8 and unterminated strings, and the guard is amended to "layout and
    styling only".
  - **(c)** Not adopted. `term.Sanitize` stays everywhere; only the dead `EscapeBidi` goes.
- **PQ6. `pick` (P16).**
  - **(a) Recommended.** Not adopted here. It replaces nothing (MADR D14's facts), so a picker is new UI, for MADR D18's
    list.
  - **(b)** `pick` for the login and setup menus on a terminal only, with the numbered question kept for a stdin that
    is not one.
  - **(c)** `pick` for those menus, and a new `/model` picker in the line session, in this plan.
- **PQ7. `lineedit`'s behaviour (P17).** The three differences are in the MADR's facts under D14.
  - **(a) Recommended**, by Q3's direction. Take the library's multiline buffer, its paste handling (`PasteInsert`)
    and its word moves. gobble keeps the history file, the external editor's process, the Ctrl+C window, `/exit`,
    `Raw`, the 120-cell cap and the paste notice. Editor tests asserting the old behaviour are changed, and each change
    is named.
  - **(b)** Keep gobble's behaviour where a host hook allows it, and file go-tui-lib requests for the rest. P17 waits on
    their tag.
  - **(c)** Defer `lineedit` to a record of its own. P17 is skipped and `internal/cli/editor` stays.

## Implementation Steps

### P0 — the records (MADR D15)

Files:

- `docs/decisions/0034-MADR-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md`;
- `docs/decisions/0034-PLAN-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md`;
- `docs/decisions/0031-MADR-sdk-current-releases-and-adoption.md`;
- `docs/decisions/0032-MADR-go-tui-lib-integration-boundary.md`;
- `docs/decisions/0032-PLAN-go-tui-lib-integration-boundary.md`;
- `docs/decisions/0029-PLAN-permission-prompt-layout.md`;
- `docs/decisions/0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md`;
- this pair;
- `docs/README.md`.

1. **0034 MADR and PLAN:**
   - Frontmatter `status: superseded by 0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md`,
     `date: 2026-10-29`.
   - Below the title, the note: *Superseded 2026-10-29, never executed. Its D2–D5 are 0035-MADR D7–D9, at
     acp-go-sdk2 `v1.26.0`.*
   - The body is unchanged.
2. **0031-MADR:**
   - At the end of D1's last amendment, a dated note naming the pins 0035 moves to. It says go-tui-lib's `v0.12.0`
     statement of 2026-10-24 no longer holds (newest `v2.6.1`, every request delivered), and that the record is 0035.
   - At D9's acp-go-sdk2 bullet, a dated note: carried out by 0035-MADR D7–D9.
   - Before writing either note, read 0031-MADR as it stands. Another session amended it on 2026-10-29, for 0036,
     with notes naming 0035 and the newest tags. A note this step would add that already says the same is not
     added twice, and its absence is recorded.
3. **0032-MADR:** an `## Amendment A2 (2026-10-29): the /v2 module and the packages now tagged` section, with:
   - the module path;
   - 0035-MADR's closure table, cited rather than copied;
   - that D3's components are tagged, with adoption by 0035-PLAN P13–P18;
   - that A1.2's module list is unchanged at `v2.6.1`.

   D1–D7 and A1 are not rewritten.
4. **0032-PLAN:** a dated note under "P3 and on": carried out by 0035-PLAN P13–P18. The status stays `in-progress`
   until P18, then becomes `complete` in P18's commit.
5. **0029-PLAN:** a dated note after "P2 stopped by the owner": P2 resumes as 0035-PLAN P14, through `ask` `v2.3.0`
   or later.
6. **0016-MADR:** a dated note at D13: carried out by 0035-MADR D7.
7. **`docs/README.md`:**
   - the rows of 0016, 0029, 0031, 0032 and 0034 name the notes;
   - the 0035 rows, added when this pair was written, are brought to the plan's state.
8. **Gates:** `make check-records markdownlint`, and `python3 scripts/check_records.py --check-all`. No Go file
   changes, so G-hosts is not run. This is records only, under the bootstrap exception.
9. **Commit.**

### P1 — the docs (MADR D16)

Files:

- `internal/cli/auth.go`;
- `internal/cli/auth_test.go`;
- `docs/guides/credentials.md`;
- `docs/README.md`.

1. **The OpenAI login message.**
   - `internal/cli/auth.go:192` becomes:
     `usageErrorf("openai's ChatGPT sign-in is not in gobble yet (0031-PLAN P4b); use --api-key -")`.
   - `internal/cli/auth_test.go:50`'s expected text changes to match.
2. **The credentials guide.**
   - `docs/guides/credentials.md:141-142` says the ChatGPT sign-in comes with 0031-PLAN P4b, not with an SDK
     release.
   - `:147`'s quoted error matches step 1.
3. **`docs/README.md`:**
   - "Build order" gains one bullet per PLAN from 0023 to 0035, in number order, after the 0022 bullet.
   - Each bullet states its status from that PLAN's frontmatter:
     - complete: 0023, 0024, 0027, 0028;
     - in progress: 0025, 0026, 0029, 0030, 0031, 0032;
     - proposed: 0033;
     - superseded: 0034;
     - 0035 is this plan.

     Each bullet's phases come from that PLAN's execution record, cited, not re-summarised.
   - The numbered list "The plans" gains item 6: "The pairs of 0023 to 0035", one line each.
   - The "I want to…" row for acp-go-sdk2 becomes "pin the sibling libraries' current tags and drop gobble's
     copies", pointing at 0035-MADR. The #5 link is kept.
4. **Seen failing (C9):** none is new. Step 1 changes an existing test's expectation, and the test fails on a copy
   with the old message.
5. **G-local, G-hosts. Commit.**

### P2 — check-libs sees what preflight sees (MADR D2)

Files: `scripts/check_libs.py`.

1. **`Library` gains three fields,** with defaults that leave the three existing entries unchanged:
   - `path`, the module path, defaulting to `github.com/maccavelli/<repo>`;
   - `major`, the major version whose tags count, defaulting to `None`, meaning v0 and v1;
   - `allow_new`, the modules a first import may add, defaulting to `()`.

   `module` returns `path`.
2. **`latest_tag(lib)`** keeps only tags whose major matches: `major == N` keeps `vN.*`, and `None` keeps `v0.*` and
   `v1.*`.
3. **`LIBRARIES` gains go-tui-lib:**

   ```python
   Library("go-tui-lib", ("./internal/cli/...", "./internal/archtest/..."), path="github.com/maccavelli/go-tui-lib/v2", major=2,
           probe=("ask", "safetext", "termcap", "glyph", "pick", "lineedit", "stream", "transcript"),
           allow_new=("github.com/charmbracelet/x/ansi", "github.com/clipperhouse/displaywidth",
                      "github.com/clipperhouse/uax29/v2", "github.com/lucasb-eyer/go-colorful", "github.com/mattn/go-runewidth"))
   ```

   `probe` is a fourth field, defaulting to `()`.
4. **A library not yet required.**
   - `pinned(lib)` returns `None` when `go list -m` says the module is not required.
   - `check_library` then writes `internal/cli/zz_checklibs_probe.go` in the scratch clone only. It is
     `package cli`, with a blank import of each `probe` package, and there is no such file in the tree.
   - It notes `pinned: not required; probing <n> packages`, and skips `api`, since there is no base.
   - `modules` allows `allow_new` and no other module.
   - That way `get and tidy` keeps the requirement, and `make archtest` in the clone checks the real closure.
5. **Two checks are added after `test`:**
   - `staticcheck`, running `make staticcheck`;
   - `lint`, running `make lint`.

   Both run in the clone, with their logs under `logs/`. `archtest` is added for go-tui-lib only, as `make archtest`.
6. **The self-test gains five offline cases:**
   1. `path` overrides `module`;
   2. `major=2` keeps `v2.6.1` and drops `v1.13.0` and `v0.12.0`, from a fixed tag list;
   3. `major=None` drops `v2.0.0`;
   4. `moved_modules` with `allow_new` passes the five modules and fails a sixth;
   5. the probe file's text for two packages is exactly the expected Go source.

   For this, `latest_tag`'s filter is a pure function of a tag list, and `moved_modules` takes `allow`.
7. **The docstring** names D2 and the two new checks. The usage line's `--lib` choices list four libraries.
8. **Verification:**
   - `python3 scripts/check_libs.py --self-test`: every case ok. Each new case is first seen failing on a copy
     (C9), with the filter inverted, then `allow` ignored.
   - **Live runs.** Each is a real run of the new steps against the real failures of MADR F1 and F2, and the
     execution record quotes them:

     | Command | Expected |
     | :--- | :--- |
     | `python3 scripts/check_libs.py --lib acp-go-sdk2 --tag v1.26.0` | `staticcheck: FAIL` and `lint: FAIL` (F2's 12 lines) |
     | `python3 scripts/check_libs.py --lib go-llmprovider-sdk --tag v1.24.1` | `test: FAIL` (F1's `TestWireOffline`); `staticcheck` and `lint` ok |
     | `python3 scripts/check_libs.py --lib go-selfupdate-lib --tag v1.22.3` | every check ok |
     | `python3 scripts/check_libs.py --lib go-tui-lib --tag v2.6.2` (deviation 1) | every check ok, `modules` ok with the five allowed, `archtest` ok |
9. **G-local.** `make preflight` lints no Python, so `python3 -m py_compile scripts/check_libs.py` is also run.
   G-hosts: the self-test on both hosts. **Commit.**

### P3 — go-selfupdate-lib `v1.22.3` (MADR D1)

*Deviation 1: the tag is `v1.23.0`, and the workflow pins are `67520be8af1ad1b6228c29e4b1aa3e61e1ca7ae4 # v1.23.0`.
Every step below reads with those values. Step 3's workflow assertion compares `v1.22.0` with `v1.23.0`.*

Files:

- `go.mod`, `go.sum`;
- `.github/workflows/ci.yml`;
- `README.md`, `docs/architecture.md`;
- `docs/decisions/0031-MADR-sdk-current-releases-and-adoption.md`, a one-line note at D1;
- `docs/README.md`.

1. C2. `python3 scripts/check_libs.py --lib go-selfupdate-lib --tag v1.22.3`: every check ok, as in P2.
2. `go get github.com/maccavelli/go-selfupdate-lib@v1.22.3`, then `go mod tidy`. `git diff --stat` shows only
   `go.mod` and `go.sum`.
3. **`ci.yml:166` and `:178`:**
   - `@84805563ed252df384f59ced45a3daf529702ab4 # v1.22.0` becomes
     `@0620e96ba7eb47d45cc94b474dc2db87d029ce40 # v1.22.3`.
   - Assert, by script, that the commit is `git ls-remote https://github.com/maccavelli/go-selfupdate-lib
     refs/tags/v1.22.3^{}`.
   - Assert that the two workflow files are identical between the tags (`git diff v1.22.0 v1.22.3 --
     .github/workflows/build-selfupdate-release.yml .github/workflows/publish-selfupdate-release.yml` in a scratch
     clone of the library is empty).
4. `README.md` "Stack" and the `docs/architecture.md` pin table show `v1.22.3`.
5. G-local, G-hosts. **Commit.**

### P4 — go-llmprovider-sdk `v1.24.1` and the offline row (MADR D1, D3)

Files:

- `go.mod`, `go.sum`;
- `llm/provider/provider.go`;
- `README.md`, `docs/architecture.md`;
- the 0031-MADR note;
- `docs/README.md`.

1. C2. `python3 scripts/check_libs.py --lib go-llmprovider-sdk --tag v1.24.1`: `test: FAIL` on `TestWireOffline`, as
   expected, and nothing else failing.
2. `go get github.com/maccavelli/go-llmprovider-sdk@v1.24.1`, then `go mod tidy`.
3. In `sentinels` (`llm/provider/provider.go:509-518`), insert `{llmprovider.ErrOffline, llm.ErrOffline},` directly
   above `{llmprovider.ErrProviderUnavailable, llm.ErrUnavailable},`.
4. **The case at `:632`.**
   - `case !kind && offline(err):` still wraps a non-SDK offline error. It stays until P5.
   - It must not wrap twice: with the row, `kind` is true for an SDK offline error, so the case is skipped.
   - `TestWireOffline` asserts `errors.Is(err, llm.ErrOffline)`, and a new assertion checks that `err.Error()` holds
     `"offline"` exactly once.
5. **Seen failing:** on a copy at the new pin without step 3's row, `go test ./llm/provider -run TestWireOffline`
   fails with MADR F1's message. With the row it passes.
6. `README.md`, `docs/architecture.md`: `v1.24.1`.
7. G-local, G-hosts. **Commit.**

### P5 — go-llmprovider-sdk's copies go (MADR D4)

Files:

- `llm/provider/provider.go`;
- `llm/provider/offline_test.go`;
- `llm/provider/provider_test.go`;
- `llm/doc.go`;
- `llm/llm.go`;
- `docs/guides/choosing-a-model.md`.

1. **Offline.**
   - Delete `offline()` and `unreachable()` (`provider.go:544-574`), and the `runtime` and `syscall` imports if they
     are unused afterward.
   - `mapError`'s `case !kind && offline(err):` goes. The `sentinels` row carries it.
   - At `:309`, the guard becomes `errors.Is(err, llmprovider.ErrOffline)`, recording `ue.URL` from the `*url.Error`
     as today.
   - `Probe`, `dialProbe`, `probeTimeout` and `offlineURL` stay.
2. **Empty answer.**
   - Delete `sdkEmptyAnswer`.
   - `emptyAnswer` becomes `errors.Is(err, llmprovider.ErrEmptyAnswer) && (reason == "" || reason == stop)`, the
     reason read from the `*llmprovider.APIError` as today.
   - `refusedAnswer` becomes the same with `reason == content_filter`.
   - `TestWireEmptyAnswer` stops pinning the message suffix (the comment at `offline_test.go:121`), and asserts
     `errors.Is(err, llm.ErrEmptyReply)` for an empty answer and `llm.StopRefusal` for `content_filter`.
3. **`max`.**
   - `reasoningFor`'s `"max": llmprovider.EffortXHigh` becomes `"max": llmprovider.EffortMax`.
   - `TestReasoningLevels`' expectation changes to match.
   - The SDK warns `WarningEffortDegraded` when it steps down. gobble does not read the SDK's warnings today, and
     [0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md](0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md) D3 makes them reach gobble, after this phase. So P5 builds no display, and 0036's P2 shows the
     degraded `max`.
4. **Docs:**
   - `llm/doc.go:47` becomes `max → max (the SDK steps down to the highest effort the model takes)`.
   - `docs/guides/choosing-a-model.md:42-43` says `max` is sent as `max`, and the provider steps it down where the
     model does not take it.
   - `llm/llm.go:45` reads `ImageInput stays false until gobble sends images (0035-MADR D18).`
5. **Seen failing:**
   - On a copy with `ErrOffline` removed from `sentinels`, `TestWireOffline` fails, which shows the row now carries
     offline alone.
   - On a copy with `refusedAnswer` reading `reason == ""`, `TestWireEmptyAnswer` fails.
6. Assert by search: no `sdkEmptyAnswer`, `func offline(` or `unreachable(` remains in `llm/`.
7. G-local, G-hosts. **Commit.**

### P6 — the Anthropic workspace (MADR D5; PQ1 (a), PQ2 (a))

Files:

- `internal/auth/store.go`, `internal/auth/store_test.go`;
- `internal/auth/resolve.go`, `internal/auth/resolve_test.go`;
- `llm/provider/provider.go`, `llm/provider/signin.go`, `llm/provider/list.go`, `llm/provider/model.go`, and their
  tests;
- `llm/errors.go`;
- `internal/cli/login.go`, `internal/cli/auth.go`, `internal/cli/acp.go`, `internal/cli/modelsjson.go`, and their
  tests;
- `acpserver/prompt.go`, for the wording;
- `docs/guides/credentials.md`.


1. **The store.**
   - `auth.Entry` gains `Workspace string`, JSON `workspace`, omitted when empty.
   - `Set` is replaced by `SetKey(ctx, path, id, key, workspace string)`. It merges `type`, `key` and `workspace`
     into the existing entry, keeping its unknown fields, as `SetOAuth`'s `merge` does (`store.go:206-284`). An
     empty workspace removes the field.
   - Every caller of `Set` moves to it:
     - `internal/cli/auth.go:123`;
     - `internal/cli/login.go:85`;
     - `internal/cli/acp.go:75`;
     - `internal/cli/piimport.go:197`, which passes `""`.
   - `TestStoreRoundTrip` gains a workspace. A new `TestStoreKeyKeepsUnknownFields` uses `piFile`'s api_key entry
     with `env`: `SetKey` keeps `env`. A file with no `workspace` reads as empty.
2. **The resolver.**
   - `auth.Credential` gains `Workspace`, set only when the key comes from the auth.json entry (`stored`,
     `resolve.go:122-136`).
   - A `--api-key` runtime key, a models.json key or an environment key carries none.
   - `TestResolveOrder` gains the case.
3. **The provider.**
   - `provider.Options` gains `Workspace string`.
   - `sdkOptions()` appends `llmprovider.For(llmprovider.ProviderClaude, claude.WithWorkspace(o.Workspace))` when it is
     not empty.
   - `List` appends `llmprovider.For(llmprovider.ProviderClaude, catalog.WithClaudeWorkspace(o.Workspace))`.
   - `lookupModel` passes the same option.
   - The workspace comes from the resolved credential where `providerOptions(id, o)` is filled
     (`internal/cli/modelsjson.go:117-122`), so that `agent.go:39`, `:57`, `title.go:39`, `models.go:138` and
     `login.go:73` all get it.
4. **The wizard.**
   - `signin.go` passes `Existing.Workspace` from the stored entry.
   - It returns `Credential{Key, Device, Workspace: res.Workspace}`.
   - `signInTo` (`login.go:68-89`) stores it.
5. **PQ1 (a).**
   - `gobble auth login` gains `--workspace`. It is refused with a usage error for any provider but Claude.
   - `gobble acp --login claude` prompts `Workspace ID (empty for none): ` after the key, reading one line as
     `readAPIKey` does.
   - The tests are in `internal/cli/auth_test.go` and the `acp --login` tests.
6. **Errors.**
   - `llm/errors.go` gains `ErrWorkspace`.
   - `sentinels` gains `{llmprovider.ErrWorkspaceRequired, llm.ErrWorkspace}` and
     `{llmprovider.ErrWorkspaceNotFound, llm.ErrWorkspace}`, above any `ErrInvalidRequest` handling.
   - `FailureKind` (`list.go:104-129`) and `StateOf` (`acpserver/choice.go:70-91`) gain a `workspace` class.
   - `promptError` (`acpserver/prompt.go:287-292`) words it: `"claude needs an Anthropic workspace for this key: run
     /login or gobble auth login claude --workspace <id>"`. The service's own message is not shown.
7. **`docs/guides/credentials.md`:** a "Workspaces" subsection covering the flag, the prompt and `/login`.
8. **Seen failing:**
   - On a copy with `Set`'s old replace kept, `TestStoreKeyKeepsUnknownFields` fails.
   - On a copy with `For(...)` dropped, a new `llm/provider` wire test, using a fake server that refuses without the
     `anthropic-workspace-id` header, fails with `llm.ErrWorkspace`. With it, it passes. The header's value is
     asserted.
9. G-local, G-hosts. **Commit.**

### P7 — acp-go-sdk2 `v1.26.0`, with the suppression and its gate (MADR D1, D6)

Files:

- `go.mod`, `go.sum`;
- the 8 files of MADR F2: `acpclient/acptest/script.go`, `acpclient/acptest/script_test.go`, `acpclient/conn.go`,
  `acpserver/agent.go`, `acpserver/agent_test.go`, `acpserver/mcp_test.go`, `acpserver/prompt.go`,
  `acpserver/prompt_test.go`;
- `.golangci.yml`;
- `Makefile`;
- `scripts/check_staticcheck_ignored.py`, new;
- `README.md`, `docs/architecture.md`;
- the 0031-MADR note;
- `docs/README.md`.

1. C2. `python3 scripts/check_libs.py --lib acp-go-sdk2 --tag v1.26.0`: `staticcheck: FAIL` and `lint: FAIL`, as P2
   saw, and nothing else failing.
2. `go get github.com/maccavelli/acp-go-sdk2@v1.26.0`, then `go mod tidy`.
3. **The directives.**
   - Above each of F2's 12 lines, at its indentation, insert
     `//lint:ignore SA1019 acp-go-sdk2 keeps this unstable-schema name under v1 (0035-MADR D6)`.
   - The lines are: `acptest/script_test.go:49`, `acptest/script.go:189` and `:190`, `conn.go:339`,
     `agent_test.go:144` and `:148`, `agent.go:746`, `mcp_test.go:200`, `prompt_test.go:140` and `:141`,
     `prompt.go:295` and `:296`.
   - Run `gofmt -w` on the eight files. `gofmt` moves a directive that follows a doc comment below a `//` line, as at
     `prompt.go:295` (probed 2026-10-29).
4. *~~Superseded by deviation 3: no `.golangci.yml` rule. Each line carries `//nolint:staticcheck // as the lint:ignore
   above` instead, the house idiom of 0002-PLAN Phase 5 deviation 3.~~ As written:* **`.golangci.yml`.** golangci-lint v2.14.0 does not honour `//lint:ignore` (probed 2026-10-29: all 12 still
   reported). After the `internal/config/genschema` rule, add:

   ```yaml
         # acp-go-sdk2's unstable-schema names (0035-MADR D6). make staticcheck bounds them:
         # scripts/check_staticcheck_ignored.py refuses any ignored finding outside them.
         - linters:
             - staticcheck
           text: "SA1019: .*github\\.com/maccavelli/acp-go-sdk2\\..* is deprecated: this declaration comes from the unstable ACP schema"
   ```

5. *Amended by deviation 3: the gate is record-based. Each `//lint:ignore` cites a NNNN-MADR or NNNN-PLAN record and is
   paired with a `//nolint:staticcheck`, each `//nolint:staticcheck` follows a `//lint:ignore`, and each ignored finding
   is under a directive of its own check. That holds every suppression in the module, the five older ones included.
   As written:* **`scripts/check_staticcheck_ignored.py`.**
   - It follows magic-cli-remote's script of the same name: the same two checks, the stdlib only, and
     `python3 -I`-safe. It is rewritten for gobble, not copied.
   - `REASON` is `"0035-MADR D6"`, `SDK` is `"github.com/maccavelli/acp-go-sdk2."`, and `NOTICE` is `"this declaration
     comes from the unstable ACP schema"`.
   - It reads directives from `git ls-files '*.go'`, and the ignored findings from
     `<staticcheck> -show-ignored -f json ./...`.
   - It exits 1 on a directive outside SA1019 or without the reason, or on an ignored finding outside the exception.
     It prints `check_staticcheck_ignored: <n> ignored finding(s), each an acp-go-sdk2 unstable-schema name`.
6. **`Makefile` `staticcheck`.** In the GOOS loop, after the staticcheck line, add
   `GOOS=$$goos CGO_ENABLED=0 $(PYTHON) scripts/check_staticcheck_ignored.py ./$(STATICCHECK_BIN) || rc=1; \`.
7. **Seen failing (C9), three copies:**
   1. A `//lint:ignore SA1019 no reason` added to a non-SDK line fails the script on the directive.
   2. A `//lint:ignore ST1000 0035-MADR D6` fails it on the check name.
   3. An ignore of `SA1019` on a deprecated standard-library call, `ioutil.ReadAll`, with the reason, fails it on the
      finding.

   Also, on a copy without step 4's rule, `make lint` reports 12 issues.
8. **Assert:** `make staticcheck` prints the script's line for each GOOS with the same count, and that count is the
   number of SA1019 records with severity `ignored` at that GOOS. The execution record keeps the three counts.
9. `README.md`, `docs/architecture.md`: `v1.26.0`.
10. G-local, G-hosts. **Commit.**

### P8 — the constructors (MADR D7)

Files: `acpserver/modes.go`, `acpserver/commands.go`, `acpserver/prompt.go`.

1. Replace, and only these:

   | Site | Today | Becomes |
   | :--- | :--- | :--- |
   | `modes.go:85` | `acp.SessionUpdate{CurrentModeUpdate: &acp.SessionCurrentModeUpdate{CurrentModeId: acp.SessionModeId(current)}}` | `acp.UpdateCurrentMode(acp.SessionModeId(current))` |
   | `commands.go:531` | `acp.SessionUpdate{ConfigOptionUpdate: &acp.SessionConfigOptionUpdate{ConfigOptions: opts}}` | `acp.UpdateConfigOptions(opts)` |
   | `commands.go:597` | `acp.SessionUpdate{SessionInfoUpdate: &acp.SessionSessionInfoUpdate{Title: acp.PatchValue(name)}}` | `acp.UpdateSessionTitle(acp.PatchValue(name))` |
   | `commands.go:728` | `acp.SessionUpdate{AvailableCommandsUpdate: &acp.SessionAvailableCommandsUpdate{AvailableCommands: …}}` | `acp.UpdateAvailableCommands(…)` |
   | `prompt.go:319` | `acp.SessionUpdate{UsageUpdate: u}`, after building `u` | `up := acp.UpdateUsage(used, size)`, then `up.UsageUpdate.Cost = …` when priced, then `return up` |

   The signatures are from `go doc` at `v1.26.0`, read again at the phase's start. One that differs is a deviation.
2. **Assert:**
   - `git diff --exit-code acpserver/testdata/session.golden`;
   - no `acp.SessionUpdate{` followed by one of those five field names remains in non-test `acpserver`;
   - `TestTranslate` and the `agent_test.go:124` golden test pass.
3. **Seen failing:** on a copy whose `UpdateUsage` call swaps `used` and `size`, the golden test fails. That shows the
   golden covers the usage update.
4. G-local, G-hosts. **Commit.**

### P9 — the refusals and the permission option (MADR D8, D10)

Files:

- `acpclient/conn.go`;
- `acpserver/agent.go`;
- `acpclient/acptest/script.go`;
- a new `acpclient/refusals_test.go`;
- `acpserver/delete_test.go`.

1. **`acpclient`'s `handler`.**
   - The struct gains the embedded field `acp.ClientRefusals`.
   - The nine methods at `conn.go:552-600` and their comment go: `ReadTextFile`, `WriteTextFile`, `CreateTerminal`,
     `KillTerminal`, `TerminalOutput`, `ReleaseTerminal`, `WaitForTerminalExit`, `CreateElicitation`,
     `CompleteElicitation`.
   - `var _ acp.Client = (*handler)(nil)` stays.
2. **`acpserver.Agent`.**
   - It embeds `acp.AgentRefusals`.
   - `DeleteSession` (`agent.go:1176-1180`) and `Logout` (`:1241-1246`) and their comments go.
   - Its own `CloseSession`, `ListSessions` and `ResumeSession` stay. Go's method promotion gives the struct's own
     methods precedence; a test asserts that each still answers.
   - `acptest.ScriptAgent` embeds `acp.AgentRefusals` too.
     - Its stubs go: `DeleteSession` (`script.go:66-70`), `Logout` (`:261-263`) and `ListSessions` (`:266-268`),
       and any other of `AgentRefusals`' five methods whose body only answers `NewMethodNotFound`.
     - `Authenticate` (`:256-258`) stays: it is ungated.
     - `acptest.Client`'s conditional fs and terminal methods (`client.go`) stay, as 0034-MADR D4 left them.
3. **The reject loop.**
   - `conn.go:504-510` becomes:

     ```go
     if o, ok := acp.PickPermissionOption(r.Options, acp.PermissionOptionKindRejectOnce, acp.PermissionOptionKindRejectAlways); ok {
         return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(o.OptionId)}, nil
     }
     ```

   - The cancelled return after it stays.
4. **Tests.**
   - `refusals_test.go` calls each of the nine client methods on the connection's handler, through `acptest` or
     directly. It asserts `acp.IsMethodNotFound(err)` and the method name, and nil for `CompleteElicitation`.
   - `delete_test.go` and the `Logout` tests keep asserting `-32601`.
   - A permission test with options `[allow_once, reject_always]` answers `reject_always`, and one with no reject
     option answers `cancelled`.
5. **Seen failing:**
   - On a copy whose `handler` overrides `ReadTextFile` to return nil, `refusals_test.go` fails.
   - On a copy whose kinds are swapped to allow, the permission test fails.
6. Assert by search: no `NewMethodNotFound` remains in `acpclient/conn.go` or in `acpserver/agent.go`'s two methods.
   `extensions.go:67`'s stays: it is the extension switch.
7. G-local, G-hosts. **Commit.**

### P10 — `Kind()` (MADR D9)

Files: `acpclient/update.go`, `acpclient/conn_test.go`.

1. `Update` gains `Variant string`, the wire's `sessionUpdate` value. `translate` sets `u.Variant = s.Kind()` before
   its switch, for every update.
2. `KindOther` stays the `Kind` of an update with no case. `KindCompaction` stays `_gobble/compaction`.
3. `TestTranslate` (`conn_test.go:55-84`) asserts:
   - `Variant == "config_option_update"` for the `KindOther` case at `:70`;
   - `"agent_message_chunk"` for the first case;
   - `"unknown"` for a notification whose `sessionUpdate` value is a number.
4. **Seen failing:** on a copy with `Variant` left empty, `TestTranslate` fails.
5. G-local, G-hosts. **Commit.**

### P11 — the connection's end (MADR D11)

Files:

- `acpserver/serve.go`;
- `acpserver/rewrite.go`, deleted;
- `acpserver/rewrite_test.go`;
- `acpserver/serve_test.go`;
- `acpclient/conn.go`;
- `acpclient/acptest/pair.go`, `acpclient/acptest/script.go`.

1. **`acpserver.Serve`'s body becomes:**

   ```go
   err := acp.ServeAgent(ctx, agent, in, out, acp.WithLogger(logger),
       acp.WithInboundRewrite(acp.ResolveStdioCommands(exec.LookPath)),
       acp.WithPromptOverlapPolicy(acp.PromptOverlapConcurrent))
   return errors.Join(err, agent.Close())
   ```

   - The construction comment keeps its 0016-MADR A1 and 0019-MADR D6 citations.
   - `endWait`, `ended` and `handlersDone` go.
   - `ServeAgent` binds the agent through `SetAgentConnection`, as before. Verified 2026-10-29 at `v1.26.0`:
     `serve.go:29-30` builds the connection with `NewAgentSideConnection`, whose constructor calls
     `SetAgentConnection` (`agent_gen.go:51`).
   - At the input's end, it waits for the handlers within the 5 s of `notificationQueueDrainTimeout`, and returns nil
     only when `EndedCleanly` holds (`serve.go:31-43`).
   - At ctx's end, it returns `errors.Join(context.Cause(ctx), c.Shutdown(bound))` (`:44-47`).
2. **`rewrite.go`.**
   - Deleted.
   - `TestResolveBareCommands` and `TestResolveBareCommandsKeepsRelative` move to exercising
     `acp.ResolveStdioCommands(lookPath)` with the same cases.
   - `TestServeResolvesBareCommand` stays as it is.
3. **The client.**
   - `acpclient.Conn.Close`'s `handlersDone(c.conn.Wait, closeTimeout)` becomes
     `ctx, cancel := context.WithTimeout(context.Background(), closeTimeout); defer cancel();
     c.conn.Connection().WaitContext(ctx)`, its error a timeout when `ctx.Err() != nil`.
   - `handlersDone` at `:207` goes.
4. **`acptest`.**
   - `pair.go`'s cleanup waits on `c.Agent.Connection().WaitContext` and `c.Client.Connection().WaitContext`, bounded
     by `closeTimeout`. Its `handlersDone` goes.
   - `ScriptAgent`'s serve (`script.go:80-92`) becomes `acp.ServeAgent(ctx, a, in, out, opts...)`.
5. **New tests in `serve_test.go`:**
   - `TestServeShutdownCancelsPrompt`: a prompt in flight when `ctx` is cancelled is answered with stop reason
     `cancelled`, and `Serve` returns an error matching `context.Canceled`.
   - `TestServeReportsABrokenInput` and `TestServeAnswersBeforeEndOfInput` stay and pass.

   The overflow-as-clean defect is fixed by deleting `ended`, whose `ErrConnectionClosed` branch was the defect.
   `Serve` exposes no queue option, so a gobble test cannot force an overflow. The fix's evidence is:
   - `ServeAgent`'s documented use of `EndedCleanly`;
   - the SDK's own tests of `EndedCleanly` at `v1.26.0`, named in the execution record;
   - the search of step 7.
6. **Seen failing:** `TestServeShutdownCancelsPrompt` is first run against a copy of the tree at P10's commit. There
   the prompt gets no `cancelled` answer, since `Serve` closes the connection under it. If it passes there, that is a
   deviation, since the MADR's claim would be wrong.
7. **Assert:**
   - no `handlersDone`, `endWait`, `func ended` or `resolveBareCommands` remains in the module;
   - `session.golden` is unchanged;
   - `close_test.go`'s two tests pass.
8. G-local, G-hosts. **Commit.**

### P12 — `agentproc.InProcess` (MADR D12)

Files: `acpclient/conn.go`, `acpclient/close_test.go`, `acpclient/conn_test.go`.

1. **`Start`.**
   - It builds `p, err := (&agentproc.InProcess{Serve: agentproc-shaped serve, Grace: closeTimeout,
     ConnectionOptions: connOpts}).Start(c.handler)`.
   - `connOpts` is today's logger, overflow policy, drop handler and queue limit.
   - The `serve` adapter is `func(ctx context.Context, in io.Reader, out io.Writer) error { return serve(ctx, in, out) }`.
     `ServeFunc`'s signature is unchanged, so `internal/cli/agent.go:243-263` does not change.
2. **The connection.**
   - `c.conn` is `p.Conn()`. `Initialize` follows as today.
   - `Conn` loses `toAgent` and `served`, and gains `proc *agentproc.Process`.
3. **`Close`** becomes `c.proc.Close()`, then `WaitContext` on the connection as in P11. The returned error joins:
   - the agent's `Wait` error, prefixed `agent: `;
   - `agentproc.ErrAbandoned`, where the runner reports it.
4. **`closeTimeout`** is the runner's `Grace`, read at `Start`, so `SetCloseTimeout` (`export_test.go`) still bounds
   it. `TestCloseWaitsForHandlers` and `TestCloseBoundedWhenAHandlerHangs` (`close_test.go:48`, `:64`) are about the
   client's own update handlers, which P11's `WaitContext` bounds. They pass unchanged, still matching
   `"did not stop"`. A change to either is a deviation.
5. **New test, `TestCloseAbandonsAStuckAgent`:**
   - A serve function that ignores both its input and its ctx makes `Close` return an error matching
     `agentproc.ErrAbandoned`.
   - The test sets `SetCloseTimeout(t, 200*time.Millisecond)`, and `Close` returns within the runner's documented
     bound: three steps of `Grace`, plus the abandonment's grace, as `go doc agentproc.InProcess` states it at the
     phase's start. The test asserts at most 2 s.
6. **Seen failing:** on a copy whose `Grace` is left unset, which means the runner's 5 s each, the new test fails its
   2 s bound.
7. **Assert:** no `io.Pipe` remains in `acpclient/conn.go`. `internal/cli`'s tests, `print_test.go:21` among them,
   pass unchanged.
8. G-local, G-hosts, with `make perf`, since `LineSessionStart` includes `Start`. **Commit.**

*Amended 2026-10-31, by deviation 12, at acp-go-sdk2 `v1.28.0`:* steps 1, 3, 4 and 6 are as deviation 12 gives them.
Steps 2, 5, 7 and 8 stand, step 5's 2 s bound included. P12 runs after P20, P21 and P22.

### P13 — archtest names `/v2` (MADR D13)

Files: `internal/archtest/check.go`, `internal/archtest/check_test.go`.

1. `tuiLib` (`check.go:40`) becomes `"github.com/maccavelli/go-tui-lib/v2"`.
2. **A new constant,** `tuiLibV1 = "github.com/maccavelli/go-tui-lib"`, and a check in rule 5:
   - an import `pathUnder(imp, tuiLibV1)` that is not `pathUnder(imp, tuiLib)` is refused anywhere in the module;
   - the message is `"go-tui-lib's v1 path; import github.com/maccavelli/go-tui-lib/v2 (0035-MADR D13)"`.
3. **`check_test.go` cases:**
   - `rule 5 v1 path refused`: `internal/cli` imports `go-tui-lib/glyph`, and it is refused;
   - `rule 5 tuitest v2 in a test`: `internal/cli` test imports `tuiLib+"/tuitest"`, and it is allowed;
   - `rule 5 tuitest v2 outside a test`: refused.

   The existing cases use `tuiLib` and so move to `/v2` with it.
4. **Seen failing:** each new case against a copy at P12's `check.go`, where the first and second fail.
5. G-local, G-hosts. **Commit.**

### P14 — go-tui-lib `v2.6.1`, and `ask` for the permission prompt (MADR D1, D14 step 1; 0029-PLAN P2; PQ4)

*Deviation 1: the tag is `v2.6.2`. Every step below reads with it.*

Files:

- `go.mod`, `go.sum`;
- `internal/cli/permission.go`, `internal/cli/permission_test.go`;
- `internal/cli/trust.go`, `internal/cli/trust_test.go`;
- `internal/cli/line.go`;
- gobble's voice catalogue: the files `0022-MADR` names for CLI texts, with three new keys;
- `README.md`, `docs/architecture.md`;
- `docs/decisions/0029-MADR-permission-prompt-layout.md` (PQ4 (a)'s amendment);
- `docs/decisions/0029-PLAN-permission-prompt-layout.md` (P2's execution record);
- `docs/README.md`.

1. C2 for go-tui-lib, major 2. `python3 scripts/check_libs.py --lib go-tui-lib --tag v2.6.1`: every check ok, as in
   P2.
2. `go get github.com/maccavelli/go-tui-lib/v2@v2.6.1`, with the first import in this commit. `go mod tidy` keeps the
   requirement and adds exactly the five modules of 0032-MADR A1.2. Assert this by comparing `go.mod`'s requirements
   before and after; a sixth module is a deviation.
3. **The question.** A new `permissionQuestion(r acpclient.PermissionRequest) ask.Question` builds:
   - `Notes`: the once-only and limit lines that `permissionLine` (`permission.go:61`) writes today;
   - `Title`: `r.FullTitle`;
   - `After`: the `Patterns: …` line;
   - `Rows`: an Allow row and a Reject row.
     - Each answer's `Key` comes from `permissionKeys` (`:25`): `y`, `a`, `n`, or `[1]`…`[9]` when `kindsRepeat`
       (`:97`) says that kinds repeat.
     - The `f` answer is present only when `acpserver.RejectOnceID` is offered, with `Reason: true`.
     - The reject row's label ends "(hides the tool)" where `r.Hides` is set.
   - `Cancel`: `{Key: 'c', Label: "cancel"}`.

   `kindsRepeat` and the key table move into the builder; nothing else of that logic changes.
4. **The prompt.** `askPermission` (`:187`) drives an `ask.NewAsker(q, ask.WithMethod(…), ask.WithGlyphs(glyph.For(caps.Unicode)),
   ask.WithCatalog(<gobble's catalogue>))` in raw mode, which `editor.Raw` gives until P17:
   1. `Resize(min(width, 120), now)` (0029-MADR D5).
   2. Read stdin, decode each read with a `termkey.Decoder`, and call `HandleInput` for each input.
   3. Call `HandleTimeout(now)` when `Timeout()` is due, and `Flush` the decoder when it is pending and the input is
      quiet.
   4. At the end of input, call `Cancel()`.
   5. After every call, drain `NextEvent()` until `AskNone`, writing each `AskWrite`'s bytes to the prompt's writer.
   6. `AskDone`'s `Response` becomes `acpclient.PermissionAnswer{OptionID: <the answer's ID>, Feedback: Response.Reason}`,
      or the cancelled outcome.

   The transcript records `View(width)`'s rows where it recorded `shown+label` (`:225`).
5. **The trust question.** `askTrust` (`trust.go:166`) builds its question the same way, and its answer switch
   (`:192-204`) stays.
6. **What goes:** `permissionLine`, `chooseKey`, `labelOf`, `readWhy`, `cutWhy` and `maxWhy` (`ask.DefaultReasonLimit`
   is the same 2048), and the code in `trust.go` that called them.
7. **Catalogue.** The reason prompt, the reason echo and the cancel text are new keys in gobble's voice catalogue,
   with the same English as `ask`'s defaults (`ask/messages.go:23-25`), so that every personality may override them.
8. **Tests.**
   - `TestPermissionLine` becomes `TestPermissionQuestion`. It asserts the `Question` for each of:
     - a once-only request;
     - repeated kinds, numbered;
     - a request without `RejectOnceID`, which has no `f`;
     - a request with `Hides`.
   - `TestPermissionPrompt` and `TestPermissionPromptOnceOnly` drive the bytes through `inline/inlinetest.Screen` at
     widths 40, 80, 120 and 200, asserting the rows of 0029-MADR D1 and D2 as amended. At 200 the rows are those at
     120.
   - `TestReadWhy` becomes reason entry: typing `f`, a reason, then enter gives `Feedback`. A 2049-byte reason is cut
     at 2048.
   - `TestPermissionLineNumbered` (`trust_test.go:356`), `TestTrustQuestion`, `TestTrustQuestionNoParent` and
     `TestLineViewTrust` are moved to the new drawing, and each change is named in the execution record.
9. **Seen failing:**
   - on a copy whose builder drops `Reason: true`, the reason test fails;
   - on a copy without the 120-cell cap, the width-200 test fails.
10. **0029-MADR:** a dated amendment of D1, carrying PQ4 (a)'s four differences.
    **0029-PLAN:** P2's execution record, with this phase's G-hosts and `make probe-conhost` results. Its P3 and its
    status follow that PLAN's own criteria, not this one's.
11. `README.md` and `docs/architecture.md` name go-tui-lib `/v2` `v2.6.1` among the pins, and `internal/cli` among
    its importers.
12. G-local with `make perf`, and G-hosts with `make perf` and `make probe-conhost`. **Commit.**

### P15 — `glyph`, and `safetext` on terminal paths (MADR D14 step 2; PQ5)

Files:

- `internal/cli/view.go`;
- `internal/cli/term/style.go`, `internal/cli/term/style_test.go`;
- `internal/cli/term/sanitize.go`, `internal/cli/term/sanitize_test.go`;
- the terminal callers of `term.Sanitize`:
  - `internal/cli/permission.go`;
  - `internal/cli/view.go`;
  - `internal/cli/complete/protocol.go`;
  - `internal/cli/render/plan.go`, `tool.go`, `commands.go`;
  - `internal/cli/editor/editor.go`;
- their tests.

1. **`glyphsFor`** (`view.go:98`) becomes `glyph.For(caps.Unicode)`. Callers read `glyph.Set`'s fields:

   | gobble | `glyph.Set` |
   | :--- | :--- |
   | `Bullet` | `Bullet` |
   | `Arrow` | `Heading` |
   | `Check` | `Done` |
   | `Cross` | `Failed` |
   | `Bar` | `MeterFill` |
   | `BarEmpty` | `MeterTrack` |
   | `Ellipsis` | `Ellipsis` |

   - gobble's glyph struct in `term/style.go` goes. `Style.Wrap` stays.
   - ASCII's ellipsis becomes the library's `~` where it was `...`, by Q3's direction. The goldens that show it are
     re-recorded and named.
   - `TestGlyphSets` becomes a test that `glyph.For(false)` and `glyph.For(true)` give the fields gobble draws.
2. **`safetext`, by PQ5 (a).**
   - Every terminal caller listed above calls `safetext.Clean(s, safetext.Text())` in place of `term.Sanitize(s)`.
     Where it wrapped the result in `strings.Fields`, it calls `safetext.Clean(s, safetext.Line())` and keeps the
     `Fields`, since `Line` does not collapse runs of spaces.
   - `print.go:201` and the piped line view keep `term.Sanitize`.
   - `sanitize.go` keeps `Sanitize` and its helpers only. `EscapeBidi` and its test go.
3. **Tests.**
   - `TestSanitize` stays, for print mode.
   - A new `TestTerminalCleaning` asserts that a zero-width space, a bidi override and an OSC 52 sequence in model
     text are absent from the line view's terminal output.
   - The view goldens with colour are re-recorded where `safetext` differs. The diff is reviewed against the MADR's
     list, and any other change stops the phase.
   - `print-json.golden` and `print-stream-json.golden` are byte-identical.
4. **Seen failing:** on a copy with one terminal caller left on `term.Sanitize`, `TestTerminalCleaning` fails for the
   zero-width space.
5. **Assert by search:** `term.Sanitize` remains only in `print.go` and the piped line view's path, and those
   locations are listed in the execution record.
6. G-local with `make perf`, and G-hosts with `make perf` and `make probe-conhost`. **Commit.**

### P16 — `pick` (MADR D14 step 3; PQ6)

By PQ6 (a), no code changes.

1. The MADR gets a dated note at D18: `pick` is new UI, not a replacement.
2. The MADR gets a dated note at D14: `termcap` is not adopted, since `caps.go` stays.
3. `docs/README.md` is updated.
4. Gates: `make check-records markdownlint`. **Commit**, records only.


### P17 — `lineedit` (MADR D14 step 4; PQ7)

Files:

- `internal/cli/editor/` (every file);
- `internal/cli/line.go`, `output.go`, `slash.go`, `config.go`, `auth.go`, `login.go`, `trust.go`;
- their tests;
- `internal/cli/term/termtest` stays: `internal/cli`'s view and reprint tests still use it.

1. **`internal/cli/editor` becomes the host of a `lineedit.Composer`.** It keeps:
   - `Raw` (used at `line.go:84`, `trust.go:209` and `login.go:282`);
   - `external.go`, with `editorCommand`, as the process behind `WithExternalEditor`;
   - the history file (`history.go`): 1000 entries, `0600`, atomic rename. It is read into and written from a
     `lineedit.MemHistory` with `ReadFrom` and `WriteTo`, whose format is gobble's;
   - the Ctrl+C window (`keys.go:122`: clear the line; on an empty line, arm 2 s, with "Press Ctrl+C again to exit"),
     driven from `EditInterrupt`;
   - `/exit` and `/quit`, and skipping an empty submission, both on `EditSubmit`;
   - the paste notice, written on a paste event when the paste has more than one line;
   - `MaxWidth` and the 120-cell cap;
   - `Compose` and `Open`, unchanged for their callers (`slash.go:37`, `config.go:19`).
2. **What goes:**
   - `buffer.go`, `draw.go`, `layout.go`, `keys.go` (all but the Ctrl+C window) and `paste.go`;
   - `reader.go`, replaced by `inline.Reader` at `auth.go:198`, `login.go:280` and `line.go:67`;
   - `editor.go`'s key loop and drawing.
3. **The Composer's options:**
   - `WithCompleter`: an adapter over `ls.complete` (`slash.go:215`);
   - `WithPaste(lineedit.PasteInsert)`;
   - `WithExternalEditor`;
   - `WithClearScreen`;
   - `WithLimit(4096)`;
   - `WithHeader`;
   - `WithRegion` of an `inline.Region`, whose reflow is chosen by the `XTERM_VERSION` check that `editor.go:308`
     makes today;
   - `WithKeymap(keymap default)`.

   It is driven like P14's Asker: `Resize`, `HandleInput`, `HandleTimeout`, then `NextEvent` until `EditNone`.
4. **Tests.**
   - `history_test.go`, `keys_test.go` (the Ctrl+C window) and `external_test.go` stay.
   - `buffer_test.go`, `draw_test.go`, `layout_test.go`, `editor_screen_test.go`, `resize_test.go`, `reader_test.go`
     and `editor_test.go` are rewritten against the host, using `inline/inlinetest.Screen`.
   - Each test that asserted a PQ7 difference is named, with its old and new expectation:
     - `TestWordKeys`: alphanumeric words;
     - `TestPasteNoticeAndPrompts`: one buffer, the notice kept;
     - the multiline tests: one buffer.
   - `TestKeysMatchXterm` and `TestKeysSplitAcrossReads` move to `termkey`'s decoding, through the host.
5. **Benchmarks:** `BenchmarkKeyDraw` and `TestKeyDrawBudget` measure the Composer's draw of one key. 0030-MADR D5's
   budget, three times the baseline, holds; a breach is a deviation.
6. **Seen failing:**
   - on a copy whose host skips the Ctrl+C window, `TestCtrlCTwiceInsideWindowExits` fails;
   - on a copy writing history without `0600`, the history mode test fails.
7. **The Windows laptop:** `conhost_windows_test.go` and `make probe-conhost` pass.
8. G-local with `make perf`, and G-hosts with `make perf` and `make probe-conhost`. **Commit.**

*Amended 2026-10-31, by deviation 12:* P17 resumes after P21 and P12, on `feature/p17-lineedit`, rebased onto `main`.
Step 8's commit is on that branch, and the owner fast-forwards `main` to it.

### P18 — `stream` and `transcript` (MADR D14 step 5; Q3; PQ3)

Files:

- `internal/cli/render/mdstream.go`, `mdstyle.go`, `table.go` (deleted), and their tests and testdata;
- `internal/cli/render/tool.go`, `commands.go`, `message.go`;
- `internal/cli/view.go`, `internal/cli/transcript.go`, `internal/cli/output.go`, `internal/cli/line.go`,
  `internal/cli/slash.go`, `internal/cli/chat.go`, `internal/cli/permission.go`;
- gobble's voice catalogue, with one new key;
- `internal/cli/testdata/view-*.golden`;
- the perf and budget tests;
- `docs/decisions/0032-PLAN-go-tui-lib-integration-boundary.md`.

1. **PQ3's facts,** re-read at the phase's start; a difference is a deviation:
   - print mode does not use `render`;
   - the line view decides by `caps.Out.Color` (`view.go:274`);
   - `replyWidth` is 0 off a terminal (`line.go:254-259`).
2. **The reply.** Each reply is a `stream.Printer`, in place of `render.Stream`, `Style`, `Tables` and `Wrap` at
   `view.go:52`, `:132`, `:275`, `:301`, `:510` and `:512`.
   - **Colour on** (a terminal, or `CLICOLOR_FORCE=1`): `WithMarkup(stream.DefaultMarkup())`, the library's table
     layout, and `WithWidth(min(width, 120))`, or `0` where `replyWidth` is 0. These are the library's defaults, by Q3.
   - **Colour off:** the zero `Markup`, `WithoutTableLayout()`, and `WithWidth(0)`, which means no wrapping
     (`go doc stream.WithWidth`). That is today's raw source, by Q3's guard.
   - `SetWidth` follows a resize.
3. **Thinking text** keeps `render.Wrap` (`view.go:519`), since the library has no plain streaming wrapper.
   `wrap.go` stays, with a comment naming the gap.
4. **The width helpers.** `displayWidth` and `truncateWidth` give way to `ansi.StringWidth` and `ansi.Truncate`
   (0032-MADR A1.4), at:
   - `tool.go:32-33`;
   - `commands.go:77`, `:91`, `:128` and `:157`;
   - `message.go:47` and `:58`.

   `table.go` is then deleted, with `mdstream.go` and `mdstyle.go`, and their tests and testdata, whose cases are the
   library's tests now (go-tui-lib issue 10).
5. **The transcript.** `internal/cli/transcript.go` becomes a mutex-guarded wrapper of one `transcript.Transcript`,
   since three goroutines share it.
   - A piece with SGR (`view.go:384`, `output.go:40`) becomes `AddDrawn`, a reply `AddReply(p)`, and plain text
     `AddText`.
   - Resizes call `Resize(width, now)`. Each idle point calls `HandleTimeout(now, idle)`, whose reprint is written
     between the P17 region's `Clear` and `Draw`. That replaces `tryReprint`.
   - The notice's text comes through `WithCatalog`, from a new key of gobble's voice catalogue (0022-MADR). Its style
     is the library's default, by Q3.
   - `WithMaxBytes` is 4 MiB, the same as today's `maxTranscript`. `WithMaxRows` is the library's default.
6. **Goldens.**
   - `view-plain.golden` is **byte-identical**: Q3's guard.
   - `view-color.golden` and `view-thinking-verbose-stats.golden` are re-recorded. Each hunk of the diff is traced to
     one of the MADR's listed differences, and a hunk that matches none stops the phase.
   - `print-json.golden` and `print-stream-json.golden` are byte-identical.
7. **Benchmarks.**
   - `benchReplyRender`, `BenchmarkReprint` and `TestReprintAllocationBudget` measure the library's path.
   - 0030-MADR D5's budgets hold.
   - The execution record quotes the new figures beside go-tui-lib issue 10's: 5.2 ms and 43,541 allocations for a
     4 MiB reprint on macOS.
8. **Seen failing:**
   - on a copy whose colour-off path uses `DefaultMarkup()`, `view-plain.golden` fails;
   - on a copy without the mutex, `make race` fails on the reprint test.
9. **Assert by search:** no `render.Stream`, `render.Style`, `render.Tables`, `displayWidth` or `truncateWidth`
   remains.
10. **0032-PLAN:** its P3 and on are recorded as carried out, and its status becomes `complete`.
11. G-local with `make perf`, and G-hosts with `make perf` and `make probe-conhost`. **Commit.**

### P19 — close-out

Files:

- this plan;
- the MADR, if a fact changed during execution;
- `docs/README.md`;
- `docs/architecture.md`.

1. **`docs/architecture.md`:**
   - its package notes say `internal/cli` imports go-tui-lib's `ask`, `glyph`, `safetext`, `lineedit`, `inline`,
     `termkey`, `stream`, `transcript` and `keymap`, and `x/ansi`;
   - `term/caps.go`, the print-mode `Sanitize`, `render.Wrap` and the editor's host stay gobble's.
2. **Lines removed:** `git diff --stat ccd8aa8..HEAD -- ':!docs' ':!*_test.go' ':!*/testdata/*'`. The execution record
   quotes it, against the MADR's estimate.
3. **Issue comments for #4, #5 and #6,** written into the execution record for the owner to post. Posting is
   outward-facing, so it is done only on the owner's word.
4. **The disclosure guard** over the outgoing commits (`AGENTS.md`, "Identifiers"), with its output recorded. Nothing
   is pushed.
5. **Status.** This plan's status becomes `complete` when every phase's criteria are met. That includes P6, so
   P6's questions must be answered first.

### P20 — acp-go-sdk2 `v1.28.0` (MADR D1, amended 2026-10-31; deviation 12)

Files: `go.mod`, `go.sum`, `README.md`, `docs/architecture.md`.

1. **Checked first:** `python3 scripts/check_libs.py --lib acp-go-sdk2 --tag v1.28.0` reports every check ok.
2. **The pin:** `go get github.com/maccavelli/acp-go-sdk2@v1.28.0`, then `go mod tidy`.
   - `go.mod` changes on that line alone.
   - `go.sum` changes in acp-go-sdk2's lines alone.
   - `github.com/coder/websocket` is in neither file.
3. **No source change.** `acpclient`, `acpserver` and `internal/cli` build and pass their tests unchanged.
4. **The docs:** `README.md:129` and `docs/architecture.md:156` name `v1.28.0`.
5. G-local, and G-hosts. **Commit.**

No check is added, so C9 has nothing to break.

### P21 — go-tui-lib/v2 `v2.7.0` (MADR D1, amended 2026-10-31; deviation 12)

Files: `go.mod`, `go.sum`, `README.md`, `docs/architecture.md`.

1. **Checked first:** `python3 scripts/check_libs.py --lib go-tui-lib --tag v2.7.0` reports every check ok, `archtest`
   among them.
2. **The pin:** `go get github.com/maccavelli/go-tui-lib/v2@v2.7.0`, then `go mod tidy`.
   - `go.mod` changes on that line alone.
   - `go.sum` changes in go-tui-lib's lines alone.
3. **No source change.** `internal/cli`'s tests pass unchanged, P14's `ask` prompt and P15's `glyph` and `safetext`
   among them.
4. **The docs:** `README.md:137` and `docs/architecture.md:160` name `v2.7.0`.
5. G-local with `make perf`, and G-hosts with `make perf` and `make probe-conhost`. **Commit.**

No check is added, so C9 has nothing to break.

### P22 — go-selfupdate-lib `v1.23.1` (MADR D1, amended 2026-10-31; deviation 12)

Files: `go.mod`, `go.sum`, `.github/workflows/ci.yml`, `README.md`, `docs/architecture.md`.

1. **Checked first:** `python3 scripts/check_libs.py --lib go-selfupdate-lib --tag v1.23.1` reports every check ok.
2. **The pin:** `go get github.com/maccavelli/go-selfupdate-lib@v1.23.1`, then `go mod tidy`.
   - `go.mod` changes on that line alone, since gobble already requires the library's new `golang.org/x` versions.
   - `go.sum` changes in go-selfupdate-lib's lines alone.
3. **The workflow pins:** both `ci.yml` lines that name `67520be8af1ad1b6228c29e4b1aa3e61e1ca7ae4 # v1.23.0` name
   `c24f5919cd9e9e3ba3736e64bcf18a6e2a51489e # v1.23.1`, the tag's commit by `git ls-remote --tags`.
   - Assert by search: no `67520be8` remains in `.github/`.
4. **No source change.** `buildinfo`, `selfupdate` and their tests pass unchanged.
5. **The docs:** `README.md:136` and `docs/architecture.md:159` name `v1.23.1`.
6. G-local, and G-hosts. **Commit.**

No check is added, so C9 has nothing to break.

## Verification

Each phase's own steps are its acceptance criteria. Across the plan:

- `go list -m github.com/maccavelli/acp-go-sdk2 github.com/maccavelli/go-llmprovider-sdk
  github.com/maccavelli/go-selfupdate-lib github.com/maccavelli/go-tui-lib/v2` reports MADR D1's four tags.
- `make check-libs` reports every library at its newest tag with every check ok, `staticcheck` and `lint` among them.
- `acpserver/testdata/session.golden` is byte-identical to `ccd8aa8`'s.
- `view-plain.golden`, `print-json.golden` and `print-stream-json.golden` are byte-identical to `ccd8aa8`'s.
- `make preflight`, `make race` and `make perf` pass on macOS, the Linux machine and the Windows laptop, and
  `make probe-conhost` passes on the Windows laptop.
- Every new test and gate has a quoted failure from a broken copy (C9).

## Rollout and Rollback

- Each phase is one commit and reverts on its own with `git revert`, except where a later phase builds on it:
  - P5 needs P4;
  - P8–P12 need P7;
  - P14–P18 need P13;
  - P15–P18 need P14's requirement.
  - P12 needs P20, and P17 needs P21 (deviation 12). P22 needs nothing after P3.

  Reverting a pin phase reverts the phases that need it, newest first.
- Nothing is pushed by this plan. The owner pushes, after the disclosure guard.
- A tag or library change that appears mid-plan does not move a pin already landed. It is a new amendment.
- A user-visible change is limited to:
  - P6, the workspace flag and prompt;
  - P11, an interrupted `gobble acp` answering `cancelled`;
  - P14, the permission prompt's layout;
  - P15 and P18, terminal styling and cleaning;
  - P17, the editor's multiline, paste and word moves.

  Each is named in its commit's message by the hook, from the diff.

## Execution record

PQ1, PQ2 and PQ4–PQ7 were answered on 2026-10-29.

### Deviation 1 (2026-10-29): newer tags, and the pin phases first

- **Found.** The owner asked: "i want to update all pins to current versions". C2's `git ls-remote --tags` that day
  listed go-selfupdate-lib `v1.23.0` and go-tui-lib `v2.6.2`, both newer than MADR D1's tags. The MADR's D1
  amendment of this date gives the evidence. acp-go-sdk2 `v1.26.0` and go-llmprovider-sdk `v1.24.1` are unchanged.
- **The owner's answers, the same day:**
  - tags: "The newest (Recommended)";
  - scope: "Pin phases, reordered (Recommended)".
- **Decision.**
  - P3 pins go-selfupdate-lib `v1.23.0`, with the workflow pins at `67520be8af1ad1b6228c29e4b1aa3e61e1ca7ae4`.
  - P2's live run and P14 use go-tui-lib `v2.6.2`.
  - The phases run P0, P2, P3, P4, P7, P13, P14, then the rest in number order.
  - P7 needs only P4's go-llmprovider-sdk pin, which it does not touch, and P13 needs nothing between them. So
    moving them ahead of P5, P6 and P8–P12 changes no phase's own steps.
  - P1 moves after P14. Its docs fix needs no tag.
- **Files added to scope:** none.

### P0 record (2026-10-29): the records

Approved by the owner's "Pin phases, reordered (Recommended)", 2026-10-29, which starts this plan. The plan's status
is now `in-progress`.

- **C1.** The tree held only this pair's deviation 1, and `HEAD` was `7894bb3`. The hooks path is the global hooks
  directory, and `go version` is `go1.27.3 darwin/arm64`.
- **Steps 1–6, done as written:**
  1. 0034's MADR and PLAN are `superseded by` 0035-MADR, with a note under each title.
  2. 0031-MADR is read as it stands, with the other session's notes of 2026-10-29. Its D1 note already named 0035 and
     the tags of that hour, so one further note gives deviation 1's tags and retires the `v0.12.0` statement. Its D9
     note gains one line: 0034 is superseded.
  3. 0032-MADR gains amendment A2.
  4. 0032-PLAN gains a note under "P3 and on".
  5. 0029-PLAN gains a note after P2's stop.
  6. 0016-MADR gains a note at D13.
- **Step 7:**
  - `docs/README.md` notes the rows of 0016, 0029, 0031 and 0032, and marks 0034 superseded and this PLAN in
    progress.
  - *Added in this step:* the two 0036 rows. They were missing from `65a1242`, the other session's commit, and
    `AGENTS.md` requires every record to be indexed. This is a fix to the index, inside step 7's file. 0036's records
    themselves are not touched.
- **Step 8, the gates, each exit 0:**
  - `make check-records markdownlint`;
  - `python3 scripts/check_records.py --check-all`;
  - `make preflight`, all 13 steps, "preflight passed". It was run although no Go file changed, since `AGENTS.md`
    asks for it before every commit.
- No deviation.

### P2 record (2026-10-29): check-libs sees what preflight sees

`scripts/check_libs.py`, P0's commit `2feb647` as the base.

**Steps 1–7, done as written, apart from the probe list:**

- `Library` gains `path`, `major`, `probe` and `allow_new`.
- `latest_tag` filters through a new pure `newest(tags, major)`.
- `pinned` returns `None` for a module gobble does not require, read from `go list -m all`.
- `check_library` behaves as follows for such a library:
  - it writes `internal/cli/zz_checklibs_probe.go` in the scratch clone only, with each blank import commented for
    revive's `blank-imports`;
  - it notes `pinned: not required`;
  - it skips `api`;
  - it allows `allow_new` in `modules`.
- `staticcheck` and `lint` run for every library after `test`, and `archtest` for a library with probe packages.
- The docstring and the `--lib` help are updated.

*Note:* the probe list is `ask`, `glyph`, `lineedit`, `safetext`, `stream` and `transcript`. Step 3 also named `pick`
and `termcap`, which PQ6 (a), answered after the step was written, does not adopt. Probing packages gobble will not
import would test nothing it ships. This follows the owner's answer, and no decision changes.

**Step 8, the verification:**

- **`python3 scripts/check_libs.py --self-test`:** all 15 cases ok, the 5 new among them.
- **Seen failing (C9), on copies of the tree** (`<scratch>/break-P2-1`, `-2`, since removed):
  - with `newest`'s major test inverted (`m != major`), the self-test exits 1:
    `self-test major 2 keeps v2.6.1 alone: FAIL`;
  - with `allow` ignored in `moved_modules`, the self-test exits 1:
    `self-test a first import may add the allowed modules alone: FAIL`.
- **Live runs.** Deviation 1's tags were used for the two libraries it moved. Each run is the expected one:

  | Command | Result |
  | :--- | :--- |
  | `--lib acp-go-sdk2 --tag v1.26.0` | exit 1: `staticcheck: FAIL` (48 SA1019 lines, F2's 16 findings on each of three GOOS) and `lint: FAIL` (`12 issues: * staticcheck: 12`); every other check ok |
  | `--lib go-llmprovider-sdk --tag v1.24.1` | exit 1: `test: FAIL`, `--- FAIL: TestWireOffline/refused` and `/unknown_host` (F1); `staticcheck` and `lint` ok |
  | `--lib go-selfupdate-lib --tag v1.23.0` | exit 0, all 14 checks ok |
  | `--lib go-tui-lib --tag v2.6.2` | exit 0: `pinned: not required; probing 6 packages`, `modules` ok with the five allowed, `staticcheck`, `lint` and `archtest` ok, `api: not required; nothing to compare` |

**Step 9, the gates:**

- `python3 -m py_compile scripts/check_libs.py`: exit 0.
- `make preflight`: "preflight passed".
- `make race`: exit 0, 36 packages ok.
- `make pre-add-check` was not run: no Go file is staged.
- **G-hosts.** The self-test passed all 15 cases on the Linux machine and on the Windows laptop, each on a copy of the
  tree (807 files).

No deviation.

### P3 record (2026-10-29): go-selfupdate-lib `v1.23.0`

Deviation 1's tag, `v1.23.0`, in place of `v1.22.3`. The base is P2's commit `0ad318c`.

1. **C2.** `git ls-remote --tags` lists `v1.23.0` as the newest `v1` tag, peeled to
   `67520be8af1ad1b6228c29e4b1aa3e61e1ca7ae4`. `check-libs` for that tag passed every check in P2.
2. **The pin.** `go get github.com/maccavelli/go-selfupdate-lib@v1.23.0`, then `go mod tidy`, each exit 0.
   `git diff --stat` shows `go.mod` (one line) and `go.sum` (four) only.
3. **The workflow pins.**
   - `ci.yml:166` and `:178` read `@67520be8af1ad1b6228c29e4b1aa3e61e1ca7ae4 # v1.23.0`.
   - That commit is `refs/tags/v1.23.0^{}` by `git ls-remote`.
   - In a scratch clone of the library, `git diff --quiet v1.22.0 v1.23.0 --
     .github/workflows/build-selfupdate-release.yml .github/workflows/publish-selfupdate-release.yml` exits 0: the
     workflows are identical.
4. **Docs.** `README.md` "Stack" and the `docs/architecture.md` pin table say `v1.23.0`. 0031-MADR has a note at D1.
5. **G-local.**
   - `make preflight`: "preflight passed".
   - `make race`: exit 0, 36 packages ok.
   - `make pre-add-check` was not run: no Go file changed.
6. **G-hosts,** on a copy of the tree (807 files) on each host:

   | Host | Go | `go test` | race | `preflight` | records and markdownlint |
   | :--- | :--- | :--- | :--- | :--- | :--- |
   | the Linux machine | `go1.27.3 linux/amd64` | exit 0 | exit 0, 36 ok | to `verify-build-metadata` | exit 0 |
   | the Windows laptop | `go1.27.3 windows/amd64` | exit 0 | exit 0, 36 ok | to `verify-build-metadata` | exit 0 |

   On both, `verify-build-metadata` stops at `git rev-parse HEAD` (exit 128), since the copy has no commit. That
   check is macOS-only (C6).

No deviation.

### Deviation 2 (2026-10-29): P4 step 4's assertion

- **Found.** Step 4 asks that `err.Error()` hold `"offline"` exactly once. The SDK's own text is `llmprovider: offline: …`,
  so a correct single wrap reads `llm: offline: llmprovider: offline: Post …`, and the assertion as written would fail
  on correct code. This was found while writing the step, before any test ran. It is the plan's error.
- **The owner's answer, the same day:** "Count llm's text (Recommended)".
- **Decision.** `TestWireOffline` asserts two things:
  - `strings.Count(err.Error(), llm.ErrOffline.Error()) == 1`, which is `"llm: offline"` exactly once;
  - `errors.Is(err, llm.ErrUnavailable)` is false.

  That pins what the step meant: gobble's sentinel wraps once, and the unavailable row no longer wins.
- **Files added to scope:** none. `llm/provider/offline_test.go` was already the step's test.

### P4 record (2026-10-29): go-llmprovider-sdk `v1.24.1` and the offline row

The base is P3's commit `8632061`.

1. **C2.** `v1.24.1` is still the newest. P2's `check-libs` run gave `test: FAIL` on `TestWireOffline` alone, as
   expected.
2. **The pin.** `go get github.com/maccavelli/go-llmprovider-sdk@v1.24.1`, then `go mod tidy`, each exit 0. Only
   `go.mod` (one line) and `go.sum` (four) change.
3. **The row.** `{llmprovider.ErrOffline, llm.ErrOffline}` sits directly above the unavailable row in `sentinels`, with
   a three-line comment citing 0035-MADR D3.
4. **The test, as deviation 2.**
   - `case !kind && offline(err):` stays until P5, as written.
   - `go test -run 'TestWireOffline|TestErrorMapping' ./llm/provider/`: ok.
5. **Seen failing (C9), on copies at the new pin:**
   - **The row removed.** `TestWireOffline` fails on both subtests with MADR F1's message:
     `err llm: unavailable: llmprovider: offline: Post "http://127.0.0.1:…/v1/responses": dial tcp …: connect: connection refused, want llm.ErrOffline`.
   - **A double wrap.** Here `case offline(err):` wraps `wrapped` again. The new assertion fails:
     `err llm: offline: llm: offline: llmprovider: offline: … : "llm: offline" 2 times, unavailable false; want once, and not unavailable`.
   - The first attempt at the second break, which dropped `!kind`, did not compile (`kind` unused). It was replaced by
     one that keeps `kind` used, and only the compiling break's result is evidence.
6. **Docs.** `README.md` "Stack" and the `docs/architecture.md` pin table say `v1.24.1`. 0031-MADR has a note at D1.
7. **G-local.**
   - `make pre-add-check FILES="llm/provider/provider.go llm/provider/offline_test.go"`:
     `2 file(s) clean (gofmt, golint, govulncheck)`.
   - `make preflight`: "preflight passed".
   - `make race`: exit 0, 36 packages ok.
8. G-hosts, on a copy of the tree (807 files): on the Linux machine (`go1.27.3 linux/amd64`) and the Windows laptop (`go1.27.3 windows/amd64`), `go test` exit 0, race exit 0 with 36 packages ok, `preflight` to `verify-build-metadata` (stopped at `git rev-parse HEAD`, exit 128: the copy has no commit, C6), and records and markdownlint exit 0.

One deviation: deviation 2, above.

### Deviation 3 (2026-10-29): P7's gate and the house idiom

- **Found.** `make staticcheck` with the gate as step 5 wrote it (directives citing `0035-MADR D6` only) failed on each
  GOOS, on five suppressions older than this plan:
  - `mcpclient/conn.go:136`, `//lint:ignore SA1019 roots are deprecated from MCP 2026-07-28 but still served, and servers
    on earlier protocols read them (0002-PLAN Phase 5, deviation 3)`;
  - `mcpclient/value.go:76` and `:79`, and `compaction/compaction.go:51` and `:53`, each `//lint:ignore ST1005 Pi's
    message, word for word (0002-PLAN Phase 5, deviation 3…)`.

  These are pre-existing and recorded. Each pairs its `//lint:ignore` with `//nolint:staticcheck // as the lint:ignore
  above`, the idiom 0002-PLAN Phase 5 deviation 3 set: standalone staticcheck reads only `//lint:ignore`, and
  golangci-lint reads only `//nolint`. Step 4's central `.golangci.yml` text rule departed from that idiom, and would
  have silenced every future acp-go-sdk2 deprecation for golangci-lint.
- **Research, for the owner's request** "dig into this a bit. research best practices and the most idiomatic
  approaches…":
  - the MCP go-sdk `v1.8.0` deprecates roots by SEP-2577, "functional during the deprecation window (at least twelve
    months)". `v1.8.0` is that module's newest version;
  - acp-go-sdk2's own boundary is its `unstable` package, which cannot take fields already merged onto stable types,
    such as `PromptResponse.Usage`.

  Three options were offered: the house idiom with a general gate; quarantining the unstable uses in one `unstable.go`
  per package, with file-wide directives; and an exact baseline file of the allowed ignored findings.
- **The owner's answer, 2026-10-29:** "House idiom + general gate (Recommended)".
- **Decision.**
  - Each of F2's 12 lines carries both the `//lint:ignore` line and `//nolint:staticcheck // as the lint:ignore
    above`.
  - `.golangci.yml` is not changed.
  - The gate is record-based, for every suppression in the module. Each `//lint:ignore` or `//lint:file-ignore`
    cites a NNNN-MADR or NNNN-PLAN record. Each `//lint:ignore` is paired with a `//nolint:staticcheck` on the next
    line, and each `//nolint:staticcheck` follows a `//lint:ignore`. Each ignored finding is under a directive of its
    own check, comma-separated lists included.
  - The MADR's D6 carries the amendment.
- **Files added to scope:** none. `.golangci.yml` leaves P7's changes.

### P7 record (2026-10-29): acp-go-sdk2 `v1.26.0`, the suppression and its gate

The base is P4's commit `bd79d2a`.

1. **C2.** `v1.26.0` is still the newest. P2's `check-libs` gave `staticcheck: FAIL` and `lint: FAIL` alone, as
   expected.
2. **The pin.** `go get github.com/maccavelli/acp-go-sdk2@v1.26.0`, then `go mod tidy`, each exit 0. Only `go.mod` (one
   line) and `go.sum` (four) change.
3. **The directives.**
   - Each of F2's 12 lines has the `//lint:ignore SA1019 acp-go-sdk2 keeps this unstable-schema name under v1
     (0035-MADR D6)` line above it.
   - The script asserted that each target line holds an F2 name.
   - `gofmt -w` moved the one under `acpUsage`'s doc comment below a `//` line.
   - By deviation 3, each target line also ends `//nolint:staticcheck // as the lint:ignore above`.
4. **`.golangci.yml`:** unchanged, by deviation 3.
5. **The gate** is `scripts/check_staticcheck_ignored.py`, mode 0644 like the repository's other scripts, as
   deviation 3 has it.
6. **`Makefile`.** `staticcheck` runs `$(PYTHON) scripts/check_staticcheck_ignored.py ./$(STATICCHECK_BIN)` per GOOS.
7. **Seen failing (C9).** Four copies of the tree (`<scratch>/break-P7-1` to `-4`, removed), each a git repository
   for `git ls-files`, with the tree's staticcheck binary. Each exits 1:
   - **no record cited:** `acpclient/conn.go:339: the reason cites no NNNN-MADR or NNNN-PLAN record`;
   - **the `//nolint` removed:** `acpclient/conn.go:340: no //nolint:staticcheck under the //lint:ignore at line 339`;
   - **a stray `//nolint:staticcheck` planted:** `acpclient/conn.go:1: //nolint:staticcheck with no //lint:ignore on
     the line above`;
   - **a fake staticcheck reporting an ignored SA1019 at an uncovered line:** `acpserver/agent.go:1: ignored SA1019
     under no directive of that check`.

   The first attempt at the fourth passed the fake as two arguments, and the script answered with its usage error
   (exit 2). Only the rerun, with the fake as one executable path, is evidence.

   Without the `//nolint` pairs, `make lint` reported `12 issues: * staticcheck: 12`, on a scratch clone of
   2026-10-29.
8. **Assert.** `make staticcheck` prints, for each of linux, darwin and windows,
   `check_staticcheck_ignored: 17 directive(s), 22 ignored finding(s), each recorded and paired`. Those are F2's 12
   directives and the 5 older ones. The findings are 18 SA1019 (F2's 16 and MCP roots' 2) and 4 ST1005. The count of
   `-show-ignored` SA1019 records is 18 on each GOOS. `make lint`: `0 issues.`
9. **Docs.** `README.md` "Stack" and the `docs/architecture.md` pin table say `v1.26.0`. 0031-MADR has a note at D1.
10. **G-local.**
    - `make pre-add-check` on the 8 Go files: `8 file(s) clean (gofmt, golint, govulncheck)`.
    - `make preflight`: "preflight passed".
    - `make race`: exit 0, 36 packages ok.
    - `acpserver/testdata/session.golden` is unchanged.
11. G-hosts, on a copy of the tree (808 files): on the Linux machine (`go1.27.3 linux/amd64`) and the Windows laptop (`go1.27.3 windows/amd64`), `go test` exit 0, race exit 0 with 36 packages ok, `preflight` through `staticcheck` and its gate to `verify-build-metadata` (stopped at `git rev-parse HEAD`, exit 128: no commit, C6), and records and markdownlint exit 0.

One deviation: deviation 3, above.

### P13 record (2026-10-29): archtest names `/v2`

The base is P7's commit `3a2129c`.

1. **The constants.** `tuiLib` is `github.com/maccavelli/go-tui-lib/v2`. `tuiLibV1` is the old path. The comment cites
   0035-MADR D13.
2. **The v1 path.** `checkRule5` refuses an edge whose target is under `tuiLibV1` and not under `tuiLib`. The check comes
   before the `internal/tui` skip, so the refusal holds everywhere, `internal/tui` included. The message reads
   `<path> (go-tui-lib's v1 path; import github.com/maccavelli/go-tui-lib/v2, 0035-MADR D13)`.
3. **New cases in `TestImportRules`,** with the paths written out, not built from `tuiLib`, so that a constant which
   drifts fails:
   - `rule 5 v1 path refused`, from `internal/cli`;
   - `rule 5 v1 path refused in internal/tui`;
   - `rule 5 tuitest v2 in a test`, which is allowed;
   - `rule 5 tuitest v2 outside a test`, which is refused.

   The existing cases build their paths from `tuiLib`, so they move to `/v2` with it.
4. **Seen failing (C9).** On a copy of the tree with only the new cases added, so with P12's `check.go` (the v1
   constant), `go test ./internal/archtest/` exits 1:
   - `rule_5_v1_path_refused` and `rule_5_v1_path_refused_in_internal/tui`: `missing "go-tui-lib's v1 path"`;
   - `rule_5_tuitest_v2_in_a_test`: `unexpected edge`.

   `outside a test` passes there too, since the old prefix already refuses it, as step 4 expected. With P13 applied,
   the package passes.
5. **G-local.**
   - `make pre-add-check` on the two files: `2 file(s) clean (gofmt, golint, govulncheck)`.
   - `make preflight`: "preflight passed".
   - `make race`: exit 0, 36 packages ok.
6. G-hosts, on a copy of the tree (808 files): on the Linux machine and the Windows laptop (`go1.27.3`), `go test` exit 0, race exit 0 with 36 packages ok, `preflight` to `verify-build-metadata` (stopped at `git rev-parse HEAD`, exit 128: no commit, C6), and records and markdownlint exit 0.

No deviation.

### Deviation 4 (2026-10-29): P14 step 7, the prompt's words

- **Found.** Step 7 put `ask`'s "reason prompt, echo and cancel" texts in gobble's voice catalogue. At `v2.6.2` those are
  formats: `"%s: "`, `"%[1]s: %[2]s"`, and `"cancel"`, which is used only when the cancel answer has no label. The
  words a person reads are gobble's own labels and echoes. `permission.go` hard-coded them, outside the catalogue, a gap
  against 0022-MADR's "every text overridable".
- **The owner's answer, 2026-10-29:** "Catalogue the prompt words (Recommended)".
- **Decision.**
  - The prompt's words are 24 voice keys, `prompt.*` in `personality/keys.go`, with today's English in the boring
    catalogue and in `TestBoringIsTodaysText`. They cover:
    - the untitled title;
    - `Patterns: {patterns}`;
    - Allow and Reject;
    - once, session, project and everywhere;
    - `(hides the tool)`;
    - both say-why labels;
    - cancel;
    - D16's four labels;
    - the eight echoes.
  - `ask`'s three formats keep their defaults.
  - The built-in personalities leave the keys to boring, as trust's facts are: they are a person's answer to a tool call
    (0022-MADR D17, `leftBoring` in `personality/builtins_test.go`). A user's own catalogue may still override them.
- **Files added to scope:** `personality/keys.go`, `personality/builtin/boring/messages.md`, `personality/boring_test.go`
  and `personality/builtins_test.go`, in place of the "voice catalogue files" step 7 named generally.

### Deviation 5 (2026-10-29): archtest and test-only imports, and a fourth prompt test

- **Found.** `go test ./...` with P14's tests failed twice:
  - `TestImportRules/rule_5_module`:
    `internal/cli imports github.com/maccavelli/go-tui-lib/v2/inline/inlinetest (its closure is unknown)`.
    `archtest` learned the closures of go-tui-lib packages from `go list -deps`, which lists no test's imports.
  - `TestLinePersonalityQuestion` (`internal/cli/personality_run_test.go`, outside step 8's list) waited for the old one-line
    level-5 question: `condition not met within 5 s`.
- **The owner's answer, 2026-10-29:** "Teach archtest test closures". The options not taken were gobble's own `termtest`
  screen in the new tests, and stopping P14.
- **Decision.**
  - `loadGraph` runs `go list -e -deps -test -json ./...`. It strips a variant's `" [p.test]"` from a go-tui-lib package,
    and skips the module's test variants and `.test` mains, whose imports each package's own entry already carries.
  - `TestLoadGraphTestClosures` checks both halves.
  - `TestLinePersonalityQuestion` reads `ask`'s drawing.
  - `docs/architecture.md`'s import-rules line names `-test`.
- **Files added to scope:** `internal/archtest/check.go`, `internal/archtest/check_test.go` and
  `internal/cli/personality_run_test.go`.

### P14 record (2026-10-29): go-tui-lib/v2 `v2.6.2`, and `ask` for the permission prompt

The base is P13's commit `3f92ae9`. 0029-PLAN P2 is carried out here.

1. **C2.** `v2.6.2` is the newest `v2` tag. P2's `check-libs` run of it passed every check, `archtest` included.
2. **The requirement.**
   - `go get github.com/maccavelli/go-tui-lib/v2@v2.6.2`, then `go mod tidy`.
   - Compared with P13's `go.mod`, it adds `github.com/maccavelli/go-tui-lib/v2 v2.6.2` and exactly 0032-MADR A1.2's five
     modules, at its versions: `x/ansi v0.11.8`, `displaywidth v0.11.0`, `uax29/v2 v2.7.0`, `go-colorful v1.4.1` and
     `go-runewidth v0.0.27`.
   - `x/ansi` is direct, since the tests import it, which A1.1 allows. No sixth module.
3. **The question.**
   - `permissionQuestion(v, r)` builds `ask.Question`. For a tool call's exact eight options (by acpserver's ids) it
     gives D1's rows:
     - Allow `[1]`…`[4]` and Reject `[5]`…`[8]`, labelled once, session, project and everywhere, each echoed in
       words;
     - `(hides the tool)` ending `[8]` when `Hides` is set;
     - `[f] reject and say why` (`Reason: true`).
   - It capitalises the whole title (`FullTitle`, else the title) and adds `?`. It shows `Patterns:` when `Saves` is set.
   - For every other question it gives D2's one row: numbered by name where kinds repeat, else `y`/`a`/`n`/`N`, with
     `[f] deny and say why` when `reject_once` is offered.
   - `OnceOnly` is a note in parentheses. `[c] cancel` ends the last row.
4. **The prompt.**
   - `askOn` draws in a region with no cursor query, so nothing waits on a terminal's reply.
   - It decodes keys with `termkey`, gives `read` a deadline when the asker or the decoder waits on the clock, cancels
     at the end of input, and returns the rows `View` leaves.
   - `askPermission` runs it in raw mode, through the editor's reader, at `promptWidth` (stderr's, at most 120, or 120
     off a terminal). It records the rows in the transcript and maps the response to the answer.
5. **The trust question** uses the same `askOn`, reading stdin with no deadline. Its answer switch is unchanged.
6. **Removed:** `permissionLine`, `chooseKey`, `labelOf`, `readWhy`, `cutWhy`, `maxWhy`, `chooseWhy`, `sayWhy` and the
   raw-key constants, together with the code in `trust.go` that called them. `ask.DefaultReasonLimit` is the same
   2048 bytes.
7. **The catalogue,** as deviation 4.
8. **Tests.**
   - `TestPermissionQuestion` asserts each question exactly:
     - the tool call;
     - the tool call hiding the tool;
     - once-only;
     - every kind once;
     - numbered;
     - untitled.
   - `TestAskOn` covers each answer, a reason with backspace, `c`, Ctrl+C, the end of input, Ctrl+C inside a reason,
     and a 2048-byte cut.
   - `TestAskOnDraws` draws D1's example at 60 and 100 columns on `inline/inlinetest`'s screen, with every row no wider
     than the width, and checks `promptWidth` at 200 columns, which gives 120.
   - `TestPermissionPrompt` and `TestPermissionPromptOnceOnly` run in the line session.
   - `TestTrustQuestion`, `TestTrustQuestionNoParent` and `TestLineViewTrust` read the drawing.
   - `TestPermissionLineNumbered` is removed; its cases are `TestPermissionQuestion`'s "numbered" and `TestAskOn`'s
     ignored key.
   - `TestLinePersonalityQuestion` follows deviation 5.
9. **Seen failing (C9), on copies of the tree:**
   - **The builder without `Reason: true`:** `TestAskOn/why` and `TestAskOn/ctrl+c_in_a_reason` fail.
   - **The cap doubled** (`2*editor.MaxWidth`): `TestAskOnDraws` fails with
     `promptWidth at 200 columns: 200, want 120`.

     The first attempt at this break, which dropped `editor.MaxWidth`, did not compile (the `editor` import was left
     unused). Only the compiling break is evidence.
   - **archtest's loader without `-test`:** `TestImportRules/rule_5_module` fails (`its closure is unknown`), and so
     does `TestLoadGraphTestClosures` (`no closure for …/inline/inlinetest`).
   - **The variant filter dropped:** `TestLoadGraphTestClosures` fails with
     `a test build's package listed: …/llm/provider [….test]`, among others.
10. **0029.** 0029-MADR has a dated amendment of D1 and D2 for PQ4 (a) and deviation 4. 0029-PLAN has a dated note that
    P2 is carried out here. P3, on real terminals, stays that PLAN's.
11. **Docs.**
    - `README.md` "Stack" names go-tui-lib `/v2` `v2.6.2`.
    - `docs/architecture.md` adds the pin, `ask` in the CLI and `-test` in the import rules.
    - 0031-MADR has a note at D1. 0035-MADR has notes at D13 and D14.
12. G-local: `make pre-add-check` on the 10 Go files, clean (gofmt, golint, govulncheck); `make preflight` passed after two fixes it caught: `go mod tidy` moved `x/ansi` to a direct requirement once the tests imported it, and golangci-lint's modernize asked for `strings.SplitSeq`. `make race`: 36 packages ok. `make perf`, each within 0030-MADR D5's three times: ReplyRender 32.98 µs (baseline 30), Reprint 9.75 ms (9.4), LineSessionStart 945 µs (830), KeyDraw 1.52 µs (1.5).
13. G-hosts, on a copy of the final tree (808 files), `go1.27.3`: on the Linux machine and the Windows laptop, `go test` exit 0, race exit 0 with 36 packages ok, `preflight` to `verify-build-metadata` (stopped at `git rev-parse HEAD`, exit 128: no commit, C6), records and markdownlint exit 0, and `make perf` passed (the Linux machine: ReplyRender 143.8 µs of 110, Reprint 33.2 ms of 29, LineSessionStart 3.35 ms of 2.9, KeyDraw 5.94 µs of 5; the Windows laptop: 28.1 µs of 54, 7.13 ms of 12, 3.52 ms of 6.9, 1.04 µs of 1.4). `make probe-conhost` on the Windows laptop: `--- PASS: TestConhost`. A first host run, on the tree before the two fixes, passed its tests, race, perf and probe-conhost and stopped `preflight` at the tidy check.

Two deviations: 4 and 5, above.

### P1 record (2026-10-30): the docs

Approved by the owner's "committed proceed", 2026-10-30. The base is P14's commit `c54ecf3`.

1. **The OpenAI login's message.** `internal/cli/auth.go:192` reads `openai's ChatGPT sign-in is not in gobble yet
   (0031-PLAN P4b); use --api-key -`, and `internal/cli/auth_test.go:50` expects it. No other code or guide carries the
   old message (a search found none).
2. **The credentials guide.** `docs/guides/credentials.md` says the ChatGPT sign-in comes with 0031-PLAN P4b, since the SDK
   has had it since `v1.15.0`. Its quoted error matches step 1.
3. **`docs/README.md`.**
   - "Build order" gains a bullet for each PLAN from 0023 to 0036. Each states that PLAN's frontmatter status and
     points to its execution record. The plan adds 0036, which did not exist when it was written.
   - The plans list gains item 6.
   - The "I want to…" row points to 0035-MADR. The #5 link is kept.
4. **Seen failing (C9).** On a copy of the tree with the old message put back in `auth.go`, `TestAuthLogin` fails.
5. **The gates.**
   - **G-local.** `make pre-add-check` on the two files: clean. `make preflight` passed. Its first run failed
     markdownlint on a new bullet at 256 characters (MD013, the limit 200), so the bullets were wrapped at 120. Two
     bullets also no longer claim an execution record: 0033 (proposed) and 0034 (never executed). `make race`: 36 ok.
     `make perf`: ReplyRender 32.6 µs (baseline 30), Reprint 9.46 ms (9.4), LineSessionStart 877 µs (830), KeyDraw
     1.51 µs (1.5).
   - **G-hosts,** on the Linux machine and the Windows laptop: `go test` and race exit 0, 36 ok; `preflight` to
     `verify-build-metadata` (no commit, C6); `make perf` passed; `make probe-conhost` on the Windows laptop passed
     `TestConhost`.
   - The hosts' first records run (exit 2) had the tree from before the wrap. A second run, on the fixed tree, gave
     `make check-records markdownlint` exit 0 on both.

No deviation.

### P5 record (2026-10-30): go-llmprovider-sdk's copies go

The base is P1's commit `9c77d5c`.

1. **Offline.**
   - `offline()` and `unreachable()` are deleted, along with the `runtime` and `syscall` imports they alone used.
   - `mapError`'s `case !kind && offline(err)`, and its `kind`, are removed. The `ErrOffline` row of P4 carries offline
     alone.
   - The stream's guard records `offlineURL` from the `*url.Error` when `errors.Is(err, llmprovider.ErrOffline)`. The
     SDK's `transport.WrapOffline` keeps the original error in the chain.
   - `Probe`, `dialProbe`, `probeTimeout` and `offlineURL` stay.
   - `mapError`'s one-case `switch` became an `if`.
2. **Empty answer.** `sdkEmptyAnswer` is deleted. `emptyAnswer` and `refusedAnswer` read
   `errors.Is(ae.Kind, llmprovider.ErrEmptyAnswer)`: the SDK's wire sets `Kind: llmprovider.ErrEmptyAnswer`, a child of
   `ErrIncomplete` (`internal/wire/finish.go:35`). `Reason` still chooses between them: empty or `stop`, against
   `content_filter`. `TestWireEmptyAnswer`'s comment no longer pins a message; its assertions already read
   `llm.ErrEmptyReply` and `llm.StopRefusal`.
3. **`max`.**
   - `reasoningFor` maps `"max"` to `llmprovider.EffortMax`, and `TestReasoningLevels` expects it.
   - No display of `WarningEffortDegraded` is built. gobble reads no SDK warning today, and 0036-MADR D3 is the record
     that makes them reach it.
4. **Docs.**
   - `llm/doc.go`'s table reads `max → max (stepped down by the SDK to the highest the model takes)`.
   - `docs/guides/choosing-a-model.md` says `max` is sent as `max` and stepped down where the model does not take it.
   - `llm/llm.go:45` reads `ImageInput stays false until gobble sends images (0035-MADR D18; 0036-MADR D7).`
5. **Seen failing (C9), on copies:**
   - With the `ErrOffline` row removed, `TestWireOffline` fails:
     `err llm: unavailable: llmprovider: offline: … connection refused, want llm.ErrOffline`. Offline now rests on the
     row alone.
   - With `refusedAnswer` reading `ae.Reason == ""`, `TestWireEmptyAnswer` fails, at `interactions`.
6. **Assert by search:** no `sdkEmptyAnswer`, `func offline(` or `unreachable(` in `llm/`.
7. **G-local.**
   - `make pre-add-check` on the five files: clean.
   - `make preflight`: "preflight passed".
   - `make race`: 36 ok.
8. G-hosts, on a copy of the tree (808 files), `go1.27.3`: on the Linux machine and the Windows laptop, `go test` and race exit 0, 36 ok; `preflight` to `verify-build-metadata` (no commit, C6); records and markdownlint exit 0.

No deviation. 0036-PLAN's P0, which waits for this phase, may now run on the owner's word.

### Deviation 6 (2026-10-30): P6's files beyond its list

- **Found.** Step 3 takes the workspace from the resolved credential "where `providerOptions(id, o)` is filled", and has
  `lookupModel` pass it. Doing so meant files outside the step's list:
  - `internal/cli/main.go` and `internal/cli/credentials.go`: `knownKeys` caches the whole `auth.Credential`, with a new
    `knownWorkspace(id)` beside `knownKey(id)`, so a lookup runs no key command again;
  - `internal/cli/agent.go:97`: `ModelInfo` gains the workspace;
  - `llm/provider/model.go`: `ModelInfo(id, apiKey, workspace, logins, model, mj)`, and `modelInfoFrom` and
    `lookupModel` gain the workspace. It is an exported change in a beta package, below v1, so `apidiff` does not
    gate it.
- **Handling.** These were made in the course of the step, as its own wording requires, not on a separate answer.
  They are recorded here so that the phase is not described as narrower than it was, and the owner is told in the
  handoff. No decision changes.
- **Files added to scope:** the four above, and the tests of the listed files (`internal/auth/logins_test.go`,
  `llm/provider/model_test.go`, the new `llm/provider/workspace_test.go` and `acpserver/workspace_test.go`).

### P6 record (2026-10-30): the Anthropic workspace

PQ1 (a) and PQ2 (a). The base is P5's commit `43649b8`.

1. **The store.**
   - `auth.Entry` gains `Workspace` (`"workspace"`).
   - `Set` is replaced by `SetKey(ctx, path, provider, key, workspace)`. It merges `type`, `key` and `workspace` into the
     entry as `SetOAuth` does: fields gobble does not read are kept, an oauth login's fields are dropped, and with no
     workspace the field goes.
   - Every caller moved: `auth.go`, `login.go`, `acp.go`, `piimport.go` (with `""`), and the store's own tests.
   - `TestStoreRoundTrip` is unchanged and passes byte for byte. The merge keeps Pi's order, and a key over `{type, key}`
     gives the same bytes.
   - New `TestStoreKeyKeepsUnknownFields` checks four things: Cloudflare's `env` is kept, the workspace is written, an
     oauth entry's tokens go under a key, and a later `SetKey` with none removes the workspace.
2. **The resolver.** `auth.Credential` gains `Workspace`, set only from an auth.json `api_key`. New `TestResolveWorkspace`
   checks that a `--api-key` runtime key carries none.
3. **The provider.**
   - `provider.Options` gains `Workspace`.
   - `New` appends `llmprovider.For(llmprovider.ProviderClaude, claude.WithWorkspace(…))`, and `List` appends
     `For(…, catalog.WithClaudeWorkspace(…))`, each only when it is set. `lookupModel` appends the listing option too.
   - `providerOptions` fills it from `knownWorkspace(id)`, so the run's provider, a switched-to provider, the small
     model, the listing and sign-in all get it. `agent.go`'s `ModelInfo` gets it as well.
4. **The wizard.** `SignIn` passes `Existing.Workspace` from `o.Workspace` and returns `Credential.Workspace` from
   `res.Workspace`, and `signInTo` stores it.
5. **PQ1 (a).**
   - `gobble auth login` has `--workspace ID`, refused for any provider but Claude, and for a control character
     (`checkWorkspace`).
   - `gobble acp --login claude` asks `Workspace ID (empty for none): ` after the key, when stdin is a terminal.
   - `TestAuthLoginWorkspace` checks the stored field and both refusals. The terminal prompt is not exercised by a test,
     since it needs a real terminal's stdin, as `readAPIKey`'s own prompt does.
6. **Errors.**
   - `llm.ErrWorkspace` ("llm: workspace needed").
   - `sentinels` gains `ErrWorkspaceRequired` and `ErrWorkspaceNotFound`, both mapping to it.
   - `FailureKind` gives `workspace`. `StateOf` gives not listed, with the way to name one.
   - `promptError` answers `workspaceAdvice` alone, without the service's message.
   - New `TestWireWorkspace` uses the SDK's measured refusal text and checks the error and the class. It also checks that
     Claude's request carries `anthropic-workspace-id: wrkspc_1` while OpenAI's carries none.
   - New `TestWorkspaceWords` checks the agent's wording.
7. **`docs/guides/credentials.md`** has "An Anthropic workspace" under "Storing a key". `ANTHROPIC_WORKSPACE_ID` is not
   read (PQ2 (a)).
8. **Seen failing (C9), on copies:**
   - **`SetKey` merging into nothing, the old replace:** `TestStoreKeyKeepsUnknownFields` fails with
     `"CLOUDFLARE_ACCOUNT_ID": "x" not kept`.

     The first attempt's anchor matched both `SetKey` and `SetOAuth`, and the copy script refused it. It was rerun with
     an anchor unique to `SetKey`.
   - **The `For(…)` dropped from `New`:** `TestWireWorkspace` fails with
     `claude: anthropic-workspace-id "", want "wrkspc_1"`.
   - **`checkWorkspace` not refusing other providers:** `TestAuthLoginWorkspace` fails.
9. **G-local.**
   - `make pre-add-check` on the 24 Go files: clean.
   - `make preflight`: "preflight passed".
   - `make race`: 36 ok.
   - `make perf`: ReplyRender 32.7 µs (baseline 30), Reprint 9.60 ms (9.4), LineSessionStart 983 µs (830), KeyDraw
     1.52 µs (1.5).
10. G-hosts, on a copy of the tree (810 files), `go1.27.3`: on the Linux machine and the Windows laptop, `go test` and race exit 0, 36 ok; `preflight` to `verify-build-metadata` (no commit, C6); records and markdownlint exit 0; `make perf` passed on both; `make probe-conhost` passed `TestConhost` on the Windows laptop.

One deviation: deviation 6, above. The go-llmprovider-sdk phases, P4 to P6, are complete.

### P8 record (2026-10-30): the constructors

The base is P6's commit `d204826`.

1. **The five literals,** replaced as the table has them. The signatures were read again with `go doc` at `v1.26.0`:
   - `UpdateCurrentMode(SessionModeId)` in `modes.go`;
   - `UpdateConfigOptions`, `UpdateSessionTitle(Patch[string])` and `UpdateAvailableCommands` in `commands.go`;
   - `UpdateUsage(used, size)` in `prompt.go`'s `contextUpdate`, whose `Cost` is set on the returned `UsageUpdate`.
2. **Assert.**
   - No `acp.SessionUpdate{` with one of the five fields remains in non-test `acpserver`.
   - `git diff --exit-code acpserver/testdata/session.golden` exits 0.
   - `go test ./acpserver/... ./acpclient/...` passes.
3. **Seen failing (C9).** On a copy with `UpdateUsage`'s arguments swapped, `TestSession` fails with
   `transcript differs from testdata/session.golden at line 6`, and so does `TestCatalogPriced`. The golden covers the
   usage update.
4. **G-local.**
   - `make pre-add-check` on the three files: clean.
   - `make preflight`: "preflight passed".
   - `make race`: 36 ok.
5. **G-hosts,** on a copy (810 files), `go1.27.3`. On the Linux machine and the Windows laptop: `go test` and race exit
   0, 36 ok; `preflight` to `verify-build-metadata` (no commit, C6); records and markdownlint exit 0.

No deviation.

### P9 record (2026-10-30): the refusals and the permission option

The base is P8's commit `f5a94d0`.

1. **`acpclient`'s `handler`** embeds `acp.ClientRefusals`, with a comment citing 0014-MADR D3. The nine methods and the
   comment above them are gone. `var _ acp.Client = (*handler)(nil)` still compiles.
2. **`acpserver.Agent`** embeds `acp.AgentRefusals`. Removed:
   - `DeleteSession` and `Logout`;
   - the "methods below are not implemented" comment, which `Logout` alone sat under.

   Its own `CloseSession`, `ListSessions` and `ResumeSession` take precedence. The existing tests show each still
   answers: `TestCloseWaitsForTurn`, `TestListSessions`, `TestListSkipsNonAbsoluteCwd`, `TestResumeWithoutReplay`,
   `TestResumeConnects` and `TestOverrideResume`.

   **`acptest.ScriptAgent`** embeds it too. Its `DeleteSession`, `Logout`, `ListSessions` and `ResumeSession` stubs
   are gone. `Authenticate` stays, ungated, as do its `SetSessionMode` and `SetSessionConfigOption`, which
   `AgentRefusals` does not cover. `acptest.Client` is unchanged.
3. **The reject loop** is `acp.PickPermissionOption(r.Options, RejectOnce, RejectAlways)`.
4. **Tests.**
   - New `acpclient/refusals_test.go`, `TestClientRefusals`. Each of the eight request methods answers
     `acp.IsMethodNotFound`, and `CompleteElicitation` returns nil.
   - The existing `TestPermissionDeclined` is the permission test step 4 names. It covers reject once, reject always,
     and cancelled where no reject option is offered.
   - `delete_test.go` and the `-32601` tests pass unchanged.
5. **Seen failing (C9), on copies:**
   - A `handler` overriding `ReadTextFile` to return nil makes `TestClientRefusals` fail with
     `fs/read_text_file: <nil>, want method not found`.
   - With `PickPermissionOption` given the allow kinds, `TestPermissionDeclined/reject_once` fails.
6. **Assert.**
   - No `NewMethodNotFound` remains in `acpserver/agent.go`.
   - In `acpclient/conn.go` one remains (`:458`): the extension-method answer, as step 6 expected for that switch.
   - `session.golden` is unchanged.
7. **G-local.**
   - `make pre-add-check` on the four files: clean.
   - `make preflight`: "preflight passed".
   - `make race`: 36 ok.
8. G-hosts, on a copy (811 files), `go1.27.3`: on the Linux machine and the Windows laptop, `go test` and race exit 0, 36 ok; `preflight` to `verify-build-metadata` (no commit, C6); records and markdownlint exit 0.

No deviation.

### Deviation 7 (2026-10-30): P10 step 3's unknown variant

- **Found.** Step 3 asked `TestTranslate` to see `Variant == "unknown"` for a notification whose `sessionUpdate` is a
  number. The SDK refuses one on the agent's side before it is sent:
  `session/prompt: {"code":-32603,…"error":"SessionUpdate: discriminator \"sessionUpdate\": json: cannot unmarshal number into Go value of type string","update":11}`.
  So no client receives one.
- **Probe.** On a scratch copy, `{"sessionUpdate":"gobble_future_update","x":1}` reached the client as `KindOther`, and
  `Kind()` named it `gobble_future_update`.
- **The owner's answer, 2026-10-30:** "Test a future variant (Recommended)".
- **Decision.** The third case is that future variant: `Variant == "gobble_future_update"`, with the `Kind` still
  `KindOther`. That is the case `Kind()` exists for, a variant gobble has no constant for.
- **Files added to scope:** none.

### P10 record (2026-10-30): `Kind()`

The base is P9's commit `8e3a6b3`.

1. **`Variant`.** `acpclient.Update` gains `Variant`, set by `translate` from `n.Update.Kind()` for every update. `KindOther`
   and `KindCompaction` are unchanged.
2. **`TestTranslate`.**
   - It sends the future variant as the twelfth update, whose kind is `KindOther`.
   - It asserts `Variant` for three updates: `agent_message_chunk` for the first, `config_option_update` for the
     `KindOther` at index 10, and `gobble_future_update` for the twelfth, as deviation 7 has it.
3. **Seen failing (C9).** On a copy with `Variant` left unset, `TestTranslate` fails:
   `update 0: variant "", want "agent_message_chunk"`, and so on for each.
4. **G-local.**
   - `make pre-add-check` on the two files: clean.
   - `make preflight`: "preflight passed".
   - `make race`: 36 ok.
5. G-hosts, on a copy (811 files), `go1.27.3`: on the Linux machine and the Windows laptop, `go test` and race exit 0, 36 ok; `preflight` to `verify-build-metadata` (no commit, C6); records and markdownlint exit 0.

One deviation: deviation 7, above.

### Deviation 8 (2026-10-30): P7's gate read files staticcheck does not

- **Found,** in P11's `make preflight`, which deleted `acpserver/rewrite.go`:
  `FileNotFoundError: [Errno 2] No such file or directory: 'acpserver/rewrite.go'` from
  `scripts/check_staticcheck_ignored.py`, on each GOOS. The gate listed `git ls-files '*.go'`, which keeps a tracked
  file deleted but not yet staged, and it skipped new, untracked files until they were staged. It is a defect in P7's
  gate, found in this plan's live run.
- **The owner's answer, 2026-10-30:** "Fix in its own commit (Recommended)".
- **Decision.** The gate lists `git ls-files -co --exclude-standard '*.go'`: tracked and untracked files, less the
  ignored. It skips a path missing on disk, so it reads what staticcheck reads. The fix is committed alone, before P11's
  commit.
- **Seen failing (C9),** on a scratch clone of P10's commit:
  - **The old gate, with a tracked Go file deleted:** `FileNotFoundError` (exit 1).
  - **The old gate, with an untracked file holding `//lint:ignore SA1019 no record`:** exit 0. It passed the file it
    never read.
  - **The fixed gate:** exit 1 on that untracked file (`the reason cites no NNNN-MADR or NNNN-PLAN record`), and no
    crash with the file deleted.
- **Checks.**
  - `python3 -m py_compile` on the script: exit 0.
  - `make staticcheck` on the working tree, with P11's deletion in it: the gate passed on each GOOS.
  - The host gates run with P11's, on a copy holding this fix.
- **Files added to scope:** `scripts/check_staticcheck_ignored.py`.

### P11 record (2026-10-30): the connection's end

The base is P10's commit `7ed439f`, then deviation 8's `bb4e5eb`.

1. **`acpserver.Serve`** calls `acp.ServeAgent(ctx, agent, in, out, WithLogger, WithInboundRewrite(ResolveStdioCommands(exec.LookPath)),
   WithPromptOverlapPolicy(PromptOverlapConcurrent))`, then `agent.Close()`.
   - `endWait`, `ended` and `handlersDone` are gone.
   - The comment keeps 0016-MADR A1 and 0019-MADR D6, and its doc says how each end is reported.
   - `ServeAgent` binds the agent through the constructor's `SetAgentConnection`, as verified when the plan was written.
2. **`rewrite.go` is deleted.** `TestResolveBareCommands` and `TestResolveBareCommandsKeepsRelative` exercise
   `stdioRewrite`, a test variable holding `acp.ResolveStdioCommands(exec.LookPath)`, with the same cases, and pass.
   `TestServeResolvesBareCommand` is unchanged and passes.
3. **The client.** `Conn.Close` waits through `c.conn.Connection().WaitContext`, bounded by `closeTimeout`, and
   `handlersDone` is gone.
   - golangci-lint's contextcheck then reported `Start`'s error path calling `Close`, which builds its own context
     (`acpclient/conn.go:153`). So `Close()` is now `closeFrom(context.Background())`, and `Start` calls `closeFrom(ctx)`.
     Its waits use `context.WithoutCancel(base)`: base's values, never its cancellation, so `closeTimeout` alone bounds
     them, as before.
   - The helper is named `closeFrom`, since revive's `confusing-naming` refused `close` beside `Close`.
   - This is a fix to the step's own code, not a deviation.
4. **`acptest`.**
   - `Connect`'s cleanup waits on both connections' `WaitContext` within one `closeTimeout`, and its `handlersDone` is
     gone.
   - `ScriptAgent.Serve` is `acp.ServeAgent(ctx, a, in, out, …)`.
   - `ScriptAgent` keeps its connection through `SetAgentConnection`, which the SDK's constructor calls before it reads,
     where `Serve` used to set it after the constructor returned.
5. **New `TestServeShutdownCancelsPrompt`.** A prompt in flight, its turn waiting on its context (the `slowStop`
   provider), is answered `"stopReason":"cancelled"` when `Serve`'s context ends, and `Serve` returns an error matching
   `context.Canceled`.
   - `TestServeReportsABrokenInput`, `TestServeAnswersBeforeEndOfInput` and `TestServeClosesConnection` pass unchanged,
     as do `close_test.go`'s two tests and `TestCleanupWaitsForHandlers`.
   - The overflow-as-clean defect is fixed by `ended`'s deletion. Its evidence is `ServeAgent`'s use of `EndedCleanly`
     (`serve.go:39`), as step 5 had it.
6. **Seen failing (C9).** The new test run against a scratch clone of P10's commit, with only the test added, fails:
   `output ended before the answer to 3`. The old `Serve` closed the connection under the prompt.
7. **Assert.**
   - No `handlersDone`, `func ended` or `resolveBareCommands` remains in the module.
   - No `endWait` remains in `acpserver` or `acpclient`. `agent/agent.go`'s `turn.endWait` is the agent's retry
     wait, another thing by the same name, and stays.
   - `session.golden` is unchanged.
8. **G-local.**
   - `make pre-add-check` on the changed files: clean.
   - `make preflight`: "preflight passed", after deviation 8's fix and step 3's contextcheck fix. Its earlier runs
     failed on those two, in that order.
   - `make race`: 36 ok.
9. G-hosts, on a copy (811 files), `go1.27.3`: on the Linux machine and the Windows laptop, `go test` and race exit 0, 36 ok; `preflight` to `verify-build-metadata` (no commit, C6); records and markdownlint exit 0. A first host run, before step 3's fix, stopped `preflight` at golangci-lint's contextcheck, the same finding.

No deviation of its own; deviation 8, P7's gate, was found here and fixed in its own commit first.

### Deviation 9 (2026-10-30): P12 held for acp-go-sdk2

- **Found.** With `acpclient.Start` on `agentproc.InProcess` and `Close` as `Process.Close()` and then the connection's
  `WaitContext`, `TestCloseBoundedWhenAHandlerHangs` never returned; `go test`'s 10 m timeout ended it. The goroutine
  dump showed `Process.Close` → `Process.Wait` → `Connection.Wait` → `sync.WaitGroup.Wait`, with the client's update
  handler blocked. At `v1.26.0`, `Process.Close` is its steps and then `Wait` (`agentproc/agentproc.go:517-519`), and
  `Wait` waits for the connection without bound (`:389`, `connection.go:870`). The grace bounds the agent, not the
  client's handlers. Step 4 required the two close tests to pass unchanged, so the step could not be met.
- **The owner's answer, 2026-10-30:** "Hold P12; ask the SDK (Recommended)". The options not taken were bounding the
  SDK's `Close` in gobble, which would rebuild the wait P11 removed, and dropping P12.
- **Decision.**
  - P12's uncommitted edits, to `acpclient/conn.go` alone and made in this phase, are reverted (`git checkout --
    acpclient/conn.go`). `acpclient.Start` stays as P11 left it: gobble's pipes, bounded by `closeTimeout`, its waits
    through `WaitContext`.
  - The issue is [acp-go-sdk2#12](https://github.com/maccavelli/acp-go-sdk2/issues/12). It asks for `Close` to bound the
    connection's wait by the grace and report the handlers it leaves, or for a `CloseContext`, or a way to stop without
    `Wait`.
  - P12 is held until a tag carries one of those. Then it runs as written, its step 4's expectation unchanged.
  - The plan continues with P15.
- **Files added to scope:** none. The issue is outward-facing, and was filed on the owner's answer.

### Deviation 10 (2026-10-30): P15 step 3's bidi claim

- **Found.** Step 3's `TestTerminalCleaning` expected a bidi override (U+202E) to be absent from terminal output.
  safetext's `Text` rules, which PQ5 chose, keep bidi, as `term.Sanitize` does: the test failed with
  `terminal true: "‮" shown in "ab‮cd end\n"`. The MADR's facts list what `Text` removes, and bidi is not
  among them. The step's claim was the plan's error.
- **The owner's answer, 2026-10-30:** "Keep Text; fix the test (Recommended)". The options not taken were escaping bidi
  in terminal model text, and safetext's `Strict` rules.
- **Decision.**
  - On a terminal, the test asserts that the zero-width space and the OSC 52 write are gone and the bidi override is
    kept.
  - Off a terminal, it asserts `term.Sanitize`'s bytes, the zero-width space kept.
  - Stripping or escaping bidi in model text is not this plan's. It would be a new decision.
- **Files added to scope:** none.

### P15 record (2026-10-30): `glyph`, and `safetext` on terminal paths

The base is deviation 9's commit `4ca7992`.

1. **`glyph`.**
   - `term.Glyphs`, `term.Unicode` and `term.ASCII` are gone from `term/style.go`; `Style.Wrap` stays.
   - `glyphsFor(c)` is `glyph.For(c.Unicode)`. Every user reads `glyph.Set`, with the fields renamed: `Arrow` →
     `Heading`, `Check` → `Done`, `Cross` → `Failed`, `Bar` → `MeterFill`, `BarEmpty` → `MeterTrack`. That covers
     `view.go`, `print.go`, `render/plan.go`, `render/tool.go`, `render/status.go`, and their tests.
   - The Unicode set's values are gobble's own. ASCII's ellipsis is `~` where it was `...`, by Q3's direction, and the
     one test that showed it, `TestToolOutputBounds`, now expects `~ 1 lines hidden`.
   - `TestGlyphSets` moved to `internal/cli/glyphs_test.go`. It checks that each field gobble draws is set in both
     sets, and that ASCII's are ASCII.
2. **`safetext`, by PQ5 (a).**
   - `term.Clean(s, terminal)` and `term.CleanLine(s, terminal)` use `safetext.Clean` with `Text()` and `Line()` on a
     terminal, and `term.Sanitize` elsewhere.
   - The line view's model text, thinking, diffs, terminal ids, plan and tool title and output, and `/help`'s command
     list call them with `caps.Out.TTY`. `ToolOutput` and `CommandList`, with `commandLines`, `entryOf` and
     `nameColumn`, gain a `terminal` argument.
   - The editor's header uses `CleanLine(…, true)`, since the editor draws on a terminal.
   - `term.Sanitize` stays at `print.go:201`, for print mode, and in `complete/protocol.go`, whose shell-completion
     output a script reads. That makes three calls in two files.
   - `EscapeBidi`, its helper `isBidi`, its test and the test's bidi variables and `escaped` helper are deleted. They
     were dead code.
3. **Tests.**
   - `TestTerminalCleaning` (`internal/cli/clean_test.go`) is as deviation 10 has it. Its characters are written as
     `\u` escapes, for staticcheck's ST1018.
   - `TestSanitize` stays, for print mode.
   - No view golden changed, and `print-json.golden` and `print-stream-json.golden` are byte-identical
     (`git diff --stat` on `*.golden` and `testdata` is empty).
4. **Seen failing (C9).** On a copy with the reply's cleaning back on `term.Sanitize`, `TestTerminalCleaning` fails with
   `terminal true: "​" shown`.
5. **Assert.** `term.Sanitize` remains only at `internal/cli/print.go:201` and `internal/cli/complete/protocol.go:130`
   and `:140`.
6. **G-local.**
   - `make pre-add-check` on the 18 Go files: clean.
   - `make preflight` passed after staticcheck's two findings were fixed: ST1018 on the test's raw format characters,
     and U1000 on `isBidi` and the dead test helpers.
   - `make race`: 36 ok.
   - `make perf`: ReplyRender 33.4 µs (baseline 30), Reprint 9.85 ms (9.4), LineSessionStart 909 µs (830), KeyDraw
     1.55 µs (1.5).
7. G-hosts, on a copy of the final tree (812 files), `go1.27.3`: on the Linux machine and the Windows laptop, `go test` and race exit 0, 36 ok; `preflight` to `verify-build-metadata` (no commit, C6); records and markdownlint exit 0; `make perf` passed on both; `make probe-conhost` passed `TestConhost`. A first host run used the tree before staticcheck's fixes.

One deviation: deviation 10, above.

### P16 record (2026-10-30): `pick`

By PQ6 (a), no code changes. The base is P15's commit `884a6a0`.

1. The MADR has a dated note at D18: `pick` is new UI, not a replacement, for a record of its own.
2. The MADR has a dated note at D14 step 3: neither `pick` nor `termcap` is adopted, and `internal/cli/term/caps.go` stays.
3. `docs/README.md`'s row is updated.
4. Gates: `make check-records markdownlint`, exit 0. Records only.

No deviation.

### Deviation 11 (2026-10-30): P17 held for go-tui-lib's `lineedit` key cost

- **Found.** Step 5 holds `BenchmarkKeyDraw`, the Composer's draw of one key, to 0030-MADR D5's budget of three times
  each host's baseline. At `v2.6.2` the whole key (`HandleInput`, then draining `NextEvent`) breaches it on all three
  hosts:

  | Host | Baseline | Budget | `lineedit` key |
  | --- | --- | --- | --- |
  | macOS | 1.5 µs | 4.5 µs | 14.9 µs, 53 allocations |
  | the Linux machine | 5 µs | 15 µs | 42.0 µs, 53 allocations |
  | the Windows laptop | 1.4 µs | 4.2 µs | 11.8 µs, 53 allocations |

  The old benchmark timed only the renderer's draw of rows laid out beforehand. On macOS, measured on scratch copies
  with three runs of 2 s each:
  - gobble's whole key at `6c5eb02` (buffer, layout, draw) is 4.24 µs, with 17 allocations;
  - `inline.Region.Draw` of rows laid out beforehand is 1.92 µs, with 5;
  - `Composer.View` alone is 6.5 µs, with 42 allocations and 9.4 KB.

  The new whole key is 3.4× the old one. A CPU and memory profile puts 92% of the bytes allocated, and about 19% of
  the CPU, in `lineedit.layout` → `layLine`: every key lays the whole line out again. Backspace's `lineinput.Line.cut`
  rebuilds the text with `strings.Join`, about 10% of the CPU. No host option changes either. MADR D14 says
  0030-MADR D5's benchmarks are no worse after any step, so step 4 cannot be met at this tag.
- **The owner's answer, 2026-10-30:** "Hold P17 for go-tui-lib". The options not taken were:
  - two budgets: the draw's old measure kept, plus a whole-key benchmark with new baselines, the regression recorded;
  - deferring `lineedit`, PQ7 (c).
- **Decision.**
  - Step 5 stands as written. P17 waits for a go-tui-lib tag whose Composer key is within the budgets above on every
    host.
  - The issue is [go-tui-lib#13](https://github.com/maccavelli/go-tui-lib/issues/13), filed on the owner's word the same day: "lineedit: a key costs about 3.4 times what
    gobble's editor did, mostly from laying the whole line out again". It gives the benchmark, the table above and
    the profile. It asks for one of four things:
    - layout from the changed row;
    - the layout kept between draws, by the text's version and the width;
    - fewer allocations in `layLine`, with each cluster's width measured once;
    - `Line.cut` without rebuilding the text.
  - P18 writes the transcript's reprint between P17's region's `Clear` and `Draw` (its step 5), and P19 closes both.
    So P18 and P19 wait with P17.
  - P17's work stays in the working tree, uncommitted, because step 8's commit needs step 5. It is described in the
    P17 record below.
- **Files added to scope:**
  - `internal/cli/prompt.go` and `internal/cli/permission.go`: `keysIn` reads with `Read(ctx, p)`, `inline.Reader`'s
    method, in place of `editor.Reader`'s `ReadContext`.
  - `internal/cli/view_wrap_test.go`.

### P17 record (2026-10-30): `lineedit`, held at step 5

The base is `6c5eb02`. Nothing is committed (deviation 11).

1. **The host** (`internal/cli/editor/editor.go`).
   - Each prompt is a `lineedit.Composer` drawn in the editor's one `inline.Region`. The region's reflow is
     `ReflowCut` under `XTERM_VERSION`, else `ReflowDown`.
   - The keymap is built once, `keymap.New(keymap.Settings{}, lineedit.DefaultRules())`, and passed with
     `WithKeymap`.
   - The Composer's other options are step 3's: `WithCompleter`, `WithPaste(PasteInsert)`, `WithLimit(4096)`,
     `WithClearScreen`, `WithHeader`, and `WithExternalEditor` when `Options.External` is set.
   - `termkey.Decoder` decodes the input; a held escape is flushed after `DefaultEscTimeout`.
   - Inputs go to the Composer one at a time, so keys typed after a submission wait for the next prompt.
   - The Composer's writes are written once for each read.
   - gobble's parts stay the host's:
     - the Ctrl+C window, driven from `EditInterrupt`;
     - `/exit` and `/quit`, and skipping an empty submission;
     - the history, added to only for submissions `ReadPrompt` returns;
     - the paste notice on errw, for a paste of more than one line;
     - the 120-column cap;
     - the reprint once resizes settle, written through the region's `Print` as one synchronized update.
   - After a submission, the empty prompt the Composer draws below it, and the region's clearing of it, are both
     dropped. That leaves the cursor at the start of the row below the line, as before.
2. **What went:** `buffer.go`, `draw.go`, `layout.go`, `keys.go`, `paste.go` and `reader.go`, with `buffer_test.go`,
   `draw_test.go`, `layout_test.go` and `reader_test.go`.
   - `buffer_test.go`'s cases moved to the new `parity_test.go`.
   - `TestLineWidth` moved to `editor_test.go`.
   - The draw tests' and the reader's cases are go-tui-lib's own tests.
3. **`history.go`:** `FileHistory` keeps a `lineedit.MemHistory`, read with `ReadFrom` and saved with `WriteTo`, whose
   format is gobble's. The 1000 entries, the `0600` mode and the atomic rename stay.
4. **The callers.**
   - `auth.go`, `login.go` and `line.go` make an `inline.NewReader`; `prompt.go` and `permission.go` read through it.
   - `line.go` sets `Options.External` to `slash.go`'s new `editLine`, which runs `Compose` out of raw mode for ctrl+x
     ctrl+e.
5. **The PQ7 changes, each named in its test.**
   - `TestReadPrompt`: a paste is one buffer, its CR made LF, and a Ctrl+C inside it is cleaned out: `"ab"`, where
     gobble's kept `"a\x03b"`.
   - `TestPasteNoticeAndPrompts`: one buffer, the notice kept, once for each paste.
   - `TestHeader`: a trailing backslash's line break is in the buffer, so the first row is `> one`, where gobble's was
     `> one\`.
   - `TestEditingMidLineOnScreen`: alt+→ goes to a word's end.
   - `TestWordKeys`: words are letters and digits, and stop at punctuation.
   - In `TestKeysMatchXterm`, the cases PQ7 changes are skipped, each naming the test that holds it: two alt+→ cases,
     Ctrl+C and the five paste cases.
   - Two other changes:
     - `TestLineSessionRedrawsOnResize` narrows the terminal, since the region redraws only at a new width;
     - `TestLineSessionWrapsReplies` measures rows with `ansi.StringWidth`, since the end of the line's synchronized
       update now comes before the reply.
   - The screen tests use `inline/inlinetest.Screen`.
6. **Seen failing (C9).** On a scratch copy, each break made its test fail:
   - the host's Ctrl+C window skipped: `TestCtrlCTwiceInsideWindowExits` fails, "deadlock: all goroutines in bubble are
     blocked";
   - the history saved `0644`: `TestHistoryCapAndFiles` fails, `mode = -rw-r--r--, want 0600`;
   - the external editor's text dropped: `TestExternalEditor` fails;
   - a notice written over the prompt: `TestNoticesAboveThePrompt` fails;
   - typed-ahead keys dropped: `TestReadPrompt/keys_typed_ahead_go_to_the_next_prompt` fails, `["one"]`;
   - the empty prompt below a submission written: `TestLineSessionWrapsReplies` fails;
   - `earliest` reversed: `TestEarliest` fails;
   - a paste's CR kept: `TestReadPrompt/pasted_lines_are_one_submission` fails, `"line1line2line3"`;
   - each key's drawing doubled: `TestKeyDrawBudget` fails, `82 bytes for 1 bytes changed, more than 41`.
7. **G-local:**
   - `make pre-add-check` on the 19 Go files: clean;
   - `make preflight`: passed, after revive's confusing-naming on `readPrompt` and `add`, renamed `nextSubmission` and
     `record`;
   - `make race`: 36 ok;
   - `make perf`: ReplyRender 33.5 µs (baseline 30), Reprint 9.73 ms (9.4), LineSessionStart 978 µs (830), and KeyDraw
     14.9 µs (1.5), the breach.
8. **G-hosts:**
   - on both hosts: `go test`, race (36 ok) and records pass. Preflight stops at `verify-build-metadata`, as on every
     copy;
   - `make perf` breaches KeyDraw on both, as deviation 11's table has it;
   - `make probe-conhost` passes on the Windows laptop.
9. **Not done:** step 8's commit.

### Deviation 12 (2026-10-31): new tags, and P12 and P17 resumed

- **Found.**
  - The owner, 2026-10-30: "there is a new release of acp-go-sdk2 out, we need to pin the latest. also, a new release
    of go-tui-lib is going to be out in about 20 minutes."
  - The module proxy lists acp-go-sdk2 `v1.27.0` and `v1.28.0` after `v1.26.0`, and go-tui-lib/v2 `v2.7.0` after
    `v2.6.2`. MADR D1 makes a newer tag a deviation for the owner, and the Rollout section makes it a new amendment.
  - **acp-go-sdk2 `v1.28.0`** answers [acp-go-sdk2#12](https://github.com/maccavelli/acp-go-sdk2/issues/12), closed
    2026-10-30, which held P12 (deviation 9).
    - `go doc agentproc.Process.Close`: "Like Wait, Close waits without bound for a client handler that ignores its
      context; CloseContext bounds it."
    - `CloseContext(ctx)` runs Close's steps. Should ctx end during them, the steps left run as `Kill`'s do.
    - Should ctx end before `Wait` would return, it returns `context.Cause(ctx)`, joined with
      `acp.ErrHandlersRunning` when the connection's handlers still run, and with the agent's error. It returns at most
      a grace after ctx ends.
    - `v1.27.0` adds the package `remote`, whose `github.com/coder/websocket` `v1.8.15` is in the SDK's `go.mod`.
      gobble imports neither. On a scratch copy, `go mod tidy` adds neither to gobble's `go.mod` or `go.sum`.
  - **go-tui-lib/v2 `v2.7.0`** carries go-tui-lib's 0023-PLAN Step 7c, its answer to
    [go-tui-lib#13](https://github.com/maccavelli/go-tui-lib/issues/13).
    - The issue is still open. go-tui-lib's record leaves it for its owner to close.
    - That library's perf gate gives the key 1.43 µs on macOS, 4.99 µs on Linux and 2.30 µs on Windows.
    - Its A12 item 2 also measures gobble's own `BenchmarkKeyDraw` at `6c5eb02` as 1.55 µs and 8 allocations. That is
      against deviation 11's 4.24 µs and 17, a figure from another harness. It is not measured again here. The
      budgets are 0030-MADR D5's either way.
  - **P17's key at `v2.7.0`.** `make perf` ran on a scratch export of `feature/p17-lineedit` (`8630b77`) with
    `go get github.com/maccavelli/go-tui-lib/v2@v2.7.0`. The tree was not touched. `internal/cli/...`'s tests passed
    on the export first.

    | Host | Budget | `v2.6.2` (deviation 11) | `v2.7.0` |
    | --- | --- | --- | --- |
    | macOS | 4.5 µs | 14.9 µs, 53 allocations | 1.479 µs, 5 allocations |
    | the Linux machine | 15 µs | 42.0 µs, 53 allocations | 3.498 µs, 5 allocations |
    | the Windows laptop | 4.2 µs | 11.8 µs, 53 allocations | 1.208 µs, 5 allocations |

    On the same runs, `internal/cli`'s `ReplyRender`, `Reprint` and `LineSessionStart` passed their budgets on every
    host.
  - The owner, 2026-10-31, after this entry's first version was committed (`505e030`), and before it was pushed or
    any of its phases ran: "go-tui-lib, acp-go-sdk2, and go-llmprovider-sdk all have new tagged releases. ensure we
    are pinning them to current release versions. also ensure we pin current go-selfupdate-lib where relevent." This
    entry was extended for it, adding P22 and moving the pin phases first.
    - **go-selfupdate-lib `v1.23.1`** (`c24f5919cd9e9e3ba3736e64bcf18a6e2a51489e`, 2026-10-30) is newer than P3's
      `v1.23.0`.
      - Its changelog: it requires `golang.org/x/sys` `v0.49.0`, `golang.org/x/term` `v0.47.0` and `golang.org/x/mod`
        `v0.42.0`, and nothing exported or visible changes. gobble's `go.mod` already has those three versions.
      - `diff -r` of the two module zips: the source is unchanged, and so is `.github/`. Only the docs, `go.mod` and
        `go.sum` differ.
      - gobble names the tag in `go.mod`, in two `ci.yml` workflow pins (`:166`, `:178`), in `README.md:136` and in
        `docs/architecture.md:159`.
    - **go-llmprovider-sdk** has no tag newer than the pinned `v1.24.1`: `git ls-remote --tags`, the GitHub releases
      (`v1.24.1`, "Latest", 2026-10-27) and `check-libs` agree.
    - acp-go-sdk2 `v1.28.0` and go-tui-lib/v2 `v2.7.0` are still the newest.
  - **`check-libs`.** `python3 scripts/check_libs.py --lib acp-go-sdk2 --tag v1.28.0` passed all 13 checks;
    `--lib go-selfupdate-lib --tag v1.23.1` all 13; `--lib go-tui-lib --tag v2.7.0` all 14. `staticcheck`, `lint` and
    `api` are among them, and `archtest` is among go-tui-lib's. `--lib go-llmprovider-sdk` checked `v1.24.1`, the pin,
    with 0 failing.
  - `feature/p17-lineedit` has never been pushed: `git ls-remote --heads origin feature/p17-lineedit` is empty.
    `main` is three commits past its base, `61f7115`.
- **The owner's answers, 2026-10-31:**
  - P12's bound: "Grace ½, ctx 1× (Recommended)". The options not taken were:
    - the grace at `closeTimeout` and ctx at twice it, which reports a hung handler after twice today's bound;
    - both at `closeTimeout`, where step 2 and the kill race at the same instant.
  - P17's return to `main`: "Rebase branch, you merge". The options not taken were:
    - cherry-picking `8630b77` onto `main`;
    - finishing on the branch, with a merge commit.
  - The order: "0035 first (Recommended)". The options not taken were:
    - the pins now, and the rest after 0037's first plan;
    - 0037's first plan first.
- **Decision.**
  - MADR D1, D12 and D14 step 4 have dated notes.
  - **The order.** The pin phases go first, as deviation 1 put them, so that every pin is current before P12 and P17:
    1. these records;
    2. P20, acp-go-sdk2 `v1.28.0`;
    3. P21, go-tui-lib/v2 `v2.7.0`;
    4. P22, go-selfupdate-lib `v1.23.1`;
    5. P12;
    6. P17, on its branch;
    7. the owner's fast-forward of `main`;
    8. P18;
    9. P19.

    Then [0037-PLAN-permission-and-file-tool-hardening.md](0037-PLAN-permission-and-file-tool-hardening.md) resumes at
    its P3.
  - **P12, steps 1, 3, 4 and 6 as amended:**
    1. `Start` builds `agentproc.InProcess{Serve: <the adapter>, Grace: closeTimeout / 2, ConnectionOptions:
       connOpts}`.
    3. `Close`, through `closeFrom(base)`:
       - `ctx, cancel := context.WithTimeout(context.WithoutCancel(base), closeTimeout)`, then
         `err := c.proc.CloseContext(ctx)`;
       - it returns nil when `err` is nil, and otherwise `agent: %w` of it;
       - when `errors.Is(err, acp.ErrHandlersRunning)`, today's words, "the connection's handlers did not stop within
         <closeTimeout> of its close", are joined in front;
       - `agentproc.ErrAbandoned`, `agentproc.ErrKilled` and `acp.ErrHandlersRunning` stay readable with `errors.Is`;
       - P11's separate `WaitContext` goes, since `CloseContext`'s wait is the connection's.
    4. `closeTimeout` bounds `Close` as a whole, and the runner's grace is half of it, so `SetCloseTimeout` bounds
       both. `TestCloseWaitsForHandlers` and `TestCloseBoundedWhenAHandlerHangs` pass unchanged; a change to either is
       a deviation.
    6. Seen failing, each on a scratch copy:
       - with `Grace` unset, so 5 s, `TestCloseAbandonsAStuckAgent` fails its 2 s bound;
       - with `Close` as `c.proc.Close()`, `TestCloseBoundedWhenAHandlerHangs` does not return, under
         `go test -timeout 60s`.
  - **P17, resumed:**
    1. After P12's commit, `git rebase main feature/p17-lineedit`. `8630b77` touches `internal/cli` alone, and `main`
       has not touched it since `61f7115`. A conflict stops the phase as a deviation.
    2. Steps 5, 7 and 8 run on the rebased branch. The step 6 breaks and the P17 record's other results are not
       repeated, since the rebase changes none of P17's files. The tests and gates run again in full.
    3. A second commit on the branch has the records: the P17 record's completion, and the `docs/README.md` row.
    4. The owner fast-forwards `main` to the branch. That is a merge, so it is human-only. P18 starts after it.
  - P20, P21 and P22 are new phases, above.
- **Files added to scope:**
  - P20 and P21: `go.mod`, `go.sum`, `README.md` and `docs/architecture.md`;
  - P22: the same, and `.github/workflows/ci.yml`;
  - the records: [0037-PLAN-permission-and-file-tool-hardening.md](0037-PLAN-permission-and-file-tool-hardening.md),
    with a dated note that it pauses after P2.

