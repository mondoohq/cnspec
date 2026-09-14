// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportdoc

import "testing"

func TestRiskFromSeverityLabelLandsInBand(t *testing.T) {
	for _, tc := range []struct {
		label string
		want  Band
	}{
		{"CRITICAL", BandCritical}, {"HIGH", BandHigh}, {"MEDIUM", BandMedium},
		{"MODERATE", BandMedium}, {"LOW", BandLow}, {"NEGLIGIBLE", BandLow},
		{"NONE", BandZero}, {"", BandZero}, {"nonsense", BandZero},
	} {
		if got := BandOf(RiskFromSeverityLabel(tc.label)); got != tc.want {
			t.Errorf("%q -> band %v, want %v", tc.label, got, tc.want)
		}
	}
}
