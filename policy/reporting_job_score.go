// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import "github.com/cockroachdb/errors"

// ChildScore represents a child's score and its impact for reporting job
// score calculation.
type ChildScore struct {
	Score  *Score
	Impact *Impact
}

// IsForwardScoreType returns true if the reporting job type is a forward-score
// type (CHECK, DATA_QUERY, CHECK_AND_DATA_QUERY, EXECUTION_QUERY). Forward-score
// jobs pass through their single child's score with optional impact adjustment,
// as opposed to aggregate jobs which combine multiple children using a ScoringSystem.
func IsForwardScoreType(t ReportingJob_Type) bool {
	return t == ReportingJob_CHECK ||
		t == ReportingJob_DATA_QUERY ||
		t == ReportingJob_CHECK_AND_DATA_QUERY ||
		t == ReportingJob_EXECUTION_QUERY
}

// CalculateReportingJobScore calculates the score for a reporting job based on
// its children's scores. This is the single code path used by both the graph
// executor and the standalone rollup.
//
// For forward-score jobs (CHECK, DATA_QUERY, etc.), the single child's score
// is forwarded with optional impact floor and DISABLED handling.
//
// For aggregate jobs (POLICY, CONTROL, FRAMEWORK, etc.), child scores are
// combined using the specified ScoringSystem. CONTROL-type jobs remap
// error scores to 0 and skip/unscored to 100 before aggregation.
func CalculateReportingJobScore(
	queryID string,
	rjType ReportingJob_Type,
	scoringSystem ScoringSystem,
	children []ChildScore,
	totalDatapoints int,
	finishedDatapoints int,
	featureFlagFailErrors bool,
) (*Score, error) {
	if IsForwardScoreType(rjType) {
		return calculateForwardScore(queryID, children, totalDatapoints, finishedDatapoints)
	}
	return calculateAggregateScore(queryID, rjType, scoringSystem, children, totalDatapoints, finishedDatapoints, featureFlagFailErrors)
}

func calculateForwardScore(queryID string, children []ChildScore, totalDatapoints int, finishedDatapoints int) (*Score, error) {
	if len(children) > 1 {
		return nil, errors.Newf("forward-score job %q has %d children, expected 0 or 1", queryID, len(children))
	}

	var s *Score

	if len(children) == 0 {
		s = &Score{
			QrId:            queryID,
			Type:            ScoreType_Unscored,
			ScoreCompletion: 100,
		}
	} else {
		c := children[0]
		if c.Score == nil {
			s = &Score{
				QrId: queryID,
				Type: ScoreType_Result,
			}
		} else {
			s = c.Score.CloneVT()
			s.QrId = queryID

			if c.Impact.GetScoring() == ScoringSystem_DISABLED {
				s.Type = ScoreType_Disabled
			} else if s.Type == ScoreType_Result {
				if c.Impact != nil && c.Impact.Value != nil {
					floor := 100 - uint32(c.Impact.Value.Value)
					if floor > s.Value {
						s.Value = floor
					}
				}
			}
		}
	}

	if totalDatapoints > 0 {
		s.DataTotal = uint32(totalDatapoints)
		s.DataCompletion = uint32((100 * finishedDatapoints) / totalDatapoints)
	}

	return s, nil
}

func calculateAggregateScore(
	queryID string,
	rjType ReportingJob_Type,
	scoringSystem ScoringSystem,
	children []ChildScore,
	totalDatapoints int,
	finishedDatapoints int,
	featureFlagFailErrors bool,
) (*Score, error) {
	var calcOpts []ScoreCalculatorOption
	if featureFlagFailErrors {
		calcOpts = append(calcOpts, WithScoreCalculatorFeatureFlagFailErrors())
	}

	calculator, err := NewScoreCalculator(scoringSystem, calcOpts...)
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		cs := child.Score
		if cs == nil {
			// If a child hasn't reported yet, we can't calculate the aggregate
			return nil, nil
		}

		if rjType == ReportingJob_CONTROL {
			cs = remapControlScore(cs)
		}

		AddSpecdScore(calculator, cs, true, child.Impact)
	}

	AddDataScore(calculator, totalDatapoints, finishedDatapoints)

	s := calculator.Calculate()
	s.QrId = queryID
	return s, nil
}

// remapControlScore remaps non-Result scores for CONTROL-type reporting jobs:
// errors become failed (value=0), skip/unscored become passing (value=100).
func remapControlScore(s *Score) *Score {
	if s.Type == ScoreType_Result {
		return s
	}
	clone := s.CloneVT()
	switch clone.Type {
	case ScoreType_Error:
		clone.Type = ScoreType_Result
		clone.Value = 0
	case ScoreType_Skip, ScoreType_Unscored:
		clone.Type = ScoreType_Result
		clone.Value = 100
	}
	return clone
}
