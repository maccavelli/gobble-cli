---
status: accepted
date: 2026-11-19
decision-makers: repository owner
consulted: "this repository at 21be074 (branch p3-acpext-v0.5.0, equal to main); go-llmprovider-sdk v1.31.0 (wizard/configure.go, wizard/auth.go); go-tui-lib v2.12.1 (inline/reader.go, termkey/input.go); Go 1.27.4 (src/internal/poll/fd_windows.go); an independent diagnosis's scratch experiments, not committed"
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# One shared terminal input for every prompt in a session, so a menu answered with CR+LF no longer skips the API key prompt

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
    person's Ctrl+C. The consequence, row R102: the first key (POSIX) or line (Windows) typed after such a browser
    sign-in is lost.
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
| R40 | Ctrl+C during a read out of raw mode on a Windows console | ~~open (Q11)~~ open: awaits live evidence (2026-11-20) | Inferred, not measured: by reading Go 1.27.4, a console read that gives no characters is the end of input (`internal/poll/fd_windows.go` `readConsole`, `os/file_windows.go:110`). Observation "the read still waits" (`probe-ctrlc` `waiting`, `menu-ctrlc` `offer-again`): the build goes on, and R37 holds on Windows. Observation "end of input" or an error (`eof`, `error`, `ended`): a named stop; the owner chooses among (a) a console read that gives nothing right after an interrupt gobble took is not the end of input, and `terminalInput` reads again (Windows only; a stubbed test and a plant), (b) menus on a Windows console read in raw mode with gobble's own line echo (option C, rejected once), (c) accept it and amend D8 and the checks: on a Windows console Ctrl+C at a menu ends the input | D14; Q11, the owner, relayed by the coordinator: "Measure both in your Windows check; stop if Ctrl+C ends input (Recommended)" | `probe-ctrlc` (P1 step 0, before any code); `menu-ctrlc` (P1b); the owner's step (B) 4 (P6) |
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
| R71 | A round succeeds | decided | The loop ends; the offer was asked once | D8 | `TestFirstRunSetup` (existing, each E from P1 step 2); proposed: its offer counted once (P1) |
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
| R80 | Which points flush | decided | Exactly two: with `discard()` before the session prompt reads, and after a repeated offer is written, before its read | D12; the same ruling | `TestFlushPoints` (A24); plants 42, 43, 44 |
| R81 | Points that do not flush | decided | The first offer (D8 keeps read-ahead from the offer to the key in the first round); every question inside a round (D4); `gobble setup`, `gobble acp --login` and `gobble auth login` (no session prompt and no repeated offer; the process ends after); the trust question (before the session; not one of the two points); the permission prompt (not one of the two points: keys typed while a turn runs can answer it, as today, a matter outside this record) | D12: the ruling names two points and no other | `TestFlushPoints` (A24, no call at each); plant 42 |
| R82 | The session-prompt flush: before every `ReadPrompt`, or only the first after a sign-in | ~~open (Q20)~~ decided (2026-11-20, third pass) | ~~Not settled: "at the session prompt" with D4's "before each `ReadPrompt`" would also drop keys typed while a turn runs, which no ruling asked for~~ Only before the first `ReadPrompt` after a sign-in or the first run, by a mark: rows R103 to R108; keys typed while a turn runs are kept for the next prompt, as today | D17; Q20, the owner, relayed by the coordinator: "Only the first prompt after a sign-in or the first run (Recommended)" | `TestFlushPoints` (A24, as extended), plants 56, 57, 58 |
| R83 | A flush while a read of a prompt that gave up is still waiting | decided | The flush runs first; the waiting read's completion is then dropped by D13; the prompt that waits reads again. On a Windows console, whether the flush also empties the line being edited in a waiting line-mode read is inferred, and measured by `abandoned` | D12; D13 | `abandoned`, `pty-abandoned` (A21, P1b); plant 46 |

**The read of a prompt that gave up (D13)**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| R84 | A read started out of raw mode by a prompt that gave up (a browser sign-in's paste prompt), on POSIX, completed by a line while the terminal is still out of raw mode | decided | The line is dropped: never a turn, never an answer; the prompt waiting reads again | D13; review case 2, the owner, relayed by the coordinator: "Drop the abandoned prompt's line when it arrives (Recommended)" | proposed: `terminput_test` abandoned rows (A28, P2); `pty-abandoned` (A21); plant 47 |
| R85 | The same read on a Windows console, which keeps it in line mode | decided | The console echoes and edits what is typed at the next prompt until Enter (inferred); that line, ending `\r\n`, is dropped; the prompt's own read follows in its own mode | D13; the same ruling | `abandoned` (A21, P1b, a stubbed browser sign-in): pinned by the pseudo-console test only; the owner makes no sign-in for a test |
| R86 | A completion that is not a line: a read started in raw mode by a prompt that gave up, or a POSIX read started out of raw mode that completes after raw mode is on; and a read the user gave up with Ctrl+C rather than the program | ~~open (Q16)~~ decided (2026-11-20, third pass) | ~~Not settled by the ruling's words ("the abandoned prompt's line")~~ Whatever completes a read the program gave up (a browser sign-in's loopback; any read whose context the program, not the person, ended) is dropped, line or keys. A read the person gave up with Ctrl+C is not dropped and goes to the next prompt, as D4 says | D17; Q16, the owner, relayed by the coordinator: "Only reads the program gave up (Recommended)" | `terminput_test` (A28, as extended): a program give-up dropped, keys included; a Ctrl+C give-up kept; `TestLoginInterrupt` (existing) unchanged; plants 47, 60; (2026-11-20, fourth pass) "whole" by D18: the 512-through-256 row of A28, plant 63 |
| R87 | A timed wait inside a prompt that is still reading (the editor's Esc timer, `askOn`'s timers) | decided | Not a prompt that gave up: the same prompt's next read takes the read, as D4 says | D13 ("a read belonging to an abandoned prompt"); D4 | proposed: `terminput_test` (A28); the editor's own tests (existing); plant 48 |
| R88 | A second prompt gives up while the first one's read still waits | decided | `inline.Reader` has one read in flight, so both belong to it; its completion is dropped once, and the next prompt reads anew | D13 | proposed: `terminput_test` (A28); plant 49 |
| R89 | The input ends while a prompt that gave up has its read waiting | decided | The end of input is not a line and is not dropped: `closed` is set, `ended()` is true, and D8 applies | D13 (lines only); D8 | proposed: `terminput_test` (A28); plant 50 |

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
| R95 | The external editor of `/edit` and ctrl+x ctrl+e, which inherits stdin (`slash.go:226-230`) | out of scope | It runs while the line editor is not reading and `terminalInput` holds nothing (D4); a read of a prompt that gave up, if one waits, would take its first keys, which D13 then drops: a rare case left as it is | Q10 | — |
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
| R102 | The first key or line typed after a browser sign-in whose loopback came first | decided | Lost: on POSIX the first key typed at the next prompt completes the read the program gave up, and is dropped with it; on a Windows console the first line (inferred: the console keeps that read in line mode, R85). The prompt then reads again | D17; Q16, the owner, relayed by the coordinator: "Only reads the program gave up (Recommended)", whose stated consequence this is | `pty-abandoned`, `abandoned` (A21), plant 47 |
| R103 | A sign-in inside the session: `/login`, or one another command starts through the same round (`ls.signInRound`), whatever its outcome (signed in, cancelled, failed) | decided | Sets the mark: the next `ReadPrompt` flushes. A `/login` refused before any question (an unknown provider) is no sign-in and sets none | D17; Q20, the owner, relayed by the coordinator: "Only the first prompt after a sign-in or the first run (Recommended)"; D4 ("Nothing typed or pasted during a sign-in becomes a turn", whatever the sign-in's end) | `TestFlushPoints` (A24, as extended); plant 57 |
| R104 | The first run, however it ends: declined at the first offer, Ctrl+C at an offer, a round signed in, the loop ended by a decline after rounds | decided | Sets the mark: the first `ReadPrompt` flushes | D17; Q20 ("or the first run") | `TestFlushPoints` (A24, as extended); plant 58 |
| R105 | The prompts after that first one: after a turn, a local command, `/edit` | decided | The mark is cleared by the flush it causes; later `ReadPrompt`s do not flush, so keys typed while a turn runs are the next prompt's text, as today | D17; Q20 ("Only the first prompt") | `TestFlushPoints` (A24, as extended); plant 56 |
| R106 | A turn that runs before that first prompt: the `/models` turn after `/login` signs in (`login.go:262`), or a prompt given on the command line after the first run (`line.go:224-228`) | decided | Keys typed while it runs are flushed at the first `ReadPrompt` that follows: the mark is consumed by the prompt read, not by a turn | D17; Q20 ("the first prompt after a sign-in or the first run": the session prompt) | `TestFlushPoints` (A24, as extended); plant 56 |
| R107 | `/logout`, the trust question, a permission prompt, a resumed session with a credential (no first run, no sign-in) | decided | Set no mark: no flush at the prompt after them | D17; Q20 ("Only … after a sign-in or the first run") | `TestFlushPoints` (A24, as extended: no call after each); plant 56 |
| R108 | The process ends, or the session ends, with the mark set | decided | Nothing: the mark lives in the session's `terminalInput` and nothing is flushed on the way out | D17; Q20 (a flush belongs to a prompt read) | `TestFlushPoints` (A24, as extended: a session ended by Ctrl+D with the mark set counts no flush); plant 61 |

~~Counts (2026-11-20, third pass): 108 rows. R52 and R53 are replaced by R91 to R96 and not counted. Of the other 106:~~
~~88 `decided`, 16 `out of scope` (R24, R36, R49, R54, R55, R73, R91 to R100), 1 awaiting live evidence (R40, measured~~
~~first at the PLAN's P1 step 0b), and 1 `open` for its words only (R101, the warning of R78).~~
*(2026-11-20, fourth pass: superseded; R101 is decided, see the fourth-pass counts.)*

Counts (2026-11-20, fourth pass): R101 decided on the owner's approval of its words. Of the 106 counted rows, 89
`decided`, 16 `out of scope`, 1 awaiting live evidence (R40), and none `open`.

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
  prompt's line". Not every case is a line: a read started in raw mode returns keys, and on POSIX a read started out
  of raw mode returns at the first key once raw mode is on (the terminal's mode is shared), which on the session
  prompt after a browser sign-in is what was typed into the paste prompt plus the first key typed. And a read the user
  gave up with Ctrl+C at a menu completes with the user's answer to the next prompt: dropping it means the repeated
  offer's first answer, or the session prompt's first line or key, is lost and must be typed again. The existing
  `TestLoginInterrupt` sends Ctrl+D after an interrupt and would wait for ever under (b).
  * (a) Drop whatever completes a read the program gave up (a browser sign-in's loopback; any `Input` whose context
    the program ended), line or keys; a read the user gave up with Ctrl+C is not dropped and goes to the next prompt,
    as D4 says today. *Recommended:* it stops what the review found, text from the paste prompt reaching a turn,
    without losing an answer the user typed to a prompt they could see. Cost: on POSIX the first key typed at the
    prompt after a browser sign-in is lost, on Windows the first line; the questions must tell the two kinds of give-up
    apart (they know which context ended).
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
