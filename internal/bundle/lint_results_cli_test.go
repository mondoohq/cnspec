// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResults_PromoteWarnings(t *testing.T) {
	newResults := func() *Results {
		return &Results{
			Entries: []*Entry{
				{RuleID: "query-deprecated-symbol", Level: LevelWarning},
				{RuleID: "query-unassigned", Level: LevelWarning},
				{RuleID: "query-uid", Level: LevelError},
			},
		}
	}

	t.Run("named rule promotes only matching warnings", func(t *testing.T) {
		r := newResults()
		r.PromoteWarnings([]string{"query-deprecated-symbol"})
		assert.Equal(t, LevelError, r.Entries[0].Level, "matched warning is promoted")
		assert.Equal(t, LevelWarning, r.Entries[1].Level, "unmatched warning stays a warning")
		assert.Equal(t, LevelError, r.Entries[2].Level, "pre-existing error is unchanged")
	})

	t.Run("all promotes every warning", func(t *testing.T) {
		r := newResults()
		r.PromoteWarnings([]string{StrictRuleAll})
		assert.Equal(t, LevelError, r.Entries[0].Level)
		assert.Equal(t, LevelError, r.Entries[1].Level)
		assert.Equal(t, LevelError, r.Entries[2].Level)
	})

	t.Run("unknown rule id is a no-op", func(t *testing.T) {
		r := newResults()
		r.PromoteWarnings([]string{"does-not-exist"})
		assert.Equal(t, LevelWarning, r.Entries[0].Level)
		assert.Equal(t, LevelWarning, r.Entries[1].Level)
		assert.Equal(t, LevelError, r.Entries[2].Level)
	})

	t.Run("empty input is a no-op", func(t *testing.T) {
		r := newResults()
		r.PromoteWarnings(nil)
		assert.True(t, r.HasWarning())
	})

	t.Run("mixed all and named keeps all-promotion semantics", func(t *testing.T) {
		r := newResults()
		r.PromoteWarnings([]string{"query-deprecated-symbol", StrictRuleAll})
		assert.Equal(t, LevelError, r.Entries[0].Level)
		assert.Equal(t, LevelError, r.Entries[1].Level)
	})

	t.Run("nil receiver does not panic", func(t *testing.T) {
		var r *Results
		assert.NotPanics(t, func() { r.PromoteWarnings([]string{StrictRuleAll}) })
	})
}
