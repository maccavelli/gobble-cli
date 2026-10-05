#!/usr/bin/env python3
"""verify-build-metadata (0008-PLAN P4 step 13, deviation 2).

Builds cmd/gobble twice into a temporary directory and checks
`gobble version --json`:

1. Stamped through go-selfupdate-lib's buildinfo variables, the build reports
   the stamped version without its "v", the commit at HEAD and the commit
   time, both of which go build records from the checkout.
2. Unstamped, it reports 0.0.0-dev+<12 hex>, with ".dirty" exactly when the
   working tree has changes (0008-MADR D19 item 1, 0008-PLAN A15).

Standard library only. Exit status 0 when every check passes, 1 otherwise.
"""
from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
# The two stamps of go-selfupdate-lib buildinfo; the Makefile's LDFLAGS uses
# the same names.
VERSION_VAR = "github.com/maccavelli/go-selfupdate-lib/buildinfo.version"
KIND_VAR = "github.com/maccavelli/go-selfupdate-lib/buildinfo.kind"
STAMP = "v1.2.3"
DEV_RE = re.compile(r"^0\.0\.0-dev\+[0-9a-f]{12}(\.dirty)?$")


@dataclass
class Check:
    name: str
    ok: bool
    detail: str


def git(*args: str) -> str:
    r = subprocess.run(["git", *args], cwd=ROOT, capture_output=True, text=True, check=True, timeout=60)
    return r.stdout.strip()


def build(out: Path, ldflags: str | None) -> None:
    cmd = ["go", "build", "-o", str(out)]
    if ldflags:
        cmd += ["-ldflags", ldflags]
    cmd.append("./cmd/gobble")
    env = {**os.environ, "CGO_ENABLED": "0"}
    r = subprocess.run(cmd, cwd=ROOT, env=env, capture_output=True, text=True, timeout=600)
    if r.returncode != 0:
        raise SystemExit(f"verify-build-metadata: go build failed:\n{r.stdout}{r.stderr}")


def version_json(binary: Path) -> dict[str, str]:
    r = subprocess.run([str(binary), "version", "--json"], cwd=ROOT, stdin=subprocess.DEVNULL,
                       capture_output=True, text=True, timeout=60)
    if r.returncode != 0:
        raise SystemExit(f"verify-build-metadata: {binary.name} version --json exited {r.returncode}:\n{r.stderr}")
    return json.loads(r.stdout)


def commit_time() -> str:
    return datetime.fromtimestamp(int(git("log", "-1", "--format=%ct")), UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def checks(tmp: Path) -> list[Check]:
    exe = ".exe" if sys.platform == "win32" else ""
    head, when = git("rev-parse", "HEAD"), commit_time()
    dirty = git("status", "--porcelain") != ""

    stamped = tmp / f"gobble-stamped{exe}"
    build(stamped, f"-X {VERSION_VAR}={STAMP} -X {KIND_VAR}=local")
    s = version_json(stamped)

    plain = tmp / f"gobble-unstamped{exe}"
    build(plain, None)
    u = version_json(plain)

    return [
        Check("stamped version", s.get("version") == STAMP.removeprefix("v"), f"got {s.get('version')!r}, want {STAMP.removeprefix('v')!r}"),
        Check("stamped commit is HEAD", s.get("commit") == head, f"got {s.get('commit')!r}, want {head!r}"),
        Check("stamped date is the commit time", s.get("date") == when, f"got {s.get('date')!r}, want {when!r}"),
        Check("name", s.get("name") == "gobble", f"got {s.get('name')!r}"),
        Check("unstamped version is 0.0.0-dev+<rev>", bool(DEV_RE.match(u.get("version", ""))), f"got {u.get('version')!r}"),
        Check("unstamped version names HEAD", u.get("version", "").startswith(f"0.0.0-dev+{head[:12]}"), f"got {u.get('version')!r}, HEAD {head[:12]}"),
        Check("unstamped .dirty matches the tree", u.get("version", "").endswith(".dirty") == dirty,
              f"got {u.get('version')!r}, tree {'dirty' if dirty else 'clean'}"),
    ]


def main() -> int:
    with tempfile.TemporaryDirectory(prefix="gobble-verify-") as d:
        results = checks(Path(d))
    for c in results:
        print(f"verify-build-metadata: {'ok  ' if c.ok else 'FAIL'} {c.name}" + ("" if c.ok else f": {c.detail}"))
    return 0 if all(c.ok for c in results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
