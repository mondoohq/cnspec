// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tooling

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	bundlefmt "go.mondoo.com/cnspec/v11/internal/bundle"
)

func Format(data []byte) ([]byte, error) {
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

func Lint(schema resources.ResourcesSchema, filename string, data []byte) []*bundlefmt.Entry {
	return bundlefmt.LintPolicyBundle(schema, filename, data)
}
