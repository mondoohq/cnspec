// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tooling

import (
	bundlefmt "go.mondoo.com/cnspec/internal/bundle"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
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

// CodeBlock is one fenced code block of a markdown text, such as a check's
// remediation.
type CodeBlock = bundlefmt.CodeBlock

// ParseCodeBlocks returns the fenced code blocks of a markdown text in order.
// It is the parser `cnspec policy graph remediation` uses, exported so that a
// program reading a remediation's code gets the same blocks that command
// prints: a fence is three or more backticks or tildes, the closing fence uses
// the same character and is at least as long, and a block left open runs to
// the end of the text. A text without a fence has no code blocks.
func ParseCodeBlocks(markdown string) []CodeBlock {
	return bundlefmt.ParseCodeBlocks(markdown)
}
