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
  `flushNext()`, and clears the mark (rows R82, R103 to R108). The repeated offer's `flushNext()` (P2b step 4) is
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
  `/logout`, after a permission prompt, or at a resumed session's first prompt with a credential; none when the
  session ends by Ctrl+D with the mark set. Plants: P3 56, 57, 58, 61.
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
| 46 | after a flush, a waiting read's completion is given to the prompt | `abandoned`, `pty-abandoned` (A21) | R83 |
| 47 | the line of a prompt that gave up is given to the next taker | `terminput_test` abandoned row (A28), `abandoned`, `pty-abandoned` | R84, R85 |
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
| 61 | the session flushes on its way out when the mark is set | `TestFlushPoints` | R108 |

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
