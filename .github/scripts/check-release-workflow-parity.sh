#!/usr/bin/env bash
# Copyright Mondoo, Inc. 2025, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# The release workflows must be identical on main and on every v{major} support
# branch.
#
# They are duplicated for a structural reason, not by choice: a `pull_request`
# workflow resolves from the merge commit, and a bump PR's head branches off its
# base, so a file that exists only on main never runs for a v13 PR. The copies
# have to exist. They are not supposed to differ.
#
# Left unchecked they drift, and the drift is invisible: each branch's file
# looks right on its own. Three divergences appeared in two days --
#   - v13 nearly shipped without the guard that refuses to tag a commit which
#     is not on the base branch;
#   - v13 did ship a version that tagged HEAD instead of the merge commit;
#   - v13's goreleaser still classified releases by substring long after main
#     derived the channel from the semver tag.
# Every one was found by hand. This is that check, automated.
set -uo pipefail

# Files that must be byte-identical across release branches.
FILES=(
  ".github/workflows/goreleaser.yml"
  ".github/workflows/auto-tag-after-mql-bump.yml"
)

# How to behave when a difference is found. On a push (post-merge) the branches
# are in their final state and a difference is a real problem, so fail. On a
# pull request the difference may simply be the first half of an intentional
# pair -- change main, then sync v13 -- so report it and let the PR through.
MODE="${1:-strict}"

# Release branches: main, plus every v{major} branch that takes part in the
# current release flow.
#
# Participation is decided by the presence of auto-tag-after-mql-bump.yml
# rather than by a hand-maintained list. That file arrived with the channel
# work, so it marks exactly the branches that use it -- v6 through v9 are end
# of life, were never migrated, and would otherwise make this check
# permanently red for branches nobody intends to touch. A new support branch
# cut from main carries the marker and is picked up with no edit here.
#
# The pattern is anchored: `v` followed by digits and nothing else, so a
# feature branch like chris/v13-something is not mistaken for a release branch.
marker=".github/workflows/auto-tag-after-mql-bump.yml"
branches=("main")
while IFS= read -r b; do
  [[ "$b" =~ ^v[0-9]+$ ]] || continue
  git show "origin/${b}:${marker}" >/dev/null 2>&1 || continue
  branches+=("$b")
done < <(git ls-remote --heads origin | sed 's|.*refs/heads/||' | sort -V)

printf 'Release branches: %s\n\n' "${branches[*]}"

if [ "${#branches[@]}" -lt 2 ]; then
  echo "Only one release branch exists, so there is nothing to compare."
  exit 0
fi

differs=0
for f in "${FILES[@]}"; do
  reference=""
  ref_branch=""
  for b in "${branches[@]}"; do
    if ! blob=$(git show "origin/${b}:${f}" 2>/dev/null); then
      printf '  n/a   %-46s not on %s\n' "$f" "$b"
      continue
    fi
    sum=$(printf '%s' "$blob" | shasum -a 256 | cut -d' ' -f1)
    if [ -z "$reference" ]; then
      reference="$sum"; ref_branch="$b"
      printf '  ref   %-46s %s (%s)\n' "$f" "${sum:0:12}" "$b"
    elif [ "$sum" = "$reference" ]; then
      printf '  ok    %-46s %s (%s)\n' "$f" "${sum:0:12}" "$b"
    else
      printf '  DIFF  %-46s %s (%s) != %s (%s)\n' "$f" "${sum:0:12}" "$b" "${reference:0:12}" "$ref_branch"
      differs=1
    fi
  done
done

echo
if [ "$differs" -eq 0 ]; then
  echo "Release workflows are identical across all release branches."
  exit 0
fi

if [ "$MODE" = "report" ]; then
  cat <<'MSG'
::warning::A release workflow differs between release branches. If this pull
request is the first half of an intentional pair -- change main, then sync the
support branch -- open the follow-up now so it is not forgotten. The same check
runs on push and will fail there until the branches agree.
MSG
  exit 0
fi

cat >&2 <<'MSG'
A release workflow differs between release branches.

These files are duplicated only because a pull_request workflow resolves from
the merge commit, so a file on main alone never runs for a support-branch PR.
They are not meant to differ in content.

To sync a support branch from main:

    git checkout -b sync-release-workflows origin/v13
    git checkout origin/main -- .github/workflows/<file>
    git commit && gh pr create --base v13

Check anything genuinely branch-specific before copying: a pinned action
version, or a step the other branch has no secrets for.
MSG
exit 1
