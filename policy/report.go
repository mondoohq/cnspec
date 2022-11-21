package policy

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
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
	s.Total++
	switch score.Type {
	case ScoreType_Unknown:
		s.Unknown++
	case ScoreType_Result:
		if score.Value < 100 {
			s.Failed.Add(score)

			if score.Value < s.Worst {
				s.Worst = score.Value
			}

		} else {
			s.Passed.Add(score)
		}
	case ScoreType_Error:
		s.Errors.Add(score)
	case ScoreType_Skip:
		s.Skipped++
	case ScoreType_Unscored:
		s.Unknown++
	default:
		log.Warn().Uint32("type", score.Type).Str("id", score.QrId).Msg("ran into unknown score type")
	}
}

func (sd *ScoreDistribution) Add(score *Score) {
	sd.Total++

	switch score.Rating() {
	case ScoreRating_aPlus, ScoreRating_a, ScoreRating_aMinus:
		sd.A++
	case ScoreRating_bPlus, ScoreRating_b, ScoreRating_bMinus:
		sd.B++
	case ScoreRating_cPlus, ScoreRating_c, ScoreRating_cMinus:
		sd.C++
	case ScoreRating_dPlus, ScoreRating_d, ScoreRating_dMinus:
		sd.D++
	case ScoreRating_failed:
		sd.F++
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
