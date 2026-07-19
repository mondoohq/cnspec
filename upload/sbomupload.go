// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"fmt"

	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/sbomupload"
	"go.mondoo.com/mql/v13/sbom"
	ranger "go.mondoo.com/ranger-rpc"
)

// sbomUploader is the slice of the Sbom client that UploadSBOM needs; a fake can
// stand in for tests.
type sbomUploader interface {
	BulkUploadSbom(ctx context.Context, in *sbomupload.BulkUploadSbomRequest) (*sbomupload.BulkUploadSbomResponse, error)
}

// UploadSBOM uploads SBOMs to Mondoo Platform via Sbom.BulkUploadSbom. The
// platform stores each SBOM and enriches it into vulnerabilities automatically —
// the client computes no VEX. createAssets lets the platform create assets that
// are not yet known (from the SBOM's own asset identification). Returns the
// number of SBOMs stored.
func UploadSBOM(ctx context.Context, opts Opts, boms []*sbom.Sbom, createAssets bool) (int, error) {
	creds, spaceMrn, err := LoadCredentials(opts)
	if err != nil {
		return 0, err
	}
	plugin, err := upstream.NewServiceAccountRangerPlugin(creds)
	if err != nil {
		return 0, fmt.Errorf("create auth plugin: %w", err)
	}
	client, err := sbomupload.NewSbomClient(creds.ApiEndpoint, ranger.DefaultHttpClient(), plugin)
	if err != nil {
		return 0, fmt.Errorf("create sbom upload client: %w", err)
	}
	return uploadSBOM(ctx, client, spaceMrn, boms, createAssets)
}

// uploadSBOM is the testable core: it issues the upload against an injected client.
func uploadSBOM(ctx context.Context, client sbomUploader, spaceMrn string, boms []*sbom.Sbom, createAssets bool) (int, error) {
	resp, err := client.BulkUploadSbom(ctx, &sbomupload.BulkUploadSbomRequest{
		SpaceMrn:     spaceMrn,
		Sboms:        boms,
		CreateAssets: createAssets,
	})
	if err != nil {
		return 0, fmt.Errorf("upload sbom: %w", err)
	}
	// GetCount is nil-safe (generated proto getter), so a nil resp on a nil err
	// yields 0 rather than panicking.
	return int(resp.GetCount()), nil
}
