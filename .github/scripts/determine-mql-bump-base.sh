#!/usr/bin/env bash
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Determine base branch for cnspec's mql bump PR.
#
# Two shapes of main's mql import have to be handled:
#
#   1. Major-suffixed path (go.mondoo.com/mql/v13). The major is read
#      straight off the path: same major as the incoming release -> main,
#      any other major -> the v{major} support branch.
#
#   2. Unversioned path (go.mondoo.com/mql). mql dropped the major suffix
#      for its v14 development line, which cnspec's main tracks through
#      pseudo-versions, so the path no longer carries a major. Released
#      majors still live on their own mql branch, and cnspec mirrors them
#      with a v{major} support branch — so the incoming release routes to
#      v{major} when that branch exists, and to main otherwise (the release
#      belongs to the unversioned development line main is already on).
#
# Inputs:
#   $1 (required)    incoming mql version (e.g. v13.36.0 or 14.0.0-rc.1)
#   $MAIN_GOMOD      content of main's go.mod. If unset, fetched via `gh api`
#                    using $GH_REPO (owner/repo) and $GH_TOKEN.
#   $KNOWN_BRANCHES  space/newline separated branch names to check the
#                    v{major} support branch against. If unset, existence is
#                    queried via `gh api` using $GH_REPO and $GH_TOKEN.
#
# Output: prints the base branch to stdout ("main" or "v{major}").
#         Exits non-zero if main's mql import cannot be found at all.
set -euo pipefail

MQL_VERSION="${1:?mql version required as first argument}"

V="${MQL_VERSION#v}"
RELEASE_MAJOR="${V%%.*}"

if [ -z "${MAIN_GOMOD+set}" ]; then
  # MAIN_GOMOD not passed in at all — fetch it. An explicit empty value
  # is respected (and will fail the "Could not find" check below), so
  # tests can exercise failure paths without touching the network.
  : "${GH_REPO:?GH_REPO or MAIN_GOMOD required}"
  MAIN_GOMOD=$(gh api "/repos/${GH_REPO}/contents/go.mod?ref=main" --jq '.content' | base64 -d)
fi

# Does branch $1 exist in the repo?
branch_exists() {
  if [ -n "${KNOWN_BRANCHES+set}" ]; then
    printf '%s' "$KNOWN_BRANCHES" | tr ' ' '\n' | grep -qx -- "$1"
    return
  fi
  : "${GH_REPO:?GH_REPO or KNOWN_BRANCHES required}"
  gh api "/repos/${GH_REPO}/branches/$1" >/dev/null 2>&1
}

MAIN_MAJOR=$(printf '%s' "$MAIN_GOMOD" \
  | grep -oE 'go\.mondoo\.com/mql/v[0-9]+' \
  | head -1 \
  | grep -oE '[0-9]+$' || true)

if [ -n "$MAIN_MAJOR" ]; then
  if [ "$RELEASE_MAJOR" = "$MAIN_MAJOR" ]; then
    echo main
  else
    echo "v${RELEASE_MAJOR}"
  fi
  exit 0
fi

# No major suffix on main. Confirm main imports mql at all before falling
# back to the support-branch lookup, so a go.mod without mql still fails
# loudly instead of silently routing somewhere.
if ! printf '%s' "$MAIN_GOMOD" | grep -qE 'go\.mondoo\.com/mql[[:space:]]+v[0-9]'; then
  echo "Could not find an mql import in main's go.mod" >&2
  exit 1
fi

if branch_exists "v${RELEASE_MAJOR}"; then
  echo "v${RELEASE_MAJOR}"
else
  echo main
fi
