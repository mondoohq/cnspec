// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"encoding/binary"
	"hash/fnv"
	"math/rand"

	"go.mondoo.com/cnspec/v13/policy"
	"google.golang.org/protobuf/proto"
)

// scoreState tracks whether each (qr_id) is currently passing in the simulated
// stream, plus a deterministic RNG for choosing which scores to flip on the
// next scan. State is per-asset because each asset evolves independently.
type scoreState struct {
	rng     *rand.Rand
	scores  []*policy.Score
	passing []bool
}

// PassingValue is the score value used to mark a score as passing in the
// load-test simulation. Real cnspec scoring uses 0..100 with 100 meaning a
// full pass; for the loadtest we collapse to a binary 100/0 model since the
// goal is to drive ingestion volume, not to reproduce nuanced scoring.
const PassingValue = 100

// FailingValue is the binary failure counterpart to PassingValue.
const FailingValue = 0

// newScoreState produces the initial (baseline) score state for an asset by
// deep-cloning the template's scores. The first scan replays the template
// verbatim; subsequent scans mutate this state in place.
func newScoreState(template *Template, seed int64, assetIdx int) *scoreState {
	scores := make([]*policy.Score, len(template.Scores))
	passing := make([]bool, len(template.Scores))
	for i, s := range template.Scores {
		scores[i] = proto.Clone(s).(*policy.Score)
		passing[i] = scores[i].Value >= PassingValue
	}
	return &scoreState{
		rng:     rand.New(rand.NewSource(perAssetSeed(seed, assetIdx))),
		scores:  scores,
		passing: passing,
	}
}

// applyChanges flips floor(len(scores) * changePct/100) scores chosen
// uniformly without replacement. Each chosen score swaps state (pass↔fail).
// changePct=0 is a no-op; changePct=100 flips everything.
//
// Mutations stack across iterations: scan N's state is scan N-1's state with
// the chosen scores flipped. This produces realistic drift (an asset that
// degraded once stays degraded until something flips it back).
func (s *scoreState) applyChanges(changePct float64) {
	if changePct <= 0 || len(s.scores) == 0 {
		return
	}
	n := int(float64(len(s.scores)) * changePct / 100.0)
	if n <= 0 {
		return
	}
	if n > len(s.scores) {
		n = len(s.scores)
	}

	indices := s.rng.Perm(len(s.scores))[:n]
	for _, i := range indices {
		s.passing[i] = !s.passing[i]
		if s.passing[i] {
			s.scores[i].Value = PassingValue
		} else {
			s.scores[i].Value = FailingValue
		}
	}
}

// snapshot returns a deep copy of the current scores so the caller can hand
// them off to a network call without later mutations racing with the wire
// payload.
func (s *scoreState) snapshot() []*policy.Score {
	out := make([]*policy.Score, len(s.scores))
	for i, sc := range s.scores {
		out[i] = proto.Clone(sc).(*policy.Score)
	}
	return out
}

func perAssetSeed(globalSeed int64, assetIdx int) int64 {
	h := fnv.New64a()
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[0:8], uint64(globalSeed))
	binary.BigEndian.PutUint64(buf[8:16], uint64(assetIdx))
	h.Write(buf[:])
	return int64(h.Sum64())
}
