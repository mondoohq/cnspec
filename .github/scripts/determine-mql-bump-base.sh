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
#      pseudo-versions, so the path carries no major to compare against.
#      Both repos instead branch a major off when main moves to the next
#      one (mql and cnspec both cut v13 on 2026-08-18), so the branches
#      answer the question the path no longer can:
#
#        cnspec has v{major}              -> v{major}, the support branch
#        mql has v{major}, cnspec has not -> error; the release comes from
#                                            an mql maintenance line whose
#                                            cnspec branch is missing, and
#                                            landing it on main would
#                                            downgrade main's mql
#        neither has v{major}             -> main, which is the only line
#                                            this major can belong to
#
#      Nothing here is pinned to a specific major, so v15 needs no change:
#      once mql and cnspec cut v14 and main moves on, v14.x releases route
#      to v14 and v15.x releases route to main.
#
# Inputs:
#   $1 (required)    incoming mql version (e.g. v13.36.0 or 14.0.0-rc.1)
#   $MAIN_GOMOD      content of main's go.mod. If unset, fetched via `gh api`
#                    using $GH_REPO (owner/repo) and $GH_TOKEN.
#   $KNOWN_BRANCHES  space/newline separated cnspec branch names to check the
#                    support branch against. If unset, existence is queried
#                    via `gh api` using $GH_REPO and $GH_TOKEN.
#   $MQL_BRANCHES    same, for mql's branches. If unset, queried via `gh api`
#                    using $MQL_REPO (default mondoohq/mql).
#
# Output: prints the base branch to stdout ("main" or "v{major}").
#         Exits non-zero if the base branch cannot be determined.
set -euo pipefail

MQL_VERSION="${1:?mql version required as first argument}"
MQL_REPO="${MQL_REPO:-mondoohq/mql}"

V="${MQL_VERSION#v}"
RELEASE_MAJOR="${V%%.*}"

if [ -z "${MAIN_GOMOD+set}" ]; then
  # MAIN_GOMOD not passed in at all — fetch it. An explicit empty value
  # is respected (and will fail the "Could not find" check below), so
  # tests can exercise failure paths without touching the network.
  : "${GH_REPO:?GH_REPO or MAIN_GOMOD required}"
  MAIN_GOMOD=$(gh api "/repos/${GH_REPO}/contents/go.mod?ref=main" --jq '.content' | base64 -d)
fi

# Does branch $2 exist in repo $1? $3 names the env var holding a
# pre-supplied branch list, so tests can answer without the network.
# Only an HTTP 404 counts as "no such branch" — any other API failure is
# fatal, so a bad token or a network blip cannot be read as "not a support
# branch" and quietly route the bump to main.
branch_exists() {
  local repo="$1" branch="$2" list_var="$3" err
  if [ -n "${!list_var+set}" ]; then
    printf '%s' "${!list_var}" | tr ' ' '\n' | grep -qx -- "$branch"
    return
  fi
  : "${repo:?repository required to look up branch ${branch}}"
  if err=$(gh api "/repos/${repo}/branches/${branch}" 2>&1 >/dev/null); then
    return 0
  fi
  if printf '%s' "$err" | grep -q 'HTTP 404'; then
    return 1
  fi
  echo "Could not query ${repo} for branch ${branch}: ${err}" >&2
  exit 1
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
# back to the branch lookup, so a go.mod without mql still fails loudly
# instead of silently routing somewhere.
if ! printf '%s' "$MAIN_GOMOD" | grep -qE 'go\.mondoo\.com/mql[[:space:]]+v[0-9]'; then
  echo "Could not find an mql import in main's go.mod" >&2
  exit 1
fi

GH_REPO="${GH_REPO:-}"
SUPPORT_BRANCH="v${RELEASE_MAJOR}"

if branch_exists "$GH_REPO" "$SUPPORT_BRANCH" KNOWN_BRANCHES; then
  echo "$SUPPORT_BRANCH"
  exit 0
fi

if branch_exists "$MQL_REPO" "$SUPPORT_BRANCH" MQL_BRANCHES; then
  cat >&2 <<EOF
mql ${MQL_VERSION} comes from the ${SUPPORT_BRANCH} maintenance branch of
${MQL_REPO}, but ${GH_REPO:-cnspec} has no ${SUPPORT_BRANCH} branch to open the bump
against. Cut ${SUPPORT_BRANCH} in ${GH_REPO:-cnspec} and re-run — landing this on main
would downgrade the mql main tracks.
EOF
  exit 1
fi

echo main
