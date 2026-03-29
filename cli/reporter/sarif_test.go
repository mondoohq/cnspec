// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/utils/iox"
)

func TestSarifConverter(t *testing.T) {
	yr := sampleReportCollection()
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	sarifReport := buf.String()

	// Verify it's valid JSON
	var parsed map[string]any
	err = json.Unmarshal([]byte(sarifReport), &parsed)
	require.NoError(t, err)

	// Verify SARIF version
	assert.Equal(t, "2.1.0", parsed["version"])

	// Verify schema
	assert.Contains(t, sarifReport, "https://raw.githubusercontent.com/oasis-tcs/sarif-spec")

	// Verify tool info
	assert.Contains(t, sarifReport, "\"name\":\"cnspec\"")
	assert.Contains(t, sarifReport, "https://cnspec.io")

	// Verify rules are present
	assert.Contains(t, sarifReport, "Ensure SNMP server is stopped and not enabled")
	assert.Contains(t, sarifReport, "Configure kubelet to capture all event creation")
	assert.Contains(t, sarifReport, "Set secure file permissions on the scheduler.conf file")

	// Verify results contain asset name
	assert.Contains(t, sarifReport, "X1")

	// Verify results contain expected levels
	// Score type 2 (Result) with value 100 -> "none" (pass)
	// Score type 4 (Error) -> "error"
	// Score type 8 (Skip) -> "none"
	// Each asset gets its own run
	runs := parsed["runs"].([]any)
	require.Len(t, runs, 1)
	run := runs[0].(map[string]any)

	// Verify run-level asset properties
	props := run["properties"].(map[string]any)
	assert.Equal(t, "X1", props["asset"])

	results := run["results"].([]any)
	require.NotEmpty(t, results)

	// Verify each result has a level and message
	for _, r := range results {
		result := r.(map[string]any)
		assert.Contains(t, result, "level")
		assert.Contains(t, result, "message")
	}
}

func TestSarifDeterministicOutput(t *testing.T) {
	yr := sampleReportCollection()

	// Run twice and verify identical output
	buf1 := bytes.Buffer{}
	writer1 := iox.IOWriter{Writer: &buf1}
	err := ConvertToSarif(yr, &writer1)
	require.NoError(t, err)

	buf2 := bytes.Buffer{}
	writer2 := iox.IOWriter{Writer: &buf2}
	err = ConvertToSarif(yr, &writer2)
	require.NoError(t, err)

	assert.Equal(t, buf1.String(), buf2.String())
}

func TestSarifNilReport(t *testing.T) {
	var yr *policy.ReportCollection

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "2.1.0", parsed["version"])
}

func TestSarifWithAssetErrors(t *testing.T) {
	yr := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			"asset1": {Name: "test-server"},
		},
		Bundle: &policy.Bundle{},
		Errors: map[string]string{
			"asset1": "connection refused",
		},
	}

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	sarifReport := buf.String()
	assert.Contains(t, sarifReport, "asset-error")
	assert.Contains(t, sarifReport, "connection refused")
	assert.Contains(t, sarifReport, "test-server")
}

func TestSarifWithNilCollectorJob(t *testing.T) {
	yr := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			"asset1": {Name: "test-server"},
		},
		Bundle: &policy.Bundle{},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			"asset1": {CollectorJob: nil},
		},
		Reports: map[string]*policy.Report{
			"asset1": {Scores: map[string]*policy.Score{}},
		},
	}

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "2.1.0", parsed["version"])

	// Should have one run for the single asset
	runs := parsed["runs"].([]any)
	require.Len(t, runs, 1)
}

func TestSarifMultipleAssets(t *testing.T) {
	yr := &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			"asset1": {Name: "server-a"},
			"asset2": {Name: "server-b"},
		},
		Bundle: &policy.Bundle{},
		Errors: map[string]string{
			"asset2": "timeout",
		},
	}

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToSarif(yr, &writer)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Each asset should have its own run
	runs := parsed["runs"].([]any)
	require.Len(t, runs, 2)

	// Verify each run is tagged with its asset name
	run1 := runs[0].(map[string]any)
	run2 := runs[1].(map[string]any)
	props1 := run1["properties"].(map[string]any)
	props2 := run2["properties"].(map[string]any)

	assets := []string{props1["asset"].(string), props2["asset"].(string)}
	assert.Contains(t, assets, "server-a")
	assert.Contains(t, assets, "server-b")

	// Only the errored asset's run should have the error result
	sarifReport := buf.String()
	assert.Contains(t, sarifReport, "timeout")
}

func TestSarifQueryRuleID(t *testing.T) {
	tests := []struct {
		name     string
		query    *policy.Mquery
		expected string
	}{
		{
			"prefers uid",
			&policy.Mquery{Uid: "my-check", Mrn: "//local.cnspec.io/run/local-execution/queries/my-check"},
			"my-check",
		},
		{
			"strips local MRN prefix",
			&policy.Mquery{Mrn: "//local.cnspec.io/run/local-execution/queries/sshd-01"},
			"sshd-01",
		},
		{
			"strips policy API MRN prefix",
			&policy.Mquery{Mrn: "//policy.api.mondoo.app/queries/mondoo-linux-security-snmp"},
			"mondoo-linux-security-snmp",
		},
		{
			"falls back to code ID",
			&policy.Mquery{CodeId: "abc123"},
			"abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, queryRuleID(tt.query))
		})
	}
}

func TestSarifScoreLevels(t *testing.T) {
	tests := []struct {
		name     string
		score    *policy.Score
		expected string
	}{
		{"nil score", nil, "none"},
		{"pass", &policy.Score{Type: policy.ScoreType_Result, Value: 100}, "none"},
		{"warning high", &policy.Score{Type: policy.ScoreType_Result, Value: 99}, "warning"},
		{"warning mid", &policy.Score{Type: policy.ScoreType_Result, Value: 50}, "warning"},
		{"fail", &policy.Score{Type: policy.ScoreType_Result, Value: 49}, "error"},
		{"fail zero", &policy.Score{Type: policy.ScoreType_Result, Value: 0}, "error"},
		{"error type", &policy.Score{Type: policy.ScoreType_Error}, "error"},
		{"skip", &policy.Score{Type: policy.ScoreType_Skip}, "none"},
		{"unknown", &policy.Score{Type: policy.ScoreType_Unknown}, "none"},
		{"unscored", &policy.Score{Type: policy.ScoreType_Unscored}, "none"},
		{"out of scope", &policy.Score{Type: policy.ScoreType_OutOfScope}, "none"},
		{"disabled", &policy.Score{Type: policy.ScoreType_Disabled}, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scoreToSarifLevel(tt.score))
		})
	}
}
