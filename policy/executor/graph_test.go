// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
)

func filterScore(qrID string, value uint32) *policy.Score {
	return &policy.Score{
		QrId:            qrID,
		Type:            policy.ScoreType_Result,
		Value:           value,
		ScoreCompletion: 100,
	}
}

// The graph emits a score every round a query's entrypoints are all resolved,
// and a provisional round can score 100 on a filter that does not actually
// match (a shared entrypoint checksum resolved by another query as a
// short-circuit nil, which score() skips instead of failing). Only the final
// score may decide whether the filter matched.
func TestFilterScoreTracker_ProvisionalPassIsRevised(t *testing.T) {
	// The Debian 8 filter on a Debian 11 host: `asset.platform == "debian"`
	// and `asset.kind != "container-image"` resolve TRUE from the Debian 9/10
	// filters (identical statements, identical checksums) while
	// `asset.version == /^8\./` is still nil, so the query provisionally
	// scores 100. Once its own execution reports the version statement as
	// false, the node recalculates to 0.
	tracker := newFilterScoreTracker()
	tracker.record(filterScore("debian8-filter", 100))
	tracker.record(filterScore("debian8-filter", 0))

	assert.NotContains(t, tracker.passing(), "debian8-filter",
		"a provisional 100 that was revised to 0 must not count as a match")
}

func TestFilterScoreTracker_Passing(t *testing.T) {
	t.Run("a single passing score matches", func(t *testing.T) {
		tracker := newFilterScoreTracker()
		tracker.record(filterScore("debian11-filter", 100))
		assert.Contains(t, tracker.passing(), "debian11-filter")
	})

	t.Run("a provisional failure upgraded to a pass matches", func(t *testing.T) {
		// The mirror of the case above: a placeholder from a query that could
		// not run poisons a shared checksum first, and the query's own
		// execution then reports the real, passing result.
		tracker := newFilterScoreTracker()
		tracker.record(filterScore("debian11-filter", 0))
		tracker.record(filterScore("debian11-filter", 100))
		assert.Contains(t, tracker.passing(), "debian11-filter")
	})

	t.Run("a failing score does not match", func(t *testing.T) {
		tracker := newFilterScoreTracker()
		tracker.record(filterScore("debian8-filter", 0))
		assert.Empty(t, tracker.passing())
	})

	t.Run("an incomplete score does not match", func(t *testing.T) {
		tracker := newFilterScoreTracker()
		s := filterScore("debian8-filter", 100)
		s.ScoreCompletion = 50
		tracker.record(s)
		assert.Empty(t, tracker.passing())
	})

	t.Run("an errored score does not match", func(t *testing.T) {
		tracker := newFilterScoreTracker()
		tracker.record(&policy.Score{
			QrId:            "debian8-filter",
			Type:            policy.ScoreType_Error,
			Value:           0,
			ScoreCompletion: 100,
		})
		assert.Empty(t, tracker.passing())
	})

	t.Run("queries are tracked independently", func(t *testing.T) {
		tracker := newFilterScoreTracker()
		tracker.record(filterScore("debian8-filter", 100))
		tracker.record(filterScore("debian11-filter", 100))
		tracker.record(filterScore("debian8-filter", 0))

		passing := tracker.passing()
		assert.NotContains(t, passing, "debian8-filter")
		assert.Contains(t, passing, "debian11-filter")
	})

	t.Run("nil scores are ignored", func(t *testing.T) {
		tracker := newFilterScoreTracker()
		tracker.record(nil)
		assert.Empty(t, tracker.passing())
	})
}

func TestDedupeAndCapScanWarnings(t *testing.T) {
	t.Run("empty input returns nil", func(t *testing.T) {
		assert.Nil(t, dedupeAndCapScanWarnings(nil))
	})

	t.Run("nil errors are skipped", func(t *testing.T) {
		out := dedupeAndCapScanWarnings([]error{nil, nil})
		assert.Empty(t, out)
	})

	t.Run("duplicate messages collapse to one", func(t *testing.T) {
		errs := []error{
			errors.New("the 'os' provider crashed: connection refused"),
			errors.New("the 'os' provider crashed: connection refused"),
			errors.New("the 'aws' provider crashed: EOF"),
		}
		out := dedupeAndCapScanWarnings(errs)
		assert.ElementsMatch(t, []string{
			"the 'os' provider crashed: connection refused",
			"the 'aws' provider crashed: EOF",
		}, out)
	})

	t.Run("count is capped", func(t *testing.T) {
		errs := make([]error, 0, maxScanWarnings+10)
		for i := 0; i < maxScanWarnings+10; i++ {
			errs = append(errs, fmt.Errorf("distinct crash #%d", i))
		}
		out := dedupeAndCapScanWarnings(errs)
		assert.Len(t, out, maxScanWarnings)
	})

	t.Run("message length is capped", func(t *testing.T) {
		long := strings.Repeat("x", maxScanWarningLen+500)
		out := dedupeAndCapScanWarnings([]error{errors.New(long)})
		require.Len(t, out, 1)
		assert.Len(t, out[0], maxScanWarningLen)
	})
}
