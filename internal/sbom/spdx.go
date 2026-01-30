// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/package-url/packageurl-go"
	"github.com/spdx/tools-golang/convert"
	"github.com/spdx/tools-golang/spdx"
	"github.com/spdx/tools-golang/spdx/v2/v2_1"
	"github.com/spdx/tools-golang/spdx/v2/v2_2"
	"github.com/spdx/tools-golang/spdx/v2/v2_3"
	"github.com/spdx/tools-golang/tagvalue"
)

func NewSPDX(format string) *Spdx {
	return &Spdx{
		Version: "2.3",
		Format:  format,
	}
}

var _ Decoder = &Spdx{}

type Spdx struct {
	opts    renderOpts
	Version string
	Format  string
}

func (s *Spdx) ApplyOptions(opts ...renderOption) {
	for _, opt := range opts {
		opt(&s.opts)
	}
}

func (s *Spdx) convertToSpdx(bom *Sbom) *spdx.Document {
	doc := &spdx.Document{
		SPDXVersion:                spdx.Version,
		SPDXIdentifier:             "DOCUMENT",
		ExternalDocumentReferences: nil,
		DocumentComment:            "",

		CreationInfo: &spdx.CreationInfo{
			Creators: []spdx.Creator{
				{
					Creator:     bom.Generator.Vendor,
					CreatorType: "Organization",
				},
				{
					Creator:     bom.Generator.Name + "-" + bom.Generator.Version,
					CreatorType: "Tool",
				},
			},
			Created: time.Now().UTC().Format(time.RFC3339),
		},
	}

	for i := range bom.Packages {
		pkg := bom.Packages[i]

		refs := []*spdx.PackageExternalReference{}

		if len(pkg.Cpes) > 0 {
			for _, cpe := range pkg.Cpes {
				refs = append(refs, &spdx.PackageExternalReference{
					RefType:  spdx.SecurityCPE23Type,
					Category: spdx.CategorySecurity,
					Locator:  cpe,
				})
			}
		}

		if pkg.Purl != "" {
			refs = append(refs, &spdx.PackageExternalReference{
				RefType:  spdx.PackageManagerPURL,
				Category: spdx.CategoryPackageManager,
				Locator:  pkg.Purl,
			})
		}

		doc.Packages = append(doc.Packages, &spdx.Package{
			PackageSPDXIdentifier:     NewSPDXPackageID(pkg),
			PackageName:               pkg.Name,
			PackageVersion:            pkg.Version,
			PackageLicenseDeclared:    pkg.Version,
			PackageDescription:        pkg.Description,
			PackageExternalReferences: refs,
			PackageFileName:           pkg.Location,
		})
	}

	return doc
}

var expr = regexp.MustCompile("[^a-zA-Z0-9.-]")

// NewSPDXPackageID creates a new SPDX ID for a package
// see https://spdx.github.io/spdx-spec/v2.3/relationships-between-SPDX-elements/
func NewSPDXPackageID(pkg *Package) spdx.ElementID {
	hash, _ := pkg.Hash()

	id := fmt.Sprintf("Package-%s-%s-%s", pkg.Type, pkg.Name, hash)
	expr.ReplaceAllString(id, "-")
	return spdx.ElementID(id)
}

func (s *Spdx) Convert(bom *Sbom) (any, error) {
	spdxLatestBom := s.convertToSpdx(bom)

	var spdxBom any
	var err error
	switch s.Version {
	case "2.1":
		doc := v2_1.Document{}
		err = convert.Document(spdxLatestBom, &doc)
		spdxBom = doc
	case "2.2":
		doc := v2_2.Document{}
		err = convert.Document(spdxLatestBom, &doc)
		spdxBom = doc
	case "2.3":
		fallthrough
	case "":
		doc := v2_3.Document{}
		err = convert.Document(spdxLatestBom, &doc)
		spdxBom = doc
	default:
		return nil, fmt.Errorf("unsupported SPDX version %q", s.Version)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to convertToCycloneDx SBOM to SPDX document: %w", err)
	}
	return spdxBom, nil
}

func (s *Spdx) Render(w io.Writer, bom *Sbom) error {
	spdxBom, err := s.Convert(bom)
	if err != nil {
		return err
	}

	switch s.Format {
	case FormatSpdxTagValue:
		err = tagvalue.Write(spdxBom, w)
		if err != nil {
			return fmt.Errorf("unable to write SPDX tag-value document: %w", err)
		}
		return nil
	case FormatSpdxJSON:
		fallthrough
	default:
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(spdxBom)
	}
}

func (s *Spdx) Parse(r io.ReadSeeker) (*Sbom, error) {
	// try to parse all supported SPDX format
	switch s.Format {
	case FormatSpdxTagValue:
		doc, err := tagvalue.Read(r)
		if err == nil && doc.SPDXVersion != "" {
			return s.convertToSbom(doc), nil
		}
	case FormatSpdxJSON:
		var doc spdx.Document
		err := json.NewDecoder(r).Decode(&doc)
		if err == nil && doc.SPDXVersion != "" {
			return s.convertToSbom(&doc), nil
		}
	}

	return nil, errors.New("unable to parse SPDX document")
}

func (s *Spdx) convertToSbom(doc *spdx.Document) *Sbom {
	bom := &Sbom{
		Generator: &Generator{
			Name: doc.CreationInfo.Creators[0].Creator,
		},
		Asset: &Asset{
			Name: doc.DocumentName,
			Platform: &Platform{
				Name:    "spdx",
				Version: doc.SPDXVersion,
				Title:   "SPDX",
			},
		},
		Packages: []*Package{},
	}

	name := ""
	var pf *Platform

	for i := range doc.Packages {
		pkg := doc.Packages[i]

		bomPkg := &Package{
			Name:        pkg.PackageName,
			Version:     pkg.PackageVersion,
			Description: pkg.PackageDescription,
			Location:    pkg.PackageFileName,
			Type:        "", // extract package type from purl, see below
			Purl:        "", // extract package type from purl, see below
			Cpes:        []string{},
		}

		for _, ref := range pkg.PackageExternalReferences {
			if ref.RefType == spdx.PackageManagerPURL {
				bomPkg.Purl = ref.Locator
				pkgUrl, err := packageurl.FromString(ref.Locator)
				if err == nil {
					bomPkg.Type = pkgUrl.Type

					// extract distro information
					m := pkgUrl.Qualifiers.Map()
					distroVal, ok := m["distro"]
					if ok {
						if pf == nil {
							pf = &Platform{}
						}
						name = distroVal
						pf.Title = distroVal
						vals := strings.Split(distroVal, "-")
						if len(vals) > 0 {
							pf.Name = vals[0]
							pf.Version = vals[1]
						}
						pf.Family = familyMap[pf.Name]
					}
					arch, ok := m["arch"]
					if ok {
						pf.Arch = arch
					}
				}
			}
			if ref.RefType == spdx.SecurityCPE23Type {
				bomPkg.Cpes = append(bomPkg.Cpes, ref.Locator)
			}
		}

		if pkg.PackageFileName != "" && s.opts.IncludeEvidence {
			bomPkg.EvidenceList = append(bomPkg.EvidenceList, &Evidence{
				Type:  EvidenceType_EVIDENCE_TYPE_FILE,
				Value: pkg.PackageFileName,
			})
		}

		bom.Packages = append(bom.Packages, bomPkg)
	}

	if name != "" {
		bom.Asset.Name = name
	}
	if pf != nil {
		bom.Asset.Platform = pf
	}

	return bom
}
