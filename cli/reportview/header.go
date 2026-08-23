// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
)

// The header band: what the scan found, and the controls that narrow it.
//
// # What it says
//
// The top row is the scan's shape at a glance: how many assets there were, how
// many of them never scanned, and the tally of check outcomes. The five outcomes
// stay five. A check that errored is not a check that failed and neither is a
// check nobody scored, so they are five separate numbers and never a single
// "problems" figure.
//
// The number that matters most is the one that is easiest to lose. An asset that
// failed to scan is not an asset with no findings: report-k8s.json is fifteen
// assets, zero reports and no bundle at all, and a header that answers "0 failed"
// for it has told the user the opposite of the truth. So the asset row leads with
// "15 not scanned", and when the report produced no checks at all the tally says
// "no checks ran" rather than printing five zeroes that read like a clean bill of
// health.
//
// # What it does
//
// Filtering is not implemented here. filter.go owns what a filter means -- so
// that the pane that edits one and the panes that obey it cannot disagree about
// it -- and this pane only builds a Filter and publishes it through
// State.SetFilter. Every count it prints comes back out of State, which is the
// same path the tree reads, so "showing 3 of 15" and the rows below it are the
// same computation.
//
// The one place the two vocabularies have to meet is the severity axis, and it
// is the trap this feature exists to avoid: severity is a property a *check*
// has. An asset that never ran a check has none, so a severity filter must not
// take it off the screen. Filter.MatchAsset already gets this right; the header's
// job is to say so out loud, which is the NOTE row.
//
// # How tall it is
//
// It grows and shrinks. Idle it is a single line; opening the search adds one,
// opening the chips adds two, an active filter adds the line that says what is
// left. HeightFor answers with exactly the number of lines Render will emit at
// that width -- both come out of compose -- so the frame's arithmetic and the
// band's contents cannot drift apart. The frame still clamps at MaxHeaderLines,
// and asking honestly is what keeps the clamp from ever being needed.

func init() {
	RegisterHeader(func(st *State) Pane {
		p := &headerPane{returnTo: PaneTree}
		if st != nil {
			p.search = st.Filter.Search
		}
		return p
	})
}

// headerLabelW is the width of the label column: STATUS, SEVERITY, FILTER,
// SEARCH, NOTE and the "✦ cnspec" wordmark all fit in it, so every row's content
// begins in the same column.
const headerLabelW = 8

// headerContentX is the column a row's content starts at: one marker cell, the
// label column, one space.
const headerContentX = headerLabelW + 2

// filterRow names a row of chips. The cursor is on exactly one of them.
type filterRow int

const (
	rowStatus filterRow = iota
	rowSeverity
)

// headerPane is the summary band, the search field and the filter chips.
//
// Everything on it is this pane's own business and lives here rather than on
// State: which row the chip cursor is on, whether the chips are showing, what
// has been typed into the search so far. The only thing that crosses the seam is
// the Filter, and it crosses through State.SetFilter.
type headerPane struct {
	// search is the text in the search field. It is published to the filter on
	// every keystroke, so the tree narrows as you type rather than on enter.
	search string
	// searching is whether the search field is open and taking keys.
	searching bool
	// open is whether the status and severity chip rows are showing.
	open bool
	// active mirrors Filter.Active(). Focusable takes no State, so the one bit
	// of the filter the pane needs before it has been handed one is kept here
	// and refreshed on every pass through setFilter and sync.
	active bool

	// row and cursor are the chip under the cursor.
	row    filterRow
	cursor int

	// returnTo is the pane focus goes back to when the header closes. It is
	// remembered rather than assumed, because the header can be reached from
	// either body pane.
	returnTo PaneID

	// severities caches the per-severity tally of every check in the report,
	// which is a walk over every check and does not change while the report
	// does not. severitiesFor is the report it was computed for.
	severities    map[string]int
	severitiesFor *reportmodel.Report
}

func (p *headerPane) ID() PaneID { return PaneHeader }

// Focusable is state-dependent on purpose. An idle header has nothing to drive,
// so it stays out of the tab cycle and "esc" keeps its ordinary meaning of quit.
// A header that is showing a search, a set of chips or an active filter does
// have something to drive, takes focus, and gets "esc" routed to it -- which is
// what makes the ladder work: esc closes the chips, esc clears the filter, esc
// quits.
func (p *headerPane) Focusable() bool { return p.searching || p.open || p.active }

// Claims are the two keys the header owns from anywhere. "/" opens the search
// while the tree still has focus, which is the only way a search field is worth
// having, and "f" reaches the chips the same way.
func (p *headerPane) Claims() []string { return []string{"/", "f"} }

func (p *headerPane) Hints(*State) []Hint {
	if p.searching {
		return []Hint{
			{Key: "type", Label: "narrow"},
			{Key: "enter", Label: "done"},
			{Key: "esc", Label: "clear search"},
		}
	}
	return []Hint{
		{Key: "←/→", Label: "chip"},
		{Key: "↑/↓", Label: "row"},
		{Key: "space", Label: "toggle"},
		{Key: "/", Label: "search"},
		{Key: "c", Label: "clear"},
		{Key: "esc", Label: "close"},
	}
}

// HeightFor is the honest answer: exactly the lines Render will emit at this
// width. Both call compose, so a row that appears cannot fail to be counted.
func (p *headerPane) HeightFor(st *State, width int) int {
	if st == nil {
		return 1
	}
	return len(p.compose(st, width))
}

func (p *headerPane) Render(st *State, rect tui.Rect) Render {
	rows := p.compose(st, rect.W)

	res := Render{Lines: make([]string, 0, len(rows))}
	for i, row := range rows {
		res.Lines = append(res.Lines, row.text)
		for _, z := range row.zones {
			// compose works in band-relative columns; the frame hit-tests in
			// absolute cells.
			z.Rect.X += rect.X
			z.Rect.Y = rect.Y + i
			res.Zones = append(res.Zones, z)
		}
	}
	return res
}

// headerRow is one rendered line and the chips on it that answer a click. The
// zone rects are relative to the band's left edge and Render moves them.
type headerRow struct {
	text  string
	zones []Zone
}

// compose builds the band. It is the single description of what the header shows
// at a given width, which is what lets HeightFor promise a number Render keeps.
func (p *headerPane) compose(st *State, w int) []headerRow {
	p.sync(st)
	if w < 1 {
		w = 1
	}

	rows := []headerRow{{text: p.summaryLine(st, w)}}
	if note, ok := p.noteLine(st, w); ok {
		rows = append(rows, headerRow{text: note})
	}
	if st.Filter.Active() {
		rows = append(rows, headerRow{text: p.filterLine(st, w)})
	}
	if p.searching {
		rows = append(rows, headerRow{text: p.searchLine(st, w)})
	}
	if p.open {
		rows = append(rows, p.chipRow(st, rowStatus, w), p.chipRow(st, rowSeverity, w))
	}
	return rows
}

// sync reconciles the pane with a focus the frame may have moved underneath it.
// A search field the user tabbed away from is closed rather than left silently
// eating keys, and the pane remembers where focus came from so it can hand it
// back when it closes.
func (p *headerPane) sync(st *State) {
	p.active = st.Filter.Active()
	if st.Focus != PaneHeader {
		p.searching = false
		if st.Focus != PaneNone {
			p.returnTo = st.Focus
		}
	}
	if p.returnTo == PaneNone || p.returnTo == PaneHeader {
		p.returnTo = PaneTree
	}
	p.clampCursor(st)
}

// --- the rows ---------------------------------------------------------------

// summaryLine is the scan's shape: assets on the left, check outcomes on the
// right.
func (p *headerPane) summaryLine(st *State, w int) string {
	assets := st.Report.AssetCounts

	left := " " + tui.StyleAccent.Render(tui.PadRight("✦ cnspec", headerLabelW)) + " " +
		tui.StyleDim.Render(plural(assets.Total, "asset"))
	// The one number that must never hide inside a total. An asset that failed
	// to scan produced no checks, so every tally to the right of here is silent
	// about it, and this is the only place it gets said.
	if assets.Errored > 0 {
		left += tui.StyleDim.Render(" · ") +
			StatusStyle(reportmodel.StatusError).Render(fmt.Sprintf("%d not scanned", assets.Errored))
	}

	right := p.tally(st, w-tui.Width(left)-1)
	if right != "" {
		right += " "
	}

	// The two keys that reach everything else on this band. They are shown only
	// while it is idle -- which is the only state in which the user has not
	// already found them -- and only on a terminal wide enough that they cost
	// nothing.
	mid := ""
	if !p.open && !p.searching && !st.Filter.Active() {
		mid = tui.Kbd("/", "search") + "   " + tui.Kbd("f", "filter")
	}
	return tui.Band3(left, mid, right, w)
}

// tally is the check outcomes, five of them, never summed.
//
// When the report produced no checks at all there is nothing to tally, and
// saying so is the honest answer -- five zeroes here would read as "everything
// passed" for a scan where nothing ran.
func (p *headerPane) tally(st *State, budget int) string {
	if budget < 1 {
		return ""
	}
	if st.Report.CheckCounts.Total == 0 {
		out := tui.StyleFaint.Render("no checks ran")
		if tui.Width(out) > budget {
			return ""
		}
		return out
	}

	counts := st.Counts()
	shown := p.tallyStatuses(st)
	for {
		out := renderTally(counts, shown)
		if tui.Width(out) <= budget || len(shown) <= 1 {
			return out
		}
		// Too wide. Drop an outcome that has nothing in it rather than merge two
		// that do: an outcome that is absent from the band is zero, and one that
		// is present is exact. When every remaining outcome is non-zero there is
		// nothing left to drop and the band gets truncated instead.
		next := dropLastEmpty(counts, shown)
		if next == nil {
			return out
		}
		shown = next
	}
}

func renderTally(counts reportmodel.Counts, shown []reportmodel.Status) string {
	parts := make([]string, 0, len(shown))
	for _, s := range shown {
		parts = append(parts, StatusStyle(s).Render(fmt.Sprintf("%d %s", countOf(counts, s), s)))
	}
	return strings.Join(parts, tui.StyleDim.Render("  "))
}

func dropLastEmpty(counts reportmodel.Counts, shown []reportmodel.Status) []reportmodel.Status {
	for i := len(shown) - 1; i >= 0; i-- {
		if countOf(counts, shown[i]) == 0 {
			res := make([]reportmodel.Status, 0, len(shown)-1)
			res = append(res, shown[:i]...)
			return append(res, shown[i+1:]...)
		}
	}
	return nil
}

// noteLine is the sentence the severity filter needs to be honest.
//
// Severity describes a check. An asset that never produced one has no severity
// to be judged by, so Filter.MatchAsset keeps it whatever severities are
// selected -- which is correct, and looks like a bug unless the header says why
// those rows are still there.
func (p *headerPane) noteLine(st *State, w int) (string, bool) {
	if len(st.Filter.Severities) == 0 {
		return "", false
	}
	kept := 0
	for _, a := range st.FilteredAssets() {
		if !a.Scanned() {
			kept++
		}
	}
	if kept == 0 {
		return "", false
	}
	body := fmt.Sprintf("%s kept: severity describes a check, and these ran none",
		plural(kept, "unscanned asset"))
	return tui.Truncate(" "+tui.StyleLabel.Render(tui.PadRight("NOTE", headerLabelW))+" "+tui.StyleFaint.Render(body), w), true
}

// filterLine says what is being filtered on, and what survives it.
func (p *headerPane) filterLine(st *State, w int) string {
	left := " " + tui.StyleLabel.Render(tui.PadRight("FILTER", headerLabelW)) + " " +
		tui.StyleText.Render(tui.Clean(st.Filter.Describe()))

	parts := []string{fmt.Sprintf("%d of %d assets", len(st.FilteredAssets()), len(st.Report.Assets))}
	// Only claim a check count when the scan produced checks to count. "0 of 0
	// checks" is noise on a report that never ran one.
	if total := st.Report.CheckCounts.Total; total > 0 {
		parts = append(parts, fmt.Sprintf("%d of %d checks", st.Counts().Total, total))
	}
	right := tui.StyleDim.Render("showing "+strings.Join(parts, " · ")) + " "

	return tui.Band(left, right, w)
}

// searchLine is the search field. The term is already published to the filter by
// the time it is drawn, so the counts on the FILTER row above move as you type.
func (p *headerPane) searchLine(st *State, w int) string {
	body := tui.StyleText.Render(tui.Clean(p.search)) + tui.StyleAccent.Render("▌")
	if p.search == "" {
		body += " " + tui.StyleFaint.Render("title or asset name")
	}
	left := " " + tui.StyleLabel.Render(tui.PadRight("SEARCH", headerLabelW)) + " " + body
	right := tui.StyleFaint.Render("enter done · esc clear") + " "
	return tui.Band(left, right, w)
}

// chipRow is one axis of the filter: the outcomes, or the severities. The counts
// on the chips are the whole report's, not the filtered view's, because "how
// many checks errored" is a fact about the scan and should not move when you
// toggle something else on.
func (p *headerPane) chipRow(st *State, row filterRow, w int) headerRow {
	label, tag := "STATUS", "status"
	if row == rowSeverity {
		label, tag = "SEVERITY", "severity"
	}

	marker := " "
	if st.Focus == PaneHeader && p.row == row {
		marker = tui.StyleAccent.Render("›")
	}
	line := marker + tui.StyleLabel.Render(tui.PadRight(label, headerLabelW)) + " "

	var zones []Zone
	col := headerContentX
	for i, c := range p.chips(st, row) {
		onCursor := st.Focus == PaneHeader && p.row == row && p.cursor == i
		text := chip(c.style, c.label, c.selected, onCursor)
		cw := tui.Width(text)
		if col+cw <= w {
			zones = append(zones, Zone{
				Rect: tui.Rect{X: col, W: cw, H: 1},
				Idx:  i,
				Tag:  tag,
			})
		}
		line += text
		col += cw
	}
	return headerRow{text: tui.Truncate(line, w), zones: zones}
}

// chipSpec is one toggleable value on a chip row.
type chipSpec struct {
	label    string
	style    lipgloss.Style
	selected bool
}

func (p *headerPane) chips(st *State, row filterRow) []chipSpec {
	if row == rowSeverity {
		counts := p.severityCounts(st)
		res := make([]chipSpec, 0, len(AllSeverities))
		for _, sev := range AllSeverities {
			res = append(res, chipSpec{
				label:    fmt.Sprintf("%s %d", sev, counts[sev]),
				style:    SeverityStyle(sev),
				selected: st.Filter.Severities[sev],
			})
		}
		return res
	}

	statuses := p.tallyStatuses(st)
	res := make([]chipSpec, 0, len(statuses))
	for _, s := range statuses {
		res = append(res, chipSpec{
			label:    fmt.Sprintf("%s %d", s, countOf(st.Report.CheckCounts, s)),
			style:    StatusStyle(s),
			selected: st.Filter.Statuses[s],
		})
	}
	return res
}

// chip renders one toggleable value. Selected and unselected are the same width,
// so a row does not shift under the cursor when something is toggled, and the
// colour is the frame's -- a chip must never pick its own.
func chip(style lipgloss.Style, label string, selected, cursor bool) string {
	if cursor {
		style = style.Underline(true).Bold(true)
	}
	if selected {
		// Reverse, rather than a hand-picked background: whatever colour the
		// rating palette gave this value becomes the band behind the label.
		return style.Reverse(true).Render(" " + label + " ")
	}
	return " " + style.Render(label) + " "
}

// --- keys -------------------------------------------------------------------

func (p *headerPane) Update(st *State, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case ClickMsg:
		switch msg.Zone.Tag {
		case "status":
			p.row, p.cursor = rowStatus, msg.Zone.Idx
		case "severity":
			p.row, p.cursor = rowSeverity, msg.Zone.Idx
		default:
			return nil, false
		}
		p.toggle(st)
		return nil, true

	case tea.KeyMsg:
		if p.searching {
			return p.typing(st, msg)
		}
		return p.command(st, msg)
	}
	return nil, false
}

// typing is the search field. Only the keys that edit text are consumed, so tab
// still cycles panes and the arrows still belong to whatever else wants them.
func (p *headerPane) typing(st *State, msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace:
		if msg.Alt {
			return nil, false
		}
		p.search += string(msg.Runes)
	case tea.KeyBackspace:
		if r := []rune(p.search); len(r) > 0 {
			p.search = string(r[:len(r)-1])
		}
	case tea.KeyCtrlU:
		p.search = ""
	case tea.KeyEnter:
		p.searching = false
		p.restoreFocus(st)
		return nil, true
	case tea.KeyEsc:
		// esc backs out of the narrowing rather than quitting. The frame only
		// routes it here because this pane has focus, and it only has focus
		// because the search is open.
		p.searching = false
		p.search = ""
		p.publishSearch(st)
		p.restoreFocus(st)
		return nil, true
	default:
		return nil, false
	}
	p.publishSearch(st)
	return nil, true
}

func (p *headerPane) command(st *State, msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "/":
		p.searching = true
		st.Focus = PaneHeader
		return nil, true

	case "f":
		switch {
		case !p.open:
			p.open = true
			st.Focus = PaneHeader
		case st.Focus != PaneHeader:
			// Reached from a body pane while the chips were already showing:
			// go to them rather than close them.
			st.Focus = PaneHeader
		default:
			p.open = false
			p.restoreFocus(st)
		}
		return nil, true

	case "left", "h":
		p.cursor--
	case "right", "l":
		p.cursor++
	case "up", "k":
		p.row = rowStatus
	case "down", "j":
		p.row = rowSeverity
	case "home":
		p.cursor = 0
	case "end":
		p.cursor = len(p.chips(st, p.row)) - 1

	case " ", "enter":
		p.toggle(st)
		return nil, true

	case "c":
		p.search = ""
		p.setFilter(st, Filter{})
		return nil, true

	case "esc":
		// The ladder: close what is open, then clear what is set, and only then
		// let the frame have it and quit.
		switch {
		case p.open:
			p.open = false
		case p.active:
			p.search = ""
			p.setFilter(st, Filter{})
		default:
			return nil, false
		}
		p.restoreFocus(st)
		return nil, true

	default:
		return nil, false
	}

	p.clampCursor(st)
	return nil, true
}

func (p *headerPane) toggle(st *State) {
	chips := p.chips(st, p.row)
	if p.cursor < 0 || p.cursor >= len(chips) {
		return
	}
	if p.row == rowSeverity {
		p.setFilter(st, st.Filter.ToggleSeverity(AllSeverities[p.cursor]))
		return
	}
	p.setFilter(st, st.Filter.ToggleStatus(p.tallyStatuses(st)[p.cursor]))
}

// setFilter is the only way this pane publishes a filter. It exists so that
// Focusable's view of "is anything filtered" cannot fall behind the filter
// itself between a keypress and the next frame.
func (p *headerPane) setFilter(st *State, f Filter) {
	st.SetFilter(f)
	p.active = st.Filter.Active()
}

// publishSearch is the only way the search term reaches the rest of the viewer.
// The rest of the filter is preserved: typing must not silently drop the chips.
func (p *headerPane) publishSearch(st *State) {
	f := st.Filter.Clone()
	f.Search = p.search
	p.setFilter(st, f)
}

func (p *headerPane) restoreFocus(st *State) {
	if p.Focusable() {
		return
	}
	st.Focus = p.returnTo
}

func (p *headerPane) clampCursor(st *State) {
	p.cursor = tui.ClampIndex(p.cursor, len(p.chips(st, p.row)))
}

// --- counting ---------------------------------------------------------------

// tallyStatuses is the outcomes worth a column. The five that always mean
// something always get one; UNKNOWN is a bucket for a score cnspec did not
// recognise and only earns its space when something landed in it.
func (p *headerPane) tallyStatuses(st *State) []reportmodel.Status {
	if st.Report.CheckCounts.Unknown > 0 {
		return AllStatuses
	}
	return AllStatuses[:len(AllStatuses)-1]
}

// severityCounts tallies every check in the report by severity. The result is
// cached against the report it was computed from: the walk is over every check
// of every asset and the answer cannot change while the report does not.
func (p *headerPane) severityCounts(st *State) map[string]int {
	if p.severitiesFor == st.Report && p.severities != nil {
		return p.severities
	}
	res := map[string]int{}
	for _, a := range st.Report.Assets {
		for _, c := range a.Checks {
			res[c.Severity]++
		}
	}
	p.severities, p.severitiesFor = res, st.Report
	return res
}

func countOf(c reportmodel.Counts, s reportmodel.Status) int {
	switch s {
	case reportmodel.StatusPass:
		return c.Passed
	case reportmodel.StatusFail:
		return c.Failed
	case reportmodel.StatusError:
		return c.Errored
	case reportmodel.StatusSkipped:
		return c.Skipped
	case reportmodel.StatusUnscored:
		return c.Unscored
	default:
		return c.Unknown
	}
}

// --- text -------------------------------------------------------------------

func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
