// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"fmt"
	"strings"

	"go.mondoo.com/cnspec/policy"
)

// WorstScoreValue is the lowest rated score across all scanned assets, or 100
// when no asset was rated. Every reporter fills WorstScore the same way
// through lowerScore, so this matches ReportCollection.GetWorstScore on a
// full result and is the only worst score a result without a collection has.
func (r *ScanResult) WorstScoreValue() uint32 {
	s := r.GetWorstScore()
	if s == nil || s.Rating() == policy.ScoreRating_unrated {
		return 100
	}
	return s.Value
}

// WorstCheckScoreValue is the lowest rated check score across all scanned
// assets, or 100 when no check was rated.
func (r *ScanResult) WorstCheckScoreValue() uint32 {
	s := r.GetWorstCheckScore()
	if s == nil || s.Rating() == policy.ScoreRating_unrated {
		return 100
	}
	return s.Value
}

// Failed reports whether the scan should be treated as a failure: an asset
// failed to scan, or a check's risk (100 minus its score) meets
// riskThreshold. The threshold judges findings, never the asset score: an
// asset score averages its policies, so one critical finding next to a
// passing policy would read as medium and slip under the threshold. The
// verdict is the same for every ReportType because Ok and WorstCheckScore are
// set for all of them. A nil result has not failed.
func (r *ScanResult) Failed(riskThreshold int) bool {
	if r == nil {
		return false
	}
	if !r.Ok {
		return true
	}
	return r.riskThresholdMet(riskThreshold)
}

// riskThresholdMet is the comparison Failed and RiskThresholdVerdict share,
// so the line that explains the exit status cannot disagree with it.
func (r *ScanResult) riskThresholdMet(riskThreshold int) bool {
	return (100 - r.WorstCheckScoreValue()) >= uint32(riskThreshold)
}

// RiskThresholdVerdict is the one line that explains how riskThreshold
// decided the exit status: whether it was met, and by which check. title names
// a check by its MRN when it can; it may be nil or return "". The verdict is
// empty at the default threshold, which no check can meet. Asset errors fail a
// scan on their own and are reported where they happen, so the verdict speaks
// only to the threshold.
func (r *ScanResult) RiskThresholdVerdict(riskThreshold int, title func(checkMrn string) string) string {
	if riskThreshold > 100 {
		return ""
	}
	met := r.riskThresholdMet(riskThreshold)

	worst := r.GetWorstCheckScore()
	if worst == nil || worst.Rating() == policy.ScoreRating_unrated || worst.Value == 100 {
		if met {
			return fmt.Sprintf("✕ Risk threshold %d met: no check failed, but a threshold of %d is met by every scan", riskThreshold, riskThreshold)
		}
		return fmt.Sprintf("✓ Risk threshold %d not met: no check failed", riskThreshold)
	}

	finding := fmt.Sprintf("%s (%d)", strings.ToUpper(worst.Rating().Text()), 100-worst.Value)
	if title != nil {
		if t := title(worst.QrId); t != "" {
			finding += " " + t
		}
	}
	if met {
		return fmt.Sprintf("✕ Risk threshold %d met by the worst failing check: %s", riskThreshold, finding)
	}
	return fmt.Sprintf("✓ Risk threshold %d not met: the worst failing check is %s", riskThreshold, finding)
}
