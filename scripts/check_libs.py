#!/usr/bin/env python3
"""Check gobble against the latest tags of its sibling libraries.

docs/decisions/0019-MADR-sibling-libraries-current-pins-and-integration.md
D14, extended by
docs/decisions/0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md
D2: for acp-go-sdk2, go-llmprovider-sdk, go-selfupdate-lib and go-tui-lib,
take the latest semver tag of the major the module path names (never a
branch), and on a scratch clone of this repository outside the working tree:

  1. go get the tag and go mod tidy, failing on any other module that moves;
  2. go build, go vet for linux, darwin and windows, go test the packages
     that import the library, and make staticcheck and make lint, which see
     a deprecation go vet does not (0035-MADR F2); for go-tui-lib also make
     archtest, which checks the CLI's boundary against the real closure;
  3. apidiff from the pinned version, failing on an incompatible change of a
     v1 module (a module below v1 is reported, not failed); apidiff's exit
     status is not its verdict, so its text is read; a library pinned at
     the tag has nothing to compare;
  4. govulncheck, and the library's GitHub security advisories, which
     govulncheck does not see.

A library gobble does not require yet (go-tui-lib, before its first
import) is probed: the scratch clone gains a file importing the packages
gobble will use, so tidy keeps the requirement, and the modules it may add
are the ones the library names; there is no pinned version to diff from.

It needs the network, so it is `make check-libs`, not part of preflight.
It reads, prints and sends no credential: every command runs without the
variables whose names look like one.

Usage:
  scripts/check_libs.py [--lib NAME] [--tag vX.Y.Z] [--from vX.Y.Z] [--keep]
  scripts/check_libs.py --self-test

Exit status is 1 when any check fails.
"""

from __future__ import annotations

import argparse
import contextlib
import io
import json
import os
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import urllib.request
from dataclasses import dataclass, field
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
OWNER = "maccavelli"
GOOSES = ("linux", "darwin", "windows")
SEMVER = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")
SECRET = re.compile(r"KEY|TOKEN|SECRET|PASSWORD", re.IGNORECASE)
REQUIRE = re.compile(r"^\s*([^\s()]+)\s+(v[^\s]+)")


@dataclass(frozen=True)
class Library:
    """A sibling library: its repository, the gobble packages that import it,
    its module path when that is not the repository's (a /vN module), the
    major whose tags count (None: v0 and v1), the packages a probe imports
    while gobble does not require it, and the modules that first import may
    add (0035-MADR D2)."""

    repo: str
    packages: tuple[str, ...]
    path: str = ""
    major: int | None = None
    probe: tuple[str, ...] = ()
    allow_new: tuple[str, ...] = ()

    @property
    def module(self) -> str:
        return self.path or f"github.com/{OWNER}/{self.repo}"


LIBRARIES = (
    Library("acp-go-sdk2", ("./acpclient/...", "./acpserver/...", "./internal/cli/...")),
    Library("go-llmprovider-sdk", ("./llm/...", "./acpserver/...", "./internal/cli/...")),
    Library("go-selfupdate-lib", ("./internal/buildinfo/...", "./internal/cli/...", "./internal/archtest/...")),
    Library("go-tui-lib", ("./internal/cli/...", "./internal/archtest/..."), path=f"github.com/{OWNER}/go-tui-lib/v2", major=2,
            probe=("ask", "glyph", "lineedit", "safetext", "stream", "transcript"),
            allow_new=("github.com/charmbracelet/x/ansi", "github.com/clipperhouse/displaywidth",
                       "github.com/clipperhouse/uax29/v2", "github.com/lucasb-eyer/go-colorful",
                       "github.com/mattn/go-runewidth")),
)

PROBE_FILE = "internal/cli/zz_checklibs_probe.go"


@dataclass
class Report:
    lib: str
    failures: list[str] = field(default_factory=list)

    def check(self, name: str, ok: bool, detail: str = "") -> bool:
        line = f"{self.lib} {name}: {'ok' if ok else 'FAIL'}{' ' + detail if detail else ''}"
        print(line, flush=True)
        if not ok:
            self.failures.append(line)
        return ok

    def note(self, name: str, detail: str) -> None:
        print(f"{self.lib} {name}: {detail}", flush=True)


def version_key(v: str) -> tuple[int, int, int]:
    m = SEMVER.match(v if v.startswith("v") else "v" + v)
    if not m:
        raise ValueError(f"not a semver version: {v}")
    return int(m[1]), int(m[2]), int(m[3])


def clean_env(**extra: str) -> dict[str, str]:
    """The environment without credential-shaped variables, with extra set."""
    env = {k: v for k, v in os.environ.items() if not SECRET.search(k)}
    env.update(GOTOOLCHAIN="go1.27.4", CGO_ENABLED="0", **extra)
    return env


def run(args: list[str], cwd: Path, log: Path, timeout: int = 1800, **env: str) -> int:
    """Run args in cwd, write their whole output to log, return the exit status."""
    with log.open("wb") as out:
        out.write(f"$ {' '.join(args)}\n".encode())
        out.flush()
        return subprocess.run(args, cwd=cwd, env=clean_env(**env), stdout=out, stderr=subprocess.STDOUT,
                              timeout=timeout).returncode


def newest(tags: set[str], major: int | None) -> str | None:
    """The newest semver tag of major (None: v0 and v1), or None."""
    ok = (lambda m: m == major) if major is not None else (lambda m: m in (0, 1))
    semver = sorted((t for t in tags if SEMVER.match(t) and ok(version_key(t)[0])), key=version_key)
    return semver[-1] if semver else None


def latest_tag(lib: Library) -> str:
    out = subprocess.run(["git", "ls-remote", "--tags", f"https://github.com/{OWNER}/{lib.repo}"],
                         capture_output=True, text=True, timeout=120, env=clean_env(), check=True).stdout
    tags = {line.split("refs/tags/")[1].removesuffix("^{}") for line in out.splitlines() if "refs/tags/" in line}
    tag = newest(tags, lib.major)
    if tag is None:
        raise RuntimeError(f"{lib.repo} has no semver tag of major {lib.major if lib.major is not None else '0 or 1'}")
    return tag


def module_version(lib: Library, root: Path) -> str:
    """The library's version as the module at root requires it."""
    return subprocess.run(["go", "list", "-m", "-f", "{{.Version}}", lib.module], cwd=root, capture_output=True,
                          text=True, timeout=120, env=clean_env(), check=True).stdout.strip()


def pinned(lib: Library) -> str | None:
    """The version gobble requires, or None when it does not require it."""
    listed = subprocess.run(["go", "list", "-m", "-f", "{{.Path}}", "all"], cwd=REPO, capture_output=True, text=True,
                            timeout=120, env=clean_env(), check=True).stdout.split()
    return module_version(lib, REPO) if lib.module in listed else None


def probe_source(lib: Library) -> str:
    """The scratch clone's file that imports lib's probe packages, gofmt-clean."""
    lines = ["// Code written by scripts/check_libs.py in a scratch clone only (0035-MADR D2).", "", "package cli", "",
             "import ("]
    for pkg in sorted(lib.probe):
        lines += ["\t// probed by check-libs before gobble's first import of the library", f'\t_ "{lib.module}/{pkg}"']
    return "\n".join(lines + [")", ""])


def requirements(gomod: str) -> dict[str, str]:
    """Every required module of a go.mod's text, by path."""
    out: dict[str, str] = {}
    block = False
    for raw in gomod.splitlines():
        line = raw.split("//")[0].strip()
        if line.startswith("require ("):
            block = True
            continue
        if block and line == ")":
            block = False
            continue
        if block or line.startswith("require "):
            m = REQUIRE.match(line.removeprefix("require "))
            if m:
                out[m[1]] = m[2]
    return out


def moved_modules(before: str, after: str, module: str, allow: tuple[str, ...] = ()) -> list[str]:
    """The modules other than module that a go.mod change adds, removes or
    moves, less those allow names as added by a first import."""
    a, b = requirements(before), requirements(after)
    return sorted(p for p in a.keys() | b.keys()
                  if p != module and a.get(p) != b.get(p) and not (p in allow and p not in a))


def incompatible(apidiff_text: str) -> list[str]:
    """apidiff's incompatible changes: the hyphen lines before "Compatible changes:"."""
    head = apidiff_text.split("Compatible changes:")[0]
    return [line.strip() for line in head.splitlines() if line.strip().startswith("- ")]


def covers(range_text: str, version: str) -> bool:
    """Whether a GitHub advisory's vulnerable_version_range covers version."""
    v = version_key(version)
    for part in range_text.split(","):
        m = re.fullmatch(r"\s*(<=|>=|<|>|=)?\s*v?(\d+\.\d+\.\d+)\s*", part)
        if not m:
            raise ValueError(f"unreadable version range: {range_text}")
        op, bound = m[1] or "=", version_key(m[2])
        if not {"=": v == bound, "<": v < bound, "<=": v <= bound, ">": v > bound, ">=": v >= bound}[op]:
            return False
    return True


def advisories(lib: Library) -> list[dict]:
    req = urllib.request.Request(f"https://api.github.com/repos/{OWNER}/{lib.repo}/security-advisories",
                                 headers={"Accept": "application/vnd.github+json", "User-Agent": "gobble-check-libs"})
    with urllib.request.urlopen(req, timeout=60) as resp:  # noqa: S310 - a fixed https URL
        return json.load(resp)


def api_paths(lib: Library, version: str, work: Path) -> tuple[Path, Path]:
    """The scratch module and the export of the library at version. Both name
    the library, so two libraries at one version do not collide (0023-PLAN
    amendment of 2026-10-19, A4)."""
    return work / f"api-{lib.repo}-{version}", work / f"{lib.repo}-{version}.exp"


def export_api(apidiff: Path, lib: Library, version: str, work: Path, logs: Path) -> Path:
    """apidiff's export of the library at version, from a scratch module requiring it."""
    mod, out = api_paths(lib, version, work)
    mod.mkdir()
    log = logs / f"{lib.repo}-api-{version}.log"
    run(["go", "mod", "init", "scratch/checklibs"], mod, log)
    run(["go", "get", f"{lib.module}@{version}"], mod, logs / f"{lib.repo}-api-get-{version}.log", GOFLAGS="-mod=mod")
    if run([str(apidiff), "-m", "-w", str(out), lib.module], mod, log, GOFLAGS="-mod=mod") != 0 or not out.exists():
        raise RuntimeError(f"apidiff could not export {lib.module}@{version}; see {log}")
    return out


def check_api(report: Report, lib: Library, base: str, tag: str, work: Path, logs: Path) -> None:
    """apidiff from base to tag. A library pinned at the tag has nothing to
    compare, and exporting one version twice would collide (0019-PLAN P5
    deviation 15)."""
    if base == tag:
        report.note("api", f"pinned at {tag}; nothing to compare")
        return
    apidiff = work / ("apidiff.exe" if os.name == "nt" else "apidiff")
    if not apidiff.exists():
        run(["go", "build", "-o", str(apidiff), "golang.org/x/exp/cmd/apidiff"], REPO, logs / "apidiff-build.log")
    try:
        old, new = export_api(apidiff, lib, base, work, logs), export_api(apidiff, lib, tag, work, logs)
        diff_log = logs / f"{lib.repo}-apidiff.log"
        run([str(apidiff), "-m", str(old), str(new)], work, diff_log)
        breaks = incompatible(diff_log.read_text(encoding="utf-8", errors="replace"))
        if version_key(tag)[0] == 0:
            report.note("api", f"{len(breaks)} incompatible changes {base}->{tag}, reported only below v1; see {diff_log}")
        else:
            report.check("api", not breaks, f"{len(breaks)} incompatible: {breaks[0]}" if breaks else "")
    except RuntimeError as err:
        report.check("api", False, str(err))


def pinned_at_tag_not_compared() -> bool:
    """check_api with the base the tag exports nothing: its work and log
    directories do not exist, so any export or build would fail the case."""
    report = Report("self-test")
    missing = Path(tempfile.gettempdir()) / "gobble-check-libs-self-test-missing"
    out = io.StringIO()
    try:
        with contextlib.redirect_stdout(out):
            check_api(report, LIBRARIES[0], "v1.0.0", "v1.0.0", missing, missing / "logs")
    except Exception:  # noqa: BLE001 - any failure of the call fails the case
        return False
    return not report.failures and "nothing to compare" in out.getvalue() and not missing.exists()


def check_library(lib: Library, tag: str, base: str, work: Path) -> Report:
    report = Report(lib.repo)
    logs = work / "logs"
    logs.mkdir(exist_ok=True)
    clone = work / f"clone-{lib.repo}"
    report.note("versions", f"pinned {base or 'nothing'}, checking {tag}")
    if run(["git", "clone", "--quiet", str(REPO), str(clone)], work, logs / f"{lib.repo}-clone.log") != 0:
        report.check("clone", False)
        return report
    before = (clone / "go.mod").read_text(encoding="utf-8")
    if base is None:
        if not lib.probe:
            report.check("pinned", False, "gobble does not require it, and it names no packages to probe")
            return report
        (clone / PROBE_FILE).write_text(probe_source(lib), encoding="utf-8")
        report.note("pinned", f"not required; probing {len(lib.probe)} packages through {PROBE_FILE}")
    ok = run(["go", "get", f"{lib.module}@{tag}"], clone, logs / f"{lib.repo}-get.log") == 0
    ok = ok and run(["go", "mod", "tidy"], clone, logs / f"{lib.repo}-tidy.log") == 0
    if not report.check("get and tidy", ok, "" if ok else f"see {logs}"):
        return report
    # tidy resolves a package gobble imports from the newest version that
    # has it, so a tag lacking one ends up as another version; every check
    # after this would then test that version (0019-PLAN P1 deviation 1).
    got = module_version(lib, clone)
    if not report.check("version", got == tag, "" if got == tag else f"tidy moved it to {got}: {tag} lacks a package gobble imports"):
        return report
    allow = lib.allow_new if base is None else ()
    moved = moved_modules(before, (clone / "go.mod").read_text(encoding="utf-8"), lib.module, allow)
    report.check("modules", not moved, "new or changed module: " + ", ".join(moved) if moved else "")
    report.check("build", run(["go", "build", "./..."], clone, logs / f"{lib.repo}-build.log") == 0)
    for goos in GOOSES:
        report.check(f"vet {goos}", run(["go", "vet", "./..."], clone, logs / f"{lib.repo}-vet-{goos}.log", GOOS=goos) == 0)
    tested = run(["go", "test", "-count=1", *lib.packages], clone, logs / f"{lib.repo}-test.log") == 0
    report.check("test", tested, "" if tested else f"see {logs / (lib.repo + '-test.log')}")
    for gate in ("staticcheck", "lint", *(("archtest",) if lib.probe else ())):
        log = logs / f"{lib.repo}-{gate}.log"
        passed = run(["make", gate], clone, log) == 0
        report.check(gate, passed, "" if passed else f"see {log}")
    if base is None:
        report.note("api", "not required; nothing to compare")
    else:
        check_api(report, lib, base, tag, work, logs)
    report.check("govulncheck", run(["go", "tool", "govulncheck", "./..."], clone, logs / f"{lib.repo}-vuln.log") == 0)
    try:
        hits = [f"advisory {a.get('ghsa_id')} covers {tag}" for a in advisories(lib)
                if a.get("state", "published") == "published"
                for v in a.get("vulnerabilities") or [] if v.get("vulnerable_version_range")
                and covers(v["vulnerable_version_range"], tag)]
        report.check("advisories", not hits, "; ".join(hits))
    except (OSError, ValueError) as err:
        report.check("advisories", False, f"could not read: {err}")
    return report


def remove_tree(path: Path) -> bool:
    """Remove path, clearing the read-only bit git sets on its objects on
    Windows; report whether it is gone."""
    def writable_and_retry(func, target, _exc):  # noqa: ANN001 - shutil's onexc signature
        os.chmod(target, stat.S_IWRITE)
        func(target)

    shutil.rmtree(path, onexc=writable_and_retry)
    return not path.exists()


def api_paths_distinct() -> bool:
    """Two libraries at one version export to paths of their own."""
    work = Path("work")
    a, b = api_paths(LIBRARIES[1], "v1.17.0", work), api_paths(LIBRARIES[2], "v1.17.0", work)
    return not set(a) & set(b)


def self_test() -> int:
    """The parsers on fixed inputs, the API step for a library pinned at the
    tag, and the API paths of two libraries at one version; each case must
    give its expected answer."""
    mod_a = "module m\n\nrequire (\n\tgithub.com/maccavelli/acp-go-sdk2 v0.14.0\n\tgolang.org/x/sys v0.48.0 // indirect\n)\n"
    mod_b = mod_a.replace("v0.14.0", "v0.17.0")
    mod_c = mod_b.replace("golang.org/x/sys v0.48.0 // indirect\n", "golang.org/x/sys v0.48.0 // indirect\n\texample.com/x v1.0.0\n")
    diff_bad = "Incompatible changes:\n- Foo: removed\n\nCompatible changes:\n- Bar: added\n"
    diff_ok = "Compatible changes:\n- Bar: added\n"
    tui = next(lib for lib in LIBRARIES if lib.repo == "go-tui-lib")
    tags = {"v0.12.0", "v1.13.0", "v2.6.1", "v2.6.1-rc1", "nightly"}
    allow = ("example.com/a", "example.com/b", "example.com/c", "example.com/d", "example.com/e")
    mod_d = mod_a + "require (\n" + "".join(f"\t{m} v1.0.0 // indirect\n" for m in (*allow, "example.com/sixth")) + ")\n"
    probe_want = ("// Code written by scripts/check_libs.py in a scratch clone only (0035-MADR D2).\n\npackage cli\n\nimport (\n"
                  "\t// probed by check-libs before gobble's first import of the library\n\t_ \"example.com/x/v2/a\"\n"
                  "\t// probed by check-libs before gobble's first import of the library\n\t_ \"example.com/x/v2/b\"\n)\n")
    cases = [
        ("an incompatible line fails", incompatible(diff_bad) == ["- Foo: removed"]),
        ("no incompatible line passes", incompatible(diff_ok) == []),
        ("only the library moving passes", moved_modules(mod_a, mod_b, "github.com/maccavelli/acp-go-sdk2") == []),
        ("adding example.com/x fails", moved_modules(mod_a, mod_c, "github.com/maccavelli/acp-go-sdk2") == ["example.com/x"]),
        ("= 0.15.0 covers v0.15.0", covers("= 0.15.0", "v0.15.0")),
        ("= 0.15.0 does not cover v0.16.0", not covers("= 0.15.0", "v0.16.0")),
        ("< 1.2.0, >= 1.0.0 covers v1.1.0", covers("< 1.2.0, >= 1.0.0", "v1.1.0")),
        ("< 1.2.0, >= 1.0.0 does not cover v1.2.0", not covers("< 1.2.0, >= 1.0.0", "v1.2.0")),
        ("a library pinned at the tag is not compared", pinned_at_tag_not_compared()),
        ("two libraries at one version do not share API paths", api_paths_distinct()),
        ("a path overrides the module", tui.module == "github.com/maccavelli/go-tui-lib/v2"
         and LIBRARIES[0].module == "github.com/maccavelli/acp-go-sdk2"),
        ("major 2 keeps v2.6.1 alone", newest(tags, 2) == "v2.6.1"),
        ("no major drops v2", newest(tags, None) == "v1.13.0"),
        ("a first import may add the allowed modules alone",
         moved_modules(mod_a, mod_d, "example.com/lib", allow) == ["example.com/sixth"]),
        ("the probe file is exact", probe_source(Library("x", (), path="example.com/x/v2", probe=("b", "a"))) == probe_want),
    ]
    for name, ok in cases:
        print(f"self-test {name}: {'ok' if ok else 'FAIL'}")
    return 0 if all(ok for _, ok in cases) else 1


def main() -> int:
    ap = argparse.ArgumentParser(description="Check gobble against the latest tags of its sibling libraries.")
    ap.add_argument("--lib", choices=[lib.repo for lib in LIBRARIES], help="one library; default all of them")
    ap.add_argument("--tag", help="the version to check; default the latest semver tag")
    ap.add_argument("--from", dest="base", help="the version to diff from; default the pinned one")
    ap.add_argument("--keep", action="store_true", help="keep the scratch directory")
    ap.add_argument("--self-test", action="store_true", help="run the offline self-test and exit")
    args = ap.parse_args()
    if args.self_test:
        return self_test()
    work = Path(tempfile.mkdtemp(prefix="gobble-check-libs-"))
    print(f"logs: {work / 'logs'}", flush=True)
    failures: list[str] = []
    for lib in LIBRARIES:
        if args.lib and lib.repo != args.lib:
            continue
        report = check_library(lib, args.tag or latest_tag(lib), args.base or pinned(lib), work)
        failures += report.failures
    kept = args.keep or bool(failures)
    if not kept and not remove_tree(work):
        print(f"could not remove {work}")
        kept = True
    print(f"{len(failures)} failing checks; " + (f"scratch kept at {work}" if kept else "scratch removed"))
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
