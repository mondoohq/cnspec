// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package checksum

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
)

func TestHashScoreRow(t *testing.T) {
	score := &policy.Score{
		QrId: "qr-1", RiskScore: 25, Type: 2, Value: 80, Weight: 1, Message: "m",
	}
	d1, err := HashScoreRow(score)
	require.NoError(t, err)

	// Every scalar field is a fold input.
	changed := &policy.Score{QrId: "qr-1", RiskScore: 25, Type: 2, Value: 81, Weight: 1, Message: "m"}
	d2, err := HashScoreRow(changed)
	require.NoError(t, err)
	assert.NotEqual(t, d1, d2)

	// Risk factors are content...
	withRisks := &policy.Score{
		QrId: "qr-1", RiskScore: 25, Type: 2, Value: 80, Weight: 1, Message: "m",
		RiskFactors: &policy.ScoredRiskFactors{Items: []*policy.ScoredRiskFactor{
			{Mrn: "//risk/1", Risk: 0.5},
			{Mrn: "//risk/2", Risk: 0.7},
		}},
	}
	d3, err := HashScoreRow(withRisks)
	require.NoError(t, err)
	assert.NotEqual(t, d1, d3)

	// ...and a multiset: item order is not.
	reordered := &policy.Score{
		QrId: "qr-1", RiskScore: 25, Type: 2, Value: 80, Weight: 1, Message: "m",
		RiskFactors: &policy.ScoredRiskFactors{Items: []*policy.ScoredRiskFactor{
			{Mrn: "//risk/2", Risk: 0.7},
			{Mrn: "//risk/1", Risk: 0.5},
		}},
	}
	d4, err := HashScoreRow(reordered)
	require.NoError(t, err)
	assert.Equal(t, d3, d4, "risk factor order is not content")
}

// TestSourceTimestampsAreNotContent pins the wall-clock exclusion: a re-scan
// that changes nothing but source observation times must not read as changed
// content.
func TestSourceTimestampsAreNotContent(t *testing.T) {
	src := func(first, last string) *policy.Score {
		return &policy.Score{
			QrId: "qr-1", Value: 80,
			Sources: &policy.Sources{Items: []*policy.Source{{
				Name: "scanner", Url: "https://example.com", Version: "1.0",
				FirstDetectedAt: first, LastUpdatedAt: last,
			}}},
		}
	}

	d1, err := HashScoreRow(src("2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"))
	require.NoError(t, err)
	d2, err := HashScoreRow(src("2026-01-01T00:00:00Z", "2026-08-12T00:00:00Z"))
	require.NoError(t, err)
	assert.Equal(t, d1, d2, "source timestamps are wall-clock noise, not content")

	// The identity fields ARE content.
	other := src("2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z")
	other.Sources.Items[0].Name = "other-scanner"
	d3, err := HashScoreRow(other)
	require.NoError(t, err)
	assert.NotEqual(t, d1, d3)

	// FixedAt is content, not observation noise: a scan whose only change is
	// the fixed-state flip must read as changed.
	fixed := src("2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z")
	fixed.Sources.Items[0].FixedAt = "2026-08-12T00:00:00Z"
	d4, err := HashScoreRow(fixed)
	require.NoError(t, err)
	assert.NotEqual(t, d1, d4, "a fixed_at flip alone is a content change")
}

func TestHashRiskRow(t *testing.T) {
	risk := &policy.ScoredRiskFactor{Mrn: "//risk/1", Risk: 0.5, IsToxic: true, IsDetected: false}
	r1, err := HashRiskRow(risk)
	require.NoError(t, err)
	r1again, err := HashRiskRow(risk)
	require.NoError(t, err)
	assert.Equal(t, r1, r1again)

	risk.IsDetected = true
	r2, err := HashRiskRow(risk)
	require.NoError(t, err)
	assert.NotEqual(t, r1, r2)
}

// TestEmptyContainersHashAsNil pins the round-trip normalization: the scandb
// writer marshals an empty-but-non-nil RiskFactors/Sources to zero bytes and
// the reader materializes it back as nil, so emptiness — not pointer
// presence — is the content. A score hashed in memory (write-time emission)
// and the same score hashed after the storage round trip must agree.
func TestEmptyContainersHashAsNil(t *testing.T) {
	base := &policy.Score{QrId: "qr-1", RiskScore: 25, Type: 2, Value: 80, Weight: 1, Message: "m"}
	dNil, err := HashScoreRow(base)
	require.NoError(t, err)

	emptied := &policy.Score{
		QrId: "qr-1", RiskScore: 25, Type: 2, Value: 80, Weight: 1, Message: "m",
		RiskFactors: &policy.ScoredRiskFactors{},
		Sources:     &policy.Sources{},
	}
	dEmpty, err := HashScoreRow(emptied)
	require.NoError(t, err)
	assert.Equal(t, dNil, dEmpty,
		"nil and empty containers are one value after the storage round trip, so they must be one value in the hash")

	// A single element still changes the checksum — the collapse is only of
	// the empty forms.
	withItem := &policy.Score{
		QrId: "qr-1", RiskScore: 25, Type: 2, Value: 80, Weight: 1, Message: "m",
		RiskFactors: &policy.ScoredRiskFactors{Items: []*policy.ScoredRiskFactor{{Mrn: "//r/1", Risk: 0.5}}},
	}
	dItem, err := HashScoreRow(withItem)
	require.NoError(t, err)
	assert.NotEqual(t, dNil, dItem)
}
