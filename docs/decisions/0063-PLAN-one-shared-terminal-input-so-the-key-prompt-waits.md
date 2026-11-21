---
status: in-progress
date: 2026-11-20
associated-madr: "0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement one shared terminal input for every prompt in a session

Associated MADR: [0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md](0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md)

*Proposed: nothing here is built until the owner approves this plan. It is written for the owner's answers of
2026-11-19 to the MADR's Q1 to Q8, and for the recommended answer, (b), to Q9, which is still open: if the owner
answers Q9 otherwise, the answer is recorded in the MADR and here, as a dated amendment, before P2b starts.*

*(2026-11-20: this banner is out of date. The owner answered Q9 with (b), the recommended answer this plan is written
for, and approved this plan, both on 2026-11-19; see the execution record. No step changes. Nothing is built yet.)*

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

*(2026-11-20, second records pass: In gains `internal/cli/conpty_windows_test.go` (`//go:build windows && conpty`)
and `internal/cli/pty_linux_test.go` (`//go:build linux && pty`), for the cases that need a stubbed sign-in on a
pseudo console or a pseudo-terminal, and a `Makefile` target `probe-pty` beside `probe-conpty`, outside `preflight`
and CI; and, in P2, the flush seam and `abandon()` in `terminput.go`, the deferred restore in `prompt.go`, and the
trust question's read in `trust.go` with no deadline (MADR D12 to D15). Out gains every reader in rows R91 to R96
(Q10). Who runs the live checks: MADR D16; an agent, through `run-on-host`, except the owner's short real-console
confirmation in P6.)*

*(2026-11-20, fourth records pass: In also gains the flush call of MADR D18, in `internal/cli/term`:
`flush_linux.go`, `flush_darwin.go`, `flush_windows.go`, `flush_other.go` and their test, `flush_test.go`. The
package already imports `golang.org/x/sys` (`console_windows.go:11`), `v0.49.0` is already required (`go.mod:24`),
and `internal/archtest` has no rule on `golang.org/x/sys` or against `internal/cli/term` importing it, so no record
and no rule changes; P2's `make archtest` gate confirms it.)*

*(2026-11-21, after P1 and P1b's build pass 1 and its review: In also gains two docs edits made with P1b, which adds
the make targets `probe-conpty` and `probe-pty`. `docs/guides/developing.md`, in no part of this Scope before, gains a
row for each target in its "Suites outside the default run" table, and the sentence under it reads "None of them";
`docs/architecture.md`, P5's file, has both targets named in its sentence on the checks that need a console. They
belong with P1b because the targets are P1b's, and the repository describes every make target where it lands. P5
edits `docs/architecture.md` as it then stands.)*

*(2026-11-21, P2's build pass 1, Deviation 2; the owner, relayed by the coordinator: "Accept all four; add both docs to P5 (Recommended)"): In also gains,
for P5, `internal/cli/term/doc.go` (its package summary names neither `FlushInput` nor `ConsoleInput`) and the
`internal/cli/term` row of `docs/architecture.md` (the same two). For P2: `internal/cli/sessions_test.go`, changed
only in how its input is built; and `internal/cli/pty_linux_test.go`, P1b's, for the one-line change to
`pty-abandoned`'s check (MADR D21).)*

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

**Step 0, added 2026-11-20 (records pass): re-ground at the phase's start commit.** Before step 1, at the commit P1
starts from, check every file and line the MADR cites (its evidence index, D1 to D10, Confirmation, the state table
and Q10 to Q14) and every one this plan cites (the re-grounding tables in the execution record), and record each as
holding or moved, with the commit. A moved line is corrected by a dated annotation in the MADR and here before step 1;
a changed behaviour, or a new reader of the terminal, is a stop, put to the owner. Also confirm that every `open` row
of the MADR's state table has the owner's answer recorded; P1 does not start with an open row.

**Step 0b, added 2026-11-20 (second records pass): the Windows measurements first, agent-run.** Before step 1, and
before any other test is written, the measurements that can stop the plan are made, since a stop costs least here.
They need no P1 test and no code change: they are P1b's probe mode (`cmd/gobble/conpty_windows_test.go`), written
first, with five cases, `probe-crlf`, `probe-typeahead`, `probe-ctrlc`, `probe-ctrlz-line`, `probe-ctrlz-raw` (the
MADR's Confirmation item 5 (A)). An agent runs them, from the repository root of the topic branch, as

```text
run-on-host --need windows --label 0063-probe -- make probe-conpty PROBE=1
```

(`run-on-host` is the fleet's host tooling, from the `available-dev-hosts` skill: it copies the working tree to the
Windows test laptop, runs the command there in Git's bash under the host's queue, and passes back the output and the
exit status.) The run prints one line per case, `0063 case=<name> result=<pass|fail> observed=<word>`, and nothing of
the screen. The agent reads only those lines:

* every one of the five lines is present (a missing line is a fail: for example a run that fell back to the session
  host, where the test does not build);
* `probe-crlf` observed `crlf`, `probe-typeahead` observed `one-line-per-read`, both `probe-ctrlz-*` observed `eof`:
  otherwise the MADR's D6 or D14 premise is wrong, a stop, and the MADR is amended before step 1;
* `probe-ctrlc` observed `waiting`: the build goes on. Observed `eof` or `error`: the named stop of row R40; the
  coordinator puts row R40's three options to the owner, and nothing further is built until the answer is recorded.
* *(2026-11-20, fourth pass.)* The probe child takes the interrupt itself (`signal.Notify` for `os.Interrupt`, as
  gobble does; P1b's probe mode), so each word means one thing. `waiting`: the child recorded `signal` and its read
  went on waiting until `x`, Enter (the build goes on). `eof`: it recorded `signal`, and the read returned nothing, the
  end of input (the R40 stop). `error`: it recorded `signal`, and the read returned another error (the R40 stop).
  `killed`: the child ended without writing its result, which means the probe did not take the interrupt; that is a
  fault in the probe, not R40's stop: the probe is fixed and run again.
* *(2026-11-20, after step 0b's build pass 1: the gate amended by the deviation entry "Deviation 1" below.)*
  `probe-ctrlc` measures the raw console, not gobble's reader, so it keeps observing `eof`: that is the console's
  behaviour, which MADR D19 now handles, and it no longer stops the build. Its expected word is `eof` from here on (a
  one-line change in `cmd/gobble/conpty_windows_test.go`, P1b's, so its result reads `pass`); until that change, the
  agent reads `probe-ctrlc` by its observed word: `eof` goes on; `waiting` also goes on (the console then never gives
  the empty read D19 excuses), recorded in the MADR; `error` is a stop, put to the owner; `killed` is a fault in the
  probe. Once Q22 is answered (a), the probe's word also says which came first, the signal or the read's return. The
  gobble-mode case `menu-ctrlc` (P1b) is the one that must show `offer-again` after P2.
  *(2026-11-21: after P2b; it needs the loop. See the correction under P1b's amendment.)*
* *(2026-11-20, Q22 answered, MADR D20.)* `probe-ctrlc`'s line gains one word after `observed=`: `first=signal` (the
  child recorded the interrupt before its read returned) or `first=read` (the read returned first; the 100 ms wait of
  D20 is what keeps D19 working). For example `0063 case=probe-ctrlc result=pass observed=eof first=signal`. Neither
  value stops the build; the agent records each run's value in the execution record. A line with no `first=` word
  after the probe file's change is a fault in the probe.

*(2026-11-20, third records pass: what waits for what.)* Step 0b does not wait for step 0's check of open rows, nor
for row R40: it is the measurement R40 awaits, and it changes no code. Step 1 waits for step 0 (no `open` row) and for
step 0b's result (R40's `waiting`, or the owner's answer to R40's stop). The one row left `open` after the third pass,
R101, is the words of a warning, not a behaviour: step 0 lets step 1 start with it open, and it must be approved before
P2 step 9 writes the message and A29 asserts it; neither is written while it is open. *(2026-11-20, fourth pass: R101's
words are approved; no row is `open`. Step 1 waits for step 0 and for step 0b's result only.)*

1. **A raw-mode seam,** so a sign-in outside the line session can be tested on a pipe: a package variable
   `termRaw = func(f *os.File) (func() error, error) { return editor.Raw(f) }`, used by `terminalQuestions`
   (`login.go:95-97`) and `newLineSession` (`line.go:86`), as `trustRaw` is (`trust.go:211`). No behaviour changes.
   *(2026-11-20, amended at P1's build pass 1; the owner, relayed by the coordinator: "`login.go` only; amend step 1
   (Recommended)".)* The seam is used by `terminalQuestions` alone. `line.go:86` keeps its direct call to
   `editor.Raw`, because the line-session tests set raw mode another way: the line harness gives the session its own
   `raw` (`line_test.go`, `lineHarness.raw`).
2. **The harness's line ends.** `lineHarness` gains `enter string` (default `"\r"`); the first-run answer becomes
   `"n" + h.enter` unless a test sets `h.answer`. A helper `eachEnter(t, func(t *testing.T, enter string))` runs a
   subtest per form, named `CR`, `LF` and `CRLF`, for `"\r"`, `"\n"` and `"\r\n"`. `TestFirstRunSetup`,
   `TestLoginMenuAndRefusals` and `TestLoginAsksThroughQuestions` answer their questions (not the editor's prompt,
   which is raw, where Enter is `\r`) with each form. A test helper `recordingPrompter` wraps a `provider.Prompter`
   and records what each `Secret` returned, so a test can run the real `provider.SignIn` and see the key it was
   given. A helper counts `firstRunQuestion` and `Warning: no model credential` in stderr. `lineHarness` gains
   `noAnswer bool`, which leaves the first offer unanswered for a test that answers it itself.
   *(2026-11-20, amended at P1's build pass 1; the owner, relayed by the coordinator: "Red for the defect in CR+LF;
   guard in CR and LF (Recommended)".)* Answered with each form, `TestFirstRunSetup` meets the defect in `CRLF`: its
   `CRLF` subtest is red for the defect at the base and passes after P2; its `CR` and `LF` subtests are guards.
   `TestLoginMenuAndRefusals` and `TestLoginAsksThroughQuestions` stay guards in every form.
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

**Additions, 2026-11-20 (records pass): tests for the state table's decided rows that had none.** Each is P1's, with
its kind; its plant (the one production change that must turn it red, on a scratch copy) joins P3 under the number
given.

* **A1 (rows R5, R23):** `TestLineEndsBeforeEachQuestion` gains two forms. `CR|LF`: K1's answer and `\r` in one chunk,
  then `\n` opening K2's chunk. "one chunk": K1's answer, E and K2's answer in one chunk. Red for the defect where K2
  is `secret` (`CR|LF`, and "one chunk" with `CRLF`); guards otherwise. Plants: P3 16 (R5), P3 17 (R23).
* **A2 (row R16):** `TestLineEndOfInputMidLine` (`lineend_test.go`): an `input` given `abc` and then the end of
  input is `errCancelled`. Guard. Plant: P3 18.
* **A3 (row R28):** `TestSecretRawFailureReadsNothing` (`lineend_test.go`): `raw` fails; `secret` returns its error,
  writes no prompt, and a stub that counts reads is not read. Guard. Plant: P3 19.
* **A4 (row R30):** `TestSecretRestoredWhenContextEnds` (`lineend_test.go`): the context ends while a secret reads;
  `errCancelled`, raw mode entered once and restored once. Guard. Plant: P3 20.
* **A5 (row R21):** `TestLoginProviderMethodMenuWaits` (`login_test.go`, over `eachEnter`): the offer declined;
  `/login openai\r`; the API key item + E; the key recorded. Red for the defect, `CRLF`. Plant: P3 mutation 1, whose
  must-fail list gains it.
* **A6 (row R56):** `TestEditorKeepsKeysTypedWithLogin` (`login_test.go`): the offer declined; `/login\r1\r` in one
  press; the provider menu waits and does not take `1`; Gemini's number + E and the key complete the sign-in; the next
  prompt the agent receives is `1`. Guard. Plant: P3 21.
* **A7 (row R15):** `TestFirstRunEndOfInputAtOfferEnds` (`firstrun_test.go`, `noAnswer`): the pipe closed with the
  first offer unanswered; the session ends within the bound, the warning at most once, no menu. Guard. Plant: P3 22.
* **A8 (row R38):** `TestFirstRunInterruptAtRepeatedOfferEnds`: one round cancelled at the key prompt, then the
  interrupt at the repeated offer; the warning once, no menu again. Red for the loop. Plant: P3 23.
* **A9 (row R65):** `TestFirstRunYesOnRepeatStartsAnotherRound`: one round cancelled, then `y`+E at the repeated
  offer; `Sign in to which provider?` shown a second time; the interrupt at that menu, then `n`+E. Red for the loop.
  Plant: P3 24.
* **A10 (row R61):** `TestFirstRunEnterAtFirstOfferSignsIn`: E alone at the first offer shows the provider menu.
  Guard. Plant: P3 25.
* **A11 (row R42):** `TestFirstRunContextEndReturns`: `e.ctx` ended while the key prompt reads; the session returns
  its cause, and `firstRunQuestion` is counted once. Guard. Plant: P3 26.
* **A12 (row R71):** `TestFirstRunSetup` (already in step 2) also counts `firstRunQuestion` once after the sign-in
  succeeds. Guard. Plant: P3 27.
  *(2026-11-20, amended at P1's build pass 1; the owner, relayed by the coordinator: "Red for the defect in CR+LF;
  guard in CR and LF (Recommended)".)* The count is a guard in `CR` and `LF`; in `CRLF` the sign-in does not succeed
  at the base, so the count is checked there from P2 on.
* **A13 (row R72):** `TestAcpLoginMethodMenuWaits` gains a case: Ctrl+C at the key prompt gives
  `sign-in cancelled; nothing was saved`, and no `firstRunQuestion`. Guard. Plant: P3 28.

*Exit criterion (2026-11-20): no blocker, every row tested, plants caught. For P1, "tested" is each row's test written
and seen at the base with the kind it is given; its plant is caught in P3.*

**Additions, 2026-11-20 (second records pass).**

* **A3 and A17 wait** for the owner's answers to Q18 (row R28) and Q19 (row R57): their rows are open again; each is
  written, and its plant (P3 19 and 21) run, only as the answer decides.
  *(2026-11-20, third pass: answered, Q18 (a) and Q19 (a). A3 and A17 are written as first specified, with plants 19
  and 21. Q16 was answered (a), so the interrupt tests below type as first written; none changes.)*
* **Tests that interrupt a read and then type** (`TestLoginInterrupt`, existing; `TestFirstRunOfferAgainAfterMenuInterrupt`,
  `TestFirstRunInterruptAtOfferEnds`, A8, A9) leave a read waiting that D13 may drop. What each types after the
  interrupt is set by the owner's answer to Q16 (row R86) at step 0, before they are written; under Q16 (a) none
  changes.
* **A20 (row R69):** `TestFirstRunEndOfInputInRoundEnds` (`firstrun_test.go`): `y`+E, Gemini's number + E, then the
  pipe closed while the key prompt reads; the cancel line, `firstRunQuestion` counted once, the warning once, the
  session ends within the bound. Red for the loop (at the base the warning is not printed). Plant: P3 mutation 10,
  whose must-fail list gains it.
* **A26 (row R31):** `TestSecretRestoredOnPanic` (`lineend_test.go`): a reader stub that panics while a secret reads;
  the test recovers it; raw mode entered once and restored once. Red at the base (the restore is not deferred); passes
  after P2 step 4. Plant: P3 54.

*Exit criterion (2026-11-20, unchanged): no blocker, every row tested, plants caught.*

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
     *(2026-11-20, fourth pass): before any read, the probe child calls `signal.Notify` for `os.Interrupt`, as gobble does
     (`main.go:54`, `term.ShutdownSignals`), so Ctrl+C reaches it as a signal and the console's default handler does not
     end it (inferred from Go 1.27.4 `runtime/os_windows.go:1047-1069`: with no `Notify`, the handler returns 0 and
     the default one ends the process). The child writes `signal` to its result file when the interrupt arrives.)*
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

**Addition A14, 2026-11-20 (records pass), row R35:** the gobble mode gains a typed-ahead case: the parent writes `y`,
Enter, `1`, Enter into the pseudo console before the offer is shown, and requires
`Enter your Gemini (Google) API key: ` with nothing more written. It measures the second half of MADR D6's third
assumption (keys typed past a line-mode read stay in the console's buffer). Expected at the base: the menu waits, the
typed-ahead answer lost with the first ask (weakness (a)); after P2, the key prompt. Plant: P3 mutation 4, run through
`make probe-conpty` on the Windows test laptop as well as by `go test`.

*(2026-11-20, second records pass: A14's expectation and plant above are wrong, as the review found. On a console a
line read returns `y\r\n` and leaves `1\r` in the console's buffer, so at the base A14 shows the CRLF defect (the
cancel line), not a waiting menu: it is red for the defect, passing after P2. Mutation 4 cannot fail it, since a new
`terminalInput` per ask still reads `1\r` from the console and drops the owed `\n` inside `/login`'s ask. Its plant is
P3 42, "flush before every question", which drops the typed-ahead `1\r` and leaves the menu waiting. Mutation 4's
must-fail list loses A14. A14 is the case `typeahead` below.)*

*Exit criterion (2026-11-20): no blocker, every row tested, plants caught.*

**Amendment, 2026-11-20 (second records pass): who runs P1b, what it prints, and the cases it gains.**

* **Who runs it** (MADR D16; the earlier answer to Q7 stands): an agent, never the owner. The probe cases run at P1
  step 0b. The rest run, at the base after P1 and again on the P2 and P2b trees, as

  ```text
  run-on-host --need windows --label 0063-conpty -- make probe-conpty
  run-on-host --need linux --label 0063-pty -- make probe-pty
  ```

  Each prints only `0063 case=<name> result=<pass|fail> observed=<word>` lines; the agent passes a run when every
  case named for that tree is present with its expected word, and stops on any other line, a missing line or a
  non-zero exit. No screen text, key or path is printed or recorded. The second run is on the Linux machine (the
  pseudo-terminal cases need a POSIX terminal; a WSL distro is a Linux target too, and any is taken, as no race is
  run).
* **Files:** the gobble-mode cases that need a stubbed sign-in cannot run from `cmd/gobble` (the stubs are
  `internal/cli`'s), so they are in `internal/cli/conpty_windows_test.go` (`//go:build windows && conpty`), whose
  child runs `Main` under `internal/cli`'s `TestMain`, which already stubs the browser opener and the sign-ins; and in
  `internal/cli/pty_linux_test.go` (`//go:build linux && pty`), which opens a pseudo-terminal with
  `golang.org/x/sys/unix` (already required; no new module). `make probe-conpty` runs both Windows files; `make
  probe-pty` the Linux one; neither is in `preflight` or CI.
* **Cases** (the MADR's Confirmation item 5 (A) gives each one's words; expected at the base, then after P2b):

  | Case | File | At the base | After P2b |
  | --- | --- | --- | --- |
  | `probe-crlf`, `probe-typeahead`, `probe-ctrlc`, `probe-ctrlz-line`, `probe-ctrlz-raw` | `cmd/gobble` | as step 0b | the same |
  | `report`, `method-menu`, `setup`, `acp-login`, `typeahead` (A14) | `cmd/gobble` | fail: the cancel line before a key | pass, `waits` or `cancelled` |
  | `masked`, `key-ctrlc`, `empty-key`, `decline`, `login-no-loop`, `nothing-saved` | `cmd/gobble` | `masked`, `no-offer`, `none` pass; the three that expect the offer again fail (no loop) | pass |
  | `menu-ctrlc` | `cmd/gobble` | fails (no loop), or `ended`: the R40 stop | `offer-again`, or `ended`: the R40 stop |
  | `key-ctrlz` | `cmd/gobble` | `ended` | `ended` |
  | `flush-prompt`, `flush-offer` (A21) | `internal/cli` | fail: the held line becomes a turn; the repeated offer is answered | pass (`flush-prompt` after P2, `flush-offer` after P2b) |
  | `abandoned` (A21) | `internal/cli` | fail: `partial` or `hello` reaches a turn | pass after P2 |
  | `pty-flush-prompt`, `pty-flush-offer`, `pty-abandoned` (A21) | `internal/cli` | as their Windows twins | as their Windows twins |

  The stubbed sign-ins: `flush-offer`'s round waits one second without reading and then fails, while the parent
  writes `y`, Enter; `abandoned`'s is a browser sign-in whose paste prompt reads through the prompter until the stub's
  own "loopback" ends its context after the parent has typed `partial`. Neither opens a browser or reaches a network.
* **Gate, added:** every case as the table says for its tree. **Stop, added:** `probe-ctrlc` or `menu-ctrlc` gives the
  R40 stop word, or any case's observed word is not one the table names.
* *(2026-11-20, Deviation 1, MADR D19.)* The table's `probe-ctrlc` and `menu-ctrlc` rows are amended: `probe-ctrlc`
  expects `eof` on every tree (the raw console's behaviour); `menu-ctrlc` gives `ended` at the base (expected there)
  and must give `offer-again` after P2, then `warned-once` when `n`, Enter answers the repeated offer; `ended` after P2
  is a failure of D19, not the R40 stop. New cases, each with its expected words at the base and after P2:

  | Case | File | At the base | After P2 |
  | --- | --- | --- | --- |
  | `offer-ctrlc`: Ctrl+C at the first offer, then `/exit` | `cmd/gobble` | `ended` (the input closed) | `warned-once`, and the prompt takes `/exit` |
  | `setup-menu-ctrlc`: `gobble setup`, Ctrl+C at the provider menu | `cmd/gobble` | `ended`, `auth.json` empty | the same |
  | `pty-ctrlc-ctrld`: the first run; `y`, Enter; Ctrl+C at the provider menu; Ctrl+D at the repeated offer | `internal/cli` (Linux) | the menu interrupt not yet a loop: no repeated offer | `warned-once` |
  | *(2026-11-20, D20)* `probe-ctrlc`'s `first=` word | `cmd/gobble` | `signal` or `read`, recorded | the same; neither stops |
  | *(2026-11-20, D20)* `auth-paste-ctrlc`: `gobble auth login` with a stubbed browser sign-in; Ctrl+C at its paste prompt | `internal/cli` | `ended`, `auth.json` empty | the same (R120) |
* *(2026-11-21, correction after P1 and P1b's build pass 1 and its review.)* The base words of five cases, as
  measured (execution record, "P1 and P1b, build pass 1"), and each case's word on the P2 tree and after P2b. The
  table's "fail (no loop)" and "the repeated offer is answered" give the reason those cases fail once the defect is
  gone; at the base another reason comes first.

  | Case | At the base | On the P2 tree | After P2b |
  | --- | --- | --- | --- |
  | `key-ctrlc` | `cancel-line-before-key`: the CR+LF defect cancels the key prompt before any key | `no-offer-again` | `offer-again` |
  | `empty-key` | `cancel-line-before-key`, as above | `no-offer-again` | `offer-again` |
  | `decline` | `cancel-line-before-key`, as above | `no-offer-again` | `warned-once` |
  | `flush-offer` | `no-offer-again`: no repeated offer exists | `no-offer-again` | `offer-waits` |
  | `pty-flush-offer` | `no-offer-again`, as above | `no-offer-again` | `offer-waits` |

  Deviation 1's words above are corrected too. `menu-ctrlc`'s `offer-again`, and `warned-once` after `n`, Enter,
  need P2b's loop, not P2 alone: on the P2 tree D19 keeps the input open, so the round is cancelled and the session's
  prompt follows with no repeated offer, and the case as written reads `other` there (the word `no-offer-again` for
  it is a P2 follow-up, execution record); `ended` on the P2 tree or later is still D19's failure. The same holds for
  `pty-ctrlc-ctrld`, whose `warned-once` comes after P2b; on the P2 tree it reads `no-offer-again`. `offer-ctrlc`'s
  `warned-once` needs only P2: Ctrl+C at the first offer is a decline already, and D19 keeps the input open for the
  prompt that takes `/exit`.

### P2 — the shared terminal input

*Class: strongest; effort: high. It changes the read path every prompt and every editor key takes.*

*(2026-11-21, D22 accepted: P2's files gain `internal/cli/term/cancelin_unix.go`, `cancelin_other.go` and `cancelin_test.go`
(steps 15, 19); `internal/cli/pty_unix_test.go` and `internal/cli/pty_darwin_test.go` (new), `pty_linux_test.go`
restructured, and the `Makefile`'s `probe-pty` comment (step 20, Q26, "Move them to a shared file (Recommended)").)*

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

**Additions, 2026-11-20 (records pass): tests for decided rows that had none.** Each is written in P2 (it needs
`terminalInput`), passes after P2, and its plant joins P3.

* **A15 (rows R9, R10, R11, R43, R44):** `terminput_test`'s table gains rows: (i) a raw read ended by `\n`, then
  `\n`: the second `\n` is given (R9; plant P3 29); (ii) the owed `\n` driven through each entry point, `line`, `key`
  and `Read`, dropped at each (R10; plant P3 30); (iii) `a\r` then the end of input: `a`, then the end of input, and
  `ended()` true (R11; plant P3 31); (iv) a stub whose read fails with an error that is not `io.EOF`: the question is
  `errCancelled` and `ended()` is true (R43; plant P3 32); (v) `\xff`, a lone `\xc3` and `\x00`, through `line`,
  `key` and `Read`, given unchanged (R44; plant P3 33).
* **A16 (row R58):** `TestTerminalInputBytesInOnePlace`: one stub chunk `a\rbc`; `line` gives `a`, `Read` gives `bc`,
  and the next `Read` makes a new read of the stub. Plant: P3 21.
* **A17 (row R57):** `TestAskOnKeysPastAnswerNotShared`: `askOn` over `terminalInput.Read`, given `y` and `hello\r`
  in one read, is answered; the next `line` makes a new read of the stub and does not see `hello`. Plant: P3 21.
* **A18 (rows R45, and D10's `gobble auth login` row):** `TestOneReaderOfStdin`: the package's non-test Go files
  call `inline.NewReader` only in `terminput.go`. Plant: P3 34.
* **A19 (row R48):** `TestTerminalInputConcurrentReads`: two goroutines read through `line` and `Read` at once from a
  stub of a known byte string; every byte is given once. Run under `-race` in P4 step 5. Plant: P3 35, red under
  `-race`.

*Exit criterion (2026-11-20): no blocker, every row tested, plants caught.*

**Additions, 2026-11-20 (second records pass): steps for MADR D12, D13 and D15, and their tests.**

* **Step 9, the flush seam** (D12): `termFlush = func(f *os.File) error`, the POSIX flush of pending input on Linux
  and macOS and `FlushConsoleInputBuffer` on Windows, each in a file of its own build tag; `terminalInput` gains
  `flushNext()`, which marks that the next read of `in` is preceded by `termFlush` on a terminal (never on a pipe or
  a file). Before each session prompt, with `discard()`, the line session calls it (when, is Q20's answer, row R82);
  what a failed flush does is Q17's answer (row R78). Both are answered at P1 step 0.
* **Step 10, `abandon()`** (D13): a prompt that returns while its read may still be waiting calls it: `questions` when
  `line` or `key` ends on its context; `askPermission` and `askTrust` when `askOn` returns cancelled; the line session
  when `ReadPrompt` ends on its context. A read so marked has its completion dropped as Q16's answer says (row R86),
  and the prompt then waiting reads again. A timed wait inside a prompt that reads again calls nothing.
* **Step 11, D15:** `secret` restores the terminal in a deferred call; `askTrust`'s read is
  `e.terminalIn().Read(e.ctx, p)`, ignoring `askOn`'s deadline contexts, as its comment says today.
* **A21 (rows R25, R51, R69, R74, R76, R83, R84, R85):** the pseudo-console and pseudo-terminal cases of P1b's
  amendment, `flush-prompt`, `flush-offer`, `abandoned` and their `pty-` twins, written in P1 and run on each tree.
  Plants: P3 43, 44, 46, 47.
  *(2026-11-21, D22 second draft: under D22 `pty-abandoned` passes only with `hello` and `again` as turns and no `partial`;
  plants 46 and 47 are then caught on POSIX by A42, and on Windows by `abandoned`.)*
  *(2026-11-21, MADR D21; the owner, relayed by the coordinator: "Amend R86/R102 to what Linux does (Recommended)".)* `pty-abandoned`
  passes, on the P2 tree and after, when no turn holds `partial` and a turn is `again`; `hello` may be a turn: on
  POSIX what was typed into the paste prompt completes the abandoned read as raw mode goes on, and nothing typed at
  gobble's prompt is lost (row R102, as amended). The check in `internal/cli/pty_linux_test.go` changes by one line,
  the one P1b test change the owner allowed, in P2's build pass 2. `abandoned` (Windows) is unchanged. Plants 46 and
  47 must still fail it: each brings `partial` to a turn.
* *(2026-11-21, D22 second draft: under D22 `first-key-dropped` is a failure on POSIX; A40's words are its split's, below.)*
* **A40 (row R102), added 2026-11-21** (P2's build pass 2; the coordinator's records gate: R102's inferred POSIX
  branch had no test): `pty-abandoned-empty` in `internal/cli/pty_linux_test.go`, on the Linux target with
  `make probe-pty`: `pty-abandoned`'s drive with nothing typed at the paste prompt; then `hello`, Enter at gobble's
  prompt; then `again`, Enter. Its words: `first-key-dropped` (a turn `ello` and a turn `again`: the first key
  completed the abandoned read and was dropped with it, as R102 infers), the pass on the P2 tree and after;
  `none-lost` (a turn `hello`: the inference is wrong, a stop, the row unchanged until the owner rules);
  `no-turn-again`, `other-turns`, and the drive's own words for a step that did not come. Plant 47 must fail it
  (`none-lost`). Windows's `abandoned` and its inferred first line are outside this addition.
* **A22 (row R46):** `TestTerminalInMadeOnce`: `e.terminalIn()` called twice, and from two goroutines at once, gives
  one value. Plant: P3 51.
* **A23 (row R60):** `TestPermissionReadsTerminalInput`: the session's `terminalInput` over a stub that counts reads;
  a permission prompt answered by one key; the stub counted the read. Plant: P3 52.
* **A24 (rows R77, R79, R80, R81):** `TestFlushPoints`: the harness stubs `termFlush` and records each call with
  stderr's length at that moment. It is called before the session prompt's read and, at a repeated offer, after the
  offer's text is written; never at the first offer, inside a round, in `gobble setup`, `gobble acp --login`, the
  trust question or a permission prompt; and the pipe's later bytes still come in order. Plants: P3 42, 43, 44, 45.
* **A25 (row R17):** `TestLongAnswerHasNoBound`: an `input` answered with 1 MiB and `\n`, and a secret with 1 MiB and
  `\r`, each read whole. Plant: P3 53.
* **A27 (row R50):** `TestTrustReadsWithoutDeadline`: `e.termIn` over a stub that records each read's context; the
  trust question answered; no read had a deadline. Plant: P3 55.
* **A28 (rows R84, R87, R88, R89):** `terminput_test` rows over a stub whose read blocks until the test gives it bytes:
  a read abandoned, then `hello\n` given: the next `line` does not see `hello` and reads anew; a read whose wait ended
  by a deadline and that the same caller reads again keeps its bytes; two prompts give up on one read: one completion
  dropped; a read abandoned, then the end of input: `ended()` true, the waiting prompt sees the end of input. Plants:
  P3 47, 48, 49, 50.

**Additions, 2026-11-20 (third records pass): the owner's answers to Q16 to Q20 (MADR D17).**

* **Step 9, as answered:** `terminalInput` keeps a mark, `flushAtPrompt`. `ls.signInRound` sets it when it returns,
  whatever the outcome; `firstRun` sets it when it returns, however it ends; a `/login` refused before any question
  sets nothing. Before each `ReadPrompt` the line session calls `discard()` and, only when the mark is set,
  ~~`flushNext()`~~ *(2026-11-21, MADR D21: flushes at once, whether or not the editor then reads the terminal)*, and
  clears the mark (rows R82, R103 to R108, R123). The repeated offer's `flushNext()` (P2b step 4) is
  unchanged. A failed `termFlush` is skipped, and the first failure in the process writes one warning on stderr, in
  the words row R101 holds once the owner approves them (row R78).
* **Step 10, as answered:** `abandon()` is called only for a read the program gave up: by `questions` when the
  context that ended is not `ls.ask`'s interrupt cancel (the SDK's `Input` context, a browser sign-in's loopback), and
  by `askPermission` when the request was withdrawn without the person's Ctrl+C. Its completion is dropped whole, a line
  or keys (rows R86, R102). A read the person interrupted is not marked and goes to the next prompt.
* **A24, extended (rows R82, R103 to R108):** `TestFlushPoints` also checks: a flush at the first `ReadPrompt` after a
  `/login` that signed in, one that was cancelled and one that failed, and none after `/login nosuch`; a flush at the
  first `ReadPrompt` after a first run declined at the first offer, and after one signed in; keys written while the
  `/models` turn after `/login` runs are not the next prompt's text; no flush at the `ReadPrompt` after a turn, after
  `/logout`, after a permission prompt, or at a resumed session's first prompt with a credential; ~~none when the
  session ends by Ctrl+D with the mark set~~. Plants: P3 56, 57, 58, 61.
  *(2026-11-21, MADR D21; the owner, relayed by the coordinator: "Flush at once at that first prompt (Recommended)".)* Struck: with the flush made at
  the prompt, a session cannot end by Ctrl+D with the mark set. In its place (row R108): a session ended by Ctrl+D at
  the first prompt after a sign-in counts that prompt's one flush and none after; the run's context ended during the
  `/models` turn after `/login` counts none. Added (row R123): `/login gemini` and `1`, Enter in one press, the key
  prompt answered with Enter: one flush, at the prompt that submits `1`, before its turn; keys typed while that turn
  runs reach the next prompt. Plant 75.
* **A28, extended (row R86):** a read the program gave up, completed by keys with no line end, is dropped whole; a
  read given up by `ls.ask`'s interrupt cancel and then completed is given to the next prompt. Plants: P3 47, 60.
* **A29 (rows R78, R101):** `TestFlushFailureWarnsOnce`: `termFlush` stubbed to fail; two flush points reached; one
  warning on stderr, in R101's approved words, and both prompts still read. Written once R101's words are approved.
  Plant: P3 59.

**Additions, 2026-11-20 (fourth records pass): MADR D18's plumbing, and R101's words.**

* **Step 1, as amended:** `newTerminalInput(in keysIn, flush func() error)`; `flush` is nil for a pipe, a file or a
  test's input that has no terminal, and `flushNext()` then marks nothing. **Step 3, as amended:** `e.terminalIn()`
  passes, when `caps.In` is true, a function that calls `termFlush(e.stdin)` and on the first failure in the process
  writes, through `e.out`, `Warning: could not clear what was typed ahead on this terminal (<error>); it may answer the
  next prompt.` (R101, approved). `termFlush` is a package variable, `term.FlushInput` in production, which the line
  harness and A24 stub.
* **Step 12, `term.FlushInput`:** the four files named in Scope's fourth-pass note; `flush_test.go` checks, on a pipe,
  that it returns an error and never blocks. Gate, added: `make archtest` exit 0 with the new import of
  `golang.org/x/sys/unix` in `internal/cli/term`; a new edge reported there is a stop, put to the owner, not worked
  around.
* **Step 10, as amended:** when a read a program gave up completes, `terminalInput` takes the completion from
  `inline.Reader` with a buffer of that read's own size (512 bytes for a question), whichever prompt is waiting, and
  drops it whole; the waiting prompt then reads with its own buffer.
* **A28, extended (MADR D18):** a question's read (512 bytes) abandoned by the program; 400 bytes then given; a
  waiting `askOn` reading through a 256-byte buffer receives none of them and its next read is a new one. Plant: P3 63.
* **A29, as approved:** `TestFlushFailureWarnsOnce` asserts R101's approved words exactly. Plants: P3 59, 62.

**Additions, 2026-11-20 (Deviation 1, MADR D19): the console read again after an interrupt.** Steps 13 and 14 wait
for the owner's answers to Q21 and Q22; P2 does not start before them (P1 step 0).

* **Step 13, `term.ConsoleInput`:** `internal/cli/term/consolein.go` (the portable rule: given a reader, whether it is
  a console, and the interrupt mark, a read that gives nothing with the mark set reads again, as Q21 and Q22
  settle), `consolein_windows.go` (a console is a handle `GetConsoleMode` accepts), `consolein_other.go` (standard
  input unchanged), and `consolein_test.go` (the stub tests below, on every system). `e.terminalIn()` passes
  `term.ConsoleInput(e.stdin, …)` to `inline.NewReader` in place of `e.stdin` (`main.go`).
* **Step 14, the mark:** `watchSignals` (`signal.go:60-89`) sets the mark on each `os.Interrupt`, before
  `intr.take()`; the mark is the process's, read by `term.ConsoleInput`.
* **A30 (row R109):** `TestConsoleReadAgainAfterInterrupt`: a console stub; the mark set; the stub's read gives
  nothing, then `n\r\n`; the reader gives `n\r\n` and no end of input. Plant: P3 64.
* **A31 (row R110):** `TestWatchSignalsMarksInterrupt`: `os.Interrupt` through the captured `notify`, once with a
  prompt taking it and once with none; the mark set both times. Plant: P3 65.
* **A32 (row R112):** `TestConsoleCtrlZAfterInterruptEnds`: the mark set; the stub gives nothing twice; the first is
  excused, the second is the end of input. Plant: P3 66.
* **A33 (row R113):** the existing `TestInterruptDuringTurn` and `TestInterruptReleased`, run after step 14. Plant:
  P3 67.
* **A34 (row R115):** `TestConsoleRawCtrlCIsAByte`: no mark; the stub gives `\x03`; the reader gives `\x03`. Plant:
  P3 68.
* **A35 (row R117):** `TestConsoleRuleOnlyOnAConsole`: not a console; the mark set; the stub gives nothing; the end of
  input. Plant: P3 69.
* **Rows R111, R114 and R116** get their tests with the owner's answers to Q21 and Q22.
  *(2026-11-20, answered, MADR D20; steps 13 and 14 no longer wait.)* Step 13's rule: the reader notes when its read
  of the console is waiting; an empty read with the mark set is excused and clears it; an empty read with no mark
  waits up to 100 ms for one; any read that returns bytes clears it. Step 14: `watchSignals` sets the mark only while
  that read is waiting.
* **A36 (row R111):** `TestConsoleMarkExcusesOneRead`: the mark set while a read waits; that read gives nothing and
  is excused; the next gives nothing and is the end of input; in a second case, bytes come between the interrupt and
  an empty read, which is then the end of input. Plant: P3 71.
* **A37 (row R114):** `TestConsoleNoMarkWithoutWaitingRead`: an interrupt while no read waits; a later empty read is
  the end of input after the wait. Plant: P3 72.
* **A38 (row R116):** `TestConsoleWaitsForLateMark`: the stub's read gives nothing; the mark is set 30 ms later; the
  read is excused and the next bytes arrive. Plant: P3 73.
* **A39 (rows R116, R122):** `TestConsoleWaitBound`: an empty read and no mark: the end of input after at least
  100 ms and under 500 ms; a mark set at 600 ms excuses nothing. Plant: P3 74.

*Exit criterion (2026-11-20, unchanged): no blocker, every row tested, plants caught.*

**Additions, 2026-11-21 (MADR D22, proposed): a POSIX read that is never left waiting.** On the owner's answer to F3
and F4, relayed by the coordinator: "Make reads cancellable on POSIX" (not the recommended option). Drafted for review; nothing is
built. Steps 15 to 18 wait for the owner's acceptance of D22, and 17 and 18 for Q23 and Q24.

* **Step 0 (D22), the measurements first.** Before step 15, the probe of the execution record's "Records draft for
  D22" (M0 to M8), a scratch program never in the tree, is run again on this Mac and on the Linux target: a Linux
  build of it (`TCGETS`/`TCSETS` for `TIOCGETA`/`TIOCSETA`, the `TCFLSH` ioctl with `TCIFLUSH` for `TIOCFLUSH`),
  driven on a pseudo terminal by the same driver (Python's `pty` module, or the opener of `pty_linux_test.go`),
  through `run-on-host --need linux` from a scratch clone; on a WSL distro and, if the fleet has one, a Linux machine
  that is not. Each line recorded verbatim. Stop if select(2) does not wait on the terminal and report its readiness
  (M2, M3, M5), or does not survive Ctrl+C (M6), on either system.
* **Step 15, the layer** (`internal/cli/term`, `cancelin_unix.go`, `cancelin_other.go`, `cancelin_test.go`): a reader of
  standard input with a gate, as D22 says: select(2) on standard input and a close-on-exec wake pipe, `EINTR`
  retried, a read(2) only with the gate open and after readiness, under the gate's lock; `Open()`, and `Close()`
  reporting whether a read completed since the gate opened. A seam for select(2), so a test can give `EINTR` and an
  error. Rows R124 to R127, R133, R134, R142, R144.
* **Step 16, the wiring:** `terminalIn()` reads a POSIX terminal through the layer, Windows through
  `term.ConsoleInput`, a pipe or a file directly; `terminalInput.fill` opens the gate before its read of the input and
  closes it when the read returns; on a read that ended on its context, `waiting` (and so `abandon()`'s drop) is set
  only when the close reported a completed read. Rows R128 to R131, R136, R137, R140, R141, R143.
* **Step 17:** Q23's answer (row R132). **Step 18:** Q24's answer (row R138).
  *(2026-11-21, D22 accepted.)* **Step 17, as answered** (Q23, "Flush at gobble's give-up (Recommended)"): on POSIX, the terminal's queue is flushed at
  the program's give-up itself, through the terminal input's flush: `askPermission`'s withdrawn branch and
  `questions.cancelled`'s program branch, inside the session and out of it; a Ctrl+C give-up does not flush; Windows
  keeps D13's drop (rows R80, R81, R84, R132). **Step 18, as answered** (Q24, "Old reads plus one warning (Recommended)"): when the readiness
  wait or its descriptor cannot be used, the process reads standard input as built, and writes once on stderr, through
  `e.out`, exactly `Warning: gobble cannot wait on this terminal without reading it (<error>); keys typed while a sign-in finishes may be lost.` (rows R138, R150).
* **A40, as split (row R102):** `pty-abandoned-empty-typed` (`hello`, Enter a key at a time, 150 ms apart) and
  `pty-abandoned-empty-pasted` (written at once); each passes as `none-lost` (turns `hello` and `again`), on Linux and
  on this Mac. `first-key-dropped` and `other-turns` are failures under D22.
* **A41 (rows R125 to R127, R133, R134):** `TestCancelableWait`, over a pipe, which select(2) serves as it does a
  terminal: a read only after data; a close before data reads nothing, and what is written after is read whole after
  the next open; a close wakes the wait at once (under 100 ms); a close during a read in progress waits for it and
  reports it; `EINTR` from the seam retried; the end of input given as it comes. Plants 76 to 79, 84.
* **A42 (rows R128, R129):** `TestCancelableGiveUp`: `terminalInput` over the layer over a pipe: a program's give-up
  and a Ctrl+C give-up with nothing read drop nothing; over a stub layer that reports a completed read, the
  program's give-up drops that completion and the Ctrl+C give-up gives it to the next prompt. Plants 80 to 82.
* **A43 (rows R130, R131), `pty-abandoned` as amended:** no turn holds `partial`, and `hello` and `again` are turns;
  on Linux and on this Mac. Plants 76, 86.
* **A44 (row R133):** `pty-menu-ctrlc-typed`: at the provider menu, `ab` typed, then Ctrl+C: the round is cancelled
  and no turn or answer holds `ab`. Plant 79 (on Linux).
* **A45 (row R134):** ~~`pty-ctrld-prompt`: Ctrl+D at gobble's empty prompt ends the session with exit 0.~~
  *(2026-11-21, D22 second draft: gobble's prompt reads in raw mode, where Ctrl+D is a byte; moved below.)*
* **A46 (rows R124, R137):** `TestTerminalInChoosesLayer`: a pipe is read directly; a pseudo terminal's slave through
  the layer. Plant 83.
* **A47 (row R140):** in `TestCancelableGiveUp`, a timed wait of the same caller: nothing read, and the bytes
  written after it come whole to its next read.
* **A48 (row R142):** `TestCancelableConcurrent`, opens, closes and reads from two goroutines; run under `-race` in P4
  step 5. Plant 85.
* **A49 (row R132)** and **A50 (row R138)** are written with Q23's and Q24's answers.
* **The pseudo-terminal cases on this Mac:** `internal/cli/pty_linux_test.go`'s cases run on macOS too, by a
  `darwin && pty` file with macOS's pseudo-terminal opener (`/dev/ptmx`, `TIOCPTYGRANT`, `TIOCPTYUNLK`,
  `TIOCPTYGNAME`), and `make probe-pty` builds on either system; outside `preflight` and CI.
* **Gate (D22):** P2's gate, plus: every A41 to A48 passing; the pseudo-terminal cases on Linux and on this Mac as
  above; the Windows cases unchanged, word for word; step 0's lines recorded.

*Exit criterion (2026-11-21): no blocker, every row tested, plants caught.*

**Additions, 2026-11-21, D22 second draft, after the independent record review (its findings 1.1 to 3.10) and M9 to M14.**
Records only; nothing built. Each amends the first draft above by its number.

* **Step 0 (D22): done** for the measurements M0 to M10 on this Mac, a WSL distro and a Linux machine that is not WSL,
  and M11 to M14 on all three (execution record, "Records draft for D22, second draft"). Repeated before step 15
  only if the probe or the hosts change.
* **Step 15, as amended:** the layer reads the descriptor Q25 decides; a seam for select(2) (`EINTR`, an error, and a
  report of each return from the wait) and a seam for read(2) (to hold a read between readiness and its return, and
  to give 0 and `EIO`); the closed gate waits on the wake pipe alone (R145); the wake pipe as R146 says, with
  `stop()`; ~~the gate closes itself after a read (R127);~~ *(2026-11-21, D22 third draft: dropped; the gate stays open until
  `fill` closes it, R127)* a descriptor at or above `FD_SETSIZE` is checked before
  `FdSet.Set` and is Q24's case.
* **Step 16, as amended:** `fill` asks `inline.Reader` with `lineBuf` always (R147) *(2026-11-21, D22 third draft: on the
  gated path; with no gate, the caller's buffer, as built)*; with no gate, `waiting` is set on
  every read that ended on its context, as built (R151); with a gate, only when the close reported a completed read.
* **Step 19:** Q25's answer (R126). **Step 20:** Q26's answer (R152).
  *(2026-11-21, D22 accepted.)* **Step 19, as answered** (Q25, "Private non-blocking descriptor (Recommended)"): the layer reads a private descriptor on the
  terminal's own path (`/proc/self/fd/0` on Linux, `F_GETPATH` on macOS), opened
  `O_RDONLY|O_NONBLOCK|O_CLOEXEC|O_NOCTTY`, once for the process; when that open fails and standard input is the
  controlling terminal (the same foreground process group on standard input and on `/dev/tty`), `/dev/tty` with the
  same flags; only then step 18's fallback; `EAGAIN` returns to the wait; raw mode and the flush stay on standard input
  (rows R125, R126, R127, R134, R138, R146, R148). **Step 20, as answered** (Q26, "Move them to a shared file (Recommended)"): the cases of
  `internal/cli/pty_linux_test.go` move word for word to `internal/cli/pty_unix_test.go` (`(linux || darwin) && pty`);
  the Linux opener stays in `pty_linux_test.go`; a new `pty_darwin_test.go` holds the macOS opener (`/dev/ptmx`,
  `TIOCPTYGRANT`, `TIOCPTYUNLK`, `TIOCPTYGNAME`); the Makefile's comment on `probe-pty` says it runs on Linux and
  macOS, outside `preflight` and CI (rows R130, R131, R152).
* **A49 (rows R80, R81, R84, R132), as answered:** `TestFlushAtGiveUp`, with `termFlush` stubbed to count: a
  permission request withdrawn while its prompt reads, over a gated input, flushes once, at the withdrawal; `gobble
  auth login`'s paste prompt ended by a stub loopback, on a terminal with a gated input, flushes once, at the give-up,
  before gobble exits; a Ctrl+C give-up at a menu flushes nothing. Plants 94, 95.
* *(2026-11-21, D23: A52's "`ended()` true after `EIO`" is asserted at the layer only; a case through `terminalInput` is
  added in the fourth pass, below.)*
* **A50 (rows R138, R150), as answered:** `TestCancelableFallbackWarnsOnce`: through the seams, select(2) failing,
  then both opens failing in another run: the input reads as built, the prompts still answer, and stderr holds exactly
  `Warning: gobble cannot wait on this terminal without reading it (<error>); keys typed while a sign-in finishes may be lost.` once, the error in its place; two causes in one process warn once. Plants 93, 97.
* **A55 (rows R126, R146), added:** `TestPrivateDescriptor`, on a pseudo terminal on Linux and on this Mac: the layer's
  descriptor is not standard input, is non-blocking and close-on-exec, standard input stays blocking, and the path's
  open failing falls back to `/dev/tty` when standard input is the controlling terminal. Plant 96.

* **A40, as split, restated:** `pty-abandoned-empty-typed` and `pty-abandoned-empty-pasted` pass as `none-lost`;
  `first-key-dropped` (build pass 2's typed result on the tree as built) and `other-turns` fail.
* **A41, as amended:** asserts exactly `(0, io.EOF)` for the end of input (R134); the read seam gives 0 and `EIO`
  (R148); the wait's seam reports that a close returned only after the waiter left select(2) (R127, plant 77).
* **A42, as amended:** a gateless case (R151, plant 90); a parked 512-byte ask, a 256-byte read, `discard()`, and only
  bytes typed after it (R147, plant 89); no read(2) through the seam between a prompt's return and the next opening
  (R127, R150, plant 84); a closed gate returns nothing (R145, plant 88); the race staged through the read seam
  (R129, plants 78, 81, 82).
* **A43, as amended:** adds `pty-gaveup-held` (R130): a line read given up with `partial` typed, raw mode on, 200 ms:
  the terminal still holds the bytes (select ready; `FIONREAD` 7 on Linux) and the layer reports no completed read.
  Plant 76.
* **A44:** unchanged, but not plant 79's (R133).
* **A45, as moved (R134):** at a prompt read out of raw mode, the provider menu: Ctrl+D cancels the round ~~and `ended()`
  is true~~. Plant 87. *(2026-11-21, D22 third draft: two cases. On the pseudo terminal: the round is cancelled, no repeated offer
  comes, and gobble exits 0 at the editor's first read. In the stub case, `ended()` is true.)*
* **A47, as amended:** plant 84 is aimed here and at A42.
* **A51 (row R141), added:** `TestPermissionTimedWaitOverLayer`: the permission prompt's timed wait over the layer
  reads nothing, and the key typed after it answers.
* **A52 (row R148), added:** a hang-up through the read seam: 0 is `io.EOF`, `EIO` the input's error, `ended()` true.
* **A53:** folded into A42 (R145).
* **A54 (row R146), added:** `TestCancelableWakePipe`: both ends' flags; a hundred closes with no waiter do not block;
  `stop()` ends a waiting Read with `io.EOF`. Plant 91.
* **Gate (D22), as amended:** A41 to A54 as written; plant 85 is P4's (A48 under `-race`); the pseudo-terminal cases on
  this Mac as Q26 decides.

*Exit criterion (2026-11-21, D22 second draft): no blocker, every row tested, plants caught; rows R126, R132, R138 and R152 wait
for Q25, Q23, Q24 and Q26.* *(2026-11-21, D22 accepted: answered; steps 17 to 20, A49, A50, A55; no row waits.)*

**Additions, 2026-11-21, D23: the fourth pass ("Fourth pass, tests only (Recommended)").** The review of build pass 3 (its plants in the
session's review folder) found no blocker; its surviving plants A, B, D, F and H become plants 98 to 102 below, each
with the test that must catch it. The owner's answers add steps 21 to 25; no other production change.

* **Step 21** ("Always flush on a POSIX terminal (Recommended)"): the give-up flush keyed on a POSIX terminal: `terminalInput` knows whether standard
  input is a Linux or macOS terminal, gated or fallen back, and `flushGiveUp` flushes on that, not on the gate (rows
  R80, R132, R138, R157).
* **Step 22** ("Retry EINTR on the read (Recommended)"): `EINTR` from the private descriptor's read(2) returns to the wait (row R153).
* **Step 23** ("Refuse a blocking descriptor (Recommended)"): `NewGated` refuses a descriptor without `O_NONBLOCK`; `OpenGated` then tries
  `/dev/tty`, then R138 (row R154).
* **Step 24** ("Check it's the same terminal"): `OpenGated` compares `fstat`'s `Rdev` of standard input and the opened path's
  descriptor; on a mismatch it closes it and tries `/dev/tty`, with its process-group check (row R155).
* **Step 25** ("Write the warning safely"): the fallback warning is queued and written once on the main flow, before the next
  prompt's or question's text, cooked, or at exit (row R156).
* **A56 (rows R80, R132, R143, R157):** `TestGiveUpFlushBySystem`: a program's give-up over a gateless input that is
  not a POSIX terminal (the Windows path) counts no flush; over a POSIX terminal's input that fell back at start-up,
  and over one that fell back later, each counts one. Plants 98, 107.
* **A57 (row R138):** `TestGatedRefusals`: through seams, the wake pipe that cannot be made, and a descriptor at or
  above `FD_SETSIZE`, each refused with an error, never a panic, and each a fallback with the warning once. Plant 99.
* **A58 (row R124):** `TestTerminalInWiresGate`, in the default suite: with `isTerminalFile` true and `openGated`
  giving a gated reader over a pipe, `terminalIn()`'s input has that gate. Plant 100.
* **A59 (row R126):** `pty-not-controlling`, in the pseudo-terminal cases: a child whose controlling terminal is one
  pseudo terminal opens another's slave as standard input with its path failing: refused (R138), not `/dev/tty`.
  Plant 101.
* **A60 (row R156):** `TestFallbackWarningOnMainFlow`: a wait that fails inside a read on `inline.Reader`'s goroutine
  writes nothing then; the next question's text is preceded by the warning, once; a run with no later text writes it
  at its end. Plant 106.
* **A61 (row R147):** `TestGatedRaceDropSize`: over `inline.Reader` and a pipe with a gate that reports a completed
  read, a 256-byte read given up by the program; 400 bytes complete it; the next 256-byte read gets none of them and
  then the next bytes. Plant 102.
* **A62 (row R153):** in `TestCancelableWait`, the read seam gives `EINTR` once, then the bytes are read. Plant 103.
* **A63 (row R154):** in `TestGatedRefusals`, a blocking pipe given to `NewGated` is refused. Plant 104.
* **A64 (row R155):** `pty-same-terminal`: a child whose controlling terminal is one pseudo terminal, given the path of
  another: what is typed on its own terminal is read, through `/dev/tty`. Plant 105.
* **A52, as amended (row R148):** adds a case through `terminalInput`: the read seam's `EIO` gives the line's error and
  `ended()` true.
* **Gate (D23):** P2's gate as amended by D22, with A56 to A64; the pseudo-terminal cases on Linux and on this Mac.

*Exit criterion (2026-11-21, D23): no blocker, every row tested, plants caught.*

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
4. *(Added 2026-11-20, second records pass; MADR D12.)* From the second round on, with `discard()`, the loop calls
   `ls.keys.flushNext()`, so the terminal's queue is emptied after the repeated offer is written and before its read
   (row R79). `flush-offer`, `pty-flush-offer` and A24 pass after this step.
5. *(2026-11-21, F5, the owner, relayed by the coordinator: "Answer the repeated offer (Recommended)".)* **Files, added:**
   `internal/cli/consentwording_test.go`, its case "the first run" alone: after the failure's line it waits for the
   repeated offer and answers `n`, Enter, then sends `\x04` at the session prompt; its assertions are unchanged. It
   is the only existing test this phase changes.
6. *(2026-11-21, deviation on step 1, accepted by the same answer.)* `ls.signInRound(want, turn) (signedIn bool, err
   error)` in place of `ls.signInRound(want string) bool`: it also returns the after-steps' error, the `/models`
   turn's, which `ls.login` returns today and `firstRun` passes on to the session, so step 3's "nothing else changes"
   holds.

**Gate:** every P1 test passes, the loop's included; `go test -count=1 -timeout 2m ./internal/cli/` exit 0;
`make pre-add-check FILES="internal/cli/login.go"` exit 0.

**Stop if:** a loop test needs changing to pass; any test needs the harness's bound raised; the warning appears more
than once in any test.

*Exit criterion (2026-11-20): no blocker, every row tested, plants caught.*

### P3 — planted breaks

*Class: mid-tier; effort: medium. Mechanical, but each failure must be read, not counted.*

**Files:** none in the tree. Each mutation on a fresh scratch copy of the P2b tree (copied, not checked out), one at a
time, under the session's scratch directory, built from nothing with `GOPROXY=off`.

| # | Mutation | Must fail |
| --- | --- | --- |
| 1 | the owed `\n` not dropped | `TestLineEndsBeforeEachQuestion/CRLF/*-secret`, `TestFirstRunKeyOnlyProviderWaits/CRLF`; (2026-11-20, the owner's ruling at P1) `TestFirstRunSetup/CRLF` |
| 2 | `rawEnd` sets `lfOwed` | `TestEnterAfterSecretAnswersTheNextLine` |
| 3 | `\n` then `\r` folded as one end | `TestLFThenCRIsTwoEnds` |
| 4 | `ls.ask` makes a new `terminalInput` per ask | `TestFirstRunTypeAhead` |
| 5 | `discard` does nothing at the session prompt | `TestSignInTypeAheadNotSentAsPrompt` |
| 6 | `terminalInput` reads ahead after a line | `terminput_test`'s read count, `TestLineSessionReadsOnlyWhenAsked` |
| 7 | `rawEnd` leaves a `\n` already read | `TestPastedKeyEndingCRLF` |
| 8 | `askTrust` reads `e.stdin` again | `TestTrustReadsTerminalInput` |
| 9 | the loop never ends on a decline (a no at the repeated offer starts another round) | `TestFirstRunDeclineOnRepeatEnds`, `TestFirstRunRepeatedOfferDefaultsNo` |
| 10 | the loop never ends on closed input (`ended()` ignored~~, the end of input taken as a cancelled round~~) | ~~`TestFirstRunClosedInputEnds`, `TestFirstRunInstantFailureClosedInputEnds`, by the harness's bound~~ (2026-11-20, fourth pass) `TestFirstRunEndOfInputInRoundEnds` (A20) alone, by its count of the offer; the two tests struck here pass under it and move to mutation 22 |
| 11 | no repeat after a cancel (the loop ends on a cancelled round) | `TestFirstRunOfferAgainAfterKeyCancel`, `…AfterEmptyKey`, `…AfterMenuInterrupt` |
| 12 | no `discard` before a repeated offer | `TestFirstRunTypeAheadDroppedBeforeRepeatedOffer` |
| 13 | the repeated offer defaults to yes | `TestFirstRunRepeatedOfferDefaultsNo` |
| 14 | the warning printed between rounds | `TestFirstRunOfferAgainAfterKeyCancel` (warning counted once) |
| 15 | mutation 1, with `eachEnter` cut to `CR` alone | expected to pass: recorded, to show the `CRLF` form is what catches it |

Each run: `go test -count=1 -timeout 2m ./internal/cli/` (and `./internal/cli/editor/` for 6), output to a scratch
file, the exit status before any filter, the failing test's message quoted. ~~Mutation 10 is expected to spin until the
harness's bound; its CPU use ends with the test binary.~~ (2026-11-20, fourth pass) Mutation 22 is the one that spins until the harness's bound,
its CPU use ending with the test binary; mutation 10 fails A20 by its count, without spinning. Then the unmutated copy passes.

**Gate:** mutations 1 to 14 each fail the named test, with its message recorded; 15 passes; the restored copy passes.

**Stop if:** a mutation from 1 to 14 passes (that test checks nothing: back to P1 for its rule), or the tree is
changed.

**Additions, 2026-11-20 (records pass): the plants of the tests added to P1, P1b and P2, and of decided rows whose
tests had none (36 to 41).** Run as mutations 1 to 14 are. Mutation 1's must-fail list gains `TestLoginProviderMethodMenuWaits/CRLF` (A5); mutation 4's gains P1b's
typed-ahead case (A14), run by `make probe-conpty` on the Windows test laptop.

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 16 | the owed `\n` dropped only when it is already in `pending`, not when it arrives in a later read | `TestLineEndsBeforeEachQuestion/CR\|LF/*-secret` (A1), `TestCRLFSplitAcrossReads` | R4, R5 |
| 17 | `secret` empties `pending` when raw mode starts | `TestLineEndsBeforeEachQuestion/one chunk/*-secret` (A1) | R23 |
| 18 | `line` returns the partial text at the end of input | `TestLineEndOfInputMidLine` (A2) | R16 |
| 19 | `secret` reads a line when raw mode fails | `TestSecretRawFailureReadsNothing` (A3) | R28 |
| 20 | `secret` returns before restoring when its read fails | `TestSecretRestoredWhenContextEnds` (A4) | R30 |
| 21 | `terminalInput.Read` leaves a copy of what it gave in `pending` | `TestEditorKeepsKeysTypedWithLogin` (A6), `TestTerminalInputBytesInOnePlace` (A16), `TestAskOnKeysPastAnswerNotShared` (A17) | R56, R57, R58 |
| 22 | the loop takes the end of input at an offer as a cancelled round | `TestFirstRunEndOfInputAtOfferEnds` (A7), (2026-11-20, fourth pass) `TestFirstRunClosedInputEnds` and `TestFirstRunInstantFailureClosedInputEnds`, each by the harness's bound | R15, R68, R69 |
| 23 | an interrupt at a repeated offer is taken as a cancelled round | `TestFirstRunInterruptAtRepeatedOfferEnds` (A8) | R38 |
| 24 | a yes at a repeated offer ends the loop | `TestFirstRunYesOnRepeatStartsAnotherRound` (A9) | R65 |
| 25 | the first offer's default is no | `TestFirstRunEnterAtFirstOfferSignsIn` (A10) | R61 |
| 26 | the loop does not check `e.ctx` after a round | `TestFirstRunContextEndReturns` (A11) | R42 |
| 27 | the loop goes on after a successful round | `TestFirstRunSetup` (A12) | R71 |
| 28 | `acpLogin` asks `firstRunQuestion` again after a cancelled sign-in | `TestAcpLoginMethodMenuWaits` (A13) | R72 |
| 29 | `rawEnd` drops a `\n` after a `\n`, as after a `\r` | `terminput_test` row (i) (A15) | R9 |
| 30 | `Read` gives bytes without applying the rule | `terminput_test` row (ii) (A15) | R10 |
| 31 | a line ended by `\r` is held until the next byte arrives | `terminput_test` row (iii) (A15) | R11 |
| 32 | `closed` is set only for `io.EOF` | `terminput_test` row (iv) (A15) | R43 |
| 33 | `terminalInput` replaces bytes that are not valid UTF-8 | `terminput_test` row (v) (A15) | R44 |
| 34 | `auth.go:301` keeps `inline.NewReader(e.stdin)` | `TestOneReaderOfStdin` (A18) | R45 |
| 35 | `Read` takes `pending` without the lock | `TestTerminalInputConcurrentReads` (A19), under `-race` | R48 |
| 36 | `line` ends a line only at `\r` | the `LF` forms of `TestLineEndsBeforeEachQuestion` | R1, R32 |
| 37 | `line` ends a line only at `\n`, keeping a `\r` in the text | the `CR` forms of `TestLineEndsBeforeEachQuestion`, its `-input`, `-choose` and `-confirm` cases among them | R2, R19 |
| 38 | a no at the first offer starts a round | `TestFirstRunDeclineEnds` | R62 |
| 39 | an interrupt at the first offer starts a round | `TestFirstRunInterruptAtOfferEnds` | R63 |
| 40 | `closed` is never set | `terminput_test` "`ended()` true after the end of input" | R14 |
| 41 | `closed` is set for a context's error too | `terminput_test` "`ended()` false after an ended context" | R41 |

The gate and the stop rule above cover 16 to 41 as they cover 1 to 14.

**Additions, 2026-11-20 (second records pass).** Mutation 4's must-fail list loses P1b's typed-ahead case (A14), which
it cannot fail (P1b, A14's note). Mutation 10's gains `TestFirstRunEndOfInputInRoundEnds` (A20), the only test that
reaches the `ended()` check after a round; `TestFirstRunClosedInputEnds` pins row R68 through mutation 22 instead.
*(2026-11-20, fourth pass): mutation 10's must-fail is A20 alone, and both `TestFirstRunClosedInputEnds` and
`TestFirstRunInstantFailureClosedInputEnds` are in mutation 22's table row, as struck and amended above.)*
Mutations 19 and 21 run only as the owner's answers to Q18 and Q19 decide. The cases of P1b run on the Windows test
laptop or the Linux machine, through `run-on-host`, on the mutated scratch copy.

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 42 | the terminal's queue flushed before every question | `typeahead` (A14), `TestFlushPoints` (A24) | R22, R35, R77, R80, R81 |
| 43 | no flush before the session prompt | `flush-prompt`, `pty-flush-prompt` (A21), `TestFlushPoints` | R25, R51, R74, R76, R80 |
| 44 | no flush before a repeated offer | `flush-offer`, `pty-flush-offer` (A21), `TestFlushPoints` | R69, R74, R76, R80 |
| 45 | the repeated offer's flush runs before its text is written | `TestFlushPoints` (A24) | R79 |
| 46 | after a flush, a waiting read's completion is given to the prompt | `abandoned`, `pty-abandoned` (A21) | R83 (2026-11-21, D22 second draft: on POSIX nothing waits; must fail A28 over a gateless input, A42's stub case and Windows's `abandoned`) |
| 47 | the line of a prompt that gave up is given to the next taker | `terminput_test` abandoned row (A28), `abandoned`, `pty-abandoned` | R84, R85 (2026-11-21, D22 second draft: as plant 46: A28 gateless, A42's stub case, `abandoned`; on POSIX `none-lost` is the pass) |
| 48 | every read whose wait ended is dropped, a timed wait's included | `terminput_test` timed-wait row (A28); the editor's Esc tests | R87 |
| 49 | each prompt that gave up drops a completion of its own | `terminput_test` two-give-ups row (A28) | R88 |
| 50 | the end of input after a give-up is dropped too | `terminput_test` end-of-input row (A28), by the bound | R89 |
| 51 | `terminalIn()` makes a new value on each call | `TestTerminalInMadeOnce` (A22) | R46 |
| 52 | `askPermission` reads `e.stdin` directly | `TestPermissionReadsTerminalInput` (A23) | R60 |
| 53 | an answer is cut at 64 KiB | `TestLongAnswerHasNoBound` (A25) | R17 |
| 54 | `secret`'s restore not deferred (today's code) | `TestSecretRestoredOnPanic` (A26) | R31 |
| 55 | `askTrust` reads with `askOn`'s deadline contexts | `TestTrustReadsWithoutDeadline` (A27) | R50 |

The gate and the stop rule above cover 42 to 55 as they cover 1 to 14.

**Additions, 2026-11-20 (third records pass).** Mutations 19 and 21 now run (Q18 and Q19 answered).

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 56 | the mark is never cleared: every `ReadPrompt` flushes | `TestFlushPoints` (A24, as extended) | R105, R106, R107 |
| 57 | a sign-in round that was cancelled or failed sets no mark | `TestFlushPoints` | R103 |
| 58 | a first run that was declined sets no mark | `TestFlushPoints` | R104 |
| 59 | a failed flush warns at every point, not once per process | `TestFlushFailureWarnsOnce` (A29) | R78 |
| 60 | a read given up by `ls.ask`'s interrupt cancel is dropped too | `terminput_test` Ctrl+C give-up row (A28, as extended); `TestLoginInterrupt` | R86 |
| 61 | ~~the session flushes on its way out when the mark is set~~ (2026-11-21, MADR D21: the session flushes once more on its way out) | `TestFlushPoints` (A24's R108 cases, as amended 2026-11-21) | R108 |

The gate and the stop rule above cover 56 to 61 as they cover 1 to 14.

**Additions, 2026-11-20 (fourth records pass).**

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 62 | the warning's words changed (the consequence clause left out) | `TestFlushFailureWarnsOnce` (A29) | R101 |
| 63 | an abandoned completion is taken with the waiting prompt's buffer, not its own read's size | `terminput_test` 512-through-256 row (A28, extended) | R86, R102 |

The gate and the stop rule above cover 62 and 63 as they cover 1 to 14.

**Additions, 2026-11-20 (Deviation 1, MADR D19).**

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 64 | a console read that gives nothing after an interrupt is passed on as the end of input | `TestConsoleReadAgainAfterInterrupt` (A30); `menu-ctrlc`, `offer-ctrlc` (P1b) | R109, R121 |
| 65 | `watchSignals` sets no mark | `TestWatchSignalsMarksInterrupt` (A31) | R110 |
| 66 | the mark is never cleared | `TestConsoleCtrlZAfterInterruptEnds` (A32) | R112 |
| 67 | `watchSignals` treats every interrupt as taken once it sets the mark | `TestInterruptDuringTurn` (existing); `setup-menu-ctrlc` (P1b) | R113, R120 |
| 68 | the byte `0x03` is taken as an interrupt and the read made again | `TestConsoleRawCtrlCIsAByte` (A34) | R115 |
| 69 | the rule applied when standard input is not a console | `TestConsoleRuleOnlyOnAConsole` (A35) | R117 |
| 70 | the rule applied on every system | `pty-ctrlc-ctrld` (P1b, Linux), by the harness's bound | R118 |

Mutation 60 (a Ctrl+C give-up dropped too) must also fail `menu-ctrlc`'s `warned-once` (row R119). The gate and the
stop rule above cover 64 to 70 as they cover 1 to 14.

**Additions, 2026-11-20 (MADR D20).** Mutation 67's must-fail list gains `auth-paste-ctrlc` (P1b, row R120).

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 71 | the mark is cleared only by an empty read, not by a read that returns bytes | `TestConsoleMarkExcusesOneRead` (A36) | R111 |
| 72 | the mark is set whether or not a read is waiting | `TestConsoleNoMarkWithoutWaitingRead` (A37) | R114 |
| 73 | an empty read with no mark does not wait | `TestConsoleWaitsForLateMark` (A38) | R116 |
| 74 | the wait is 1 s, not 100 ms | `TestConsoleWaitBound` (A39) | R116, R122 |

The gate and the stop rule above cover 71 to 74 as they cover 1 to 14.

**Additions, 2026-11-21 (MADR D21).**

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 75 | the session prompt's flush made due at the editor's first read of the terminal, not made at the prompt (P2's build pass 1) | `TestFlushPoints` (A24's R123 case) | R103, R105, R123 |

The gate and the stop rule above cover 75 as they cover 1 to 14.

**Additions, 2026-11-21 (MADR D22, proposed).**

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 76 | the layer reads before readiness (a blocking read, as built) | `TestCancelableWait` (A41), `pty-abandoned` (A43) | R126, R130 (2026-11-21, D22 second draft: and `pty-gaveup-held` (A43, added), which does not depend on timing) |
| 77 | closing the gate does not wake the wait | `TestCancelableWait` (A41), by its 100 ms bound | R127 (2026-11-21, D22 second draft: observable through A41's seam reporting each return from the wait, since a close returns only after the waiter left select(2)) |
| 78 | closing the gate does not wait for a read in progress, or reports none | `TestCancelableWait` (A41), `TestCancelableGiveUp` (A42) | R127, R129 (2026-11-21, D22 second draft: staged through the read(2) seam) |
| 79 | `EINTR` ends the wait as an error | `TestCancelableWait` (A41); ~~`pty-menu-ctrlc-typed` (A44) on Linux~~ (2026-11-21, D22 second draft: A44 cannot catch it; A41's seam pins it) | R125, R133 |
| 80 | a give-up with nothing read marks a drop, as D13 did | `TestCancelableGiveUp` (A42); `pty-abandoned-empty-typed`, `-pasted` (A40) | R128 |
| 81 | a program's give-up gives a completion read before the close to the next prompt | `TestCancelableGiveUp` (A42) | R129 |
| 82 | a Ctrl+C give-up drops a completion read before the close | `TestCancelableGiveUp` (A42) | R129 |
| 83 | the layer used for a pipe | `TestTerminalInChoosesLayer` (A46) | R137 |
| 84 | the gate left open after a read returns | `TestCancelableWait` (A41): bytes written after a completed read are not read until the next open | R127 (2026-11-21, D22 second draft: aimed at A42 and A47, where `fill` closes the gate; ~~R127 now has the gate close itself after a read~~ (2026-11-21, D22 third draft: dropped)) |
| 85 | the gate's state read without its lock | `TestCancelableConcurrent` (A48), under `-race` | R142 (2026-11-21, D22 second draft: caught in P4 step 5, not in P2) |
| ~~86~~ | ~~the first prompt's flush made before raw mode is on (half a line typed into the abandoned prompt then survives to the prompt)~~ | ~~`pty-abandoned` (A43)~~ (2026-11-21, D22 second draft: dropped: M14 shows a flush out of raw mode empties half a line too, so it changes nothing) | R131 |

The gate and the stop rule above cover 76 to 86 as they cover 1 to 14; 17 and 18's plants come with Q23's and Q24's
answers.

**Additions, 2026-11-21, D22 second draft.**

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 87 | a read that gives 0 bytes returns `(0, nil)` | `TestCancelableWait` (A41, exactly `(0, io.EOF)`), A45 as moved | R134 |
| 88 | the layer returns an error, or `(0, nil)`, when the gate closes | `TestCancelableGiveUp` (A42) | R145 |
| 89 | `fill` asks `inline.Reader` with the caller's size | `TestCancelableGiveUp` (A42, the 512-then-256 case) | R147 (2026-11-21, D22 third draft: on the gated path; with no gate the caller's size is as built, plant 63) |
| 90 | with no gate, `waiting` is never set | A28 (existing); `TestCancelableGiveUp` (A42, gateless) | R151 |
| 91 | the wake pipe's write end blocking, or not drained before a wait | `TestCancelableWakePipe` (A54) | R146 (2026-11-21, D23: the credit fixed: A54 catches the blocking write end; the pipe not drained is caught by A42's closed-gate case, `TestCancelableGiveUp`) |

The gate and the stop rule above cover 87 to 91 as they cover 1 to 14. Plant 86 is dropped (M14); 85 is caught in P4.

**Additions, 2026-11-21, D22 third draft.**

* **A42, as amended again:** a stub layer giving `0x04`, then one giving `0x00`, to `readSecret`: cancelled, then
  ignored (R89); the gate closed with bytes ready, and the select seam sees no return of the wait over 200 ms (R145).
* **A45:** as its dated note says, two cases.
* **Gate (D22):** unchanged but for the above.

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 92 | the closed gate's wait includes the terminal | `TestCancelableGiveUp` (A42, the closed gate with bytes ready) | R145 |

The gate and the stop rule above cover 92 as they cover 1 to 14.

**Additions, 2026-11-21, D22 accepted.**

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 93 | the fallback warns at every cause, or its words changed | `TestCancelableFallbackWarnsOnce` (A50) | R138 |
| 94 | no flush at the program's give-up | `TestFlushAtGiveUp` (A49) | R80, R84, R132 |
| 95 | a Ctrl+C give-up flushes too | `TestFlushAtGiveUp` (A49) | R81, R132 |
| 96 | the layer reads standard input itself, not the private descriptor | `TestPrivateDescriptor` (A55); `TestCancelableWait` (A41, `EAGAIN`) | R126 |
| 97 | a failed wait ends the input in place of the fallback | `TestCancelableFallbackWarnsOnce` (A50) | R138 |

The gate and the stop rule above cover 93 to 97 as they cover 1 to 14.

**Additions, 2026-11-21, D23 (the fourth pass).**

| # | Mutation | Must fail | Row |
| --- | --- | --- | --- |
| 98 | the give-up flush made on any input with a flush (the review's plant A: no gate or system check) | `TestGiveUpFlushBySystem` (A56, the gateless input that is not a POSIX terminal) | R132, R143 |
| 99 | the `FD_SETSIZE` check turned off (the review's plant B) | `TestGatedRefusals` (A57) | R138 |
| 100 | the gate never wired, `return g, nil` (the review's plant D) | `TestTerminalInWiresGate` (A58), in the default suite | R124 |
| 101 | no process-group check before `/dev/tty` (the review's plant F) | `pty-not-controlling` (A59) | R126 |
| 102 | a read that ended on its context marked with the caller's size, not the size asked (the review's plant H) | `TestGatedRaceDropSize` (A61) | R147 |
| 103 | `EINTR` from read(2) taken as an error | `TestCancelableWait` (A62) | R153 |
| 104 | `NewGated` takes a blocking descriptor | `TestGatedRefusals` (A63) | R154 |
| 105 | no `Rdev` check after opening the path | `pty-same-terminal` (A64) | R155 |
| 106 | the warning written from the goroutine that fell back | `TestFallbackWarningOnMainFlow` (A60) | R156 |
| 107 | the give-up flush keyed on the gate, as built | `TestGiveUpFlushBySystem` (A56, the fallback cases) | R80, R157 |

The gate and the stop rule above cover 98 to 107 as they cover 1 to 14.

*Exit criterion (2026-11-20): no blocker, every row tested, plants caught.*

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
   *(2026-11-20, second records pass: and `run-on-host --need linux -- make probe-pty`; both runs read as P1b's
   amendment says, by their `0063 case=` lines only, every case as its "After P2b" column, run by an agent.)*
7. `run-heavy -- make check-test-isolation` (0053-MADR D4): the new tests read no host file above their temp root.
8. `make check-records`, `make markdownlint`, and the disclosure guard's `redact` over the changed files.

**Gate:** every command exit 0, with its summary line recorded.

**Stop if:** any gate fails. Record the file, line and failure, whether it fails on the base too, and stop; no gate is
loosened or skipped.

*Exit criterion (2026-11-20): no blocker, every row tested, plants caught.*

### P5 — docs

*Class: mid-tier; effort: medium. Prose to the repository's register, examples from gobble's own output.*

**Files:** `docs/guides/credentials.md`, `docs/architecture.md`, `README.md`, `docs/README.md`, this plan, the MADR's
status note.

*(2026-11-21, P2's build pass 1, Deviation 2; the owner, relayed by the coordinator: "Accept all four; add both docs to P5 (Recommended)".)* Files gain
`internal/cli/term/doc.go`: its package summary names the flush of a terminal's input (`FlushInput`) and the console
input with D19's rule (`ConsoleInput`). Step 2 also covers the `internal/cli/term` row of `docs/architecture.md`, for
the same two.

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

*Exit criterion (2026-11-20): no blocker, every row tested, plants caught.*

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

*Exit criterion (2026-11-20): no blocker, every row tested, plants caught.*

**Amendment, 2026-11-20 (second records pass; MADR D16).** Step 2 is superseded by the MADR's Confirmation item 5 as
amended. Its (A) list is agent-run, in P1 step 0b, P1b and P4 step 6, and is not repeated here. P6 is now:

1. **The build,** by an agent: as step 1, from P5's commit, placed on the Windows laptop at
   `%USERPROFILE%\gobble-check\gobble.exe`. The agent reports the commit and the file's SHA-256; nothing else.
2. **The owner's real-console confirmation,** the MADR's item 5 (B), steps 1 to 8: Console Host with PowerShell 5.1,
   then Windows Terminal with PowerShell 7. The owner reports one word per step, as each step says. No step needs a
   provider key or a sign-in; the key typed is made up and is never sent, and the proxy variables point at a closed
   local port.
3. **The record,** by the coordinator: each step's word per terminal, the PowerShell versions and the Windows build;
   never a key or screen text beyond what the owner reports.

**Gate (amended):** every step's word as item 5 (B) expects. **Stop (amended):** any other word. Step (B) 4 giving
"gobble ended" is row R40's named stop, which the agent-run `menu-ctrlc` should already have shown; if it did not,
the pseudo console and Console Host differ, and that difference is recorded in the MADR before the stop goes to the
owner.

**Then the plan stops.** The release is decided by the owner after this check has passed (MADR Q8); no release, tag or
`v0.6.1` work is part of this plan.

## Verification

* P1's red tests seen failing at the base, quoted; the defect's passing after P2 and the loop's after P2b.
* P1b's probe and gobble mode on the Windows test laptop, at the base and after P2b.
* P3's fourteen mutations each failing its test, quoted, and the fifteenth passing.
  *(2026-11-20: and mutations 16 to 41 each failing theirs; every decided row of the MADR's state table pinned by the
  test its row names.)* *(2026-11-20, second pass: and 42 to 55; P1b's cases by their `0063 case=` lines.)*
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
  *(2026-11-20: met. Q9 was answered (b) on 2026-11-19, recorded in the MADR's "Owner questions" and in the execution
  record below; no amendment of the steps is needed.)*
* `docs/guides/credentials.md`, `docs/architecture.md`, `README.md` and `docs/README.md` current.
* This plan's status `complete`; the release left to the owner's decision after P6 (Q8).

## Execution record

*Empty: nothing has been executed.*

*(2026-11-20: still true of the phases: none has started and nothing is built. The entries below are records only:
the owner's rulings, and a re-grounding of this plan's citations.)*

*(2026-11-20, later: P1 steps 0 and 0b have run; see "P1 step 0, pass 1" and "P1 step 0b, build pass 1" at the end.
Step 0b stopped on row R40: `probe-ctrlc` observed `eof`. Nothing after step 0b is built.)*

### Owner rulings (2026-11-19, recorded 2026-11-20)

Both relayed by the coordinator; each is quoted as the label of the option the owner chose.

* **Q9:** "The repeated offer is `[y/N]`: Enter ends the loop, `y` tries again (Recommended)". That is the MADR's Q9
  (b), the answer this plan was written for (P1 step 5, P2b step 2, P3 mutation 13, P5 step 1), so no step changes.
* **Approval of this plan:** "Approve the whole plan; I commit the records now and each phase as you hand it to me
  (Recommended)". Every phase, P1 to P6, is approved; each phase's commit is the owner's.
* **Status.** The front matter went from `proposed` to `in-progress`, with the date 2026-11-20. That follows the
  records skill (a plan is `proposed` until approved, `in-progress` while its phases land) and this repository's own
  precedent for an approved plan with no phase started: 0051-PLAN and 0052-PLAN went from `proposed` to
  `in-progress` at their approval, in commit `011b63a`, before any phase was built.

### Re-grounding note (2026-11-20, at `09a6adf`)

The plan's base, named in the MADR (`consulted`, "Evidence index", Confirmation item 1) and in `docs/README.md`, is
`21be074`. `main` is now `09a6adf`, 14 commits later. Between them, `prompt.go`, `login.go`, `auth.go`, `trust.go`,
`permission.go`, `internal/cli/editor/`, `login_test.go`, `line_test.go` and `cmd/gobble/` have no change at all;
`line.go` and `credentials.go` have one line changed each, and `main.go`'s `runEnv` gained fields (0036-PLAN P1 to
P4). P1 records the commit it runs at; this note does not change what "the base" means.

| Citation in this plan | At `09a6adf` |
| --- | --- |
| `login.go:95-97`, `terminalQuestions` over `inline.NewReader(e.stdin)` and `editor.Raw` | holds |
| `line.go:86`, `newLineSession`'s `raw` | holds |
| `trust.go:211`, `trustRaw` | holds |
| `login.go:226`, `ls.ask` makes `questions` over `ls.keys` | holds |
| `auth.go:301`, `questions` over `inline.NewReader(e.stdin)` | holds |
| `trust.go:184`, the trust question reads `e.stdin.Read` | holds |
| `permission.go:249`, `askPermission` takes `ls.keys` | holds |
| `login.go:300-316`, `firstRun` | holds (`firstRunQuestion` is `:295`, `signInWith` `:24`) |
| go-llmprovider-sdk `wizard/configure.go:794`, `Enter your %s API key` | holds; the pin is still `v1.31.0` |
| `golang.org/x/sys` `v0.49.0` (P1b) | holds, a direct requirement in `go.mod` |
| `Makefile`'s `probe-conhost` (P1b's neighbour) | holds (`:281`) |
| Tests and helpers named: `lineHarness`, `rawCheckedInput`, `asked`, `typedKeySignIn`, `keyLister`, `TestLineSessionReadsOnlyWhenAsked`, `TestLoginInterrupt`, `TestFirstRunSetup`, `TestLoginMenuAndRefusals`, `TestLoginAsksThroughQuestions`, `h.e.intr.take()` | all present (`line_test.go:25`, `:54`, `:194`; `login_test.go:40`, `:184`, `:203`, `:262`, `:285`, `:315`, `:415`) |
| `questions`' `pending` and `afterCR` (P2 step 4) | hold (`prompt.go:37-40`) |
| `editor.New(in *inline.Reader, …)` (P2 step 7) | holds (`internal/cli/editor/editor.go:146`) |
| `docs/guides/credentials.md`, "Signing in from a session", its `Claude (Anthropic) API key:` example and "Yes runs `/login`; no goes on as before" (P5 step 1) | hold (`:353`, `:366`, `:382`) |
| `README.md`, "Signing in where you are" (P5 step 3) | holds (`:120`, a list item) |
| `docs/architecture.md`'s `internal/cli` and `internal/cli/editor` rows (P5 step 2) | present; the `internal/cli` row was reworded since `21be074` by 0036's phases, so P5 edits it as it then stands |

Lines changed since `21be074` that this plan does not cite, but the MADR's evidence does: `line.go:138`, inside the
MADR's `line.go:137-141`, and `credentials.go:241`, inside its `credentials.go:240-243`, each now call
`e.chooseWith(…)` where they called `provider.ChooseWith(e.hasCredential, …)` (0036-PLAN P4: models.json's Chat
Completions servers are registered on the run's set before the choice). The first run's condition, `setup` at
`line.go:137`, is unchanged, so the offer is made in the same cases. The line numbers the MADR gives still hold.

*(2026-11-20: not all of them. `internal/cli/modelflags.go` also changed between `21be074` and `09a6adf` (`4b10e0b`,
five lines inserted at `:49`), so the MADR's `modelflags.go:82-92` is `:87-97`. Annotated in the MADR's evidence
index; see "Re-grounding at `b1994a0`" below.)*

### State table (2026-11-20 ruling: to be added before P1)

* The MADR `0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md` carries no state table, which the records
  skill requires before a MADR that changes behaviour is `accepted`.
* The owner, relayed by the coordinator, ruled on 2026-11-20: "Add the table before P1, when 0063 is released
  (Recommended)". The table is added in a records pass before P1 starts, when 0063 is released for building. Any row
  of it that is `open` goes to the owner in one prompt before P1.
* 0063 is not released for building. Nothing else about this plan or the MADR changes, and no phase has started.
  *(2026-11-20, later: the owner released 0063 the same day; the records pass below added the table. No phase has
  started.)*

### Records pass before P1 (2026-11-20)

On the owner's ruling "Add the table before P1, when 0063 is released (Recommended)", relayed by the coordinator.
Documents only: no code, no test, nothing run but the records check and markdownlint.

**What was added.** The MADR's "State table" (after Confirmation): 73 rows, R1 to R73; 61 `decided`, 6 `out of
scope` (R24, R36, R49, R54, R55, R73), 6 `open` (R17, R31, R40, R50, R52, R53), put to the owner as the MADR's Q10 to
Q14. In this plan: step 0 of P1 (re-ground at the phase's start commit, and no start with an open row); the additions
A1 to A19 in P1, P1b and P2; mutations 16 to 41 in P3; and each phase's exit criterion, "no blocker, every row tested,
plants caught".

**Decided rows and the tests that pin them.** "existing" is a test in the tree at `b1994a0`; "A*n*" an addition of
this pass; a mutation number is P3's.

| Row | Test | Phase | Plant |
| --- | --- | --- | --- |
| R1, R32 | `TestLineEndsBeforeEachQuestion/LF/*`; `terminput_test` table; the existing sign-in tests answered with each E (P1 step 2) | P1, P2 | 36 |
| R2 | `TestLineEndsBeforeEachQuestion/CR/*`; `terminput_test` table | P1, P2 | 37 |
| R3 | `TestLineEndsBeforeEachQuestion/CRLF/*-secret` | P1 | 1 |
| R4 | `TestCRLFSplitAcrossReads`; `terminput_test` "owed `\n` in a later read" | P1, P2 | 16 |
| R5 | `TestLineEndsBeforeEachQuestion/CR\|LF/*-secret` (A1) | P1 | 16 |
| R6 | `TestLFThenCRIsTwoEnds` | P1 | 3 |
| R7 | `TestEnterAfterSecretAnswersTheNextLine` | P1 | 2 |
| R8 | `TestPastedKeyEndingCRLF` | P1 | 7 |
| R9 | `terminput_test` row (i) (A15) | P2 | 29 |
| R10 | `terminput_test` "a byte that clears `lfOwed`" and row (ii) (A15) | P2 | 30 |
| R11 | `terminput_test` row (iii) (A15) | P2 | 31 |
| R12, R67 | `TestQuestionsChoose`, `TestQuestionsConfirmAndInput` (existing) | P2 step 4 | planted in 0025-PLAN P5 |
| R13 | `TestEmptyKeyCancels`; `TestFirstRunOfferAgainAfterEmptyKey` | P1 | 11 |
| R14 | `TestQuestionsChoose` (existing); `terminput_test` "`ended()` true after the end of input" | P2 | 40 |
| R15 | `TestFirstRunEndOfInputAtOfferEnds` (A7) | P1 | 22 |
| R16 | `TestQuestionsSecret` (existing); `TestLineEndOfInputMidLine` (A2) | P1 | 18 |
| R18 | the five `*Waits` tests, `CRLF`; P1b gobble mode; P6 steps 4, 8, 9, 12, 13 | P1, P1b, P6 | 1 |
| R19 | `TestLineEndsBeforeEachQuestion/*/*-input`, `-choose`, `-confirm` | P1 | 37 |
| R20 | `TestLoginNoArgumentMenuWaits`, its `/login gemini` case | P1 | none (see below) |
| R21 | `TestLoginProviderMethodMenuWaits` (A5) | P1 | 1 |
| R22 | `TestFirstRunTypeAhead` | P1 | 4 |
| R23 | `TestLineEndsBeforeEachQuestion/one chunk/*-secret` (A1) | P1 | 17 |
| R25, R51 | `TestFirstRunTypeAhead`; `TestSignInTypeAheadNotSentAsPrompt`; `terminput_test` "`discard` keeps `lfOwed`" | P1, P2 | 4, 5 |
| R26, R27, R29 | `TestLoginSlash`, `TestQuestionsConfirmAndInput`, `TestQuestionsSecret` (existing); P6 step 5 | P2 step 4, P6 | planted in 0025-PLAN P5 |
| R28 | `TestSecretRawFailureReadsNothing` (A3) | P1 | 19 |
| R30 | `TestSecretRestoredWhenContextEnds` (A4) | P1 | 20 |
| R33 | the owner's check in Console Host | P6 | none: a check on a real console |
| R34 | P1b probe and gobble mode; the owner's check in Windows Terminal | P1b, P6 | none: a measurement |
| R35 | P1b gobble mode, typed ahead (A14) | P1b | 4 |
| R37 | `TestFirstRunOfferAgainAfterMenuInterrupt`; `TestLoginInterrupt` (existing) | P1 | 11 |
| R38 | `TestFirstRunInterruptAtRepeatedOfferEnds` (A8) | P1 | 23 |
| R39 | `TestFirstRunOfferAgainAfterKeyCancel`; `TestQuestionsSecret` (existing) | P1 | 11 |
| R41 | `TestQuestionsPrompter` (existing); `terminput_test` "`ended()` false after an ended context" | P2 | 41 |
| R42 | `TestFirstRunContextEndReturns` (A11) | P1 | 26 |
| R43 | `terminput_test` row (iv) (A15) | P2 | 32 |
| R44 | `terminput_test` row (v) (A15) | P2 | 33 |
| R45 | `TestFirstRunTypeAhead`; `TestTrustReadsTerminalInput`; `TestOneReaderOfStdin` (A18) | P1, P2 | 4, 8, 34 |
| R46 | `TestTrustReadsTerminalInput`; the harness's `e.termIn` | P2 | 8 |
| R47 | `terminput_test` read count; `TestLineSessionReadsOnlyWhenAsked` (existing) | P2 | 6 |
| R48 | `TestTerminalInputConcurrentReads` (A19), under `-race` | P2, P4 | 35 |
| R56 | `TestEditorKeepsKeysTypedWithLogin` (A6) | P1 | 21 |
| R57 | `TestAskOnKeysPastAnswerNotShared` (A17) | P2 | 21 |
| R58 | `TestTerminalInputBytesInOnePlace` (A16) | P2 | 21 |
| R59 | `TestLogoutSlash` (existing) | P2 | planted in 0025-PLAN P5 |
| R60 | `TestPermissionPrompt`, `TestPermissionPromptOnceOnly` (existing) | P2 step 5 | none: the type change is checked by the compiler |
| R61 | `TestFirstRunEnterAtFirstOfferSignsIn` (A10) | P1 | 25 |
| R62 | `TestFirstRunDeclineEnds` | P1 | 38 |
| R63 | `TestFirstRunInterruptAtOfferEnds` | P1 | 39 |
| R64 | `TestFirstRunRepeatedOfferDefaultsNo` | P1 | 13 |
| R65 | `TestFirstRunYesOnRepeatStartsAnotherRound` (A9) | P1 | 24 |
| R66 | `TestFirstRunDeclineOnRepeatEnds` | P1 | 9 |
| R68 | `TestFirstRunClosedInputEnds` | P1 | 10 |
| R69 | `TestFirstRunTypeAheadDroppedBeforeRepeatedOffer`; `TestFirstRunInstantFailureClosedInputEnds` | P1 | 10, 12 |
| R70 | `TestFirstRunOfferAgainAfterFailure` | P1 | 14 |
| R71 | `TestFirstRunSetup`, its offer counted once (A12) | P1 | 27 |
| R72 | `TestLoginDoesNotLoop`; `TestSetupDoesNotLoop`; `TestAcpLoginMethodMenuWaits`'s cancel case (A13) | P1 | 28 |

Mutations 36 to 41 (in P3's additions) were added by this pass for decided rows whose tests had no plant.

**(a) Decided rows that had no test,** each now with an addition and its plant: R5 and R23 (A1), R16's line case
(A2), R28 (A3), R30 (A4), R21 (A5), R56 (A6), R15 (A7), R38 (A8), R65 (A9), R61 (A10), R42 (A11), R71's count (A12),
R72's `acp --login` case (A13), R35 (A14), R9, R10's entry points, R11, R43 and R44 (A15), R58 (A16), R57 (A17), R45's
`auth login` and `setup` readers (A18), R48 (A19). Rows whose test still has no plant: R20 (the `/login gemini`
guard: no single change reaches it that the CRLF tests do not already catch), R33 and R34 (a check and a measurement
on a real or pseudo console), and R60 (the permission prompt's reader is a type the compiler checks).

**(b) Tests in this plan that map to no row:** P3 mutation 15 (`eachEnter` cut to `CR`, expected to pass): it checks
that the `CRLF` form is what catches mutation 1, a property of the tests, not of a row. Every other test in P1 to P6
maps to a row above.

### Re-grounding at `b1994a0` (2026-11-20, records pass)

`main` is `b1994a0`, 17 commits after `21be074` and 4 after `09a6adf`. Since `09a6adf` only test files changed under
`internal/` (`internal/cli/chatserver_test.go`, `modelscmd_test.go`, `refusal_test.go`; `acpserver/warninglog_test.go`),
none of them cited. The feature commits `7ee9a94` (image prompts in `acpserver`) and `9332cef` (images from `read`
and `@path`) and the test commit `93ecb5e` add or change no reader of standard input.

* Every citation in this plan's re-grounding table at `09a6adf` holds at `b1994a0`, by the same lines.
* The MADR's citations hold, except `modelflags.go:82-92`, now `:87-97` (annotated in the MADR and above).
* **The MADR's list of readers is not complete.** D1 is right that only `line.go:163`, `login.go:96` and `auth.go:301`
  call `inline.NewReader` on stdin, and it lists the trust question's direct read and `gobble acp --login`'s
  `ReadPassword` and `bufio` read. It does not list: `ReadPassword` reached from `gobble --api-key -` in the line
  session's process (`chat.go:64-66`, after the trust question at `:49-50`) and from `gobble auth login --api-key -`
  (`auth.go:122`); `gobble auth import --from-pi`'s questions (`piimport.go:61-94`); `gobble update`'s confirmation
  (`update.go:143`, `:166-181`, through go-selfupdate-lib `v1.26.3`); and the external editor that inherits stdin
  (`slash.go:226-230`, `config.go:284-291`). Each can share the input or is another process, so the decision holds;
  which of them join it is the MADR's Q10 (rows R52, R53). No reader was added after the plan's base.
  *(2026-11-20, second pass: Q10 answered; they stay outside it, rows R91 to R96.)*

### Second records pass (2026-11-20)

On a record review's findings and the owner's rulings of 2026-11-20, relayed by the coordinator: the review's cases 1
to 3 and Q10 to Q14 (MADR D12 to D15), and who runs the live checks (MADR D16). Documents only; nothing run but the
records check and markdownlint. Each finding was checked at its cited lines in the tree and the module cache before it
was applied; every statement the review makes about a Windows console is inferred from source, and the record keeps
it marked so.

**The review's findings, each checked:**

* Case 1 (input the terminal holds, never seen by `discard()`): holds; ctrl+j submits (go-tui-lib
  `lineedit/keys.go:93`). Applied: D12, rows R74 to R83.
* Case 2 (the read a browser sign-in leaves waiting): holds (go-llmprovider-sdk `wizard/auth.go:916-962`,
  `llm/provider/signin.go:441-446`, `inline/reader.go:59-70`). Applied: D13, rows R84 to R89; what the ruling's words
  leave is Q16.
* Case 3 (Ctrl+Z on a Windows console): holds by reading (`fd_windows.go:710-715`, `os/file_windows.go:110`,
  `inline/reader.go:72-73`, `:90-91`; `prompt.go:287` on POSIX). Applied: D14, row R90, R44 annotated.
* The suggested exclusions: applied, rows R97 to R100; R54's reason corrected.
* R28, R57: hold; both are open again (Q18, Q19). R44: holds; corrected, still decided, from D3 and D6, with R90.
* R35, A14 and mutation 4: holds; A14's expectation corrected, its plant now 42. R68, R69 and mutation 10: holds;
  A20 added, R68 pinned by 22. R25, R51, R69 on a pipe only: holds; A21. R46: holds; A22. R44's stubs: R90's probes.
  R60: holds; A23.

**Decided rows added or changed, and the tests that pin them** (the rest as the first pass's table above, except R28
and R57, open again, whose A3 and A17 wait):

| Row | Test | Phase | Plant |
| --- | --- | --- | --- |
| R17 | `TestLongAnswerHasNoBound` (A25) | P2 | 53 |
| R22, R35 | `typeahead` (A14, corrected); `probe-typeahead` measures R35 | P1 step 0b, P1b | 42 |
| R25, R51 | `flush-prompt`, `pty-flush-prompt` (A21), beside their pipe tests | P1b | 43 |
| R31 | `TestSecretRestoredOnPanic` (A26) | P1, passing after P2 | 54 |
| R46 | `TestTerminalInMadeOnce` (A22) | P2 | 51 |
| R50 | `TestTrustReadsWithoutDeadline` (A27) | P2 | 55 |
| R60 | `TestPermissionReadsTerminalInput` (A23) | P2 | 52 |
| R68 | `TestFirstRunClosedInputEnds` | P1 | 22 |
| R69 | `TestFirstRunEndOfInputInRoundEnds` (A20); `flush-offer`, `pty-flush-offer` | P1, P1b | 10, 44 |
| R74, R76 | `pty-flush-prompt`, `pty-flush-offer`; `flush-prompt`, `flush-offer` (A21) | P1b | 43, 44 |
| R75 | the owner's step (B) 5 | P6 | none: a real console |
| R77, R80, R81 | `TestFlushPoints` (A24) | P2 | 42, 43, 44 |
| R79 | `TestFlushPoints` (A24), `flush-offer` | P2, P1b | 45 |
| R83 | `abandoned`, `pty-abandoned` (A21) | P1b | 46 |
| R84 | `terminput_test` abandoned row (A28); `pty-abandoned` | P2, P1b | 47 |
| R85 | `abandoned` (A21): the pseudo-console test only | P1b | 47 |
| R87 | `terminput_test` timed-wait row (A28); the editor's Esc tests | P2 | 48 |
| R88 | `terminput_test` two-give-ups row (A28) | P2 | 49 |
| R89 | `terminput_test` end-of-input row (A28) | P2 | 50 |
| R90 | `probe-ctrlz-line`, `probe-ctrlz-raw`, `key-ctrlz`; the owner's step (B) 6 | P1 step 0b, P1b, P6 | none: the runtime's behaviour; another word is a deviation |

**Not decided, and where each is settled:** R40 awaits live evidence, measured first by `probe-ctrlc` at P1 step 0b
(before any code), then `menu-ctrlc` in P1b and the owner's step (B) 4 in P6. R28, R57, R78, R82 and R86 are the MADR's
Q18, Q19, Q17, Q20 and Q16, which P1 step 0 requires answered.

**Re-check of the rows this pass added or changed, against the review's three questions.** Missing cases: three
found and put as Q16 (a read the user gave up with Ctrl+C, and completions that are not lines), Q17 (a failing flush)
and Q20 (a flush before every prompt would drop typing during a turn); the permission prompt's typed-ahead keys are
recorded in R81 as outside this record. Rows decided only by a ruling's or D8's words: R79 rests on D8 holding "as
written", which the ruling says; R81 on the ruling naming two points; R87 to R89 on D13's "a read belonging to an
abandoned prompt" and on D8. Tests that cannot pin: R75 is pinned by the owner's step only, R85 by the pseudo console
only, and R90 by measurement only; each says so in its row.

**Who runs what (MADR D16).** Agent-run: P1 step 0b's probes, P1b's cases, P3's mutations of them, and P4 step 6, all
through `run-on-host`, read by their `0063 case=` lines. Owner-only: the MADR's item 5 (B), eight steps in two
windows, because a pseudo console does not show what a classic console window does with real key events, or how
PowerShell 5.1 and 7 leave the console's mode. No check needs a provider key, a sign-in or a browser.

### Third records pass (2026-11-20)

The owner answered Q16 to Q20, relayed by the coordinator (MADR D17): "Only reads the program gave up (Recommended)",
"Go on, with one warning on stderr (Recommended)", "Keep that, recorded and tested (Recommended)", "Dropped, as today
(Recommended)", "Only the first prompt after a sign-in or the first run (Recommended)". Documents only.

* Rows R28, R57, R78, R82 and R86 are decided; rows R101 to R108 added: R101 the warning's words, R102 Q16's stated
  consequence, R103 to R108 the mark of Q20. Every case of the mark is settled by the ruling's words or by D4.
* Final counts: 108 rows; R52 and R53 replaced; of the other 106, 88 `decided`, 16 `out of scope`, R40 awaiting live
  evidence (measured first at P1 step 0b), and R101 `open` for its words only.
* Tests: A3 and A17 run as first written; A24 and A28 extended; A29 added; plants 56 to 61.
* What waits for what: P1 step 0b runs before step 0's check and before R40 is settled; step 1 waits for step 0 and
  step 0b; R101 blocks only P2 step 9's message and A29.

### Fourth records pass (2026-11-20)

On the owner's approval of R101's words, relayed by the coordinator ("Plain words, with the consequence
(Recommended)"), and an independent check of the third pass, each item verified at its cited lines first.

* **Mutation 10 (rows R68, R69):** its must-fail is now A20 alone; `TestFirstRunClosedInputEnds` and
  `TestFirstRunInstantFailureClosedInputEnds` move to mutation 22, which is the one that spins to the bound; the spin
  sentence is struck and corrected (P3).
* **R40's probe:** verified: with no `signal.Notify`, Go's console handler lets the default one end the process
  (`runtime/os_windows.go:1047-1069`, inferred for a console). The probe child now takes `os.Interrupt`, as gobble does
  (`main.go:54`), records it, and each observed word has one meaning; `killed` is a fault in the probe (P1b, step 0b).
* **Stale check references:** R18, R27, R33 and R34 now name the cases and the owner's (B) steps.
* **D12's plumbing:** verified: D1's struct and `terminalIn()` carry no file or writer. MADR D18 places
  `term.FlushInput` in `internal/cli/term` (four per-OS files), binds it in `terminalIn()` with the warning writer, and
  P2 gains step 12 and amended steps 1 and 3. `internal/archtest` (`internal/archtest/check.go`) has no rule on
  `golang.org/x/sys`, and the package already imports it, so there is no stop; P2's `make archtest` gate confirms it.
* **Pass-2 text against D17:** D12, D13 and the owner's step (B) 5 each carry a dated "narrowed by D17" note.
* **The whole drop:** verified (`prompt.go:201`, `permission.go:206`, `inline/reader.go:75-77`); step 10 takes an
  abandoned completion with its own read's size; A28 gains the 512-through-256 case; mutation 63.
* **Counts:** 108 rows; R52 and R53 replaced; of the other 106, 89 `decided`, 16 `out of scope`, R40 awaiting live
  evidence, none `open`.

### P1 step 0, pass 1: re-grounding at `fa35986` (2026-11-20)

On the branch `feature/0063-shared-terminal-input`, created from `main` at `fa35986`, named by the owner. Read-only:
no code, nothing run but reading. Between `b1994a0` (the last re-grounding) and `fa35986` the only commit is `fa35986`
itself, which changes this plan, the MADR and `docs/README.md` and nothing else; so no cited code line can have moved.
Each citation was read again at `fa35986` all the same.

| Citations | At `fa35986` |
| --- | --- |
| The MADR's evidence index: `prompt.go:30-41`, `:68-75`, `:178-197`, `:199-208`, `:251-294` (`:272-274`); `login.go:74-90`, `:95-97`, `:212-231` (`:226`), `:252-255`, `:300-316`, `:323-336`; `credentials.go:204-232`, `:240-243`; `modelflags.go:87-97` (as annotated); `line.go:137-141`, `:163-171`; `acp.go:99-114`, `:117-125`, `:138-151`; `auth.go:301`; `trust.go:180-184`; `permission.go:185-230`, `:248-274`; `editor/editor.go:105-146` | hold |
| D1 to D18, Confirmation and the state table: `prompt.go:201`, `:237-244`, `:287`; `permission.go:206`, `:249`, `:257-263`; `trust.go:164-167`, `:184`, `:211`; `line.go:86`, `:137`, `:224-228`; `line_test.go:126`; `login.go:24`, `:100`, `:205-225`, `:262`, `:295`, `:307-310`, `:325`; `auth.go:122`, `:310`; `chat.go:49-50`, `:64-66`; `credentials.go:213`, `:222`; `main.go:54` (`term.ShutdownSignals`), `:175-184`; `term/console_windows.go:11`; `go.mod:24` | hold |
| Q10 to Q14 and rows R36, R54, R91 to R96: `piimport.go:61-94`, `:78`; `update.go:143`, `:166-181`; `slash.go:226-230`; `config.go:284-291`; `browser.go:14-24`; `tool/builtin/shell.go:85`; `input.go:53-56`; `mode.go:23-24`; `acp.go:49-53`, `:145`; `llm/provider/signin.go:441-446` | hold |
| This plan's tables: `Makefile`'s `probe-conhost` (`:281`); `lineHarness`, `rawCheckedInput`, `TestLineSessionReadsOnlyWhenAsked` (`line_test.go:25`, `:54`, `:194`); `asked`, `keyLister`, `typedKeySignIn`, `TestLoginInterrupt`, `TestLoginMenuAndRefusals`, `TestLoginAsksThroughQuestions`, `TestFirstRunSetup` (`login_test.go:40`, `:184`, `:203`, `:262`, `:285`, `:315`, `:415`); `h.e.intr.take()` | hold |
| Pins and their lines: go-llmprovider-sdk `v1.31.0` (`wizard/configure.go:463`, `:794`; `wizard/auth.go:528`, `:916-962`); go-tui-lib `v2.12.1` (`inline/reader.go:9-21`, `:49-78`, `:72-73`, `:75-77`, `:90-91`; `termkey/input.go:35-41`; `lineedit/keys.go:93`); `golang.org/x/term` `v0.47.0` (`terminal.go:849-851`); `golang.org/x/sys` `v0.49.0`; Go 1.27.4 (`internal/poll/fd_windows.go:653-720`, `:672`, `:710-715`; `os/file_windows.go:110`; `runtime/os_windows.go:1047-1069`) | hold |

* **Moved:** none; no annotation was needed in either record.
* **Behaviour or readers:** no change. `inline.NewReader` is called on stdin at `login.go:96`, `auth.go:301` and
  `line.go:163` only; `ReadPassword` at `credentials.go:213` and the trust question's `e.stdin.Read` at `trust.go:184`,
  as D1 and rows R91 to R96 record. No new reader of the terminal.
* **State table:** 89 `decided`, 16 `out of scope`, R52 and R53 replaced, R40 awaiting live evidence; no row `open`.

### P1 step 0b, build pass 1 (2026-11-20)

**What was built** (uncommitted, unstaged, on the branch at `fa35986`): `cmd/gobble/conpty_windows_test.go`
(`//go:build windows && conpty`): the pseudo-console parent (`CreatePseudoConsole`, `PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE`)
and the probe child mode only, with the five probe cases; and the `Makefile` target `probe-conpty`, beside
`probe-conhost`, outside `preflight` and CI, where `PROBE=1` runs `^TestConptyProbe$` alone. The gobble-mode cases,
`internal/cli/conpty_windows_test.go` and the Linux file are P1b's and were not written. The parent drains the pseudo
console's output unread and prints one `0063 case=` line per case.

Three details the plan's words do not name, none changing what a case measures:

* The child is started with `STARTF_USESTDHANDLES` and no handles, so it takes the pseudo console's standard handles,
  not the parent's pipes under ssh; if its stdin is still not a console it reads `CONIN$`, the same console input.
* Before `signal.Notify`, the child calls `SetConsoleCtrlHandler(NULL, FALSE)`, which undoes an ignore-Ctrl+C state
  inherited from the process that started it (as a process group's first process has); a program started from a shell
  in a console window has no such state.
* `probe-typeahead` types one extra Enter only if the second read is still waiting after 5 s (a first read that took
  both lines); the case then reads `two-lines-in-one-read`. It was not needed: the run below gave one line per read.

**Builds and lints, on the session host:** `GOOS=windows go vet -tags conpty ./cmd/gobble/` exit 0;
`GOOS=windows go test -tags conpty -c -o /dev/null ./cmd/gobble/` exit 0; `gofmt -l cmd/gobble/` exit 0, no file;
`GOOS=windows golangci-lint run --config .golangci.yml --build-tags conpty ./cmd/gobble/` (v2.15.0) exit 0,
`0 issues.` (a first run reported one `modernize` `embedlit` finding, fixed before the runs below).

**The harness seen to fail first,** on a scratch clone of the branch with the two files copied in, never the tree: the
child's line `log.line("read " + word + " " + strconv.Quote(string(buf[:n])))` was changed, for `probe-crlf` only, to
quote the read with its trailing `\n` cut. Run from the clone's root as
`run-on-host --need windows --label 0063-probe -- make probe-conpty PROBE=1`, on the Windows test laptop: exit 2, and

```text
0063 case=probe-crlf result=fail observed=cr
```

The clone was then restored (the file compared equal to the tree's). That run's other four lines matched the run
below, `probe-ctrlc` among them (`observed=eof`).

**The run,** from the repository root of the branch, the tree at `fa35986` with the probe file and the `Makefile`
change uncommitted: `run-on-host --need windows --label 0063-probe -- make probe-conpty PROBE=1`. It reached the
Windows test laptop (copied to `<scratch>` there). Exit status 2. The five lines, verbatim:

```text
0063 case=probe-crlf result=pass observed=crlf
0063 case=probe-typeahead result=pass observed=one-line-per-read
0063 case=probe-ctrlc result=fail observed=eof
0063 case=probe-ctrlz-line result=pass observed=eof
0063 case=probe-ctrlz-raw result=pass observed=eof
```

**What each word means, per step 0b:**

* `probe-crlf` `crlf`: a line-mode read of `1`, Enter on a pseudo console gives `1\r\n`. MADR D6's third assumption,
  first half, holds there (row R34).
* `probe-typeahead` `one-line-per-read`: two lines typed before two line-mode reads come one per read; the second stays
  in the console's buffer (row R35).
* `probe-ctrlz-line`, `probe-ctrlz-raw` `eof`: Ctrl+Z is the end of input in both modes, as D14 inferred (row R90).
* `probe-ctrlc` `eof`: the child recorded `signal` (the interrupt reached a handler taken as gobble takes it), and the
  line-mode read that was waiting returned nothing, the end of input. **This is row R40's named stop.** Measured twice,
  in the plant run and in this run. Nothing was changed to make it pass. Under step 0b the coordinator puts row R40's
  three options to the owner, and nothing further is built (P1 step 1 included) until the answer is recorded here and
  in the MADR.

### Deviation 1 (2026-11-20, P1 step 0b): Ctrl+C at a line-mode read on a Windows console ends the input

* **Found.** P1 step 0b's run on the Windows test laptop, as recorded in "P1 step 0b, build pass 1" above, gave
  `0063 case=probe-ctrlc result=fail observed=eof`, in the plant run and in the real run: after the probe child took
  the interrupt (it recorded `signal`), the line-mode read that was waiting returned nothing, which Go reports as the
  end of input. The other four probes gave the expected words (`crlf`, `one-line-per-read`, `eof`, `eof`). This is row
  R40's named stop. With no change, `inline.Reader` would keep that end of input for good and Ctrl+C at a menu would
  close the input for the process on Windows.
* **Ruling.** The owner, relayed by the coordinator: "(a) After an interrupt gobble took, read again (Recommended)",
  row R40's option (a). Recorded in the MADR as D19, with rows R109 to R121 and R40 `decided`; two points of the new
  mechanism are open there as Q21 (rows R111, R114) and Q22 (row R116).
* **Where the code lands.** P2, with the shared input, as its steps 13 and 14: `internal/cli/term/consolein.go`,
  `consolein_windows.go`, `consolein_other.go`, `consolein_test.go` (new); `internal/cli/signal.go` and
  `internal/cli/main.go` (changed); `internal/cli/signal_test.go` (A31). P1b gains the cases `offer-ctrlc`,
  `setup-menu-ctrlc` and `pty-ctrlc-ctrld`, and `probe-ctrlc`'s expected word becomes `eof` in
  `cmd/gobble/conpty_windows_test.go`. P3 gains mutations 64 to 70. `golang.org/x/sys/windows` is already imported by
  `internal/cli/term`; no new module, no archtest rule touched (P2's `make archtest` gate confirms it).
* **Gate.** Step 0b's gate is amended so `probe-ctrlc`'s `eof`, the console's behaviour, no longer stops the build;
  `menu-ctrlc` is the case that must show `offer-again` after P2. *(2026-11-21: after P2b, which brings the loop; on
  the P2 tree it must not give `ended`. Corrected under P1b's amendment.)*
* **What waits.** P1 step 1 waits for step 0: no row `open`, so for the owner's answers to Q21 and Q22.
  *(2026-11-20: answered, MADR D20; with R120 widened to `gobble auth login`'s paste prompt. Rows R111, R114 and
  R116 are decided and R122 added; P2 gains A36 to A39, P1b the `first=` word and `auth-paste-ctrlc`, P3 mutations 71
  to 74. No row is `open`; step 1 waits only for step 0's re-check.)*

### P1 and P1b, build pass 1 (2026-11-20): P1 built and run at the base; stopped on a guard red at the base

**Step 0, re-checked at `473a29e`** (the records commit after `fa35986`): it changes `docs/README.md`, this plan and the
MADR only, so every citation of "P1 step 0, pass 1" still holds. The MADR's state table: 104 `decided`, 16 `out of
scope`, R52 and R53 replaced, none `open`, none awaiting live evidence.

**The base.** P1's tests run on the tree itself: P2's code does not exist, so the tree at `473a29e` plus P1's test files
and step 1's seam is the base. Nothing was reverted to show a test red.

**What was built** (uncommitted, unstaged):

* Step 1: `termRaw` in `internal/cli/login.go` (`var termRaw = editor.Raw`; a literal first, shortened as the linter
  asked), used by `terminalQuestions`. `line.go:86` was left as it is: the coordinator's boundary names the seam in
  `login.go` alone, and the line harness sets `ls.raw` directly, so no test needs it there. A difference from step 1's
  words ("used by `terminalQuestions` and `newLineSession`"), for the owner to confirm or amend.
* Step 2 (`line_test.go`, `login_test.go`): `lineHarness.enter` and `noAnswer`, the first-run answer `"n" + enter`
  unless `answer` is set; `eachEnter`; `recordingPrompter`, which records each `Secret` and then ends the real
  `provider.SignIn` with `errKeyRecorded` so nothing is listed or saved; `offerCount` and `warningCount`;
  `TestFirstRunSetup`, `TestLoginMenuAndRefusals` and `TestLoginAsksThroughQuestions` over `eachEnter`.
* Steps 3 to 5 and A1 to A13, A20, A26: `internal/cli/lineend_test.go`, `internal/cli/firstrun_test.go` (new), and
  `login_test.go`. A10 ends by closing the input, not Ctrl+D, so it ends at every phase.
* Docs: `make probe-conpty` added to `docs/guides/developing.md`'s "Suites outside the default run" and to
  `docs/architecture.md`'s sentence on the console checks. `make probe-pty` does not exist yet and is not described.

**Run at the base** (session host, one package): `go test -count=1 -timeout 8m -v ./internal/cli/`, exit 1, 54 leaf
tests failing, the same 54 before and after the lint fixes; no test reached the network or left a listing unowned.
Each test, its kind, and its red line at the base:

| Test | Kind | At the base |
| --- | --- | --- |
| `TestLineEndsBeforeEachQuestion` (with A1's `CR\|LF` and `one chunk` forms) | red for the defect where K2 is the secret; guards otherwise | red: `CRLF/*-secret`, `CR\|LF/*-secret`, `one_chunk_CRLF/*-secret`, nine subtests, each `secret = "", <nil>; want "the-key"`; the other 75 pass |
| `TestEnterAfterSecretAnswersTheNextLine` | red for the defect | `input = "", cancelled (EOF); want "" at once` |
| `TestCRLFSplitAcrossReads`, `TestPastedKeyEndingCRLF`, `TestLFThenCRIsTwoEnds` | guards | pass |
| `TestLineEndOfInputMidLine` (A2), `TestSecretRawFailureReadsNothing` (A3), `TestSecretRestoredWhenContextEnds` (A4) | guards | pass |
| `TestSecretRestoredOnPanic` (A26) | red at the base | `raw mode entered 1, restored 0; want once each` |
| `TestFirstRunKeyOnlyProviderWaits`, `TestFirstRunMethodMenuProviderWaits`, `TestLoginNoArgumentMenuWaits`, `TestLoginProviderMethodMenuWaits` (A5), `TestSetupKeyWaits`, `TestAcpLoginMethodMenuWaits` | red for the defect, `CRLF` | each `CRLF` subtest: `secret = "", cancelled; want "sk-test-typed"`; `CR`, `LF`, `/named` and `/cancel` (A13) pass |
| `TestFirstRunTypeAhead` | red for the defect, every form | `CR`, `LF`, `CRLF`: `condition not met within 5 s` (the key prompt never shown: the menu waits) |
| `TestSignInTypeAheadNotSentAsPrompt`, `TestEmptyKeyCancels`, `TestEditorKeepsKeysTypedWithLogin` (A6) | guards | pass |
| `TestFirstRunOfferAgainAfterKeyCancel`, `…AfterEmptyKey`, `…AfterMenuInterrupt`, `…AfterFailure` (refused, offline), `TestFirstRunRepeatedOfferDefaultsNo`, `TestFirstRunDeclineOnRepeatEnds`, `TestFirstRunTypeAheadDroppedBeforeRepeatedOffer`, `TestFirstRunInterruptAtRepeatedOfferEnds` (A8), `TestFirstRunYesOnRepeatStartsAnotherRound` (A9) | red for the loop | every subtest: `the offer again, [y/N]: not before the session's prompt` |
| `TestFirstRunEndOfInputInRoundEnds` (A20) | red for the loop | every subtest: `the warning was printed 0 times; want 1` |
| `TestFirstRunDeclineEnds`, `TestFirstRunInterruptAtOfferEnds`, `TestFirstRunClosedInputEnds`, `TestFirstRunInstantFailureClosedInputEnds`, `TestLoginDoesNotLoop`, `TestSetupDoesNotLoop`, `TestFirstRunEndOfInputAtOfferEnds` (A7), `TestFirstRunEnterAtFirstOfferSignsIn` (A10), `TestFirstRunContextEndReturns` (A11) | guards | pass |
| `TestLoginMenuAndRefusals`, `TestLoginAsksThroughQuestions` (step 2, each E) | existing | pass |
| `TestFirstRunSetup` (step 2, each E; A12's count) | existing; A12 a guard | **`CRLF` red**: `login_test.go:444: condition not met within 5 s`, the wait for `claude/claude-x1` after the key; `CR` and `LF` pass, A12's count holds in them |
| `TestLineSessionReadsOnlyWhenAsked` | existing | pass |

**The stop.** `TestFirstRunSetup/CRLF` is red at the base. Answered with CRLF, it is the report's own path: the menu
answer's `\n` reaches the key prompt, which cancels before the key is pressed, as `TestFirstRunKeyOnlyProviderWaits/CRLF`
records (`secret = "", cancelled`). Step 6's gate says "every guard and every existing test passes", and A12 calls
the test a guard; its stop rule says a guard red at the base is a finding put to the owner. Not pre-existing: CRLF was
not in the test before step 2. Nothing was changed to make it pass. Resolutions, for the owner:

1. *(Recommended.)* Record `TestFirstRunSetup/CRLF` as red for the defect, passing after P2, with A12's count a guard
   in `CR` and `LF` and checked in `CRLF` from P2 on; mutation 1's must-fail list gains it. Cost: a dated amendment of
   step 2, A12 and row R71's test cell; no code.
2. Leave `TestFirstRunSetup` at `CR` and `LF`, the `CRLF` path being `TestFirstRunKeyOnlyProviderWaits`'s. Cost: step
   2's "with each form" is narrowed for one test: less coverage, close to a workaround.
3. Split the test: its existing body over `CR` and `LF` as the guard, and a `CRLF` twin named red for the defect. Cost:
   as 1, with one more test name in the records.

Doing nothing leaves step 6's gate failed, so P1 cannot close.

~~**Not done in this pass, and why:** P1b (the gobble-mode cases in `cmd/gobble`, `internal/cli/conpty_windows_test.go`,
`internal/cli/pty_linux_test.go`, `make probe-pty`), `probe-ctrlc`'s `first=` word and the re-run of step 0b's five
probes, and the `run-on-host` runs of `make probe-conpty` and `make probe-pty`: all wait for the owner's answer to the
stop above.~~ *(2026-11-20: done after the owner's answer, below, in the same pass.)*

**Gates, on the session host:** `gofmt -l internal/cli/` exit 0, no file; `go vet ./internal/cli/` exit 0;
`golangci-lint run --config .golangci.yml ./internal/cli/` (v2.15.0) exit 0, `0 issues.` (a first run found 17:
`unlambda` on the seam, `perfsprint` in the tests, and `confusing-naming` on three `testX` helpers, renamed; fixed,
and the package run again with the same 54 failures); `npx markdownlint-cli2@0.23.2` on the two docs exit 0.

**The owner's answers (2026-11-20), relayed by the coordinator; the same build pass goes on.**

* `TestFirstRunSetup/CRLF`: "Red for the defect in CR+LF; guard in CR and LF (Recommended)". Recorded as dated
  amendments of step 2, A12 and mutation 1's must-fail list here, and of row R71 in the MADR.
* The seam: "`login.go` only; amend step 1 (Recommended)". Recorded as a dated note on step 1.

**P1's gate, as amended:** met. Every test marked red fails at the base for the reason given (the table above, with
`TestFirstRunSetup/CRLF` now red for the defect); every guard and every other existing test passes; `go vet
./internal/cli/` exit 0. P1 has no blocker; its plants are P3's.

**P1b, built** (uncommitted, unstaged):

* `cmd/gobble/conpty_windows_test.go`: `probe-ctrlc` expects `eof` (Deviation 1) and its line gains `first=` (D20); the
  pseudo console's screen is kept in memory for the cases and never printed; gobble mode (`TestConptyGobble`), which
  runs gobble's main in a pseudo console per case, with `GOBBLE_HOME` a temporary directory, every credential
  variable empty and the proxy variables naming a closed local port, and the cases `report`, `masked`, `key-ctrlc`,
  `empty-key`, `menu-ctrlc`, `decline`, `typeahead`, `login-no-loop`, `method-menu`, `key-ctrlz`, `setup`,
  `acp-login`, `offer-ctrlc`, `setup-menu-ctrlc` and `nothing-saved`. `masked` and `key-ctrlz` reach the key prompt by
  `/login gemini`, which has no menu, so they measure what they name at the base too.
* `internal/cli/conpty_windows_test.go` (`windows && conpty`): `flush-prompt`, `flush-offer`, `abandoned`,
  `auth-paste-ctrlc`; and `internal/cli/pty_linux_test.go` (`linux && pty`): `pty-flush-prompt`, `pty-flush-offer`,
  `pty-abandoned`, `pty-ctrlc-ctrld`. Each child is this package's test binary running `Main` over a scripted agent,
  with the case's stubbed sign-in; it inherits the parent's environment, the network guard's proxy among it, so a
  request would reach the guard and fail the run. The two files share their last part word for word, each keeping
  its own copy.
* `Makefile`: `probe-conpty` runs both Windows files (`PROBE=1` the probes alone); `probe-pty` runs the Linux file;
  neither in `preflight` or CI. Docs: `make probe-pty` added beside `make probe-conpty` in `docs/guides/developing.md`
  and in `docs/architecture.md`'s sentence.
* Words. A case's observed word is the one the MADR's item 5 (A) gives when it passes. At the base, a case that cannot
  pass names what it saw: `cancel-line-before-key` (the table's "the cancel line before a key"), `turn` ("the held
  line becomes a turn"), `reached-turn` ("`partial` or `hello` reaches a turn"), `no-offer-again` (no repeated
  offer), `ended`. `result=` compares the word with the case's word after P2b, so at the base the cases expected to
  fail read `fail`, and the runs exit non-zero.

**Runs, on the tree at `473a29e` with P1's and P1b's files uncommitted, read by their `0063 case=` lines only.**

`run-on-host --need windows --label 0063-probe -- make probe-conpty PROBE=1` (step 0b's probes again, with `first=`),
on the Windows test laptop: exit 0.

```text
0063 case=probe-crlf result=pass observed=crlf
0063 case=probe-typeahead result=pass observed=one-line-per-read
0063 case=probe-ctrlc result=pass observed=eof first=read
0063 case=probe-ctrlz-line result=pass observed=eof
0063 case=probe-ctrlz-raw result=pass observed=eof
```

`first=read`: the read returned before the probe child had the interrupt, so D20's 100 ms wait for the mark is what
keeps D19 working on this console. Recorded, as D20 asks; not a stop.

`run-on-host --need windows --label 0063-conpty -- make probe-conpty`, on the Windows test laptop: exit 2, 24 lines.

```text
0063 case=probe-crlf result=pass observed=crlf
0063 case=probe-typeahead result=pass observed=one-line-per-read
0063 case=probe-ctrlc result=pass observed=eof first=read
0063 case=probe-ctrlz-line result=pass observed=eof
0063 case=probe-ctrlz-raw result=pass observed=eof
0063 case=report result=fail observed=cancel-line-before-key
0063 case=masked result=pass observed=masked
0063 case=key-ctrlc result=fail observed=cancel-line-before-key
0063 case=empty-key result=fail observed=cancel-line-before-key
0063 case=menu-ctrlc result=fail observed=ended
0063 case=decline result=fail observed=cancel-line-before-key
0063 case=typeahead result=fail observed=cancel-line-before-key
0063 case=login-no-loop result=pass observed=no-offer
0063 case=method-menu result=fail observed=cancel-line-before-key
0063 case=key-ctrlz result=pass observed=ended
0063 case=setup result=fail observed=cancel-line-before-key
0063 case=acp-login result=fail observed=cancel-line-before-key
0063 case=offer-ctrlc result=fail observed=ended
0063 case=setup-menu-ctrlc result=pass observed=ended
0063 case=nothing-saved result=pass observed=none
0063 case=flush-prompt result=fail observed=turn
0063 case=flush-offer result=fail observed=no-offer-again
0063 case=abandoned result=fail observed=reached-turn
0063 case=auth-paste-ctrlc result=pass observed=ended
```

`run-on-host --need linux --label 0063-pty -- make probe-pty`, on a Linux target (the dispatcher chose a WSL distro;
no race is run, so any Linux target is taken): exit 2, 4 lines.

```text
0063 case=pty-flush-prompt result=fail observed=turn
0063 case=pty-flush-offer result=fail observed=no-offer-again
0063 case=pty-abandoned result=fail observed=reached-turn
0063 case=pty-ctrlc-ctrld result=fail observed=no-offer-again
```

**Against the table for the base.** As the table says: the probes; `report`, `method-menu`, `setup`, `acp-login` and
`typeahead` (the cancel line before a key); `masked`, `login-no-loop` and `nothing-saved` passing; `menu-ctrlc`,
`offer-ctrlc` and `setup-menu-ctrlc` `ended` (Deviation 1's base); `key-ctrlz` `ended`; `flush-prompt`, `abandoned`,
`auth-paste-ctrlc` and the `pty-` twins of the first two; `pty-ctrlc-ctrld` with no repeated offer. Two rows fail,
as the table expects, but for another reason than the one it gives:

* `key-ctrlc`, `empty-key` and `decline` ("fail (no loop)") read `cancel-line-before-key`: on a console every first-run
  round passes the provider menu, whose CRLF answer cancels the key prompt at the base before any key, so the loop is
  never reached. Their loop reason can show only after P2.
* `flush-offer` and `pty-flush-offer` ("the repeated offer is answered") read `no-offer-again`: at the base there is no
  repeated offer to answer. Their flush reason can show only after P2b.

Neither word is a stop word (`probe-ctrlc` `error`, `menu-ctrlc` `ended` after P2); both are put to the coordinator
as findings on the table's wording for the base, not worked around.

**Gates, on the session host:** `gofmt -l cmd internal` exit 0, no file; `go vet ./...` exit 0; `make lint`
(golangci-lint v2.15.0, both modules) exit 0, `0 issues.` each; `GOOS=windows golangci-lint run --build-tags conpty
./cmd/gobble/ ./internal/cli/` exit 0, `0 issues.`; `CGO_ENABLED=0 GOOS=linux golangci-lint run --build-tags pty
./internal/cli/` exit 0, `0 issues.` (one earlier `unused` finding, a helper only the Windows file uses, moved out of
the shared part); `GOOS=windows go vet -tags conpty` and `go test -tags conpty -c` of both packages exit 0;
`CGO_ENABLED=0 GOOS=linux go vet -tags pty` and `go test -tags pty -c` of `internal/cli` exit 0; `make
check-records` exit 0; markdownlint on the two docs exit 0.

**Review, 2026-11-21.** An independent review of P1 and P1b passed: no blocker; it ran the package again on its own
copy and found the same 54 failing leaf tests, each for the reason recorded above. Its record follow-ups are applied
(the dated correction of the base words under P1b's amendment, with Deviation 1's "after P2" for `menu-ctrlc`
corrected there, at step 0b's gate and in Deviation 1's gate; and the dated Scope amendment for the two docs). Its
test findings are follow-ups for P2, the tests unchanged now:

1. `report` and `typeahead` (`cmd/gobble/conpty_windows_test.go:774-878`) and `TestFirstRunTypeAhead`
   (`internal/cli/login_test.go:823`) pick Gemini, the provider menu's default item, so a stray `\n` answering the
   menu still reaches Gemini's key prompt and passes. In P2: check that nothing follows the key prompt, or pick a
   provider other than the default.
2. `TestFirstRunInstantFailureClosedInputEnds` (`internal/cli/firstrun_test.go:366`) never reaches `noSignIn`: the
   provider menu reads the end of input first. In P2: answer the menu before closing the input.
3. No P1 test delivers the owed `\n` after a delay, so an `lfOwed` that lapsed after a fixed wait would pass P1. In
   P2: pin it in `terminput_test` with a stub whose read blocks until the test gives it the `\n`.
4. (Found while correcting the table.) `menu-ctrlc`'s drive reads `other` on the P2 tree, where no repeated offer comes
   and gobble goes on; in P2 it should name that `no-offer-again`, as the corrected table gives.

### P2, build pass 1 (2026-11-21)

**Outcome: built; every local gate passes; stopped on two findings** (F1 and F2 below), each a row the build cannot meet
as written. On the branch `feature/0063-shared-terminal-input` at `0eb985a`, the tree clean at the start; nothing
staged or committed. P2b, P3 to P6 not started.

**What each step did** (uncommitted, unstaged):

* Steps 1 and 12 (as amended), and 2: `internal/cli/terminput.go` (new): `terminalInput` with `in`, `flush`, the
  mutex-guarded `pending`, `lfOwed`, `closed` and the input's error, and a `turn` held by the prompt reading, so bytes
  are given in the order read and the mutex is never held across a read of `in`. `newTerminalInput(in, flush)`;
  `Read`, `line`, `key`, `rawEnd`, `discard`, `ended`; one method, `owedLocked`, applies D3's rule to the next byte
  for every entry point. A read of `in` is made only for a prompt that finds `pending` empty, with its buffer
  (`line` and `key` 512 bytes); a read started on an ended context is never made. `term.FlushInput` in
  `internal/cli/term/flush_linux.go` (the `TCFLSH` ioctl with `TCIFLUSH`), `flush_darwin.go` (`TIOCFLUSH` with the
  read queue), `flush_windows.go` (`FlushConsoleInputBuffer`), `flush_other.go` (an error), and `flush_test.go`.
* Step 3 (as amended): `runEnv` gains `termInMu`, `termIn` and `flushWarn` (`main.go`). `terminalIn()` and
  `flushTerminal()` are in `terminput.go`, not `main.go`, because A18 requires `inline.NewReader` in `terminput.go`
  alone. `terminalIn()` reads `term.ConsoleInput(e.stdin, e.intr.consoleMark())` and, when `onTerminal()`, binds
  `flushTerminal`, which calls the `termFlush` seam (`term.FlushInput`) and on the first failure writes R101's
  words through `e.out`, once.
* Steps 4, 11: `questions.in` is `*terminalInput`; `pending`, `afterCR` and `fill` are gone; `lineContext` reads
  `in.line`, `readSecret` reads `in.key` and calls `in.rawEnd(c)` on Enter; `secret` restores raw mode in a deferred
  call that also writes the line end.
* Step 5: `ls.ask` over `ls.keys`; `terminalQuestions` and `auth.go`'s paste prompter over `e.terminalIn()`; the trust
  question reads `e.terminalIn().Read(e.ctx, p)` through a method, `trustKeys`, which ignores `askOn`'s contexts (a
  closure using `e.ctx` was refused by `contextcheck`); `askPermission` reads `ls.keys`, now `*terminalInput`.
* Step 6: `lineSession.in` is gone; `ls.keys = e.terminalIn()`; `editor.New(ls.keys, …)`; before each `ReadPrompt`,
  `ls.keys.beforePrompt()`: `discard()` and, when the mark is set, the mark cleared and `flushNext()` (step 9). The
  harness (`line_test.go`, `start`) sets `h.e.termIn` over `inline.NewReader(&rawCheckedInput{…})`, and gains
  `flushes` (the flush bound through `flushTerminal`, with `termFlush` stubbed) and `tty`, a terminal queue a stubbed
  flush empties. `sessions_test.go:327` gave its pipe through the removed `in` field and now sets `e.termIn` the same
  way: a file outside P2's named list, changed only in how it builds its input.
* Step 7: `editor.Input`; `New(in Input, …)`; `doc.go`'s sentence on the one reader.
* Step 9 (as answered): the mark is `terminalInput.signedIn`, set by `signedInRound()`. `ls.login` sets it after its
  ask, whatever the outcome, and not for a provider refused before any question: it is the body P2b's step 1 moves
  into `ls.signInRound`, and the mark moves with it. `firstRun` sets it in a deferred call. `flushNext()` marks the
  next read of `in` to flush first, and marks nothing with no flush.
* Step 10 (as amended): `abandon()` marks a read of `in` that ended on its context and may still wait; the next read
  takes its completion with that read's own size and drops it, the end of input excepted. `questions` call it when
  the context that ended is not the question's own (`ls.ask`'s, cancelled by the person's Ctrl+C); `askPermission`
  when the request's context ended and the run's did not.
* Steps 13 and 14: `internal/cli/term/consolein.go` (`InterruptMark`, `MarkWait` 100 ms, `NewConsoleReader`,
  `ConsoleInput`), `consolein_windows.go` (`GetConsoleMode`), `consolein_other.go`, `consolein_test.go`;
  `interrupts.console` and `consoleMark()`, and `watchSignals` calls `Interrupted()` on each `os.Interrupt` before
  `intr.take()` (`signal.go`).
* Tests: `terminput_test.go` (step 2's table as `TestTerminalInputRule`, `TestTerminalInputEnds`,
  `TestTerminalInputReadsOnlyWhenAsked`; A15 to A19, A22, A25, A28 as extended); `flushpoints_test.go` (A24 as
  extended, A29); `readers_test.go` (step 8, A23 with a withdrawn and a Ctrl+C case, A27); `signal_test.go` (A31);
  `consolein_test.go` (A30, A32, A34 to A39); `flush_test.go` (step 12). A33 is the existing
  `TestInterruptDuringTurn` and `TestInterruptReleased`, run after step 14: pass. `asked` and `askedFrom` build their
  input through `newTerminalInput(…, nil)` (step 2).

**The review's follow-ups, applied:**

1. `TestFirstRunTypeAhead` (`login_test.go`) answers the menu with Claude, not the default Gemini, and requires that
   nothing follows Claude's key prompt; `report` and `typeahead` (`cmd/gobble/conpty_windows_test.go`) keep the MADR's
   script (`1`, Enter) and gain `keyPromptLast`: the key prompt must be the last thing on the screen, else
   `after-key-prompt`. At `0eb985a` on a scratch clone the test is still red in every form (`login_test.go:834:
   condition not met within 5 s`); with mutation 1 on a scratch copy of the P2 tree its `CRLF` subtest fails
   (`login_test.go:835: condition not met within 5 s`: Gemini's prompt came, the stray `\n` having taken the
   menu's default).
2. `TestFirstRunInstantFailureClosedInputEnds` (`firstrun_test.go`) answers the provider menu and waits for
   `Error: tests open no browser and sign in nowhere` before it closes the input; still a guard at `0eb985a` and here.
3. `TestOwedLFAfterADelay` (`terminput_test.go`): the owed `\n` arrives 700 ms after the read that waits for it, over
   `inline.Reader` and a pipe. A plant on a scratch copy that clears `lfOwed` after a read of over 500 ms fails it
   (`the line after a late \n = "", <nil>; want "b"`) while `TestCRLFSplitAcrossReads` passes.
4. `menu-ctrlc` returns `no-offer-again` where it returned `other`. The Windows run below reads it.

**Gates**, on this Mac unless named:

| Command | Exit | Output line |
| --- | --- | --- |
| `go test -count=1 ./internal/cli/editor/` | 0 | `ok   github.com/maccavelli/gobble-cli/internal/cli/editor 2.100s` |
| `go test -count=1 ./internal/cli/term/` | 0 | `ok   github.com/maccavelli/gobble-cli/internal/cli/term 1.692s` |
| `go test -count=1 -timeout 10m -json ./internal/cli/` | 1 | 33 failing leaf tests, every one red for the loop (below); `TestLineSessionReadsOnlyWhenAsked` passes, its count of cooked reads 1 as before |
| `make archtest` | 0 | `ok   github.com/maccavelli/gobble-cli/internal/archtest 1.435s` |
| `make pre-add-check FILES="<the 30 changed .go files>"` | 0 | `go-precheck: 30 file(s) clean (gofmt, golangci-lint, govulncheck).` (a first run found `gofmt` in `flushpoints_test.go`, `contextcheck` in `trust.go`, and two `gocritic` in `consolein.go` and `terminput.go`; fixed) |
| `GOOS=windows go vet -tags conpty ./cmd/gobble/ ./internal/cli/...` | 0 | no output |
| `CGO_ENABLED=0 GOOS=linux go vet -tags pty ./internal/cli/...` | 0 | no output |
| `GOOS=windows golangci-lint run --config .golangci.yml --build-tags conpty ./cmd/gobble/ ./internal/cli/...` (v2.15.0) | 0 | `0 issues.` |
| `CGO_ENABLED=0 GOOS=linux golangci-lint run --config .golangci.yml --build-tags pty ./internal/cli/...` | 0 | `0 issues.` |
| `GOOS=freebsd go vet ./internal/cli/term/` (`flush_other.go`) | 0 | no output |
| `run-on-host --need windows --label 0063-p2-conpty -- make probe-conpty` | 2 | 24 lines, below |
| `run-on-host --need linux --label 0063-p2-pty -- make probe-pty` (a WSL distro) | 2 | 4 lines, below |

**Failing leaf tests on `./internal/cli/`: 33, each red for the loop for P1's reason.** 30 fail with `the offer again,
[y/N]: not before the session's prompt`: `TestFirstRunOfferAgainAfterKeyCancel`, `…AfterEmptyKey`,
`…AfterMenuInterrupt`, `TestFirstRunRepeatedOfferDefaultsNo`, `TestFirstRunDeclineOnRepeatEnds`,
`TestFirstRunTypeAheadDroppedBeforeRepeatedOffer`, `TestFirstRunInterruptAtRepeatedOfferEnds`,
`TestFirstRunYesOnRepeatStartsAnotherRound`, each `/CR`, `/LF`, `/CRLF`, and `TestFirstRunOfferAgainAfterFailure`
`/refused` and `/offline`, each in three forms; 3 fail with `the warning was printed 0 times; want 1`:
`TestFirstRunEndOfInputInRoundEnds/CR`, `/LF`, `/CRLF`. Every test P1 marked red for the defect passes; every guard
and every other test passes. At `0eb985a` the same run had 54; the 21 that now pass are the set P1 recorded red
at the base and passing after P2, `TestSecretRestoredOnPanic` (A26) among them.

**`0063 case=` lines.** The Windows test laptop, exit 2:

```text
0063 case=probe-crlf result=pass observed=crlf
0063 case=probe-typeahead result=pass observed=one-line-per-read
0063 case=probe-ctrlc result=pass observed=eof first=read
0063 case=probe-ctrlz-line result=pass observed=eof
0063 case=probe-ctrlz-raw result=pass observed=eof
0063 case=report result=pass observed=waits
0063 case=masked result=pass observed=masked
0063 case=key-ctrlc result=fail observed=no-offer-again
0063 case=empty-key result=fail observed=no-offer-again
0063 case=menu-ctrlc result=fail observed=no-offer-again
0063 case=decline result=fail observed=no-offer-again
0063 case=typeahead result=pass observed=waits
0063 case=login-no-loop result=pass observed=no-offer
0063 case=method-menu result=pass observed=waits
0063 case=key-ctrlz result=pass observed=ended
0063 case=setup result=pass observed=cancelled
0063 case=acp-login result=pass observed=cancelled
0063 case=offer-ctrlc result=pass observed=warned-once
0063 case=setup-menu-ctrlc result=pass observed=ended
0063 case=nothing-saved result=pass observed=none
0063 case=flush-prompt result=pass observed=no-turn
0063 case=flush-offer result=fail observed=no-offer-again
0063 case=abandoned result=pass observed=dropped
0063 case=auth-paste-ctrlc result=pass observed=ended
```

A Linux target (a WSL distro, as at P1b), exit 2:

```text
0063 case=pty-flush-prompt result=pass observed=no-turn
0063 case=pty-flush-offer result=fail observed=no-offer-again
0063 case=pty-abandoned result=fail observed=reached-turn
0063 case=pty-ctrlc-ctrld result=fail observed=no-offer-again
```

Against the tables for the P2 tree: every Windows line is as expected (the `fail` lines are the P2b words of the
correction of 2026-11-21: `key-ctrlc`, `empty-key`, `decline`, `menu-ctrlc` and `flush-offer` `no-offer-again`;
`menu-ctrlc` is not `ended`, so D19 holds; `probe-ctrlc` `first=read` again). On Linux, `pty-flush-prompt`,
`pty-flush-offer` and `pty-ctrlc-ctrld` are as expected; **`pty-abandoned` is not**: it must be `dropped` after P2
(F1).

**F1 (stop): `pty-abandoned` reads `reached-turn` on the P2 tree.** Not pre-existing in this form: at the base it read
`reached-turn` because nothing was dropped. A diagnostic on a scratch clone, the case alone with its word naming the
turns, gave `observed=reached-turn:hello,again`: `partial`, typed into the paste prompt the program gave up, was
dropped, and `hello`, typed at gobble's prompt after it, became a turn. Cause, from that run and the kernel's
terminal driver (inferred for macOS from the BSD driver): the line session puts the terminal back in raw mode
(`line.go`, `outOfRaw`) before the prompt reads; turning canonical mode off hands what was typed into the waiting
line read (`partial`) to that read at once; D13 drops it as the completion; the prompt's flush (step 9: at the
editor's first read, after raw mode is on) finds nothing; `hello` is read whole by the editor. The MADR's model,
Q16's text and rows R86 and R102, has the waiting read complete with the first key typed after raw mode is on, so
that key is the one lost: an asserted fact this run contradicts. A second diagnostic, the same clone with the flush
also made when `/login`'s round ends, while the terminal is still out of raw mode, gave
`0063 case=pty-abandoned result=pass observed=dropped`. Resolutions, for the owner:

1. *(Recommended.)* Keep the build, and amend the records to what was measured: on POSIX the read a program gave up
   completes as raw mode starts with what was typed into its prompt, which is dropped; the first key at the next
   prompt is lost only when nothing was typed there (R86, R102); `pty-abandoned` passes when no turn holds `partial`
   and `again` is one. Cost: dated MADR and PLAN amendments and a one-line change to the case's check in
   `internal/cli/pty_linux_test.go`, a P1b test, which needs the owner's word. Nothing typed at a prompt the person can
   see is lost.
2. Flush when a sign-in round or the first run ends, while the terminal is out of raw mode, as well as at the first
   prompt. Cost: a third flush point (R80's "exactly two", A24's counts and its plants change); the first key typed at
   the prompt after such a sign-in is lost, as R102 says.
3. Make the session-prompt flush run at once before the line session puts the terminal back in raw mode for the
   prompt, when the mark is set, in place of step 9's flush at the editor's first read. Cost: as 2 for the lost key;
   it also settles F2; A24's R108 case changes (below).

Doing nothing leaves P2's gate failed on the Linux case.

**F2 (stop): keys the editor holds across a sign-in meet the mark at another prompt than R103 and R105 say.** Found by
reading, while writing A24, and pinned by its R108 case. When keys typed after `/login` in the same press (or paste)
are decoded by the editor, they stay its own (R56); at the first prompt after the sign-in the editor submits them
without reading the terminal, so the flush step 9 makes due at "the next read of `in`" runs at the prompt after that
turn: R103 ("the next `ReadPrompt` flushes") is not met for that prompt, and keys typed during the held keys' turn are
flushed, against R105. A24's R108 case ("a session ended by Ctrl+D with the mark set counts no flush") can only be
built on exactly this: the Ctrl+D is held by the editor with `/login gemini`, and the session ends with the flush due
and never made (`TestFlushPoints/Ctrl+D_with_the_mark_set`, passing). No row decides keys held by the editor and the
mark together. Resolutions:

1. *(Recommended, with F1's resolution 3.)* Flush at once at the session prompt when the mark is set, before the
   terminal goes back to raw mode; R103 and R105 then hold with held keys; R108's words still hold (nothing is flushed
   on the way out), but A24's R108 case counts the flush made at that prompt and is re-worded. Cost: step 9's
   `flushNext()` at the prompt becomes a flush made then (P2b's repeated offer keeps `flushNext()`), dated
   amendments of step 9, A24 and plant 61, about ten lines in `line.go` and `terminput.go`.
2. Keep the flush lazy, and cancel a due flush that no read took when `ReadPrompt` returns. Cost: no flush at all after
   such a sign-in, against R103; nothing typed during the held turn is lost.
3. Keep it as built and record the case in R105. Cost: keys typed while the held keys' turn runs are dropped.

Doing nothing leaves R103 and R105 unmet in that case, with no test that shows it.

**Seen to fail before use.** P3 owns the plants; before handing over, each test P2 adds or tightens was run once
against a deliberate break on a scratch copy of the tree (never the tree), one break at a time, the copy restored
after each: 36 breaks, each failing its test with the message it names, among them plants 1 (with follow-up 1), a
variant of 6 (the read's buffer), 8, 21, 29, 30, 34, 41, 42, 47, 48, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 62, 63, 64, 65,
66, 68, 69, 71, 72, 73 and 74, the withdrawn permission request, follow-up 3's lapse, and `FlushInput` giving no error
on a pipe. Not run: 35 (`-race`, P4), and 49 and 61, whose breaks need more than one changed line and are P3's.

**Waiting for P2b**, each with its rows: the 33 tests above (R13, R37 to R39, R64 to R66, R69, R70); A24's case for the
repeated offer's flush, after its text is written (R79, R80; plants 44, 45); `flush-offer` and `pty-flush-offer`
(R69, R74, R76); `key-ctrlc`, `empty-key`, `decline` and `menu-ctrlc`'s `offer-again`, then `warned-once` (R13, R37,
R39, R64, R66, R109, R119); `pty-ctrlc-ctrld`'s `warned-once` (R118); the mark set by `ls.signInRound` (R103), now
set by `ls.login`, whose body P2b moves there.

**Not done, and why:** A19 under `-race` (P4 step 5); P3's plants as a phase; the docs outside P2's files (P5); F1
and F2, which wait for the owner.

**Docs** (P2's file list): `internal/cli/editor/doc.go` and `editor.go`'s `Input` and `New` comments fixed; every new
or changed exported name in `internal/cli/term` has its comment. Open for P5: `docs/architecture.md:133`
(`internal/cli`), `:135` (`internal/cli/editor`), `:137` (`internal/cli/term`, which gains `FlushInput` and
`ConsoleInput`; ~~not in P5's step 2 as written~~ *(2026-11-21: P5's step 2, Deviation 2)*); `docs/guides/credentials.md:353-370`, `:366`, `:382`; `README.md:120`;
`docs/README.md:143-144`; and `internal/cli/term/doc.go:1-3`, whose package summary names neither, ~~in no phase's list~~ *(2026-11-21: P5's, Deviation 2)*.

#### Deviation 2 (2026-11-21, P2's build pass 1): F1 and F2, and the build's four choices

* **Found.** The two stops recorded above. F1: `pty-abandoned` read `reached-turn` on the P2 tree; a diagnostic named
  the turns `hello` and `again`: on Linux, what was typed into the abandoned paste prompt completes its read when the
  session prompt puts the terminal in raw mode, against the MADR's Q16, R86 and R102. F2: keys the line editor holds
  across a sign-in put the session prompt's flush, made due at the editor's first read, off to a later prompt, against
  R103 and R105; no row decided the case.
* **Rulings** (the owner, relayed by the coordinator): F1, "Amend R86/R102 to what Linux does (Recommended)": the build stands; R86, R102 and Q16's
  statement amended to the measured behaviour (MADR D21); `pty-abandoned`'s check changes by one line. F2,
  "Flush at once at that first prompt (Recommended)": the flush at the first prompt after a sign-in or the first run is made at that prompt, whether or
  not the editor then reads the terminal; R123 added; R108's test and plant 61 re-worded; plant 75 added (MADR D21;
  A24 and P3 above).
* **Files added to scope:** `internal/cli/pty_linux_test.go` (P1b's, the one-line `pty-abandoned` check) and
  `internal/cli/sessions_test.go` in P2; `internal/cli/term/doc.go` and the `internal/cli/term` row of
  `docs/architecture.md` in P5 (Scope and P5, dated notes above).
* **Where the code lands.** P2's build pass 2: the flush at the prompt (`line.go`, `terminput.go`), A24's cases as
  amended, the `pty-abandoned` check, and the re-runs of P2's gate.

**The build's four choices, accepted** (the owner, relayed by the coordinator: "Accept all four; add both docs to P5 (Recommended)"):
*(2026-11-21, the coordinator's records gate: the owner's own words were "option 1", choosing "Accept all four; add
both docs to P5 (Recommended)".)*

1. *(2026-11-21.)* `terminalIn()` and `flushTerminal()` are in `internal/cli/terminput.go`, not `main.go` as steps 3 and
   13 say: A18 requires `inline.NewReader` in `terminput.go` alone. `runEnv`'s fields stay in `main.go`.
2. *(2026-11-21.)* The mark of step 9 is set by `ls.login`, whatever the sign-in's outcome and not for a provider refused
   before any question, until P2b's step 1 moves `ls.login`'s body, and the mark with it, into `ls.signInRound`.
3. *(2026-11-21.)* `internal/cli/sessions_test.go:327`, outside P2's file list, set the `lineSession.in` field step 6
   removes; it now sets `e.termIn` over `inline.NewReader` of its pipe, changed only in how its input is built.
4. *(2026-11-21.)* The trust question reads through a method, `trustKeys`, which reads the terminal input with the run's
   context, in place of a closure using `e.ctx`, which `contextcheck` refused; the behaviour is step 11's.

### P2, build pass 2 (2026-11-21)

**Outcome: built; every local gate passes; stopped on two findings (F3, F4) about rows R102 and R86 as amended by MADR
D21.** On the branch at `0eb985a`, nothing staged or committed. The coordinator gated STEP A's records and sent this
pass: the owner's words "option 1" added to Deviation 2; F2's code; A24's re-worded R108 cases, the R123 case and
plant 75; the one-line `pty-abandoned` check; A40, `pty-abandoned-empty`; the gate again.

**What this pass did:**

* `internal/cli/terminput.go`: `beforePrompt()` flushes at once when the mark is set, whether or not the editor then
  reads the terminal, and clears the mark (MADR D21, rows R103, R105, R123); a failed flush is skipped, and
  `flushTerminal` still warns once. `flushNext()` stays for P2b's repeated offer; its comment says so.
* `internal/cli/flushpoints_test.go`: `TestFlushPoints`'s case "Ctrl+D with the mark set" is replaced by "Ctrl+D at
  the first prompt after a sign-in" (one flush, none after) and "run ended during the models turn" (none), row R108
  as amended; "keys held across a sign-in" is added, row R123: `/login gemini` and `1`, Enter in one press, Enter at
  the key prompt; one flush when the held `1`'s turn runs; `typed-during`, typed while it runs, is the next turn.
* `internal/cli/terminput_test.go`: `TestFlushNext`, for step 9's `flushNext()`, which no longer had a caller and
  which `golangci-lint`'s `unused` refused: it flushes once before the next read of the input, not while pending
  holds bytes, and not at all with no terminal.
* `internal/cli/pty_linux_test.go`: `ttyAbandoned`'s check is `case r.turnHolds("partial"):` (was `partial` or
  `hello`), with its comment and the shared-part comment annotated (MADR D21); A40, `pty-abandoned-empty`
  (`ptyAbandonedEmpty`, words `first-key-dropped`, `none-lost`, `no-turn-again`, `other-turns`), whose child runs
  `pty-abandoned`'s stubbed sign-in.
* Records: the owner's words in Deviation 2; A40 here and in R102's latest row's test cell.

**Gates**, on this Mac unless named:

| Command | Exit | Output line |
| --- | --- | --- |
| `go test -count=1 -timeout 10m -json ./internal/cli/` | 1 | the same 33 failing leaf tests as build pass 1, the same two messages (30 `the offer again, [y/N]: not before the session's prompt`, 3 `the warning was printed 0 times; want 1`); `TestLineSessionReadsOnlyWhenAsked` passes |
| `go test -count=1 ./internal/cli/editor/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/editor 1.846s` |
| `go test -count=1 ./internal/cli/term/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/term 1.407s` |
| `make archtest` | 0 | `ok github.com/maccavelli/gobble-cli/internal/archtest 1.223s` |
| `make pre-add-check FILES="<the 31 changed .go files>"` | 0 | `go-precheck: 31 file(s) clean (gofmt, golangci-lint, govulncheck).` (a first run: `func (*terminalInput).flushNext is unused (unused)`; `TestFlushNext` added) |
| `go vet ./internal/cli/... ./cmd/gobble/` | 0 | no output |
| `GOOS=windows go vet -tags conpty ./cmd/gobble/ ./internal/cli/...` | 0 | no output |
| `CGO_ENABLED=0 GOOS=linux go vet -tags pty ./internal/cli/...` | 0 | no output |
| `run-on-host --need windows --label 0063-p2-conpty -- make probe-conpty` | 2 | 24 lines, the same words as build pass 1 |
| `run-on-host --need linux --label 0063-p2-pty -- make probe-pty` (a WSL distro) | 2 | 5 lines, below |

The Windows test laptop's lines are build pass 1's, word for word, each as the table for the P2 tree expects. The
Linux target:

```text
0063 case=pty-flush-prompt result=pass observed=no-turn
0063 case=pty-flush-offer result=fail observed=no-offer-again
0063 case=pty-abandoned result=pass observed=dropped
0063 case=pty-ctrlc-ctrld result=fail observed=no-offer-again
0063 case=pty-abandoned-empty result=fail observed=other-turns
```

`pty-flush-offer` and `pty-ctrlc-ctrld` read the P2-tree words; `pty-abandoned` passes; `pty-abandoned-empty` does
not (F3).

**Seen to fail**, on scratch copies, never the tree: plant 75 (the prompt's flush made due at the editor's first
read) fails `TestFlushPoints/keys_held_across_a_sign-in` (`when the held keys' turn runs: 0 flushes; want 1`); plant
61 (the session flushes once more on its way out) fails both R108 cases (`2 flushes; want 1`; `a run ended with the
mark set: 1 flushes; want 0`); `flushNext()` marking nothing fails `TestFlushNext` (`flushes at reads []`). Plant 47
on a scratch clone, on the Linux target: `pty-abandoned-empty` reads `none-lost` (caught); **`pty-abandoned` reads
`dropped`, so its check no longer catches plant 47** (F4).

**F3 (stop): `pty-abandoned-empty` reads `other-turns`.** A diagnostic on a scratch clone named the turns: with
`hello`, Enter written at once, as the case does, the only turn is `again`: the whole burst completed the abandoned
read and was dropped. The same drive typing `hello`, Enter a key at a time, 150 ms apart, read `first-key-dropped`
(a turn `ello`, a turn `again`). So R102's inferred branch holds for keys typed one at a time; a paste, or keys that
arrive in one read, are dropped whole. The row is not changed.

**F4 (stop): on the pass-2 tree, what was typed into the paste prompt no longer completes the abandoned read.** A
diagnostic of `pty-abandoned` on a scratch clone of this tree, three runs, named the turns `again`, `again`,
`again`: `partial` and the burst `hello`, Enter were both lost. With F2's flush made at once as the prompt starts,
the flush runs right after raw mode goes on and, in these runs, before the kernel hands `partial` to the waiting
read, so `partial` is flushed and the read still waits; the first read at gobble's prompt then completes it and is
dropped (F3's behaviour). At build pass 1, with the flush at the editor's first read, `partial` reached the read
first (`reached-turn:hello,again`, then a turn `hello`). Which one happens is a race between gobble's flush and the
kernel's wake-up of the waiting read. R102 as amended by D21 ("nothing typed at the session prompt is lost") does not
hold on this tree; R86's latest row holds in the race's other order only. And `pty-abandoned`'s new check passes
either way, so it no longer catches plant 47 (which A21's D21 note requires); `pty-abandoned-empty` does.

Resolutions, for the owner (F3 and F4 together):

1. *(Recommended.)* Make the outcome one, not a race: at the first prompt after a sign-in or the first run, flush
   before the terminal goes back to raw mode for the prompt, not after; `partial` is then flushed while the read is
   still a line read and never handed over, and the abandoned read takes the first read at the prompt: one key when
   typed, a whole paste when pasted. R102 and R86 amended to that, measured (F3's two runs); `pty-abandoned`'s
   expected turns `again` and the dropped `hello` named, and A40 as two cases, typed (`first-key-dropped`) and pasted
   (`burst-dropped`). Cost: the flush moves into the line session's return to raw mode (`line.go`), D21 and A24
   amended again; the first key, or a paste, at the first prompt after such a sign-in is lost.
2. Keep this tree, and record the race: R102 and R86 state both orders and that either may happen; `pty-abandoned`
   keeps its check (both orders pass); plant 47 is caught by `pty-abandoned-empty` alone. Cost: a row that decides two
   outcomes by timing; the test cannot pin which.
3. Revert F2's placement for POSIX only, flushing at the editor's first read as at build pass 1, which F1 measured.
   Cost: F2's ruling is narrowed to Windows and the held-keys case R123 comes back on POSIX; against the owner's
   ruling as given.

Doing nothing leaves R102 and R86 asserting what this tree does not do, and A40 failing.

**Not done, and why:** no change to R102, R86 or A40's words beyond A40's record (the coordinator's instruction: a
contradiction is a stop); P2b to P6 not started; `-race` is P4's.

### Records draft for D22 (2026-11-21)

**The owner's answer** to F3 and F4 of build pass 2, relayed by the coordinator: "Make reads cancellable on POSIX". The options shown
were "Flush before raw mode; third pass (Recommended)", "Make reads cancellable on POSIX" and "Keep this tree; record the race"; the owner chose the second, which was not the recommendation. Its stated cost: it replaces
D13's mechanism for Linux and macOS, so it needs a new decision, a records pass, new tests and plants; Windows is
unchanged; 0063 is delayed by at least one records pass and a build. Records only: no code file was changed,
reverted or stashed; the tree's code is build pass 2's.

**Measurements**, a scratch program (never the tree) on this Mac, macOS 27.2.1, its standard input the slave of a
pseudo terminal that a Python `pty` driver opened, typing at each step. Run twice; the second run, with M8 added,
gave the same lines for M0 to M7 as the first:

```text
M0 isatty termios err=<nil> icanon=true isig=true
M1 poll-idle n=0 revents=0x0 err=<nil>
M1 poll-data n=1 revents=0x1 err=<nil>
M1 poll-data read n=2 "a\n" err=<nil>
M2 select-idle n=0 ttyready=false err=<nil>
M2 select-data n=1 ttyready=true err=<nil>
M2 select-data read n=2 "b\n" err=<nil>
M3 cancel n=1 ttyready=false wakeready=true err=<nil>
M3 next-wait n=1 ttyready=true err=<nil>
M3 next-wait read n=4 "abc\n" err=<nil>
M5 canonical-partial n=0 ttyready=false err=<nil>
M5 raw-on set-err=<nil> n=1 ttyready=true err=<nil>
M5 after-flush flush-err=<nil> n=0 ttyready=false err=<nil>
M5 raw-on-no-flush n=1 ttyready=true err=<nil>
M5 raw-on-no-flush read n=4 "half" err=<nil>
M6 ctrl-c select n=0 ttyready=false err=<nil> eintr=false after=2s signal=yes
M6 after ctrl-c select n=0 ttyready=false err=<nil>
M7 ctrl-d select n=1 ttyready=true err=<nil>
M7 ctrl-d read n=0 "" err=<nil>
M4 poller setnonblock-err=<nil> deadline-err=<nil> nodeadline=false read n=0 err=read tty: i/o timeout timeout=true after=300ms
M4 poller-data n=2 "c\n" err=<nil>
M8 dev-tty poll-idle n=1 revents=0x20 err=<nil>
M8 dev-tty select-idle n=0 ready=false err=<nil>
```

What each shows: poll(2) serves the pseudo terminal's slave (M1) but refuses the same terminal opened as `/dev/tty`
(M8, `POLLNVAL`), where select(2) waits (M8), so select(2) is the wait on macOS; select(2) waits, reports a line
ready, and is woken by a pipe without reading the terminal (M2, M3); what is typed after a cancelled wait is read
whole by the next (M3); half a line typed in canonical mode while nothing reads is not ready, becomes ready when raw
mode goes on, and is gone after a flush (M5); Ctrl+C reaches the process as a signal, empties the half line, and
does not end select(2) on macOS (M6); Ctrl+D on an empty line is ready and reads nothing (M7); the runtime poller
serves a deadline on the terminal once it is non-blocking (M4), which D22 does not use (the flag is the shared open
file description's). Linux is not measured: D22's step 0.

### Records draft for D22, second draft (2026-11-21)

After the independent record review of the first draft (its report: findings 1.1 to 3.10), the coordinator asked for
step 0's Linux measurement and four more (M11 to M14), then a second draft of the records. Records only: no code was
changed, reverted or stashed. The probe is a scratch program, never in the tree: build-tagged termios calls (Linux
`TCGETS`, `TCSETS`, `TCFLSH` with `TCIFLUSH`; macOS `TIOCGETA`, `TIOCSETA`, `TIOCFLUSH`), driven on a pseudo
terminal by a Python `pty` driver, run on this Mac, and on a WSL distro and a Linux machine that is not WSL through
`run-on-host --need linux --label 0063-d22-probe` (the second with `--race`, which only keeps a run out of WSL
distros; nothing ran under `-race`). Each run exited 0.

**M0 to M10, the Linux hosts** (identical lines on both):

```text
M0 isatty termios err=<nil> icanon=true isig=true
M1 poll-idle n=0 revents=0x0 err=<nil>
M1 poll-data n=1 revents=0x1 err=<nil>
M1 poll-data read n=2 "a\n" err=<nil>
M2 select-idle n=0 ttyready=false err=<nil>
M2 select-data n=1 ttyready=true err=<nil> eintr-retried=0
M2 select-data read n=2 "b\n" err=<nil>
M3 cancel n=1 ttyready=false wakeready=true err=<nil> eintr-retried=0
M3 next-wait n=1 ttyready=true err=<nil> eintr-retried=0
M3 next-wait read n=4 "abc\n" err=<nil>
M5 canonical-partial n=0 ttyready=false err=<nil>
M5 raw-on set-err=<nil> n=1 ttyready=true err=<nil>
M5 after-flush flush-err=<nil> n=0 ttyready=false err=<nil>
M5 raw-on-no-flush n=1 ttyready=true err=<nil>
M5 raw-on-no-flush read n=4 "half" err=<nil>
M6 ctrl-c bare-select n=0 ttyready=false err=<nil> eintr=false after=2s signals=interrupt;
M6 after ctrl-c select n=0 ttyready=false err=<nil> eintr-retried=0
M6 ctrl-c retried-select n=0 ttyready=false err=<nil> eintr-retried=0 after=2s signals=interrupt;
M7 ctrl-d select n=1 ttyready=true err=<nil> eintr-retried=0
M7 ctrl-d read n=0 "" err=<nil>
M9 sigusr1 bare-select n=0 err=<nil> eintr=false after=2s signals=user defined signal 1;
M9 sigusr1 retried-select n=1 ttyready=true err=<nil> eintr-retried=0 signals=user defined signal 1;
M9 retried-select read n=2 "d\n" err=<nil>
M4 poller setnonblock-err=<nil> deadline-err=<nil> nodeadline=false read n=0 err=read tty: i/o timeout timeout=true after=300ms
M4 poller-data n=2 "c\n" err=<nil>
M8 dev-tty poll-idle n=0 revents=0x0 err=<nil>
M8 dev-tty select-idle n=0 ready=false err=<nil>
M10 hangup select n=1 ttyready=true err=<nil> eintr-retried=0 after=400ms signals=none
M10 hangup read n=0 "" err=<nil>
M10 hangup read-again n=0 "" err=<nil>
M10 hangup select-again n=1 ttyready=true err=<nil>
M10 hangup blocking-read n=-1 "" err=input/output error after=400ms signals=none
M10 hangup read-again n=0 "" err=<nil>
```

On this Mac, M0 to M8 as in "Records draft for D22"; the added M9 and M10:

```text
M9 sigusr1 bare-select n=0 err=<nil> eintr=false after=2s signals=user defined signal 1;
M9 sigusr1 retried-select n=1 ttyready=true err=<nil> eintr-retried=0 signals=user defined signal 1;
M9 retried-select read n=2 "d\n" err=<nil>
M10 hangup select n=1 ttyready=true err=<nil> eintr-retried=0 after=400ms signals=none
M10 hangup read n=0 "" err=<nil>
M10 hangup read-again n=0 "" err=<nil>
M10 hangup select-again n=1 ttyready=true err=<nil>
M10 hangup blocking-read n=0 "" err=<nil> after=400ms signals=none
M10 hangup read-again n=0 "" err=<nil>
```

**M11 to M14, the Linux hosts** (identical lines on both):

```text
M0 isatty termios err=<nil>
M11 line ready n=1 ttyready=true err=<nil>
M11 after ctrl-c select n=0 ttyready=false err=<nil>
M11 read on fd 0: blocked after 2s
M11 the blocked read ended with n=8 "unblock\n" err=<nil> eagain=false
M12 ttyname path-ok=true env-match=true err=<nil>
M12 open err=<nil> fd0 nonblock=false private nonblock=true
M12 idle read private n=-1 "" err=resource temporarily unavailable eagain=true
M12 select private n=1 ready=true err=<nil>
M12 read private n=3 "p1\n" err=<nil> eagain=false
M12 fd 0 after n=0 ready=false err=<nil>
M12 line2 ready n=1 ready=true err=<nil>
M12 read private after ctrl-c n=-1 "" err=resource temporarily unavailable eagain=true after=0s
M12 raw set-err=<nil> select private n=1 ready=true err=<nil>
M12 raw read private n=1 "r" err=<nil> eagain=false
M12 flush fd 0 err=<nil>; read private n=-1 "" err=resource temporarily unavailable eagain=true
M12 fd0 after nonblock=false
M13 canonical ready n=1 ttyready=true err=<nil>
M13 raw-on ready n=1 ttyready=true err=<nil>
M13 raw read returned n=1 "\x00" err=<nil> eagain=false
M14 canonical flush err=<nil>; raw-on ready n=0 ttyready=false err=<nil>
```

**M11 to M14, this Mac:**

```text
M0 isatty termios err=<nil>
M11 line ready n=1 ttyready=true err=<nil>
M11 after ctrl-c select n=0 ttyready=false err=<nil>
M11 read on fd 0: blocked after 2s
M11 the blocked read ended with n=8 "unblock\n" err=<nil> eagain=false
M12 ttyname path-ok=true env-match=true err=<nil>
M12 open err=<nil> fd0 nonblock=false private nonblock=true
M12 idle read private n=-1 "" err=resource temporarily unavailable eagain=true
M12 select private n=1 ready=true err=<nil>
M12 read private n=3 "p1\n" err=<nil> eagain=false
M12 fd 0 after n=0 ready=false err=<nil>
M12 line2 ready n=1 ready=true err=<nil>
M12 read private after ctrl-c n=-1 "" err=resource temporarily unavailable eagain=true after=0s
M12 raw set-err=<nil> select private n=1 ready=true err=<nil>
M12 raw read private n=1 "r" err=<nil> eagain=false
M12 flush fd 0 err=<nil>; read private n=-1 "" err=resource temporarily unavailable eagain=true
M12 fd0 after nonblock=false
M13 canonical ready n=1 ttyready=true err=<nil>
M13 raw-on ready n=1 ttyready=true err=<nil>
M13 raw read returned n=1 "\x04" err=<nil> eagain=false
M14 canonical flush err=<nil>; raw-on ready n=0 ttyready=false err=<nil>
```

What they decide: select(2) waits, reports readiness and survives Ctrl+C on Linux as on macOS (D22's step 0: no stop);
no `EINTR` was seen on any host (M6, M9), so the retry is a guard only (R125, R133); poll(2) accepts `/dev/tty` on
Linux (M8); a hang-up is ready to select(2) and reads 0, or `EIO` in a read already waiting on Linux (M10, R148); a
read(2) of standard input after a line was reported ready can wait, if Ctrl+C empties the line first (M11, every
host: Q25); a private non-blocking descriptor on the terminal's path sees what is typed, gives `EAGAIN` with nothing
queued and after such a Ctrl+C, leaves standard input blocking, and follows standard input's raw mode and flush (M12,
every host: Q25 (a)); a Ctrl+D typed out of raw mode while nothing reads is one byte once raw mode is on, `0x04` on
macOS and `0x00` on Linux, not the end of input (M13, R89's latest row); a flush out of raw mode empties half a line
(M14: plant 86 dropped, R131 needs no "after raw mode").

### Records draft for D22, third draft (2026-11-21)

The delta confirmation found 27 of the review's 30 findings resolved and 3 open as owner questions (Q23, Q25, Q26).
On the coordinator's dispositions: statuses and counts (item 1); R127 and `EAGAIN` (2); R127's self-close dropped
(3); R147 on the gated path only (4); Q25 (a)'s flags, fallback, cost and amends lists (5); M15 (6); R89 (7); R145's
case and plant 92 (8); A45's two cases (9). Records only; no code. One deviation from item 5's wording, with
evidence: "the same device number" cannot tell whether standard input is the controlling terminal, because `/dev/tty`
reports its own device number on every host (M15, `rdev-equals-fd0=false`); Q25 (a) uses the foreground process
group instead, which M15 shows the same on both.

**M15**, the probe's `m15` run (the private descriptor opened `O_RDONLY|O_NONBLOCK|O_CLOEXEC|O_NOCTTY`; `/dev/tty`
with the same flags), each run exit 0. This Mac:

```text
M15 path ok=true err=<nil>
M15 open path with O_NOCTTY err=<nil> private nonblock=true fd0 nonblock=false
M15 fd0 rdev=268435474 private rdev=268435474 same-pgrp=true
M15 open /dev/tty with the same flags err=<nil> nonblock=true rdev=33554432 rdev-equals-fd0=false
M15 after both opens: controlling terminal pgrp=38403 (fd0 pgrp=38403)
M15 ctrl-d select private n=1 ready=true err=<nil>
M15 ctrl-d read private n=0 "" err=<nil> eagain=false
M15 ctrl-d read private again n=-1 "" err=resource temporarily unavailable eagain=true
M15 dev-tty select n=1 ready=true err=<nil>
M15 dev-tty read n=2 "q\n" err=<nil> eagain=false
M15 dev-tty idle read n=-1 "" err=resource temporarily unavailable eagain=true
M15 hangup select private n=1 ready=true err=<nil> after=400ms
M15 hangup read private n=0 "" err=<nil> eagain=false
M15 hangup read private again n=0 "" err=<nil> eagain=false
M15 hangup select dev-tty n=1 ready=true err=<nil>
M15 hangup read dev-tty n=0 "" err=<nil> eagain=false
```

A WSL distro:

```text
M15 path ok=true err=<nil>
M15 open path with O_NOCTTY err=<nil> private nonblock=true fd0 nonblock=false
M15 fd0 rdev=34818 private rdev=34818 same-pgrp=true
M15 open /dev/tty with the same flags err=<nil> nonblock=true rdev=1280 rdev-equals-fd0=false
M15 after both opens: controlling terminal pgrp=572 (fd0 pgrp=572)
M15 ctrl-d select private n=1 ready=true err=<nil>
M15 ctrl-d read private n=0 "" err=<nil> eagain=false
M15 ctrl-d read private again n=-1 "" err=resource temporarily unavailable eagain=true
M15 dev-tty select n=1 ready=true err=<nil>
M15 dev-tty read n=2 "q\n" err=<nil> eagain=false
M15 dev-tty idle read n=-1 "" err=resource temporarily unavailable eagain=true
M15 hangup select private n=1 ready=true err=<nil> after=400ms
M15 hangup read private n=0 "" err=<nil> eagain=false
M15 hangup read private again n=0 "" err=<nil> eagain=false
M15 hangup select dev-tty n=1 ready=true err=<nil>
M15 hangup read dev-tty n=0 "" err=<nil> eagain=false
```

A Linux machine that is not WSL:

```text
M15 path ok=true err=<nil>
M15 open path with O_NOCTTY err=<nil> private nonblock=true fd0 nonblock=false
M15 fd0 rdev=34817 private rdev=34817 same-pgrp=true
M15 open /dev/tty with the same flags err=<nil> nonblock=true rdev=1280 rdev-equals-fd0=false
M15 after both opens: controlling terminal pgrp=5499 (fd0 pgrp=5499)
M15 ctrl-d select private n=1 ready=true err=<nil>
M15 ctrl-d read private n=0 "" err=<nil> eagain=false
M15 ctrl-d read private again n=-1 "" err=resource temporarily unavailable eagain=true
M15 dev-tty select n=1 ready=true err=<nil>
M15 dev-tty read n=2 "q\n" err=<nil> eagain=false
M15 dev-tty idle read n=-1 "" err=resource temporarily unavailable eagain=true
M15 hangup select private n=1 ready=true err=<nil> after=400ms
M15 hangup read private n=0 "" err=<nil> eagain=false
M15 hangup read private again n=0 "" err=<nil> eagain=false
M15 hangup select dev-tty n=1 ready=true err=<nil>
M15 hangup read dev-tty n=0 "" err=<nil> eagain=false
```

The device numbers and process groups differ by host, as expected; every other word is the same on all three.

### P2, build pass 3 (2026-11-21)

*Opened, records only.* The owner chose D22, the owner, relayed by the coordinator: "Make reads cancellable on POSIX", over the recommended option; its stated cost was
that "it replaces D13's mechanism for Linux and macOS, so it needs a new decision, a records pass, new tests and
plants; Windows is unchanged; 0063 is delayed by at least one records pass and a build". That choice is the owner's
word for this third build pass, which the rule of no third pass without the owner otherwise forbids. The owner then
answered Q23 to Q26, the owner, relayed by the coordinator: each quoted as the option's label: "Flush at gobble's give-up (Recommended)", "Old reads plus one warning (Recommended)", "Private non-blocking descriptor (Recommended)",
"Move them to a shared file (Recommended)". D22 is accepted (MADR); P2's steps 15 to 20, A40 to A55 and plants 76 to 97 are the pass's scope,
with the shared pseudo-terminal file (Q26). Nothing is built yet: the coordinator gates these records first.

*(2026-11-21, the build, after the coordinator's gate of the accepted records.)*

**Outcome: built; every gate passes.** On the branch at `0eb985a`, nothing staged or committed.

**What was built** (steps 15 to 20 as answered):

* `internal/cli/term/cancelin.go` (`ErrNoGate`, `GatedOptions`, with the `Wait`, `Read` and `Returned` seams),
  `cancelin_unix.go` (`Gated`: `OpenGated`, `NewGated`, `Open`, `Close`, `Read`, `Stop`, `Descriptor`),
  `cancelin_other.go`, and, for the terminal's path, `cancelin_path_linux.go` (`/proc/self/fd`),
  `cancelin_path_darwin.go` (`F_GETPATH`) and `cancelin_path_other.go` (none: `/dev/tty`), three files the step's
  list did not name. The private descriptor is opened `O_RDONLY|O_NONBLOCK|O_CLOEXEC|O_NOCTTY`; `/dev/tty` with the
  same flags when the path fails and the foreground process group matches; select(2) on it and the wake pipe; `EINTR`
  retried above the `Wait` seam, so a test can give it; a read(2) only with the gate open and the terminal ready,
  under the gate's lock, `EAGAIN` back to the wait; the closed gate waits on the wake pipe alone; a close wakes the
  wait and returns after the waiter left it; the wake pipe non-blocking and close-on-exec, drained before each wait;
  `FD_SETSIZE` checked; a wait that fails falls back, once, to reading standard input as built, and every close then
  reports a read.
* `internal/cli/terminput.go`: `terminalSource` (a POSIX terminal through `Gated`; Windows through
  `ConsoleInput`; a pipe or a file as it is), with the `openGated` and `isTerminalFile` seams; `warnWait`, the owner's
  words once (`runEnv.waitWarn`, `main.go`); `fill` opens the gate for its read, asks `lineBuf` on the gated path,
  closes it, and marks a read waiting only when the close reported one, or always with no gate; `flushGiveUp`.
  `prompt.go` (`questions.cancelled`'s program branch) and `permission.go` (the withdrawn branch) flush at the
  give-up, on a gated input only.
* Q26: `internal/cli/pty_unix_test.go` (`(linux || darwin) && pty`) holds the cases, moved word for word from
  `pty_linux_test.go`, which keeps the Linux opener and `queued`; `pty_darwin_test.go` holds the macOS opener; the
  `Makefile`'s `probe-pty` comment, and `docs/guides/developing.md`'s row for it ("Linux and macOS"). After the move,
  as dated edits: A40 split (`pty-abandoned-empty-typed`, `-pasted`); `pty-abandoned` requires `hello` and `again` as
  turns (A43), so its shared tail now differs from the Windows copy, annotated; A44 `pty-menu-ctrlc-typed`; A45
  `pty-menu-ctrld`; A43 `pty-gaveup-held`, in the test process itself; A46's terminal part and A55 in
  `TestPtyPrivateDescriptor`, with a child on its own pseudo terminal for the `/dev/tty` fallback.
* Tests: `term/cancelin_test.go` (A41 `TestCancelableWait`, A52 `TestCancelableHangUp`, A50's part
  `TestCancelableFallsBack`, A54 `TestCancelableWakePipe`, A48 `TestCancelableConcurrent`, run here without `-race`);
  `internal/cli/gated_unix_test.go` (A42 `TestCancelableGiveUp`, A47 `TestCancelableTimedWait`, A51
  `TestPermissionTimedWaitOverLayer`, A50 `TestCancelableFallbackWarnsOnce`, A46's pipe part
  `TestTerminalInChoosesLayer`); `internal/cli/giveup_test.go` (A49 `TestFlushAtGiveUp`). Two lint fixes after the
  first runs: `closeFDs` for the descriptors' closes (`errcheck`), and `TestCancelableHangUp` split from
  `TestCancelableWait` (`gocognit`); a wrapper left unused by the A40 split removed; the package, `make probe-pty` on
  both systems and the pre-add check were run again after them.

**Gates:**

| Command | Exit | Output line |
| --- | --- | --- |
| `go test -count=1 -timeout 10m -json ./internal/cli/` | 1 | the same 33 failing leaf tests, the same two messages (30 `the offer again, [y/N]: not before the session's prompt`, 3 `the warning was printed 0 times; want 1`); `TestLineSessionReadsOnlyWhenAsked` passes; no change in the list |
| `go test -count=1 ./internal/cli/term/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/term 2.155s` |
| `go test -count=1 ./internal/cli/editor/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/editor 2.203s` |
| `make archtest` | 0 | `ok github.com/maccavelli/gobble-cli/internal/archtest 1.282s` (no new edge) |
| `make pre-add-check FILES="<the 42 changed .go files>"` | 0 | `go-precheck: 42 file(s) clean (gofmt, golangci-lint, govulncheck).` |
| `go vet ./internal/cli/... ./cmd/gobble/`; with `-tags pty` | 0, 0 | no output |
| `CGO_ENABLED=0 GOOS=linux go vet -tags pty ./internal/cli/...` | 0 | no output |
| `GOOS=windows go vet -tags conpty ./cmd/gobble/ ./internal/cli/...` | 0 | no output |
| `GOOS=freebsd go vet ./internal/cli/term/` | 0 | no output |
| `golangci-lint` (v2.15.0) with `pty` on darwin and on linux, with `conpty` on windows | 0 | `0 issues.` each |
| `run-on-host --need windows --label 0063-p2-conpty -- make probe-conpty` | 2 | 24 lines, the same as build pass 2, word for word |
| `make probe-pty`, this Mac | 2 | 11 lines, below |
| `run-on-host --need linux --race --label 0063-p2-pty -- make probe-pty` (the Linux machine that is not WSL; `--race` only keeps a run out of WSL distros) | 2 | 11 lines, below |

The Windows run was made before the two lint fixes, which change only `unix`-built files; the Linux and macOS runs
after them. The Windows test laptop:

```text
0063 case=probe-crlf result=pass observed=crlf
0063 case=probe-typeahead result=pass observed=one-line-per-read
0063 case=probe-ctrlc result=pass observed=eof first=read
0063 case=probe-ctrlz-line result=pass observed=eof
0063 case=probe-ctrlz-raw result=pass observed=eof
0063 case=report result=pass observed=waits
0063 case=masked result=pass observed=masked
0063 case=key-ctrlc result=fail observed=no-offer-again
0063 case=empty-key result=fail observed=no-offer-again
0063 case=menu-ctrlc result=fail observed=no-offer-again
0063 case=decline result=fail observed=no-offer-again
0063 case=typeahead result=pass observed=waits
0063 case=login-no-loop result=pass observed=no-offer
0063 case=method-menu result=pass observed=waits
0063 case=key-ctrlz result=pass observed=ended
0063 case=setup result=pass observed=cancelled
0063 case=acp-login result=pass observed=cancelled
0063 case=offer-ctrlc result=pass observed=warned-once
0063 case=setup-menu-ctrlc result=pass observed=ended
0063 case=nothing-saved result=pass observed=none
0063 case=flush-prompt result=pass observed=no-turn
0063 case=flush-offer result=fail observed=no-offer-again
0063 case=abandoned result=pass observed=dropped
0063 case=auth-paste-ctrlc result=pass observed=ended
```

This Mac:

```text
0063 case=pty-flush-prompt result=pass observed=no-turn
0063 case=pty-flush-offer result=fail observed=no-offer-again
0063 case=pty-abandoned result=pass observed=dropped
0063 case=pty-ctrlc-ctrld result=fail observed=no-offer-again
0063 case=pty-abandoned-empty-typed result=pass observed=none-lost
0063 case=pty-abandoned-empty-pasted result=pass observed=none-lost
0063 case=pty-menu-ctrlc-typed result=pass observed=no-ab
0063 case=pty-menu-ctrld result=pass observed=ended-at-prompt
0063 case=pty-gaveup-held result=pass observed=held
0063 case=pty-private-descriptor result=pass observed=private
0063 case=pty-dev-tty-fallback result=pass observed=dev-tty
```

The Linux machine that is not WSL:

```text
0063 case=pty-flush-prompt result=pass observed=no-turn
0063 case=pty-flush-offer result=fail observed=no-offer-again
0063 case=pty-abandoned result=pass observed=dropped
0063 case=pty-ctrlc-ctrld result=fail observed=no-offer-again
0063 case=pty-abandoned-empty-typed result=pass observed=none-lost
0063 case=pty-abandoned-empty-pasted result=pass observed=none-lost
0063 case=pty-menu-ctrlc-typed result=pass observed=no-ab
0063 case=pty-menu-ctrld result=pass observed=ended-at-prompt
0063 case=pty-gaveup-held result=pass observed=held
0063 case=pty-private-descriptor result=pass observed=private
0063 case=pty-dev-tty-fallback result=pass observed=dev-tty
```

Every line is as the P2 tree expects: Windows unchanged; `pty-flush-offer` and `pty-ctrlc-ctrld` read the P2b words
`no-offer-again`; every D22 case passes on both POSIX systems.

**Each new test seen to fail**, one plant at a time on a scratch copy of the tree, never the tree; plants 76 to 97
as the records assign them to P2's tests (85 is P4's, under `-race`; 86 is dropped). Each failure line:

```text
P76 the layer reads before readiness: exit 1: cancelin_test.go:105: 155641 reads before any data; want none
P77 a close does not wake the wait: exit 1: cancelin_test.go:146: the close did not return: the wait was not woken
P78 a close reports no read in progress: exit 1: cancelin_test.go:175: the close did not report the read
P79 EINTR ends the wait as an error: exit 1: cancelin_test.go:195: read "", EOF; want "i": EINTR retried
P80 a give-up with nothing read marks a drop: exit 1: gated_unix_test.go:85: the line was not read
P81 a program's give-up keeps a race completion: exit 1: gated_unix_test.go:201: the next line = "raced", <nil>; want "next"
P82 a Ctrl+C give-up drops a race completion: exit 1: gated_unix_test.go:201: the next line = "next", <nil>; want "raced"
P83 the layer used for a pipe: exit 1: gated_unix_test.go:323: a pipe warned:
P84 fill leaves the gate open when its read ends on its context: exit 1: gated_unix_test.go:117: 1 reads of the terminal with no prompt waiting; want none
P87 a 0-byte read gives (0, nil): exit 1: cancelin_test.go:222: read "", <nil>; want exactly (0, io.EOF)
P88 the closed gate returns an error: exit 1: gated_unix_test.go:86: the next line = "", the gate closed; want "next": nothing was read, so nothing is dropped
P89 fill asks with the caller's size: exit 1: gated_unix_test.go:159: after discard the read gave "pppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppp", <nil>; want "z" alon
P90 with no gate, waiting is never set: exit 1: gated_unix_test.go:171: the next line = "hello", <nil>; want "next"
P91 the wake pipe blocking: exit 1: cancelin_test.go:275: wake pipe end 4: flags 0x0, <nil>; want O_NONBLOCK
P92 the closed gate's wait includes the terminal: exit 1: gated_unix_test.go:136: the closed gate's wait returned 304696 times in 200 ms with bytes ready; want none
P93 the fallback warns at every cause: exit 1: gated_unix_test.go:293: 2 warnings for two causes; want one
P94 no flush at the program's give-up: exit 1: giveup_test.go:54: 0 flushes at the withdrawal; want one
P95 a Ctrl+C give-up flushes too: exit 1: giveup_test.go:91: 1 flushes at the person's give-up; want none
P97 a failed wait ends the input: exit 1: gated_unix_test.go:306: the prompt read "", cancelled (bad file descriptor); want "y" from standard input as built
P76 on pty-gaveup-held: exit 1: pty_unix_test.go:375: pty-gaveup-held: observed taken, want held
P96 the layer reads standard input itself: exit 1: pty_unix_test.go:398: pty-private-descriptor: observed blocking, want private
```

**Not done, and why:** `-race` (A48, plant 85) is P4's; P2b to P6 are not started.

**Docs:** `docs/guides/developing.md:100` fixed ("Linux and macOS"); the `Makefile`'s `probe-pty` comment fixed;
every new exported name in `internal/cli/term` has its comment. Still P5's: `docs/architecture.md:133`, `:135`, `:137`
(the term row now also lacks `Gated`), `internal/cli/term/doc.go` (now also `OpenGated`), `docs/guides/credentials.md`,
`README.md:120`, `docs/README.md`'s 0063 rows.

### P2, build pass 4 (2026-11-21)

*Opened, records only.* An independent review of build pass 3 found no blocker; five of its plants survived and one
case was undecided. The owner ruled, the owner, relayed by the coordinator: each quoted as the option's label: "Fourth pass, tests only (Recommended)", the owner's word
for this fourth pass, tests only, with no production change beyond the next answers; "Always flush on a POSIX terminal (Recommended)"; and the four
hardenings, "Retry EINTR on the read (Recommended)", "Refuse a blocking descriptor (Recommended)", "Write the warning safely" and "Check it's the same terminal". MADR D23 and rows R153 to R157;
steps 21 to 25, A56 to A64 and plants 98 to 107 above. There is no fifth pass. Nothing is built yet: the coordinator
gates these records first.

*(2026-11-21, the build, after the coordinator's gate of the D23 records.)*

**Outcome: built; every gate passes.** On the branch at `0eb985a`, nothing staged or committed.

**What was built** (steps 21 to 25, as D23 rules):

* Step 21 ("Always flush on a POSIX terminal (Recommended)"): `terminalInput.posix`, set by `terminalIn` from
  `runEnv.posixTerm`, which `terminalSource` sets on any terminal where `openGated` did not answer `ErrNoGate`:
  gated, or fallen back at start-up or later. `flushGiveUp` flushes on it, not on the gate (rows R80, R132, R157).
* Step 22 ("Retry EINTR on the read (Recommended)"): in `Gated.Read`, `EINTR` from the read returns to the wait,
  a check before `EAGAIN`'s (row R153).
* Step 23 ("Refuse a blocking descriptor (Recommended)"): `NewGated` reads `F_GETFL` first and refuses a
  descriptor without `O_NONBLOCK` (`errBlocking`); a refused descriptor stays the caller's. `GatedOptions` gains a
  `Pipe` seam, the wake pipe's maker, for A57 (row R154).
* Step 24 ("Check it's the same terminal"): `OpenGated` opens the path through `sameTerminal`, which compares
  `fstat`'s `Rdev` of standard input and of the opened descriptor. A failed open, a different terminal, or a
  blocking descriptor is "not opened", and the layer goes on to `/dev/tty` (`controllingTTY`, its
  foreground-process-group check unchanged), then R138; any other refusal of `NewGated`'s (the wake pipe,
  `FD_SETSIZE`) is R138 at once (rows R126, R138, R155).
* Step 25 ("Write the warning safely"): `OnFallback` is `queueWaitWarn`: the warning is queued
  (`runEnv.waitQueued`, under `waitMu`), once in the process with `warnWait`'s `sync.Once`, and `tellWaitWarn`
  writes it on the main flow, the words unchanged (`waitWarning`): in `questions.choose`, `confirm`,
  `inputContext` and `secret` before their text (`secret` before raw mode), in `beforePrompt` before the session
  prompt, in `askPermission` before raw mode, first in `askTrust`, and in `run` after the command returns. A
  fallback at start-up, made on the main flow in `terminalSource`, still warns at once (row R156).
* Tests: `term/cancelin_test.go`: A62 (`TestCancelableWait`'s case `EINTR from the read returns to the wait`, its
  body the named function `eintrReadWaitsAgain` for `gocognit`), and `TestGatedRefusals` (A57 and A63: the wake
  pipe, a descriptor at `FD_SETSIZE`, the wake pipe's read end at `FD_SETSIZE`, a blocking pipe).
  `internal/cli/gated_unix_test.go`: A56 `TestGiveUpFlushBySystem`; `TestGatedRefusals` through `terminalIn`
  (each refusal a fallback with the warning once, in the refusal's words); A58 `TestTerminalInWiresGate`; A60
  `TestFallbackWarningOnMainFlow` (before the next question's text; at the run's end, through `run`); A61
  `TestGatedRaceDropSize`; A52, as amended, `TestCancelableHangUp` through `terminalInput`. `pty_unix_test.go`: A59
  `pty-not-controlling` and A64 `pty-same-terminal` in `TestPtyPrivateDescriptor`, through `privateChild`, which
  `devTTYFallback` now uses, and a path passed to `TestPtyPrivateChild`.
* The two credit fixes: the closed gate's case of A42 notes that it is the one that catches a wake pipe not drained
  (plant 91, row R146); A52's case through `terminalInput`, above.
* Two tests changed with the rulings, by dated note in each: `gatedStub` (`giveup_test.go`) says its input is a
  POSIX terminal's, since the flush is keyed on that (step 21); A50's case `the wait fails` asks a question after the
  read that fell back and counts the warning there, since R156 moves the warning to the next question's text; its
  assertion is unchanged.
* Fixes after the first runs: two trailing comments in `prompt.go` moved above their lines (`gofmt`); A62's body made
  a named function (`gocognit`); the test helpers' open-files-limit code dropped, since FreeBSD's `Rlimit` fields
  are `int64` and Go raises the limit at start-up. Every gate below was run again after them.

**Gates:**

| Command | Exit | Output line |
| --- | --- | --- |
| `go test -count=1 -json ./internal/cli/` | 1 | the same 33 failing leaf tests as build pass 3 (30 `the offer again, [y/N]: not before the session's prompt`, 3 `the warning was printed 0 times; want 1`); 785 passes; no change in the list |
| `go test -count=1 ./internal/cli/term/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/term 2.468s` |
| `go test -count=1 ./internal/cli/editor/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/editor 1.839s` |
| `make archtest` | 0 | `ok github.com/maccavelli/gobble-cli/internal/archtest 1.139s` |
| `make pre-add-check FILES="<the 42 changed .go files>"` | 0 | `go-precheck: 42 file(s) clean (gofmt, golangci-lint, govulncheck).` |
| `CGO_ENABLED=0 go vet ./internal/cli/... ./cmd/gobble/`; with `-tags pty` | 0, 0 | no output |
| `CGO_ENABLED=0 GOOS=linux go vet -tags pty ./internal/cli/...` | 0 | no output |
| `CGO_ENABLED=0 GOOS=windows go vet -tags conpty ./internal/cli/... ./cmd/gobble/` | 0 | no output |
| `CGO_ENABLED=0 GOOS=freebsd go vet ./internal/cli/term/` | 0 | no output |
| `golangci-lint` (v2.15.0) with `pty` on darwin and on linux, with `conpty` on windows | 0 | `0 issues.` each |
| `run-heavy -- go test -race -count=1 ./internal/cli/term/`, this Mac | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/term 3.544s` |
| `run-on-host --need windows -- make probe-conpty` | 2 | 24 lines, the same as build pass 3, word for word |
| `make probe-pty`, this Mac | 2 | 13 lines, below |
| `run-on-host --need linux --race -- make probe-pty` (the Linux machine that is not WSL) | 2 | 13 lines, below |
| `make check-records` | 0 | after this entry |

The Windows test laptop:

```text
0063 case=probe-crlf result=pass observed=crlf
0063 case=probe-typeahead result=pass observed=one-line-per-read
0063 case=probe-ctrlc result=pass observed=eof first=read
0063 case=probe-ctrlz-line result=pass observed=eof
0063 case=probe-ctrlz-raw result=pass observed=eof
0063 case=report result=pass observed=waits
0063 case=masked result=pass observed=masked
0063 case=key-ctrlc result=fail observed=no-offer-again
0063 case=empty-key result=fail observed=no-offer-again
0063 case=menu-ctrlc result=fail observed=no-offer-again
0063 case=decline result=fail observed=no-offer-again
0063 case=typeahead result=pass observed=waits
0063 case=login-no-loop result=pass observed=no-offer
0063 case=method-menu result=pass observed=waits
0063 case=key-ctrlz result=pass observed=ended
0063 case=setup result=pass observed=cancelled
0063 case=acp-login result=pass observed=cancelled
0063 case=offer-ctrlc result=pass observed=warned-once
0063 case=setup-menu-ctrlc result=pass observed=ended
0063 case=nothing-saved result=pass observed=none
0063 case=flush-prompt result=pass observed=no-turn
0063 case=flush-offer result=fail observed=no-offer-again
0063 case=abandoned result=pass observed=dropped
0063 case=auth-paste-ctrlc result=pass observed=ended
```

This Mac:

```text
0063 case=pty-flush-prompt result=pass observed=no-turn
0063 case=pty-flush-offer result=fail observed=no-offer-again
0063 case=pty-abandoned result=pass observed=dropped
0063 case=pty-ctrlc-ctrld result=fail observed=no-offer-again
0063 case=pty-abandoned-empty-typed result=pass observed=none-lost
0063 case=pty-abandoned-empty-pasted result=pass observed=none-lost
0063 case=pty-menu-ctrlc-typed result=pass observed=no-ab
0063 case=pty-menu-ctrld result=pass observed=ended-at-prompt
0063 case=pty-gaveup-held result=pass observed=held
0063 case=pty-private-descriptor result=pass observed=private
0063 case=pty-dev-tty-fallback result=pass observed=dev-tty
0063 case=pty-not-controlling result=pass observed=refused
0063 case=pty-same-terminal result=pass observed=own-terminal
```

The Linux machine that is not WSL:

```text
0063 case=pty-flush-prompt result=pass observed=no-turn
0063 case=pty-flush-offer result=fail observed=no-offer-again
0063 case=pty-abandoned result=pass observed=dropped
0063 case=pty-ctrlc-ctrld result=fail observed=no-offer-again
0063 case=pty-abandoned-empty-typed result=pass observed=none-lost
0063 case=pty-abandoned-empty-pasted result=pass observed=none-lost
0063 case=pty-menu-ctrlc-typed result=pass observed=no-ab
0063 case=pty-menu-ctrld result=pass observed=ended-at-prompt
0063 case=pty-gaveup-held result=pass observed=held
0063 case=pty-private-descriptor result=pass observed=private
0063 case=pty-dev-tty-fallback result=pass observed=dev-tty
0063 case=pty-not-controlling result=pass observed=refused
0063 case=pty-same-terminal result=pass observed=own-terminal
```

Every line is as the P2 tree expects: Windows unchanged; `pty-flush-offer` and `pty-ctrlc-ctrld` read the P2b words
`no-offer-again`; every D22 and D23 case passes on both POSIX systems, `pty-not-controlling` and `pty-same-terminal`
among them.

**Each new test seen to fail**, one plant at a time on a scratch copy of the final tree, never the tree: plants 98
to 107, two of them in both packages, and two more of this pass's (X1, the run's-end write removed; X2, the
POSIX-terminal mark never set). Each failure line:

```text
98 the give-up flush on any input with a flush: exit 1: gated_unix_test.go:358: 1 flushes at the program's give-up; want 0
99 the FD_SETSIZE check off (term): exit 1: cancelin_test.go:380: NewGated(1024) = &{1024 4 7 { <nil> <nil> 0x102d4fce0 0x102d4c610 <nil> <nil>} {{} {0 0}} 0x4771b14dc9c0 false false false 0 false false {{} {0 0}}}, <nil>; want it refused at FD_SETSIZE
99 the FD_SETSIZE check off (cli): exit 1: gated_unix_test.go:396: NewGated took the descriptor; want it refused
100 the gate never wired: exit 1: gated_unix_test.go:461: the input's gate is <nil>; want the gated reader openGated gave
101 no process-group check before /dev/tty: exit 1: 0063 case=pty-not-controlling result=fail observed=dev-tty-opened
102 the caller's size marked: exit 1: gated_unix_test.go:557: the next 256-byte read gave "pppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppppp", <nil>; want "next": the 400 bytes droppe
103 EINTR from read(2) an error: exit 1: cancelin_test.go:244: read "", interrupted system call after 1 reads; want "r" after EINTR, then a read
104 a blocking descriptor taken (term): exit 1: cancelin_test.go:403: NewGated of a blocking pipe = &{6 4 7 { <nil> <nil> 0x10063bd40 0x100638610 <nil> <nil>} {{} {0 0}} 0x6716253129c0 false false false 0 false false {{} {0 0}}}, <nil>; want it refused
104 a blocking descriptor taken (cli): exit 1: gated_unix_test.go:396: NewGated took the descriptor; want it refused
105 no Rdev check: exit 1: 0063 case=pty-same-terminal result=fail observed=other-terminal
106 the warning written from the goroutine: exit 1: gated_unix_test.go:484: the warning was written while the read that fell back ran:
107 the give-up flush keyed on the gate: exit 1: gated_unix_test.go:358: 0 flushes at the program's give-up; want 1
X1 no warning at the run's end: exit 1: gated_unix_test.go:529: the warning shown 0 times at the run's end; want once:
X2 the give-up's posix mark never set: exit 1: gated_unix_test.go:358: 0 flushes at the program's give-up; want 1
```

**The review's own plant runner**, unchanged (its script and plant list copied, compared byte for byte), against a
scratch copy of the final tree, with the review's list of failing tests:

```text
A  (cli): the runner stops: AssertionError ('A', 'old text count', 0). Its line, flushGiveUp's gate check,
   is the one step 21 replaced; plant 98 is the same mutation on the new line, caught above.
B  (term): exit=1, 2 new failing leaves: TestGatedRefusals/a_descriptor_at_FD_SETSIZE,
   TestGatedRefusals/the_wake_pipe's_read_end_at_FD_SETSIZE. Caught.
C  (term): exit=1: TestCancelableWait/a_close_wakes_the_wait_and_returns_after_the_waiter_left_it. Caught.
D  (cli): exit=1: TestTerminalInWiresGate. Caught.
E  (term): exit=0, none: not caught there, as at build pass 3; R146 credits the closed gate's case (E2).
E2 (cli): exit=1: TestCancelableGiveUp/a_closed_gate's_wait_does_not_return_for_bytes_ready. Caught.
F  (term, default suite): exit=0, none: no default-suite test reaches /dev/tty; plant 101 is A59's, a
   pseudo-terminal case (Fpty).
Fpty (cli, -tags pty): exit=1: pty_unix_test.go:402: pty-not-controlling: observed dev-tty-opened, want refused. Caught.
G  (cli): exit=1: TestCancelableFallbackWarnsOnce/both_opens_fail (a nil-pointer panic). Caught.
H  (cli): exit=1: TestGatedRaceDropSize. Caught.
I  (term): exit=1: TestCancelableWait/a_close_during_a_read_waits_for_it_and_reports_it. Caught.
Dpty (cli, -tags pty): exit=1: pty_unix_test.go:396: pty-private-descriptor: observed no-gate-chosen, want private. Caught.
```

**Not done, and why:** P2b to P6 are not started.

**Docs:** every new name has its comment (`GatedOptions.Pipe`, `errBlocking`, `sameTerminal`, `gatedOn`,
`notOpened`, `waitWarning`, `queueWaitWarn`, `tellWaitWarn`, `warnQueued`). Still P5's, as build pass 3 lists them;
no guide describes Q24's warning yet, so P5's write-up of it carries R156's place for it.

### P2, build pass 5 (2026-11-21)

*Opened, records first.* The re-review of build pass 4 found one blocker, which the coordinator confirmed: R156's
"with the terminal in its cooked state" does not hold at the session prompt. `beforePrompt` wrote a queued warning,
and at `beforePrompt` in the session loop the terminal is always raw: the session enters raw mode before the loop
and returns to it after every turn through `outOfRaw`. Raw mode clears `OPOST`, and the warning ends with a bare
`\n`, so a fallback in the middle of a session would print its warning as a staircase just before the editor's
prompt. The owner approved a fifth pass for this fix only, the owner, relayed by the coordinator: "Fifth pass, this fix only (Recommended)". There
is no sixth pass without the owner.

* **The fix:** at the session prompt, a queued warning is written out of raw mode through the session's existing
  `outOfRaw` step, before the editor's prompt; `beforePrompt` no longer writes it. The question paths and the
  write at the run's end stay as they are. No other production change. Of the two ways the review named, the
  write moved into `outOfRaw` after a turn would leave a warning queued during the first run's questions unwritten
  until after the first turn, past the session prompt's text, so the write goes through `outOfRaw` from the loop,
  only when a warning is queued.
* **Tests:** `TestFallbackWarningOnMainFlow` gains two cases. At the session prompt: raw mode stubbed and counted,
  a warning queued as a fallback queues it, and the terminal not raw when the warning is written; once. Written
  once: a fallback, then two later questions, and the warning counted exactly once (the review's plant P, the
  queue not cleared, survives build pass 4 and must fail it).
* **Plants**, on a scratch copy only: the warning written while raw (`beforePrompt`'s write as build pass 4 made it)
  must fail the session-prompt case; plant P must fail the once case.
* **Plant M, recorded and not tested:** the review's plant M leaves the descriptor open on an `Rdev` mismatch in
  `sameTerminal`. It is a leak of one descriptor, once in the process, with no effect a test can see: the layer goes
  on to `/dev/tty` and reads as it should. It is recorded here and not tested.
* **Gate (D23, as pass 4):** `./internal/cli/` with exactly the 33 loop leaves; `term` and `editor`; `archtest`;
  `pre-add-check` on every changed `.go` file; the five vets; `make check-records`;
  `run-heavy -- go test -race -count=1 ./internal/cli/term/` on this Mac; `make probe-pty` on this Mac; the
  review's own plant runner, with its added plants, on a scratch copy.

*(2026-11-21, the build, with no stop after the records, as the coordinator directed.)*

**Outcome: built; every gate passes.** On the branch at `0eb985a`, nothing staged or committed.

**What was built:** `runEnv.tellWaitWarnOutOfRaw` (`terminput.go`): when a warning is queued (`waitPending`), it is
written through the session's `outOfRaw`; with none, raw mode is left alone. The session loop (`line.go`) calls it
before `beforePrompt`, which no longer writes the warning. The question paths and the write at the run's end are as
build pass 4 left them. The call is a helper, not two `if`s in the loop, because `(*lineSession).run` passed
`gocognit`'s limit (52 of 50) with them; the pre-add check was run again after the move.

**Tests:** `TestFallbackWarningOnMainFlow` gains `written once over two later questions` (a fallback, then two later
questions, the warning counted once) and `at the session prompt, out of raw mode` (the line harness's raw mode
stubbed and counted, the credential set so the session prompt comes first, a warning queued as a fallback queues it,
and stderr's write of it noting how many raw modes are not restored: none, and once).

**Plants**, on a scratch copy of the final tree, never the tree:

```text
W the warning written while raw: exit 1: gated_unix_test.go:541: the warning was written with the terminal raw (1 raw modes not restored); want it out of raw mode
P the queue not cleared: exit 1: gated_unix_test.go:519: the warning shown 2 times over two later questions; want once:
```

**Gates:**

| Command | Exit | Output line |
| --- | --- | --- |
| `go test -count=1 -json ./internal/cli/` | 1 | the same 33 failing leaf tests as build passes 3 and 4; 787 passes |
| `go test -count=1 ./internal/cli/term/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/term 2.148s` |
| `go test -count=1 ./internal/cli/editor/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/editor 1.841s` |
| `make archtest` | 0 | `ok github.com/maccavelli/gobble-cli/internal/archtest 1.138s` |
| `make pre-add-check FILES="<the 42 changed .go files>"` | 0 | `go-precheck: 42 file(s) clean (gofmt, golangci-lint, govulncheck).` |
| `CGO_ENABLED=0 go vet ./internal/cli/... ./cmd/gobble/`; with `-tags pty` | 0, 0 | no output |
| `CGO_ENABLED=0 GOOS=linux go vet -tags pty ./internal/cli/...` | 0 | no output |
| `CGO_ENABLED=0 GOOS=windows go vet -tags conpty ./internal/cli/... ./cmd/gobble/` | 0 | no output |
| `CGO_ENABLED=0 GOOS=freebsd go vet ./internal/cli/term/` | 0 | no output |
| `make check-records` | 0 | after this entry |
| `run-heavy -- go test -race -count=1 ./internal/cli/term/`, this Mac | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/term 3.327s` |
| `make probe-pty`, this Mac | 2 | 13 lines, below, the same as build pass 4 |

This Mac:

```text
0063 case=pty-flush-prompt result=pass observed=no-turn
0063 case=pty-flush-offer result=fail observed=no-offer-again
0063 case=pty-abandoned result=pass observed=dropped
0063 case=pty-ctrlc-ctrld result=fail observed=no-offer-again
0063 case=pty-abandoned-empty-typed result=pass observed=none-lost
0063 case=pty-abandoned-empty-pasted result=pass observed=none-lost
0063 case=pty-menu-ctrlc-typed result=pass observed=no-ab
0063 case=pty-menu-ctrld result=pass observed=ended-at-prompt
0063 case=pty-gaveup-held result=pass observed=held
0063 case=pty-private-descriptor result=pass observed=private
0063 case=pty-dev-tty-fallback result=pass observed=dev-tty
0063 case=pty-not-controlling result=pass observed=refused
0063 case=pty-same-terminal result=pass observed=own-terminal
```

**The review's own plant runner**, unchanged (its script, its plant list with the added plants, and its list of
failing tests, copied and compared byte for byte), against a scratch copy of the final tree:

```text
A1   (cli): caught: TestGiveUpFlushBySystem/a_gateless_input_that_is_not_a_POSIX_terminal
A2   (cli): caught: TestGiveUpFlushBySystem/a_fallback_at_start-up
B    (term): caught: TestGatedRefusals/a_descriptor_at_FD_SETSIZE, /the_wake_pipe's_read_end_at_FD_SETSIZE
C    (term): caught: TestCancelableWait/a_close_wakes_the_wait_and_returns_after_the_waiter_left_it
D    (cli): caught: TestTerminalInWiresGate
Dpty (cli, pty): caught: pty-private-descriptor: observed no-gate-chosen, want private
E    (term): not caught, as at passes 3 and 4; R146 credits E2
E2   (cli): caught: TestCancelableGiveUp/a_closed_gate's_wait_does_not_return_for_bytes_ready
F    (term): not caught, as at pass 4; plant 101 is A59's (Fpty)
Fpty (cli, pty): caught: pty-not-controlling: observed dev-tty-opened, want refused
G    (cli): caught: TestCancelableFallbackWarnsOnce/both_opens_fail (a nil-pointer panic)
H    (cli): caught: TestGatedRaceDropSize
I    (term): caught: TestCancelableWait/a_close_during_a_read_waits_for_it_and_reports_it
J    (term): caught: TestCancelableWait/EINTR_from_the_read_returns_to_the_wait
K    (term): caught: TestGatedRefusals/a_blocking_descriptor
L    (term): not caught there; plant 105 is A64's, a pseudo-terminal case (Lpty)
Lpty (cli, pty): caught: pty-same-terminal: observed other-terminal, want own-terminal
M    (term): not caught, recorded above: a one-time descriptor leak no test can see
Mpty (cli, pty): not caught, the same
N    (cli): caught: TestGiveUpFlushBySystem/a_fallback_at_start-up
N2   (cli): caught: TestGiveUpFlushBySystem/a_gateless_input_that_is_not_a_POSIX_terminal
O    (cli): caught: TestFallbackWarningOnMainFlow/before_the_next_question's_text
P    (cli): caught: TestFallbackWarningOnMainFlow/written_once_over_two_later_questions
Q    (cli): caught: TestFallbackWarningOnMainFlow/at_the_run's_end
R    (cli): not caught (exit 1 from the 33 loop leaves alone, no new one): queueWaitWarn without the
     sync.Once; no reachable path queues after warnWait, since a start-up fallback has no gate to call
     OnFallback, and a gate calls it once
```

**Not done, and why:** the Windows and Linux pseudo-terminal runs were not asked for this pass: Windows does not build
this path differently (its vet is above), and the session prompt's write is common code. P2b to P6 are not started.

*(2026-11-21, after the fix-only check, which found no blocker.)* Plant R, `queueWaitWarn` without its `sync.Once`, is recorded as an equivalent mutant, decided by the coordinator on evidence: `OnFallback` is called only from `Gated.fallBack`, for the first fallback only, under `g.mu` (`term/cancelin_unix.go:348-357`); `terminalIn` makes one input, so there is one `Gated` (`gatedOn`, `term/cancelin_unix.go:91-105`); and `warnWait` is called only when `openGated` fails, when there is no `Gated` (`terminput.go:132-134`). So `OnFallback` can neither fire twice nor follow `warnWait`; the coordinator's trace and the independent check found the same. It is not tested.

### P2b, build pass 1 (2026-11-21)

**Outcome: stopped on a finding (F5), before P2b's own tests and plants.** On the branch at `d2cc403`, nothing staged
or committed. The coordinator sent the phase after P2's commit: steps 1 to 4, the mark moved into `ls.signInRound`
(R103), every item P2's records list as waiting for P2b, and D22 and D23 where they touch the loop.

**What was built** (steps 1, 2 and 4, `internal/cli/login.go` alone):

* `ls.signInRound(want, turn) (signedIn bool, err error)`: `ls.login`'s body, with the mark (R103). Step 1 writes
  its result as `bool`; it also returns the after-steps' error, the `/models` turn's, which `ls.login` returns today
  and `firstRun` passes on to the session, so step 3's "nothing else changes" holds. `ls.login` keeps its refusal of
  an unknown provider before any question, and calls the round once.
* `firstRun`: the loop of step 2, in its order, with step 4's `flushNext()` beside `discard()` from the second round
  on; the offer's default true in the first round only (Q9); `ended()` or not yes after the offer, and `ended()`
  after a round, end it with the lead and the warning once (a helper, `noCredential`); a round that signed in
  returns its error; the run's own end returns `context.Cause(e.ctx)`. No round counter. `firstRun` still marks the
  first prompt however it ends (R104).

**First run of the package** (`go test -count=1 -timeout 2m -json ./internal/cli/`, exit 1): the 33 loop leaves
pass; 830 passes; one leaf fails, outside P2b's file list:

```text
TestConsentWordingOnEverySurface/the_first_run
    consentwording_test.go:173: the line session did not end; stdout "", stderr "No provider has a credential yet.
    Sign in to one now? [Y/n]: Sign in to which provider? [...] Sign in to one now? [y/N]: "
```

#### F5 (2026-11-21, P2b's build pass 1): a test outside the file list assumes one round

* **Evidence.** `internal/cli/consentwording_test.go:160-177` (0052's, last changed in `cecc892`), its case "the
  first run": the first offer answered with Enter, OpenAI's plan chosen, the wizard's consent cancelled, so the
  round fails (`openai sign-in failed (other)`); then `\x04`, meant for the session prompt. With D8's loop the
  repeated offer, `[y/N]`, comes first; on the harness's pipe `\x04` is a byte, not the end of input, so the offer's
  line read waits and the harness's 10 s bound ends the test. It passed at `d2cc403` (not among P2's 33 leaves), so
  the loop alone turns it red. It is not a loop test and not in P2b's file list (`login.go`, `firstrun_test.go`),
  and no record names it: the stop rule's "anything unexpected".
* **Resolutions**, recommended first:
  1. Answer the repeated offer in that case: after the failure's line, wait for `Sign in to one now? [y/N]: ` and
     press `n`, Enter, then `\x04` at the session prompt. About three lines; the case's assertions (the wording
     asked before any URL, no URL opened, no request beyond discovery) are unchanged; the file is added to P2b's
     list by a dated note. Cost: one records amendment.
  2. End the case's input (`h.keys.Close()`) in place of `\x04`: the loop ends at the offer as the end of input.
     Also about three lines, but the case then ends through R68's path rather than a person's answer.
* **Doing nothing:** the package cannot pass its gate (`go test ./internal/cli/` exit 0), so P2b cannot finish.

**Not done, and why:** A24's repeated-offer case, the test for the warning before the repeated offer's text, the
plants (9 to 14, 22, 44, 45, 57), the gate and the probe cases on the three hosts: they wait for the ruling on F5.

### P2b, build pass 2 (2026-11-21)

*Opened, records first.* The owner ruled on F5, the owner, relayed by the coordinator: "Answer the repeated offer (Recommended)": `consentwording_test.go`'s case "the
first run" answers the repeated offer (P2b's item 5 above), and the same answer accepts `signInRound`'s signature
(item 6). This pass finishes P2b as briefed: A24's repeated-offer flush case; a test that a queued fallback warning
is written before the repeated offer's text (D23, row R156, in the loop), with a harness option that gives the
harness's input the session's warning writer; plants 9 to 14, 22, 44, 45 and 57 on a scratch copy; the full gate;
and the probe cases on the Windows test laptop, this Mac and the Linux machine that is not WSL.

*(2026-11-21, the build.)*

**Outcome: built; every gate passes; P2b is done.** On the branch at `d2cc403`, nothing staged or committed.

**What this pass did:**

* `internal/cli/login.go`: as build pass 1 left it (steps 1, 2 and 4; item 6's signature); no change in this pass.
* `internal/cli/consentwording_test.go`: F5's resolution (item 5): the first run's case waits for the repeated offer,
  answers `n`, Enter, and sends `\x04` at the session prompt; its assertions unchanged.
* `internal/cli/flushpoints_test.go`: A24's case "repeated offer" (rows R79, R80): no flush at the first offer or
  inside the round; one at the repeated offer, after its text is written; one more at the first prompt; two in all.
  Its body is the named function `flushAtRepeatedOffer`, since `TestFlushPoints` passed `gocognit`'s limit (54 of
  50) with it inline; the pre-add check and the gate were run again after the move.
* `internal/cli/firstrun_test.go`: `TestFirstRunWarningBeforeRepeatedOffer` (D23, row R156, in the loop): a warning
  queued during a round is shown once, after the cancel line and before the repeated offer's text, and not again at
  the session prompt.
* `internal/cli/line_test.go`: the harness option the brief named, `lineHarness.tell`, which gives the harness's
  input the warning writer `terminalIn` gives the process's (`tellWaitWarn`); a helper, no test changed.

**Each plant red**, one at a time on a scratch copy, never the tree (44 and 45 again after the `gocognit` move): P3's
plants 9 to 14, 22, 44, 45 and 57, and two of this pass's (X3, the repeated offer's question writing no queued
warning; X4, F5's case as it was). Plant 22 spins, each spin paced by 1 ms so its output stays small; the harness's
bound catches it.

```text
9 a no at the repeated offer starts another round: exit 1: firstrun_test.go:216: the warning: not within 5 s; stderr:
10 ended() ignored: exit 1: firstrun_test.go:314: the offer was made 2 times; want 1
11 the loop ends on a cancelled round: exit 1: firstrun_test.go:131: the offer again, [y/N]: not before the session's prompt; stderr:
12 no discard before a repeated offer: exit 1: firstrun_test.go:250: the offer again, [y/N]: not within 5 s; stderr:
13 the repeated offer defaults to yes: exit 1: firstrun_test.go:214: the offer again, [y/N]: not within 5 s; stderr:
14 the warning printed between rounds: exit 1: firstrun_test.go:135: the warning was printed 2 times; want 1
22 the end of input at an offer taken as a cancelled round (each spin paced 1 ms, so the buffer stays small): exit 1: firstrun_test.go:355: the line session did not end; stdout "", stderr "No provider has a credential yet. Sign in to one now? [Y/n]: Sign in to which provider?\n  1  Gemini (Google) (gemini)     none — /login gemini\n  2  OpenAI (openai)    
44 no flush before a repeated offer: exit 1: flushpoints_test.go:412: condition not met within 5 s
45 the repeated offer's flush before its text: exit 1: flushpoints_test.go:415: the repeated offer's flush came at stderr 702, before its text ended at 763:
57 a cancelled or failed round sets no mark: exit 1: flushpoints_test.go:163: condition not met within 5 s
X3 the question writes no queued warning before its text: exit 1: firstrun_test.go:507: the fallback warning at 849; want it after the cancel line (586) and before the repeated offer (624):
X4 F5's case as it was (no answer to the repeated offer): exit 1: consentwording_test.go:177: the line session did not end; stdout "", stderr "No provider has a credential yet. Sign in to one now? [Y/n]: Sign in to which provider?\n  1  Gemini (Google) (gemini)     none — /login gemini\n  2  OpenAI (opena
```

**Gates:**

| Command | Exit | Output line |
| --- | --- | --- |
| `go test -count=1 -timeout 2m -json ./internal/cli/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli 55.709s`; no failing leaf; 834 passes, the 33 loop leaves among them |
| `go test -count=1 ./internal/cli/term/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/term 2.129s` |
| `go test -count=1 ./internal/cli/editor/` | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/editor 1.807s` |
| `make archtest` | 0 | `ok github.com/maccavelli/gobble-cli/internal/archtest` (cached: no package it reads changed since the uncached run of this pass, `1.276s`) |
| `make pre-add-check FILES="<the 5 changed .go files>"` | 0 | `go-precheck: 5 file(s) clean (gofmt, golangci-lint, govulncheck).` |
| `CGO_ENABLED=0 go vet ./internal/cli/... ./cmd/gobble/`; with `-tags pty` | 0, 0 | no output |
| `CGO_ENABLED=0 GOOS=linux go vet -tags pty ./internal/cli/...` | 0 | no output |
| `CGO_ENABLED=0 GOOS=windows go vet -tags conpty ./internal/cli/... ./cmd/gobble/` | 0 | no output |
| `CGO_ENABLED=0 GOOS=freebsd go vet ./internal/cli/term/` | 0 | no output |
| `make check-records` | 0 | after this entry |
| `run-heavy -- go test -race -count=1 ./internal/cli/term/`, this Mac | 0 | `ok github.com/maccavelli/gobble-cli/internal/cli/term 3.292s` |
| `run-on-host --need windows -- make probe-conpty` | 0 | 24 lines, every one `pass` |
| `make probe-pty`, this Mac | 0 | 13 lines, every one `pass` |
| `run-on-host --need linux --race -- make probe-pty` (the Linux machine that is not WSL) | 0 | 13 lines, every one `pass`, the same as this Mac's |

The Windows and Linux runs were made before the `gocognit` move, which only moved a test's body into a function;
this Mac's after it. The Windows test laptop:

```text
0063 case=probe-crlf result=pass observed=crlf
0063 case=probe-typeahead result=pass observed=one-line-per-read
0063 case=probe-ctrlc result=pass observed=eof first=read
0063 case=probe-ctrlz-line result=pass observed=eof
0063 case=probe-ctrlz-raw result=pass observed=eof
0063 case=report result=pass observed=waits
0063 case=masked result=pass observed=masked
0063 case=key-ctrlc result=pass observed=offer-again
0063 case=empty-key result=pass observed=offer-again
0063 case=menu-ctrlc result=pass observed=offer-again
0063 case=decline result=pass observed=warned-once
0063 case=typeahead result=pass observed=waits
0063 case=login-no-loop result=pass observed=no-offer
0063 case=method-menu result=pass observed=waits
0063 case=key-ctrlz result=pass observed=ended
0063 case=setup result=pass observed=cancelled
0063 case=acp-login result=pass observed=cancelled
0063 case=offer-ctrlc result=pass observed=warned-once
0063 case=setup-menu-ctrlc result=pass observed=ended
0063 case=nothing-saved result=pass observed=none
0063 case=flush-prompt result=pass observed=no-turn
0063 case=flush-offer result=pass observed=offer-waits
0063 case=abandoned result=pass observed=dropped
0063 case=auth-paste-ctrlc result=pass observed=ended
```

This Mac, and the Linux machine that is not WSL (the same lines):

```text
0063 case=pty-flush-prompt result=pass observed=no-turn
0063 case=pty-flush-offer result=pass observed=offer-waits
0063 case=pty-abandoned result=pass observed=dropped
0063 case=pty-ctrlc-ctrld result=pass observed=warned-once
0063 case=pty-abandoned-empty-typed result=pass observed=none-lost
0063 case=pty-abandoned-empty-pasted result=pass observed=none-lost
0063 case=pty-menu-ctrlc-typed result=pass observed=no-ab
0063 case=pty-menu-ctrld result=pass observed=ended-at-prompt
0063 case=pty-gaveup-held result=pass observed=held
0063 case=pty-private-descriptor result=pass observed=private
0063 case=pty-dev-tty-fallback result=pass observed=dev-tty
0063 case=pty-not-controlling result=pass observed=refused
0063 case=pty-same-terminal result=pass observed=own-terminal
```

Every case reads its word for after P2b: `flush-offer` and `pty-flush-offer` `offer-waits`; `key-ctrlc`,
`empty-key` and `menu-ctrlc` `offer-again`; `decline` and `pty-ctrlc-ctrld` `warned-once`; the rest as P2 left them.
No test shows the no-credential warning more than once; no bound was raised; no loop test changed.

**Not done, and why:** P3 to P6 are not started.

**Docs:** `login.go`'s comments for `login`, `signInRound`, `firstRun` and `noCredential`. The guides' account of the
first run (the offer made again after a failed or cancelled sign-in) is P5's.
