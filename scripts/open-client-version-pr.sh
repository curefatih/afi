#!/usr/bin/env bash
# Push release tags and open + auto-merge a PR for version-file commits.
# main is PR-protected, so the workflow must not push commits to main.
#
# Usage (CI):
#   bash scripts/open-client-version-pr.sh
#
# Env:
#   GITHUB_REF_NAME — base branch (default: main)
#   GH_TOKEN / GITHUB_TOKEN — required for gh (contents + pull-requests write)
#   AUTO_MERGE=1    — default; set 0 to only open the PR
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

BASE_BRANCH="${GITHUB_REF_NAME:-main}"
PR_BRANCH="chore/clients-version-bump"
AUTO_MERGE="${AUTO_MERGE:-1}"

echo "==> Push release tags"
git push origin --tags

git fetch origin "${BASE_BRANCH}"

ahead="$(git rev-list --count "origin/${BASE_BRANCH}..HEAD" 2>/dev/null || echo 0)"
if [[ "${ahead}" == "0" ]]; then
  echo "No version-bump commits ahead of origin/${BASE_BRANCH} — nothing to PR."
  exit 0
fi

echo "==> Push ${ahead} commit(s) to ${PR_BRANCH}"
git push origin "HEAD:refs/heads/${PR_BRANCH}" --force

if [[ -z "${GH_TOKEN:-${GITHUB_TOKEN:-}}" ]]; then
  echo "WARNING: no GH_TOKEN/GITHUB_TOKEN — branch pushed but PR not opened" >&2
  exit 0
fi
export GH_TOKEN="${GH_TOKEN:-${GITHUB_TOKEN}}"

if ! gh pr view "${PR_BRANCH}" --json number --jq .number >/dev/null 2>&1; then
  echo "==> Open version bump PR"
  gh pr create \
    --base "${BASE_BRANCH}" \
    --head "${PR_BRANCH}" \
    --title "chore(clients): sync published package versions [skip release]" \
    --body "$(cat <<'EOF'
Automated version bump after a successful npm/PyPI client publish.

Opened because `main` is PR-only. This workflow auto-merges the PR when
repository rules allow (see clients/README.md).

`[skip release]` prevents a republish loop on merge.
EOF
)"
fi

url="$(gh pr view "${PR_BRANCH}" --json url --jq .url)"
echo "PR: ${url}"

if [[ "${AUTO_MERGE}" != "1" ]]; then
  echo "AUTO_MERGE=0 — leaving PR open"
  exit 0
fi

echo "==> Auto-merge version bump PR"
# Prefer immediate squash merge (works when the only rule is "PR required").
# Fall back to --auto (queues until required checks pass; needs repo
# "Allow auto-merge" enabled).
if gh pr merge "${PR_BRANCH}" --squash --delete-branch; then
  echo "Merged ${PR_BRANCH} into ${BASE_BRANCH}"
  exit 0
fi

echo "Immediate merge blocked by branch rules — enabling auto-merge…"
if gh pr merge "${PR_BRANCH}" --squash --auto --delete-branch; then
  echo "Auto-merge enabled for ${url}"
  echo "It will merge once required checks/reviews are satisfied."
  exit 0
fi

cat >&2 <<EOF
WARNING: could not merge ${PR_BRANCH}.
Enable one of:
  1. Repo Settings → General → Allow auto-merge
  2. Ruleset bypass for github-actions[bot] on version-bump PRs
  3. No required reviews/checks for chore/clients-version-bump
PR left open: ${url}
EOF
exit 0
