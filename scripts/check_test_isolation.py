#!/usr/bin/env python3
"""Check that no file above a test binary's temp root changes a test.

docs/decisions/0053-MADR-keep-tests-off-host-files-above-their-temp-root.md
D4: make a root of this script's own under the system's temp directory and
plant in it every file a test was found to read from above its temp root:

  AGENTS.md                    the context-file walk (prompt.ContextFiles)
  .agents/skills/anc/SKILL.md  the project-skill walk (prompt.ProjectSkillDirs)
  .git/HEAD                    the repository finders (fsx.GitRepo, repoTop)
  .gitignore, holding *        the ignore rules read from a repository's top

Then run the whole suite, both modules, with two sibling temp roots below
them, each at least 130 characters long, longer than a 120-column trust
menu fits on one row (0053-MADR F5): TMPDIR, TMP and TEMP name the first,
where os.TempDir() and os.MkdirTemp("", ...) put a directory, and GOTMPDIR
the second, where t.TempDir() puts one when GOTMPDIR is set, overriding
the host's own (0053-MADR A1). Fail on:

  * any failing test or package;
  * any file under the root that is neither planted nor under a temp
    root: a test that wrote above its temp root (0053-MADR F4).

It is a second full test run, so it is `make check-test-isolation`, not
part of preflight.

Usage:
  scripts/check_test_isolation.py [--keep] [--run REGEXP]

--keep leaves the root, and the go test -json logs kept beside it; --run
passes -run to go test.
Exit status is 1 when the check fails, 2 when it cannot run.
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MODULES = [("root", REPO), ("acpext", os.path.join(REPO, "acpext"))]

PLANTS = {
    # About 40 KB: a scripted compaction test sees an ancestor's file only
    # when it moves the system prompt's token count far enough, which 4 KB
    # does or does not depending on the temp root's length (0053-PLAN P1).
    "AGENTS.md": "An ancestor's context file: no test may read it. " * 800 + "\n",
    os.path.join(".agents", "skills", "anc", "SKILL.md"):
        "---\nname: anc\ndescription: An ancestor's skill; no test may find it.\n---\nBody.\n",
    os.path.join(".git", "HEAD"): "ref: refs/heads/main\n",
    ".gitignore": "*\n",
}

# The temp root's length: over F5's limit, and short of Windows's 260 with
# room for the tests' own directories below it.
MIN_TEMP, MAX_TEMP = 130, 160


def temp_root(root: str, name: str) -> str:
    """A directory below root/name whose absolute path is MIN_TEMP to MAX_TEMP long."""
    path = os.path.join(root, name)
    while len(path) < MIN_TEMP:
        path = os.path.join(path, "d" * min(20, MIN_TEMP - len(path)))
    if len(path) > MAX_TEMP:
        raise SystemExit(f"check-test-isolation: the system's temp directory is too long: {root}")
    os.makedirs(path)
    return path


def run_module(name: str, cwd: str, env: dict, log: str, run: str | None) -> tuple[int, dict]:
    """go test -json in cwd; the exit status and the failing top-level tests by package."""
    cmd = ["go", "test", "-count=1", "-timeout=20m", "-json"]
    if run:
        cmd += ["-run", run]
    cmd.append("./...")
    with open(log, "w", encoding="utf-8") as out:
        code = subprocess.run(cmd, cwd=cwd, env=env, stdout=out, stderr=subprocess.STDOUT, check=False).returncode
    failed: dict[str, dict[str, list[str]]] = {}
    output: dict[tuple[str, str], list[str]] = {}
    with open(log, encoding="utf-8") as f:
        for line in f:
            try:
                ev = json.loads(line)
            except json.JSONDecodeError:
                if line.strip():
                    failed.setdefault(f"{name}: (not go test -json output)", {}).setdefault(line.strip()[:200], [])
                continue
            pkg, test, action = ev.get("Package", ev.get("ImportPath", "")), ev.get("Test", ""), ev.get("Action")
            if action == "output" and test:
                output.setdefault((pkg, test), []).append(ev.get("Output", ""))
            elif action == "fail" and test and "/" not in test:
                lines = [l.strip() for l in output.get((pkg, test), []) if "_test.go:" in l]
                failed.setdefault(pkg, {})[test] = lines[:3]
            elif action in ("fail", "build-fail") and not test and pkg not in failed:
                failed.setdefault(pkg, {})["(the package failed: build, setup or TestMain)"] = []
    return code, failed


def strays(root: str, temp: tuple[str, ...]) -> list[str]:
    """Files under root, outside the temp roots, that were not planted."""
    planted = {os.path.normpath(p) for p in PLANTS}
    temps = {os.path.normpath(t) for t in temp}
    out = []
    for d, dirs, files in os.walk(root):
        if os.path.normpath(d) in temps:
            dirs[:] = []  # the tests' own: not walked
            continue
        for f in files:
            rel = os.path.relpath(os.path.join(d, f), root)
            if os.path.normpath(rel) not in planted:
                out.append(rel)
    return sorted(out)


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--keep", action="store_true", help="leave the root and its logs")
    ap.add_argument("--run", help="passed to go test as -run")
    args = ap.parse_args()

    root = tempfile.mkdtemp(prefix="gobble-isolation-")
    logs = tempfile.mkdtemp(prefix="gobble-isolation-logs-")  # beside root, not in it: no log is a stray
    try:
        for rel, text in PLANTS.items():
            p = os.path.join(root, rel)
            os.makedirs(os.path.dirname(p), exist_ok=True)
            with open(p, "w", encoding="utf-8") as f:
                f.write(text)
        tmp, gotmp = temp_root(root, "t"), temp_root(root, "g")
        env = dict(os.environ, TMPDIR=tmp, TMP=tmp, TEMP=tmp, GOTMPDIR=gotmp)
        print(f"check-test-isolation: planted {', '.join(sorted(PLANTS))} above two temp roots: "
              f"TMPDIR, TMP and TEMP of {len(tmp)} characters, GOTMPDIR of {len(gotmp)}")
        bad = False
        for name, cwd in MODULES:
            code, failed = run_module(name, cwd, env, os.path.join(logs, f"go-test-{name}.json"), args.run)
            tests = sum(len(v) for v in failed.values())
            print(f"== {name} module: go test exit {code}; {tests} failing in {len(failed)} package(s)")
            for pkg in sorted(failed):
                for test, lines in sorted(failed[pkg].items()):
                    print(f"   FAIL {pkg} {test}")
                    for l in lines:
                        print(f"        {l[:240]}")
            if code != 0 or failed:
                bad = True
        left = strays(root, (tmp, gotmp))
        print(f"== files written above the temp roots: {len(left)}")
        for rel in left:
            print(f"   {rel}")
        if left:
            bad = True
        if args.keep:
            print(f"check-test-isolation: kept {root} and the logs in {logs}")
        print("check-test-isolation: " + ("FAILED" if bad else "ok"))
        return 1 if bad else 0
    finally:
        if not args.keep:
            shutil.rmtree(root, ignore_errors=True)
            shutil.rmtree(logs, ignore_errors=True)


if __name__ == "__main__":
    sys.exit(main())
