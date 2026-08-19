#!/usr/bin/env bash
# Determine base branch for cnspec's mql bump PR.
#
# When main's mql import major matches the incoming mql release's major,
# the bump PR opens against main. Otherwise it opens against the v{major}
# support branch (e.g. v13.35.1 while main is on mql v14 -> v13).
#
# Inputs:
#   $1 (required)  incoming mql version (e.g. v13.35.1 or 14.0.0-rc.1)
#   $MAIN_GOMOD    content of main's go.mod. If unset, fetched via `gh api`
#                  using $GH_REPO (owner/repo) and $GH_TOKEN.
#
# Output: prints the base branch to stdout ("main" or "v{major}").
#         Exits non-zero if main's mql import major cannot be determined.
set -euo pipefail

MQL_VERSION="${1:?mql version required as first argument}"

V="${MQL_VERSION#v}"
RELEASE_MAJOR="${V%%.*}"

if [ -z "${MAIN_GOMOD:-}" ]; then
  : "${GH_REPO:?GH_REPO or MAIN_GOMOD required}"
  MAIN_GOMOD=$(gh api "/repos/${GH_REPO}/contents/go.mod?ref=main" --jq '.content' | base64 -d)
fi

MAIN_MAJOR=$(printf '%s' "$MAIN_GOMOD" \
  | grep -oE 'go\.mondoo\.com/mql/v[0-9]+' \
  | head -1 \
  | grep -oE '[0-9]+$' || true)

if [ -z "$MAIN_MAJOR" ]; then
  echo "Could not extract mql major from main's go.mod" >&2
  exit 1
fi

if [ "$RELEASE_MAJOR" = "$MAIN_MAJOR" ]; then
  echo main
else
  echo "v${RELEASE_MAJOR}"
fi
