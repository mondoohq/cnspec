// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"sort"
	"strings"

	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/policy"
)

// Filter narrows what the viewer shows. It lives in the frame rather than in the
// header pane because two panes have to agree about it: the pane that edits it
// and the pane that obeys it. Both call the Match methods here, so "what the
// filter means" is decided once.
//
// A zero Filter matches everything. An empty Statuses or Severities set means
// "no restriction on this axis", not "match nothing".
type Filter struct {
	// Search is matched case-insensitively as a substring of a check title or
	// an asset name.
	Search string
	// Statuses restricts to these outcomes. All six of reportmodel's statuses
	// are distinct here: a check that errored is not a check that failed and
	// must never be folded into one bucket.
	Statuses map[reportmodel.Status]bool
	// Severities restricts to these severity labels, which are the
	// policy.ScoreRatingText* values that reportmodel.Check.Severity carries.
	Severities map[string]bool
}

// AllStatuses is every status a filter can select, in the order a UI should
// offer them: the ones that need action first.
var AllStatuses = []reportmodel.Status{
	reportmodel.StatusFail,
	reportmodel.StatusError,
	reportmodel.StatusPass,
	reportmodel.StatusSkipped,
	reportmodel.StatusUnscored,
	reportmodel.StatusUnknown,
}

// AllSeverities is every severity label a filter can select, worst first.
var AllSeverities = []string{
	policy.ScoreRatingTextCritical,
	policy.ScoreRatingTextHigh,
	policy.ScoreRatingTextMedium,
	policy.ScoreRatingTextLow,
	policy.ScoreRatingTextNone,
}

// Active reports whether the filter restricts anything at all.
func (f Filter) Active() bool {
	return f.Search != "" || len(f.Statuses) > 0 || len(f.Severities) > 0
}

// Equal compares two filters by value, ignoring the difference between a nil
// set and an empty one.
func (f Filter) Equal(o Filter) bool {
	if f.Search != o.Search || len(f.Statuses) != len(o.Statuses) || len(f.Severities) != len(o.Severities) {
		return false
	}
	for k, v := range f.Statuses {
		if o.Statuses[k] != v {
			return false
		}
	}
	for k, v := range f.Severities {
		if o.Severities[k] != v {
			return false
		}
	}
	return true
}

// Clone returns a deep copy, so a pane can edit a filter and only publish it
// through State.SetFilter once it is complete.
func (f Filter) Clone() Filter {
	res := Filter{Search: f.Search}
	if len(f.Statuses) > 0 {
		res.Statuses = make(map[reportmodel.Status]bool, len(f.Statuses))
		for k, v := range f.Statuses {
			res.Statuses[k] = v
		}
	}
	if len(f.Severities) > 0 {
		res.Severities = make(map[string]bool, len(f.Severities))
		for k, v := range f.Severities {
			res.Severities[k] = v
		}
	}
	return res
}

// ToggleStatus flips one status on or off, returning the new filter. The
// receiver is not modified.
func (f Filter) ToggleStatus(s reportmodel.Status) Filter {
	res := f.Clone()
	if res.Statuses[s] {
		delete(res.Statuses, s)
		return res
	}
	if res.Statuses == nil {
		res.Statuses = map[reportmodel.Status]bool{}
	}
	res.Statuses[s] = true
	return res
}

// ToggleSeverity flips one severity on or off, returning the new filter.
func (f Filter) ToggleSeverity(sev string) Filter {
	res := f.Clone()
	if res.Severities[sev] {
		delete(res.Severities, sev)
		return res
	}
	if res.Severities == nil {
		res.Severities = map[string]bool{}
	}
	res.Severities[sev] = true
	return res
}

// MatchText reports whether s matches the search term.
func (f Filter) MatchText(s string) bool {
	if f.Search == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(f.Search))
}

// MatchStatus reports whether a status is selected.
func (f Filter) MatchStatus(s reportmodel.Status) bool {
	if len(f.Statuses) == 0 {
		return true
	}
	return f.Statuses[s]
}

// MatchSeverity reports whether a severity label is selected.
func (f Filter) MatchSeverity(sev string) bool {
	if len(f.Severities) == 0 {
		return true
	}
	return f.Severities[sev]
}

// MatchCheck reports whether a check survives the filter.
func (f Filter) MatchCheck(c *reportmodel.Check) bool {
	if c == nil {
		return false
	}
	return f.MatchStatus(c.Status) && f.MatchSeverity(c.Severity) && f.MatchText(c.Title)
}

// MatchPolicy reports whether a policy survives, which it does when any of its
// checks does or when the policy's own name matches the search and its status is
// selected. A policy with no checks left is still worth showing when the user
// searched for it by name.
func (f Filter) MatchPolicy(p *reportmodel.Policy) bool {
	if p == nil {
		return false
	}
	for _, c := range p.Checks {
		if f.MatchCheck(c) {
			return true
		}
	}
	return len(p.Checks) == 0 && f.MatchText(p.Name) && f.MatchStatus(p.Status)
}

// MatchAsset reports whether an asset survives the filter.
//
// An asset matches when any of its checks does -- or when the asset itself does.
// That second clause is the one that matters: an asset that failed to scan has
// no checks at all, so judging assets purely by their checks would hide exactly
// the assets that need attention behind an apparently empty screen. Filtering by
// ERROR must show them; filtering by severity, which is a property a check has
// and an unscanned asset cannot, must not remove them.
func (f Filter) MatchAsset(a *reportmodel.Asset) bool {
	if a == nil {
		return false
	}
	if f.matchesAssetItself(a) {
		return true
	}
	for _, c := range a.Checks {
		if f.MatchCheck(c) {
			return true
		}
	}
	return false
}

func (f Filter) matchesAssetItself(a *reportmodel.Asset) bool {
	if !f.MatchText(a.Name) || !f.MatchStatus(a.Status) {
		return false
	}
	// A severity filter is about checks. A scanned asset is represented by its
	// checks, which MatchAsset tries next; an asset that never produced one has
	// nothing to be judged by and stays.
	return len(f.Severities) == 0 || !a.Scanned()
}

// Describe renders the active restrictions as a short human-readable string,
// e.g. `"ssh" · FAIL, ERROR · critical`. It is empty when the filter is not
// active.
func (f Filter) Describe() string {
	var parts []string
	if f.Search != "" {
		parts = append(parts, `"`+f.Search+`"`)
	}
	if len(f.Statuses) > 0 {
		var st []string
		for _, s := range AllStatuses {
			if f.Statuses[s] {
				st = append(st, string(s))
			}
		}
		parts = append(parts, strings.Join(st, ", "))
	}
	if len(f.Severities) > 0 {
		var sev []string
		for _, s := range AllSeverities {
			if f.Severities[s] {
				sev = append(sev, s)
			}
		}
		// A severity outside the known set is still worth naming.
		if len(sev) != len(f.Severities) {
			sev = sortedSet(f.Severities)
		}
		parts = append(parts, strings.Join(sev, ", "))
	}
	return strings.Join(parts, " · ")
}

func sortedSet(m map[string]bool) []string {
	res := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			res = append(res, k)
		}
	}
	sort.Strings(res)
	return res
}
