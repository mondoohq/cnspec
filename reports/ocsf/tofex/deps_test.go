// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tofex

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackageImportsNoOtherCnspecPackage keeps tofex movable with the OCSF types.
// It may use reports/ocsf, which it maps from; any other cnspec package would tie
// it to cnspec the way reports/ocsf/convert is tied, and the mapping would no
// longer move with the schema if that is published on its own.
func TestPackageImportsNoOtherCnspecPackage(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	require.NoError(t, err, "go list failed")

	allowed := map[string]bool{
		"go.mondoo.com/cnspec/reports/ocsf":       true,
		"go.mondoo.com/cnspec/reports/ocsf/tofex": true,
	}
	var offenders []string
	for dep := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		dep = strings.TrimSpace(dep)
		if strings.HasPrefix(dep, "go.mondoo.com/cnspec") && !allowed[dep] {
			offenders = append(offenders, dep)
		}
	}

	assert.Empty(t, offenders,
		"reports/ocsf/tofex may import reports/ocsf and no other cnspec package")
}
