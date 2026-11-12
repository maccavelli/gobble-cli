---
status: in-progress
date: 2026-11-12
associated-madr: "0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement releasing gobble v0.6.0 now from main, and the ACP sign-in redesign as v0.7.0

Associated MADR: [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md)

## Goal

gobble `v0.6.0` is published from `main` as it stands at `cc1d55e`, records apart (MADR D1): an annotated tag on
`main`'s head, a green release run, and an immutable Release with its seven assets, built with Go 1.27.4. Every gobble
record that names `v0.6.0` as the ACP sign-in redesign's release says `v0.7.0` (MADR D2), and the records in the other
repositories that must follow are listed for the cross-repository coordinator (MADR D5). The plan is complete when the
MADR's Confirmation 1 to 7 is met.

The MADR is `accepted` (the owner, 2026-11-12). This PLAN is `proposed`: nothing below starts before the owner approves
it, whole or by phase. The tag in P2 needs the owner's explicit word on the day, in addition to that approval.

## Scope

**Base.** `main` at `cc1d55e`, level with `origin/main`. The tree holds this pair, untracked and staged, and nothing
else. `go.mod` requires Go 1.27.4, acp-go-sdk2 `v1.36.0`, go-llmprovider-sdk `v1.29.0`, go-selfupdate-lib `v1.26.2`,
go-tui-lib `v2.12.0` and acpext `v0.3.0`.

**In scope:**

* **P0:** dated notes in six records, `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`, its
  PLAN, `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`, its PLAN,
  `0054-MADR-go-1-27-4-and-sibling-pin-refresh.md` and its PLAN; `docs/README.md`'s two rows for this pair, and a note
  in its 0054 MADR row.
* **P1:** the gates on three hosts, and this PLAN's records. No file outside `docs/` changes.
* **P2:** the push of `main`, the tag `v0.6.0` and its push (the owner's), the release run, and reading the Release.
* **P3:** this PLAN's execution record, the MADR's Completion section, the statuses, `docs/README.md`'s status cells, and
  one dated note in 0054-MADR.

**Out of scope:**

* Any change to code, tests, `go.mod`, `go.sum`, workflows, `README.md`, `docs/architecture.md` or the guides. A gate
  that fails and needs one stops the plan (C4).
* go-llmprovider-sdk `v1.30.0` (MADR F10): 0051-PLAN P2's.
* An acpext tag (MADR D3). `acpext/v0.5.0` stays 0051-PLAN P3's.
* Editing the other repositories' records. P0 step 4 lists them; the coordinator makes those edits under each
  repository's own rules.
* magic-cli-remote's adoption of `v0.6.0` (MADR D5, "Not decided here").
* Deleting or moving any tag, and editing a published Release (MADR D4).

## Conventions

* **C1. Start state.** Each phase first reads `git status --short`, `HEAD`, `origin/main`, and `git diff --stat cc1d55e
  HEAD -- . ':!docs'`, which is empty in every phase. A difference from this plan is a deviation.
* **C2. Evidence.** Each command writes to a scratch log, and its exit status is taken before any filter. Temporary
  paths are quoted as `<TMP>`. Tests run in a shell with no proxy variable set, the shell 0054-PLAN's P3 record's
  accepted race run used.
* **C3. Records are edited by assertion.** Each note is placed by a script that asserts the quoted text occurs once
  before it inserts, and the file is read back after. The original text is never rewritten: a note follows it.
* **C4. Deviations stop the phase.** A failing gate, a newer Go patch release, a module that moves, a job that fails, or a
  Release that differs from P2's acceptance stops the phase. The owner is given the evidence and the ways to resolve it,
  and the choice is recorded here, dated, before work continues. A failed job is read from its log, not re-run until
  it passes.
* **C5. A test this plan does not touch fails** (`TestRuleStoreConcurrent` among them, 0054-MADR "Observed, not
  decided"): 0054-PLAN's C5 applies as written. The phase stops; the output is quoted as printed; the test is not
  skipped; a record of its own is opened.
* **C6. The freeze.** From P1's commit until P2's release run ends, nothing but this plan's commits lands on `main`.
  `sbom` compares the tag with `main`'s head when it runs (`.github/workflows/ci.yml:165-170`).
* **C7. Commits and pushes.** The agent stages by path. The coordinator commits with `git commit --no-edit`; the global
  `prepare-commit-msg` hook writes the message. P0 and P1 may be one commit or two, as the owner chooses. The disclosure
  guard runs before every push (`AGENTS.md`, "Identifiers"), and each push and the tag are the owner's, on an explicit
  word given in that turn.

## Implementation Steps

### P0 — the records

1. This PLAN becomes `in-progress` on the owner's approval. The MADR is already `accepted`.
2. **The notes,** each dated 2026-11-12, after the text it amends, quoting it, and linking
   [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md)
   (C3). The line numbers are `cc1d55e`'s. "Redesign → `v0.7.0`" means the text named the redesign's release; "stays
   true" means it named something this release carries, and the note says that `v0.6.0` is 0055's release.

   | Record | Line | Text it amends (abridged) | Note |
   | :--- | :--- | :--- | :--- |
   | 0051-MADR | `:180` | "moving gobble-cli and magic-cli-remote together from `v1.35.0` to `v1.36.0` before gobble's `v0.6.0`" | stays true: both moved before it |
   | 0051-PLAN | `:527` | "then gobble's next release, `v0.6.0`, tagged by the owner" | redesign → `v0.7.0`; `v0.6.0` was cut earlier, by 0055 |
   | 0051-PLAN | `:530`, `:531`, `:533` | "`v0.6.0` pins acp-go-sdk2 `v1.36.0`, in step with magic-cli-remote … `v0.6.0` is the gobble release that must pin the same acp-go-sdk2 tag … If either repository's pin moves again before `v0.6.0`" | stays true, and met by 0055's `v0.6.0`; `v0.7.0` must pin magic-cli-remote's tag again, read at 0246's G2, and the "moves in both, or the release waits" rule runs on to `v0.7.0` (MADR D2) |
   | 0051-PLAN | `:538` | "a gobble at or above `v0.6.0`" | redesign → `v0.7.0` |
   | 0051-PLAN | `:544` | "Magic is the only known one, and moves with `v0.6.0`" | redesign → `v0.7.0` |
   | 0051-PLAN | `:547` | "Rollback after gobble `v0.6.0`: the previous binary still reads `auth.json`" | redesign → `v0.7.0`; the previous binary is then `v0.6.0`'s |
   | 0051-PLAN | `:572` | "`v0.6.0` pins acp-go-sdk2 `v1.36.0` in step with magic-cli-remote" | stays true, as `:530` |
   | 0052-MADR | `:394` | "Run by the first `v0.6.0` process that resolves credentials" (D9) | redesign → `v0.7.0` |
   | 0052-PLAN | `:372` | "the owner tags gobble `v0.6.0` with 0051's work" | redesign → `v0.7.0` |
   | 0052-PLAN | `:402` | "P4; P5; P6; gobble `v0.6.0`, tagged by the owner" | redesign → `v0.7.0` |
   | 0052-PLAN | `:408` | "`v0.6.0` pins acp-go-sdk2 `v1.36.0`, in step with magic-cli-remote" | stays true, as 0051-PLAN `:530` |
   | 0052-PLAN | `:417` | "The migration runs once, at the first `v0.6.0` start. A plan session never existed in a release" | redesign → `v0.7.0`; the second sentence stays true at `v0.6.0`, which carries the four plan fields with no method that creates a plan session (MADR F5) |
   | 0052-PLAN | `:423` | "Rollback after `v0.6.0`: the previous binary reads `auth.json` with `kind` and `consents`" | redesign → `v0.7.0`; the previous binary is then `v0.6.0`'s, whose store is `cc1d55e`'s |
   | 0054-MADR | `:22` | "the owner approved moving … from acp-go-sdk2 `v1.35.0` to `v1.36.0` before gobble's `v0.6.0`" | stays true |
   | 0054-MADR | `:126` | "the owner's ruling of 2026-11-11 that `v0.6.0` is the gobble release that must pin the same tag" | stays true, met by 0055's `v0.6.0` |
   | 0054-MADR | `:186`, `:187` | D3, "Lockstep … before gobble's `v0.6.0` is tagged … If either repository's pin moves again before `v0.6.0`" | stays true; the rule runs on to `v0.7.0` |
   | 0054-MADR | `:225` | D6, "That release is `v0.6.0`, the owner's to tag after 0051-PLAN and 0052-PLAN" | no longer true: `v0.6.0` is cut before them, by 0055; the release after 0051-PLAN and 0052-PLAN is `v0.7.0` |
   | 0054-MADR | `:228`, `:231` | D6's note, "No `v0.5.1` is cut; the fix reaches users in `v0.6.0` … until its user moves to `v0.6.0`, whenever that is tagged" | stays true; `v0.6.0` is 0055's release, cut now |
   | 0054-MADR | `:306` | D7, "magic-cli-remote's gobble wire fixtures are recaptured against `v0.6.0` in any case (… 0246-PLAN …, Gate G2 and P7's L4)" | no longer true as written: 0246's recapture reads its G2 release, `v0.7.0`; `v0.6.0` is the first release whose `initialize` advertises `systemPrompt`, and a recapture at `v0.6.0` is magic-cli-remote's choice (MADR F9) |
   | 0054-MADR | `:371` | Consequences, "installed `v0.5.0` binaries stay exposed … until their users move to `v0.6.0`" | stays true |
   | 0054-MADR | `:452` | Pros of A, "reach acp-go-sdk2 `v1.36.0` well before `v0.6.0`" | unchanged: an option's argument, kept as written; no note |
   | 0054-MADR | `:511` | "Not decided here", "Whether a `v0.5.1` should carry Go 1.27.4 ahead of `v0.6.0`" | stays true; the note links 0055 for the release that did |
   | 0054-MADR | `:534` | Revisions, "No `v0.5.1` … the fix reaches users in `v0.6.0`" | stays true |
   | 0054-PLAN | `:221` | P2, "they are recaptured at `v0.6.0` in any case (magic-cli-remote" | as 0054-MADR `:306` |
   | 0054-PLAN | `:228`, `:269` | "MADR D3's lockstep is read at `v0.6.0`" | stays true |
   | 0054-PLAN | `:329` | Rollout, "The pins and the Go fix reach users with the next gobble release, `v0.6.0`" | stays true; it is 0055's |
   | 0054-PLAN | `:333` | Rollback of P2, "which matters at `v0.6.0` and not before" | stays true |

   0054-MADR `:452` is the one occurrence left without a note, on purpose. 0051-MADR `:720` ("requires a gobble at or
   above this release") names no number and means the redesign's, so it needs none.
3. **`docs/README.md`.** The 0055 MADR and PLAN rows after 0054's PLAN row, in their neighbours' form: the title as a
   link, then the status cell. In the 0054 MADR row, after "D6 no release tagged, no `v0.5.1`", the words "(2026-11-12:
   `v0.6.0` cut early by 0055)". The 0051 and 0052 rows name no release and do not change. They are not written with
   the pair, since the index is a tracked file outside the bootstrap exception, as 0054-PLAN's P0 step 2 treated its own
   rows.
4. **For the coordinator, not this repository's work.** The records outside this repository that name gobble's
   `v0.6.0`, read 2026-11-12, and what each must say. Each is amended under its own repository's rules, as dated notes.

   | Repository | Record | Where | What changes |
   | :--- | :--- | :--- | :--- |
   | magic-cli-remote | `0179-MADR-gobble-native-acp-provider.md` | D14 gate 3's A10 note; A10 ("knowingly unmet until gobble `v0.6.0`", "holds again from `v0.6.0`", "Where the gate is checked again", "Alternatives"); the Baselines' re-read for A10; the 2026-11-11 history entry | gate 3 is met by gobble `v0.6.0`, both sides on acp-go-sdk2 `v1.36.0`, from the day 0055-PLAN P2 publishes it; the G2 release, the redesign, is `v0.7.0`, and gate 3 is read again there; whether `KnownGoodVersion` and the fixture move to `0.6.0` before the cut-over is that repository's |
   | magic-cli-remote | `0179-PLAN-gobble-native-acp-provider.md` | entry "P4 — 2026-11-11", "Gate 3 at this commit" | the unmet window it names ends at gobble `v0.6.0` |
   | magic-cli-remote | `0217-MADR-update-every-pin-to-its-current-release.md` | F3k (the lockstep ruling, "`v0.6.0` is the gobble release that must carry the same tag", the system-prompt note), D4's amendment | the lockstep and system-prompt statements stay true of `v0.6.0`; "covered by the fixture recapture of 0246 … at its gate G2" now reads `v0.7.0`, so `v0.6.0`'s `initialize` frame is in no fixture until a recapture; the rejected option "gobble's `v0.6.0` would then be cut on v1.35.0" is history and stays |
   | magic-cli-remote | `0217-PLAN-update-every-pin-to-its-current-release.md` | P3j: "The lockstep", "in any case before gobble's `v0.6.0` is cut", step 4's note, "Not an acceptance criterion of this phase", Rollback, "P3j, executed" | gate 3 is met at gobble `v0.6.0`; step 4's "arrives with 0246's recapture" reads `v0.7.0`; the rollback's "after `v0.6.0` is cut treat it as a stop" now applies |
   | magic-cli-remote | `0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md` | Consequences ("waits for … gobble's `v0.6.0`", R1's "P4 already waits for gobble's `v0.6.0` (G2)", "four more things gobble's `v0.6.0` must hold"); the 0179 A10 note in More Information | redesign → `v0.7.0`; the A10 note: met at `v0.6.0` |
   | magic-cli-remote | `0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md` | Gate G2 ("a gobble release at or above `v0.6.0`", "the G2 release is gobble-cli `v0.6.0`", "a release tagged `v0.6.0` on another tag"); Rollout and Rollback's order step 3 and its notes | redesign → `v0.7.0`; G2's gate-3 check stays, read from `v0.7.0`'s `go.mod`. Until amended, G2 still stops on `v0.6.0`, which has no `_meta.gobble.methods` and whose acpext has no `fresh` |
   | magic-cli-remote | `0060-MADR-local-unsigned-build-and-install.md` | F4, More Information | **no change:** its `v0.6.0` is magic-cli-remote's own release of 2026-08-01 |
   | go-llmprovider-sdk | `0073-MADR-publish-an-auth-contract-module.md` | the notes of 2026-11-11 (the lockstep ruling; "before gobble's `v0.6.0`") | stays true of `v0.6.0`; gobble's release on the contract is `v0.7.0` |
   | go-llmprovider-sdk | `0073-PLAN-publish-an-auth-contract-module.md` | the cross-repository order, step 3 ("gobble then tags `acpext/v0.5.0` … and its own `v0.6.0`; Magic's cut-over waits for both") | redesign → `v0.7.0`; the ruling's note stays true |
   | go-llmprovider-sdk | `0074-MADR-one-sign-in-handle-and-method-facts.md` | the deferral "to be raised after gobble-cli `v0.6.0` and magic-cli-remote's cut-over have shipped" | redesign → `v0.7.0` |
   | go-llmprovider-sdk | `0074-PLAN-one-sign-in-handle-and-method-facts.md` | Rollout ("tags `acpext/v0.5.0` and then gobble-cli's own next release, `v0.6.0`") and the ruling's note | redesign → `v0.7.0`; the ruling's note stays true |
   | go-llmprovider-sdk | `0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md` | Rollout ("Magic relays the `confirm` step after gobble's `v0.6.0`") and the ruling's note | redesign → `v0.7.0`; the ruling's note stays true |
   | go-llmprovider-sdk | `0076-MADR-auth-error-kinds-and-doc-drift.md` | the deferral "after gobble-cli `v0.6.0` and magic-cli-remote's cut-over have shipped" | redesign → `v0.7.0` |
   | fleet docs | `0006-MADR-sequence-the-auth-contract-effort-across-four-repositories.md` | Context's two release hazards; D1 step 5; D3; D4; D10; D11's "No interim gobble security release"; Consequences on gate 3 | redesign → `v0.7.0` in the hazards, D1 step 5 and D10; D1 gains gobble `v0.6.0` as a release before the chain; D3 stays true; D4: gate 3 met at `v0.6.0`, checked again at G2 on `v0.7.0`; D11: the owner's decision of 2026-11-12 cuts `v0.6.0` now with the fix, so the exposure ends at it (no patch of `v0.5.0` is cut) |
   | fleet docs | `0006-PLAN-sequence-the-auth-contract-effort-across-four-repositories.md` | Goal; the tag table's T6; stage 1's lockstep rule; stage 7 (title, exit check, install rule); stage 8 (G2, the recapture, the exit check); hazards H2 and H4; the lockstep command `git -C gobble-cli show v0.6.0:go.mod`; the execution log's 0179 A10 line | T6 → `v0.7.0`, and a new row for gobble `v0.6.0` (cut by gobble 0055-PLAN P2; check: the release job, and `go.mod` at the tag pins acp-go-sdk2 `v1.36.0`); stage 7, stage 8 and H2 → `v0.7.0`; H4 ends at `v0.6.0`; the lockstep command holds for `v0.6.0` and is run again for `v0.7.0` |

5. `python3 -B scripts/check_records.py --check-all` and `make check-records markdownlint`: exit 0 each.
6. Staged by path; the coordinator commits (C7).

Acceptance: every row of step 2 marked for a note carries one that quotes its text and links 0055-MADR, read back after
the edit; `grep -n 'v0\.7\.0'` finds each "redesign → `v0.7.0`" row's note; no line of the six records is deleted or
reworded (`git diff` of each shows additions only); step 3's rows link the two files; step 4's table is in this record
for the coordinator; the checks of step 5 pass.

### P1 — release preparation

This repository keeps no changelog file and no version line in `README.md` (MADR D1), so preparation is the gates and
the record.

1. **C1.** `HEAD` is P0's commit or `cc1d55e` with P0 staged, as the owner chose.
2. **Measure.** `go.dev/dl/?mode=json`'s newest stable Go; `go version` on macOS, the Linux machine and the Windows
   laptop. A Go patch newer than `go1.27.4`, or a host not on `go1.27.4`, stops the phase (C4): a release cut for a Go
   security fix is not cut on a Go that has a newer one without the owner's word.
3. **macOS gates** (C2):
   * `make preflight`: exit 0, "preflight passed", with govulncheck's "No vulnerabilities found." for both modules.
   * `GOFLAGS=-count=1 make race`: exit 0, no line cached.
   * `make check-test-isolation`: exit 0, "check-test-isolation: ok".
   * `python3 scripts/check_libs.py --self-test`: exit 0.
   * `python3 scripts/check_libs.py --lib <name> --tag <pin>` for each pin the release ships: acp-go-sdk2 `v1.36.0`,
     go-llmprovider-sdk `v1.29.0`, go-selfupdate-lib `v1.26.2`, go-tui-lib `v2.12.0`. Each gives `0 failing checks`,
     its GitHub security advisories included, which govulncheck does not see. The newest tag of each, from `git
     ls-remote --tags`, is recorded; go-llmprovider-sdk `v1.30.0` is expected and is not pinned (MADR F10).
4. **The Linux machine and the Windows laptop,** each at `go1.27.4`, from a copy made from `git ls-files -co
   --exclude-standard`, as 0054-PLAN's P1 to P3 records made it: the Linux copy streamed, the Windows copy by `scp`
   with `-o ControlPath=none` on every Windows `ssh`, and no `TMP` or `TEMP` override. On each: `go test -count=1
   ./...`, `CGO_ENABLED=1 go test -count=1 -race ./...` and `go -C acpext test -count=1 ./...`, each exit 0 with no
   `FAIL`. The copies and the remote processes are removed after. `make preflight` is a macOS gate: it needs the git
   history.
5. **The identifier scan** of the staged diff, for the local account name, the development hosts' names, the owner's
   mail domain and real-machine home and temporary paths: nothing found. A finding is described, never quoted.
6. **The record.** This PLAN's P1 record: each command, its exit status and the lines that carry its verdict. `make
   check-records markdownlint`: exit 0. Staged by path; the coordinator commits (C7).

Acceptance: step 2 finds no newer Go and every host on `go1.27.4`; every gate of steps 3 and 4 exits 0; the four
`check_libs.py` runs give `0 failing checks`; step 5 finds nothing; `git diff --stat cc1d55e HEAD -- . ':!docs'` is
empty after the commit.

### P2 — the tag (the owner's word)

1. **C1, and the freeze (C6).** The tree is clean; `HEAD` is P1's commit; `git diff --stat cc1d55e HEAD -- . ':!docs'`
   is empty; `git log --oneline cc1d55e..HEAD` lists this plan's commits alone.
2. **The disclosure guard,** run by hand over the outgoing commits as `AGENTS.md` gives it: exit 0.
3. **On the owner's word: `git push origin main`.** The push prints no "Bypassed rule violations" line. The push is
   docs-only, so the Docs workflow runs and CI does not (`.github/workflows/ci.yml:3-27`); the Docs run concludes
   `success`. `git rev-parse origin/main` equals `HEAD`.
4. **Before the tag:** `curl -fsS https://proxy.golang.org/github.com/maccavelli/gobble-cli/@v/list` lists no `v0.6.0`
   (`@v/list` alone); `git ls-remote --tags origin v0.6.0` prints nothing; `gh release view v0.6.0` reports no release.
5. **On the owner's word, the tag,** in `v0.5.0`'s form:

   ```sh
   git tag -a v0.6.0 -m v0.6.0 "$(git rev-parse origin/main)"
   git cat-file -p v0.6.0     # object <main's head>, type commit, tag v0.6.0, message v0.6.0
   git push origin refs/tags/v0.6.0
   ```

   The push prints no "Bypassed rule violations" line. The guard runs on it as on any push.
6. **The run.** `gh run list --workflow CI --branch v0.6.0 --json databaseId,headSha,event,status` names one run whose
   `headSha` is `main`'s head; `gh run watch <id> --exit-status` exits 0; `gh run view <id> --json jobs` gives every
   job's conclusion. No push to `main` until it ends (C6). A failed job stops the phase (C4, MADR D4).
7. **The Release** (only once `release / publish` has concluded `success`):
   * `gh release view v0.6.0 --json tagName,isDraft,isPrerelease,isImmutable,publishedAt,assets` and `gh api
     repos/maccavelli/gobble-cli/releases/latest --jq .tag_name`;
   * `gh release verify v0.6.0`;
   * the assets downloaded with `gh release download v0.6.0 --dir <TMP>/v0.6.0`; there `shasum -a 256 -c SHA256SUMS`,
     and `gh attestation verify <asset> --repo maccavelli/gobble-cli` for each of the five binaries;
   * `<TMP>/v0.6.0/gobble-darwin-arm64 version --identity` and `version --json`, after `chmod +x`.
8. **Gate 3** (MADR F8): `git show v0.6.0:go.mod` and `git -C <magic-cli-remote> show origin/master:go.mod`, after a
   fetch there, each give acp-go-sdk2 `v1.36.0`.
9. **developing.md's "After each `v*` release":**
   * `gh api 'repos/maccavelli/gobble-cli/rulesets/rule-suites?ref=refs/tags/v0.6.0&time_period=day' --jq '[.[] |
     select(.result == "bypass")] | length'` gives `0`;
   * on macOS, the Linux machine and the Windows laptop, each in a scratch home as 0048-PLAN's P3 ran it, `v0.5.0`'s
     release binary for that platform at `<scratch home>/.local/bin/gobble[.exe]`: `gobble update --check` exits 10 and
     names `v0.6.0`;
   * on the Windows laptop, then: start `gobble acp` from that binary and leave it running, run `gobble update --yes`
     from the same file (exit 0), stop `gobble acp`, and run a plain `gobble update` from the installed binary: it
     reports `v0.6.0` up to date and exits 0, and no `.gobble.exe.selfupdate-*` file is left beside it.
10. **The proxy,** after the Release is published: `go list -m github.com/maccavelli/gobble-cli@v0.6.0` resolves. Not
    before: a fetch fixes the version on the proxy.
11. **The record.** This PLAN's P2 record: the run id, each job and its conclusion, the Release's URL and `publishedAt`,
    each asset's SHA-256, and every reading above. Staged; the coordinator commits it with P3.

Acceptance, all of it:

* `v0.6.0` is an annotated tag whose object is `main`'s head, with the message `v0.6.0`.
* The tag's CI run concludes `success`, with `validate (ubuntu-24.04)`, `validate (macos-15)`, `validate
  (windows-2025)`, `test-isolation`, `sbom`, `build / build`, `build / identity (gobble, darwin/arm64)`, `(gobble,
  linux/amd64)`, `(gobble, linux/arm64)`, `(gobble, windows/amd64)`, `(gobble, windows/arm64)` and `release / publish`
  each `success`, and `release / sign` `skipped`, as in run `42718701229`.
* The Release `v0.6.0` is not a draft or a prerelease, is immutable, and is GitHub's Latest. Its assets are exactly
  `gobble-darwin-arm64`, `gobble-linux-amd64`, `gobble-linux-arm64`, `gobble-windows-amd64.exe`,
  `gobble-windows-arm64.exe`, `gobble.spdx.json` and `SHA256SUMS`.
* `gh release verify v0.6.0` passes; `shasum -a 256 -c` reports `OK` for the five binaries; each attestation verifies.
* The macOS binary prints `v0.6.0 (release) <main's head, abbreviated>` for `--identity`, and `version --json` gives
  `"version":"0.6.0"` and `"go":"1.27.4"`.
* Step 8 gives `v1.36.0` on both sides; step 9 gives `0` bypasses, exit 10 naming `v0.6.0` on three hosts, and the
  Windows sequence as written; step 10 resolves.

### P3 — close-out

1. This PLAN's execution record holds P0 to P2's records, with every deviation dated.
2. The MADR's Completion section: Confirmation 1 to 7, each with its evidence from the records, or named unmet with its
   reason.
3. This PLAN `complete`; the MADR stays `accepted`. `docs/README.md`'s two status cells follow, with the run id and the
   Release.
4. 0054-MADR: a dated note after D6's note of 2026-11-11, quoting "An installed `v0.5.0` binary stays exposed until its
   user moves to `v0.6.0`, whenever that is tagged": `v0.6.0` was published on its date, release run `<id>`, by
   0055-PLAN P2; and one after Consequences' "no gobble release pins `v1.35.0` or `v1.36.0` yet": `v0.6.0` pins
   `v1.36.0`.
5. `make check-records markdownlint`: exit 0. Staged; the coordinator commits; the push is docs-only, on the owner's
   word.
6. The handoff to the coordinator: the Release's URL and run id, gate 3 met, and P0 step 4's table, for the other
   repositories' amendments and magic-cli-remote's adoption decision.

Acceptance: Confirmation 1 to 7 met, or each unmet item named with its reason.

## Verification

Each command's exit status is captured before any filter.

| Confirmation | Command or reading | Where |
| :--- | :--- | :--- |
| 1 the tree is `cc1d55e`'s, records apart; the pins | `git diff --stat cc1d55e v0.6.0 -- . ':!docs'`; `git show v0.6.0:go.mod` | P1 acceptance, P2 steps 1, 8 |
| 2 the tag's form; no acpext tag | `git cat-file -p v0.6.0`; `git ls-remote --tags origin 'acpext/*'` lists `v0.1.0` to `v0.4.0` alone | P2 step 5 |
| 3 the run | `gh run view <id> --json jobs` | P2 step 6 |
| 4 the Release | `gh release view`; `gh release verify`; `shasum -a 256 -c`; `gh attestation verify`; `version --identity`, `version --json` | P2 step 7 |
| 5 gate 3 | both `go.mod` files | P2 step 8 |
| 6 the post-release checks | the rule-suite query; `gobble update --check` on three hosts; the Windows sequence | P2 step 9 |
| 7 the records | `grep -n '0055-MADR'` in the six records; `git diff` additions only; `make check-records markdownlint` | P0 steps 2 to 5 |

## Rollout and Rollback

* **Rollout.** P0 and P1 are docs-only commits on `main`, pushed by the owner. The tag is the owner's. Users get
  `v0.6.0` from the Release, or with `gobble update` from `v0.5.0`. magic-cli-remote takes it, or keeps `0.4.0`, by
  its own records.
* **If CI fails after the tag** (MADR D4). The `release` job needs `build`, which needs `validate` and `sbom`, so a
  failure before `release / publish` publishes nothing. The tag stays; it is not moved, and not deleted. The failing
  job is read from its log and not re-run to green (C4); C5 applies to a test this plan does not touch. The fix lands on
  `main` by an amendment of this pair, or by a record of its own when it is code, and the release is `v0.6.1`, tagged
  on the new head in the same form. `v0.6.0` stays an unpublished tag, and the records say so.
* **If `release / publish` fails part-way,** the state of the Release is recorded as `gh release view` shows it, and
  nothing is edited by hand; the fix is `v0.6.1` as above.
* **After a published `v0.6.0`.** The Release is immutable and stays. A defect is fixed by `v0.6.1`. `v0.5.0`'s
  binaries stay published, and a user who goes back to one is exposed to GO-2026-6859 again.
* **Before the tag.** P0 and P1 are records; reverting either commit undoes it.
* **The redesign.** `v0.7.0` is unaffected by any of this: a fix before it is a `v0.6.x` patch (MADR D2).

## Execution record

Not started. The MADR was accepted by the owner on 2026-11-12; this PLAN awaits the owner's approval. Writing the pair
changed no file but the pair's two: `docs/README.md` has no 0055 rows yet (P0 step 3), and no record named in P0 has a
note yet.
*(2026-11-12: the coordinator committed the pair as `554f3f4`, and the owner approved this PLAN the same day; P0 was
run and staged, see its record below. P1 to P3 are not started.)*
*(2026-11-12, later: the coordinator committed P0 as `7ee7591`; P1 was run and staged, see its record below. P2 and P3
are not started.)*

### P0 record (2026-11-12)

The coordinator committed this pair as `554f3f4`; the owner approved the PLAN the same day.

C1: branch `main`, `HEAD` `554f3f4`, `origin/main` `cc1d55e`, one commit ahead, the tree clean; `git diff --stat cc1d55e
HEAD -- . ':!docs'` empty. P0 is staged alone; P1 follows once it is committed (C7). No deviation.

1. **Status.** This PLAN's front matter `status` is `in-progress`, set by an assertion that the old line occurred once.
2. **The notes.** The line numbers of step 2's table are `cc1d55e`'s and hold at `554f3f4`, which changed only this
   pair: `grep -n 'v0\.6\.0'` gives the same 32 lines in the six records, none moved. A script placed 24 note blocks, 27
   notes, each dated 2026-11-12, quoting the text it amends and linking 0055-MADR, as new lines after it: after the
   line that ends the quoted sentence where the paragraph runs on (0051-MADR `:180`), otherwise after the last line of
   the paragraph or list item, and after that item's notes of 2026-11-11 where it has them. Before inserting, it
   asserted that each anchor line held its text and that each quoted phrase occurred once in its record (0054-PLAN's
   `:228` and `:269` close two acceptances with the same sentence, so that sentence was asserted to occur twice, each
   anchored by line); after, that every original line survived in order. 0051-MADR 1 note, 0051-PLAN 6, 0052-MADR 1,
   0052-PLAN 5, 0054-MADR 9, 0054-PLAN 5; 0054-MADR `:452` none, as step 2 says. `git diff --numstat` of the six
   records: 4, 23, 4, 18, 37 and 16 lines added, 0 deleted. Read back in full after the edit.
3. **`docs/README.md`.** The 0055 MADR and PLAN rows follow 0054's PLAN row, the title as a link, then the status cell;
   in the 0054 MADR row, "(2026-11-12: `v0.6.0` cut early by 0055)" after "D6 no release tagged, no `v0.5.1`".
4. **For the coordinator:** step 4's table, unchanged. No file outside this repository was edited.
5. **Checks,** each to a scratch log, its exit status taken before any filter:
   * `python3 -B scripts/check_records.py --check-all`: exit 0, no output.
   * `make check-records markdownlint`: exit 0, "Summary: 0 issues in 0 files".
   * The acceptance's `grep`: each of the nine "redesign → `v0.7.0`" rows (0051-PLAN `:527`, `:538`, `:544`, `:547`;
     0052-MADR `:394`; 0052-PLAN `:372`, `:402`, `:417`, `:423`) has a note naming `v0.7.0`; `grep -c '2026-11-12'`
     in each of the six records equals its note count.
   * `git diff --numstat`: the six records add lines only; `docs/README.md` adds two rows and changes one, the 0054
     MADR row, by step 3's words alone (numstat 3 and 1); this PLAN changes its `status` line and adds its records.
   * The identifier scan of the diff's added lines (the local account name, the development hosts' names, the owner's
     mail domain, and real-machine home and temporary paths): nothing found, exit 0. The scan was first seen to fail,
     exit 1 with two hits, on a scratch copy of the diff with a planted account name and home path.

   The two record checks and the scan were run again after this record was written, on the staged diff: exit 0 each.

### P1 record (2026-11-12)

The coordinator committed P0 as `7ee7591`.

C1: branch `main`, `HEAD` `7ee7591`, `origin/main` `cc1d55e`, two commits ahead (`554f3f4`, `7ee7591`, both records),
the tree clean; `git diff --stat cc1d55e HEAD -- . ':!docs'` exit 0, empty. C2: each command wrote to a scratch log,
and its exit status was taken before any filter; every test shell had no `HTTP_PROXY`, `HTTPS_PROXY` or `ALL_PROXY`
set (`env | grep -i proxy` printed nothing on macOS and the Linux machine, and `NO_PROXY` alone on the Windows laptop).
No deviation.

1. **C1** as above: `HEAD` is P0's commit.
2. **Measure.** `go.dev/dl/?mode=json` (curl exit 0): the stable releases are `go1.27.4` and `go1.26.11`, so no Go
   patch is newer than `go1.27.4`. `go version`: `go1.27.4 darwin/arm64` on macOS, `go1.27.4 linux/amd64` on the Linux
   machine, `go1.27.4 windows/amd64` on the Windows laptop, exit 0 each. No stop.
3. **macOS gates:**
   * `make preflight`: exit 0, "preflight passed": "go-precheck: 638 file(s) clean (gofmt, golint, govulncheck)", `go
     vet` for both modules, `staticcheck` 2026.2.2 for linux, darwin and windows (17 directives, 22 ignored findings,
     each recorded and paired), `golangci-lint` "0 issues." for both modules, `govulncheck` "No vulnerabilities found."
     for both modules, "check-acpext-deps: ok", `archtest` ok, "apidiff: no-op below v1.0.0", seven
     `verify-build-metadata` lines ok, `check-records`, `check-icon`, and `markdownlint` "Summary: 0 issues in 0 files".
   * `GOFLAGS=-count=1 make race`: exit 0, 39 `ok` (38 root packages and acpext), no `FAIL`, no line cached;
     `llm/provider` ok (9.757s), `permission` ok (11.346s), so `TestRuleStoreConcurrent` passed.
   * `make check-test-isolation`: exit 0, "== root module: go test exit 0; 0 failing in 0 package(s)", "== acpext
     module: go test exit 0; 0 failing in 0 package(s)", "== files written above the temp roots: 0",
     "check-test-isolation: ok".
   * `python3 scripts/check_libs.py --self-test`: exit 0, 15 lines, each `: ok`.
   * `python3 scripts/check_libs.py --lib <name> --tag <pin>`, each exit 0, "<name> advisories: ok" and "0 failing
     checks; scratch removed": acp-go-sdk2 `v1.36.0`, go-llmprovider-sdk `v1.29.0`, go-selfupdate-lib `v1.26.2`,
     go-tui-lib `v2.12.0`. The newest `vX.Y.Z` tag of each, from `git ls-remote --tags` (exit 0 each): acp-go-sdk2
     `v1.36.0`, go-llmprovider-sdk `v1.30.0` (expected and not pinned, MADR F10), go-selfupdate-lib `v1.26.2`,
     go-tui-lib `v2.12.0`.
4. **The Linux machine and the Windows laptop,** each at `go1.27.4`, from a copy made from `git ls-files -co
   --exclude-standard` at `7ee7591` (949 files, the count found on each host, no `._*` file), archived with `tar
   --no-mac-metadata`. The Linux copy was streamed over `ssh`; the Windows copy went by `scp` and was unpacked by a
   separate `ssh`, with `-o ControlPath=none` on every Windows `ssh`, and no `TMP` or `TEMP` override:

   | Host | `go test -count=1 ./...` | `CGO_ENABLED=1 go test -count=1 -race ./...` | `go -C acpext test -count=1 ./...` |
   | :--- | :--- | :--- | :--- |
   | the Linux machine | exit 0, 38 `ok`, no `FAIL`, none cached | exit 0, 38 `ok`, no `FAIL`, none cached | exit 0, `ok` |
   | the Windows laptop | exit 0, 38 `ok`, no `FAIL`, none cached | exit 0, 38 `ok`, no `FAIL`, none cached | exit 0, `ok` |

   `TestRuleStoreConcurrent` passed on both, as part of `permission`'s `ok`. This run's remote copies were removed
   afterwards and no process of it was left on either host; older entries under the hosts' `~/tmp` were left.
5. **The identifier scan** of the staged diff's added lines (the local account name, the development hosts' names, the
   owner's mail domain, and real-machine home and temporary paths): nothing found, exit 0.
6. **The record.** `make check-records markdownlint` and `python3 -B scripts/check_records.py --check-all`: exit 0
   each, after this record was written. Staged by path: this PLAN alone.

Acceptance: no newer Go, every host on `go1.27.4`; every gate of steps 3 and 4 exit 0; the four `check_libs.py` runs
`0 failing checks`; the scan found nothing; `git diff --stat cc1d55e HEAD -- . ':!docs'` is empty at `7ee7591`, and
P1's commit changes this record alone.

**Not done, by the PLAN.** No commit, push or tag. P2 (the push of `main`, the tag `v0.6.0` on the owner's word, the
release run and the Release) and P3 (close-out) are not started. The freeze of C6 starts at P1's commit.
