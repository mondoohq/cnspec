// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package checksum

import (
	"fmt"
	"testing"

	"go.mondoo.com/cnspec/policy"
)

// The score/risk checksum pass runs over every scored row of every scan, on
// the client (C0 emission) and the server (backfill + verification
// recompute), so per-row cost is fleet-wide cost. Together with mql's
// llx/checksum benchmarks (data and resource rows), these pin the shapes
// this package owns: the bare score (the floor — most checks carry no risk
// factors or sources), the loaded score (risk factors + sources, the
// realistic ceiling), and the four-column risk row.

// benchScore returns a bare score row: scalars only.
func benchScore() *policy.Score {
	return &policy.Score{
		QrId:      "//policy.api.mondoo.app/queries/mondoo-linux-security-permissions-on-etc-shadow",
		RiskScore: 25,
		Type:      2,
		Value:     80,
		Weight:    1,
		Message:   "expected file permissions 0o640, got 0o644",
	}
}

// benchLoadedScore returns a score carrying risk factors and sources — the
// shape a finding-bearing score takes on a real asset.
func benchLoadedScore() *policy.Score {
	s := benchScore()
	items := make([]*policy.ScoredRiskFactor, 5)
	for i := range items {
		items[i] = &policy.ScoredRiskFactor{
			Mrn:        fmt.Sprintf("//policy.api.mondoo.app/risks/internet-facing-%d", i),
			Risk:       0.4,
			IsToxic:    i%2 == 0,
			IsDetected: true,
		}
	}
	s.RiskFactors = &policy.ScoredRiskFactors{Items: items}
	s.Sources = &policy.Sources{Items: []*policy.Source{
		{Name: "scanner", Url: "https://example.com/scan", Version: "13.0.0"},
		{Name: "integration", Url: "https://example.com/aws", Version: "2.1.0"},
	}}
	return s
}

func BenchmarkHashScoreRow_Bare(b *testing.B) {
	score := benchScore()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := HashScoreRow(score); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashScoreRow_Loaded(b *testing.B) {
	score := benchLoadedScore()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := HashScoreRow(score); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashRiskRow(b *testing.B) {
	risk := &policy.ScoredRiskFactor{
		Mrn:        "//policy.api.mondoo.app/risks/internet-facing",
		Risk:       0.4,
		IsToxic:    true,
		IsDetected: true,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := HashRiskRow(risk); err != nil {
			b.Fatal(err)
		}
	}
}
