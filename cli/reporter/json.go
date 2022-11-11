package reporter

import (
	"encoding/json"
	"errors"
	"strconv"

	cr "go.mondoo.com/cnquery/cli/reporter"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnspec/policy"
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

func ReportCollectionToJSON(data *policy.ReportCollection, out shared.OutputHelper) error {
	if data == nil {
		return nil
	}

	queryMrnIdx := map[string]string{}
	for i := range data.Bundle.Queries {
		query := data.Bundle.Queries[i]
		queryMrnIdx[query.CodeId] = query.Mrn
	}

	out.WriteString(
		"{" +
			"\"assets\":")
	assets, err := json.Marshal(data.Assets)
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
			mrn := queryMrnIdx[qid]
			// policies and other stuff
			if mrn == "" {
				continue
			}
			// controls
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
			mrn := queryMrnIdx[qid]
			// policies and other stuff
			if mrn == "" {
				continue
			}

			if printScore(report.Scores[qid], mrn, out, pre2) {
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
