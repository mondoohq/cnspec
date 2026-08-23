// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
)

// detailRect is the content area a test renders into: absolute cells, the way
// the frame hands them over.
func detailRect(w, h int) tui.Rect {
	return tui.Rect{X: 1, Y: 2, W: w, H: h}
}

// renderDetail runs one frame of a fresh pane and returns the whole body, not
// just the visible window, so a test can assert on sections that scrolled off.
func renderDetail(st *State, w int) (*detailPane, []string) {
	p := &detailPane{}
	p.Render(st, detailRect(w, 1000))
	return p, p.lines
}

// stateFor builds a state selecting the named check of the first asset.
func stateFor(t *testing.T, fixture, title string) (*State, *reportmodel.Check) {
	t.Helper()
	st := NewState(loadReport(t, fixture))
	require.NotEmpty(t, st.Report.Assets)
	asset := st.Report.Assets[0]
	for _, c := range asset.Checks {
		if c.Title == title {
			st.SelectCheck(asset, nil, c)
			return st, c
		}
	}
	t.Fatalf("no check titled %q on %s", title, asset.Name)
	return nil, nil
}

// --- the invariant ----------------------------------------------------------

// One element of Render.Lines is exactly one terminal row, and never wider than
// the rect. This is the invariant the whole frame is built on: a line wider than
// the pane wraps in the terminal and pushes every row below it down by one, and
// a line containing a newline does the same thing without even being wide.
//
// It is asserted over every check of the fixture at every width that has ever
// broken something, including widths narrower than the indent itself.
func TestDetailLinesFitTheRect(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	asset := st.Report.Assets[0]
	require.NotEmpty(t, asset.Checks)

	widths := []int{1, 2, 3, 4, 8, 20, 37, 60, 80, 120, 200}

	for _, check := range asset.Checks {
		st.SelectCheck(asset, nil, check)
		for _, w := range widths {
			p := &detailPane{}
			r := p.Render(st, detailRect(w, 24))
			require.LessOrEqual(t, len(r.Lines), 24,
				"%s at w=%d: rendered more rows than the rect is tall", check.Title, w)
			for i, ln := range p.lines {
				require.NotContains(t, ln, "\n",
					"%s at w=%d: line %d holds a newline, so it is not one row", check.Title, w, i)
				require.LessOrEqual(t, tui.Width(ln), w,
					"%s at w=%d: line %d is %d cells wide (%q)", check.Title, w, i, tui.Width(ln), ansi.Strip(ln))
			}
		}
	}
}

// The width evidence for the one section that is arbitrarily long: a 900-byte
// remediation body full of shell, at a pane too narrow to hold one of its lines.
// Wrapping is the frame's Wrap, so a token longer than the pane is broken rather
// than allowed to overflow.
func TestLongRemediationWrapsToNarrowPane(t *testing.T) {
	st, check := stateFor(t, fixtureUbuntu, "Ensure X Window System is not installed")

	d := check.Detail()
	require.NotEmpty(t, d.Remediation, "this fixture check is the long-remediation case")
	longest := 0
	for _, item := range d.Remediation {
		for _, ln := range strings.Split(item.Desc, "\n") {
			longest = max(longest, len(ln))
		}
	}
	require.Greater(t, longest, 60, "the remediation must be wider than the panes below")

	for _, w := range []int{24, 30, 40} {
		_, lines := renderDetail(st, w)
		for i, ln := range lines {
			require.LessOrEqual(t, tui.Width(ln), w, "w=%d line %d: %q", w, i, ansi.Strip(ln))
		}
		joined := ansi.Strip(strings.Join(lines, "\n"))
		require.Contains(t, joined, "REMEDIATION")
		// The body survived the fold: a word from deep inside the script is
		// still there, so wrapping did not truncate the section away.
		require.Contains(t, strings.ReplaceAll(joined, "\n", ""), "package manager")
	}
}

// --- markdown ---------------------------------------------------------------

// colorful pins the color profile the markdown renderer draws for. lipgloss
// detects Ascii whenever stdout is not a terminal, which is every test run, and
// a test that wants to see a bold sequence has to ask for a profile that has
// one.
func colorful(t *testing.T) {
	t.Helper()
	was := markdownProfile
	markdownProfile = func() termenv.Profile { return termenv.ANSI256 }
	t.Cleanup(func() { markdownProfile = was })
}

// The three markdown sections are rendered, not printed as their source. This is
// the whole point of the file: the reader sees bold words, headings, list
// markers and a code block, and none of the markup that produced them.
func TestMarkdownIsRenderedNotPrinted(t *testing.T) {
	colorful(t)
	st, check := stateFor(t, fixtureUbuntu, "Ensure X Window System is not installed")

	d := check.Detail()
	require.Contains(t, d.Description, "**", "the source is markdown, or this test proves nothing")
	require.Contains(t, d.Remediation[0].Desc, "```bash")
	require.Contains(t, d.Remediation[0].Desc, "### ")

	_, lines := renderDetail(st, 80)
	out := ansi.Strip(strings.Join(lines, "\n"))

	// The markup is gone.
	require.NotContains(t, out, "**")
	require.NotContains(t, out, "```")
	for _, ln := range lines {
		// An ATX heading is hashes then a space. A shebang inside a code block
		// is neither, and it stays exactly as it was written.
		require.NotRegexp(t, `^#+ `, strings.TrimSpace(ansi.Strip(ln)),
			"a heading is still wearing its hashes: %q", ansi.Strip(ln))
	}
	require.Contains(t, out, "#!/bin/bash", "a shebang is code, not a heading")

	// The words it marked up are not.
	require.Contains(t, out, "Why this matters")
	require.Contains(t, out, "RHEL/Fedora/Amazon Linux and derivatives")
	require.Contains(t, out, "yum remove xorg-x11*")
	require.Contains(t, out, "•", "a bullet list is drawn with bullets")

	// And the emphasis became an escape sequence rather than two asterisks.
	require.Contains(t, strings.Join(lines, "\n"), "\x1b[1m", "nothing was rendered bold")
}

// The section that is code stays code. MQL goes nowhere near the renderer: a
// markdown parser would eat its underscores and asterisks and fold its lines,
// and the query the check ran is not a document.
func TestMqlIsNotRenderedAsMarkdown(t *testing.T) {
	colorful(t)
	st, check := stateFor(t, fixtureUbuntu, "Ensure X Window System is not installed")
	d := check.Detail()
	require.Contains(t, d.Mql, "*", "this query carries a character markdown would eat")

	_, lines := renderDetail(st, 200)
	start := indexOfLine(lines, "QUERY")
	require.GreaterOrEqual(t, start, 0)

	rows := make([]string, 0, len(d.Mql))
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(ansi.Strip(lines[i])) == "" {
			break
		}
		rows = append(rows, strings.TrimSpace(ansi.Strip(lines[i])))
	}
	require.Equal(t, tui.Lines(d.Mql), rows, "the query is not the source it ran")
}

// The invariant again, aimed at the row most likely to break it: a fenced code
// block whose lines are far wider than the pane. glamour does not wrap fenced
// code the way it wraps prose -- at a narrow enough width it stops wrapping
// altogether -- so the renderer measures and folds every row itself.
func TestMarkdownCodeBlockNeverOverflows(t *testing.T) {
	colorful(t)
	src := "Run it:\n\n```bash\n" +
		"sudo launchctl list | grep com.apple.smbd && echo 'the SMB service is loaded, which it should not be' >&2\n" +
		"nmcli connection modify eth0 ipv4.dns \"1.1.1.1 8.8.8.8\" ipv4.ignore-auto-dns yes\n" +
		"```\n\nA passing result returns **no output**.\n"

	for w := 1; w <= 60; w++ {
		lines := markdown.Lines(src, w)
		require.NotEmpty(t, lines, "w=%d: rendered nothing", w)
		for i, ln := range lines {
			require.NotContains(t, ln, "\n", "w=%d row %d holds a newline", w, i)
			require.NotContains(t, ln, "\r", "w=%d row %d holds a carriage return", w, i)
			require.LessOrEqual(t, tui.Width(ln), w,
				"w=%d row %d is %d cells (%q)", w, i, tui.Width(ln), ansi.Strip(ln))
		}
		// Folding is not truncation: the command survives being cut up.
		flat := strings.ReplaceAll(ansi.Strip(strings.Join(lines, "")), " ", "")
		require.Contains(t, flat, "grepcom.apple.smbd", "w=%d: the command lost its middle", w)
	}
}

// Carriage returns are what break lipgloss's width maths, and the source of
// these fields comes from a reporter whose newline is "\r\n" on Windows. Nothing
// the renderer emits may carry one.
func TestMarkdownNormalizesWindowsNewlines(t *testing.T) {
	colorful(t)
	crlf := "**Bold**\r\n\r\n1. First step.\r\n2. Second step.\r\n\r\n```bash\r\nchmod 600 /etc/group-\r\n```\r\n"

	lines := markdown.Lines(crlf, 40)
	require.NotEmpty(t, lines)
	for i, ln := range lines {
		require.NotContains(t, ln, "\r", "row %d kept a carriage return", i)
		require.LessOrEqual(t, tui.Width(ln), 40)
	}
	require.Contains(t, ansi.Strip(strings.Join(lines, "\n")), "chmod 600 /etc/group-")
}

// A source that renders to nothing at all -- an HTML comment, say -- falls back
// to the plain wrapped source. A section the check actually filled in must never
// draw as an empty label.
func TestMarkdownFallsBackRatherThanRenderNothing(t *testing.T) {
	colorful(t)
	const hidden = "<!-- there is nothing to see here -->"
	require.Empty(t, markdown.Lines(hidden, 40), "this source is the renders-to-nothing case")

	b := &detailBuf{w: 40}
	b.section("DESCRIPTION")
	b.markdown(hidden)
	require.Len(t, b.out, 2)
	require.Contains(t, ansi.Strip(b.out[1]), "nothing to see here")
}

// Rendering markdown is not free, and Render runs on every frame and every
// mouse event. It has to happen inside the body cache: once per selection and
// width, not once per wheel notch.
func TestMarkdownRunsInsideTheBodyCache(t *testing.T) {
	colorful(t)
	st, _ := stateFor(t, fixtureUbuntu, "Ensure X Window System is not installed")

	p := &detailPane{}
	before := markdown.Renders()
	p.Render(st, detailRect(80, 20))
	first := markdown.Renders() - before
	require.Greater(t, first, 1, "this check has a description and three remediations")

	for i := 0; i < 20; i++ {
		p.Render(st, detailRect(80, 20))
		p.Update(st, tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	}
	require.Equal(t, first, markdown.Renders()-before, "the cache did not hold")

	// A resize is a new body, and so is a new selection.
	p.Render(st, detailRect(60, 20))
	require.Equal(t, 2*first, markdown.Renders()-before)
}

// --- an asset that never scanned --------------------------------------------

// The k8s fixture is fifteen assets, no reports, fifteen errors and no bundle.
// Selecting one of them must say the scan did not happen and show the error. It
// must not render an empty check page, which is indistinguishable from a clean
// asset.
func TestUnscannedAssetSaysWhy(t *testing.T) {
	st := NewState(loadReport(t, fixtureK8s))
	require.Len(t, st.Report.Assets, 15)

	for _, asset := range st.Report.Assets {
		require.False(t, asset.Scanned())
		require.NotEmpty(t, asset.ScanError)
		require.Empty(t, asset.Checks)

		st.SelectAsset(asset)
		_, lines := renderDetail(st, 80)
		out := ansi.Strip(strings.Join(lines, "\n"))

		require.Equal(t, "Asset", detailTitle(st))
		require.Contains(t, out, "THIS ASSET WAS NOT SCANNED")
		require.Contains(t, out, statusGlyph(reportmodel.StatusError)+" "+string(reportmodel.StatusError))
		// The reason, wrapped, is the point of the page.
		require.Contains(t, strings.ReplaceAll(out, "\n", " "),
			strings.Fields(asset.ScanError)[0])
		// None of the vocabulary of a scanned asset: no tally that reads as
		// "we looked and found nothing", no findings list.
		require.NotContains(t, out, "FINDINGS")
		require.NotContains(t, out, "0 total")
		require.NotContains(t, out, "CHECKS")
	}
}

// An asset that scanned gets the other page entirely: its tally and its
// findings, and nothing about a scan that did not happen.
func TestScannedAssetPage(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	asset := st.Report.Assets[0]
	require.True(t, asset.Scanned())
	st.SelectAsset(asset)

	_, lines := renderDetail(st, 100)
	out := ansi.Strip(strings.Join(lines, "\n"))

	require.Contains(t, out, "ubuntu:24.04")
	require.Contains(t, out, "Ubuntu 24.04.2 LTS")
	require.Contains(t, out, "CHECKS")
	require.Contains(t, out, "24 total")
	require.Contains(t, out, "FINDINGS")
	require.NotContains(t, out, "THIS ASSET WAS NOT SCANNED")

	// Every finding, and only the findings: two failed plus four errored, one
	// row each, and nothing that passed.
	require.Equal(t, 6, asset.Counts.Findings())
	at := indexOfLine(lines, "FINDINGS")
	require.GreaterOrEqual(t, at, 0)
	rows := lines[at+1:]
	require.Len(t, rows, 6)
	for _, ln := range rows {
		require.NotContains(t, ansi.Strip(ln), string(reportmodel.StatusPass))
	}

	// And they are drawn the way the tree draws the same checks: a glyph, the
	// severity dots, then the title. This is a list being scanned, so it takes
	// the tree's trade rather than the status row's -- spelling PASS/FAIL and
	// CRIT out here would re-say the shape and cost the title twelve cells.
	failed, errored := 0, 0
	for _, ln := range rows {
		s := ansi.Strip(ln)
		require.NotContains(t, s, string(reportmodel.StatusFail))
		require.NotContains(t, s, "CRIT")
		switch {
		case strings.Contains(s, "✗ ●●●● "):
			failed++
		// The four errored checks configure no impact, so their dot column is
		// four spaces rather than four hollow dots -- the same distinction the
		// tree makes, and for the same reason.
		case strings.Contains(s, "! "+strings.Repeat(" ", SeverityDotsWidth)+" "):
			errored++
		}
	}
	require.Equal(t, 2, failed, "%q", rows)
	require.Equal(t, 4, errored, "%q", rows)
}

// --- an errored check is not a failed check ---------------------------------

// The four errored checks of the ubuntu fixture could not run at all. The page
// has to say ERROR, carry the reason, and put it above the assessment, which for
// an errored check reads "[failed] ..." only because the query never completed.
func TestErroredCheckLeadsWithTheError(t *testing.T) {
	st, check := stateFor(t, fixtureUbuntu, "Enable strict mode")
	d := check.Detail()
	require.Equal(t, reportmodel.StatusError, d.Status)
	require.NotEmpty(t, d.Error)
	require.NotEmpty(t, d.Assessment)

	_, lines := renderDetail(st, 100)
	out := ansi.Strip(strings.Join(lines, "\n"))

	require.Contains(t, out, string(reportmodel.StatusError))
	require.Contains(t, out, "sshd_config' not found")
	// Not a single FAIL anywhere on the page. The assessment says "[failed]"
	// in lower case because the query never completed, and that lower-case
	// word is the whole reason the error is hoisted above it.
	require.NotContains(t, out, string(reportmodel.StatusFail),
		"an errored check must not be labelled as a failed one")
	require.Contains(t, out, "[failed]")

	errAt := indexOfLine(lines, "ERROR")
	resultAt := indexOfLine(lines, "RESULT")
	require.GreaterOrEqual(t, errAt, 0)
	require.GreaterOrEqual(t, resultAt, 0)
	require.Less(t, errAt, resultAt, "the reason a check could not run comes first")
	// The reason is under its own label, not folded into the status row.
	require.Contains(t, ansi.Strip(lines[errAt+1]), "1 error occurred")
}

// A check that failed carries no ERROR section at all, and its status row says
// FAIL. The two outcomes stay two outcomes.
func TestFailedCheckHasNoErrorSection(t *testing.T) {
	st, check := stateFor(t, fixtureUbuntu, "Ensure secure permissions on /etc/group- are set")
	d := check.Detail()
	require.Equal(t, reportmodel.StatusFail, d.Status)
	require.Empty(t, d.Error)

	_, lines := renderDetail(st, 100)
	require.Equal(t, -1, indexOfLine(lines, "ERROR"))

	out := ansi.Strip(strings.Join(lines, "\n"))
	require.Contains(t, out, string(reportmodel.StatusFail))
	require.Contains(t, out, "CRIT", "severity is what the check is worth")
	require.Contains(t, out, "impact 100")
}

// --- section order ----------------------------------------------------------

// The stack follows junit's detailedCheckBody, with the error hoisted. A check
// with every optional section present pins the order.
func TestSectionOrder(t *testing.T) {
	st, check := stateFor(t, fixtureUbuntu, "Only use strong Ciphers")
	d := check.Detail()
	require.NotEmpty(t, d.Description)
	require.NotEmpty(t, d.Mql)
	require.NotEmpty(t, d.Audit)
	require.NotEmpty(t, d.Assessment)
	require.NotEmpty(t, d.Error)
	require.NotEmpty(t, d.Remediation)
	require.NotEmpty(t, d.References)
	require.NotEmpty(t, d.Policies)

	_, lines := renderDetail(st, 100)

	want := []string{
		"ERROR", "DESCRIPTION", "QUERY", "AUDIT", "RESULT",
		"REMEDIATION", "REFERENCES", "POLICIES",
	}
	at := -1
	for _, label := range want {
		i := indexOfLine(lines, label)
		require.Greater(t, i, at, "%s is out of order", label)
		at = i
	}

	out := ansi.Strip(strings.Join(lines, "\n"))
	require.Contains(t, out, "Mondoo SSH Server Security")
	require.Contains(t, out, "CIS RHEL 7")
	require.Contains(t, out, "itsecure.hu")
}

// MQL is code: every source line stays one row. The frame's Wrap would fold a
// long query onto a second line and move an operator onto a line of its own.
func TestMqlKeepsItsLineStructure(t *testing.T) {
	st, check := stateFor(t, fixtureUbuntu, "Ensure secure permissions on /etc/group- are set")
	d := check.Detail()
	sourceLines := strings.Count(d.Mql, "\n") + 1
	require.Greater(t, len(d.Mql), 100, "this check's query is the multi-line case")

	// Wide enough that nothing is cut, and narrow enough that Wrap certainly
	// would have folded: the row count is the same either way. The block is
	// measured label to label, so a blank line inside the query is still one of
	// its rows.
	for _, w := range []int{40, 400} {
		_, lines := renderDetail(st, w)
		start := indexOfLine(lines, "QUERY")
		next := indexOfLine(lines, "REMEDIATION")
		require.GreaterOrEqual(t, start, 0)
		require.Greater(t, next, start)

		// start+1 .. next-2, because section() puts a blank before each label.
		rows := next - start - 2
		require.Equal(t, sourceLines, rows, "w=%d: the query gained or lost rows", w)
	}
}

// --- a check with nothing in it ---------------------------------------------

// Every field of CheckDetail may be empty. The pane still has to render
// something a person can read, rather than zero lines or a blank box.
func TestEmptyCheckStillRenders(t *testing.T) {
	st := NewState(reportmodel.New(nil))
	st.Select(Selection{Check: &reportmodel.Check{}})

	for _, w := range []int{1, 3, 40} {
		_, lines := renderDetail(st, w)
		require.NotEmpty(t, lines, "w=%d: an empty check rendered nothing at all", w)
		for _, ln := range lines {
			require.LessOrEqual(t, tui.Width(ln), w)
			require.NotContains(t, ln, "\n")
		}
		require.Equal(t, "Check", detailTitle(st))
	}

	// The status row is the floor. A zero-value Check has no status either, and
	// a row of blank cells is not an outcome, so it reports UNKNOWN rather than
	// nothing -- and it is the only thing on the page.
	full := ansi.Strip(strings.Join(detailBody(st, 60), "\n"))
	require.Contains(t, full, string(reportmodel.StatusUnknown))
	for _, label := range []string{"DESCRIPTION", "QUERY", "AUDIT", "RESULT", "ERROR",
		"FAILING RESOURCES", "REMEDIATION", "REFERENCES", "COMPLIANCE", "POLICIES"} {
		require.NotContains(t, full, label, "an empty check must not grow an empty %s section", label)
	}
}

// Nothing selected at all is the third page, and it is one honest line rather
// than an empty pane.
func TestNothingSelected(t *testing.T) {
	st := NewState(reportmodel.New(nil))
	require.Nil(t, st.Sel.Asset)

	_, lines := renderDetail(st, 40)
	require.Len(t, lines, 1)
	require.Equal(t, "nothing selected", ansi.Strip(lines[0]))
	require.Equal(t, "Detail", detailTitle(st))
}

// --- scrolling --------------------------------------------------------------

// The pane returns exactly the rows it wants visible, and scrolling moves the
// window rather than the content.
func TestScrollWindowsTheBody(t *testing.T) {
	st, _ := stateFor(t, fixtureUbuntu, "Ensure X Window System is not installed")

	p := &detailPane{}
	const h = 10
	first := p.Render(st, detailRect(80, h))
	require.Len(t, first.Lines, h)
	require.Greater(t, len(p.lines), h, "this check is longer than the pane")

	_, ok := p.Update(st, key("down"))
	require.True(t, ok)
	second := p.Render(st, detailRect(80, h))
	require.Len(t, second.Lines, h)
	require.Equal(t, first.Lines[1], second.Lines[0], "one row down is one row down")

	// A page moves a pane's worth.
	p.scroll.Off = 0
	_, ok = p.Update(st, key("pgdown"))
	require.True(t, ok)
	require.Equal(t, h, p.scroll.Off)

	// The end clamps to the last full page and never past it.
	_, ok = p.Update(st, key("G"))
	require.True(t, ok)
	p.Render(st, detailRect(80, h))
	require.Equal(t, len(p.lines)-h, p.scroll.Off)

	_, ok = p.Update(st, key("g"))
	require.True(t, ok)
	require.Equal(t, 0, p.scroll.Off)

	// A key the pane has no use for falls through to the frame.
	_, ok = p.Update(st, key("x"))
	require.False(t, ok)
}

// A short body does not scroll at all: there is nothing below the fold.
func TestShortBodyDoesNotScroll(t *testing.T) {
	st := NewState(reportmodel.New(nil))
	p := &detailPane{}
	p.Render(st, detailRect(40, 20))

	_, ok := p.Update(st, key("down"))
	require.True(t, ok, "the pane still consumes the key")
	p.Render(st, detailRect(40, 20))
	require.Equal(t, 0, p.scroll.Off)
}

// A new selection starts at the top. The pane watches SelectionRev, because the
// tree is what bumps it and the pane has no other way to know.
func TestSelectionResetsScroll(t *testing.T) {
	st := NewState(loadReport(t, fixtureUbuntu))
	asset := st.Report.Assets[0]
	require.Greater(t, len(asset.Checks), 1)

	st.SelectCheck(asset, nil, asset.Checks[1])
	p := &detailPane{}
	p.Render(st, detailRect(80, 10))
	p.Update(st, key("pgdown"))
	p.Render(st, detailRect(80, 10))
	require.Greater(t, p.scroll.Off, 0)

	before := st.SelectionRev
	st.SelectCheck(asset, nil, asset.Checks[2])
	require.Greater(t, st.SelectionRev, before, "the selection actually changed")

	p.Render(st, detailRect(80, 10))
	require.Equal(t, 0, p.scroll.Off, "a new check opens at its top")
}

// A resize rebuilds the body at the new width but keeps the reader where they
// were: re-wrapping is not a reason to jump back to the title.
func TestResizeKeepsScroll(t *testing.T) {
	st, _ := stateFor(t, fixtureUbuntu, "Ensure X Window System is not installed")
	p := &detailPane{}
	p.Render(st, detailRect(80, 10))
	p.Update(st, key("pgdown"))
	p.Render(st, detailRect(80, 10))
	off, wide := p.scroll.Off, len(p.lines)
	require.Greater(t, off, 0)

	p.Render(st, detailRect(40, 10))
	require.Equal(t, 40, p.width)
	require.Greater(t, len(p.lines), wide, "a narrower pane needs more rows")
	require.Equal(t, off, p.scroll.Off)
}

// The wheel scrolls without moving focus, which is the frame's job; the pane
// only has to answer it.
func TestWheelScrolls(t *testing.T) {
	st, _ := stateFor(t, fixtureUbuntu, "Ensure X Window System is not installed")
	p := &detailPane{}
	p.Render(st, detailRect(80, 10))

	_, ok := p.Update(st, tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	require.True(t, ok)
	require.Equal(t, detailWheelStep, p.scroll.Off)

	_, ok = p.Update(st, tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	require.True(t, ok)
	require.Equal(t, 0, p.scroll.Off)

	_, ok = p.Update(st, tea.MouseMsg{Button: tea.MouseButtonLeft})
	require.False(t, ok, "a click is not this pane's business")
}

// --- the seam ---------------------------------------------------------------

// The pane installs itself into the detail slot and answers the interface the
// frame expects of it.
func TestDetailPaneIsRegistered(t *testing.T) {
	p := buildPane(PaneDetail, nil)
	require.IsType(t, &detailPane{}, p)
	require.Equal(t, PaneDetail, p.ID())
	require.True(t, p.Focusable())
	require.Nil(t, p.Claims())
	require.NotEmpty(t, p.Hints(nil))
}

// Rendering through the whole frame keeps the view exactly the terminal size,
// which is the frame's invariant and the pane's obligation to it.
func TestDetailInsideTheFrame(t *testing.T) {
	for _, fixture := range []string{fixtureUbuntu, fixtureK8s} {
		report := loadReport(t, fixture)
		for _, s := range termSizes {
			m := sized(NewModel(report), s.w, s.h)
			m.state.Focus = PaneDetail
			lines := viewLines(m)
			require.Len(t, lines, s.h, "%s %dx%d", fixture, s.w, s.h)
			for i, ln := range lines {
				require.LessOrEqual(t, ansi.StringWidth(ln), s.w,
					"%s %dx%d: line %d", fixture, s.w, s.h, i)
			}
		}
	}
}

// indexOfLine is the row a section label is on, or -1. The label is the whole
// line, so "ERROR" does not match the status row of an errored check.
func indexOfLine(lines []string, label string) int {
	for i, ln := range lines {
		if strings.TrimSpace(ansi.Strip(ln)) == label {
			return i
		}
	}
	return -1
}
