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

type AggregateReporter struct {
	assetReports map[string]*policy.Report
	assetErrors  map[string]error
}

func NewAggregateReporter() *AggregateReporter {
	return &AggregateReporter{
		assetReports: map[string]*policy.Report{},
		assetErrors:  map[string]error{},
	}
}

func (r *AggregateReporter) AddReport(asset *asset.Asset, results *AssetReport) {
	r.assetReports[asset.Mrn] = results.Report
}

func (r *AggregateReporter) AddScanError(asset *asset.Asset, err error) {
	r.assetErrors[asset.Mrn] = err
}

func (r *AggregateReporter) Reports() []*policy.Report {
	res := make([]*policy.Report, len(r.assetReports))
	var i int
	for _, report := range r.assetReports {
		res[i] = report
		i++
	}
	return res
}

func (r *AggregateReporter) Error() error {
	var err error

	for _, curError := range r.assetErrors {
		err = multierror.Append(err, curError)
	}
	return err
}
