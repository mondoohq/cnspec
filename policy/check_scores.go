// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

// ExceptedQrIds returns the query and policy ids that an exception, an ignore
// or a disable takes out of scoring: every reporting job that some parent
// reaches over an IGNORE_SCORE or DISABLED edge. Reporters list these checks
// as exceptions rather than as findings.
func (rp *ResolvedPolicy) ExceptedQrIds() map[string]struct{} {
	if rp == nil || rp.CollectorJob == nil {
		return nil
	}

	uuidToQrId := make(map[string]string, len(rp.CollectorJob.ReportingJobs))
	for _, rj := range rp.CollectorJob.ReportingJobs {
		uuidToQrId[rj.Uuid] = rj.QrId
	}

	exceptions := map[string]struct{}{}
	for _, rj := range rp.CollectorJob.ReportingJobs {
		for childUuid, impact := range rj.ChildJobs {
			if impact == nil {
				continue
			}
			if impact.Scoring == ScoringSystem_IGNORE_SCORE ||
				impact.Scoring == ScoringSystem_DISABLED {
				if qrId, ok := uuidToQrId[childUuid]; ok {
					exceptions[qrId] = struct{}{}
				}
			}
		}
	}
	return exceptions
}

// WorstCheckScore is the lowest rated score among the checks in report: the
// worst finding on the asset. A check's score is kept under its MRN with the
// check's impact already applied, so a failing check with impact 80 reads as
// risk 80 here, the same risk the reporters print for it.
//
// Excepted checks (see ExceptedQrIds) are left out, and so is every result
// that is not a rated ScoreType_Result: a check that errored, was skipped, or
// has not finished has no measured risk. It returns nil when no check was
// rated.
func WorstCheckScore(resolved *ResolvedPolicy, report *Report) *Score {
	if resolved == nil || resolved.CollectorJob == nil || report == nil {
		return nil
	}

	excepted := resolved.ExceptedQrIds()
	var worst *Score
	for _, rj := range resolved.CollectorJob.ReportingJobs {
		if rj.Type != ReportingJob_CHECK && rj.Type != ReportingJob_CHECK_AND_DATA_QUERY {
			continue
		}
		if _, ok := excepted[rj.QrId]; ok {
			continue
		}
		score := report.Scores[rj.QrId]
		if score == nil || score.Type != ScoreType_Result || score.Rating() == ScoreRating_unrated {
			continue
		}
		if worst == nil || score.Value < worst.Value {
			worst = score
		}
	}
	return worst
}
