---
status: accepted
date: 2026-11-06
decision-makers: repository owner
consulted: "the repository's rulesets, rule suites, collaborators, invitations, deploy keys, environments, releases, immutable-release setting and Actions permissions (GitHub API, read 2026-11-06); proxy.golang.org's version lists; GitHub's documentation on rulesets and immutable releases (read 2026-11-06); prepare-commit-msg's 0025-MADR and 0025-PLAN, Amendment A1 and Phase 4; go-selfupdate-lib v1.24.0's building-releases guide and publish workflow"
informed: "gobble-cli#12; prepare-commit-msg (fleet consistency)"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Protect `main` and acpext's tags with rulesets that have no bypass, and `v*` release tags with immutable releases

## Context and Problem Statement

gobble-cli has one ruleset, made by hand on 2026-10-20 at the owner's word, "restrict pushing v* tags to me, the repo
owner" (`docs/decisions/0004-PLAN-go-module-architecture.md`, its entry "Phase 4's release precondition, met after
the first tags", `:1307-1325`). gobble-cli#12 asks to replace it, as prepare-commit-msg did on 2026-11-06. The question
is which refs gobble-cli protects, with what rules, and who may bypass them.

### What is there today

Read with `gh api` and `git` on 2026-11-06.

* **The ruleset.** One, id `25481802`, "release tags: owner only":
  * `target: tag`, `enforcement: active`, `ref_name` including `refs/tags/v*` and excluding nothing;
  * the rules `creation`, `update`, `deletion` and `non_fast_forward`;
  * one bypass actor, `RepositoryRole` `5` (the repository admin role), `bypass_mode: always`; the owner's token reads
    `current_user_can_bypass: always`.
* **`main`.** No ruleset and no branch protection: `GET …/branches/main/protection` is 404 "Branch not protected", and
  `GET …/rules/branches/main` lists 0 rules. The repository is public, its default branch `main`.
* **Who can write.** One collaborator, the owner, with the role `admin`. No invitation, no deploy key, no GitHub App
  installation that reaches the repository, no environment.
* **Workflow tokens.** `default_workflow_permissions: read`, and `ci.yml` sets `contents: read` for every job but
  `release` (`ci.yml:32-33`). `release` holds `contents: write`, `id-token: write` and `attestations: write`
  (`ci.yml:176-183`). It calls go-selfupdate-lib's publish workflow, which runs `gh release create "$TAG" --draft
  --verify-tag --generate-notes` (`publish-selfupdate-release.yml:358-371` at `v1.24.0`). `--verify-tag` requires
  the tag to exist, so no workflow creates, moves or deletes a tag. A tag a workflow's `GITHUB_TOKEN` pushes starts no
  workflow run.
* **What triggers CI.** `ci.yml` runs on pushes to `main` and on `v*` tags; `docs.yml` on pushes to `main`. Neither
  runs on any other branch or tag.
* **Tags.** `v0.1.0`, whose run failed `verify-build-metadata` and published nothing; `v0.1.1`, released on
  2026-10-20; `acpext/v0.1.0`, pushed on 2026-11-06 by `0042-PLAN-acpext-extension-contract-module.md`'s P1.
* **Releases.** One, `v0.1.1`, `immutable: true`. `acpext/v0.1.0` has none: acpext's versions are module tags, and
  0042 publishes no GitHub release for them.
* **Immutable releases.** `GET …/immutable-releases`: `{"enabled":true,"enforced_by_owner":false}`.
* **The module proxy.** `proxy.golang.org`'s `@v/list` lists `v0.1.0` and `v0.1.1` for the root, and `v0.1.0` for
  `github.com/maccavelli/gobble-cli/acpext`. Every version it lists has its content fixed in the checksum database.
* **Rule suites.** `GET …/rulesets/rule-suites?time_period=month` returns 0 evaluations: no `v*` tag has been pushed
  since the ruleset was made, and `acpext/v0.1.0` does not match `refs/tags/v*`.

### What GitHub documents

Read on 2026-11-06.

* **The rules** ([available rules for rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets)):
  * restrict creations: "only users with bypass permissions can create branches or tags whose name matches the
    pattern you specify";
  * restrict updates: "only users with bypass permissions can push to branches or tags whose name matches";
  * restrict deletions: "only users with bypass permissions can delete branches or tags whose name matches";
  * block force pushes: "You can prevent users from force pushing to the targeted branches or tags."
* **Patterns** ([creating rulesets for a repository](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository)):
  `fnmatch`, and "the `*` wildcard does not match directory separators (`/`)". So `refs/tags/v*` cannot match
  `refs/tags/acpext/v0.1.0`, whatever its first letter.
* **Bypass** ([about rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)):
  "you can allow certain users to bypass the rules in the ruleset", repository administrators among the actors that
  may be listed. Whether an administrator who is **not** listed is bound is not stated.
* **Immutable releases** ([immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases)):
  * "Once an immutable release is published, its associated Git tag is locked to a specific commit, cannot be
    changed, and cannot be deleted while the release exists."
  * "If you delete the immutable release, you can delete the tag, but you cannot reuse the same tag name."
  * Immutability "will only apply to future releases"
    ([preventing changes to your releases](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/establish-provenance-and-integrity/prevent-release-changes)).
  * Nothing is said of a tag with no release, or of whether administrators are exempt.

### What prepare-commit-msg found

prepare-commit-msg's `docs/decisions/0025-MADR-remove-ruleset-bypass-actors.md` (accepted 2026-11-06), on the same
owner's account:

* Its rule suites showed every release tag push from `v1.8.0` to `v1.10.3` with the result `bypass`, each failing
  `creation` and passing the rest. So the owner, an administrator, is evaluated by a ruleset like any other actor, and
  passes `creation` only through the bypass. That is the evidence that a rule with no bypass actor binds the owner.
* `non_fast_forward` "does not refuse a move to a descendant commit"; only `update` refuses every move of a tag (its
  D2).
* Amendment A1 dropped its tag ruleset and kept a `main` ruleset with no bypass, relying on immutable releases for
  release tags. Its PLAN's Deviation 3 narrowed failed-release recovery: an unreleased tag may be reused only if the
  proxy's `@v/list` does not list its version, and `.info` must not be used to check, since asking for it can make the
  proxy fetch that version.
* It did not probe a refusal live: "A probe against a real ref would delete or rewrite it if the rule did not hold".

### The problem

* **The bypass fires on every release and restricts no one.** `creation` with an Admin bypass refuses every actor but
  the owner, who is the only one that can write. Every `v*` push gets through by bypassing it, and git prints
  "Bypassed rule violations". prepare-commit-msg saw it on six releases; gobble-cli has pushed no `v*` tag since,
  so its next release would be the first.
* **Removing the bypass alone stops every release,** since `creation` then refuses every `v*` tag.
* **Published release tags are already locked** by immutable releases. The tag ruleset adds protection only to `v*`
  tags without a published release, such as `v0.1.0`.
* **`main` can be force-pushed or deleted** by the owner, by an agent working under the owner's login, or by a
  workflow token given `contents: write`. Nothing refuses it.
* **acpext's tags have no protection at all.** The tag ruleset does not match them, and immutable releases do not
  reach them, since they have no release. A module version, once the proxy lists it, must never move: the checksum
  database has fixed its content, so the same version on another commit fails verification for anyone fetching
  outside the proxy, and the tag then misnames what `go` serves.

## Decision Drivers

* **No bypass actor.** A rule binds everyone, the owner and agents under the owner's login included. A push the rules
  would refuse is refused, not reported as bypassed.
* **Releases keep working:** the owner pushes an annotated `v*` or `acpext/v*` tag, and no rule fires on it.
* **`main` cannot be deleted or force-pushed.**
* **A published version never moves.** A `v*` tag with a published release, and every acpext tag, stays on its
  commit.
* **A failed release can still be recovered,** as far as the module proxy allows.
* **Each protection is proven,** on refs where a failure of the rule can harm nothing.
* **Fleet consistency** with prepare-commit-msg's 0025-MADR A1, where gobble-cli's facts are the same.

## Considered Options

* **A. Keep the `v*` tag ruleset with its Admin bypass.**
* **B. Keep it, and remove the bypass.**
* **C. gobble-cli#12 as written:** delete the tag ruleset, add a `main` ruleset with no bypass, and rely on immutable
  releases. acpext's tags stay unprotected.
* **D. C, and a tag ruleset for `acpext/v*` with `update` and `deletion`, no `creation`, and no bypass.**
* **E. C, and one tag ruleset for both `v*` and `acpext/v*`,** with `update` and `deletion`, no `creation`, and no
  bypass.
* **F. C, and a GitHub release for each acpext tag,** so that immutable releases lock it.

## Decision Outcome

Chosen option: **"D"**, because it is the only option in which:

* no actor has a bypass;
* releases still work;
* `main` and every acpext tag are protected;
* a failed `v*` release can still be recovered where the proxy allows it;
* gobble-cli's `v*` handling matches prepare-commit-msg's A1.

### D1. The `v*` tag ruleset goes

Ruleset `25481802` is deleted. A published `v*` release's tag is locked by immutable releases, which stay on (D4).
This supersedes 0004-PLAN Phase 4's precondition "a tag ruleset restricts `v*` to the owner" (`:604`), and the
ruleset its entry of 2026-10-20 records (`:1307-1325`).

### D2. A `main` ruleset with no bypass

A branch ruleset, `gobble-cli-main`:

```json
{
  "name": "gobble-cli-main",
  "target": "branch",
  "enforcement": "active",
  "bypass_actors": [],
  "conditions": {"ref_name": {"include": ["~DEFAULT_BRANCH"], "exclude": []}},
  "rules": [{"type": "deletion"}, {"type": "non_fast_forward"}]
}
```

* It refuses a force-push to `main` and its deletion, from every actor. A fast-forward push, gobble-cli's only kind,
  is not refused.
* `deletion` may duplicate GitHub's own handling of a default branch. The documentation says only to "choose a new
  default branch first", and does not say a default branch cannot be deleted. It costs nothing to keep.
* Blocking force pushes also stops an administrator from renaming the default branch without a bypass ([available
  rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets)).
  Renaming `main` would then need the ruleset edited first.

### D3. A ruleset for acpext's tags, with no bypass

A tag ruleset, `gobble-cli-acpext-tags`:

```json
{
  "name": "gobble-cli-acpext-tags",
  "target": "tag",
  "enforcement": "active",
  "bypass_actors": [],
  "conditions": {"ref_name": {"include": ["refs/tags/acpext/v*"], "exclude": []}},
  "rules": [{"type": "update"}, {"type": "deletion"}]
}
```

* **`update`** refuses every move of an acpext tag, a move to a descendant commit included, which `non_fast_forward`
  would not refuse. So `non_fast_forward` is not added: it refuses nothing `update` lets through.
* **`deletion`** refuses deleting one.
* **No `creation` rule,** so a new acpext tag is pushed with no rule firing, and no bypass is needed.
* **The pattern.** `*` does not match `/`, so `refs/tags/acpext/v*` matches `acpext/v0.1.0` and every
  `acpext/vX.Y.Z` tag. It matches no `v*` tag of the root.
* **Why acpext and not `v*`.** Every acpext tag is a module version the proxy lists within minutes. 0042-PLAN P1 step
  11 asks the proxy for it right after the push. So an acpext tag is never recoverable by deleting it, and the rule
  costs nothing a maintainer could use. A `v*` tag whose run failed may still be unlisted (D5); the rule would take
  that recovery away.
* **A wrong acpext tag** is fixed as 0042-PLAN's Rollback already says: the next patch version, with a `retract` in
  acpext's `go.mod` if the bad version must not be used.

### D4. Immutable releases stay on

The setting is left as it is. It is what locks each published `v*` release's tag. It is a repository setting, not a
ruleset, so this record changes nothing of it, and its Confirmation reads it.

### D5. Recovering a failed `v*` release

As prepare-commit-msg's A1-3 is narrowed by its Deviation 3:

* **A `v*` tag whose release was never published** may be deleted and pushed again on the fixed commit, but only if
  `https://proxy.golang.org/github.com/maccavelli/gobble-cli/@v/list` does not list that version. Check with
  `@v/list` alone: asking for the version's `.info`, `.mod` or `.zip` can make the proxy fetch it, which fixes it.
* **If the proxy lists it,** the tag stays, and the fix takes the next patch version. `v0.1.0` is listed, so it stays
  for good.
* **A tag whose release was published** stays; its release is immutable. The fix is the next patch version.
* **An acpext tag** stays (D3).

### D6. The departure from go-selfupdate-lib's advice

go-selfupdate-lib's `docs/guides/building-releases.md` (`:256-262` at `v1.24.0`) asks to "Restrict who can create
`v*` tags … with creations restricted to the owner or the release managers", because "The publish workflow attests
whatever a `v*` tag builds". gobble-cli departs from it:

* one account can write, and it is the owner;
* no workflow can push a tag that starts a release: a token-pushed tag starts no run, and the publish job creates no
  tag;
* restricting creation to the owner, with one write actor, excludes no one, and works only through a bypass.

If a second account ever gets write access, or a workflow is given a token that can push tags and start runs, the
advice applies again, and restricting `v*` creation needs a new record.

### D7. Proving the rules

GitHub does not document whether an administrator who is not a bypass actor is bound. prepare-commit-msg's rule
suites show that the owner is evaluated, but no refusal was observed there. So each rule this record relies on is
proven by a refusal, on temporary rulesets over refs that nothing reads:

* refs named `ruleset-probe/…`, which match neither `~DEFAULT_BRANCH`, `refs/tags/v*` nor `refs/tags/acpext/v*`, start
  no workflow (`ci.yml` and `docs.yml` run on `main` and `v*` alone), and are not module versions;
* each temporary ruleset is deleted, and then its probe refs.

The PLAN gives the probes. A rule that is not refused as this record says is a deviation, and the switch does not go
ahead.

### D8. The documents

* `docs/guides/developing.md` gains a section on releasing: which refs are protected and by what, D5's recovery, and
  D3's rule for acpext tags.
* `docs/README.md`'s line "A tag ruleset now restricts `v*` to the repository admin role, the owner" (`:169-170`) is
  annotated as superseded by this record, not rewritten.
* 0004-PLAN's precondition (`:604`) and its entry of 2026-10-20 (`:1307`) get dated notes naming this record. The
  records stay as written.

### Consequences

* Good, because no rule has a bypass actor, so no push is ever "bypassed"; a rule refuses or passes.
* Good, because the next `v*` release fires no rule: there is no `v*` ruleset.
* Good, because `main` can no longer be force-pushed or deleted, by the owner, by an agent under the owner's login, or
  by a workflow token.
* Good, because acpext's tags, which no release protects, cannot be moved or deleted, so the tag always names what the
  proxy serves for its version.
* Good, because each rule is proven by a refusal, which neither 0004-PLAN's ruleset nor prepare-commit-msg's could
  show.
* Neutral, because published `v*` tags keep the same protection, now from immutable releases alone.
* Bad, because a `v*` tag with no published release (`v0.1.0` today) can be moved or deleted by the owner or an agent
  under the owner's login.
  * Neutral, because D5 says when that is allowed, and the proxy and checksum database fix what `go` fetches for
    every listed version.
* Bad, because a wrong acpext tag cannot be deleted even before the proxy has it, short of editing the ruleset.
  * Neutral, because 0042 already fixes a bad acpext version with the next patch and a `retract`, and its release step
    puts every tag in the proxy at once.
* Bad, because renaming `main` needs the `main` ruleset edited first.
* Neutral, because each ruleset is one API call to remove or restore, and the deleted ruleset's JSON is recorded
  (0004-PLAN `:1310-1315`, and the PLAN's Rollback).
* Neutral, because gobble-cli keeps one ruleset more than prepare-commit-msg, for a module kind that repository does
  not have.

### Confirmation

1. `GET repos/maccavelli/gobble-cli/rulesets` lists exactly `gobble-cli-main` and `gobble-cli-acpext-tags`; each has
   `bypass_actors: []`, and the owner's token reads `current_user_can_bypass: never` for each.
2. `GET repos/maccavelli/gobble-cli/rules/branches/main` lists `deletion` and `non_fast_forward`.
3. `gobble-cli-acpext-tags`'s `ref_name` includes only `refs/tags/acpext/v*`, and its rules are `update` and
   `deletion`.
4. `GET repos/maccavelli/gobble-cli/immutable-releases` reports `"enabled":true`.
5. D7's probes saw a refusal of each rule from the owner's account, and a probe tag created with no rule firing.
6. After the switch, a push to `main` has a rule suite with the result `pass`, `gobble-cli-main` evaluated.
7. `docs/guides/developing.md` states D3's and D5's rules, and the records carry their notes (D8).
8. *(After this record, at the next `v*` release; not a gate on the PLAN.)* The tag push prints no "Bypassed rule
   violations", no rule suite records a `bypass`, and the release reports `immutable: true`.

## Pros and Cons of the Options

### A. Keep the `v*` tag ruleset with its Admin bypass

* Good, because nothing changes.
* Bad, because its bypass fires on every release and restricts no one.
* Bad, because `main` and acpext's tags stay unprotected.

### B. Keep it, and remove the bypass

* Good, because no actor has a bypass.
* Bad, because `creation` then refuses every `v*` tag, so no release can be made.

### C. gobble-cli#12 as written

* Good, because no actor has a bypass, releases work, and `main` is protected.
* Good, because it matches prepare-commit-msg's A1 exactly.
* Bad, because acpext's tags, which no release protects, can be moved or deleted. A moved one misnames its version.

### D. C, and a ruleset for acpext's tags

* Good, because of C's goods, and every acpext tag is protected.
* Good, because it needs no bypass: it has no `creation` rule.
* Neutral, because it is one ruleset more than prepare-commit-msg has, for tags that repository does not have.
* Bad, because a mistaken acpext tag cannot be deleted even before the proxy lists it.

### E. C, and one tag ruleset for `v*` and `acpext/v*`

* Good, because every tag is protected, `v0.1.0` included.
* Bad, because a failed `v*` release the proxy does not list can no longer be recovered by re-pushing its tag, except
  by editing the ruleset: a bypass by another name, which prepare-commit-msg's D3 and A1 rejected.
* Bad, because the `v*` half repeats what immutable releases already do for published tags.

### F. C, and a GitHub release for each acpext tag

* Good, because one mechanism, immutable releases, locks every published tag.
* Bad, because the releases page then mixes gobble's binaries with acpext's module tags, and each acpext release must
  be kept from being marked Latest.
* Bad, because publishing a release is a second manual step after each acpext tag; until it is done, the tag is
  unprotected.
* Bad, because GitHub does not say whether immutability binds an administrator, which D's probe can show for a
  ruleset.

## More Information

### Not decided here

* **The Actions policy.** `allowed_actions: all` and `sha_pinning_required: false` differ from prepare-commit-msg's.
  That is a separate decision (gobble-cli#12, "Out of scope").
* **2FA on the owner's account,** which a ruleset on a user-owned repository cannot require (0004-PLAN `:1321-1323`).
* **Required status checks or pull requests on `main`.** gobble-cli commits straight to `main`, on the owner's word.

### Related records and issues

* [gobble-cli#12](https://github.com/maccavelli/gobble-cli/issues/12), which this record answers.
* [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md): Phase 4's precondition and the ruleset's
  creation, superseded by D1.
* [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md): D5, acpext's tags,
  which D3 protects; and
  [0042-PLAN-acpext-extension-contract-module.md](0042-PLAN-acpext-extension-contract-module.md)'s Rollback, D3's
  recovery for a bad acpext version.
* prepare-commit-msg `docs/decisions/0025-MADR-remove-ruleset-bypass-actors.md`, Amendment A1, and
  `docs/decisions/0025-PLAN-remove-ruleset-bypass-actors.md`, Deviation 3.
* go-selfupdate-lib `docs/guides/building-releases.md`, the advice D6 departs from.
