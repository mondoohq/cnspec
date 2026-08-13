// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/internal/sbom"
)

func findPkg(pkgs []*sbom.Package, name string) *sbom.Package {
	for _, p := range pkgs {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// TestPrinterDriversInSbom covers Windows print drivers reaching the SBOM.
//
// The load-bearing detail is the JSON key. BomFields is populated from the
// compiled datapoint label, and for a list resource that label is
// "<resource>.list" -- windows.printerDrivers.list, the same shape as
// packages.list and npm.packages.list. Getting it wrong fails silently: the
// field stays empty and the drivers never appear, with no error anywhere. The
// fixture carries the verbatim key so the tag and the label cannot drift apart
// unnoticed.
func TestPrinterDriversInSbom(t *testing.T) {
	report, err := LoadReport("testdata/windows-print-drivers.json")
	require.NoError(t, err)

	boms := GenerateBom(report)
	require.Len(t, boms, 1)
	bom := boms[0]

	assert.Equal(t, "windows", bom.Asset.Platform.Name)

	driver := findPkg(bom.Packages, "RICOH PCL6 UniversalDriver V4.x")
	require.NotNil(t, driver, "the print driver did not reach the SBOM")
	assert.Equal(t, "4.40.0.0", driver.Version)
	assert.Equal(t, "pkg:windows-driver/ricoh/pcl6-universaldriver-v4.x@4.40.0.0", driver.Purl)
	// The type is what routes the package to the driver ecosystem downstream.
	// It must not fall back to generic.
	assert.Equal(t, "windows-driver", driver.Type)

	// A driver whose manufacturer the spooler does not report has no purl -- a
	// driver name alone is not an identity, since the same name is shipped by
	// many vendors. It is still listed.
	builtin := findPkg(bom.Packages, "Microsoft Print To PDF")
	require.NotNil(t, builtin, "a purl-less driver must still be listed")
	assert.Empty(t, builtin.Purl)
	assert.Equal(t, "windows-driver", builtin.Type)
}
