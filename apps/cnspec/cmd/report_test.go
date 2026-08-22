// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/internal/reportfixture"
)

// `report view` has to be wired onto the hidden report command next to `cmp`,
// and it takes exactly one file.
func TestViewSubcommandIsRegistered(t *testing.T) {
	var found bool
	for _, c := range reportCmd.Commands() {
		if c.Name() == "view" {
			found = true
			require.Error(t, c.Args(c, []string{}), "view needs a file")
			require.Error(t, c.Args(c, []string{"a", "b"}), "view takes one file")
			require.NoError(t, c.Args(c, []string{"a"}))
		}
	}
	require.True(t, found, "`cnspec report view` is not registered")
}

// The three ways this goes wrong all have to produce something a user can act
// on, rather than a stack trace or -- worse -- a viewer that opens on an empty
// screen because a reduced report decoded into a collection with no reports.
func TestLoadReportForViewErrors(t *testing.T) {
	dir := t.TempDir()

	_, err := loadReportForView(filepath.Join(dir, "nope.json"))
	require.ErrorContains(t, err, "no such report")

	_, err = loadReportForView(dir)
	require.ErrorContains(t, err, "is a directory")

	garbage := filepath.Join(dir, "garbage.json")
	require.NoError(t, os.WriteFile(garbage, []byte("not json at all"), 0o600))
	_, err = loadReportForView(garbage)
	require.ErrorContains(t, err, "failed to parse report collection")

	// A json-v1/json-v2 report decodes "successfully" into a collection with no
	// reports, which is indistinguishable from a scan where every asset failed.
	// The loader has to name both formats: the one given and the one needed.
	reduced := filepath.Join(dir, "reduced.json")
	require.NoError(t, os.WriteFile(reduced,
		[]byte(`{"assets":{},"data":{},"scores":{},"errors":{}}`), 0o600))
	_, err = loadReportForView(reduced)
	require.ErrorContains(t, err, "reduced report")
	require.ErrorContains(t, err, "json-full")
}

// The happy path: a collection on disk loads and builds a model with its assets
// intact. loadReportForView takes a path, so the shared recorded scan -- which
// is embedded, not a file -- is written out first.
func TestLoadReportForView(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report-ubuntu.json")
	require.NoError(t, os.WriteFile(path, reportfixture.UbuntuScanJSON(), 0o600))

	collection, err := loadReportForView(path)
	require.NoError(t, err)

	report := reportmodel.New(collection)
	require.Len(t, report.Assets, 1)
	require.Equal(t, "ubuntu:24.04", report.Assets[0].Name)
	require.True(t, report.Assets[0].Scanned())
}

// A multi-asset scan where every asset failed still loads, and every one of them
// shows up as an errored asset rather than as nothing at all.
func TestLoadReportForViewAllErrored(t *testing.T) {
	collection, err := loadReportForView("../../../cli/reporter/testdata/report-k8s.json")
	require.NoError(t, err)

	report := reportmodel.New(collection)
	require.Len(t, report.Assets, 15)
	require.Equal(t, 15, report.AssetCounts.Errored)
	for _, a := range report.Assets {
		require.False(t, a.Scanned())
		require.NotEmpty(t, a.ScanError)
	}
}
