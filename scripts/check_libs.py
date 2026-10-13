#!/usr/bin/env python3
"""Check gobble against the latest tags of its sibling libraries.

docs/decisions/0019-MADR-sibling-libraries-current-pins-and-integration.md
D14: for acp-go-sdk2, go-llmprovider-sdk and go-selfupdate-lib, take the
latest semver tag (never a branch), and on a scratch clone of this
repository outside the working tree:

  1. go get the tag and go mod tidy, failing on any other module that moves;
  2. go build, go vet for linux, darwin and windows, and go test the
     packages that import the library;
  3. apidiff from the pinned version, failing on an incompatible change of a
     v1 module (a module below v1 is reported, not failed); apidiff's exit
     status is not its verdict, so its text is read;
  4. govulncheck, and the library's GitHub security advisories, which
     govulncheck does not see.

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
    repo: str
    packages: tuple[str, ...]

    @property
    def module(self) -> str:
        return f"github.com/{OWNER}/{self.repo}"


LIBRARIES = (
    Library("acp-go-sdk2", ("./acpclient/...", "./acpserver/...", "./internal/cli/...")),
    Library("go-llmprovider-sdk", ("./llm/...", "./acpserver/...", "./internal/cli/...")),
    Library("go-selfupdate-lib", ("./internal/buildinfo/...", "./internal/cli/...", "./internal/archtest/...")),
)


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
    env.update(GOTOOLCHAIN="go1.27.2", CGO_ENABLED="0", **extra)
    return env


def run(args: list[str], cwd: Path, log: Path, timeout: int = 1800, **env: str) -> int:
    """Run args in cwd, write their whole output to log, return the exit status."""
    with log.open("wb") as out:
        out.write(f"$ {' '.join(args)}\n".encode())
        out.flush()
        return subprocess.run(args, cwd=cwd, env=clean_env(**env), stdout=out, stderr=subprocess.STDOUT,
                              timeout=timeout).returncode


def latest_tag(lib: Library) -> str:
    out = subprocess.run(["git", "ls-remote", "--tags", f"https://github.com/{OWNER}/{lib.repo}"],
                         capture_output=True, text=True, timeout=120, env=clean_env(), check=True).stdout
    tags = {line.split("refs/tags/")[1].removesuffix("^{}") for line in out.splitlines() if "refs/tags/" in line}
    semver = sorted((t for t in tags if SEMVER.match(t)), key=version_key)
    if not semver:
        raise RuntimeError(f"{lib.repo} has no semver tag")
    return semver[-1]


def module_version(lib: Library, root: Path) -> str:
    """The library's version as the module at root requires it."""
    return subprocess.run(["go", "list", "-m", "-f", "{{.Version}}", lib.module], cwd=root, capture_output=True,
                          text=True, timeout=120, env=clean_env(), check=True).stdout.strip()


def pinned(lib: Library) -> str:
    return module_version(lib, REPO)


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


def moved_modules(before: str, after: str, module: str) -> list[str]:
    """The modules other than module that a go.mod change adds, removes or moves."""
    a, b = requirements(before), requirements(after)
    return sorted(p for p in a.keys() | b.keys() if p != module and a.get(p) != b.get(p))


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


def export_api(apidiff: Path, lib: Library, version: str, work: Path, logs: Path) -> Path:
    """apidiff's export of the library at version, from a scratch module requiring it."""
    mod = work / f"api-{version}"
    mod.mkdir()
    log = logs / f"{lib.repo}-api-{version}.log"
    run(["go", "mod", "init", "scratch/checklibs"], mod, log)
    run(["go", "get", f"{lib.module}@{version}"], mod, logs / f"{lib.repo}-api-get-{version}.log", GOFLAGS="-mod=mod")
    out = work / f"{version}.exp"
    if run([str(apidiff), "-m", "-w", str(out), lib.module], mod, log, GOFLAGS="-mod=mod") != 0 or not out.exists():
        raise RuntimeError(f"apidiff could not export {lib.module}@{version}; see {log}")
    return out


def check_library(lib: Library, tag: str, base: str, work: Path) -> Report:
    report = Report(lib.repo)
    logs = work / "logs"
    logs.mkdir(exist_ok=True)
    clone = work / f"clone-{lib.repo}"
    report.note("versions", f"pinned {base}, checking {tag}")
    if run(["git", "clone", "--quiet", str(REPO), str(clone)], work, logs / f"{lib.repo}-clone.log") != 0:
        report.check("clone", False)
        return report
    before = (clone / "go.mod").read_text(encoding="utf-8")
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
    moved = moved_modules(before, (clone / "go.mod").read_text(encoding="utf-8"), lib.module)
    report.check("modules", not moved, "new or changed module: " + ", ".join(moved) if moved else "")
    report.check("build", run(["go", "build", "./..."], clone, logs / f"{lib.repo}-build.log") == 0)
    for goos in GOOSES:
        report.check(f"vet {goos}", run(["go", "vet", "./..."], clone, logs / f"{lib.repo}-vet-{goos}.log", GOOS=goos) == 0)
    tested = run(["go", "test", "-count=1", *lib.packages], clone, logs / f"{lib.repo}-test.log") == 0
    report.check("test", tested, "" if tested else f"see {logs / (lib.repo + '-test.log')}")
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


def self_test() -> int:
    """The parsers on fixed inputs; each case must give its expected answer."""
    mod_a = "module m\n\nrequire (\n\tgithub.com/maccavelli/acp-go-sdk2 v0.14.0\n\tgolang.org/x/sys v0.48.0 // indirect\n)\n"
    mod_b = mod_a.replace("v0.14.0", "v0.17.0")
    mod_c = mod_b.replace("golang.org/x/sys v0.48.0 // indirect\n", "golang.org/x/sys v0.48.0 // indirect\n\texample.com/x v1.0.0\n")
    diff_bad = "Incompatible changes:\n- Foo: removed\n\nCompatible changes:\n- Bar: added\n"
    diff_ok = "Compatible changes:\n- Bar: added\n"
    cases = [
        ("an incompatible line fails", incompatible(diff_bad) == ["- Foo: removed"]),
        ("no incompatible line passes", incompatible(diff_ok) == []),
        ("only the library moving passes", moved_modules(mod_a, mod_b, "github.com/maccavelli/acp-go-sdk2") == []),
        ("adding example.com/x fails", moved_modules(mod_a, mod_c, "github.com/maccavelli/acp-go-sdk2") == ["example.com/x"]),
        ("= 0.15.0 covers v0.15.0", covers("= 0.15.0", "v0.15.0")),
        ("= 0.15.0 does not cover v0.16.0", not covers("= 0.15.0", "v0.16.0")),
        ("< 1.2.0, >= 1.0.0 covers v1.1.0", covers("< 1.2.0, >= 1.0.0", "v1.1.0")),
        ("< 1.2.0, >= 1.0.0 does not cover v1.2.0", not covers("< 1.2.0, >= 1.0.0", "v1.2.0")),
    ]
    for name, ok in cases:
        print(f"self-test {name}: {'ok' if ok else 'FAIL'}")
    return 0 if all(ok for _, ok in cases) else 1


def main() -> int:
    ap = argparse.ArgumentParser(description="Check gobble against the latest tags of its sibling libraries.")
    ap.add_argument("--lib", choices=[lib.repo for lib in LIBRARIES], help="one library; default all three")
    ap.add_argument("--tag", help="the version to check; default the latest semver tag")
    ap.add_argument("--from", dest="base", help="the version to diff from; default the pinned one")
    ap.add_argument("--keep", action="store_true", help="keep the scratch directory")
    ap.add_argument("--self-test", action="store_true", help="test the parsers and exit")
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
