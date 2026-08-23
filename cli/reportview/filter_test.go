// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/policy"
)

func TestZeroFilterMatchesEverything(t *testing.T) {
	var f Filter
	require.False(t, f.Active())
	require.Empty(t, f.Describe())

	st := NewState(loadReport(t, fixtureUbuntu))
	require.Len(t, st.FilteredAssets(), 1)
	require.Len(t, st.FilteredChecks(st.Report.Assets[0].Checks), 24)
}

// The six statuses stay six. A filter on FAIL must not sweep up ERROR, and a
// filter on ERROR must not sweep up FAIL: a check that could not run and a check
// that ran and failed are different findings with different fixes.
func TestStatusFilterKeepsOutcomesApart(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	checks := st.Report.Assets[0].Checks

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusFail: true}})
	failed := st.FilteredChecks(checks)
	require.Len(t, failed, 2)
	for _, c := range failed {
		require.Equal(t, reportmodel.StatusFail, c.Status)
	}

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusError: true}})
	errored := st.FilteredChecks(checks)
	require.Len(t, errored, 4)
	for _, c := range errored {
		require.Equal(t, reportmodel.StatusError, c.Status)
	}

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{
		reportmodel.StatusFail: true, reportmodel.StatusError: true,
	}})
	require.Len(t, st.FilteredChecks(checks), 6)
}

func TestSearchIsCaseInsensitiveSubstring(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	checks := st.Report.Assets[0].Checks

	st.SetFilter(Filter{Search: "PERMISSIONS"})
	upper := st.FilteredChecks(checks)
	require.Len(t, upper, 8)

	st.SetFilter(Filter{Search: "permissions"})
	require.Equal(t, upper, st.FilteredChecks(checks), "the search is case-insensitive")

	st.SetFilter(Filter{Search: "no such check anywhere"})
	require.Empty(t, st.FilteredChecks(checks))
}

// Severity is what a check is worth, and is independent of how it did. The
// ubuntu fixture is 10 critical, 10 high and 4 with none.
func TestSeverityFilter(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	checks := st.Report.Assets[0].Checks

	st.SetFilter(Filter{Severities: map[string]bool{policy.ScoreRatingTextCritical: true}})
	require.Len(t, st.FilteredChecks(checks), 10)

	st.SetFilter(Filter{Severities: map[string]bool{
		policy.ScoreRatingTextCritical: true, policy.ScoreRatingTextHigh: true,
	}})
	require.Len(t, st.FilteredChecks(checks), 20)

	// Severity and status are combined with AND: the critical checks that failed, not the
	// critical checks plus the failed ones.
	st.SetFilter(Filter{
		Severities: map[string]bool{policy.ScoreRatingTextCritical: true},
		Statuses:   map[reportmodel.Status]bool{reportmodel.StatusFail: true},
	})
	require.Len(t, st.FilteredChecks(checks), 2)
}

// The case the viewer exists to get right: an asset that failed to scan has no
// checks at all, so a filter that judges assets only by their checks hides
// exactly the assets that need attention. Filtering by ERROR must show them, and
// a severity filter -- severity being a property of a check -- must not remove
// them.
func TestFilterKeepsUnscannedAssets(t *testing.T) {
	st := NewState(loadReport(t, fixtureK8s))
	require.Len(t, st.Report.Assets, 15)
	for _, a := range st.Report.Assets {
		require.False(t, a.Scanned())
		require.Empty(t, a.Checks)
	}

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusError: true}})
	require.Len(t, st.FilteredAssets(), 15, "filtering for errors must show the errored assets")

	st.SetFilter(Filter{Severities: map[string]bool{policy.ScoreRatingTextCritical: true}})
	require.Len(t, st.FilteredAssets(), 15, "an unscanned asset has no severity to be judged by")

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusPass: true}})
	require.Empty(t, st.FilteredAssets(), "none of them passed")

	st.SetFilter(Filter{Search: "kube-proxy"})
	assets := st.FilteredAssets()
	require.NotEmpty(t, assets)
	require.Less(t, len(assets), 15)
}

// A scanned asset survives on the strength of its checks, and disappears when
// none of them matches.
func TestFilterOnScannedAsset(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusFail: true}})
	require.Len(t, st.FilteredAssets(), 1)

	st.SetFilter(Filter{Search: "no check has this in its title"})
	require.Empty(t, st.FilteredAssets())
}

func TestFilterCountsMatchWhatIsShown(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	require.Equal(t, 24, st.Counts().Total)

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusError: true}})
	c := st.Counts()
	require.Equal(t, 4, c.Total)
	require.Equal(t, 4, c.Errored)
	require.Equal(t, 0, c.Passed)
}

func TestToggleAndClone(t *testing.T) {
	base := Filter{Search: "tls"}

	one := base.ToggleStatus(reportmodel.StatusFail)
	require.True(t, one.Statuses[reportmodel.StatusFail])
	require.Empty(t, base.Statuses, "toggling must not touch the receiver")

	two := one.ToggleSeverity(policy.ScoreRatingTextHigh)
	require.True(t, two.Severities[policy.ScoreRatingTextHigh])
	require.Empty(t, one.Severities)

	require.False(t, two.ToggleStatus(reportmodel.StatusFail).Statuses[reportmodel.StatusFail])
	require.Equal(t, `"tls" · FAIL · HIGH`, two.Describe())

	clone := two.Clone()
	require.True(t, two.Equal(clone))
	clone.Statuses[reportmodel.StatusPass] = true
	require.False(t, two.Equal(clone), "a clone must not share its maps")
}

func TestFilterEqualIgnoresNilVersusEmpty(t *testing.T) {
	require.True(t, Filter{}.Equal(Filter{Statuses: map[reportmodel.Status]bool{}}))
	require.False(t, Filter{Search: "a"}.Equal(Filter{}))
}

// SetFilter bumps the revision only when something actually changed, so a pane
// that rebuilds its rows on FilterRev does not rebuild them on every keystroke
// that changed nothing.
func TestFilterRevision(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	require.Equal(t, 0, st.FilterRev)

	st.SetFilter(Filter{Search: "ssh"})
	require.Equal(t, 1, st.FilterRev)
	st.SetFilter(Filter{Search: "ssh"})
	require.Equal(t, 1, st.FilterRev)
	st.SetFilter(Filter{})
	require.Equal(t, 2, st.FilterRev)
}

// MatchAsset and MatchCheck are nil-safe: a pane walking a partially built tree
// must not panic.
func TestMatchNil(t *testing.T) {
	var f Filter
	require.False(t, f.MatchAsset(nil))
	require.False(t, f.MatchCheck(nil))
	require.False(t, f.MatchPolicy(nil))
}
