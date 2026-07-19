// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"errors"
	"testing"

	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/sbomupload"
	"go.mondoo.com/mql/v13/sbom"
)

type fakeUploader struct {
	got  *sbomupload.BulkUploadSbomRequest
	resp *sbomupload.BulkUploadSbomResponse
	err  error
}

func (f *fakeUploader) BulkUploadSbom(_ context.Context, in *sbomupload.BulkUploadSbomRequest) (*sbomupload.BulkUploadSbomResponse, error) {
	f.got = in
	return f.resp, f.err
}

func TestUploadSBOM(t *testing.T) {
	boms := []*sbom.Sbom{{Packages: []*sbom.Package{{Name: "openssl", Version: "1.1.1"}}}}
	fu := &fakeUploader{resp: &sbomupload.BulkUploadSbomResponse{Count: 1}}

	count, err := uploadSBOM(context.Background(), fu, "//captain.api.mondoo.app/spaces/s1", boms, true)
	if err != nil {
		t.Fatalf("uploadSBOM: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if fu.got.GetSpaceMrn() != "//captain.api.mondoo.app/spaces/s1" {
		t.Errorf("space = %q", fu.got.GetSpaceMrn())
	}
	if !fu.got.GetCreateAssets() {
		t.Error("create_assets not forwarded")
	}
	if len(fu.got.GetSboms()) != 1 {
		t.Errorf("sboms not forwarded: %+v", fu.got.GetSboms())
	}
}

func TestUploadSBOMError(t *testing.T) {
	fu := &fakeUploader{err: errors.New("boom")}
	if _, err := uploadSBOM(context.Background(), fu, "s1", []*sbom.Sbom{{}}, false); err == nil {
		t.Fatal("expected error")
	}
}
