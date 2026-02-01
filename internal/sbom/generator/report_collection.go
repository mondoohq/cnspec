// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package generator

import (
	"encoding/json"
	"os"

	cr "go.mondoo.com/cnquery/v12/cli/reporter"
	"sigs.k8s.io/yaml"
)

// Structures to parse the data from cnspec report
type BomAsset struct {
	Name          string            `json:"name,omitempty"`
	Platform      string            `json:"platform,omitempty"`
	Version       string            `json:"version,omitempty"`
	Build         string            `json:"build,omitempty"`
	Family        []string          `json:"family,omitempty"`
	Arch          string            `json:"arch,omitempty"`
	CPEs          []string          `json:"cpes.map,omitempty"`
	IDs           []string          `json:"ids,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	PlatformTitle string            `json:"platformTitle,omitempty"`
}

type BomPackage struct {
	Name    string   `json:"name,omitempty"`
	Version string   `json:"version,omitempty"`
	Origin  string   `json:"origin,omitempty"`
	Arch    string   `json:"arch,omitempty"`
	Format  string   `json:"format,omitempty"`
	Purl    string   `json:"purl,omitempty"`
	CPEs    []string `json:"cpes.map,omitempty"`
	// used by python packages
	// deprecated: remove once python.packages uses files
	FilePath string `json:"file.path,omitempty"`
	// used by os packages
	FilePaths []string `json:"files.map,omitempty"`
}

type KernelInstalled struct {
	Name    string
	Running bool
	Version string
}

type BomFields struct {
	Asset           *BomAsset         `json:"asset,omitempty"`
	Packages        []BomPackage      `json:"packages.list,omitempty"`
	PythonPackages  []BomPackage      `json:"python.packages,omitempty"`
	NpmPackages     []BomPackage      `json:"npm.packages.list,omitempty"`
	KernelInstalled []KernelInstalled `json:"kernel.installed,omitempty"`
}

func (b *BomFields) ToJSON() ([]byte, error) {
	return json.Marshal(b)
}

func LoadReport(filename string) (*cr.Report, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var report *cr.Report
	err = yaml.Unmarshal(data, &report)
	if err != nil {
		return nil, err
	}
	return report, nil
}
