// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandaloneWhereIn(t *testing.T) {
	conf := newConf(schema)

	cases := []struct {
		name string
		mql  string
		want []string
	}{
		{"filter reads the element", `users.where(name == "root").length > 0`, nil},
		{"all reads the element", `users.all(shell != "")`, nil},
		{"named binding", `users.all(u: u.name != "")`, nil},
		{"array contains a value", `[1, 2].contains(1)`, nil},
		{"outer element read from a nested block", `users.all(u: groups.any(gid == u.gid))`, nil},
		{"all ignores the element", `users.all(processes.length > 0)`, []string{"users"}},
		{"where ignores the element", `users.where(processes.length > 0).length > 0`, []string{"users"}},
		{"any ignores the element", `users.any(packages.length > 0)`, []string{"users"}},
		{"only the outer filter ignores its element", `users.all(groups.any(name == "wheel"))`, []string{"users"}},
		{"compile error", `users.all(`, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, standaloneWhereIn(tc.mql, conf))
		})
	}
}

func TestLintStandaloneWhere_NamesQueryAndLine(t *testing.T) {
	b := &Bundle{
		Queries: []*Mquery{
			{
				Uid:         "constant-filter",
				Mql:         "users.all(processes.length > 0)",
				FileContext: FileContext{Line: 12, Column: 5},
			},
			{
				Uid: "element-filter",
				Mql: `users.all(name != "")`,
			},
		},
	}

	entries := lintStandaloneWhere(newConf(schema), "test.mql.yaml", b)
	require.Len(t, entries, 1)
	assert.Equal(t, QueryStandaloneWhereRuleID, entries[0].RuleID)
	assert.Equal(t, LevelWarning, entries[0].Level)
	assert.Contains(t, entries[0].Message, "constant-filter")
	assert.Contains(t, entries[0].Message, "'users'")
	require.Len(t, entries[0].Location, 1)
	assert.Equal(t, Location{File: "test.mql.yaml", Line: 12, Column: 5}, entries[0].Location[0])
}

func TestLintStandaloneWhere_Filters(t *testing.T) {
	b := &Bundle{
		Queries: []*Mquery{
			{
				Uid: "filtered",
				Mql: "true",
				Filters: &Filters{
					Items: map[string]*Mquery{
						"a": {Mql: "users.where(processes.length > 0).length > 0"},
					},
					FileContext: FileContext{Line: 3, Column: 7},
				},
			},
		},
	}

	entries := lintStandaloneWhere(newConf(schema), "test.mql.yaml", b)
	require.Len(t, entries, 1)
	assert.Equal(t, FilterStandaloneWhereRuleID, entries[0].RuleID)
	assert.Contains(t, entries[0].Message, "query 'filtered'")
	assert.Equal(t, 3, entries[0].Location[0].Line)
}
