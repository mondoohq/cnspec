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

func (r *Results) sarifReport(rootDir string) (*sarif.Report, error) {
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
	for i := range rules {
		r := rules[i]
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
	report, err := r.sarifReport(rootDir)
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
