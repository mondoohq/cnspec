#!/usr/bin/env bats
# Tests for .github/scripts/determine-mql-bump-base.sh
#
# Each test sets MAIN_GOMOD in the environment so the script runs without
# calling gh api. See the script's header for the input contract.

setup() {
  SCRIPT="$BATS_TEST_DIRNAME/../determine-mql-bump-base.sh"
}

@test "matching major routes to main (v-prefixed tag)" {
  MAIN_GOMOD='require go.mondoo.com/mql/v14 v14.0.0' \
    run bash "$SCRIPT" v14.5.0
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "matching major routes to main (no v prefix)" {
  MAIN_GOMOD='require go.mondoo.com/mql/v14 v14.0.0' \
    run bash "$SCRIPT" 14.5.0
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "older major routes to v{major}" {
  MAIN_GOMOD='require go.mondoo.com/mql/v14 v14.0.0' \
    run bash "$SCRIPT" v13.35.1
  [ "$status" -eq 0 ]
  [ "$output" = "v13" ]
}

@test "future major routes to v{major}" {
  MAIN_GOMOD='require go.mondoo.com/mql/v14 v14.0.0' \
    run bash "$SCRIPT" v15.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "v15" ]
}

@test "pre-release suffix does not change routing" {
  MAIN_GOMOD='require go.mondoo.com/mql/v14 v14.0.0' \
    run bash "$SCRIPT" v14.0.0-rc.1
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "pre-release suffix on older major routes to v{major}" {
  MAIN_GOMOD='require go.mondoo.com/mql/v14 v14.0.0' \
    run bash "$SCRIPT" v13.35.1-rc.1
  [ "$status" -eq 0 ]
  [ "$output" = "v13" ]
}

@test "main on v13 today: v13 release routes to main" {
  MAIN_GOMOD='require go.mondoo.com/mql/v13 v13.35.0' \
    run bash "$SCRIPT" v13.35.1
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "go.mod without an mql import fails loudly" {
  MAIN_GOMOD='require go.mondoo.com/other v1.0.0' \
    run bash "$SCRIPT" v13.35.1
  [ "$status" -ne 0 ]
  [[ "$output" == *"Could not extract mql major"* ]]
}

@test "missing version argument errors out" {
  MAIN_GOMOD='require go.mondoo.com/mql/v14 v14.0.0' \
    run bash "$SCRIPT"
  [ "$status" -ne 0 ]
}

@test "picks the first mql import when multiple present" {
  # If go.mod somehow has multiple mql majors (edge case), head -1 wins.
  MAIN_GOMOD=$'require (\n\tgo.mondoo.com/mql/v14 v14.0.0\n\tgo.mondoo.com/mql/v13 v13.0.0 // indirect\n)' \
    run bash "$SCRIPT" v14.5.0
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}
