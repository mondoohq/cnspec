// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"go.mondoo.com/cnspec/cli/reportmodel"
)

// State is what the panes share, and nothing more.
//
// The rule for whether something belongs here: does another pane have to see it?
// The selected check does -- the tree sets it and the detail draws it. The
// filter does -- the header sets it and the tree obeys it. A scroll offset, a
// cursor, a set of collapsed nodes and an open twisty do not: they are the
// pane's own business and live on the pane's own struct, which is what lets
// three panes be written at once without editing one file between them.
//
// A pane mutates State in place through the methods below. The methods exist so
// that a change carries its consequences with it: Select bumps SelectionRev so a
// detail pane knows to scroll back to the top, and SetFilter bumps FilterRev so
// a tree knows to rebuild its rows.
type State struct {
	// Report is the model being browsed. It is never nil, but it may be empty.
	Report *reportmodel.Report

	// Sel is what the panes agree is selected right now.
	Sel Selection
	// SelectionRev increments on every change to Sel. A pane that caches
	// anything derived from the selection compares this against the revision it
	// cached at, rather than comparing pointers.
	SelectionRev int

	// Filter narrows what the tree shows. It is owned by the header pane and
	// read by everyone else; use Filtered* rather than reimplementing it.
	Filter Filter
	// FilterRev increments on every change to Filter.
	FilterRev int

	// Focus is the pane that keys go to. The frame maintains it; a pane may set
	// it to hand focus on (the tree does this when enter opens a check).
	Focus PaneID

	// Notice is a single line of feedback shown in the footer instead of the key
	// hints. The frame clears it on the next keypress, so it is for "copied",
	// "no more matches" and the like, not for permanent status.
	Notice string
}

// Selection is the asset, policy and check the panes agree on.
//
// The three fields are not independent: Check belongs to Policy, which ran on
// Asset. A selection may stop early -- an asset with no policies selects only an
// asset, and an asset that failed to scan can never select more than itself.
type Selection struct {
	Asset  *reportmodel.Asset
	Policy *reportmodel.Policy
	Check  *reportmodel.Check
}

// NewState builds the shared state for a report. A nil report is replaced by an
// empty one so no pane has to nil-check it.
func NewState(report *reportmodel.Report) *State {
	if report == nil {
		report = reportmodel.New(nil)
	}
	st := &State{Report: report, Focus: PaneTree}
	// Select something immediately: a viewer that opens on an empty detail pane
	// looks broken, and the first asset is the only defensible default.
	if len(report.Assets) > 0 {
		st.Select(Selection{Asset: report.Assets[0]})
	}
	return st
}

// Select replaces the selection and bumps SelectionRev. It is a no-op when
// nothing actually changes, so a tree may call it on every cursor move without
// resetting the detail pane's scroll.
func (s *State) Select(sel Selection) {
	if s.Sel == sel {
		return
	}
	s.Sel = sel
	s.SelectionRev++
}

// SelectAsset selects an asset and clears the policy and check beneath it.
func (s *State) SelectAsset(a *reportmodel.Asset) {
	s.Select(Selection{Asset: a})
}

// SelectCheck selects a check within an asset, optionally within a policy.
func (s *State) SelectCheck(a *reportmodel.Asset, p *reportmodel.Policy, c *reportmodel.Check) {
	s.Select(Selection{Asset: a, Policy: p, Check: c})
}

// SetFilter replaces the filter and bumps FilterRev. Like Select it is a no-op
// when nothing changes.
func (s *State) SetFilter(f Filter) {
	if s.Filter.Equal(f) {
		return
	}
	s.Filter = f
	s.FilterRev++
}

// FilteredAssets is the assets that survive the current filter, in model order.
// It is the one implementation of "what the tree shows", so the header's count
// and the tree's rows can never disagree.
func (s *State) FilteredAssets() []*reportmodel.Asset {
	if !s.Filter.Active() {
		return s.Report.Assets
	}
	res := make([]*reportmodel.Asset, 0, len(s.Report.Assets))
	for _, a := range s.Report.Assets {
		if s.Filter.MatchAsset(a) {
			res = append(res, a)
		}
	}
	return res
}

// FilteredChecks is the checks of an asset that survive the current filter, in
// model order.
func (s *State) FilteredChecks(checks []*reportmodel.Check) []*reportmodel.Check {
	if !s.Filter.Active() {
		return checks
	}
	res := make([]*reportmodel.Check, 0, len(checks))
	for _, c := range checks {
		if s.Filter.MatchCheck(c) {
			res = append(res, c)
		}
	}
	return res
}

// Counts tallies the checks that survive the current filter across every asset,
// so the header can report "showing N of M" without walking the tree itself.
func (s *State) Counts() reportmodel.Counts {
	if !s.Filter.Active() {
		return s.Report.CheckCounts
	}
	var res reportmodel.Counts
	for _, a := range s.Report.Assets {
		for _, c := range s.FilteredChecks(a.Checks) {
			res.Add(c.Status)
		}
	}
	return res
}
