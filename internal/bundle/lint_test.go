// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
	"go.mondoo.com/cnspec/v11/internal/bundle"
)

var schema resources.ResourcesSchema

func init() {
	runtime := testutils.Local()
	schema = runtime.Schema()
}

func TestLintPass(t *testing.T) {
	file := "../../examples/example.mql.yaml"
	rootDir := "../../examples"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 0, len(results.Entries))
	assert.False(t, results.HasError())

	report, err := results.SarifReport(rootDir)
	require.NoError(t, err)

	assert.Equal(t, 1, len(report.Runs))
	assert.Equal(t, len(bundle.LinterRules), len(report.Runs[0].Tool.Driver.Rules))
	assert.Equal(t, 0, len(report.Runs[0].Results))

	data, err := results.ToSarif(rootDir)
	require.NoError(t, err)
	assert.True(t, len(data) > 0)
}

func TestLintPassComplex(t *testing.T) {
	file := "../../examples/complex.mql.yaml"
	rootDir := "../../examples"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 0, len(results.Entries))
	assert.False(t, results.HasError())

	report, err := results.SarifReport(rootDir)
	require.NoError(t, err)

	assert.Equal(t, 1, len(report.Runs))
	assert.Equal(t, len(bundle.LinterRules), len(report.Runs[0].Tool.Driver.Rules))
	assert.Equal(t, 0, len(report.Runs[0].Results))

	data, err := results.ToSarif(rootDir)
	require.NoError(t, err)
	assert.True(t, len(data) > 0)
}

func TestLintFail(t *testing.T) {
	file := "./testdata/failing_lint.mql.yaml"
	rootDir := "./testdata"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 5, len(results.Entries))
	assert.True(t, results.HasError())

	report, err := results.SarifReport(rootDir)
	require.NoError(t, err)

	assert.Equal(t, 1, len(report.Runs))
	assert.Equal(t, len(bundle.LinterRules), len(report.Runs[0].Tool.Driver.Rules))
	assert.Equal(t, 5, len(report.Runs[0].Results))

	data, err := results.ToSarif(rootDir)
	require.NoError(t, err)
	assert.True(t, len(data) > 0)
}

func TestLintFail_MixQueries(t *testing.T) {
	file := "./testdata/mixing-queries.mql.yaml"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 1, len(results.Entries))
	assert.True(t, results.HasError())

	entry := results.Entries[0]
	assert.Equal(t, "query-used-as-different-types", entry.RuleID)
	assert.Equal(t, "query sshd-sshd-01 is used as a check and data query", entry.Message)
}

func TestLintWarn_NoFilterGroupCheck(t *testing.T) {
	file := "./testdata/fail_noFiltersGroupAndCheck.mql.yaml"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 2, len(results.Entries))
	assert.True(t, results.HasWarning())

	entry := results.Entries[0]
	assert.Equal(t, "policy-missing-asset-filter", entry.RuleID)
	assert.Equal(t, "Policy mondoo-aws-security doesn't define an asset filter.", entry.Message)
}

func TestLintWarn_NoFilterGroupAndCheckVariant(t *testing.T) {
	file := "./testdata/fail_noFiltersGroupAndCheckVariant.mql.yaml"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 2, len(results.Entries))
	assert.True(t, results.HasWarning())
	entry := results.Entries[0]
	assert.Equal(t, "policy-missing-asset-filter", entry.RuleID)
	assert.Equal(t, "Policy mondoo-aws-security doesn't define an asset filter.", entry.Message)
}

func TestLintNoWarn_FiltersGroupLevel(t *testing.T) {
	file := "./testdata/pass_filtersGroupLevel.mql.yaml"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 1, len(results.Entries))
	assert.False(t, results.HasWarning())
}

func TestLintNoWarn_NoFiltersGroupLevelFiltersCheck(t *testing.T) {
	file := "./testdata/pass_noFilterGroupFiltersCheck.mql.yaml"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 1, len(results.Entries))
	assert.False(t, results.HasWarning())
}

func TestLintNoWarn_NoFilterGroupFiltersCheckVariant(t *testing.T) {
	file := "./testdata/pass_noFilterGroupFiltersCheckVariant.mql.yaml"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 1, len(results.Entries))
	assert.False(t, results.HasWarning())
}

func TestLintNoWarn_NoFilterGroupFiltersVariantOnly(t *testing.T) {
	file := "./testdata/pass_noFilterGroupVariantsOnly.mql.yaml"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 1, len(results.Entries))
	assert.False(t, results.HasWarning())
}
