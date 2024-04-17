// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package advisories

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/mvd"
)

func TestFindVulnerablePackageWithoutNamespace(t *testing.T) {
	advisory := &mvd.Advisory{
		Fixed: []*mvd.Package{
			{Name: "pkg1", Version: "1.0.0"},
			{Name: "pkg2", Version: "2.0.0"},
			{Name: "pkg2", Version: "3.0.0"},
			{Name: "pkg3", Version: "3.0.0"},
		},
	}

	installedPkg := &mvd.Package{Name: "pkg2", Version: "2.0.0"}

	match := findVulnerablePackageWithoutNamespace(advisory, installedPkg)

	require.NotNil(t, match)
	require.Equal(t, "pkg2", match.Name)
	require.Equal(t, "3.0.0", match.Version)
}
