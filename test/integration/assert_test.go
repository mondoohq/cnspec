// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reporter"
	mqlreporter "go.mondoo.com/mql/cli/reporter"
	"google.golang.org/protobuf/encoding/protojson"
)

// The tests in this file need no daemon, no cluster and no network. They test
// the suite's own logic against synthetic reports.
//
// They exist because the assertions are the whole product here: an assertion
// helper with a bug is indistinguishable from no assertion at all. This is the
// cheapest way to know they still bite.

func scoreValues(statuses ...string) map[string]*reporter.ScoreValue {
	out := make(map[string]*reporter.ScoreValue, len(statuses))
	for i, s := range statuses {
		risk := uint32(0)
		if s != "pass" {
			risk = 100
		}
		out[string(rune('a'+i))] = &reporter.ScoreValue{Status: s, RiskScore: risk}
	}
	return out
}

func TestStatusCounts(t *testing.T) {
	got := statusCounts(scoreValues("pass", "pass", "fail", "error", "skip", "skip"))
	assert.Equal(t, map[string]int{"pass": 2, "fail": 1, "error": 1, "skip": 2}, got)
	assert.Empty(t, statusCounts(nil))
}

func TestVerdictCount(t *testing.T) {
	tests := []struct {
		name     string
		statuses []string
		want     int
	}{
		{"pass and fail both count", []string{"pass", "fail"}, 2},
		// The shape that is hardest to notice from outside: cnspec connected,
		// produced an asset and scored every check, but nothing was evaluated.
		{"all skipped is no verdict", []string{"skip", "skip", "skip"}, 0},
		{"all errored is no verdict", []string{"error", "error"}, 0},
		{"unscored and disabled do not count", []string{"unscored", "disabled", "out of scope"}, 0},
		{"mixed", []string{"pass", "skip", "error", "fail", "unknown"}, 2},
		{"empty", nil, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, verdictCount(scoreValues(tc.statuses...)))
		})
	}
}

func TestCountProviderPanics(t *testing.T) {
	// Abridged from a real run: a nil dereference in the os provider's kernel
	// resource, recovered by the plugin layer. The scan still exited 0.
	const line = `x recovered panic in provider method=GetData panic="invalid memory address or nil pointer dereference" stack="goroutine 134 [running]:..."`

	assert.Equal(t, 0, countProviderPanics(nil))
	assert.Equal(t, 0, countProviderPanics([]byte("→ discover related assets for 1 asset(s)\n")))
	assert.Equal(t, 1, countProviderPanics([]byte(line)))
	assert.Equal(t, 3, countProviderPanics([]byte(line+"\n"+line+"\n"+line)))
}

func TestCheckScoresDropsTheAggregate(t *testing.T) {
	const mrn = "//policy.api.mondoo.com/assets/abc"
	rep := &reporter.Report{
		Assets: map[string]*mqlreporter.Asset{},
		Scores: map[string]*reporter.ScoreValues{
			mrn: {Values: map[string]*reporter.ScoreValue{
				mrn:       {Status: "fail", RiskScore: 100}, // the asset's own aggregate
				"check-1": {Status: "pass"},
				"check-2": {Status: "fail", RiskScore: 100},
			}},
		},
	}
	got := checkScores(t, rep, mrn)
	require.Len(t, got, 2, "the asset's aggregate score must not be counted as a check")
	assert.NotContains(t, got, mrn)
}

// TestDecodeReportNeedsProtojson pins the reason decodeReport does not use
// encoding/json.
//
// protojson emits a proto field's JSON name (platformName) while the generated
// Go struct tag carries the proto name (platform_name). Decoding real cnspec
// output with encoding/json therefore leaves PlatformName empty, and an
// assertion on the platform would compare against "" and quietly hold. This
// test fails if that ever stops being true, at which point the comment in
// decodeReport is wrong and should be removed rather than trusted.
func TestDecodeReportNeedsProtojson(t *testing.T) {
	const payload = `{"assets":{"//assets/abc":{"mrn":"//assets/abc","name":"alpine:3.20","platformName":"alpine"}}}`

	rep := &reporter.Report{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	require.NoError(t, opts.Unmarshal([]byte(payload), rep))
	require.Len(t, rep.GetAssets(), 1)
	for _, a := range rep.GetAssets() {
		assert.Equal(t, "alpine", a.GetPlatformName(),
			"protojson must populate the platform name")
	}
}
