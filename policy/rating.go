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
	// None (100) - we are merging none with low for now

	// Low (61-99)
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
	case s.Value >= 61:
		return ScoreRating_bMinus

	// Medium (31-60)
	case s.Value >= 50:
		return ScoreRating_cPlus
	case s.Value >= 40:
		return ScoreRating_c
	case s.Value >= 31:
		return ScoreRating_cMinus

	// High (11-30)
	case s.Value >= 25:
		return ScoreRating_dPlus
	case s.Value >= 15:
		return ScoreRating_d
	case s.Value >= 11:
		return ScoreRating_dMinus

	// Critical (0-10)
	case s.Value <= 10:
		return ScoreRating_failed
	}
	return ScoreRating_unrated
}

var ScoreRatingsText = map[int32]string{
	0:  ScoreRatingTextUnrated,  // Unscored/Unrated
	1:  ScoreRatingTextLow,      // A+
	2:  ScoreRatingTextLow,      // A
	3:  ScoreRatingTextLow,      // A-
	4:  ScoreRatingTextLow,      // B+
	5:  ScoreRatingTextLow,      // B
	6:  ScoreRatingTextLow,      // B-
	7:  ScoreRatingTextMedium,   // C+
	8:  ScoreRatingTextMedium,   // C
	9:  ScoreRatingTextMedium,   // C-
	10: ScoreRatingTextHigh,     // D+
	11: ScoreRatingTextHigh,     // D
	12: ScoreRatingTextHigh,     // D-
	13: ScoreRatingTextCritical, // F
	14: ScoreRatingTextError,    // X
	15: ScoreRatingTextSkip,     // S
}

const (
	ScoreRatingTextError    = "ERROR"
	ScoreRatingTextSkip     = "SKIP"
	ScoreRatingTextUnrated  = "UNRATED"
	ScoreRatingTextNone     = "NONE"
	ScoreRatingTextLow      = "LOW"
	ScoreRatingTextMedium   = "MEDIUM"
	ScoreRatingTextHigh     = "HIGH"
	ScoreRatingTextCritical = "CRITICAL"
)

// Text returns a string representation of the score rating
func (r ScoreRating) Text() string {
	return enumName(ScoreRatingsText, int32(r))
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

func (r ScoreRating) CategoryLabel() string {
	return enumName(ScoreRating_CategoryLabel, int32(r))
}

func enumName(m map[int32]string, v int32) string {
	s, ok := m[v]
	if ok {
		return s
	}
	return strconv.Itoa(int(v))
}
