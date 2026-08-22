// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
)

// The placeholder panes. Each slot falls back to one of these until a real pane
// registers itself, which is what makes the frame runnable on its own: the
// command works end to end today and each pane is swapped in without touching
// anything else.
//
// They are deliberately thin -- an asset list, a summary line, a plain dump of
// the selection -- but they are not fake. They select, they scroll, they answer
// clicks and they show an errored asset as an error, so the seam is exercised by
// something real rather than by a box that says "tree pane goes here".

func newPlaceholder(id PaneID, _ *State) Pane {
	switch id {
	case PaneHeader:
		return &summaryPane{}
	case PaneTree:
		return &assetListPane{}
	case PaneDetail:
		return &plainDetailPane{}
	default:
		return nil
	}
}

// --- header -----------------------------------------------------------------

// summaryPane is one line: how many assets, how many of them never scanned, and
// the tally of checks. It stands in for the real header (summary, search and
// filter chips) and is not focusable, because there is nothing on it to drive.
type summaryPane struct{}

func (p *summaryPane) ID() PaneID      { return PaneHeader }
func (p *summaryPane) Focusable() bool { return false }
func (p *summaryPane) Claims() []string {
	return nil
}
func (p *summaryPane) Hints(*State) []Hint { return nil }

func (p *summaryPane) Update(*State, tea.Msg) (tea.Cmd, bool) { return nil, false }

func (p *summaryPane) HeightFor(*State, int) int { return 1 }

func (p *summaryPane) Render(st *State, rect tui.Rect) Render {
	assets := st.Report.AssetCounts
	checks := st.Counts()

	left := " " + tui.StyleAccent.Render("✦ cnspec") + "  " +
		tui.StyleDim.Render(fmt.Sprintf("%d assets", assets.Total))
	// An asset that could not be scanned is the one number that must not hide
	// inside a total.
	if assets.Errored > 0 {
		left += tui.StyleDim.Render(" · ") +
			StatusStyle(reportmodel.StatusError).Render(fmt.Sprintf("%d not scanned", assets.Errored))
	}

	right := strings.Join([]string{
		StatusStyle(reportmodel.StatusFail).Render(fmt.Sprintf("%d failed", checks.Failed)),
		StatusStyle(reportmodel.StatusError).Render(fmt.Sprintf("%d errored", checks.Errored)),
		StatusStyle(reportmodel.StatusPass).Render(fmt.Sprintf("%d passed", checks.Passed)),
	}, tui.StyleDim.Render(" · ")) + " "

	if st.Filter.Active() {
		left += tui.StyleDim.Render(" · ") + tui.StyleFaint.Render(st.Filter.Describe())
	}

	return Render{Lines: []string{tui.Band(left, right, rect.W)}}
}

// --- tree -------------------------------------------------------------------

// assetListPane is a flat list of assets: enough to navigate a multi-asset scan
// and to prove the selection seam, but not the collapsible asset -> policy ->
// check tree that replaces it.
type assetListPane struct {
	cursor int
	scroll tui.Scroll
}

func (p *assetListPane) ID() PaneID       { return PaneTree }
func (p *assetListPane) Focusable() bool  { return true }
func (p *assetListPane) Claims() []string { return nil }
func (p *assetListPane) Hints(*State) []Hint {
	return []Hint{{Key: "↑/↓", Label: "asset"}, {Key: "enter", Label: "detail"}}
}

func (p *assetListPane) Render(st *State, rect tui.Rect) Render {
	assets := st.FilteredAssets()
	p.cursor = tui.ClampIndex(p.cursor, len(assets))
	// The cursor is followed here rather than where it moves, because this is
	// the only place that knows how many rows fit.
	p.scroll.EnsureVisible(p.cursor, len(assets), rect.H)
	off := p.scroll.Apply(len(assets), rect.H)

	res := Render{
		Title:  "Assets",
		Status: tui.Position(p.cursor, len(assets), rect.H),
	}
	if len(assets) == 0 {
		res.Lines = []string{tui.StyleFaint.Render("no assets match")}
		return res
	}

	for i := off; i < len(assets) && i < off+rect.H; i++ {
		a := assets[i]
		row := i - off
		res.Lines = append(res.Lines, p.row(st, a, i == p.cursor, rect.W))
		res.Zones = append(res.Zones, Zone{
			Rect: tui.Rect{X: rect.X, Y: rect.Y + row, W: rect.W, H: 1},
			Idx:  i,
			Tag:  "asset",
		})
	}
	return res
}

func (p *assetListPane) row(st *State, a *reportmodel.Asset, selected bool, w int) string {
	// A selected row is a full-width band, which is what makes the viewer read
	// as an application rather than as printed lines. The band's own colors have
	// to win, so its text goes in unstyled.
	if selected {
		style := tui.BandInactive
		if st.Focus == PaneTree {
			style = tui.BandSelected
		}
		return tui.Bar(" "+string(a.Status)+"  "+tui.Clean(a.Name), w, style)
	}

	label := StatusStyle(a.Status).Render(tui.PadRight(string(a.Status), statusLabelWidth))
	name := tui.StyleText.Render(tui.Clean(a.Name))
	if !a.Scanned() {
		// An asset that failed to scan is not an asset with no findings, and the
		// list is where that has to be visible.
		name = StatusStyle(reportmodel.StatusError).Render(tui.Clean(a.Name))
	}
	return tui.Truncate(" "+label+" "+name, w)
}

func (p *assetListPane) Update(st *State, msg tea.Msg) (tea.Cmd, bool) {
	assets := st.FilteredAssets()

	switch msg := msg.(type) {
	case ClickMsg:
		if msg.Zone.Tag != "asset" {
			return nil, false
		}
		p.cursor = msg.Zone.Idx
		p.sync(st, assets)
		return nil, true

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.scroll.Move(-1, len(assets), 1)
			return nil, true
		case tea.MouseButtonWheelDown:
			p.scroll.Move(1, len(assets), 1)
			return nil, true
		}
		return nil, false

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "ctrl+p":
			p.cursor--
		case "down", "j", "ctrl+n":
			p.cursor++
		case "home", "g":
			p.cursor = 0
		case "end", "G":
			p.cursor = len(assets) - 1
		case "enter", "right", "l":
			p.sync(st, assets)
			st.Focus = PaneDetail
			return nil, true
		default:
			return nil, false
		}
		p.cursor = tui.ClampIndex(p.cursor, len(assets))
		p.sync(st, assets)
		return nil, true
	}
	return nil, false
}

func (p *assetListPane) sync(st *State, assets []*reportmodel.Asset) {
	p.cursor = tui.ClampIndex(p.cursor, len(assets))
	if p.cursor < len(assets) {
		st.SelectAsset(assets[p.cursor])
	}
}

// --- detail -----------------------------------------------------------------

// plainDetailPane dumps what is selected as wrapped text. The real detail pane
// lays the same information out in sections; this one exists so the seam has
// something on the other end of it.
type plainDetailPane struct {
	scroll tui.Scroll
	// rev is the selection the scroll offset belongs to. A new selection starts
	// at the top rather than halfway down the previous one.
	rev int
}

func (p *plainDetailPane) ID() PaneID       { return PaneDetail }
func (p *plainDetailPane) Focusable() bool  { return true }
func (p *plainDetailPane) Claims() []string { return nil }
func (p *plainDetailPane) Hints(*State) []Hint {
	return []Hint{{Key: "↑/↓", Label: "scroll"}}
}

func (p *plainDetailPane) Render(st *State, rect tui.Rect) Render {
	if p.rev != st.SelectionRev {
		p.rev = st.SelectionRev
		p.scroll.Off = 0
	}

	lines := p.lines(st, rect.W)
	off := p.scroll.Apply(len(lines), rect.H)

	end := off + rect.H
	if end > len(lines) {
		end = len(lines)
	}
	return Render{
		Title:  p.title(st),
		Status: tui.Position(off, len(lines), rect.H),
		Lines:  lines[off:end],
	}
}

func (p *plainDetailPane) title(st *State) string {
	if st.Sel.Check != nil {
		return "Check"
	}
	if st.Sel.Asset != nil {
		return "Asset"
	}
	return "Detail"
}

func (p *plainDetailPane) lines(st *State, w int) []string {
	if st.Sel.Check != nil {
		return p.checkLines(st.Sel.Check, w)
	}
	if st.Sel.Asset != nil {
		return p.assetLines(st, st.Sel.Asset, w)
	}
	return []string{tui.StyleFaint.Render("nothing selected")}
}

func (p *plainDetailPane) assetLines(st *State, a *reportmodel.Asset, w int) []string {
	var res []string
	res = append(res, tui.Wrap(tui.StyleText.Render(tui.Clean(a.Name)), w)...)
	if a.PlatformName != "" {
		res = append(res, tui.Wrap(tui.StyleDim.Render(tui.Clean(a.PlatformName)), w)...)
	}
	res = append(res, "")
	res = append(res, tui.StyleLabel.Render("STATUS")+" "+StatusStyle(a.Status).Render(string(a.Status)))

	if !a.Scanned() {
		// The whole point of the model's ScanError: say why there is nothing
		// here, rather than showing an empty pane that looks like a clean bill
		// of health.
		res = append(res, "")
		res = append(res, tui.StyleLabel.Render("THIS ASSET WAS NOT SCANNED"))
		msg := a.ScanError
		if msg == "" {
			msg = "the scan produced no report and no error for this asset"
		}
		for _, ln := range tui.Wrap(tui.Clean(msg), w) {
			res = append(res, StatusStyle(reportmodel.StatusError).Render(ln))
		}
		return res
	}

	c := a.Counts
	res = append(res, "")
	res = append(res, tui.StyleLabel.Render("CHECKS")+" "+tui.StyleDim.Render(fmt.Sprintf(
		"%d total · %d passed · %d failed · %d errored · %d skipped · %d unscored",
		c.Total, c.Passed, c.Failed, c.Errored, c.Skipped, c.Unscored)))

	findings := 0
	res = append(res, "")
	res = append(res, tui.StyleLabel.Render("FINDINGS"))
	for _, check := range st.FilteredChecks(a.Checks) {
		if !check.Status.IsFinding() {
			continue
		}
		findings++
		res = append(res, tui.Truncate(" "+StatusLabel(check.Status)+" "+
			SeverityBadge(check.Severity)+" "+tui.StyleText.Render(tui.Clean(check.Title)), w))
	}
	if findings == 0 {
		res = append(res, tui.StyleFaint.Render(" none"))
	}
	return res
}

func (p *plainDetailPane) checkLines(c *reportmodel.Check, w int) []string {
	d := c.Detail()
	var res []string
	res = append(res, tui.Wrap(tui.StyleText.Render(tui.Clean(d.Title)), w)...)
	res = append(res, "")
	res = append(res, StatusLabel(d.Status)+" "+SeverityBadge(d.Severity))
	for _, section := range []struct{ label, body string }{
		{"DESCRIPTION", d.Description},
		{"MQL", d.Mql},
		{"ASSESSMENT", d.Assessment},
		{"ERROR", d.Error},
	} {
		if section.body == "" {
			continue
		}
		res = append(res, "", tui.StyleLabel.Render(section.label))
		res = append(res, tui.Wrap(section.body, w)...)
	}
	return res
}

func (p *plainDetailPane) Update(_ *State, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.scroll.Off--
			return nil, true
		case tea.MouseButtonWheelDown:
			p.scroll.Off++
			return nil, true
		}
		return nil, false
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.scroll.Off--
		case "down", "j":
			p.scroll.Off++
		case "pgup":
			p.scroll.Off -= 10
		case "pgdown":
			p.scroll.Off += 10
		case "home", "g":
			p.scroll.Off = 0
		default:
			return nil, false
		}
		if p.scroll.Off < 0 {
			p.scroll.Off = 0
		}
		return nil, true
	}
	return nil, false
}
