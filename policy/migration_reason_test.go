// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
)

func TestMigrationReason_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		in   string
		want policy.Migration_Reason
	}{
		{`"renumbered"`, policy.Migration_REASON_RENUMBERED},
		{`"RENUMBERED"`, policy.Migration_REASON_RENUMBERED},
		{`"Retitled"`, policy.Migration_REASON_RETITLED},
		{`"removed_upstream"`, policy.Migration_REASON_REMOVED_UPSTREAM},
		{`"moved_to_manual"`, policy.Migration_REASON_MOVED_TO_MANUAL},
		{`"merged"`, policy.Migration_REASON_MERGED},
		{`"split"`, policy.Migration_REASON_SPLIT},
		{`"replaced"`, policy.Migration_REASON_REPLACED},
		{`"removed"`, policy.Migration_REASON_REMOVED},
		{`"temporary"`, policy.Migration_REASON_TEMPORARY},
		{`"REASON_MERGED"`, policy.Migration_REASON_MERGED},
		{`"reason_split"`, policy.Migration_REASON_SPLIT},
		{`"unspecified"`, policy.Migration_REASON_UNSPECIFIED},
		{`""`, policy.Migration_REASON_UNSPECIFIED},
		{`null`, policy.Migration_REASON_UNSPECIFIED},
		{`3`, policy.Migration_REASON_REMOVED_UPSTREAM},
		{`0`, policy.Migration_REASON_UNSPECIFIED},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			var r policy.Migration_Reason
			require.NoError(t, json.Unmarshal([]byte(c.in), &r))
			assert.Equal(t, c.want, r)
		})
	}

	for _, in := range []string{`"no-such-reason"`, `"remove"`, `42`, `-1`, `true`, `{}`} {
		t.Run("invalid "+in, func(t *testing.T) {
			var r policy.Migration_Reason
			assert.Error(t, json.Unmarshal([]byte(in), &r))
		})
	}
}

func TestMigrationReasonName(t *testing.T) {
	assert.Equal(t, "", policy.MigrationReasonName(policy.Migration_REASON_UNSPECIFIED))
	assert.Equal(t, "renumbered", policy.MigrationReasonName(policy.Migration_REASON_RENUMBERED))
	assert.Equal(t, "removed_upstream", policy.MigrationReasonName(policy.Migration_REASON_REMOVED_UPSTREAM))
	assert.Equal(t, "", policy.MigrationReasonName(policy.Migration_Reason(42)))

	// every named reason parses back from its short name
	for num := range policy.Migration_Reason_name {
		r := policy.Migration_Reason(num)
		if r == policy.Migration_REASON_UNSPECIFIED {
			continue
		}
		got, ok := policy.ParseMigrationReason(policy.MigrationReasonName(r))
		require.True(t, ok, r.String())
		assert.Equal(t, r, got)
	}
}

func TestBundleFromYAML_MigrationDescriptions(t *testing.T) {
	data := []byte(`
migration_groups:
- title: Example benchmark 1.0.0 to 2.0.0
  description: The upstream benchmark renumbered its controls in 2.0.0.
  refs:
  - title: Example benchmark 2.0.0 release notes
    url: https://example.com/benchmark/2.0.0
  stages:
  - title: Changes
    query_migrations:
    - action: REMOVE
      reason: removed_upstream
      description: Release 2.0.0 of the benchmark removed this recommendation.
      refs:
      - title: Benchmark ticket 1234
        url: https://example.com/tickets/1234
      source:
        uid: example-check-1
    - action: MODIFY
      reason: RENUMBERED
      source:
        uid: example-check-2
      target:
        uid: example-check-3
`)
	b, err := policy.BundleFromYAML(data)
	require.NoError(t, err)
	require.Len(t, b.MigrationGroups, 1)

	group := b.MigrationGroups[0]
	assert.Equal(t, "The upstream benchmark renumbered its controls in 2.0.0.", group.Description)
	require.Len(t, group.Refs, 1)
	assert.Equal(t, "https://example.com/benchmark/2.0.0", group.Refs[0].Url)

	migrations := group.Stages[0].QueryMigrations
	require.Len(t, migrations, 2)
	assert.Equal(t, policy.Migration_REMOVE, migrations[0].Action)
	assert.Equal(t, policy.Migration_REASON_REMOVED_UPSTREAM, migrations[0].Reason)
	assert.Equal(t, "Release 2.0.0 of the benchmark removed this recommendation.", migrations[0].Description)
	require.Len(t, migrations[0].Refs, 1)
	assert.Equal(t, "Benchmark ticket 1234", migrations[0].Refs[0].Title)
	assert.Equal(t, policy.Migration_MIGRATION_MODIFY, migrations[1].Action)
	assert.Equal(t, policy.Migration_REASON_RENUMBERED, migrations[1].Reason)

	// the new fields survive the binary wire format
	raw, err := b.MarshalVT()
	require.NoError(t, err)
	var decoded policy.Bundle
	require.NoError(t, decoded.UnmarshalVT(raw))
	got := decoded.MigrationGroups[0]
	assert.Equal(t, group.Description, got.Description)
	assert.Equal(t, group.Refs[0].Title, got.Refs[0].Title)
	assert.Equal(t, migrations[0].Reason, got.Stages[0].QueryMigrations[0].Reason)
	assert.Equal(t, migrations[0].Description, got.Stages[0].QueryMigrations[0].Description)
	assert.Equal(t, migrations[0].Refs[0].Url, got.Stages[0].QueryMigrations[0].Refs[0].Url)
}
