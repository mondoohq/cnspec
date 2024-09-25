// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"sort"
	"strings"

	"go.mondoo.com/cnquery/v11/cli/printer"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnspec/v11/policy"
)

var mqlQueryNames = map[string]string{
	"true": "General",
	"platform.family.contains(_ == 'unix') || platform.family.contains(_ == 'linux') || platform.family.contains(_ == 'windows')": "Operating System",
	"platform.name == \"vmware-vsphere\"": "vSphere",
	"platform.name == \"vmware-esxi\"":    "ESXi",
	"platform.name == \"aws\"":            "Amazon Web Services",
	"platform.name == \"arista-eos\"":     "Arista",
}

func hasAssetFilter(supported []*explorer.Mquery, filters *explorer.Filters) bool {
	if len(supported) == 0 || filters == nil || len(filters.Items) == 0 {
		return false
	}

	for _, query := range filters.Items {
		for j := range supported {
			if supported[j].Mql == query.Mql {
				return true
			}
		}
	}
	return false
}

func renderAssetOverview(print *printer.Printer, policyObj *policy.Policy, report *policy.Report, bundle *policy.PolicyBundleMap, resolvedPolicy *policy.ResolvedPolicy, scoringData []reportRow) string {
	var res bytes.Buffer

	type row struct {
		Title string
		Value string
	}

	res.WriteString(print.H2(policyObj.Name))
	results := report.RawResults()

	// TODO: refactor once we have the json/dict export for policies since it will make the access a lot easier
	// iterate over the data queries get the title and display the individual results
	for i := range policyObj.Groups {
		group := policyObj.Groups[i]

		// filter by asset filter that do not match
		if !hasAssetFilter(resolvedPolicy.Filters, group.Filters) {
			continue
		}

		// FIXME: use spec name from bundle if available
		// FIXME: while transitioning to v2 policy this code now really needs cleanup
		category := "General"
		if group.Filters != nil {
			for i := range group.Filters.Items {
				f := group.Filters.Items[i]
				if c, ok := mqlQueryNames[f.Mql]; ok {
					category = c
					break
				}
			}
		}

		table := []row{}
		maxKeyWidth := 0

		for j := range group.Queries {
			q := group.Queries[j]
			query := bundle.Queries[q.Mrn]

			if len(query.Title) > maxKeyWidth {
				maxKeyWidth = len(query.Title)
			}

			codeBundle := resolvedPolicy.GetCodeBundle(query)
			if codeBundle == nil {
				res.WriteString(NewLineCharacter + print.Error("failed to find code bundle for query '"+query.Mrn+"' in bundle"))
				continue
			}

			// print data results
			filteredResults := codeBundle.FilterResults(results)
			value := print.Datas(codeBundle, filteredResults)

			table = append(table, row{
				Title: query.Title,
				Value: value,
			})
		}

		// sort row by title
		sort.Slice(table, func(i, j int) bool {
			return table[i].Title < table[j].Title
		})

		res.WriteString(print.Primary(category) + ":" + NewLineCharacter)
		for i := range table {
			row := table[i]
			whitespace := maxKeyWidth - len(row.Title)
			res.WriteString("  " + row.Title + ":")
			res.WriteString(strings.Repeat(" ", whitespace+1))
			writeQueryCompact(&res, row.Value)
		}
		res.WriteString(NewLineCharacter)
	}

	return res.String()
}
