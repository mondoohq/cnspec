// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package fmt

import "go.mondoo.com/cnspec/v10/policy"
import bundlefmt "go.mondoo.com/cnspec/v10/internal/bundle"

func ToFormattedYAML(p *policy.Bundle) ([]byte, error) {
	data, err := p.ToYAML()
	if err != nil {
		return nil, err
	}

	b, err := bundlefmt.ParseYaml(data)
	if err != nil {
		return nil, err
	}
	fmtData, err := bundlefmt.FormatBundle(b, true)
	if err != nil {
		return nil, err
	}

	return fmtData, nil
}
