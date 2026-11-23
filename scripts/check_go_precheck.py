#!/usr/bin/env python3
"""Test scripts/go-precheck.sh offline, against its observable contract.

docs/decisions/0069-MADR-pre-add-script-follow-ups-of-0064.md (state table
G1 to G19) and docs/decisions/0069-PLAN-pre-add-script-follow-ups-of-0064.md.

Every case runs the script in a scratch git repository (with an `acpext`
directory, which the lint step enters) whose PATH starts with a stub
directory holding `go` and `golangci-lint`: each call is appended to a
counter file and exits 0. gofmt is the real one. No PATH of symlinks is
built, which is fragile on Windows. Each test runs under the `bash` on PATH
and also under `/bin/bash` where that is another program (bash 3.2 on macOS).

The script reads only its arguments, its environment and the repository it
is run in, so this harness checks a port of it unchanged: pass the port to
`--script`.

Usage:
  scripts/check_go_precheck.py [--script PATH] [--bash PATH]
  scripts/check_go_precheck.py --self-test [--script PATH] [--bash PATH]

Without --self-test, the tests run against the script. With it, they run
against the script and then each plant (one production change on a scratch
copy of the script) must make its named test fail; a plant that no test
catches fails the self-test.

Exit status is 1 when a test fails, or a plant is not caught.
"""

from __future__ import annotations

import argparse
import contextlib
import os
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass, field
from pathlib import Path
from typing import Callable

ROOT = Path(__file__).resolve().parent.parent
DEFAULT_SCRIPT = ROOT / "scripts" / "go-precheck.sh"

SKIP_VARS = ("GO_PRECHECK_SKIP_LINT", "GO_PRECHECK_SKIP_VULN")
ALL_STEPS = "(gofmt, golangci-lint, govulncheck)"

CLEAN = "package p\n"
UNFORMATTED = "package p\n\nfunc f(){ }\n"
BROKEN = "package\n"


class Fail(Exception):
    """A test's expectation did not hold."""


class HarnessError(Exception):
    """The harness could not set a case up; never an expected red."""


@dataclass
class Result:
    rc: int
    out: str
    err: str
    calls: list[str] = field(default_factory=list)

    @property
    def text(self) -> str:
        return self.out + self.err


def _norm(b: bytes) -> str:
    return b.decode("utf-8", "replace").replace("\r\n", "\n")


def _stub(path: Path, name: str) -> None:
    with open(path, "w", newline="\n") as fh:
        fh.write('#!/bin/sh\necho "%s $*" >> "$STUB_COUNT"\nexit 0\n' % name)
    path.chmod(0o755)


def _gofmt_dirs(path: str) -> list[str]:
    """Every directory on `path` that holds a gofmt.

    Found as the script finds it (`command -v gofmt`): shutil.which over each
    entry, which also tries the Windows executable suffixes. Hosts can carry
    more than one Go install, so hiding one directory is not enough.
    """
    return [p for p in path.split(os.pathsep) if p and shutil.which("gofmt", path=p)]


def _without_gofmt(path: str) -> str:
    """`path` with every directory that holds a gofmt removed."""
    gds = _gofmt_dirs(path)
    return os.pathsep.join(
        p for p in path.split(os.pathsep) if p and not any(_same_dir(p, g) for g in gds))


def _same_dir(a: str, b: str) -> bool:
    return os.path.normcase(os.path.realpath(a)) == os.path.normcase(os.path.realpath(b))


def run_case(
    script: Path,
    bash: str,
    files: dict[str, str],
    args: list[str],
    env: dict[str, str] | None = None,
    track: tuple[str, ...] = (),
    remove: tuple[str, ...] = (),
    no_gofmt: bool = False,
    base_path: str | None = None,
) -> Result:
    """Run the script once in a fresh scratch repository."""
    with tempfile.TemporaryDirectory(prefix="goprecheck-") as td:
        tmp = Path(td)
        repo = tmp / "repo"
        stubs = tmp / "stubs"
        repo.mkdir()
        stubs.mkdir()
        counter = tmp / "calls.txt"
        counter.write_text("")
        _stub(stubs / "go", "go")
        _stub(stubs / "golangci-lint", "golangci-lint")
        (repo / "acpext").mkdir()
        for rel, body in files.items():
            with open(repo / rel, "w", newline="\n") as fh:
                fh.write(body)
        git = subprocess.run(["git", "init", "-q"], cwd=repo, capture_output=True)
        if git.returncode != 0:
            raise HarnessError("git init failed: " + _norm(git.stderr))
        if track:
            git = subprocess.run(["git", "add", "--", *track], cwd=repo, capture_output=True)
            if git.returncode != 0:
                raise HarnessError("git add failed: " + _norm(git.stderr))
        for rel in remove:
            (repo / rel).unlink()

        base = os.environ.get("PATH", "") if base_path is None else base_path
        if no_gofmt:
            if not _gofmt_dirs(base):
                raise HarnessError("gofmt is not on PATH; the harness needs the real one")
            real_git = shutil.which("git")
            base = _without_gofmt(base)
            if shutil.which("git", path=base) is None:
                # git shares a directory with a gofmt: reach it through a
                # wrapper in the stubs directory instead.
                if real_git is None:
                    raise HarnessError("git is not on PATH")
                with open(stubs / "git", "w", newline="\n") as fh:
                    fh.write('#!/bin/sh\nexec "%s" "$@"\n' % Path(real_git).as_posix())
                (stubs / "git").chmod(0o755)
        full = dict(os.environ)
        # BASH_ENV is sourced by a non-interactive bash and may rewrite PATH,
        # which would put the real tools in front of the stubs.
        for v in SKIP_VARS + ("BASH_ENV", "ENV"):
            full.pop(v, None)
        full["PATH"] = str(stubs) + os.pathsep + base
        full["STUB_COUNT"] = counter.as_posix()
        full.update(env or {})
        p = subprocess.run(
            [bash, str(script), *args], cwd=repo, env=full, capture_output=True
        )
        calls = [ln for ln in _norm(counter.read_bytes()).split("\n") if ln]
        return Result(p.returncode, _norm(p.stdout), _norm(p.stderr), calls)


def need(cond: bool, msg: str, r: Result | None = None) -> None:
    if not cond:
        detail = ""
        if r is not None:
            detail = "\n    rc=%d\n    stdout=%r\n    stderr=%r\n    calls=%r" % (
                r.rc, r.out, r.err, r.calls)
        raise Fail(msg + detail)


SKIP_BOTH = {v: "1" for v in SKIP_VARS}
GOFMT_FAIL_HEADER = "gofmt: failed with exit status"
NOTICE = "go-precheck: skipping %s: no such file (deleted?)"


def t1(script: Path, bash: str) -> None:
    """G1: a named file gofmt cannot parse, both skips set."""
    r = run_case(script, bash, {"bad.go": BROKEN}, ["bad.go"], SKIP_BOTH)
    need(r.rc == 1, "T1: exit 1 expected", r)
    need(GOFMT_FAIL_HEADER in r.err, "T1: gofmt header missing", r)
    need("bad.go" in r.err.split(GOFMT_FAIL_HEADER, 1)[1], "T1: gofmt's own message missing", r)
    need("clean" not in r.text, "T1: no clean line expected", r)


def t2(script: Path, bash: str) -> None:
    """G2: parses but unformatted."""
    r = run_case(script, bash, {"unfmt.go": UNFORMATTED}, ["unfmt.go"], SKIP_BOTH)
    need(r.rc == 1, "T2: exit 1 expected", r)
    need("are not formatted" in r.err, "T2: not-formatted text missing", r)
    need("  unfmt.go" in r.err, "T2: file not named", r)
    need("clean" not in r.text, "T2: no clean line expected", r)


def t3(script: Path, bash: str) -> None:
    """G3: one unparseable and one unformatted file in one call."""
    r = run_case(
        script, bash, {"bad.go": BROKEN, "unfmt.go": UNFORMATTED},
        ["bad.go", "unfmt.go"], SKIP_BOTH)
    need(r.rc == 1, "T3: exit 1 expected", r)
    need(GOFMT_FAIL_HEADER in r.err, "T3: gofmt error missing", r)
    need("are not formatted" in r.err and "  unfmt.go" in r.err, "T3: unformatted list missing", r)


def t4(script: Path, bash: str) -> None:
    """G4, G19: every step ran, clean files."""
    r = run_case(script, bash, {"a.go": CLEAN}, ["a.go"])
    need(r.rc == 0, "T4: exit 0 expected", r)
    need("go-precheck: 1 file(s) clean %s." % ALL_STEPS in r.out, "T4: closing line wrong", r)
    lint = [c for c in r.calls if c.startswith("golangci-lint")]
    need(len(lint) == 2, "T4: lint should run once per module (two calls)", r)
    need(any(c.startswith("go tool govulncheck ./...") for c in r.calls), "T4: govulncheck not run", r)


def t5(script: Path, bash: str) -> None:
    """G5, G6, G7: the closing line names the steps that ran."""
    cases = (
        ({"GO_PRECHECK_SKIP_LINT": "1"}, "(gofmt, govulncheck)", "golangci-lint: skipped"),
        ({"GO_PRECHECK_SKIP_VULN": "1"}, "(gofmt, golangci-lint)", "govulncheck: skipped"),
        (SKIP_BOTH, "(gofmt)", "golangci-lint: skipped"),
    )
    for env, names, skipped in cases:
        r = run_case(script, bash, {"a.go": CLEAN}, ["a.go"], env)
        need(r.rc == 0, "T5: exit 0 expected for %r" % env, r)
        need("go-precheck: 1 file(s) clean %s." % names in r.out,
             "T5: closing line should be %s for %r" % (names, env), r)
        need(skipped in r.err, "T5: skipped line missing for %r" % env, r)


def t6(script: Path, bash: str) -> None:
    """G8, G18: a deleted Go path is skipped with a notice and not counted."""
    r = run_case(script, bash, {}, ["gone.go"])
    need(r.rc == 0, "T6: a missing path alone: exit 0 expected", r)
    need(NOTICE % "gone.go" in r.err.split("\n"), "T6: missing path alone: notice missing", r)
    need(r.calls == [], "T6: missing path alone: nothing else should run", r)

    r = run_case(script, bash, {"a.go": CLEAN}, ["gone.go", "a.go"])
    need(r.rc == 0, "T6: missing path with a clean file: exit 0 expected", r)
    need(NOTICE % "gone.go" in r.err.split("\n"), "T6: missing path with a file: notice missing", r)
    need("go-precheck: 1 file(s) clean" in r.out, "T6: N must count existing files only", r)

    r = run_case(
        script, bash, {"a.go": CLEAN, "removed.go": CLEAN}, [],
        track=("a.go", "removed.go"), remove=("removed.go",))
    need(r.rc == 0, "T6: no-argument mode, tracked file removed: exit 0 expected", r)
    need(NOTICE % "removed.go" in r.err.split("\n"), "T6: no-argument mode: notice missing", r)
    need("go-precheck: 1 file(s) clean" in r.out, "T6: no-argument mode: N must count existing files only", r)


def t7(script: Path, bash: str) -> None:
    """G9, G10: no Go file exits 0 at once; non-Go arguments are ignored."""
    r = run_case(script, bash, {"README.md": "x\n"}, ["README.md"])
    need(r.rc == 0, "T7: docs only: exit 0 expected", r)
    need(r.calls == [], "T7: docs only: no go or golangci-lint call expected", r)
    r = run_case(script, bash, {"README.md": "x\n", "a.go": CLEAN}, ["README.md", "a.go"])
    need(r.rc == 0, "T7: mixed: exit 0 expected (gofmt must see only the .go)", r)
    need("go-precheck: 1 file(s) clean" in r.out, "T7: mixed: one file expected", r)


GOFMT_MISSING = "go-precheck: gofmt not found in PATH."


def _check_gofmt_missing(r: Result, tag: str) -> None:
    """G11: exit 2, and standard error is exactly the guard's line.

    The guard must be what stops the script. Without the guard the script goes
    on: with gofmt hidden alone the gofmt run fails (exit 1, a `gofmt: failed`
    block); with gofmt's directory also holding `mktemp` the `mktemp` call
    fails (exit 2, a `command not found` line). Exact standard error rejects
    both.
    """
    need(r.rc == 2, "%s: exit 2 expected" % tag, r)
    need(GOFMT_MISSING in r.err, "%s: message missing" % tag, r)
    need(r.err.strip() == GOFMT_MISSING,
         "%s: standard error must be the guard's line and nothing else" % tag, r)


def t8(script: Path, bash: str) -> None:
    """G11: gofmt missing from PATH."""
    r = run_case(script, bash, {"a.go": CLEAN}, ["a.go"], SKIP_BOTH, no_gofmt=True)
    _check_gofmt_missing(r, "T8")


def _shared_tool_dir(td: Path) -> str:
    """A directory holding a gofmt together with `mktemp`, `sed`, `rm` and `git`.

    Mimics a runner where gofmt sits in a system directory: hiding it hides the
    tools the script needs next. gofmt is a copy of the real one; the others
    are one-line `sh` wrappers (no symlinks, which are fragile on Windows).
    """
    real = shutil.which("gofmt")
    if not real:
        raise HarnessError("gofmt is not on PATH; the harness needs the real one")
    d = td / "shared"
    d.mkdir()
    shutil.copy2(real, d / os.path.basename(real))
    for tool in ("mktemp", "sed", "rm", "git"):
        exe = shutil.which(tool)
        if exe is None:
            raise HarnessError("%s is not on PATH; the harness needs it" % tool)
        with open(d / tool, "w", newline="\n") as fh:
            fh.write('#!/bin/sh\nexec "%s" "$@"\n' % Path(exe).as_posix())
        (d / tool).chmod(0o755)
    return str(d)


def t8b(script: Path, bash: str) -> None:
    """G11 on CI's layout: gofmt shares a directory with mktemp and the other tools."""
    with tempfile.TemporaryDirectory(prefix="goprecheck-shared-") as td:
        r = run_case(script, bash, {"a.go": CLEAN}, ["a.go"], SKIP_BOTH,
                     no_gofmt=True, base_path=_shared_tool_dir(Path(td)))
    _check_gofmt_missing(r, "T8b")


def t8_second_gofmt() -> None:
    """The fix matters: with a second gofmt on PATH, hiding only the first
    install's directory (the old T8 logic) leaves gofmt visible; hiding every
    directory that holds one does not."""
    real = shutil.which("gofmt")
    if not real:
        raise HarnessError("gofmt is not on PATH; the harness needs the real one")
    with tempfile.TemporaryDirectory(prefix="goprecheck-gofmt2-") as td:
        shutil.copy2(real, Path(td) / os.path.basename(real))
        path = td + os.pathsep + os.environ.get("PATH", "")
        first = os.path.dirname(shutil.which("gofmt", path=path))
        old = os.pathsep.join(p for p in path.split(os.pathsep) if p and not _same_dir(p, first))
        need(shutil.which("gofmt", path=old) is not None,
             "T8 second gofmt: the old logic should still see a gofmt")
        need(shutil.which("gofmt", path=_without_gofmt(path)) is None,
             "T8 second gofmt: every gofmt directory should be hidden")


def bash_version(bash: str) -> tuple[int, int]:
    p = subprocess.run(
        [bash, "-c", "echo ${BASH_VERSINFO[0]} ${BASH_VERSINFO[1]}"], capture_output=True)
    try:
        major, minor = _norm(p.stdout).split()
        return int(major), int(minor)
    except ValueError as exc:
        raise HarnessError("cannot read the version of %s" % bash) from exc


def t9(script: Path, bash: str) -> None:
    """G16: under a bash older than 4.4 an empty array under set -u is an error."""
    r = run_case(script, bash, {"a.go": CLEAN}, ["a.go"], SKIP_BOTH)
    need(r.rc == 0 and "unbound variable" not in r.err,
         "T9: unbound variable or failure under bash %d.%d" % bash_version(bash), r)


TESTS: list[tuple[str, Callable[[Path, str], None]]] = [
    ("T1", t1), ("T2", t2), ("T3", t3), ("T4", t4), ("T5", t5),
    ("T6", t6), ("T7", t7), ("T8", t8), ("T8b", t8b),
]


def find_bashes(first: str) -> list[str]:
    out: list[str] = []
    for cand in (first, "/bin/bash"):
        exe = shutil.which(cand) if not os.path.isabs(cand) else (cand if os.path.exists(cand) else None)
        if exe and not any(_same_dir(exe, o) for o in out):
            out.append(exe)
    if not out:
        raise HarnessError("no bash found")
    return out


def old_bashes(bashes: list[str]) -> list[str]:
    return [b for b in bashes if bash_version(b) < (4, 4)]


def run_suite(script: Path, bashes: list[str], only: str | None = None) -> tuple[set[str], list[str], bool]:
    """Return (names of failing tests, report lines, whether T9 ran)."""
    failed: set[str] = set()
    lines: list[str] = []
    for name, fn in TESTS:
        if only and name != only:
            continue
        for b in bashes:
            try:
                fn(script, b)
                lines.append("ok   %s [%s]" % (name, b))
            except Fail as exc:
                failed.add(name)
                lines.append("FAIL %s [%s] %s" % (name, b, exc))
    ran9 = False
    if only in (None, "T9"):
        olds = old_bashes(bashes)
        for b in olds:
            ran9 = True
            try:
                t9(script, b)
                lines.append("ok   T9 [%s %d.%d]" % ((b,) + bash_version(b)))
            except Fail as exc:
                failed.add("T9")
                lines.append("FAIL T9 [%s] %s" % (b, exc))
        if not olds:
            lines.append("skip T9: no bash older than 4.4 on this host (not a pass)")
    return failed, lines, ran9


# Each plant: id, the test that must turn red, text in the fixed script (must
# occur once), and its replacement.
PLANTS: list[tuple[str, str, str, str]] = [
    ("PL1", "T1", 'if [ "$gofmt_rc" -ne 0 ]; then', "if false; then"),
    ("PL2", "T5", "clean (${ran:-no step ran}).", "clean (gofmt, golangci-lint, govulncheck)."),
    ("PL3", "T6", 'echo "go-precheck: skipping $1: no such file (deleted?)" >&2', ":"),
    ("PL4", "T7", '[ "${#files[@]}" -eq 0 ] && exit 0', '[ "${#files[@]}" -eq 0 ] && true'),
    ("PL5", "T8", "need gofmt || exit 2", "need gofmt || true"),
    ("PL5b", "T8b", "need gofmt || exit 2", "need gofmt || true"),
    ("PL6", "T2", "sed 's/^/  /' >&2\n    failed=1", "sed 's/^/  /' >&2"),
    ("PL7", "T9", "\nfailed=0\n", '\nfailed=0\nnone=()\n: "${none[@]}"\n'),
]


def apply_plant(text: str, old: str, new: str) -> str:
    assert text.count(old) == 1, "plant text must occur once in the script: %r" % old
    return text.replace(old, new)


def self_test(script: Path, bashes: list[str]) -> int:
    failed, lines, _ = run_suite(script, bashes)
    print("\n".join(lines))
    bad = 0
    if failed:
        print("self-test: the script fails %s" % ", ".join(sorted(failed)))
        bad += 1
    try:
        t8_second_gofmt()
        print("ok   T8 second gofmt: old logic fooled, new logic hides every gofmt")
    except Fail as exc:
        bad += 1
        print("FAIL %s" % exc)
    olds = old_bashes(bashes)
    src = script.read_text()
    with tempfile.TemporaryDirectory(prefix="goprecheck-plant-") as td:
        for pid, test, old, new in PLANTS:
            if test == "T9" and not olds:
                print("plant %s: not catchable here (no bash older than 4.4); n/a" % pid)
                continue
            copy = Path(td) / ("plant-%s.sh" % pid)
            with open(copy, "w", newline="\n") as fh:
                fh.write(apply_plant(src, old, new))
            red, plines, _ = run_suite(copy, bashes, only=test)
            if test in red:
                print("plant %s: caught by %s" % (pid, test))
            else:
                bad += 1
                print("plant %s: NOT caught: %s stayed green\n%s" % (pid, test, "\n".join(plines)))
    print("self-test: %s" % ("FAILED" if bad else "ok"))
    return 1 if bad else 0


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n", 1)[0])
    ap.add_argument("--script", type=Path, default=DEFAULT_SCRIPT)
    ap.add_argument("--bash", default="bash", help="the bash under test first (default: bash on PATH)")
    ap.add_argument("--self-test", action="store_true")
    ns = ap.parse_args(argv)
    script = ns.script.resolve()
    if not script.is_file():
        print("check_go_precheck: no script at %s" % script, file=sys.stderr)
        return 2
    try:
        bashes = find_bashes(ns.bash)
        if ns.self_test:
            return self_test(script, bashes)
        failed, lines, _ = run_suite(script, bashes)
    except HarnessError as exc:
        print("check_go_precheck: harness error: %s" % exc, file=sys.stderr)
        return 2
    print("\n".join(lines))
    if failed:
        print("check_go_precheck: FAILED %s" % ", ".join(sorted(failed)))
        return 1
    print("check_go_precheck: ok")
    return 0


if __name__ == "__main__":
    with contextlib.suppress(BrokenPipeError):
        sys.exit(main(sys.argv[1:]))
