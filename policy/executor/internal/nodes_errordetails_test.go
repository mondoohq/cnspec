// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/types"
)

// Scores carry the classification of what made them errors, and the coverage
// gaps of the data they were computed from (mql ADR-46).

func scoreOf(t *testing.T, results ...*llx.RawData) *policy.Score {
	t.Helper()
	nodeData := &ReportingQueryNodeData{
		queryID: "testqueryid",
		results: map[string]*DataResult{},
	}
	for i, data := range results {
		checksum := fmt.Sprintf("checksum%d", i)
		nodeData.results[checksum] = &DataResult{
			checksum: checksum,
			resolved: true,
			value:    &llx.RawResult{CodeID: checksum, Data: data},
		}
	}
	nodeData.initialize()
	env := nodeData.recalculate()
	require.NotNil(t, env)
	require.NotNil(t, env.score)
	return env.score
}

func TestScoreCarriesTheErrorKinds(t *testing.T) {
	denied := llx.Forbidden(errors.New("not authorized"), llx.WithPermissions("ec2:DescribeInstances"))
	throttled := llx.TooManyRequests(errors.New("slow down"))

	score := scoreOf(t,
		&llx.RawData{Type: types.Bool, Error: denied},
		&llx.RawData{Type: types.Bool, Error: throttled},
		// The same denial again: one detail, not two.
		&llx.RawData{Type: types.Bool, Error: fmt.Errorf("listing: %w", denied)},
		// Unclassified: its message is in the score's message, it adds no detail.
		&llx.RawData{Type: types.Bool, Error: errors.New("boom")},
	)

	assert.Equal(t, policy.ScoreType_Error, score.Type)
	require.Len(t, score.ErrorDetails, 2)
	assert.Equal(t, llx.ErrorKind_ERROR_KIND_FORBIDDEN, score.ErrorDetails[0].Kind)
	assert.Equal(t, []string{"ec2:DescribeInstances"}, score.ErrorDetails[0].Permissions)
	assert.Equal(t, llx.ErrorKind_ERROR_KIND_TOO_MANY_REQUESTS, score.ErrorDetails[1].Kind)
	assert.Contains(t, score.Message, "boom")
}

func TestScoreWithOnlyUnclassifiedErrorsHasNoDetails(t *testing.T) {
	score := scoreOf(t, &llx.RawData{Type: types.Bool, Error: errors.New("boom")})
	assert.Equal(t, policy.ScoreType_Error, score.Type)
	assert.Nil(t, score.ErrorDetails)
}

func TestPassingScoreHasNoDetails(t *testing.T) {
	score := scoreOf(t, llx.BoolTrue)
	assert.Equal(t, policy.ScoreType_Result, score.Type)
	assert.Nil(t, score.ErrorDetails)
}

func TestScoreOverPartialDataKeepsItsOutcome(t *testing.T) {
	// A check over a list with one refused region: it scores on the regions
	// that answered, and the gap rides on the score (mql ADR-46 §8).
	gap := llx.Forbidden(errors.New("denied"),
		llx.WithScope(llx.ErrorScope_ERROR_SCOPE_PARTITION, "eu-west-1"),
		llx.WithPermissions("ec2:DescribeAddresses"))
	passing := llx.BoolTrue.WithCoverageGaps([]*llx.Error{gap})
	score := scoreOf(t, passing)
	assert.Equal(t, policy.ScoreType_Result, score.Type)
	assert.Equal(t, uint32(100), score.Value)
	require.Len(t, score.ErrorDetails, 1)
	assert.Equal(t, llx.ErrorKind_ERROR_KIND_FORBIDDEN, score.ErrorDetails[0].Kind)
	assert.Equal(t, llx.ErrorScope_ERROR_SCOPE_PARTITION, score.ErrorDetails[0].Scope)
	assert.Equal(t, "eu-west-1", score.ErrorDetails[0].ScopeId)

	failing := llx.BoolFalse.WithCoverageGaps(llx.UnionCoverageGaps([]*llx.Error{gap}, llx.CoverageGapsFrom(errors.New("region timed out"))))
	score = scoreOf(t, failing)
	assert.Equal(t, policy.ScoreType_Result, score.Type)
	assert.Equal(t, uint32(0), score.Value)
	// The unclassified gap is kept too: it is part of what the data is missing.
	require.Len(t, score.ErrorDetails, 2)
	assert.Equal(t, llx.ErrorKind_ERROR_KIND_UNSPECIFIED, score.ErrorDetails[1].Kind)
}

func TestAssetVanishedIsReadByKind(t *testing.T) {
	// The kind decides, not the message.
	vanished := llx.AssetVanished(errors.New("the container is gone"))
	score := scoreOf(t, &llx.RawData{Type: types.Bool, Error: vanished})
	assert.Equal(t, policy.ScoreType_Unscored, score.Type)
	require.Len(t, score.ErrorDetails, 1)
	assert.Equal(t, llx.ErrorKind_ERROR_KIND_ASSET_VANISHED, score.ErrorDetails[0].Kind)

	// The old message without the kind is an ordinary error now.
	score = scoreOf(t, &llx.RawData{Type: types.Bool, Error: errors.New("could not find resource aws.ec2")})
	assert.Equal(t, policy.ScoreType_Error, score.Type)

	// A placeholder broadcast for a query that could not run keeps its kind.
	score = scoreOf(t, &llx.RawData{Type: types.Bool, Error: &queryRunError{originCodeID: "testqueryid", err: vanished}})
	assert.Equal(t, policy.ScoreType_Unscored, score.Type)
}
