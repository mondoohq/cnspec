// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEvents() *Events {
	meta := Metadata{
		Version:    string(Version130),
		Product:    Product{Name: "cnspec", VendorName: "Mondoo", Version: "12.0.0"},
		LoggedTime: 1700000000000,
	}
	return &Events{
		ComplianceFindings: []ComplianceFinding{
			{
				base: base{
					ActivityID: ActivityCreate, ActivityName: "Create",
					CategoryUID: CategoryFindings, CategoryName: "Findings",
					ClassUID: ClassUIDComplianceFinding, ClassName: "Compliance Finding",
					TypeUID: ClassUIDComplianceFinding*100 + ActivityCreate,
					Time:    1700000000002,
					// Deliberately the higher severity, to prove the sort is on time.
					SeverityID: SeverityHigh, Severity: SeverityName(SeverityHigh),
					StatusID: StatusNew, Status: StatusName(StatusNew),
					Metadata: meta,
					Unmapped: map[string]string{"score": "0"},
				},
				Compliance: Compliance{
					Standards: []string{"cis-aws-foundations-benchmark"},
					Control:   "1.4",
					StatusID:  ComplianceStatusFail,
					Status:    ComplianceStatusName(ComplianceStatusFail),
				},
				FindingInfo: FindingInfo{UID: "check-b", Title: "Root account has no access key"},
				Resources:   []ResourceDetails{{UID: "//assets/1", Name: "prod", Labels: []string{"env=prod"}}},
			},
			{
				base: base{
					ActivityID: ActivityCreate, CategoryUID: CategoryFindings,
					ClassUID: ClassUIDComplianceFinding,
					TypeUID:  ClassUIDComplianceFinding*100 + ActivityCreate,
					Time:     1700000000001,
					Metadata: meta,
				},
				Compliance:  Compliance{Standards: []string{"mondoo"}, StatusID: ComplianceStatusPass},
				FindingInfo: FindingInfo{UID: "check-a", Title: "SSH root login disabled"},
			},
		},
		InventoryInfos: []InventoryInfo{
			{
				base: base{
					ActivityID: ActivityCollect, CategoryUID: CategoryDiscovery,
					ClassUID: ClassUIDInventoryInfo,
					TypeUID:  ClassUIDInventoryInfo*100 + ActivityCollect,
					Time:     1700000000000, SeverityID: SeverityInformational,
					Metadata: meta,
				},
				Device: Device{TypeID: DeviceTypeServer, UID: "//assets/1", Name: "prod", OS: &OS{Name: "Ubuntu", TypeID: OSTypeLinux}},
			},
		},
	}
}

func TestClassesAndSort(t *testing.T) {
	events := testEvents()
	assert.Equal(t, 3, events.Len())
	// vulnerability findings are empty, so they must not show up
	assert.Equal(t, []string{ClassComplianceFinding, ClassInventoryInfo}, events.Classes())

	events.Sort()
	assert.Equal(t, "check-a", events.ComplianceFindings[0].FindingInfo.UID,
		"events must be ordered by time, not by input order")
}

func TestWriteJSON(t *testing.T) {
	events := testEvents()
	events.Sort()

	buf := bytes.Buffer{}
	require.NoError(t, events.WriteJSON(&buf))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 3, "one JSON object per line")

	var first map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &first))

	// the embedded base must be flattened, not nested under "base"
	assert.EqualValues(t, ClassUIDComplianceFinding, first["class_uid"])
	assert.EqualValues(t, 200301, first["type_uid"])
	assert.NotContains(t, first, "base")
	// empty optional values are dropped
	assert.NotContains(t, first, "remediation")
	assert.NotContains(t, first, "device")
	// required objects are always there
	assert.Contains(t, first, "metadata")
	assert.Contains(t, first, "compliance")
	assert.Contains(t, first, "finding_info")

	var last map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[2]), &last))
	assert.EqualValues(t, ClassUIDInventoryInfo, last["class_uid"])
}

func TestWriteParquetRoundTrip(t *testing.T) {
	events := testEvents()
	events.Sort()

	buf := bytes.Buffer{}
	require.NoError(t, events.WriteParquetClass(ClassComplianceFinding, &buf))

	rows, err := parquet.Read[ComplianceFinding](bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "check-a", rows[0].FindingInfo.UID)
	assert.Equal(t, int64(1700000000001), rows[0].Time)
	assert.Equal(t, ClassUIDComplianceFinding, rows[1].ClassUID)
	assert.Equal(t, "Root account has no access key", rows[1].FindingInfo.Title)
	assert.Equal(t, []string{"cis-aws-foundations-benchmark"}, rows[1].Compliance.Standards)
	require.Len(t, rows[1].Resources, 1)
	assert.Equal(t, "//assets/1", rows[1].Resources[0].UID)
	assert.Equal(t, map[string]string{"score": "0"}, rows[1].Unmapped)
	assert.Nil(t, rows[0].Device, "an unset optional group must read back as nil")

	file, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.NotEmpty(t, file.Metadata().RowGroups)
	for _, col := range file.Metadata().RowGroups[0].Columns {
		assert.Equal(t, "ZSTD", col.MetaData.Codec.String(),
			"Security Lake prefers zstd for custom source objects")
	}
}

func TestWriteParquetEveryClass(t *testing.T) {
	events := testEvents()
	for _, class := range events.Classes() {
		buf := bytes.Buffer{}
		require.NoError(t, events.WriteParquetClass(class, &buf), class)
		assert.NotEmpty(t, buf.Bytes(), class)
	}

	require.Error(t, events.WriteParquetClass("nope", &bytes.Buffer{}))
	require.Error(t, events.WriteJSONClass("nope", &bytes.Buffer{}))
}

func TestParseVersion(t *testing.T) {
	v, err := ParseVersion("")
	require.NoError(t, err)
	assert.Equal(t, DefaultVersion, v)

	v, err = ParseVersion("1.9.0")
	require.NoError(t, err)
	assert.Equal(t, Version190, v)
	assert.True(t, v.AtLeast(Version130))
	assert.False(t, Version130.AtLeast(Version190))

	_, err = ParseVersion("1.2.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1.3.0, 1.9.0")
}
