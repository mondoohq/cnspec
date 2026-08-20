// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package checksum

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/llx"
)

// Golden vectors, mirroring mql's llx/checksum goldens for the row kinds it
// owns. These pinned bits ARE the shared contract with every consumer that
// recomputes score/risk checksums (the scandb pass here, the server's
// backfill and verification recompute): all the other tests in this package
// are relative — both sides of every comparison move together under an
// algorithm change — so without these, an innocent refactor (reordered
// folds, a renamed domain string, a changed sentinel) ships silently and
// every stored checksum stops comparing without the AlgoVersion bump the
// change requires.
//
// If one of these tests fails, you changed the canonicalization: either
// revert, or bump llx/checksum's AlgoVersion (coordinating with every
// consumer) and re-pin.

func TestHashScoreRowGolden(t *testing.T) {
	bare := &policy.Score{
		QrId: "golden-check", RiskScore: 25, Type: 2, Value: 80, Weight: 1,
		Message: "expected file permissions 0o640, got 0o644",
	}
	d, err := HashScoreRow(bare)
	require.NoError(t, err)
	assert.Equal(t, uint64(0xb925ffea8e783082), d, "bare score golden moved — see the package comment above")

	// The loaded shape exercises every fold this package owns: risk factors
	// (with a Data result map), sources with vendor and fixed state, and the
	// timestamp exclusions (FirstDetectedAt/LastUpdatedAt are set but must
	// not contribute; FixedAt is set and must).
	loaded := &policy.Score{
		QrId: "golden-check", RiskScore: 25, Type: 2, Value: 80, Weight: 1,
		Message: "expected file permissions 0o640, got 0o644",
		RiskFactors: &policy.ScoredRiskFactors{Items: []*policy.ScoredRiskFactor{
			{
				Mrn: "//policy.api.mondoo.app/risks/internet-facing", Risk: -0.4,
				IsToxic: true, IsDetected: true,
				Data: map[string]*llx.Result{
					"data-query-1": {CodeId: "data-query-1", Data: llx.StringPrimitive("exposed")},
				},
			},
			{Mrn: "//policy.api.mondoo.app/risks/eol", Risk: 0.2},
		}},
		Sources: &policy.Sources{Items: []*policy.Source{{
			Name: "scanner", Url: "https://example.com/scan", Version: "13.0.0",
			Vendor:          policy.Source_MONDOO,
			FirstDetectedAt: "2026-01-01T00:00:00Z",
			LastUpdatedAt:   "2026-08-12T00:00:00Z",
			FixedAt:         "2026-08-15T00:00:00Z",
		}}},
	}
	dl, err := HashScoreRow(loaded)
	require.NoError(t, err)
	assert.Equal(t, uint64(0x8c82ebc76368d08c), dl, "loaded score golden moved — see the package comment above")

	// The observation timestamps are excluded from the fold, so changing
	// them must not move the golden.
	loaded.Sources.Items[0].LastUpdatedAt = "2026-08-16T00:00:00Z"
	dl2, err := HashScoreRow(loaded)
	require.NoError(t, err)
	assert.Equal(t, dl, dl2)
}

func TestHashRiskRowGolden(t *testing.T) {
	risk := &policy.ScoredRiskFactor{
		Mrn: "//policy.api.mondoo.app/risks/internet-facing", Risk: 0.4,
		IsToxic: true, IsDetected: true,
	}
	d, err := HashRiskRow(risk)
	require.NoError(t, err)
	assert.Equal(t, uint64(0xc618f726fd50c75a), d, "risk row golden moved — see the package comment above")
}
