// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func press(m Model, s string) (Model, tea.Cmd) {
	nm, cmd := m.Update(key(s))
	return nm.(Model), cmd
}

// TestZonesMatchRender is the test that proves the click map and the picture
// agree. Both come out of one Render call per pane, and this is what keeps them
// from drifting: the row a zone names must be the row the renderer drew there.
func TestZonesMatchRender(t *testing.T) {
	report := loadReport(t, fixtureK8s)
	// The placeholder goes in explicitly: this is the frame's contract, so the
	// test names the pane whose row format it asserts against rather than
	// taking whichever one happens to be registered. The registered tree proves
	// the same property over its own rows in tree_test.go.
	m := sized(NewModel(report, &assetListPane{}), 120, 40)

	f := m.build()
	lines := viewLines(m)
	require.NotEmpty(t, f.zones)

	assets := m.state.FilteredAssets()
	// A row is " " + the status label + " " + the name, truncated to the pane. A
	// long name is cut, so the assertion is against as much of it as fits.
	fits := f.layout.TreeContent.W - statusLabelWidth - 2

	for _, z := range f.zones {
		require.Equal(t, PaneTree, z.Pane)
		require.Equal(t, "asset", z.Tag)
		require.GreaterOrEqual(t, z.Rect.Y, f.layout.TreeContent.Y, "a zone must lie inside the pane")
		require.Less(t, z.Rect.Y, f.layout.TreeContent.Y+f.layout.TreeContent.H)
		require.Equal(t, f.layout.TreeContent.X, z.Rect.X)

		got := ansi.Strip(lines[z.Rect.Y])
		want := []rune(assets[z.Idx].Name)
		if len(want) > fits {
			want = want[:fits-1] // one cell goes to the ellipsis
		}
		require.Contains(t, got, string(want), "zone at y=%d claims asset %d (%q)",
			z.Rect.Y, z.Idx, assets[z.Idx].Name)
	}
}

// A click selects what it landed on, and gives the pane that owns the zone the
// keyboard.
func TestClickSelectsAsset(t *testing.T) {
	report := loadReport(t, fixtureK8s)
	m := sized(NewModel(report), 120, 40)
	m.state.Focus = PaneDetail

	f := m.build()
	var target Zone
	for _, z := range f.zones {
		if z.Idx == 3 {
			target = z
		}
	}
	require.Equal(t, 3, target.Idx, "expected a zone for the fourth asset")

	nm, _ := m.Update(tea.MouseMsg{
		X: target.Rect.X, Y: target.Rect.Y,
		Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m = nm.(Model)

	require.Equal(t, PaneTree, m.state.Focus, "the click moves focus to the pane it landed in")
	require.Equal(t, m.state.FilteredAssets()[3], m.state.Sel.Asset)
}

// A click outside every zone changes nothing.
func TestClickOutsideAnyZone(t *testing.T) {
	report := loadReport(t, fixtureUbuntu)
	m := sized(NewModel(report), 120, 40)
	before := m.state.SelectionRev

	nm, _ := m.Update(tea.MouseMsg{
		X: 0, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	require.Equal(t, before, nm.(Model).state.SelectionRev)
}

// --- errored assets ---------------------------------------------------------

// An asset that failed to scan is not an asset with no findings. The k8s fixture
// is fifteen assets, no reports and no bundle at all: the viewer has to show
// fifteen rows and say why each of them is empty, not draw an empty screen.
func TestErroredAssetsAreVisible(t *testing.T) {
	report := loadReport(t, fixtureK8s)
	require.Len(t, report.Assets, 15)
	require.Equal(t, 15, report.AssetCounts.Errored)

	m := sized(NewModel(report), 120, 40)
	f := m.build()
	out := ansi.Strip(m.View())

	// Every asset is drawn and every one of them is clickable: fifteen rows,
	// fifteen zones, one per asset, none of them skipped.
	require.Len(t, f.rendered[PaneTree].Lines, 15)
	require.Len(t, f.zones, 15)
	seen := map[int]bool{}
	for _, z := range f.zones {
		seen[z.Idx] = true
	}
	require.Len(t, seen, 15)

	// Every row says ERROR rather than looking like a clean asset -- as the
	// glyph the tree draws an outcome with, which is a shape and not a color, so
	// this holds on a terminal with none.
	for _, ln := range f.rendered[PaneTree].Lines {
		require.Contains(t, ansi.Strip(ln), statusGlyph(reportmodel.StatusError))
	}

	// The header says how many never scanned.
	require.Contains(t, out, "15 not scanned")
	// And the detail pane says why, rather than showing a clean bill of health.
	require.Contains(t, out, "THIS ASSET WAS NOT SCANNED")
	require.Contains(t, out, "asset doesn't support any")
	require.NotContains(t, out, "0 total")
	require.NotContains(t, out, "no assets match")
}

// The single-asset happy path still shows its findings.
func TestScannedAssetShowsFindings(t *testing.T) {
	report := loadReport(t, fixtureUbuntu)
	require.Len(t, report.Assets, 1)
	require.True(t, report.Assets[0].Scanned())

	m := sized(NewModel(report), 160, 50)
	out := ansi.Strip(m.View())

	require.Contains(t, out, "ubuntu:24.04")
	require.Contains(t, out, "FINDINGS")
	require.NotContains(t, out, "THIS ASSET WAS NOT SCANNED")

	// The four outcomes of this fixture stay four outcomes.
	c := report.Assets[0].Counts
	require.Equal(t, 24, c.Total)
	require.Equal(t, 18, c.Passed)
	require.Equal(t, 2, c.Failed)
	require.Equal(t, 4, c.Errored)
}

// --- routing and focus ------------------------------------------------------

// Tab cycles the focusable panes and skips the ones that only display.
func TestTabCyclesFocusablePanes(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	require.Equal(t, PaneTree, m.state.Focus, "the tree has focus on open")

	m, _ = press(m, "tab")
	require.Equal(t, PaneDetail, m.state.Focus)
	m, _ = press(m, "tab")
	require.Equal(t, PaneTree, m.state.Focus, "the summary header is not focusable, so it is skipped")
	m, _ = press(m, "shift+tab")
	require.Equal(t, PaneDetail, m.state.Focus)
}

// A key the focused pane consumes never reaches the frame, and a key it does not
// consume always does.
func TestKeyFallsThroughToTheFrame(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)

	// "down" belongs to the tree; the frame must not also act on it.
	m, cmd := press(m, "down")
	require.Nil(t, cmd)
	require.Equal(t, PaneTree, m.state.Focus)

	// "?" belongs to nobody, so the frame takes it.
	require.False(t, m.showHelp)
	m, _ = press(m, "?")
	require.True(t, m.showHelp)
	require.Contains(t, ansi.Strip(m.View()), "previous pane")

	// "q" belongs to nobody either, and the frame quits.
	_, cmd = press(m, "q")
	require.NotNil(t, cmd, "q must quit")
}

// The ? hint is labelled for what it gets you. Every other label on the compact
// line -- pane, copy, export, quit -- names an outcome, and "keys" named the
// mechanism instead: a reader who is stuck wants help, and would not think to
// look for a list of keys. The expanded form stays "less", which names the
// direction of the toggle on the one line that is already truncating.
func TestTheHelpHintOffersHelp(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 200, 40)

	label := func(m Model, key string) string {
		for _, h := range m.frameHints() {
			if h.Key == key {
				return h.Label
			}
		}
		return ""
	}

	require.Equal(t, "help", label(m, "?"))
	require.Contains(t, ansi.Strip(m.View()), "? help")
	require.NotContains(t, ansi.Strip(m.View()), "? keys")

	m, _ = press(m, "?")
	require.True(t, m.showHelp)
	require.Equal(t, "less", label(m, "?"))
	// Asserted on the hint rather than on the frame: the expanded line is long
	// enough that even 200 columns truncate it before ?, which is the whole
	// reason its label is not lengthened to repeat the noun.
	require.NotContains(t, ansi.Strip(m.View()), "? keys")
}

// A pane that claims a key gets focus and the key, even when the focus was
// somewhere else. This is how a filter pane owns "/" without the frame knowing
// what a filter is.
func TestClaimedKeyMovesFocusAndIsDelivered(t *testing.T) {
	claimer := &claimingPane{}
	m := sized(NewModel(loadReport(t, fixtureUbuntu), claimer), 120, 40)
	require.Equal(t, PaneTree, m.state.Focus)

	m, _ = press(m, "/")
	require.Equal(t, PaneDetail, m.state.Focus, "the claiming pane takes focus")
	require.Equal(t, 1, claimer.claimed, "and receives the key exactly once")

	// Once it has focus the ordinary path delivers it, still exactly once.
	m, _ = press(m, "/")
	require.Equal(t, 2, claimer.claimed)
}

// A pane that refuses a key it claims lets the frame have it.
func TestUnhandledClaimStillFallsThrough(t *testing.T) {
	claimer := &claimingPane{refuse: true}
	m := sized(NewModel(loadReport(t, fixtureUbuntu), claimer), 120, 40)

	_, cmd := press(m, "q")
	require.NotNil(t, cmd, "the pane refused q, so the frame quits")
}

// ctrl+c always quits, whatever a pane thinks.
func TestCtrlCAlwaysQuits(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu), &claimingPane{}), 120, 40)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)
}

// The wheel scrolls whatever the pointer is over and leaves focus alone: taking
// the keyboard because someone looked at a pane would be a surprise.
func TestWheelDoesNotStealFocus(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureK8s)), 120, 40)
	l := m.layout()

	nm, _ := m.Update(tea.MouseMsg{
		X: l.DetailContent.X, Y: l.DetailContent.Y,
		Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown,
	})
	m = nm.(Model)
	require.Equal(t, PaneTree, m.state.Focus)
	require.Equal(t, PaneDetail, l.PaneAt(l.DetailContent.X, l.DetailContent.Y))
}

// A message that is neither a key nor a mouse event reaches every pane, which is
// how a pane can react to something the frame knows nothing about.
func TestBroadcast(t *testing.T) {
	spy := &claimingPane{}
	m := sized(NewModel(loadReport(t, fixtureUbuntu), spy), 120, 40)
	require.Equal(t, 1, spy.broadcasts, "the resize itself is a broadcast")

	m.Update(struct{ custom string }{"hello"})
	require.Equal(t, 2, spy.broadcasts)
}

// --- selection --------------------------------------------------------------

// The selection revision is what a detail pane watches to know it should scroll
// back to the top; re-selecting the same thing must not bump it.
func TestSelectionRevision(t *testing.T) {
	st := NewState(loadReport(t, fixtureK8s))
	require.Equal(t, 1, st.SelectionRev, "opening on the first asset is a selection")

	first := st.Sel.Asset
	st.SelectAsset(first)
	require.Equal(t, 1, st.SelectionRev, "re-selecting the same asset changes nothing")

	st.SelectAsset(st.Report.Assets[1])
	require.Equal(t, 2, st.SelectionRev)
}

// Moving the cursor in the tree is what the detail pane draws.
func TestTreeSelectionDrivesDetail(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureK8s)), 120, 40)
	assets := m.state.FilteredAssets()

	m, _ = press(m, "down")
	require.Equal(t, assets[1], m.state.Sel.Asset)
	require.Contains(t, ansi.Strip(m.View()), assets[1].Name)

	// Enter hands focus over to the detail pane, which is where the reading is.
	m, _ = press(m, "enter")
	require.Equal(t, PaneDetail, m.state.Focus)
}

// An empty report renders rather than panicking, and says so.
func TestEmptyReport(t *testing.T) {
	m := sized(NewModel(reportmodel.New(nil)), 100, 30)
	out := ansi.Strip(m.View())
	require.Len(t, viewLines(m), 30)
	require.Contains(t, out, "no assets match")
	require.Contains(t, out, "nothing selected")
	require.Nil(t, m.state.Sel.Asset)
}

// A zero-size terminal draws nothing at all rather than one stray line.
func TestZeroSizeRendersNothing(t *testing.T) {
	require.Empty(t, NewModel(loadReport(t, fixtureUbuntu)).View())
}

// Passing a pane replaces whatever that slot would otherwise hold and leaves
// the other two alone. The other two are whatever the registry says, which is a
// real pane once one has registered itself and the placeholder until then.
func TestPaneOverrideReplacesOneSlot(t *testing.T) {
	m := NewModel(loadReport(t, fixtureUbuntu), &claimingPane{})
	require.IsType(t, &claimingPane{}, m.detail)
	require.IsType(t, buildPane(PaneTree, m.state), m.tree)
	require.IsType(t, buildPane(PaneHeader, m.state), m.header)
}

// A slot takes one pane. Two panes fighting over one is a wiring mistake, and it
// has to fail at init rather than resolve itself by init order -- otherwise
// which pane you get depends on the order the files happen to be compiled in.
//
// The test uses a slot of its own so it cannot disturb the real three.
func TestRegisterRejectsDuplicates(t *testing.T) {
	const slot = PaneID(99)
	t.Cleanup(func() { delete(registry, slot) })

	want := &claimingPane{}
	register(slot, func(*State) Pane { return want })
	require.Same(t, want, buildPane(slot, nil))

	require.Panics(t, func() { register(slot, func(*State) Pane { return nil }) })
	require.Panics(t, func() { register(PaneID(98), nil) })
}

// An unregistered slot falls back to its placeholder, which is what makes the
// frame runnable before any pane exists.
func TestUnregisteredSlotFallsBackToPlaceholder(t *testing.T) {
	// The fallback is what is under test, not which panes happen to be
	// compiled in, so the registry is emptied for the duration.
	saved := registry
	registry = map[PaneID]PaneFactory{}
	t.Cleanup(func() { registry = saved })

	require.IsType(t, &summaryPane{}, buildPane(PaneHeader, nil))
	require.IsType(t, &assetListPane{}, buildPane(PaneTree, nil))
	require.IsType(t, &plainDetailPane{}, buildPane(PaneDetail, nil))
	require.Nil(t, buildPane(PaneNone, nil))
}

// claimingPane is a detail-slot pane that claims "/" and counts what it is sent.
type claimingPane struct {
	refuse     bool
	claimed    int
	broadcasts int
}

func (p *claimingPane) ID() PaneID       { return PaneDetail }
func (p *claimingPane) Focusable() bool  { return true }
func (p *claimingPane) Claims() []string { return []string{"/", "q"} }
func (p *claimingPane) Hints(*State) []Hint {
	return []Hint{{Key: "/", Label: "search"}}
}

func (p *claimingPane) Render(_ *State, rect tui.Rect) Render {
	return Render{Title: "Claim", Lines: []string{strings.Repeat("x", rect.W)}}
}

func (p *claimingPane) Update(_ *State, msg tea.Msg) (tea.Cmd, bool) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		p.broadcasts++
		return nil, false
	}
	if p.refuse {
		return nil, false
	}
	if k.String() == "/" {
		p.claimed++
		return nil, true
	}
	return nil, false
}
