// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import "go.mondoo.com/cnspec/policy"

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
	return (100 - r.WorstCheckScoreValue()) >= uint32(riskThreshold)
}
