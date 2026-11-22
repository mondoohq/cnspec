// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tofex

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
)

func loadFixture(t *testing.T, name string) *ocsf.DetectionFinding {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	var f ocsf.DetectionFinding
	require.NoError(t, json.Unmarshal(raw, &f))
	return &f
}

func TestDetectionFindingFromJSON(t *testing.T) {
	f := loadFixture(t, "reverse-shell.json")

	got, err := DetectionFinding(f, Options{})
	require.NoError(t, err)

	assert.Equal(t, "8d2f7c1e-reverse-shell-host-1", got.Id)
	assert.Equal(t, "linux-reverse-shell-dup2", got.Ref, "the analytic groups findings of one detection")
	assert.Equal(t, "Reverse shell via dup2 to a socket", got.Summary)
	assert.Equal(t, "example-sensor", got.Source.GetName())
	assert.Equal(t, fex.Status_STATUS_AFFECTED, got.Status)
	assert.Nil(t, got.RemediatedAt)
	assert.Equal(t, int64(1763820000), got.FirstSeenAt.GetSeconds())
	assert.Equal(t, int64(1763820005), got.LastSeenAt.GetSeconds())

	d := got.Details
	assert.Equal(t, fex.FindingDetail_CATEGORY_THREAT, d.Category)
	assert.Equal(t, fex.SeverityRating_SEVERITY_RATING_HIGH, d.Severity.GetRating())
	assert.Equal(t, fex.Confidence_CONFIDENCE_HIGH, d.Confidence)
	assert.Equal(t, "A shell had its standard streams duplicated onto a TCP socket.", d.Description)
	assert.Equal(t, "TA0002,TA0011", d.Properties[PropertyTactics])
	assert.Equal(t, "T1059.004,T1071", d.Properties[PropertyTechniques])
	assert.Equal(t, "1.3.0", d.Properties["ocsf/metadata.version"])
	assert.Equal(t, "runtime", d.Properties["ocsf/unmapped.pipeline"])

	require.Len(t, got.Affects, 1)
	c := got.Affects[0].Component
	assert.Equal(t, "host-1-uid", c.Id)
	assert.Equal(t, "host-1", c.Identifiers["hostname"])
	assert.Equal(t, "Debian GNU/Linux", c.Identifiers["os-platform"])

	// One evidence per artifact, in the documented order.
	sources := make([]string, 0, len(got.Evidences))
	for _, ev := range got.Evidences {
		sources = append(sources, ev.Properties[PropertySource])
		assert.Equal(t, "TA0002", ev.GetTactic().GetId(), "every evidence carries the first tactic")
		assert.Equal(t, "T1059.004", ev.GetTechnique().GetId(), "the sub-technique is the more specific id")
		assert.Equal(t, fex.Confidence_CONFIDENCE_HIGH, ev.Confidence)
	}
	assert.Equal(t, []string{
		"evidences.0.actor.process",
		"evidences.0.process",
		"evidences.0.connection",
		"evidences.1.query",
		"evidences.1.container",
	}, sources)

	actor := got.Evidences[0].GetProcess()
	require.NotNil(t, actor)
	assert.Equal(t, "nginx: worker process", actor.Cmdline)

	p := got.Evidences[1].GetProcess()
	require.NotNil(t, p)
	assert.Equal(t, "/bin/sh -i", p.Cmdline)
	assert.Equal(t, int64(4242), p.Pid)
	assert.Equal(t, "/usr/bin/dash", p.Binary.GetPath())
	assert.Equal(t, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", p.Binary.GetSha256())
	assert.Equal(t, "www-data", p.User.GetName())
	assert.Equal(t, "33", p.User.GetId())
	require.NotNil(t, p.Parent)
	assert.Equal(t, int64(4100), p.Parent.Pid)
	assert.Equal(t, "/usr/sbin/nginx", p.Parent.Binary.GetPath())
	props := got.Evidences[1].Properties
	assert.Equal(t, "proc-4242", props["ocsf/process.uid"])
	assert.Equal(t, "/usr/sbin/nginx,/usr/lib/systemd/systemd", props["ocsf/process.lineage"])
	assert.Equal(t, "proc-4100", props["ocsf/process.parent_process.uid"])
	assert.Equal(t, "da39a3ee5e6b4b0d3255bfef95601890afd80709", props["ocsf/process.file.hashes.sha1"],
		"a hash fex has no field for is kept")

	conn := got.Evidences[2].GetConnection()
	require.NotNil(t, conn)
	assert.Equal(t, "10.0.0.5", conn.SourceAddress)
	assert.Equal(t, int32(51515), conn.SourcePort)
	assert.Equal(t, "203.0.113.7", conn.DestinationAddress)
	assert.Equal(t, int32(4444), conn.DestinationPort)
	assert.Equal(t, fex.Connection_TCP, conn.Protocol)
	assert.Equal(t, "2", got.Evidences[2].Properties["ocsf/connection_info.direction_id"])

	dns := got.Evidences[3].GetDnsRecord()
	require.NotNil(t, dns)
	assert.Equal(t, "c2.example.net", dns.Name)
	assert.Equal(t, "A", dns.Type)

	ctr := got.Evidences[4].GetContainer()
	require.NotNil(t, ctr)
	assert.Equal(t, "web", ctr.Name)
	assert.Equal(t, "nginx:1.27", ctr.ImageUri)
	assert.Equal(t, "sha256:abc", ctr.Digest)
}

func TestDetectionFindingRoundTripsThroughTheWriter(t *testing.T) {
	// A finding built with the generated constructor and written with the
	// package's own JSON writer converts the same as the hand-written fixture.
	f := ocsf.NewDetectionFinding(ocsf.DetectionFindingActivityCreate)
	f.Time = 1763820000000
	f.SeverityID = ocsf.SeverityCritical
	f.Metadata.Product.Name = "example-sensor"
	f.FindingInfo.UID = "f-1"
	f.FindingInfo.Title = "Credential file read"
	f.Evidences = []ocsf.Evidences{{File: &ocsf.File{Path: "/root/.aws/credentials"}}}

	raw, err := json.Marshal(f)
	require.NoError(t, err)
	var back ocsf.DetectionFinding
	require.NoError(t, json.Unmarshal(raw, &back))

	got, err := DetectionFinding(&back, Options{})
	require.NoError(t, err)
	assert.Equal(t, fex.SeverityRating_SEVERITY_RATING_CRITICAL, got.Details.Severity.GetRating())
	assert.Equal(t, "Credential file read", got.Ref, "without an analytic the title groups")
	require.Len(t, got.Evidences, 1)
	assert.Equal(t, "/root/.aws/credentials", got.Evidences[0].GetFile().GetPath())
	assert.Nil(t, got.Evidences[0].GetTactic())
	assert.Nil(t, got.Affects, "no device, no component")
}

func TestDetectionFindingOptions(t *testing.T) {
	f := loadFixture(t, "reverse-shell.json")
	affects := []*fex.Affects{{Component: &fex.Component{Id: "asset-1", Identifiers: map[string]string{"platform-id": "//example/asset-1"}}}}

	got, err := DetectionFinding(f, Options{
		Source:   "my-sensor",
		Category: fex.FindingDetail_CATEGORY_MALWARE,
		Affects:  affects,
	})
	require.NoError(t, err)
	assert.Equal(t, "my-sensor", got.Source.GetName())
	assert.Equal(t, "my-sensor", got.Details.Severity.GetSource().GetName())
	assert.Equal(t, fex.FindingDetail_CATEGORY_MALWARE, got.Details.Category)
	assert.Equal(t, affects, got.Affects)
}

func TestDetectionFindingStatus(t *testing.T) {
	for _, tc := range []struct {
		statusID int
		want     fex.Status
		closed   bool
	}{
		{ocsf.StatusUnknown, fex.Status_STATUS_AFFECTED, false},
		{ocsf.StatusNew, fex.Status_STATUS_AFFECTED, false},
		{ocsf.StatusInProgress, fex.Status_STATUS_AFFECTED, false},
		{ocsf.StatusOther, fex.Status_STATUS_AFFECTED, false},
		{ocsf.StatusSuppressed, fex.Status_STATUS_NOT_AFFECTED, true},
		{ocsf.StatusResolved, fex.Status_STATUS_FIXED, true},
		{statusArchived, fex.Status_STATUS_FIXED, true},
		{statusDeleted, fex.Status_STATUS_FIXED, true},
	} {
		t.Run(ocsf.StatusName(tc.statusID), func(t *testing.T) {
			f := loadFixture(t, "reverse-shell.json")
			f.StatusID = tc.statusID
			got, err := DetectionFinding(f, Options{})
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Status)
			assert.Equal(t, tc.closed, got.RemediatedAt != nil)
		})
	}
}

func TestDetectionFindingSeverity(t *testing.T) {
	for id, want := range map[int]fex.SeverityRating{
		ocsf.SeverityUnknown:       fex.SeverityRating_SEVERITY_RATING_UNSPECIFIED,
		ocsf.SeverityInformational: fex.SeverityRating_SEVERITY_RATING_NONE,
		ocsf.SeverityLow:           fex.SeverityRating_SEVERITY_RATING_LOW,
		ocsf.SeverityMedium:        fex.SeverityRating_SEVERITY_RATING_MEDIUM,
		ocsf.SeverityHigh:          fex.SeverityRating_SEVERITY_RATING_HIGH,
		ocsf.SeverityCritical:      fex.SeverityRating_SEVERITY_RATING_CRITICAL,
		ocsf.SeverityFatal:         fex.SeverityRating_SEVERITY_RATING_CRITICAL,
		ocsf.SeverityOther:         fex.SeverityRating_SEVERITY_RATING_UNSPECIFIED,
	} {
		assert.Equal(t, want, severityOf(id), ocsf.SeverityName(id))
	}
}

func TestDetectionFindingErrors(t *testing.T) {
	_, err := DetectionFinding(nil, Options{})
	assert.Error(t, err)

	f := loadFixture(t, "reverse-shell.json")
	f.FindingInfo.UID = ""
	_, err = DetectionFinding(f, Options{})
	assert.ErrorContains(t, err, "finding_info.uid")

	f = loadFixture(t, "reverse-shell.json")
	f.Metadata.Product.Name = ""
	_, err = DetectionFinding(f, Options{})
	assert.ErrorContains(t, err, "no source")

	f = loadFixture(t, "reverse-shell.json")
	f.ClassUID = ocsf.ClassUIDComplianceFinding
	_, err = DetectionFinding(f, Options{})
	assert.ErrorContains(t, err, "not a Detection Finding")
}
