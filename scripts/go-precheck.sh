#!/usr/bin/env bash
# Go pre-add checks: gofmt (not gofumpt), golangci-lint, and govulncheck.
#
# Same contract as the fleet script and the global pre-add hook: called from
# `make pre-add-check`. govulncheck runs from the module tool directive
# (`go tool govulncheck`), not from a global install.
#
# The lint step is the exact command `make lint` runs over both modules, so
# the two cannot drift; whole-tree, not per-file (0064-MADR; the per-file
# golint step is retired fleet-wide — dotfiles 0028-MADR).
#
# Usage:
#   scripts/go-precheck.sh [file.go ...]
#
# With no arguments it checks the whole tree; with arguments, only those
# (non-Go arguments are ignored, so callers can pass a whole changed-file
# list). Lint runs over both modules whole regardless of the argument list;
# gofmt still scopes to the files.
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
  # docs and other files has nothing for gofmt to say, though lint still runs.
  [ "${#files[@]}" -eq 0 ] && echo "go-precheck: no Go files to format." >&2
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

if [ "${#files[@]}" -gt 0 ]; then
  # 1. gofmt. Plain gofmt, not gofumpt.
  need gofmt || exit 2
  unformatted="$(gofmt -l "${files[@]}")"
  if [ -n "$unformatted" ]; then
    echo "gofmt: these files are not formatted (run 'make fmt' or 'gofmt -w <file>'):" >&2
    printf '%s\n' "$unformatted" | sed 's/^/  /' >&2
    failed=1
  fi
fi

# 2. golangci-lint, the `make lint` command over both modules: whole-tree per
# module, config by absolute path, host GOOS — so a commit cannot pass a
# weaker check than the tree's own lint gate (0064-MADR; the per-file golint
# step is retired — dotfiles 0028-MADR + its amendment).
need golangci-lint || exit 2
lint_out=""
for m in . acpext; do
  out="$( (cd "$m" && golangci-lint run --config "$REPO_ROOT/.golangci.yml" ./...) 2>&1)" || failed=1
  [ -n "$out" ] && lint_out+="$out"$'\n'
done
if [ -n "$lint_out" ]; then
  echo "golangci-lint:" >&2
  printf '%s' "$lint_out" | tail -40 | sed 's/^/  /' >&2
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
  echo "go-precheck: ${#files[@]} file(s) clean (gofmt, golangci-lint, govulncheck)."
fi
exit "$failed"
