// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import "strconv"

// cnspec score rating ranges follow this model:
//
// A 80  ..  100 (100 A+ 95 A 85 A- 80)
// B 60  ..   79 ( 79 B+ 75 B 65 B- 60)
// C 30  ..   59 ( 59 C+ 50 C 40 C- 30)
// D 10  ..   29 ( 29 D+ 25 D 15 D- 10)
// F 0   ..    9
// U not completed
// S skip
// X error
func (s *Score) Rating() ScoreRating {
	if s == nil || s.Type == ScoreType_Unknown || s.Type == ScoreType_Unscored {
		return ScoreRating_unrated
	}

	if s.Type == ScoreType_Skip {
		return ScoreRating_skip
	}

	if s.Type == ScoreType_Error {
		return ScoreRating_error
	}

	if s.Completion() == 0 {
		return ScoreRating_unrated
	}

	switch {
	case s.Value >= 95:
		return ScoreRating_aPlus
	case s.Value >= 85:
		return ScoreRating_a
	case s.Value >= 80:
		return ScoreRating_aMinus
	case s.Value >= 75:
		return ScoreRating_bPlus
	case s.Value >= 65:
		return ScoreRating_b
	case s.Value >= 60:
		return ScoreRating_bMinus
	case s.Value >= 50:
		return ScoreRating_cPlus
	case s.Value >= 40:
		return ScoreRating_c
	case s.Value >= 30:
		return ScoreRating_cMinus
	case s.Value >= 29:
		return ScoreRating_dPlus
	case s.Value >= 15:
		return ScoreRating_d
	case s.Value >= 10:
		return ScoreRating_dMinus
	case s.Value < 10:
		return ScoreRating_failed
	}
	return ScoreRating_unrated
}

var ScoreRating_Letters = map[int32]string{
	0:  "U",
	1:  "A",
	2:  "A",
	3:  "A",
	4:  "B",
	5:  "B",
	6:  "B",
	7:  "C",
	8:  "C",
	9:  "C",
	10: "D",
	11: "D",
	12: "D",
	13: "F",
	14: "X",
	15: "S",
}

func (r ScoreRating) Letter() string {
	return enumName(ScoreRating_Letters, int32(r))
}

var ScoreRating_CategoryLabel = map[int32]string{
	0:  "Unrated",   // Unscored/Unrated
	1:  "Excellent", // A+
	2:  "Excellent", // A
	3:  "Excellent", // A-
	4:  "Good",      // B+
	5:  "Good",      // B
	6:  "Good",      // B-
	7:  "Fair",      // C+
	8:  "Fair",      // C
	9:  "Fair",      // C-
	10: "Poor",      // D+
	11: "Poor",      // D
	12: "Poor",      // D-
	13: "Fail",      // F
	14: "Error",     // X
	15: "Skip",      // S
}

func (r ScoreRating) CategoryLabel() string {
	return enumName(ScoreRating_CategoryLabel, int32(r))
}

var ScoreRating_FailureLabel = map[int32]string{
	0:  "unrated",  // Unscored/Unrated
	1:  "low",      // A+
	2:  "low",      // A
	3:  "low",      // A-
	4:  "low",      // B+
	5:  "low",      // B
	6:  "low",      // B-
	7:  "medium",   // C+
	8:  "medium",   // C
	9:  "medium",   // C-
	10: "high",     // D+
	11: "high",     // D
	12: "high",     // D-
	13: "critical", // F
	14: "error",    // X
	15: "skip",     // S
}

func (r ScoreRating) FailureLabel() string {
	return enumName(ScoreRating_FailureLabel, int32(r))
}

func enumName(m map[int32]string, v int32) string {
	s, ok := m[v]
	if ok {
		return s
	}
	return strconv.Itoa(int(v))
}
