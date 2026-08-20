// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
