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
4. **`.golangci.yml`.** golangci-lint v2.14.0 does not honour `//lint:ignore` (probed 2026-10-29: all 12 still
   reported). After the `internal/config/genschema` rule, add:

   ```yaml
         # acp-go-sdk2's unstable-schema names (0035-MADR D6). make staticcheck bounds them:
         # scripts/check_staticcheck_ignored.py refuses any ignored finding outside them.
         - linters:
             - staticcheck
           text: "SA1019: .*github\\.com/maccavelli/acp-go-sdk2\\..* is deprecated: this declaration comes from the unstable ACP schema"
   ```

5. **`scripts/check_staticcheck_ignored.py`.**
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
