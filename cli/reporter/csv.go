// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/csv"
	"strconv"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/utils/iox"
)

// scanCSVRow is one line of the scan CSV: one check, on one asset.
//
// One row per asset × check, rather than one per asset, because the whole
// reason to ask for CSV is a spreadsheet -- filter by status, sort by score,
// pivot by asset. An asset-shaped row would need the check results nested in a
// cell, which is the one thing CSV cannot express.
type scanCSVRow struct {
	Asset           string
	AssetMrn        string
	Platform        string
	PlatformVersion string
	Check           string
	Title           string
	Status          string
	Score           string
	Impact          string
	Message         string
}

func (r scanCSVRow) toSlice() []string {
	return []string{
		r.Asset, r.AssetMrn, r.Platform, r.PlatformVersion,
		r.Check, r.Title, r.Status, r.Score, r.Impact, r.Message,
	}
}

func scanCSVHeader() []string {
	return scanCSVRow{
		Asset:           "Asset",
		AssetMrn:        "Asset MRN",
		Platform:        "Platform",
		PlatformVersion: "Platform Version",
		Check:           "Check",
		Title:           "Title",
		Status:          "Status",
		Score:           "Score",
		Impact:          "Impact",
		Message:         "Message",
	}.toSlice()
}

// ConvertToCSV writes a scan report as CSV.
//
// Status uses the same vocabulary as the JSON report -- "pass", "fail",
// "error", "skip", "unscored", "disabled", "out of scope" -- because it comes
// from the same place, gatherScoreValue. A format that renamed the outcomes
// would make two exports of one scan disagree.
//
// Score is the 0-100 score, where 100 is a pass. The JSON report carries the
// inverted riskScore instead; here the score is the number a spreadsheet will
// be sorted and conditionally formatted on, and "higher is better" matches
// every other place cnspec shows a score to a person.
func ConvertToCSV(data *policy.ReportCollection, out iox.OutputHelper) error {
	w := csv.NewWriter(out)
	if err := w.Write(scanCSVHeader()); err != nil {
		return err
	}

	// data is nil when no asset was scanned. A header-only file is the honest
	// answer: the export ran and found nothing, which a consumer can tell apart
	// from a truncated download.
	if data == nil {
		w.Flush()
		return w.Error()
	}

	// Sorted throughout, so re-running the same scan produces the same file.
	// Go randomises map iteration and these get diffed and committed.
	for _, assetMrn := range sortedKeys(data.Errors) {
		asset := data.Assets[assetMrn]
		row := scanCSVRow{
			Asset:    asset.GetName(),
			AssetMrn: assetMrn,
			Platform: asset.GetPlatform().GetName(),
			// An asset that failed to scan gets a row of its own rather than
			// being left out. Omitting it would make a failed scan and a clean
			// one look the same in a spreadsheet, which is the mistake the
			// whole reporting path is careful not to make.
			Status:  "error",
			Message: data.Errors[assetMrn],
		}
		if err := w.Write(escapeCSVRow(row.toSlice())); err != nil {
			return err
		}
	}

	var queries map[string]*policy.Mquery
	if data.Bundle != nil {
		queries = reportdoc.QueryMap(data.Bundle.ToMap())
	}

	for _, assetMrn := range sortedKeys(data.Assets) {
		asset := data.Assets[assetMrn]

		report, ok := data.Reports[assetMrn]
		if !ok {
			continue // the asset error above already covers it
		}
		resolved, ok := data.ResolvedPolicies[assetMrn]
		if !ok {
			continue
		}

		for _, id := range sortedKeys(report.Scores) {
			// ReportingQueries is what separates a check from the aggregate
			// scores that share the same map -- the asset's own score, and one
			// per policy. Without this filter the CSV would carry rows that
			// look like checks and have no query behind them.
			if _, ok := resolved.GetCollectorJob().GetReportingQueries()[id]; !ok {
				continue
			}
			query, ok := queries[id]
			if !ok {
				continue
			}

			score := report.Scores[id]
			row := scanCSVRow{
				Asset:           asset.GetName(),
				AssetMrn:        assetMrn,
				Platform:        asset.GetPlatform().GetName(),
				PlatformVersion: asset.GetPlatform().GetVersion(),
				Check:           checkIdentifier(query, id),
				Title:           query.GetTitle(),
				Status:          gatherScoreValue(score).GetStatus(),
				Impact:          impactValue(query),
				Message:         score.MessageLine(),
			}
			if score != nil && score.Type == policy.ScoreType_Result {
				// Only a scored result has a meaningful number. Writing 0 for a
				// skipped or errored check would sort it alongside a real
				// failure.
				row.Score = strconv.FormatUint(uint64(score.Value), 10)
			}

			if err := w.Write(escapeCSVRow(row.toSlice())); err != nil {
				return err
			}
		}
	}

	w.Flush()
	return w.Error()
}

// checkIdentifier prefers the UID, which is what the check is called in the
// policy YAML and what someone reading the spreadsheet can search for. An
// upstream-resolved bundle has MRNs and no UIDs, so the MRN is the fallback,
// and the score's own id is the last resort.
func checkIdentifier(query *policy.Mquery, scoreID string) string {
	if uid := query.GetUid(); uid != "" {
		return uid
	}
	if mrn := query.GetMrn(); mrn != "" {
		return mrn
	}
	return scoreID
}

// impactValue renders a check's declared impact, empty when it declares none.
// Empty rather than 0: an undeclared impact is not the lowest impact.
func impactValue(query *policy.Mquery) string {
	v := query.GetImpact().GetValue()
	if v == nil {
		return ""
	}
	return strconv.FormatInt(int64(v.GetValue()), 10)
}
