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
	for _, check := range GetPolicyLintChecks() {
		rules = append(rules, Rule{
			ID:          check.ID,
			Name:        check.Name,
			Description: check.Description,
		})
	}

	// Add rules from registered query checks
	for _, check := range GetQueryLintChecks() {
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
	// Note: bundleInvalidUidRuleID is covered by specific policy/query UID checks.

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

	// create a run for tfsec
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
		// Validate and sanitize line and column values to prevent integer overflow
		// SARIF uses int values which must fit within a 32-bit signed integer range
		// to ensure compatibility with various SARIF parsers
		line := sanitizeLineColumn(l.Line)
		column := sanitizeLineColumn(l.Column)

		region := sarif.NewRegion().WithStartLine(line).WithStartColumn(column)
		loc := sarif.NewPhysicalLocation().WithArtifactLocation(artifactLocation(rootDir, l.File)).WithRegion(region)
		sarifLocs = append(sarifLocs, sarif.NewLocation().WithPhysicalLocation(loc))
	}

	return sarifLocs
}

// sanitizeLineColumn ensures that line/column values are valid for SARIF output.
// Invalid values (<=0 or outside safe integer range) are replaced with 1.
// This prevents integer overflow errors when the SARIF file is parsed.
//
// Background: The yaml.v3 parser can occasionally produce extremely large line/column
// numbers (e.g., 18446744073709552000, which exceeds 2^64) when parsing certain
// edge cases such as:
// - Files with Unicode/encoding issues
// - Extremely long lines or deeply nested structures
// - Parser bugs triggered by specific YAML constructs
//
// These invalid values would cause GitHub's SARIF parser to fail with:
// "strconv.ParseInt: parsing \"18446744073709552000\": value out of range"
func sanitizeLineColumn(value int) int {
	// Max safe value for SARIF integer fields
	// Using 2^31-1 (max 32-bit signed integer) for maximum compatibility.
	// This is a conservative choice because:
	// 1. Many SARIF parsers use 32-bit integers for line/column numbers
	// 2. JSON parsers in different languages have varying integer limits
	// 3. JavaScript (often used for SARIF processing) is only safe up to 2^53-1
	// 4. No real source file would ever have > 2 billion lines
	// 5. GitHub's SARIF parser has been observed to fail with values near 2^64
	const maxSafeValue = 2147483647 // 2^31 - 1 (2,147,483,647)

	if value <= 0 || value > maxSafeValue {
		return 1
	}
	return value
}
