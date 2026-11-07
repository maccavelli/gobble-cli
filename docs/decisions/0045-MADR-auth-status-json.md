---
status: proposed
date: 2026-11-07
decision-makers: repository owner
consulted: "gobble-cli#13 (opened 2026-11-07 for magic-cli-remote); this repository at 3dbf756 (main, 2026-11-07): internal/cli/auth.go, credentials.go, modelsjson.go, internal/auth/resolve.go and store.go, auth_test.go, acpext_test.go, docs/guides/credentials.md"
informed: "magic-cli-remote (0179-MADR-gobble-native-acp-provider.md, D3, D4 and Q1)"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Give `gobble auth status` a `--json` form that reports every key provider's credential presence in one call

## Context and Problem Statement

magic-cli-remote ("Magic") is adding gobble as a provider (magic-cli-remote `0179-MADR-gobble-native-acp-provider.md`).
Its phone settings screen shows, for each of gobble's key providers, whether a credential is there and where it comes
from, never the credential. The phone asks on every `providers.list`, which it sends often, and before any gobble
session exists. [gobble-cli#13](https://github.com/maccavelli/gobble-cli/issues/13) asks for one machine-readable call
that answers this, and suggests three ways to provide it.

The question for this record is what that call is, what it reports, and what it promises to keep stable.

### What exists at `3dbf756`

* **`gobble auth status`** (`internal/cli/auth.go:323-343`) prints a table: each `provider.KeyProviders()` id and
  `credentialSource(id)` (`:345-367`).
  * The cells are prose, for example `auth.json, an OAuth login, expires 2026-11-07 10:00` or `GEMINI_API_KEY`.
  * `credentialSource` reads the auth.json entry first: `auth.json` for an `api_key`, an OAuth login with its expiry,
    "not used (0005-PLAN F4-3b)" for an OAuth entry of a provider that has no login, or a type gobble does not use.
    Then it uses `auth.Resolver.Has` (`internal/auth/resolve.go:98-119`), which gives `models.json`, the variable's
    name, or nothing.
  * `Has` never runs a command. A `!command` key counts as present without being run. A models.json key whose
    variables are unset falls through to the provider's variable.
  * `TestAuthStatus` (`internal/cli/auth_test.go:76-105`) pins the text, and checks that no value appears.
* **`gobble auth check --provider <id> --json`** (`:377-454`) is Pi's check: `{status, provider, reason, authType}`,
  exit 0, 1 or 2. It answers for one provider per process, and it goes through `Resolver.Credential`, which runs a
  stored key's `!command`. Covering the nine key providers takes nine processes.
* **A file that cannot be read is a warning, and is then ignored.**
  * `newResolver` (`internal/cli/credentials.go:43-62`) logs `auth.json: <error>; the file is not used`.
  * `readModelsJSON` (`internal/cli/modelsjson.go:52-60`) logs `models.json cannot be read, so it is ignored`.
  * Both go to stderr and the log file. A caller that reads only stdout sees `none`, or a variable, where an
    unreadable file was. It cannot tell an unreadable file from no credential.
* **The JSON precedent in the `auth` group.**
  * `gobble auth login --json` (`:60-88`; 0005-PLAN F4-3c decision 6) writes events alone on stdout. It discards stderr
    and turns off log lines to it, and the log file keeps them. A failure is an `error` event on stdout, with the exit
    code unchanged. Its `expires` is RFC 3339, in UTC.
  * `auth check --json` indents by two spaces.
* **The order.** `acpext.AuthProviders()` is `KeyProviders()`'s order, and `internal/cli/acpext_test.go:105` holds the
  two equal.
* **`ensureSettingsFile`** runs first in every `auth` command (`internal/cli/settings.go:163-181`). On a first run it
  creates `settings.yaml` from the template, and a failure is a warning.
* **Exit codes** (`internal/cli/exit.go`): 0 ok, 1 failure, 2 usage, 3 auth. `auth check` uses 2 for Pi's `invalid`.

## Decision Drivers

* **One process answers for every key provider,** in `KeyProviders()` order.
* **Presence only.** No key, token or any part of one. No network call, no command run, no OAuth refresh, no write to
  `auth.json`.
* **stdout carries the JSON alone, and stderr stays empty,** so a caller can parse stdout without filtering.
* **A caller can tell "no credential" from "gobble could not read its files".**
* **One truth for both forms.** The text and the JSON come from the same rows, so they cannot disagree.
* **A documented shape that a test pins,** kept stable across releases.
* **The smallest change.** No new module, no `acpext` change, no new tag beyond gobble's next release.

## Considered Options

* **A.** `gobble auth status --json`: the rows `auth status` already computes, as data.
* **B.** `gobble auth check --all --json`: Pi's check result for every key provider, as an array.
* **C.** An ACP extension request, `_gobble/auth_status`, served by `gobble acp`, with its types in `acpext`.
* **D.** No change: Magic runs `auth check --provider <id> --json` once per provider, concurrently, and caches the
  answers.

## Decision Outcome

Chosen option: **A**, because `auth status` already computes the right answer without running anything. It only needs
a data form, and that form serves a caller with no gobble session. B resolves credentials rather than reporting
presence. C needs a running `gobble acp` before the phone has any session. D costs nine processes per refresh, and
still runs key commands.

### D1. The command

* `gobble auth status --json` prints one JSON object, then a newline, on stdout, indented by two spaces as
  `auth check --json` is.
* Without `--json`, `auth status` is unchanged, byte for byte.
* It takes no provider argument. It always answers for every key provider.

### D2. The shape

```json
{
  "providers": [
    {"provider": "gemini", "source": "env", "variable": "GEMINI_API_KEY", "usable": true},
    {"provider": "openai", "source": "auth.json", "authType": "api_key", "usable": true},
    {"provider": "claude", "source": "auth.json", "authType": "oauth", "usable": false, "reason": "oauth_unused", "expires": "2026-06-09T10:13:20Z"},
    {"provider": "grok", "source": "auth.json", "authType": "oauth", "usable": true, "expires": "2026-06-09T10:13:20Z"},
    {"provider": "huggingface", "source": "models.json", "usable": true},
    {"provider": "kilo", "source": "none", "usable": false}
  ]
}
```

(Each object is shown on one line here. The command indents every field.)

* **`providers`** holds one object for each `provider.KeyProviders()` id, in that order. Ollama, which takes no key,
  is not among them.
* **`provider`** is gobble's id (`claude`, not Pi's `anthropic`).
* **`source`** is where the credential comes from: `auth.json`, `models.json`, `env` or `none`. It follows the order
  that `Has` reads in. An `auth.json` entry owns its provider, so a provider with an entry has source `auth.json` even
  when the entry is not usable.
* **`variable`** is the environment variable's name, when `source` is `env`. It is a name, never a value. The text
  form already prints it.
* **`authType`** is `api_key` or `oauth`, when `source` is `auth.json` and the entry has one of those types.
* **`usable`** is `Has`'s answer: whether gobble would use the credential. A `!command` key counts as usable without
  being run, as Pi's check counts one configured. A run can still fail if the command fails.
* **`expires`** is an OAuth entry's expiry, as RFC 3339 in UTC, as `auth login --json` writes it. It is there only when
  the entry records one. An expired login can still be usable, since the provider refreshes it when it runs.
* **`reason`** is a short code. It is there only when `usable` is false and `source` is not `none`:
  * `oauth_unused`: an OAuth entry for a provider gobble has no login for (0005-PLAN F4-3b);
  * `empty_key`: an `api_key` entry with no key;
  * `unknown_type`: an entry of a type gobble does not use.
* Fields that do not apply are left out, not written as `null` or `""`.

### D3. One set of rows

* A single function computes each provider's row from the run's resolver.
* The text form renders those rows into today's cells, unchanged. That includes an `api_key` entry with an empty key,
  which the text still shows as `auth.json`.
* The JSON form writes the same rows as data.
* `TestAuthStatus` passes unchanged.

### D4. What it does not do

* It sends no request, runs no `!command`, refreshes no login, and writes nothing to `auth.json`. `Has` never runs a
  command, and nothing on this path builds a provider.
* `ensureSettingsFile` still runs, as in every `auth` command.
* With `--json`, stderr and log lines to stderr are turned off, as `auth login --json` does. The log file keeps them.
  So a warning, such as a `settings.yaml` that cannot be created, cannot reach stderr.

### D5. When it cannot answer

* An `auth.json` or `models.json` that exists but cannot be read or parsed means gobble cannot say where any credential
  comes from. A config directory that cannot be resolved means the same.
* In each of these cases, `--json` writes `{"error": {"file": "<name>", "message": "<message>"}}` on stdout and exits 2.
  * The file is `auth.json`, `models.json`, or `""` for the directory.
  * The message is gobble's error for that file, which names its path and the fault, never a value.
* The text form keeps today's behaviour: a warning, then the rows as gobble will use them.
* models.json diagnostics, such as an unknown field, are not errors here. A file with diagnostics still answers.
* Exit 2 is the code `auth check` uses for Pi's `invalid`, which also means gobble could not judge.

### D6. Stability

* The shape is gobble's own. It is documented in `docs/guides/credentials.md` and pinned by a golden file,
  `internal/cli/testdata/auth-status.json.golden`.
* A field or a `source` or `reason` value may be added in a release, and a consumer ignores what it does not know.
* Renaming or removing a field or a value, or changing what one means, breaks consumers. It needs a record that says so.

### D7. Not in `acpext`

* The shape is a CLI command's output, not part of the ACP protocol. `acpext` holds ACP extension names, types and
  fixed lists (`0042-MADR-acpext-extension-contract-module.md`).
* Putting the type there would need an `acpext/v*` tag and a root `go.mod` change for a JSON object that Magic decodes
  itself.
* If gobble ever serves these rows over ACP, the type moves to `acpext` then, under a record of its own.

### D8. Release

* It ships in gobble's next `v*` release. Magic names that tag as the first version with the command, and keeps using
  `auth check` until then.
* Tagging stays the owner's, under `0043-MADR-protect-main-and-acpext-tags-without-bypass.md`.

### Consequences

* Good, because one cheap process answers what nine did. It starts no network and runs no command.
* Good, because the phone can tell an unreadable file from a missing credential.
* Good, because the text and the JSON cannot drift apart.
* Neutral, because `ensureSettingsFile` may still create `settings.yaml` on the first call, as every `auth` command
  does.
* Bad, because gobble now keeps a JSON shape stable. It is small, and D6 allows additions.
* Bad, because the two forms report an unreadable file differently: the JSON as an error, the text as a warning.

### Confirmation

1. `gobble auth status --json` prints the golden JSON for a home that holds every row kind in D2. It exits 0, with
   stderr empty.
2. No value from the fixture appears in stdout or stderr. A `!command` key's command is not run: a command that would
   create a file leaves no file.
3. `auth.json` is byte-identical after the call.
4. An unreadable or malformed `auth.json`, and an unreadable `models.json`, each give D5's error object and exit 2,
   with stderr empty and no value from the file.
5. `TestAuthStatus` passes unchanged, and the text and the JSON agree on each provider's source.
6. Each new test is seen failing on a scratch copy before the code lands.
7. `make preflight` passes, and CI passes on its three runners.
8. `docs/guides/credentials.md` documents the shape, its fields and D5.

## Pros and Cons of the Options

### A. `gobble auth status --json`

* Good, because the command already computes every row from `Has`, which runs nothing.
* Good, because it works with no gobble session and no `gobble acp` process.
* Neutral, because it adds a gobble shape, beside Pi's `check` shape.
* Bad, because the JSON's error handling (D5) differs from the text's.

### B. `gobble auth check --all --json`

* Good, because it reuses Pi's check result, which a Pi-aware client already reads.
* Bad, because `check` resolves each credential through `Credential`, which runs a stored key's `!command`. That is
  more than presence, and it can be slow or have side effects.
* Bad, because an array's exit code cannot mean what single `check`'s 0, 1 and 2 mean.

### C. `_gobble/auth_status` over ACP

* Good, because the type would sit in `acpext`, beside `AuthProviders()`.
* Bad, because it needs a running `gobble acp`, and the phone asks before any session exists.
* Bad, because the logic would still be in the CLI, since `acpext` holds no behaviour (0042-MADR). It also needs an
  `acpext/v*` tag.

### D. No change

* Good, because it costs gobble nothing.
* Bad, because every refresh starts nine processes, each running `ensureSettingsFile` and resolving a credential.
* Bad, because `check` cannot tell an unreadable `auth.json` from no credential either: the file is ignored with a
  warning on stderr.

## More Information

### Owner questions

* **Q1. `variable`.** D2 adds the variable's name, which gobble-cli#13's shape does not have. *Recommended: keep it.*
  The text form already prints it, and it is not a secret. Not chosen: leave it out.
* **Q2. The error form.** D5 writes an error object on stdout and exits 2. *Recommended:* this follows `auth login
  --json`'s error event and gobble-cli#13's exit 2. Not chosen: an error on stderr with stdout empty, which breaks
  "stderr stays empty"; or rows as gobble would use them, which hides the unreadable file.

*Answered 2026-11-07: "Q1. as recommended. Q2. as recommended." D2 keeps `variable`, and D5 stands as written.*

### Not decided here

* Whether `auth check` gains `--all`. Nothing needs it once A lands.
* A JSON form for any other command.

### Related records and issues

* [0045-PLAN-auth-status-json.md](0045-PLAN-auth-status-json.md): this record's plan.
* [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F4-3a (the `auth` commands, Pi's resolution
  order), F4-3b (OAuth entries and logins), F4-3c decision 6 (`auth login --json`).
* [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md): why the shape is not
  in `acpext`.
* [0043-MADR-protect-main-and-acpext-tags-without-bypass.md](0043-MADR-protect-main-and-acpext-tags-without-bypass.md):
  the release it ships in.
* [gobble-cli#13](https://github.com/maccavelli/gobble-cli/issues/13): the request.
* magic-cli-remote `0179-MADR-gobble-native-acp-provider.md`: the consumer, its D3, D4 and Q1.
