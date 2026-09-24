// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
)

var _ VulnReporter = &AggregateReporter{}

type AggregateReporter struct {
	mu               sync.Mutex
	assets           map[string]*inventory.Asset
	assetReports     map[string]*policy.Report
	assetVulnReports map[string]*mvd.VulnReport
	assetErrors      map[string]error
	// assetWarnings holds non-fatal issues (AddScanWarning) keyed by asset
	// MRN, kept separate from assetErrors so a warning never makes Reports()
	// report the asset as failed. Serialized into policy.ReportCollection's
	// Warnings field -- see Reports().
	assetWarnings    map[string][]string
	bundle           *policy.Bundle
	resolvedPolicies map[string]*policy.ResolvedPolicy
	worstScore       *policy.Score
}

func NewAggregateReporter() *AggregateReporter {
	return &AggregateReporter{
		assets:           make(map[string]*inventory.Asset),
		assetReports:     map[string]*policy.Report{},
		assetErrors:      map[string]error{},
		assetWarnings:    map[string][]string{},
		resolvedPolicies: map[string]*policy.ResolvedPolicy{},
		assetVulnReports: map[string]*mvd.VulnReport{},
	}
}

func (r *AggregateReporter) AddBundle(bundle *policy.Bundle) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bundle == nil {
		r.bundle = bundle
		return
	}
	r.bundle = policy.Merge(r.bundle, bundle)
}

func (r *AggregateReporter) AddReport(asset *inventory.Asset, results *AssetReport) {
	log.Debug().Str("asset", asset.Name).Msg("add scan result to report")

	r.mu.Lock()
	defer r.mu.Unlock()
	r.assets[asset.Mrn] = asset
	r.assetReports[asset.Mrn] = results.Report
	r.resolvedPolicies[asset.Mrn] = results.ResolvedPolicy

	if r.worstScore == nil || results.Report.Score.Value < r.worstScore.Value {
		r.worstScore = results.Report.Score
	}
}

func (r *AggregateReporter) AddVulnReport(asset *inventory.Asset, vulnReport *gql.VulnReport) {
	if vulnReport == nil {
		return
	}
	log.Debug().Str("asset", asset.Name).Msg("add scan result to report")

	mvdVulnReport := gql.ConvertToMvdVulnReport(vulnReport)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assets[asset.Mrn] = asset
	r.assetVulnReports[asset.Mrn] = mvdVulnReport
}

func (r *AggregateReporter) AddScanError(asset *inventory.Asset, err error) {
	log.Debug().Err(err).Str("asset", asset.Name).Msg("add scan error to report")
	if err != nil && strings.Contains(strings.ToUpper(err.Error()), "TOOMANYREQUESTS") {
		log.Warn().Msg("container registry rate limit reached. Configure registry credentials to authenticate and increase your pull rate limit. See https://www.docker.com/increase-rate-limit")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assets[asset.Mrn] = asset
	r.assetErrors[asset.Mrn] = err
}

// AddScanWarning records non-fatal issues for an asset that still produced a
// report. Kept out of assetErrors deliberately: Reports() derives Ok and the
// asset's presence in the error collection from assetErrors alone, and the
// CLI exits non-zero whenever that collection is non-empty (see
// apps/cnspec/cmd/scan.go), so folding a warning in there would flip the run
// to a failing exit code for a scan that mostly succeeded.
func (r *AggregateReporter) AddScanWarning(asset *inventory.Asset, warnings []string) {
	if len(warnings) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assets[asset.Mrn] = asset
	r.assetWarnings[asset.Mrn] = append(r.assetWarnings[asset.Mrn], warnings...)
}

// Warnings returns the non-fatal issues recorded via AddScanWarning, keyed
// by asset MRN. Also reachable via Reports().Result.Full.Warnings, this is
// the in-process shortcut -- e.g. for a test asserting a crash was recorded
// without being treated as a scan failure.
func (r *AggregateReporter) Warnings() map[string][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string][]string, len(r.assetWarnings))
	for k, v := range r.assetWarnings {
		out[k] = append([]string(nil), v...)
	}
	return out
}

func (r *AggregateReporter) Reports() *ScanResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	errors := make(map[string]string, len(r.assetErrors))
	for k, v := range r.assetErrors {
		errors[k] = v.Error()
	}

	var warnings map[string]*policy.ScanWarnings
	if len(r.assetWarnings) > 0 {
		warnings = make(map[string]*policy.ScanWarnings, len(r.assetWarnings))
		for k, v := range r.assetWarnings {
			warnings[k] = &policy.ScanWarnings{Messages: append([]string(nil), v...)}
		}
	}

	return &ScanResult{
		Ok:         len(errors) == 0,
		WorstScore: r.worstScore,
		Result: &ScanResult_Full{
			Full: &policy.ReportCollection{
				Assets:           r.assets,
				Reports:          r.assetReports,
				Errors:           errors,
				Bundle:           r.bundle,
				ResolvedPolicies: r.resolvedPolicies,
				VulnReports:      r.assetVulnReports,
				Warnings:         warnings,
			},
		},
	}
}

func (r *AggregateReporter) AccumulatedBytes() (reports, resolvedPolicies, vulnReports, assets int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rpt := range r.assetReports {
		reports += rpt.SizeVT()
	}
	for _, rp := range r.resolvedPolicies {
		resolvedPolicies += rp.SizeVT()
	}
	for _, vr := range r.assetVulnReports {
		vulnReports += vr.SizeVT()
	}
	for _, a := range r.assets {
		assets += a.SizeVT()
	}
	return
}

func (r *AggregateReporter) Error() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var err error

	for _, curError := range r.assetErrors {
		err = multierror.Append(err, curError)
	}
	return err
}
