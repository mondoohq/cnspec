// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/checksums"
	"go.mondoo.com/cnquery/v10/mqlc"
	"go.mondoo.com/cnquery/v10/utils/multierr"
)

func (r *RiskFactor) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, r.Mrn, MRN_RESOURCE_RISK, r.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", r.Uid).Msg("failed to refresh mrn")
		return multierr.Wrap(err, "failed to refresh mrn for query "+r.Title)
	}

	r.Mrn = nu
	r.Uid = ""

	for i := range r.Checks {
		if err := r.Checks[i].RefreshMRN(ownerMRN); err != nil {
			return err
		}
	}

	return nil
}

func (r *RiskFactor) RefreshChecksum(conf mqlc.CompilerConfig) error {
	c := checksums.New.
		Add(r.Mrn).
		Add(r.Title).
		Add(r.Docs.Active).
		Add(r.Docs.Inactive).
		AddUint(uint64(r.Scope)).
		AddUint(uint64(r.Magnitude))

	var err error
	c, err = r.Filters.ComputeChecksum(c, r.Mrn, conf)
	if err != nil {
		return err
	}

	for i := range r.Checks {
		check := r.Checks[i]
		if err := check.RefreshChecksum(context.Background(), conf, nil); err != nil {
			return err
		}
	}

	if r.IsAbsolute {
		c = c.Add("1")
	} else {
		c = c.Add("0")
	}

	for i := range r.Software {
		cur := r.Software[i]
		c = c.Add(cur.Type).
			Add(cur.Name).
			Add(cur.Namespace).
			Add(cur.Version).
			Add(cur.MqlMrn)
	}
	for i := range r.Resources {
		cur := r.Resources[i]
		c = c.Add(cur.Selector)
	}

	r.Checksum = c.String()
	return nil
}

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

			if score.RiskFactors == nil {
				score.RiskFactors = &ScoredRiskFactors{}
			}
			score.RiskFactors.Items = append(score.RiskFactors.Items, &ScoredRiskFactor{
				Mrn:        r.Mrn,
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
			if score.RiskFactors == nil {
				score.RiskFactors = &ScoredRiskFactors{}
			}
			score.RiskFactors.Items = append(score.RiskFactors.Items, &ScoredRiskFactor{
				Mrn:  r.Mrn,
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
		if score.RiskFactors == nil {
			score.RiskFactors = &ScoredRiskFactors{}
		}
		score.RiskFactors.Items = append(score.RiskFactors.Items, &ScoredRiskFactor{
			Mrn:  r.Mrn,
			Risk: r.Magnitude,
		})
		return
	}

	score.Value = uint32(100 - float32(100-score.Value)*(1-r.Magnitude))
	if score.RiskFactors == nil {
		score.RiskFactors = &ScoredRiskFactors{}
	}
	score.RiskFactors.Items = append(score.RiskFactors.Items, &ScoredRiskFactor{
		Mrn:  r.Mrn,
		Risk: -r.Magnitude,
	})
}
