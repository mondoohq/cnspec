package scan

import (
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnspec/policy"
)

type NoOpReporter struct {
	assets map[string]*policy.Asset
}

func NewNoOpReporter(assetList []*asset.Asset) Reporter {
	assets := make(map[string]*policy.Asset, len(assetList))
	for i := range assetList {
		cur := assetList[i]
		assets[cur.Mrn] = &policy.Asset{
			Mrn:  cur.Mrn,
			Name: cur.Name,
			Url:  cur.Url,
		}
	}
	return &NoOpReporter{assets: assets}
}

func (r *NoOpReporter) AddReport(asset *asset.Asset, results *AssetReport) {
}

func (r *NoOpReporter) AddScanError(asset *asset.Asset, err error) {
}

func (r *NoOpReporter) Reports() *ScanResult {
	return &ScanResult{
		Result: &ScanResult_None{},
	}
}
