// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"go.mondoo.com/cnspec/policy"
	cr "go.mondoo.com/mql/cli/reporter"
	"go.mondoo.com/mql/utils/iox"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func ConvertToProto(data *policy.ReportCollection) (*Report, error) {
	protoReport := &Report{
		Assets: map[string]*cr.Asset{},
		Data:   map[string]*cr.DataValues{},
		Errors: map[string]string{},
		Scores: map[string]*ScoreValues{},
	}

	if data == nil {
		return protoReport, nil
	}

	// Queries with identical MQL compile to the same code id and run once, so a
	// code id can stand for several queries; each of them gets an entry.
	qid2mrns := map[string][]string{}
	aggregateQueries := []string{}
	if data.Bundle != nil {
		for i := range data.Bundle.Queries {
			query := data.Bundle.Queries[i]
			if query.CodeId == "" {
				aggregateQueries = append(aggregateQueries, query.Mrn)
			} else {
				qid2mrns[query.CodeId] = append(qid2mrns[query.CodeId], query.Mrn)
			}
		}
	}

	// fill in assets
	for assetMrn, a := range data.Assets {
		var platformName string
		if a.Platform != nil {
			platformName = a.Platform.Name
		}
		pAsset := &cr.Asset{
			Mrn:          a.GetMrn(),
			Name:         a.GetName(),
			PlatformName: platformName,
			TraceId:      a.GetTraceId(),
			Labels:       a.GetLabels(),
		}
		protoReport.Assets[assetMrn] = pAsset
	}

	// convert the data points to json
	for id, report := range data.Reports {
		assetMrn := prettyPrintString(id)

		resolved, ok := data.ResolvedPolicies[id]
		if !ok {
			return nil, errors.New("cannot find resolved pack for " + id + " in report")
		}

		reportingJobByQrId := map[string]*policy.ReportingJob{}
		for _, job := range resolved.CollectorJob.ReportingJobs {
			reportingJobByQrId[job.QrId] = job
		}

		// Code id to MRN, recovered from the resolved policy.
		//
		// qid2mrn above is built from bundle.Queries, which only helps when those
		// queries carry a code id. A query pack's do not: the pack defines its
		// queries inline in groups and they are compiled during resolution, so
		// the bundle attached to the report holds definitions with an empty
		// CodeId. Every data query then mapped to an empty MRN and was dropped,
		// which is why a query pack produced scores and no data -- `cnspec sbom`
		// emitted an SBOM with no packages and "no data points found".
		//
		// The resolved policy has both halves: an EXECUTION_QUERY job is keyed by
		// code id, and the DATA_QUERY job that owns it is keyed by MRN and names
		// it as a child.
		execCodeIdByUuid := map[string]string{}
		for _, job := range resolved.CollectorJob.ReportingJobs {
			if job.Type == policy.ReportingJob_EXECUTION_QUERY {
				execCodeIdByUuid[job.Uuid] = job.QrId
			}
		}
		resolvedQid2Mrns := map[string][]string{}
		for _, job := range resolved.CollectorJob.ReportingJobs {
			if job.Type != policy.ReportingJob_DATA_QUERY && job.Type != policy.ReportingJob_CHECK_AND_DATA_QUERY {
				continue
			}
			for childUuid := range job.ChildJobs {
				if codeId, ok := execCodeIdByUuid[childUuid]; ok {
					resolvedQid2Mrns[codeId] = append(resolvedQid2Mrns[codeId], job.QrId)
				}
			}
		}

		results := report.RawResults()
		if resolved.ExecutionJob == nil {
			continue
		}
		for qid, query := range resolved.ExecutionJob.Queries {
			mrns := queryMrns(qid2mrns[qid], reportingJobByQrId)
			if len(mrns) == 0 {
				// a query pack's queries are only identifiable through the
				// resolved policy; see resolvedQid2Mrns above
				mrns = resolvedQid2Mrns[qid]
			}

			// policies and other stuff have no query mrn
			for _, mrn := range mrns {
				// checks
				if rj, ok := reportingJobByQrId[mrn]; ok {
					if rj.Type != policy.ReportingJob_DATA_QUERY && rj.Type != policy.ReportingJob_CHECK_AND_DATA_QUERY {
						continue
					}
				}

				buf := &bytes.Buffer{}
				w := iox.IOWriter{Writer: buf}
				err := cr.CodeBundleToJSON(query.Code, results, &w)
				if err != nil {
					return nil, err
				}

				var v *structpb.Value
				var jsonStruct map[string]any
				err = json.Unmarshal([]byte(buf.Bytes()), &jsonStruct)
				if err == nil {
					v, err = structpb.NewValue(jsonStruct)
					if err != nil {
						return nil, err
					}
				} else {
					v, err = structpb.NewValue(buf.String())
					if err != nil {
						return nil, err
					}
				}

				if protoReport.Data[assetMrn] == nil {
					protoReport.Data[assetMrn] = &cr.DataValues{
						Values: map[string]*cr.DataValue{},
					}
				}

				protoReport.Data[assetMrn].Values[mrn] = &cr.DataValue{
					Content: v,
				}
			}
		}
	}

	// convert scores
	for mrn, report := range data.Reports {
		if protoReport.Scores[mrn] == nil {
			protoReport.Scores[mrn] = &ScoreValues{
				Values: map[string]*ScoreValue{},
			}
		}

		score := gatherScoreValue(report.Scores[mrn])
		if score != nil {
			protoReport.Scores[mrn].Values[mrn] = score
		}

		resolved, ok := data.ResolvedPolicies[mrn]
		if !ok {
			return nil, errors.New("cannot find resolved pack for " + mrn + " in report")
		}

		reportingJobByQrId := map[string]*policy.ReportingJob{}
		for _, job := range resolved.GetCollectorJob().GetReportingJobs() {
			reportingJobByQrId[job.QrId] = job
		}

		// Getters, not field access: a ResolvedPolicy with no ExecutionJob is a
		// nil pointer dereference here, and this runs on the `-o json` path, so
		// the failure is a panic in the middle of writing a report rather than
		// an error. Reached with a resolved policy that carries only a
		// CollectorJob.
		for qid := range resolved.GetExecutionJob().GetQueries() {
			// policies and other stuff have no query mrn
			for _, qmrn := range queryMrns(qid2mrns[qid], reportingJobByQrId) {
				score := gatherScoreValue(report.Scores[qid])
				if score != nil {
					protoReport.Scores[mrn].Values[qmrn] = score
				}
			}
		}

		for _, qmrn := range aggregateQueries {
			score := gatherScoreValue(report.Scores[qmrn])
			if score != nil {
				protoReport.Scores[mrn].Values[qmrn] = score
			}
		}
	}

	for id, errStatus := range data.Errors {
		assetMrn := prettyPrintString(id)
		errorMsg := errStatus
		protoReport.Errors[assetMrn] = errorMsg
	}

	return protoReport, nil
}

func (r *Report) ToJSON() ([]byte, error) {
	return protojson.Marshal(r)
}

func (r *Report) ToCnqueryReport() *cr.Report {
	report := &cr.Report{
		Assets: map[string]*cr.Asset{},
		Data:   map[string]*cr.DataValues{},
		Errors: map[string]string{},
	}

	for id, asset := range r.Assets {
		report.Assets[id] = &cr.Asset{
			Mrn:     asset.GetMrn(),
			Name:    asset.GetName(),
			TraceId: asset.GetTraceId(),
			Labels:  asset.GetLabels(),
		}
	}

	for id, data := range r.Data {
		report.Data[id] = &cr.DataValues{
			Values: map[string]*cr.DataValue{},
		}
		for mid, value := range data.GetValues() {
			report.Data[id].Values[mid] = &cr.DataValue{
				Content: value.GetContent(),
			}
		}
	}

	for id, err := range r.Errors {
		report.Errors[id] = err
	}

	return report
}

func JsonValue(v *structpb.Value) ([]byte, error) {
	return protojson.Marshal(v)
}

// similar to llx.PrettyPrintString but no double quotes around the string
func prettyPrintString(s string) string {
	res := s
	res = strings.ReplaceAll(res, "\\n", "\n")
	res = strings.ReplaceAll(res, "\\t", "\t")
	return res
}

func gatherScoreValue(score *policy.Score) *ScoreValue {
	if score == nil {
		return nil
	}

	status := score.TypeLabel()
	if score.Type == policy.ScoreType_Result {
		if score.Value == 100 {
			status = "pass"
		} else {
			status = "fail"
		}
	}

	return &ScoreValue{
		Status:    status,
		RiskScore: 100 - score.Value,
	}
}
