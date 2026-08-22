// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/policy"
)

// The tree pane is tested the way the launcher is: against rendered line counts,
// widths and the zone-to-row map, never against a screenshot. A screenshot test
// fails when a colour changes and passes when the pane draws a row nobody can
// click.

// treeFor builds a pane and the state it draws, with nothing else in between.
func treeFor(t *testing.T, fixture string) (*treePane, *State) {
	t.Helper()
	return newTree(), NewState(loadReport(t, fixture))
}

// treeRect is a left pane the size the frame gives it on a 120-column terminal.
var treeRect = tui.Rect{X: 2, Y: 1, W: 41, H: 20}

// treeKey builds the key messages the pane binds, including the ones the frame's
// own helper does not need.
func treeKey(s string) tea.KeyMsg {
	types := map[string]tea.KeyType{
		"up": tea.KeyUp, "down": tea.KeyDown,
		"left": tea.KeyLeft, "right": tea.KeyRight,
		"shift+left": tea.KeyShiftLeft, "shift+right": tea.KeyShiftRight,
		"enter": tea.KeyEnter, "home": tea.KeyHome, "end": tea.KeyEnd,
		"pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown, " ": tea.KeySpace,
	}
	if kt, ok := types[s]; ok {
		return tea.KeyMsg{Type: kt}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// send presses keys and insists the pane consumed each of them.
func send(t *testing.T, p *treePane, st *State, keys ...string) {
	t.Helper()
	for _, k := range keys {
		_, ok := p.Update(st, treeKey(k))
		require.True(t, ok, "the tree should consume %q", k)
	}
}

func rowKinds(rows []row) []nodeKind {
	res := make([]nodeKind, len(rows))
	for i, r := range rows {
		res[i] = r.kind
	}
	return res
}

// --- geometry ---------------------------------------------------------------

// The invariant the whole viewer rests on: one element of Render.Lines is one
// terminal row, and no row is wider than the rect it was given. A pane that
// breaks either one scrolls the alt-screen buffer or wraps a line, and both push
// the layout off by a row that nothing afterwards can recover.
func TestTreeRowsAreOneLineAndFitTheRect(t *testing.T) {
	for _, fixture := range []string{fixtureUbuntu, fixtureK8s} {
		for _, unfold := range []bool{false, true} {
			p, st := treeFor(t, fixture)
			if unfold {
				p.foldAll(st, true)
			}
			// Widths from "one cell" upwards: the narrow end is where a
			// truncation that measures bytes instead of cells shows up.
			for _, w := range []int{1, 2, 3, 5, 8, 12, 20, 30, 41, 46, 80, 200} {
				for _, h := range []int{1, 2, 5, 20, 60} {
					rect := tui.Rect{X: 2, Y: 1, W: w, H: h}
					res := p.Render(st, rect)

					require.LessOrEqual(t, len(res.Lines), h,
						"%s w=%d h=%d: more lines than the rect is tall", fixture, w, h)
					for i, ln := range res.Lines {
						require.NotContains(t, ln, "\n",
							"%s w=%d h=%d: line %d holds a newline", fixture, w, h, i)
						require.LessOrEqual(t, tui.Width(ln), w,
							"%s w=%d h=%d: line %d is %d cells wide", fixture, w, h, i, tui.Width(ln))
					}
					require.LessOrEqual(t, tui.Width(res.Title), w,
						"%s w=%d: the title must fit the panel it is inlaid into", fixture, w)

					for _, z := range res.Zones {
						require.Equal(t, rect.X, z.Rect.X)
						require.Equal(t, rect.W, z.Rect.W)
						require.Equal(t, 1, z.Rect.H, "a row is one line, so a zone is one line")
						require.GreaterOrEqual(t, z.Rect.Y, rect.Y)
						require.Less(t, z.Rect.Y, rect.Y+rect.H)
					}
				}
			}
		}
	}
}

// What the glyph bought, measured rather than asserted. The status column was
// eight cells of PASS/FAIL/ERROR/UNSCORED on every row of the pane and is one
// now, and the seven cells went to the name -- which is the thing the pane was
// truncating, and the whole point of the change.
//
// This is pinned at a width where the titles are actually cut, because that is
// where it matters: at 41 columns, which is the left pane of a 120-column
// terminal, seven more cells is seven more characters of every check title.
func TestTheTitleGainsWhatTheStatusWordCost(t *testing.T) {
	require.Equal(t, 7, statusLabelWidth-StatusGlyphWidth,
		"the word cost eight cells and the glyph costs one")

	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)
	rows := p.build(st)

	// The prefix a check row spends before its title: the indent, the fold
	// column, the glyph and the dots, each with the space after it that keeps
	// the columns apart.
	const prefix = 2*treeIndent + treeIndent + StatusGlyphWidth + 1 + SeverityDotsWidth + 1
	require.Equal(t, 13, prefix)

	for _, w := range []int{41, 74} {
		res := p.Render(st, tui.Rect{X: 0, Y: 0, W: w, H: 40})
		for i, r := range rows {
			if r.kind != nodeCheck {
				continue
			}
			line := ansi.Strip(res.Lines[i])
			// Every title starts at the same column, and gets everything left.
			require.Equal(t, min(tui.Width(r.check.Title), w-prefix), tui.Width(line)-prefix,
				"w=%d row %d: %q", w, i, line)
		}
	}
}

// Every zone names the row that was drawn where it points. The two come out of
// one pass over one rect, and this is what proves they did not drift: zone k
// sits on line k and its Idx is the row that produced that line.
func TestTreeZonesMapToTheRowsTheyDraw(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	// Wide enough that nothing is truncated, so the assertion is about which
	// row was drawn rather than about how much of it fits.
	rect := tui.Rect{X: 4, Y: 2, W: 120, H: 12}
	rows := p.build(st)
	res := p.Render(st, rect)

	require.Len(t, res.Zones, len(res.Lines), "one zone per drawn row, no more and no less")
	require.Len(t, res.Lines, rect.H)

	for k, z := range res.Zones {
		require.Equal(t, rect.Y+k, z.Rect.Y, "zone %d must sit on line %d", k, k)
		require.Equal(t, res.Zones[0].Idx+k, z.Idx, "zones follow the rows in order")

		r := rows[z.Idx]
		line := ansi.Strip(res.Lines[k])
		require.Equal(t, r.kind.tag(), z.Tag)
		require.Contains(t, line, statusGlyph(p.status(r)), "row %d draws its own status", z.Idx)

		switch r.kind {
		case nodeAsset:
			require.Contains(t, line, r.asset.Name)
		case nodePolicy:
			require.Contains(t, line, r.policy.Name)
		default:
			require.Contains(t, line, r.check.Title)
		}
	}
}

// Scrolling moves which rows are drawn, not which row a zone claims: after a
// jump to the end, zone k still names the row on line k.
func TestTreeZonesFollowScrolling(t *testing.T) {
	p, st := treeFor(t, fixtureK8s)
	rect := tui.Rect{X: 2, Y: 1, W: 60, H: 5}

	p.Render(st, rect) // the render is what tells the pane how many rows fit
	send(t, p, st, "end")
	rows := p.build(st)
	res := p.Render(st, rect)

	require.Len(t, res.Lines, 5)
	require.Len(t, res.Zones, 5)
	require.Positive(t, p.scroll.Off, "fifteen assets do not fit in five lines")
	require.Equal(t, len(rows)-1, p.cursor)

	for k, z := range res.Zones {
		require.Equal(t, p.scroll.Off+k, z.Idx)
		require.Equal(t, rect.Y+k, z.Rect.Y)
		require.Contains(t, ansi.Strip(res.Lines[k]), rows[z.Idx].asset.Name)
	}
	// And the cursor is one of the rows on screen, which is the point of
	// following it in the first place.
	require.Equal(t, p.cursor, res.Zones[len(res.Zones)-1].Idx)
}

// A pane one cell wide still renders: it is what a terminal dragged to nothing
// gives, and the answer has to be a short line rather than a panic.
func TestTreeSurvivesAOneCellRect(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	res := p.Render(st, tui.Rect{X: 0, Y: 0, W: 1, H: 1})
	require.Len(t, res.Lines, 1)
	require.LessOrEqual(t, tui.Width(res.Lines[0]), 1)

	require.Empty(t, rowLine("anything", "12", 0), "a rect with no width draws no row")
}

// --- what the tree is made of -----------------------------------------------

// An asset that failed to scan is not an asset with no findings. The k8s fixture
// is fifteen of them, with no bundle at all: fifteen rows, every one of them
// labelled, and not one of them an empty node that invites you to open nothing.
func TestErroredAssetsAreLabelledLeaves(t *testing.T) {
	p, st := treeFor(t, fixtureK8s)
	rows := p.build(st)

	require.Len(t, rows, 15)
	for _, r := range rows {
		require.Equal(t, nodeAsset, r.kind)
		require.False(t, r.branch, "an asset that never scanned has nothing to unfold")
		require.False(t, r.asset.Scanned())
		require.Equal(t, reportmodel.StatusError, r.asset.Status)
	}

	// Unfolding everything cannot conjure children for them either.
	p.foldAll(st, true)
	require.Len(t, p.build(st), 15)

	// Wide enough that the longest asset name is not cut, so the assertion is
	// about what the row says rather than about how much of it fits.
	res := p.Render(st, tui.Rect{X: 0, Y: 0, W: 80, H: 20})
	require.Len(t, res.Lines, 15)
	require.Len(t, res.Zones, 15)
	for i, ln := range res.Lines {
		line := ansi.Strip(ln)
		require.Contains(t, line, statusGlyph(reportmodel.StatusError), "row %d", i)
		require.Contains(t, line, "not scanned", "row %d must say why it is empty", i)
		require.Contains(t, line, rows[i].asset.Name)
	}
}

// One asset is not a level worth walking through, so a single-asset report opens
// with its asset already unfolded. Several assets are, so they start folded and
// the tree opens as a list of assets.
func TestSingleAssetOpensUnfolded(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	rows := p.build(st)

	require.Equal(t, []nodeKind{nodeAsset, nodePolicy, nodePolicy}, rowKinds(rows),
		"the one asset is open and its policies are the level you land on")
	require.True(t, rows[0].open)
	require.True(t, rows[0].branch)
	for _, r := range rows[1:] {
		require.False(t, r.open, "the policies themselves start folded")
		require.True(t, r.branch)
	}

	multi, mst := treeFor(t, fixtureK8s)
	for _, r := range multi.build(mst) {
		require.False(t, r.open, "several assets start folded, so the tree opens as a list")
	}
}

// The tree keeps all six outcomes apart. A check that could not run is not a
// check that passed, and a viewer that folds the two together is worse than one
// that shows nothing.
func TestStatusesStayDistinct(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	var counts reportmodel.Counts
	for _, r := range p.build(st) {
		if r.kind == nodeCheck {
			counts.Add(p.status(r))
		}
	}
	require.Equal(t, st.Report.Assets[0].Counts, counts)
	require.Equal(t, 24, counts.Total)
	require.Equal(t, 18, counts.Passed)
	require.Equal(t, 2, counts.Failed)
	require.Equal(t, 4, counts.Errored)

	// And the rows say so in a shape, not only in colour: colour is the first
	// thing a pipe, a screenshot or a colour-blind reader loses. The row draws
	// its outcome as one glyph now rather than as a word, and the glyphs are six
	// distinct shapes for exactly this reason -- see
	// TestTheStatusGlyphKeepsTheOutcomesApart.
	res := p.Render(st, tui.Rect{X: 0, Y: 0, W: 80, H: 40})
	var passed, failed, errored int
	for _, ln := range res.Lines {
		// The fold marker leads the row, so it comes off before the glyph.
		switch fields := strings.Fields(strings.TrimLeft(ansi.Strip(ln), " ▾▸")); fields[0] {
		case statusGlyph(reportmodel.StatusPass):
			passed++
		case statusGlyph(reportmodel.StatusFail):
			failed++
		case statusGlyph(reportmodel.StatusError):
			errored++
		}
	}
	require.Equal(t, 18, passed)
	require.Equal(t, 4+1, errored, "four errored checks and the policy they belong to")
	require.Equal(t, 2+1+1, failed, "two failing checks, their policy, and the asset")
}

// --- ordering ---------------------------------------------------------------

// treeText renders the pane wide enough that nothing is truncated and reduces
// each row to its words, so the assertions below are about what a row says and
// where it sits rather than about where the padding fell.
func treeText(t *testing.T, p *treePane, st *State) []string {
	t.Helper()
	res := p.Render(st, tui.Rect{X: 0, Y: 0, W: 74, H: 40})
	out := make([]string, len(res.Lines))
	for i, ln := range res.Lines {
		out[i] = strings.Join(strings.Fields(ansi.Strip(ln)), " ")
	}
	return out
}

func checkTitles(rows []row) []string {
	var res []string
	for _, r := range rows {
		if r.kind == nodeCheck {
			res = append(res, r.check.Title)
		}
	}
	return res
}

func policyNames(rows []row) []string {
	var res []string
	for _, r := range rows {
		if r.kind == nodePolicy {
			res = append(res, r.policy.Name)
		}
	}
	return res
}

func assetNames(rows []row) []string {
	var res []string
	for _, r := range rows {
		if r.kind == nodeAsset {
			res = append(res, r.asset.Name)
		}
	}
	return res
}

// The default order, pinned. This is the whole point of the pane: a scan is
// opened to find out what is broken, and the two failing checks of the ubuntu
// fixture are the first two checks on the screen rather than lines 12 and 16 of
// a wall of PASS.
//
// Every key the sort uses is a column the row draws -- outcome, then the
// severity band, then the title -- so this listing is also the proof that a
// reader can see why the list is in the order it is in. Both of those columns
// are now drawn rather than spelled: "✗" is FAIL and "●●●●" is CRITICAL, and the
// listing below is what the ladder looks like once they are, with ●●●● above
// ●●●○ inside each run of one outcome.
//
// The asset row carries its risk meter at the right, which is the one row kind
// that does: see treePane.parts for why the policy and check rows do not. Its
// four dots are the asset's realized risk and not a severity -- the two are told
// apart by which end of the row they are on, and by the number in front of them.
func TestTheDefaultOrderLeadsWithFindings(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	require.Equal(t, []string{
		"▾ ✗ ubuntu:24.04 · Ubuntu 24.04.2 LTS 100 ●●●●",
		"▾ ✗ Mondoo Linux Security 2/20",
		"✗ ●●●● Ensure secure permissions on /etc/group- are set",
		"✗ ●●●● Ensure secure permissions on /etc/passwd- are set",
		"✓ ●●●● Ensure X Window System is not installed",
		"✓ ●●●● Ensure root group is empty",
		"✓ ●●●● Ensure secure permissions on /etc/group are set",
		"✓ ●●●● Ensure secure permissions on /etc/gshadow are set",
		"✓ ●●●● Ensure secure permissions on /etc/gshadow- are set",
		"✓ ●●●● Ensure secure permissions on /etc/passwd are set",
		"✓ ●●●● Ensure secure permissions on /etc/shadow are set",
		"✓ ●●●● Ensure secure permissions on /etc/shadow- are set",
		"✓ ●●●○ Ensure all GIDs in /etc/passwd exist in /etc/group",
		"✓ ●●●○ Ensure default group for the root account is GID 0",
		"✓ ●●●○ Ensure each user is a member of a group",
	}, treeText(t, p, st)[:15])

	// And the tail of the same listing, which is where the two cases the
	// severity column has to keep apart both live: four errored checks that
	// state no severity at all, drawn blank, under a policy row that says the
	// same thing with "!" and a 4/4 tally. treeText squeezes the runs of spaces
	// out, so the blank column shows up here as the absence of any dots --
	// TestACheckWithNoSeverityDrawsNoDots measures the cells.
	require.Equal(t, []string{
		"▾ ! Mondoo SSH Server Security 4/4",
		"! Enable strict mode",
		"! Only use strong Ciphers",
		"! Set the port to 22",
		"! Set the protocol to 2",
	}, treeText(t, p, st)[22:])

	// The errored policy comes after the failing one, and its checks with it.
	require.Equal(t, []string{"Mondoo Linux Security", "Mondoo SSH Server Security"},
		policyNames(p.build(st)))
}

// A list that reshuffles between renders is worse than a list in the wrong
// order, because the wrong order can at least be learned. Every tie is broken by
// title and then by MRN, so the sequence is the same on every pass -- from the
// same pane, and from a pane that has never drawn anything.
func TestTheOrderIsIdenticalOnEveryRender(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	want := treeText(t, p, st)
	require.Len(t, want, 27)
	for i := 0; i < 5; i++ {
		require.Equal(t, want, treeText(t, p, st), "render %d drew a different list", i)
	}

	// A fold round-trip goes through a rebuild rather than through the cache,
	// which is where a sort that depended on the incoming order would show up.
	p.foldAll(st, false)
	p.foldAll(st, true)
	require.Equal(t, want, treeText(t, p, st))

	// And a second pane over a second load of the same fixture agrees, so the
	// order is a function of the report and of nothing else.
	fresh, fst := treeFor(t, fixtureUbuntu)
	fresh.foldAll(fst, true)
	require.Equal(t, want, treeText(t, fresh, fst))
}

// The pane sorts a copy. reportmodel hands out its own slices when no filter is
// active, and a sort in place would reorder the model under the header, the
// detail pane and the next reader of Report.Assets.
func TestSortingDoesNotReorderTheModel(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	asset := st.Report.Assets[0]

	before := append([]*reportmodel.Check(nil), asset.Checks...)
	policies := append([]*reportmodel.Policy(nil), asset.Policies...)
	own := append([]*reportmodel.Check(nil), asset.Policies[0].Checks...)

	p.foldAll(st, true)
	p.build(st)

	require.Equal(t, before, asset.Checks, "the model's checks are still in model order")
	require.Equal(t, policies, asset.Policies)
	require.Equal(t, own, asset.Policies[0].Checks)
	require.IsIncreasing(t, checkTitlesOf(before), "and model order is still the alphabet")
}

func checkTitlesOf(checks []*reportmodel.Check) []string {
	res := make([]string, len(checks))
	for i, c := range checks {
		res[i] = c.Title
	}
	return res
}

// The ladder, spelled out. Which outcome is worse than which is a decision about
// the product rather than an accident of a comparison function, so it is a named
// list in tree.go and this is the test that says what the list means.
//
// FAIL outranks ERROR: both are findings, but a failing check has proved a
// problem and named the thing to fix, while an errored check has only proved
// that the scan could not tell. Everything that returned no verdict at all still
// sorts above PASS, which is last because it is the only outcome that asks
// nothing of the reader.
func TestTheRankLadderRunsFromFailToPass(t *testing.T) {
	ladder := []reportmodel.Status{
		reportmodel.StatusFail,
		reportmodel.StatusError,
		reportmodel.StatusSkipped,
		reportmodel.StatusUnscored,
		reportmodel.StatusUnknown,
		reportmodel.StatusPass,
	}
	ranks := make([]int, len(ladder))
	for i, s := range ladder {
		ranks[i] = statusRank(s)
	}
	require.IsIncreasing(t, ranks, "the six outcomes are six distinct ranks, in this order")
	require.Equal(t, rankUnknown, statusRank(reportmodel.Status("something new")))

	// Severity is ranked as the badge the row draws, not as the impact number
	// behind it, so the order is one a reader can see.
	sev := []int{
		severityRank(policy.ScoreRatingTextCritical),
		severityRank(policy.ScoreRatingTextHigh),
		severityRank(policy.ScoreRatingTextMedium),
		severityRank(policy.ScoreRatingTextLow),
		severityRank(policy.ScoreRatingTextNone),
	}
	require.IsIncreasing(t, sev)
	require.Equal(t, severityRank(policy.ScoreRatingTextNone), severityRank("whatever comes next"))
}

// A node with children is ranked by the worst outcome under it, so the levels of
// the tree cannot disagree about which news is worse. A tally with nothing in it
// falls back to the node's own status, which is all an asset that never scanned
// has to offer.
func TestCountsRankIsTheWorstOutcomeUnderTheNode(t *testing.T) {
	tally := func(pass, fail, errored, skipped int) reportmodel.Counts {
		var c reportmodel.Counts
		for i := 0; i < pass; i++ {
			c.Add(reportmodel.StatusPass)
		}
		for i := 0; i < fail; i++ {
			c.Add(reportmodel.StatusFail)
		}
		for i := 0; i < errored; i++ {
			c.Add(reportmodel.StatusError)
		}
		for i := 0; i < skipped; i++ {
			c.Add(reportmodel.StatusSkipped)
		}
		return c
	}

	require.Equal(t, rankFail, countsRank(tally(99, 1, 50, 0), reportmodel.StatusPass),
		"one failure outranks fifty errors, exactly as one failing check does")
	require.Equal(t, rankError, countsRank(tally(99, 0, 1, 0), reportmodel.StatusPass))
	require.Equal(t, rankSkipped, countsRank(tally(99, 0, 0, 1), reportmodel.StatusPass))
	require.Equal(t, rankPass, countsRank(tally(1, 0, 0, 0), reportmodel.StatusPass))
	require.Equal(t, rankError, countsRank(reportmodel.Counts{}, reportmodel.StatusError),
		"no checks at all is the asset that never scanned, and it is an ERROR")
}

// --- ordering: the levels above a check -------------------------------------

// The model nodes below are built by hand rather than loaded, because the
// fixtures hold one shape each and the ordering has to be shown on a report
// where the alphabet and the findings disagree. They are model data only: an
// Asset built this way has no report behind it, so these tests read rows and
// never render them.

func synthCheck(title string, status reportmodel.Status) *reportmodel.Check {
	return &reportmodel.Check{
		Mrn: "//check/" + title, CodeId: title, Title: title,
		Status: status, Severity: policy.ScoreRatingTextNone,
	}
}

func synthPolicy(mrn, name string, statuses ...reportmodel.Status) *reportmodel.Policy {
	p := &reportmodel.Policy{Mrn: mrn, Name: name, Status: reportmodel.StatusPass}
	for i, s := range statuses {
		p.Checks = append(p.Checks, synthCheck(name+"/"+strconv.Itoa(i), s))
		p.Counts.Add(s)
	}
	return p
}

func synthAsset(name string, status reportmodel.Status, policies ...*reportmodel.Policy) *reportmodel.Asset {
	a := &reportmodel.Asset{Mrn: "//asset/" + name, Name: name, Status: status}
	for _, p := range policies {
		a.Policies = append(a.Policies, p)
		a.Checks = append(a.Checks, p.Checks...)
		for _, c := range p.Checks {
			a.Counts.Add(c.Status)
		}
	}
	return a
}

func synthState(assets ...*reportmodel.Asset) *State {
	return NewState(&reportmodel.Report{Assets: assets})
}

// Policies are ordered like the checks inside them: the worst outcome first,
// then the one carrying more findings, then the alphabet. The synthetic "Other
// checks" node keeps its place at the back of whatever rank it lands in -- it is
// not a policy anybody wrote, so it is the last place to look at that rank.
func TestPoliciesLeadWithTheirWorstOutcome(t *testing.T) {
	pass, fail, erred := reportmodel.StatusPass, reportmodel.StatusFail, reportmodel.StatusError
	asset := synthAsset("host", fail,
		synthPolicy("//p/a", "aaa clean", pass, pass),
		synthPolicy("//p/b", "bbb errored", erred),
		synthPolicy("//p/c", "ccc one failure", fail, pass),
		synthPolicy("//p/d", "ddd three failures", fail, fail, fail),
		synthPolicy("", reportmodel.UngroupedPolicyName, fail, pass),
	)

	p := newTree()
	st := synthState(asset)
	require.Equal(t, []string{
		"ddd three failures",
		"ccc one failure",
		reportmodel.UngroupedPolicyName,
		"bbb errored",
		"aaa clean",
	}, policyNames(p.build(st)))

	// And the alphabet is still there for anyone who wants it, synthetic node
	// last as the model has it.
	send(t, p, st, "s")
	require.Equal(t, []string{
		"aaa clean", "bbb errored", "ccc one failure", "ddd three failures",
		reportmodel.UngroupedPolicyName,
	}, policyNames(p.build(st)))
}

// Assets are ordered the same way, and an asset that never scanned is ranked on
// the only outcome it has: its own ERROR. It lands among the errored assets --
// visible, and above everything that is merely clean -- rather than sorted into
// oblivion at the bottom of the list.
func TestAssetsLeadWithTheirWorstOutcome(t *testing.T) {
	pass, fail, erred := reportmodel.StatusPass, reportmodel.StatusFail, reportmodel.StatusError
	st := synthState(
		synthAsset("alpha clean", pass, synthPolicy("//p/a", "policy", pass, pass)),
		synthAsset("bravo never scanned", erred),
		synthAsset("charlie one failure", fail, synthPolicy("//p/c", "policy", fail, pass)),
		synthAsset("delta two failures", fail, synthPolicy("//p/d", "policy", fail, fail)),
	)

	p := newTree()
	require.Equal(t, []string{
		"delta two failures",
		"charlie one failure",
		"bravo never scanned",
		"alpha clean",
	}, assetNames(p.build(st)))

	// The asset with no checks is a leaf, and it is still one of the four rows.
	rows := p.build(st)
	require.Len(t, rows, 4, "several assets start folded, so this is the whole tree")
	require.False(t, rows[2].branch, "an asset that never scanned has nothing to unfold")

	send(t, p, st, "s")
	require.Equal(t, []string{
		"alpha clean", "bravo never scanned", "charlie one failure", "delta two failures",
	}, assetNames(p.build(st)))
}

// The fifteen assets of the k8s fixture all errored, so the outcome key cannot
// separate them and the tie-break has to: fifteen rows, in name order, the same
// fifteen rows the alphabet gives.
func TestAllErroredAssetsFallBackToTheAlphabet(t *testing.T) {
	p, st := treeFor(t, fixtureK8s)
	byOutcome := assetNames(p.build(st))
	require.Len(t, byOutcome, 15)
	require.IsIncreasing(t, byOutcome)

	send(t, p, st, "s")
	require.Equal(t, byOutcome, assetNames(p.build(st)))
}

// --- ordering: getting the alphabet back ------------------------------------

// One good default is not quite enough: a reader who knows the name of the check
// they want should not have to read an outcome-ordered list to find it. "s"
// cycles, the footer says which order is showing, and the cursor stays on the
// row it was on -- the list moves under the reader rather than handing them a
// different row.
func TestTheSortKeyCyclesTheOrder(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	byOutcome := treeText(t, p, st)
	require.Equal(t, OrderOutcome, p.order, "the tree opens on findings")
	require.Contains(t, p.Hints(st), Hint{Key: "s", Label: "sort: outcome"})

	// Park the cursor on the second failing check, which the alphabet puts
	// somewhere in the middle of the passes.
	rows := p.build(st)
	p.moveTo(st, 3, rows)
	want := st.Sel.Check
	require.Equal(t, reportmodel.StatusFail, want.Status)

	send(t, p, st, "s")
	require.Equal(t, OrderName, p.order)
	require.Equal(t, "sorted by name", st.Notice, "the footer says what just happened")
	require.Contains(t, p.Hints(st), Hint{Key: "s", Label: "sort: name"})

	rows = p.build(st)
	require.Equal(t, want, rows[p.cursor].check, "the cursor stayed on the check it was on")
	require.Equal(t, want, st.Sel.Check, "and the detail pane is still showing it")
	require.Greater(t, p.cursor, 3, "which the alphabet has moved down the list")

	// Name order is the model's own order: titles ascending within each policy.
	require.Equal(t, checkTitlesOf(st.Report.Assets[0].Policies[0].Checks),
		checkTitles(rows)[:20])
	require.NotEqual(t, byOutcome, treeText(t, p, st))

	send(t, p, st, "s")
	require.Equal(t, OrderOutcome, p.order, "and cycles back round")
	require.Equal(t, byOutcome, treeText(t, p, st), "to exactly the list it started with")
	require.Equal(t, want, st.Sel.Check)
}

// The sort key has to be the tree's alone. The header claims "/" and "f" from
// anywhere, and a key two panes both want is a key that does something different
// depending on where the focus happens to be.
func TestTheSortKeyCollidesWithNothing(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	for _, id := range []PaneID{PaneHeader, PaneDetail} {
		require.NotContains(t, buildPane(id, st).Claims(), "s",
			"the %s pane claims the tree's sort key", id)
	}

	// And it is not one of the frame's own bindings either: the tree consumes
	// it, so the frame never sees it.
	p := newTree()
	_, consumed := p.Update(st, treeKey("s"))
	require.True(t, consumed)
}

// --- folding ----------------------------------------------------------------

// Right opens a node, steps into an open one and hands a leaf to the detail
// pane; left closes it again and otherwise climbs back out. Repeating left ends
// at the top of the tree rather than somewhere in the middle of it.
func TestFoldingWithTheArrowKeys(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	require.Len(t, p.build(st), 3)

	send(t, p, st, "down") // onto the first policy
	require.Equal(t, nodePolicy, p.build(st)[p.cursor].kind)

	send(t, p, st, "right")
	rows := p.build(st)
	require.Len(t, rows, 3+20, "the first policy holds twenty checks")
	require.Equal(t, 1, p.cursor, "opening a node does not move the cursor off it")
	require.True(t, rows[1].open)

	send(t, p, st, "right")
	require.Equal(t, 2, p.cursor, "right again steps into the open node")
	require.Equal(t, nodeCheck, p.build(st)[p.cursor].kind)

	send(t, p, st, "left")
	require.Equal(t, 1, p.cursor, "left climbs from a leaf to its parent")

	send(t, p, st, "left")
	require.Len(t, p.build(st), 3, "and closes the parent it climbed to")

	send(t, p, st, "left")
	require.Equal(t, 0, p.cursor, "left again climbs to the asset")
	send(t, p, st, "left")
	require.Len(t, p.build(st), 1, "which folds away to a single row")
}

// Enter and space toggle a branch, which is the other half of the same idea for
// anyone who reaches for a mouse-shaped key rather than an arrow.
func TestEnterAndSpaceToggleABranch(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	send(t, p, st, "down", "enter")
	require.Len(t, p.build(st), 23)
	send(t, p, st, "enter")
	require.Len(t, p.build(st), 3)
	send(t, p, st, " ")
	require.Len(t, p.build(st), 23)
}

// Fold-all works on the whole tree and not only on what is on screen, so the
// answer does not depend on where the cursor happened to be.
func TestFoldAllOpensAndClosesEverything(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)

	send(t, p, st, "shift+right")
	require.Len(t, p.build(st), 1+2+24, "the asset, both policies and every check")

	send(t, p, st, "shift+left")
	require.Len(t, p.build(st), 1, "and back to the asset alone")
	require.Equal(t, 0, p.cursor)
	require.Equal(t, st.Report.Assets[0], st.Sel.Asset)
	require.Nil(t, st.Sel.Check, "folding away the selected check must not leave it selected")
}

// --- selection --------------------------------------------------------------

// Moving the cursor is what the detail pane watches, so every level of the tree
// has to publish what it is: a check selects all three levels, a policy two and
// an asset one.
func TestSelectionFollowsTheCursor(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)
	rows := p.build(st)
	before := st.SelectionRev

	send(t, p, st, "home")
	require.Equal(t, rows[0].asset, st.Sel.Asset)
	require.Nil(t, st.Sel.Policy)
	require.Nil(t, st.Sel.Check)

	send(t, p, st, "down")
	require.Equal(t, nodePolicy, rows[1].kind)
	require.Equal(t, rows[1].policy, st.Sel.Policy)
	require.Nil(t, st.Sel.Check)

	send(t, p, st, "down")
	require.Equal(t, nodeCheck, rows[2].kind)
	require.Equal(t, rows[2].asset, st.Sel.Asset)
	require.Equal(t, rows[2].policy, st.Sel.Policy)
	require.Equal(t, rows[2].check, st.Sel.Check)
	require.Greater(t, st.SelectionRev, before)

	// Re-selecting the same row is not a selection change: a detail pane that
	// scrolled back to the top on every keypress would be unusable.
	rev := st.SelectionRev
	send(t, p, st, "down", "up")
	require.Equal(t, rev+2, st.SelectionRev)
	p.Update(st, treeKey("down"))
	p.Update(st, treeKey("up"))
	require.Equal(t, rev+4, st.SelectionRev)
	send(t, p, st, "home")
	rev = st.SelectionRev
	send(t, p, st, "home")
	require.Equal(t, rev, st.SelectionRev, "the cursor did not move, so nothing was reselected")
}

// A leaf has nothing to open, so opening it means reading it: the tree hands the
// keyboard to the detail pane.
func TestALeafHandsOffToTheDetailPane(t *testing.T) {
	p, st := treeFor(t, fixtureK8s)
	st.Focus = PaneTree

	send(t, p, st, "enter")
	require.Equal(t, PaneDetail, st.Focus)
	require.Equal(t, st.Report.Assets[0], st.Sel.Asset)

	st.Focus = PaneTree
	send(t, p, st, "right")
	require.Equal(t, PaneDetail, st.Focus, "right on a leaf reads it as well")

	// A branch is a different matter: enter opens it and focus stays put.
	tree, ust := treeFor(t, fixtureUbuntu)
	ust.Focus = PaneTree
	send(t, tree, ust, "down", "enter")
	require.Equal(t, PaneTree, ust.Focus)
}

// --- filtering --------------------------------------------------------------

// Filtering is the frame's, and the tree only obeys it. Restricting to one
// status leaves the checks of that status, the policies that hold them and the
// assets they ran on -- and nothing else.
func TestFilterNarrowsTheTree(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusFail: true}})
	rows := p.build(st)
	require.Equal(t, []nodeKind{nodeAsset, nodePolicy, nodeCheck, nodeCheck}, rowKinds(rows),
		"one asset and one policy left, so the tree unfolds through both")
	for _, r := range rows {
		if r.kind == nodeCheck {
			require.Equal(t, reportmodel.StatusFail, r.check.Status)
		}
	}

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusError: true}})
	rows = p.build(st)
	require.Len(t, rows, 1+1+4, "the four errored checks are a different policy")
	for _, r := range rows {
		if r.kind == nodeCheck {
			require.Equal(t, reportmodel.StatusError, r.check.Status)
		}
	}

	st.SetFilter(Filter{})
	require.Len(t, p.build(st), 3, "clearing the filter restores the tree")
}

// The cursor is remembered by node, not by index, so filtering something out
// from above it does not silently move it onto a different check.
func TestTheCursorSurvivesAFilterChange(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	p.foldAll(st, true)

	// Park the cursor on a failing check somewhere down the list.
	rows := p.build(st)
	var want *reportmodel.Check
	for i, r := range rows {
		if r.kind == nodeCheck && r.check.Status == reportmodel.StatusFail {
			p.moveTo(st, i, rows)
			want = r.check
			break
		}
	}
	require.NotNil(t, want)
	require.Positive(t, p.cursor)

	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusFail: true}})
	rows = p.build(st)
	require.Equal(t, want, rows[p.cursor].check, "the cursor stayed on the same check")

	// And when the cursor's row is filtered away it keeps its place in the list
	// rather than jumping to the top or off the end.
	st.SetFilter(Filter{Statuses: map[reportmodel.Status]bool{reportmodel.StatusPass: true}})
	rows = p.build(st)
	require.GreaterOrEqual(t, p.cursor, 0)
	require.Less(t, p.cursor, len(rows))
	require.Equal(t, rows[p.cursor].key, p.cursorKey)
}

// A filter that matches nothing draws one honest line and no zones, rather than
// an empty panel that reads as a clean bill of health.
func TestAnEmptyTreeSaysSo(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	st.SetFilter(Filter{Search: "no-check-is-called-this"})

	require.Empty(t, p.build(st))
	res := p.Render(st, treeRect)
	require.Len(t, res.Lines, 1)
	require.Contains(t, ansi.Strip(res.Lines[0]), "no assets match")
	require.Empty(t, res.Zones)
	require.Equal(t, "0", res.Status)

	// And the keys do not panic on it.
	for _, k := range []string{"down", "up", "right", "left", "enter", "end"} {
		_, consumed := p.Update(st, treeKey(k))
		require.False(t, consumed, "%q has nothing to act on", k)
	}
}

// --- mouse ------------------------------------------------------------------

// A click selects the row it landed on; a click on the fold marker of a branch
// selects it and folds it, so the pointer can do what the arrow keys do.
func TestClickSelectsAndTheTwistyFolds(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	res := p.Render(st, treeRect)
	require.Len(t, res.Zones, 3)

	// Somewhere in the middle of the second row: the policy, not its marker.
	z := res.Zones[1]
	_, consumed := p.Update(st, ClickMsg{Zone: z, Mouse: tea.MouseMsg{X: z.Rect.X + 10, Y: z.Rect.Y}})
	require.True(t, consumed)
	require.Equal(t, 1, p.cursor)
	require.Equal(t, p.build(st)[1].policy, st.Sel.Policy)
	require.Len(t, p.build(st), 3, "a click on the row itself does not unfold it")

	// And now on the marker of the same row, which sits at the row's own indent.
	_, consumed = p.Update(st, ClickMsg{
		Zone:  z,
		Mouse: tea.MouseMsg{X: z.Rect.X + treeIndent, Y: z.Rect.Y},
	})
	require.True(t, consumed)
	require.Len(t, p.build(st), 23, "the marker unfolds")
	require.Equal(t, 1, p.cursor)

	// A zone that names a row the tree no longer has is ignored rather than
	// indexed into.
	_, consumed = p.Update(st, ClickMsg{Zone: Zone{Idx: 9999, Tag: "check"}})
	require.False(t, consumed)
}

// The wheel scrolls what the pointer is over without moving the cursor: taking
// the selection along would mean looking at a list changes what is open in the
// pane next to it.
func TestTheWheelScrollsWithoutMovingTheCursor(t *testing.T) {
	p, st := treeFor(t, fixtureK8s)
	rect := tui.Rect{X: 0, Y: 0, W: 60, H: 5}
	p.Render(st, rect)

	_, consumed := p.Update(st, tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	require.True(t, consumed)
	require.Equal(t, 0, p.cursor)
	require.Equal(t, st.Report.Assets[0], st.Sel.Asset)

	// The render that follows pulls the offset back to the cursor, so the check
	// is on the offset the pane holds rather than on what it draws next.
	require.Positive(t, p.scroll.Off)

	_, consumed = p.Update(st, tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	require.True(t, consumed)
	require.Equal(t, 0, p.scroll.Off)

	_, consumed = p.Update(st, tea.MouseMsg{Button: tea.MouseButtonMiddle})
	require.False(t, consumed, "the tree has no use for the other buttons")
}

// --- plumbing ---------------------------------------------------------------

// The pane installs itself in the tree slot, and does it without a State: the
// frame builds panes before it has anything to draw.
func TestTreeRegistersItself(t *testing.T) {
	require.IsType(t, &treePane{}, buildPane(PaneTree, nil))

	p := newTree()
	require.Equal(t, PaneTree, p.ID())
	require.True(t, p.Focusable())
	require.Nil(t, p.Claims(), "the tree needs no key while another pane has focus")
	require.NotEmpty(t, p.Hints(nil))
}

// A key the tree does not bind falls through to the frame, which is how "?" and
// "q" keep working while the tree has focus.
func TestUnboundKeysFallThrough(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	for _, k := range []string{"?", "q", "esc", "tab", "/"} {
		_, consumed := p.Update(st, treeKey(k))
		require.False(t, consumed, "%q belongs to the frame", k)
	}
	_, consumed := p.Update(st, struct{ custom string }{"hello"})
	require.False(t, consumed)
}

// The title names what is on screen, and is short enough for the border it is
// inlaid into: tui.PanelTop drops a title it cannot fit rather than cutting it.
func TestTitleNamesWhatIsOnScreen(t *testing.T) {
	p, st := treeFor(t, fixtureUbuntu)
	require.Equal(t, "ubuntu:24.04", p.Render(st, treeRect).Title)

	k8s, kst := treeFor(t, fixtureK8s)
	res := k8s.Render(kst, treeRect)
	require.Equal(t, "Assets", res.Title)
	require.Equal(t, "15", res.Status, "everything fits, so the count is the whole story")
	require.Equal(t, "1/15", k8s.Render(kst, tui.Rect{X: 0, Y: 0, W: 41, H: 5}).Status,
		"and when it does not, where the cursor is in it")

	for _, w := range []int{6, 10, 41} {
		title := p.Render(st, tui.Rect{X: 0, Y: 0, W: w, H: 4}).Title
		require.LessOrEqual(t, tui.Width(title), w)
	}
}

// Page up and page down move by what the last render could show, so a taller
// terminal pages further. Home and end go to the ends of the list.
func TestPagingMovesByAScreen(t *testing.T) {
	p, st := treeFor(t, fixtureK8s)
	p.Render(st, tui.Rect{X: 0, Y: 0, W: 60, H: 4})

	send(t, p, st, "pgdown")
	require.Equal(t, 4, p.cursor)
	send(t, p, st, "pgdown")
	require.Equal(t, 8, p.cursor)
	send(t, p, st, "pgup")
	require.Equal(t, 4, p.cursor)

	send(t, p, st, "end")
	require.Equal(t, 14, p.cursor)
	send(t, p, st, "pgdown")
	require.Equal(t, 14, p.cursor, "the end of the list is the end of the list")
	send(t, p, st, "home")
	require.Equal(t, 0, p.cursor)
	send(t, p, st, "pgup")
	require.Equal(t, 0, p.cursor)
}
