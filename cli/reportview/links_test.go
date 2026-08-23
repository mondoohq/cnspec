// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/tui"
)

// refURL is the URL both reference-carrying checks of the ubuntu fixture cite.
// It is 88 cells wide, which is wider than either pane on an 80-column terminal
// -- i.e. exactly the case truncation has to survive.
const refURL = "http://www.itsecure.hu/library/image/CIS_Red_Hat_Enterprise_Linux_7_Benchmark_v2.1.1.pdf"

// osc8 is the two halves of the sequence termenv.Hyperlink emits.
const (
	osc8Open  = "\x1b]8;;"
	osc8Close = "\x1b]8;;\x1b\\"
)

// withOpener swaps the platform opener for one that records what it was asked to
// open. The real one would launch a browser on the machine running the tests and
// would fail on a build box with no xdg-open, for a reason that has nothing to
// do with this viewer.
func withOpener(t *testing.T, err error) *[]string {
	t.Helper()
	var got []string
	was := urlOpener
	urlOpener = func(raw string) error {
		got = append(got, raw)
		return err
	}
	t.Cleanup(func() { urlOpener = was })
	return &got
}

// --- the sequence -----------------------------------------------------------

// A hyperlink is zero width. This is the property that lets a link go into a
// pane measured in exact cells at all: the wrapped URL takes the same number of
// columns as the bare one, so a row that fitted before still fits.
func TestHyperlinkIsZeroWidth(t *testing.T) {
	link := hyperlink(refURL, refURL)

	require.Equal(t, 88, ansi.StringWidth(refURL))
	require.Equal(t, ansi.StringWidth(refURL), ansi.StringWidth(link),
		"the sequence took cells: %q", link)
	require.Equal(t, osc8Open+refURL+"\x1b\\"+refURL+osc8Close, link)
}

// And a truncated hyperlink is still a hyperlink. ansi.Truncate cuts the visible
// text and keeps the sequences either side of it, so a URL wider than the pane
// comes out shortened and clickable rather than as an escape sequence leaking
// into the row below.
func TestATruncatedLinkIsStillALink(t *testing.T) {
	link := hyperlink(refURL, refURL)

	for _, w := range []int{1, 5, 20, 30, 60, 87} {
		cut := ansi.Truncate(link, w, "…")
		require.Equal(t, w, ansi.StringWidth(cut), "w=%d: %q", w, cut)
		require.True(t, strings.HasPrefix(cut, osc8Open+refURL+"\x1b\\"),
			"w=%d lost the target, so it is no longer a link: %q", w, cut)
		require.True(t, strings.HasSuffix(cut, osc8Close),
			"w=%d lost the terminator, so the escape leaks: %q", w, cut)
		// The target is intact even though the text is not: what is clicked is
		// the whole URL, not the visible stub.
		require.Contains(t, cut, refURL)
		require.True(t, strings.HasSuffix(ansi.Strip(cut), "…"),
			"w=%d did not actually cut: %q", w, ansi.Strip(cut))
		require.True(t, strings.HasPrefix(refURL, strings.TrimSuffix(ansi.Strip(cut), "…")),
			"w=%d shows something other than the head of the URL: %q", w, ansi.Strip(cut))
	}
}

// --- what gets a link -------------------------------------------------------

func TestOnlyWebAddressesAreLinked(t *testing.T) {
	for _, ok := range []string{
		"http://example.com",
		"https://mondoo.com/docs/cnspec",
		"https://example.com/a?b=c#d",
	} {
		require.True(t, linkable(ok), ok)
	}
	for _, no := range []string{
		"",
		"CIS RHEL 7",
		"example.com",
		"/etc/ssh/sshd_config",
		"mailto:security@example.com",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"http://",
		"https://",
		"-nasty",
	} {
		require.False(t, linkable(no), no)
	}
}

// The viewer never offers a link it would then refuse: the same predicate gates
// the drawing and the opening.
func TestARefusedSchemeIsNeverDrawnAsALink(t *testing.T) {
	b := newDetailBuf(80)
	b.reference("file:///etc/passwd")

	require.Empty(t, b.links, "nothing to click")
	require.Empty(t, b.urls)
	require.Len(t, b.out, 1)
	require.Contains(t, b.out[0], "file:///etc/passwd", "it is still shown, as text")
	require.NotContains(t, b.out[0], osc8Open)

	got := withOpener(t, nil)
	require.Error(t, openURL("file:///etc/passwd"))
	require.Empty(t, *got, "the opener was never reached")
}

// --- in the pane ------------------------------------------------------------

// The REFERENCES section of a real check: the URL is one row, it is wrapped in
// OSC 8, and the pane publishes a zone over exactly the cells it drew.
func TestAReferenceIsDrawnAsALinkWithAZone(t *testing.T) {
	st, check := stateFor(t, fixtureUbuntu, "Only use strong Ciphers")
	require.Len(t, check.Detail().References, 1)

	p, lines := renderDetail(st, 100)
	require.Len(t, p.urls, 1)
	require.Equal(t, refURL, p.urls[0])
	require.Len(t, p.links, 1)

	link := p.links[0]
	row := lines[link.Row]
	require.Contains(t, row, osc8Open+refURL+"\x1b\\", "the row is not a link")
	require.True(t, strings.HasSuffix(row, osc8Close))
	require.Equal(t, refURL, ansi.Strip(row)[link.Col:], "the zone does not cover the URL")
	require.Equal(t, detailIndent, link.Col)
	require.Equal(t, tui.Width(refURL), link.W)

	// It sits under the section label, where the plain text used to.
	require.Equal(t, "REFERENCES", ansi.Strip(lines[link.Row-2]))
}

// A pane too narrow for the URL draws a shortened link, and the zone shrinks
// with it: a click has to land on cells the reader can see.
func TestANarrowPaneKeepsTheLinkWhole(t *testing.T) {
	st, _ := stateFor(t, fixtureUbuntu, "Only use strong Ciphers")

	for _, w := range []int{20, 37, 40, 60, 80, 100, 200} {
		p, lines := renderDetail(st, w)
		require.Len(t, p.links, 1, "w=%d", w)
		link := p.links[0]
		row := lines[link.Row]

		require.LessOrEqual(t, tui.Width(row), w, "w=%d: %q", w, ansi.Strip(row))
		require.NotContains(t, ansi.Strip(row), "\x1b", "w=%d leaked an escape", w)
		require.Contains(t, row, refURL, "w=%d lost the target", w)
		require.True(t, strings.HasSuffix(row, osc8Close), "w=%d lost the terminator", w)
		require.Equal(t, link.W, tui.Width(ansi.Strip(row))-link.Col, "w=%d: the zone and the text disagree", w)
	}
}

// The whole pane invariant, restated over the section that now emits escape
// sequences of its own: one element of Lines is one terminal row, never wider
// than the rect, at every width and for every check of the fixture.
func TestLinkedRowsStillFitTheRect(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	asset := st.Report.Assets[0]

	for _, check := range asset.Checks {
		st.SelectCheck(asset, nil, check)
		for _, w := range []int{1, 2, 3, 4, 8, 20, 37, 60, 80, 120, 200} {
			p := &detailPane{}
			p.Render(st, detailRect(w, 24))
			for i, ln := range p.lines {
				require.NotContains(t, ln, "\n", "%s w=%d line %d", check.Title, w, i)
				require.LessOrEqual(t, tui.Width(ln), w,
					"%s w=%d line %d: %q", check.Title, w, i, ansi.Strip(ln))
			}
		}
	}
}

// A link zone is only published while its row is on screen, exactly like a COPY
// button's. A zone for a row that scrolled away is a click that lands on
// whatever is there now.
func TestLinkZonesFollowTheViewport(t *testing.T) {
	st, _ := stateFor(t, fixtureUbuntu, "Only use strong Ciphers")

	p := &detailPane{}
	rect := detailRect(100, 8)
	p.Render(st, rect)
	require.Len(t, p.links, 1)
	row := p.links[0].Row

	require.Empty(t, linkZonesOf(p.Render(st, rect)), "the references are far below the fold")

	// Scrolling to the end brings the references on screen; the offset the pane
	// settles on is what the zone's row is measured against.
	p.scroll.Off = len(p.lines)
	zones := linkZonesOf(p.Render(st, rect))
	require.Len(t, zones, 1)
	require.Equal(t, rect.Y+row-p.scroll.Off, zones[0].Rect.Y, "the zone is not on the row it drew")
	require.Equal(t, rect.X+detailIndent, zones[0].Rect.X)
	require.Equal(t, tui.Width(refURL), zones[0].Rect.W)
}

// --- clicking ---------------------------------------------------------------

// The click the terminal never sees. Mouse tracking is on, so a plain click on
// an OSC 8 link is delivered to the application; the pane turns it into the
// command that opens the URL.
func TestClickingALinkOpensIt(t *testing.T) {
	got := withOpener(t, nil)
	st, _ := stateFor(t, fixtureUbuntu, "Only use strong Ciphers")

	p := &detailPane{}
	rect := detailRect(100, 8)
	p.Render(st, rect)
	// Scroll to the end, which is where the references are.
	p.scroll.Off = len(p.lines)
	zones := linkZonesOf(p.Render(st, rect))
	require.Len(t, zones, 1)
	zone := zones[0]

	cmd, handled := p.Update(st, ClickMsg{
		Zone:  zone,
		Mouse: tea.MouseMsg{X: zone.Rect.X, Y: zone.Rect.Y, Action: tea.MouseActionPress},
	})
	require.True(t, handled)
	require.NotNil(t, cmd)

	msg, ok := cmd().(openDoneMsg)
	require.True(t, ok)
	require.NoError(t, msg.Err)
	require.Equal(t, []string{refURL}, *got, "the whole URL is opened, not the visible stub")
	require.Equal(t, "opened "+refURL, openNotice(msg))
}

// Even when the pane drew a stub. What is opened is the URL, not what fitted.
func TestClickingATruncatedLinkOpensTheWholeURL(t *testing.T) {
	got := withOpener(t, nil)
	st, _ := stateFor(t, fixtureUbuntu, "Only use strong Ciphers")

	p := &detailPane{}
	rect := tui.Rect{X: 1, Y: 2, W: 30, H: 8}
	p.Render(st, rect)
	p.scroll.Off = len(p.lines)
	zones := linkZonesOf(p.Render(st, rect))
	require.Len(t, zones, 1)
	zone := zones[0]
	require.Less(t, zone.Rect.W, tui.Width(refURL), "this pane is too narrow for the URL, or the test proves nothing")

	cmd, handled := p.Update(st, ClickMsg{Zone: zone})
	require.True(t, handled)
	cmd()
	require.Equal(t, []string{refURL}, *got)
}

// A click that appears to do nothing is the complaint this package keeps
// getting, so a failure is a sentence in the footer and not a swallowed error.
func TestAFailedOpenIsSaidOutLoud(t *testing.T) {
	withOpener(t, errors.New("exec: \"xdg-open\": executable file not found in $PATH"))

	msg, ok := openURLCmd(refURL)().(openDoneMsg)
	require.True(t, ok)
	require.Error(t, msg.Err)

	notice := openNotice(msg)
	require.Contains(t, notice, "could not open")
	require.Contains(t, notice, refURL)
	require.Contains(t, notice, "xdg-open")
	require.NotContains(t, notice, "\n", "a notice is one footer row")

	// And it reaches the footer.
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	next, _ := m.Update(msg)
	require.Equal(t, notice, next.(Model).state.Notice)
}

// The reason is the opener's own words, so a missing helper and a helper that
// declined are told apart by the person who can fix either.
func TestTheOpenerNamesWhatRefused(t *testing.T) {
	require.Equal(t, "open", first(openerFor("darwin", refURL)))
	require.Equal(t, []string{refURL}, second(openerFor("darwin", refURL)))

	require.Equal(t, "rundll32", first(openerFor("windows", refURL)))
	require.Equal(t, []string{"url.dll,FileProtocolHandler", refURL}, second(openerFor("windows", refURL)))

	require.Equal(t, "xdg-open", first(openerFor("linux", refURL)))
	require.Equal(t, []string{refURL}, second(openerFor("freebsd", refURL)))
}

// A click on a zone this pane does not own, or one carrying an index that is not
// a URL, is not a click it handles. Neither can happen through the frame; both
// would be a crash if the bounds were trusted.
func TestABadLinkZoneIsIgnored(t *testing.T) {
	got := withOpener(t, nil)
	st, _ := stateFor(t, fixtureUbuntu, "Only use strong Ciphers")
	p, _ := renderDetail(st, 100)

	for _, zone := range []Zone{
		{Tag: linkZoneTag, Idx: -1},
		{Tag: linkZoneTag, Idx: len(p.urls)},
		{Tag: "twisty", Idx: 0},
	} {
		cmd, handled := p.Update(st, ClickMsg{Zone: zone})
		require.False(t, handled, "%+v", zone)
		require.Nil(t, cmd)
	}
	require.Empty(t, *got)
}

// A check with no references publishes no link zones, and its COPY buttons are
// untouched by any of this.
func TestACheckWithoutReferencesHasNoLinks(t *testing.T) {
	st, check := stateFor(t, fixtureUbuntu, "Ensure X Window System is not installed")
	require.Empty(t, check.Detail().References)

	p := &detailPane{}
	res := p.Render(st, detailRect(100, 40))
	require.Empty(t, p.links)
	require.Empty(t, p.urls)
	require.Empty(t, linkZonesOf(res))
	require.NotEmpty(t, p.buttons, "the code blocks are still there")
	for _, z := range res.Zones {
		require.Equal(t, copyZoneTag, z.Tag)
	}
}

func linkZonesOf(r Render) []Zone {
	var res []Zone
	for _, z := range r.Zones {
		if z.Tag == linkZoneTag {
			res = append(res, z)
		}
	}
	return res
}

func first(name string, _ []string) string    { return name }
func second(_ string, args []string) []string { return args }
