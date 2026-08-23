// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
)

func loadCollection(t *testing.T, fixture string) *policy.ReportCollection {
	t.Helper()
	if fixture == fixtureUbuntu {
		collection, err := reportfixture.UbuntuScan()
		require.NoError(t, err)
		return collection
	}

	raw, err := os.ReadFile("../reporter/testdata/" + fixture + ".json")
	require.NoError(t, err)

	collection := &policy.ReportCollection{}
	require.NoError(t, json.Unmarshal(raw, collection))
	return collection
}

// --- what is offered --------------------------------------------------------

// The list is only worth anything if every row on it writes. This is the test
// that keeps exportFormats honest: it does not read the switch in
// cli_reporter.go and reason about it, it writes every offered format for both
// fixtures and reads the files back.
func TestEveryOfferedFormatWrites(t *testing.T) {
	for _, fixture := range []string{fixtureUbuntu, fixtureK8s} {
		collection := loadCollection(t, fixture)
		formats := exportable(collection)
		require.NotEmpty(t, formats)

		dir := t.TempDir()
		for _, f := range formats {
			t.Run(filepath.Base(fixture)+"/"+f.Name, func(t *testing.T) {
				path, n, err := writeExport(context.Background(), collection, f, dir, "report")
				require.NoError(t, err)
				require.Equal(t, filepath.Join(dir, "report"+f.Suffix), path)
				require.Positive(t, n, "an export with no bytes in it is not an export")

				raw, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Len(t, raw, int(n), "the reported size must be the size on disk")

				if strings.HasSuffix(f.Suffix, ".json") || f.Name == "sarif" {
					require.True(t, json.Valid(raw), "%s must be valid json", f.Name)
				}
				if f.Name == "junit" {
					require.True(t, bytes.HasPrefix(bytes.TrimSpace(raw), []byte("<")), "junit must be xml")
				}
			})
		}
	}
}

// The list is exactly four rows, and they are the four a reader would name. It
// is pinned rather than inferred: dropping the older shapes and the terminal
// renderings is the whole point of the list being this length, and a row
// creeping back in should have to say so here.
func TestFourFormatsAreOffered(t *testing.T) {
	shown := make([]string, 0, len(exportFormats))
	writes := make([]string, 0, len(exportFormats))
	for _, f := range exportFormats {
		shown = append(shown, f.Name)
		writes = append(writes, f.Format)
	}
	require.Equal(t, []string{"json", "yaml", "junit", "sarif"}, shown)
	// The names on the rows are not the reporter's names for two of them: json
	// is the full collection and yaml is the current yaml, and neither carries
	// the version in what the reader sees.
	require.Equal(t, []string{"json-full", "yaml-v2", "junit", "sarif"}, writes)

	// None of the shapes this list dropped, under either name.
	for _, gone := range []string{"json-v1", "json-v2", "yaml-v1", "compact", "summary", "full", "report", "csv"} {
		for _, f := range exportFormats {
			require.NotEqual(t, gone, f.Name, "%s is offered again", gone)
			require.NotEqual(t, gone, f.Format, "%s is written again", gone)
		}
	}
}

// Every format has its own file name. Two formats sharing one would turn a
// second export into a collision with the first, for no reason the user could
// see. Every name is also one the reporter actually knows.
func TestOfferedFormatsHaveDistinctNames(t *testing.T) {
	seen := map[string]string{}
	names := map[string]bool{}
	for _, f := range exportFormats {
		require.NotContains(t, seen, f.Suffix, "%s and %s both write %s", seen[f.Suffix], f.Name, f.Suffix)
		seen[f.Suffix] = f.Name

		require.False(t, names[f.Name], "%s is offered twice", f.Name)
		names[f.Name] = true

		// The suffix follows the shown name, not the reporter's: with one json
		// on offer there is no reason for a .full.json.
		require.True(t, strings.Contains(f.Suffix, f.Name),
			"%s writes %s, which does not name it", f.Name, f.Suffix)

		_, ok := reporter.Formats[f.Format]
		require.True(t, ok, "%s is not a format cli/reporter knows", f.Format)
	}
}

// The columns are as wide as the widest thing that goes in them, and no wider:
// a name column short of the longest name truncates a row, and one longer pads
// every row with air the box then has to be wide enough for.
func TestTheColumnsFitTheFormats(t *testing.T) {
	name, suffix := 0, 0
	for _, f := range exportFormats {
		name = max(name, tui.Width(f.Name))
		suffix = max(suffix, tui.Width(f.Suffix))
	}
	require.Equal(t, name, exportNameW)
	require.Equal(t, suffix, exportSuffixW)
}

// csv is in reporter.Formats and cannot be written: FormatCSV has no arm in
// WriteReport's switch, so it lands in the default branch. The exclusion is not
// a matter of taste, and this is the assertion that says so out loud -- if the
// CSV arm is ever implemented, this fails and the format should be offered.
func TestCSVIsExcludedBecauseItCannotBeWritten(t *testing.T) {
	for _, f := range exportFormats {
		require.NotEqual(t, "csv", f.Name)
		require.NotEqual(t, "csv", f.Format)
	}

	conf, err := reporter.ParseConfig("csv")
	require.NoError(t, err)
	err = reporter.NewReporter(conf, false).
		WithOutput(&bytes.Buffer{}).
		WriteReport(context.Background(), loadCollection(t, fixtureUbuntu))
	require.ErrorContains(t, err, "unknown reporter type")
}

// junit and sarif need a policy bundle, and the k8s fixture -- fifteen assets
// that never scanned -- has none. Offering them for such a report would be
// offering a row that is guaranteed to fail, so both are dropped; and to be
// sure that is a real limitation rather than a superstition, the test also
// writes them and watches them refuse.
func TestBundlelessReportDropsTheBundleFormats(t *testing.T) {
	collection := loadCollection(t, fixtureK8s)
	require.Nil(t, collection.Bundle, "the k8s fixture is the no-bundle case")

	offered := []string{}
	for _, f := range exportable(collection) {
		offered = append(offered, f.Name)
	}
	// Two rows, not four, and they are the two that do not need a bundle.
	require.Equal(t, []string{"json", "yaml"}, offered)

	dir := t.TempDir()
	for _, f := range exportFormats {
		if !f.needsBundle {
			continue
		}
		_, _, err := writeExport(context.Background(), collection, f, dir, "report")
		require.ErrorContains(t, err, "no policy bundle found", "%s", f.Name)
		require.NoFileExists(t, filepath.Join(dir, "report"+f.Suffix), "a failed render must not leave a file")
	}

	// The report that does have a bundle gets all four.
	require.Len(t, exportable(loadCollection(t, fixtureUbuntu)), len(exportFormats))
}

// A viewer opened on nothing has nothing to export, and says so rather than
// writing four empty files.
func TestNothingToExport(t *testing.T) {
	require.Nil(t, exportable(nil))

	m := sized(NewModel(reportmodel.New(nil)), 100, 30)
	m, _ = press(m, "e")
	require.False(t, m.export.open)
	require.Contains(t, ansi.Strip(m.View()), "nothing to export")
}

// --- the bytes --------------------------------------------------------------

// Every offered format is a machine format, and the bytes reach the file exactly
// as the writer produced them. An escape byte inside a collection came from
// scanned data -- a command's captured output -- and an exporter that tidied it
// away would be corrupting the artifact, not cleaning it.
//
// This is also what pins the removal of the ANSI strip that used to run here: it
// existed for compact, summary, full and report, which color their output from a
// profile decided at init from the terminal, and none of them is offered any
// more.
func TestExportedBytesAreWhatTheScanSaw(t *testing.T) {
	collection := loadCollection(t, fixtureUbuntu)
	for _, a := range collection.Assets {
		a.Name = "\x1b[31mubuntu\x1b[0m"
	}

	dir := t.TempDir()
	for _, f := range exportable(collection) {
		_, _, err := writeExport(context.Background(), collection, f, dir, "report")
		require.NoError(t, err, f.Name)
	}

	// json-encoded, but still there: the byte the scan saw survived rather than
	// being tidied away by an exporter that decided it knew better.
	raw, err := os.ReadFile(filepath.Join(dir, "report"+formatNamed(t, "json").Suffix))
	require.NoError(t, err)
	require.Contains(t, string(raw), "\\u001b[31mubuntu", "json keeps what the scan saw")

	// And nothing on the list renders for a screen, which is what makes that
	// safe for all four of them rather than for some.
	for _, f := range exportFormats {
		require.NotContains(t, f.Suffix, ".txt", "%s renders for a terminal", f.Name)
	}
}

// json is the row that round-trips, which is what makes "export it, reopen it" a
// thing a user can do. It is `json` on screen and json-full on the wire: the
// whole *policy.ReportCollection rather than a reduction of it, and a reduction
// is not something reporter.LoadCollectionFile can read back.
func TestTheJSONRowRoundTrips(t *testing.T) {
	f := formatNamed(t, "json")
	require.Equal(t, "json-full", f.Format, "the reduced json would not reopen")
	require.Equal(t, ".json", f.Suffix)

	dir := t.TempDir()
	path, _, err := writeExport(context.Background(), loadCollection(t, fixtureUbuntu), f, dir, "report")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "report.json"), path)

	reloaded, err := reporter.LoadCollectionFile(path)
	require.NoError(t, err)

	again := reportmodel.New(reloaded)
	require.Len(t, again.Assets, 1)
	require.Equal(t, 24, again.Assets[0].Counts.Total)
	require.Equal(t, 2, again.Assets[0].Counts.Failed)
	require.Equal(t, 4, again.Assets[0].Counts.Errored)
	require.NotNil(t, reloaded.Bundle, "the bundle survives, so the reopened report still offers all four")
}

// And the reduced json is not what that row writes. Under a name that suggests
// it is the same artifact it is a different one, so the mistake this renaming
// could have made is worth an assertion of its own: reporter refuses a reduced
// report by name when asked to load it back.
func TestTheReducedJSONWouldNotReopen(t *testing.T) {
	dir := t.TempDir()
	reduced := exportFormat{Name: "json-v2", Format: "json-v2", Suffix: ".v2.json"}
	path, _, err := writeExport(context.Background(), loadCollection(t, fixtureUbuntu), reduced, dir, "report")
	require.NoError(t, err)

	_, err = reporter.LoadCollectionFile(path)
	require.ErrorContains(t, err, "this is a reduced report")
}

// --- where the file goes ----------------------------------------------------

func TestExportBaseName(t *testing.T) {
	// One asset: the file is named after it, in a charset that is safe to
	// paste into a path.
	require.Equal(t, "ubuntu-24-04", exportBaseName(loadReport(t, fixtureUbuntu)))
	// Fifteen: the export is the whole collection, so naming it after one of
	// them would be a lie about what is in the file.
	require.Equal(t, exportFallbackName, exportBaseName(loadReport(t, fixtureK8s)))
	require.Equal(t, exportFallbackName, exportBaseName(reportmodel.New(nil)))

	for _, tc := range []struct{ in, want string }{
		{"ubuntu:24.04", "ubuntu-24-04"},
		{"  ", ""},
		{"../../etc/passwd", "etc-passwd"},
		{"arn:aws:ec2:us-east-1:123:instance/i-0abc", "arn-aws-ec2-us-east-1-123-instance-i-0abc"},
		{strings.Repeat("x", 200), strings.Repeat("x", exportNameMax)},
		{"K8s Cluster (prod)", "k8s-cluster-prod"},
	} {
		got := exportSlug(tc.in)
		require.Equal(t, tc.want, got, "slug(%q)", tc.in)
		require.NotContains(t, got, string(filepath.Separator))
		require.LessOrEqual(t, len(got), exportNameMax)
	}
}

// An export never replaces a file that is already there. The path is derived
// rather than typed, so the user did not get a chance to say "yes, that one",
// and quietly overwriting the artifact they exported a minute ago is not a
// recoverable mistake.
func TestExportRefusesToOverwrite(t *testing.T) {
	dir := t.TempDir()
	f := formatNamed(t, "json")
	existing := filepath.Join(dir, "report"+f.Suffix)
	require.NoError(t, os.WriteFile(existing, []byte("mine"), 0o600))

	_, _, err := writeExport(context.Background(), loadCollection(t, fixtureUbuntu), f, dir, "report")
	require.ErrorContains(t, err, "already exists")

	raw, err := os.ReadFile(existing)
	require.NoError(t, err)
	require.Equal(t, "mine", string(raw), "the file that was there must be untouched")
}

// --- the modal --------------------------------------------------------------

// e opens the picker, esc closes it, and while it is open no key reaches a pane
// or the frame: the tree does not move, the help does not toggle, and q does
// not quit the program out from under it.
func TestExportModalTakesEveryKey(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureK8s)), 120, 40)
	before := m.state.Sel.Asset

	m, _ = press(m, "e")
	require.True(t, m.export.open)
	require.Len(t, m.export.formats, len(exportFormats)-2, "no bundle, so junit and sarif are gone")

	m, _ = press(m, "down")
	require.Equal(t, 1, m.export.cursor)
	require.Equal(t, before, m.state.Sel.Asset, "the tree must not have moved behind the modal")

	m, _ = press(m, "?")
	require.False(t, m.showHelp, "the frame's own bindings do not fire either")

	m, cmd := press(m, "q")
	require.Nil(t, cmd, "q closes the modal rather than quitting")
	require.False(t, m.export.open)

	// And the cursor survives nothing: reopening starts fresh.
	m, _ = press(m, "e")
	require.Zero(t, m.export.cursor)
	m, _ = press(m, "esc")
	require.False(t, m.export.open)
}

// G and g reach the ends of the list, and neither runs off it.
func TestExportModalCursorBounds(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	m, _ = press(m, "e")

	m, _ = press(m, "up")
	require.Zero(t, m.export.cursor, "up at the top stays at the top")

	m, _ = press(m, "G")
	require.Equal(t, len(m.export.formats)-1, m.export.cursor)
	m, _ = press(m, "down")
	require.Equal(t, len(m.export.formats)-1, m.export.cursor, "down at the end stays at the end")

	m, _ = press(m, "g")
	require.Zero(t, m.export.cursor)
}

// The mouse does nothing while the picker is up. The panes underneath still
// render and still publish their zones, so without the guard a click would
// select a row nobody can see.
func TestExportModalSwallowsTheMouse(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureK8s)), 120, 40)
	f := m.build()
	require.NotEmpty(t, f.zones)
	target := f.zones[len(f.zones)-1]

	m, _ = press(m, "e")
	rev := m.state.SelectionRev
	nm, _ := m.Update(tea.MouseMsg{
		X: target.Rect.X, Y: target.Rect.Y,
		Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	require.Equal(t, rev, nm.(Model).state.SelectionRev)
}

// The whole point of the modal is that the answer to "where does it go" is on
// screen before the key is pressed.
func TestExportModalShowsTheTarget(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	m, _ = press(m, "e")
	out := ansi.Strip(m.View())

	require.Contains(t, out, exportTitle)
	require.Contains(t, out, "json")
	require.Contains(t, out, "→ ./ubuntu-24-04.json")
	// The footer stops advertising keys that cannot fire.
	require.Contains(t, out, "cancel")
	require.NotContains(t, out, "next pane")
}

// The binding is only useful if it is discoverable, and the frame's hint line
// is where the viewer publishes its own keys.
func TestExportIsAdvertised(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 200, 40)
	require.Contains(t, ansi.Strip(m.View()), "e export")

	m, _ = press(m, "?")
	require.Contains(t, ansi.Strip(m.View()), "e export report")
}

// e is free: no pane switches on it. The one place it means something else is
// the header's search field, which consumes every rune before the frame is
// reached -- typing "e" into a search must type an e, not open a modal.
func TestExportKeyDoesNotFightTheSearch(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)

	m, _ = press(m, "/")
	require.Equal(t, PaneHeader, m.state.Focus)
	m, _ = press(m, "e")
	require.False(t, m.export.open, "the search field ate it")
	require.Contains(t, ansi.Strip(m.View()), "e")

	// Out of the search, it opens.
	m, _ = press(m, "esc")
	m, _ = press(m, "e")
	require.True(t, m.export.open)
}

// The geometry tests in layout_test.go assert the view is exactly the terminal
// size at ten sizes. The modal is drawn into the same body those tests measure,
// so it has to hold the same invariant -- including at 20x3, where the box is
// taller than the room it has.
func TestExportModalKeepsTheViewExact(t *testing.T) {
	for _, fixture := range []string{fixtureUbuntu, fixtureK8s} {
		report := loadReport(t, fixture)
		for _, s := range termSizes {
			for _, state := range []struct {
				name string
				mut  func(Model) Model
			}{
				{"open", func(m Model) Model { return m }},
				{"busy", func(m Model) Model { m.export.busy = true; return m }},
				{"failed", func(m Model) Model {
					m.export.err = errorForTest("report.json already exists, remove it first")
					return m
				}},
				// An error is the one string in the box nobody in this package
				// wrote, and a newline in it would be a row the geometry cannot
				// see.
				{"failed-multiline", func(m Model) Model {
					m.export.err = errorForTest("cannot render yaml:\r\n  line 3:\n\ttab here")
					return m
				}},
				{"end", func(m Model) Model { m.export.cursor = len(m.export.formats) - 1; return m }},
			} {
				m := sized(NewModel(report), s.w, s.h)
				m, _ = press(m, "e")
				m = state.mut(m)

				lines := viewLines(m)
				if len(lines) != s.h {
					t.Errorf("%s %s %dx%d: rendered %d lines, want exactly %d",
						fixture, state.name, s.w, s.h, len(lines), s.h)
				}
				for i, ln := range lines {
					if w := ansi.StringWidth(ln); w > s.w {
						t.Errorf("%s %s %dx%d: line %d is %d cells wide, want <= %d",
							fixture, state.name, s.w, s.h, i, w, s.w)
					}
					if strings.Contains(ln, "\n") {
						t.Errorf("%s %s %dx%d: line %d holds a newline", fixture, state.name, s.w, s.h, i)
					}
				}
			}
		}
	}
}

// The box is the size of what is in it, not the size of what it covers. Four
// rows in a box the height of the terminal is mostly air with a question
// somewhere in it.
func TestTheModalFitsItsContents(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)
	m, _ = press(m, "e")
	require.Len(t, m.export.formats, 4)

	body := tui.Rect{X: 0, Y: m.layout().BodyTop, W: 120, H: m.layout().BodyH}
	box := m.export.box(body)

	// One line per format, plus the prompt, the blank under it, the status row
	// and the border: nine, not the thirty-eight the body offers.
	require.Equal(t, 4+exportHeadLines+exportTailLines+tui.BorderLines, box.H)
	require.Less(t, box.H, body.H)
	require.Less(t, box.W, body.W)

	// Wide enough for the longest row without truncating it, and no wider.
	widest := 0
	for _, f := range m.export.formats {
		widest = max(widest, exportCursorW+tui.Width(exportRow(f, exportRowFullW)))
	}
	require.Equal(t, widest+tui.BorderCols, box.W)

	// Centered, so it does not read as a pane that lost its border.
	require.Equal(t, (body.W-box.W)/2, box.X)
	require.Equal(t, body.Y+(body.H-box.H)/2, box.Y)

	// And it shrinks again for a report that offers only two rows.
	k8s := sized(NewModel(loadReport(t, fixtureK8s)), 120, 40)
	k8s, _ = press(k8s, "e")
	require.Len(t, k8s.export.formats, 2)
	require.Equal(t, box.H-2, k8s.export.box(body).H)
}

// Whatever the box comes out as, render hands back exactly the body it was given
// and no line of it is wider. This is the same invariant the two real panes hold,
// asserted at the modal's own seam rather than only through the frame.
func TestTheModalFillsTheBodyExactly(t *testing.T) {
	for _, fixture := range []string{fixtureUbuntu, fixtureK8s} {
		m := sized(NewModel(loadReport(t, fixture)), 120, 40)
		m, _ = press(m, "e")

		for _, s := range termSizes {
			rect := tui.Rect{X: 0, Y: 1, W: s.w, H: max(s.h-2, 1)}
			lines := m.export.render("cnspec-report", rect)
			require.Len(t, lines, rect.H, "%s %dx%d", fixture, s.w, s.h)
			for i, ln := range lines {
				require.LessOrEqual(t, tui.Width(ln), rect.W, "%s %dx%d line %d", fixture, s.w, s.h, i)
				require.NotContains(t, ln, "\n", "%s %dx%d line %d", fixture, s.w, s.h, i)
			}
			box := m.export.box(rect)
			require.LessOrEqual(t, box.W, rect.W)
			require.LessOrEqual(t, box.H, rect.H)
		}
	}
}

// A box too narrow for the long prompt says the short one rather than showing
// the long one with its end cut off. The destination is still on screen: the
// status row under the list carries the whole path.
func TestTheNarrowModalKeepsItsPrompt(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 40, 10)
	m, _ = press(m, "e")
	out := ansi.Strip(m.View())

	require.Contains(t, out, exportPromptShort)
	require.NotContains(t, out, exportPrompt)
	require.Contains(t, out, "→ ./ubuntu-24-04.json")
	// And the list is still a list: the rows that do not fit are counted rather
	// than dropped silently.
	require.Contains(t, out, "json")
	require.Contains(t, out, "more")
}

// A narrow terminal drops the columns it cannot fit rather than showing three
// of them cut to stubs.
func TestExportRowNarrows(t *testing.T) {
	f := formatNamed(t, "json")
	require.Contains(t, exportRow(f, 80), f.Desc)
	require.Contains(t, exportRow(f, 30), f.Suffix)
	require.NotContains(t, exportRow(f, 30), f.Desc)
	require.NotContains(t, exportRow(f, 12), f.Suffix)
	require.Contains(t, exportRow(f, 12), f.Name, "the name is the last thing to go")
}

// --- the round trip through the event loop ----------------------------------

// enter writes the file, and the footer says what landed where.
func TestExportEndToEnd(t *testing.T) {
	// The fixture path is relative to the package, so it is read before the
	// working directory moves to the one the export writes into.
	report := loadReport(t, fixtureUbuntu)
	dir := t.TempDir()
	t.Chdir(dir)

	m := sized(NewModel(report), 120, 40)
	m, _ = press(m, "e")
	m, _ = press(m, "down") // json -> yaml
	require.Equal(t, "yaml", m.export.current().Name)

	m, cmd := press(m, "enter")
	require.True(t, m.export.busy)
	require.NotNil(t, cmd, "the write is a command, not something the event loop does inline")
	// The row's own name, not the reporter's yaml-v2: the modal and the footer
	// speak the same four words.
	require.Contains(t, ansi.Strip(m.View()), "writing yaml…")

	msg := cmd()
	nm, _ := m.Update(msg)
	m = nm.(Model)

	require.False(t, m.export.open, "a success closes the picker")
	out := ansi.Strip(m.View())
	require.Contains(t, out, "wrote ./ubuntu-24-04.yaml")
	require.FileExists(t, filepath.Join(dir, "ubuntu-24-04.yaml"))

	// The next key dismisses the notice, which is what a one-shot notice is.
	m, _ = press(m, "down")
	require.NotContains(t, ansi.Strip(m.View()), "wrote ./")
}

// A write that fails has to say so on the screen. It says it in the box, which
// stays open: the footer notice is cleared by the very next key, and "your
// export did not happen" is not a message to show for one keystroke.
func TestFailedExportIsVisible(t *testing.T) {
	report := loadReport(t, fixtureUbuntu)
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ubuntu-24-04.json"), []byte("mine"), 0o600))

	m := sized(NewModel(report), 120, 40)
	m, _ = press(m, "e")
	m, cmd := press(m, "enter")

	nm, _ := m.Update(cmd())
	m = nm.(Model)

	require.True(t, m.export.open, "a failure keeps the box up")
	require.False(t, m.export.busy)
	require.Error(t, m.export.err)
	require.Contains(t, ansi.Strip(m.View()), "! ubuntu-24-04.json already exists")

	// Picking another format after the refusal works, and clears the message.
	m, _ = press(m, "down")
	m, cmd = press(m, "enter")
	require.NoError(t, cmd().(ExportDoneMsg).Err)
}

// A result that arrives after the box was closed still has to be heard, so it
// goes to the footer.
func TestExportResultAfterClosingLandsInTheFooter(t *testing.T) {
	m := sized(NewModel(loadReport(t, fixtureUbuntu)), 120, 40)

	nm, _ := m.Update(ExportDoneMsg{Format: "sarif", Err: errorForTest("disk full")})
	require.Contains(t, ansi.Strip(nm.(Model).View()), "export failed: disk full")

	// A multi-line error is flattened rather than allowed to add a row the
	// geometry cannot see.
	nm, _ = m.Update(ExportDoneMsg{Format: "yaml", Err: errorForTest("cannot render:\n  line 3")})
	lines := viewLines(nm.(Model))
	require.Len(t, lines, 40)
	require.Contains(t, ansi.Strip(lines[39]), "export failed: cannot render: line 3")

	nm, _ = m.Update(ExportDoneMsg{Format: "sarif", Path: "/tmp/x.sarif", Size: 2048})
	require.Contains(t, ansi.Strip(nm.(Model).View()), "wrote ./x.sarif (2.0 KB)")
}

// The write runs off the event loop while the viewer keeps drawing the same
// collection. Every writer only reads it -- this is the test that holds them to
// that, and it is the reason the export does not have to clone megabytes first.
func TestExportDoesNotRaceTheView(t *testing.T) {
	report := loadReport(t, fixtureUbuntu)
	names := []string{"json", "yaml", "junit", "sarif"}
	cmds := make([]tea.Cmd, len(names))
	for i, name := range names {
		cmds[i] = exportCmd(report.Collection(), formatNamed(t, name), name)
	}

	dir := t.TempDir()
	t.Chdir(dir)
	m := sized(NewModel(report), 120, 40)

	var wg sync.WaitGroup
	results := make([]tea.Msg, len(cmds))
	for i, cmd := range cmds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = cmd()
		}()
	}
	for i := 0; i < 200; i++ {
		_ = m.View()
	}
	wg.Wait()

	for i, res := range results {
		msg, ok := res.(ExportDoneMsg)
		require.True(t, ok)
		require.NoError(t, msg.Err, "%s", names[i])
	}
}

func TestHumanSize(t *testing.T) {
	for _, tc := range []struct {
		n    int64
		want string
	}{
		{0, "0 B"}, {999, "999 B"}, {1024, "1.0 KB"}, {2560, "2.5 KB"},
		{1024 * 1024, "1.0 MB"}, {2464164, "2.4 MB"}, {3 * 1024 * 1024 * 1024, "3.0 GB"},
	} {
		require.Equal(t, tc.want, humanSize(tc.n))
	}
}

// formatNamed is the offered format with this name, so a test can name the one
// it means rather than index into the list.
func formatNamed(t *testing.T, name string) exportFormat {
	t.Helper()
	for _, f := range exportFormats {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no offered format named %q", name)
	return exportFormat{}
}

// errorForTest is a plain error; the export path only ever renders err.Error().
type errorForTest string

func (e errorForTest) Error() string { return string(e) }
