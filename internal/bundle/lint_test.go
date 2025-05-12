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

func TestResults_SarifReport(t *testing.T) {
	file := "./testdata/pass_linter.yaml"
	rootDir := "./testdata"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)
	report, err := results.SarifReport(rootDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(report.Runs))
	assert.Equal(t, len(bundle.AllLinterRules()), len(report.Runs[0].Tool.Driver.Rules))
	assert.Equal(t, 0, len(report.Runs[0].Results))
}

func TestLintPass(t *testing.T) {
	file := "./testdata/pass_linter.yaml"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)
	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 0, len(results.Entries))
	assert.False(t, results.HasError())
}

func TestLinter_Fail_PolicyUidRuleID(t *testing.T) {
	file := "./testdata/fail_PolicyUidRuleID.yaml"
	results, err := bundle.Lint(schema, file)
	require.NoError(t, err)
	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 2, len(results.Entries))
	assert.Equal(t, "policy 'Ubuntu Benchmark 1' (at line 3) does not define a UID", results.Entries[0].Message)
	assert.Equal(t, "policy-uid", results.Entries[0].RuleID)
	assert.Equal(t, "Could not compile policy bundle: failed to refresh policy : failed to refresh mrn for policy Ubuntu Benchmark 1 : cannot refresh MRN with an empty UID", results.Entries[1].Message)
	assert.Equal(t, "bundle-compile-error", results.Entries[1].RuleID)
}
