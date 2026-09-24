// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func TestAggregateReport(t *testing.T) {
	b := &policy.Bundle{
		Policies: []*policy.Policy{
			{
				Uid:  "policy1",
				Name: "Policy 1",
			},
		},
	}

	r := NewAggregateReporter()
	r.AddBundle(b)
	assert.Equal(t, r.bundle, b)

	b2 := &policy.Bundle{
		Policies: []*policy.Policy{
			{
				Uid:  "policy2",
				Name: "Policy 2",
			},
		},
	}

	r.AddBundle(b2)
	assert.Equal(t, r.bundle, policy.Merge(b, b2))
}

// TestAggregateReporter_AddScanWarning_DoesNotFailAnAssetWithAReport is the
// regression test for the choice made in scan_pipeline.go: a provider crash
// discovered after RunAssetJob must not turn an otherwise-successful scan
// into a failed one. AddScanError would: Reports().Result.Full.Errors would
// gain an entry for this asset and apps/cnspec/cmd/scan.go exits non-zero
// whenever that map is non-empty. AddScanWarning must do neither.
func TestAggregateReporter_AddScanWarning_DoesNotFailAnAssetWithAReport(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "crashed-but-scored"}

	r := NewAggregateReporter()
	r.AddReport(asset, &AssetReport{
		Mrn:    asset.Mrn,
		Report: &policy.Report{Score: &policy.Score{Value: 80}},
	})
	r.AddScanWarning(asset, []string{"the 'os' provider crashed: connection refused"})

	result := r.Reports()
	require.True(t, result.Ok, "a warning must not flip Reports().Ok")

	full := result.GetFull()
	require.NotNil(t, full)
	assert.Empty(t, full.Errors, "a warning must not appear in the errors map that drives the CLI exit code")
	assert.Contains(t, full.Reports, asset.Mrn, "the report must survive the warning, not be dropped")
	assert.Equal(t, uint32(80), full.Reports[asset.Mrn].Score.Value)

	require.Contains(t, full.Warnings, asset.Mrn, "the warning must be carried onto the wire-serialized ReportCollection")
	assert.Equal(t, []string{"the 'os' provider crashed: connection refused"}, full.Warnings[asset.Mrn].Messages)
}

// TestAggregateReporter_Reports_OmitsWarningsFieldWhenThereAreNone keeps the
// wire message unchanged for the common case (no crash): Warnings should be
// nil, not an empty-but-present map.
func TestAggregateReporter_Reports_OmitsWarningsFieldWhenThereAreNone(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "clean-host"}

	r := NewAggregateReporter()
	r.AddReport(asset, &AssetReport{
		Mrn:    asset.Mrn,
		Report: &policy.Report{Score: &policy.Score{Value: 100}},
	})

	full := r.Reports().GetFull()
	require.NotNil(t, full)
	assert.Nil(t, full.Warnings)
}

func TestAggregateReporter_AddScanWarning_RecordsAgainstWarnings(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "crashed-but-scored"}

	r := NewAggregateReporter()
	r.AddScanWarning(asset, []string{"first crash", "second crash"})

	warnings := r.Warnings()
	require.Contains(t, warnings, asset.Mrn)
	assert.Equal(t, []string{"first crash", "second crash"}, warnings[asset.Mrn])

	// Mutating the returned slice must not corrupt the reporter's own copy.
	warnings[asset.Mrn][0] = "tampered"
	assert.Equal(t, "first crash", r.Warnings()[asset.Mrn][0])
}

func TestAggregateReporter_AddScanWarning_EmptyIsNoOp(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "no-warnings"}

	r := NewAggregateReporter()
	r.AddScanWarning(asset, nil)

	assert.Empty(t, r.Warnings())
	// No AddReport/AddScanError call either, so the asset must not have been
	// registered just because AddScanWarning saw it with nothing to add.
	assert.NotContains(t, r.assets, asset.Mrn)
}

// TestReportCriticalErrors_DedupesByMessage covers the dedup this PR adds:
// mql may hand back the same crash diagnostic more than once (an older mql
// build without its own dedup, or two distinct provider crashes), and
// reportCriticalErrors must collapse repeats into one warning entry and one
// log line rather than reporting the same crash N times.
func TestReportCriticalErrors_DedupesByMessage(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "flaky-host"}
	r := NewAggregateReporter()

	errs := []error{
		errors.New("the 'os' provider crashed: connection refused"),
		errors.New("the 'os' provider crashed: connection refused"),
		errors.New("the 'os' provider crashed: connection refused"),
		errors.New("the 'aws' provider crashed: EOF"),
	}

	reportCriticalErrors(r, asset, errs)

	warnings := r.Warnings()[asset.Mrn]
	require.Len(t, warnings, 2, "3 repeats of one crash + 1 distinct crash = 2 unique warnings")
	assert.Contains(t, warnings, "the 'os' provider crashed: connection refused")
	assert.Contains(t, warnings, "the 'aws' provider crashed: EOF")
}

func TestReportCriticalErrors_NoErrorsIsNoOp(t *testing.T) {
	asset := &inventory.Asset{Mrn: "//assets/1", Name: "healthy-host"}
	r := NewAggregateReporter()

	reportCriticalErrors(r, asset, nil)

	assert.Empty(t, r.Warnings())
	assert.NotContains(t, r.assets, asset.Mrn)
}
