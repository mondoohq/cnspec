// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ocsf/ocsf-toolkit/eventschema"
	"github.com/ocsf/ocsf-toolkit/jsonio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
)

// TestOcsfSchemaValidation runs every event the reporter emits through the OCSF
// project's own validator (github.com/ocsf/ocsf-toolkit) against the compiled
// schema of the version it claims in metadata.version.
//
// This is the gate that catches what a Go unit test cannot: attributes that do
// not exist in the class, values of the wrong type, enum siblings that disagree,
// missing required attributes, and profile attributes used without the profile
// being declared. It is why cnspec output is safe to hand to a data lake.
func TestOcsfSchemaValidation(t *testing.T) {
	reports := map[string]*policy.ReportCollection{
		"sample":     reportfixture.Sample(),
		"detailed":   reportfixture.Detailed(),
		"cloud":      cloudAssetReportCollection(),
		"advisories": advisoryReportCollection(),
		"scan error": erroredReportCollection(),
	}

	// Both finding classes are validated: a check is reported as one or the
	// other, so covering only the default would leave class 2004 unchecked.
	findingClasses := map[string]ocsf.FindingClasses{
		"compliance": ocsf.FindingsCompliance,
		"detection":  ocsf.FindingsDetection,
	}

	for _, version := range ocsf.SupportedVersions() {
		t.Run(version, func(t *testing.T) {
			pipeline := ocsfValidationPipeline(t, ocsf.Version(version))

			for name, report := range reports {
				for className, findings := range findingClasses {
					t.Run(name+"/"+className, func(t *testing.T) {
						validateOcsfEvents(t, pipeline, report, Options{
							Version:     ocsf.Version(version),
							Findings:    findings,
							IncludeData: true,
						}, name+" "+version+" "+className)
					})
				}
			}
		})
	}
}

// validateOcsfEvents converts a report and runs every event it produces through
// the OCSF validator, failing on errors and on warnings alike.
func validateOcsfEvents(t *testing.T, pipeline eventschema.EventProcessorPipeline, report *policy.ReportCollection, opts Options, label string) {
	events, err := convertAt(report, opts, fixedScanTime)
	require.NoError(t, err)
	require.NotZero(t, events.Len(), "the fixture must produce events to validate")

	buf := bytes.Buffer{}
	require.NoError(t, events.WriteJSON(&buf))

	for i, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		event, err := jsonio.DecodeObject(strings.NewReader(line))
		require.NoError(t, err, "event %d must be valid JSON", i)

		res, err := pipeline.ProcessEvent(event)
		require.NoError(t, err)

		for _, issue := range res.Validation.Errors {
			assert.Fail(t, "OCSF validation error",
				"event %d (%s): %s at %q", i, label, issue.Message, issue.AttributePath)
		}
		// Warnings cover deprecated attributes, which is how a version bump tells
		// us the mapping has to move on.
		for _, issue := range res.Validation.Warnings {
			assert.Fail(t, "OCSF validation warning",
				"event %d (%s): %s at %q", i, label, issue.Message, issue.AttributePath)
		}
	}
}

// ocsfValidationPipeline loads the compiled OCSF schema of a version and builds a
// validating processor from it. The schemas are checked in gzipped under
// reports/ocsf/schemas, where they are also the input the types are generated from; see
// the README there for how they are produced.
func ocsfValidationPipeline(t *testing.T, version ocsf.Version) eventschema.EventProcessorPipeline {
	t.Helper()

	compressed, err := os.Open(filepath.Join("..", "schemas", "schema-"+string(version)+".json.gz"))
	require.NoError(t, err, "every supported OCSF version needs a compiled schema in reports/ocsf/schemas")
	defer compressed.Close() //nolint: errcheck

	gz, err := gzip.NewReader(compressed)
	require.NoError(t, err)
	defer gz.Close() //nolint: errcheck

	// eventschema.New reads the schema from a file, so unpack it next to the test.
	path := filepath.Join(t.TempDir(), "schema-"+string(version)+".json")
	f, err := os.Create(path)
	require.NoError(t, err)
	_, err = io.Copy(f, gz)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	schema, err := eventschema.New(path)
	require.NoError(t, err)

	pipeline, err := schema.NewEventProcessorPipeline(eventschema.NewValidation())
	require.NoError(t, err)
	return pipeline
}
