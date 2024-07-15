// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/checksums"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/utils/multierr"
)

type ScoredRiskInfo struct {
	*RiskFactor
	*ScoredRiskFactor
}

func (r *RiskFactor) DetectScope() {
	if r.Scope != ScopeType_UNSCOPED {
		return
	}
	resourceScoped := len(r.Resources) != 0
	softwareScoped := len(r.Software) != 0
	if resourceScoped && softwareScoped {
		r.Scope = ScopeType_SOFTWARE_AND_RESOURCE
	} else if resourceScoped {
		r.Scope = ScopeType_RESOURCE
	} else if softwareScoped {
		r.Scope = ScopeType_SOFTWARE
	} else {
		r.Scope = ScopeType_ASSET
	}
}

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

func (r *RiskFactor) ExecutionChecksum(ctx context.Context, conf mqlc.CompilerConfig) (checksums.Fast, error) {
	c := checksums.New.
		AddUint(uint64(r.Scope)).
		Add(strconv.FormatFloat(float64(r.GetMagnitude().GetValue()), 'f', -1, 64))

	if r.GetMagnitude().GetIsToxic() {
		c = c.AddUint(1)
	} else {
		c = c.AddUint(0)
	}

	var err error
	c, err = r.Filters.ComputeChecksum(c, r.Mrn, conf)
	if err != nil {
		return c, err
	}

	for i := range r.Checks {
		check := r.Checks[i]
		if err := check.RefreshChecksum(ctx, conf, nil); err != nil {
			return c, err
		}

		if check.Checksum == "" {
			return c, errors.New("failed to get checksum for risk query " + check.Mrn)
		}

		c = c.Add(check.Checksum)
	}

	for i := range r.Software {
		sw := r.Software[i]
		c = c.Add(sw.MqlMrn).Add(sw.Name).Add(sw.Namespace).Add(sw.Type).Add(sw.Version)
	}

	for i := range r.Resources {
		rc := r.Resources[i]
		c = c.Add(rc.Name)
	}

	return c, nil
}

// RefreshChecksum updates the Checksum field of this RiskFactor and returns
// both the ExecutionChecksum and the ContentChecksum.
func (r *RiskFactor) RefreshChecksum(ctx context.Context, conf mqlc.CompilerConfig) (checksums.Fast, checksums.Fast, error) {
	csum := checksums.New

	esum, err := r.ExecutionChecksum(ctx, conf)
	if err != nil {
		return esum, csum, err
	}

	csum = csum.AddUint(uint64(esum)).
		Add(r.Mrn).
		Add(r.Title)

	if r.Docs != nil {
		csum = csum.Add(r.Docs.Active).Add(r.Docs.Inactive)
	}

	r.Checksum = csum.String()
	return esum, csum, nil
}

func (r *RiskFactor) AdjustRiskScore(score *Score, isDetected bool) {
	// Absolute risk factors only play a role when they are detected.
	if r.GetMagnitude().GetIsToxic() {
		if isDetected {
			nu := int(score.RiskScore) - int(r.GetMagnitude().GetValue()*100)
			if nu < 0 {
				score.RiskScore = 0
			} else if nu > 100 {
				score.RiskScore = 100
			} else {
				score.RiskScore = uint32(nu)
			}

			if score.RiskFactors == nil {
				score.RiskFactors = &ScoredRiskFactors{}
			}
			score.RiskFactors.Items = append(score.RiskFactors.Items, &ScoredRiskFactor{
				Mrn:        r.Mrn,
				Risk:       r.GetMagnitude().GetValue(),
				IsToxic:    true,
				IsDetected: isDetected,
			})
			return
		}
		// We don't adjust anything in case an absolute risk factor is not detected
		return
	}

	if r.GetMagnitude().GetValue() < 0 {
		if isDetected {
			score.RiskScore = uint32(100 - float32(100-score.RiskScore)*(1+r.GetMagnitude().GetValue()))
			if score.RiskFactors == nil {
				score.RiskFactors = &ScoredRiskFactors{}
			}
			score.RiskFactors.Items = append(score.RiskFactors.Items, &ScoredRiskFactor{
				Mrn:        r.Mrn,
				Risk:       r.GetMagnitude().GetValue(),
				IsDetected: isDetected,
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
			Mrn:        r.Mrn,
			Risk:       r.GetMagnitude().GetValue(),
			IsDetected: isDetected,
		})
		return
	}

	score.RiskScore = uint32(100 - float32(100-score.RiskScore)*(1-r.GetMagnitude().GetValue()))
	if score.RiskFactors == nil {
		score.RiskFactors = &ScoredRiskFactors{}
	}
	score.RiskFactors.Items = append(score.RiskFactors.Items, &ScoredRiskFactor{
		Mrn:        r.Mrn,
		Risk:       -r.GetMagnitude().GetValue(),
		IsDetected: isDetected,
	})
}

func (s *ScoredRiskFactors) Add(other *ScoredRiskFactors) {
	if other == nil {
		return
	}

	for i := range other.Items {
		nu := other.Items[i]

		found := false
		for j := range s.Items {
			if s.Items[j].Mrn == nu.Mrn {
				s.Items[j] = nu
				found = true
				break
			}
		}

		if !found {
			s.Items = append(s.Items, nu)
		}
	}
}

func (s *RiskMagnitude) UnmarshalJSON(data []byte) error {
	var f float32
	if err := json.Unmarshal(data, &f); err == nil {
		s.Value = f
		return nil
	}

	type tmp RiskMagnitude
	return json.Unmarshal(data, (*tmp)(s))
}

func (s *RiskFactor) UnmarshalJSON(data []byte) error {
	type TmpRiskFactorType RiskFactor
	type tmp struct {
		*TmpRiskFactorType `json:",inline"`
		IsAbsolute         *bool `json:"is_absolute"`
	}

	t := tmp{TmpRiskFactorType: (*TmpRiskFactorType)(s)}
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	if s.Magnitude == nil {
		s.Magnitude = &RiskMagnitude{}
	}

	if t.IsAbsolute != nil {
		s.Magnitude.IsToxic = *t.IsAbsolute
	}

	return nil
}

func (s *RiskFactor) Migrate() {
	if s.Magnitude == nil {
		s.Magnitude = &RiskMagnitude{
			Value:   s.DeprecatedV11Magnitude,
			IsToxic: s.DeprecatedV11IsAbsolute,
		}
	}
	s.DeprecatedV11IsAbsolute = s.Magnitude.IsToxic
	s.DeprecatedV11Magnitude = s.Magnitude.Value
}
