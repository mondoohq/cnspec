// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"sort"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v9/cli/printer"
	"go.mondoo.com/cnspec/v9/cli/components"
	"go.mondoo.com/cnspec/v9/policy"
)

var colorProfile func(string) termenv.Color = termenv.ColorProfile().Color

func renderMetaPolicy(print *printer.Printer, policyObj *policy.Policy, report *policy.Report, bundle *policy.PolicyBundleMap, resolvedPolicy *policy.ResolvedPolicy, scoringData []reportRow) string {
	var res bytes.Buffer

	// custom name for space or asset mrn
	name := policyObj.Name
	res.WriteString(print.H2(name))

	// extract list of policies and gather policy name from bundle
	policies := map[string]string{}
	for i := range policyObj.Groups {
		group := policyObj.Groups[i]
		for k := range group.Policies {
			ref := group.Policies[k]
			name := ref.Mrn
			if p, ok := bundle.Policies[ref.Mrn]; ok && p.Name != "" {
				name = p.Name
			}
			policies[ref.Mrn] = name
		}
	}

	// sort list of policies by asset overview, then name
	policyList := []string{}
	for key := range policies {
		policyList = append(policyList, key)
	}

	// this sorts the list of policies by name but ensures all unrated policies are following below
	// TODO: we need to improve that for data policies, will be solved once we can distingush between
	// not applicable and unrated
	sort.Slice(policyList, func(i, j int) bool {
		// check for asset overview policy
		if policyList[i] == assetOverviewPolicyMrn {
			return true
		}
		if policyList[j] == assetOverviewPolicyMrn {
			return false
		}

		// sort unscored policies at the end
		scoreI := report.Scores[policyList[i]]
		scoreJ := report.Scores[policyList[j]]

		if scoreI == nil && scoreJ != nil {
			return true
		}

		if scoreJ == nil && scoreI != nil {
			return true
		}

		// sort by name
		return policies[policyList[i]] < policies[policyList[j]]
	})

	// render policy list
	mircoScoreCard := components.NewMicroScoreCard()
	for i := range policyList {
		k := policyList[i]
		score := report.Scores[k]

		// do not print not applicable policies in overview
		if score == nil || score.Type == policy.ScoreType_Unknown {
			continue
		}

		res.WriteString("■ ")
		res.WriteString(mircoScoreCard.Render(score))
		res.WriteString(" ")
		res.WriteString(policies[k])
		res.WriteString(NewLineCharacter)
	}
	res.WriteString(NewLineCharacter)

	return res.String()
}
