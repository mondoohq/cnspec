// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tooling

import (
	bundlefmt "go.mondoo.com/cnspec/v13/internal/bundle"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
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
	SkipProviderDownload bool
}

func Lint(schema resources.ResourcesSchema, filename string, data []byte, opts LintOptions) []*bundlefmt.Entry {
	return bundlefmt.LintPolicyBundle(schema, filename, data, bundlefmt.LintOptions{
		SkipProviderDownload: opts.SkipProviderDownload,
	})
}
