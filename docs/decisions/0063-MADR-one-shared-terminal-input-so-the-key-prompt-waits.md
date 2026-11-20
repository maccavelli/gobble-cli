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
