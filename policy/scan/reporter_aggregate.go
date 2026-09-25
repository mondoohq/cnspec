// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/llx"
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
	if llx.KindOf(err) == llx.ErrorKind_ERROR_KIND_TOO_MANY_REQUESTS && pullsFromRegistry(asset) {
		log.Warn().Msg("container registry rate limit reached. Configure registry credentials to authenticate and increase your pull rate limit. See https://www.docker.com/increase-rate-limit")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assets[asset.Mrn] = asset
	r.assetErrors[asset.Mrn] = err
}

// pullsFromRegistry reports whether scanning asset pulls an image from a
// container registry, where a rate limit usually means anonymous pulls.
func pullsFromRegistry(asset *inventory.Asset) bool {
	for _, conn := range asset.GetConnections() {
		switch conn.GetType() {
		case "docker-image", "docker-registry", "container-registry", "registry-image":
			return true
		}
	}
	return false
}

func (r *AggregateReporter) Reports() *ScanResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	errors := make(map[string]string, len(r.assetErrors))
	var errorDetails map[string]*llx.ErrorDetail
	for k, v := range r.assetErrors {
		errors[k] = v.Error()
		if detail := llx.ErrorDetailOf(v); detail != nil {
			if errorDetails == nil {
				errorDetails = map[string]*llx.ErrorDetail{}
			}
			errorDetails[k] = detail
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
				ErrorDetails:     errorDetails,
				Bundle:           r.bundle,
				ResolvedPolicies: r.resolvedPolicies,
				VulnReports:      r.assetVulnReports,
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
