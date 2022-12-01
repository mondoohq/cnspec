package reporter

import (
	"bytes"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/stringx"
	"go.mondoo.com/cnspec/policy"
)

var mqlQueryNames = map[string]string{
	"true": "General",
	"platform.family.contains(_ == 'unix') || platform.family.contains(_ == 'linux') || platform.family.contains(_ == 'windows')": "Operating System",
	"platform.name == \"vmware-vsphere\"": "vSphere",
	"platform.name == \"vmware-esxi\"":    "ESXi",
	"platform.name == \"aws\"":            "Amazon Web Services",
	"platform.name == \"arista-eos\"":     "Arista",
}

func hasAssetFilter(assetFilters []*policy.Mquery, filter *policy.Mquery) bool {
	if assetFilters == nil || filter == nil {
		return false
	}
	for i := range assetFilters {
		if assetFilters[i].Query == filter.Query {
			return true
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

	stringResult := func(bundle *llx.CodeBundle, results map[string]*llx.RawResult) string {
		var res strings.Builder
		for k := range results {
			v := results[k]
			if v == nil {
				log.Warn().Str("checksum", k).Msg("missing result")
				continue
			}
			r := v.Data
			res.WriteString(print.Data(r.Type, r.Value, v.CodeID, bundle, ""))
		}
		return res.String()
	}

	res.WriteString(print.H2(policyObj.Name))
	results := report.RawResults()

	// TODO: refactor once we have the json/dict export for policies since it will make the access a lot easier
	// iterate over the data queries get the title and display the individual results
	for i := range policyObj.Specs {
		spec := policyObj.Specs[i]

		// filter by asset filter that do not match
		if !hasAssetFilter(resolvedPolicy.Filters, spec.AssetFilter) {
			continue
		}

		// FIXME: use spec name from bundle if available
		category, ok := mqlQueryNames[spec.AssetFilter.Query]
		if !ok {
			category = "General"
		}

		table := []row{}
		maxKeyWidth := 0

		for qid := range spec.DataQueries {
			query := bundle.Queries[qid]

			if len(query.Title) > maxKeyWidth {
				maxKeyWidth = len(query.Title)
			}

			codeBundle := resolvedPolicy.GetCodeBundle(query)
			if codeBundle == nil {
				res.WriteString(NewLineCharacter + print.Error("failed to find code bundle for query '"+qid+"' in bundle"))
				continue
			}

			// print data results
			// copy all contents where we have labels
			filteredResults := map[string]*llx.RawResult{}
			for i := range codeBundle.CodeV2.Checksums {
				checksum := codeBundle.CodeV2.Checksums[i]

				res, ok := results[checksum]
				if ok {
					filteredResults[checksum] = res
				}
			}

			value := stringResult(codeBundle, filteredResults)

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

			if strings.Contains(row.Value, "\n") {
				res.WriteString(NewLineCharacter)
				res.WriteString(stringx.Indent(2, row.Value))
			} else {
				res.WriteString(row.Value)
				res.WriteString(NewLineCharacter)
			}
		}
		res.WriteString(NewLineCharacter)
	}

	return res.String()
}
