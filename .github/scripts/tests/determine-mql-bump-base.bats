#!/usr/bin/env bats
# Tests for .github/scripts/determine-mql-bump-base.sh
#
# Each test sets MAIN_GOMOD (and, for the unversioned-path cases,
# KNOWN_BRANCHES) in the environment so the script runs without calling
# gh api. See the script's header for the input contract.

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
  [[ "$output" == *"Could not find an mql import"* ]]
}

# --- main on mql's unversioned (v14+) module path -------------------------
# mql dropped the major suffix for its v14 line, so main's go.mod pins
# go.mondoo.com/mql at a pseudo-version and carries no major to compare.

@test "unversioned main: release with a support branch routes there" {
  MAIN_GOMOD='require go.mondoo.com/mql v0.0.0-20260825052922-32d60cce639f' \
    KNOWN_BRANCHES='main v13' \
    run bash "$SCRIPT" v13.36.0
  [ "$status" -eq 0 ]
  [ "$output" = "v13" ]
}

@test "unversioned main: release without a support branch routes to main" {
  MAIN_GOMOD='require go.mondoo.com/mql v0.0.0-20260825052922-32d60cce639f' \
    KNOWN_BRANCHES='main v13' \
    run bash "$SCRIPT" v14.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "unversioned main: no v-prefix on the incoming version" {
  MAIN_GOMOD='require go.mondoo.com/mql v0.0.0-20260825052922-32d60cce639f' \
    KNOWN_BRANCHES='main v13' \
    run bash "$SCRIPT" 13.36.0
  [ "$status" -eq 0 ]
  [ "$output" = "v13" ]
}

@test "unversioned main: pre-release suffix does not change routing" {
  MAIN_GOMOD='require go.mondoo.com/mql v0.0.0-20260825052922-32d60cce639f' \
    KNOWN_BRANCHES='main v13' \
    run bash "$SCRIPT" v13.37.0-rc.1
  [ "$status" -eq 0 ]
  [ "$output" = "v13" ]
}

@test "unversioned main: real go.mod shape from main routes v13 to v13" {
  # Abridged copy of main's go.mod: the commented replace directive must not
  # be mistaken for the import, and the require line is what decides.
  MAIN_GOMOD=$'module go.mondoo.com/cnspec\n\n// replace go.mondoo.com/mql => ../mql\n\nrequire (\n\tgo.mondoo.com/mondoo-go v0.0.0-20260822000727-1d813d6a83c7\n\tgo.mondoo.com/mql v0.0.0-20260825052922-32d60cce639f\n)' \
    KNOWN_BRANCHES='main v13' \
    run bash "$SCRIPT" v13.36.0
  [ "$status" -eq 0 ]
  [ "$output" = "v13" ]
}

@test "a major-suffixed import still wins over the unversioned fallback" {
  # Both shapes present (e.g. mid-migration): the suffixed one decides.
  MAIN_GOMOD=$'require (\n\tgo.mondoo.com/mql/v13 v13.35.0\n\tgo.mondoo.com/mql v0.0.0-20260825052922-32d60cce639f // indirect\n)' \
    KNOWN_BRANCHES='main v13' \
    run bash "$SCRIPT" v13.36.0
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
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
