// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRiskThresholdExitCodes pins the arithmetic behind `--risk-threshold`.
//
// Why this is worth a test of its own: the default threshold is 101, and the
// comparison is `100 - worst check score >= threshold`. The left side can never
// reach 101, so **at the default a scan exits 0 no matter how badly it
// scores** -- only an asset-level error changes that. Every `cnspec scan` in
// someone's CI that relies on a non-zero exit is relying on this flag, and an
// off-by-one here turns every one of those gates into "always passes" or
// "always fails" without any other symptom.
//
// The threshold judges the worst failing check, not the asset score. The asset
// score averages its policies, so it sits below the worst finding whenever any
// policy passes; probing it would put both thresholds under that finding and
// expect an exit 0 the scan must not give.
//
// The expectation is derived from the scan itself rather than hardcoded:
// reading the worst check's risk out of the report and probing either side of
// it holds whatever the content and the base image do.
func TestRiskThresholdExitCodes(t *testing.T) {
	requireDocker(t)
	pullImage(t, imageDebian)

	base := []string{
		"scan", "docker", imageDebian, "-f", linuxSecurity,
		"--detect-cicd=false", "-o", "json",
	}

	// 1. The default threshold is inert: whatever this image scores, exit 0.
	res := run(t, defaultScenarioTimeout, base...)
	require.Equal(t, 0, res.exitCode,
		"the default risk threshold must never fail a scan that connected\n%s", res.dump())
	requireNoProviderPanic(t, res)

	rep := decodeReport(t, res)
	mrn, _ := requireOneAsset(t, rep)
	requireNoAssetErrors(t, rep)

	// The worst finding: the highest risk among failing checks. The entry
	// under the asset's own MRN is the asset score, which the threshold
	// does not judge.
	var risk uint32
	failing := 0
	for id, v := range rep.GetScores()[mrn].GetValues() {
		if id == mrn || v.GetStatus() != "fail" {
			continue
		}
		failing++
		risk = max(risk, v.GetRiskScore())
	}
	require.NotZero(t, failing, "no failing check to derive a threshold from")
	t.Logf("%s: %d failing checks, worst risk=%d", imageDebian, failing, risk)
	require.LessOrEqual(t, risk, uint32(100), "risk score out of range")

	// 2. threshold == risk: `risk >= threshold` holds, so exit 1.
	atRisk := append(slices.Clone(base), "--risk-threshold", fmt.Sprint(risk))
	res = run(t, defaultScenarioTimeout, atRisk...)
	assert.Equal(t, 1, res.exitCode,
		"--risk-threshold %d on a scan with risk %d must exit 1\n%s", risk, risk, res.dump())
	// -o json keeps stdout a report, so the reason for the exit 1 is on stderr.
	assert.Contains(t, string(res.stderr), fmt.Sprintf("✕ Risk threshold %d met by the worst failing check", risk),
		"the scan must say which check met the threshold\n%s", res.dump())

	// 3. threshold == risk+1: the comparison just fails, so exit 0. This is the
	// side that catches an off-by-one; step 2 alone passes even if the
	// comparison is `>` instead of `>=`.
	aboveRisk := append(slices.Clone(base), "--risk-threshold", fmt.Sprint(risk+1))
	res = run(t, defaultScenarioTimeout, aboveRisk...)
	assert.Equal(t, 0, res.exitCode,
		"--risk-threshold %d on a scan with risk %d must exit 0\n%s", risk+1, risk, res.dump())
	if risk+1 <= 100 {
		assert.Contains(t, string(res.stderr), fmt.Sprintf("✓ Risk threshold %d not met", risk+1),
			"the scan must say the threshold was not met\n%s", res.dump())
	}
}
