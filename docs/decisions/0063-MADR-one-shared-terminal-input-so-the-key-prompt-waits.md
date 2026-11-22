---
status: accepted
date: 2026-11-19
decision-makers: repository owner
consulted: "this repository at 21be074 (branch p3-acpext-v0.5.0, equal to main); go-llmprovider-sdk v1.31.0 (wizard/configure.go, wizard/auth.go); go-tui-lib v2.12.1 (inline/reader.go, termkey/input.go); Go 1.27.4 (src/internal/poll/fd_windows.go); an independent diagnosis's scratch experiments, not committed"
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# One shared terminal input for every prompt in a session, so a menu answered with CR+LF no longer skips the API key prompt

*(2026-11-21, status note: accepted 2026-11-19; built through P5 of the plan, ready for P6, the owner's check on a Windows console. No decision or asserted fact changes.)*
*(2026-11-21, later, status note: P6 passed, the owner's check on a real Windows console in PowerShell 5.1 and 7; Confirmation item 5 met; the plan is `complete`. The release is the owner's decision, Q8.)*

## Context and Problem Statement

The owner reported on 2026-11-19: "gobble-cli, or llmprovider-sdk has a bad bug in initial setup. if a user has no env
variables defined for API KEYs for any provider, when you run gobble the first time, it asks if you want to setup a
provider, as it should. you choose "y" and it opens the list of providers, if you select any provider that requires an
API key, when you choose the menu item selection number, eg: 1 for gemini, it skips prompting for the API KEY entirely,
and goes to the error page saying there is no key defined." It was seen on a Windows laptop, in PowerShell 5.1 and
PowerShell 7.x terminals.

An independent diagnosis, whose claims this record re-derived from the code at `21be074` (the evidence index below),
found the defect in gobble alone; the SDK asks for the key correctly.

**What happens.** The first-run offer, the provider menu and any method menu are read as lines out of raw mode
(0025-PLAN deviation 7). A Windows console's line-mode read returns the line with `\r\n`. `lineContext` ends the answer
at the `\r`, sets `afterCR` and leaves the `\n` in `q.pending`. The SDK's wizard then asks for the key through
`prompter.Secret`; `readSecret`, in raw mode, takes the leftover `\n` as Enter on an empty key, since it never looks at
`afterCR`. `prompter.Secret` turns the empty answer into `errCancelled`, the line session prints "Sign-in cancelled;
nothing was saved.", and the user's first prompt fails with `Error: no model credential: set one of GEMINI_API_KEY, …`.
The defect came with the sign-in itself, `978d38d` (2026-10-20), and is in every release from `v0.2.0` to `v0.6.0`.

**Why no test caught it.** Every sign-in test feeds `\r` alone (`internal/cli/line_test.go:126`, `login_test.go`), none
the `\r\n` a Windows console's line read gives; 0025-PLAN P5's pseudo-terminal run was on macOS; its Windows runs were
`go test` over ssh, with no console.

**Neighbouring weaknesses**, from the diagnosis and from this record's reading:

* **(a) Read-ahead is dropped between asks.** `ls.ask` builds a fresh `questions` for each ask (`login.go:226`), so
  whatever was read past one answer is lost before the next ask; the first run is two asks, the offer and then
  `/login`'s. `gobble setup`, `gobble acp --login` and `gobble auth login` each make their own `inline.Reader`
  (`login.go:96`, `auth.go:301`); the trust question reads `e.stdin` directly (`trust.go:184`); the permission prompt
  and the line editor read the session's `inline.Reader` and keep their own decoded read-ahead
  (`permission.go:225-230`; `editor/editor.go`, `held`).
* **(b) An empty key is worded as a deliberate cancel,** and after a first-run sign-in that is cancelled the session
  does not repeat the no-credential warning (`line.go:137-141` skips it when the offer is made; `login.go:253-255`;
  `login.go:300-316`), so the user learns of it only from the next turn's error.
* **(c) A converted line end can answer two questions.** On Unix, a scripted `y\r\n` into a pseudo-terminal with its
  normal conversion arrived as two lines: the empty one took the menu's default, and the next line, `1`, was saved as
  Gemini's key, unchecked because the listing was offline. Seen with scripted input only, not with typing.
* **(d) Found by this record's reading, not run:** `afterCR` is also set when a secret ends at `\r` in raw mode
  (`prompt.go:272-274`). If the same sign-in then asks a line question, such as Claude's workspace when a key needs one
  (go-llmprovider-sdk `wizard/configure.go:463`), a terminal in cooked mode on macOS or Linux turns the user's Enter
  into `\n`, and `lineContext` drops it (`prompt.go:181-183`): the first Enter is swallowed.

**The owner's decision.** Asked which fix this record should propose, the owner chose "One shared input layer for every
prompt in a session". As it was put to the owner: all prompts share one read-ahead buffer and one CR+LF rule, which also
fixes weakness (a); a moderate change to the line session's plumbing; it amends an earlier gobble record. The
coordinator had recommended the smaller option A below; the owner chose B.

### Evidence index

Re-derived at `21be074`, 2026-11-19:

| Claim | Where |
| --- | --- |
| A line ends at the first `\r` or `\n`; after a `\r`, `afterCR` is set and the rest kept in `pending`; a `\n` is dropped only by the next `lineContext` | `internal/cli/prompt.go:178-197` |
| `readSecret` takes the next byte of `pending`, `\r` or `\n` ends it, and it never reads `afterCR` | `prompt.go:251-294`, the line end at `:272-274` |
| An empty secret is `errCancelled` | `prompt.go:68-75` |
| Questions keep `pending` and `afterCR` per `questions` value | `prompt.go:30-41` |
| `fill` reads up to 512 bytes into `pending` | `prompt.go:199-208` |
| A fresh `questions` per ask | `internal/cli/login.go:212-231` (`:226`) |
| `/login`'s provider menu, then the wizard | `login.go:74-90` |
| The cancel line, and nothing after it | `login.go:252-255` |
| The next turn's error with no credential is `no model credential: set one of …` | `internal/cli/credentials.go:240-243`; `internal/cli/modelflags.go:82-92` |
| The first run is two asks; the warning is printed only when the offer is declined | `login.go:300-316`; `internal/cli/line.go:137-141`, `:163-171` |
| `gobble setup` and `gobble acp --login` build their own reader | `login.go:95-97`, `:323-336`; `internal/cli/acp.go:99-114` |
| `gobble auth login`'s paste prompter builds its own reader, with no raw mode | `internal/cli/auth.go:301` |
| The trust question reads `e.stdin` directly, in raw mode | `internal/cli/trust.go:180-184` |
| The permission prompt reads the session's reader; `askOn` drops what it read past the answer | `internal/cli/permission.go:185-230`, `:248-274` |
| The line editor takes the concrete `*inline.Reader` and keeps decoded input in `held` | `internal/cli/editor/editor.go:105-146` |
| `gobble acp --login` for a provider with no sign-in reads the key with `x/term`'s `ReadPassword`, and Claude's workspace with a `bufio` line read of `e.stdin`, with no menu before them | `internal/cli/credentials.go:204-232`; `acp.go:117-125`, `:138-151` |
| `x/term`'s own line reader drops a `\n` already read right after a `\r` | `golang.org/x/term` `v0.47.0` `terminal.go:849-851` |
| The wizard asks the key through `Secret` | go-llmprovider-sdk `v1.31.0` `wizard/configure.go:794`; a pasted credential `wizard/auth.go:528` |
| The workspace question, a line question after a secret | `wizard/configure.go:463` |
| One `inline.Reader` reads only when asked, a read starting in the asker's terminal mode; on Windows a console read keeps the mode it started in | go-tui-lib `v2.12.1` `inline/reader.go:9-21`, `:49-78`; [0027-MADR-native-line-editor-word-wrap.md](0027-MADR-native-line-editor-word-wrap.md) D5, as amended |
| A Windows console read returns what `ReadConsoleW` gives, converted to UTF-8, `\r\n` not changed | Go 1.27.4 `src/internal/poll/fd_windows.go:653-720` (`:672`) |
| A decoded key keeps its bytes; a paste keeps only its text | go-tui-lib `termkey/input.go:35-41` |
| Introduced in `978d38d` (2026-10-20); first released in `v0.2.0`; `prompt.go` unchanged since `v0.6.0` | `git tag --contains 978d38d`; `git diff v0.6.0 HEAD -- internal/cli/prompt.go` empty |

*(2026-11-20, re-grounded at `b1994a0` in the records pass before P1. Every citation above holds, with one exception:
`internal/cli/modelflags.go:82-92` is now `:87-97` (`noCredentialMessage`, `set one of …` at `:96`); `4b10e0b` inserted
five lines above it, before the PLAN's re-grounding at `09a6adf`, whose note missed it. `prompt.go`, `login.go`,
`auth.go`, `trust.go`, `permission.go`, `acp.go` and `internal/cli/editor/` have no change since `21be074`;
`credentials.go:240-243` and `line.go:137-141` hold as the PLAN's note says. go-llmprovider-sdk `v1.31.0`, go-tui-lib
`v2.12.1`, `golang.org/x/term` `v0.47.0`, `golang.org/x/sys` `v0.49.0` and Go 1.27.4 are still the pins, and their
cited lines hold. The readers of the terminal listed here and in D1 are not all of them: see the state table, rows R52
and R53.)*

The diagnosis's experiments, in its scratch directory, not committed:

* `zz_crlf_scratch_test.go`: `TestScratchSecretAfterCRLFChoice` (a choice answered `1\r\n`, then a secret) and
  `TestScratchFirstRunCRLFWaitsForKey` (the line session, the real `provider.SignIn`, `y\r\n` then `1\r\n`).
  `red.out`: both fail at head, the second with the screen ending `Enter your Gemini (Google) API key:` then
  `Sign-in cancelled; nothing was saved.`; `green.out`: both pass with option A's six lines (`fix.diff`), and the
  existing question and sign-in tests with them. `sdkmain-red.out`: the same failures with the SDK's own head.
* `q4.out`: `/login` with no argument and a menu pick cancels before a key; `/login gemini`, with no menu, waits.
* `pty_drive.py` and `pty-*.out`: a scratch-built binary driven on a macOS pseudo-terminal, with `ICRNL` off so a line
  read ends `\r\n` as on a Windows console: the key prompt skipped at head and at `v0.6.0`, and waiting with option A;
  with normal conversion, weakness (c).

**Not verified:**

* Nothing was run on a Windows console. That a console's line read gives `1\r\n` is read from Go's source and
  `ReadConsoleW`'s line mode, not measured; the owner's report places the failure there.
* The browser, device and plan sign-ins, which read through `Input` and `Confirm` and skip the `\n`; `gobble setup`;
  and `gobble acp --login` under `\r\n`: by reading only. `acp --login` reaches the method menu only since `cecc892`,
  after `v0.6.0`.
* Weakness (d): by reading only.

## Decision Drivers

* The owner's report: the key prompt must wait for a key on a Windows console, in both terminals named.
* The owner's choice of fix shape: one read-ahead buffer and one CR+LF rule for every prompt in a session.
* One reader owns the terminal's input for the process and reads only when asked, so every read starts in its asker's
  mode ([0027-MADR-native-line-editor-word-wrap.md](0027-MADR-native-line-editor-word-wrap.md) D5).
* A key is never echoed in clear: secrets stay in raw mode with `*` echo, the other questions out of it
  ([0025-PLAN-provider-choice-and-auth.md](0025-PLAN-provider-choice-and-auth.md) deviation 7).
* No change to go-llmprovider-sdk or go-tui-lib, and no new module.
* The line editor, which every prompt of every session goes through, keeps its behaviour.
* Testable without a console, by byte-level tests for every line end, plus a check on a real Windows console.

## Considered Options

* **A. The secret skips one `\n` after a `\r`-ended answer** (the diagnosis's `fix.diff`, about six lines in
  `readSecret`).
* **B. One shared terminal input per process,** holding the read-ahead and the line-end state for every prompt.
* **C. Every question read in raw mode,** so Enter is `\r` everywhere and no line read is ever cooked.
* **D. Let an empty secret through to the SDK** instead of cancelling.

## Decision Outcome

Chosen option: "B. One shared terminal input per process", because the owner chose it (the choice quoted in the
Context): it fixes the report and weakness (a) in one place, and gives every reader of the terminal
one line-end rule rather than one more special case in `readSecret`. The decision is the owner's; the design below is
this record's. Its open questions Q1 to Q8 were answered by the owner on 2026-11-19 ("Owner questions" below): Q1 to Q4,
Q6 and Q7 as recommended, Q5 and Q8 not. Q5's answer adds the first-run loop (D8); the one point of it that answer
did not settle, Q9, the owner answered the same day as recommended. The owner approved
[0063-PLAN-one-shared-terminal-input-so-the-key-prompt-waits.md](0063-PLAN-one-shared-terminal-input-so-the-key-prompt-waits.md)
on 2026-11-19; nothing is built yet.

*(2026-11-20: the owner released 0063 for building with the ruling "Add the table before P1, when 0063 is released
(Recommended)". The records pass that ruling calls for added the state table below ("State table", after
Confirmation). It has six `open` rows, put to the owner as Q10 to Q14 under "Owner questions". The decision above and
every owner answer are unchanged; the front matter is left as it stands until the owner rules on it, since the
records skill holds a record with an open row to be `proposed`.)*

*(2026-11-20, second records pass: the owner answered Q10 to Q14 and ruled on the review's cases 1 to 3 (D12 to D16
below). The table now has six rows that are not decided or out of scope: R40, which awaits live evidence, and five
put to the owner as Q16 to Q20. The front matter is unchanged.)*

- **D1. One terminal input per process** (the owner's decision). A new type in a new file,
  `internal/cli/terminput.go`:

  ```go
  // terminalInput is the one reader of the terminal's input in a process: every
  // prompt reads through it, so what one answer leaves is the next one's, and a
  // line end is read one way everywhere (0063-MADR).
  type terminalInput struct {
      in      keysIn     // the inline.Reader over stdin, or a test's
      mu      sync.Mutex // guards pending and lfOwed; never held across a read of in
      pending []byte     // read from in and not yet given to any prompt
      lfOwed  bool       // a line read out of raw mode ended at \r: a \n next is that end's
      closed  bool       // in gave an error that is not a context's: the input is closed for good
  }
  ```

  Its methods: `Read(ctx, p)`, `line(ctx)`, `key(ctx)`, `rawEnd(c)`, `discard()` (D2 to D4), and `ended()` (D8).

  - `runEnv` holds it (`termIn *terminalInput`), made on first use over `inline.NewReader(e.stdin)` by
    `e.terminalIn()`. The line session, `gobble setup`, `gobble acp --login`, `gobble auth login`'s paste prompter, and
    the trust question (Q2, answered "Every reader in the process, each keeping its own decoding (Recommended)") take it
    from there; nothing else in the process calls `inline.NewReader` on stdin. Tests set
    `termIn` to one over their own input, as the line harness gives its pipe today.
  - It never reads ahead. It reads `in` only for a prompt that needs a byte and finds `pending` empty, with that
    prompt's buffer, so 0027-MADR D5's rule holds: each read starts in the mode of whoever asked.
- **D2. How each prompt gets its bytes.**
  - `questions` keeps `ctx`, `out` and `raw` and loses `in keysIn`, `pending`, `afterCR` and `fill`; it holds the
    `*terminalInput`. `ls.ask` still makes a `questions` per ask (it carries that ask's context), but the read-ahead and
    the line-end state live in the `terminalInput`, so they survive from one ask to the next.
  - `choose`, `confirm` and `input` read through `lineContext`, now `q.in.line(ctx)`: the text up to the line end,
    then `plainLine` as today. They read out of raw mode, a line at a time (0025-PLAN deviation 7 stands).
  - `readSecret` takes bytes one at a time from `q.in.key(ctx)`, in raw mode (deviation 7 stands), and on Enter calls
    `q.in.rawEnd(c)` (D3). `secret` with no raw mode (`raw == nil`) reads a line, as today.
  - The line editor, the permission prompt and the trust question read through `(*terminalInput).Read(ctx, p)`, which
    gives `pending` first, the D3 rule applied, then one read of `in`. `editor.New` takes an interface,
    `editor.Input` (`Read(context.Context, []byte) (int, error)`), in place of `*inline.Reader`; both types satisfy it,
    so the editor's own tests are unchanged. The editor's and `askOn`'s decoding, and what each holds decoded past an
    answer, stay theirs (Q2).
  - `gobble acp --login` for a provider with no sign-in keeps `x/term`'s `ReadPassword` and the `bufio` workspace read
    (`credentials.go:204-232`, `acp.go:138-151`): no menu comes before them in that process, nothing has read ahead
    of them, and neither leaves a stray line end (`ReadPassword` reads through the `\n`).
- **D3. One rule for line ends** (Q1, answered "CR, LF and CR+LF each end one line; LF then CR counts as two
  (Recommended)").
  - A line ends at `\r`, at `\n`, or at `\r\n`, which is one end.
  - After a `\r` that ended a line read out of raw mode, the next `\n` any prompt is given is that end's and is dropped,
    whether it is already in `pending` or arrives in a later read (`lfOwed`). On a Windows console, Enter in a line read
    is expected to give `\r\n` (D6, to be measured); a line longer than the read's buffer can split the two across
    reads.
  - After a `\r` that ended a raw read (a secret's Enter), a `\n` already in `pending` directly after it is dropped (a
    pasted key ending `\r\n`, as `x/term`'s line reader does), and nothing is owed: in raw mode Enter is `\r` alone
    (0027-PLAN P3 found every key, Enter among them, arriving as `x/term` reads it, on the three systems), so a later
    `\n` is the user's next Enter in cooked mode. This removes weakness (d).
  - `\n` then `\r` is two ends: no terminal measured sends it for one key, and on Unix a cooked answer's `\n` followed by
    Enter in a raw secret is exactly that pair.
  - Any byte given to any prompt clears `lfOwed`.
- **D4. Read-ahead, cancel and the end** (Q3, answered "Dropped silently, as today (Recommended)"; between the
  first run's rounds, D8).
  - From question to question, across asks, nothing is dropped: the first run's offer, `/login`'s menu, a method menu
    and the key share one `pending`.
  - When the line session's prompt reads again after a sign-in, `pending` is dropped, `lfOwed` kept: before each
    `ReadPrompt` the line session calls `termIn.discard()`. Nothing typed or pasted during a sign-in becomes a turn sent
    to a model.
  - A cancelled question (Ctrl+C, the end of input, an ended context) leaves `pending` as it is; it is dropped at the
    session prompt as above, or when the process ends (`gobble setup`, `gobble acp --login`).
  - Ctrl+C keeps its meaning in each mode: a signal out of raw mode (`ls.ask`'s `intr.during`), byte `0x03` in a secret.
    A read the context gave up on stays `inline.Reader`'s, taken by the next read, as today.
- **D5. Paste.** Bracketed paste stays off while a question reads (`login.go:205-225`), so a paste arrives as typed
  bytes. A key pasted with a line end ends the secret, and D3 drops the `\n` of a `\r\n`. A paste of several lines into
  a question answers it with the first line and leaves the rest in `pending` for the questions after it in the same
  sign-in, as one `questions` does today; what is left is dropped at the session prompt (D4).
- **D6. Raw and cooked hand over without loss or duplication.** Each byte read is in exactly one place: `inline.Reader`'s
  kept bytes, then `pending`, then the prompt that takes it. A mode change does not touch bytes already read: on
  Windows, a cooked line handed to a raw secret is read as raw bytes (`\r` an Enter, the owed `\n` dropped). The design
  assumes three things about Windows:
  - a console read keeps the mode it started in (0027-MADR D5; measured there);
  - in raw mode with virtual-terminal input, Enter is `\r` alone (0027-PLAN P3, which measured Enter arriving as
    `x/term` reads it);
  - a line-mode read returns one line ending `\r\n`, and keys typed past it stay in the console's buffer. **To be
    measured** on a console (Q7, and the owner's check).
- **D7. An empty key** (Q4, answered "Still cancel, with today's wording (Recommended)"): Enter on an empty key prompt
  still cancels that sign-in, with the same words. Once D3 removes the stray line end, an empty Enter there is the
  user's own. In the first run the offer then comes back (D8).
- **D8. The first run offers the sign-in again until it succeeds or is declined** (Q5, answered "Offer the sign-in
  again until it succeeds or is declined"; as it was put to the owner: the first-run offer repeats after a failed or
  cancelled sign-in, more guidance for a new user, and a loop to get right).
  - **Where.** In `lineSession.firstRun` alone (`login.go:300-316`). `ls.login`'s body is split into one sign-in round
    that returns its outcome (signed in, cancelled, failed) after printing its line as today; `/login` calls it once
    and prints what it prints now. `/login`, `gobble setup` and `gobble acp --login` do not loop: setup and acp call
    `e.signIn` directly, and `/login` is a command the user typed again at will.
  - **A round** is the offer (`confirm`, `firstRunQuestion`), then, on yes, `/login`'s sign-in: the provider menu, a
    method menu when the provider has one, the key.
  - **The loop ends** on:
    - a successful sign-in, which goes on as today (credentials read again, the provider listed, its models shown);
    - `n` or `no` at the offer, or Enter alone at a repeated offer, whose default is no (Q9);
    - Ctrl+C at the offer, which is the user declining the offer itself, as today's code takes it (`login.go:307-310`);
    - the end of input, at the offer or inside a round (below);
    - the process's own context ending, whose error is returned as today.
  - **The offer comes back** after a sign-in that was cancelled or failed: Ctrl+C at the provider menu (the signal
    `ls.ask` takes) or at a method menu; Ctrl+C or an empty Enter (D7) at the key prompt; three refused keys; an
    offline listing, a provider's outage, or any other error the sign-in returns. Retrying may not help with an outage
    or while offline; the owner's answer repeats the offer all the same, and the user leaves with `n`.
  - **Between rounds** gobble prints the round's own line, as `/login` prints it: `Sign-in cancelled; nothing was
    saved.`, or `Error: ` and the sign-in's error in gobble's words (for example `claude refused 3 keys; none was
    saved`, or the listing's failure kind). Then the offer again, with the same question, as `[y/N]` (Q9). The
    no-credential warning is not printed between rounds: the question itself says no provider has a credential.
  - **Read-ahead between rounds is dropped:** `termIn.discard()` runs before each repeated offer (not before the
    first). What was typed during a round was meant for that round's questions, and the user has not seen the new
    offer; nothing typed before it may answer it, least of all as a yes. `lfOwed` is kept, so the `\n` of a `\r\n`
    that ended the round is still dropped. Within the first round, read-ahead is kept from the offer to the key (D4).
  - **Not talking to a person.** The first run is offered only on a terminal (`caps.In`, `line.go:137`), as today.
    Inside the loop, `terminalInput` records `closed` once its input gives an error that is not a context's: the end of
    input, or a read failure; `ended()` reports it. `inline.Reader` gives that error to every later read (`inline/reader.go:72-73`), so
    every question after it would fail at once. The loop checks `ended()` after the offer and after each round, and
    ends as a decline. Because read-ahead is dropped before each repeated offer, no round starts without a fresh read
    from the terminal, so a sign-in that fails at once cannot repeat without a person answering; no round counter is
    kept. Tests pin both: input closed after a cancelled round, and a sign-in that fails at once with input closed
    after the first yes, each under the harness's bounded wait, so a loop that spins fails rather than hangs.
  - **When it ends without a credential** (declined, Ctrl+C at the offer, the end of input): the lead and the warning a
    declined offer prints today (`personality.LeadNoCredential`, `noCredentialMessage`), once.
- **D9. Weakness (c)** (Q6, answered "Leave it and record it (Recommended)"): unchanged. A terminal that turns one Enter into two line ends before gobble
  reads them cannot be told from two Enters; a line read in cooked mode on Unix returns one line per read. Recorded,
  not fixed.
- **D10. What each path does after the change.**

  | Path | Before | After |
  | --- | --- | --- |
  | First run (offer, menu, method menu, key) | two asks, each with its own `pending`; on a Windows console the key prompt cancels at once | one `pending` from the offer to the key; the key prompt waits; a cancelled or failed sign-in brings the offer back, read-ahead dropped first; `n`, Ctrl+C at the offer or the end of input ends it with the warning (D8) |
  | `/login` with no argument | the key prompt cancels at once after a `\r\n` menu answer | waits; no loop |
  | `/login <provider>` | waits (no provider menu); a method menu (OpenAI, Grok, Kilo) skips the key | waits after every menu |
  | `gobble setup` | its own reader; skips the key after the menu (by reading) | `e.terminalIn()`; waits; no loop; "setup cancelled; nothing was saved" unchanged |
  | `gobble acp --login openai\|grok\|kilo` | its own reader; skips the key after the method menu (by reading; since `cecc892`) | `e.terminalIn()`; waits |
  | `gobble acp --login <other>` | `x/term` `ReadPassword`, no menu | unchanged |
  | `gobble auth login` (paste prompter) | its own reader, no raw mode | `e.terminalIn()`; otherwise unchanged |
  | Permission prompt, trust question | the session's reader; `e.stdin` directly | `terminalInput.Read` (Q2) |
  | Line editor | the session's reader | `terminalInput.Read`; `pending` dropped before each prompt |

- **D11. Records and releases.** It completes [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md)
  D6's deviation-7 note and D7, extends [0027-MADR-native-line-editor-word-wrap.md](0027-MADR-native-line-editor-word-wrap.md)
  D5's one reader to every prompt, and changes the reading under
  [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md) D7's prompter, each
  with a dated pointer note. No go-llmprovider-sdk change or release is needed; gobble's pin stays `v1.31.0`, and
  go-tui-lib's `v2.12.1`. The release is decided after the owner's Windows check has passed (Q8, answered "Decide after
  the Windows check has passed"): the PLAN builds and verifies on today's code, its last phase is that check, and it
  stops there; no `v0.6.1` work is planned now.

*Amendment, 2026-11-20 (second records pass): D12 to D16 below record the owner's rulings of 2026-11-20, relayed by
the coordinator, on a record review's findings and on Q10 to Q14. D1 to D11 are unchanged except where a dated note
says so. What any of them says about a Windows console is inferred from Go's and the libraries' source, not
measured; the measurements are named where they are made.*

- **D12. The terminal's own input is flushed at two points** (review case 1; the owner: "Flush the terminal's input at
  those points (Recommended)"). D4, D5 and D8 hold only for bytes gobble has read: a line read out of raw mode returns
  at most one line on a real terminal, so whatever else was typed or pasted stays in the terminal's own queue, where
  `discard()` cannot reach it. So, at the session prompt and before a repeated offer, gobble also empties that queue,
  with one call per platform: on POSIX the flush of pending input, `tcflush(fd, TCIFLUSH)` (`golang.org/x/sys/unix`
  `v0.49.0`, already required, has no `Tcflush` function: on Linux it is the `TCFLSH` ioctl with `TCIFLUSH`, on macOS
  the `TIOCFLUSH` ioctl with the read queue), on Windows the console's input-buffer flush (`FlushConsoleInputBuffer`,
  through `golang.org/x/sys/windows`). With it, D4, D5 and D8 hold as written.
  - The call goes through a seam (`termFlush`, as `termRaw` is), which the line harness stubs and counts; it is made
    only on a terminal, never on a pipe or a file.
  - At a repeated offer, the flush runs after the offer is written and before its read, so nothing typed before the
    offer was shown answers it (D8). At the session prompt it runs with `discard()`, before `ReadPrompt`.
  - No other prompt flushes: the state table, rows R74 to R83, says which and why, and what is still open.
  - *(2026-11-20, fourth pass: narrowed by D17. "At the session prompt" is only the first `ReadPrompt` after a
    sign-in or after the first run, by the mark of rows R103 to R108; `discard()` alone runs before the others. Where
    the flush call lives and how it reaches standard input and stderr: D18.)*
- **D13. The line of a prompt that gave up is dropped** (review case 2; the owner: "Drop the abandoned prompt's line
  when it arrives (Recommended)"). A browser sign-in whose loopback arrives first ends the paste prompt's read
  (go-llmprovider-sdk `v1.31.0` `wizard/auth.go:916-962`; `llm/provider/signin.go:441-446`); its read of the terminal
  stays waiting in `inline.Reader` (`inline/reader.go:59-70`), and `discard()` cannot drop it. A line that completes a
  read belonging to an abandoned prompt is discarded: never a turn, never an answer; the prompt that is waiting then
  reads again.
  - A prompt that returns while its read may still be waiting calls `terminalInput.abandon()`: `questions` when its
    read ends on its context, `askOn`'s callers when it returns cancelled, the line editor when `ReadPrompt` ends on its
    context. A timed wait inside a prompt that reads again (the editor's Esc timer, `askOn`'s timers) is not giving
    up. What a completion that is not a line becomes, and whether a read the user gave up with Ctrl+C is dropped, the
    ruling's words do not settle: rows R86 and Q16.
  - *(2026-11-20, fourth pass: narrowed by D17. Only a read the program gave up is marked: `questions` when the
    context that ended is not `ls.ask`'s interrupt cancel, and `askPermission` when the request was withdrawn without
    the person's Ctrl+C. `askOn`'s callers do not call `abandon()` for a cancel the person made, and the line editor
    never calls it: `ReadPrompt` ends on its context only when the process is ending. The drop takes the whole
    completion, D18.)*
- **D14. Ctrl+Z and Ctrl+C on a Windows console** (review case 3 and Q11; the owner: "Measure both in your Windows
  check; stop if Ctrl+C ends input (Recommended)").
  - Ctrl+Z (byte `0x1A`) on a Windows console is the end of input, decided from D8 ("the end of input"): by reading,
    Go's console read stops at `0x1A` and gives nothing (`internal/poll/fd_windows.go:710-715`), which is the end of
    input (`os/file_windows.go:110`, `ZeroReadIsEOF`), and `inline.Reader` keeps that for good
    (`inline/reader.go:72-73`, `:90-91`). So Ctrl+Z at the key prompt, or Ctrl+Z then Enter at a menu, ends the first
    run as a decline, and the session ends at the editor's next read. On POSIX the same key is a byte: the secret
    ignores it (`prompt.go:287`). Measured by the pseudo-console probe and in the real-console confirmation.
  - Ctrl+C during a read out of raw mode on a Windows console awaits live evidence (row R40). If the measurement
    shows the read still waiting, the build goes on. If it shows the end of input, or an error, the build stops and the
    owner chooses among the options recorded on row R40.
  - *(2026-11-20, after P1 step 0b: measured. On the Windows test laptop's pseudo console, `probe-ctrlc` observed
    `eof` twice: the waiting line-mode read returned nothing, the end of input, after the probe child had taken the
    interrupt. The build stopped as designed; the owner chose row R40's option (a), recorded as D19.)*
- **D15. Q12 to Q14 answered** (the owner, 2026-11-20): the trust question reads the shared input with the run's
  context and no deadline, as it reads stdin today ("No deadline, as today (Recommended)"); an answer's length has no
  bound ("No bound; record it (Recommended)"); `secret` restores the terminal in a deferred call, so a panic restores
  it before it goes on ("Defer the restore at the secret prompt (Recommended)").
- **D16. Who runs the live checks** (the owner, 2026-11-20: "script as many live runs as possible and run them, leave
  for me the minimum possible"; for credentials, "API keys from the environment only (Recommended)"). Every check a
  pseudo console or a pseudo-terminal can drive is a committed test that an agent runs on the Windows test laptop or
  the Linux machine through the fleet's host tooling (`run-on-host`), not the owner. The owner keeps only what needs a
  real console window (Confirmation item 5, as amended). No 0063 check needs a provider key: every key typed is made
  up, no key is ever sent, and the proxy variables point at a closed local port so a request, if one were made, fails
  on the host. No agent runs a sign-in, uses a saved session, or opens a browser.
- **D17. Q16 to Q20 answered** (amendment, 2026-11-20, third records pass; the owner, relayed by the coordinator).
  - **Q16, "Only reads the program gave up (Recommended)":** D13's drop applies to a read the program gave up (the
    read left behind when a browser sign-in's loopback arrives first; any read whose context the program ended),
    whatever completes it, a line or keys. A read the person interrupted with Ctrl+C is not dropped and goes to the next
    prompt, as D4 says. So `abandon()` is called only for a program's give-up: `questions` when the context that
    ended is not `ls.ask`'s interrupt cancel, and `askOn`'s callers when the request was withdrawn without the
    person's Ctrl+C. The consequence, row R102: the first ~~key (POSIX) or~~ line (Windows) typed after such a browser
    sign-in is lost. *(2026-11-21, D21: on POSIX, what was typed into the abandoned prompt is lost, measured on
    Linux; the first key only when nothing was typed there, inferred.)*
  - **Q17, "Go on, with one warning on stderr (Recommended)":** a flush that fails is skipped, and one warning is
    written on stderr, once per process (row R78). Its words are proposed in row R101 and await the owner's approval.
  - **Q18, "Keep that, recorded and tested (Recommended)":** when raw mode cannot be switched on for the key prompt,
    the error is returned, no prompt is written and nothing is read (row R28).
  - **Q19, "Dropped, as today (Recommended)":** what `askOn` decoded past an answer is dropped when it returns (row
    R57); Q2's "keeping" is narrowed by a dated note.
  - **Q20, "Only the first prompt after a sign-in or the first run (Recommended)":** the session-prompt flush of D12
    runs only at the first `ReadPrompt` after a sign-in or after the first run. `terminalInput` keeps a mark: a sign-in
    round inside the session sets it, whatever its outcome, and so does the first run, however it ends; the flush at
    the next `ReadPrompt` clears it; nothing else sets it (`/logout`, the trust question, a permission prompt, a
    resumed session with no first run). `discard()` still runs before every `ReadPrompt` (D4). Rows R103 to R108.
  - *(2026-11-20, fourth pass.)* R101's words, proposed in the third pass, are approved by the owner, relayed by the
    coordinator: "Plain words, with the consequence (Recommended)":
    `Warning: could not clear what was typed ahead on this terminal (<error>); it may answer the next prompt.`
- **D18. Where the flush and the drop live** (amendment, 2026-11-20, fourth records pass; no new decision, the
  plumbing D12, D13 and D17 need).
  - **The flush call** is `term.FlushInput(f *os.File) error` in `internal/cli/term`, which already holds the per-OS
    terminal calls and imports `golang.org/x/sys` (`console_windows.go:11`): `flush_linux.go` (the `TCFLSH` ioctl
    with `TCIFLUSH`), `flush_darwin.go` (the `TIOCFLUSH` ioctl with the read queue), `flush_windows.go`
    (`windows.FlushConsoleInputBuffer`), and `flush_other.go` for any other system, which reports that it cannot
    flush, R78's case. `golang.org/x/sys` `v0.49.0` is already required (`go.mod:24`); `internal/archtest` has no rule
    on `golang.org/x/sys` and none that keeps `internal/cli/term` from it, so no record changes.
  - **The binding.** D1's struct carries no file and no writer, so `terminalInput` gains one field, `flush func()
    error`, nil for a pipe, a file or a test's input. `e.terminalIn()` sets it when standard input is a terminal
    (`caps.In`) to a function that calls `termFlush(e.stdin)` (the seam, which is `term.FlushInput` in production) and,
    on the first failure in the process, writes R101's warning through `e.out` (stderr), once.
  - **The drop takes the whole completion.** `terminalInput` records the buffer size of the read a program gave up
    (512 bytes for a question, `prompt.go:201`); when that read completes, it takes the completion from `inline.Reader`
    with a buffer of that size, whichever prompt is waiting, and drops it. `inline.Reader` gives at most the asker's
    buffer (`inline/reader.go:75-77`), so a waiting `askOn` (256 bytes, `permission.go:206`) asking for it directly
    would get its tail.
- **D19. After an interrupt gobble took, a Windows console read that gives nothing is read again** (amendment,
  2026-11-20, a deviation found at P1 step 0b; the owner, relayed by the coordinator: "(a) After an interrupt gobble
  took, read again (Recommended)", option (a) of row R40).
  - **What was found.** On a pseudo console, Ctrl+C during a read out of raw mode ends that read with nothing, which Go
    reports as the end of input (`probe-ctrlc` observed `eof`, twice; the PLAN's deviation entry quotes the lines).
    `inline.Reader` keeps the end of input for good (`inline/reader.go:72-73`, `:90-91`), so with no change Ctrl+C at a
    menu would close the input for the process, and D8's "the offer comes back after Ctrl+C at the provider menu"
    would not hold on Windows.
  - **The decision.** On Windows only, a console read that gives nothing right after gobble received Ctrl+C is not
    the end of input: the shared input reads again. The behaviour then matches POSIX: Ctrl+C at the provider menu
    cancels that round and the offer comes again (R37); Ctrl+C at an offer declines it (D8) and the session goes on.
  - **Where it lives** (this record's design; no further ruling): below `inline.Reader`, so go-tui-lib does not change
    (Decision drivers). `e.terminalIn()` gives `inline.NewReader` not `e.stdin` itself but a reader of it,
    `term.ConsoleInput`, which on Windows, when standard input is a console, applies the rule, and elsewhere is
    standard input unchanged. The mark it reads is set where gobble receives the interrupt: `watchSignals`
    (`signal.go:60-89`), when it receives an `os.Interrupt`, before it hands the interrupt on (when the mark is kept, Q21). The rows R109 to R121 decide what
    the ruling and the existing decisions settle; when the mark is set and cleared (Q21) and how long a read waits
    for a mark that has not yet arrived (Q22) are open.
  - **What it does not change.** Ctrl+Z is still the end of input (D14, R90); a second Ctrl+C still follows 0008-MADR
    D5; a read in raw mode never sees Ctrl+C as an interrupt (R115); pipes, files and POSIX are as before.
- **D20. Q21 and Q22 answered, and R120 widened** (amendment, 2026-11-20; the owner, relayed by the coordinator).
  - **Q21, "Set only if a read is waiting; excuses one empty read (Recommended)":** `term.ConsoleInput` notes when
    its read of the console is waiting; `watchSignals`' mark is set only then. The mark excuses only the next read
    that gives nothing, and is cleared by that read or by any read that returns bytes. So Ctrl+C then Ctrl+Z still ends
    the input (R90, R112), and a Ctrl+C with no read waiting excuses nothing (R114). Rows R111, R114.
  - **Q22, "Wait up to 100 ms for the mark; the probe reports the order (Recommended)":** an empty console read with
    no mark waits up to 100 ms for one before it counts as the end of input (R116); a real Ctrl+Z on a console ends
    the input up to 100 ms late (R122). The probe `probe-ctrlc` adds one word to its line, `first=signal` (the
    interrupt reached the child before the read returned) or `first=read` (the read returned first, so the wait is
    what makes D19 work). Neither value stops the build; each run's value is recorded.
  - **R120,** "Yes, widen the row to include it (Recommended)": Ctrl+C at `gobble auth login`'s paste prompt, where
    no prompt takes the interrupt, ends gobble too.
  - With these, no row is `open`.
- **D21. F1 and F2 of P2's build pass 1 answered** (amendment, 2026-11-21; the owner, relayed by the coordinator).
  - **F1, "Amend R86/R102 to what Linux does (Recommended)":** measured at the PLAN's P2, build pass 1, on a Linux
    pseudo-terminal (the case `pty-abandoned`, and a diagnostic of it that named the turns: `hello` and `again`, no
    `partial`): on POSIX, a read started out of raw mode by a prompt the program gave up is completed, when the session
    prompt puts the terminal in raw mode, by what was typed into that prompt and not ended; D13 drops it with the
    read, and what is then typed at the session prompt is read by the line editor whole. macOS is inferred from the
    BSD terminal driver, not measured. The earlier statement (Q16, D17's Q16 bullet, rows R86 and R102), that such a
    read completes with the first key typed after raw mode is on, holds only when nothing was typed into the
    abandoned prompt (inferred, not measured). Windows is unchanged: the console keeps the read in line mode, and the
    first line typed is dropped (`abandoned` read `dropped`). The build stands; rows R86 and R102 are amended, and
    `pty-abandoned` passes when no turn holds `partial` and `again` is one.
  - **F2, "Flush at once at that first prompt (Recommended)":** when D17's mark is set, the terminal's queue is flushed
    at the session prompt itself, before `ReadPrompt`, whether or not the line editor then reads the terminal, and the
    mark is cleared there; not at the editor's first read of the terminal, which keys the editor holds across a
    sign-in (row R56) put off to a later prompt. With it R103 and R105 hold in that case, which row R123 decides. R108
    is unchanged, nothing is flushed on the way out; its test is re-worded for the flush made at the prompt.
- **D22. On Linux and macOS a read the prompt gave up is cancelled, not left waiting** (amendment, 2026-11-21;
  ~~**proposed**: the owner's direction is decided, this design is a draft for review;~~ **accepted** (2026-11-21, D22 accepted,
  the owner's answers to Q23 to Q26, below); the front matter is unchanged).
  - **The owner's answer** to F3 and F4 of the PLAN's P2, build pass 2, relayed by the coordinator: "Make reads cancellable on POSIX".
    The options shown were "Flush before raw mode; third pass (Recommended)", "Make reads cancellable on POSIX" and "Keep this tree; record the race"; the owner chose the second, which was not the recommendation.
    Its cost, as stated with it: it replaces D13's mechanism for Linux and macOS, so it needs a new decision, a
    records pass, new tests and plants; Windows is unchanged; 0063 is delayed by at least one records pass and a build.
  - **What it decides.** On a POSIX terminal no read of standard input is outstanding after a prompt returns: a read
    is started only when a prompt waits, and only once the terminal has bytes ready, ~~so it never waits in the kernel~~
    *(2026-11-21, D22 second draft: not so on standard input itself: a line reported ready and then emptied by Ctrl+C leaves the read
    waiting, M11; how it cannot wait is Q25)*; when the prompt stops waiting, nothing is read. Nothing typed is taken by a read nobody waits for, so a later mode
    change cannot hand it to one: F4's race is impossible by construction, not by timing.
  - **Where.** A layer under `inline.NewReader`, beside `term.ConsoleInput`: on POSIX (`unix` build tag), when
    standard input is a terminal, `terminalIn()` reads it through the layer; on Windows through `term.ConsoleInput`,
    as built; a pipe or a file through neither (rows R124, R137, R143).
  - **The wait** (measured on this Mac, PLAN execution record "Records draft for D22"): select(2) on standard input
    and a close-on-exec wake pipe, with no timeout, retried on `EINTR`. Not poll(2): on macOS it refused the terminal
    opened as `/dev/tty` (`revents=0x20`, `POLLNVAL`) where select(2) waited (M8), though it served the pseudo
    terminal's slave (M1). Not the runtime poller through a deadline: it worked (M4), but needs `O_NONBLOCK` on the
    terminal's open file description, which the shell and every process on that terminal share and which a crash
    would leave set (row R125). ~~Linux is not measured yet: step 0 of the PLAN's amendment.~~ *(2026-11-21, D22 second draft:
    measured on a WSL distro and a Linux machine that is not WSL, M0 to M14; on Linux poll(2) accepts `/dev/tty` too
    (M8), and no `EINTR` was seen on any host (M6, M9).)*
  - **The gate.** `terminalInput` opens the layer's gate when a prompt's read of the input waits and closes it when
    that read returns, with bytes, the end of input or its context. A read(2) runs only with the gate open, after
    readiness, under the gate's lock; closing takes the lock, so it waits for a read(2) in progress and reports
    whether one completed since the gate opened (rows R126, R127). *(2026-11-21, D22 second draft: holding the lock across a read
    of standard input can hang a close, M11; Q25. The gate's mechanics are completed below.)*
  - **D13 on POSIX.** With nothing read for a prompt that gave up, there is nothing to drop. The drop applies only to a
    completion read before the gate closed, the race of a give-up with a byte's arrival: the program's give-up drops
    it (D13, D17), the person's Ctrl+C gives it to the next prompt (Q16) (rows R128, R129). `abandon()` marks a drop
    only when the close reported a completed read. Windows keeps D13, D18, D19 and D20 unchanged (row R143).
  - **With F2 (D21).** The first prompt after a sign-in or the first run flushes as it starts, after raw mode is on;
    no read is outstanding then, so what was typed into the sign-in's prompts and left unread is emptied, every time
    (rows R130, R131). What a program gives up outside a sign-in is Q23. A wait that cannot be used is Q24.
  - **Second draft (2026-11-21), after an independent record review and the measurements M9 to M14** on this Mac, a
    WSL distro and a Linux machine that is not WSL (the PLAN's execution record, "Records draft for D22, second
    draft"). Still proposed. What the review found missing, and its decision:
    - *The read can still wait* (M11, every host): select(2) reports a line ready, Ctrl+C empties it, and a read(2)
      of standard input then waits for the next line. A read through a private non-blocking descriptor on the
      terminal's own path gives `EAGAIN` there instead (M12, every host). Which way is Q25; until it is answered, R126
      is `open`.
    - *A closed gate* (row R145): the layer's Read does not return; it waits on the wake pipe alone, standard input
      out of the set, until the gate opens; it returns only bytes, `io.EOF` or a read error, never for a close
      (inline.Reader's goroutine ends on an error and reads again at once on `(0, nil)`).
    - *The wake pipe* (row R146): both ends non-blocking and close-on-exec; the waiter drains it to `EAGAIN` before
      each wait; `EAGAIN` on a write is not an error; it lives for the process; a test's layer is ended by `stop()`
      in `t.Cleanup`. A close returns only after the waiter has left select(2).
    - ~~*One read per opening* (row R127): the gate closes itself after a read(2) returns, so no read follows without
      a new opening; `fill`'s close is then idempotent.~~ *(2026-11-21, D22 third draft: dropped: inline.Reader makes one inner
      read per ask, so a self-close changes nothing a test can see; the gate stays open until `fill` closes it.)*
    - *The ask's size* (row R147): `fill` asks `inline.Reader` with `lineBuf` always; what the caller's buffer
      cannot take stays in `pending`, where `discard()` and the flush's rules reach it. *(2026-11-21, D22 third draft: on the
      gated path only, a POSIX terminal; with no gate, the caller's buffer, as built, so R143 holds as decided.)*
    - *No gate* (row R151): where there is no gate (a Windows console, a pipe, a test's input), `fill` sets
      `waiting` on every read that ended on its context, as built (D13).
    - *Hang-up* (row R148, M10): select(2) reports it ready; a read gives 0, the end of input, or `EIO` on Linux in a
      read already waiting, the input's error; D8 applies either way. A background start is out of scope (R149).
    - *Other readers of standard input* (row R150): with the gate closed, no read of gobble's waits when another
      reader or a child reads; R95's risk returns only with Q24's fallback.
    - *A queued Ctrl+D* (row R89's latest, M13): typed out of raw mode while nothing reads, it is not the end of
      input once raw mode is on: it is read as one byte, `0x04` on macOS and `0x00` on Linux.
    - *The flush and raw mode* (M14): a flush out of raw mode empties half a line too; R131 needs no "after raw mode".
    - *The BSDs* (row R144): `flush_other.go` cannot flush (R78), so R131 does not hold there.
    - *Sign-ins outside the session* (`gobble auth login`, `gobble acp --login`, `gobble setup`): what was typed into
      an abandoned paste prompt stays in the terminal after gobble exits; folded into Q23, as rewritten.
    - *Statuses.* Rows that rest on D22's mechanism read "decided (D22, proposed)"; the counts separate them.
  - **Third draft (2026-11-21), after the delta confirmation, on the coordinator's dispositions, and M15** (this Mac,
    a WSL distro and a Linux machine that is not WSL; the PLAN's execution record, "Records draft for D22, third
    draft"). Still proposed. Every row whose POSIX outcome needs the layer reads "decided (D22, proposed)" (R83, R84,
    R88, R89, R128, R130, R137, R148). R127's self-close is dropped: it cannot be tested. R147 holds on the gated path
    only, so R143 stays literally true. Q25 (a) gains `O_NOCTTY` and a `/dev/tty` fallback before Q24's. Through a
    private descriptor, M15 shows Ctrl+D on an empty line and a hang-up each read once as 0, the end of input, and
    `O_NOCTTY` changes nothing seen. R89 takes the review's text. R145 gains a case that shows its wait cannot spin.
  - **Accepted (2026-11-21).** On the owner's direction, the owner, relayed by the coordinator: "Make reads cancellable on POSIX", and the owner's answers to the
    four questions its drafts left open, the owner, relayed by the coordinator: each quoted as the option's label:
    - **Q23:** "Flush at gobble's give-up (Recommended)". On POSIX, the terminal's queue is flushed at the program's give-up itself:
      `askPermission`'s withdrawn branch and `questions.cancelled`'s program branch, inside the session and out of it.
      Three flush points, not two (R80, R81, R84, R132).
    - **Q24:** "Old reads plus one warning (Recommended)". When the readiness wait or its descriptor cannot be used, the process reads standard
      input as built (a blocking read, D13's drop), with one warning on stderr, the owner's approved words:
      `Warning: gobble cannot wait on this terminal without reading it (<error>); keys typed while a sign-in finishes may be lost.` (R138, R150).
    - **Q25:** "Private non-blocking descriptor (Recommended)". The layer reads a private descriptor on the terminal's own path, opened
      `O_RDONLY|O_NONBLOCK|O_CLOEXEC|O_NOCTTY`; when that open fails and standard input is the controlling terminal
      (the same foreground process group on standard input and on `/dev/tty`), `/dev/tty` with the same flags; only
      then Q24's fallback (R125, R126, R127, R134, R138, R146, R148).
    - **Q26:** "Move them to a shared file (Recommended)", the owner's word for restructuring P1b's `pty_linux_test.go` beyond D21's one line:
      the cases move word for word to `pty_unix_test.go` (`(linux || darwin) && pty`), with the openers in
      `pty_linux_test.go` and a new `pty_darwin_test.go`, and the Makefile's comment updated (R130, R131, R152).
    Every row that read "decided (D22, proposed)" reads "decided (D22)"; no row is open.
- **D23. After P2's build pass 3: the review's unpinned rows, one undecided case, and four hardenings**
  (amendment, 2026-11-21; the owner, relayed by the coordinator: each quoted as the option's label). An independent review of build pass 3
  found no blocker; five plants survived (rows R124, R126, R132 with R143, R138, R147) and one case was undecided.
  - "Fourth pass, tests only (Recommended)": a fourth P2 build pass adds the tests that pin those rows, and no production change beyond what
    the next two answers add.
  - "Always flush on a POSIX terminal (Recommended)": the flush at gobble's give-up (D22, Q23) applies on any Linux or macOS terminal, whether its reads
    go through the gate or through Q24's fallback, whichever fallback: at start-up or later. It is keyed on "a POSIX
    terminal", not on "has a gate". Windows keeps D13's drop and no give-up flush (R143). This settles the case the
    review found undecided: as built, a start-up fallback did not flush and a later one did (rows R80, R132, R138,
    R157).
  - "Retry EINTR on the read (Recommended)": `EINTR` from read(2) on the private descriptor returns to the wait, as `EAGAIN` does (row R153).
  - "Refuse a blocking descriptor (Recommended)": `NewGated` reads the descriptor's `F_GETFL` and refuses one without `O_NONBLOCK`, with an error;
    `OpenGated` takes that as a failed open: `/dev/tty`, then R138 (row R154).
  - "Check it's the same terminal": after opening the terminal's path, `OpenGated` compares `fstat`'s `Rdev` of standard input and of
    the opened descriptor; on a mismatch it closes it and goes on to the `/dev/tty` step, which keeps its
    foreground-process-group check (M15: `/dev/tty` reports its own device), then R138 (row R155).
  - "Write the warning safely": Q24's warning is never written from `inline.Reader`'s goroutine. It is queued, and written once, on
    the main flow, at the next point where gobble writes a prompt's or a question's text, before that text, with the
    terminal in its cooked state; or at exit if no such point comes. Its words are unchanged: `Warning: gobble cannot wait on this terminal without reading it (<error>); keys typed while a sign-in finishes may be lost.` (row
    R156).

### Consequences

* Good, because the key prompt waits for a key on a Windows console after the first-run offer, `/login`'s menu and a
  method menu, which is the owner's report.
* Good, because typing ahead across the first run's questions is kept (weakness (a)), and one value owns the
  terminal's input for the whole process, where four places made or bypassed a reader.
* Good, because an Enter after a secret is no longer swallowed by the next line question on macOS and Linux
  (weakness (d), by reading).
* Good, because nothing changes in the SDK, go-tui-lib or the module graph.
* Neutral, because a paste of several lines into a question still answers the questions after it in the same sign-in,
  as today; it no longer reaches the session's prompt.
* Bad, because the change crosses `prompt.go`, `login.go`, `line.go`, `auth.go`, `trust.go`, `permission.go`, `main.go`
  and `editor.New`'s signature, where A changes one function.
* Bad, because every key the line editor reads now passes through `terminalInput.Read`: a defect there reaches every
  prompt of every session. The editor's own tests run unchanged through it, the line-session tests run with each line
  end, and the PLAN's planted breaks check the rule; the cost per read is a length check on an empty slice.
* Bad, because a user typing Ctrl+J as the very first key after a line question ended by `\r` loses that one key.
* Good, because a new user whose first sign-in is cancelled or fails is offered it again, with the reason on screen,
  rather than left at a prompt whose next turn fails (weakness (b)).
* Bad, because the first run becomes a loop, which must end on every way out (a decline, Ctrl+C at the offer, the end
  of input) and must never repeat without a person's answer. D8 settles each, and the PLAN pins each with a test and a
  planted break.
* Bad, because a user who is offline, or whose provider is down, is offered the same sign-in again, which cannot help
  until that changes; they leave with `n`, or Enter, since the repeated offer defaults to no (Q9).
* Users can see: on Windows, the key prompt waits; in the first run, the offer again after a cancelled or failed
  sign-in, and the warning once when they decline; on macOS and Linux, a workspace or other line question after a key
  accepts the first Enter.

### Confirmation

1. **Tests**, each seen red at `21be074` where the defect is there and green after: the question reader for each line
   end (`\r`, `\n`, `\r\n`) before a secret and before each other kind of question; the line session's first run with
   a key-only provider and with a method-menu provider, `/login` with no argument, `gobble setup` and
   `gobble acp --login`, each with each line end; the read-ahead kept across asks and dropped at the session prompt;
   the line harness answering with each line end; the first-run loop's every way back to the offer and every way out,
   the end of input under a bounded wait, and `/login` and `gobble setup` not looping. The list, and which tests are
   red for the defect, red for the new loop, or guards, is the PLAN's P1.
2. **Planted breaks** on scratch copies, each new rule seen to fail (PLAN P3).
3. **Gates** as this repository defines them (PLAN P4), and `go test` of `internal/cli` on the Windows test laptop,
   which has no console.
4. **A pseudo-console test on Windows** (Q7, answered "A committed pseudo-console test, run on the Windows laptop by a
   make target, outside CI (Recommended)"): PLAN P1b.
5. **The owner's check on a real Windows console,** in Windows PowerShell 5.1 in a Console Host window and in
   PowerShell 7.x in Windows Terminal, with a build of the fixed tree (PLAN P6). No real key is typed: the key typed is
   made up, and every sign-in is left with Ctrl+C before anything is sent. In each terminal:

   1. **Open the terminal.** For Console Host: press Win+R, run `conhost.exe powershell.exe`. For the other: open
      Windows Terminal with a PowerShell 7 tab. Run `$PSVersionTable.PSVersion`: `5.1…` in the first, `7.…` in the
      second.
   2. **Start from nothing, in this window only.** Paste and run, one line at a time:

      ```powershell
      $env:GOBBLE_HOME = Join-Path $env:TEMP 'gobble-check-home'
      Remove-Item -Recurse -Force $env:GOBBLE_HOME -ErrorAction SilentlyContinue; New-Item -ItemType Directory $env:GOBBLE_HOME | Out-Null
      foreach ($v in 'GEMINI_API_KEY','ANTHROPIC_API_KEY','OPENAI_API_KEY','XAI_API_KEY','OPENCODE_API_KEY','HF_TOKEN','KILO_API_KEY','TOGETHER_API_KEY') { Remove-Item "Env:$v" -ErrorAction SilentlyContinue }
      $work = Join-Path $env:TEMP 'gobble-check-work'; New-Item -ItemType Directory -Force $work | Out-Null; Set-Location $work
      $g = Join-Path $env:USERPROFILE 'gobble-check\gobble.exe'
      ```

      `$g` is where the build from PLAN P6 was put. These settings end when the window closes.
   3. **Confirm there is no credential:** `& $g auth status`. Every row's `credential` column reads `none`.
   4. **The report's path.** Run `& $g`. After any opening lines, expect
      `No provider has a credential yet. Sign in to one now? [Y/n]:`. Type
      `y`, Enter. Expect `Sign in to which provider?`, nine numbered providers, `Number [1]:`. Type `1`, Enter. Expect
      `Enter your Gemini (Google) API key:` and the cursor waiting after it, with nothing else printed.
      **Fail** if `Sign-in cancelled; nothing was saved.` appears without a key typed.
   5. **The key is read masked, and the offer comes back after Ctrl+C.** Type `not-a-real-key` (14 characters):
      fourteen `*` appear, never the letters. Press Ctrl+C. Expect `Sign-in cancelled; nothing was saved.`, then the
      offer again: `No provider has a credential yet. Sign in to one now? [y/N]:` (Q9).
   6. **The offer comes back after an empty key.** Type `y`, Enter; type `1`, Enter; at the key prompt press Enter
      alone. Expect `Sign-in cancelled; nothing was saved.` (Q4), then the offer again.
   7. **Declining ends it.** Type `n`, Enter. Expect the no-credential lead and
      `Warning: no model credential: set one of GEMINI_API_KEY, …` once, then gobble's prompt, and no further offer.
   8. **`/login` with a key-only provider.** Type `/login`, Enter; at `Number [1]:` type `3` (Claude), Enter. Expect
      `Enter your Claude (Anthropic) API key:`, waiting. Press Ctrl+C. Expect the cancel line.
   9. **A method menu.** Type `/login`, Enter; type `2` (OpenAI), Enter. A second numbered menu asks how to sign in;
      type the number of the item that names an API key, Enter. Expect `Enter your OpenAI API key:`, waiting. Press
      Ctrl+C. Expect the cancel line.
   10. **`/login` does not loop.** Type `/login gemini`, Enter; at the key prompt press Enter alone. Expect
       `Sign-in cancelled; nothing was saved.` and gobble's prompt, with no offer.
   11. Type `/exit`, Enter.
   12. **`gobble setup`.** Run `& $g setup`. Expect the provider menu; type `1`, Enter. Expect
       `Enter your Gemini (Google) API key:`, waiting. Press Ctrl+C. Expect `Error: setup cancelled; nothing was saved`,
       and PowerShell's prompt: no offer.
   13. **`gobble acp --login`.** Run `& $g acp --login openai`. Pick the API key item, Enter. Expect the key prompt,
       waiting. Press Ctrl+C. Expect `Error: sign-in cancelled; nothing was saved`.
   14. **Nothing was saved:** `& $g auth status` reads `none` on every row, as in step 3.
   15. Close the window, and delete the folders `gobble-check-home` and `gobble-check-work` under `%TEMP%`.

   Before the fixed build, the same steps 1 to 4 with the installed `gobble` `0.6.0` (`gobble --version`) are expected
   to show the report's failure at step 4; that run, if the owner makes it, is the first reproduction on a console.
   What is recorded is which steps passed in which terminal and any screen text that differed, never a key.

   *(2026-11-20: item 5 above is superseded by item 5 as amended below, after the owner's ruling D16. It is kept as
   written; its steps are now split between agent-run cases and a short owner-only confirmation.)*

5. **As amended 2026-11-20: the Windows checks, agent-run first, then the owner's real-console confirmation.**

   **(A) Agent-run.** Committed tests, started by an agent through the fleet's host tooling (`run-on-host --need
   windows` for the pseudo console, `--need linux` for the pseudo-terminal), never by the owner. Each case prints one
   line and nothing else of the screen or of any key: `0063 case=<name> result=<pass|fail> observed=<word>`. The agent
   passes the run only when every case's line is present and reads `result=pass`; a missing line (for example a run
   that fell back to a host where the test does not build) is a fail. The observed word is the only judgement, and the
   two marked "stop" decide the build without any reading of the screen.

   | Case | What the test does | Pass, and its observed word | Rows |
   | --- | --- | --- | --- |
   | `probe-crlf` | one line-mode read; types `1`, Enter | the read gives `1\r\n`: `crlf` | R34 |
   | `probe-typeahead` | types `1`, Enter, `2`, Enter before two line-mode reads | one line per read: `one-line-per-read` | R35 |
   | `probe-ctrlc` | Ctrl+C during a line-mode read, then `x`, Enter | `waiting` (the read gave `x\r\n`); `eof` or `error` is the R40 stop | R40 |
   | `probe-ctrlz-line` | Ctrl+Z, Enter in a line-mode read | `eof` | R90 |
   | `probe-ctrlz-raw` | Ctrl+Z in a raw read | `eof` | R90 |
   | `report` | the first run: `y`, Enter; `1`, Enter | the key prompt still waits after 2 s, no cancel line: `waits` | R18, R34 |
   | `masked` | types `not-a-real-key` at the key prompt | fourteen `*`, none of the letters: `masked` | R27 |
   | `key-ctrlc` | Ctrl+C at the key prompt | the cancel line, then the offer ending `[y/N]: `: `offer-again` | R39, R64 |
   | `empty-key` | `y`, Enter; `1`, Enter; Enter at the key prompt | the cancel line, then the offer: `offer-again` | R13 |
   | `menu-ctrlc` | `y`, Enter; Ctrl+C at the provider menu | the offer again: `offer-again`; `ended` is the R40 stop | R37, R40 |
   | `decline` | `n`, Enter at a repeated offer | the warning once, gobble's prompt: `warned-once` | R66 |
   | `typeahead` | `y`, Enter, `1`, Enter written before the offer is shown | the key prompt with nothing more written: `waits` | R22, R35 |
   | `flush-prompt` | `n`, Enter, `hello`, Enter in one write at the first offer | no turn, no `no model credential` line: `no-turn` | R25, R51, R76 |
   | `flush-offer` | a stubbed round waits 1 s and fails; `y`, Enter written during it | the repeated offer shown and still waiting: `offer-waits` | R69, R76 |
   | `abandoned` | a stubbed browser sign-in: `partial` typed at its paste prompt, then the stub's loopback ends it; `hello`, Enter at gobble's prompt; then `again`, Enter | no turn holds `partial` or `hello`; a turn `again`: `dropped` | R83, R85 |
   | `login-no-loop` | `/login gemini`; Enter at the key prompt | the cancel line, no offer: `no-offer` | R72 |
   | `method-menu` | `/login`; `2`; the API key item | the key prompt waits: `waits` | R18, R21 |
   | `key-ctrlz` | Ctrl+Z at the key prompt | gobble ends: `ended` | R90 |
   | `setup` | `gobble setup`; `1`, Enter; Ctrl+C at the key prompt | waits, then `setup cancelled; nothing was saved`: `cancelled` | R18, R72 |
   | `acp-login` | `gobble acp --login openai`; the API key item; Ctrl+C | waits, then the cancel line: `cancelled` | R18, R72 |
   | `nothing-saved` | after every case | the temporary home's `auth.json` has no entry: `none` | R27 |
   | `pty-flush-prompt`, `pty-flush-offer`, `pty-abandoned` | the three cases above on a Linux pseudo-terminal | as above | R25, R51, R69, R74, R83, R84 |

   *(2026-11-20, fourth pass: the probe child of `probe-ctrlc` calls `signal.Notify` for `os.Interrupt` before its
   read, as gobble does, so Ctrl+C reaches it as a signal; without it, Go's console handler would let the default one
   end the child (inferred, `runtime/os_windows.go:1047-1069`), which would read as `error`. The child records that
   the signal came; a child that ends without its result reads `killed`, a fault in the probe, not R40's stop. The
   PLAN's P1 step 0b gives each word's meaning.)*

   *(2026-11-21, D21: `pty-abandoned`'s pass is not its Windows twin's. On a Linux pseudo-terminal it reads
   `dropped` when no turn holds `partial` and a turn is `again`; `hello` may be a turn, since on POSIX what was typed
   into the paste prompt completes the abandoned read and nothing typed at gobble's prompt is lost (row R102, as
   amended). `abandoned`, on the Windows test laptop, keeps the words in the table.)*

   Every case runs with `GOBBLE_HOME` a temporary directory, no credential variable, and `HTTPS_PROXY` and
   `HTTP_PROXY` pointing at a closed local port; every key typed is made up, and none is sent. The sign-in in
   `flush-offer` and `abandoned` is a stub inside the test binary, which opens no browser and reaches no network.

   **(B) Owner-only: the real-console confirmation.** A pseudo console is the console host fed text through a pipe:
   the test's text, or Windows Terminal's own encoding of key events. So what a classic console window does with a
   real keyboard's key events, how PowerShell 5.1 and 7 leave the console's mode when they start gobble, and what Ctrl+C
   and Ctrl+Z do there are inferred from the agent-run cases, not shown by them. These steps confirm them, with the
   check build of PLAN P6. The owner reports one word per step.

   1. **Open Console Host with PowerShell 5.1:** Win+R, `conhost.exe powershell.exe`. Paste the setup lines of the
      earlier list's step 2, then `$env:HTTPS_PROXY = 'http://127.0.0.1:9'; $env:HTTP_PROXY = $env:HTTPS_PROXY`.
      Report: nothing.
   2. **The report's path.** Run `& $g`; type `y`, Enter; `1`, Enter. Report: does the key prompt wait, yes or no.
   3. **The key is hidden.** Type a few letters, then Ctrl+C. Report: only `*` shown, yes or no; the offer came back,
      yes or no.
   4. **Ctrl+C at a menu.** Type `y`, Enter; at `Number [1]:` press Ctrl+C. Report: the offer came back, gobble ended,
      or other.
   5. **Typed-ahead lines are dropped.** Close gobble (`n`, Enter, then `/exit`; if the first `n`, Enter after step 4's
      Ctrl+C is not taken, type it again: Q16 decides whether it is; (2026-11-20, fourth pass), narrowed by D17: under Q16's answer a read the person interrupted is not dropped, so the first `n`, Enter is taken), run `& $g` again, and at the first
      offer paste these two lines at once: `n` and `hello`. Report: anything printed after gobble's prompt appeared,
      yes or no.
   6. **Ctrl+Z.** Type `/exit`, Enter; run `& $g`; `y`, Enter; `1`, Enter; at the key prompt press Ctrl+Z. Report:
      gobble ended, yes or no.
   7. **Windows Terminal with PowerShell 7:** open a PowerShell 7 tab and do steps 1 (without `conhost.exe`), 2 and 4
      again. Report as there.
   8. **Clean up.** `& $g auth status`: every row `none`. Close both windows and delete `gobble-check-home` and
      `gobble-check-work` under `%TEMP%`. Report: every row `none`, yes or no.

   Not in (B): the masked count, the empty key, the declines, `/login`, `gobble setup`, `gobble acp --login`, the
   flush before a repeated offer and the abandoned read; each is in (A), and a browser sign-in would need a real
   sign-in, which the owner does not make for a test. The 0.6.0 reproduction in the earlier list is not needed: the
   agent-run cases run at the base first (PLAN P1 step 0 and P1b).

### State table

*Amendment, 2026-11-20: the records pass before P1, on the owner's ruling "Add the table before P1, when 0063 is
released (Recommended)". Derived from D1 to D11, the owner's answers Q1 to Q9 and the code at `b1994a0`. A row is
`decided` only where this record or an owner answer decides it; "as today" in a row is this record's own words (D2,
D4, D10), not a reading of the code. Tests named "proposed" are the PLAN's additions of 2026-11-20, each with its plant
in P3. "E" is each line end the PLAN's `eachEnter` gives (`\r`, `\n`, `\r\n`). Owner answers are quoted by label in
"Owner questions".*

**Line ends (D3, Q1)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R1 | A question's answer ends with LF | decided | The answer is the text before `\n`; nothing is owed | D3; Q1 "CR, LF and CR+LF each end one line; LF then CR counts as two (Recommended)" | `TestLineEndsBeforeEachQuestion/LF/*` (P1); `terminput_test` table (P2) |
| R2 | A question's answer ends with CR, read out of raw mode | decided | The answer is the text before `\r`; a `\n` next, now or in a later read, is that end's and is dropped | D3 | `TestLineEndsBeforeEachQuestion/CR/*` (P1); `terminput_test` table (P2) |
| R3 | A question's answer ends with CR+LF in one read, then a secret (the report's bytes) | decided | One end; the secret gets the key typed after it, not `""` | D3; Q1 | `TestLineEndsBeforeEachQuestion/CRLF/*-secret`, red for the defect (P1); P3 mutation 1 |
| R4 | CR and LF of one end split across two reads, before a line question | decided | The `\n` opening the next read is dropped; the next answer is the next line | D3 (`lfOwed`) | `TestCRLFSplitAcrossReads` (P1, guard); `terminput_test` "owed `\n` in a later read" (P2) |
| R5 | CR and LF of one end split across two reads, before a secret (a line longer than the read's buffer) | decided | The secret gets its key; the owed `\n` is dropped though it arrives after raw mode is on | D3 ("whether it is already in `pending` or arrives in a later read") | proposed: `TestLineEndsBeforeEachQuestion` form `CR\|LF` (P1) |
| R6 | LF then CR | decided | Two ends: `y\n` then `\r` at a secret is an empty secret, which cancels | Q1 (a); D3 | `TestLFThenCRIsTwoEnds` (P1); P3 mutation 3 |
| R7 | A raw secret ends at `\r`; the next line question's Enter arrives as `\n` in a later read (weakness (d)) | decided | Nothing is owed; that question's answer is `""` at once | D3 | `TestEnterAfterSecretAnswersTheNextLine`, red for the defect (P1); P3 mutation 2 |
| R8 | A secret given `key\r\n` in one read (a pasted key) | decided | The secret is `key`; the `\n` already read is dropped | D3; D5 | `TestPastedKeyEndingCRLF` (P1); P3 mutation 7 |
| R9 | A raw read ended by `\n` (Ctrl+J), then `\n` | decided | The second `\n` is given; only a `\r` drops or owes a `\n` | D3 | proposed: `terminput_test` row (P2) |
| R10 | The owed `\n` meets each entry point (`line`, `key`, `Read`); any other byte first | decided | Dropped at each; any byte given clears it, so the line editor's first key loses a lone Ctrl+J only (Consequences) | D3, last bullet; Consequences | `terminput_test` "a byte that clears `lfOwed`" (P2); proposed: the same rule driven through each entry point (P2) |
| R11 | A lone CR, then the end of input | decided | The line before the `\r` is the answer; the next question meets the end of input and `ended()` is true | D3 ("A line ends at `\r`"); D8 | proposed: `terminput_test` row (P2) |

**Empty input and the end of input**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R12 | An empty line at a menu, a yes or no, an `input` | decided | The menu's default, the yes or no's default, `""`: as today | D2 ("then `plainLine` as today") | `TestQuestionsChoose`, `TestQuestionsConfirmAndInput` (existing; through `terminalInput` from P2 step 4) |
| R13 | An empty line at the key prompt | decided | The sign-in is cancelled with today's words; in the first run the offer comes back | Q4 "Still cancel, with today's wording (Recommended)"; D7; D8 | `TestEmptyKeyCancels` (P1); `TestFirstRunOfferAgainAfterEmptyKey` (P1) |
| R14 | The end of input before any byte, at a question | decided | The question is cancelled (`errCancelled`); `ended()` is true | D4; D8 | `TestQuestionsChoose` (existing); `terminput_test` "`ended()` true after the end of input" (P2) |
| R15 | The end of input before any byte, at the first-run offer | decided | The loop ends as a decline: the lead and the warning at most once, no menu, the session ends within the bound | D8 ("the end of input, at the offer or inside a round") | proposed: `TestFirstRunEndOfInputAtOfferEnds` (P1) |
| R16 | The end of input in the middle of a line or of a secret | decided | The question is cancelled; the partial text is no answer and stays in `pending` | D4 ("A cancelled question (Ctrl+C, the end of input, an ended context) leaves `pending` as it is") | `TestQuestionsSecret` (existing, `{"sk"}`); proposed: `TestLineEndOfInputMidLine` (P1) |
| R17 | Input that never ends a line; a very long line or paste | ~~open (Q13)~~ decided (2026-11-20) | ~~Today: `questions.pending` (`prompt.go:199-208`) and a secret's buffer (`prompt.go:251-294`) grow without a bound~~ No bound: an answer of any length is read whole and given to its question; nothing is kept past the sign-in (D4) | D15; Q13, the owner, relayed by the coordinator: "No bound; record it (Recommended)" | proposed: `TestLongAnswerHasNoBound` (A25, P2) |

**Questions in sequence, type-ahead and paste**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R18 | A menu answered with E, then at once the key prompt (the report): the first run, `/login` with no argument, `gobble setup`, `gobble acp --login` with a method menu | decided | The key prompt waits and takes the key typed | Context; D3; D10 | `TestFirstRunKeyOnlyProviderWaits`, `TestFirstRunMethodMenuProviderWaits`, `TestLoginNoArgumentMenuWaits`, `TestSetupKeyWaits`, `TestAcpLoginMethodMenuWaits`, each `CRLF` red for the defect (P1); ~~P1b gobble mode; P6 steps 4, 8, 9, 12, 13~~ (2026-11-20, fourth pass) the cases `report`, `method-menu`, `setup` and `acp-login` (P1b, agent-run); the owner's step (B) 2 (P6) |
| R19 | A question answered with E, then at once a visible-text question (Claude's workspace after a key; a menu; a yes or no) | decided | The second question gets its own answer | D3 | `TestLineEndsBeforeEachQuestion/*/*-input`, `-choose`, `-confirm` (P1, guards) |
| R20 | `/login <provider>` for a provider with no method menu | decided | The key prompt waits, before and after | D10 | `TestLoginNoArgumentMenuWaits` (its `/login gemini` guard, P1) |
| R21 | `/login <provider>` for a provider with a method menu (OpenAI, Grok, Kilo) | decided | The key prompt waits after the method menu | D10 ("a method menu … skips the key" → "waits after every menu") | proposed: `TestLoginProviderMethodMenuWaits` (P1) |
| R22 | Answers typed ahead across the offer and the menu, in one read | decided | Kept across asks: the key prompt appears with no second press | D4; weakness (a) | `TestFirstRunTypeAhead`, red for the defect (P1); P3 mutation 4. (2026-11-20) That pins bytes gobble has read, on a pipe; on a real terminal one line read returns one line and the terminal holds the rest, which no flush inside a round drops (D12): the `typeahead` case and A14 (P1b), plant 42 |
| R23 | A menu answer, E and the key in one read, before the key prompt | decided | The key bytes in `pending` are the secret | D4 ("a method menu and the key share one `pending`") | proposed: `TestLineEndsBeforeEachQuestion` form "one chunk" (P1) |
| R24 | Keys typed ahead while the terminal is out of raw mode are echoed by the terminal, a key among them | out of scope | The terminal echoes at the time of typing, before any reader runs; no reader can stop it. D6 hands over bytes already read | — | — |
| R25 | Pasted multi-line text into a question | decided | The first line answers it; the rest feeds the next questions of the same sign-in and is dropped at the session prompt | D5; D4; Q3 "Dropped silently, as today (Recommended)" | `TestFirstRunTypeAhead` (P1); `TestSignInTypeAheadNotSentAsPrompt` (P1); P3 mutation 5. (2026-11-20) Those run on a pipe, where one read returns every line; the lines a real terminal holds are dropped by D12's flush: `flush-prompt` and `pty-flush-prompt` (A21, P1b), plant 43; the owner's step (B) 5 on Console Host |
| R26 | Bracketed paste while a question reads | decided | Off while a question reads; markers taken out of a line | D5 | `TestLoginSlash`, `TestQuestionsConfirmAndInput` (existing) |

**The secret prompt and the terminal's mode**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R27 | Raw mode switched on for a secret | decided | `*` for each character; the value written nowhere | Decision drivers; D2 (deviation 7 stands) | `TestQuestionsSecret` (existing); ~~P6 step 5~~ (2026-11-20, fourth pass) the case `masked` (P1b, agent-run); the owner's step (B) 3 (P6) |
| R28 | Raw mode cannot be switched on | ~~decided~~ ~~open (Q18) (2026-11-20)~~ decided (2026-11-20, third pass) | ~~The secret returns the error, writes no prompt and reads nothing; in the first run that round fails and the offer comes back~~ Today (`prompt.go:237-241`): the secret returns the error, writes no prompt and reads nothing. The drivers decide only that no key is read with echo on; what happens instead is today's code, not a decision. (2026-11-20, third pass) Recorded as the outcome: the error is returned, no prompt is written, nothing is read; in the first run that round fails and the offer comes back (D8) | ~~Decision drivers ("A key is never echoed in clear"); D8 ("any other error the sign-in returns")~~ ~~none~~ D17; Q18, the owner, relayed by the coordinator: "Keep that, recorded and tested (Recommended)"; D8 | ~~proposed: `TestSecretRawFailureReadsNothing` (P1)~~ ~~A3 waits for Q18~~ `TestSecretRawFailureReadsNothing` (A3, P1), plant 19 |
| R29 | A secret ends by Enter, Ctrl+C (byte `0x03`) or the end of input | decided | Raw mode entered once and restored once before the next question, which reads out of raw mode | D2 ("They read out of raw mode"; "in raw mode (deviation 7 stands)") | `TestQuestionsSecret` (existing) |
| R30 | The context ends while a secret reads | decided | `errCancelled`; restored once | D2; D4 | proposed: `TestSecretRestoredWhenContextEnds` (P1) |
| R31 | A panic while a secret has raw mode on | ~~open (Q14)~~ decided (2026-11-20) | ~~Today: the restore is not deferred (`prompt.go:237-244`) and `Main` recovers only Kong's exit (`main.go:175-184`): the terminal stays raw~~ The restore is deferred: raw mode is restored once before the panic goes on | D15; Q14, the owner, relayed by the coordinator: "Defer the restore at the secret prompt (Recommended)" | proposed: `TestSecretRestoredOnPanic` (A26, P1; red at the base) |

**Kinds of standard input**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R32 | A POSIX terminal in cooked mode, Enter as `\n` | decided | As R1 | D3 | the `LF` forms (P1) |
| R33 | A Windows console in Console Host (Windows PowerShell 5.1) | decided | The key prompt waits; ~~the rest of the check as written~~ (2026-11-20, fourth pass) each step of Confirmation item 5 (B) gives its expected word | Decision drivers; D6; Confirmation item 5, as amended | ~~P6, Console Host~~ the owner's steps (B) 1 to 6, in Console Host (P6) |
| R34 | A pseudo console (Windows Terminal, PowerShell 7.x) | decided | A line-mode read gives `"1\r\n"` (D6's third assumption, to be measured); the key prompt waits | D6; Q7 "A committed pseudo-console test, run on the Windows laptop by a make target, outside CI (Recommended)" | ~~P1b probe and gobble mode; P6, Windows Terminal~~ (2026-11-20, fourth pass) the cases `probe-crlf` (P1 step 0b) and `report` (P1b), agent-run; the owner's step (B) 7, in Windows Terminal (P6) |
| R35 | A Windows console: keys typed past a line-mode read | decided | They stay in the console's buffer for the next read (D6's third assumption, to be measured) | D6 | ~~proposed: P1b gobble mode, typed ahead (P1b)~~ (2026-11-20) `probe-typeahead` measures it (P1 step 0); A14, as corrected, and the `typeahead` case pin that no flush inside a round drops it, plant 42 |
| R36 | Standard input not a terminal: a pipe, a file, the null device, closed; and the uses of a non-terminal stdin (a piped prompt, `input.go:53-56`; a piped `--api-key -`, `credentials.go:222`; the ACP protocol, `acp.go:49-53`) | out of scope | No question is asked off a terminal: every caller checks first (`login.go:100`, `:325`; `auth.go:310`; `chat.go:49`; `line.go:137`), and a non-terminal stdin selects print mode (`mode.go:23-24`); this record changes none of these | — | — |

**Interrupts, contexts and read errors**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R37 | Ctrl+C (the signal) at a menu out of raw mode in the line session | decided | The question is cancelled through `ls.ask`; in a first-run round the offer comes back; in `/login` the cancel line | D4; D8 | `TestFirstRunOfferAgainAfterMenuInterrupt`, red for the loop (P1); `TestLoginInterrupt` (existing) |
| R38 | Ctrl+C at a repeated offer | decided | The loop ends as a decline: the warning once, no menu | D8 ("Ctrl+C at the offer") | proposed: `TestFirstRunInterruptAtRepeatedOfferEnds` (P1) |
| R39 | Ctrl+C (byte `0x03`) at the key prompt | decided | The sign-in is cancelled; in the first run the offer comes back | D4; D8 | `TestFirstRunOfferAgainAfterKeyCancel` (P1); `TestQuestionsSecret` (existing) |
| R40 | Ctrl+C during a read out of raw mode on a Windows console | ~~open (Q11)~~ ~~open: awaits live evidence (2026-11-20)~~ decided (2026-11-20, after P1 step 0b) | Inferred, not measured: by reading Go 1.27.4, a console read that gives no characters is the end of input (`internal/poll/fd_windows.go` `readConsole`, `os/file_windows.go:110`). Observation "the read still waits" (`probe-ctrlc` `waiting`, `menu-ctrlc` `offer-again`): the build goes on, and R37 holds on Windows. Observation "end of input" or an error (`eof`, `error`, `ended`): a named stop; the owner chooses among (a) a console read that gives nothing right after an interrupt gobble took is not the end of input, and `terminalInput` reads again (Windows only; a stubbed test and a plant), (b) menus on a Windows console read in raw mode with gobble's own line echo (option C, rejected once), (c) accept it and amend D8 and the checks: on a Windows console Ctrl+C at a menu ends the input | D14; Q11, the owner, relayed by the coordinator: "Measure both in your Windows check; stop if Ctrl+C ends input (Recommended)" | `probe-ctrlc` (P1 step 0, before any code); `menu-ctrlc` (P1b); the owner's step (B) 4 (P6). (2026-11-20, after P1 step 0b) Measured: `probe-ctrlc` observed `eof`, twice. Option (a) chosen, the owner, relayed by the coordinator: "(a) After an interrupt gobble took, read again (Recommended)": D19, rows R109 to R121 |
| R41 | A read the context gave up on (Ctrl+C's cancel; `Input`'s context) | decided | The wait ends; the read stays `inline.Reader`'s and is taken by the next read; `ended()` is false | D4, last bullet. (2026-11-20) Narrowed by D13: it holds for a wait inside a prompt that reads again (R87); the line of a prompt that gave up is dropped (R84 to R89) | `TestQuestionsPrompter` (existing); `terminput_test` "`ended()` false after an ended context" (P2) |
| R42 | The process's context ends during a first-run round | decided | Its cause is returned; no further offer | D8 ("the process's own context ending, whose error is returned as today") | proposed: `TestFirstRunContextEndReturns` (P1) |
| R43 | A read fails with an error that is not the end of input | decided | The question is cancelled; `closed` is set and `ended()` is true; the loop ends as a decline | D8 ("the end of input, or a read failure") | proposed: `terminput_test` row (P2) |
| R44 | Bytes not valid UTF-8, or any control byte | decided | ~~`terminalInput` gives every byte unchanged; each reader decodes as it does now~~ (2026-11-20) `terminalInput` gives every byte it reads unchanged, except the `\n` D3 drops; each reader decodes as it does now. On a Windows console `0x1A` never reaches it: the runtime makes it the end of input (R90) | D6 ("Each byte read is in exactly one place"); Q2 "Every reader in the process, each keeping its own decoding (Recommended)" | proposed: `terminput_test` row (P2) |

**Readers of the terminal**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R45 | The process's readers: the line session, `gobble setup`, `gobble acp --login` with a menu, `gobble auth login`'s paste prompter, the trust question | decided | Each reads through `e.terminalIn()`; nothing else calls `inline.NewReader` on stdin | D1; Q2 | `TestFirstRunTypeAhead` (P3 mutation 4); `TestTrustReadsTerminalInput` (P2; P3 mutation 8); proposed: `TestOneReaderOfStdin` (P2) |
| R46 | A reader made before the shared input (the trust question, `chat.go:49-50`) and one after (the line session) | decided | One `terminalInput`, made on first use | D1 | `TestTrustReadsTerminalInput` (P2); the harness's `e.termIn` (P2 step 6). (2026-11-20) Neither pins "made once": proposed `TestTerminalInMadeOnce` (A22, P2), plant 51 |
| R47 | No read ahead | decided | `in` is read only for a prompt that needs a byte and finds `pending` empty, with that prompt's buffer, once per need | D1 | `terminput_test` read count (P2); `TestLineSessionReadsOnlyWhenAsked` (existing); P3 mutation 6 |
| R48 | Two goroutines read at once | decided | No data race: the lock guards `pending`, `lfOwed` and `closed` and is never held across a read of `in`; every byte is given once | D1 | proposed: `TestTerminalInputConcurrentReads` under `-race` (P2; run in P4 step 5) |
| R49 | Two prompts asked at once (two permission requests in flight) | out of scope | This record changes where a prompt's bytes come from, not when prompts are asked; `inline.Reader` lets one read wait at a time, as now | — | — |
| R50 | The trust question's reads, given `askOn`'s deadlines | ~~open (Q12)~~ decided (2026-11-20) | ~~Today it reads `e.stdin` and ignores deadlines (`trust.go:164-167`, `:184`), so no read is left waiting; D2 does not say which context it reads with~~ It reads `terminalInput.Read` with the run's context and no deadline; a timed wait waits for the next key; no read is left waiting | D15; Q12, the owner, relayed by the coordinator: "No deadline, as today (Recommended)" | proposed: `TestTrustReadsWithoutDeadline` (A27, P2), plant 55 |
| R51 | The session's main loop and the first-run wizard take turns | decided | The first run ends before the editor's first read; before each `ReadPrompt`, `pending` is dropped and `lfOwed` kept | D4; Q3 | `TestSignInTypeAheadNotSentAsPrompt` (P1; P3 mutation 5); `terminput_test` "`discard` keeps `lfOwed`" (P2). (2026-11-20) On a pipe only; what a terminal holds: `flush-prompt`, `pty-flush-prompt` (A21), plant 43 |
| R52 | Readers of a terminal stdin D1 does not list: `x/term` `ReadPassword` for `gobble --api-key -` in the line session's process (`credentials.go:213`, from `chat.go:64-66`, after the trust question) and for `gobble auth login --api-key -` (`auth.go:122`); `gobble auth import --from-pi`'s questions (`piimport.go:61-94`); `gobble update`'s confirmation through go-selfupdate-lib (`update.go:143`, `:166-181`) | ~~open (Q10)~~ replaced by R91 to R94 (2026-11-20) | Today each reads the terminal itself, a byte or a line at a time, with no read ahead | Q10, the owner, relayed by the coordinator: "Record them as outside the shared input, with reasons (Recommended)" | — |
| R53 | A child that inherits stdin while gobble holds input: the external editor of `/edit` and ctrl+x ctrl+e (`slash.go:226-230`), and of `gobble config edit` (`config.go:284-291`) | ~~open (Q10)~~ replaced by R95 and R96 (2026-11-20) | Today the child reads the terminal directly; a read the context gave up on, if one is waiting in `inline.Reader`, would take the child's first keys | Q10, as R52 | — |
| R54 | Children that do not inherit stdin: the browser opener (`browser.go:14-24`), the shell tool (`tool/builtin/shell.go:85`), MCP servers (a pipe), a `models.json` command | out of scope | ~~None of them reads the terminal~~ (2026-11-20) They are given no stdin; one that opens the terminal itself (`/dev/tty`, `CONIN$`) reads it beside gobble, which no reader in gobble can prevent | — | — |
| R55 | `gobble acp --login` for a provider with no sign-in (`ReadPassword` and the `bufio` workspace read) | out of scope | D2, last bullet: no menu comes before them in that process; the PLAN's Scope, Out | — | — |
| R56 | What one reader decoded past its answer, when another reads next: the line editor's `held` keys | decided | They stay the editor's: keys read with `/login`'s Enter are the editor's next prompt, not the menu's answer | Q2 (a) ("The editor and the drawn prompts keep what they decode past an answer, as today") | proposed: `TestEditorKeepsKeysTypedWithLogin` (P1) |
| R57 | The same, for `askOn` (the permission prompt, the trust question) | ~~decided~~ ~~open (Q19) (2026-11-20)~~ decided (2026-11-20, third pass) | ~~Dropped by `askOn` when it returns; never given to the next reader~~ Today `askOn`'s decoder is local, so what it decoded past an answer is dropped when it returns; Q2 (a)'s words say the drawn prompts "keep" it, so the record does not decide it. (2026-11-20, third pass) Dropped when `askOn` returns, as today; never given to the next reader | ~~Q2 (a)~~ ~~none~~ D17; Q19, the owner, relayed by the coordinator: "Dropped, as today (Recommended)" | ~~proposed: `TestAskOnKeysPastAnswerNotShared` (P2)~~ ~~A17 waits for Q19~~ `TestAskOnKeysPastAnswerNotShared` (A17, P2), plant 21 |
| R58 | Bytes `inline.Reader` kept, `pending`, and the prompt that takes them | decided | Each byte is in one place at a time and given once | D6 | proposed: `TestTerminalInputBytesInOnePlace` (P2) |
| R59 | `/logout`'s menu | decided | Read through the shared input, as every question | D2 | `TestLogoutSlash` (existing) |
| R60 | The permission prompt | decided | Its keys come through `terminalInput.Read` | D2; D10; Q2 | `TestPermissionPrompt`, `TestPermissionPromptOnceOnly` (existing, through `ls.keys` from P2 step 5). (2026-11-20) They cannot tell which reader gave the keys: proposed `TestPermissionReadsTerminalInput` (A23, P2), plant 52 |

**The first run's repeated offer (D8, Q5, Q9)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R61 | The first offer, `[Y/n]`: Enter alone | decided | A round starts: the provider menu is shown | Q9 ("The first offer keeps `[Y/n]`, as today") | proposed: `TestFirstRunEnterAtFirstOfferSignsIn` (P1) |
| R62 | The first offer: `n` | decided | The lead and the warning once; the question once | D8; Q5 | `TestFirstRunDeclineEnds` (P1) |
| R63 | The first offer: Ctrl+C | decided | As `n`, no menu | D8 | `TestFirstRunInterruptAtOfferEnds` (P1) |
| R64 | A repeated offer, `[y/N]`: Enter alone | decided | The loop ends with the warning; no menu | Q9 "The repeated offer is `[y/N]`: Enter ends the loop, `y` tries again (Recommended)" | `TestFirstRunRepeatedOfferDefaultsNo` (P1); P3 mutation 13 |
| R65 | A repeated offer: `y` | decided | Another round: the provider menu shown again | Q9; D8 | proposed: `TestFirstRunYesOnRepeatStartsAnotherRound` (P1) |
| R66 | A repeated offer: `n` | decided | The warning once; the session's prompt runs a turn | D8 | `TestFirstRunDeclineOnRepeatEnds` (P1); P3 mutation 9 |
| R67 | A repeated offer: other text | decided | `Answer y or n.` and the offer asked again | D2 (`confirm` as today) | `TestQuestionsConfirmAndInput` (existing) |
| R68 | A repeated offer: the end of input | decided | The loop ends as a decline, within the bound | D8 | `TestFirstRunClosedInputEnds` (P1); ~~P3 mutation 10~~ (2026-11-20) plant 22: mutation 10 leaves it passing, since the offer's own end of input is already a decline |
| R69 | The loop's bound | decided | No round counter; read-ahead dropped before each repeated offer, so no round starts without a fresh read | D8 | `TestFirstRunTypeAheadDroppedBeforeRepeatedOffer`, `TestFirstRunInstantFailureClosedInputEnds` (P1); P3 mutations ~~10,~~ 12. (2026-11-20) Mutation 10 is caught by `TestFirstRunEndOfInputInRoundEnds` (A20), which ends the input inside a round and counts the offer; what a terminal holds before a repeated offer: `flush-offer`, `pty-flush-offer` (A21), plant 44 |
| R70 | A round fails: three refused keys, offline, an outage, any error | decided | The round's line, then the offer; no warning between rounds | D8 | `TestFirstRunOfferAgainAfterFailure` (P1); P3 mutation 14 |
| R71 | A round succeeds | decided | The loop ends; the offer was asked once | D8 | `TestFirstRunSetup` (existing, each E from P1 step 2); proposed: its offer counted once (P1). (2026-11-20, at P1's build pass 1; the owner, relayed by the coordinator: "Red for the defect in CR+LF; guard in CR and LF (Recommended)") Its `CRLF` subtest is red for the defect at the base (the key prompt cancels before the key) and passes after P2; the count is a guard in `CR` and `LF` and is checked in `CRLF` from P2 on; P3 mutation 1 must fail it |
| R72 | `/login`, `gobble setup`, `gobble acp --login` | decided | Each asks once; no offer, no loop | D8 ("`/login`, `gobble setup` and `gobble acp --login` do not loop") | `TestLoginDoesNotLoop`, `TestSetupDoesNotLoop` (P1); proposed: a cancel case in `TestAcpLoginMethodMenuWaits` (P1) |
| R73 | One Enter turned into two line ends by the terminal (weakness (c)) | out of scope | Left and recorded | Q6 "Leave it and record it (Recommended)"; D9; the PLAN's Scope, Out | — |

~~Counts: 61 `decided`, 6 `out of scope` (R24, R36, R49, R54, R55, R73), 6 `open` (R17, R31, R40, R50, R52, R53).~~

*Rows added 2026-11-20 (second records pass), from the owner's rulings D12 to D16, the review's cases and its
suggested exclusions. "Inferred" marks a statement about a Windows console read from source, not measured; the case
that measures it is named. The `0063 case=` names are Confirmation item 5 (A), as amended.*

**The terminal's own input queue (D12)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R74 | A POSIX terminal or pseudo-terminal, input it holds at the session prompt or before a repeated offer | decided | `tcflush(TCIFLUSH)` empties it: no held line becomes a turn or answers the repeated offer | D12; review case 1, the owner, relayed by the coordinator: "Flush the terminal's input at those points (Recommended)" | `pty-flush-prompt`, `pty-flush-offer` (A21, P1b, agent-run on the Linux machine); plants 43, 44 |
| R75 | A Windows console in Console Host, the same | decided | `FlushConsoleInputBuffer` empties it (inferred) | D12; the same ruling | the owner's step (B) 5 (P6), the session-prompt point only; the repeated-offer point is inferred from R76. No plant: a real console |
| R76 | A pseudo console (Windows Terminal), the same | decided | The same call on the pseudo console's input empties it (inferred; measured by the cases) | D12; the same ruling | `flush-prompt`, `flush-offer` (A21, P1b, agent-run on the Windows test laptop); plants 43, 44 |
| R77 | A pipe or a file | decided | No flush: there is no terminal queue to empty, and questions are never asked off a terminal (R36). In tests, the flush is the `termFlush` seam the harness stubs; a pipe's unread bytes stay in the pipe and come, in order, to the next read, since the shared input never reads ahead | D12 ("the terminal's own input queue"); R36 | proposed: `TestFlushPoints` (A24, P2), plant 42 |
| R78 | The flush call fails (not a terminal after all, a console handle under another program, an I/O error); also a system with no flush call (`flush_other.go`) | ~~open (Q17)~~ decided (2026-11-20, third pass) | ~~Today: no flush exists~~ gobble goes on without the flush and writes one warning on stderr, once per process, whichever point failed; its words are R101's | D17; Q17, the owner, relayed by the coordinator: "Go on, with one warning on stderr (Recommended)" | proposed: `TestFlushFailureWarnsOnce` (A29, P2), plant 59 |
| R79 | Input that arrives between the flush and the read | decided | It is kept and answers the prompt shown: at a repeated offer the flush runs after the offer is written, so only what was typed before it was shown is dropped | D12; D8 ("nothing typed before it may answer it"), which the ruling says holds as written | `TestFlushPoints` (A24: the flush is called after the offer's text is written); `flush-offer`; plant 45 |
| R80 | Which points flush | decided | Exactly two: with `discard()` before the session prompt reads, and after a repeated offer is written, before its read | D12; the same ruling | `TestFlushPoints` (A24); plants 42, 43, 44 (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R81 | Points that do not flush | decided | The first offer (D8 keeps read-ahead from the offer to the key in the first round); every question inside a round (D4); `gobble setup`, `gobble acp --login` and `gobble auth login` (no session prompt and no repeated offer; the process ends after); the trust question (before the session; not one of the two points); the permission prompt (not one of the two points: keys typed while a turn runs can answer it, as today, a matter outside this record) | D12: the ruling names two points and no other | `TestFlushPoints` (A24, no call at each); plant 42 (2026-11-21, D22 second draft: a flush at the program's give-up is Q23 (a), as rewritten, which would amend this row) (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R82 | The session-prompt flush: before every `ReadPrompt`, or only the first after a sign-in | ~~open (Q20)~~ decided (2026-11-20, third pass) | ~~Not settled: "at the session prompt" with D4's "before each `ReadPrompt`" would also drop keys typed while a turn runs, which no ruling asked for~~ Only before the first `ReadPrompt` after a sign-in or the first run, by a mark: rows R103 to R108; keys typed while a turn runs are kept for the next prompt, as today | D17; Q20, the owner, relayed by the coordinator: "Only the first prompt after a sign-in or the first run (Recommended)" | `TestFlushPoints` (A24, as extended), plants 56, 57, 58 |
| R83 | A flush while a read of a prompt that gave up is still waiting | decided | The flush runs first; the waiting read's completion is then dropped by D13; the prompt that waits reads again. On a Windows console, whether the flush also empties the line being edited in a waiting line-mode read is inferred, and measured by `abandoned` | D12; D13 | `abandoned`, `pty-abandoned` (A21, P1b); plant 46 (2026-11-21, D22 second draft: on POSIX, see its D22 latest row, after R152) |

**The read of a prompt that gave up (D13)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R84 | A read started out of raw mode by a prompt that gave up (a browser sign-in's paste prompt), on POSIX, completed by a line while the terminal is still out of raw mode | decided | The line is dropped: never a turn, never an answer; the prompt waiting reads again | D13; review case 2, the owner, relayed by the coordinator: "Drop the abandoned prompt's line when it arrives (Recommended)" | proposed: `terminput_test` abandoned rows (A28, P2); `pty-abandoned` (A21); plant 47 (2026-11-21, D22 second draft: on POSIX, see its D22 latest row, after R152) |
| R85 | The same read on a Windows console, which keeps it in line mode | decided | The console echoes and edits what is typed at the next prompt until Enter (inferred); that line, ending `\r\n`, is dropped; the prompt's own read follows in its own mode | D13; the same ruling | `abandoned` (A21, P1b, a stubbed browser sign-in): pinned by the pseudo-console test only; the owner makes no sign-in for a test |
| R86 | A completion that is not a line: a read started in raw mode by a prompt that gave up, or a POSIX read started out of raw mode that completes after raw mode is on; and a read the user gave up with Ctrl+C rather than the program | ~~open (Q16)~~ decided (2026-11-20, third pass) | ~~Not settled by the ruling's words ("the abandoned prompt's line")~~ Whatever completes a read the program gave up (a browser sign-in's loopback; any read whose context the program, not the person, ended) is dropped, line or keys. A read the person gave up with Ctrl+C is not dropped and goes to the next prompt, as D4 says | D17; Q16, the owner, relayed by the coordinator: "Only reads the program gave up (Recommended)" | `terminput_test` (A28, as extended): a program give-up dropped, keys included; a Ctrl+C give-up kept; `TestLoginInterrupt` (existing) unchanged; plants 47, 60; (2026-11-20, fourth pass) "whole" by D18: the 512-through-256 row of A28, plant 63. (2026-11-21, D21: what completes such a read on POSIX, measured; see R86's latest row, after R122) |
| R87 | A timed wait inside a prompt that is still reading (the editor's Esc timer, `askOn`'s timers) | decided | Not a prompt that gave up: the same prompt's next read takes the read, as D4 says | D13 ("a read belonging to an abandoned prompt"); D4 | proposed: `terminput_test` (A28); the editor's own tests (existing); plant 48 |
| R88 | A second prompt gives up while the first one's read still waits | decided | `inline.Reader` has one read in flight, so both belong to it; its completion is dropped once, and the next prompt reads anew | D13 | proposed: `terminput_test` (A28); plant 49 (2026-11-21, D22 second draft: on POSIX, see its D22 latest row, after R152) |
| R89 | The input ends while a prompt that gave up has its read waiting | decided | The end of input is not a line and is not dropped: `closed` is set, `ended()` is true, and D8 applies | D13 (lines only); D8 | proposed: `terminput_test` (A28); plant 50 (2026-11-21, D22 second draft: on POSIX, see its D22 latest row, after R152) |

**Ctrl+Z on a Windows console (D14)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R90 | Ctrl+Z (`0x1A`) at the key prompt, or Ctrl+Z then Enter at a menu, on a Windows console | decided | The end of input (inferred from `fd_windows.go:710-715` and `os/file_windows.go:110`): the question is cancelled, `closed` is set, the first run ends as a decline, the session ends at the editor's next read. On POSIX it is a byte (R44) | D14, from D8 ("the end of input"); Q11, the owner, relayed by the coordinator: "Measure both in your Windows check; stop if Ctrl+C ends input (Recommended)" | `probe-ctrlz-line`, `probe-ctrlz-raw` (P1 step 0); `key-ctrlz` (P1b); the owner's step (B) 6. No plant: the runtime's behaviour; another observation is a deviation |

**Readers outside the shared input (Q10)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R91 | `x/term` `ReadPassword` for `gobble --api-key -` on a terminal, in the line session's process (`credentials.go:213`, from `chat.go:64-66`) | out of scope | It reads after the trust question, which now reads with no deadline (R50), and before the session's first question, so nothing waits in `inline.Reader` when it reads; it reads a byte at a time through its own line end and leaves nothing | Q10, the owner, relayed by the coordinator: "Record them as outside the shared input, with reasons (Recommended)" | — |
| R92 | `ReadPassword` for `gobble auth login --api-key -` and `--method api-key` (`auth.go:122`) | out of scope | Alone in its process: no other prompt reads the terminal there | Q10 | — |
| R93 | `gobble auth import --from-pi`'s `[y/N]` questions (`piimport.go:61-94`) | out of scope | Alone in its process; it reads a byte at a time, so nothing past an answer is taken | Q10 | — |
| R94 | `gobble update`'s confirmation, through go-selfupdate-lib (`update.go:143`, `:166-181`) | out of scope | Alone in its process; the library reads a byte at a time | Q10 | — |
| R95 | The external editor of `/edit` and ctrl+x ctrl+e, which inherits stdin (`slash.go:226-230`) | out of scope | It runs while the line editor is not reading and `terminalInput` holds nothing (D4); a read of a prompt that gave up, if one waits, would take its first keys, which D13 then drops: a rare case left as it is | Q10 | — (2026-11-21, D22 second draft: on POSIX, see its D22 latest row, after R152) |
| R96 | `gobble config edit`'s editor (`config.go:284-291`) | out of scope | Alone in its process; no prompt reads the terminal before it | Q10 | — |

**Excluded on the review's suggestion**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R97 | A terminal handed to gobble already in raw mode | out of scope | The record assumes a terminal in its normal mode, as every gobble prompt does today; each restore returns the terminal to the mode it found | — | — |
| R98 | The first run with stderr redirected (`line.go:137` checks `caps.In` only) | out of scope | The offer is written where stderr goes and waits for an answer as the first offer does today; D8 never repeats it without one. When the questions are shown is outside this record | — | — |
| R99 | A suspend (Ctrl+Z on POSIX, `SIGTSTP`) at a question | out of scope | gobble handles no suspend today; a suspend in a raw secret leaves the terminal raw until it resumes. Outside this record | — | — |
| R100 | `gobble.exe` started from WSL | out of scope | Its stdin is not a Windows console there, so no question is asked (R36) | — | — |

~~Counts (2026-11-20, second pass): 100 rows. R52 and R53 are replaced by R91 to R96 and not counted. Of the other 98:
76 `decided`, 16 `out of scope` (R24, R36, R49, R54, R55, R73, R91 to R100), 6 neither: R40, which awaits live
evidence, and R28, R57, R78, R82 and R86, open as Q18, Q19, Q17, Q20 and Q16.~~

*Rows added 2026-11-20 (third records pass), from the owner's answers to Q16 to Q20 (D17).*

**The flush warning, Q16's consequence, and the mark that makes a session prompt flush (D17)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R101 | The words of R78's warning | ~~open (wording only)~~ decided (2026-11-20, fourth pass) | Proposed, then approved unchanged: `Warning: could not clear what was typed ahead on this terminal (<error>); it may answer the next prompt.` ~~The behaviour around it is decided (R78); only the words await the owner~~ | ~~Q17 decides that a warning is written, not its words~~ D17; the owner, relayed by the coordinator: "Plain words, with the consequence (Recommended)" | `TestFlushFailureWarnsOnce` (A29) asserts these words; plants 59, 62 |
| R102 | The first key or line typed after a browser sign-in whose loopback came first | decided | ~~Lost: on POSIX the first key typed at the next prompt completes the read the program gave up, and is dropped with it;~~ (2026-11-21, D21: measured otherwise on Linux; see R102's latest row, after R122) on a Windows console the first line (inferred: the console keeps that read in line mode, R85). The prompt then reads again | D17; Q16, the owner, relayed by the coordinator: "Only reads the program gave up (Recommended)", whose stated consequence this is | `pty-abandoned`, `abandoned` (A21), plant 47 |
| R103 | A sign-in inside the session: `/login`, or one another command starts through the same round (`ls.signInRound`), whatever its outcome (signed in, cancelled, failed) | decided | Sets the mark: the next `ReadPrompt` flushes. A `/login` refused before any question (an unknown provider) is no sign-in and sets none. (2026-11-21, D21: "flushes" made precise, at once at that prompt; see R103's latest row, after R122) | D17; Q20, the owner, relayed by the coordinator: "Only the first prompt after a sign-in or the first run (Recommended)"; D4 ("Nothing typed or pasted during a sign-in becomes a turn", whatever the sign-in's end) | `TestFlushPoints` (A24, as extended); plant 57 |
| R104 | The first run, however it ends: declined at the first offer, Ctrl+C at an offer, a round signed in, the loop ended by a decline after rounds | decided | Sets the mark: the first `ReadPrompt` flushes | D17; Q20 ("or the first run") | `TestFlushPoints` (A24, as extended); plant 58 |
| R105 | The prompts after that first one: after a turn, a local command, `/edit` | decided | The mark is cleared by the flush it causes; later `ReadPrompt`s do not flush, so keys typed while a turn runs are the next prompt's text, as today. (2026-11-21, D21: with keys the editor holds, see R105's latest row and R123, after R122) | D17; Q20 ("Only the first prompt") | `TestFlushPoints` (A24, as extended); plant 56 |
| R106 | A turn that runs before that first prompt: the `/models` turn after `/login` signs in (`login.go:262`), or a prompt given on the command line after the first run (`line.go:224-228`) | decided | Keys typed while it runs are flushed at the first `ReadPrompt` that follows: the mark is consumed by the prompt read, not by a turn | D17; Q20 ("the first prompt after a sign-in or the first run": the session prompt) | `TestFlushPoints` (A24, as extended); plant 56 |
| R107 | `/logout`, the trust question, a permission prompt, a resumed session with a credential (no first run, no sign-in) | decided | Set no mark: no flush at the prompt after them | D17; Q20 ("Only … after a sign-in or the first run") | `TestFlushPoints` (A24, as extended: no call after each); plant 56 |
| R108 | The process ends, or the session ends, with the mark set | decided | Nothing: the mark lives in the session's `terminalInput` and nothing is flushed on the way out | D17; Q20 (a flush belongs to a prompt read) | `TestFlushPoints` (A24, as extended: ~~a session ended by Ctrl+D with the mark set counts no flush~~ (2026-11-21, D21: re-worded for the flush made at the prompt; see R108's latest row, after R122)); plant 61 |

~~Counts (2026-11-20, third pass): 108 rows. R52 and R53 are replaced by R91 to R96 and not counted. Of the other 106:~~
~~88 `decided`, 16 `out of scope` (R24, R36, R49, R54, R55, R73, R91 to R100), 1 awaiting live evidence (R40, measured~~
~~first at the PLAN's P1 step 0b), and 1 `open` for its words only (R101, the warning of R78).~~
*(2026-11-20, fourth pass: superseded; R101 is decided, see the fourth-pass counts.)*

Counts (2026-11-20, fourth pass): R101 decided on the owner's approval of its words. Of the 106 counted rows, 89
`decided`, 16 `out of scope`, 1 awaiting live evidence (R40), and none `open`.

*Rows added (2026-11-20, after P1 step 0b): D19, the deviation at P1 step 0b. "Stub" tests run on every system over a stubbed console
reader, through the portable part of `term.ConsoleInput`; the plants are the PLAN's P3 64 to 70.*

**Ctrl+C on a Windows console after the measurement (D19)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R109 | Ctrl+C during a read out of raw mode on a Windows console, gobble having received the interrupt | decided | The read that gives nothing is not the end of input: the shared input reads again. At the provider menu the round is cancelled and the offer comes again; `n`, Enter then answers it | D19; the owner, relayed by the coordinator: "(a) After an interrupt gobble took, read again (Recommended)"; D8 | stub `TestConsoleReadAgainAfterInterrupt` (A30, P2); `menu-ctrlc` (P1b): `offer-again` then `warned-once`; plant 64 |
| R110 | How gobble knows an interrupt came | decided | A mark in the process (one, as there is one terminal input per process, D1), set by `watchSignals` when it receives an `os.Interrupt`, before it hands the interrupt on, whether a prompt takes it or not; when the mark is kept and cleared is Q21 | D19 ("right after gobble received Ctrl+C": where gobble receives it); D1 | `TestWatchSignalsMarksInterrupt` (A31, P2); plant 65 |
| R111 | When the mark is set and cleared, and which read that gives nothing it excuses ("right after") | ~~open (Q21)~~ decided (2026-11-20, Q21 and Q22) | ~~Not settled by the ruling's words~~ The mark is set when gobble receives Ctrl+C while a console read is waiting; it excuses only the next read that gives nothing, and is cleared by that read or by any read that returns bytes | D20; Q21, the owner, relayed by the coordinator: "Set only if a read is waiting; excuses one empty read (Recommended)" | stub `TestConsoleMarkExcusesOneRead` (A36, P2); plant 71 |
| R112 | Ctrl+C, then Ctrl+Z, at the same read | decided | The first read that gives nothing is excused; the next, with no new interrupt, is the end of input, so Ctrl+Z still ends the input. Each interrupt excuses one such read at most, however Q21 is answered (its option (b) would break this, and says so) | D14; R90; D19 | stub `TestConsoleCtrlZAfterInterruptEnds` (A32, P2); plant 66 |
| R113 | Two Ctrl+C in a row | decided | The first is taken by the prompt reading (a menu's round is cancelled); a second that a prompt takes is that prompt's (at a repeated offer, a decline, D8); a second that no prompt takes ends gobble with 130 (0008-MADR D5). The mark changes neither | D8; 0008-MADR D5; D19 | `TestInterruptDuringTurn`, `TestInterruptReleased` (existing); plant 67 |
| R114 | Ctrl+C while no read of the console is waiting (during a turn, between prompts) | ~~open (Q21)~~ decided (2026-11-20, Q21 and Q22) | ~~Not settled: whether that interrupt's mark may excuse a read that gives nothing later~~ No mark is set: a read that gives nothing later is the end of input | D20; Q21, the owner, relayed by the coordinator: "Set only if a read is waiting; excuses one empty read (Recommended)" | stub `TestConsoleNoMarkWithoutWaitingRead` (A37, P2); plant 72 |
| R115 | Ctrl+C at a prompt read in raw mode (the key prompt, the line editor, a permission or trust prompt) | decided | Unaffected: raw mode turns processed input off (`golang.org/x/term` `v0.47.0` `term_windows.go:30`), so Ctrl+C is the byte `0x03`, no interrupt is raised and no read gives nothing | D4 ("byte `0x03` in a secret"); D19 ("a read out of raw mode") | stub `TestConsoleRawCtrlCIsAByte` (A34, P2); `key-ctrlc` (P1b); plant 68 |
| R116 | The read gives nothing before `watchSignals` has set the mark (the console's handler runs on a thread of its own; inferred from Go 1.27.4 `runtime/os_windows.go:1047-1069`) | ~~open (Q22)~~ decided (2026-11-20, Q21 and Q22) | ~~Not settled: whether the read waits for the mark, and for how long~~ An empty console read with no mark waits up to 100 ms for one: a mark that arrives in time excuses it; none in time, and it is the end of input. The probe reports which came first (`first=signal` or `first=read`) | D20; Q22, the owner, relayed by the coordinator: "Wait up to 100 ms for the mark; the probe reports the order (Recommended)" | stub `TestConsoleWaitsForLateMark` (A38, P2), plant 73; stub `TestConsoleWaitBound` (A39, P2), plant 74; `probe-ctrlc`'s `first=` word (P1 step 0b, P1b) |
| R117 | Standard input on Windows that is a pipe or a file | decided | No change: the rule applies only when standard input is a console; a read that gives nothing there is the end of input | D19 ("a console read") | stub `TestConsoleRuleOnlyOnAConsole` (A35, P2); plant 69 |
| R118 | POSIX | decided | No change: `term.ConsoleInput` is standard input unchanged; a read out of raw mode goes on waiting after Ctrl+C, and Ctrl+D after it is still the end of input | D19 ("on Windows only") | `pty-ctrlc-ctrld` (P1b, Linux): Ctrl+C at the provider menu, then Ctrl+D at the repeated offer, `warned-once`; plant 70 |
| R119 | The read issued again after an excused read, and D17 | decided | It is a read the person gave up with Ctrl+C, so it is not dropped (D17): what completes it answers the prompt then waiting, the repeated offer | D17; D19 | `menu-ctrlc` (P1b): `n`, Enter after `offer-again` is taken at once; plant 60 |
| R120 | Ctrl+C at a menu read out of raw mode in `gobble setup` or `gobble acp --login`, where no prompt takes the interrupt; (2026-11-20, Q21 and Q22) and at `gobble auth login`'s paste prompt | decided | gobble ends, as an interrupt no prompt takes does (0008-MADR D5); nothing is saved | 0008-MADR D5; D10; R72; the owner, relayed by the coordinator: "Yes, widen the row to include it (Recommended)" | `setup-menu-ctrlc` (P1b): `ended`, `auth.json` empty; (2026-11-20, Q21 and Q22) `auth-paste-ctrlc` (P1b, `internal/cli`, a stubbed browser sign-in): `ended`, `auth.json` empty; plant 67 |
| R121 | Ctrl+C at the first offer, or at a repeated one, on a Windows console | decided | A decline: the lead and the warning once, and the session's prompt goes on (the input is not closed) | D8 ("Ctrl+C at the offer"); D19 | `offer-ctrlc` (P1b): `warned-once`, then a prompt that takes `/exit`; plant 64 |

~~Counts (2026-11-20, after P1 step 0b): 121 rows. R52 and R53 are replaced and not counted. Of the other 119: 100 `decided` (R40 among
them), 16 `out of scope`, 3 `open` (R111 and R114 as Q21, R116 as Q22); none awaits live evidence.~~

*Row added (2026-11-20, Q21 and Q22), Q22's stated consequence:*

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R122 | A real end of input on a Windows console with no mark (Ctrl+Z; Ctrl+Z, Enter) | decided | It ends the input up to 100 ms late: the read waits that long for a mark that does not come | D20; Q22, the owner, relayed by the coordinator: "Wait up to 100 ms for the mark; the probe reports the order (Recommended)"; D14 | stub `TestConsoleWaitBound` (A39, P2): the end of input after at least 100 ms and well under 1 s; plant 74 |

Counts (2026-11-20, Q21 and Q22): 122 rows. R52 and R53 are replaced and not counted. Of the other 120: 104 `decided`, 16 `out of
scope`, none `open`, none awaiting live evidence.

*Rows changed and added (2026-11-21, D21: the owner's answers to F1 and F2 of the PLAN's P2, build pass 1). Each
changed row keeps its first text above, annotated; the row here is its latest.*

**Latest rows: what completes an abandoned read on POSIX, and the session prompt's flush (D21)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R86 | A completion that is not a line: a read started in raw mode by a prompt that gave up, or a POSIX read started out of raw mode that completes after raw mode is on; and a read the user gave up with Ctrl+C rather than the program | decided (2026-11-21, latest) | Whatever completes a read the program gave up is dropped, line or keys, taken whole with the read's own size (D18); a read the person gave up with Ctrl+C is not dropped and goes to the next prompt, as D4 says. On POSIX the completion of a read started out of raw mode by such a prompt is, when the session prompt puts the terminal in raw mode, what was typed into that prompt and not ended (measured on Linux; macOS inferred from the BSD terminal driver); when nothing was typed there, the first key typed after raw mode is on (inferred) | D17; Q16, "Only reads the program gave up (Recommended)"; D21; F1, the owner, relayed by the coordinator: "Amend R86/R102 to what Linux does (Recommended)" | `terminput_test` (A28, as extended): a program give-up dropped, keys included, whole; a Ctrl+C give-up kept; `TestLoginInterrupt` (existing); `pty-abandoned` (A21, as amended 2026-11-21); plants 47, 60, 63 (2026-11-21: D22, proposed, restates this row; see its D22 row, after R144) |
| R102 | The first key or line typed after a browser sign-in whose loopback came first | decided (2026-11-21, latest) | On POSIX, what was typed into the paste prompt is dropped as the session prompt's raw mode goes on, and nothing typed at the session prompt is lost (measured on Linux at P2's build pass 1; macOS inferred from the BSD terminal driver); when nothing was typed into the paste prompt, the first key typed at the session prompt completes the read and is dropped with it (inferred, not measured). On a Windows console, the first line typed (inferred: the console keeps that read in line mode, R85; `abandoned` reads `dropped`). The prompt then reads again | D17; Q16, "Only reads the program gave up (Recommended)"; D21; F1, the owner, relayed by the coordinator: "Amend R86/R102 to what Linux does (Recommended)" | `pty-abandoned` (A21, as amended 2026-11-21: no turn holds `partial`; a turn `again`), `abandoned` (A21), plant 47; (2026-11-21, P2 build pass 2, the coordinator's records gate) the branch with nothing typed into the paste prompt: `pty-abandoned-empty` (A40), `first-key-dropped`, plant 47 (2026-11-21: D22, proposed, restates this row; see its D22 row, after R144) |
| R103 | A sign-in inside the session: `/login`, or one another command starts through the same round (`ls.signInRound`), whatever its outcome (signed in, cancelled, failed) | decided (2026-11-21, latest) | Sets the mark. The next `ReadPrompt` flushes at once, at that prompt and before it reads, whether or not the line editor then reads the terminal, and clears the mark there (R123). A `/login` refused before any question (an unknown provider) is no sign-in and sets none | D17; Q20, "Only the first prompt after a sign-in or the first run (Recommended)"; D4; D21; F2, the owner, relayed by the coordinator: "Flush at once at that first prompt (Recommended)" | `TestFlushPoints` (A24, as extended and as amended 2026-11-21); plants 57, 75 |
| R105 | The prompts after that first one: after a turn, a local command, `/edit` | decided (2026-11-21, latest) | The mark is cleared by the flush made at the first prompt (R103), whether or not that prompt read the terminal; later `ReadPrompt`s do not flush, so keys typed while a turn runs are the next prompt's text, as today, the turn of keys the editor held included (R123) | D17; Q20, "Only the first prompt"; D21; F2, the owner, relayed by the coordinator: "Flush at once at that first prompt (Recommended)" | `TestFlushPoints` (A24, as extended and as amended 2026-11-21); plants 56, 75 |
| R108 | The process ends, or the session ends, with the mark set | decided (2026-11-21, latest) | Nothing: the mark lives in the session's `terminalInput` and nothing is flushed on the way out. Since the flush is made as the first prompt after a sign-in or the first run starts, the mark can be set at the end only when the run ends before that prompt (the process's context ending during the `/models` turn after `/login`) | D17; Q20 (a flush belongs to a prompt read); D21 | `TestFlushPoints` (A24, as amended 2026-11-21): a session ended by Ctrl+D at the first prompt after a sign-in counts that prompt's one flush and none after; the run's context ended during the `/models` turn after `/login` counts none; plant 61, as amended |

**Added (2026-11-21, D21): keys the line editor holds across a sign-in**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R123 | Keys the line editor decoded with `/login`'s Enter and holds across the sign-in (R56), which the first prompt after the sign-in submits without a read of the terminal | decided | That prompt flushes at once, before it reads, and clears the mark (R103); the held keys are its submission, as R56 says; keys typed while their turn runs are the next prompt's text (R105) | D21; F2, the owner, relayed by the coordinator: "Flush at once at that first prompt (Recommended)"; R56; R103; R105 | `TestFlushPoints` (A24, as amended 2026-11-21): `/login gemini` and `1`, Enter in one press, the key prompt answered with Enter: one flush, at the prompt that submits `1`, before its turn; keys typed while that turn runs reach the next prompt; plant 75 (2026-11-21: D22, proposed, restates this row; see its D22 row, after R144) |

Counts (2026-11-21, D21): 123 rows. R52 and R53 are replaced and not counted. Of the other 121: 105 `decided`, 16 `out of
scope`, none `open`, none awaiting live evidence. R86, R102, R103, R105 and R108 have a latest row above; their first
rows are kept, annotated.

*Rows added (2026-11-21, D22, **proposed**: drafted for review on the owner's answer "Make reads cancellable on POSIX"). "Decided" here
is decided by that answer, an earlier decision or a measurement named in the row; the mechanism follows D22 as
drafted. M1 to M8 are the measurements on this Mac in the PLAN's execution record, "Records draft for D22".*

**A POSIX read that is never left waiting (D22)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R124 | Where a POSIX terminal's reads are made | ~~decided~~ decided (D22, proposed) | A layer under `inline.NewReader`, beside `term.ConsoleInput`, used by `terminalIn()` when standard input is a terminal on a `unix` system | the owner, relayed by the coordinator: "Make reads cancellable on POSIX"; D22 | `TestTerminalInChoosesLayer` (A46) (2026-11-21, D22 second draft: see its latest row, after R152) |
| R125 | How the layer waits for the terminal | ~~decided~~ decided (D22, proposed) | select(2) on standard input and a close-on-exec wake pipe, no timeout, `EINTR` retried. Not poll(2) (macOS: `POLLNVAL` on `/dev/tty`, M8); not the runtime poller (needs `O_NONBLOCK` on the terminal's shared open file description, M4) | D22; M1, M2, M4, M8 | `TestCancelableWait` (A41); step 0 re-measures (2026-11-21, D22 second draft: see its latest row, after R152) |
| R126 | A read(2) of the terminal | ~~decided~~ open (Q25) | Made only with the gate open, after select(2) reports the terminal ready, under the gate's lock, so it never waits in the kernel | D22 | `TestCancelableWait` (A41); plant 76 (2026-11-21, D22 second draft: see its latest row, after R152) |
| R127 | The gate | ~~decided~~ decided (D22, proposed) | Opened by `terminalInput` when a prompt's read of the input waits; closed when that read returns. Closing wakes the wait at once, waits for a read(2) in progress, and reports whether one completed since the gate opened | D22 | `TestCancelableWait` (A41); plants 77, 78, 84 (2026-11-21, D22 second draft: see its latest row, after R152) |
| R128 | A prompt gives up, by the program or by the person, before anything was read | decided | Nothing is read; what was typed stays in the terminal and is read by the next prompt that waits, in that prompt's mode; nothing is dropped | the owner's answer; D22 | `TestCancelableGiveUp` (A42); `pty-abandoned-empty-typed`, `pty-abandoned-empty-pasted` (A40, as split); plant 80 (2026-11-21, D22 third draft: see its third-draft latest row, after R152's second-draft rows) |
| R129 | A give-up racing a byte's arrival: a read(2) completed before the gate closed | ~~decided~~ decided (D22, proposed) | That completion was typed while the abandoned prompt was shown. A program's give-up drops it (D13, D17), taken whole (D18); the person's Ctrl+C gives it to the next prompt (Q16) | D13; D17; D18; D22 | `TestCancelableGiveUp` (A42), a stub layer that reports a completed read; plants 81, 82 (2026-11-21, D22 second draft: see its latest row, after R152) |
| R130 | Raw mode switched on or off after a give-up | decided | No read is outstanding, so nothing typed into the abandoned prompt is handed to a waiting read: F4's race cannot happen (M5: half a line left in the kernel is readable once raw mode is on) | the owner's answer; D22; M5 | `pty-abandoned` (A43, as amended), on Linux and on this Mac; plant 76 (2026-11-21, D22 second draft: see its latest row, after R152) |
| R131 | The first prompt after a sign-in or the first run (D21) | ~~decided~~ decided (D22, proposed) | Its flush, after raw mode is on, empties what was typed into the sign-in's prompts and left unread, every time (M5: nothing ready after the flush); what is typed at that prompt after it is read whole | D21; D22; M5 | `pty-abandoned` (A43): no turn holds `partial`; `hello` and `again` are turns; plant 86 (2026-11-21, D22 second draft: see its latest row, after R152) |
| R132 | Keys typed into a prompt the program gave up outside a sign-in (a withdrawn permission request), left unread in the terminal | ~~open (Q23)~~ open (Q23, as rewritten) | Not settled: D13 dropped them by dropping the read; with no read they reach the next prompt unless flushed | — | waits for Q23 (A49) (2026-11-21, D22 second draft: see its latest row, after R152) |
| R133 | Ctrl+C at a prompt read out of raw mode, during the wait | ~~decided~~ decided (D22, proposed) | The signal, as D4 and D8 say; the terminal discards the half line (M6: nothing ready after it); select(2) is not ended by it on macOS (M6: it waited out its 2 s), and `EINTR` from it on Linux is retried; `ls.ask`'s cancel closes the gate | D4; D8; D22; M6 | `TestCancelableWait` (A41, `EINTR`); `pty-menu-ctrlc-typed` (A44); plant 79 (2026-11-21, D22 second draft: see its latest row, after R152) |
| R134 | Ctrl+D on an empty line, read out of raw mode | ~~decided~~ decided (D22, proposed) | select(2) reports it ready and the read gives nothing: the end of input, kept for good (M7; D8) | D8; D22; M7 | `TestCancelableWait` (A41, the end of input); `pty-ctrld-prompt` (A45) (2026-11-21, D22 second draft: see its latest row, after R152) |
| R135 | Ctrl+Z on POSIX (`SIGTSTP`), during the wait | out of scope | As R99: gobble handles no suspend; in raw mode it is a byte (R44) | — | — |
| R136 | The interrupt mark (D19, D20) on POSIX | decided | Not read: the mark is a Windows console's; the layer never consults it | D19 ("on Windows only"); D22 | `TestWatchSignalsMarksInterrupt` (existing, unchanged) (2026-11-21, D22 second draft: see its latest row, after R152) |
| R137 | Standard input a pipe or a file | decided | No layer: read as today; no question is asked off a terminal (R36) | R36; R77; D22 | `TestTerminalInChoosesLayer` (A46); plant 83 (2026-11-21, D22 third draft: see its third-draft latest row, after R152's second-draft rows) |
| R138 | The readiness wait cannot be used: select(2) fails with an error other than `EINTR`, the wake pipe cannot be made, or a descriptor is at or above `FD_SETSIZE` | ~~open (Q24)~~ open (Q24, as updated) | Not settled | — | waits for Q24 (A50) (2026-11-21, D22 second draft: see its latest row, after R152) |
| R139 | A system neither POSIX nor Windows | out of scope | No release target (Linux, macOS, Windows); standard input is read as built, with D13's drop, as `flush_other.go` reports no flush (R78) | — | — |
| R140 | The line editor's timed waits (the Esc timer, a resize, the reprint) | ~~decided~~ decided (D22, proposed) | Each wait that ends closes the gate; the editor's next read opens it; nothing is read in between, nothing is lost, and no read runs across the line session's switches out of raw mode | D13 (R87); D22 | `TestCancelableGiveUp` (A47, a timed wait); the editor's tests (existing) (2026-11-21, D22 second draft: see its latest row, after R152) |
| R141 | The trust question and the permission prompt | ~~decided~~ decided (D22, proposed) | The trust question reads with no deadline (R50); the permission prompt's timed waits open and close the gate as the editor's do; a withdrawn request is R132 | D15; D22 | `TestTrustReadsWithoutDeadline`, `TestPermissionReadsTerminalInput` (existing) (2026-11-21, D22 second draft: see its latest row, after R152) |
| R142 | Two goroutines and the gate | ~~decided~~ decided (D22, proposed) | The gate's state is under its lock; the wake pipe is written by the closer and drained by the waiter; no data race | R48; D22 | `TestCancelableConcurrent` (A48) under `-race` (P4); plant 85 (2026-11-21, D22 second draft: see its latest row, after R152) |
| R143 | Windows | decided | Unchanged: D13, D18, D19 and D20 as built; the layer is not used | the owner, relayed by the coordinator: "Make reads cancellable on POSIX" ("on POSIX"); the stated cost "Windows is unchanged" | the Windows cases (P1b), unchanged (2026-11-21, D23: see its latest row, after the accepted count) |
| R144 | The BSDs and other `unix` systems | decided | The layer applies (`unix` build tag); not measured there: inferred from select(2)'s POSIX semantics | the owner's answer ("POSIX"); D22 | none: no release target; noted (2026-11-21, D22 second draft: see its latest row, after R152) |

**Latest rows: R86, R102 and R123 under D22 (proposed)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R86 | A completion that is not a line: a read started in raw mode by a prompt that gave up, or a POSIX read started out of raw mode that completes after raw mode is on; and a read the user gave up with Ctrl+C rather than the program | ~~decided (2026-11-21, D22, proposed)~~ decided (D22) | On POSIX no read is left waiting, so nothing completes after a give-up; only a completion read before the gate closed (R129) is dropped, for a program's give-up. On Windows, as D13, D17 and D18 say | D17; D21; D22; the owner, relayed by the coordinator: "Make reads cancellable on POSIX" | A42; A43; on Windows, A28 and `abandoned` (2026-11-21, D22 accepted) |
| R102 | The first key or line typed after a browser sign-in whose loopback came first | ~~decided (2026-11-21, D22, proposed)~~ decided (D22) | On POSIX nothing typed at the session prompt is lost, typed or pasted: what was typed into the paste prompt stays unread and is emptied by the first prompt's flush (R131). On a Windows console, as before: the first line (inferred, R85; `abandoned` reads `dropped`) | D21; D22; the owner, relayed by the coordinator: "Make reads cancellable on POSIX" | `pty-abandoned` (A43), `pty-abandoned-empty-typed` and `pty-abandoned-empty-pasted` (A40, as split: `none-lost` each), on Linux and on this Mac; `abandoned` (Windows) (2026-11-21, D22 accepted) |
| R123 | Keys the line editor decoded with `/login`'s Enter and holds across the sign-in (R56) | ~~decided (2026-11-21, D22, proposed)~~ decided (D22) | As before (the prompt flushes at once and clears the mark); with D22, every key typed before that prompt and not read is in the terminal when it flushes, none in a waiting read | D21; D22 | `TestFlushPoints` (A24); plant 75 (2026-11-21, D22 accepted) |

~~Counts (2026-11-21, D22 drafted): 144 rows. R52 and R53 are replaced and not counted. Of the other 142: 122 `decided`~~
~~(the 17 rows D22 adds among them, by the owner's answer and the measurements; D22 itself is proposed), 18 `out of~~
~~scope`, 2 `open` (R132 as Q23, R138 as Q24).~~
*(2026-11-21, D22 third draft: superseded by the third draft's count, recomputed from each row's latest status.)*

*Rows added and changed (2026-11-21, D22 second draft, after the independent record review and M9 to M14; still proposed). The
first draft's rows above are kept, annotated; each row here is its latest. "decided (D22, proposed)" rests on D22's
mechanism, not yet accepted; "inferred" marks what no measurement shows.*

**D22, second draft: latest rows**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R124 | Where a POSIX terminal's reads are made | ~~decided (D22, proposed)~~ decided (D22) | A layer under `inline.NewReader`, beside `term.ConsoleInput`, used by `terminalIn()` when standard input is a terminal on a `unix` system | the owner, relayed by the coordinator: "Make reads cancellable on POSIX"; D22 | `TestTerminalInChoosesLayer` (A46); plant 83 (2026-11-21, D22 accepted) (2026-11-21, D23: see its latest row, after the accepted count) |
| R125 | How the layer waits for the terminal | decided (D22, proposed) | select(2) on the descriptor it reads and the wake pipe, no timeout. `EINTR` is retried as a guard only: none was seen on any host, for Ctrl+C or a signal to the process (M6, M9). Not poll(2): macOS refuses `/dev/tty` (M8; Linux accepts it). Not the runtime poller: it needs `O_NONBLOCK` on standard input's shared open file description (M4). Which descriptor is read is Q25 | D22; M1, M2, M4, M6, M8, M9 on three hosts | `TestCancelableWait` (A41), its select(2) seam giving `EINTR`; plant 79 (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R126 | A read(2) after select(2) reported the terminal ready | open (Q25) | Not settled: on standard input it can still wait (M11: a line reported ready, Ctrl+C, and the read waits for the next line); on a private non-blocking descriptor it gives `EAGAIN` (M12) | — | waits for Q25 (A41, step 19) (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R127 | The gate | decided (D22, proposed) | Opened by `fill` for one read of the input; it closes itself when a read(2) returns, and `fill` closes it when its read returns, whatever the cause. A close wakes the wait, returns only after the waiter has left select(2), and reports whether a read completed since the opening | D22 | `TestCancelableWait` (A41), a seam reporting each return from the wait; `TestCancelableGiveUp` (A42), a read(2) seam; plants 77, 78, 84 (2026-11-21, D22 third draft: see its third-draft latest row, after R152's second-draft rows) |
| R129 | A give-up racing a byte's arrival: a read(2) completed before the gate closed | ~~decided (D22, proposed)~~ decided (D22) | As the first draft: the program's give-up drops it (D13, D17, D18); the person's Ctrl+C gives it to the next prompt (Q16). Staged through the read(2) seam | D13; D17; D18; D22 | `TestCancelableGiveUp` (A42), the read(2) seam holding a read between readiness and its return; plants 78, 81, 82 (2026-11-21, D22 accepted) |
| R130 | Raw mode switched on or off after a give-up | ~~decided~~ decided (D22, proposed) | As the first draft (no read is outstanding: F4's race cannot happen). Pinned by a case that does not depend on timing | the owner's answer; D22; M5 | `pty-gaveup-held` (A43, added): a line read given up with `partial` typed, raw mode on, 200 ms: the terminal still holds the bytes (select ready; `FIONREAD` 7 on Linux) and the layer reports no completed read; on Linux and, by Q26, on this Mac; plant 76 (2026-11-21, D22 third draft: status, item 1) (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R131 | The first prompt after a sign-in or the first run (D21) | decided (D22, proposed) | Its flush empties what was typed into the sign-in's prompts and left unread, on Linux and macOS, every time (M5, M14: out of raw mode too); not on the BSDs, where nothing flushes (R78, R144) | D21; D22; M5; M14 | `pty-abandoned` (A43); plant 86 dropped (M14: a flush out of raw mode empties half a line as well, so it changes nothing) (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R132 | Keys typed into a prompt the program gave up, left unread: a withdrawn permission request; the paste prompt of a sign-in outside the session (`gobble auth login`, `gobble acp --login`, `gobble setup`) | open (Q23, as rewritten) | Not settled | — | waits for Q23 (A49) (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R133 | Ctrl+C at a prompt read out of raw mode, during the wait | ~~decided (D22, proposed)~~ decided (D22) | The signal, as D4 and D8 say; the terminal empties the half line (M6, M11); select(2) is not ended by it on any host (M6); `ls.ask`'s cancel closes the gate. A line already reported ready is Q25 | D4; D8; D22; M6; M11 | `pty-menu-ctrlc-typed` (A44); `TestCancelableWait` (A41, `EINTR` from the seam) (2026-11-21, D22 accepted) |
| R134 | Ctrl+D on an empty line, read out of raw mode | decided (D22, proposed) | The wait reports it ready and the read gives nothing: the layer returns exactly `(0, io.EOF)`, never `(0, nil)`, and the input ends for good (M7; D8) | D8; D22; M7 | `TestCancelableWait` (A41, exactly `(0, io.EOF)`); A45, as moved to a prompt read out of raw mode; plant 87 (2026-11-21, D22 third draft: see its third-draft latest row, after R152's second-draft rows) |
| R136 | The interrupt mark (D19, D20) on POSIX | decided | Not read by the layer | D19 ("on Windows only"); D22 | none: by construction (the layer has no mark) |
| R138 | The readiness wait or its descriptor cannot be used: select(2) fails with an error other than `EINTR`, the wake pipe cannot be made, a descriptor is at or above `FD_SETSIZE` (checked before `FdSet.Set`, which panics past it), or, under Q25 (a), the private descriptor cannot be opened | open (Q24, as updated) | Not settled | — | waits for Q24 (A50) (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R140 | The line editor's timed waits | ~~decided (D22, proposed)~~ decided (D22) | As the first draft | D13 (R87); D22 | `TestCancelableGiveUp` (A47, a timed wait over the layer); plant 84 (2026-11-21, D22 accepted) |
| R141 | The trust question and the permission prompt | ~~decided (D22, proposed)~~ decided (D22) | As the first draft | D15; D22 | `TestPermissionTimedWaitOverLayer` (A51, added): the permission prompt's timed wait over the layer: nothing read, and the key typed after it answers (2026-11-21, D22 accepted) |
| R142 | Two goroutines and the gate | ~~decided (D22, proposed)~~ decided (D22) | As the first draft | R48; D22 | `TestCancelableConcurrent` (A48) under `-race`, in P4 step 5: plant 85 is caught there, not in P2 (2026-11-21, D22 accepted) |
| R144 | The BSDs and other `unix` systems | decided | The layer applies (`unix` build tag), inferred from select(2)'s POSIX semantics, not measured. R131 does not hold there: `flush_other.go` cannot flush (R78), so what was typed into an abandoned prompt reaches the first prompt | the owner's answer ("POSIX"); D22; R78 | none: no release target; noted |

**D22, second draft: added rows**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R145 | The layer's Read while the gate is closed | decided (D22, proposed) | It stays in the call and waits on the wake pipe alone, the terminal not in the set, so type-ahead cannot make it spin; it returns only bytes, `io.EOF` or a read error, never because of a close (an error would end inline.Reader's goroutine for good; `(0, nil)` would make it read again at once) | D22 | `TestCancelableGiveUp` (A42); plant 88 (2026-11-21, D22 third draft: see its third-draft latest row, after R152's second-draft rows) |
| R146 | The wake pipe | decided (D22, proposed) | Both ends non-blocking and close-on-exec; drained to `EAGAIN` before each wait; `EAGAIN` on a close's write is not an error; one pipe for the process; a test's layer ended by `stop()` (wakes the waiter, returns `io.EOF`, closes the pipe), in `t.Cleanup` | D22 | `TestCancelableWakePipe` (A54, added): both ends' flags; a hundred closes with no waiter do not block; `stop()` ends a waiting Read with `io.EOF`; plant 91 (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R147 | The size `fill` asks `inline.Reader` for | decided (D22, proposed) | Always `lineBuf`; what the caller's buffer cannot take stays in `pending`, where `discard()` and the rules of D3 and D4 reach it, not in `inline.Reader`'s own store | D22; D4 | `TestCancelableGiveUp` (A42): a parked 512-byte ask, a 256-byte read, `discard()`, and only bytes typed after it arrive; plant 89 (2026-11-21, D22 third draft: see its third-draft latest row, after R152's second-draft rows) |
| R148 | The terminal hangs up (its master closed) during the wait | ~~decided~~ decided (D22, proposed) | select(2) reports it ready and the read gives 0: `io.EOF`; a read of standard input already waiting gives `EIO` on Linux, the input's error (0 on macOS); either way the input ends for good and D8 applies; nothing spins (M10, three hosts) | D8; R43; M10 | `TestCancelableWait` (A41, the read seam giving 0 and `EIO`); A52 (added) (2026-11-21, D22 third draft: status, item 1; see its third-draft latest row) |
| R149 | gobble started in the background of a job-control shell | out of scope | As R99 and R135: a background read is the shell's job control, not a prompt's | — | — |
| R150 | Other readers of the same standard input: acpserver after `--login` (`acp.go:50`), `readWorkspace` (`acp.go:145`), `ReadPassword` (`credentials.go:213`), and the children handed it (`slash.go:228`, `config.go:287`; R95) | decided (D22, proposed) | With the gate closed no read of gobble's waits, so each gets every byte typed while it reads; under Q24's fallback, R95's risk returns | D22; R95 | `TestCancelableGiveUp` (A42): no read(2) through the seam between a prompt's return and the next opening (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R151 | No gate: a Windows console, a pipe, a test's input (the harness, the A28 stubs) | decided | `fill` sets `waiting` on every read that ended on its context, as built, so D13's drop holds there; with a gate, only when the close reported a completed read | D13; D18; R143 | A28 (existing); `TestCancelableGiveUp` (A42, a gateless case); plant 90 |
| R152 | The pseudo-terminal cases on this Mac | open (Q26) | Not settled: the cases and the Linux opener share one `linux && pty` file | — | waits for Q26 (step 20) (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |

**D22, second draft: rows D22 changes on POSIX (review 2.2)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R83 | A flush while a read of a prompt that gave up is still waiting | ~~decided (Windows); D22, proposed (POSIX)~~ ~~decided (D22, proposed)~~ decided (D22) | On POSIX no such read waits (R128); the flush empties what is in the terminal. On Windows, as before | D12; D13; D22 | Windows: `abandoned`; POSIX: `pty-gaveup-held` (A43) (2026-11-21, D22 third draft: status, item 1) (2026-11-21, D22 accepted) |
| R84 | A read started out of raw mode by a prompt that gave up, completed by a line, on POSIX | decided (D22, proposed) | On POSIX it is not read at all: the line stays in the terminal, emptied by the first prompt's flush after a sign-in (R131), or as Q23 decides; a completion in the race of R129 is dropped | D13; D22 | A42; A43 (2026-11-21, D22 third draft: counted under D22, proposed) (2026-11-21, D22 accepted: see its accepted latest row, after the third draft's count) |
| R88 | A second prompt gives up while the first one's read still waits | ~~decided (Windows); D22, proposed (POSIX)~~ ~~decided (D22, proposed)~~ decided (D22) | On POSIX no read waits; at most one race completion (R129), dropped once. On Windows, as before | D13; D22 | A28 (Windows, gateless); A42 (2026-11-21, D22 third draft: status, item 1) (2026-11-21, D22 accepted) |
| R89 | The input ends while a prompt that gave up has its read waiting | decided (Windows); D22, proposed (POSIX) | On POSIX nothing reads, so a Ctrl+D typed out of raw mode stays queued: once raw mode is on it is one byte, `0x04` on macOS and `0x00` on Linux (M13), not the end of input; after a sign-in the first prompt's flush empties it and the session goes on; without a flush (Q23 (c)), the next raw prompt reads the byte, which at gobble's empty prompt on macOS is Ctrl+D | D8; D13; D22; M13 | A42 (a stub layer); `pty-gaveup-held` (A43) with Ctrl+D (2026-11-21, D22 third draft: see its third-draft latest row, after R152's second-draft rows) |
| R95 | The external editor of `/edit`, which inherits standard input | ~~decided (D22, proposed, POSIX); out of scope elsewhere~~ decided (D22) | On POSIX no read of gobble's waits while the gate is closed, so the editor gets every key; on Windows and under Q24's fallback, as R95 says | Q10; D22 | R150's case (A42) (2026-11-21, D22 accepted) |

~~Counts (2026-11-21, D22 second draft): 152 rows. R52 and R53 are replaced and not counted. Of the other 150: 113 `decided`, 14~~
~~`decided (D22, proposed)` (R124, R125, R127, R129, R131, R133, R134, R140 to R142, R145 to R147, R150), 4 `open`~~
~~(R126 as Q25, R132 as Q23, R138 as Q24, R152 as Q26), 19 `out of scope`. Of the 113, eight earlier rows rest on D22~~
~~for their POSIX part, each with a latest row: R83, R84, R86, R88, R89, R95, R102, R123.~~
*(2026-11-21, D22 third draft: superseded by the third draft's count, recomputed from each row's latest status.)*

*Rows changed (2026-11-21, D22 third draft). Each earlier row is kept, annotated; each row here is its latest.*

**D22, third draft: latest rows**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R89 | The input ends while a prompt that gave up has its read waiting; on POSIX, a Ctrl+D typed into a prompt the program gave up | ~~decided (D22, proposed)~~ decided (D22) | On POSIX nothing reads, so the Ctrl+D stays queued. Read next out of raw mode, it is the end of input (D8). Read next in raw mode, it is one byte, `0x04` on macOS or `0x00` on Linux (M13): at `readSecret`, `0x04` cancels and `0x00` is ignored; at the line editor, `0x00` is ignored and `0x04` at an empty prompt ends the session. R131's flush after a sign-in, or Q23's answer, empties it; the `gobble acp --login` route, where acpserver would read the end of input, is Q23's. On Windows, as before | D8; D13; D22; M13 | `TestCancelableGiveUp` (A42): a stub layer giving `0x04`, then `0x00`, to `readSecret` (cancelled; ignored); `pty-gaveup-held` (A43) with Ctrl+D (2026-11-21, D22 accepted) |
| R127 | The gate | ~~decided (D22, proposed)~~ decided (D22) | Opened by `fill` for one read of the input; it stays open until `fill` closes it when its read returns, whatever the cause ~~; it closes itself when a read(2) returns~~ (2026-11-21, D22 third draft: the self-close is dropped: one inner read per ask, so nothing a test can see). A close wakes the wait, returns only after the waiter has left select(2), and reports whether a read completed since the opening. Under Q25 (a), `EAGAIN` from the private descriptor returns to the wait | D22 | `TestCancelableWait` (A41), the wait's seam; `TestCancelableGiveUp` (A42), the read(2) seam; plants 77, 78, 84 (2026-11-21, D22 accepted) |
| R128 | A prompt gives up, by the program or by the person, before anything was read | ~~decided (D22, proposed)~~ decided (D22) | As the first draft: nothing is read; what was typed stays in the terminal for the next prompt that waits, in its mode; nothing is dropped | the owner's answer; D22 | as the first draft; plant 80 (2026-11-21, D22 accepted) |
| R137 | Standard input a pipe or a file | ~~decided (D22, proposed)~~ decided (D22) | No layer: read as today; no question is asked off a terminal (R36) | R36; R77; D22 | `TestTerminalInChoosesLayer` (A46); plant 83 (2026-11-21, D22 accepted) |
| R145 | The layer's Read while the gate is closed | ~~decided (D22, proposed)~~ decided (D22) | As the second draft: it waits on the wake pipe alone, so type-ahead cannot make it spin, and returns only bytes, `io.EOF` or a read error | D22 | `TestCancelableGiveUp` (A42), and a case added: the gate closed with bytes ready, the select seam sees no return of the wait over 200 ms; plants 88, 92 (2026-11-21, D22 accepted) |
| R147 | The size `fill` asks `inline.Reader` for | ~~decided (D22, proposed)~~ decided (D22) | With a gate (a POSIX terminal), `lineBuf` always, what the caller cannot take kept in `pending`. With no gate (Windows, a pipe, a test's input), the caller's buffer, as built, so R143 holds and plant 63 keeps A28's 512-then-256 row | D22; D4; R143 | `TestCancelableGiveUp` (A42, the gated 512-then-256 case), plant 89; A28 (no gate), plant 63 (2026-11-21, D22 accepted) (2026-11-21, D23: see its latest row, after the accepted count) |

**D22, third draft: rows resting on M15**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R134 | Ctrl+D on an empty line, read out of raw mode | ~~decided (D22, proposed)~~ decided (D22) | As the second draft, and measured through the private descriptor of Q25 (a): select(2) reports it ready, the read gives 0 once, the end of input, and `EAGAIN` after (M15, three hosts); the layer returns exactly `(0, io.EOF)` and the input ends for good | D8; D22; M7; M15 | A41; A45 (both cases); plant 87 (2026-11-21, D22 accepted) |
| R148 | The terminal hangs up during the wait | ~~decided (D22, proposed)~~ decided (D22) | As the second draft, and measured through the private descriptor and through `/dev/tty`: select(2) reports it ready and every read gives 0, the end of input (M15, three hosts); `EIO` was seen only in a read of standard input already waiting on Linux (M10) | D8; R43; M10; M15 | A41; A52 (2026-11-21, D22 accepted) (2026-11-21, D23: see its latest row, after the accepted count) |

~~Counts (2026-11-21, D22 third draft, recomputed from each row's latest status): 152 rows. 2 are replaced and not counted (R52, R53). Of the other 150: 102 `decided`; 26 `decided (D22, proposed)` (R83, R84, R86, R88, R89, R95, R102, R123, R124, R125, R127, R128, R129, R130, R131, R133, R134, R137, R140, R141, R142, R145, R146, R147, R148, R150); 4 `open` (R126, R132, R138, R152); 18 `out of scope`. These four, with `replaced`, are every status the table uses.~~ *(2026-11-21, D22 accepted: superseded by the count below.)*

*Rows changed (2026-11-21, D22 accepted: the owner's answers to Q23 to Q26). Each earlier row is kept, annotated; each row
here is its latest. Rows that read "decided (D22, proposed)" and need no new text read "decided (D22)" where they
stand.*

**D22 accepted: latest rows**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R80 | Which points flush | decided (D22) | Three: with `discard()` before the session prompt reads (the first after a sign-in or the first run, D17); after a repeated offer is written, before its read; and, on POSIX, at the program's give-up itself: a withdrawn permission request (`askPermission`) and a question whose context the program ended (`questions.cancelled`), inside the session and out of it | D12; D17; D22; Q23, the owner, relayed by the coordinator: "Flush at gobble's give-up (Recommended)" | `TestFlushPoints` (A24); `TestFlushAtGiveUp` (A49); plants 42 to 44, 94, 95 (2026-11-21, D23: see its latest row, after the accepted count) |
| R81 | Points that do not flush | decided (D22) | As R81 says, except that on POSIX a program's give-up flushes (R80): at a withdrawn permission request, and at a paste prompt a browser sign-in's loopback ended, in the session and in `gobble setup`, `gobble acp --login` and `gobble auth login`. A person's Ctrl+C give-up does not flush | D12; D22; Q23, the owner, relayed by the coordinator: "Flush at gobble's give-up (Recommended)" | `TestFlushPoints` (A24); `TestFlushAtGiveUp` (A49); plant 95 |
| R84 | A read started out of raw mode by a prompt that gave up, completed by a line, on POSIX | decided (D22) | It is not read: what was typed into that prompt is flushed at the give-up itself (R80); keys typed in the instant before it are lost | D13; D22; Q23, the owner, relayed by the coordinator: "Flush at gobble's give-up (Recommended)" | `TestFlushAtGiveUp` (A49); A42; A43; plant 94 |
| R125 | How the layer waits for the terminal | decided (D22) | select(2) on the private descriptor (R126) and the wake pipe, no timeout; `EINTR` retried as a guard only (no `EINTR` seen, M6, M9). Not poll(2) (macOS refuses `/dev/tty`, M8); not the runtime poller on standard input (its shared open file description, M4) | D22; Q25, the owner, relayed by the coordinator: "Private non-blocking descriptor (Recommended)"; M1, M2, M4, M6, M8, M9, M12, M15 | `TestCancelableWait` (A41); plant 79 |
| R126 | A read(2) after select(2) reported the terminal ready | decided (D22) | Made on a private descriptor on the terminal's own path, opened `O_RDONLY\|O_NONBLOCK\|O_CLOEXEC\|O_NOCTTY` (or `/dev/tty` with the same flags, when the path cannot be opened and standard input is the controlling terminal, by the same foreground process group): `EAGAIN` returns to the wait, so the read never waits, even after Ctrl+C has emptied a line reported ready (M11, M12, M15); raw mode and the flush stay on standard input and apply to it (M12). Both opens failing is R138; `su` or `sudo -u` (`EACCES`, inferred, not measured) reaches `/dev/tty` or R138 | D22; Q25, the owner, relayed by the coordinator: "Private non-blocking descriptor (Recommended)"; M11, M12, M15 | `TestCancelableWait` (A41, the read seam giving `EAGAIN`); `TestPrivateDescriptor` (A55, added); plant 96 (2026-11-21, D23: see its latest row, after the accepted count) |
| R130 | Raw mode switched on or off after a give-up | decided (D22) | As the second draft (no read is outstanding: F4's race cannot happen), pinned on Linux and on this Mac by the shared pseudo-terminal cases | the owner's answer "Make reads cancellable on POSIX"; D22; M5; Q26, the owner, relayed by the coordinator: "Move them to a shared file (Recommended)" | `pty-gaveup-held` (A43), in `pty_unix_test.go`, on Linux and on this Mac; plant 76 |
| R131 | The first prompt after a sign-in or the first run (D21) | decided (D22) | As the second draft: its flush empties what was typed into the sign-in's prompts and left unread, on Linux and macOS, every time; not on the BSDs (R78, R144). Pinned on both systems by the shared cases | D21; D22; M5; M14; Q26, the owner, relayed by the coordinator: "Move them to a shared file (Recommended)" | `pty-abandoned` (A43), in `pty_unix_test.go`, on Linux and on this Mac |
| R132 | Keys typed into a prompt the program gave up, left unread: a withdrawn permission request; the paste prompt of a sign-in outside the session | decided (D22) | On POSIX they are flushed at the give-up itself (R80): never a turn, never an answer, never the shell's or acpserver's input; keys typed in the instant before the give-up are lost. A person's Ctrl+C give-up does not flush (Q16). Windows keeps D13's drop | D13; D22; Q23, the owner, relayed by the coordinator: "Flush at gobble's give-up (Recommended)" | `TestFlushAtGiveUp` (A49): a withdrawn permission request, and `gobble auth login`'s paste prompt ended by a stub loopback, each flush once; a Ctrl+C give-up does not; plants 94, 95 (2026-11-21, D23: see its latest row, after the accepted count) |
| R138 | The readiness wait or its descriptor cannot be used: select(2) fails other than with `EINTR`, the wake pipe cannot be made, a descriptor is at or above `FD_SETSIZE`, or neither the private descriptor nor `/dev/tty` can be opened | decided (D22) | For the rest of the process, standard input is read as built (a blocking read, D13's drop), and one warning is written on stderr, once, in the owner's approved words: `Warning: gobble cannot wait on this terminal without reading it (<error>); keys typed while a sign-in finishes may be lost.` | D22; Q24, the owner, relayed by the coordinator: "Old reads plus one warning (Recommended)"; Q25 | `TestCancelableFallbackWarnsOnce` (A50): each cause through the seams; the words asserted exactly; one warning for two causes; the prompts still read; plants 93, 97 (2026-11-21, D23: see its latest row, after the accepted count) |
| R146 | The wake pipe, and the private descriptor | decided (D22) | The wake pipe as the second draft says. The private descriptor (R126) is opened once, with the pipe, for the process, close-on-exec, and lives as long; a test's layer closes both in `stop()` | D22; Q25, the owner, relayed by the coordinator: "Private non-blocking descriptor (Recommended)" | `TestCancelableWakePipe` (A54); `TestPrivateDescriptor` (A55); plant 91 (2026-11-21, D23: see its latest row, after the accepted count) |
| R150 | Other readers of the same standard input | decided (D22) | With the gate closed no read of gobble's waits, so each gets every byte typed while it reads. Under R138's fallback, R95's risk returns, which the warning names | D22; R95; Q24, the owner, relayed by the coordinator: "Old reads plus one warning (Recommended)" | `TestCancelableGiveUp` (A42); `TestCancelableFallbackWarnsOnce` (A50) |
| R152 | The pseudo-terminal cases on this Mac | decided (D22) | The cases move word for word to `internal/cli/pty_unix_test.go` (`(linux \|\| darwin) && pty`); the Linux opener stays in `pty_linux_test.go`, the macOS opener (`/dev/ptmx`, `TIOCPTYGRANT`, `TIOCPTYUNLK`, `TIOCPTYGNAME`) is in a new `pty_darwin_test.go`; `make probe-pty` runs on either system, outside `preflight` and CI, and the Makefile's comment says so | Q26, the owner, relayed by the coordinator: "Move them to a shared file (Recommended)", the owner's word for the move beyond D21's one line | `make probe-pty` on Linux and on this Mac (step 20); the cases' words as on Linux |

~~Counts (2026-11-21, D22 accepted, recomputed from each row's latest status): 152 rows. 2 are replaced and not counted (R52, R53). Of the other 150: 100 `decided`; 32 `decided (D22)` (R80, R81, R83, R84, R86, R88, R89, R95, R102, R123, R124, R125, R126, R127, R128, R129, R130, R131, R132, R133, R134, R137, R138, R140, R141, R142, R145, R146, R147, R148, R150, R152); 18 `out of scope`; none `open`. These, with `replaced`, are every status the table uses.~~ *(2026-11-21, D23: superseded by the count below.)*

*Rows changed and added (2026-11-21, D23: the owner's rulings after build pass 3's review). Each earlier row is
kept, annotated; each row here is its latest.*

**D23: latest rows**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R80 | Which points flush | decided (D22) | As D22's latest row, with the give-up flush keyed on a POSIX terminal: with `discard()` before the session prompt reads (the first after a sign-in or the first run); after a repeated offer is written, before its read; and, on a Linux or macOS terminal, gated or fallen back to reading as built, at the program's give-up itself (`askPermission`'s withdrawn branch, `questions.cancelled`'s program branch), inside the session and out of it | D12; D17; D22; Q23; D23, the owner, relayed by the coordinator: "Always flush on a POSIX terminal (Recommended)" | `TestFlushPoints` (A24); `TestFlushAtGiveUp` (A49); `TestGiveUpFlushBySystem` (A56); plants 94, 95, 98, 107 |
| R124 | Where a POSIX terminal's reads are made | decided (D22) | As D22's latest row; pinned in the default suite, not only under `pty` | D22 | `TestTerminalInChoosesLayer` (A46); `TestTerminalInWiresGate` (A58, added); plants 83, 100 |
| R126 | A read(2) after select(2) reported the terminal ready | decided (D22) | As D22's latest row. `/dev/tty` is opened only when standard input is the controlling terminal, by the same foreground process group: a pseudo terminal that is not the controlling terminal, its path failing, gives R138, not `/dev/tty` | D22; Q25; M15 | `TestCancelableWait` (A41); `TestPrivateDescriptor` (A55); `pty-not-controlling` (A59, added); plants 96, 101 |
| R132 | Keys typed into a prompt the program gave up, left unread | decided (D22) | On a Linux or macOS terminal, gated or fallen back, they are flushed at the give-up itself (R80): never a turn, never an answer, never the shell's or acpserver's input; keys typed in the instant before the give-up are lost. A person's Ctrl+C give-up does not flush (Q16). Windows keeps D13's drop and does not flush (R143) | D13; D22; Q23; D23, the owner, relayed by the coordinator: "Always flush on a POSIX terminal (Recommended)" | `TestFlushAtGiveUp` (A49); `TestGiveUpFlushBySystem` (A56): a gateless input that is not a POSIX terminal counts no flush; a start-up fallback and a later fallback each flush once; plants 94, 95, 98, 107 |
| R138 | The readiness wait or its descriptor cannot be used: select(2) fails other than with `EINTR`, the wake pipe cannot be made, a descriptor is at or above `FD_SETSIZE`, or neither the private descriptor nor `/dev/tty` can be opened (a blocking descriptor, R154, or a different terminal, R155, counting as not opened) | decided (D22) | As D22's latest row: standard input read as built for the rest of the process, with the warning once, now written on the main flow (R156); on a POSIX terminal the give-up flush still applies (R80) | D22; Q24; D23, the owner, relayed by the coordinator: "Always flush on a POSIX terminal (Recommended)"; "Refuse a blocking descriptor (Recommended)"; "Check it's the same terminal"; "Write the warning safely" | `TestCancelableFallbackWarnsOnce` (A50); `TestGatedRefusals` (A57, added): the wake pipe failing and a descriptor at or above `FD_SETSIZE`, through seams; plants 93, 97, 99 |
| R143 | Windows | decided (D22) | Unchanged: D13, D18, D19 and D20 as built; the layer is not used; no flush at a give-up | the owner's answer "Make reads cancellable on POSIX"; D23 | the Windows cases (P1b), unchanged; `TestGiveUpFlushBySystem` (A56, the gateless input that is not a POSIX terminal); plant 98 |
| R146 | The wake pipe, and the private descriptor | decided (D22) | As D22's latest row. A pipe not drained before a wait is caught by the closed gate's case, not by A54 | D22; Q25 | `TestCancelableWakePipe` (A54: the ends' flags, closes with no waiter); `TestCancelableGiveUp` (A42, the closed gate's case: the pipe drained); plant 91 |
| R147 | The size `fill` asks `inline.Reader` for, and the size of a race completion's drop | decided (D22) | As D22's latest row; and on the gated path a read that ended on its context is marked with the size asked, `lineBuf`, not the caller's: a race completion abandoned by a 512-byte ask and then met by `askOn`'s 256-byte read is dropped whole (D18) | D18; D22; R143 | `TestCancelableGiveUp` (A42); `TestGatedRaceDropSize` (A61, added); plants 89, 102 |
| R148 | The terminal hangs up during the wait | decided (D22) | As D22's latest row; through the terminal input, `EIO` from the read closes the input: `ended()` is true | D8; R43; M10; M15 | `TestCancelableHangUp` (A52), with a case through `terminalInput`; A41 |

**D23: added rows**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R153 | `EINTR` from read(2) on the private descriptor | decided (D22) | Returns to the wait, as `EAGAIN` does; never an error and never the end of input | D23, the owner, relayed by the coordinator: "Retry EINTR on the read (Recommended)" | `TestCancelableWait` (A62, added: the read seam gives `EINTR` once, then the bytes); plant 103 |
| R154 | A descriptor without `O_NONBLOCK` given to the layer | decided (D22) | `NewGated` reads `F_GETFL` and refuses it with an error; `OpenGated` takes that as a failed open: `/dev/tty`, then R138 | D23, the owner, relayed by the coordinator: "Refuse a blocking descriptor (Recommended)" | `TestGatedRefusals` (A63, added: a blocking pipe refused); plant 104 |
| R155 | The terminal's path opens a different terminal than standard input's | decided (D22) | `OpenGated` compares `fstat`'s `Rdev` of standard input and of the opened descriptor; on a mismatch it closes it and goes on to `/dev/tty`, which keeps its foreground-process-group check (M15), then R138 | D23, the owner, relayed by the coordinator: "Check it's the same terminal"; M15 | `pty-same-terminal` (A64, added): a child whose controlling terminal is one pseudo terminal, given the path of another: what is typed on its own is read; plant 105 |
| R156 | Q24's warning, when the fallback happens on `inline.Reader`'s goroutine | decided (D22) | It is queued, never written there, and written once on the main flow: before the next prompt's or question's text, with the terminal in its cooked state, or at exit if no such text comes; the words unchanged | D23, the owner, relayed by the coordinator: "Write the warning safely" | `TestFallbackWarningOnMainFlow` (A60, added): no warning while the failing read runs; once before the next question's text; once at the run's end when no text comes; plant 106 (2026-11-21, build pass 5, the owner, relayed by the coordinator: "Fifth pass, this fix only (Recommended)": at the session prompt the warning is written out of raw mode, through `outOfRaw`, before the editor's prompt; `TestFallbackWarningOnMainFlow` gains the case at the session prompt, with raw mode stubbed: the terminal is not raw when the warning is written; and the case written once: a fallback, then two later questions, the warning counted once; plants: the warning written while raw, and the queue not cleared) |
| R157 | The give-up flush, by kind of input | decided (D22) | On a Linux or macOS terminal it flushes whatever reads it: the gate, a start-up fallback, a later fallback. A pipe or a file has nothing to flush; Windows does not flush (R143) | D23, the owner, relayed by the coordinator: "Always flush on a POSIX terminal (Recommended)" | `TestGiveUpFlushBySystem` (A56); plants 98, 107 |

Counts (2026-11-21, D23, recomputed from each row's latest status): 157 rows. 2 are replaced and not counted (R52, R53). Of the other 155: 99 `decided`; 38 `decided (D22)` (R80, R81, R83, R84, R86, R88, R89, R95, R102, R123, R124, R125, R126, R127, R128, R129, R130, R131, R132, R133, R134, R137, R138, R140, R141, R142, R143, R145, R146, R147, R148, R150, R152, R153, R154, R155, R156, R157); 18 `out of scope`; none `open`. These, with `replaced`, are every status the table uses.

## Pros and Cons of the Options

### A. The secret skips one `\n` after a `\r`-ended answer

* Good, because it is about six lines in one function, proven red then green on the repository's own harness and on a
  pseudo-terminal with `ICRNL` off (`red.out`, `green.out`, `pty-*.out`).
* Good, because it touches nothing the line editor reads, so it carries the least risk.
* Bad, because it leaves weakness (a): the first run's offer and its menu are still two asks with two `pending`s.
* Bad, because the rule lives in two readers that must agree (`lineContext` and `readSecret`), and the editor, the
  permission prompt and the trust question each still read the terminal their own way.
* Bad, because it keeps weakness (d): a raw secret's `\r` still owes a `\n` to the next line question.

### B. One shared terminal input per process

* Good, because one value holds the read-ahead and the line-end rule for every prompt, so a rule is written and tested
  once.
* Good, because it fixes weakness (a) and, with D3's distinction between raw and cooked line ends, weakness (d).
* Good, because it makes 0027-MADR D5's "one reader owns stdin" a structure, not a convention four call sites keep.
* Neutral, because it keeps 0025-PLAN deviation 7: secrets in raw mode, the other questions out of it.
* Bad, because it is a moderate change across seven files and the editor's constructor, and its review must cover the
  editor's read path.
* Bad, because its scope and its hand-back to the editor were choices the owner's first words did not settle; the
  owner settled them on 2026-11-19 (Q2, Q3).

### C. Every question read in raw mode

* Good, because with one read mode Enter is `\r` on every system, and a cooked line end never reaches a question.
* Bad, because gobble must then echo and edit visible answers itself, a line editor for each question, and the
  owner chose cooked menus and a raw secret in 0025-PLAN deviation 7.
* Bad, because Ctrl+C in a question becomes a byte rather than the signal `ls.ask` routes today.
* Bad, because a Windows console read already waiting in cooked mode keeps that mode (0027-MADR D5), so a raw-only
  design still has to handle a cooked line it did not start.

### D. Let an empty secret through to the SDK

* Good, because it is a one-line change.
* Bad, because it is a workaround: the stray line end still answers the key prompt, and the SDK then lists or refuses
  an empty key, so the user still is not asked. Rejected.

## Owner questions

Each was put with three options and a recommendation. The owner answered Q1 to Q8 on 2026-11-19, relayed by the
coordinator; each answer is quoted as the option's label. Q1 to Q4, Q6 and Q7 took the recommended option. Q5 and Q8
did not: the record gives the owner's choice as it was put, and no reason the owner did not give. Q9 arose from Q5's
answer and was answered the same day, as recommended. No owner question is open. The PLAN is written for these
answers, and the owner approved it on 2026-11-19 (its execution record).

*(2026-11-20: "No owner question is open" was true until the state table was added. Its six open rows are Q10 to Q14
below, put to the owner as one prompt before P1. None changes the decision or an answer above.)*

* **Q1. The line-end rule (D3).**
  * (a) `\r`, `\n` and `\r\n` each end a line; after a `\r` that ended a line read out of raw mode, the next `\n` is
    dropped even when it comes in a later read; after a raw secret's `\r`, only a `\n` already read is dropped; `\n`
    then `\r` is two ends. *Recommended:* it covers a Windows console's line read, a pasted `\r\n`, and Unix's cooked
    Enter after a secret (weakness (d)).
  * (b) As (a), and `\n` then `\r` is also one end. Cost: on Unix, Enter on an empty raw secret right after a cooked
    answer would be dropped, so an intended cancel needs a second Enter.
  * (c) Fold `\r\n` only when both bytes come in one read. Cost: a long line split across reads leaves a stray `\n`,
    and existing tests that send the `\n` in a later chunk change.
  * **Answer (2026-11-19): (a),** "CR, LF and CR+LF each end one line; LF then CR counts as two (Recommended)".
* **Q2. Which readers go through the shared input.**
  * (a) Every reader of the terminal in the process: the sign-in questions, the line editor, the permission prompt and
    the trust question. The editor and the drawn prompts keep what they decode past an answer, as today.
    *Recommended:* one owner of stdin and one line-end rule, without reaching inside the editor's decoding.
  * (b) As (a), and the editor and the drawn prompts also give back what they read past an answer, re-encoded from
    their decoded keys and pastes, so read-ahead is shared both ways. Cost: a change inside the editor's read loop and
    `askOn`, with risk to paste and the two-stage Ctrl+C; no report asks for it.
  * (c) The sign-in questions only (choose, confirm, input, secret), across asks. Cost: the editor, the permission
    prompt and the trust question keep reading the terminal their own way, and setup's and acp's readers stay
    separate.
  * **Answer (2026-11-19): (a),** "Every reader in the process, each keeping its own decoding (Recommended)".
  * *(2026-11-20: narrowed by the owner's answer to Q10, relayed by the coordinator, "Record them as outside the
    shared input, with reasons (Recommended)": "every reader" is every reader inside an interactive session (the line
    session, its first run, `gobble setup`, `gobble acp --login` with a menu, `gobble auth login`'s paste prompter,
    the trust question). The readers found since, rows R91 to R96, stay outside it, each for the reason its row
    gives. On "keeping its own decoding" for `askOn`, see Q19.)*
  * *(2026-11-20, third pass: "keeping" is narrowed by the owner's answer to Q19, relayed by the coordinator,
    "Dropped, as today (Recommended)": the line editor keeps what it decoded past an answer for its next prompt;
    `askOn`, for the permission prompt and the trust question, drops it when it returns. Row R57.)*
* **Q3. What a sign-in typed ahead becomes when the session's prompt reads again.**
  * (a) Dropped, silently, as today. *Recommended:* nothing typed or pasted during a sign-in, a key's paste included,
    is sent to a model as a turn.
  * (b) Given to the line editor as typed, so a line typed ahead is the next prompt.
  * (c) Dropped, with a notice of how many characters were dropped.
  * **Answer (2026-11-19): (a),** "Dropped silently, as today (Recommended)".
* **Q4. Enter on an empty key prompt.**
  * (a) Still cancels the sign-in, with today's words. *Recommended:* once the stray line end is gone, an empty Enter
    is deliberate, and Ctrl+C and Enter then agree.
  * (b) Asks once more, saying an empty key cancels, and cancels on a second empty Enter.
  * (c) Asks again until a key is typed or Ctrl+C is pressed.
  * **Answer (2026-11-19): (a),** "Still cancel, with today's wording (Recommended)". In the first run, the offer then
    comes back (Q5, D8).
* **Q5. A first-run sign-in that ends without a credential.**
  * (a) After its own line, the line session prints the lead and the warning a declined offer prints, naming the
    variables. *Recommended:* the user learns at once that no provider is set up (weakness (b)).
  * (b) As today: the cancel or error line alone.
  * (c) Offer the sign-in again, until it succeeds or is declined.
  * **Answer (2026-11-19): (c),** "Offer the sign-in again until it succeeds or is declined", not the recommended
    option. As it was put to the owner: the first-run offer repeats after a failed or cancelled sign-in; more guidance
    for a new user; a loop to get right. D8 is the design; (a)'s lead and warning remain, printed once, when the loop
    ends without a credential.
* **Q6. Weakness (c), one Enter turned into two line ends by the terminal.**
  * (a) Leave it, and record it. *Recommended:* only scripted input showed it, and the two ends cannot be told from
    two Enters.
  * (b) Refuse, and ask again for, a typed key of digits only. Cost: a rule on key shapes gobble does not own; that no
    provider's key is digits only is not checked here and would be, against each provider's documented format.
  * (c) Discard input typed before each question's prompt is shown. Cost: it undoes the read-ahead the owner chose.
  * **Answer (2026-11-19): (a),** "Leave it and record it (Recommended)".
* **Q7. A console-level test on Windows.**
  * (a) A committed pseudo-console test: a Windows-only, build-tagged test that starts a child on a pseudo console
    (`CreatePseudoConsole`, in `golang.org/x/sys` `v0.49.0`, already required), types the first run's answers with
    Enter, and checks that the first line read ends `\r\n` and that the key prompt takes the key; a `make` target,
    run on the Windows test laptop through `run-on-host`, not in preflight or CI. *Recommended:* it runs headless, so
    over ssh, measures D6's third assumption, and stays as a regression test. Cost: new test code.
  * (b) A sign-in case added to the live Console Host suite (`make probe-conhost`,
    `internal/cli/editor/conhost_windows_test.go`). Cost: it needs the laptop's desktop session, opens a window and
    borrows the clipboard, so it is run by hand.
  * (c) The owner's check alone.
  * **Answer (2026-11-19): (a),** "A committed pseudo-console test, run on the Windows laptop by a make target, outside
    CI (Recommended)". PLAN P1b, which stops if a pseudo console cannot be created in the dispatcher's ssh session.
  * *(2026-11-20: the answer stands. Only who starts the run changes, by the owner's ruling D16: an agent starts it
    on the Windows test laptop through the fleet's host tooling (`run-on-host --need windows`), not the owner.)*
* **Q8. Which release carries the fix** (the release itself waits for the owner's word; the PLAN stops before it).
  * (a) `v0.6.1`, cut from `v0.6.0` with this change alone, by a record of its own, as
    [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md)
    D2 requires of a release before `v0.7.0`. *Recommended:* on a Windows console, every user of the released version
    with no key variable who takes the first-run offer and picks a key provider meets the defect. Cost: the change
    applied a second time on `v0.6.0`'s base, where `login.go`,
    `auth.go` and `acp.go` differ, and its own gates and release run.
  * (b) Carried by `v0.7.0` with the sign-in redesign; no earlier release. Cost: Windows users keep the defect until
    then; `/login gemini` and an environment variable are the ways round it.
  * (c) Decided after the owner's check (PLAN P6), with its result in hand.
  * **Answer (2026-11-19): (c),** "Decide after the Windows check has passed", not the recommended option. The PLAN
    builds and verifies on today's code and stops after P6; no `v0.6.1` work is planned now.
* **Q9. The default answer of a repeated first-run offer (D8).** The first offer keeps `[Y/n]`, as today.
  * (a) The repeated offer keeps `[Y/n]`: Enter starts another sign-in.
  * (b) The repeated offer is `[y/N]`: Enter ends the loop with the warning, and `y` starts another sign-in.
    *Recommended:* after a sign-in that was just cancelled or failed, an Enter more likely means "move on" than "again";
    a stray empty line (weakness (c)) then ends the loop rather than starting a round; and the loop's way out takes one
    key.
  * (c) `[Y/n]` after a cancelled sign-in, `[y/N]` after a failed one (refused keys, offline, an outage), where
    retrying at once is less likely to help.
  * **Answer (2026-11-19): (b),** "The repeated offer is `[y/N]`: Enter ends the loop, `y` tries again (Recommended)".

*Added 2026-11-20, from the state table's open rows. Each is open; the recommended option is listed first.*

* **Q10. Readers of the terminal that D1 does not list (rows R52, R53).** At `b1994a0` the terminal is also read by
  `x/term` `ReadPassword` for `gobble --api-key -` in the line session's process, after the trust question and before
  the session (`credentials.go:213`, from `chat.go:64-66`), and for `gobble auth login --api-key -` (`auth.go:122`); by
  `gobble auth import --from-pi`'s `[y/N]` questions (`piimport.go:61-94`); by `gobble update`'s confirmation
  (go-selfupdate-lib, `update.go:143`, `:166-181`); and by the external editor that `/edit`, ctrl+x ctrl+e and
  `gobble config edit` run with stdin inherited (`slash.go:226-230`, `config.go:284-291`). Each reads the terminal
  itself today, with no read ahead.
  * (a) Record all of them as outside the shared input, each with its reason in the table: `auth login`, `auth import`,
    `update` and `config edit` are alone in their process; `--api-key -` reads before the session's first question,
    with nothing waiting in `inline.Reader` provided the trust question reads with no deadline (Q12 (a)); the editor
    child runs while `terminalInput` holds nothing (D4) and the line editor is not reading. *Recommended:* no code,
    and no reader there shares the terminal with a question that can leave bytes behind. Cost: Q2's "every reader" is
    then read as every reader that shares the terminal with another prompt; `ReadPassword` stays in two commands.
  * (b) Route the in-process ones through `e.terminalIn()`: `--api-key -` through `questions.secret`, `auth import`
    through `questions.confirm`, `update` through an `io.Reader` over `terminalInput.Read`; children unchanged. Cost:
    `credentials.go`, `chat.go`, `piimport.go` and `update.go` join P2 with tests and plants; `--api-key -` on a
    terminal echoes `*` where it now echoes nothing.
  * (c) Route only `--api-key -`, the one in the line session's process, through `questions.secret`; the rest as (a).
    Cost: `credentials.go` and `chat.go` join P2; two kinds of key prompt remain.
* **Q11. Ctrl+C during a read out of raw mode on a Windows console (row R40).** By reading Go 1.27.4, a console read
  that gives no characters is the end of input (`internal/poll/fd_windows.go` `readConsole`; `os/file_windows.go:110`,
  `ZeroReadIsEOF`); whether `ReadConsoleW` gives none on Ctrl+C in line mode is not measured. If it does, the shared
  input is closed for the process: the first run ends as a decline where D8 offers again, and the line editor's next
  read ends the session. Today the session's one reader has the same exposure after Ctrl+C at `/login`'s menu.
  * (a) Measure before P2: P1b's gobble mode sends Ctrl+C at the provider menu and expects the offer again, and the
    owner's check gains that step in both terminals; if the input ends, stop for a deviation. *Recommended:* D8's
    path is checked where it can fail, before any code depends on it. Cost: one P1b case and one check step; a
    possible deviation later.
  * (b) Decide now that a read giving nothing on a Windows console right after an interrupt gobble took is not the
    end of input, and `terminalInput` reads again. Cost: a Windows-only rule in D1 and D8, with a stubbed test and a
    plant, chosen before anything is measured.
  * (c) Accept and record it: on a Windows console Ctrl+C at a menu may end the first run and the session; Ctrl+C at
    a key prompt, a byte in raw mode, is not affected. Cost: D8's menu-interrupt path holds only off Windows consoles.
* **Q12. The context the trust question reads with (row R50).** Today `askTrust` reads `e.stdin` and ignores
  `askOn`'s deadlines (`trust.go:164-167`, `:184`), so no read is left waiting; D2 moves it to
  `terminalInput.Read(ctx, p)` without saying which context. With `askOn`'s deadlines, a timed wait (a lone Esc) can
  leave a read waiting in `inline.Reader`, started in raw mode; on Windows a console read keeps the mode it started in
  (D6), so the first-run offer after it would take keys read raw and not echoed.
  * (a) Read with the run's context and no deadline, as today: a timed wait waits for the next key, as the comment at
    `trust.go:166-167` says. *Recommended:* today's behaviour, and no read is left waiting for the next reader.
    Cost: none beyond a test.
  * (b) Read with `askOn`'s deadlines, and let a waiting read go to the next reader (D4). Cost: on Windows the first
    offer may read keys raw and unechoed; a lone Esc behaves differently from today.
  * (c) Read with deadlines, then wait for any waiting read to finish and drop it before the next prompt. Cost: the
    wait lasts until a key is pressed, and that key is lost; it reads ahead, against D1.
* **Q13. A line with no end, and how long a line may be (row R17).** Today `questions.pending` (`prompt.go:199-208`)
  and a secret's buffer (`prompt.go:251-294`) grow without a bound; other readers bound theirs (4 KiB, `acp.go:145`;
  4096 bytes, `piimport.go:78`; 64 KiB, `credentials.go:222`).
  * (a) No bound, recorded: questions are asked only on a terminal, whose line discipline bounds a line read out of
    raw mode; a secret is typed or pasted by the user; nothing is kept past the sign-in (D4). *Recommended:* no
    rule that a terminal can reach today. Cost: a very large paste into a key prompt is held in memory until Enter.
  * (b) Bound each answer at 64 KiB, as `--api-key -`; a longer one cancels the question with a message. Cost: a new
    rule, its words and a test.
  * (c) Bound each answer at 4096 bytes, as `auth import` and go-selfupdate-lib. Cost: as (b), and a pasted
    credential over 4 KiB is refused.
* **Q14. A panic while raw mode is on (row R31).** Today `secret` restores the terminal after `readSecret` returns,
  not in a deferred call (`prompt.go:237-244`), as the trust question (`trust.go:180-186`) and the permission prompt
  (`permission.go:257-263`) do, and `Main` recovers only Kong's exit (`main.go:175-184`): a panic there leaves the
  terminal raw, and on Windows the console keeps that mode after gobble exits.
  * (a) Defer the restore in `secret` alone, the prompt this record changes; one test with a reader stub that
    panics, and a plant. *Recommended:* the key prompt is where a raw terminal hides what is typed next. Cost: a few
    lines in `prompt.go`, a test and a plant in P2.
  * (b) Defer the restore at every raw-mode entry the shared input serves: the secret, the trust question and the
    permission prompt. Cost: `trust.go` and `permission.go` change beyond D2, with a test and a plant each.
  * (c) Record it out of scope: a panic is a defect fixed where it happens. Cost: a panic during a key prompt leaves
    the user's terminal without echo.

*Answers, 2026-11-20, the owner, relayed by the coordinator, each quoted as the option's label:*

* **Q10:** "Record them as outside the shared input, with reasons (Recommended)". Rows R91 to R96; Q2's answer is
  narrowed by a dated note.
* **Q11,** with the review's case 3 (Ctrl+Z): "Measure both in your Windows check; stop if Ctrl+C ends input
  (Recommended)". D14; rows R40 and R90.
* **Q12:** "No deadline, as today (Recommended)". D15; row R50.
* **Q13:** "No bound; record it (Recommended)". D15; row R17.
* **Q14:** "Defer the restore at the secret prompt (Recommended)". D15; row R31.
* **Q15** (the front matter while rows are open) was not asked: the coordinator relayed that with these rulings no
  row needing the owner remained except any this pass found. This pass found five, Q16 to Q20 below.
* The review's cases 1 and 2 were put to the owner beside these: "Flush the terminal's input at those points
  (Recommended)" (D12) and "Drop the abandoned prompt's line when it arrives (Recommended)" (D13).

*Added 2026-11-20 (second records pass), from rows the rulings above leave open. Each is open; the recommended
option is listed first.*

* **Q16. Which reads of a prompt that gave up are dropped, and what is dropped (row R86).** D13 drops "the abandoned
  prompt's line". Not every case is a line: a read started in raw mode returns keys, and ~~on POSIX a read started out
  of raw mode returns at the first key once raw mode is on (the terminal's mode is shared), which on the session
  prompt after a browser sign-in is what was typed into the paste prompt plus the first key typed~~ *(2026-11-21, D21:
  measured on Linux, such a read returns as raw mode goes on, with what was typed into the paste prompt alone; the
  first key completes it only when nothing was typed there, inferred)*. And a read the user
  gave up with Ctrl+C at a menu completes with the user's answer to the next prompt: dropping it means the repeated
  offer's first answer, or the session prompt's first line or key, is lost and must be typed again. The existing
  `TestLoginInterrupt` sends Ctrl+D after an interrupt and would wait for ever under (b).
  * (a) Drop whatever completes a read the program gave up (a browser sign-in's loopback; any `Input` whose context
    the program ended), line or keys; a read the user gave up with Ctrl+C is not dropped and goes to the next prompt,
    as D4 says today. *Recommended:* it stops what the review found, text from the paste prompt reaching a turn,
    without losing an answer the user typed to a prompt they could see. Cost: ~~on POSIX the first key typed at the
    prompt after a browser sign-in is lost,~~ *(2026-11-21, D21: on POSIX, what was typed into the paste prompt is
    lost, and the first key at the next prompt only when nothing was typed there)* on Windows the first line; the
    questions must tell the two kinds of give-up apart (they know which context ended).
  * (b) Drop whatever completes any read of a prompt that gave up, Ctrl+C's included: the ruling's words read
    literally. Cost: after Ctrl+C at a menu the next answer is lost and typed again; `TestLoginInterrupt`, the menu
    interrupt tests and the owner's step (B) 4 need a sacrificial input.
  * (c) Drop up to and including the first line end of what completes such a read, of either kind, and pass on the
    rest; a completion with no line end passes on whole. Cost: on POSIX, a partial redirect URL typed into the paste
    prompt without Enter reaches the first prompt when raw mode starts, the case the ruling meant to stop.
* **Q17. The flush fails (row R78).** `tcflush` or `FlushConsoleInputBuffer` can fail: a handle that is not what
  `caps.In` took it for (a console under another program), an I/O error.
  * (a) Go on without it, logged at debug. *Recommended:* the prompt still works; only typing held by the terminal can
    reach it, as today. Cost: D4 and D8's guarantee lapses on that terminal with no word to the user.
  * (b) Go on, with one warning on stderr per process. Cost: a new message that users of unusual terminals see.
  * (c) At a repeated offer, end the first run as a decline; at the session prompt, go on. Cost: a terminal that
    cannot flush never gets a second offer.
* **Q18. Raw mode cannot be switched on for the key prompt (row R28).** Today the secret returns the error, writes no
  prompt and reads nothing; in the first run the round then fails and the offer comes back (D8). The drivers decide only
  that no key is read with echo on.
  * (a) As today, recorded: the error, no prompt, nothing read; the round fails. *Recommended:* no key can be read in
    clear, and the reason is on screen. Cost: none beyond A3, its test.
  * (b) Fall back to `x/term`'s `ReadPassword`, which turns echo off its own way. Cost: a second reader of the
    terminal beside the shared input, against D1.
  * (c) Cancel with a sentence naming the failure, and in the first run end the loop rather than offer again. Cost: a
    new message, and a way out of the loop D8 does not have.
* **Q19. What `askOn` decoded past an answer (row R57).** Today its decoder is local, so keys decoded past the
  permission prompt's or the trust question's answer are dropped when it returns. Q2 (a)'s words say the drawn prompts
  "keep" what they decode past an answer.
  * (a) Dropped when `askOn` returns, as today, recorded. *Recommended:* no key decoded for one drawn prompt answers
    another one unseen. Cost: keys typed past a permission answer are lost, as now.
  * (b) Kept by each kind of drawn prompt for its next question, a decoder held by the session. Cost: a stray key can
    answer the next permission prompt unseen; a change around `askOn`.
  * (c) Given back to the shared input as bytes (Q2 (b), which the owner did not choose). Cost: decoded keys and
    pastes re-encoded inside `askOn`.
* **Q20. Which session prompts flush (row R82).** D12 flushes "at the session prompt"; D4's `discard()` runs before
  each `ReadPrompt`. A flush before every `ReadPrompt` also drops what a user types while a turn runs, which today is
  the next prompt's text.
  * (a) Flush only before the first `ReadPrompt` after a sign-in or after the first run. *Recommended:* it is what
    the review's case asked for, and typing ahead during a model's answer keeps working. Cost: a mark set by a
    sign-in; `discard()` and the flush run at different points.
  * (b) Before every `ReadPrompt`, with `discard()`. Cost: keys typed while a turn runs are lost in every session.
  * (c) Before every `ReadPrompt` that follows a local command or a sign-in, not one that follows a turn. Cost: a rule
    by what ran last, which `/edit` and the other local commands make hard to state.

*Answers, 2026-11-20 (third records pass), the owner, relayed by the coordinator, each quoted as the option's label
(D17):*

* **Q16:** "Only reads the program gave up (Recommended)". Rows R86, R102.
* **Q17:** "Go on, with one warning on stderr (Recommended)". Row R78; the warning's words, row R101, are proposed
  and await the owner's approval, the one thing about this record still open to the owner. *(2026-11-20, fourth
  pass: approved unchanged, "Plain words, with the consequence (Recommended)"; D17.)*
* **Q18:** "Keep that, recorded and tested (Recommended)". Row R28.
* **Q19:** "Dropped, as today (Recommended)". Row R57; Q2's answer narrowed by a dated note.
* **Q20:** "Only the first prompt after a sign-in or the first run (Recommended)". Rows R82, R103 to R108; every case
  of the mark is settled by the ruling's words or by D4, so none is put to the owner.

*Added 2026-11-20, after P1 step 0b: Q11's measurement gave the end of input, and the owner chose row R40's option
(a), "(a) After an interrupt gobble took, read again (Recommended)" (D19). Two points of that new mechanism are open.
Each is open; the recommended option is listed first.*

* **Q21. The interrupt mark's life (rows R111, R114).** D19 excuses a console read that gives nothing "right after"
  gobble received Ctrl+C. When the mark is set, which such read it excuses, and when it is cleared, the words do not
  say; nor whether a Ctrl+C with no read waiting (during a turn) may excuse one later.
  * (a) Set only when gobble receives Ctrl+C while a read of the console is waiting; it excuses the first read that
    then gives nothing, and is cleared by that read or by any read that returns bytes. *Recommended:* one interrupt
    excuses at most one empty read, so Ctrl+Z after it, or later, is still the end of input (R90), and a Ctrl+C during
    a turn excuses nothing. Cost: the shared input must know whether a read is waiting when the mark is set.
  * (b) Set on every Ctrl+C gobble receives; it excuses every read that gives nothing until a read returns bytes.
    Cost: Ctrl+C then Ctrl+Z with no key between is not the end of input, against R90 and R112.
  * (c) Excuses a read that gives nothing within a fixed time of the interrupt (for example 500 ms), with no clearing.
    Cost: a timing rule; a Ctrl+Z pressed soon after Ctrl+C is not the end of input, and a slow machine can miss the
    window.
* **Q22. A read that gives nothing before the mark is set (row R116).** The console runs its Ctrl+C handler on a
  thread of its own (inferred), so the empty read can reach the shared input before `watchSignals` has set the mark.
  * (a) An empty console read with no mark waits up to 100 ms for one before it is taken as the end of input; P1b's
    probe adds which came first, the signal or the read's return, to its observed word, to show whether the wait is
    ever needed. *Recommended:* it closes the race at a cost no person notices, and the probe shows whether it
    matters. Cost: Ctrl+Z ends the input up to 100 ms later; one more probe word.
  * (b) No wait: the mark is taken as it stands. Cost: when the race is lost, Ctrl+C at a menu closes the input, the
    stop the ruling removed.
  * (c) Wait up to 1 s. Cost: Ctrl+Z on a console ends the input up to a second late.

*Answers, 2026-11-20, the owner, relayed by the coordinator, each quoted as the option's label (D20):* **Q21:** "Set
only if a read is waiting; excuses one empty read (Recommended)", rows R111, R114. **Q22:** "Wait up to 100 ms for the
mark; the probe reports the order (Recommended)", rows R116, R122. **R120:** "Yes, widen the row to include it
(Recommended)". No owner question is open.

*Added 2026-11-21, D22 (proposed): the rows its draft leaves open. Each is open; the recommended option is listed
first.*

* ~~**Q23. Keys typed into a prompt the program gave up outside a sign-in (row R132).**~~ *(2026-11-21, D22 second draft:
  superseded by Q23, as rewritten, below.)* Under D13 a withdrawn permission
  request's read was dropped with what completed it. Under D22 nothing is read, so the keys stay in the terminal, and
  no sign-in mark flushes them.
  * (a) A program's give-up sets the same mark a sign-in sets, so the next session prompt flushes them. *Recommended:*
    D13's "never a turn, never an answer" kept by the same deterministic flush as D21, one rule for every give-up of
    the program. Cost: keys typed between the withdrawal and the next prompt are lost; `askPermission` sets the mark;
    a test and a plant.
  * (b) Leave them for the next prompt, where they show as typed text, sent only on Enter. Cost: a half-typed reason
    for a withdrawn permission request can start the next prompt's text.
  * (c) Read and drop, at the give-up, what the terminal already holds, without waiting. Cost: out of raw mode a half
    line cannot be read before its Enter, so it works for raw-mode prompts only; a read at give-up is what D22 removes.
* **Q24. The readiness wait cannot be used (row R138):** select(2) fails with an error other than `EINTR`, the wake
  pipe cannot be made, or a descriptor is at or above `FD_SETSIZE`.
  * (a) For the rest of the process, read as built (a blocking read, D13's drop), with one warning on stderr naming
    the error. *Recommended:* every prompt still works, and the warning names the cost, F4's race. Cost: a second
    warning's words for the owner to approve; a fallback path, its test and its plant.
  * (b) Take it as a read failure: the input is closed (`ended()`), a question cancels, the session ends at the
    editor's next read. Cost: gobble cannot be used on that terminal.
  * (c) Read as built, silently. Cost: the race comes back with no word to anyone.

*2026-11-21, D22 second draft: Q23 rewritten as the record review proposed (its 2.3), Q24 updated, and Q25 and Q26 added. Each is
open; the recommended option is listed first; each names the rows it amends.*

* **Q23, as rewritten. Keys typed into a prompt the program gave up (row R132).** On POSIX, under D22, they stay in
  the terminal, unread: at a withdrawn permission request inside the session; and at the paste prompt of a sign-in
  outside it (`gobble auth login`, `gobble acp --login`, `gobble setup`), where gobble then exits, so the shell runs a
  line typed there or keeps a pasted redirect URL, its code included, in its history, and under `gobble acp --login`
  acpserver reads it as protocol input (`acp.go:50`). *(2026-11-21, D22 third draft: a Ctrl+D typed there would reach acpserver as
  the end of input, R89.)*
  * (a) On POSIX, flush the terminal's queue at the program's give-up itself: `askPermission`'s withdrawn branch and
    `questions.cancelled`'s program branch. *Recommended:* one rule wherever the program gives up, inside the session
    or out of it. Cost: a third flush point; keys typed in the instant before the give-up are lost; a test and a
    plant. Amends R80 ("exactly two"), R81, R84's latest row and R132.
  * (b) A program's give-up sets the mark a sign-in sets, so the next session prompt flushes (the first draft's (a)).
    Cost: keys typed for the rest of the turn are lost too; nothing helps outside the session, where there is no next
    prompt. Amends R105, R107 ("a permission prompt ... set no mark") and R132.
  * (c) Leave them (the first draft's (b)). Cost: a half-typed reason can start the next prompt's text; outside the
    session, the shell runs it or keeps it in history, and acpserver reads it. Amends R132.
* **Q24, as updated. The readiness wait or its descriptor cannot be used (row R138).** It now also covers a private
  descriptor that cannot be opened, under Q25 (a).
  * (a) For the rest of the process, read standard input as built (a blocking read, D13's drop), with one warning on
    stderr naming the error. *Recommended:* every prompt still works, and the warning names the cost. Cost: F4's race
    and R95's risk come back there (R150); the warning's words for the owner to approve; a fallback path, its test and
    its plant. Amends R138, R150.
  * (b) Take it as a read failure: the input is closed (`ended()`), a question cancels, the session ends at the
    editor's next read. Cost: gobble cannot be used on that terminal. Amends R138, R43's reach.
  * (c) Read as built, silently. Cost: the race comes back with no word. Amends R138.
* **Q25. How the layer's read(2) cannot wait (row R126).** select(2) reported a line ready, Ctrl+C emptied it, and a
  read(2) of standard input then waited for the next line, on every host (M11); with the gate's lock held across that
  read, the close that Ctrl+C causes would wait with it.
  * (a) Read through a private descriptor: the terminal's own path (`ttyname` of standard input: `/proc/self/fd/0` on
    Linux, `F_GETPATH` on macOS), opened ~~`O_RDONLY|O_NONBLOCK|O_CLOEXEC`~~ `O_RDONLY|O_NONBLOCK|O_CLOEXEC|O_NOCTTY`
    *(2026-11-21, D22 third draft)*, its own open file description, so R125's
    objection does not apply; `EAGAIN` returns to the wait; raw mode and the flush stay on standard input.
    *(2026-11-21, D22 third draft: if the path cannot be opened and standard input is the controlling terminal, `/dev/tty` is
    opened with the same flags (M15: it sees what is typed); only then Q24's fallback. Standard input is known to be
    the controlling terminal by its foreground process group, the same read on standard input and on `/dev/tty`
    (M15); not by device numbers, since `/dev/tty` reports its own (M15). `O_NOCTTY` changed nothing seen (M15 against
    M12).)*
    *Recommended:* measured on every host (M12): it sees what is typed, gives `EAGAIN` with nothing queued and after
    Ctrl+C empties a ready line, leaves standard input blocking, and follows standard input's raw mode and flush.
    Cost: a second open of the terminal, whose failure is Q24; the path read per system. Amends R125, R126, R138, R146.
    *(2026-11-21, D22 third draft: also the case of `su` or `sudo -u`, where the terminal's path is not the user's and the open
    fails with `EACCES` (inferred, not measured): `/dev/tty` or Q24's fallback then applies. Also amends R127, R134 and
    R148.)*
  * (b) Keep standard input, and do not hold the lock across read(2): a read that waits after a close is an
    outstanding read, dropped by D13 when it completes. Cost: F4's race comes back in the window between readiness and
    the read (M11 shows Ctrl+C opens it). Amends R126, R127, R129, R130. *(2026-11-21, D22 third draft: also R150.)*
  * (c) Keep the draft and record the window. Cost: Ctrl+C after a line is ready, at a prompt read out of raw mode,
    hangs that prompt until another line is typed (M11). Amends R126, R133.
* **Q26. The pseudo-terminal cases on this Mac (row R152).** `internal/cli/pty_linux_test.go` is `linux && pty`, holds
  both the cases and the Linux opener, and the Makefile says `probe-pty` is Linux only; D21 allowed one line of it to
  change.
  * (a) Move the cases, word for word, to `pty_unix_test.go` (`(linux || darwin) && pty`), with the openers in
    `pty_linux_test.go` and a new `pty_darwin_test.go`, and update the Makefile's comment. *Recommended:* one copy of
    the cases, run on both systems. Cost: P1b's file restructured beyond D21's one line, which needs the owner's word.
    Amends R130, R131, R152 and the PLAN's P1b files.
  * (b) A macOS copy of the cases in a `darwin && pty` file. Cost: two copies that can drift. Amends R152.
  * (c) No pseudo-terminal run on macOS: R130 and R131 rest there on the probe's measurements alone. Cost: macOS
    untested end to end. Amends R130, R131, R152.

*Answers (2026-11-21, D22 accepted), the owner, relayed by the coordinator: each quoted as the option's label:* **Q23:** "Flush at gobble's give-up (Recommended)". **Q24:**
"Old reads plus one warning (Recommended)", with the warning's words `Warning: gobble cannot wait on this terminal without reading it (<error>); keys typed while a sign-in finishes may be lost.` **Q25:** "Private non-blocking descriptor (Recommended)". **Q26:** "Move them to a shared file (Recommended)". No owner
question is open.

## More Information

* [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md) D6 (deviation 7's note) and D7, and
  [0025-PLAN-provider-choice-and-auth.md](0025-PLAN-provider-choice-and-auth.md) deviation 7 and P5's execution
  record: the questions, raw mode for secrets, and the macOS pseudo-terminal run.
* [0027-MADR-native-line-editor-word-wrap.md](0027-MADR-native-line-editor-word-wrap.md) D5 as amended, and
  [0027-PLAN-native-line-editor-word-wrap.md](0027-PLAN-native-line-editor-word-wrap.md) P3 and P5: the reader that
  reads only when asked, and the Windows console measurements.
* [0029-MADR-permission-prompt-layout.md](0029-MADR-permission-prompt-layout.md): the permission prompt's layout, which
  this record does not change; under Q2 (a) its keys are read through the shared input.
* [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md) D7: `/login`, setup
  and the first run through the SDK's wizard, over gobble's prompter.
* [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md)
  D2: release numbering before `v0.7.0`.
* [0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md](0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md):
  no race result from a WSL distro is taken.
* The plan: [0063-PLAN-one-shared-terminal-input-so-the-key-prompt-waits.md](0063-PLAN-one-shared-terminal-input-so-the-key-prompt-waits.md).
