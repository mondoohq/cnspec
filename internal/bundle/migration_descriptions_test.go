// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
)

// A migration can record why it exists: a reason, a description and
// references, on each migration and on its group. Bundles that carry them must
// lint cleanly (in particular without a bundle-unknown-field entry) and keep
// them through cnspec policy fmt.
func TestMigrationDescriptions(t *testing.T) {
	file := "./testdata/pass-migration-descriptions.mql.yaml"

	t.Run("lint", func(t *testing.T) {
		results, err := Lint(schema, testLintOptions, file)
		require.NoError(t, err)
		for _, entry := range results.Entries {
			t.Logf("entry: %s %s", entry.RuleID, entry.Message)
		}
		assert.Empty(t, results.Entries)
		assert.False(t, results.HasError())
	})

	assertDescriptions := func(t *testing.T, b *Bundle) {
		t.Helper()
		require.Len(t, b.MigrationGroups, 1)
		group := b.MigrationGroups[0]
		assert.Equal(t, "Release 2.0.0 of the benchmark renumbered one control and removed another.", group.Description)
		require.Len(t, group.Refs, 1)
		assert.Equal(t, "Example Benchmark 2.0.0 release notes", group.Refs[0].Title)
		assert.Equal(t, "https://example.com/benchmark/2.0.0/release-notes", group.Refs[0].Url)

		require.Len(t, group.Stages, 1)
		migrations := group.Stages[0].QueryMigrations
		require.Len(t, migrations, 2)

		assert.Equal(t, Migration_Action(policy.Migration_MIGRATION_MODIFY), migrations[0].Action)
		assert.Equal(t, Migration_Reason(policy.Migration_REASON_RENUMBERED), migrations[0].Reason)
		assert.Equal(t, "The control moved from 1.1 to 1.3 without other changes.", migrations[0].Description)
		assert.Empty(t, migrations[0].Refs)

		assert.Equal(t, Migration_Action(policy.Migration_REMOVE), migrations[1].Action)
		assert.Equal(t, Migration_Reason(policy.Migration_REASON_REMOVED_UPSTREAM), migrations[1].Reason)
		assert.Equal(t, "Release 2.0.0 of the benchmark removed this recommendation.", migrations[1].Description)
		require.Len(t, migrations[1].Refs, 1)
		assert.Equal(t, "Benchmark ticket 1234", migrations[1].Refs[0].Title)
		assert.Equal(t, "https://example.com/tickets/1234", migrations[1].Refs[0].Url)

		// the lint rules see the same values
		proto := yacGroup2ProtoGroup(group)
		assert.Equal(t, group.Description, proto.Description)
		require.Len(t, proto.Refs, 1)
		assert.Equal(t, group.Refs[0].Url, proto.Refs[0].Url)
		protoMigration := yacMigration2ProtoMigration(migrations[1])
		assert.Equal(t, policy.Migration_REASON_REMOVED_UPSTREAM, protoMigration.Reason)
		assert.Equal(t, migrations[1].Description, protoMigration.Description)
		require.Len(t, protoMigration.Refs, 1)
		assert.Equal(t, "Benchmark ticket 1234", protoMigration.Refs[0].Title)
	}

	t.Run("fmt round-trip", func(t *testing.T) {
		data, err := os.ReadFile(file)
		require.NoError(t, err)

		b, err := ParseYaml(data)
		require.NoError(t, err)
		assertDescriptions(t, b)

		formatted, err := FormatBundle(b, true)
		require.NoError(t, err)
		assert.Contains(t, string(formatted), "reason: renumbered\n")
		assert.Contains(t, string(formatted), "reason: removed_upstream\n")

		reparsed, err := ParseYaml(formatted)
		require.NoError(t, err)
		assertDescriptions(t, reparsed)

		// formatting is stable
		again, err := FormatBundle(reparsed, true)
		require.NoError(t, err)
		assert.Equal(t, string(formatted), string(again))

		// and the formatted bundle still lints cleanly
		entries := LintPolicyBundle(schema, "formatted.mql.yaml", formatted, testLintOptions)
		for _, entry := range entries {
			t.Logf("entry: %s %s", entry.RuleID, entry.Message)
		}
		assert.Empty(t, entries)
	})

	t.Run("an unknown reason is reported", func(t *testing.T) {
		_, err := ParseYaml([]byte(`
migration_groups:
- stages:
  - query_migrations:
    - action: REMOVE
      reason: no-such-reason
      source:
        uid: example-check
`))
		assert.Error(t, err)
	})

	t.Run("an unset reason is not written", func(t *testing.T) {
		b := &Bundle{MigrationGroups: []*MigrationGroup{{
			Title: "Unexplained removal",
			Stages: []*MigrationStage{{QueryMigrations: []*Migration{{
				Action: Migration_Action(policy.Migration_REMOVE),
				Source: &MigrationSource{Uid: "example-check"},
			}}}},
		}}}
		formatted, err := FormatBundle(b, false)
		require.NoError(t, err)
		assert.NotContains(t, string(formatted), "reason:")
		assert.NotContains(t, string(formatted), "description:")
		assert.NotContains(t, string(formatted), "refs:")
	})
}
