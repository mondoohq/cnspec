// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/policy"
)

// The tree pane: asset -> policy -> check, collapsible.
//
// A scan is a tree and not a list, because a check belongs to a policy and a
// policy ran on an asset, and the same check on two assets is two different
// outcomes. Flattening that into one list of findings loses the only thing that
// tells you where to go and fix something.
//
// Everything the pane needs to remember -- the cursor, the scroll offset, which
// nodes are folded -- lives on treePane. Nothing about folding is in State,
// because no other pane has any business knowing what this one has open.
//
// # What a row is
//
// The pane keeps a flat []row, rebuilt whenever the filter or the fold state
// changes, and every visible row is exactly one terminal line and exactly one
// Zone. Row i of the slice is drawn at y = rect.Y + i - scroll, and the zone
// emitted for it carries Idx = i, which is what makes a click land on the thing
// the user pointed at rather than on the thing that was there before the last
// resize.
//
// # Folding
//
// fold records only what the user has explicitly opened or closed. Everything
// else falls back to a default, which is what lets a single-asset report open
// with its one asset already unfolded without that decision surviving as state
// the user then has to undo. See treePane.open.
//
// # Order
//
// The tree opens on what is broken, not on what happens to come first in the
// alphabet: see Order and the rank ladder below. The model hands the pane its
// nodes in name order and the pane re-sorts a copy of them, so the order is the
// viewer's decision and not something a later change to reportmodel can move.

func init() {
	RegisterTree(func(*State) Pane { return newTree() })
}

// nodeKind is the level of the tree a row sits at.
type nodeKind int

const (
	nodeAsset nodeKind = iota
	nodePolicy
	nodeCheck
)

// tag is the Zone tag for rows of this kind. It is what a click handler reads to
// know what it was given, and what a test reads to check that the click map and
// the picture agree.
func (k nodeKind) tag() string {
	switch k {
	case nodePolicy:
		return "policy"
	case nodeCheck:
		return "check"
	default:
		return "asset"
	}
}

// row is one line of the tree.
//
// key identifies the node across a rebuild: a fold state and a cursor are both
// remembered by key rather than by index, so filtering something out from above
// the cursor does not move the cursor onto a different check.
type row struct {
	kind  nodeKind
	depth int
	key   string
	// parent is the key of the row this one hangs off, empty at the top level.
	parent string
	// branch says the node has children to show. An asset that failed to scan
	// is not a branch: it is a leaf that says why, not an empty node that
	// invites you to open nothing.
	branch bool
	// open says its children are currently shown.
	open bool

	asset  *reportmodel.Asset
	policy *reportmodel.Policy
	check  *reportmodel.Check
}

// treePane is the left pane.
type treePane struct {
	cursor int
	// cursorKey is the identity of the row the cursor is on, so that a rebuild
	// puts the cursor back on the same node rather than on the same index.
	cursorKey string
	scroll    tui.Scroll
	// page is the number of rows the last render could show, which is what
	// page up and page down move by. It is a render-time fact, so it is only
	// ever written there.
	page int

	// fold holds the nodes the user has explicitly opened (true) or closed
	// (false). A node that is not in here uses its default, see open.
	fold map[string]bool
	// order is how the rows are sorted. It starts at OrderOutcome, which is the
	// whole point of the pane opening on findings rather than on the alphabet.
	order Order
	// revision is bumped by every change of the pane's own that alters the rows
	// -- a fold or a change of order -- so build can tell in one comparison
	// whether the rows it cached are still the rows to draw.
	revision int

	// rows is the flattened tree, and the three revisions below are what it was
	// built from. Rebuilding is cheap but not free -- a large scan is tens of
	// thousands of checks -- and Render is called on every mouse event.
	rows      []row
	built     bool
	builtFor  *reportmodel.Report
	filterRev int
	builtRev  int
}

func newTree() *treePane {
	return &treePane{fold: map[string]bool{}, page: defaultPage}
}

// defaultPage is what page up and page down move by before anything has been
// rendered, i.e. only in a test that drives Update without a Render.
const defaultPage = 10

func (p *treePane) ID() PaneID       { return PaneTree }
func (p *treePane) Focusable() bool  { return true }
func (p *treePane) Claims() []string { return nil }

// Hints name the keys of the pane. The sort hint says what the tree is sorted by
// now rather than what the key will do next, because "outcome" is the answer to
// the question a reader of an unfamiliar list actually has.
func (p *treePane) Hints(*State) []Hint {
	return []Hint{
		{Key: "↑/↓", Label: "move"},
		{Key: "←/→", Label: "fold"},
		{Key: "enter", Label: "open"},
		{Key: "⇧←/⇧→", Label: "fold all"},
		{Key: "s", Label: "sort: " + p.order.Label()},
	}
}

// --- building the rows ------------------------------------------------------

// build returns the flattened tree for the current filter and fold state, and
// puts the cursor back on the node it was on.
func (p *treePane) build(st *State) []row {
	if p.built && p.builtFor == st.Report && p.filterRev == st.FilterRev && p.builtRev == p.revision {
		return p.rows
	}

	p.rows = p.flatten(st)
	p.built = true
	p.builtFor = st.Report
	p.filterRev = st.FilterRev
	p.builtRev = p.revision
	p.locate()
	return p.rows
}

func (p *treePane) flatten(st *State) []row {
	assets := p.sortAssets(st.FilteredAssets())
	// A single-asset report opens on its asset already unfolded: the top level
	// of a one-asset tree carries no information the panel title does not
	// already have, and making the user press right to get past it is a step
	// that answers nothing. It stays a row, though -- it is where the asset's
	// own status and platform live, and it is the only thing to select when the
	// asset never scanned.
	sole := len(assets) == 1

	var rows []row
	for _, a := range assets {
		policies := p.policiesOf(st, a)
		key := assetKey(a)
		open := len(policies) > 0 && p.open(key, sole)
		rows = append(rows, row{
			kind: nodeAsset, key: key, branch: len(policies) > 0, open: open, asset: a,
		})
		if !open {
			continue
		}

		// The same reasoning one level down: one asset with one policy is a
		// chain of two rows that say nothing, so unfold through it.
		solePolicy := sole && len(policies) == 1
		for _, pol := range policies {
			checks := p.sortChecks(st.FilteredChecks(pol.Checks))
			pkey := key + "|p:" + policyKey(pol)
			popen := len(checks) > 0 && p.open(pkey, solePolicy)
			rows = append(rows, row{
				kind: nodePolicy, depth: 1, key: pkey, parent: key,
				branch: len(checks) > 0, open: popen, asset: a, policy: pol,
			})
			if !popen {
				continue
			}
			for _, c := range checks {
				rows = append(rows, row{
					kind: nodeCheck, depth: 2, key: pkey + "|c:" + checkKey(c), parent: pkey,
					asset: a, policy: pol, check: c,
				})
			}
		}
	}
	return rows
}

// policiesOf is the asset's policies that survive the filter, in the pane's
// order. What "survive" means is Filter.MatchPolicy's business, not this pane's:
// the header edits the filter and the tree obeys it, and the two cannot disagree
// about what it means if only one of them defines it.
func (p *treePane) policiesOf(st *State, a *reportmodel.Asset) []*reportmodel.Policy {
	if !st.Filter.Active() {
		return p.sortPolicies(a.Policies)
	}
	res := make([]*reportmodel.Policy, 0, len(a.Policies))
	for _, pol := range a.Policies {
		if st.Filter.MatchPolicy(pol) {
			res = append(res, pol)
		}
	}
	return p.sortPolicies(res)
}

// open reports whether a node's children are shown: what the user last decided,
// or the default when they have not decided anything.
func (p *treePane) open(key string, byDefault bool) bool {
	if v, ok := p.fold[key]; ok {
		return v
	}
	return byDefault
}

func assetKey(a *reportmodel.Asset) string {
	return "a:" + a.Mrn + "\x00" + a.Name
}

// policyKey identifies a policy within its asset. The synthetic "Other checks"
// node has no MRN, hence the name as well.
func policyKey(p *reportmodel.Policy) string {
	return p.Mrn + "\x00" + p.Name
}

// checkKey identifies a check within its policy. A check has an MRN or, in an
// unsigned bundle, only a code id.
func checkKey(c *reportmodel.Check) string {
	if c.Mrn != "" {
		return c.Mrn
	}
	return c.CodeId
}

// locate puts the cursor back on the row it was on before the rebuild. When
// that row is gone -- filtered out, or folded away inside a parent -- the cursor
// keeps its index and adopts whatever is there now, which is the least
// surprising thing a list can do.
func (p *treePane) locate() {
	if p.cursorKey != "" {
		for i := range p.rows {
			if p.rows[i].key == p.cursorKey {
				p.cursor = i
				return
			}
		}
	}
	p.clamp()
	p.cursorKey = p.keyAt(p.cursor)
}

func (p *treePane) clamp() {
	if p.cursor >= len(p.rows) {
		p.cursor = len(p.rows) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *treePane) keyAt(i int) string {
	if i < 0 || i >= len(p.rows) {
		return ""
	}
	return p.rows[i].key
}

// --- ordering ---------------------------------------------------------------

// Order is how the tree sorts what it draws. It is per-pane state: no other pane
// cares, so it does not live in State.
type Order int

const (
	// OrderOutcome is the default: the worst thing first, at every level of the
	// tree. A scan is read to find what needs fixing, and a viewer that opens on
	// a screen of PASS has made the reader do the scanning.
	OrderOutcome Order = iota
	// OrderName is the plain alphabet, which is what you want when you know the
	// name of the check you are looking for and not its outcome. It is also the
	// order reportmodel hands the nodes over in.
	OrderName
	// orderCount is the number of orders, so cycle wraps without a table.
	orderCount
)

// Label names the order for the footer hint.
func (o Order) Label() string {
	if o == OrderName {
		return "name"
	}
	return "outcome"
}

// The rank ladder: how loudly an outcome asks to be looked at. It is a named
// list rather than a chain of comparisons inside a less function, because the
// question it answers -- is a failure worse than an error? -- is a decision
// about the product and belongs somewhere a reader can find it and argue with
// it.
//
// FAIL outranks ERROR. Both are findings (reportmodel.Counts.Findings counts
// them together) and both need a human, but a failing check has *proved* a
// problem and names the thing to go and fix, while an errored check only proves
// that the scan could not tell. Proof before doubt.
//
// Doubt still beats good news, though, so ERROR, SKIPPED, UNSCORED and UNKNOWN
// all sort above PASS: each of them is a check that returned no verdict, which
// is a gap in the scan and worth a glance. PASS is last because it is the only
// outcome that asks nothing of the reader.
//
// This is the one place the six statuses are collapsed onto a single axis, and
// they stay six everywhere else: the ladder orders rows, it never merges them.
const (
	rankFail = iota
	rankError
	rankSkipped
	rankUnscored
	rankUnknown
	rankPass
)

// statusRank is one outcome's place on the ladder.
func statusRank(s reportmodel.Status) int {
	switch s {
	case reportmodel.StatusFail:
		return rankFail
	case reportmodel.StatusError:
		return rankError
	case reportmodel.StatusSkipped:
		return rankSkipped
	case reportmodel.StatusUnscored:
		return rankUnscored
	case reportmodel.StatusPass:
		return rankPass
	default:
		return rankUnknown
	}
}

// severityRank orders the severities, worst first. It ranks the label rather
// than the impact number so that it ranks exactly what the row draws -- the
// label and riskBandDots read the same vocabulary, policy.ScoreRatingText*, so
// the rank and the dot count move together, and an unrecognized label sorts last
// rather than first.
//
// Drawing the band as dots did not change this. The sort is a display decision
// about what the reader can see, and what they can see is still the band: four
// dots above three above two. The label stays the key because it is the thing
// both the rank and the count are derived from.
func severityRank(severity string) int {
	switch severity {
	case policy.ScoreRatingTextCritical:
		return 0
	case policy.ScoreRatingTextHigh:
		return 1
	case policy.ScoreRatingTextMedium:
		return 2
	case policy.ScoreRatingTextLow:
		return 3
	default: // NONE, and anything the reporter grows later
		return 4
	}
}

// countsRank is where a node with children sits on the same ladder: the worst
// outcome anything under it produced. A policy with one failing check outranks a
// policy with fifty errored ones, exactly as one failing check outranks one
// errored check -- the levels of the tree do not get to disagree about which
// news is worse.
//
// A tally with nothing in it has no worst outcome and falls back to the node's
// own status. That is the asset that never scanned: no checks at all, an ERROR
// of its own, and therefore a place near the top rather than at the bottom of
// the list with the clean assets.
func countsRank(c reportmodel.Counts, own reportmodel.Status) int {
	switch {
	case c.Failed > 0:
		return rankFail
	case c.Errored > 0:
		return rankError
	case c.Skipped > 0:
		return rankSkipped
	case c.Unscored > 0:
		return rankUnscored
	case c.Unknown > 0:
		return rankUnknown
	case c.Passed > 0:
		return rankPass
	default:
		return statusRank(own)
	}
}

// The three sorts below copy before they sort. FilteredAssets, policiesOf and
// FilteredChecks all hand back the model's own slice when no filter is active,
// and sorting one of those in place would reorder the model underneath every
// other reader of it.
//
// Each comparison ends in a total order over distinct nodes -- title or name,
// then MRN, then code id -- so two rows that tie on outcome and severity always
// come out the same way round. A list that reshuffles between renders is worse
// than a list in the wrong order: the wrong order can at least be learned.

func (p *treePane) sortAssets(in []*reportmodel.Asset) []*reportmodel.Asset {
	out := append([]*reportmodel.Asset(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return p.lessAsset(out[i], out[j]) })
	return out
}

func (p *treePane) sortPolicies(in []*reportmodel.Policy) []*reportmodel.Policy {
	out := append([]*reportmodel.Policy(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return p.lessPolicy(out[i], out[j]) })
	return out
}

func (p *treePane) sortChecks(in []*reportmodel.Check) []*reportmodel.Check {
	out := append([]*reportmodel.Check(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return p.lessCheck(out[i], out[j]) })
	return out
}

// lessAsset orders the assets of a report: the worst first, then the one with
// more findings in it, then the alphabet. An asset that failed to scan carries
// no checks, so it is ranked on its own ERROR status and lands among the
// errored assets rather than being sorted out of sight.
func (p *treePane) lessAsset(a, b *reportmodel.Asset) bool {
	if p.order == OrderOutcome {
		if ra, rb := countsRank(a.Counts, a.Status), countsRank(b.Counts, b.Status); ra != rb {
			return ra < rb
		}
		if fa, fb := a.Counts.Findings(), b.Counts.Findings(); fa != fb {
			return fa > fb
		}
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	return a.Mrn < b.Mrn
}

// lessPolicy orders the policies of one asset the same way, and keeps the
// synthetic "Other checks" node last among the policies it ties with: it is not
// a policy anybody wrote, so it is the last place to look at any given rank.
func (p *treePane) lessPolicy(a, b *reportmodel.Policy) bool {
	if p.order == OrderOutcome {
		if ra, rb := countsRank(a.Counts, a.Status), countsRank(b.Counts, b.Status); ra != rb {
			return ra < rb
		}
		if fa, fb := a.Counts.Findings(), b.Counts.Findings(); fa != fb {
			return fa > fb
		}
	}
	if (a.Mrn == "") != (b.Mrn == "") {
		return b.Mrn == ""
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	return a.Mrn < b.Mrn
}

// lessCheck orders the checks of one policy: outcome, then severity, then title.
//
// Severity is compared as the band and not as Check.Impact, the number behind
// it, on purpose. Impact is the finer key, but the row does not draw it:
// ordering ten HIGH checks 89, 84, 70 puts them in an order that looks like no
// order at all to someone reading ten identical rows of ●●●○. Every key this
// sort uses is a thing the row shows, so a reader can see why the list is in the
// order it is in and can predict where the next check will be.
func (p *treePane) lessCheck(a, b *reportmodel.Check) bool {
	if p.order == OrderOutcome {
		if ra, rb := statusRank(a.Status), statusRank(b.Status); ra != rb {
			return ra < rb
		}
		if ra, rb := severityRank(a.Severity), severityRank(b.Severity); ra != rb {
			return ra < rb
		}
	}
	if a.Title != b.Title {
		return a.Title < b.Title
	}
	if a.Mrn != b.Mrn {
		return a.Mrn < b.Mrn
	}
	return a.CodeId < b.CodeId
}

// cycle moves to the next order and rebuilds. The cursor is remembered by node
// rather than by index, so re-sorting the list keeps the reader on the row they
// were reading -- it moves under them, it does not swap for another one.
func (p *treePane) cycle(st *State) {
	p.order = (p.order + 1) % orderCount
	p.revision++
	p.resync(st)
	st.Notice = "sorted by " + p.order.Label()
}

// --- rendering --------------------------------------------------------------

func (p *treePane) Render(st *State, rect tui.Rect) Render {
	rows := p.build(st)
	if rect.H > 0 {
		p.page = rect.H
	}

	// The cursor is followed here rather than where it moves, because this is
	// the only place that knows how many rows fit.
	p.scroll.EnsureVisible(p.cursor, len(rows), rect.H)
	off := p.scroll.Apply(len(rows), rect.H)

	res := Render{
		Title:  p.title(st, rect),
		Status: tui.Position(p.cursor, len(rows), rect.H),
	}
	if len(rows) == 0 {
		res.Lines = []string{tui.StyleFaint.Render("no assets match")}
		return res
	}

	for i := off; i < len(rows) && i < off+rect.H; i++ {
		res.Lines = append(res.Lines, p.line(st, rows[i], i == p.cursor, rect.W))
		res.Zones = append(res.Zones, Zone{
			Rect: tui.Rect{X: rect.X, Y: rect.Y + i - off, W: rect.W, H: 1},
			Idx:  i,
			Tag:  rows[i].kind.tag(),
		})
	}
	return res
}

// title names what is on screen: the asset, when there is only one, and the
// count when there are several. It is truncated here rather than in the frame
// because tui.PanelTop drops a title it cannot fit rather than cutting it.
func (p *treePane) title(st *State, rect tui.Rect) string {
	assets := st.FilteredAssets()
	title := "Assets"
	if len(assets) == 1 {
		title = tui.Clean(assets[0].Name)
	}
	// The border eats two cells either side of the content and the position
	// indicator sits at the other end of the same edge, so the title gets what
	// is left of the content width rather than all of it.
	room := rect.W - titleMargin
	if room < 1 {
		room = 1
	}
	return tui.Truncate(title, room)
}

// line renders one row into w cells.
//
// The selected row is a full-width band, and the band's own colors have to win,
// so the styled line is stripped before it goes in. Stripping after the layout
// rather than laying out twice is safe because an escape sequence is zero cells
// wide: the stripped string is the same width as the styled one.
func (p *treePane) line(st *State, r row, selected bool, w int) string {
	left, right := p.parts(r)
	out := rowLine(left, right, w)
	if !selected {
		return out
	}
	style := tui.BandInactive
	if st.Focus == PaneTree {
		style = tui.BandSelected
	}
	return tui.Bar(ansi.Strip(out), w, style)
}

// parts renders a row as its left-hand text and the right-aligned tag that
// follows it. Both are styled; rowLine does the fitting.
//
// # Why the outcome is a glyph and not a word, on every kind of row
//
// The status column used to be eight cells of PASS/FAIL/ERROR/UNSCORED on every
// row of the pane, which is a lot of a narrow pane to spend saying the same
// half-dozen words a few hundred times. StatusGlyph says it in one, and the
// seven cells go to the name -- which is the thing the pane was truncating.
//
// It changes on the asset and policy rows too, and not only on the check rows
// this was asked for, because a status column that says "FAIL" on a policy and
// "✗" on the checks inside it is one fact in two vocabularies, in one column, on
// adjacent lines. A reader would have to learn both to read either. Nothing else
// about those rows moves: the asset keeps its risk meter and the policy its
// findings tally, which are the right-hand facts particular to each and are not
// what this is about.
func (p *treePane) parts(r row) (string, string) {
	prefix := strings.Repeat(" ", r.depth*treeIndent) + p.twisty(r) + StatusGlyph(p.status(r)) + " "

	switch r.kind {
	case nodeAsset:
		if !r.asset.Scanned() {
			// An asset that failed to scan is not an asset with no findings,
			// and the row itself has to say so: it is a leaf with an error, not
			// an empty node someone will read as a clean bill of health. The
			// label goes in the right-hand slot rather than after the name,
			// where a long asset name would push it off the end of a narrow
			// pane -- which is exactly the pane a fifteen-asset failure is read
			// in.
			return prefix + StatusStyle(reportmodel.StatusError).Render(tui.Clean(r.asset.Name)),
				tui.StyleFaint.Render(notScanned)
		}
		name := tui.StyleText.Render(tui.Clean(r.asset.Name))
		if r.asset.PlatformName != "" {
			name += tui.StyleFaint.Render(" · " + tui.Clean(r.asset.PlatformName))
		}
		// The asset's risk meter goes in the right-hand slot, which was empty
		// on this kind of row and is therefore the one place in this pane where
		// a meter costs no width that was carrying something else. It is also
		// the row where a meter is worth most: an asset is the thing a reader
		// picks between, and "which of these fifteen machines first" is exactly
		// the question one number and four dots answer. rowLine drops it on a
		// pane too narrow for it, ahead of cutting the name.
		return prefix + name, RiskMeter(r.asset.Score)

	case nodePolicy:
		// A policy row already spends its right-hand slot on the findings tally,
		// which says more about a policy than its rolled-up risk does: "3/24" is
		// how much of this policy needs work, and that is the number a reader
		// descends into it for.
		return prefix + tui.StyleText.Render(tui.Clean(r.policy.Name)), countTag(r.policy.Counts)

	default:
		// The check row draws its severity as dots rather than as the words
		// CRIT, HIGH, MED and LOW. The four cells are the same either way, so
		// this buys no width -- it buys the glance. A column of words has to be
		// read a row at a time;
		// a column of dots has a shape, and the shape is the answer to "where in
		// this policy is the damage", which is the question the pane is open
		// for. The dots are colored only where the severity was realized, so a
		// screen of passes goes grey and the findings are the only color on it.
		// See the note above SeverityDots for the whole of that reasoning.
		//
		// It is still severity and not risk: the count is what the check is
		// worth, so a passing critical check keeps its four dots. The realized
		// number is a keystroke away in the detail pane, on a row with the room
		// to label it as such.
		return prefix + SeverityDots(checkSeverity(r.check), r.check.Status.IsFinding()) + " " +
			tui.StyleText.Render(tui.Clean(r.check.Title)), ""
	}
}

// checkSeverity is the severity band a check states, and empty when it states
// none at all.
//
// reportmodel derives Check.Severity from Check.Impact, and an unset impact is
// 0, which RiskSeverityLabel reads as NONE -- so Severity alone cannot tell a
// check nobody rated from a check somebody rated as harmless. HasImpact can, and
// the two get drawn differently: blank against ○○○○. See SeverityDots.
func checkSeverity(c *reportmodel.Check) string {
	if !c.HasImpact {
		return ""
	}
	return c.Severity
}

const (
	// treeIndent is the indent per level, and also the width of the fold-marker
	// column that follows it: an asset's status begins at column 2, a policy's
	// at column 4 and a check's at column 6.
	treeIndent = 2
	// notScanned is what an asset that never ran says about itself.
	notScanned = "not scanned"
)

// twisty is the fold marker, treeIndent cells wide so that a row with one and a
// row without line up. A leaf has none, which is the difference between "there
// is nothing here" and "there is nothing here yet".
func (p *treePane) twisty(r row) string {
	switch {
	case !r.branch:
		return "  "
	case r.open:
		return tui.StyleAccent.Render("▾") + " "
	default:
		return tui.StyleDim.Render("▸") + " "
	}
}

// status is the outcome a row is drawn with. A check row shows the check's own
// outcome, a policy row the policy's, an asset row the asset's.
func (p *treePane) status(r row) reportmodel.Status {
	switch r.kind {
	case nodeAsset:
		return r.asset.Status
	case nodePolicy:
		return r.policy.Status
	default:
		return r.check.Status
	}
}

// countTag is the "3/24" a policy row carries: how many of its checks need
// attention out of how many ran. The findings half is drawn in the failure color
// when there are any, because that is the number the eye is looking for.
func countTag(c reportmodel.Counts) string {
	if c.Total == 0 {
		return ""
	}
	if f := c.Findings(); f > 0 {
		return StatusStyle(reportmodel.StatusFail).Render(strconv.Itoa(f)) +
			tui.StyleFaint.Render("/"+strconv.Itoa(c.Total))
	}
	return tui.StyleFaint.Render(strconv.Itoa(c.Total))
}

// rowLine fits a left-hand string and a right-aligned tag into w cells. The
// right-hand tag is dropped rather than allowed to squeeze the left one to
// nothing, and the result is always at most w cells wide.
func rowLine(left, right string, w int) string {
	if w < 1 {
		return ""
	}
	rw := tui.Width(right)
	if rw == 0 || rw+minLeftWidth >= w {
		return tui.Truncate(left, w)
	}
	left = tui.Truncate(left, w-rw-1)
	return left + strings.Repeat(" ", w-rw-tui.Width(left)) + right
}

// minLeftWidth is how much room the left-hand text needs before a right-aligned
// tag is worth drawing at all.
const minLeftWidth = 12

// titleMargin is the room the position indicator and the border corners need at
// the other end of the top edge the title is inlaid into.
const titleMargin = 8

// --- keys and clicks --------------------------------------------------------

func (p *treePane) Update(st *State, msg tea.Msg) (tea.Cmd, bool) {
	rows := p.build(st)

	switch msg := msg.(type) {
	case ClickMsg:
		return p.click(st, msg, rows)

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.scroll.Move(-wheelRows, len(rows), p.page)
			return nil, true
		case tea.MouseButtonWheelDown:
			p.scroll.Move(wheelRows, len(rows), p.page)
			return nil, true
		}
		return nil, false

	case tea.KeyMsg:
		return p.key(st, msg, rows)
	}
	return nil, false
}

// wheelRows is how far one notch of the wheel scrolls.
const wheelRows = 3

func (p *treePane) click(st *State, msg ClickMsg, rows []row) (tea.Cmd, bool) {
	if msg.Zone.Idx < 0 || msg.Zone.Idx >= len(rows) {
		return nil, false
	}
	p.moveTo(st, msg.Zone.Idx, rows)

	// A click on the fold marker folds; a click anywhere else on the row only
	// selects. Both select, so a click can never move the cursor somewhere the
	// detail pane is not looking.
	r := rows[msg.Zone.Idx]
	if col := msg.Mouse.X - msg.Zone.Rect.X - r.depth*treeIndent; r.branch && col >= 0 && col < treeIndent {
		p.toggle(st, r)
	}
	return nil, true
}

func (p *treePane) key(st *State, msg tea.KeyMsg, rows []row) (tea.Cmd, bool) {
	if len(rows) == 0 {
		return nil, false
	}
	cur := rows[p.cursor]

	switch msg.String() {
	case "up", "k", "ctrl+p":
		p.moveTo(st, p.cursor-1, rows)
	case "down", "j", "ctrl+n":
		p.moveTo(st, p.cursor+1, rows)
	case "pgup", "ctrl+u":
		p.moveTo(st, p.cursor-p.page, rows)
	case "pgdown", "ctrl+d":
		p.moveTo(st, p.cursor+p.page, rows)
	case "home", "g":
		p.moveTo(st, 0, rows)
	case "end", "G":
		p.moveTo(st, len(rows)-1, rows)

	case "right", "l":
		// Right walks in: it opens a closed node, steps into an open one, and
		// hands a leaf over to the detail pane, which is where the reading is.
		switch {
		case cur.branch && !cur.open:
			p.toggle(st, cur)
		case cur.branch:
			p.moveTo(st, p.cursor+1, p.build(st))
		default:
			st.Focus = PaneDetail
		}

	case "left", "h":
		// And left walks back out: it closes an open node and otherwise goes up
		// to the parent, so repeating it always ends at the top of the tree.
		switch {
		case cur.branch && cur.open:
			p.toggle(st, cur)
		case cur.parent != "":
			p.moveToKey(st, cur.parent)
		}

	case "enter", " ", "space":
		if cur.branch {
			p.toggle(st, cur)
			return nil, true
		}
		p.selectRow(st, cur)
		st.Focus = PaneDetail

	case "shift+right":
		p.foldAll(st, true)
	case "shift+left":
		p.foldAll(st, false)

	case "s":
		p.cycle(st)

	default:
		return nil, false
	}
	return nil, true
}

// moveTo puts the cursor on a row and selects it. Every cursor movement goes
// through here, which is what keeps the detail pane showing what the cursor is
// on rather than what it was on two keys ago.
func (p *treePane) moveTo(st *State, idx int, rows []row) {
	if len(rows) == 0 {
		return
	}
	p.cursor = idx
	p.clamp()
	p.cursorKey = p.keyAt(p.cursor)
	p.selectRow(st, rows[p.cursor])
}

func (p *treePane) moveToKey(st *State, key string) {
	for i := range p.rows {
		if p.rows[i].key == key {
			p.moveTo(st, i, p.rows)
			return
		}
	}
}

// selectRow publishes what the cursor is on. A check selects all three levels, a
// policy the asset and the policy, an asset only itself -- which is exactly what
// Selection documents, and what an asset that never scanned can offer.
func (p *treePane) selectRow(st *State, r row) {
	switch r.kind {
	case nodeCheck:
		st.SelectCheck(r.asset, r.policy, r.check)
	case nodePolicy:
		st.Select(Selection{Asset: r.asset, Policy: r.policy})
	default:
		st.SelectAsset(r.asset)
	}
}

// toggle folds or unfolds one node.
func (p *treePane) toggle(st *State, r row) {
	if !r.branch {
		return
	}
	p.fold[r.key] = !r.open
	p.revision++
	p.resync(st)
}

// foldAll opens or closes every asset and policy of the tree, not only the ones
// currently on screen: "collapse all" that left a node open three levels down
// would be a lie the next keypress exposes.
func (p *treePane) foldAll(st *State, open bool) {
	for _, a := range st.FilteredAssets() {
		key := assetKey(a)
		p.fold[key] = open
		for _, pol := range p.policiesOf(st, a) {
			p.fold[key+"|p:"+policyKey(pol)] = open
		}
	}
	p.revision++
	p.resync(st)
}

// resync rebuilds after a fold change and publishes whatever the cursor ended up
// on. Folding a subtree away moves the cursor, and a selection left pointing at
// a check nobody can see any more is a detail pane showing something the tree
// does not.
func (p *treePane) resync(st *State) {
	rows := p.build(st)
	if len(rows) > 0 {
		p.selectRow(st, rows[p.cursor])
	}
}
