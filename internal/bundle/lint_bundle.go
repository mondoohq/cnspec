// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

const (
	BundleGlobalPropsDeprecatedRuleID = "bundle-global-props-deprecated"
)

func GetBundleLintChecks() []LintCheck {
	return []LintCheck{
		{
			ID:          BundleGlobalPropsDeprecatedRuleID,
			Name:        "Policy Bundle Global Properties Deprecated",
			Description: "Checks if the policy bundle defines global properties",
			Severity:    LevelWarning,
			Run:         runCheckBundleGlobalPropsDeprecated,
		},
	}
}

func runCheckBundleGlobalPropsDeprecated(ctx *LintContext, item any) []*Entry {
	bundle, ok := item.(*Bundle)
	if !ok {
		return nil
	}

	if len(bundle.Props) == 0 {
		return nil
	}

	return []*Entry{{
		RuleID:  BundleGlobalPropsDeprecatedRuleID,
		Message: "Defining global properties in a policy bundle is deprecated. Define properties within individual policies and queries instead.",
		Level:   LevelError,
		Location: []Location{{
			File:   ctx.FilePath,
			Line:   bundle.Props[0].FileContext.Line,
			Column: bundle.Props[0].FileContext.Column,
		}},
	}}
}
