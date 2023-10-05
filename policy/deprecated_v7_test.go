// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/explorer"
)

func TestV7Fill(t *testing.T) {
	b, err := bundleFromSingleFile("./deprecated_v7.mql.yaml")
	require.NoError(t, err)
	require.NotNil(t, b)

	b.FillV7()
	require.Equal(t, []*DeprecatedV7_Policy{
		{
			Name:          "Example policy 1",
			Version:       "1.0.0",
			Uid:           "example1",
			ScoringSystem: explorer.ScoringSystem_WORST,
			Authors:       []*DeprecatedV7_Author{{Name: "Mondoo", Email: "hello@mondoo.com"}},
			Props: map[string]string{
				"homeProp": "",
			},
			Specs: []*DeprecatedV7_PolicySpec{{
				Policies: map[string]*DeprecatedV7_ScoringSpec{},
				ScoringQueries: map[string]*DeprecatedV7_ScoringSpec{
					"sshd-01": nil,
					"sshd-02": nil,
					"sshd-03": nil,
				},
				DataQueries: map[string]QueryAction{
					"sshd-d-1":  QueryAction_UNSPECIFIED,
					"home-info": QueryAction_UNSPECIFIED,
				},
				AssetFilter: &DeprecatedV7_Mquery{
					Query:    "asset.family.contains(_ == 'unix')",
					CodeId:   "M/J+yy3Inwo=",
					Checksum: "snTEELSBE7I=",
					Type:     "\x04",
					Title:    "asset.family.contains(_ == 'unix')",
				},
			}},
		},
	}, b.DeprecatedV7Policies)
}
