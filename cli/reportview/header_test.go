// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/policy"
)

// The header is asserted on as lines and widths rather than as a screenshot: a
// golden picture of a terminal breaks on every colour change and proves nothing
// about the two things that matter, which are that the numbers are true and that
// the band is exactly as tall as it said it would be.

// headerFor builds the registered header for a fixture, which is also a check
// that init() put the real pane in the slot rather than the placeholder.
func headerFor(t *testing.T, fixture string) (*headerPane, *State) {
	t.Helper()
	st := NewState(loadReport(t, fixture))
	p, ok := buildPane(PaneHeader, st).(*headerPane)
	require.True(t, ok, "the header slot must hold the registered header pane")
	return p, st
}

// headerRows renders the band at a width and strips the styling, so an assertion
// is about the words and the columns rather than about the escape sequences.
func headerRows(p *headerPane, st *State, w int) []string {
	lines := p.Render(st, tui.Rect{W: w, H: MaxHeaderLines}).Lines
	res := make([]string, len(lines))
	for i, ln := range lines {
		res[i] = ansi.Strip(ln)
	}
	return res
}

// headerWidths are the terminal widths the band has to survive, from a wide
// window down to one narrower than a single row of chips.
var headerWidths = []int{200, 160, 120, 100, 80, 76, 60, 40, 20, 8, 1}

// --- the summary ------------------------------------------------------------

// Five outcomes stay five. Pass, fail, error, skipped and unscored are different
// things with different fixes, and the band reports each of them as its own
// number rather than adding any two of them together.
func TestSummaryKeepsFiveOutcomesApart(t *testing.T) {
	p, st := headerFor(t, fixtureUbuntu)
	line := headerRows(p, st, 160)[0]

	require.Contains(t, line, "1 asset")
	for _, want := range []string{"2 FAIL", "4 ERROR", "18 PASS", "0 SKIPPED", "0 UNSCORED"} {
		require.Contains(t, line, want, "each outcome gets its own number")
	}
	// 2 failed plus 4 errored is six findings, and the band must never say so:
	// a check that could not run is not a check that ran and failed.
	require.NotContains(t, line, "6 ")
	require.NotContains(t, line, "24 ", "the total is not an outcome")

	// UNKNOWN is a bucket for a score cnspec did not recognise. Nothing landed
	// in it here, so it does not take a column.
	require.Equal(t, 0, st.Report.CheckCounts.Unknown)
	require.NotContains(t, line, "UNKNOWN")
}

// The trap this feature must not fall into. report-k8s.json is fifteen assets,
// zero reports, fifteen errors and no bundle: "0 failed" would be a lie, and
// five zeroes would read as a clean bill of health for a scan where nothing ran.
func TestSummaryTellsTheUnscannedStory(t *testing.T) {
	p, st := headerFor(t, fixtureK8s)
	require.Equal(t, 15, len(st.Report.Assets))
	require.Equal(t, 15, st.Report.AssetCounts.Errored)
	require.Equal(t, 0, st.Report.CheckCounts.Total)

	line := headerRows(p, st, 160)[0]
	require.Contains(t, line, "15 assets")
	require.Contains(t, line, "15 not scanned")
	require.Contains(t, line, "no checks ran")

	for _, lie := range []string{"0 FAIL", "0 PASS", "0 ERROR", "0 failed", "0 total"} {
		require.NotContains(t, line, lie, "a scan that ran nothing has no outcome tally")
	}
}

// A narrow band drops an outcome that has nothing in it rather than merging two
// that do. What is shown is exact; what is missing is zero.
func TestNarrowSummaryDropsOnlyEmptyOutcomes(t *testing.T) {
	p, st := headerFor(t, fixtureUbuntu)

	full := headerRows(p, st, 160)[0]
	require.Contains(t, full, "0 UNSCORED")

	narrow := headerRows(p, st, 46)[0]
	// The three that happened are never dropped, however little room there is.
	require.Contains(t, narrow, "2 FAIL")
	require.Contains(t, narrow, "4 ERROR")
	require.NotContains(t, narrow, "0 UNSCORED")
	require.NotContains(t, narrow, "0 SKIPPED")
}

// --- height -----------------------------------------------------------------

// HeightFor is a promise, and this is the test that keeps it. A band that asks
// for four lines and draws five is a band whose last row lands on the panel
// below it.
func TestHeightForMatchesTheLinesItEmits(t *testing.T) {
	for _, fixture := range []string{fixtureUbuntu, fixtureK8s} {
		for _, w := range headerWidths {
			for name, setup := range headerStates() {
				p, st := headerFor(t, fixture)
				setup(p, st)

				want := p.HeightFor(st, w)
				lines := p.Render(st, tui.Rect{W: w, H: MaxHeaderLines}).Lines

				assert.Len(t, lines, want, "%s %s w=%d", fixture, name, w)
				assert.LessOrEqual(t, want, MaxHeaderLines, "%s %s w=%d", fixture, name, w)
				for i, ln := range lines {
					assert.NotContains(t, ln, "\n", "%s %s w=%d line %d", fixture, name, w, i)
					assert.LessOrEqual(t, tui.Width(ln), w, "%s %s w=%d line %d", fixture, name, w, i)
				}
			}
		}
	}
}

// headerStates are the shapes the band takes, keyed by name so a failure says
// which one broke.
func headerStates() map[string]func(*headerPane, *State) {
	sev := func(st *State) {
		st.SetFilter(st.Filter.ToggleSeverity(policy.ScoreRatingTextCritical))
	}
	return map[string]func(*headerPane, *State){
		"idle": func(*headerPane, *State) {},
		"searching": func(p *headerPane, st *State) {
			st.Focus, p.searching = PaneHeader, true
		},
		"searching with a term": func(p *headerPane, st *State) {
			st.Focus, p.searching, p.search = PaneHeader, true, "ssh"
			p.publishSearch(st)
		},
		"chips open": func(p *headerPane, st *State) {
			st.Focus, p.open = PaneHeader, true
		},
		"filtered, closed": func(_ *headerPane, st *State) { sev(st) },
		"everything at once": func(p *headerPane, st *State) {
			st.Focus, p.open, p.searching, p.search = PaneHeader, true, true, "ssh"
			p.publishSearch(st)
			sev(st)
		},
	}
}

// The band grows when there is something to say and shrinks back when there is
// not. Idle it is one line, which is what leaves the body the room it needs.
func TestBandGrowsAndShrinks(t *testing.T) {
	p, st := headerFor(t, fixtureUbuntu)
	require.Equal(t, 1, p.HeightFor(st, 120), "idle: the summary and nothing else")

	st.Focus = PaneHeader
	p.searching = true
	require.Equal(t, 2, p.HeightFor(st, 120), "the search field")

	p.search = "permissions"
	p.publishSearch(st)
	require.Equal(t, 3, p.HeightFor(st, 120), "and the line that says what survives it")

	p.open = true
	require.Equal(t, 5, p.HeightFor(st, 120), "plus a row of chips per axis")

	p.searching, p.search = false, ""
	p.publishSearch(st)
	require.Equal(t, 3, p.HeightFor(st, 120), "the search field and its count both go")

	p.open = false
	require.Equal(t, 1, p.HeightFor(st, 120), "back to the summary")
}

// The tallest the band ever gets is every row at once, and it still fits inside
// what the frame allows. The sixth row is the severity note, which only a report
// with unscanned assets can produce.
func TestTallestBandFitsTheFrame(t *testing.T) {
	p, st := headerFor(t, fixtureK8s)
	st.Focus, p.open, p.searching, p.search = PaneHeader, true, true, "kube"
	p.publishSearch(st)
	st.SetFilter(st.Filter.ToggleSeverity(policy.ScoreRatingTextHigh))

	require.Equal(t, MaxHeaderLines, p.HeightFor(st, 160))
	rows := headerRows(p, st, 160)
	require.Len(t, rows, MaxHeaderLines)
	require.Contains(t, rows[1], "NOTE")
	require.Contains(t, rows[2], "FILTER")
	require.Contains(t, rows[3], "SEARCH")
	require.Contains(t, rows[4], "STATUS")
	require.Contains(t, rows[5], "SEVERITY")
}

// --- filtering never hides an unscanned asset -------------------------------

// The rule the whole feature turns on. Severity is a property a check has; an
// asset that never ran one has none to be judged by, so no severity filter may
// take it off the screen -- and the band has to say why those rows are still
// there, or the behaviour reads as a bug.
func TestSeverityFilterKeepsUnscannedAssetsAndSaysSo(t *testing.T) {
	p, st := headerFor(t, fixtureK8s)
	st.Focus, p.open, p.row = PaneHeader, true, rowSeverity

	for i := range AllSeverities {
		p.cursor = i
		p.toggle(st)
	}
	require.Len(t, st.Filter.Severities, len(AllSeverities))
	require.Len(t, st.FilteredAssets(), 15, "not one of them may be filtered away")

	rows := headerRows(p, st, 160)
	require.Contains(t, rows[0], "15 not scanned", "the summary never stops saying it")
	require.Contains(t, rows[0], "no checks ran")
	require.Contains(t, rows[1], "15 unscanned assets kept")
	require.Contains(t, rows[1], "severity describes a check")
	require.Contains(t, rows[2], "showing 15 of 15 assets")
	require.NotContains(t, rows[2], "checks", "a scan with no checks does not count them")
}

// The note is about unscanned assets, so a report where everything scanned does
// not carry it: it would be an explanation of something that did not happen.
func TestNoNoteWhenEverythingScanned(t *testing.T) {
	p, st := headerFor(t, fixtureUbuntu)
	st.SetFilter(st.Filter.ToggleSeverity(policy.ScoreRatingTextCritical))

	rows := headerRows(p, st, 160)
	require.Len(t, rows, 2)
	require.Contains(t, rows[1], "FILTER")
	require.NotContains(t, strings.Join(rows, "\n"), "unscanned")
}

// A status filter narrows the k8s report the way the model says it should, and
// the band's own count is the one the tree will draw from.
func TestFilterLineCountsWhatTheTreeShows(t *testing.T) {
	p, st := headerFor(t, fixtureUbuntu)
	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusError: true}})

	rows := headerRows(p, st, 160)
	require.Contains(t, rows[1], "showing 1 of 1 assets")
	require.Contains(t, rows[1], fmt.Sprintf("%d of 24 checks", st.Counts().Total))
	require.Equal(t, 4, st.Counts().Total)
	// The summary tally follows the filter, so the two halves of the band agree.
	require.Contains(t, rows[0], "4 ERROR")
	require.Contains(t, rows[0], "0 PASS")
}

// --- search -----------------------------------------------------------------

// "/" belongs to the header from anywhere, and what is typed into it narrows
// what the tree is looking at on every keystroke rather than on enter.
func TestSearchNarrowsFilteredAssets(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureK8s)), 120, 40)
	require.Equal(t, PaneTree, m.state.Focus)
	require.Len(t, m.state.FilteredAssets(), 15)

	m, _ = press(m, "/")
	require.Equal(t, PaneHeader, m.state.Focus, "the header claims / whatever has focus")

	m = typeInto(m, "kube-proxy")
	require.Equal(t, "kube-proxy", m.state.Filter.Search)

	narrowed := m.state.FilteredAssets()
	require.NotEmpty(t, narrowed)
	require.Less(t, len(narrowed), 15)
	for _, a := range narrowed {
		require.Contains(t, strings.ToLower(a.Name), "kube-proxy")
	}
	require.Contains(t, ansi.Strip(m.View()),
		fmt.Sprintf("showing %d of 15 assets", len(narrowed)))

	// Backspacing widens it again, one character at a time.
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = nm.(Model)
	require.Equal(t, "kube-prox", m.state.Filter.Search)
	require.GreaterOrEqual(t, len(m.state.FilteredAssets()), len(narrowed))
}

// A search must not fight the frame for the alphabet: every printable key is a
// character while the field is open, including the ones that otherwise quit,
// open the search or open the chips.
func TestFrameKeysAreTextWhileSearching(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	m, _ = press(m, "/")

	m, cmd := press(m, "q")
	require.Nil(t, cmd, "q is a character here, not a quit")
	m = typeInto(m, "f/ x")

	require.Equal(t, "qf/ x", m.state.Filter.Search)
	require.True(t, m.header.(*headerPane).searching, "and the field is still open")
}

// esc clears the search and hands focus back, rather than quitting. It only gets
// the chance because the header is focusable while the field is open.
func TestEscClearsTheSearchRatherThanQuitting(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureK8s)), 120, 40)
	m, _ = press(m, "/")
	m = typeInto(m, "kube-proxy")
	require.Less(t, len(m.state.FilteredAssets()), 15)

	m, cmd := press(m, "esc")
	require.Nil(t, cmd, "esc must not quit while a search is open")
	require.Empty(t, m.state.Filter.Search)
	require.Len(t, m.state.FilteredAssets(), 15)
	require.Equal(t, PaneTree, m.state.Focus, "focus goes back where it came from")
	require.Equal(t, 1, m.headerHeight(), "and the band shrinks back to one line")

	// Nothing is open and nothing is filtered, so esc is the frame's again.
	_, cmd = press(m, "esc")
	require.NotNil(t, cmd, "esc quits once there is nothing left to back out of")
}

// enter closes the field and keeps the term: the search is committed as it is
// typed, so there is nothing left for enter to apply.
func TestEnterClosesTheSearchAndKeepsIt(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureK8s)), 120, 40)
	m, _ = press(m, "/")
	m = typeInto(m, "kube")
	m, _ = press(m, "enter")

	require.Equal(t, "kube", m.state.Filter.Search)
	require.False(t, m.header.(*headerPane).searching)
	require.Equal(t, PaneHeader, m.state.Focus,
		"a filter is still set, so the band keeps focus and esc can clear it")
	require.Equal(t, 2, m.headerHeight(), "the summary and what survives the filter")
}

// --- chips ------------------------------------------------------------------

// "f" reaches the chips from the body, and space toggles the one under the
// cursor through the frame's own vocabulary rather than a second idea of what a
// filter means.
func TestChipKeysToggleTheFilter(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)

	m, _ = press(m, "f")
	require.Equal(t, PaneHeader, m.state.Focus)
	require.Equal(t, 3, m.headerHeight(), "the summary plus a row per axis")

	// The cursor opens on the first status, which is FAIL.
	m, _ = press(m, " ")
	require.True(t, m.state.Filter.Statuses[reportmodel.StatusFail])
	require.Equal(t, 2, m.state.Counts().Total, "24 checks, 2 of which failed")

	// Right moves along the row; down moves to the severity axis.
	m, _ = press(m, "right")
	m, _ = press(m, " ")
	require.True(t, m.state.Filter.Statuses[reportmodel.StatusError])
	require.Equal(t, 6, m.state.Counts().Total, "fail and error, kept apart and added up by nobody")

	// Down moves to the severity axis and keeps the column, the way a grid does.
	m, _ = press(m, "down")
	require.Equal(t, 1, m.header.(*headerPane).cursor)
	m, _ = press(m, "home")
	m, _ = press(m, " ")
	require.True(t, m.state.Filter.Severities[policy.ScoreRatingTextCritical])

	// The two axes are combined with AND, and the outcomes within the status axis are not
	// merged: what is left is the critical checks that failed or errored.
	left := m.state.FilteredChecks(m.state.Report.Assets[0].Checks)
	require.NotEmpty(t, left)
	require.Equal(t, len(left), m.state.Counts().Total)
	for _, c := range left {
		require.Equal(t, policy.ScoreRatingTextCritical, c.Severity)
		require.True(t, c.Status.IsFinding())
	}

	// Toggling the same chip again takes it off.
	m, _ = press(m, " ")
	require.Empty(t, m.state.Filter.Severities)
}

// A chip's zone has to name the cells the chip was drawn in, or a click lands on
// the neighbour. Both come out of one Render, and this is what pins them
// together.
func TestChipZonesLandOnTheChipTheyName(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 160, 40)
	m, _ = press(m, "f")

	f := m.build()
	rendered := f.rendered[PaneHeader]
	require.NotEmpty(t, rendered.Zones)

	band := f.layout.Header
	seen := map[string]int{}
	for _, z := range rendered.Zones {
		require.Equal(t, PaneHeader, z.Pane)
		require.True(t, z.Rect.Hit(z.Rect.X, z.Rect.Y), "a zone covers its own corner")
		require.GreaterOrEqual(t, z.Rect.Y, band.Y)
		require.Less(t, z.Rect.Y, band.Y+band.H)
		require.LessOrEqual(t, z.Rect.X+z.Rect.W, band.X+band.W, "a zone stays inside the band")
		seen[z.Tag]++

		// The cells the zone claims are the cells the chip occupies.
		row := []rune(ansi.Strip(rendered.Lines[z.Rect.Y-band.Y]))
		require.LessOrEqual(t, z.Rect.X+z.Rect.W, len(row))
		text := strings.TrimSpace(string(row[z.Rect.X : z.Rect.X+z.Rect.W]))
		if z.Tag == "severity" {
			require.Equal(t, fmt.Sprintf("%s ", AllSeverities[z.Idx]), text[:len(AllSeverities[z.Idx])+1])
		}
	}
	require.Equal(t, len(AllSeverities), seen["severity"])
	require.Positive(t, seen["status"])
}

// Clicking a chip toggles it and moves the cursor onto it, so the keyboard picks
// up where the mouse left off.
func TestClickingAChipTogglesIt(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 160, 40)
	m, _ = press(m, "f")

	var target Zone
	for _, z := range m.build().rendered[PaneHeader].Zones {
		if z.Tag == "severity" && z.Idx == 1 {
			target = z
		}
	}
	require.Equal(t, "severity", target.Tag)

	nm, _ := m.Update(tea.MouseMsg{
		X: target.Rect.X, Y: target.Rect.Y,
		Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m = nm.(Model)

	require.True(t, m.state.Filter.Severities[AllSeverities[1]])
	require.Equal(t, PaneHeader, m.state.Focus)

	p := m.header.(*headerPane)
	require.Equal(t, rowSeverity, p.row)
	require.Equal(t, 1, p.cursor)
}

// A closed band answers no clicks at all, which is what keeps a click on the top
// row of an idle viewer from doing something invisible.
func TestClosedBandHasNoZones(t *testing.T) {
	for _, fixture := range []string{fixtureUbuntu, fixtureK8s} {
		m := sized(NewModel(loadReport(t, fixture)), 120, 40)
		for _, z := range m.build().zones {
			require.NotEqual(t, PaneHeader, z.Pane, "%s: an idle band is not clickable", fixture)
		}
	}
}

// --- focus and the esc ladder -----------------------------------------------

// The band stays out of the tab cycle until it has something to drive, and
// rejoins it the moment it does.
func TestBandJoinsTheTabCycleOnlyWhenItHasSomethingToDo(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	require.False(t, m.header.Focusable())

	m, _ = press(m, "tab")
	require.Equal(t, PaneDetail, m.state.Focus)
	m, _ = press(m, "tab")
	require.Equal(t, PaneTree, m.state.Focus, "an idle band is skipped")

	m, _ = press(m, "f")
	require.True(t, m.header.Focusable())
	m, _ = press(m, "tab")
	require.Equal(t, PaneTree, m.state.Focus)
	m, _ = press(m, "tab")
	m, _ = press(m, "tab")
	require.Equal(t, PaneHeader, m.state.Focus, "an open band is in the cycle")
}

// esc backs out one step at a time and only quits once there is nothing left to
// back out of: close the chips, clear the filter, then quit.
func TestEscLadder(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)

	m, _ = press(m, "f")
	m, _ = press(m, " ")
	require.True(t, m.state.Filter.Active())

	m, cmd := press(m, "esc")
	require.Nil(t, cmd, "the first esc closes the chips")
	require.False(t, m.header.(*headerPane).open)
	require.True(t, m.state.Filter.Active(), "and leaves the filter alone")
	require.Equal(t, PaneHeader, m.state.Focus, "the band still holds a filter, so it keeps focus")

	m, cmd = press(m, "esc")
	require.Nil(t, cmd, "the second esc clears the filter")
	require.False(t, m.state.Filter.Active())
	require.Equal(t, PaneTree, m.state.Focus)

	_, cmd = press(m, "esc")
	require.NotNil(t, cmd, "the third esc quits")
}

// "c" clears every axis at once without closing anything, which is the escape
// hatch from a filter that has been narrowed into an empty screen.
func TestClearKeyEmptiesEveryAxis(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	m, _ = press(m, "/")
	m = typeInto(m, "zzz")
	m, _ = press(m, "enter")
	m, _ = press(m, "f")
	m, _ = press(m, " ")
	require.True(t, m.state.Filter.Active())
	require.Empty(t, m.state.FilteredAssets())

	m, _ = press(m, "c")
	require.False(t, m.state.Filter.Active())
	require.Empty(t, m.state.Filter.Search)
	require.Len(t, m.state.FilteredAssets(), 1)
	require.True(t, m.header.(*headerPane).open, "clearing is not closing")
}

// Tabbing away from an open search closes the field rather than leaving it
// silently swallowing keys it will never see.
func TestTabbingAwayClosesTheSearchField(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	m, _ = press(m, "/")
	m = typeInto(m, "ssh")
	require.True(t, m.header.(*headerPane).searching)

	m, _ = press(m, "tab")
	_ = m.View() // the band reconciles with the frame's focus as it draws
	require.False(t, m.header.(*headerPane).searching)
	require.Equal(t, "ssh", m.state.Filter.Search, "the term it committed survives")
}

// --- wiring -----------------------------------------------------------------

// The header is built once per model and may be handed a state that already
// carries a filter, e.g. from a caller that preselected one.
func TestHeaderAdoptsAPresetFilter(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	st.SetFilter(Filter{Search: "ssh"})

	p, ok := buildPane(PaneHeader, st).(*headerPane)
	require.True(t, ok)
	require.Equal(t, "ssh", p.search, "the field opens on the term already in force")

	rows := headerRows(p, st, 160)
	require.Len(t, rows, 2)
	require.Contains(t, rows[1], `"ssh"`)
	require.True(t, p.Focusable(), "a filter is set, so esc has somewhere to go")
}

// The band survives a report with no assets at all rather than dividing by the
// number of them.
func TestEmptyReportBand(t *testing.T) {
	st := NewState(reportmodel.New(nil))
	p := &headerPane{returnTo: PaneTree}

	require.Equal(t, 1, p.HeightFor(st, 80))
	require.Contains(t, headerRows(p, st, 80)[0], "0 assets")
	require.Contains(t, headerRows(p, st, 80)[0], "no checks ran")
}

// typeInto sends each character of s as its own keypress, which is what the
// terminal does.
func typeInto(m Model, s string) Model {
	for _, r := range s {
		m, _ = press(m, string(r))
	}
	return m
}
