// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/cli/tui"
)

// termSizes covers the sizes that actually break things: the 80x24 default, a
// narrow window that has to drop to one pane, and a tall one where the list is
// long enough to need scrolling.
var termSizes = []struct{ w, h int }{
	{80, 24}, {100, 30}, {120, 40}, {200, 60}, {70, 24}, {60, 15}, {40, 10},
}

func sized(m Model, w, h int) Model {
	nm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return nm.(Model)
}

func viewLines(m Model) []string {
	return strings.Split(m.View(), "\n")
}

// TestViewNeverOverflows is the test that keeps the launcher inside the
// terminal. A view taller than the screen scrolls the alt-screen buffer, which
// pushes the header off and leaves the cursor pointing at rows nobody can see.
func TestViewNeverOverflows(t *testing.T) {
	for _, focus := range []focusArea{focusList, focusForm} {
		for _, s := range termSizes {
			m := sized(newTestModel(), s.w, s.h)
			m.focus = focus
			lines := viewLines(m)
			if len(lines) != s.h {
				t.Errorf("focus=%d %dx%d: rendered %d lines, want exactly %d",
					focus, s.w, s.h, len(lines), s.h)
			}
			for i, ln := range lines {
				if w := tui.Width(ln); w > s.w {
					t.Errorf("focus=%d %dx%d: line %d is %d cells wide, want <= %d",
						focus, s.w, s.h, i, w, s.w)
				}
			}
		}
	}
}

// The full catalog is the real stress case: many categories, many rows.
func TestViewNeverOverflowsWithFullCatalog(t *testing.T) {
	base := NewModel(BuildCatalog())
	for _, s := range termSizes {
		m := sized(base, s.w, s.h)
		if lines := viewLines(m); len(lines) != s.h {
			t.Errorf("%dx%d: rendered %d lines, want exactly %d", s.w, s.h, len(lines), s.h)
		}
		// Scroll to the very bottom and re-check: the last page is where an
		// off-by-one in the offset clamp shows up.
		m.list.cursor = len(m.list.selectable) - 1
		m.list.ensureVisible(m.listH())
		if lines := viewLines(m); len(lines) != s.h {
			t.Errorf("%dx%d at end of list: rendered %d lines, want exactly %d", s.w, s.h, len(lines), s.h)
		}
	}
}

// TestZonesMatchRender asserts that a click zone lands on the thing the
// renderer drew at that line. Both come from computeLayout, and this is what
// proves they have not drifted.
func TestZonesMatchRender(t *testing.T) {
	m := selectEntry(t, sized(newTestModel(), 120, 40), "ssh")
	m.focus = focusForm
	m.detail.loadCursorField()
	l := computeLayout(m)
	lines := viewLines(m)

	lineAt := func(y int) string {
		if y < 0 || y >= len(lines) {
			t.Fatalf("y=%d outside the rendered view (%d lines)", y, len(lines))
		}
		return ansi.Strip(lines[y])
	}

	var sawEntry, sawField, sawRun bool
	for _, z := range l.Zones {
		switch z.Kind {
		case tui.ZoneEntry:
			want := m.list.filtered[m.list.rows[m.list.selectable[z.Idx]].idx].Name
			if got := lineAt(z.Rect.Y); !strings.Contains(got, want) {
				t.Errorf("entry zone at y=%d should be %q, line is %q", z.Rect.Y, want, got)
			}
			sawEntry = true
		case tui.ZoneField:
			want := m.detail.form.Fields()[z.Idx].Label
			if got := lineAt(z.Rect.Y); !strings.Contains(got, want) {
				t.Errorf("field zone at y=%d should be %q, line is %q", z.Rect.Y, want, got)
			}
			sawField = true
		case tui.ZoneRun:
			if got := lineAt(z.Rect.Y); !strings.Contains(got, "Scan") {
				t.Errorf("run zone at y=%d should hold the scan button, line is %q", z.Rect.Y, got)
			}
			sawRun = true
		}
	}
	if !sawEntry || !sawField || !sawRun {
		t.Fatalf("missing zones: entry=%v field=%v run=%v", sawEntry, sawField, sawRun)
	}
}

func TestClickSelectsListEntry(t *testing.T) {
	m := sized(newTestModel(), 120, 40)
	l := computeLayout(m)

	var target tui.Zone
	for _, z := range l.Zones {
		if z.Kind == tui.ZoneEntry && z.Idx == 2 {
			target = z
		}
	}
	if target.Kind != tui.ZoneEntry {
		t.Fatal("no entry zone for selectable index 2")
	}
	nm, _ := m.Update(tea.MouseMsg{
		X: tui.InnerX, Y: target.Rect.Y,
		Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	if got := nm.(Model).list.cursor; got != 2 {
		t.Fatalf("expected the click to select entry 2, got %d", got)
	}
}

// Below minTwoPaneWidth the launcher shows one pane, following focus, rather
// than squeezing two unreadable columns into the window.
func TestNarrowTerminalUsesOnePane(t *testing.T) {
	m := sized(newTestModel(), 60, 20)
	if computeLayout(m).TwoPane {
		t.Fatal("expected a single pane at width 60")
	}
	if !strings.Contains(ansi.Strip(m.View()), "local") {
		t.Fatal("expected the list pane while focus is on the list")
	}

	m = selectEntry(t, m, "aws")
	m.focus = focusForm
	out := ansi.Strip(m.View())
	if !strings.Contains(out, "an AWS account") {
		t.Fatal("expected the detail pane once focus moves off the list")
	}
}

// The scroll offset must survive a resize that grows the terminal past the end
// of the list.
func TestOffsetClampsOnResize(t *testing.T) {
	m := sized(NewModel(BuildCatalog()), 120, 20)
	m.list.cursor = len(m.list.selectable) - 1
	m.list.ensureVisible(m.listH())
	if m.list.offset.Off == 0 {
		t.Skip("catalog too short to scroll at this size")
	}
	m = sized(m, 120, 200)
	l := computeLayout(m)
	// Everything now fits, so the list must be scrolled back to the top rather
	// than left showing a window past the end of the rows.
	if off := l.offsetRow(m.list); off != 0 {
		t.Fatalf("offset %d after growing past the list (rows=%d listH=%d), want 0", off, len(m.list.rows), l.ListH)
	}
	if lines := viewLines(m); len(lines) != 200 {
		t.Fatalf("rendered %d lines after resize, want 200", len(lines))
	}
}

// A notice with a newline in it must not make the view taller than the
// terminal. This is a bug that shipped: the launcher assigned err.Error()
// straight into lastErr at five call sites and drew it through ansi.Truncate,
// which cuts a line to a width and leaves every newline in it -- so a
// three-line provider error rendered a 24-row terminal as 26 rows and pushed
// the footer, the one row that says how to get out, off the bottom of the
// screen.
//
// The fixture is deliberately the shape that broke it: a wrapped error chain
// with embedded newlines, of the kind an exec failure or a cloud SDK produces.
// A single-line fixture asserts nothing here.
func TestMultiLineNoticeKeepsTheViewInsideTheTerminal(t *testing.T) {
	notices := []string{
		"could not install the aws provider:\nGet \"https://releases.mondoo.com\": dial tcp: lookup failed\nno such host",
		"exit status 1\r\nprovider crashed\r\n",
		"one\n\n\nfour",
		strings.Repeat("wrapped: ", 40) + "\nand a second line",
	}
	for _, notice := range notices {
		for _, s := range termSizes {
			for _, field := range []string{"lastErr", "lastWarn"} {
				m := sized(newTestModel(), s.w, s.h)
				if field == "lastErr" {
					m.lastErr = notice
				} else {
					m.lastWarn = notice
				}
				lines := viewLines(m)
				if len(lines) != s.h {
					t.Errorf("%s %dx%d: rendered %d lines, want exactly %d",
						field, s.w, s.h, len(lines), s.h)
				}
				for i, ln := range lines {
					if w := ansi.StringWidth(ln); w > s.w {
						t.Errorf("%s %dx%d: line %d is %d cells wide, want <= %d",
							field, s.w, s.h, i, w, s.w)
					}
				}
			}
		}
	}
}

// The same rule for the other string the launcher does not write: the reason a
// value picker came back empty, which is rendered into a panel line rather than
// into the footer. A newline there makes the panel taller than the number of
// lines the layout measured it for, which is the same failure one pane over.
func TestMultiLineSourceErrorKeepsTheViewInsideTheTerminal(t *testing.T) {
	for _, s := range termSizes {
		m := selectEntry(t, sized(newTestModel(), s.w, s.h), "aws")
		m.focus = focusForm
		for _, fd := range m.detail.form.Fields() {
			if fd.Source() == "" {
				continue
			}
			setSourceErr(&m, fd.Source(), errors.New(
				"cannot read ~/.aws/config:\npermission denied\nrun aws configure"))
		}
		if lines := viewLines(m); len(lines) != s.h {
			t.Errorf("%dx%d: rendered %d lines, want exactly %d", s.w, s.h, len(lines), s.h)
		}
	}
}
