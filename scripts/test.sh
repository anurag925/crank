#!/bin/sh
# test.sh — run crank's test suites.
#
# Usage:
#   ./scripts/test.sh [target]
#
# Targets:
#   unit         Fast in-process tests (registry, context, manifest, utils, ...).
#   integration  Template-generation tests in internal/bootstrap.
#   e2e          End-to-end tests: build the binary + compile generated projects
#                (requires network access; gated by the `e2e` build tag).
#   all          unit + integration + e2e (default).
#   cover        unit + integration with a coverage profile (coverage.out).
#
# The unit and integration suites share the same `go test ./...` invocation
# because they live in the same packages; they are split out here only as a
# convenience for running the fast, network-free subset.

set -eu

cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

target="${1:-all}"

# Packages that contain only fast, network-free tests.
UNIT_PKGS="./internal/... ./cmd/..."

run_fast() {
  echo "==> Running unit + integration tests"
  go test "$@" $UNIT_PKGS
}

run_e2e() {
  echo "==> Running e2e tests (builds binary + compiles generated projects)"
  echo "    note: requires network access for 'go get'"
  go test -tags e2e -timeout 20m ./e2e/...
}

case "$target" in
  unit | integration)
    run_fast
    ;;
  e2e)
    run_e2e
    ;;
  cover)
    echo "==> Running tests with coverage profile -> coverage.out"
    go test -coverprofile=coverage.out $UNIT_PKGS
    go tool cover -func=coverage.out | tail -n 1
    ;;
  all)
    run_fast
    run_e2e
    ;;
  *)
    echo "unknown target: $target" >&2
    echo "valid targets: unit, integration, e2e, all, cover" >&2
    exit 2
    ;;
esac

echo "==> Done."
