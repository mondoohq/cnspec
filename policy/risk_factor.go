// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

func (r *RiskFactor) AdjustScore(score *Score, isDetected bool) {
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

			score.RiskFactors = append(score.RiskFactors, &ScoredRiskFactor{
				Id:         r.Uid,
				Risk:       r.Magnitude,
				IsAbsolute: true,
			})
			return
		}
		// We don't adjust anything in case an absolute risk factor is not detected
		return
	}

	if r.Magnitude < 0 {
		if isDetected {
			score.Value = uint32(100 - float32(100-score.Value)*(1+r.Magnitude))
			score.RiskFactors = append(score.RiskFactors, &ScoredRiskFactor{
				Id:   r.Uid,
				Risk: r.Magnitude,
			})
			return
		}
		// Relative risk factors that only decrease risk don't get flagged in
		// case they are not detected
		return
	}

	// For relative risk factors we have to adjust both the detected and
	// not detected score. The non-detected score needs to be decreased,
	// since it's a relative risk factors. The detected score just needs
	// the flag to indicate its risk was "increased" (relative to non-detected)
	if isDetected {
		score.RiskFactors = append(score.RiskFactors, &ScoredRiskFactor{
			Id:   r.Uid,
			Risk: r.Magnitude,
		})
		return
	}

	score.Value = uint32(100 - float32(100-score.Value)*(1-r.Magnitude))
	score.RiskFactors = append(score.RiskFactors, &ScoredRiskFactor{
		Id:   r.Uid,
		Risk: -r.Magnitude,
	})
}
