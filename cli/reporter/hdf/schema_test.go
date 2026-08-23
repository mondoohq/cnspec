// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package hdf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
	"go.mondoo.com/cnspec/cli/reporter/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// The OHDF/exec-json schema, checked in from
// https://github.com/mitre/heimdall2/blob/master/libs/inspecjs/schemas/exec-json.json
// so conformance is proven on every run rather than by hand. Refresh it when MITRE
// revises the schema; a failure here means our document stopped being readable by
// Heimdall and the SAF CLI.
const ohdfSchemaFile = "./testdata/ohdf-exec-json-schema.json"

func ohdfSchema(t *testing.T) *gojsonschema.Schema {
	t.Helper()
	abs, err := filepath.Abs(ohdfSchemaFile)
	require.NoError(t, err)

	schema, err := gojsonschema.NewSchema(gojsonschema.NewReferenceLoader("file://" + abs))
	require.NoError(t, err, "could not load the OHDF schema")
	return schema
}

// assertValidOHDF fails with the schema's own messages, which name the offending
// field, so a break points straight at the mapping that caused it.
func assertValidOHDF(t *testing.T, schema *gojsonschema.Schema, doc []byte, label string) {
	t.Helper()

	res, err := schema.Validate(gojsonschema.NewBytesLoader(doc))
	require.NoError(t, err, "%s: could not validate", label)
	if res.Valid() {
		return
	}

	var b strings.Builder
	for _, e := range res.Errors() {
		b.WriteString("\n  " + e.String())
	}
	t.Errorf("%s does not conform to the OHDF schema:%s", label, b.String())
}

// TestHDFConformsToSchema validates what the converter actually emits against the
// published OHDF schema, for the report shapes that exercise different parts of the
// mapping.
func TestHDFConformsToSchema(t *testing.T) {
	pinHDFClock(t)
	schema := ohdfSchema(t)

	errored := reportfixture.Sample()
	errored.Errors = map[string]string{reportfixture.AssetMrn: "could not connect to asset"}

	noVulns := reportfixture.Sample()
	noVulns.VulnReports = nil

	tests := []struct {
		name   string
		report *policy.ReportCollection
	}{
		{"empty scan", nil},
		{"checks and vulnerabilities", reportfixture.Sample()},
		{"checks only", noVulns},
		{"asset error", errored},
		{"recorded ubuntu scan", loadReportCollection(t, "../testdata/report-ubuntu.json")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertValidOHDF(t, schema, toHDFBytes(t, test.report), test.name)
		})
	}
}

// TestHDFDirConformsToSchema covers the multi-asset path: every file written into
// the directory has to be a valid OHDF document in its own right.
func TestHDFDirConformsToSchema(t *testing.T) {
	pinHDFClock(t)
	schema := ohdfSchema(t)

	dir := t.TempDir()
	files, err := ConvertToDir(multiAssetReportCollection(t), dir)
	require.NoError(t, err)
	require.Len(t, files, 2, "expected one file per asset")

	for _, path := range files {
		doc, err := os.ReadFile(path)
		require.NoError(t, err)
		assertValidOHDF(t, schema, doc, filepath.Base(path))
	}
}

// TestHDFSingleProfilePerDocument is the invariant behind the file-per-asset split.
// Heimdall and the SAF CLI resolve a document down to one root profile and tally
// only that one, so a second, unlinked profile is dropped without an error - the
// scan reads cleaner than it is. Every document must carry exactly one profile.
func TestHDFSingleProfilePerDocument(t *testing.T) {
	pinHDFClock(t)

	docs, err := hdfDocuments(multiAssetReportCollection(t))
	require.NoError(t, err)
	require.Len(t, docs, 2)

	for _, doc := range docs {
		assert.Len(t, doc.report.Profiles, 1,
			"asset %q: findings outside the first profile are dropped by OHDF consumers", doc.name)
	}

	// The vulnerability findings ride in that one profile rather than a second one.
	report := toHDF(t, reportfixture.Sample())
	require.Len(t, report.Profiles, 1)
	assert.NotNil(t, findHDFControl(report.Profiles[0], hdfVulnPackageID),
		"vulnerability controls must live in the asset's profile")

	// They stay distinguishable through their own control group.
	var vulnGroup *hdfGroup
	for _, group := range report.Profiles[0].Groups {
		if group.ID == hdfVulnGroupID {
			vulnGroup = group
		}
	}
	require.NotNil(t, vulnGroup, "expected a vulnerabilities group")
	assert.Contains(t, vulnGroup.Controls, hdfVulnPackageID)
}

// TestHDFProfileChecksumIsPerAsset is the regression for a fleet scan collapsing in
// Heimdall. Every asset scanned against the same policies resolves to the same
// execution graph, and the checksum used to be that graph alone - so a hundred
// documents, each with its own asset, its own controls and its own results, all
// claimed to be the same profile.
func TestHDFProfileChecksumIsPerAsset(t *testing.T) {
	pinHDFClock(t)

	r := multiAssetReportCollection(t)
	// what a real scan carries and the hand-built fixture does not: one resolved
	// policy shared by every asset it was scanned against
	for _, resolved := range r.ResolvedPolicies {
		resolved.GraphExecutionChecksum = "6+kRFHvaz8A="
	}

	docs, err := hdfDocuments(r)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	assert.NotEqual(t, docs[0].report.Profiles[0].Sha256, docs[1].report.Profiles[0].Sha256,
		"two assets scanned against one policy must not share a profile identity")

	// It still has to be stable: a profile checksum identifies the profile, not the
	// run, so re-scanning an asset must not look like a new profile because a check
	// changed its verdict.
	rerun := multiAssetReportCollectionWith(t, "6+kRFHvaz8A=")
	rerun.Reports[reportfixture.AssetMrn].Scores["+u6doYoYG5E="] =
		&policy.Score{Type: policy.ScoreType_Result, Value: 0}

	again, err := hdfDocuments(rerun)
	require.NoError(t, err)
	assert.Equal(t, docs[0].report.Profiles[0].Sha256, again[0].report.Profiles[0].Sha256)
}

// TestHDFDirWritesOneFilePerAsset covers the naming: asset names carry characters
// that do not belong in a path, and two assets can share one.
func TestHDFDirWritesOneFilePerAsset(t *testing.T) {
	pinHDFClock(t)

	r := multiAssetReportCollection(t)
	r.Assets[reportfixture.AssetMrn].Name = "docker.io/library/nginx:1.25"
	r.Assets[reportfixture.AssetMrn+"-two"].Name = "docker.io/library/nginx:1.25"

	dir := t.TempDir()
	files, err := ConvertToDir(r, dir)
	require.NoError(t, err)

	names := make([]string, 0, len(files))
	for _, path := range files {
		names = append(names, filepath.Base(path))
	}
	assert.Equal(t, []string{
		"docker.io-library-nginx-1.25.hdf.json",
		"docker.io-library-nginx-1.25-2.hdf.json",
	}, names, "colliding asset names must not overwrite each other")
}

// TestHDFFileBase covers the filename stem. Asset names are attacker-adjacent input
// in the sense that they come from whatever was scanned - a container image
// reference, a cloud resource ARN, a Kubernetes object - so a path separator must
// never survive into a filename, and a long one must not blow the filesystem's limit.
func TestHDFFileBase(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "web-02", "web-02"},
		{"image reference", "docker.io/library/nginx:1.25", "docker.io-library-nginx-1.25"},
		{"arn", "arn:aws:ec2:us-east-1:1:instance/i-0abc", "arn-aws-ec2-us-east-1-1-instance-i-0abc"},
		{"traversal", "../../etc/passwd", "etc-passwd"},
		{"absolute path", "/etc/shadow", "etc-shadow"},
		{"windows traversal", `..\..\windows\system32`, "windows-system32"},
		{"newlines", "name\nwith\nnewlines", "name-with-newlines"},
		{"non-ascii", "héllo wörld", "h-llo-w-rld"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := hdfFileBase(&hdfDocument{name: test.in, assetMrn: "//mrn/" + test.in})
			assert.Equal(t, test.want, got)
			assert.NotContains(t, got, "/")
			assert.NotContains(t, got, `\`)
		})
	}

	// Names that sanitize away entirely fall back to the MRN digest rather than to
	// something the filesystem would reject.
	for _, in := range []string{"", ".", "..", "....", "-", "\x00"} {
		got := hdfFileBase(&hdfDocument{name: in, assetMrn: "//mrn/x"})
		assert.Equal(t, "asset-"+hdfSha256("//mrn/x")[:12], got, "input %q", in)
	}

	// A long name is capped, and two that share a prefix stay distinct.
	longA := hdfFileBase(&hdfDocument{name: strings.Repeat("x", 300) + "-a", assetMrn: "//mrn/a"})
	longB := hdfFileBase(&hdfDocument{name: strings.Repeat("x", 300) + "-b", assetMrn: "//mrn/b"})
	assert.LessOrEqual(t, len(longA), hdfMaxFileBase)
	assert.NotEqual(t, longA, longB, "truncated names must not collide")
}

// TestHDFDirLongAssetName is the regression: a name past the filesystem's limit used
// to fail the write and abandon every asset after it, so one long name cost the
// whole export.
func TestHDFDirLongAssetName(t *testing.T) {
	pinHDFClock(t)

	r := multiAssetReportCollection(t)
	r.Assets[reportfixture.AssetMrn].Name = strings.Repeat("x", 300)

	dir := t.TempDir()
	files, err := ConvertToDir(r, dir)
	require.NoError(t, err)
	assert.Len(t, files, 2, "both assets must be written")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

// TestHDFDirReportsFailuresWithoutAbandoningTheRest covers the other half: when one
// asset genuinely cannot be written, the remaining assets still are, and the failure
// is reported rather than swallowed.
func TestHDFDirReportsFailuresWithoutAbandoningTheRest(t *testing.T) {
	pinHDFClock(t)

	dir := t.TempDir()
	// Occupy the path the first asset would write to, so os.Create fails on it.
	require.NoError(t, os.Mkdir(filepath.Join(dir, "X1.hdf.json"), 0o755))

	files, err := ConvertToDir(multiAssetReportCollection(t), dir)
	require.Error(t, err, "the unwritable asset must be reported")
	assert.Contains(t, err.Error(), "X1")

	require.Len(t, files, 1, "the other asset must still be written")
	assert.Equal(t, "web-02.hdf.json", filepath.Base(files[0]))
}

// TestHDFMultiAssetStream documents what a multi-asset scan looks like on stdout:
// an array of documents, since one OHDF document cannot describe several targets.
func TestHDFMultiAssetStream(t *testing.T) {
	pinHDFClock(t)

	var reports []*hdfReport
	require.NoError(t, json.Unmarshal(toHDFBytes(t, multiAssetReportCollection(t)), &reports))
	require.Len(t, reports, 2)

	for _, report := range reports {
		assert.Len(t, report.Profiles, 1)
	}
	// Each document names its own target rather than a shared placeholder.
	assert.NotEqual(t, reports[0].Platform.TargetID, reports[1].Platform.TargetID)
}

func loadReportCollection(t *testing.T, path string) *policy.ReportCollection {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var r policy.ReportCollection
	require.NoError(t, json.Unmarshal(raw, &r))
	return &r
}

// multiAssetReportCollection clones the sample scan onto a second asset running a
// different platform, so the multi-asset paths have something to work on.
func multiAssetReportCollection(t *testing.T) *policy.ReportCollection {
	t.Helper()

	r := reportfixture.Sample()
	secondMrn := reportfixture.AssetMrn + "-two"
	r.Assets[secondMrn] = &inventory.Asset{
		Name:        "web-02",
		PlatformIds: []string{"//platformid.api.mondoo.app/hostname/web-02"},
		Platform:    &inventory.Platform{Name: "debian", Version: "12", Arch: "amd64"},
	}
	r.ResolvedPolicies[secondMrn] = r.ResolvedPolicies[reportfixture.AssetMrn]

	scores := map[string]*policy.Score{}
	for id := range r.Reports[reportfixture.AssetMrn].Scores {
		scores[id] = &policy.Score{Type: policy.ScoreType_Result, Value: 100}
	}
	r.Reports[secondMrn] = &policy.Report{
		ScoringMrn: secondMrn,
		EntityMrn:  secondMrn,
		Scores:     scores,
		Score:      &policy.Score{Value: 100, ScoreCompletion: 100, DataCompletion: 100},
	}
	return r
}

// multiAssetReportCollectionWith is multiAssetReportCollection with the graph
// checksum a real scan would carry.
func multiAssetReportCollectionWith(t *testing.T, graphChecksum string) *policy.ReportCollection {
	t.Helper()

	r := multiAssetReportCollection(t)
	for _, resolved := range r.ResolvedPolicies {
		resolved.GraphExecutionChecksum = graphChecksum
	}
	return r
}
