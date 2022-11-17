package scan

import (
	"github.com/hashicorp/go-multierror"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnspec/policy"
)

type Reporter interface {
	AddReport(asset *asset.Asset, results *AssetReport)
	AddScanError(asset *asset.Asset, err error)
}

type AssetReport struct {
	Mrn            string
	ResolvedPolicy *policy.ResolvedPolicy
	Bundle         *policy.Bundle
	Report         *policy.Report
}

type AggregateReporter struct {
	assets           map[string]*policy.Asset
	assetReports     map[string]*policy.Report
	assetErrors      map[string]error
	bundle           *policy.Bundle
	resolvedPolicies map[string]*policy.ResolvedPolicy
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
	r.assets[asset.Mrn] = &policy.Asset{
		Mrn:  asset.Mrn,
		Name: asset.Name,
		Url:  asset.Url,
	}
	r.assetReports[asset.Mrn] = results.Report
	r.resolvedPolicies[asset.Mrn] = results.ResolvedPolicy

	r.bundle = results.Bundle
}

func (r *AggregateReporter) AddScanError(asset *asset.Asset, err error) {
	r.assetErrors[asset.Mrn] = err
}

func (r *AggregateReporter) Reports() *policy.ReportCollection {
	errors := make(map[string]string, len(r.assetErrors))
	for k, v := range r.assetErrors {
		errors[k] = v.Error()
	}

	return &policy.ReportCollection{
		Assets:           r.assets,
		Reports:          r.assetReports,
		Errors:           errors,
		Bundle:           r.bundle,
		ResolvedPolicies: r.resolvedPolicies,
	}
}

func (r *AggregateReporter) Error() error {
	var err error

	for _, curError := range r.assetErrors {
		err = multierror.Append(err, curError)
	}
	return err
}
