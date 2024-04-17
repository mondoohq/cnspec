// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	cr "go.mondoo.com/cnquery/v11/cli/reporter"
	"go.mondoo.com/cnquery/v11/shared"
	"go.mondoo.com/cnspec/v11/policy"
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

	// fill in assets
	for assetMrn, a := range data.Assets {
		var platformName string
		if a.Platform != nil {
			platformName = a.Platform.Name
		}
		pAsset := &cr.Asset{
			Mrn:          a.Mrn,
			Name:         a.Name,
			PlatformName: platformName,
			TraceId:      a.TraceId,
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

		results := report.RawResults()
		if resolved.ExecutionJob == nil {
			continue
		}
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

			buf := &bytes.Buffer{}
			w := shared.IOWriter{Writer: buf}
			err := cr.CodeBundleToJSON(query.Code, results, &w)
			if err != nil {
				return nil, err
			}

			var v *structpb.Value
			var jsonStruct map[string]interface{}
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

		for qid := range resolved.ExecutionJob.Queries {
			qmrn := qid2mrn[qid]
			// policies and other stuff
			if qmrn == "" {
				continue
			}

			score := gatherScoreValue(report.Scores[qid])
			if score != nil {
				protoReport.Scores[mrn].Values[qmrn] = score
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
			Mrn:     asset.Mrn,
			Name:    asset.Name,
			TraceId: asset.TraceId,
		}
	}

	for id, data := range r.Data {
		report.Data[id] = &cr.DataValues{
			Values: map[string]*cr.DataValue{},
		}
		for mid, value := range data.Values {
			report.Data[id].Values[mid] = &cr.DataValue{
				Content: value.Content,
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
		Score:  score.Value,
		Status: status,
	}
}
