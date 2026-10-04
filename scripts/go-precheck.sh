#!/usr/bin/env bash
# Go pre-add checks: gofmt (not gofumpt), per-file golint, and govulncheck.
#
# Same contract as the fleet script and the global pre-add hook: called from
# `make pre-add-check`. govulncheck runs from the module tool directive
# (`go tool govulncheck`), not from a global install.
#
# Usage:
#   scripts/go-precheck.sh [file.go ...]
#
# With no arguments it checks every tracked Go file; with arguments, only those
# (non-Go arguments are ignored, so callers can pass a whole changed-file list).
#
# Exit codes: 0 all clear · 1 a check failed · 2 a required tool is missing.
#
# Env:
#   GO_PRECHECK_SKIP_VULN=1   skip govulncheck (offline work; the pre-commit
#                             hook still runs it)
set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT" || exit 1

files=()
if [ "$#" -gt 0 ]; then
  for f in "$@"; do
    case "$f" in
    *.go) [ -f "$f" ] && files+=("$f") ;;
    esac
  done
  # Arguments that contain no Go files are not an error: a mixed `git add` of
  # docs and other files has nothing for this script to say.
  [ "${#files[@]}" -eq 0 ] && exit 0
else
  while IFS= read -r f; do
    [ -n "$f" ] && files+=("$f")
  done < <(git ls-files '*.go')
fi

need() {
  command -v "$1" >/dev/null 2>&1 && return 0
  echo "go-precheck: $1 not found in PATH." >&2
  return 1
}

failed=0

if [ "${#files[@]}" -eq 0 ]; then
  echo "go-precheck: no Go files to format or lint." >&2
else
  # 1. gofmt. Plain gofmt, not gofumpt.
  need gofmt || exit 2
  unformatted="$(gofmt -l "${files[@]}")"
  if [ -n "$unformatted" ]; then
    echo "gofmt: these files are not formatted (run 'make fmt' or 'gofmt -w <file>'):" >&2
    printf '%s\n' "$unformatted" | sed 's/^/  /' >&2
    failed=1
  fi

  # 2. golint, per file so the output names what to fix.
  # The binary is the module tool directive, not a global install.
  if ! go tool golint -h >/dev/null 2>&1; then
    echo "go-precheck: go tool golint is not available (tool directive golang.org/x/lint/golint)." >&2
    exit 2
  fi
  lint_out=""
  for f in "${files[@]}"; do
    out="$(go tool golint "$f" 2>&1)" || true
    [ -n "$out" ] && lint_out+="$out"$'\n'
  done
  if [ -n "$lint_out" ]; then
    echo "golint:" >&2
    printf '%s' "$lint_out" | sed 's/^/  /' >&2
    failed=1
  fi
fi

# 3. govulncheck, over the module. It is a property of the whole build rather
# than of the edited files. GO_PRECHECK_SKIP_VULN=1 skips it.
if [ "${GO_PRECHECK_SKIP_VULN:-0}" = "1" ]; then
  echo "govulncheck: skipped (GO_PRECHECK_SKIP_VULN=1)" >&2
else
  if ! go tool govulncheck -version >/dev/null 2>&1; then
    echo "go-precheck: go tool govulncheck is not available (tool directive golang.org/x/vuln/cmd/govulncheck)." >&2
    exit 2
  fi
  go tool govulncheck ./...
  case $? in
  0) ;;
  *) [ "$failed" -eq 0 ] && failed=1 ;;
  esac
fi

if [ "$failed" -eq 0 ]; then
  echo "go-precheck: ${#files[@]} file(s) clean (gofmt, golint, govulncheck)."
fi
exit "$failed"
