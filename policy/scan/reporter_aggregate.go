// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnspec/v10/policy"
)

var _ VulnReporter = &AggregateReporter{}

type AggregateReporter struct {
	assets           map[string]*inventory.Asset
	assetReports     map[string]*policy.Report
	assetVulnReports map[string]*mvd.VulnReport
	assetErrors      map[string]error
	bundle           *policy.Bundle
	resolvedPolicies map[string]*policy.ResolvedPolicy
	worstScore       *policy.Score
}

func NewAggregateReporter() *AggregateReporter {
	return &AggregateReporter{
		assets:           make(map[string]*inventory.Asset),
		assetReports:     map[string]*policy.Report{},
		assetErrors:      map[string]error{},
		resolvedPolicies: map[string]*policy.ResolvedPolicy{},
		assetVulnReports: map[string]*mvd.VulnReport{},
	}
}

func (r *AggregateReporter) AddBundle(bundle *policy.Bundle) {
	if r.bundle == nil {
		r.bundle = bundle
		return
	}
	r.bundle = policy.Merge(r.bundle, bundle)
}

func (r *AggregateReporter) AddReport(asset *inventory.Asset, results *AssetReport) {
	log.Debug().Str("asset", asset.Name).Msg("add scan result to report")

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
	r.assets[asset.Mrn] = asset
	r.assetVulnReports[asset.Mrn] = mvdVulnReport
}

func (r *AggregateReporter) AddScanError(asset *inventory.Asset, err error) {
	log.Debug().Str("asset", asset.Name).Msg("add scan error to report")
	r.assets[asset.Mrn] = asset
	r.assetErrors[asset.Mrn] = err
}

func (r *AggregateReporter) Reports() *ScanResult {
	errors := make(map[string]string, len(r.assetErrors))
	for k, v := range r.assetErrors {
		errors[k] = v.Error()
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
			},
		},
	}
}

func (r *AggregateReporter) Error() error {
	var err error

	for _, curError := range r.assetErrors {
		err = multierror.Append(err, curError)
	}
	return err
}
