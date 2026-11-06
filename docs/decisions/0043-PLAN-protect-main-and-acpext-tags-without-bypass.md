---
status: in-progress
date: 2026-11-06
associated-madr: "0043-MADR-protect-main-and-acpext-tags-without-bypass.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement protecting `main` and acpext's tags with rulesets that have no bypass, and `v*` release tags with immutable releases

Associated MADR: [0043-MADR-protect-main-and-acpext-tags-without-bypass.md](0043-MADR-protect-main-and-acpext-tags-without-bypass.md)

## Goal

gobble-cli has two rulesets, both with no bypass actor, and no other:

* `gobble-cli-main`, which refuses a force-push to `main` or its deletion (MADR D2);
* `gobble-cli-acpext-tags`, which refuses moving or deleting an `acpext/v*` tag (D3).

The `v*` tag ruleset `25481802` is gone (D1). Immutable releases stay on (D4). Each rule was seen refusing the
owner's own push on a probe ref first (D7). The guide states D3's and D5's rules (D8). The plan is complete when the
MADR's Confirmation 1 to 7 is met. Confirmation 8 is the next release's, and is carried by the guide.

## Scope

**Base.** `main` at `2487444`, equal to `origin/main`. The live state is the MADR's "What is there today", read on
2026-11-06.

**In scope:**

* the repository's rulesets, through `gh api` (P1's temporary ones, and P2's switch);
* probe refs under `refs/tags/ruleset-probe/` and `refs/heads/ruleset-probe/`, pushed and removed in P1;
* `docs/guides/developing.md`, a new section;
* `docs/README.md`, the index rows and an annotation at `:169-170`;
* `docs/decisions/0004-PLAN-go-module-architecture.md`, two dated notes;
* this pair.

**Out of scope** (the MADR's "Not decided here"): the Actions policy, the owner's 2FA, and required checks or pull
requests on `main`. No Go file, workflow, Makefile or `go.mod` changes. No real ref is moved, deleted or rewritten.

## Conventions every phase follows

* **C1. Start state.** Each phase first re-reads what it relies on: `git status`, `HEAD`, `origin/main`, and the live
  rulesets. A difference from what this plan states is a deviation.
* **C2. Owner's word.** Every live change (a ruleset created or deleted, a ref pushed or deleted) is made only on the
  owner's word for that phase, in the same turn. A push carries the word for it, as `AGENTS.md` asks for pushes and
  tags.
* **C3. Evidence.** Every API response and push output is written to a scratch log, its exit status taken first. The
  phase's record quotes the lines it relies on.
* **C4. No real ref is touched by a probe.** A probe ref is under `ruleset-probe/`. A temporary ruleset includes only
  `ruleset-probe/` patterns. A command that names any other ref is a deviation.
* **C5. Deviations stop the phase,** and are recorded here, and in the MADR when a decision or fact changes, before
  work goes on.
* **C6. Seen failing.** The confirmation checker (P1 step 1) is seen failing on planted inputs before it is trusted.
* **C7. Records and commits.** `make check-records markdownlint` passes, and the identifier scan finds nothing, before
  each commit. `git commit --no-edit`, on the owner's word naming `main`.
* **C8. Identifiers.** No hostname, account name or real-machine path in a committed file. Ruleset ids and API paths
  are fine.

## Implementation Steps

### P0 — the records

Files: this pair; `docs/README.md`.

1. `docs/README.md` gains the two 0043 rows, `proposed`, and an "I want to…" row for protecting refs.
2. `make check-records markdownlint` passes; the identifier scan finds nothing.
3. Committed on the owner's word.

### P1 — the probes (MADR D7)

**Gate:** P0 committed, and the owner's word for P1, which covers its pushes.

Files: this plan.

1. **The confirmation checker,** a scratch Python script (`check_rulesets.py`), never committed. It takes the
   rulesets either from the live API or from saved JSON files, and asserts the MADR's Confirmation 1 to 4:
   * the ruleset names are exactly `gobble-cli-main` and `gobble-cli-acpext-tags`;
   * each has `bypass_actors: []`, and the owner reads `current_user_can_bypass: never`;
   * `gobble-cli-main` targets `~DEFAULT_BRANCH` with exactly `deletion` and `non_fast_forward`, and
     `rules/branches/main` lists both;
   * `gobble-cli-acpext-tags` includes only `refs/tags/acpext/v*`, with exactly `update` and `deletion`;
   * `immutable-releases` is `enabled: true`.

   It exits 1 on any other state, naming what differs. **Seen failing (C6)** on saved JSON:
   * today's live state, which has ruleset `25481802`;
   * the target state with one bypass actor planted;
   * the target state with `non_fast_forward` added to the acpext ruleset;
   * the target state with the acpext pattern `refs/tags/acpext/**`;
   * the target state with immutable releases off.

   It passes on the target state as the MADR writes it.
2. **Start state (C1).** ~~`git rev-parse HEAD origin/main` equal;~~ `GET …/rulesets` lists only `25481802`;
   `git ls-remote origin 'refs/*/ruleset-probe/*'` lists nothing. Record two commits for the probes: `A`,
   ~~`HEAD~1`~~ `origin/main~1`, and `B`, ~~`HEAD`~~ `origin/main`. `B` descends from `A`. *(Deviation 1: both
   commits are already on GitHub, so no probe push sends a new commit.)*
3. **The temporary tag ruleset,** `gobble-cli-probe-tags`, with the acpext ruleset's rules, no bypass, and a pattern
   of the acpext form:

   ```json
   {
     "name": "gobble-cli-probe-tags",
     "target": "tag",
     "enforcement": "active",
     "bypass_actors": [],
     "conditions": {"ref_name": {"include": ["refs/tags/ruleset-probe/v*"], "exclude": []}},
     "rules": [{"type": "update"}, {"type": "deletion"}]
   }
   ```

   `gh api --method POST repos/maccavelli/gobble-cli/rulesets --input <file>`. Record its id and
   `current_user_can_bypass`, which must be `never`.
4. **The tag probes,** each a `git push` of a refspec, so no local tag is made:
   1. `git push origin A:refs/tags/ruleset-probe/v0.0.1`. **Expected: accepted**, with no "Bypassed rule violations"
      line: there is no `creation` rule (D3).
   2. `git push origin A:refs/tags/ruleset-probe/x/v0.0.1`. **Expected: accepted.** The pattern does not match it,
      since `*` does not match `/`.
   3. `git push --force origin B:refs/tags/ruleset-probe/v0.0.1`, a move to a descendant commit. **Expected:
      refused** by `update`. This is the move `non_fast_forward` would not refuse (MADR D3).
   4. `git push origin :refs/tags/ruleset-probe/v0.0.1`. **Expected: refused** by `deletion`.
   5. `git push origin :refs/tags/ruleset-probe/x/v0.0.1`. **Expected: accepted**, since the pattern does not match
      it.
   6. `git ls-remote origin 'refs/tags/ruleset-probe/*'` lists `ruleset-probe/v0.0.1` at `A`.
5. **The temporary branch ruleset,** `gobble-cli-probe-branch`, with the `main` ruleset's rules and no bypass:

   ```json
   {
     "name": "gobble-cli-probe-branch",
     "target": "branch",
     "enforcement": "active",
     "bypass_actors": [],
     "conditions": {"ref_name": {"include": ["refs/heads/ruleset-probe/*"], "exclude": []}},
     "rules": [{"type": "deletion"}, {"type": "non_fast_forward"}]
   }
   ```

   Created as in step 3; its `current_user_can_bypass` must be `never`.
6. **The branch probes:**
   1. `git push origin A:refs/heads/ruleset-probe/main`. **Expected: accepted** (a creation; no rule restricts it).
   2. `git push origin B:refs/heads/ruleset-probe/main`, a fast-forward. **Expected: accepted.** gobble-cli's pushes
      to `main` are all of this kind.
   3. `git push --force origin A:refs/heads/ruleset-probe/main`, a rewrite. **Expected: refused** by
      `non_fast_forward`.
   4. `git push origin :refs/heads/ruleset-probe/main`. **Expected: refused** by `deletion`.
   5. `git ls-remote origin 'refs/heads/ruleset-probe/*'` lists `ruleset-probe/main` at `B`.
   6. No workflow run started for any probe ref: `gh run list` shows none for a `ruleset-probe/` ref.
7. **The rule suites,** `GET …/rulesets/rule-suites?ref=<ref>&time_period=hour` for each probe ref. Expected: each
   refused push has the result `fail`, each accepted one `pass`, and none has `bypass`. Each is quoted.
8. **Cleanup,** in this order, since the rules refuse deleting their own refs:
   1. `DELETE …/rulesets/<probe-tags id>`, then `DELETE …/rulesets/<probe-branch id>`;
   2. `git push origin :refs/tags/ruleset-probe/v0.0.1 :refs/heads/ruleset-probe/main`, now accepted;
   3. `git ls-remote origin 'refs/*/ruleset-probe/*'` lists nothing, and `GET …/rulesets` lists only `25481802`.
9. **What a result means.**
   * Every expectation met: D7 holds, and P2 may follow.
   * A refusal that does not happen, or an accepted push that is refused, is a deviation. P2 does not start, and the
     MADR is revisited.
   * Cleanup is done in either case.
10. **The record.** The checker's seen-failing runs, each probe's push output and rule suite, the ids, and the cleanup.
    Committed on the owner's word.

### P2 — the switch (MADR D1 to D4)

**Gate:** P1 complete with every expectation met, and the owner's word for P2.

Files: this plan.

1. **Start state (C1).** As P1 step 2, and the checker against the live API fails, naming ruleset `25481802`.
2. **Save the ruleset being deleted.** `GET …/rulesets/25481802` is written to a scratch file, and its body is
   recorded in the record, for Rollback. It must match 0004-PLAN's record (`:1310-1315`).
3. **Create `gobble-cli-main`** from the MADR's D2 JSON. Record its id, and that `current_user_can_bypass` is
   `never`.
4. **Create `gobble-cli-acpext-tags`** from D3's JSON. Record likewise.
5. **Delete `25481802`:** `gh api --method DELETE repos/maccavelli/gobble-cli/rulesets/25481802`, expecting 204. The
   new rulesets come first, so `main` and acpext's tags are never without their rule.
6. **Confirm** (Confirmation 1 to 4): the checker against the live API passes. `GET …/rules/branches/main` lists
   `deletion` and `non_fast_forward`.
7. **The record,** then the commit, then the push to `main`, each on the owner's word. The push is a fast-forward, so
   `gobble-cli-main` lets it through.
8. **Confirmation 6:** `GET …/rulesets/rule-suites?ref=refs/heads/main&time_period=hour` shows that push with the
   result `pass`, and `gobble-cli-main` among its evaluated rulesets. P3's record quotes it.

### P3 — the documents and close-out (MADR D5, D6, D8)

**Gate:** P2 pushed.

Files: `docs/guides/developing.md`, `docs/README.md`, `docs/decisions/0004-PLAN-go-module-architecture.md`, this pair.

1. **`docs/guides/developing.md`** gains "## Releasing", after "CI and the release rehearsal":
   * which refs are protected, and by what: `main` by `gobble-cli-main`; acpext's tags by `gobble-cli-acpext-tags`;
     each published `v*` release's tag by immutable releases; no rule has a bypass;
   * tagging a release: an annotated `v*` tag on `main`'s head, which fires no rule; an `acpext/vX.Y.Z` tag, as 0042-MADR
     D5 orders it;
   * recovering a failed `v*` release (MADR D5), with the `@v/list` check as a command and why not `.info`;
   * a wrong acpext tag: it stays; the next patch, with a `retract` if needed (D3);
   * after each `v*` release, check Confirmation 8: no "Bypassed rule violations", no `bypass` rule suite, and the
     release `immutable: true`;
   * the conditions under which go-selfupdate-lib's advice applies again (D6).
2. **`docs/README.md`:**
   * `:169-170`, "A tag ruleset now restricts `v*` to the repository admin role, the owner.", gets an italic note
     after it: superseded on 2026-11-06 by 0043-MADR, the ruleset deleted, `main` and acpext's tags protected with no
     bypass. The sentence stays.
   * the two 0043 rows give the status.
3. **0004-PLAN** gets two dated italic notes, its text above them unchanged:
   * after the precondition "a tag ruleset restricts `v*` to the owner, with 2FA" (`:604`): superseded by 0043-MADR
     D1; immutable releases are the protection for published `v*` tags;
   * at the end of the entry of 2026-10-20 on the tag ruleset (`:1307-1325`): the ruleset was deleted on the date of
     P2, its body recorded in 0043-PLAN.
4. **The MADR** becomes `accepted`; this plan `complete`, with P3's record.
5. **Checks (C7)** pass. Committed and pushed on the owner's word. That push's rule suite is `pass`, as P2 step 8's.
6. **gobble-cli#12** gets a comment on the owner's word: what was done, each Confirmation with its evidence, and that
   Confirmation 8 waits for the next release. It is closed on the owner's word.

## Verification

| MADR Confirmation | Phase | Check |
| :--- | :--- | :--- |
| 1 exactly two rulesets, no bypass, `never` | P2 | step 6, the checker |
| 2 `main`'s rules | P2 | step 6 |
| 3 the acpext ruleset's pattern and rules | P2 | step 6, the checker |
| 4 immutable releases on | P2 | step 6, the checker |
| 5 each rule seen refusing the owner; a creation with no rule firing | P1 | steps 4, 6 and 7 |
| 6 a push to `main` passes `gobble-cli-main` | P2, P3 | P2 step 8; P3 step 5 |
| 7 the guide and the records | P3 | steps 1 to 3 |
| 8 the next release: no bypass, immutable | after P3 | the guide's release check (P3 step 1) |

## Rollout and Rollback

* **P1 leaves nothing behind.** Its rulesets and refs are deleted in step 8, whatever the probes showed. If cleanup
  fails partway, the record names what is left; a probe ref is deleted after its ruleset, by the same commands.
* **P2 is undone by three calls,** none touching a ref:
  * `DELETE …/rulesets/<gobble-cli-main id>` and `DELETE …/rulesets/<gobble-cli-acpext-tags id>`;
  * `POST …/rulesets` with the body P2 step 2 saved, which re-creates the `v*` ruleset with its Admin bypass. Its
    new id differs from `25481802`.
* **P3 is undone** by reverting its commit. The rulesets are not affected.
* **User-visible:** none. No command, output, configuration or release asset changes.
* **For the owner:**
  * a force-push to `main`, or deleting it, is refused;
  * an acpext tag cannot be moved or deleted;
  * the next `v*` tag push fires no rule.
* **For an agent:** the same. A force-push it is asked to make is refused by GitHub, not left to its own care.

## Execution record

Each phase appends its record here.

### P0 record (2026-11-06): the records

The pair and `docs/README.md`'s rows, committed as `ed06112` on the owner's "Commit the records to main then do
p1". `make check-records markdownlint` passed, 0 issues; the identifier scan found nothing. Not pushed.

### Deviation 1 (2026-11-06): P1's probe commits come from `origin/main`

- **Found** at P1 step 2. The step expects `HEAD` to equal `origin/main`, and takes `A` as `HEAD~1` and `B` as
  `HEAD`. After P0, `HEAD` is `ed06112`, one commit ahead of `origin/main` (`2487444`) and unpushed. Pushing
  `HEAD` to a probe ref would send that commit to GitHub.
- **Decision,** the owner's ("Probe on origin/main"): `A` is `origin/main~1` (`14201ed`) and `B` is
  `origin/main` (`2487444`), both already on GitHub. `B` descends from `A`, which is all the probes need. No
  new commit leaves the machine. P0's commit stays local until the owner pushes it.
- **Files added to scope:** none. The MADR does not change.

### P1 record (2026-11-06): the probes

On the owner's "Commit the records to main then do p1". The base is `ed06112`, unpushed; the probes used `origin/main`
(deviation 1). Every expectation was met: 18 of 18.

1. **The checker** (step 1), `check_rulesets.py`, scratch. Seen failing (C6), each exit 1, naming what differs:
   - today's live state: "rulesets are ['release tags: owner only'] (ids [25481802]), want exactly gobble-cli-main and
     gobble-cli-acpext-tags", and "rules/branches/main lists [], want deletion and non_fast_forward";
   - a bypass actor on `gobble-cli-main`: "bypass_actors [{'actor_id': 5, …}], want []", and
     "current_user_can_bypass 'always', want 'never'";
   - `non_fast_forward` added to the acpext ruleset: "rules ['deletion', 'non_fast_forward', 'update'], want
     ['deletion', 'update']";
   - the pattern `refs/tags/acpext/**`: "ref_name {'include': ['refs/tags/acpext/**'], …}, want include
     ['refs/tags/acpext/v*']";
   - immutable releases off: "immutable-releases {'enabled': False, …}, want enabled true".

   On the target state as the MADR writes it: exit 0.
2. **Start state** (step 2): `GET …/rulesets` listed only `25481802`; `git ls-remote origin 'refs/*/ruleset-probe/*'`
   listed nothing. `A` is `14201ed`, `B` is `2487444`, and `B` descends from `A`.
3. **The temporary tag ruleset** (step 3): `gobble-cli-probe-tags`, id `26714934`, `bypass_actors: []`,
   `current_user_can_bypass: never`.
4. **The tag probes** (step 4), each from the owner's account:
   - 4.1, `A` to `refs/tags/ruleset-probe/v0.0.1`: accepted, `[new tag]`, no "Bypassed rule violations";
   - 4.2, `A` to `refs/tags/ruleset-probe/x/v0.0.1`: accepted;
   - 4.3, a force-push of `B`, a descendant, to the probe tag: **refused**, `remote: error: GH013: Repository rule
     violations found for refs/tags/ruleset-probe/v0.0.1.` and `- Cannot update this protected ref.`, `! [remote
     rejected] … (push declined due to repository rule violations)`;
   - 4.4, deleting the probe tag: **refused**, `GH013` and `- Cannot delete this tag`;
   - 4.5, deleting `ruleset-probe/x/v0.0.1`: accepted, so `*` did not match `/`;
   - 4.6, `ls-remote` listed the probe tag still at `A`.
5. **The temporary branch ruleset** (step 5): `gobble-cli-probe-branch`, id `26714949`, `bypass_actors: []`,
   `current_user_can_bypass: never`.
6. **The branch probes** (step 6):
   - 6.1, creating `refs/heads/ruleset-probe/main` at `A`: accepted;
   - 6.2, a fast-forward to `B`: accepted, `14201ed..2487444`;
   - 6.3, a force-push back to `A`: **refused**, `GH013` and `- Cannot force-push to this branch`;
   - 6.4, deleting it: **refused**, `GH013` and `- Cannot delete this branch`;
   - 6.5, `ls-remote` listed it still at `B`;
   - 6.6, no workflow run had a `ruleset-probe/` ref.
7. **The rule suites** (step 7), `time_period=hour`:
   - the probe tag: `0000000->14201ed` `pass`, `14201ed->2487444` `fail`, `14201ed->0000000` `fail`;
   - `ruleset-probe/x/v0.0.1`: none, since no ruleset matched it;
   - the probe branch: `0000000->14201ed` `pass`, `14201ed->2487444` `pass`, `2487444->14201ed` `fail`,
     `2487444->0000000` `fail`;
   - no suite has the result `bypass`.
8. **Cleanup** (step 8): `DELETE` of `26714934` and `26714949`, exit 0; then both probe refs deleted, exit 0. After it,
   `ls-remote` listed no probe ref and `GET …/rulesets` listed `[25481802]`.
9. **What it shows** (step 9): with no bypass actor, `update`, `deletion` and `non_fast_forward` each refused the
   owner, an administrator; `update` refused a move to a descendant commit; a creation with no `creation` rule fired
   nothing; and the acpext form of pattern matched one level only. MADR D7 holds, so P2 may follow on the owner's
   word.
10. **Not done:** the commit of this record, and the push of `ed06112` and it, each on the owner's word.
