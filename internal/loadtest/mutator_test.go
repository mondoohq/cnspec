// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
)

func makeStateTemplate(n int) *Template {
	scores := make([]*policy.Score, n)
	for i := 0; i < n; i++ {
		v := uint32(PassingValue)
		if i%2 == 1 {
			v = FailingValue
		}
		scores[i] = &policy.Score{QrId: "q" + itoa(i), Value: v, Type: 1, Weight: 1}
	}
	return &Template{Scores: scores}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

func TestMutatorBaselineEqualsTemplate(t *testing.T) {
	tpl := makeStateTemplate(10)
	state := newScoreState(tpl, 1, 0)
	for i, s := range state.scores {
		require.Equal(t, tpl.Scores[i].Value, s.Value, "baseline score %d must match template", i)
	}
}

func TestMutatorZeroChangeIsNoop(t *testing.T) {
	tpl := makeStateTemplate(10)
	state := newScoreState(tpl, 1, 0)
	before := state.snapshot()
	state.applyChanges(0)
	after := state.snapshot()
	for i := range before {
		require.Equal(t, before[i].Value, after[i].Value)
	}
}

func TestMutatorFullChangeFlipsAll(t *testing.T) {
	tpl := makeStateTemplate(10)
	state := newScoreState(tpl, 1, 0)
	before := state.snapshot()
	state.applyChanges(100)
	after := state.snapshot()
	for i := range before {
		require.NotEqual(t, before[i].Value, after[i].Value, "every score should flip at change-pct=100")
	}
}

func TestMutatorChangePctApproximate(t *testing.T) {
	tpl := makeStateTemplate(100)
	state := newScoreState(tpl, 1, 0)
	before := state.snapshot()
	state.applyChanges(20)
	after := state.snapshot()

	flipped := 0
	for i := range before {
		if before[i].Value != after[i].Value {
			flipped++
		}
	}
	require.Equal(t, 20, flipped, "20%% of 100 scores should flip")
}

func TestMutatorDeterministicWithSameSeed(t *testing.T) {
	tpl := makeStateTemplate(50)

	s1 := newScoreState(tpl, 99, 7)
	s2 := newScoreState(tpl, 99, 7)
	for i := 0; i < 5; i++ {
		s1.applyChanges(10)
		s2.applyChanges(10)
	}
	for i := range s1.scores {
		require.Equal(t, s1.scores[i].Value, s2.scores[i].Value, "score %d diverged across runs with same seed", i)
	}
}

func TestMutatorDifferentSeedsDiverge(t *testing.T) {
	tpl := makeStateTemplate(50)
	s1 := newScoreState(tpl, 1, 0)
	s2 := newScoreState(tpl, 2, 0)
	s1.applyChanges(20)
	s2.applyChanges(20)
	diffs := 0
	for i := range s1.scores {
		if s1.scores[i].Value != s2.scores[i].Value {
			diffs++
		}
	}
	require.Greater(t, diffs, 0, "different seeds should produce different mutations")
}

func TestMutatorSnapshotIsDeepCopy(t *testing.T) {
	tpl := makeStateTemplate(3)
	state := newScoreState(tpl, 0, 0)
	snap := state.snapshot()
	// Mutate the snapshot; state must be unchanged.
	snap[0].Value = 12345
	require.NotEqual(t, uint32(12345), state.scores[0].Value)
}
