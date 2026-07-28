#!/usr/bin/env bash
# Unit tests for scripts/semver.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=semver.sh
source "${ROOT}/scripts/semver.sh"

fail=0
assert_eq() {
  local got="$1" want="$2" label="$3"
  if [[ "$got" != "$want" ]]; then
    echo "FAIL ${label}: got=${got@Q} want=${want@Q}" >&2
    fail=1
  fi
}

assert_eq "$(semver_bump_patch 1.0.0)" "1.0.1" "bump_patch"
assert_eq "$(semver_bump_patch 1.2.9)" "1.2.10" "bump_patch_roll"
assert_eq "$(semver_cmp 1.0.1 1.0.0)" "1" "cmp_gt"
assert_eq "$(semver_cmp 1.0.0 1.0.1)" "2" "cmp_lt"
assert_eq "$(semver_cmp 1.0.0 1.0.0)" "0" "cmp_eq"
assert_eq "$(semver_next_publish 1.0.0 "")" "1.0.0" "next_first"
assert_eq "$(semver_next_publish 1.1.0 1.0.5)" "1.1.0" "next_local_ahead"
assert_eq "$(semver_next_publish 1.0.0 1.0.0)" "1.0.1" "next_same_bump"
assert_eq "$(semver_next_publish 1.0.0 1.0.3)" "1.0.4" "next_behind_bump"

if [[ "${fail}" -ne 0 ]]; then
  echo "semver tests failed" >&2
  exit 1
fi
echo "semver tests passed"
