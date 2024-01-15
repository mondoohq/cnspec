// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/json"
	"errors"
	"strconv"

	cr "go.mondoo.com/cnquery/v10/cli/reporter"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/shared"
	"go.mondoo.com/cnspec/v10/policy"
)

func printScore(score *policy.Score, mrn string, out shared.OutputHelper, prefix string) bool {
	if score == nil {
		return false
	}

	status := score.TypeLabel()
	if score.Type == policy.ScoreType_Result {
		if score.Value == 100 {
			status = "pass"
		} else {
			status = "fail"
		}
	}

	out.WriteString(prefix + llx.PrettyPrintString(mrn) +
		":{\"score\":" + strconv.FormatUint(uint64(score.Value), 10) + "," +
		"\"status\":\"" + status + "\"}")
	return true
}

// asssetPrintable is a snapshot of the fields that get exported
// when doing things like JSON output
type assetPrintable struct {
	Mrn          string `protobuf:"bytes,1,opt,name=mrn,proto3" json:"mrn,omitempty"`
	Name         string `protobuf:"bytes,18,opt,name=name,proto3" json:"name,omitempty"`
	Url          string `protobuf:"bytes,19,opt,name=url,proto3" json:"url,omitempty"`
	PlatformName string `protobuf:"bytes,20,opt,name=platformName,proto3" json:"platformName,omitempty"`
}

func prepareAssetsForPrinting(assets map[string]*inventory.Asset) map[string]*assetPrintable {
	printableAssets := map[string]*assetPrintable{}
	for k, a := range assets {
		pAsset := &assetPrintable{
			Mrn:          a.Mrn,
			Name:         a.Name,
			Url:          a.Url,
			PlatformName: getPlatformNameForAsset(a),
		}
		printableAssets[k] = pAsset
	}

	return printableAssets
}

func ReportCollectionToJSON(data *policy.ReportCollection, out shared.OutputHelper) error {
	if data == nil {
		return nil
	}

	var qid2mrn map[string]string
	aggregateQueries := []string{}
	if data.Bundle != nil {
		qid2mrn = make(map[string]string, len(data.Bundle.Queries))
		for i := range data.Bundle.Queries {
			query := data.Bundle.Queries[i]
			if query.CodeId == "" {
				aggregateQueries = append(aggregateQueries, query.Mrn)
			} else {
				qid2mrn[query.CodeId] = query.Mrn
			}
		}
	} else {
		qid2mrn = make(map[string]string, 0)
	}

	out.WriteString(
		"{" +
			"\"assets\":")
	// preserve json output to ignore recently introduce fields
	printableAssets := prepareAssetsForPrinting(data.Assets)
	// assets, err := json.Marshal(data.Assets)
	assets, err := json.Marshal(printableAssets)
	if err != nil {
		return err
	}
	out.WriteString(string(assets))

	out.WriteString("," +
		"\"data\":" +
		"{")
	pre := ""
	for id, report := range data.Reports {
		out.WriteString(pre + llx.PrettyPrintString(id) + ":{")
		pre = ","

		resolved, ok := data.ResolvedPolicies[id]
		if !ok {
			return errors.New("cannot find resolved pack for " + id + " in report")
		}

		results := report.RawResults()
		pre2 := ""
		for qid, query := range resolved.ExecutionJob.Queries {
			mrn := qid2mrn[qid]
			// policies and other stuff
			if mrn == "" {
				continue
			}
			// checks
			if _, ok := report.Scores[qid]; ok {
				continue
			}

			out.WriteString(pre2 + llx.PrettyPrintString(mrn) + ":")
			pre2 = ","

			err := cr.BundleResultsToJSON(query.Code, results, out)
			if err != nil {
				return err
			}
		}
		out.WriteString("}")
	}

	out.WriteString("}," +
		"\"scores\":" +
		"{")
	pre = ""
	for id, report := range data.Reports {
		out.WriteString(pre + llx.PrettyPrintString(id) + ":{")
		pre = ","

		resolved, ok := data.ResolvedPolicies[id]
		if !ok {
			return errors.New("cannot find resolved pack for " + id + " in report")
		}

		pre2 := ""
		// try to get the policy first
		if printScore(report.Scores[id], id, out, pre2) {
			pre2 = ","
		}

		for qid := range resolved.ExecutionJob.Queries {
			mrn := qid2mrn[qid]
			// policies and other stuff
			if mrn == "" {
				continue
			}

			if printScore(report.Scores[qid], mrn, out, pre2) {
				pre2 = ","
			}
		}

		for _, mrn := range aggregateQueries {
			if printScore(report.Scores[mrn], mrn, out, pre2) {
				pre2 = ","
			}
		}

		out.WriteString("}")
	}

	out.WriteString("}," +
		"\"errors\":" +
		"{")
	pre = ""
	for id, err := range data.Errors {
		out.WriteString(pre + llx.PrettyPrintString(id) + ":" + llx.PrettyPrintString(err))
		pre = ","
	}
	out.WriteString("}}")

	return nil
}
