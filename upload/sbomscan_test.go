// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"errors"
	"testing"

	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/sbomscan"
	"go.mondoo.com/mql/v13/sbom"
)

type fakeScanner struct {
	gotMrn  string
	gotSbom *sbom.Sbom
	resp    *sbomscan.ScanUploadedSbomResponse
	err     error
}

func (f *fakeScanner) ScanUploadedSbom(_ context.Context, in *sbomscan.ScanUploadedSbomRequest) (*sbomscan.ScanUploadedSbomResponse, error) {
	f.gotMrn = in.GetAssetMrn()
	f.gotSbom = in.GetSbom()
	return f.resp, f.err
}

func TestScanSBOM(t *testing.T) {
	bom := &sbom.Sbom{Packages: []*sbom.Package{{Name: "openssl", Version: "1.1.1"}}}
	fs := &fakeScanner{resp: &sbomscan.ScanUploadedSbomResponse{
		Vex: []*fex.VulnerabilityExchange{{Id: "CVE-2024-0001"}},
	}}

	vex, err := scanSBOM(context.Background(), fs, "//captain.api.mondoo.app/spaces/s1", bom)
	if err != nil {
		t.Fatalf("scanSBOM: %v", err)
	}
	if fs.gotMrn != "//captain.api.mondoo.app/spaces/s1" {
		t.Errorf("scoped mrn = %q", fs.gotMrn)
	}
	if len(fs.gotSbom.Packages) != 1 {
		t.Errorf("sbom not forwarded: %+v", fs.gotSbom)
	}
	if len(vex) != 1 || vex[0].GetId() != "CVE-2024-0001" {
		t.Errorf("vex = %+v", vex)
	}
}

func TestScanSBOMError(t *testing.T) {
	fs := &fakeScanner{err: errors.New("boom")}
	if _, err := scanSBOM(context.Background(), fs, "s1", &sbom.Sbom{}); err == nil {
		t.Fatal("expected error")
	}
}
