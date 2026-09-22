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
// comparison is `100 - worstScore >= threshold`. The left side can never reach
// 101, so **at the default a scan exits 0 no matter how badly it scores** --
// only an asset-level error changes that. Every `cnspec scan` in someone's CI
// that relies on a non-zero exit is relying on this flag, and an off-by-one
// here turns every one of those gates into "always passes" or "always fails"
// without any other symptom.
//
// The expectation is derived from the scan itself rather than hardcoded. An
// earlier version of this test asserted that alpine could not score 100 against
// mondoo-linux-security; it does (15 pass, 5 skip, aggregate risk 0), so the
// test failed for a reason that had nothing to do with cnspec. Reading the
// risk score out of the report and probing either side of it holds whatever the
// content and the base image do.
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

	overall := rep.GetScores()[mrn].GetValues()[mrn]
	require.NotNil(t, overall, "no aggregate score to derive a threshold from")
	risk := overall.GetRiskScore()
	t.Logf("%s scored risk=%d (status %q)", imageDebian, risk, overall.GetStatus())
	require.LessOrEqual(t, risk, uint32(100), "risk score out of range")

	// 2. threshold == risk: `risk >= threshold` holds, so exit 1.
	atRisk := append(slices.Clone(base), "--risk-threshold", fmt.Sprint(risk))
	res = run(t, defaultScenarioTimeout, atRisk...)
	assert.Equal(t, 1, res.exitCode,
		"--risk-threshold %d on a scan with risk %d must exit 1\n%s", risk, risk, res.dump())

	// 3. threshold == risk+1: the comparison just fails, so exit 0. This is the
	// side that catches an off-by-one; step 2 alone passes even if the
	// comparison is `>` instead of `>=`.
	aboveRisk := append(slices.Clone(base), "--risk-threshold", fmt.Sprint(risk+1))
	res = run(t, defaultScenarioTimeout, aboveRisk...)
	assert.Equal(t, 0, res.exitCode,
		"--risk-threshold %d on a scan with risk %d must exit 0\n%s", risk+1, risk, res.dump())
}
