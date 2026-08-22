// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/cockroachdb/errors"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/policy"
)

// The check the fixture gives this feature its best workout on: three
// remediations, six code blocks between them in three languages, one of them a
// multi-command bash script.
const copyFixtureCheck = "Ensure X Window System is not installed"

// clipboardSpy replaces the real clipboard for the duration of a test.
//
// Calling the real one would overwrite whatever the developer running the test
// had copied, and on a build box with no pbcopy, xclip or wl-copy it would fail
// for a reason that has nothing to do with the viewer.
func clipboardSpy(t *testing.T, err error) *string {
	t.Helper()
	var got string
	was := clipboardWrite
	clipboardWrite = func(s string) error {
		got = s
		return err
	}
	t.Cleanup(func() { clipboardWrite = was })
	return &got
}

// run drives a command the way bubbletea would and returns the message.
func run(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// styled pins lipgloss's own color profile, which is what the chrome styles in
// theme.go render through. lipgloss detects Ascii whenever stdout is not a
// terminal -- every test run -- and with Ascii every style renders as the plain
// string, so a test that wants to tell the armed button from the rest has to ask
// for a profile that has an escape sequence to tell them apart with.
func styled(t *testing.T) {
	t.Helper()
	was := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(was) })
}

// --- what a snippet is ------------------------------------------------------

// The whole point of the feature: what lands on the clipboard is the source, not
// the screen. The pane folded this block at the right edge, set it in two
// columns from the left and coloured it; none of that may survive into the
// clipboard, or the pasted command does not run.
func TestCopiedTextIsTheSourceNotTheScreen(t *testing.T) {
	colorful(t)
	_, check := stateFor(t, fixtureUbuntu, copyFixtureCheck)

	var ansible Snippet
	for _, s := range checkSnippets(check) {
		if s.From == "remediation [ansible]" {
			ansible = s
		}
	}
	require.NotEmpty(t, ansible.Text, "the ansible remediation has a fenced block")
	require.Equal(t, "yaml", ansible.Lang)

	// Not a fence in sight, and not an escape byte either.
	require.NotContains(t, ansible.Text, "```")
	require.NotContains(t, ansible.Text, "\x1b")
	require.Equal(t, ansible.Text, ansi.Strip(ansible.Text))

	// The block's own indentation is intact -- it is yaml, and yaml is
	// indentation -- while the pane's two columns of section indent are not.
	require.True(t, strings.HasPrefix(ansible.Text, "---\n- name:"),
		"the block starts at column zero:\n%q", ansible.Text)
	require.Contains(t, ansible.Text, "\n      ansible.builtin.yum:")

	// And the lines are the author's, not the pane's. The pane folded this
	// document at 60 cells; every line here is the length it was written at.
	rendered := ansi.Strip(strings.Join(markdown.Lines(check.Detail().Remediation[1].Desc, 60), "\n"))
	require.Contains(t, rendered, "- name: Remove X Window System packages on")
	require.NotContains(t, rendered, `- name: Remove X Window System packages on RHEL/Fedora/Amazon Linux`,
		"this line is too long for the pane, so the pane must have folded it")
	require.Contains(t, ansible.Text, `- name: Remove X Window System packages on RHEL/Fedora/Amazon Linux`,
		"and the copy must not have")
}

// A multi-command script comes out whole, in order, with its blank lines: it is
// a program, and half of one is worse than none.
func TestCopiedScriptIsIntact(t *testing.T) {
	_, check := stateFor(t, fixtureUbuntu, copyFixtureCheck)

	var script Snippet
	for _, s := range checkSnippets(check) {
		if s.From == "remediation [bash]" {
			script = s
		}
	}
	require.True(t, strings.HasPrefix(script.Text, "#!/bin/bash\nset -e\n"))
	require.True(t, strings.HasSuffix(script.Text, "exit 1\nfi"),
		"the block ends where the fence did, with no trailing blank line: %q",
		script.Text[len(script.Text)-40:])
	require.Contains(t, script.Text, "\nelif command -v apt-get >/dev/null 2>&1; then\n")
	require.Equal(t, strings.Count(script.Text, "\n")+1, 19)
}

// The query is a snippet in its own right: pasting it into `cnspec shell` is how
// you find out what the check saw.
func TestQueryIsCopyable(t *testing.T) {
	_, check := stateFor(t, fixtureUbuntu, copyFixtureCheck)
	snips := checkSnippets(check)
	require.NotEmpty(t, snips)
	require.Equal(t, "the query", snips[0].From, "the query comes before the remediation")
	require.Equal(t, "mql", snips[0].Lang)
	require.Equal(t, check.Detail().Mql, snips[0].Text)
}

// Every snippet is named by where it came from and what it is, because a check
// with the same fix three times over cannot be reported as "copied".
func TestSnippetsAreLabelledByOriginAndLanguage(t *testing.T) {
	_, check := stateFor(t, fixtureUbuntu, copyFixtureCheck)

	labels := make([]string, 0, 6)
	for _, s := range checkSnippets(check) {
		labels = append(labels, s.Label()+" "+s.Lang)
	}
	require.Equal(t, []string{
		"the query mql",
		// The cli fix is three commands, one per distro family, so they are
		// numbered; the other two are one block each and are not.
		"remediation [cli] #1 bash",
		"remediation [cli] #2 bash",
		"remediation [cli] #3 bash",
		"remediation [ansible] yaml",
		"remediation [bash] bash",
	}, labels)

	require.Equal(t, "remediation [cli] #2 (bash, 26 B)",
		checkSnippets(check)[2].Describe())
}

// --- finding the blocks -----------------------------------------------------

// The cases a scan for backticks gets wrong. Each is what CommonMark says, and
// goldmark is what says it.
func TestFencedBlockEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []mdCode
	}{
		{
			name: "a fence may be tildes, and may then contain backticks",
			src:  "before\n\n~~~sh\necho `date`\n~~~\n\nafter\n",
			want: []mdCode{{Lang: "sh", Text: "echo `date`"}},
		},
		{
			name: "a fence need not carry a language",
			src:  "a\n\n```\necho hi\n```\n",
			want: []mdCode{{Lang: "", Text: "echo hi"}},
		},
		{
			name: "an unclosed fence runs to the end of the document",
			src:  "before\n\n```bash\necho hi\nand more\n",
			want: []mdCode{{Lang: "bash", Text: "echo hi\nand more"}},
		},
		{
			name: "a longer fence is not closed by a shorter one inside it",
			src:  "a\n\n````md\n```\nnested\n```\n````\n",
			want: []mdCode{{Lang: "md", Text: "```\nnested\n```"}},
		},
		{
			name: "an indented fence loses its indent, not the block's own",
			src:  "a\n\n   ```bash\n   if x; then\n     echo hi\n   fi\n   ```\n",
			want: []mdCode{{Lang: "bash", Text: "if x; then\n  echo hi\nfi"}},
		},
		{
			name: "blank lines around the content are the author's spacing",
			src:  "a\n\n```bash\n\necho hi\n\n```\n",
			want: []mdCode{{Lang: "bash", Text: "echo hi"}},
		},
		{
			name: "an empty fence has nothing to copy",
			src:  "a\n\n```\n```\n\nb\n",
			want: nil,
		},
		{
			name: "an indented code block is not a fenced one",
			src:  "a\n\n    echo indented\n\nb\n",
			want: nil,
		},
		{
			name: "a fence nested in a list is left alone",
			src:  "1. do this:\n\n   ```bash\n   echo hi\n   ```\n\n2. done\n",
			want: nil,
		},
		{
			name: "several fences come back in document order",
			src:  "a\n\n```bash\none\n```\n\nb\n\n```yaml\nk: v\n```\n",
			want: []mdCode{{Lang: "bash", Text: "one"}, {Lang: "yaml", Text: "k: v"}},
		},
		{
			name: "prose that merely mentions a backtick is not a block",
			src:  "set the `PATH` and you are done\n",
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := mdSource(tc.src)
			got := fencedCodes(src)
			require.Len(t, got, len(tc.want))
			for i := range tc.want {
				require.Equal(t, tc.want[i].Lang, got[i].Lang)
				require.Equal(t, tc.want[i].Text, got[i].Text)
				// The span brackets the whole block, fences included, so the
				// renderer can cut the document at it without losing a byte.
				block := src[got[i].Start:got[i].Stop]
				require.True(t,
					strings.HasPrefix(strings.TrimLeft(block, " "), "```") ||
						strings.HasPrefix(strings.TrimLeft(block, " "), "~~~"),
					"the span starts at the opening fence: %q", block)
			}
		})
	}
}

// The spans must tile the document: everything outside them is prose, and
// nothing may be counted twice or dropped.
func TestFencedSpansTileTheDocument(t *testing.T) {
	report := loadReport(t, fixtureUbuntu)
	sources := 0
	for _, a := range report.Assets {
		for _, c := range a.Checks {
			d := c.Detail()
			srcs := []string{d.Description, d.Audit}
			for _, r := range d.Remediation {
				srcs = append(srcs, r.Desc)
			}
			for _, raw := range srcs {
				src := mdSource(raw)
				codes := fencedCodes(src)
				if len(codes) == 0 {
					continue
				}
				sources++
				var rebuilt strings.Builder
				at := 0
				for _, c := range codes {
					require.GreaterOrEqual(t, c.Start, at, "spans overlap")
					require.Greater(t, c.Stop, c.Start)
					require.LessOrEqual(t, c.Stop, len(src))
					rebuilt.WriteString(src[at:c.Start])
					rebuilt.WriteString(src[c.Start:c.Stop])
					at = c.Stop
				}
				rebuilt.WriteString(src[at:])
				require.Equal(t, src, rebuilt.String())
			}
		}
	}
	require.Greater(t, sources, 20, "the fixture is supposed to be full of these")
}

// Splitting the document to find the blocks must not change how it reads. Every
// markdown source in the fixture is rendered both ways and the rows compared.
func TestSegmentedRenderMatchesWholeDocument(t *testing.T) {
	colorful(t)
	report := loadReport(t, fixtureUbuntu)
	compared := 0
	for _, a := range report.Assets {
		for _, c := range a.Checks {
			d := c.Detail()
			srcs := []string{d.Description, d.Audit}
			for _, r := range d.Remediation {
				srcs = append(srcs, r.Desc)
			}
			for _, src := range srcs {
				if strings.TrimSpace(src) == "" {
					continue
				}
				compared++
				var pieces []string
				for i, b := range markdown.Blocks(src, 60, 60) {
					if i > 0 {
						pieces = append(pieces, "")
					}
					pieces = append(pieces, b.Lines...)
				}
				require.Equal(t, markdown.Lines(src, 60), pieces,
					"rendering %q in pieces does not match rendering it whole", d.Title)
			}
		}
	}
	require.Greater(t, compared, 50)
}

// Splitting a document renders it at two widths -- prose at the pane's, code a
// button's strip narrower. A renderer cache with one slot would rebuild
// glamour's whole pipeline at every fence, so it has room for both, and the room
// is bounded so a drag of the terminal's edge cannot fill it.
func TestSegmentingDoesNotThrashTheRenderer(t *testing.T) {
	colorful(t)
	m := &markdownRenderer{}

	src := "prose\n\n```bash\none\n```\n\nmore prose\n\n```yaml\nk: v\n```\n\nand more\n"
	for i := 0; i < 50; i++ {
		require.NotEmpty(t, m.Blocks(src, 60, 60-copyGutter))
	}
	require.Len(t, m.trs, 2, "one renderer for the prose width and one for the code width")

	// Every width a resize sweeps through, and it still does not grow.
	for w := 30; w < 120; w++ {
		m.Blocks(src, w, w-copyGutter)
	}
	require.LessOrEqual(t, len(m.trs), markdownMaxTRs)
}

// --- the button -------------------------------------------------------------

// A COPY sits at the top right of every code block, and the row it sits on is
// still exactly one row of exactly the pane's width: the block is *rendered*
// narrower to make space, so the button covers no part of the command.
func TestCopyButtonSitsOnEveryCodeBlock(t *testing.T) {
	colorful(t)
	st, _ := stateFor(t, fixtureUbuntu, copyFixtureCheck)

	p := &detailPane{}
	rect := detailRect(64, 400)
	r := p.Render(st, rect)

	require.Len(t, p.buttons, 6, "six code blocks, six buttons")
	require.Len(t, r.Zones, 6, "and one clickable zone each")

	for i, btn := range p.buttons {
		row := ansi.Strip(r.Lines[btn.Row])
		require.True(t, strings.HasSuffix(row, copyLabel),
			"button %d is not at the right edge of %q", i, row)
		require.Equal(t, rect.W, tui.Width(r.Lines[btn.Row]),
			"the button row is not exactly the pane's width")

		// The first line of the block is still all there in front of it: the
		// button took its strip out of the width the block was rendered at
		// rather than being painted over the command.
		snip := p.snips[btn.Idx]
		first := strings.SplitN(snip.Text, "\n", 2)[0]
		if tui.Width(first) <= rect.W-copyGutter-detailIndent {
			require.Contains(t, row, first,
				"button %d hid part of the block it copies", i)
			require.NotContains(t, row, "…",
				"button %d had to truncate the block's first line", i)
		}
	}

	// The zones are where the buttons were drawn, in absolute cells.
	for i, z := range r.Zones {
		require.Equal(t, copyZoneTag, z.Tag)
		require.Equal(t, rect.X+rect.W-copyLabelW, z.Rect.X)
		require.Equal(t, copyLabelW, z.Rect.W)
		require.Equal(t, 1, z.Rect.H)
		require.Equal(t, rect.Y+p.buttons[i].Row, z.Rect.Y)
		require.Equal(t, p.buttons[i].Idx, z.Idx)
	}
}

// The armed button -- the one y takes -- is the topmost block *on screen*, and
// it is the only one wearing the band. Scrolling is still an aim, so scrolling
// still has to move it.
//
// The two ends of the page are the interesting part, and both changed when n and
// p arrived: an arm is a thing the user can see now, so a page scrolled to a
// place with no code on it arms nothing at all rather than reaching off screen
// for the nearest block. See the arming rule at the top of copy.go.
func TestArmedButtonFollowsTheScroll(t *testing.T) {
	colorful(t)
	st, _ := stateFor(t, fixtureUbuntu, copyFixtureCheck)

	p := &detailPane{}
	p.Render(st, detailRect(64, 20))
	require.Greater(t, len(p.buttons), 2)

	// This check opens on a long description: its first fence is well below the
	// twenty rows on screen, so at the top of the page nothing is armed.
	require.Greater(t, p.buttons[0].Row, 20)
	require.Equal(t, -1, p.armed())
	_, ok := p.CopyTarget(st)
	require.False(t, ok, "y takes nothing it cannot show")

	// Scroll so the first block's row is the top row: that one.
	p.scroll.Off = p.buttons[0].Row
	p.Render(st, detailRect(64, 20))
	require.Equal(t, 0, p.armed())

	// One row past it, and the next block takes over.
	p.scroll.Off = p.buttons[0].Row + 1
	p.Render(st, detailRect(64, 20))
	require.Equal(t, 1, p.armed())

	// Scrolled past every block -- a long check ends in POLICIES, which has no
	// code in it -- nothing is armed rather than the last block being armed
	// somewhere above the top of the screen.
	p.scroll.Off = len(p.lines)
	p.Render(st, detailRect(64, 20))
	require.Equal(t, -1, p.armed())
	_, ok = p.CopyTarget(st)
	require.False(t, ok)
}

// Exactly one button on screen is banded, and it is the armed one. This is the
// whole of what makes it obvious which block y would take.
func TestArmedButtonIsTheOnlyAccentedOne(t *testing.T) {
	colorful(t)
	styled(t)
	st, _ := stateFor(t, fixtureUbuntu, copyFixtureCheck)

	p := &detailPane{}
	p.Render(st, detailRect(64, 400))
	require.Greater(t, len(p.buttons), 2)

	// Both are bands, so both read as buttons; only the accent one says "y
	// takes this". Drawing the unarmed ones as faint text made them look
	// disabled, which they are not -- a click copies them just as well.
	band := tui.BandSelected.Render(copyLabel)
	quiet := tui.BandInactive.Render(copyLabel)
	require.NotEqual(t, band, quiet, "armed and unarmed must not render the same")

	// Three blocks in view at once, with the topmost of them armed. The viewport
	// has to be shorter than the page or the scroll offset clamps back to zero.
	p.scroll.Off = p.buttons[1].Row
	r := p.Render(st, detailRect(64, 20))

	var accented, quiets []int
	for i, ln := range r.Lines {
		switch {
		case strings.Contains(ln, band):
			accented = append(accented, i)
		case strings.Contains(ln, quiet):
			quiets = append(quiets, i)
		}
	}
	require.Equal(t, []int{0}, accented, "one accented button, on the topmost visible block")
	require.NotEmpty(t, quiets, "and the blocks below it still look like buttons")
	require.Equal(t, 1, p.armed())
}

// A pane too narrow to set a button beside a block draws none: a COPY wider than
// the command it copies is the bug, not the feature. The key still works.
func TestNoButtonsInAVeryNarrowPane(t *testing.T) {
	colorful(t)
	st, _ := stateFor(t, fixtureUbuntu, copyFixtureCheck)

	p := &detailPane{}
	r := p.Render(st, detailRect(12, 200))
	require.False(t, p.room)
	require.Empty(t, r.Zones)
	for _, ln := range r.Lines {
		require.NotContains(t, ansi.Strip(ln), "COPY")
	}

	// The blocks were still found, so y still has something to take.
	require.NotEmpty(t, p.buttons)
	snip, ok := p.CopyTarget(st)
	require.True(t, ok)
	require.Equal(t, "the query", snip.From)
}

// The frame's invariant, with the buttons on: every row is one row, no wider
// than the terminal, at every size the geometry tests use.
func TestButtonsKeepTheGeometry(t *testing.T) {
	colorful(t)
	report := loadReport(t, fixtureUbuntu)
	asset := report.Assets[0]

	for _, c := range asset.Checks {
		st := NewState(report)
		st.SelectCheck(asset, nil, c)
		for _, s := range termSizes {
			p := &detailPane{}
			rect := detailRect(s.w, s.h)
			r := p.Render(st, rect)
			for i, ln := range r.Lines {
				require.LessOrEqual(t, tui.Width(ln), rect.W,
					"%dx%d row %d: %q", s.w, s.h, i, ansi.Strip(ln))
				require.NotContains(t, ln, "\n")
			}
			for _, z := range r.Zones {
				require.GreaterOrEqual(t, z.Rect.X, rect.X)
				require.LessOrEqual(t, z.Rect.X+z.Rect.W, rect.X+rect.W)
				require.GreaterOrEqual(t, z.Rect.Y, rect.Y)
				require.Less(t, z.Rect.Y, rect.Y+rect.H)
			}
		}
	}
}

// --- copying ----------------------------------------------------------------

// The key takes the armed block, puts the source on the clipboard and says so.
func TestCopyKeyCopiesTheArmedBlock(t *testing.T) {
	colorful(t)
	got := clipboardSpy(t, nil)

	report := loadReport(t, fixtureUbuntu)
	m := sized(NewModel(report), 120, 40)
	asset := report.Assets[0]
	var check = asset.Checks[0]
	for _, c := range asset.Checks {
		if c.Title == copyFixtureCheck {
			check = c
		}
	}
	m.state.SelectCheck(asset, nil, check)
	m.View() // one frame, so the detail pane knows where its blocks are

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	msg := run(cmd)
	require.IsType(t, copyDoneMsg{}, msg)

	nm, _ = nm.(Model).Update(msg)
	m = nm.(Model)

	require.Equal(t, check.Detail().Mql, *got, "the top block is the query")
	require.Equal(t, "copied the query (mql, 85 B) to the clipboard", m.state.Notice)
	require.Contains(t, m.View(), ansi.Strip("copied the query"))
}

// A click takes the block under the pointer, not the armed one: the button you
// pressed is the answer to which block you meant.
func TestClickOnACopyButtonTakesThatBlock(t *testing.T) {
	colorful(t)
	got := clipboardSpy(t, nil)

	st, check := stateFor(t, fixtureUbuntu, copyFixtureCheck)
	p := &detailPane{}
	rect := detailRect(64, 400)
	r := p.Render(st, rect)
	require.Len(t, r.Zones, 6)

	// The ansible playbook, which is the fifth block and never the armed one at
	// this scroll position.
	z := r.Zones[4]
	require.NotEqual(t, p.armed(), z.Idx)

	cmd, handled := p.Update(st, ClickMsg{Zone: z, Mouse: tea.MouseMsg{X: z.Rect.X, Y: z.Rect.Y}})
	require.True(t, handled)
	msg := run(cmd)
	require.IsType(t, copyDoneMsg{}, msg)
	require.NoError(t, msg.(copyDoneMsg).Err)

	require.Equal(t, "remediation [ansible]", msg.(copyDoneMsg).Snippet.From)
	require.True(t, strings.HasPrefix(*got, "---\n- name: Remove X Window System packages"))
	require.Contains(t, check.Detail().Remediation[1].Desc, *got,
		"what was copied is a literal substring of the markdown it came from")
}

// The same click, through the whole frame: hit-tested against the zones of the
// frame just rendered, so the button that responds is the button that was drawn.
func TestClickThroughTheFrameCopies(t *testing.T) {
	colorful(t)
	got := clipboardSpy(t, nil)

	report := loadReport(t, fixtureUbuntu)
	m := sized(NewModel(report), 140, 44)
	asset := report.Assets[0]
	for _, c := range asset.Checks {
		if c.Title == copyFixtureCheck {
			m.state.SelectCheck(asset, nil, c)
		}
	}
	m.View()

	var target Zone
	for _, z := range m.build().zones {
		if z.Tag == copyZoneTag {
			target = z
			break
		}
	}
	require.Equal(t, PaneDetail, target.Pane, "the zone belongs to the detail pane")

	_, cmd := m.Update(tea.MouseMsg{
		X: target.Rect.X, Y: target.Rect.Y,
		Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	msg := run(cmd)
	require.IsType(t, copyDoneMsg{}, msg)
	require.NotEmpty(t, *got)
	require.Equal(t, msg.(copyDoneMsg).Snippet.Text, *got)
}

// A check with no code in it says so rather than opening something empty. This
// package has a rule against empty pickers and the same rule applies to a key
// that appears to do nothing.
func TestNothingToCopySaysSo(t *testing.T) {
	// report-k8s is fifteen assets that never scanned: no bundle, no checks, no
	// code anywhere in it.
	m := sized(NewModel(loadReport(t, fixtureK8s)), 100, 30)
	m.View()
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	require.Nil(t, cmd)
	require.Equal(t, "nothing to copy: no check is selected", nm.(Model).state.Notice)

	// And a check whose sections hold prose but no fenced block.
	st := &State{Report: loadReport(t, fixtureUbuntu)}
	asset := st.Report.Assets[0]
	for _, c := range asset.Checks {
		d := c.Detail()
		if d.Mql == "" && len(checkSnippets(c)) == 0 {
			st.SelectCheck(asset, nil, c)
			p := &detailPane{}
			p.Render(st, detailRect(80, 40))
			_, ok := p.CopyTarget(st)
			require.False(t, ok)
			return
		}
	}
}

// A clipboard that refused says why, in the words of whatever refused it. There
// is no hedging in either direction: clipboard.WriteAll returning nil means the
// text is on the clipboard, and an error means it is not.
func TestCopyFailureNamesTheReason(t *testing.T) {
	clipboardSpy(t, errors.New(`exec: "xclip": executable file not found in $PATH`))

	snip := Snippet{From: "remediation [cli]", Lang: "bash", Text: "yum remove xorg-x11*", Nth: 1, Of: 1}
	msg := run(copyCmd(snip))
	require.IsType(t, copyDoneMsg{}, msg)
	done := msg.(copyDoneMsg)
	require.Error(t, done.Err)

	notice := copyNotice(done)
	require.Equal(t,
		`copy failed: the system clipboard refused it: exec: "xclip": executable file not found in $PATH`,
		notice)
	require.NotContains(t, notice, "copied ")
}

// A notice is one row of a fixed-height view, so it can never carry a newline:
// a writer, an encoder or the operating system may put one in an error and the
// frame has no way to see it.
func TestCopyNoticeIsOneLine(t *testing.T) {
	notice := copyNotice(copyDoneMsg{Err: errors.New("xclip died\nsignal: broken pipe\r\n")})
	require.NotContains(t, notice, "\n")
	require.NotContains(t, notice, "\r")
	require.Equal(t, "copy failed: xclip died signal: broken pipe", notice)
}

// --- the key ----------------------------------------------------------------

// y is free, and stays free while the header's search field is open: that field
// consumes every rune before the frame is reached, so typing a y into a search
// types a y.
func TestCopyKeyDoesNotBreakSearch(t *testing.T) {
	got := clipboardSpy(t, nil)

	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "yes" {
		nm, _ = nm.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m = nm.(Model)

	require.Equal(t, "yes", m.state.Filter.Search)
	require.Empty(t, *got, "the y went into the search field, not the clipboard")
}

// The keys are published, so they can be found without reading this file: y
// fires the copy and n/p aim it, and neither is any use undiscovered.
//
// They are advertised in two different places for one reason, room. The compact
// line is priority-ordered and the frame's keys sit at the end of it, so a sixth
// frame hint pushes ? and q off a 120-column terminal; n/p therefore rides the
// detail pane's hints, which is the pane that draws the band they move. The ?
// list, which is the complete key map, carries them as the frame bindings they
// are.
func TestCopyKeysAreInTheHints(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)

	find := func(hints []Hint, key string) string {
		for _, h := range hints {
			if h.Key == key {
				return h.Label
			}
		}
		return ""
	}

	require.Equal(t, "copy", find(m.frameHints(), "y"))
	require.Contains(t, ansi.Strip(m.View()), "y copy")
	require.Contains(t, ansi.Strip(m.View()), "q quit",
		"the compact line still reaches its last hint at this width")

	// The pair, on the pane that shows what they move.
	var d Pane = &detailPane{}
	require.Equal(t, "block", find(d.Hints(m.state), "n/p"))
	m.state.Focus = PaneDetail
	require.Contains(t, ansi.Strip(m.View()), "n/p block")

	// And ? spells both out, as the frame's own.
	m.showHelp = true
	require.Equal(t, "copy the highlighted code block", find(m.frameHints(), "y"))
	require.Equal(t, "next / previous code block", find(m.frameHints(), "n/p"))
	out := ansi.Strip(sized(m, 200, 40).View())
	require.Contains(t, out, "n/p next / previous code block")
	require.Contains(t, out, "y copy the highlighted code block")
}

// --- CopyTarget -------------------------------------------------------------

// Below tui.MinTwoPaneWidth only the focused pane is drawn, so the detail pane
// may not have been rendered for the current selection when y is pressed from
// the tree. It answers from the check rather than refusing.
func TestCopyWorksFromTheTreeInAOnePaneLayout(t *testing.T) {
	colorful(t)
	got := clipboardSpy(t, nil)

	report := loadReport(t, fixtureUbuntu)
	m := sized(NewModel(report), 60, 24)
	require.False(t, m.layout().TwoPane, "this width is one pane at a time")

	asset := report.Assets[0]
	for _, c := range asset.Checks {
		if c.Title == copyFixtureCheck {
			m.state.SelectCheck(asset, nil, c)
		}
	}
	m.state.Focus = PaneTree
	m.View()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	msg := run(cmd)
	require.IsType(t, copyDoneMsg{}, msg)
	require.NoError(t, msg.(copyDoneMsg).Err)
	require.Equal(t, "the query", msg.(copyDoneMsg).Snippet.From)
	require.NotEmpty(t, *got)
}

// The detail pane is the one pane that offers a target, and it does so through
// the optional interface rather than by the frame naming it.
func TestDetailPaneIsACopySource(t *testing.T) {
	var p Pane = &detailPane{}
	_, ok := p.(CopySource)
	require.True(t, ok)

	var tp Pane = &treePane{}
	_, ok = tp.(CopySource)
	require.False(t, ok, "the tree has no code in it")
}

// A snippet with nothing in it never reaches the clipboard: writing an empty
// string would clear whatever the user had copied and report success.
func TestEmptySnippetIsRefused(t *testing.T) {
	got := clipboardSpy(t, nil)
	msg := run(copyCmd(Snippet{From: "remediation"}))
	require.Error(t, msg.(copyDoneMsg).Err)
	require.Empty(t, *got)
}

// The button strip is reserved out of the code's width rather than painted over
// it, which is the whole reason no command loses a character to it.
func TestButtonStripIsReservedNotOverwritten(t *testing.T) {
	w, room := copyWidth(40)
	require.True(t, room)
	require.Equal(t, 40-copyGutter, w)

	row := copyButtonRow(tui.StyleText.Render(strings.Repeat("x", 40-copyGutter)), 40, false)
	require.Equal(t, 40, tui.Width(row))
	require.Equal(t, strings.Repeat("x", 33)+"  COPY ", ansi.Strip(row))
	require.NotContains(t, ansi.Strip(row), "…")

	// Too narrow for a strip at all.
	w, room = copyWidth(10)
	require.False(t, room)
	require.Equal(t, 10, w)
}

// The rect a zone claims is inside the pane it came from, which is what the
// frame's hit-tester assumes.
func TestCopyZonesStayInsideTheRect(t *testing.T) {
	colorful(t)
	st, _ := stateFor(t, fixtureUbuntu, copyFixtureCheck)
	p := &detailPane{}
	rect := tui.Rect{X: 7, Y: 3, W: 50, H: 30}
	r := p.Render(st, rect)
	for _, z := range r.Zones {
		require.True(t, z.Rect.Hit(z.Rect.X, z.Rect.Y))
		require.GreaterOrEqual(t, z.Rect.X, rect.X)
		require.LessOrEqual(t, z.Rect.X+z.Rect.W, rect.X+rect.W)
		require.GreaterOrEqual(t, z.Rect.Y, rect.Y)
		require.Less(t, z.Rect.Y, rect.Y+rect.H)
	}
}

// --- aiming: n and p ---------------------------------------------------------

// armedFixture is a pane rendered on the six-block check at a size where the
// page is six times the height of the viewport, which is the shape the whole
// feature exists for: several buttons on screen at once and no way to say which
// one you meant.
func armedFixture(t *testing.T) (*State, *detailPane) {
	t.Helper()
	colorful(t)
	st, _ := stateFor(t, fixtureUbuntu, copyFixtureCheck)
	p := &detailPane{}
	p.Render(st, detailRect(64, 20))
	require.Len(t, p.buttons, 6)
	return st, p
}

// The point of the whole change: the arm is a selection you drive, not a
// consequence of where you happened to stop scrolling. n takes the next block, p
// the one before, and every step is a block of this check in order.
func TestArmKeysWalkTheBlocks(t *testing.T) {
	st, p := armedFixture(t)

	// Nothing is on screen to start with -- this check opens on its description
	// -- so n means "the first block below the fold" rather than "one past
	// nothing".
	require.Equal(t, -1, p.armed())
	require.True(t, p.ArmCopy(st, 1))
	require.Equal(t, 0, p.armed())

	for want := 1; want < len(p.buttons); want++ {
		require.True(t, p.ArmCopy(st, 1))
		require.Equal(t, want, p.armed())
		p.Render(st, detailRect(64, 20))
		require.Equal(t, want, p.armed(), "and the render agrees with the key")
	}
	for want := len(p.buttons) - 2; want >= 0; want-- {
		require.True(t, p.ArmCopy(st, -1))
		require.Equal(t, want, p.armed())
	}

	// Both ends hold rather than wrapping: a cursor that jumps from the last
	// block of a check to the first also throws the pane from the bottom of the
	// page to the top, which reads as a scroll accident, not a selection.
	require.True(t, p.ArmCopy(st, -1))
	require.Equal(t, 0, p.armed())
	for range len(p.buttons) + 2 {
		require.True(t, p.ArmCopy(st, 1))
	}
	require.Equal(t, len(p.buttons)-1, p.armed())
}

// An arm the user cannot see is worse than the arbitrary behaviour it replaces,
// so moving it scrolls it into view -- with a few rows of the block under it
// where the pane can spare them, because a band on the bottom row shows you the
// button and not the command.
func TestArmScrollsTheBlockIntoView(t *testing.T) {
	st, p := armedFixture(t)

	for i := range p.buttons {
		require.True(t, p.ArmCopy(st, 1))
		require.Equal(t, i, p.armed())

		off, end := p.viewport()
		row := p.buttons[i].Row
		require.GreaterOrEqual(t, row, off, "block %d is above the viewport", i)
		require.Less(t, row, end, "block %d is below the viewport", i)

		// The rendered frame shows it too, not just the offsets.
		r := p.Render(st, detailRect(64, 20))
		require.Len(t, r.Lines, 20)
		require.Contains(t, ansi.Strip(r.Lines[row-off]), "COPY")

		if row+detailArmPeek < len(p.lines) {
			require.Less(t, row+detailArmPeek, end,
				"block %d got the band but none of the block", i)
		}
	}
}

// The explicit arm wins over the topmost block on screen. Without this the key
// would be a scroll with extra steps: n scrolls the second block into view, the
// first is still on screen above it, and "topmost" would hand the arm straight
// back.
func TestExplicitArmBeatsTheTopmostBlock(t *testing.T) {
	st, p := armedFixture(t)

	require.True(t, p.ArmCopy(st, 1))
	require.True(t, p.ArmCopy(st, 1))
	require.Equal(t, 1, p.armed())

	off, end := p.viewport()
	require.GreaterOrEqual(t, p.buttons[0].Row, off, "the first block is still on screen")
	require.Less(t, p.buttons[0].Row, end)
	require.Equal(t, 1, p.armed(), "and the arm stays where it was put")

	// The band is on the armed one and only on it.
	styled(t)
	r := p.Render(st, detailRect(64, 20))
	var banded []int
	for i, ln := range r.Lines {
		if strings.Contains(ln, tui.BandSelected.Render(copyLabel)) {
			banded = append(banded, i+off)
		}
	}
	require.Equal(t, []int{p.buttons[1].Row}, banded)
}

// The other half of the rule: a plain scroll that carries the armed block off
// the screen gives the arm back to the topmost block still on it. Scrolling was
// the only way to aim before this, and it is still an aim -- what it must not do
// is leave the band somewhere nobody can see it.
func TestScrollingAwayFromTheArmGivesItBack(t *testing.T) {
	st, p := armedFixture(t)

	require.True(t, p.ArmCopy(st, 1))
	require.True(t, p.ArmCopy(st, 1))
	require.True(t, p.armSet)
	armed := p.armed()

	// A scroll that keeps it on screen keeps it armed, even though a block above
	// it is now the topmost one.
	_, handled := p.Update(st, key("up"))
	require.True(t, handled)
	require.True(t, p.armSet)
	require.Equal(t, armed, p.armed())

	// A page down puts it above the top of the screen, and the arm reverts.
	_, handled = p.Update(st, key("pgdown"))
	require.True(t, handled)
	require.False(t, p.armSet, "the explicit arm did not survive scrolling off screen")

	off, end := p.viewport()
	require.Less(t, p.buttons[armed].Row, off, "the old arm really is off screen")
	if i := p.armed(); i >= 0 {
		require.GreaterOrEqual(t, p.buttons[i].Row, off)
		require.Less(t, p.buttons[i].Row, end)
	}

	// The wheel is a plain scroll too, and drops it the same way.
	require.True(t, p.ArmCopy(st, -1))
	require.True(t, p.armSet)
	for range 20 {
		p.Update(st, tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	}
	require.False(t, p.armSet)
}

// A new check is a new page, and the block you picked on the last one is not on
// it. The arm goes back to the implicit rule, which at the top of a fresh page
// is the first block on screen.
func TestArmResetsWithTheSelection(t *testing.T) {
	colorful(t)
	report := loadReport(t, fixtureUbuntu)
	st := NewState(report)
	asset := report.Assets[0]

	var first, second *reportmodel.Check
	for _, c := range asset.Checks {
		if len(checkSnippets(c)) < 2 {
			continue
		}
		switch {
		case first == nil:
			first = c
		case second == nil:
			second = c
		}
	}
	require.NotNil(t, second, "the fixture has two checks with code in them")

	st.SelectCheck(asset, nil, first)
	p := &detailPane{}
	p.Render(st, detailRect(64, 20))
	require.True(t, p.ArmCopy(st, 1))
	require.True(t, p.ArmCopy(st, 1))
	require.True(t, p.armSet)

	st.SelectCheck(asset, nil, second)
	p.Render(st, detailRect(64, 20))
	require.False(t, p.armSet, "the arm did not follow the selection to a new check")
	require.Equal(t, 0, p.scroll.Off, "and neither did the scroll")
}

// n and p are frame keys, like y and for the same reason: the tree is where the
// cursor spends its time, and a pair that only aimed while the detail pane had
// focus would work in fewer places than the key that fires.
func TestArmKeysWorkFromTheTree(t *testing.T) {
	colorful(t)
	got := clipboardSpy(t, nil)

	report := loadReport(t, fixtureUbuntu)
	m := sized(NewModel(report), 120, 40)
	asset := report.Assets[0]
	for _, c := range asset.Checks {
		if c.Title == copyFixtureCheck {
			m.state.SelectCheck(asset, nil, c)
		}
	}
	m.View()

	require.Equal(t, PaneTree, m.state.Focus, "the tree has focus, as it does on open")
	p := m.pane(PaneDetail).(*detailPane)
	require.Equal(t, 0, p.armed(), "the query is on screen and armed")

	m, cmd := press(m, "n")
	require.Nil(t, cmd)
	require.Equal(t, PaneTree, m.state.Focus, "and aiming did not steal focus")
	require.Equal(t, 1, p.armed())
	require.Empty(t, m.state.Notice)

	// y takes what n aimed at.
	m, cmd = press(m, "y")
	msg := run(cmd)
	require.IsType(t, copyDoneMsg{}, msg)
	require.Equal(t, "remediation [cli]", msg.(copyDoneMsg).Snippet.From)
	require.Equal(t, "yum remove xorg-x11*", *got)

	// And p takes it back to the query.
	m, _ = press(m, "p")
	require.Equal(t, 0, p.armed())
	_, cmd = press(m, "y")
	msg = run(cmd)
	require.Equal(t, "the query", msg.(copyDoneMsg).Snippet.From)
}

// A key that appears to do nothing is a key the user gives up on, so n and p say
// what is missing rather than sitting there -- the same rule y already follows.
func TestNothingToArmSaysSo(t *testing.T) {
	// report-k8s is fifteen assets that never scanned: no bundle, no checks, no
	// code anywhere in it.
	m := sized(NewModel(loadReport(t, fixtureK8s)), 100, 30)
	m.View()
	for _, k := range []string{"n", "p"} {
		nm, cmd := press(m, k)
		require.Nil(t, cmd)
		require.Equal(t, "nothing to highlight: no check is selected", nm.state.Notice)
	}

	// And a check whose sections hold prose but not one fenced block. Every
	// check of the ubuntu fixture has an MQL query, which is a block in its own
	// right, so this one is built rather than found.
	colorful(t)
	report := loadReport(t, fixtureUbuntu)
	asset := report.Assets[0]
	prose := &reportmodel.Check{
		Title:  "a check with nothing to run",
		Status: reportmodel.StatusFail,
		Query: &policy.Mquery{
			Title: "a check with nothing to run",
			Docs: &policy.MqueryDocs{
				Desc:  "Prose, and **bold** prose at that, but no fence anywhere in it.",
				Audit: "Look at it and see.",
			},
		},
	}
	require.Empty(t, checkSnippets(prose))

	m = sized(NewModel(report), 100, 30)
	m.state.SelectCheck(asset, nil, prose)
	m.View()
	nm, cmd := press(m, "n")
	require.Nil(t, cmd)
	require.Equal(t, "nothing to highlight: this check has no code blocks", nm.state.Notice)

	// y agrees, in its own words.
	nm, cmd = press(nm, "y")
	require.Nil(t, cmd)
	require.Equal(t, "nothing to copy: this check has no code blocks", nm.state.Notice)
}

// Scrolled to a stretch of the page with no code on it, y says so and names the
// key that gets to one, rather than reaching off screen for the nearest block.
func TestNothingInViewSaysWhichKeyReachesOne(t *testing.T) {
	colorful(t)
	report := loadReport(t, fixtureUbuntu)
	// Short enough that the tail of the check -- REFERENCES and POLICIES, which
	// have no code in them -- fills the screen on its own.
	m := sized(NewModel(report), 120, 24)
	asset := report.Assets[0]
	for _, c := range asset.Checks {
		if c.Title == copyFixtureCheck {
			m.state.SelectCheck(asset, nil, c)
		}
	}
	m.View()

	p := m.pane(PaneDetail).(*detailPane)
	_, handled := p.Update(m.state, key("end"))
	require.True(t, handled)
	m.View()
	require.Equal(t, -1, p.armed(), "the end of this check has no code on it")

	nm, cmd := press(m, "y")
	require.Nil(t, cmd)
	require.Equal(t,
		"nothing to copy: no code block is in view (press n for the next one)",
		nm.state.Notice)

	// And n gets to one, from there.
	nm, cmd = press(nm, "n")
	require.Nil(t, cmd)
	require.Empty(t, nm.state.Notice)
	require.GreaterOrEqual(t, p.armed(), 0)
	_, cmd = press(nm, "y")
	require.IsType(t, copyDoneMsg{}, run(cmd))
}

// n and p are free, and stay free while the header's search field is open: that
// field consumes every rune before the frame is reached, so typing an n into a
// search types an n.
func TestArmKeysDoNotBreakSearch(t *testing.T) {
	colorful(t)
	report := loadReport(t, fixtureUbuntu)
	m := sized(NewModel(report), 120, 40)
	asset := report.Assets[0]
	for _, c := range asset.Checks {
		if c.Title == copyFixtureCheck {
			m.state.SelectCheck(asset, nil, c)
		}
	}
	m.View()
	p := m.pane(PaneDetail).(*detailPane)
	was := p.armed()

	m, _ = press(m, "/")
	for _, r := range "no ping" {
		m, _ = press(m, string(r))
	}
	require.Equal(t, "no ping", m.state.Filter.Search)
	require.False(t, p.armSet, "the runes went into the search field, not at the arm")
	require.Equal(t, was, p.armed())
}

// The invariant, swept: whatever the pane has been through -- any size, any
// scroll position, any number of n and p -- the block y would take is a block
// whose button is on screen. This is what the accent band promises, and its
// absence is what made the old behaviour a guess.
func TestCopyNeverTakesABlockOffScreen(t *testing.T) {
	colorful(t)
	report := loadReport(t, fixtureUbuntu)
	asset := report.Assets[0]

	steps := []tea.Msg{
		key("n"), key("n"), key("down"), key("n"), key("pgdown"),
		key("p"), key("end"), key("n"), key("home"), key("p"),
		tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress},
		key("G"), key("n"), key("n"), key("up"),
	}

	for _, c := range asset.Checks {
		st := NewState(report)
		st.SelectCheck(asset, nil, c)

		for _, s := range termSizes {
			p := &detailPane{}
			rect := detailRect(s.w, s.h)
			p.Render(st, rect)

			for _, msg := range steps {
				switch k, isKey := msg.(tea.KeyMsg); {
				case isKey && k.String() == "n":
					require.Equal(t, len(p.buttons) > 0, p.ArmCopy(st, 1))
				case isKey && k.String() == "p":
					require.Equal(t, len(p.buttons) > 0, p.ArmCopy(st, -1))
				default:
					p.Update(st, msg)
				}
				p.Render(st, rect)

				off, end := p.viewport()
				i := p.armed()
				snip, ok := p.CopyTarget(st)
				if i < 0 {
					require.False(t, ok, "%dx%d: y had a target with nothing on screen", s.w, s.h)
					continue
				}
				require.True(t, ok)
				row := p.buttons[i].Row
				require.GreaterOrEqual(t, row, off,
					"%dx%d: y would copy a block above the screen", s.w, s.h)
				require.Less(t, row, end,
					"%dx%d: y would copy a block below the screen", s.w, s.h)
				require.Equal(t, p.snips[p.buttons[i].Idx].Text, snip.Text)
			}
		}
	}
}
