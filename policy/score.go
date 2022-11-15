package policy

import (
	"fmt"
	"strings"
)

const (
	ScoreType_Unknown uint32 = 0
	ScoreType_Result  uint32 = 1 << iota
	ScoreType_Error
	ScoreType_Skip
	ScoreType_Unscored
)

// TypeLabel prints the score's type in a human-readable way
func (s *Score) TypeLabel() string {
	switch s.Type {
	case ScoreType_Unknown:
		return "unknown"
	case ScoreType_Error:
		return "error"
	case ScoreType_Result:
		return "result"
	case ScoreType_Skip:
		return "skip"
	case ScoreType_Unscored:
		return "unscored"
	default:
		return "unknown type"
	}
}

func (s *Score) HumanStatus() string {
	if s == nil {
		return "N/A"
	}

	return fmt.Sprintf("%d (completion: %d%%)", s.Value, s.Completion())
}

// Completion of the score based on its data and scoring completion
func (s *Score) Completion() uint32 {
	return (s.DataCompletion + s.ScoreCompletion) / 2
}

// MessageLine prints the message as a single line
func (s *Score) MessageLine() string {
	if s == nil {
		return ""
	}

	res := strings.TrimSpace(s.Message)
	return strings.ReplaceAll(res, "\n", " ")
}

func (s *ScoringSpec) ImpactValue() *ImpactValue {
	if s.GetImpact() != nil {
		return s.Impact
	}

	// FIXME: DEPRECATED, remove in v8.0 vv
	if s.GetSeverity() != nil {
		return s.Severity
	}
	// ^^
	return nil
}
