// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackageImportsNoCnspecPackage is the whole reason reports/ocsf and
// reports/ocsf/convert are two packages.
//
// This package is the OCSF schema and the writers for it, and nothing about
// cnspec: that is what keeps it extractable as go.mondoo.com/ocsf, which
// docs/adr/0005-ocsf-type-generation.md, doc.go and CLAUDE.md all state as the
// reason for the split. Until now nothing enforced it, and the import that
// breaks it is the easy one to add -- one policy.Score in a helper and the
// package is no longer publishable, with every test still green.
//
// The check runs over the transitive dependencies, not the import block, because
// an indirect edge costs the same as a direct one.
func TestPackageImportsNoCnspecPackage(t *testing.T) {
	// -deps of the package under test; the test files themselves are not included,
	// which is right: a test may use whatever it likes, the shipped package may not.
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	require.NoError(t, err, "go list failed")

	var offenders []string
	for _, dep := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		dep = strings.TrimSpace(dep)
		// The package itself is go.mondoo.com/cnspec/reports/ocsf; everything else
		// under go.mondoo.com/cnspec is an edge that must not exist.
		if dep == "go.mondoo.com/cnspec/reports/ocsf" {
			continue
		}
		if strings.HasPrefix(dep, "go.mondoo.com/cnspec") {
			offenders = append(offenders, dep)
		}
	}

	assert.Empty(t, offenders,
		"reports/ocsf must import no cnspec package, so it stays extractable as go.mondoo.com/ocsf; "+
			"the cnspec mapping belongs in reports/ocsf/convert")
}
