#!/usr/bin/env bats
# Tests for .github/scripts/verify-tested.sh
#
# gh is replaced by a stub on PATH that answers from fixture files, so the
# tests never reach the network. Fixtures, under $STUB:
#
#   runs/<workflow>/<sha>   one "status conclusion" line per query; the last
#                           line repeats once the earlier ones are used up,
#                           which is how a run that appears late is modelled.
#                           A missing file means the commit has no run.
#   paths/<workflow>        the workflow's push path filters, one per line
#   ancestors               the commits after SHA, newest first
#   compare                 files changed between the passing ancestor and SHA

setup() {
  SCRIPT="$BATS_TEST_DIRNAME/../verify-tested.sh"
  STUB="$BATS_TEST_TMPDIR/stub"
  mkdir -p "$STUB/runs/pr-test-lint.yml" "$STUB/paths" "$STUB/bin"
  : >"$STUB/ancestors"
  : >"$STUB/compare"

  cat >"$STUB/bin/gh" <<'GH'
#!/usr/bin/env bash
# Stub of `gh api`, answering from $STUB fixtures.
url=$2
# Real gh can write notices to stderr on a successful call.
[ -n "${STUB_STDERR_NOISE:-}" ] && echo "warning: a notice on stderr" >&2
case "$url" in
  */actions/workflows/*/runs\?head_sha=*)
    wf=${url#*/actions/workflows/}; wf=${wf%%/runs*}
    sha=${url#*head_sha=}; sha=${sha%%&*}
    f="$STUB/runs/$wf/$sha"
    [ -f "$f" ] || { echo "none none"; exit 0; }
    n="$f.calls"; c=$(( $(cat "$n" 2>/dev/null || echo 0) + 1 )); echo "$c" >"$n"
    total=$(wc -l <"$f")
    [ "$c" -gt "$total" ] && c=$total
    sed -n "${c}p" "$f"
    ;;
  */contents/.github/workflows/*)
    wf=${url#*/contents/.github/workflows/}; wf=${wf%%\?*}
    [ -f "$STUB/paths/$wf" ] || { echo "gh: Not Found (HTTP 404)" >&2; exit 1; }
    echo "on:"; echo "  push:"; echo "    paths:"
    sed 's/^/      - "/; s/$/"/' "$STUB/paths/$wf"
    ;;
  */commits\?sha=*) echo "$url" >>"$STUB/commits-requests"; cat "$STUB/ancestors" ;;
  */compare/*)      cat "$STUB/compare" ;;
  *) echo "stub: unexpected $url" >&2; exit 1 ;;
esac
GH
  chmod +x "$STUB/bin/gh"

  export PATH="$STUB/bin:$PATH" STUB
  export REPO=mondoohq/example SHA=tagged
  export WORKFLOWS=pr-test-lint.yml
  export GRACE_SECONDS=0 POLL_SECONDS=0 DEADLINE_SECONDS=5

  printf '%s\n' '**.go' '**.mod' 'go.sum' 'content/**' >"$STUB/paths/pr-test-lint.yml"
}

runs() { printf '%s\n' "${@:2}" >"$STUB/runs/pr-test-lint.yml/$1"; }

@test "a successful run releases" {
  runs tagged "completed success"
  run bash "$SCRIPT"
  [ "$status" -eq 0 ]
}

@test "a failed run refuses" {
  runs tagged "completed failure"
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"concluded 'failure'"* ]]
}

@test "a cancelled run refuses: no verdict is not a pass" {
  runs tagged "completed cancelled"
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"concluded 'cancelled'"* ]]
}

@test "a run that registers late is waited for, not reported missing" {
  # The race seen cutting v14.1.0: the tag push starts its run a moment after
  # this job first asks.
  GRACE_SECONDS=5 runs tagged "none none" "none none" "in_progress none" "completed success"
  GRACE_SECONDS=5 run bash "$SCRIPT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"waiting for one to register"* ]]
}

@test "a run still going at the deadline refuses" {
  runs tagged "in_progress none"
  DEADLINE_SECONDS=0 run bash "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"still 'in_progress'"* ]]
}

@test "no run, nothing watched changed since a passing ancestor: releases" {
  # v14.0.0-rc.9: the tagged commit only changed a workflow file.
  printf '%s\n' parent >"$STUB/ancestors"
  runs parent "completed success"
  printf '%s\n' .github/workflows/goreleaser.yml >"$STUB/compare"
  run bash "$SCRIPT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"nothing it watches has changed since parent"* ]]
}

@test "no run, but code changed under a cancelled run in between: refuses" {
  # The September burst, one commit on: a docs-only commit sits on top of a
  # commit whose run was cancelled while queued. Its code was never tested, and
  # walking past it to the last green run must not hide that.
  printf '%s\n' cancelled green >"$STUB/ancestors"
  runs cancelled "completed cancelled"
  runs green "completed success"
  printf '%s\n' README.md providers/os/resources/packages.go >"$STUB/compare"
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"providers/os/resources/packages.go"* ]]
  [[ "$output" == *"never been tested"* ]]
}

@test "no run, and a nested path matches a directory glob: refuses" {
  printf '%s\n' parent >"$STUB/ancestors"
  runs parent "completed success"
  printf '%s\n' content/mondoo-linux-security.mql.yaml >"$STUB/compare"
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"matching 'content/**'"* ]]
}

@test "no run, and no passing ancestor within reach: refuses" {
  printf '%s\n' a b c >"$STUB/ancestors"
  runs a "completed cancelled"
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"none of its last"* ]]
}

@test "no run, and a diff too large to inspect: refuses" {
  printf '%s\n' parent >"$STUB/ancestors"
  runs parent "completed success"
  seq 1 300 | sed 's/^/docs\/page-/' >"$STUB/compare"
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"too large to check"* ]]
}

@test "no run, and the path filters cannot be read: refuses" {
  rm "$STUB/paths/pr-test-lint.yml"
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Could not read the path filters"* ]]
}

@test "every workflow is checked, and one failing is enough to refuse" {
  mkdir -p "$STUB/runs/pr-test-generated-files.yaml"
  runs tagged "completed success"
  printf '%s\n' "completed failure" >"$STUB/runs/pr-test-generated-files.yaml/tagged"
  WORKFLOWS="pr-test-lint.yml pr-test-generated-files.yaml" run bash "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"pr-test-lint.yml succeeded"* ]]
  [[ "$output" == *"pr-test-generated-files.yaml for tagged concluded 'failure'"* ]]
}

@test "a notice on stderr during a successful call does not corrupt the result" {
  # With stderr mixed into the response, "warning:" was parsed as the run's
  # status, and the gate waited out its deadline on a green commit.
  runs tagged "completed success"
  STUB_STDERR_NOISE=1 run bash "$SCRIPT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"pr-test-lint.yml succeeded"* ]]
}

@test "the walk asks for MAX_ANCESTORS ancestors, not counting the commit itself" {
  # The listing starts with SHA itself, so it needs one more entry than the
  # number of ancestors the error message promises were checked.
  printf '%s\n' parent >"$STUB/ancestors"
  runs parent "completed success"
  printf '%s\n' README.md >"$STUB/compare"
  MAX_ANCESTORS=7 run bash "$SCRIPT"
  [ "$status" -eq 0 ]
  grep -q 'per_page=8' "$STUB/commits-requests"
}
