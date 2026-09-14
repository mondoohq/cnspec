// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportdoc

import (
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
)

// ScoreRisk is the inverse of a score value: 0 (no risk) to 100 (critical).
func ScoreRisk(score *policy.Score) int32 {
	if score == nil {
		return 0
	}
	return 100 - int32(score.Value)
}

// Band is the band a cnspec 0-100 rating value falls in -- a risk, or a check's
// configured impact, both of which run on that same scale.
//
// Every exporter agrees on where the boundaries sit and disagrees on what to
// call the bands, so the two are kept apart on purpose. The boundaries live here
// once; each format maps a band to its own vocabulary. That is what stops the
// Medium cut moving in one exporter and not another, which would be a silent
// disagreement about what "medium" means rather than a visible one.
//
// The names below are the bands themselves, not any consumer's word for them.
// Critical/High/Medium/Low are uncontested; the lowest band is not, so it is
// named after the boundary that defines it -- a value of exactly zero. cnspec's
// rating text calls that band "none", OCSF and CVSS call it "informational", and
// OCSF's impact enum calls it "unknown". All three are correct for their reader,
// and none of them belongs in the shared definition.
type Band int

const (
	// BandZero is the band below every threshold: the value is exactly 0.
	BandZero Band = iota
	BandLow
	BandMedium
	BandHigh
	BandCritical
)

// BandOf places a cnspec rating value (0-100 -- a risk, or an impact) into its
// band. These four boundaries are the single definition of them; mirror them
// nowhere else.
//
//	90 .. 100 → BandCritical
//	70 ..  89 → BandHigh
//	40 ..  69 → BandMedium
//	 1 ..  39 → BandLow
//	        0 → BandZero
func BandOf(value int32) Band {
	switch {
	case value >= 90:
		return BandCritical
	case value >= 70:
		return BandHigh
	case value >= 40:
		return BandMedium
	case value >= 1:
		return BandLow
	default:
		return BandZero
	}
}

// RiskSeverityLabel maps a cnspec risk value (0-100, the inverse of a score
// value) to the severity label cnspec uses everywhere else, from
// policy.ScoreRatingsText. Other formats name the same bands differently; see
// Band.
func RiskSeverityLabel(risk int32) string {
	switch BandOf(risk) {
	case BandCritical:
		return policy.ScoreRatingTextCritical
	case BandHigh:
		return policy.ScoreRatingTextHigh
	case BandMedium:
		return policy.ScoreRatingTextMedium
	case BandLow:
		return policy.ScoreRatingTextLow
	default:
		return policy.ScoreRatingTextNone
	}
}

// RiskFromSeverityLabel maps a VEX severity label to the cnspec risk value at
// the floor of its band, so a VEX-sourced finding lands in the same band as a
// scan-sourced one and every downstream mapping -- SARIF level and
// security-severity, OCSF severity_id -- keeps deriving from BandOf alone.
//
// The label vocabulary is fex's, including the CVSS synonyms it accepts
// (MODERATE, WARNING, NEGLIGIBLE, ...), so the ranking is not restated here.
// Floors rather than midpoints: they are the CVSS band boundaries GitHub code
// scanning reads out of security-severity, so CRITICAL renders as 9.0 and HIGH
// as 7.0 rather than as a value inside the band that no scale defines.
func RiskFromSeverityLabel(label string) int32 {
	switch fex.SeverityRank(label) {
	case 4:
		return 90
	case 3:
		return 70
	case 2:
		return 40
	case 1:
		return 10
	default:
		return 0
	}
}
