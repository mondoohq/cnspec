// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/utils/iox"
)

// Two checks with identical MQL compile to the same code id, and the executor
// runs that code once. The reporters named each code id by a single query
// (the last one in the bundle), so only one of the checks was reported and
// the other one had no result at all. The bundle also holds a third query with
// the same MQL that was not resolved for the asset (as a Windows twin of a
// Linux query would be); it must not be reported.
func identicalChecksReport() *policy.ReportCollection {
	const (
		assetMrn = "//assets/asset1"
		codeId   = "shared-code-id"
		checkA   = "//queries/check-a"
		checkB   = "//queries/check-b"
	)
	return &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{assetMrn: {Mrn: assetMrn, Name: "asset1"}},
		Bundle: &policy.Bundle{Queries: []*policy.Mquery{
			{Mrn: checkA, CodeId: codeId, Mql: `asset.name != ""`},
			{Mrn: checkB, CodeId: codeId, Mql: `asset.name != ""`},
			{Mrn: "//queries/other-platform", CodeId: codeId, Mql: `asset.name != ""`},
		}},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{assetMrn: {
			ExecutionJob: &policy.ExecutionJob{Queries: map[string]*policy.ExecutionQuery{codeId: {}}},
			CollectorJob: &policy.CollectorJob{ReportingJobs: map[string]*policy.ReportingJob{
				"rj-a": {Uuid: "rj-a", QrId: checkA, Type: policy.ReportingJob_CHECK},
				"rj-b": {Uuid: "rj-b", QrId: checkB, Type: policy.ReportingJob_CHECK},
			}},
		}},
		Reports: map[string]*policy.Report{assetMrn: {Scores: map[string]*policy.Score{
			codeId: {QrId: codeId, Type: policy.ScoreType_Result, Value: 100, ScoreCompletion: 100},
		}}},
	}
}

func TestIdenticalChecksAreAllReported(t *testing.T) {
	t.Run("proto", func(t *testing.T) {
		report, err := ConvertToProto(identicalChecksReport())
		require.NoError(t, err)
		values := report.Scores["//assets/asset1"].Values
		require.Contains(t, values, "//queries/check-a")
		require.Contains(t, values, "//queries/check-b")
		assert.Equal(t, "pass", values["//queries/check-a"].Status)
		assert.Equal(t, "pass", values["//queries/check-b"].Status)
		assert.NotContains(t, values, "//queries/other-platform")
	})

	t.Run("json", func(t *testing.T) {
		buf := bytes.Buffer{}
		require.NoError(t, ConvertToJSON(identicalChecksReport(), &iox.IOWriter{Writer: &buf}))
		require.True(t, json.Valid(buf.Bytes()), buf.String())
		var out struct {
			Scores map[string]map[string]any `json:"scores"`
		}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
		assert.Contains(t, out.Scores["//assets/asset1"], "//queries/check-a")
		assert.Contains(t, out.Scores["//assets/asset1"], "//queries/check-b")
		assert.NotContains(t, out.Scores["//assets/asset1"], "//queries/other-platform")
	})
}

// The Ubuntu fixture's bundle holds Linux, macOS and Windows data queries with
// identical code; only the Linux ones are resolved for the Ubuntu asset, so
// they are the ones reported.
func TestSharedCodeIsReportedUnderTheResolvedQuery(t *testing.T) {
	yr, err := reportfixture.UbuntuScan()
	require.NoError(t, err)
	report, err := ConvertToProto(yr)
	require.NoError(t, err)

	for _, values := range report.Data {
		assert.Contains(t, values.Values, "//policy.api.mondoo.app/queries/mondoo-linux-installed-packages")
		assert.NotContains(t, values.Values, "//policy.api.mondoo.app/queries/mondoo-windows-packages")
		assert.Contains(t, values.Values, "//policy.api.mondoo.app/queries/mondoo-linux-mounts")
		assert.NotContains(t, values.Values, "//policy.api.mondoo.app/queries/mondoo-macos-mounts")
	}
}
