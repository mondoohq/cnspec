// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"errors"
	"os"

	"sigs.k8s.io/yaml"
)

func (v *ScoreValues) GetScore(mrn string) *ScoreValue {
	if s, ok := v.Values[mrn]; ok {
		return s
	}

	return nil
}

// FromSingleFile loads a cnspec report bundle from a single file
func FromSingleFile(path string) (*Report, error) {
	reportData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return FromJSON(reportData)
}

// FromJSON creates a cnspec report from json contents
func FromJSON(data []byte) (*Report, error) {
	var res Report
	err := yaml.Unmarshal(data, &res)
	return &res, err
}

// AssetMrn returns the MRN of the asset if there is only one
func (r Report) AssetMrn() (string, error) {

	if len(r.Assets) > 1 {
		return "", errors.New("report contains more than one asset")
	}

	if len(r.Assets) == 0 {
		return "", errors.New("report contains no assets")
	}

	for _, asset := range r.Assets {
		return asset.Mrn, nil
	}

	// should not happen
	return "", errors.New("report contains no assets")
}
