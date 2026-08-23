// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Device Inventory Info (OCSF class 5001): one event per asset, so the lake knows
// what was scanned even when the scan produced no findings.

package convert

import (
	"bytes"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
	cr "go.mondoo.com/mql/cli/reporter"
	"go.mondoo.com/mql/utils/iox"
)

// inventoryInfo reports the asset itself, so the lake knows what was scanned even
// when the scan produced no findings.
func (c *converter) inventoryInfo(r *policy.ReportCollection, ctx *assetContext) ocsf.InventoryInfo {
	res := ocsf.NewInventoryInfo(ocsf.InventoryInfoActivityCollect)
	res.Cloud = ctx.cloud
	if ctx.device != nil {
		res.Device = *ctx.device
	}
	res.Time = c.now
	res.SeverityID = ocsf.SeverityInformational
	res.Severity = ocsf.SeverityName(res.SeverityID)
	// device is a class attribute of inventory_info, not a host-profile one, so
	// only the cloud profile has to be declared here.
	var profiles []string
	if ctx.cloud != nil {
		profiles = append(profiles, ocsf.ProfileCloud)
	}
	res.Metadata = c.metadata(profiles...)
	res.Unmapped = c.assetUnmapped(r, ctx)
	return res
}

// assetUnmapped carries the asset details and, when data output is enabled, the
// results of the data-only queries of the scan.
func (c *converter) assetUnmapped(r *policy.ReportCollection, ctx *assetContext) map[string]string {
	res := map[string]string{}
	if ctx.assetMrn != "" {
		res["asset_mrn"] = ctx.assetMrn
	}
	asset := ctx.asset
	if len(asset.PlatformIds) > 0 {
		res["platform_ids"] = strings.Join(asset.PlatformIds, ",")
	}
	for k, v := range asset.Labels {
		res["label."+k] = v
	}
	for k, v := range asset.Annotations {
		res["annotation."+k] = v
	}
	if asset.Platform != nil {
		if asset.Platform.Kind != "" {
			res["platform_kind"] = asset.Platform.Kind
		}
		if asset.Platform.Runtime != "" {
			res["platform_runtime"] = asset.Platform.Runtime
		}
	}

	if !c.includeData {
		return res
	}
	for mrn, value := range assetDataResults(r, ctx.assetMrn) {
		// Capped for the same reason an assessment is: one data query over a large
		// resource set renders megabytes of JSON, and an oversized event is dropped
		// whole by an HTTP collector rather than trimmed.
		res["data."+mrn] = truncateUnmapped(value, "data")
	}
	return res
}

// assetDataResults renders the results of the asset's data-only queries as JSON,
// keyed by query MRN.
func assetDataResults(r *policy.ReportCollection, assetMrn string) map[string]string {
	report, ok := r.Reports[assetMrn]
	if !ok {
		return nil
	}
	resolved, ok := r.ResolvedPolicies[assetMrn]
	if !ok || resolved.ExecutionJob == nil || resolved.CollectorJob == nil {
		return nil
	}

	qid2mrn := map[string]string{}
	if r.Bundle != nil {
		for _, query := range r.Bundle.Queries {
			if query.CodeId != "" {
				qid2mrn[query.CodeId] = query.Mrn
			}
		}
	}

	reportingJobs := map[string]*policy.ReportingJob{}
	for _, job := range resolved.CollectorJob.ReportingJobs {
		reportingJobs[job.QrId] = job
	}

	results := report.RawResults()
	res := map[string]string{}
	for qid, query := range resolved.ExecutionJob.Queries {
		mrn := qid2mrn[qid]
		if mrn == "" {
			continue
		}
		if job, ok := reportingJobs[mrn]; ok &&
			job.Type != policy.ReportingJob_DATA_QUERY && job.Type != policy.ReportingJob_CHECK_AND_DATA_QUERY {
			continue
		}

		buf := &bytes.Buffer{}
		w := iox.IOWriter{Writer: buf}
		if err := cr.CodeBundleToJSON(query.Code, results, &w); err != nil {
			log.Warn().Err(err).Str("query", mrn).Msg("could not render a data query result for the OCSF report")
			continue
		}
		res[mrn] = strings.TrimSpace(buf.String())
	}
	return res
}
