// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/explorer"
)

func TestEvidenceFillUidIfEmpty(t *testing.T) {
	t.Run("fill uid for evidence with no uid", func(t *testing.T) {
		e := Evidence{
			Title: "test",
		}
		e.fillUidIfEmpty("framework", "control", "suffix")
		require.Equal(t, e.Uid, "framework-control-evidence-suffix")
	})
	t.Run("fill uid for evidence with uid", func(t *testing.T) {
		e := Evidence{
			Title: "test",
			Uid:   "uid",
		}
		e.fillUidIfEmpty("framework", "control", "fallback")
		// uid doesnt change if already set
		require.Equal(t, e.Uid, "uid")
	})
}

func TestGenerateFrameworkMap(t *testing.T) {
	c := &Control{
		Uid: "control-uid",
		Evidence: []*Evidence{
			{
				Uid:   "evidence-uid",
				Title: "evidence-title",
				Desc:  "evidence-desc",
				Checks: []*explorer.Mquery{
					{Uid: "check1"},
					{Uid: "check2"},
				},
				Queries: []*explorer.Mquery{
					{Uid: "query1"},
					{Uid: "query2"},
				},
			},
		},
	}

	cm := c.GenerateEvidenceControlMap()
	owner := &Framework{
		Uid: "soc-2",
	}
	policies := []*Policy{{Uid: "policy-uid"}}
	fm := GenerateFrameworkMap([]*ControlMap{cm}, owner, policies)
	expected := &FrameworkMap{
		FrameworkOwner: &explorer.ObjectRef{Uid: "soc-2"},
		Uid:            "soc-2-evidence-mapping",
		Controls:       []*ControlMap{cm},
		PolicyDependencies: []*explorer.ObjectRef{
			{Uid: "policy-uid"},
		},
	}
	require.Equal(t, expected, fm)
}

func TestGenerateEvidenceControlMap(t *testing.T) {
	t.Run("generate control map with no evidence", func(t *testing.T) {
		c := &Control{
			Uid: "control-uid",
		}
		cm := c.GenerateEvidenceControlMap()
		require.Nil(t, cm)
	})
	t.Run("generate control map with evidence", func(t *testing.T) {
		c := &Control{
			Uid: "control-uid",
			Evidence: []*Evidence{
				{
					Uid:   "evidence-uid",
					Title: "evidence-title",
					Desc:  "evidence-desc",
					Checks: []*explorer.Mquery{
						{Uid: "check1"},
						{Uid: "check2"},
					},
					Queries: []*explorer.Mquery{
						{Uid: "query1"},
						{Uid: "query2"},
					},
				},
			},
		}

		cm := c.GenerateEvidenceControlMap()
		expected := &ControlMap{
			Uid: "control-uid",
			Checks: []*ControlRef{
				{Uid: "check1"},
				{Uid: "check2"},
			},
			Queries: []*ControlRef{
				{Uid: "query1"},
				{Uid: "query2"},
			},
		}
		require.Equal(t, expected, cm)
	})
}

func TestEvidenceConvertToPolicy(t *testing.T) {
	e := &Evidence{
		Uid:   "evidence-uid",
		Title: "evidence-title",
		Desc:  "evidence-desc",
		Checks: []*explorer.Mquery{
			{Uid: "check1"},
			{Uid: "check2"},
		},
		Queries: []*explorer.Mquery{
			{Uid: "query1"},
			{Uid: "query2"},
		},
	}
	pol := e.convertToPolicy()
	expected := &Policy{
		Uid:  "evidence-uid-policy",
		Name: "evidence-title-policy",
		Docs: &PolicyDocs{Desc: "evidence-desc"},
		Groups: []*PolicyGroup{
			{
				Uid:     "evidence-queries",
				Type:    GroupType_CHAPTER,
				Queries: []*explorer.Mquery{{Uid: "query1"}, {Uid: "query2"}},
			},
			{
				Uid:    "evidence-checks",
				Type:   GroupType_CHAPTER,
				Checks: []*explorer.Mquery{{Uid: "check1"}, {Uid: "check2"}},
			},
		},
	}
	require.Equal(t, expected, pol)
}

func TestGenerateEvidencePolicies(t *testing.T) {
	e1 := &Evidence{
		Uid:   "evidence-uid",
		Title: "evidence-title",
		Checks: []*explorer.Mquery{
			{Uid: "check1"},
			{Uid: "check2"},
		},
	}
	e2 := &Evidence{
		Uid:   "evidence-uid-2",
		Title: "evidence-title-2",
		Queries: []*explorer.Mquery{
			{Uid: "query1"},
			{Uid: "query2"},
		},
	}
	control := &Control{
		Uid:      "control-uid",
		Evidence: []*Evidence{e1, e2},
	}
	policies := control.GenerateEvidencePolicies("framework-uid")
	expected := []*Policy{
		{
			Uid:  "evidence-uid-policy",
			Name: "evidence-title-policy",
			Groups: []*PolicyGroup{
				{
					Uid:    "evidence-checks",
					Type:   GroupType_CHAPTER,
					Checks: []*explorer.Mquery{{Uid: "check1"}, {Uid: "check2"}},
				},
			},
		},
		{
			Uid:  "evidence-uid-2-policy",
			Name: "evidence-title-2-policy",
			Groups: []*PolicyGroup{
				{
					Uid:     "evidence-queries",
					Type:    GroupType_CHAPTER,
					Queries: []*explorer.Mquery{{Uid: "query1"}, {Uid: "query2"}},
				},
			},
		},
	}
	require.Equal(t, expected, policies)
}
