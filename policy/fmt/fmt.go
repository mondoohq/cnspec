// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package fmt

import (
	"go.mondoo.com/cnspec/v12/policy"
	"go.mondoo.com/cnspec/v12/policy/tooling"
)

// Deprecated: use tooling.Format
func ToFormattedYAML(p *policy.Bundle) ([]byte, error) {
	data, err := p.ToYAML()
	if err != nil {
		return nil, err
	}
	return tooling.Format(data)
}
