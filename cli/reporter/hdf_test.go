// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/utils/iox"
)

// sampleAssetMrn is the asset sampleReportCollection reports on.
const sampleAssetMrn = "//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2"

// pinHDFClock freezes the timestamp the converter stamps on results that have no
// report time of their own, so a rendered document is comparable across runs.
func pinHDFClock(t *testing.T) {
	t.Helper()
	original := hdfTimeNow
	hdfTimeNow = func() time.Time { return time.Date(2026, 8, 22, 10, 30, 0, 0, time.UTC) }
	t.Cleanup(func() { hdfTimeNow = original })
}

// toHDFBytes runs the converter and returns the raw document.
func toHDFBytes(t *testing.T, r *policy.ReportCollection) []byte {
	t.Helper()
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	require.NoError(t, ConvertToHDF(r, &writer))
	return buf.Bytes()
}

// toHDF runs the converter and parses the result back into the OHDF types.
func toHDF(t *testing.T, r *policy.ReportCollection) *hdfReport {
	t.Helper()
	var report hdfReport
	require.NoError(t, json.Unmarshal(toHDFBytes(t, r), &report))
	return &report
}

func findHDFProfile(report *hdfReport, name string) *hdfProfile {
	for _, profile := range report.Profiles {
		if profile.Name == name {
			return profile
		}
	}
	return nil
}

func findHDFControl(profile *hdfProfile, id string) *hdfControl {
	if profile == nil {
		return nil
	}
	for _, control := range profile.Controls {
		if control.ID == id {
			return control
		}
	}
	return nil
}

func TestHDFConverter(t *testing.T) {
	pinHDFClock(t)
	report := toHDF(t, sampleReportCollection())

	// A single-asset scan reports that asset as the platform.
	assert.Equal(t, "ubuntu", report.Platform.Name)
	assert.Equal(t, "22.04", report.Platform.Release)
	assert.Equal(t, "//platformid.api.mondoo.app/hostname/X1", report.Platform.TargetID)
	assert.NotEmpty(t, report.Version)

	// One profile, named after the asset, holding its checks and its vulnerabilities.
	require.Len(t, report.Profiles, 1)
	checks := findHDFProfile(report, "X1")
	require.NotNil(t, checks, "expected a profile named after the asset")

	// The profile carries the fields OHDF requires.
	for _, profile := range report.Profiles {
		assert.NotEmpty(t, profile.Name)
		assert.NotEmpty(t, profile.Sha256, "profile %q needs a checksum", profile.Name)
		assert.NotNil(t, profile.Supports)
		assert.NotNil(t, profile.Attributes)
		assert.NotNil(t, profile.Groups)
		assert.NotNil(t, profile.Controls)
	}

	assert.Equal(t, []map[string]string{{"platform-name": "ubuntu", "release": "22.04"}}, checks.Supports)

	// three checks plus the vulnerability finding
	require.Len(t, checks.Controls, 4)
	for _, control := range checks.Controls {
		assert.NotEmpty(t, control.ID)
		require.Len(t, control.Results, 1, "control %q", control.ID)
		assert.Equal(t, "2026-08-22T10:30:00Z", control.Results[0].StartTime)
	}
}

// TestHDFStatusMapping pins the score type to result status mapping, including the
// impact of 0 that makes Heimdall render a check as "Not Applicable".
func TestHDFStatusMapping(t *testing.T) {
	pinHDFClock(t)
	report := toHDF(t, sampleReportCollection())
	checks := findHDFProfile(report, "X1")
	require.NotNil(t, checks)

	passed := findHDFControl(checks, "mondoo-linux-security-snmp-server-is-not-enabled")
	require.NotNil(t, passed)
	assert.Equal(t, hdfStatusPassed, passed.Results[0].Status)
	assert.Greater(t, passed.Impact, 0.0, "a passing check must not read as Not Applicable")

	errored := findHDFControl(checks, "mondoo-kubernetes-security-kubelet-event-record-qps")
	require.NotNil(t, errored)
	assert.Equal(t, hdfStatusError, errored.Results[0].Status)

	skipped := findHDFControl(checks, "mondoo-kubernetes-security-secure-scheduler_conf")
	require.NotNil(t, skipped)
	assert.Equal(t, hdfStatusSkipped, skipped.Results[0].Status)
	assert.Equal(t, 0.0, skipped.Impact, "a skipped check is Not Applicable in Heimdall")
	assert.NotEmpty(t, skipped.Results[0].SkipMessage)
}

func TestHDFStatus(t *testing.T) {
	tests := []struct {
		name  string
		score *policy.Score
		want  string
	}{
		{"nil score", nil, hdfStatusSkipped},
		{"passed", &policy.Score{Type: policy.ScoreType_Result, Value: 100}, hdfStatusPassed},
		{"failed", &policy.Score{Type: policy.ScoreType_Result, Value: 40}, hdfStatusFailed},
		{"error", &policy.Score{Type: policy.ScoreType_Error}, hdfStatusError},
		{"skip", &policy.Score{Type: policy.ScoreType_Skip}, hdfStatusSkipped},
		{"unscored", &policy.Score{Type: policy.ScoreType_Unscored}, hdfStatusSkipped},
		{"out of scope", &policy.Score{Type: policy.ScoreType_OutOfScope}, hdfStatusSkipped},
		{"disabled", &policy.Score{Type: policy.ScoreType_Disabled}, hdfStatusSkipped},
		{"unknown", &policy.Score{Type: policy.ScoreType_Unknown}, hdfStatusSkipped},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, hdfStatus(test.score))
		})
	}
}

// TestHDFControlImpact checks that the cnspec severity bands survive the trip onto
// the OHDF 0.0-1.0 scale: cnspec calls impact 70 HIGH and Heimdall calls 0.7 high,
// so a converted value has to land in the same bucket it started in.
func TestHDFControlImpact(t *testing.T) {
	impactQuery := func(impact int32) *policy.Mquery {
		return &policy.Mquery{Impact: &policy.Impact{Value: &policy.ImpactValue{Value: impact}}}
	}
	result := func(value uint32) *policy.Score {
		return &policy.Score{Type: policy.ScoreType_Result, Value: value}
	}

	tests := []struct {
		name  string
		query *policy.Mquery
		score *policy.Score
		want  float64
	}{
		{"critical", impactQuery(100), result(0), 1},
		{"critical band", impactQuery(90), result(0), 0.9},
		{"high band", impactQuery(70), result(0), 0.7},
		{"medium band", impactQuery(40), result(0), 0.4},
		{"low band", impactQuery(10), result(0), 0.1},
		{"no severity is floored, never Not Applicable", impactQuery(0), result(0), hdfMinImpact},
		{"unset impact falls back to the score", &policy.Mquery{}, result(20), 0.8},
		{"unset impact on a passing check is floored", &policy.Mquery{}, result(100), hdfMinImpact},
		{
			"an errored check with no severity is not reported as critical",
			&policy.Mquery{},
			&policy.Score{Type: policy.ScoreType_Error},
			hdfMinImpact,
		},
		{
			"skipped is Not Applicable",
			impactQuery(90),
			&policy.Score{Type: policy.ScoreType_Skip},
			0,
		},
		{
			"disabled is Not Applicable",
			impactQuery(90),
			&policy.Score{Type: policy.ScoreType_Disabled},
			0,
		},
		{
			"unscored keeps its severity so it reads as Not Reviewed",
			impactQuery(90),
			&policy.Score{Type: policy.ScoreType_Unscored},
			0.9,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.InDelta(t, test.want, hdfControlImpact(test.query, test.score), 1e-9)
		})
	}
}

// TestHDFSeverityLabel pins the bands Heimdall and the SAF CLI bucket findings by.
func TestHDFSeverityLabel(t *testing.T) {
	tests := []struct {
		impact float64
		want   string
	}{
		{1, "critical"},
		{0.9, "critical"},
		{0.89, "high"},
		{0.7, "high"},
		{0.69, "medium"},
		{0.4, "medium"},
		{0.39, "low"},
		{0.1, "low"},
		{0.09, "none"},
		{0, "none"},
	}

	for _, test := range tests {
		assert.Equal(t, test.want, hdfSeverityLabel(test.impact), "impact %v", test.impact)
	}
}

// TestHDFUnratedCheckStaysCounted guards the finding that a check with no configured
// severity must not fall into the "none" band. The SAF CLI's summary buckets a
// finding by its severity tag and has no counter for "none", so an unrated check
// that errored would vanish from the tally - the scan would look cleaner than it is.
func TestHDFUnratedCheckStaysCounted(t *testing.T) {
	pinHDFClock(t)

	r := sampleReportCollection()
	for _, query := range r.Bundle.Queries {
		query.Impact = nil
	}
	r.Reports[sampleAssetMrn].Scores["057itYF8s30="] = &policy.Score{Type: policy.ScoreType_Error}

	report := toHDF(t, r)
	control := findHDFControl(findHDFProfile(report, "X1"), "mondoo-kubernetes-security-kubelet-event-record-qps")
	require.NotNil(t, control)

	assert.Equal(t, hdfStatusError, control.Results[0].Status)
	assert.GreaterOrEqual(t, control.Impact, hdfMinImpact)
	assert.NotEqual(t, "none", control.Tags["severity"],
		"an unrated finding tagged \"none\" is dropped from the SAF summary")
}

// TestHDFNistTags pins the transform that makes cnspec's compliance mapping legible
// to Heimdall's NIST 800-53 views.
func TestHDFNistTags(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
		want []string
	}{
		{"no tags at all", nil, []string{hdfNistUnmappedTag}},
		{
			"mapped control",
			map[string]string{hdfNistTagKey: "nist-sp-800-53-rev5-si-7"},
			[]string{"SI-7"},
		},
		{
			"control enhancement",
			map[string]string{hdfNistTagKey: "nist-sp-800-53-rev5-ac-2-1"},
			[]string{"AC-2 (1)"},
		},
		{
			"multi-digit control",
			map[string]string{hdfNistTagKey: "nist-sp-800-53-rev5-cm-14"},
			[]string{"CM-14"},
		},
		{
			"explicitly unmapped",
			map[string]string{hdfNistTagKey: "false"},
			[]string{hdfNistUnmappedTag},
		},
		{
			"other frameworks do not leak in",
			map[string]string{"compliance/iso-27001-2022": "iso-27001-2022-a-8-8"},
			[]string{hdfNistUnmappedTag},
		},
		{
			"unrecognized shape",
			map[string]string{hdfNistTagKey: "nist-sp-800-53-rev5-whatever"},
			[]string{hdfNistUnmappedTag},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, hdfNistTags(&policy.Mquery{Tags: test.tags}))
		})
	}
}

// TestHDFCheckTags verifies that the compliance mappings and cnspec context ride
// along on every control.
func TestHDFCheckTags(t *testing.T) {
	pinHDFClock(t)

	r := sampleReportCollection()
	r.Bundle.Queries[0].Tags = map[string]string{
		hdfNistTagKey:               "nist-sp-800-53-rev5-si-7",
		"compliance/iso-27001-2022": "iso-27001-2022-a-8-8",
		"compliance/hipaa":          "false",
	}
	r.Bundle.Queries[0].Impact = &policy.Impact{Value: &policy.ImpactValue{Value: 80}}

	report := toHDF(t, r)
	control := findHDFControl(findHDFProfile(report, "X1"), "mondoo-linux-security-snmp-server-is-not-enabled")
	require.NotNil(t, control)

	assert.Equal(t, []any{"SI-7"}, control.Tags["nist"])
	assert.Equal(t, "high", control.Tags["severity"])
	assert.Equal(t, "iso-27001-2022-a-8-8", control.Tags["compliance/iso-27001-2022"])
	assert.NotContains(t, control.Tags, "compliance/hipaa", "mappings turned off must be dropped")
	assert.Equal(t, "X1", control.Tags["asset"])
	assert.Equal(t, r.Bundle.Queries[0].Mrn, control.Tags["queryMrn"])
}

// TestHDFPolicyGroups checks that the policy a control came from survives, even
// though the profile is the asset rather than the policy.
func TestHDFPolicyGroups(t *testing.T) {
	pinHDFClock(t)

	r := sampleReportCollection()
	r.Bundle.Policies = []*policy.Policy{{
		Mrn:  "//policy.api.mondoo.app/policies/mondoo-linux-security",
		Name: "Linux Security by Mondoo",
		Groups: []*policy.PolicyGroup{{
			Checks: []*policy.Mquery{
				{Mrn: r.Bundle.Queries[0].Mrn},
				// a check of the policy that did not report for this asset
				{Mrn: "//policy.api.mondoo.app/queries/not-in-this-scan"},
			},
		}},
	}}

	report := toHDF(t, r)
	checks := findHDFProfile(report, "X1")
	require.NotNil(t, checks)

	var policyGroup *hdfGroup
	for _, group := range checks.Groups {
		if group.ID == "//policy.api.mondoo.app/policies/mondoo-linux-security" {
			policyGroup = group
		}
	}
	require.NotNil(t, policyGroup, "expected a group for the policy")
	require.NotNil(t, policyGroup.Title)
	assert.Equal(t, "Linux Security by Mondoo", *policyGroup.Title)
	assert.Equal(t, []string{"mondoo-linux-security-snmp-server-is-not-enabled"}, policyGroup.Controls)

	// The policy is attributed on the control itself as well.
	control := findHDFControl(checks, "mondoo-linux-security-snmp-server-is-not-enabled")
	require.NotNil(t, control)
	assert.Equal(t, []any{"Linux Security by Mondoo"}, control.Tags["policies"])
}

// TestHDFVulnerabilities checks the advisory findings that ride in the asset's
// profile.
func TestHDFVulnerabilities(t *testing.T) {
	pinHDFClock(t)
	report := toHDF(t, sampleReportCollection())

	// The sample report has an affected package but no advisory covering it, so it
	// lands in the catch-all control.
	control := findHDFControl(findHDFProfile(report, "X1"), hdfVulnPackageID)
	require.NotNil(t, control)
	assert.Equal(t, 1.0, control.Impact)
	assert.Equal(t, "critical", control.Tags["severity"])
	assert.Equal(t, []any{"SI-2", "RA-5"}, control.Tags["nist"])

	require.Len(t, control.Results, 1)
	assert.Equal(t, hdfStatusFailed, control.Results[0].Status)
	assert.Equal(t, "libssl1.1 1.1.1f-3ubuntu2.19", control.Results[0].CodeDesc)
	assert.Contains(t, control.Results[0].Message, "Update to 1.1.1f-3ubuntu2.20.")
}

// TestHDFAssetError makes sure an asset that could not be scanned reports as a
// failure rather than as an empty, apparently clean profile.
func TestHDFAssetError(t *testing.T) {
	pinHDFClock(t)

	r := sampleReportCollection()
	r.Errors = map[string]string{sampleAssetMrn: "could not connect to asset"}

	report := toHDF(t, r)
	control := findHDFControl(findHDFProfile(report, "X1"), hdfAssetErrorID)
	require.NotNil(t, control, "expected an asset-error control")
	assert.Equal(t, 1.0, control.Impact)
	require.Len(t, control.Results, 1)
	assert.Equal(t, hdfStatusError, control.Results[0].Status)
	assert.Equal(t, "could not connect to asset", control.Results[0].Message)

	// The failure is also visible in the passthrough data.
	require.NotNil(t, report.Passthrough)
	require.Len(t, report.Passthrough.AuxiliaryData, 1)
}

// TestHDFDuplicateAssetNames covers assets that share a display name - several
// containers of one image. They must still come out as separate documents; the
// filenames they are written to are covered by TestHDFDirWritesOneFilePerAsset.
func TestHDFDuplicateAssetNames(t *testing.T) {
	pinHDFClock(t)

	r := multiAssetReportCollection(t)
	r.Assets[sampleAssetMrn+"-two"].Name = "X1"

	docs, err := hdfDocuments(r)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	assert.NotEqual(t, docs[0].assetMrn, docs[1].assetMrn)

	// Each document names its own target rather than a shared placeholder.
	assert.Equal(t, "ubuntu", docs[0].report.Platform.Name)
	assert.Equal(t, "debian", docs[1].report.Platform.Name)
}

// TestHDFDeterministic guards against map iteration leaking into the output: two
// conversions of the same report must be byte-identical, or every re-scan looks
// like a change to whatever consumes the document.
func TestHDFDeterministic(t *testing.T) {
	pinHDFClock(t)

	r := sampleReportCollection()
	first := toHDFBytes(t, r)
	for i := 0; i < 5; i++ {
		assert.Equal(t, string(first), string(toHDFBytes(t, r)))
	}
}

func TestHDFEmptyReport(t *testing.T) {
	pinHDFClock(t)

	report := toHDF(t, nil)
	assert.Equal(t, hdfToolName, report.Platform.Name)
	assert.Empty(t, report.Profiles)
	assert.Nil(t, report.Statistics.Duration)
}

func TestHDFMissingBundle(t *testing.T) {
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToHDF(&policy.ReportCollection{}, &writer)
	assert.ErrorContains(t, err, "no policy bundle found")
}

// TestHDFUbuntuReport runs the converter over a full recorded scan and checks the
// shape of the document rather than individual values.
func TestHDFUbuntuReport(t *testing.T) {
	pinHDFClock(t)

	raw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)

	var r policy.ReportCollection
	require.NoError(t, json.Unmarshal(raw, &r))

	report := toHDF(t, &r)
	require.NotEmpty(t, report.Profiles)

	var controls int
	for _, profile := range report.Profiles {
		assert.NotEmpty(t, profile.Sha256)
		for _, control := range profile.Controls {
			controls++
			assert.NotEmpty(t, control.ID)
			assert.GreaterOrEqual(t, control.Impact, 0.0)
			assert.LessOrEqual(t, control.Impact, 1.0)
			assert.NotEmpty(t, control.Tags["nist"], "control %q needs a NIST tag", control.ID)
			// The severity tag is what the SAF CLI buckets by, so it has to agree
			// with the impact rather than being derived separately.
			assert.Equal(t, hdfSeverityLabel(control.Impact), control.Tags["severity"],
				"control %q: severity tag and impact disagree", control.ID)
			require.NotEmpty(t, control.Results, "control %q needs a result", control.ID)
			if control.Impact > 0 {
				assert.NotEqual(t, "none", control.Tags["severity"],
					"control %q would drop out of the SAF summary", control.ID)
			}
			for _, result := range control.Results {
				assert.Contains(t,
					[]string{hdfStatusPassed, hdfStatusFailed, hdfStatusSkipped, hdfStatusError},
					result.Status)
				assert.NotEmpty(t, result.CodeDesc)
				assert.NotEmpty(t, result.StartTime)
			}
		}
	}
	assert.NotZero(t, controls, "expected the recorded scan to produce controls")
}

// TestHDFReporterFormat checks the format is reachable through the reporter, the
// way `cnspec scan -o hdf` reaches it.
func TestHDFReporterFormat(t *testing.T) {
	pinHDFClock(t)

	conf, err := ParseConfig("hdf")
	require.NoError(t, err)
	assert.Equal(t, FormatHDF, conf.format)

	buf := bytes.Buffer{}
	require.NoError(t, NewReporter(conf, false).WithOutput(&buf).
		WriteReport(t.Context(), sampleReportCollection()))

	var report hdfReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	assert.NotEmpty(t, report.Profiles)
}
