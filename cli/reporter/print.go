// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"sort"
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnspec/v10/policy"
)

type Format byte

const (
	FormatCompact Format = iota + 1
	FormatSummary
	FormatFull
	FormatReport
	FormatYAML
	FormatJSON
	FormatJUnit
	FormatCSV
)

// Formats that are supported by the reporter
var Formats = map[string]Format{
	"compact": FormatCompact,
	"summary": FormatSummary,
	"full":    FormatFull,
	"":        FormatCompact,
	"report":  FormatReport,
	"yaml":    FormatYAML,
	"yml":     FormatYAML,
	"json":    FormatJSON,
	"junit":   FormatJUnit,
	"csv":     FormatCSV,
}

func AllFormats() string {
	var res []string
	for k := range Formats {
		if k != "" && // default if nothing is provided, ignore
			k != "yml" { // don't show both yaml and yml
			res = append(res, k)
		}
	}
	sort.Strings(res)
	return strings.Join(res, ", ")
}

func (r *Reporter) scoreColored(rating policy.ScoreRating, s string) string {
	switch rating {
	case policy.ScoreRating_aPlus, policy.ScoreRating_a, policy.ScoreRating_aMinus:
		return termenv.String(s).Foreground(r.Colors.Good).String()
	case policy.ScoreRating_bPlus, policy.ScoreRating_b, policy.ScoreRating_bMinus:
		return termenv.String(s).Foreground(r.Colors.Low).String()
	case policy.ScoreRating_cPlus, policy.ScoreRating_c, policy.ScoreRating_cMinus:
		return termenv.String(s).Foreground(r.Colors.Medium).String()
	case policy.ScoreRating_dPlus, policy.ScoreRating_d, policy.ScoreRating_dMinus:
		return termenv.String(s).Foreground(r.Colors.High).String()
	case policy.ScoreRating_failed:
		return termenv.String(s).Foreground(r.Colors.Critical).String()
	}
	return s
}

func getPlatformNameForAsset(asset *inventory.Asset) string {
	platformName := ""
	if asset.Platform != nil {
		if asset.Platform.Title == "" {
			platformName = asset.Platform.Name
		} else {
			platformName = asset.Platform.Title
		}
	}
	return platformName
}
