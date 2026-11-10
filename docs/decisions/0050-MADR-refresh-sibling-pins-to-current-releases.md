---
status: accepted
date: 2026-11-10
decision-makers: repository owner
consulted: "this repository at cdfaead (main, 0049-PLAN P8 committed, 2026-11-10); the module proxy's and GitHub's tags of acp-go-sdk2, go-llmprovider-sdk, go-selfupdate-lib and go-tui-lib; go-selfupdate-lib v1.26.0's CHANGELOG.md; go-llmprovider-sdk v1.29.0's release notes; scripts/check_libs.py runs of 2026-11-10"
informed: "0049-PLAN (its P5, go-llmprovider-sdk #13's fields)"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Pin each sibling library's current release: go-llmprovider-sdk v1.29.0, go-selfupdate-lib v1.26.0, acp-go-sdk2 v1.35.0

## Context and Problem Statement

gobble pins each of its four sibling libraries at a tag, and 0035-MADR D2 has each pin moved to the library's current
release after `make check-libs` passes for it. Three of the four have released since gobble last moved them. On
2026-11-10 the owner asked for the pins to be refreshed to the current releases. One of them, acp-go-sdk2 `v1.35.0`, was
still in its CI when asked; the owner tags it, then gobble pins it.

### What holds at `cdfaead` (read 2026-11-10)

* **F1. The pins** (`go.mod`), against each library's newest tag on the module proxy and on GitHub (both agree):

  | Library | Pinned | Newest | Move |
  | :--- | :--- | :--- | :--- |
  | `github.com/maccavelli/acp-go-sdk2` | `v1.34.2` | `v1.34.2`; `v1.35.0` in its CI | to `v1.35.0` once tagged |
  | `github.com/maccavelli/go-llmprovider-sdk` | `v1.28.0` | `v1.29.0` | minor |
  | `github.com/maccavelli/go-selfupdate-lib` | `v1.25.0` | `v1.26.0` | minor |
  | `github.com/maccavelli/go-tui-lib/v2` | `v2.12.0` | `v2.12.0` | none |

* **F2. `check_libs.py` passes both tags that exist.** `scripts/check_libs.py --lib go-selfupdate-lib --tag v1.26.0`
  and `--lib go-llmprovider-sdk --tag v1.29.0`, each on a scratch clone, both gave `0 failing checks`. Each run covers
  get and tidy with no other module moving, build, vet for three OSes, the importing packages' tests, staticcheck,
  lint, apidiff (compatible), govulncheck and the GitHub advisories. So neither tag breaks gobble as it stands.
* **F3. go-selfupdate-lib `v1.26.0`** (its `CHANGELOG.md`, 2026-11-09):
  * **Added:** `SymlinkInvocationError`, with `Link`, `Target` and a `SymlinkSource`, which unwraps to
    `ErrSymlinkInvocation`. Also four sentinels: `ErrSpecialModeBits`, `ErrNoAllowedRoot`,
    `service.ErrBinaryMismatch` and `taskscheduler.ErrLogonType`.
  * **Changed:** eleven run-time messages now state the fact and name no Go identifier. Among them is the symlink
    refusal, which reads `<link> links to <target>; start <target> itself to update it` for a program started
    through the link.
  * **What gobble reads:** it uses `ErrSymlinkInvocation` with `errors.Is` (`internal/cli/update.go:213`), which
    holds. `TestUpdateSymlink` and the allowed-root test check only text that is kept (`refusing to update through a
    symlink`, `links to`, `outside allowed roots`), and pass.
  * **Not changed:** no `.github/workflows` file differs between `v1.25.0` and `v1.26.0`.
* **F4. gobble's CI runs go-selfupdate-lib's reusable release workflows,** pinned to `v1.25.0`'s commit
  (`.github/workflows/ci.yml:171`, `:183`, `875dec3c… # v1.25.0`). 0048-MADR Q6 (a) moves them with the module, so
  that one version is named everywhere. `v1.26.0`'s tag peels to `7c677842dface4b14b65b2e8b46aa0a7543bed13`.
* **F5. gobble's symlink hint now repeats the library.** `symlinkHint` (`internal/cli/update.go:48`, 0048-MADR A2-3)
  follows the refusal with "gobble does not update through a symlink; run the file the link points to, named above,
  as `<that file> update`". `v1.26.0`'s own line already says to start the target itself.
* **F6. go-llmprovider-sdk `v1.29.0` is the tag that carries #13's fields.**
  * [maccavelli/go-llmprovider-sdk#13](https://github.com/maccavelli/go-llmprovider-sdk/issues/13) closed
    2026-11-10. `v1.29.0`'s notes (its 0072-MADR) add `APIError.ResetAt`, the instant the service named, and
    `APIError.Limit`. `Limit` holds OpenCode Go's limit name, ChatGPT's window in minutes and plan, Gemini's per-day
    flag and violated quotas, and Kilo's credits link. The notes also add `ClassifyStreamFailureBody`.
  * Two fixes come with it. OpenCode's `BudgetLimitError` and `CreditLimitExceeded` become `ErrQuotaExhausted`, and
    a stream failure now carries the delay its error states.
  * This is the tag that 0049-MADR D10 and Q4 (a) wait for. 0049-PLAN P5 pins it and maps those fields, after its
    steps are amended to the tag's names.
  * A bare pin compiles and passes (F2), but uses none of the new fields. gobble's `llm/provider` keeps its own
    reconstructions (0049-MADR D4, D5).
  * Until 0049's P5, three answers change with the pin alone:
    * an OpenCode `BudgetLimitError` is a quota with no `limit`, where it was `rate`;
    * a Gemini per-day quota inside a stream gets no `limit`, where it was `rate`, since gobble's Gemini case requires
      HTTP 429;
    * `CreditLimitExceeded` is `credits` only on HTTP 402.
  * The owner's [maccavelli/gobble-cli#15](https://github.com/maccavelli/gobble-cli/issues/15), filed 2026-11-10,
    lists these changes and the fields, and asks for P5.
* **F7. Docs naming the pins:** `README.md:155-163`, `docs/architecture.md:189-194`, and the comment at
  `llm/provider/provider.go:736`, which cites `v1.28.0`'s classifier.

## Decision Drivers

* **Current releases** (0035-MADR D2), each one checked by `make check-libs` before it is pinned.
* **One version named everywhere:** the module, the CI workflows and the docs (0048-MADR Q6 (a)).
* **No silent drift from 0049:** go-llmprovider-sdk `v1.29.0` is that record's P5 tag, so its pin and its adoption
  must not contradict 0049-MADR D10.
* **Small, reviewable steps:** a pin that needs no code change is not held back by one that does.

## Considered Options

* **A. One record pins all three:** go-selfupdate-lib and go-llmprovider-sdk now, acp-go-sdk2 when tagged. 0049-PLAN
  P5 starts from the pinned `v1.29.0` and adopts its fields.
* **B. This record pins go-selfupdate-lib and acp-go-sdk2. go-llmprovider-sdk moves only through 0049-PLAN P5,** with
  its pin and adoption together.
* **C. Wait for acp-go-sdk2 `v1.35.0` and move all three in one step.**

## Decision Outcome

Chosen option: "A", because it does what the owner asked, every pin current, in one record. The two tags that exist need no
code change (F2), and acp-go-sdk2 follows once tagged. The adoption of `v1.29.0`'s fields stays where 0049 put it,
reviewed as a phase of its own. The owner chose it with Q1 (a), below.

### D1. The pins

* go-selfupdate-lib `v1.25.0` → `v1.26.0`; go-llmprovider-sdk `v1.28.0` → `v1.29.0` (Q1); go-tui-lib stays at
  `v2.12.0`, already current.
* acp-go-sdk2 `v1.34.2` → `v1.35.0`, once the owner has tagged it and the module proxy serves it, and only after
  `make check-libs --lib acp-go-sdk2 --tag v1.35.0` passes. If that tag needs a code change, the change is planned by
  amendment before it is made.
* Each move is `go get <module>@<tag>` and `go mod tidy`, with no other module moving. A module that moves is a
  deviation.

### D2. CI and docs name the same version

* `.github/workflows/ci.yml`'s two `uses:` lines move to `7c677842dface4b14b65b2e8b46aa0a7543bed13 # v1.26.0`.
* `README.md`'s and `docs/architecture.md`'s pins name the new tags.
* `llm/provider/provider.go:736`'s comment names `v1.29.0` under Q1 (a) or (b).

### D3. 0049's P5 starts from the pin (under Q1 (a))

0049-PLAN P5's step 1, the pin, is done by this record. P5 is amended to start from `v1.29.0` as pinned, to name the
tag's fields (`APIError.ResetAt`, `APIError.Limit` and its parts, `ClassifyStreamFailureBody`), and to cover
`BudgetLimitError`. It then runs on the owner's word as its own phase. 0049-MADR D10 is amended to match.

### D4. Questions for the owner

* **Q1. go-llmprovider-sdk `v1.29.0`, the tag 0049's P5 waits for (F6).**
  * **(a) Pin it here with the others, and let 0049-PLAN P5 adopt its fields afterwards, from the pin.**
    *(recommended)* Every pin is current at once, and `check-libs` shows the bare pin is safe. The adoption
    (`ResetAt`, `Limit`, stream failures, `BudgetLimitError`) gets its own amended phase and tests. Cost: 0049's P5
    and D10 are amended to say the pin came from 0050.
  * (b) Leave it to 0049-PLAN P5, which pins and adopts in one phase, run next. This record pins the other two. Cost:
    go-llmprovider-sdk stays at `v1.28.0` until P5 is amended, built and committed.
  * (c) Pin it here and close 0049's P5 without adopting the fields. Not recommended: it gives up 0049-MADR D10 and
    Q4 (a), so magic-cli-remote keeps gobble's reconstructed `resetAt` and gets no stated window.
* **Q2. gobble's symlink hint after go-selfupdate-lib `v1.26.0` (F5).**
  * **(a) Keep it as it is.** *(recommended)* 0048-MADR A2-3 chose it. It still adds the command to run
    (`<that file> update`), and it changes no test. Cost: the two lines say nearly the same thing.
  * (b) Drop the hint when the library's refusal is a `SymlinkInvocationError` from an invocation, since the library's
    line already gives the remedy. This amends 0048-MADR A2-3, and `TestUpdateSymlink` changes.
  * (c) Shorten the hint to the command alone, `hint: run <target> update`, with the target read from
    `SymlinkInvocationError.Target`. This also amends 0048-MADR A2-3, and the test changes.

#### Answers (2026-11-10)

The owner was asked with three options for each question and chose the recommended one for both: Q1 "Pin now,
P5 adopts", Q2 "Keep the hint"; and to go on through P1 at once.

* **Q1:** (a). go-llmprovider-sdk `v1.29.0` is pinned here with go-selfupdate-lib `v1.26.0`. 0049-PLAN P5 and
  0049-MADR D10 are amended to start from that pin, and P5 maps the tag's fields as its own phase (D3).
* **Q2:** (a). gobble's symlink hint stays as 0048-MADR A2-3 chose it; `internal/cli/update.go` does not change.

### Consequences

* Good, because every sibling pin is the library's current release, with `check-libs` evidence for each.
* Good, because the CI workflows and the module name one go-selfupdate-lib version.
* Neutral, because under Q1 (a) gobble runs on `v1.29.0` without its new fields until 0049's P5 lands. Its
  classification fixes apply at once. A `BudgetLimitError` is a quota with no `limit` until then.
* Bad, because acp-go-sdk2's move waits on the owner's tag, so this record completes in two steps.

### Confirmation

1. `go.mod` requires go-llmprovider-sdk `v1.29.0` (under Q1 (a)), go-selfupdate-lib `v1.26.0` and acp-go-sdk2 `v1.35.0`,
   with go-tui-lib `v2.12.0`. `go mod tidy` is clean, and no other module moved.
2. `make check-libs` for each moved library, at its tag: `0 failing checks`.
3. `ci.yml`'s two workflow pins name `v1.26.0`'s commit. `README.md` and `docs/architecture.md` name the new tags.
4. `make preflight` and `make race` pass, and `go test ./...` and the race tests pass on the Linux machine and the
   Windows laptop.
5. Under Q1 (a), 0049-PLAN P5 and 0049-MADR D10 are amended to start from the pin.

## Pros and Cons of the Options

### A. One record pins all three

* Good, because every pin is current, as asked.
* Good, because the SDK's adoption keeps the review 0049 gave it.
* Neutral, because gobble runs on `v1.29.0` for a time without its new fields.

### B. go-llmprovider-sdk only through 0049's P5

* Good, because 0049-MADR D10 is kept to the letter: P5 pins and adopts.
* Bad, because one pin stays behind until a larger phase is planned and built.

### C. Wait for acp-go-sdk2, then all three

* Good, because there is one commit.
* Bad, because two safe pins wait on another repository's CI.

## More Information

* Not decided here: adopting go-selfupdate-lib `v1.26.0`'s new sentinels (`ErrSpecialModeBits`, `ErrNoAllowedRoot`)
  for more specific hints in `gobble update`; that is an addition to 0048 of its own.
* [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md):
  D2, `check-libs` before a pin.
* [0048-MADR-gobble-update-command.md](0048-MADR-gobble-update-command.md): Q6 (a), the workflow pins; A2-3, the
  symlink hint.
* [0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md):
  D10 and Q4, the SDK's #13 fields.
* [0050-PLAN-refresh-sibling-pins-to-current-releases.md](0050-PLAN-refresh-sibling-pins-to-current-releases.md):
  the implementation.
