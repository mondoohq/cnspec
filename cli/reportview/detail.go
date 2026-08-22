// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
)

// The detail pane: everything the report knows about whatever is selected.
//
// It draws two different things, because the tree selects two different things.
// A check gets the section stack below. An asset -- selected when the cursor is
// on an asset row, or when the asset has no checks to descend into -- gets its
// platform, its tally and its findings, or, when it never scanned, the reason
// why. Those are not the same page with fields missing: an asset that failed to
// scan has no checks at all, and rendering it as an empty check list is the one
// way this pane can lie.
//
// # Section order
//
// The stack follows cli/reporter/junit.go's detailedCheckBody, which is the
// existing answer in this repo to "what does a reader want to know about a
// finding, and in what order":
//
//	description -> query -> assessment -> failing locations -> remediation -> references
//
// with three additions it has no place for, each put where it answers the
// question the section above it raises:
//
//   - AUDIT sits under QUERY. Both answer "how is this decided" -- one as the
//     query that ran, one as the steps to check it by hand.
//   - COMPLIANCE and POLICIES sit at the bottom. They are provenance: which
//     framework control this satisfies and which policies pulled it in. Neither
//     helps you fix anything, so neither belongs above the fix.
//
// The one deliberate departure is ERROR, which junit puts after the assessment
// and this pane puts immediately under the status row. Detail.Error is populated
// only for a ScoreType_Error score, so its presence *is* the errored case, and
// for that case it is the headline: the check proved nothing, and the assessment
// below it reads "[failed] ..." only because the query never completed. Burying
// that eight sections down invites the reader to mistake an error for a failure.
//
// # Markdown
//
// Three of the sections are markdown *source*: the description, the audit steps
// and every remediation body. They go through markdown.go, which renders them
// rather than printing their markup. Nothing else does: MQL and the assessment
// are code with a line structure of their own, and the error, the failing
// locations, the references and the compliance tags are plain strings that never
// carried markup to begin with.
//
// A reference URL is the one plain string that is still more than text: it is
// drawn as an OSC 8 hyperlink and carries a click zone, so it can be opened
// rather than retyped. See links.go, which also says why the URLs inside the
// markdown sections are left to glamour.

func init() {
	RegisterDetail(func(*State) Pane { return &detailPane{} })
}

// detailPane renders the selection and scrolls it. Everything it remembers
// between frames -- the offset, the geometry the offset was clamped against, the
// cached lines -- is its own; the frame and the other panes see none of it.
type detailPane struct {
	scroll tui.Scroll

	// rev and width are what the cache was built for. A new selection or a
	// resize invalidates it, and a new selection also scrolls back to the top:
	// opening a check halfway down the previous one is disorienting.
	cached bool
	rev    int
	width  int
	lines  []string

	// snips is every copyable block of the selected check and buttons says which
	// row each one's COPY sits on, both built in the same pass as lines so a
	// button can never name a block that moved. room is false in a pane too
	// narrow to give a button a strip of its own, in which case none is drawn.
	snips   []Snippet
	buttons []detailButton
	room    bool

	// urls is every clickable URL of the selected check and links says which
	// cells each one occupies, built in the same pass as lines for the same
	// reason the buttons are: a zone computed in a second pass is a zone that
	// can drift off the row it names. See links.go.
	urls  []string
	links []detailLink

	// armAt is the block n and p chose, and armSet says one was chosen. The zero
	// value is no explicit choice, which is what a fresh pane and a newly
	// selected check both want: see the arming rule in copy.go.
	armAt  int
	armSet bool

	// visible is the height of the last render, which is what Update needs to
	// page and to jump to the end. Update has no rect of its own.
	visible int
}

// detailButton is one COPY affordance: the row it is drawn on, and the snippet
// it copies.
type detailButton struct {
	Row int
	Idx int
}

// detailLink is one clickable URL: the cells it occupies -- the row, the column
// it starts at and how wide the visible text came out after truncation -- and
// the URL it points at.
//
// W is the *rendered* width and not the length of the URL, because a link too
// wide for the pane is drawn short: the zone has to cover the cells the reader
// can see and no others.
type detailLink struct {
	Row int
	Col int
	W   int
	Idx int
}

func (p *detailPane) ID() PaneID       { return PaneDetail }
func (p *detailPane) Focusable() bool  { return true }
func (p *detailPane) Claims() []string { return nil }

// Hints. n and p are the frame's keys, not this pane's -- they aim the copy the
// same way y fires it, and both work from the tree as well. The short form is
// advertised here because this is the pane that draws the band they move, and
// because the frame's own compact list has no room left for it; the ? list
// carries them as the frame bindings they are. See Model.frameHints.
func (p *detailPane) Hints(*State) []Hint {
	return []Hint{
		{Key: "↑/↓", Label: "scroll"},
		{Key: "pgup/pgdn", Label: "page"},
		{Key: "g/G", Label: "top/end"},
		{Key: "n/p", Label: "block"},
	}
}

func (p *detailPane) Render(st *State, rect tui.Rect) Render {
	lines := p.body(st, rect.W)
	p.visible = rect.H

	off := p.scroll.Apply(len(lines), rect.H)
	end := off + rect.H
	if end > len(lines) {
		end = len(lines)
	}
	// Copied rather than sliced: the buttons are drawn onto the visible rows,
	// and writing them through into the cache would bake this frame's armed
	// block into every later one.
	rows := append([]string(nil), lines[off:end]...)

	// The links go in first and the buttons after, so that a button wins a
	// hit-test if the two ever overlapped (the frame lets a later zone win). No
	// reference row carries a code block, so today they cannot.
	zones := p.linkZones(rect, off, end)

	return Render{
		Title:  detailTitle(st),
		Status: tui.Position(off, len(lines), rect.H),
		Lines:  rows,
		Zones:  append(zones, p.drawButtons(rows, rect, off, end)...),
	}
}

// linkZones is a clickable region for every URL on screen.
//
// Nothing is drawn: the link was rendered into its row as an OSC 8 sequence when
// the body was built, and this only says which cells it landed on. That is the
// difference from drawButtons, which has to paint its affordance onto the rows
// of this frame.
func (p *detailPane) linkZones(rect tui.Rect, off, end int) []Zone {
	if len(p.links) == 0 {
		return nil
	}
	var zones []Zone
	for _, l := range p.links {
		if l.Row < off || l.Row >= end || l.W < 1 {
			continue
		}
		w := min(l.W, rect.W-l.Col)
		if w < 1 {
			continue
		}
		zones = append(zones, Zone{
			Rect: tui.Rect{X: rect.X + l.Col, Y: rect.Y + l.Row - off, W: w, H: 1},
			Idx:  l.Idx,
			Tag:  linkZoneTag,
		})
	}
	return zones
}

// drawButtons puts a COPY on every code block whose first row is on screen, and
// returns a clickable zone for each. The armed one -- the block the y key would
// take -- wears the accent band; the rest wear the inactive one, because a click
// copies those just as well.
func (p *detailPane) drawButtons(rows []string, rect tui.Rect, off, end int) []Zone {
	if !p.room || len(p.buttons) == 0 {
		return nil
	}
	armed := p.armedIn(off, end)
	var zones []Zone
	for i, btn := range p.buttons {
		if btn.Row < off || btn.Row >= end {
			continue
		}
		row := btn.Row - off
		rows[row] = copyButtonRow(rows[row], rect.W, i == armed)
		zones = append(zones, Zone{
			Rect: tui.Rect{X: rect.X + rect.W - copyLabelW, Y: rect.Y + row, W: copyLabelW, H: 1},
			Idx:  btn.Idx,
			Tag:  copyZoneTag,
		})
	}
	return zones
}

// armedIn is the index into buttons of the block a copy key would take while
// rows [off, end) are the ones on screen, or -1 when no block is on screen at
// all. It is the whole of the arming rule stated in copy.go, in one place:
//
//   - the block n and p chose, if that block is on screen;
//   - otherwise the topmost one that is;
//   - otherwise nothing.
//
// The viewport bound is what makes the rule structural rather than a promise
// kept by every caller: an explicit arm that a scroll, a resize or a reflow has
// pushed off screen is not returned by this function, so y cannot take it.
func (p *detailPane) armedIn(off, end int) int {
	inView := func(i int) bool {
		if i < 0 || i >= len(p.buttons) {
			return false
		}
		row := p.buttons[i].Row
		return row >= off && row < end
	}
	if p.armSet && inView(p.armAt) {
		return p.armAt
	}
	for i := range p.buttons {
		if inView(i) {
			return i
		}
	}
	return -1
}

// viewport is the rows on screen as [off, end), from the last render. Update has
// no rect of its own, and the arming rule is about what is on screen right now.
func (p *detailPane) viewport() (int, int) {
	visible := p.visible
	if visible < 1 {
		visible = 1
	}
	return p.scroll.Off, p.scroll.Off + visible
}

// armed is armedIn over the current viewport.
func (p *detailPane) armed() int {
	return p.armedIn(p.viewport())
}

// CopyTarget implements CopySource: the block the y key takes.
//
// It answers from the check rather than refusing when the pane has not been
// drawn for the current selection, which happens below tui.MinTwoPaneWidth --
// there only the focused pane is rendered, and y must still work while the tree
// has focus. With nothing rendered there is no viewport to aim with, and no pane
// on screen to have aimed in, so it is the first block of the check.
func (p *detailPane) CopyTarget(st *State) (Snippet, bool) {
	if !p.cached || p.rev != st.SelectionRev {
		snips := checkSnippets(st.Sel.Check)
		if len(snips) == 0 {
			return Snippet{}, false
		}
		return snips[0], true
	}
	i := p.armed()
	if i < 0 || i >= len(p.buttons) {
		return Snippet{}, false
	}
	idx := p.buttons[i].Idx
	if idx < 0 || idx >= len(p.snips) {
		return Snippet{}, false
	}
	return p.snips[idx], true
}

// ArmCopy implements CopySource: move the armed block delta places and scroll it
// into view.
//
// An arm that is not on screen is an arm the user cannot see, so this is the one
// place that sets one and it always follows with the scroll that reveals it.
func (p *detailPane) ArmCopy(st *State, delta int) bool {
	// Not drawn for this selection -- the same one-pane case CopyTarget covers.
	// There is no viewport to move an arm through and no pane on screen to move
	// it in, so all this can honestly answer is whether there is anything to aim
	// at; the pane arms its first block when it is next drawn.
	if !p.cached || p.rev != st.SelectionRev {
		return len(checkSnippets(st.Sel.Check)) > 0
	}
	if len(p.buttons) == 0 {
		return false
	}
	off, end := p.viewport()
	next := p.nextArm(p.armedIn(off, end), delta, off, end)
	p.armAt, p.armSet = next, true
	p.reveal(p.buttons[next].Row, end-off)
	return true
}

// nextArm is the block delta places from the armed one, clamped at both ends: a
// list cursor that wraps from the last block to the first also jumps the pane
// from the bottom of the check to the top, which reads as a scroll accident.
//
// With nothing on screen to count from, n takes the first block below the
// viewport and p the last one above it, so the direction still means what it
// says. With no block in that direction it takes the nearest one in the other,
// which is the same clamp by another name.
func (p *detailPane) nextArm(at, delta, off, end int) int {
	last := len(p.buttons) - 1
	if at >= 0 {
		return min(max(at+delta, 0), last)
	}
	if delta > 0 {
		for i, btn := range p.buttons {
			if btn.Row >= end {
				return i
			}
		}
		return last
	}
	for i := last; i >= 0; i-- {
		if p.buttons[i].Row < off {
			return i
		}
	}
	return 0
}

// detailArmPeek is how many rows of a block n and p try to bring on screen under
// its COPY. A band on the bottom row, with the command it copies below the fold,
// is a selection you cannot read before taking it.
const detailArmPeek = 3

// reveal scrolls the least that brings a block's first row on screen, and a few
// rows of the block under it when the pane is tall enough to spare them. The
// second call is what guarantees the row itself: the first may have pushed it
// off the top of a viewport shorter than the peek.
func (p *detailPane) reveal(row, visible int) {
	total := len(p.lines)
	p.scroll.EnsureVisible(min(row+detailArmPeek, total-1), total, visible)
	p.scroll.EnsureVisible(row, total, visible)
}

// dropArm gives up an explicit arm that a plain scroll has moved out of view.
//
// This is the rule in copy.go doing its half of the work: armedIn would ignore
// the arm anyway, but dropping it here means a scroll that leaves and returns
// does not bring the old choice back with it. Once you have scrolled away from
// the block you picked, the topmost one on screen is the one y takes.
func (p *detailPane) dropArm() {
	if p.armSet && p.armed() != p.armAt {
		p.armSet = false
	}
}

// body is the rendered detail, cached per selection and width. Detail() walks
// the raw results to build an assessment and Render runs on every frame and
// every mouse event, so recomputing it forty times a second is work nobody asked
// for.
func (p *detailPane) body(st *State, w int) []string {
	if p.cached && p.rev == st.SelectionRev && p.width == w {
		return p.lines
	}
	// Watch SelectionRev rather than the pointers in st.Sel: the tree bumps it
	// on every change, and a check reselected after an edit to the filter is a
	// new page even when it is the same *Check.
	if !p.cached || p.rev != st.SelectionRev {
		p.scroll.Off = 0
		// A new check is a new page, and the block you picked on the last one is
		// not on it. Back to the implicit arm, which at the top of a fresh page
		// is the check's first block.
		p.armSet = false
	}
	p.cached, p.rev, p.width = true, st.SelectionRev, w

	b := detailContent(st, w)
	p.lines, p.snips, p.buttons, p.room = b.out, b.snips, b.buttons, b.room
	p.urls, p.links = b.urls, b.links
	return p.lines
}

func detailTitle(st *State) string {
	switch {
	case st.Sel.Check != nil:
		return "Check"
	case st.Sel.Asset != nil:
		return "Asset"
	default:
		return "Detail"
	}
}

// detailBody is the rendered rows of the selection, and is what every test that
// only cares about the text calls.
func detailBody(st *State, w int) []string {
	return detailContent(st, w).out
}

// detailContent is one full pass over the selection: the rows, the copyable
// blocks and where their buttons go, all out of the same walk.
func detailContent(st *State, w int) *detailBuf {
	b := newDetailBuf(w)
	switch {
	case st.Sel.Check != nil:
		checkLines(b, st.Sel.Check)
	case st.Sel.Asset != nil:
		assetLines(b, st, st.Sel.Asset)
	default:
		b.push(tui.StyleFaint.Render("nothing selected"))
	}
	return b
}

// --- the check page ---------------------------------------------------------

func checkLines(b *detailBuf, c *reportmodel.Check) {
	d := c.Detail()
	// Every copyable block of this check, in the order the sections below draw
	// them. The buffer claims them one at a time as it reaches each block, so
	// the button on a row and the snippet the footer names are the same object
	// rather than two walks that agree by luck.
	b.snips = detailSnippets(d)

	b.para(tui.StyleText, d.Title)
	// A check with no title at all still gets its status row, so the pane never
	// renders as blank.
	b.blank()
	b.push(statusRow(d))

	// See the package comment: for an errored check this is the headline, not a
	// footnote under a result that does not exist.
	if d.Error != "" {
		b.section("ERROR")
		b.wrapped(StatusStyle(reportmodel.StatusError), d.Error)
	}

	if d.Description != "" {
		b.section("DESCRIPTION")
		b.markdown(d.Description)
	}

	if d.Mql != "" {
		b.section("QUERY")
		// MQL is code: one source line stays one row, cut at the right edge
		// rather than folded, so an operator never moves to a line of its own.
		// It gets a COPY of its own -- pasting the query into `cnspec shell` is
		// how you find out what the check actually saw.
		b.codeBlock(tui.StyleText, d.Mql)
	}

	if audit := strings.TrimSpace(d.Audit); audit != "" {
		b.section("AUDIT")
		b.markdown(audit)
	}

	if d.Assessment != "" {
		b.section("RESULT")
		// The assessment is laid out by the MQL printer in expected-vs-actual
		// columns; wrapping it would scramble that.
		b.code(tui.StyleDim, d.Assessment)
	}

	if len(d.FailingLocations) > 0 {
		b.section("FAILING RESOURCES")
		for _, loc := range d.FailingLocations {
			b.body(tui.StyleDim.Render(tui.Clean(loc)))
		}
	}

	if len(d.Remediation) > 0 {
		b.section("REMEDIATION")
		remediationLines(b, d.Remediation)
	}

	if len(d.References) > 0 {
		b.section("REFERENCES")
		for _, ref := range d.References {
			if ref.Title != "" {
				b.wrapped(tui.StyleText, ref.Title)
			}
			if ref.Url != "" {
				b.reference(ref.Url)
			}
		}
	}

	if len(d.Compliance) > 0 {
		b.section("COMPLIANCE")
		for _, tag := range sortedComplianceTags(d.Compliance) {
			b.wrapped(tui.StyleDim, tag+": "+d.Compliance[tag])
		}
	}

	if len(d.Policies) > 0 {
		b.section("POLICIES")
		for _, ref := range d.Policies {
			b.wrapped(tui.StyleDim, ref.Name)
		}
	}
}

// statusRow is the one line that says what happened. It is the tree's check row
// said again with the words attached:
//
//	tree     ✗ ●●●● Ensure secure permissions on /etc/group- are set
//	detail   ✗ FAIL      ●●●● CRIT   impact 100
//
// # The glyph and the word, not one or the other
//
// The tree spends one cell on an outcome and four on a severity because it is a
// list of hundreds of rows being scanned for a shape. This is one row being
// read, and it is also the first place a reader who has just met "✗" and "●●●●"
// can find out what they said. So it draws both: the glyph and the dots are the
// tree's vocabulary, unchanged and in the same order, and FAIL and CRIT are the
// legend printed next to them. Neither alone would do that job -- words only
// would make the reader re-learn the row they came from, glyphs only would leave
// the vocabulary unexplained anywhere in the viewer.
//
// Every field is fixed-width (StatusMarkWidth, SeverityMarkWidth), so the impact
// starts in the same column on every check and arrowing down the tree does not
// make the row jitter.
//
// # What was dropped, and why it was the risk
//
// The row used to carry a fourth fact: the realized risk, drawn as a number and
// its own four dots. Two four-dot groups on one line meaning different things is
// exactly the confusion the tree was built to avoid, so one of them had to go --
// and the risk is the one that says nothing the rest of the row does not.
//
// That is structural rather than a judgement about this fixture. RiskOf states a
// number only for a ScoreType_Result score, and reporter.ScoreToSarifKind
// defines PASS as such a score with value 100 and FAIL as one with value below
// it. So on a check row "risk 0" is precisely PASS, "risk 100" is precisely
// FAIL, and "-" is precisely every other outcome: the risk column was the status
// column restated as arithmetic.
//
// The impact stays, because it is the one number that is not recoverable from
// what is drawn beside it -- a check at 89 and one at 70 both say HIGH.
//
// The meter itself is not gone from the viewer. It still draws asset rows, in
// the tree and on the asset page below, where a rolled-up risk really is a fact
// of its own and there is no severity to collide with.
func statusRow(d reportmodel.CheckDetail) string {
	status := d.Status
	if status == "" {
		// reportmodel always resolves a status, but a zero-value Check does not,
		// and a row of blank cells is not an outcome anybody can read.
		status = reportmodel.StatusUnknown
	}
	row := StatusMark(status) + "  " + SeverityMark(detailSeverity(d), status.IsFinding())
	if d.HasImpact {
		row += "  " + tui.StyleFaint.Render("impact "+strconv.FormatInt(int64(d.Impact), 10))
	}
	return row
}

// detailSeverity is the severity band a check page states, and empty when the
// check states none at all. It is checkSeverity over a composed CheckDetail,
// and it exists for the same reason: reportmodel derives Severity from Impact,
// an unset impact is 0, and RiskSeverityLabel reads 0 as NONE -- so without the
// HasImpact gate the four errored ssh checks of the fixture would be badged
// NONE, which is the page claiming somebody rated them harmless.
func detailSeverity(d reportmodel.CheckDetail) string {
	if !d.HasImpact {
		return ""
	}
	return d.Severity
}

// remediationLines renders each fix. An item with a platform id ("bash",
// "terraform") is announced by it, because a page of three unlabelled fixes does
// not say which one applies to the machine in front of you.
func remediationLines(b *detailBuf, items []reportmodel.RemediationItem) {
	for i, item := range items {
		if i > 0 {
			b.blank()
		}
		if id := item.Id; id != "" && id != "default" {
			b.body(tui.StyleLabel.Render("[" + id + "]"))
		}
		b.markdown(item.Desc)
	}
}

// sortedComplianceTags orders the framework tags. A map iterates in a random
// order, and a pane whose rows shuffle between frames is unreadable.
func sortedComplianceTags(m map[string]string) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

// --- the asset page ---------------------------------------------------------

func assetLines(b *detailBuf, st *State, a *reportmodel.Asset) {
	b.para(tui.StyleText, a.Name)
	if a.PlatformName != "" {
		b.para(tui.StyleDim, a.PlatformName)
	}

	b.blank()
	// The same shape as the check page's status row -- glyph, then word -- so the
	// two pages of this pane say an outcome one way. What follows it differs
	// because the tree differs: an asset row carries the meter and no severity,
	// so there is no second dot group for it to collide with, and the meter is
	// worth more here than anywhere else. This is the page that is open when the
	// question is "how bad is this machine".
	//
	// An asset that never scanned reaches this row too, and answers "-": there is
	// no report to take a risk from, which is exactly what the section below then
	// spells out.
	//
	// The STATUS label that used to head this row is gone with it. It was naming
	// a column that now announces itself -- the glyph is the tree's, the word is
	// beside it -- and the cells go to the meter on a narrow pane.
	b.push(StatusMark(a.Status) + "  " + riskField(a.Score))

	if !a.Scanned() {
		// The whole reason reportmodel keeps ScanError: say that the scan did
		// not happen and why. An asset with no report has no checks, and drawing
		// it as a check list with nothing in it reads as a clean bill of health.
		b.section("THIS ASSET WAS NOT SCANNED")
		msg := a.ScanError
		if msg == "" {
			msg = "the scan produced no report and no error for this asset"
		}
		b.wrapped(StatusStyle(reportmodel.StatusError), msg)
		return
	}

	c := a.Counts
	b.section("CHECKS")
	b.wrapped(tui.StyleDim, fmt.Sprintf(
		"%d total · %d passed · %d failed · %d errored · %d skipped · %d unscored",
		c.Total, c.Passed, c.Failed, c.Errored, c.Skipped, c.Unscored))

	b.section("FINDINGS")
	findings := 0
	for _, check := range st.FilteredChecks(a.Checks) {
		if !check.Status.IsFinding() {
			continue
		}
		findings++
		// A list, so it takes the tree's trade rather than the status row's:
		// glyph and dots, no words. These are the same rows the tree draws for
		// the same checks, and spending twelve cells per row on PASS/FAIL/CRIT
		// here would both re-say what the shape already says and push the title
		// -- the thing a reader is looking for -- off a narrow pane.
		b.body(StatusGlyph(check.Status) + " " +
			SeverityDots(checkSeverity(check), check.Status.IsFinding()) + " " +
			tui.StyleText.Render(tui.Clean(check.Title)))
	}
	if findings == 0 {
		b.body(tui.StyleFaint.Render("none"))
	}
}

// --- input ------------------------------------------------------------------

func (p *detailPane) Update(_ *State, msg tea.Msg) (tea.Cmd, bool) {
	total, visible := len(p.lines), p.visible
	if visible < 1 {
		visible = 1
	}

	switch msg := msg.(type) {
	case ClickMsg:
		// The two things in this pane that respond to a click. A click on a
		// COPY takes that block, not the armed one: the button under the pointer
		// is the answer to which one the user meant. A click on a reference URL
		// opens it -- the terminal never sees that click, because the viewer
		// runs with mouse tracking on. See links.go.
		switch msg.Zone.Tag {
		case copyZoneTag:
			if msg.Zone.Idx < 0 || msg.Zone.Idx >= len(p.snips) {
				return nil, false
			}
			return copyCmd(p.snips[msg.Zone.Idx]), true
		case linkZoneTag:
			if msg.Zone.Idx < 0 || msg.Zone.Idx >= len(p.urls) {
				return nil, false
			}
			return openURLCmd(p.urls[msg.Zone.Idx]), true
		}
		return nil, false

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.scroll.Move(-detailWheelStep, total, visible)
		case tea.MouseButtonWheelDown:
			p.scroll.Move(detailWheelStep, total, visible)
		default:
			return nil, false
		}
		// The wheel is a plain scroll like any other: it can carry the block the
		// user armed off the screen, and when it does the arm goes with it.
		p.dropArm()
		return nil, true

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "ctrl+p":
			p.scroll.Move(-1, total, visible)
		case "down", "j", "ctrl+n":
			p.scroll.Move(1, total, visible)
		case "pgup", "ctrl+b":
			p.scroll.Move(-visible, total, visible)
		case "pgdown", "ctrl+f", " ":
			p.scroll.Move(visible, total, visible)
		case "home", "g":
			p.scroll.Move(-total, total, visible)
		case "end", "G":
			p.scroll.Move(total, total, visible)
		default:
			return nil, false
		}
		// Every branch above is a plain scroll, and a plain scroll that moves the
		// armed block off screen hands the arm back to the topmost block in view.
		p.dropArm()
		return nil, true
	}
	return nil, false
}

// detailWheelStep is how many rows one notch of the wheel moves. Three is what
// the rest of the terminal does.
const detailWheelStep = 3

// --- line building ----------------------------------------------------------

// detailBuf accumulates rendered rows and is the only thing in this file that
// appends to the output, so the invariant -- one element is one terminal row, no
// wider than the pane -- holds in one place instead of at twenty call sites.
type detailBuf struct {
	w   int
	out []string

	// snips is the check's copyable blocks, set by checkLines before any section
	// is drawn; at is how many of them have been claimed so far. buttons is
	// where the COPY for each claimed one goes.
	snips   []Snippet
	at      int
	buttons []detailButton

	// room says the pane is wide enough to set a button beside a code block
	// rather than on top of it. codeW is the width a code block is rendered at,
	// which is the body width less the button's strip when there is one.
	room  bool
	codeW int

	// urls is the clickable URLs of the page and links is where each one landed,
	// filled in by reference as the section is drawn.
	urls  []string
	links []detailLink
}

func newDetailBuf(w int) *detailBuf {
	b := &detailBuf{w: w}
	_, body := b.indent()
	b.codeW, b.room = copyWidth(body)
	return b
}

// claim takes the next snippet for a code block that is about to be drawn, and
// records a button on the row that block starts on.
//
// It records one even in a pane with no room to draw it: the row is still where
// that block begins, which is what the copy key aims with, and a terminal too
// narrow for the affordance is not a terminal where the key should go dead.
//
// The text is compared rather than trusted. detailSnippets and the sections
// below walk the same three markdown sources and the same query in the same
// order, so they line up -- but if one of them ever stops lining up, a button
// that copies the wrong block is far worse than no button, and a mismatch stops
// the claiming for the rest of the page.
func (b *detailBuf) claim(text string) {
	if b.at >= len(b.snips) || b.snips[b.at].Text != text {
		return
	}
	b.buttons = append(b.buttons, detailButton{Row: len(b.out), Idx: b.at})
	b.at++
}

// detailIndent is how far a section body is set in from its label.
const detailIndent = 2

var detailPad = strings.Repeat(" ", detailIndent)

// push adds one row, truncated to the pane. Every other method goes through it.
func (b *detailBuf) push(s string) {
	b.out = append(b.out, tui.Truncate(s, b.w))
}

func (b *detailBuf) blank() {
	b.out = append(b.out, "")
}

// body adds one already-composed row at the body indent.
func (b *detailBuf) body(s string) {
	pad, w := b.indent()
	b.push(pad + tui.Truncate(s, w))
}

// section starts a labelled block, blank-separated from whatever came before. It
// leaves no leading blank line on the first section of a page.
func (b *detailBuf) section(label string) {
	if len(b.out) > 0 {
		b.blank()
	}
	b.push(tui.StyleLabel.Render(label))
}

// para lays prose out at the left margin: the title block, which has no label
// above it to be indented under.
func (b *detailBuf) para(style lipgloss.Style, text string) {
	for _, ln := range tui.Wrap(tui.Clean(text), b.w) {
		b.push(style.Render(ln))
	}
}

// wrapped lays prose out under a section label, folded at the body width. This
// is what makes a 900-byte remediation readable in a 30-column pane.
func (b *detailBuf) wrapped(style lipgloss.Style, text string) {
	pad, w := b.indent()
	for _, ln := range tui.Wrap(tui.Clean(text), w) {
		b.push(pad + style.Render(ln))
	}
}

// reference draws one URL of the REFERENCES section.
//
// A web address becomes a link: one row, an OSC 8 hyperlink for terminals that
// understand one and a Zone for the mouse, cut at the right edge rather than
// folded. Cut rather than folded because a link has to be one row -- half a URL
// on the row below is neither clickable nor readable, and a zone spanning two
// rows is not a zone. The sequence survives the cut (see links.go), so a URL too
// wide for the pane comes out shortened and still valid.
//
// Anything that is not http or https is drawn as it always was: wrapped plain
// text. The viewer will not open it, so it must not look like something that
// opens.
func (b *detailBuf) reference(raw string) {
	text := tui.Clean(raw)
	if !linkable(text) {
		b.wrapped(tui.StyleDim, text)
		return
	}
	pad, w := b.indent()
	row := tui.Truncate(hyperlink(text, tui.StyleDim.Render(text)), w)
	b.links = append(b.links, detailLink{
		Row: len(b.out), Col: tui.Width(pad), W: tui.Width(row), Idx: len(b.urls),
	})
	b.urls = append(b.urls, text)
	b.push(pad + row)
}

// markdown renders markdown source under a section label, folded at the body
// width. The rows are pushed as glamour produced them: it brings its own colors,
// and wrapping a row in a lipgloss style would reset those escapes halfway
// through the line. A source that renders to nothing at all falls back to the
// plain wrapped text, so a section that has source is never drawn empty.
//
// The document is rendered in pieces, one per fenced code block, so that each
// block's first row is known and can carry a COPY. The pieces are separated by a
// blank row, which is the gap glamour itself puts between a paragraph and the
// code under it.
func (b *detailBuf) markdown(text string) {
	pad, w := b.indent()
	blocks := markdown.Blocks(text, w, b.codeW)
	if len(blocks) == 0 {
		b.wrapped(tui.StyleText, text)
		return
	}
	for i, blk := range blocks {
		if i > 0 {
			b.blank()
		}
		if blk.Code != nil {
			b.claim(blk.Code.Text)
		}
		for _, ln := range blk.Lines {
			b.push(pad + ln)
		}
	}
}

// code lays text out under a section label without folding it: one source line
// is one row, cut at the right edge. MQL and the assessment printer both depend
// on their own line structure, and a fold in the middle of a token turns either
// into something that no longer reads as the thing it is.
func (b *detailBuf) code(style lipgloss.Style, text string) {
	b.codeAt(style, text, b.width())
}

// codeBlock is code that can be copied: the same layout, cut a button's strip
// narrower so the COPY on its first row hides none of it.
func (b *detailBuf) codeBlock(style lipgloss.Style, text string) {
	b.claim(strings.TrimSpace(tui.Clean(text)))
	b.codeAt(style, text, b.codeW)
}

func (b *detailBuf) codeAt(style lipgloss.Style, text string, w int) {
	pad, _ := b.indent()
	for _, ln := range tui.Lines(text) {
		b.push(pad + tui.Truncate(style.Render(ln), w))
	}
}

// width is the room a section body has after the indent.
func (b *detailBuf) width() int {
	_, w := b.indent()
	return w
}

// indent is the body prefix and the width left for the body after it. A pane too
// narrow to indent into gives the text the whole width rather than one column of
// it.
func (b *detailBuf) indent() (string, int) {
	if b.w <= detailIndent+1 {
		return "", max(b.w, 1)
	}
	return detailPad, b.w - detailIndent
}
