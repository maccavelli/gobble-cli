#!/usr/bin/env python3
"""Hold every staticcheck suppression to the house idiom: recorded and paired.

docs/decisions/0002-PLAN-cli-acp-headless-mcp-v1.md (Phase 5, deviation 3)
set the idiom: make preflight runs staticcheck alone, which reads only
`//lint:ignore`, and golangci-lint, which reads only `//nolint`, so a line
that must be suppressed carries both:

    //lint:ignore <CHECK> <reason> (<NNNN-MADR or NNNN-PLAN record>)
    <the line> //nolint:staticcheck // as the lint:ignore above

docs/decisions/0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md
D6 (amended by its PLAN's deviation 3) puts acp-go-sdk2's unstable-schema
names under the same idiom, and has this script hold it for every
suppression in the module:

  1. every `//lint:ignore` and `//lint:file-ignore` in a tracked Go file
     cites a record (NNNN-MADR or NNNN-PLAN) in its reason;
  2. every `//lint:ignore` is followed by a line carrying
     `//nolint:staticcheck`, and every `//nolint:` naming staticcheck
     follows a `//lint:ignore`, so neither linter is silenced alone;
  3. `staticcheck -show-ignored -f json ./...` reports no ignored finding
     but under such a directive, of the check that directive names.

Findings that are not ignored are staticcheck's own run's business. GOOS is
inherited, so `make staticcheck` runs this once per GOOS.

Usage: scripts/check_staticcheck_ignored.py <staticcheck binary>
       (from the repository root)

Exit status is 1 when a suppression breaks the idiom, and 2 on a usage error.
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys

DIRECTIVE = re.compile(r"//lint:(ignore|file-ignore)\s+(\S+)\s*(.*)")
RECORD = re.compile(r"\b\d{4}-(?:MADR|PLAN)\b")
NOLINT = re.compile(r"//nolint:([A-Za-z0-9_,-]+)")


def scan() -> tuple[list[str], dict[tuple[str, int], set[str]], dict[str, set[str]]]:
    """Directives that break rules 1 and 2; the checks each line-directive names, by the (file, line) it
    covers; each file's file-wide checks. A directive may name several checks, comma-separated."""
    files = subprocess.run(["git", "ls-files", "*.go"], capture_output=True, text=True, check=True).stdout.split()
    bad: list[str] = []
    covers: dict[tuple[str, int], set[str]] = {}
    file_wide: dict[str, set[str]] = {}
    for path in files:
        with open(path, encoding="utf-8") as fh:
            lines = fh.read().splitlines()
        for i, line in enumerate(lines):
            n = i + 1
            m = DIRECTIVE.search(line)
            if m:
                kind, check, reason = m.groups()
                if not RECORD.search(reason):
                    bad.append(f"{path}:{n}: the reason cites no NNNN-MADR or NNNN-PLAN record: {line.strip()}")
                if kind == "file-ignore":
                    file_wide.setdefault(path, set()).update(check.split(","))
                    continue
                nxt = lines[i + 1] if i + 1 < len(lines) else ""
                lint = NOLINT.search(nxt)
                if not lint or "staticcheck" not in lint.group(1).split(","):
                    bad.append(f"{path}:{n + 1}: no //nolint:staticcheck under the //lint:ignore at line {n}")
                covers[(path, n + 1)] = set(check.split(","))
            lint = NOLINT.search(line)
            if lint and "staticcheck" in lint.group(1).split(","):
                above = lines[i - 1] if i else ""
                if not DIRECTIVE.search(above) or "file-ignore" in above:
                    bad.append(f"{path}:{n}: //nolint:staticcheck with no //lint:ignore on the line above")
    return bad, covers, file_wide


def bad_ignored(staticcheck: str, covers: dict[tuple[str, int], set[str]], file_wide: dict[str, set[str]]) -> tuple[list[str], int]:
    """Ignored findings not under a directive of their own check (rule 3), and the count that are."""
    r = subprocess.run([staticcheck, "-show-ignored", "-f", "json", "./..."], capture_output=True, text=True)
    root = os.getcwd()
    bad, ok = [], 0
    for line in r.stdout.splitlines():
        try:
            rec = json.loads(line)
        except json.JSONDecodeError:
            bad.append(f"unreadable staticcheck output: {line[:200]}")
            continue
        if rec.get("severity") != "ignored":
            continue
        loc = rec.get("location", {})
        path = os.path.relpath(loc.get("file", ""), root).replace(os.sep, "/")
        code = rec.get("code")
        if code in covers.get((path, loc.get("line")), set()) | file_wide.get(path, set()):
            ok += 1
            continue
        bad.append(f"{path}:{loc.get('line')}: ignored {code} under no directive of that check: {rec.get('message', '')[:160]}")
    if r.returncode not in (0, 1) or (not r.stdout and r.stderr.strip()):
        bad.append(f"staticcheck -show-ignored exited {r.returncode}: {r.stderr.strip()[:300]}")
    return bad, ok


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: scripts/check_staticcheck_ignored.py <staticcheck binary>", file=sys.stderr)
        return 2
    bad, covers, file_wide = scan()
    ignored, ok = bad_ignored(sys.argv[1], covers, file_wide)
    for b in bad + ignored:
        print(f"suppression outside the idiom (0002-PLAN Phase 5 deviation 3; 0035-MADR D6): {b}")
    if bad or ignored:
        return 1
    print(f"check_staticcheck_ignored: {len(covers)} directive(s), {ok} ignored finding(s), each recorded and paired")
    return 0


if __name__ == "__main__":
    sys.exit(main())
