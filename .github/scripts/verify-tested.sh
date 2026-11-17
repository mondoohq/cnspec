#!/usr/bin/env bash
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Refuse to release a commit whose tests did not pass.
#
# For each workflow in WORKFLOWS, the tagged commit SHA must either have a run
# that concluded `success`, or have no run because nothing that workflow watches
# has changed since the nearest ancestor whose run did succeed.
#
# "No run" used to be read as a verdict on the first query, and it is usually
# not one. It has meant two other things, each of which failed a good release:
#
#   - not yet: the tag push starts its own test run, and this job can ask before
#     GitHub has registered it. It reported "no run" while tests were starting.
#   - not applicable: the test workflows are path-filtered, so a commit that only
#     changes a workflow file never gets a run. v14.0.0-rc.9 was such a commit.
#
# Neither may become a blanket pass. In September a burst of merges cancelled
# five consecutive test runs on main while they were still queued; those
# commits had no verdict at all, and the code in them was never tested. So the
# fallback does not trust "nothing ran" -- it finds the last commit that passed
# and requires that nothing the workflow watches has changed since.
#
# Inputs (environment):
#   REPO               owner/name
#   SHA                the commit being released
#   WORKFLOWS          space-separated workflow files (default below)
#   GRACE_SECONDS      how long to wait for a run to appear (default 300)
#   DEADLINE_SECONDS   how long to wait for a run to finish (default 2700)
#   POLL_SECONDS       polling interval (default 30)
#   MAX_ANCESTORS      how far back to look for a passing run (default 30)
#
# Requires gh and yq. Exits non-zero, with an ::error:: per workflow, when the
# release must not proceed.
set -euo pipefail

: "${REPO:?REPO is required}"
: "${SHA:?SHA is required}"
WORKFLOWS="${WORKFLOWS:-pr-test-lint.yml pr-test-generated-files.yaml}"
GRACE_SECONDS="${GRACE_SECONDS:-300}"
DEADLINE_SECONDS="${DEADLINE_SECONDS:-2700}"
POLL_SECONDS="${POLL_SECONDS:-30}"
MAX_ANCESTORS="${MAX_ANCESTORS:-30}"

# The compare API stops listing files at 300. A diff that large cannot be shown
# to leave watched paths alone, so it is treated as touching them.
COMPARE_FILE_LIMIT=300

for tool in gh yq; do
  command -v "$tool" >/dev/null 2>&1 || {
    echo "::error::verify-tested needs ${tool}, which is not installed."
    exit 1
  }
done

# gh with retries, because a transient API failure must not decide a release
# either way. Prints the response on stdout; returns non-zero only once retries
# are exhausted.
#
# stderr is kept apart from the response. Mixed in, any line gh writes there on a
# successful call -- a notice or a warning -- becomes the first line callers
# parse, and run_state would read "warning:" as the run's status: neither `none`
# nor `completed`, so the gate would wait out its deadline on a green commit.
api() {
  local out err errfile attempt delay=5
  errfile=$(mktemp)
  for attempt in 1 2 3 4 5; do
    if out=$(gh api "$@" 2>"$errfile"); then
      rm -f "$errfile"
      printf '%s\n' "$out"
      return 0
    fi
    err=$(cat "$errfile")
    # A missing commit or file is an answer, not an outage; retrying it only
    # delays the same refusal.
    case "$err" in
      *"HTTP 404"*|*"HTTP 422"*)
        echo "GitHub API: ${err}" >&2
        rm -f "$errfile"
        return 1
        ;;
    esac
    echo "GitHub API call failed (attempt ${attempt}): ${err}" >&2
    sleep "$delay"
    delay=$((delay * 2))
  done
  rm -f "$errfile"
  return 1
}

# Latest run of a workflow for a commit, as "status conclusion", or "none none".
run_state() {
  api "repos/${REPO}/actions/workflows/$1/runs?head_sha=$2&per_page=1" \
    --jq '.workflow_runs[0] | "\(.status // "none") \(.conclusion // "none")"'
}

# The push path filters of a workflow, as it was at the commit being released.
watched_paths() {
  api "repos/${REPO}/contents/.github/workflows/$1?ref=$2" \
    -H "Accept: application/vnd.github.raw" \
    | yq -r '.on.push.paths // [] | .[]'
}

# A commit with no run is acceptable only if the code that workflow tests is
# identical to the code in an ancestor where the workflow passed.
#
# Walks back from SHA to the nearest ancestor whose run succeeded, then checks
# every file changed between that ancestor and SHA against the workflow's path
# filters. Any match means watched code changed after the last green run --
# including changes in intermediate commits whose own runs were cancelled -- so
# the release is refused. Globs are matched with bash's `==`, in which `*` also
# crosses `/`. That can only over-match relative to GitHub, which errs towards
# refusing a release, never towards admitting an untested one.
covered_by_ancestor() {
  local wf=$1 base="" anc state paths files file pattern count

  paths=$(watched_paths "$wf" "$SHA") || {
    echo "::error::Could not read the path filters of ${wf} at ${SHA}, so cannot tell whether it should have run."
    return 1
  }
  if [ -z "$paths" ]; then
    echo "::error::${wf} has no push path filter, so every commit should have a run, and ${SHA} has none."
    return 1
  fi

  # The listing starts with SHA itself, hence one more than MAX_ANCESTORS.
  for anc in $(api "repos/${REPO}/commits?sha=${SHA}&per_page=$((MAX_ANCESTORS + 1))" --jq '.[1:][].sha'); do
    state=$(run_state "$wf" "$anc")
    if [ "$state" = "completed success" ]; then
      base=$anc
      break
    fi
  done

  if [ -z "$base" ]; then
    echo "::error::No ${wf} run for ${SHA}, and none of its last ${MAX_ANCESTORS} ancestors has a passing one either."
    return 1
  fi

  files=$(api "repos/${REPO}/compare/${base}...${SHA}" --jq '.files[].filename')
  count=$(printf '%s\n' "$files" | grep -c . || true)
  if [ "$count" -ge "$COMPARE_FILE_LIMIT" ]; then
    echo "::error::No ${wf} run for ${SHA}, and the diff from its last passing ancestor ${base:0:12} is too large to check."
    return 1
  fi

  while IFS= read -r file; do
    [ -n "$file" ] || continue
    while IFS= read -r pattern; do
      # shellcheck disable=SC2053 # the pattern is meant to be a glob
      if [[ "$file" == $pattern ]]; then
        echo "::error::No ${wf} run for ${SHA}, but ${file} (matching '${pattern}') changed since ${base:0:12}, the last commit it passed on. That code has never been tested."
        return 1
      fi
    done <<<"$paths"
  done <<<"$files"

  echo "::notice::No ${wf} run for ${SHA}: nothing it watches has changed since ${base:0:12}, where it passed (${count} files changed, none matching its path filters)."
  return 0
}

check_workflow() {
  local wf=$1 start now state status conclusion
  start=$(date +%s)

  while :; do
    now=$(date +%s)
    if ! state=$(run_state "$wf" "$SHA"); then
      echo "::error::Could not query ${wf} for ${SHA}."
      return 1
    fi
    read -r status conclusion <<<"$state"

    if [ "$status" = "none" ]; then
      # The tag push starts its own run, which can take a moment to register.
      if [ $((now - start)) -lt "$GRACE_SECONDS" ]; then
        echo "No ${wf} run for ${SHA} yet; waiting for one to register"
        sleep "$POLL_SECONDS"
        continue
      fi
      covered_by_ancestor "$wf"
      return
    fi

    if [ "$status" = "completed" ]; then
      if [ "$conclusion" = "success" ]; then
        echo "::notice::${wf} succeeded for ${SHA}"
        return 0
      fi
      # `cancelled` is not a milder failure: the suite produced no verdict.
      echo "::error::${wf} for ${SHA} concluded '${conclusion}', not success. Re-run it and re-tag, or dispatch with skip-test-gate if this is an emergency."
      return 1
    fi

    if [ $((now - start)) -ge "$DEADLINE_SECONDS" ]; then
      echo "::error::${wf} for ${SHA} is still '${status}' after $((DEADLINE_SECONDS / 60)) minutes. Refusing to release on an unfinished test run."
      return 1
    fi

    echo "${wf} is ${status} for ${SHA}; waiting"
    sleep "$POLL_SECONDS"
  done
}

failed=0
for wf in $WORKFLOWS; do
  check_workflow "$wf" || failed=1
done
exit "$failed"
