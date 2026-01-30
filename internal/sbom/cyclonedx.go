// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"errors"
	"io"
	"strings"
	"time"

	cyclonedx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
	"github.com/package-url/packageurl-go"
)

func NewCycloneDX(format string) *CycloneDX {
	switch format {
	case FormatCycloneDxXML:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatXML,
		}
	default:
		return &CycloneDX{
			Format: cyclonedx.BOMFileFormatJSON,
		}
	}
}

var _ Decoder = &CycloneDX{}

type CycloneDX struct {
	opts   renderOpts
	Format cyclonedx.BOMFileFormat
}

func (ccx *CycloneDX) convertToCycloneDx(bom *Sbom) (*cyclonedx.BOM, error) {
	sbom := cyclonedx.NewBOM()
	sbom.SerialNumber = uuid.New().URN()
	sbom.Metadata = &cyclonedx.Metadata{
		Timestamp: time.Now().Format(time.RFC3339),
		Tools: &cyclonedx.ToolsChoice{
			Components: &[]cyclonedx.Component{
				{
					Type:    cyclonedx.ComponentTypeApplication,
					Author:  bom.Generator.Vendor,
					Name:    bom.Generator.Name,
					Version: bom.Generator.Version,
				},
			},
		},
		Component: &cyclonedx.Component{
			BOMRef: uuid.New().String(),
			// TODO: understand the device type
			// Type: cyclonedx.ComponentTypeContainer,
			Type: cyclonedx.ComponentTypeDevice,
			Name: bom.Asset.Name,
		},
	}

	components := []cyclonedx.Component{}

	// add os as component
	cpe := ""
	if len(bom.Asset.Platform.Cpes) > 0 {
		cpe = bom.Asset.Platform.Cpes[0]
	}

	components = append(components, cyclonedx.Component{
		BOMRef:  uuid.New().String(),
		Type:    cyclonedx.ComponentTypeOS,
		Name:    bom.Asset.Platform.Name,
		Version: bom.Asset.Platform.Version,
		CPE:     cpe,
	})

	// add os packages as components
	for i := range bom.Packages {
		pkg := bom.Packages[i]
		cpe := ""
		if len(pkg.Cpes) > 0 && ccx.opts.IncludeCPE {
			cpe = pkg.Cpes[0]
		}

		fileLocations := []cyclonedx.EvidenceOccurrence{}

		// pkg.Location is deprecated, use pkg.Evidences instead
		if pkg.Location != "" {
			fileLocations = append(fileLocations, cyclonedx.EvidenceOccurrence{
				Location: pkg.Location,
			})
		}

		if pkg.EvidenceList != nil && ccx.opts.IncludeEvidence {
			for i := range pkg.EvidenceList {
				e := pkg.EvidenceList[i]
				if e.Type == EvidenceType_EVIDENCE_TYPE_FILE {
					fileLocations = append(fileLocations, cyclonedx.EvidenceOccurrence{
						Location: e.Value,
					})
				}
			}
		}

		var evidence *cyclonedx.Evidence
		if len(fileLocations) > 0 {
			evidence = &cyclonedx.Evidence{
				Occurrences: &fileLocations,
			}
		}

		bomPkg := cyclonedx.Component{
			BOMRef:     uuid.New().String(), // temporary, we need to store the relationships next
			Type:       cyclonedx.ComponentTypeLibrary,
			Name:       pkg.Name,
			Version:    pkg.Version,
			PackageURL: pkg.Purl,
			CPE:        cpe,
			Evidence:   evidence,
		}

		components = append(components, bomPkg)
	}

	sbom.Components = &components

	return sbom, nil
}

func (s *CycloneDX) ApplyOptions(opts ...renderOption) {
	for _, opt := range opts {
		opt(&s.opts)
	}
}

func (ccx *CycloneDX) Convert(bom *Sbom) (any, error) {
	return ccx.convertToCycloneDx(bom)
}

func (ccx *CycloneDX) Render(w io.Writer, bom *Sbom) error {
	sbom, err := ccx.convertToCycloneDx(bom)
	if err != nil {
		return err
	}
	enc := cyclonedx.NewBOMEncoder(w, ccx.Format)
	enc.SetPretty(true)
	return enc.Encode(sbom)
}

func (ccx *CycloneDX) Parse(r io.ReadSeeker) (*Sbom, error) {
	doc := &cyclonedx.BOM{
		Components: &[]cyclonedx.Component{},
	}
	err := cyclonedx.NewBOMDecoder(r, ccx.Format).Decode(doc)
	if err != nil {
		return nil, err
	}

	return ccx.convertCycloneDxToSbom(doc)
}

func (ccx *CycloneDX) convertCycloneDxToSbom(bom *cyclonedx.BOM) (*Sbom, error) {
	if bom == nil {
		return nil, nil
	}

	// check if the BOM is empty
	if bom.Metadata == nil || bom.Metadata.Component == nil || bom.Components == nil {
		return nil, errors.New("not a valid cyclone dx BOM")
	}

	rootComponent := bom.Metadata.Component
	title := rootComponent.Description
	version := rootComponent.Version
	if title == "" {
		title = "CycloneDX"
	}
	if version == "" {
		version = bom.SpecVersion.String()
	}
	sbom := &Sbom{
		Asset: &Asset{
			Name: rootComponent.Name,
			Platform: &Platform{
				Name:    "cyclonedx",
				Version: version,
				Title:   title,
			},
		},
		Packages: make([]*Package, 0),
	}

	switch rootComponent.Type {
	case cyclonedx.ComponentTypeOS:
		hostnameId := "//platformid.api.mondoo.app/hostname/" + rootComponent.Name
		sbom.Asset.PlatformIds = append(sbom.Asset.PlatformIds, hostnameId)
	case cyclonedx.ComponentTypeContainer:
		// we need to figure out where to get the container ID from properly. For now, we use the BOMRef
		bomRefId := "//platformid.api.mondoo.app/runtime/docker/images/" + rootComponent.BOMRef
		sbom.Asset.PlatformIds = append(sbom.Asset.PlatformIds, bomRefId)
	}

	if bom.Metadata.Tools != nil {
		if bom.Metadata.Tools.Components != nil {
			// last one wins :-) - we only support one tool
			for _, component := range *bom.Metadata.Tools.Components {
				sbom.Generator = &Generator{
					Name:    component.Name,
					Version: component.Version,
					Vendor:  component.Author,
				}
			}
		}

		// if we have no generator info, fallback to trying tools. these are deprecated
		// but might still be present
		if sbom.Generator == nil && bom.Metadata.Tools.Tools != nil {
			for _, tool := range *bom.Metadata.Tools.Tools {
				sbom.Generator = &Generator{
					Name:    tool.Name,
					Version: tool.Version,
					Vendor:  tool.Vendor,
				}
			}
		}
	}

	for _, component := range *bom.Components {
		pkg := &Package{
			Name:        component.Name,
			Version:     component.Version,
			Purl:        component.PackageURL,
			Description: component.Description,
		}

		// parse purl to gather package type
		if component.PackageURL != "" {
			url, err := packageurl.FromString(component.PackageURL)
			if err == nil {
				pkg.Type = url.Type
			}
		}

		if component.CPE != "" {
			pkg.Cpes = []string{component.CPE}
		}

		if component.Evidence != nil && component.Evidence.Occurrences != nil && ccx.opts.IncludeEvidence {
			pkg.EvidenceList = make([]*Evidence, 0)
			for _, e := range *component.Evidence.Occurrences {
				pkg.EvidenceList = append(pkg.EvidenceList, &Evidence{
					Type:  EvidenceType_EVIDENCE_TYPE_FILE,
					Value: e.Location,
				})
			}
		}

		switch component.Type {
		case cyclonedx.ComponentTypeOS:
			sbom.Asset.Platform.Name = component.Name
			sbom.Asset.Platform.Version = component.Version
			sbom.Asset.Platform.Title = component.Description
			sbom.Asset.Platform.Family = familyMap[strings.ToLower(component.Name)]
			if len(component.CPE) > 0 {
				sbom.Asset.Platform.Cpes = []string{component.CPE}
			}
			sbom.Packages = append(sbom.Packages, pkg)
		case cyclonedx.ComponentTypeLibrary:
			sbom.Packages = append(sbom.Packages, pkg)
		case cyclonedx.ComponentTypeApplication:
			sbom.Packages = append(sbom.Packages, pkg)
		}
	}

	return sbom, nil
}

var familyMap = map[string][]string{
	"windows": {"windows", "os"},
	"macos":   {"darwin", "bsd", "unix", "os"},
	"debian":  {"linux", "unix", "os"},
	"ubuntu":  {"linux", "unix", "os"},
	"centos":  {"linux", "unix", "os"},
	"alpine":  {"linux", "unix", "os"},
	"fedora":  {"linux", "unix", "os"},
	"rhel":    {"linux", "unix", "os"},
}
