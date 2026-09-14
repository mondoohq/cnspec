// Copyright Mondoo, Inc. 2024, 2026
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

	osComponent := cyclonedx.Component{
		BOMRef:  uuid.New().String(),
		Type:    cyclonedx.ComponentTypeOS,
		Name:    bom.Asset.Platform.Name,
		Version: bom.Asset.Platform.Version,
		CPE:     cpe,
	}
	// CycloneDX has no field for the architecture, title or family of an
	// operating system, so they go in properties. Without the architecture a
	// vulnerability scan of a re-read SBOM matches nothing: the packages carry
	// arch in their purls, but the platform the advisories are selected for does
	// not, and the scan silently returns no findings.
	osProps := []cyclonedx.Property{}
	addProp := func(name, value string) {
		if value == "" {
			return
		}
		osProps = append(osProps, cyclonedx.Property{Name: name, Value: value})
	}
	addProp(propPlatformArch, bom.Asset.Platform.Arch)
	addProp(propPlatformTitle, bom.Asset.Platform.Title)
	addProp(propPlatformFamily, strings.Join(bom.Asset.Platform.Family, ","))
	if len(osProps) > 0 {
		osComponent.Properties = &osProps
	}
	components = append(components, osComponent)

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

		// The source package this binary was built from. CycloneDX has no field
		// for it, and it is what vulnerability matching needs: Debian and RPM
		// advisories are published against the source package, so an advisory for
		// glibc has to reach the installed libc6 and libc-bin. Dropping it is why
		// a scan of a re-read SBOM returned no findings while the same asset
		// scanned directly returned them.
		pkgProps := []cyclonedx.Property{}
		if pkg.Origin != "" {
			pkgProps = append(pkgProps, cyclonedx.Property{Name: propPackageOrigin, Value: pkg.Origin})
		}
		if pkg.Architecture != "" {
			pkgProps = append(pkgProps, cyclonedx.Property{Name: propPackageArch, Value: pkg.Architecture})
		}
		if len(pkgProps) > 0 {
			bomPkg.Properties = &pkgProps
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
			// familyMap is the fallback for SBOMs from other tools; a document we
			// wrote carries the real values in properties below.
			sbom.Asset.Platform.Family = familyMap[strings.ToLower(component.Name)]
			if len(component.CPE) > 0 {
				sbom.Asset.Platform.Cpes = []string{component.CPE}
			}
			if component.Properties != nil {
				for _, prop := range *component.Properties {
					switch prop.Name {
					case propPlatformArch:
						sbom.Asset.Platform.Arch = prop.Value
					case propPlatformTitle:
						sbom.Asset.Platform.Title = prop.Value
					case propPlatformFamily:
						if prop.Value != "" {
							sbom.Asset.Platform.Family = strings.Split(prop.Value, ",")
						}
					}
				}
			}
			sbom.Packages = append(sbom.Packages, pkg)
		case cyclonedx.ComponentTypeLibrary:
			applyPackageProperties(pkg, component.Properties)
			sbom.Packages = append(sbom.Packages, pkg)
		case cyclonedx.ComponentTypeApplication:
			// Same as a library: this component becomes a package, so its
			// properties are package properties. The OS component is the one
			// exception -- its properties describe the platform.
			applyPackageProperties(pkg, component.Properties)
			sbom.Packages = append(sbom.Packages, pkg)
		}
	}

	return sbom, nil
}

// Property names for the platform fields CycloneDX does not model. They are
// namespaced so a reader can tell them from another tool's properties.
const (
	propPlatformArch   = "mondoo:platform:arch"
	propPlatformTitle  = "mondoo:platform:title"
	propPlatformFamily = "mondoo:platform:family"
	propPackageOrigin  = "mondoo:package:origin"
	propPackageArch    = "mondoo:package:arch"
)

// applyPackageProperties restores the package fields CycloneDX does not model.
func applyPackageProperties(pkg *Package, props *[]cyclonedx.Property) {
	if pkg == nil || props == nil {
		return
	}
	for _, prop := range *props {
		switch prop.Name {
		case propPackageOrigin:
			pkg.Origin = prop.Value
		case propPackageArch:
			pkg.Architecture = prop.Value
		}
	}
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
