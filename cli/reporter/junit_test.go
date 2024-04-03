// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/shared"
	"go.mondoo.com/cnspec/v10/policy"
	"testing"
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
					"+u6doYoYG5E=": &policy.Score{
						Type:  2, // result
						Value: 100,
					},
					"057itYF8s30=": &policy.Score{
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
	}
}

func TestJunitConverter(t *testing.T) {
	yr := sampleReportCollection()
	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}
	err := ConvertToJunit(yr, &writer)
	require.NoError(t, err)

	junitReport := buf.String()
	assert.Contains(t, junitReport, "name=\"Policy Report for X1\"")
	assert.Contains(t, junitReport, "<testcase name=\"Ensure SNMP server is stopped and not enabled\" classname=\"score\"></testcase>")
	assert.Contains(t, junitReport, "<testcase name=\"Configure kubelet to capture all event creation\" classname=\"score\">\n\t\t\t<failure message=\"\" type=\"error\"></failure>\n\t\t</testcase>")
	assert.Contains(t, junitReport, "<testcase name=\"Set secure file permissions on the scheduler.conf file\" classname=\"score\">\n\t\t\t<skipped message=\"skipped\"></skipped>\n\t\t</testcase>")
}

func TestJunitNilReport(t *testing.T) {
	var yr *policy.ReportCollection

	buf := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &buf}
	err := ConvertToJunit(yr, &writer)
	require.NoError(t, err)

	assert.Equal(t, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites></testsuites>\n", buf.String())
}
