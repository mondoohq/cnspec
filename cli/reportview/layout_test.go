// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/policy"
)

// The fixtures live with the reporter tests, which is where they came from.
// report-ubuntu is one asset that scanned; report-k8s is fifteen that did not,
// with no bundle at all. Both are raw collections rather than json-full
// artifacts, so they are read with json.Unmarshal the way the reporter tests
// read them.
const (
	fixtureUbuntu = "../reporter/testdata/report-ubuntu.json"
	fixtureK8s    = "../reporter/testdata/report-k8s.json"
)

func loadReport(t *testing.T, path string) *reportmodel.Report {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	collection := &policy.ReportCollection{}
	require.NoError(t, json.Unmarshal(raw, collection))
	return reportmodel.New(collection)
}

// termSizes covers the sizes that actually break things: the 80x24 default, a
// narrow window that has to drop to one pane, and a window so short that the
// header, the body and the footer cannot all have the room they want.
var termSizes = []struct{ w, h int }{
	{80, 24}, {100, 30}, {120, 40}, {200, 60}, {76, 24}, {70, 24}, {60, 15}, {40, 10}, {30, 6}, {20, 3},
}

func sized(m Model, w, h int) Model {
	nm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return nm.(Model)
}

func viewLines(m Model) []string {
	return strings.Split(m.View(), "\n")
}

// TestViewIsExactlyTerminalSize is the test that keeps the viewer inside the
// terminal. A view taller than the screen scrolls the alt-screen buffer, which
// pushes the header off and leaves the cursor pointing at rows nobody can see; a
// line wider than the screen wraps and does the same thing one row at a time.
func TestViewIsExactlyTerminalSize(t *testing.T) {
	for _, fixture := range []string{fixtureUbuntu, fixtureK8s} {
		report := loadReport(t, fixture)
		for _, focus := range []PaneID{PaneTree, PaneDetail} {
			for _, s := range termSizes {
				m := sized(NewModel(report), s.w, s.h)
				m.state.Focus = focus
				lines := viewLines(m)
				if len(lines) != s.h {
					t.Errorf("%s focus=%s %dx%d: rendered %d lines, want exactly %d",
						fixture, focus, s.w, s.h, len(lines), s.h)
				}
				for i, ln := range lines {
					if w := ansi.StringWidth(ln); w > s.w {
						t.Errorf("%s focus=%s %dx%d: line %d is %d cells wide, want <= %d",
							fixture, focus, s.w, s.h, i, w, s.w)
					}
				}
			}
		}
	}
}

// The same at the end of a long list: the last page is where an off-by-one in
// the scroll clamp shows up.
func TestViewIsExactlyTerminalSizeAtEndOfList(t *testing.T) {
	report := loadReport(t, fixtureK8s)
	for _, s := range termSizes {
		m := sized(NewModel(report), s.w, s.h)
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		m = nm.(Model)
		if lines := viewLines(m); len(lines) != s.h {
			t.Errorf("%dx%d at end of list: rendered %d lines, want exactly %d", s.w, s.h, len(lines), s.h)
		}
	}
}

// A growing terminal must not leave a pane scrolled past the end of its rows.
func TestScrollClampsOnResize(t *testing.T) {
	report := loadReport(t, fixtureK8s)
	// The placeholder is passed in explicitly: this is a frame test about the
	// scroll contract, so it names the pane it drives rather than taking
	// whichever one happens to be registered.
	m := sized(NewModel(report, &assetListPane{}), 120, 12)
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = nm.(Model)
	_ = m.View() // Render is what scrolls: it is the only place that knows how many rows fit

	tree := m.tree.(*assetListPane)
	require.Positive(t, tree.scroll.Off, "the k8s fixture should be long enough to scroll at height 12")

	m = sized(m, 120, 200)
	_ = m.View() // Render is what clamps
	require.Equal(t, 0, tree.scroll.Off, "everything fits now, so the list must be back at the top")
	require.Len(t, viewLines(m), 200)
}

// TestLayoutAccountsForEveryLine checks the arithmetic directly rather than
// through the renderer: header plus body plus footer is the terminal, exactly.
func TestLayoutAccountsForEveryLine(t *testing.T) {
	for _, s := range termSizes {
		for _, want := range []int{1, 2, 3, MaxHeaderLines, MaxHeaderLines + 10} {
			l := computeLayout(s.w, s.h, want, PaneTree)

			require.GreaterOrEqual(t, l.HeaderH, 1, "%dx%d header=%d", s.w, s.h, want)
			require.LessOrEqual(t, l.HeaderH, MaxHeaderLines, "%dx%d header=%d", s.w, s.h, want)
			require.Equal(t, l.HeaderH, l.Header.H)

			// The body is what is left, unless the terminal is too short to
			// leave anything, in which case the body claims its minimum and the
			// renderer clips.
			leftover := s.h - l.HeaderH - tui.FooterLines
			if leftover >= MinBodyLines {
				require.Equal(t, leftover, l.BodyH, "%dx%d header=%d", s.w, s.h, want)
				require.Equal(t, s.h, l.HeaderH+l.BodyH+tui.FooterLines)
			} else {
				require.Equal(t, MinBodyLines, l.BodyH, "%dx%d header=%d", s.w, s.h, want)
			}

			require.Equal(t, l.BodyTop, l.HeaderH, "the body starts where the header ends")
			require.Equal(t, s.h-1, l.Footer.Y, "the footer is the last line")
		}
	}
}

// The two panels must tile the width exactly: left, one column of gutter, right.
func TestPanelsTileTheWidth(t *testing.T) {
	for _, s := range termSizes {
		l := computeLayout(s.w, s.h, 1, PaneTree)
		if !l.TwoPane {
			require.Equal(t, s.w, l.TreePanel.W, "%dx%d: a single pane takes the whole width", s.w, s.h)
			require.Equal(t, s.w, l.DetailPanel.W)
			continue
		}
		require.Equal(t, s.w, l.TreePanel.W+1+l.DetailPanel.W, "%dx%d", s.w, s.h)
		require.Equal(t, l.TreePanel.W+1, l.DetailPanel.X, "%dx%d", s.w, s.h)
	}
}

// The content rect a pane renders into is the inside of its border, and the
// panel it lives in is drawn at exactly the size the layout gave it.
func TestContentRectIsInsideTheBorder(t *testing.T) {
	l := computeLayout(120, 40, 2, PaneTree)

	require.Equal(t, tui.InnerX, l.TreeContent.X)
	require.Equal(t, l.TreePanel.Y+1, l.TreeContent.Y)
	require.Equal(t, tui.InnerWidth(l.TreePanel.W), l.TreeContent.W)
	require.Equal(t, tui.InnerHeight(l.TreePanel.H), l.TreeContent.H)

	require.Equal(t, l.DetailPanel.X+tui.InnerX, l.DetailContent.X)
	require.Equal(t, tui.InnerWidth(l.DetailPanel.W), l.DetailContent.W)

	// The accessors a pane reaches the geometry through agree with the fields.
	for _, id := range []PaneID{PaneHeader, PaneTree, PaneDetail} {
		rect, visible := l.ContentFor(id)
		require.True(t, visible, "%s should be drawn at 120x40", id)
		require.Positive(t, rect.W)
		require.Positive(t, rect.H)
	}
	tree, ok := l.PanelFor(PaneTree)
	require.True(t, ok)
	require.Equal(t, l.TreePanel, tree)
	require.Equal(t, Content(tree), l.TreeContent)

	_, ok = l.PanelFor(PaneHeader)
	require.False(t, ok, "the header band has no bordered panel")
	_, ok = l.ContentFor(PaneNone)
	require.False(t, ok)
}

// PaneAt is what routes a scroll wheel: it answers with the pane the pointer is
// over, and with nothing when the pointer is on the footer.
func TestPaneAt(t *testing.T) {
	l := computeLayout(120, 40, 1, PaneTree)

	require.Equal(t, PaneHeader, l.PaneAt(0, 0))
	require.Equal(t, PaneTree, l.PaneAt(l.TreeContent.X, l.TreeContent.Y))
	require.Equal(t, PaneDetail, l.PaneAt(l.DetailContent.X, l.DetailContent.Y))
	require.Equal(t, PaneNone, l.PaneAt(0, l.Footer.Y))

	// In one-pane mode only the pane on screen answers.
	narrow := computeLayout(60, 20, 1, PaneTree)
	require.Equal(t, PaneTree, narrow.PaneAt(2, narrow.BodyTop+1))
	narrow = computeLayout(60, 20, 1, PaneDetail)
	require.Equal(t, PaneDetail, narrow.PaneAt(2, narrow.BodyTop+1))
}

// Below tui.MinTwoPaneWidth the viewer shows one pane, following focus, rather
// than squeezing two unreadable columns into the window.
func TestNarrowTerminalUsesOnePane(t *testing.T) {
	report := loadReport(t, fixtureUbuntu)
	m := sized(NewModel(report), 60, 20)

	l := m.layout()
	require.False(t, l.TwoPane)
	require.True(t, l.ShowTree)
	require.False(t, l.ShowDetail)
	require.Contains(t, ansi.Strip(m.View()), "ubuntu:24.04")

	m.state.Focus = PaneDetail
	l = m.layout()
	require.False(t, l.ShowTree)
	require.True(t, l.ShowDetail)
	// The asset page's status row, which is the one thing every detail page has.
	require.Contains(t, ansi.Strip(m.View()), "risk ")
}

// At exactly tui.MinTwoPaneWidth both panes appear; one column narrower they do
// not. The boundary is worth pinning because both sides of it are drawn by
// different code paths.
func TestTwoPaneBoundary(t *testing.T) {
	report := loadReport(t, fixtureUbuntu)

	wide := sized(NewModel(report), tui.MinTwoPaneWidth, 24)
	require.True(t, wide.layout().TwoPane)
	require.Len(t, viewLines(wide), 24)

	narrow := sized(NewModel(report), tui.MinTwoPaneWidth-1, 24)
	require.False(t, narrow.layout().TwoPane)
	require.Len(t, viewLines(narrow), 24)
}

// A header that asks for more than it can have gets clamped, and the view still
// occupies exactly the terminal.
func TestGreedyHeaderIsClamped(t *testing.T) {
	report := loadReport(t, fixtureUbuntu)
	for _, s := range termSizes {
		m := sized(NewModel(report, &greedyHeader{want: 99}), s.w, s.h)
		l := m.layout()
		require.LessOrEqual(t, l.HeaderH, MaxHeaderLines, "%dx%d", s.w, s.h)
		require.GreaterOrEqual(t, l.BodyH, MinBodyLines, "%dx%d", s.w, s.h)
		require.Len(t, viewLines(m), s.h, "%dx%d", s.w, s.h)
	}
}

// greedyHeader is a header pane that always asks for more lines than it should
// get, and fills every one of them.
type greedyHeader struct{ want int }

func (p *greedyHeader) ID() PaneID                             { return PaneHeader }
func (p *greedyHeader) Focusable() bool                        { return false }
func (p *greedyHeader) Claims() []string                       { return nil }
func (p *greedyHeader) Hints(*State) []Hint                    { return nil }
func (p *greedyHeader) Update(*State, tea.Msg) (tea.Cmd, bool) { return nil, false }
func (p *greedyHeader) HeightFor(*State, int) int              { return p.want }
func (p *greedyHeader) Render(_ *State, rect tui.Rect) Render {
	lines := make([]string, p.want)
	for i := range lines {
		lines[i] = strings.Repeat("H", rect.W+20) // deliberately too wide
	}
	return Render{Lines: lines}
}
