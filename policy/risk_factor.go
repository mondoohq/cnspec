// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

func (r *RiskFactor) AdjustScore(score *Score, isDetected bool) *ScoredRiskFactor {
	// Absolute risk factors only play a role when they are detected.
	if r.IsAbsolute {
		if isDetected {
			nu := int(score.Value) - int(r.Magnitude*100)
			if nu < 0 {
				score.Value = 0
			} else if nu > 100 {
				score.Value = 100
			} else {
				score.Value = uint32(nu)
			}

			return &ScoredRiskFactor{
				Id:         r.Uid,
				Risk:       r.Magnitude,
				IsAbsolute: true,
			}
		} else {
			return nil
		}
	}

	if r.Magnitude < 0 {
		if isDetected {
			score.Value = uint32(100 - float32(100-score.Value)*(1+r.Magnitude))
			return &ScoredRiskFactor{
				Id:   r.Uid,
				Risk: r.Magnitude,
			}
		} else {
			return nil
		}
	}

	if isDetected {
		return &ScoredRiskFactor{
			Id:   r.Uid,
			Risk: r.Magnitude,
		}
	}

	score.Value = uint32(100 - float32(100-score.Value)*(1-r.Magnitude))
	return &ScoredRiskFactor{
		Id:   r.Uid,
		Risk: -r.Magnitude,
	}
}
