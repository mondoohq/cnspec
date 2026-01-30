// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"io"

	"github.com/package-url/packageurl-go"
	"github.com/protobom/protobom/pkg/reader"
	protobom_sbom "github.com/protobom/protobom/pkg/sbom"
)

func NewProtobom() *Protobom {
	return &Protobom{}
}

type Protobom struct {
	opts renderOpts
}

func (s *Protobom) Parse(f io.ReadSeeker) (*Sbom, error) {
	reader := reader.New()

	document, err := reader.ParseStream(f)
	if err != nil {
		return nil, err
	}

	return s.convertToSbom(document), nil
}

func (s *Protobom) convertToSbom(doc *protobom_sbom.Document) *Sbom {
	bom := &Sbom{
		Asset: &Asset{
			Name: doc.Metadata.Name,
		},
		Packages: make([]*Package, 0),
	}

	if doc.Metadata != nil && len(doc.Metadata.Tools) > 0 {
		bom.Generator = &Generator{
			Name:    doc.Metadata.Tools[0].Name,
			Version: doc.Metadata.Tools[0].Version,
			Vendor:  doc.Metadata.Tools[0].Vendor,
		}
	}

	if doc.GetNodeList() == nil || len(doc.GetNodeList().GetNodes()) == 0 {
		return bom // no nodes, return empty SBOM
	}

	for _, node := range doc.GetNodeList().GetNodes() {
		pkg := &Package{
			Name:    node.Name,
			Version: node.Version,
		}

		for key, identifier := range node.GetIdentifiers() {
			if key == int32(protobom_sbom.SoftwareIdentifierType_PURL) {
				pkg.Purl = identifier
				if purl, err := packageurl.FromString(identifier); err == nil {
					pkg.Type = purl.Type
				}
			} else if key == int32(protobom_sbom.SoftwareIdentifierType_CPE23) {
				pkg.Cpes = append(pkg.Cpes, identifier)
			} else if key == int32(protobom_sbom.SoftwareIdentifierType_CPE22) {
				pkg.Cpes = append(pkg.Cpes, identifier)
			}
		}

		if !s.opts.IncludeCPE {
			// if CPEs are not included, clear them
			pkg.Cpes = nil
		}

		purposes := node.GetPrimaryPurpose()
		if len(purposes) > 0 {
			switch purposes[0] {
			case protobom_sbom.Purpose_OPERATING_SYSTEM:
				bom.Asset.Platform = &Platform{
					Name:    pkg.Name,
					Version: pkg.Version,
					Title:   pkg.Description,
				}
				bom.Asset.Platform.Family = familyMap[pkg.Name]
				bom.Asset.Platform.Cpes = pkg.Cpes
			case protobom_sbom.Purpose_APPLICATION:
				bom.Packages = append(bom.Packages, pkg)
			}
		}
	}

	return bom
}
