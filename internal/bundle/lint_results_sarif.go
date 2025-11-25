// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

const (
	sarifError   = "error"
	sarifWarning = "warning"
	sarifNote    = "note"
	sarifNone    = "none"
)

// Rule is used for SARIF output and listing available rules.
// This structure matches the original LinterRules.
type Rule struct {
	ID          string
	Name        string
	Description string
}

// AllLinterRules dynamically generates the list of all linter rules
// from registered checks and adds bundle-specific static rules.
func sarifLinterRules() []Rule {
	var rules []Rule

	// Add rules from registered policy checks
	for _, check := range GetPolicyLintRules() {
		rules = append(rules, Rule{
			ID:          check.ID,
			Name:        check.Name,
			Description: check.Description,
		})
	}

	// Add rules from registered query checks
	for _, check := range GetQueryLintRules() {
		rules = append(rules, Rule{
			ID:          check.ID,
			Name:        check.Name,
			Description: check.Description,
		})
	}

	// Add bundle-level static rules (those not tied to a specific policy/query item)
	// These were previously in LinterRules and are typically checked outside the item-loop.
	rules = append(rules, Rule{
		ID:          BundleCompileErrorRuleID, // Define these constants
		Name:        "MQL compile error",
		Description: "Could not compile the MQL bundle",
	})
	rules = append(rules, Rule{
		ID:          BundleInvalidRuleID,
		Name:        "Invalid bundle",
		Description: "The bundle is not properly YAML formatted",
	})
	rules = append(rules, Rule{
		ID:          BundleUnknownFieldRuleID,
		Name:        "Bundle unknown field",
		Description: "The bundle YAML contains fields not defined in the schema",
	})
	rules = append(rules, Rule{
		ID:          BundleGlobalPropsDeprecatedRuleID,
		Name:        "Bundle global props deprecated",
		Description: "Global properties in policy bundles are deprecated",
	})
	// Add sub-rules that are used by policy/query checks but not registered as separate LintChecks
	rules = append(rules, Rule{
		ID:          BundleInvalidUidRuleID,
		Name:        "Invalid UID format",
		Description: "UID does not meet naming requirements",
	})
	rules = append(rules, Rule{
		ID:          PolicyUidUniqueRuleID,
		Name:        "Policy UID uniqueness",
		Description: "Policy UID must be unique within the file",
	})
	rules = append(rules, Rule{
		ID:          QueryUidUniqueRuleID,
		Name:        "Query UID uniqueness",
		Description: "Query UID must be unique within the file",
	})

	return rules
}

func (r *Results) SarifReport(rootDir string) (*sarif.Report, error) {
	linterRules := sarifLinterRules()
	absRootPath, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	// create a new report object
	report, err := sarif.New(sarif.Version210)
	if err != nil {
		return nil, err
	}

	// create a run for cnspec
	run := sarif.NewRunWithInformationURI("cnspec", "https://cnspec.io")

	// create a new rule for each rule id
	ruleIndex := map[string]int{}
	for i := range linterRules {
		r := linterRules[i]
		run.AddRule(r.ID).
			WithName(r.Name).
			WithDescription(r.Description)
		ruleIndex[r.ID] = i
	}

	// add the location as a unique artifact
	for i := range r.BundleLocations {
		artifact := run.AddArtifact()
		artifact.WithLocation(artifactLocation(absRootPath, r.BundleLocations[i]))
	}

	// add results for each entry
	for i := range r.Entries {
		e := r.Entries[i]
		result := sarif.NewRuleResult(e.RuleID).
			WithRuleIndex(ruleIndex[e.RuleID]).
			WithMessage(sarif.NewTextMessage(e.Message)).
			WithLevel(toSarifLevel(e.Level)).
			WithLocations(toSarifLocations(absRootPath, e.Location))
		run.AddResult(result)
	}

	// add the run to the report
	report.AddRun(run)

	return report, nil
}

func (r *Results) ToSarif(rootDir string) ([]byte, error) {
	report, err := r.SarifReport(rootDir)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	report.Write(&buf)
	return buf.Bytes(), nil
}

func toSarifLevel(level string) string {
	switch strings.ToUpper(level) {
	case "ERROR":
		return sarifError
	case "WARNING":
		return sarifWarning
	case "NOTE":
		return sarifNote
	default:
		return sarifNone
	}
}

func artifactLocation(rootDir string, filename string) *sarif.ArtifactLocation {
	if rootDir != "" {
		// if we have a root dir, we need to strip it from the filename
		relativePath, err := filepath.Rel(rootDir, filename)
		if err == nil {
			return sarif.NewArtifactLocation().WithUri(relativePath).WithUriBaseId("%SRCROOT%")
		}
		// if we can't get a relative path, just use the full path
	}

	if !strings.Contains(filename, "://") {
		filename = "file://" + filename
	}

	return sarif.NewSimpleArtifactLocation(filename)
}

func toSarifLocations(rootDir string, locations []Location) []*sarif.Location {
	sarifLocs := []*sarif.Location{}

	for i := range locations {
		l := locations[i]
		region := sarif.NewRegion().WithStartLine(l.Line).WithStartColumn(l.Column)
		loc := sarif.NewPhysicalLocation().WithArtifactLocation(artifactLocation(rootDir, l.File)).WithRegion(region)
		sarifLocs = append(sarifLocs, sarif.NewLocation().WithPhysicalLocation(loc))
	}

	return sarifLocs
}
