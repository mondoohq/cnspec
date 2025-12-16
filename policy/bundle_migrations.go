// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import "fmt"

// StageSummary summarizes the effects of a migration stage
// used for validation purposes
type StageSummary struct {
	Index int
	Title string

	// UID effects
	Consumes map[string]*Migration // UID → migration
	Produces map[string]*Migration // UID → migration

	// Categorized (for better errors)
	Renames map[string]string // from → to
	Deletes map[string]bool   // from → to
	Creates map[string]bool   // from → to
}

func SummarizeStage(stage *MigrationStage, index int) (*StageSummary, []error) {
	s := &StageSummary{
		Index:    index,
		Title:    stage.Title,
		Consumes: make(map[string]*Migration),
		Produces: make(map[string]*Migration),
		Renames:  make(map[string]string),
		Deletes:  make(map[string]bool),
		Creates:  make(map[string]bool),
	}

	var errs []error

	for _, m := range stage.Migrations {
		src := ""
		if m.Source != nil {
			src = m.Source.Uid
		}
		dst := ""
		if m.Destination != nil {
			dst = m.Destination.Uid
		}

		switch m.Action {

		case Migration_MODIFY:
			// consume source
			if _, ok := s.Consumes[src]; ok {
				errs = append(errs, fmt.Errorf(
					"UID %q consumed multiple times in stage %q",
					src, s.Title,
				))
			}
			s.Consumes[src] = m

			// produce destination
			if _, ok := s.Produces[dst]; ok {
				errs = append(errs, fmt.Errorf(
					"UID %q produced multiple times in stage %q",
					dst, s.Title,
				))
			}
			s.Produces[dst] = m

			s.Renames[src] = dst

		case Migration_REMOVE:
			if _, ok := s.Consumes[src]; ok {
				errs = append(errs, fmt.Errorf(
					"UID %q consumed multiple times in stage %q",
					src, s.Title,
				))
			}
			s.Consumes[src] = m
			s.Deletes[src] = true

		case Migration_CREATE:
			if _, ok := s.Produces[dst]; ok {
				errs = append(errs, fmt.Errorf(
					"UID %q produced multiple times in stage %q",
					dst, s.Title,
				))
			}
			s.Produces[dst] = m
			s.Creates[dst] = true
		}
	}

	return s, errs
}

func (s *StageSummary) LintStageConsumeProduce() []error {
	var errs []error

	for uid := range s.Consumes {
		if _, ok := s.Produces[uid]; ok {
			errs = append(errs, fmt.Errorf(
				"UID %q is both consumed and produced in stage %q (unordered execution, likely a rename chain)",
				uid, s.Title,
			))
		}
	}

	return errs
}

// Validate checks if a migration is correctly configured
func (m *Migration) Validate() []error {
	var errs []error

	switch m.Action {
	case Migration_REMOVE:
		if m.Source == nil {
			errs = append(errs, fmt.Errorf("REMOVE migrations must have source defined"))
		} else if m.Source.Uid == "" {
			errs = append(errs, fmt.Errorf("REMOVE migrations must have source.uid defined"))
		}
	case Migration_MODIFY:
		if m.Source == nil || m.Destination == nil {
			errs = append(errs, fmt.Errorf("MODIFY migrations must have both source and destination defined"))
		} else if m.Source.Uid == "" || m.Destination.Uid == "" {
			errs = append(errs, fmt.Errorf("MODIFY migrations must have both source.uid and destination.uid defined"))
		}
	case Migration_CREATE:
		if m.Destination == nil {
			errs = append(errs, fmt.Errorf("CREATE migrations must have destination defined"))
		} else if m.Destination.Uid == "" {
			errs = append(errs, fmt.Errorf("CREATE migrations must have destination.uid defined"))
		}
	default:
		errs = append(errs, fmt.Errorf("unknown migration action: %s", m.Action.String()))
	}

	return errs
}

// LintCrossMigrationStage validates the cross-stage migration requirements
// by walking stages backwards and ensuring that all produced UIDs are needed
// in the final state, and that all consumed UIDs are produced by some prior stage.
func LintCrossMigrationStage(
	stages []*StageSummary,
	final map[string]bool,
) []error {
	var errs []error

	// First pass: collect all UIDs that are consumed by any stage (for intermediate UID validation)
	consumedByLaterStages := make(map[string]bool)
	producedByLaterStages := make(map[string]bool)

	for i := 0; i < len(stages); i++ {
		stage := stages[i]
		for uid := range stage.Consumes {
			consumedByLaterStages[uid] = true
		}
		for uid := range stage.Produces {
			producedByLaterStages[uid] = true
		}
	}

	// Walk stages backwards
	for i := len(stages) - 1; i >= 0; i-- {
		stage := stages[i]

		// Track what UIDs are consumed/produced by stages AFTER this one
		laterConsumes := make(map[string]bool)
		laterProduces := make(map[string]bool)
		for j := i + 1; j < len(stages); j++ {
			for uid := range stages[j].Consumes {
				laterConsumes[uid] = true
			}
			for uid := range stages[j].Produces {
				laterProduces[uid] = true
			}
		}

		// 1. Handle productions (CREATE / MODIFY destination)
		// These UIDs are being created or are rename targets
		// Validate that produced UIDs are either:
		// - In final state, OR
		// - Consumed by a later stage (intermediate UID)
		for uid := range stage.Produces {
			if !final[uid] && !laterConsumes[uid] {
				errs = append(errs, fmt.Errorf(
					"stage %q produces UID %q which is not part of final state and not consumed by later stages",
					stage.Title, uid,
				))
			}
		}

		// 2. Handle consumptions (MODIFY source / REMOVE)
		// These are UIDs being renamed from or deleted
		for uid := range stage.Consumes {
			// Check if this consumed UID appears in the final state
			// If it does, that's an error UNLESS it's recreated by a later stage
			if final[uid] {
				if stage.Deletes[uid] {
					// Check if it's recreated by a later stage
					if !laterProduces[uid] {
						errs = append(errs, fmt.Errorf(
							"stage %q deletes UID %q which is still referenced in final state",
							stage.Title, uid,
						))
					}
				} else if stage.Renames[uid] != "" {
					// Check if the original UID is recreated by a later stage
					if !laterProduces[uid] {
						errs = append(errs, fmt.Errorf(
							"stage %q renames UID %q to %q, but %q is still referenced in final state",
							stage.Title, uid, stage.Renames[uid], uid,
						))
					}
				}
			}
		}
	}

	return errs
}
