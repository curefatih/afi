#!/usr/bin/env bash
# Shared semver helpers for client release scripts.
# shellcheck shell=bash

semver_is_valid() {
  [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]
}

# Print 0 if equal, 1 if $1 > $2, 2 if $1 < $2 (GNU sort -V semantics).
semver_cmp() {
  local a="$1" b="$2"
  if [[ "$a" == "$b" ]]; then
    echo 0
    return
  fi
  local first
  first="$(printf '%s\n%s\n' "$a" "$b" | sort -V | head -n1)"
  if [[ "$first" == "$a" ]]; then
    echo 2
  else
    echo 1
  fi
}

semver_bump_patch() {
  local v="$1"
  local major minor patch rest
  if [[ ! "$v" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(.*)$ ]]; then
    echo "invalid semver: ${v}" >&2
    return 1
  fi
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  patch="${BASH_REMATCH[3]}"
  rest="${BASH_REMATCH[4]}"
  if [[ -n "$rest" ]]; then
    echo "refusing to auto-bump pre-release version: ${v}" >&2
    return 1
  fi
  echo "${major}.${minor}.$((patch + 1))"
}

# Choose the version to publish given local and registry versions.
# - no registry version → use local
# - local > registry → use local
# - local <= registry → bump patch from registry (so any path change can release)
semver_next_publish() {
  local local_v="$1"
  local published_v="${2:-}"
  if [[ -z "$published_v" ]]; then
    echo "$local_v"
    return
  fi
  case "$(semver_cmp "$local_v" "$published_v")" in
    1) echo "$local_v" ;;
    *) semver_bump_patch "$published_v" ;;
  esac
}
