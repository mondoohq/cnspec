// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import "go.mondoo.com/cnspec/v12/policy"

const (
	BundleMigrationsConfigurationValidationRuleID   = "bundle-migrations-configuration-validation"
	BundleMigrationsValidateStagesRuleID            = "bundle-migrations-validate-stages"
	BundleMigrationsValidateCrossStageProduceRuleID = "bundle-migrations-validate-cross-stage-produce"
	BundleMigrationsValidateGroupConditions         = "bundle-migrations-validate-group-conditions"
)

func GetBundleMigrationsLintRules() []LintRule {
	return []LintRule{
		{
			ID:          BundleMigrationsConfigurationValidationRuleID,
			Name:        "Migrations Configuration Validation",
			Description: "Ensures that migrations are defined correctly.",
			Severity:    LevelWarning,
			Run:         runRuleBundleMigrationsConfigurationValidation,
		},
		{
			ID:          BundleMigrationsValidateStagesRuleID,
			Name:        "Migrations Stages Validation",
			Description: "Validates the logical consistency of individual migration stages.",
			Severity:    LevelError,
			Run:         runRuleBundleMigrationsValidateStages,
		},
		{
			ID:          BundleMigrationsValidateCrossStageProduceRuleID,
			Name:        "Migrations Cross-Stage Produce Validation",
			Description: "Validates that produced UIDs in one stage are consumed in subsequent stages and matches queries in the bundle.",
			Severity:    LevelError,
			Run:         runRuleBundleMigrationsValidateCrossStageProduce,
		},
		{
			ID:          BundleMigrationsValidateGroupConditions,
			Name:        "Migrations Group Conditions Validation",
			Description: "Validates that migration groups have valid source and target policy conditions.",
			Severity:    LevelError,
			Run:         runRuleBundleMigrationsValidateGroupConditions,
		},
	}
}

func yacMigration2ProtoMigration(migration *Migration) *policy.Migration {
	res := &policy.Migration{
		Action: policy.Migration_Action(migration.Action),
	}
	if migration.Source != nil {
		res.Source = &policy.MigrationSource{
			Uid:    migration.Source.Uid,
			Sha256: migration.Source.Sha256,
		}
	}
	if migration.Target != nil {
		res.Target = &policy.MigrationTarget{
			Uid:    migration.Target.Uid,
			Sha256: migration.Target.Sha256,
		}
	}
	return res
}

func yacStage2ProtoStage(stage *MigrationStage) *policy.MigrationStage {
	res := &policy.MigrationStage{
		Title:           stage.Title,
		QueryMigrations: []*policy.Migration{},
	}
	for _, m := range stage.QueryMigrations {
		res.QueryMigrations = append(res.QueryMigrations, yacMigration2ProtoMigration(m))
	}
	return res
}

func yacConditions2ProtoConditions(conditions *MigrationConditions) *policy.MigrationConditions {
	if conditions == nil {
		return nil
	}
	res := &policy.MigrationConditions{}
	if conditions.SourcePolicy != nil {
		res.SourcePolicy = &policy.MigrationPolicyRef{
			Uid:     conditions.SourcePolicy.Uid,
			Version: conditions.SourcePolicy.Version,
		}
	}
	if conditions.TargetPolicy != nil {
		res.TargetPolicy = &policy.MigrationPolicyRef{
			Uid:     conditions.TargetPolicy.Uid,
			Version: conditions.TargetPolicy.Version,
		}
	}
	return res
}

func yacGroup2ProtoGroup(group *MigrationGroup) *policy.MigrationGroup {
	res := &policy.MigrationGroup{
		Title:      group.Title,
		Conditions: yacConditions2ProtoConditions(group.Conditions),
		Stages:     []*policy.MigrationStage{},
	}
	for _, s := range group.Stages {
		res.Stages = append(res.Stages, yacStage2ProtoStage(s))
	}
	return res
}

func runRuleBundleMigrationsConfigurationValidation(ctx *LintContext, item any) (res []*Entry) {
	bundle, ok := item.(*Bundle)
	if !ok {
		return nil
	}

	for _, group := range bundle.MigrationGroups {
		for _, stage := range group.Stages {
			for _, migration := range stage.QueryMigrations {
				protoMigration := yacMigration2ProtoMigration(migration)
				errs := protoMigration.Validate()

				for _, err := range errs {
					res = append(res, &Entry{
						RuleID:  BundleMigrationsConfigurationValidationRuleID,
						Level:   LevelError,
						Message: err.Error(),
						Location: []Location{{
							File:   ctx.FilePath,
							Line:   migration.FileContext.Line,
							Column: migration.FileContext.Column,
						}},
					})
				}
			}
		}
	}

	return res
}

func runRuleBundleMigrationsValidateStages(ctx *LintContext, item any) (res []*Entry) {
	bundle, ok := item.(*Bundle)
	if !ok {
		return nil
	}

	for _, group := range bundle.MigrationGroups {
		for stageIndex, stage := range group.Stages {
			summary, errs := policy.SummarizeStage(yacStage2ProtoStage(stage), stageIndex)
			errs = append(errs, summary.LintStageConsumeProduce()...)

			for _, err := range errs {
				res = append(res, &Entry{
					RuleID:  BundleMigrationsValidateStagesRuleID,
					Level:   LevelError,
					Message: err.Error(),
					Location: []Location{{
						File:   ctx.FilePath,
						Line:   stage.FileContext.Line,
						Column: stage.FileContext.Column,
					}},
				})
			}
		}
	}

	return res
}

func runRuleBundleMigrationsValidateCrossStageProduce(ctx *LintContext, item any) (res []*Entry) {
	bundle, ok := item.(*Bundle)
	if !ok {
		return nil
	}

	queries := make(map[string]bool)
	for _, query := range bundle.Queries {
		queries[query.Uid] = true
	}

	var stages []*policy.StageSummary
	for _, group := range bundle.MigrationGroups {
		if group.Conditions == nil || group.Conditions.TargetPolicy == nil {
			continue
		}
		found := false
		for _, query := range bundle.Policies {
			if query.Uid == group.Conditions.TargetPolicy.Uid && query.Version == group.Conditions.TargetPolicy.Version {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		for stageIndex, stage := range group.Stages {
			summary, _ := policy.SummarizeStage(yacStage2ProtoStage(stage), stageIndex)
			stages = append(stages, summary)
		}
	}

	errs := policy.LintCrossMigrationStage(stages, queries)
	for _, err := range errs {
		res = append(res, &Entry{
			RuleID:  BundleMigrationsValidateCrossStageProduceRuleID,
			Level:   LevelError,
			Message: err.Error(),
			Location: []Location{{
				File:   ctx.FilePath,
				Line:   1,
				Column: 1,
			}},
		})
	}

	return res
}

func runRuleBundleMigrationsValidateGroupConditions(ctx *LintContext, item any) (res []*Entry) {
	bundle, ok := item.(*Bundle)
	if !ok {
		return nil
	}

	for _, group := range bundle.MigrationGroups {
		errs := policy.LintMigrationGroupConditions(
			yacGroup2ProtoGroup(group),
		)
		for _, err := range errs {
			res = append(res, &Entry{
				RuleID:  BundleMigrationsValidateGroupConditions,
				Level:   LevelError,
				Message: err.Error(),
				Location: []Location{{
					File:   ctx.FilePath,
					Line:   group.FileContext.Line,
					Column: group.FileContext.Column,
				}},
			})
		}
	}

	return res
}
