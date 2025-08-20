// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/explorer"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v12/utils/iox"
	"go.mondoo.com/cnspec/v12/policy"
)

func sampleReportCollection() *policy.ReportCollection {
	return &policy.ReportCollection{
		Assets: map[string]*inventory.Asset{
			"//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2": {
				Name:        "X1",
				PlatformIds: []string{"//platformid.api.mondoo.app/hostname/X1"},
				State:       inventory.State_STATE_ONLINE,
				Platform: &inventory.Platform{
					Name:    "ubuntu",
					Arch:    "amd64",
					Kind:    "baremetal",
					Version: "22.04",
					Family:  []string{"debian", "linux", "unix", "os"},
				},
			},
		},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{
			"//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2": {
				CollectorJob: &policy.CollectorJob{
					ReportingQueries: map[string]*policy.StringArray{
						"+u6doYoYG5E=": nil,
						"057itYF8s30=": nil,
						"GyJVAziB/tU=": nil,
					},
				},
			},
		},
		Bundle: &policy.Bundle{
			Policies: nil, // not needed for this test since junit does not sort by policy
			Queries: []*explorer.Mquery{
				{
					Mrn:    "//policy.api.mondoo.app/queries/mondoo-linux-security-snmp-server-is-not-enabled",
					CodeId: "+u6doYoYG5E=",
					Title:  "Ensure SNMP server is stopped and not enabled",
				},
				{
					Mrn:    "//policy.api.mondoo.app/queries/mondoo-kubernetes-security-kubelet-event-record-qps",
					CodeId: "057itYF8s30=",
					Title:  "Configure kubelet to capture all event creation",
				},
				{
					Mrn:    "//policy.api.mondoo.app/queries/mondoo-kubernetes-security-secure-scheduler_conf",
					CodeId: "GyJVAziB/tU=",
					Title:  "Set secure file permissions on the scheduler.conf file",
				},
			},
		},
		Reports: map[string]*policy.Report{
			"//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2": {
				ScoringMrn: "//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2",
				EntityMrn:  "//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2",
				Score: &policy.Score{
					Value:           29,
					ScoreCompletion: 100,
					DataCompletion:  100,
				},
				// add passed, failed and skipped test
				Scores: map[string]*policy.Score{
					"+u6doYoYG5E=": {
						Type:  2, // result
						Value: 100,
					},
					"057itYF8s30=": {
						Type:  4, // error
						Value: 0,
					},
					"GyJVAziB/tU=": {
						Type:  8, // skip
						Value: 0,
					},
				},
			},
		},
		VulnReports: map[string]*mvd.VulnReport{
			"//assets.api.mondoo.app/spaces/dazzling-golick-767384/assets/2DRZ1cCWFyTYCArycAXHwvn1oU2": {
				Packages: []*mvd.Package{
					{
						Name:      "libssl1.1",
						Version:   "1.1.1f-3ubuntu2.19",
						Affected:  true,
						Score:     100,
						Available: "1.1.1f-3ubuntu2.20",
					},
				},
				Stats: &mvd.ReportStats{
					Packages: &mvd.ReportStatsPackages{
						Total:    1,
						Critical: 1,
						Affected: 1,
					},
				},
			},
		},
	}
}

func TestJunitConverter(t *testing.T) {
	yr := sampleReportCollection()
	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToJunit(yr, &writer)
	require.NoError(t, err)

	junitReport := buf.String()
	assert.Contains(t, junitReport, "name=\"Policy Report for X1\"")
	assert.Contains(t, junitReport, "<testcase name=\"Ensure SNMP server is stopped and not enabled\" classname=\"score\"></testcase>")
	assert.Contains(t, junitReport, "<testcase name=\"Configure kubelet to capture all event creation\" classname=\"score\">\n\t\t\t<failure message=\"\" type=\"error\"></failure>\n\t\t</testcase>")
	assert.Contains(t, junitReport, "<testcase name=\"Set secure file permissions on the scheduler.conf file\" classname=\"score\">\n\t\t\t<skipped message=\"skipped\"></skipped>\n\t\t</testcase>")
	assert.Contains(t, junitReport, "<testsuite name=\"Vulnerability Report for")
	assert.Contains(t, junitReport, "<property name=\"report.packages.total\" value=\"1\"></property>")
	assert.Contains(t, junitReport, "<property name=\"report.packages.critical\" value=\"1\"></property>")
	assert.Contains(t, junitReport, "<testcase name=\"libssl1.1\" classname=\"vulnerability\">")
	assert.Contains(t, junitReport, "<failure message=\"Update libssl1.1 to 1.1.1f-3ubuntu2.20\"><![CDATA[libssl1.1 with version1.1.1f-3ubuntu2.19 has known vulnerabilities (score 10)]]></failure>")
}

func TestJunitNilReport(t *testing.T) {
	var yr *policy.ReportCollection

	buf := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &buf}
	err := ConvertToJunit(yr, &writer)
	require.NoError(t, err)

	assert.Equal(t, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites></testsuites>\n", buf.String())
}
