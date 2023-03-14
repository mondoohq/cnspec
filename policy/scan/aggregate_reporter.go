package scan

import (
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnspec/policy"
)

type AggregateReporter struct {
	assets           map[string]*policy.Asset
	assetReports     map[string]*policy.Report
	assetErrors      map[string]error
	bundle           *policy.Bundle
	resolvedPolicies map[string]*policy.ResolvedPolicy
	worstScore       *policy.Score
}

func NewAggregateReporter() *AggregateReporter {
	return &AggregateReporter{
		assets:           make(map[string]*policy.Asset),
		assetReports:     map[string]*policy.Report{},
		assetErrors:      map[string]error{},
		resolvedPolicies: map[string]*policy.ResolvedPolicy{},
	}
}

func (r *AggregateReporter) AddReport(asset *asset.Asset, results *AssetReport) {
	log.Debug().Str("asset", asset.Name).Msg("add scan result to report")
	platformName := ""
	if asset.Platform.Title == "" {
		platformName = asset.Platform.Name
	} else {
		platformName = asset.Platform.Title
	}

	r.assets[asset.Mrn] = &policy.Asset{
		Mrn:          asset.Mrn,
		Name:         asset.Name,
		PlatformName: platformName,
		Url:          asset.Url,
	}
	r.assetReports[asset.Mrn] = results.Report
	r.resolvedPolicies[asset.Mrn] = results.ResolvedPolicy

	r.bundle = results.Bundle
	if r.worstScore == nil || results.Report.Score.Value < r.worstScore.Value {
		r.worstScore = results.Report.Score
	}
}

func (r *AggregateReporter) AddScanError(asset *asset.Asset, err error) {
	log.Debug().Str("asset", asset.Name).Msg("add scan error to report")
	r.assets[asset.Mrn] = &policy.Asset{
		Mrn:  asset.Mrn,
		Name: asset.Name,
		Url:  asset.Url,
	}
	if asset.Platform != nil {
		platformName := ""
		if asset.Platform.Title == "" {
			platformName = asset.Platform.Name
		} else {
			platformName = asset.Platform.Title
		}
		r.assets[asset.Mrn].PlatformName = platformName
	}
	r.assetErrors[asset.Mrn] = err
}

func (r *AggregateReporter) Reports() *ScanResult {
	errors := make(map[string]*explorer.ErrorStatus, len(r.assetErrors))
	for k, v := range r.assetErrors {
		errors[k] = explorer.NewErrorStatus(v)
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
