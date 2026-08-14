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
// The fixture is captured verbatim from a real scan of a Windows 11 Pro host
// (platform windows 26200), not hand-written -- including the packed
// DriverVersion unpacked to a dotted quad, which is how 2814751477605035
// becomes 10.0.26100.8875.
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
	assert.Equal(t, "26200", bom.Asset.Platform.Version)

	driver := findPkg(bom.Packages, "Microsoft Print To PDF")
	require.NotNil(t, driver, "the print driver did not reach the SBOM")
	assert.Equal(t, "10.0.26100.4484", driver.Version)
	assert.Equal(t, "pkg:windows-driver/microsoft/microsoft-print-to-pdf@10.0.26100.4484", driver.Purl)
	// The type is what routes the package to the driver ecosystem downstream.
	// It must not fall back to generic.
	assert.Equal(t, "windows-driver", driver.Type)

	// Every driver on the captured host reported a manufacturer, so every one
	// of them carries a purl.
	for _, name := range []string{
		"Universal Print Class Driver",
		"Microsoft Virtual Print Class Driver",
		"Microsoft IPP Class Driver",
	} {
		d := findPkg(bom.Packages, name)
		require.NotNil(t, d, "%s did not reach the SBOM", name)
		assert.NotEmpty(t, d.Purl, "%s should carry a purl", name)
		assert.Equal(t, "windows-driver", d.Type)
	}

	// Synthetic row: no driver on the captured host lacked a manufacturer, but
	// the resource returns an empty purl when one is missing -- a driver name
	// alone is not an identity, since the same name is shipped by many vendors.
	// Such a driver must still be listed.
	vendorless := findPkg(bom.Packages, "Vendorless Test Driver")
	require.NotNil(t, vendorless, "a purl-less driver must still be listed")
	assert.Empty(t, vendorless.Purl)
	assert.Equal(t, "windows-driver", vendorless.Type)
}
