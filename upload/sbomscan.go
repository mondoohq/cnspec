// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"fmt"

	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/sbomscan"
	"go.mondoo.com/mql/v13/sbom"
	ranger "go.mondoo.com/ranger-rpc"
)

// sbomScanner is the slice of the ExtendedVulnMgmt client that ScanSBOM needs; a
// fake can stand in for tests.
type sbomScanner interface {
	ScanUploadedSbom(ctx context.Context, in *sbomscan.ScanUploadedSbomRequest) (*sbomscan.ScanUploadedSbomResponse, error)
}

// ScanSBOM sends an SBOM to Mondoo Platform's ExtendedVulnMgmt.ScanUploadedSbom
// and returns the resulting VEX. The scan is ephemeral — the platform stores
// nothing; the caller uploads the returned VEX itself (e.g. via UploadFindings).
// This is the same flow xgrep uses for dependency vulnerability scanning.
func ScanSBOM(ctx context.Context, opts Opts, bom *sbom.Sbom) ([]*fex.VulnerabilityExchange, error) {
	creds, spaceMrn, err := LoadCredentials(opts)
	if err != nil {
		return nil, err
	}
	plugin, err := upstream.NewServiceAccountRangerPlugin(creds)
	if err != nil {
		return nil, fmt.Errorf("create auth plugin: %w", err)
	}
	scanner, err := sbomscan.NewExtendedVulnMgmtClient(creds.ApiEndpoint, ranger.DefaultHttpClient(), plugin)
	if err != nil {
		return nil, fmt.Errorf("create vuln scan client: %w", err)
	}
	return scanSBOM(ctx, scanner, spaceMrn, bom)
}

// scanSBOM is the testable core: it issues the scan against an injected scanner.
func scanSBOM(ctx context.Context, scanner sbomScanner, spaceMrn string, bom *sbom.Sbom) ([]*fex.VulnerabilityExchange, error) {
	resp, err := scanner.ScanUploadedSbom(ctx, &sbomscan.ScanUploadedSbomRequest{
		// The field keeps the server's historical name asset_mrn; we pass the
		// space scope, which the server resolves.
		AssetMrn: spaceMrn,
		Sbom:     bom,
	})
	if err != nil {
		return nil, fmt.Errorf("scan sbom: %w", err)
	}
	return resp.GetVex(), nil
}
