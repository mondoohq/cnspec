// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportdoc

import "go.mondoo.com/cnspec/policy"

// Outcome is what happened to a check on one asset, as the report formats need
// to talk about it: it either produced a verdict (Pass, Fail), could not be
// evaluated (Error), was deliberately not evaluated (Skipped), ran without being
// scored (Unscored), or reported nothing at all (Unknown).
//
// It exists for the same reason Band does, one level up. policy.ScoreType is a
// seven-value enum with three flavors of "did not apply" and a zero value that
// means "no score reported"; every format collapses it, and they all have to
// collapse it the same way. Deriving that collapse per format is how it drifts.
//
// The vocabulary is deliberately not any format's. SARIF's kind enum has no
// error member, so SARIF folds Error into "fail" and prints the word ERROR
// beside it; OHDF has an error status and uses it; OCSF has neither and reports
// the outcome as its own producer-defined status. When the shared step was
// SARIF's kind, three of those formats had to detect the Error case and undo the
// fold - each one reaching back into policy.ScoreType behind the helper that was
// supposed to have answered the question. Outcome is what they collapse to
// instead, and every format maps it to its own words afterwards.
type Outcome int

const (
	// OutcomeUnknown is the absence of an answer: no score, or a score type that
	// carries none. It is the zero value so that a missing score reads as
	// "nothing is known", never as a verdict.
	OutcomeUnknown Outcome = iota
	OutcomePass
	OutcomeFail
	// OutcomeError is a check that could not be evaluated. It is not a Fail: a
	// provider that could not answer is a gap in coverage, and reporting it as a
	// violation is a false positive with a compliance consequence.
	OutcomeError
	// OutcomeSkipped is a check the scan deliberately did not evaluate, whether
	// by a platform filter, by scope, or by being disabled.
	OutcomeSkipped
	// OutcomeUnscored is a check that ran and produced data but no verdict.
	OutcomeUnscored
)

// OutcomeOf collapses a cnspec score onto the outcome its report formats agree
// on. This is the single definition of that collapse; mirror it nowhere else.
//
//	Result, value 100          → OutcomePass
//	Result, any other value    → OutcomeFail
//	Error                      → OutcomeError
//	Skip, OutOfScope, Disabled → OutcomeSkipped
//	Unscored                   → OutcomeUnscored
//	Unknown, or no score       → OutcomeUnknown
func OutcomeOf(score *policy.Score) Outcome {
	if score == nil {
		return OutcomeUnknown
	}

	switch score.Type {
	case policy.ScoreType_Result:
		if score.Value == 100 {
			return OutcomePass
		}
		return OutcomeFail
	case policy.ScoreType_Error:
		return OutcomeError
	case policy.ScoreType_Skip, policy.ScoreType_OutOfScope, policy.ScoreType_Disabled:
		return OutcomeSkipped
	case policy.ScoreType_Unscored:
		return OutcomeUnscored
	default:
		return OutcomeUnknown
	}
}

// Label is the outcome in the words cnspec itself uses, which is what the SARIF
// status line, the OCSF status_code and the compact output all print. Other
// formats have their own vocabulary for the same six outcomes; see Outcome.
func (o Outcome) Label() string {
	switch o {
	case OutcomePass:
		return "PASS"
	case OutcomeFail:
		return "FAIL"
	case OutcomeError:
		return "ERROR"
	case OutcomeSkipped:
		return "SKIPPED"
	case OutcomeUnscored:
		return "UNSCORED"
	default:
		return "UNKNOWN"
	}
}
