// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/testutils"
)

var schema resources.ResourcesSchema

func init() {
	runtime := testutils.Local()
	schema = runtime.Schema()
}

func TestResults_SarifReport(t *testing.T) {
	file := "./testdata/pass-rules.mql.yaml"
	rootDir := "./testdata"
	results, err := Lint(schema, file)
	require.NoError(t, err)
	report, err := results.SarifReport(rootDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(report.Runs))
	assert.Equal(t, len(sarifLinterRules()), len(report.Runs[0].Tool.Driver.Rules))
	assert.Equal(t, 0, len(report.Runs[0].Results))
}

func TestSanitizeLineColumn(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "valid line number",
			input:    42,
			expected: 42,
		},
		{
			name:     "zero value",
			input:    0,
			expected: 1,
		},
		{
			name:     "negative value",
			input:    -1,
			expected: 1,
		},
		{
			name:     "max safe value",
			input:    2147483647,
			expected: 2147483647,
		},
		{
			name:     "overflow value (simulated with max int)",
			input:    2147483648,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLineColumn(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResults_SarifReportWithInvalidLocations(t *testing.T) {
	// Create a Results object with invalid line/column values
	results := &Results{
		BundleLocations: []string{"test.yaml"},
		Entries: []*Entry{
			{
				RuleID:  "test-rule",
				Level:   LevelError,
				Message: "Test error with invalid location",
				Location: []Location{
					{
						File:   "test.yaml",
						Line:   -1, // Invalid line
						Column: 0,  // Invalid column
					},
				},
			},
			{
				RuleID:  "test-rule-2",
				Level:   LevelWarning,
				Message: "Test warning with overflow location",
				Location: []Location{
					{
						File:   "test.yaml",
						Line:   2147483648, // Overflow value
						Column: 2147483648, // Overflow value
					},
				},
			},
		},
	}

	report, err := results.SarifReport(".")
	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify that the report was generated successfully
	assert.Equal(t, 1, len(report.Runs))
	assert.Equal(t, 2, len(report.Runs[0].Results))

	// Verify that locations were sanitized to valid values
	for _, result := range report.Runs[0].Results {
		for _, location := range result.Locations {
			if location.PhysicalLocation != nil && location.PhysicalLocation.Region != nil {
				// StartLine and StartColumn should be sanitized to 1
				assert.NotNil(t, location.PhysicalLocation.Region.StartLine)
				assert.Greater(t, *location.PhysicalLocation.Region.StartLine, 0)
				assert.LessOrEqual(t, *location.PhysicalLocation.Region.StartLine, 2147483647)

				if location.PhysicalLocation.Region.StartColumn != nil {
					assert.Greater(t, *location.PhysicalLocation.Region.StartColumn, 0)
					assert.LessOrEqual(t, *location.PhysicalLocation.Region.StartColumn, 2147483647)
				}
			}
		}
	}
}

func TestLinter_Pass(t *testing.T) {
	file := "./testdata/pass-rules.mql.yaml"
	results, err := Lint(schema, file)
	require.NoError(t, err)
	assert.Equal(t, 1, len(results.BundleLocations))
	assert.Equal(t, 0, len(results.Entries))
	assert.False(t, results.HasError())
}

func TestLinter_Fail(t *testing.T) {
	findEntry := func(entries []*Entry, id string) *Entry {
		for _, entry := range entries {
			if entry.RuleID == id {
				return entry
			}
		}
		return nil
	}

	t.Run("fail-global-props", func(t *testing.T) {
		file := "./testdata/fail-bundle-global-props.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "bundle-global-props-deprecated", results.Entries[0].RuleID)
		assert.Equal(t, "error", results.Entries[0].Level)
		assert.Equal(t, "Defining global properties in a policy bundle is deprecated. Define properties within individual policies and queries instead.", results.Entries[0].Message)
	})

	t.Run("fail-policy-uid", func(t *testing.T) {
		file := "./testdata/fail-policy-uid.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 2, len(results.Entries))

		result := findEntry(results.Entries, "policy-uid")
		assert.Equal(t, "policy 'Ubuntu Benchmark 1' (at line 2) does not define a UID", result.Message)
		assert.Equal(t, "policy-uid", result.RuleID)
		assert.Equal(t, "error", result.Level)

		result = findEntry(results.Entries, "bundle-compile-error")
		assert.Contains(t, result.Message, "cannot refresh MRN with an empty UID")
		assert.Equal(t, "bundle-compile-error", result.RuleID)
	})

	t.Run("fail-policy-name", func(t *testing.T) {
		file := "./testdata/fail-policy-name.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "policy 'ubuntu-bench-1' does not define a name", results.Entries[0].Message)
		assert.Equal(t, "policy-name", results.Entries[0].RuleID)
		assert.Equal(t, "error", results.Entries[0].Level)
	})

	t.Run("fail-policy-missing-asset-filter-variants", func(t *testing.T) {
		file := "./testdata/fail-policy-missing-asset-filter-variants.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "policy 'mondoo-aws-security', group 'AWS IAM' (line 16): Check 'mondoo-aws-security-access-keys-rotated' lacks an asset filter or variants, and the group also has no filter.", results.Entries[0].Message)
		assert.Equal(t, "policy-missing-asset-filter", results.Entries[0].RuleID)
		assert.Equal(t, "warning", results.Entries[0].Level)
	})

	t.Run("fail-policy-missing-asset-filter-groups", func(t *testing.T) {
		file := "./testdata/fail-policy-missing-asset-filter-groups.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "policy 'mondoo-aws-security', group 'AWS IAM' (line 16): Check 'mondoo-aws-security-access-keys-rotated' lacks an asset filter or variants, and the group also has no filter.", results.Entries[0].Message)
		assert.Equal(t, "policy-missing-asset-filter", results.Entries[0].RuleID)
		assert.Equal(t, "warning", results.Entries[0].Level)
	})

	t.Run("fail-policy-missing-checks", func(t *testing.T) {
		file := "./testdata/fail-policy-missing-checks.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 2, len(results.Entries))
		assert.Equal(t, "policy 'ubuntu-bench-1', group 'Configure Ubuntu 1' (line 14) has no checks, data queries, or sub-policies defined", results.Entries[0].Message)
		assert.Equal(t, "policy-missing-checks", results.Entries[0].RuleID)
		assert.Equal(t, "error", results.Entries[0].Level)
		assert.Equal(t, "Global query UID 'ubuntu-1-1' is defined but not assigned to any policy", results.Entries[1].Message)
		assert.Equal(t, "query-unassigned", results.Entries[1].RuleID)
		assert.Equal(t, "warning", results.Entries[1].Level)
	})

	t.Run("fail-policy-missing-version", func(t *testing.T) {
		file := "./testdata/fail-policy-missing-version.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "policy 'ubuntu-bench-1' is missing version", results.Entries[0].Message)
		assert.Equal(t, "policy-missing-version", results.Entries[0].RuleID)
		assert.Equal(t, "error", results.Entries[0].Level)
	})

	t.Run("fail-policy-wrong-version", func(t *testing.T) {
		file := "./testdata/fail-policy-wrong-version.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 2, len(results.Entries))

		result := findEntry(results.Entries, "policy-wrong-version")
		assert.Equal(t, "policy 'ubuntu-bench-1' has invalid version 'test.1.2.3.4': Invalid Semantic Version", result.Message)
		assert.Equal(t, "policy-wrong-version", result.RuleID)
		assert.Equal(t, "error", result.Level)

		result = findEntry(results.Entries, "bundle-compile-error")
		assert.Equal(t, "Could not compile policy bundle: failed to validate policy: policy '//local.cnspec.io/run/local-execution/policies/ubuntu-bench-1' version 'test.1.2.3.4' is not a valid semver version", result.Message)
		assert.Equal(t, "bundle-compile-error", result.RuleID)
		assert.Equal(t, "error", result.Level)
	})

	t.Run("fail-policy-required-tags-missing", func(t *testing.T) {
		file := "./testdata/fail-policy-required-tags-missing.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 2, len(results.Entries))
		assert.Equal(t, "policy 'ubuntu-bench-1' does not contain the required tag `mondoo.com/category`", results.Entries[0].Message)
		assert.Equal(t, "policy-required-tags-missing", results.Entries[0].RuleID)
		assert.Equal(t, "warning", results.Entries[0].Level)
		assert.Equal(t, "policy 'ubuntu-bench-1' does not contain the required tag `mondoo.com/platform`", results.Entries[1].Message)
		assert.Equal(t, "policy-required-tags-missing", results.Entries[1].RuleID)
		assert.Equal(t, "warning", results.Entries[1].Level)
	})

	t.Run("fail-query-uid-unique", func(t *testing.T) {
		file := "./testdata/fail-query-uid-unique.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "Global query UID 'ubuntu-1-1' is used multiple times in the same file", results.Entries[0].Message)
		assert.Equal(t, "query-uid-unique", results.Entries[0].RuleID)
		assert.Equal(t, "error", results.Entries[0].Level)
	})

	t.Run("fail-query-name", func(t *testing.T) {
		file := "./testdata/fail-query-name.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "Global query 'ubuntu-hard-2-2' does not define a title", results.Entries[0].Message)
		assert.Equal(t, "query-name", results.Entries[0].RuleID)
		assert.Equal(t, "error", results.Entries[0].Level)
	})

	t.Run("fail-query-variant-uses-non-default-fields", func(t *testing.T) {
		file := "./testdata/fail-query-variant-uses-non-default-fields.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "Query variant 'ubuntu-hard-2-1-var1' should not define 'impact'", results.Entries[0].Message)
		assert.Equal(t, "query-variant-uses-non-default-fields", results.Entries[0].RuleID)
		assert.Equal(t, "warning", results.Entries[0].Level)
	})

	t.Run("fail-query-missing-mql", func(t *testing.T) {
		file := "./testdata/fail-query-missing-mql.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 2, len(results.Entries))

		result := findEntry(results.Entries, "query-missing-mql")
		assert.Equal(t, "Global query 'ubuntu-hard-2-2' has no variants and must define MQL", result.Message)
		assert.Equal(t, "query-missing-mql", result.RuleID)
		assert.Equal(t, "error", result.Level)

		result = findEntry(results.Entries, "bundle-compile-error")
		assert.Equal(t, "Could not compile policy bundle: failed to validate query '//local.cnspec.io/run/local-execution/queries/ubuntu-hard-2-2': failed to compile query '': query is not implemented '//local.cnspec.io/run/local-execution/queries/ubuntu-hard-2-2'\n", result.Message)
		assert.Equal(t, "bundle-compile-error", result.RuleID)
		assert.Equal(t, "error", result.Level)
	})

	t.Run("fail-query-unassigned", func(t *testing.T) {
		file := "./testdata/fail-query-unassigned.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "Global query UID 'ubuntu-hard-1-1' is defined but not assigned to any policy", results.Entries[0].Message)
		assert.Equal(t, "query-unassigned", results.Entries[0].RuleID)
		assert.Equal(t, "warning", results.Entries[0].Level)
	})

	t.Run("fail-query-used-as-different-types", func(t *testing.T) {
		file := "./testdata/fail-query-used-as-different-types.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "Query UID 'sshd-sshd-01' is used as both a check and a data query in policies", results.Entries[0].Message)
		assert.Equal(t, "query-used-as-different-types", results.Entries[0].RuleID)
		assert.Equal(t, "error", results.Entries[0].Level)
	})

	t.Run("fail-bundle-unknown-field", func(t *testing.T) {
		file := "./testdata/fail-bundle-unknown-field.mql.yaml"
		results, err := Lint(schema, file)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results.Entries))
		assert.Equal(t, "Bundle file fail-bundle-unknown-field.mql.yaml contains unknown fields: error unmarshaling JSON: while decoding JSON: json: unknown field \"unknown_field\"", results.Entries[0].Message)
		assert.Equal(t, "bundle-unknown-field", results.Entries[0].RuleID)
		assert.Equal(t, "error", results.Entries[0].Level)
	})
}
