---
status: accepted
date: 2026-11-14
decision-makers: repository owner
consulted: "this repository at cccef77 (main, 2026-11-14); the earlier `-race ./internal/cli/` log of the 0037 P1 session; 30 race runs of `internal/cli` across the laptop's WSL2 distro, the Linux machine, the Windows laptop and the macOS laptop, 2026-11-14, recorded under More Information; CI run 43468419809 (5147e09, 2026-11-13); 0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md's host table; the Go 1.27.4 runtime source (runtime/malloc.go, runtime/race/README) and LLVM compiler-rt's tsan_rtl.cpp and tsan_defs.h; the owner's rulings of 2026-11-14 on Q1 to Q3, relayed by the coordinator"
informed: "0037-PLAN-resume-prefix-and-failed-turn-history.md, whose P1 gate run found the failure; the fleet's dev-host routing, which owns where `-race` suites run: the dotfiles repository's 0015-MADR-load-aware-dev-host-test-dispatch.md, whose amendment is pending"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Treat the `internal/cli` race reports on the laptop's WSL2 distro as a fault of that environment, not a gobble race, and decide where `-race` runs

## Context and Problem Statement

On 2026-11-14 the 0037 P1 gate ran `make race` at `cccef77` on the laptop's WSL2 distro (`Ubuntu-24.04`), and it exited 2.
`go test -race ./internal/cli/` printed 8 or 9 `WARNING: DATA RACE` reports per run, and a different set of failing tests
on each run. `acpserver`, `agent` and `jsonx` passed. The reports name config and file reads (`fsx.ReadRegularFile`,
`config.Resolve`, YAML parsing, reached from `(*runEnv).defaultTrust`), Kong's parser, model lookups and others.

The owner's ruling, 2026-11-14: "Root-cause it first". Find out, with evidence, whether this is a product race (two
goroutines touching shared state in real use) or test-only sharing, before any fix. As relayed with it: no tolerance
tricks, so no `t.Skip`, no dropping `t.Parallel` to hide a real race, and no `-race` exclusions.

This record answers that question. The answer is neither: every report is one goroutine said to race with itself, and
the same test binary, byte for byte, reports nothing on the Linux machine. Which decision follows was the owner's; the
rulings of 2026-11-14 are under Decision Outcome.

### What the reports are

Every report read, 161 of them over 24 package runs at `cccef77` (the P1 log among them), has the same goroutine on both sides:
`Write at X by goroutine N` and `Previous write at X by goroutine N` (or `Read`). Not one report names two goroutines.
The distinct pairs, by the first gobble or library frame (`file:line` at `cccef77` and the pinned modules):

| # | Memory | Access and previous access (one goroutine) | Whose goroutine | Seen in |
| :--- | :--- | :--- | :--- | :--- |
| R1 | the `bytes.Buffer` local to `readAtMost` | `internal/fsx/regular.go:70` (`buf.ReadFrom`, the `read` syscall) after `regular.go:69` (`buf.Grow`), from `config.readLayer` (`internal/config/resolve.go:130`) and `config.Resolve` (`resolve.go:96`), reached from `(*runEnv).defaultTrust` (`internal/cli/trust.go:103`) or `resolveSettings` (`settings.go:82`) | the test's, running `Main` | `TestMainExitCodes`, `TestPrintModeTrust` |
| R2 | the YAML scanner's buffer, local to one parse | `go.yaml.in/yaml/v3` `scannerc.go:558` (`growslice`) after a write by the same scan | the test's | `TestMainExitCodes` |
| R3 | the `opts` slice local to `lookupModelWith` | `llm/provider/model.go:150` after the same line, in the same call | the test's, via `offlineModels` (`internal/cli/modelflags.go:46`) | `TestPickModel`, `TestChoiceWiring` |
| R4 | a pooled `regexp` backtracker | `regexp/backtrack.go:133` after the same line, from `kong.interpolate` in `newParser` (`internal/cli/main.go:66`) | the test's | `TestAuthLogin` |
| R5 | Kong's tag parse state | `kong` `tag.go:103` after the same line, in `kong.New` from `newParser` | the test's | `TestProjectSettingsNeedTrust` |
| R6 | a `Descriptor` copy | `go-llmprovider-sdk` `registry.go:43` (`Descriptor.clone` from `Descriptors`) after `registry.go:43` (from `Register`), in `provider.Label` (`llm/provider/ambient.go:103`) | one ACP request handler of acp-go-sdk2 | `TestToolFlagsSelectResourceTools` |
| R7 | the line editor's layout buffer | `go-tui-lib` `lineedit/layout.go:95` after the same line | the line session's, started by `lineHarness.start` (`internal/cli/line_test.go:123`) | `TestLineCompactThroughTheAgent` |
| R8 | an ML-KEM key in a TLS handshake | `crypto/internal/fips140/mlkem/mlkem768.go:421` after `mlkem768.go:128`, one `ClientHandshake` | `net/http`'s dial goroutine | `TestUpdateIgnoresArchive` |

R1 to R8 each touch memory that one goroutine allocated and that no other goroutine is handed: a local variable
(R1, R3), a parse or handshake's own state (R2, R5, R8), a value taken from a `sync.Pool` (R4), or a copy made for
the caller (R6, R7). A goroutine's own accesses are ordered by program order, so none of the 8 is a data race in Go's
memory model. None is test-only sharing either: `internal/cli` has no `t.Parallel` call (0 in 300 test functions), the
package variables the tests replace (`signInWith`, `listModels`, `modelFacts` and the rest) are set by `TestMain` or a
sequential test, and the run never reported a second goroutine on any address.

### Where it reproduces

| Environment | Binary | Runs | Red runs | Reports | Two-goroutine reports |
| :--- | :--- | :--- | :--- | :--- | :--- |
| the laptop's WSL2 distro `Ubuntu-24.04`, kernel `6.6.87.2-microsoft-standard-WSL2`, 32 threads, go1.27.4 | built there | 24 package runs | 24 | 161 | 0 |
| the same, the one test `-run '^TestPickModel$' -count=40` | built there | 3 | 3 | 19 | 0 |
| the Linux machine, kernel 7.0, 4 vCPUs, go1.27.4 | built there | 5 package runs | 0 | 0 | 0 |
| the Linux machine | **the WSL2 distro's binary, sha256 `87d6317e…`, byte for byte** | 2 package runs, 2 × `TestPickModel` × 40 | 0 | 0 | 0 |
| the Windows laptop, natively (the same CPU as the WSL2 distro), go1.27.4 | built there | 2 package runs, 2 × `TestPickModel` × 40 | 0 | 0 | 0 |
| the macOS laptop, go1.27.4 | built there | 5 package runs | 0 | 0 | 0 |
| CI, `ubuntu-24.04` and `macos-15`, run 43468419809 at `5147e09` (the same Go files as `cccef77`) | built there | 1 each | 0 | 0 | 0 |

The binary that fails on the WSL2 distro passes on the Linux machine, so the code and the build are not what differs:
the environment is. Inside the distro the failure does not depend on the scheduler: it stays with `-test.cpu=1`, with
`GODEBUG=asyncpreemptoff=1`, with the process pinned to CPUs 0-3 or to CPUs 16-31, and with go1.27.3.

### What it is not

* **Not a shadow reset.** With `GORACE=verbosity=2` the race runtime logs each reset of its shadow memory
  (`shadow_begin-shadow_end`). Two `internal/cli` runs logged none and reported 29 and 13 races. A gobble-free test that
  forced 36 resets reported none.
* **Not slot pressure.** The race runtime has 256 slots for live goroutines (`kThreadSlotCount`, tsan_defs.h). The
  package's live goroutines peaked at 65.
* **Not the Go 1.27.4 bump** (0054): go1.27.3 fails the same way, and so do older commits.
* **Not a runtime allocation path that skips the race runtime.** `mallocgc` calls `racemalloc` (runtime/malloc.go:1175);
  the size-specialised allocator is off under `-race` (malloc.go:1039); `freegc` is behind the `RuntimeFreegc`
  experiment, which is off by default in go1.27.4.

The mechanism inside the race runtime under WSL2 is not established. A gobble-free reproducer was not found: three
synthetic tests (goroutine-local work under heavy synchronisation, 1000 parked goroutines, 1000 workers) were clean on
the distro.

### Since when

Not since a commit. The same one-goroutine reports appear on the distro, and fail the run, at `e5ab039` (2026-10-15,
go1.27.2; 1 red run of 3) and `ac757a0` (2026-11-11; 3 of 3), and appear at `dafb87b` (2026-11-01). They were not caught, as far as the records show, because no recorded `-race` run of
this package was made in the distro before 2026-11-14: the host table of 0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md ran `-race ./...` on the Linux machine and
the Windows laptop, both green, and CI runs `make race` on `ubuntu-24.04` and `macos-15`, green at `5147e09`. The fleet
made the distro a `--need linux` target on 2026-11-12.

## Decision Drivers

* The owner's ruling: root cause first, and no tolerance tricks.
* A `make race` that is red for a reason outside gobble hides a real race behind the noise, and a red that is waved
  through teaches the next reader to wave one through.
* Race coverage of `internal/cli` on Linux must not be lost: CI and the Linux machine give it today.
* Every gobble change in reply to a report would change code that has no race, to quiet a detector.

## Considered Options

* **A.** Change nothing in gobble; record the verdict; run `-race` suites of this repository off the WSL2 distro until
  the one-test reproducer is clean there, and report the reproducer upstream.
* **B.** Change nothing yet; update the distro's kernel (`wsl --update`) and repeat the reproducer before choosing A or
  keeping the distro.
* **C.** Change gobble's code or tests so the distro stops reporting: for example a different buffer in `readAtMost`,
  fewer `append`s in `lookupModelWith`, or a different parser set-up.
* **D.** Keep the distro as a `-race` target and treat its `internal/cli` result as advisory.

## Decision Outcome

Chosen option: "A", with B's kernel update as its first step and without A's upstream report, because the evidence
puts the fault in the environment, A keeps every gate's meaning and the package's race coverage on CI and the Linux
machine, and B alone leaves `make race` red on the distro if the update does not help. No gobble file changes.

*(Proposed 2026-11-14 as "A, with B's kernel update as its first step". The owner's rulings of 2026-11-14, relayed by
the coordinator, follow; the options above stay as they were written.)*

* **Q1, 2026-11-14: "Update WSL, keep -race off it until green".** Run `wsl --update`, then rerun the reproducer. Until
  the reproducer is green in the distro, `-race` suites are not routed to the WSL distros; suites without `-race` still
  may be. The routing change is an amendment to the dotfiles repository's
  `0015-MADR-load-aware-dev-host-test-dispatch.md`. It is pending, and it is not written by this record or its plan.
* **Q2, 2026-11-14: "No, it's a host fault".** Keeping `-race` off a host whose race detector misreports is not one of
  the `-race` exclusions the ruling of "Root-cause it first" forbids. The host is re-admitted by one check: the
  reproducer `go test -race -count=40 -run '^TestPickModel$' ./internal/cli/`, green in the distro in three runs.
* **Q3, 2026-11-14: "Don't chase it".** No gobble-free reproducer is sought and no issue is filed upstream, with
  golang/go or with the WSL project. Option A's upstream report is dropped.

### Consequences

* Good, because no gobble code or test is changed to quiet reports about memory that one goroutine owns.
* Good, because `make race` keeps failing on any real race: a planted two-goroutine race in `internal/cli` is reported
  as two goroutines and fails the run (More Information).
* Bad, because A narrows where the fleet runs `-race`: the distro loses it until it is shown clean. That is a host
  choice, not a package or test exclusion; the owner ruled on 2026-11-14 (Q2) that it is not a forbidden `-race`
  exclusion.
* Bad, because the fault is not chased (Q3): it may show in other repositories' `-race` runs on the distro, and nobody
  upstream is told of it. The routing of Q1 keeps those runs off the distro too, until the reproducer is green.
* Bad, because the mechanism is not known; a later Go or WSL release could change the result either way.

### Confirmation

* The reproducer `go test -race -count=40 -run '^TestPickModel$' ./internal/cli/` is red on the distro before any
  change (3 of 3 runs, 5 to 8 reports each, all one-goroutine) and green on the Linux machine with the same binary.
* After the chosen option: on the distro, the reproducer and `make race` are green (kernel update), or no `-race` suite
  of this repository is routed there (routing); on the Linux machine, CI and the macOS laptop, `make race` stays green.
* Re-admission (Q2): after `wsl --update`, the reproducer is green in the distro in three runs, with no report of either
  kind. Until then the distro takes no `-race` suite.

## Pros and Cons of the Options

### A. Change nothing in gobble; route `-race` off the distro; report upstream

* Good, because it acts on the evidence: the same binary is clean elsewhere.
* Good, because CI, the Linux machine and the macOS laptop keep `make race` on every package.
* Neutral, because the routing lives in the fleet's dev-host tooling, outside this repository.
* Bad, because the distro, otherwise the fleet's fastest Linux target, stops running `-race`.

### B. Update the distro's kernel and repeat the reproducer

* Good, because it is cheap and may remove the fault without a routing change.
* Bad, because it changes the laptop, and the outcome is unknown; if it does not help, A is still needed.

### C. Change gobble so the distro stops reporting

* Bad, because R1 to R8 are not races: any change quiets a detector, which the ruling forbids.
* Bad, because the reports move from test to test between runs, so no finite set of changes is known to end them.

### D. Keep the distro and treat its result as advisory

* Bad, because it is a tolerance trick: a red gate that is read as green.

## More Information

### Runs (2026-11-14)

All runs used `CGO_ENABLED=1`, `GOAMD64=v1`, and the test binary from `go test -race -c ./internal/cli/` at `cccef77`
unless named. "Same-goroutine" counts reports whose two accesses name one goroutine.

* The distro, package runs (`-test.count=1`): 9 (the P1 log), 10, 6, 3 and 7 (`-test.cpu=4`), 4 (`GORACE=verbosity=2`),
  29 and 13 (`GORACE=verbosity=2`, no reset logged), 14, 9, 12, 6 (with a live-goroutine probe, peak 65), 14 and 11
  (`-trimpath`), 1 and 1 (`-test.cpu=1`), 1 and 2 (`GODEBUG=asyncpreemptoff=1`), 1 and 2 (`taskset -c 0-3`), 2 and 1
  (`taskset -c 16-31`), 2 and 1 (`-trimpath`, sha256 `87d6317e…`). Every run red, every report same-goroutine.
* The distro, go1.27.3 (the `go` line lowered in a scratch copy only): 1 and 2 reports, both red.
* The distro, `TestPickModel` alone, `-test.count=40`: 6, 8 and 5 reports.
* The distro, older commits, two runs each: `e5ab039` 1 and 0, `dafb87b` 1 and 1, `ac757a0` 1 and 8 reports (this
  batch's exit statuses were lost to a script fault, so it was repeated with `-test.v`). Repeated, three runs each:
  `e5ab039` 0, 0 and 1 reports, the last red with `--- FAIL: TestModelsInfo` "race detected during execution of test";
  `ac757a0` 2, 1 and 5 reports, all three red, the same way. Every report same-goroutine.
* The Linux machine: 5 package runs, 0 reports. The distro's binary (sha256 `87d6317e…`): 2 package runs and 2 × 40
  `TestPickModel`, 0 reports, exit 0.
* The Windows laptop, natively: 2 package runs and 2 × 40 `TestPickModel`, 0 reports, exit 0.
* The macOS laptop: 5 package runs, 0 reports.
* Gobble-free tests in the distro (a scratch module): 128 goroutines doing goroutine-local work under 400,000 lock
  releases each, 36 shadow resets logged, 0 reports; 1000 parked goroutines and 1000 workers, 0 reports.

### The gate still sees a real race

A scratch copy of `cccef77` added `TestPlantedCrossGoroutineRace`: the test goroutine and a goroutine it starts both
`append` to one `runEnv.overrides`. On the macOS laptop, `go test -race -count=1 -run
'^(TestPlantedCrossGoroutineRace|TestPickModel)$' ./internal/cli/` exited 1 with 2 reports, both naming two goroutines
(`plant_race_test.go:15` in the started goroutine, `plant_race_test.go:18` in the test's), and
`--- FAIL: TestPlantedCrossGoroutineRace`. `TestPickModel` passed.

### Not verified, with the step that settles it

* The mechanism in the race runtime under WSL2. Not to be settled: the owner ruled on 2026-11-14 (Q3) not to chase it.
* Whether a newer WSL kernel clears it: the plan's re-admission check, after `wsl --update` (Q1).
* The distro's other Linux targets (`Ubuntu-26.04`, `AlmaLinux-10`) could not build with `-race`: neither has `gcc`.

### Related

* [0037-PLAN-resume-prefix-and-failed-turn-history.md](0037-PLAN-resume-prefix-and-failed-turn-history.md), whose P1 gate run found the failure.
* [0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md), the toolchain this ran on; not the cause.
* [0058-PLAN-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md](0058-PLAN-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md)
* The dotfiles repository's `0015-MADR-load-aware-dev-host-test-dispatch.md`: where the fleet routes suites; the
  amendment that keeps `-race` off the WSL distros (Q1) is pending there.
