---
status: proposed
date: 2026-11-19
associated-madr: "0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement one shared terminal input for every prompt in a session

Associated MADR: [0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md](0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md)

*Proposed: nothing here is built until the owner approves this plan. It is written for the owner's answers of
2026-11-19 to the MADR's Q1 to Q8, and for the recommended answer, (b), to Q9, which is still open: if the owner
answers Q9 otherwise, the answer is recorded in the MADR and here, as a dated amendment, before P2b starts.*

## Goal

On a Windows console, the key prompt after the first-run offer, `/login`'s provider menu or a method menu waits for a
key. Every reader of the terminal in a gobble process reads through one `terminalInput`, which keeps what is read past
an answer from one question to the next and applies MADR D3's line-end rule. The first run offers the sign-in again
after a cancelled or failed sign-in, until one succeeds or the offer is declined, and never repeats without a person's
answer (MADR D8). Each new rule is seen to fail on a planted break; the gates pass; a pseudo-console test passes on
the Windows test laptop; the owner sees it work in Windows PowerShell 5.1 in Console Host and in PowerShell 7.x in
Windows Terminal. The plan stops there: the release is decided after that check (MADR Q8).

## Scope

* **In:** `internal/cli` (`terminput.go`, new; `prompt.go`, `login.go`, `line.go`, `auth.go`, `trust.go`,
  `permission.go`, `main.go`, and their tests); `internal/cli/editor/editor.go` and `doc.go` (the `Input` interface);
  `cmd/gobble/conpty_windows_test.go` and a `Makefile` target (Q7); `docs/guides/credentials.md`,
  `docs/architecture.md`, `README.md`, `docs/README.md`; this plan's execution record and the MADR's status notes.
* **Out:** go-llmprovider-sdk and go-tui-lib (no change, no pin moves); `acpext`; any new module; the line editor's and
  `askOn`'s own decoding and read-ahead (Q2); `gobble acp --login`'s key-only path (`x/term` `ReadPassword` and the
  `bufio` workspace read); weakness (c) (Q6, MADR D9); a loop in `/login`, `gobble setup` or `gobble acp --login`
  (MADR D8); a release, a tag or any `v0.6.1` work (Q8); `AGENTS.md`, which does not describe the prompt layer (read
  2026-11-19); a changelog, which this repository does not keep.

## Implementation Steps

Work on a topic branch from `main`, named by the owner. Each step's command output goes to a scratch file, its exit
status taken before any filter. Heavy runs go through `run-heavy` on the session host or `run-on-host` on a fleet host,
one at a time per host; no race result from a WSL distro is taken
([0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md](0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md)).

Each test below is one of three kinds: **red for the defect** (fails at the base, passes after P2), **red for the
loop** (fails at the base and after P2, passes after P2b), or a **guard** (passes at the base and after every phase).

### P1 — tests first, red at the base

*Class: strongest; effort: high. The tests decide what "fixed" means; the line ends, the asks, the loop's ways out and
the paths are easy to cover in a way that passes on the broken code.*

**Files:** `internal/cli/lineend_test.go` (new), `internal/cli/firstrun_test.go` (new), `internal/cli/login_test.go`,
`internal/cli/line_test.go`, and one behaviour-neutral seam in `internal/cli/login.go`.

1. **A raw-mode seam,** so a sign-in outside the line session can be tested on a pipe: a package variable
   `termRaw = func(f *os.File) (func() error, error) { return editor.Raw(f) }`, used by `terminalQuestions`
   (`login.go:95-97`) and `newLineSession` (`line.go:86`), as `trustRaw` is (`trust.go:211`). No behaviour changes.
2. **The harness's line ends.** `lineHarness` gains `enter string` (default `"\r"`); the first-run answer becomes
   `"n" + h.enter` unless a test sets `h.answer`. A helper `eachEnter(t, func(t *testing.T, enter string))` runs a
   subtest per form, named `CR`, `LF` and `CRLF`, for `"\r"`, `"\n"` and `"\r\n"`. `TestFirstRunSetup`,
   `TestLoginMenuAndRefusals` and `TestLoginAsksThroughQuestions` answer their questions (not the editor's prompt,
   which is raw, where Enter is `\r`) with each form. A test helper `recordingPrompter` wraps a `provider.Prompter`
   and records what each `Secret` returned, so a test can run the real `provider.SignIn` and see the key it was
   given. A helper counts `firstRunQuestion` and `Warning: no model credential` in stderr. `lineHarness` gains
   `noAnswer bool`, which leaves the first offer unanswered for a test that answers it itself.
3. **The question reader, by bytes** (`lineend_test.go`, over `asked`, so they build at the base):
   - `TestLineEndsBeforeEachQuestion`: for each form E, each first question K1 (`choose` of two, `confirm`, `input`)
     answered with its text and E in one chunk, and each next question K2 (`secret`, `choose`, `confirm`, `input`)
     answered in a later chunk (a secret with `the-key\r`, as raw Enter gives it), K2 gets its own answer. **Red for
     the defect:** E = `\r\n` and K2 = `secret`, three cases, each `secret = "", <nil>; want "the-key"`; the rest are
     guards.
   - `TestEnterAfterSecretAnswersTheNextLine` (weakness (d)): a secret ended by `\r`, then an `input` answered by `\n`
     alone in a later chunk, returns `""` at once. **Red for the defect:** at the base the `\n` is dropped and the read
     meets the end of input (`errCancelled`).
   - Guards: `TestCRLFSplitAcrossReads` (an `input` answered `a\r`, then `\nb\n` in the next chunk, gives `a` then
     `b`); `TestPastedKeyEndingCRLF` (a secret given `the-key\r\n` in one chunk, then an `input` given `x\n`, gives
     `the-key` then `x`); `TestLFThenCRIsTwoEnds` (Q1: a `confirm` answered `y\n`, then a secret given `\r`, is an
     empty secret, which `prompter.Secret` cancels).
4. **The sign-in paths, each over `eachEnter`** (`login_test.go`):
   - `TestFirstRunKeyOnlyProviderWaits`: the real `provider.SignIn` behind `recordingPrompter`; `y`+E; at
     `Number [1]:`, Gemini's number (from `provider.KeyProviders()`) + E; at `Enter your Gemini (Google) API key: `,
     `sk-test-typed\r`. The recorded secret is `sk-test-typed` with no error. **Red for the defect, `CRLF`:** the
     secret is recorded as `errCancelled` before the key is pressed.
   - `TestFirstRunMethodMenuProviderWaits`: as above for OpenAI; at the method menu, the number of the item whose label
     names an API key, read from the menu printed, + E. The key prompt is `Enter your OpenAI API key: `. **Red for the
     defect, `CRLF`.**
   - `TestLoginNoArgumentMenuWaits`: the offer declined; `/login\r` at the editor; Gemini's number + E; the key is
     recorded. **Red for the defect, `CRLF`.** `/login gemini\r` then the key: guard.
   - `TestSetupKeyWaits`: `gobble setup` with `e.stdin` an `os.Pipe` read end, `caps.In` true and `termRaw` stubbed;
     Gemini's number + E; the key recorded. **Red for the defect, `CRLF`.**
   - `TestAcpLoginMethodMenuWaits`: `gobble acp --login openai` on the same pipe; the API key item + E; the key
     recorded. **Red for the defect, `CRLF`.**
   - `TestFirstRunTypeAhead` (weakness (a)): `y`+E+Gemini's number+E in one press; the key prompt appears without a
     second press. **Red for the defect, every form:** the menu waits, the typed-ahead answer dropped with the first
     ask.
   - Guards: `TestSignInTypeAheadNotSentAsPrompt` (Q3: with `typedKeySignIn` and `keyLister`, `/login claude\r`, then
     the key pressed as `sk-test-GOOD\rhello\r` in one press: the key is saved and the agent receives no prompt
     `hello`); `TestEmptyKeyCancels` (Q4: `/login gemini\r`, then `\r` at the key prompt: the cancel line).
5. **The first-run loop** (`firstrun_test.go`, over `eachEnter` for the questions' answers; Gemini's sign-in through
   `typedKeySignIn` unless a test says otherwise; every test ends the session with Ctrl+D and `h.wait`, whose 10 s
   bound makes a loop that never ends fail rather than hang; the package run adds `-timeout 2m`):
   - **Red for the loop:**
     - `TestFirstRunOfferAgainAfterKeyCancel`: `y`, Gemini, then Ctrl+C (`\x03`) at the key prompt: the cancel line,
       then `firstRunQuestion` a second time, ending `[y/N]: ` (Q9 (b)); `n`+E: the warning once, and the question
       counted twice in all.
     - `TestFirstRunOfferAgainAfterEmptyKey`: as above with `\r` at the key prompt (Q4).
     - `TestFirstRunOfferAgainAfterMenuInterrupt`: at the provider menu, the interrupt as `TestLoginInterrupt` sends it
       (`h.e.intr.take()`): the cancel line, then the offer again.
     - `TestFirstRunOfferAgainAfterFailure`: `signInWith` returns `&provider.SignInError{ID: "claude", Keys: 3}`, and
       in a second case an error of kind offline: `Error: claude refused 3 keys; none was saved` (or the offline
       error's line), then the offer again; no warning between rounds.
     - `TestFirstRunRepeatedOfferDefaultsNo` (Q9 (b)): after one cancelled round, E alone at the repeated offer ends
       the loop with the warning, and no menu is shown again.
     - `TestFirstRunDeclineOnRepeatEnds`: after one cancelled round, `n`+E: the warning once, the session's prompt,
       and a turn typed there runs.
     - `TestFirstRunTypeAheadDroppedBeforeRepeatedOffer`: at the key prompt, `\x03y\r` in one press: the repeated offer
       is shown and waits; `Sign in to which provider?` is not shown a second time before `n`+E is pressed, which ends
       the loop.
   - **Guards** (the loop's ways out, which hold at the base and must hold after it):
     - `TestFirstRunDeclineEnds`: `n`+E at the first offer: the warning once, the question once.
     - `TestFirstRunInterruptAtOfferEnds`: the interrupt at the first offer: the warning once, no menu.
     - `TestFirstRunClosedInputEnds`: `y`, Gemini, `\r` at the key prompt, then the harness's pipe closed: the session
       ends within the bound, the warning printed once at most.
     - `TestFirstRunInstantFailureClosedInputEnds`: `signInWith` fails at once without asking; `y`+E, then the pipe
       closed: the session ends within the bound.
     - `TestLoginDoesNotLoop`: the offer declined; `/login gemini\r`, `\r` at the key prompt: the cancel line, and
       `firstRunQuestion` still counted once.
     - `TestSetupDoesNotLoop`: `gobble setup` on the pipe, Ctrl+C at the key prompt: `setup cancelled; nothing was
       saved`, and no `firstRunQuestion`.
6. **Run them at the base,** on the session host (one package):
   `go test -count=1 -timeout 2m -run '<the new tests>' ./internal/cli/ > <scratch>/p1-red.out 2>&1; echo $?`. Record,
   from the whole file, each red test with its failure line, and each guard passing.

**Gate:** every test marked red fails, for the reason given, and nothing else does; every guard and every existing
test passes; `go vet ./internal/cli/` exit 0.

**Stop if:** a red test fails for another reason, or passes; a guard fails at the base (a finding: record it and put it
to the owner); the seam changes any existing test's result.

### P1b — a pseudo-console test on Windows (Q7)

*Class: strongest; effort: high. Windows process and console APIs, run remotely.*

**Files:** `cmd/gobble/conpty_windows_test.go` (new, `//go:build windows && conpty`), `Makefile` (`probe-conpty`,
beside `probe-conhost`, not in `preflight` or CI).

1. The test creates a pseudo console (`windows.CreatePseudoConsole`, `golang.org/x/sys` `v0.49.0`, already required)
   and starts the test binary again inside it, through `windows.NewProcThreadAttributeList` and
   `PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE`. Its `TestMain` runs one of two child modes, chosen by an environment
   variable:
   - **probe:** one read of stdin in the console's default line mode, its bytes written with `%q` to a file the parent
     names. The parent types `1` and Enter (`1\r` into the pseudo console's input). This measures MADR D6's third
     assumption: expected `"1\r\n"`.
   - **gobble:** `main()`, with `GOBBLE_HOME` a temp directory, no credential variable, and `HTTPS_PROXY` and
     `HTTP_PROXY` set to a closed local port so nothing leaves the host. The parent reads the console's output, answers
     `y`, Enter, `1`, Enter, and waits for `Enter your Gemini (Google) API key: `; it then requires that
     `Sign-in cancelled` does not appear within two seconds, types `not-a-real-key`, sees fourteen `*`, sends Ctrl+C,
     sees the cancel line and the offer again (the loop), answers `n`, Enter, and sees the warning.
2. Run at the base plus P1's tests and P1b, on the Windows test laptop, through the dispatcher's
   `run-on-host --need windows -- make probe-conpty`. Expected: the probe gives `"1\r\n"`; the gobble mode fails with
   the cancel line before a key.

**Gate:** both results as expected, recorded verbatim.

**Stop if:** a pseudo console cannot be created, or the child cannot be started in it, in the dispatcher's ssh session
on the Windows test laptop: record the error and stop, and the coordinator puts Q7 to the owner again (the MADR's
(b) and (c) remain). Stop too if the probe's bytes are not `"1\r\n"`: the MADR's premise is wrong for a pseudo
console, and the MADR is amended before P2.

### P2 — the shared terminal input

*Class: strongest; effort: high. It changes the read path every prompt and every editor key takes.*

**Files:** `internal/cli/terminput.go` (new), `internal/cli/terminput_test.go` (new), `prompt.go`, `login.go`,
`line.go`, `auth.go`, `trust.go`, `permission.go`, `main.go`; `internal/cli/editor/editor.go`, `editor/doc.go`; the
test constructions in `login_test.go` (`asked`) and `line_test.go` (`start`).

1. **`terminalInput`** as MADR D1 and D3 give it: `newTerminalInput(in keysIn)`; `Read(ctx, p)`; `line(ctx)`;
   `key(ctx)`; `rawEnd(c byte)`; `discard()`; `ended()`. One unexported method applies the rule to the next byte
   given, for every entry point. The mutex guards `pending`, `lfOwed` and `closed`, and is released before any read of
   `in`. A read of `in` is made only when the caller needs a byte and `pending` is empty after the rule, with the
   caller's buffer (`line` and `key` use 512 bytes, as `fill` does); when the rule drops the only byte a read gave, the
   same caller reads again. An error from `in` that is not the context's sets `closed`.
2. **`terminput_test.go`:** the rule by table (each form, raw and cooked ends, the owed `\n` in the same and in a later
   read, `\n\r`, a byte that clears `lfOwed`); `discard` drops `pending` and keeps `lfOwed`; no read of `in` while
   `pending` holds a byte, and one read per need, with a stub that counts reads; the end of input and an ended context
   returned as `inline.Reader` returns them; `ended()` true after the end of input and false after an ended context.
3. **`runEnv`** (`main.go`): `termIn *terminalInput`, and `terminalIn()`, which makes it once over
   `inline.NewReader(e.stdin)` under a lock.
4. **`questions`** (`prompt.go`): `in` becomes `*terminalInput`; `pending`, `afterCR` and `fill` go; `lineContext`
   reads `q.in.line(ctx)` then `plainLine`; `readSecret` reads `q.in.key(q.ctx)` and calls `q.in.rawEnd(c)` on Enter.
   The doc comments name 0063-MADR.
5. **The callers:** `ls.ask` makes `questions` over `ls.keys` (`login.go:226`); `terminalQuestions` over
   `e.terminalIn()` with `termRaw`; `auth.go:301` over `e.terminalIn()`; `trust.go:184` reads `e.terminalIn().Read`
   (Q2); `askPermission` takes `ls.keys` as now, its type changed (`permission.go:249`).
6. **The line session** (`line.go`): the `in` field goes; `ls.keys = e.terminalIn()`; `editor.New(ls.keys, …)`;
   `ls.keys.discard()` before each `ed.ReadPrompt` (Q3). The harness sets `h.e.termIn` to a `terminalInput` over
   `inline.NewReader(&rawCheckedInput{…})` in `start`, which is how its pipe reaches the session.
7. **The editor:** `editor.Input`, an interface with `Read(context.Context, []byte) (int, error)`; `New(in Input, …)`
   and `Editor.in Input`; `doc.go`'s sentence on the one reader says the line session's reader may be a layer over it.
8. **A test for the trust question's reader:** `TestTrustReadsTerminalInput`, `e.termIn` a stub that records a read,
   `trustRaw` stubbed; the trust question's key is read through it.

**Gate:** every test red for the defect now passes (P1b's on the Windows test laptop); the tests red for the loop still
fail, for the reasons P1 recorded; every guard passes; `go test -count=1 ./internal/cli/editor/` exit 0, and
`./internal/cli/` with only the loop's red tests failing, `TestLineSessionReadsOnlyWhenAsked` passing; `make archtest`
exit 0; `make pre-add-check FILES="<every changed .go file>"` exit 0.

**Stop if:** a P1 test must be changed to pass, beyond how the harness and `asked` build their input (steps 2 and 6);
any existing test fails; `TestLineSessionReadsOnlyWhenAsked` counts a different number of cooked reads; `archtest`
reports a new edge.

### P2b — the first-run loop (MADR D8)

*Class: strongest; effort: medium. A small change, but every way out of a loop must be right, and a wrong one spins.*

Its own phase, after P2, because it depends on `discard()` and `ended()`, its tests are a set of their own, and it
reviews and reverts apart from the line-end fix.

**Files:** `internal/cli/login.go`; its tests are P1's `firstrun_test.go`.

1. **One sign-in round:** `ls.login`'s body becomes `ls.signInRound(want string) bool`, which runs the sign-in, prints
   its line as `ls.login` does now (the cancel line, `Error: …`, or the after-steps on success) and reports whether it
   signed in. `ls.login` calls it once; `/login` does not loop.
2. **The loop in `firstRun`** (`login.go:300-316`), in this order each round:
   - from the second round on, `ls.keys.discard()`;
   - the offer: `confirm(firstRunQuestion, def)`, with `def` true in the first round and false after (Q9 (b));
   - an error other than `errCancelled` is returned, as today;
   - `ls.keys.ended()`, or not yes (`n`, Enter on a `[y/N]` offer, Ctrl+C at the offer): the lead and the warning
     once, and return;
   - `ls.signInRound("")`: signed in returns; if `e.ctx` ended, return its cause; if `ls.keys.ended()`, the lead and
     the warning once, and return; otherwise the next round.
3. Nothing else changes: `gobble setup` and `gobble acp --login` call `e.signIn` directly.

**Gate:** every P1 test passes, the loop's included; `go test -count=1 -timeout 2m ./internal/cli/` exit 0;
`make pre-add-check FILES="internal/cli/login.go"` exit 0.

**Stop if:** a loop test needs changing to pass; any test needs the harness's bound raised; the warning appears more
than once in any test.

### P3 — planted breaks

*Class: mid-tier; effort: medium. Mechanical, but each failure must be read, not counted.*

**Files:** none in the tree. Each mutation on a fresh scratch copy of the P2b tree (copied, not checked out), one at a
time, under the session's scratch directory, built from nothing with `GOPROXY=off`.

| # | Mutation | Must fail |
| --- | --- | --- |
| 1 | the owed `\n` not dropped | `TestLineEndsBeforeEachQuestion/CRLF/*-secret`, `TestFirstRunKeyOnlyProviderWaits/CRLF` |
| 2 | `rawEnd` sets `lfOwed` | `TestEnterAfterSecretAnswersTheNextLine` |
| 3 | `\n` then `\r` folded as one end | `TestLFThenCRIsTwoEnds` |
| 4 | `ls.ask` makes a new `terminalInput` per ask | `TestFirstRunTypeAhead` |
| 5 | `discard` does nothing at the session prompt | `TestSignInTypeAheadNotSentAsPrompt` |
| 6 | `terminalInput` reads ahead after a line | `terminput_test`'s read count, `TestLineSessionReadsOnlyWhenAsked` |
| 7 | `rawEnd` leaves a `\n` already read | `TestPastedKeyEndingCRLF` |
| 8 | `askTrust` reads `e.stdin` again | `TestTrustReadsTerminalInput` |
| 9 | the loop never ends on a decline (a no at the repeated offer starts another round) | `TestFirstRunDeclineOnRepeatEnds`, `TestFirstRunRepeatedOfferDefaultsNo` |
| 10 | the loop never ends on closed input (`ended()` ignored, the end of input taken as a cancelled round) | `TestFirstRunClosedInputEnds`, `TestFirstRunInstantFailureClosedInputEnds`, by the harness's bound |
| 11 | no repeat after a cancel (the loop ends on a cancelled round) | `TestFirstRunOfferAgainAfterKeyCancel`, `…AfterEmptyKey`, `…AfterMenuInterrupt` |
| 12 | no `discard` before a repeated offer | `TestFirstRunTypeAheadDroppedBeforeRepeatedOffer` |
| 13 | the repeated offer defaults to yes | `TestFirstRunRepeatedOfferDefaultsNo` |
| 14 | the warning printed between rounds | `TestFirstRunOfferAgainAfterKeyCancel` (warning counted once) |
| 15 | mutation 1, with `eachEnter` cut to `CR` alone | expected to pass: recorded, to show the `CRLF` form is what catches it |

Each run: `go test -count=1 -timeout 2m ./internal/cli/` (and `./internal/cli/editor/` for 6), output to a scratch
file, the exit status before any filter, the failing test's message quoted. Mutation 10 is expected to spin until the
harness's bound; its CPU use ends with the test binary. Then the unmutated copy passes.

**Gate:** mutations 1 to 14 each fail the named test, with its message recorded; 15 passes; the restored copy passes.

**Stop if:** a mutation from 1 to 14 passes (that test checks nothing: back to P1 for its rule), or the tree is
changed.

### P4 — gates

*Class: mid-tier; effort: low. Run, read, record.*

**Files:** none; a failure goes back to its phase.

1. `make pre-add-check FILES="<every changed .go file>"` on the session host.
2. `make lint` (golangci-lint `v2.15.0`, both modules; the binary's `version` line recorded).
3. `make preflight`, through `run-heavy -- make preflight` on the session host or `run-on-host -- make preflight`.
4. `go test -count=1 ./...` in both modules on macOS, the Linux machine and the Windows test laptop, each through
   `run-on-host --need darwin|linux|windows`.
5. The race detector: `run-on-host --race --need linux -- make race` and `run-heavy -- make race` on macOS; never a WSL
   distro.
6. `run-on-host --need windows -- go test -count=1 ./internal/cli/...`, recorded as a run **without a console**: it
   checks the code on Windows, not a console's line ends; and `run-on-host --need windows -- make probe-conpty` on the
   P2b tree: the probe `"1\r\n"`, the gobble mode passing.
7. `run-heavy -- make check-test-isolation` (0053-MADR D4): the new tests read no host file above their temp root.
8. `make check-records`, `make markdownlint`, and the disclosure guard's `redact` over the changed files.

**Gate:** every command exit 0, with its summary line recorded.

**Stop if:** any gate fails. Record the file, line and failure, whether it fails on the base too, and stop; no gate is
loosened or skipped.

### P5 — docs

*Class: mid-tier; effort: medium. Prose to the repository's register, examples from gobble's own output.*

**Files:** `docs/guides/credentials.md`, `docs/architecture.md`, `README.md`, `docs/README.md`, this plan, the MADR's
status note.

1. `docs/guides/credentials.md`, "Signing in from a session":
   - the key prompt's words are the SDK's, `Enter your Claude (Anthropic) API key:` (go-llmprovider-sdk
     `wizard/configure.go:794`), where the guide shows `Claude (Anthropic) API key:`; the example is taken again from
     gobble's own output in a scratch home with no credential and no network, as 0025-PLAN deviation 8 did;
   - an answer may end with any Enter a terminal sends, `\r`, `\n` or `\r\n`; keys typed ahead answer the next question
     of the same sign-in and are dropped when the session's prompt returns; an empty key cancels;
   - the first run, where the guide now says "Yes runs `/login`; no goes on as before": a sign-in that is cancelled or
     fails, an empty key included, brings the offer back, `[y/N]` (Q9), until a sign-in succeeds or the offer is
     declined with `n`, Enter or Ctrl+C; then the warning naming the variables; `/login` and `gobble setup` ask once.
2. `docs/architecture.md`: the `internal/cli` row says one terminal input per process, which every prompt reads
   through, and that the first run offers the sign-in until it succeeds or is declined; the `internal/cli/editor` row
   says the editor reads through an `Input`.
3. `README.md`'s "Signing in where you are": the first run offers the sign-in again after one that is cancelled or
   fails, until one succeeds or the offer is declined.
4. `docs/README.md`: the 0063 rows' status.

**Gate:** `make check-records`, `make markdownlint` and the disclosure guard's `redact`, each exit 0; the guide's
example is pasted from the scratch run, not typed.

**Stop if:** the scratch run's output differs from what the MADR says a user sees.

### P6 — the owner's check on a Windows console, and the stop

*Class: the owner, at the Windows laptop; the build by a mid-tier agent, effort low; the record by the coordinator.*

1. **The build:** from P5's commit, `CGO_ENABLED=0 GOOS=windows GOARCH=<the laptop's: amd64 or arm64, from
   $env:PROCESSOR_ARCHITECTURE> go build -o gobble.exe ./cmd/gobble`, put where the MADR's check step 2 expects it
   (`%USERPROFILE%\gobble-check\gobble.exe`). It is a check build, not a release.
2. **The check:** the MADR's Confirmation item 5, steps 1 to 15, in Windows PowerShell 5.1 in Console Host and in
   PowerShell 7.x in Windows Terminal. Steps 5 to 7 show the offer coming back after Ctrl+C and after an empty key,
   and ending on `n`; step 10 shows `/login` not looping. Before it, if the owner wants the reproduction, steps 1 to 4
   with the installed `0.6.0`.
3. **The record:** per terminal, each step passed or the screen text that differed, never a key; the PowerShell
   versions and the Windows build (`winver`).

**Gate:** every step as the MADR says, in both terminals.

**Stop if:** any step differs. Record the screen text, no key, and stop; a fix is a deviation, recorded and approved
before it is made.

**Then the plan stops.** The release is decided by the owner after this check has passed (MADR Q8); no release, tag or
`v0.6.1` work is part of this plan.

## Verification

* P1's red tests seen failing at the base, quoted; the defect's passing after P2 and the loop's after P2b.
* P1b's probe and gobble mode on the Windows test laptop, at the base and after P2b.
* P3's fourteen mutations each failing its test, quoted, and the fifteenth passing.
* P4's gates, each exit 0, with their summaries.
* P6's owner check, in both terminals.

## Rollout and Rollback

* **Commits,** by the owner on the coordinator's word, after the pre-add check: P1's tests red for the defect, P1's
  guards, P1b and P2 as one commit (the loop's red tests are left out of it, since they fail until P2b); P1's loop
  tests and P2b as one commit; P5 as one; P6's record as one. Nothing is pushed or merged to `main` without the owner's
  word; no tag.
* **Rollback:** revert P2b's commit alone to return to a single offer; revert both code commits to return to today's
  reading (P5's docs with them). Nothing persists that the old code reads differently: no file format, setting or
  `auth.json` field changes. `probe-conpty` is outside `preflight` and CI.

## Completion criteria

* The MADR's Confirmation items 1 to 5 met.
* Q9 answered by the owner and recorded in the MADR, and this plan amended if the answer is not (b).
* `docs/guides/credentials.md`, `docs/architecture.md`, `README.md` and `docs/README.md` current.
* This plan's status `complete`; the release left to the owner's decision after P6 (Q8).

## Execution record

*Empty: nothing has been executed.*
