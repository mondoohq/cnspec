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

func TestGenerateEvidenceControlMap(t *testing.T) {
	t.Run("generate control map with no evidence", func(t *testing.T) {
		c := &Control{
			Uid: "control-uid",
		}
		cm := c.generateEvidenceControlMap()
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

		cm := c.generateEvidenceControlMap()
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

	t.Run("generate control map with multiple evidences", func(t *testing.T) {
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
				{
					Uid:   "evidence-uid-2",
					Title: "evidence-title-2",
					Desc:  "evidence-desc-2",
					Checks: []*explorer.Mquery{
						{Uid: "check3"},
						{Uid: "check4"},
					},
					Queries: []*explorer.Mquery{
						{Uid: "query3"},
						{Uid: "query4"},
					},
				},
			},
		}

		cm := c.generateEvidenceControlMap()
		expected := &ControlMap{
			Uid: "control-uid",
			Checks: []*ControlRef{
				{Uid: "check1"},
				{Uid: "check2"},
				{Uid: "check3"},
				{Uid: "check4"},
			},
			Queries: []*ControlRef{
				{Uid: "query1"},
				{Uid: "query2"},
				{Uid: "query3"},
				{Uid: "query4"},
			},
		}
		require.Equal(t, expected, cm)
	})
}

func TestGenerateEvidenceFrameworkMap(t *testing.T) {
	t.Run("generate framework map with no evidence", func(t *testing.T) {
		f := &Framework{
			Uid: "framework-uid",
			Groups: []*FrameworkGroup{
				{
					Uid: "group-uid",
					Controls: []*Control{
						{
							Uid: "control-uid",
						},
					},
				},
			},
		}
		evidenceFm := f.generateEvidenceFrameworkMap(nil)
		require.Nil(t, evidenceFm)
		f = &Framework{}
		evidenceFm = f.generateEvidenceFrameworkMap(nil)
		require.Nil(t, evidenceFm)
	})
	t.Run("generate framework map with evidence", func(t *testing.T) {
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

		f := &Framework{
			Uid: "framework-uid",
			Groups: []*FrameworkGroup{
				{
					Uid: "group-uid",
					Controls: []*Control{
						c,
					},
				},
			},
		}

		fm := f.generateEvidenceFrameworkMap(&Policy{Uid: "policy-uid"})
		expected := &FrameworkMap{
			FrameworkOwner: &explorer.ObjectRef{Uid: "framework-uid"},
			Uid:            "framework-uid-evidence-mapping",
			Controls: []*ControlMap{
				{
					Uid: "control-uid",
					Checks: []*ControlRef{
						{Uid: "check1"},
						{Uid: "check2"},
					},
					Queries: []*ControlRef{
						{Uid: "query1"},
						{Uid: "query2"},
					},
				},
			},
			PolicyDependencies: []*explorer.ObjectRef{{Uid: "policy-uid"}},
		}
		require.Equal(t, expected, fm)
	})

	t.Run("generate framework map with multiple evidences", func(t *testing.T) {
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
				{
					Uid:   "evidence-uid-2",
					Title: "evidence-title-2",
					Desc:  "evidence-desc-2",
					Checks: []*explorer.Mquery{
						{Uid: "check3"},
						{Uid: "check4"},
					},
					Queries: []*explorer.Mquery{
						{Uid: "query3"},
						{Uid: "query4"},
					},
				},
			},
		}
		c1 := &Control{
			Uid: "control-uid-2",
			Evidence: []*Evidence{
				{
					Uid:   "evidence-uid-3",
					Title: "evidence-title-3",
					Desc:  "evidence-desc-3",
					Checks: []*explorer.Mquery{
						{Uid: "check5"},
						{Uid: "check6"},
					},
					Queries: []*explorer.Mquery{
						{Uid: "query5"},
						{Uid: "query6"},
					},
				},
			},
		}

		f := &Framework{
			Uid: "framework-uid",
			Groups: []*FrameworkGroup{
				{
					Uid:      "group-uid",
					Controls: []*Control{c, c1},
				},
			},
		}

		fm := f.generateEvidenceFrameworkMap(&Policy{Uid: "policy-uid"})
		expected := &FrameworkMap{
			FrameworkOwner: &explorer.ObjectRef{Uid: "framework-uid"},
			Uid:            "framework-uid-evidence-mapping",
			Controls: []*ControlMap{
				{
					Uid: "control-uid",
					Checks: []*ControlRef{
						{Uid: "check1"},
						{Uid: "check2"},
						{Uid: "check3"},
						{Uid: "check4"},
					},
					Queries: []*ControlRef{
						{Uid: "query1"},
						{Uid: "query2"},
						{Uid: "query3"},
						{Uid: "query4"},
					},
				},
				{
					Uid: "control-uid-2",
					Checks: []*ControlRef{
						{Uid: "check5"},
						{Uid: "check6"},
					},
					Queries: []*ControlRef{
						{Uid: "query5"},
						{Uid: "query6"},
					},
				},
			},
			PolicyDependencies: []*explorer.ObjectRef{{Uid: "policy-uid"}},
		}
		require.Equal(t, expected, fm)
	})
}

func TestGenerateEvidencePolicy(t *testing.T) {
	t.Run("generate policy with no evidence", func(t *testing.T) {
		f := &Framework{
			Uid: "framework-uid",
			Groups: []*FrameworkGroup{
				{
					Uid: "group-uid",
					Controls: []*Control{
						{
							Uid: "control-uid",
						},
					},
				},
			},
		}
		pol := f.generateEvidencePolicy()
		require.Nil(t, pol)
		f = &Framework{}
		pol = f.generateEvidencePolicy()
		require.Nil(t, pol)
	})
	t.Run("generate policy with evidence", func(t *testing.T) {
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

		f := &Framework{
			Uid:  "framework-uid",
			Name: "soc2",
			Groups: []*FrameworkGroup{
				{
					Uid: "group-uid",
					Controls: []*Control{
						c,
					},
				},
			},
		}

		pol := f.generateEvidencePolicy()
		expected := &Policy{
			Uid:  "framework-uid-evidence-policy",
			Name: "soc2-evidence-policy",
			Groups: []*PolicyGroup{
				{
					Uid:     "evidence-uid",
					Title:   "evidence-title",
					Type:    GroupType_CHAPTER,
					Docs:    &PolicyGroupDocs{Desc: "evidence-desc"},
					Queries: []*explorer.Mquery{{Uid: "query1"}, {Uid: "query2"}},
					Checks:  []*explorer.Mquery{{Uid: "check1"}, {Uid: "check2"}},
				},
			},
		}
		require.Equal(t, expected, pol)
	})
	t.Run("generate policy with multiple evidences", func(t *testing.T) {
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
				{
					Uid:   "evidence-uid-2",
					Title: "evidence-title-2",
					Desc:  "evidence-desc-2",
					Checks: []*explorer.Mquery{
						{Uid: "check3"},
						{Uid: "check4"},
					},
					Queries: []*explorer.Mquery{
						{Uid: "query3"},
						{Uid: "query4"},
					},
				},
			},
		}
		c1 := &Control{
			Uid: "control-uid-2",
			Evidence: []*Evidence{
				{
					Uid:   "evidence-uid-3",
					Title: "evidence-title-3",
					Desc:  "evidence-desc-3",
					Checks: []*explorer.Mquery{
						{Uid: "check5"},
						{Uid: "check6"},
					},
					Queries: []*explorer.Mquery{
						{Uid: "query5"},
						{Uid: "query6"},
					},
				},
			},
		}

		f := &Framework{
			Uid:  "framework-uid",
			Name: "soc2",
			Groups: []*FrameworkGroup{
				{
					Uid:      "group-uid",
					Controls: []*Control{c, c1},
				},
			},
		}

		pol := f.generateEvidencePolicy()
		expected := &Policy{
			Uid:  "framework-uid-evidence-policy",
			Name: "soc2-evidence-policy",
			Groups: []*PolicyGroup{
				{
					Uid:     "evidence-uid",
					Title:   "evidence-title",
					Type:    GroupType_CHAPTER,
					Docs:    &PolicyGroupDocs{Desc: "evidence-desc"},
					Queries: []*explorer.Mquery{{Uid: "query1"}, {Uid: "query2"}},
					Checks:  []*explorer.Mquery{{Uid: "check1"}, {Uid: "check2"}},
				},
				{
					Uid:     "evidence-uid-2",
					Title:   "evidence-title-2",
					Type:    GroupType_CHAPTER,
					Docs:    &PolicyGroupDocs{Desc: "evidence-desc-2"},
					Queries: []*explorer.Mquery{{Uid: "query3"}, {Uid: "query4"}},
					Checks:  []*explorer.Mquery{{Uid: "check3"}, {Uid: "check4"}},
				},
				{
					Uid:     "evidence-uid-3",
					Title:   "evidence-title-3",
					Type:    GroupType_CHAPTER,
					Docs:    &PolicyGroupDocs{Desc: "evidence-desc-3"},
					Queries: []*explorer.Mquery{{Uid: "query5"}, {Uid: "query6"}},
					Checks:  []*explorer.Mquery{{Uid: "check5"}, {Uid: "check6"}},
				},
			},
		}
		require.Equal(t, expected, pol)
	})
}

func TestEvidenceConvertToPolicyGroup(t *testing.T) {
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
	polGroup := e.convertToPolicyGroup()
	expected := &PolicyGroup{
		Uid:     "evidence-uid",
		Title:   "evidence-title",
		Type:    GroupType_CHAPTER,
		Docs:    &PolicyGroupDocs{Desc: "evidence-desc"},
		Queries: []*explorer.Mquery{{Uid: "query1"}, {Uid: "query2"}},
		Checks:  []*explorer.Mquery{{Uid: "check1"}, {Uid: "check2"}},
	}
	require.Equal(t, expected, polGroup)
}

func TestGenerateEvidenceObjects(t *testing.T) {
	t.Run("generate evidence objects with no evidence", func(t *testing.T) {
		f := &Framework{
			Uid:  "framework-uid",
			Name: "soc2",
			Groups: []*FrameworkGroup{
				{
					Uid: "group-uid",
					Controls: []*Control{
						{
							Uid: "control-uid",
						},
					},
				},
			},
		}
		pol, fm := f.GenerateEvidenceObjects()
		require.Nil(t, pol)
		require.Nil(t, fm)
		f = &Framework{}
		pol, fm = f.GenerateEvidenceObjects()
		require.Nil(t, pol)
		require.Nil(t, fm)
	})

	t.Run("generate evidence objects with evidence", func(t *testing.T) {
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

		f := &Framework{
			Uid:  "framework-uid",
			Name: "soc2",
			Groups: []*FrameworkGroup{
				{
					Uid: "group-uid",
					Controls: []*Control{
						c,
					},
				},
			},
		}

		pol, fm := f.GenerateEvidenceObjects()
		expectedPol := &Policy{
			Uid:  "framework-uid-evidence-policy",
			Name: "soc2-evidence-policy",
			Groups: []*PolicyGroup{
				{
					Uid:     "evidence-uid",
					Title:   "evidence-title",
					Type:    GroupType_CHAPTER,
					Docs:    &PolicyGroupDocs{Desc: "evidence-desc"},
					Queries: []*explorer.Mquery{{Uid: "query1"}, {Uid: "query2"}},
					Checks:  []*explorer.Mquery{{Uid: "check1"}, {Uid: "check2"}},
				},
			},
		}
		expectedFm := &FrameworkMap{
			FrameworkOwner: &explorer.ObjectRef{Uid: "framework-uid"},
			Uid:            "framework-uid-evidence-mapping",
			Controls: []*ControlMap{
				{
					Uid: "control-uid",
					Checks: []*ControlRef{
						{Uid: "check1"},
						{Uid: "check2"},
					},
					Queries: []*ControlRef{
						{Uid: "query1"},
						{Uid: "query2"},
					},
				},
			},
			PolicyDependencies: []*explorer.ObjectRef{{Uid: "framework-uid-evidence-policy"}},
		}
		require.Equal(t, expectedPol, pol)
		require.Equal(t, expectedFm, fm)
		// check that the original framework's evidence is cleared.
		require.Nil(t, f.Groups[0].Controls[0].Evidence)
	})
}
