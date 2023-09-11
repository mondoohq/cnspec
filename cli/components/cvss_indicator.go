// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream/mvd/cvss"
)

func NewCvssIndicator() CvssIndicator {
	theme := colors.DefaultColorTheme

	cvssRatingColorMapping := map[cvss.Severity]termenv.Color{
		cvss.None:     theme.Good,
		cvss.Low:      theme.Low,
		cvss.Medium:   theme.Medium,
		cvss.High:     theme.High,
		cvss.Critical: theme.Critical,
		cvss.Unknown:  theme.Unknown,
	}

	return CvssIndicator{
		indicatorChar:          '■',
		cvssRatingColorMapping: cvssRatingColorMapping,
	}
}

type CvssIndicator struct {
	indicatorChar rune

	// colors for cvss ratings
	cvssRatingColorMapping map[cvss.Severity]termenv.Color
}

func (ci CvssIndicator) Render(severity cvss.Severity) string {
	return termenv.String(string(ci.indicatorChar)).Foreground(ci.rating(severity)).String()
}

func (ci CvssIndicator) rating(r cvss.Severity) termenv.Color {
	c, ok := ci.cvssRatingColorMapping[r]
	if ok {
		return c
	}
	return ci.cvssRatingColorMapping[cvss.Unknown]
}
