// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"encoding/json"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/sbom"
)

// mqlSbomWrapper is the top-level wrapper: {"sbom": {"packages": [...]}}
type mqlSbomWrapper struct {
	Sbom mqlSbomData `json:"sbom"`
}

type mqlSbomData struct {
	Packages []mqlSbomPackage `json:"packages"`
}

// mqlSbomPackage mirrors the SPDX-like JSON output from MQL queries such as
// github.repository.sbom.
type mqlSbomPackage struct {
	Name             string           `json:"name"`
	VersionInfo      string           `json:"versionInfo"`
	CopyrightText    string           `json:"copyrightText"`
	DownloadLocation string           `json:"downloadLocation"`
	ExternalRefs     []mqlExternalRef `json:"externalRefs"`
	FilesAnalyzed    bool             `json:"filesAnalyzed"`
	LicenseDeclared  any              `json:"licenseDeclared"`
	SpdxId           string           `json:"spdxId"`
	LicenseConcluded string           `json:"licenseConcluded"`
	Supplier         string           `json:"supplier"`
}

type mqlExternalRef struct {
	ReferenceCategory string `json:"referenceCategory"`
	ReferenceLocator  string `json:"referenceLocator"`
	ReferenceType     string `json:"referenceType"`
}

// ExtractSbomPackages examines raw results for SBOM package data and converts
// them to sbom.Package protos. It follows the same JSON intermediary pattern as
// ExtractFindings: serialize RawData to JSON, unmarshal into Go structs, then
// convert to proto.
func ExtractSbomPackages(results []*llx.RawResult, codeBundleMap map[string]*llx.CodeBundle) []*sbom.Package {
	var packages []*sbom.Package

	for _, rr := range results {
		if rr.Data == nil || rr.Data.Value == nil || rr.Data.Error != nil {
			continue
		}

		cb, ok := codeBundleMap[rr.CodeID]
		if !ok {
			continue
		}

		jsonBytes := rr.Data.JSON(rr.CodeID, cb)
		if len(jsonBytes) == 0 {
			continue
		}

		extracted := parseSbomPackagesFromJSON(jsonBytes)
		packages = append(packages, extracted...)
	}

	return packages
}

// parseSbomPackagesFromJSON tries to unmarshal JSON bytes as SBOM data.
// It handles the wrapper format {"sbom": {"packages": [...]}}, a bare array
// of packages, and a single package object.
func parseSbomPackagesFromJSON(data []byte) []*sbom.Package {
	// Try as wrapper: {"sbom": {"packages": [...]}}
	var wrapper mqlSbomWrapper
	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Sbom.Packages) > 0 {
		return convertSbomPackages(wrapper.Sbom.Packages)
	}

	// Try as bare array of packages
	var packages []mqlSbomPackage
	if err := json.Unmarshal(data, &packages); err == nil {
		return convertSbomPackages(packages)
	}

	// Try as a single package
	var single mqlSbomPackage
	if err := json.Unmarshal(data, &single); err == nil && single.Name != "" {
		return convertSbomPackages([]mqlSbomPackage{single})
	}

	return nil
}

func convertSbomPackages(packages []mqlSbomPackage) []*sbom.Package {
	var result []*sbom.Package
	for i := range packages {
		if packages[i].Name == "" {
			continue
		}
		result = append(result, mqlPackageToProto(&packages[i]))
	}
	return result
}

func mqlPackageToProto(p *mqlSbomPackage) *sbom.Package {
	return &sbom.Package{
		Name:    p.Name,
		Version: p.VersionInfo,
		Purl:    extractPurl(p.ExternalRefs),
		Vendor:  p.Supplier,
	}
}

// extractPurl finds the PURL from the externalRefs array.
func extractPurl(refs []mqlExternalRef) string {
	for _, ref := range refs {
		if ref.ReferenceType == "purl" {
			return ref.ReferenceLocator
		}
	}
	return ""
}
