// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"fmt"
	"io"
)

var _ Decoder = &CnspecBOM{}

type CnspecBOM struct {
	opts renderOpts
}

func (s *CnspecBOM) ApplyOptions(opts ...renderOption) {
	for _, opt := range opts {
		opt(&s.opts)
	}
}

func (ccx *CnspecBOM) Convert(bom *Sbom) (any, error) {
	// nothing to do, the cnspec BOM is already in the correct format
	return bom, nil
}

func (ccx *CnspecBOM) Render(output io.Writer, bom *Sbom) error {
	if !ccx.opts.IncludeEvidence {
		// if we do not include evidence, we remove all evidence from the BOM
		for _, pkg := range bom.Packages {
			pkg.EvidenceList = nil
		}
	}

	if !ccx.opts.IncludeCPE {
		// if we do not include CPE, we remove all CPE from the BOM
		for _, pkg := range bom.Packages {
			pkg.Cpes = nil
		}

		if bom.Asset != nil && bom.Asset.Platform != nil {
			bom.Asset.Platform.Cpes = nil
		}
	}

	enc := json.NewEncoder(output)
	enc.SetIndent("", "  ")
	return enc.Encode(bom)
}

func (ccx *CnspecBOM) Parse(r io.ReadSeeker) (*Sbom, error) {
	var s Sbom
	err := json.NewDecoder(r).Decode(&s)
	if err != nil {
		return nil, err
	}

	// Test if the SBOM has a valid structure
	if s.Asset == nil {
		return nil, fmt.Errorf("unable to parse cnspec SBOM: missing asset information")
	}

	return &s, nil
}
