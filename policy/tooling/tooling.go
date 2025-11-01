// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tooling

import (
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"
	bundlefmt "go.mondoo.com/cnspec/v12/internal/bundle"
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

type LintOptions struct {
	AutoUpdateProviders bool
}

func Lint(schema resources.ResourcesSchema, filename string, data []byte, opts LintOptions) []*bundlefmt.Entry {
	return bundlefmt.LintPolicyBundle(schema, filename, data, bundlefmt.LintOptions{
		AutoUpdateProviders: opts.AutoUpdateProviders,
	})
}
