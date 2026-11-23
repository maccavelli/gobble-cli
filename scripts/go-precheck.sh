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
# list). Arguments that hold no Go file exit 0 at once, before lint and
# govulncheck, so a docs-only add needs no network (0064-MADR, state table S1).
# Otherwise lint runs over both modules whole regardless of the argument list;
# gofmt still scopes to the files.
#
# A .go argument that is not a file (a deleted path, a mistyped one) is skipped
# with a notice on standard error and is not counted; the same in the
# no-argument mode for a tracked file missing from the working tree
# (0069-MADR, row G8). gofmt's own failure, such as a file that does not
# parse, fails the run. The closing line names only the steps that ran.
#
# Exit codes: 0 all clear · 1 a check failed · 2 a required tool is missing.
#
# Env:
#   GO_PRECHECK_SKIP_VULN=1   skip govulncheck (offline work; the pre-commit
#                             hook still runs it)
#   GO_PRECHECK_SKIP_LINT=1   skip the golangci-lint step; `make preflight` sets
#                             it because its own `lint` step follows
#                             (0064-MADR, state table S12)
set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT" || exit 1

files=()

# A Go path that is not a file (a deletion being staged, a mistyped path, a
# path written relative to a subdirectory: the script works from the
# repository root) is skipped with a notice, in both modes (0069-MADR, row G8).
add_go() {
  if [ -f "$1" ]; then
    files+=("$1")
  else
    echo "go-precheck: skipping $1: no such file (deleted?)" >&2
  fi
}

if [ "$#" -gt 0 ]; then
  for f in "$@"; do
    case "$f" in
    *.go) add_go "$f" ;;
    esac
  done
  # Arguments that contain no Go files are not an error: a mixed `git add` of
  # docs and other files has nothing for this script to say.
  [ "${#files[@]}" -eq 0 ] && exit 0
else
  while IFS= read -r f; do
    [ -n "$f" ] && add_go "$f"
  done < <(git ls-files '*.go')
fi

need() {
  command -v "$1" >/dev/null 2>&1 && return 0
  echo "go-precheck: $1 not found in PATH." >&2
  return 1
}

failed=0

# The steps that ran, for the closing line (a string: an empty array under
# set -u is an error before bash 4.4).
ran=""
ran_step() { ran="${ran:+$ran, }$1"; }

if [ "${#files[@]}" -gt 0 ]; then
  # 1. gofmt. Plain gofmt, not gofumpt. Its own exit status counts: a file
  # that does not parse prints to standard error, lists nothing, and exits 2.
  need gofmt || exit 2
  ran_step gofmt
  gofmt_err="$(mktemp)" || exit 2
  trap 'rm -f "$gofmt_err"' EXIT
  unformatted="$(gofmt -l "${files[@]}" 2>"$gofmt_err")"
  gofmt_rc=$?
  if [ "$gofmt_rc" -ne 0 ]; then
    echo "gofmt: failed with exit status $gofmt_rc:" >&2
    sed 's/^/  /' "$gofmt_err" >&2
    failed=1
  fi
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
if [ "${GO_PRECHECK_SKIP_LINT:-0}" = "1" ]; then
  echo "golangci-lint: skipped (GO_PRECHECK_SKIP_LINT=1)" >&2
else
  need golangci-lint || exit 2
  ran_step golangci-lint
  for m in . acpext; do
    out="$( (cd "$m" && golangci-lint run --config "$REPO_ROOT/.golangci.yml" ./...) 2>&1)" || {
      failed=1
      # A module's output is printed whole, and only when its run fails.
      echo "golangci-lint ($m):" >&2
      printf '%s\n' "$out" | sed 's/^/  /' >&2
    }
  done
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
  ran_step govulncheck
  go tool govulncheck ./...
  case $? in
  0) ;;
  *) [ "$failed" -eq 0 ] && failed=1 ;;
  esac
fi

if [ "$failed" -eq 0 ]; then
  # N counts the Go files checked (existing files only); the parenthesis names
  # only the steps that ran, a skipped step is dropped.
  echo "go-precheck: ${#files[@]} file(s) clean (${ran:-no step ran})."
fi
exit "$failed"
