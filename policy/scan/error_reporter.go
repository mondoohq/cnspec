// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"strings"

	"go.mondoo.com/cnquery/v9/cli/theme"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnspec/v9/policy"
	pbStatus "go.mondoo.com/ranger-rpc/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

type ErrorReporter struct {
	assets     map[string]*policy.Asset
	errors     map[string]string
	worstScore *policy.Score
}

func NewErrorReporter() Reporter {
	return &ErrorReporter{assets: make(map[string]*policy.Asset), errors: make(map[string]string)}
}

func (r *ErrorReporter) AddReport(asset *inventory.Asset, results *AssetReport) {
	if r.worstScore == nil || results.Report.Score.Value < r.worstScore.Value {
		r.worstScore = results.Report.Score
	}
}

func (r *ErrorReporter) AddVulnReport(asset *inventory.Asset, vulnReport *gql.VulnReport) {
}

func (c *ErrorReporter) AddScanError(asset *inventory.Asset, err error) {
	if c.errors == nil {
		c.errors = make(map[string]string)
	}
	name := findNameForAsset(asset)
	errMsg := assetScanErrToString(asset, err)
	c.assets[asset.Mrn] = &policy.Asset{
		Mrn:  asset.Mrn,
		Name: asset.Name,
		Url:  asset.Url,
	}
	c.errors[name] = errMsg
}

func (r *ErrorReporter) Reports() *ScanResult {
	return &ScanResult{
		Ok:         len(r.errors) == 0,
		WorstScore: r.worstScore,
		Result:     &ScanResult_Errors{Errors: &ErrorCollection{Errors: r.errors}},
	}
}

func findNameForAsset(assetObj *inventory.Asset) string {
	if assetObj == nil {
		return "unknown"
	}
	if assetObj.Name != "" {
		return assetObj.Name
	}
	if assetObj.Mrn != "" {
		return assetObj.Mrn
	}
	return "unknown"
}

func assetScanErrToString(assetObj *inventory.Asset, err error) string {
	st, ok := pbStatus.FromError(err)
	if !ok {
		return err.Error()
	}

	builder := strings.Builder{}
	builder.WriteString(st.Message())
	builder.WriteRune('\n')

	// print error details (optional)
	for _, detail := range st.Details() {
		switch t := detail.(type) {
		case *errdetails.ErrorInfo:
			if t.Domain == policy.POLICY_SERVICE_NAME {
				switch t.Reason {
				case "no-matching-policy":
					builder.WriteString("We could not find a policy that fits to your asset.\n")
					if t.Metadata != nil {
						builder.WriteString("Enable policies at: ")
						builder.WriteString(theme.DefaultTheme.Secondary(assetObj.Url))
						builder.WriteRune('\n')
					}
				}
			}
		}
	}

	return builder.String()
}
