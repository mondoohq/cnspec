// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// `file` declares an explicit `permissions()` accessor, and `file.permissions`
// is also a resource name. Spelling out the dotted path compiles to a bare
// `file.permissions` with no id, so every field on it reads null.
func TestShadowedAccessor_BareDottedPath(t *testing.T) {
	q := &Mquery{
		Uid:         "test-bare-dotted-path",
		Mql:         "file.permissions.user_readable == true",
		FileContext: FileContext{Line: 5, Column: 1},
	}

	entries := walkQueryForShadowedAccessors(schema, newConf(schema), "test.mql.yaml", q)
	require.Len(t, entries, 1)
	assert.Equal(t, QueryShadowedAccessorRuleID, entries[0].RuleID)
	assert.Equal(t, LevelWarning, entries[0].Level)
	assert.Contains(t, entries[0].Message, "test-bare-dotted-path")
	assert.Contains(t, entries[0].Message, "file.permissions")
	// the message must name the accessor and the parent that owns it
	assert.Contains(t, entries[0].Message, "'permissions' accessor on 'file'")
	assert.Equal(t, 5, entries[0].Location[0].Line)
}

// The same value reached through the accessor is the correct form and must
// stay silent.
func TestShadowedAccessor_ThroughAccessor(t *testing.T) {
	for _, mql := range []string{
		`file("/etc/passwd").permissions.user_readable == true`,
		`file("/etc/passwd") { permissions.user_readable == true }`,
	} {
		q := &Mquery{Uid: "ok", Mql: mql, FileContext: FileContext{Line: 1, Column: 1}}
		entries := walkQueryForShadowedAccessors(schema, newConf(schema), "test.mql.yaml", q)
		assert.Empty(t, entries, "accessor form must not be flagged: %s", mql)
	}
}

// A dotted path into a resource the schema synthesized a field for (the parent
// declares no accessor of that name) is the only way to reach it, so it must
// not be flagged. Without this distinction the rule would fire on every
// `aws.ec2.instance` / `k8s.pod` style query.
func TestShadowedAccessor_ImplicitResourceNotFlagged(t *testing.T) {
	q := &Mquery{
		Uid:         "implicit",
		Mql:         "asset.name != ''",
		FileContext: FileContext{Line: 1, Column: 1},
	}
	entries := walkQueryForShadowedAccessors(schema, newConf(schema), "test.mql.yaml", q)
	assert.Empty(t, entries)
}

// A query that does not compile is bundle-compile-error's problem, not ours.
func TestShadowedAccessor_UncompilableQueryIgnored(t *testing.T) {
	q := &Mquery{
		Uid:         "broken",
		Mql:         "this.is.not.a.resource",
		FileContext: FileContext{Line: 1, Column: 1},
	}
	entries := walkQueryForShadowedAccessors(schema, newConf(schema), "test.mql.yaml", q)
	assert.Empty(t, entries)
}

// One entry per shadowed resource, however many times the query mentions it.
func TestShadowedAccessor_DeduplicatesPerResource(t *testing.T) {
	q := &Mquery{
		Uid:         "dupes",
		Mql:         "file.permissions.user_readable == true && file.permissions.group_readable == false",
		FileContext: FileContext{Line: 1, Column: 1},
	}
	entries := walkQueryForShadowedAccessors(schema, newConf(schema), "test.mql.yaml", q)
	require.Len(t, entries, 1)
}

// Known over-reporting, pinned so the trade-off stays visible: `os.date`
// exposes only `time()`/`timezone()` accessors and does resolve standalone,
// but the schema cannot distinguish those from accessors that read cache
// values the parent stored, so the rule reports it. This is the noise the
// experimental label covers -- if the rule later learns to tell them apart,
// this test should flip to Empty.
func TestShadowedAccessor_SelfSufficientAccessorsStillReported(t *testing.T) {
	q := &Mquery{
		Uid:         "computed",
		Mql:         "os.date.timezone != ''",
		FileContext: FileContext{Line: 1, Column: 1},
	}
	entries := walkQueryForShadowedAccessors(schema, newConf(schema), "test.mql.yaml", q)
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].Message, "[experimental]")
}

// A resource that declares init(...) resolves itself from args or the scanned
// asset, so reaching it by name is correct.
func TestShadowedAccessor_ResourceWithInitNotFlagged(t *testing.T) {
	q := &Mquery{
		Uid:         "has-init",
		Mql:         "file(\"/etc/passwd\").exists",
		FileContext: FileContext{Line: 1, Column: 1},
	}
	entries := walkQueryForShadowedAccessors(schema, newConf(schema), "test.mql.yaml", q)
	assert.Empty(t, entries)
}
