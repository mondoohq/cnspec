// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
)

func (r *Report) RawResults() map[string]*llx.RawResult {
	results := map[string]*llx.RawResult{}

	// covert all proto results to raw results
	for k := range r.Data {
		result := r.Data[k]
		results[k] = result.RawResultV2()
	}

	return results
}

// Stats computes the stats for this report
func (r *Report) ComputeStats(resolved *ResolvedPolicy) {
	res := Stats{
		Failed: &ScoreDistribution{},
		Passed: &ScoreDistribution{},
		Errors: &ScoreDistribution{},
	}

	queries := resolved.CollectorJob.ReportingQueries

	for id, score := range r.Scores {
		if _, ok := queries[id]; ok {
			res.Add(score)
		}
	}

	r.Stats = &res
}
func (s *Stats) Add(score *Score) {
	s.AddCount(score, 1)
}

func (s *Stats) AddCount(score *Score, count uint32) {
	s.Total += count
	switch score.Type {
	case ScoreType_Unknown:
		s.Unknown += count
	case ScoreType_Result:
		if score.Value < 100 {
			s.Failed.AddCount(score, count)

			if score.Value < s.Worst {
				s.Worst = score.Value
			}

		} else {
			s.Passed.AddCount(score, count)
		}
	case ScoreType_Error:
		s.Errors.AddCount(score, count)
	case ScoreType_Skip:
		s.Skipped += count
	case ScoreType_Unscored:
		s.Unknown += count
	default:
		log.Warn().Uint32("type", score.Type).Str("id", score.QrId).Msg("ran into unknown score type")
	}
}

// this function also handles nil scores and updates the score distribution accordingly
func (sd *ScoreDistribution) Add(score *Score) {
	sd.AddCount(score, 1)
}

func (sd *ScoreDistribution) AddCount(score *Score, count uint32) {
	sd.AddRatingCount(score.Rating(), count)
}

// this function also handles nil scores and updates the score distribution accordingly
func (sd *ScoreDistribution) Remove(score *Score) {
	sd.RemoveRating(score.Rating())
}

func (sd *ScoreDistribution) AddRating(scoreRating ScoreRating) {
	sd.AddRatingCount(scoreRating, 1)
}

func (sd *ScoreDistribution) AddRatingCount(scoreRating ScoreRating, count uint32) {
	sd.Total += count
	switch scoreRating {
	case ScoreRating_aPlus, ScoreRating_a, ScoreRating_aMinus:
		sd.A += count
	case ScoreRating_bPlus, ScoreRating_b, ScoreRating_bMinus:
		sd.B += count
	case ScoreRating_cPlus, ScoreRating_c, ScoreRating_cMinus:
		sd.C += count
	case ScoreRating_dPlus, ScoreRating_d, ScoreRating_dMinus:
		sd.D += count
	case ScoreRating_failed:
		sd.F += count
	case ScoreRating_error:
		sd.Error += count
	case ScoreRating_unrated:
		sd.Unrated += count
	}
}

func (x *ScoreDistribution) AddScoreDistribution(y *ScoreDistribution) *ScoreDistribution {
	return &ScoreDistribution{
		Total:   x.GetTotal() + y.GetTotal(),
		A:       x.GetA() + y.GetA(),
		B:       x.GetB() + y.GetB(),
		C:       x.GetC() + y.GetC(),
		D:       x.GetD() + y.GetD(),
		F:       x.GetF() + y.GetF(),
		Error:   x.GetError() + y.GetError(),
		Unrated: x.GetUnrated() + y.GetUnrated(),
	}
}

func (sd *ScoreDistribution) RemoveRating(scoreRating ScoreRating) {
	sd.Total--
	switch scoreRating {
	case ScoreRating_aPlus, ScoreRating_a, ScoreRating_aMinus:
		sd.A--
	case ScoreRating_bPlus, ScoreRating_b, ScoreRating_bMinus:
		sd.B--
	case ScoreRating_cPlus, ScoreRating_c, ScoreRating_cMinus:
		sd.C--
	case ScoreRating_dPlus, ScoreRating_d, ScoreRating_dMinus:
		sd.D--
	case ScoreRating_failed:
		sd.F--
	case ScoreRating_error:
		sd.Error--
	case ScoreRating_unrated:
		sd.Unrated--
	}
}

func (p *ReportCollection) ToJSON() ([]byte, error) {
	// removes the data to ensure the data is not exported
	// NOTE: this has the side-effect that data is manipulated and a console print on the same struct
	// would not work. When we need that, we need to copy the struct before we export it
	for k := range p.Reports {
		p.Reports[k].Data = nil
	}

	// pretty print json
	return json.MarshalIndent(p, "", "  ")
}

func (r *ReportCollection) GetWorstScore() uint32 {
	worstScore := uint32(100) // pass
	for _, r := range r.Reports {
		if r == nil || r.Score == nil {
			continue
		}

		if r.Score.Value < worstScore {
			worstScore = r.Score.Value
		}
	}

	return worstScore
}
