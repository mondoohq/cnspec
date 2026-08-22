// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/policy"
)

// Exporting is cheap here because the viewer never reduced anything: it was
// built from a whole *policy.ReportCollection and still holds it (see
// reportmodel.Report.Collection), and cli/reporter already knows how to write
// every output format cnspec has. So "export" is a format picker, a file name
// and a tea.Cmd -- no second rendering path, and nothing the CLI can produce
// that the viewer cannot.
//
// # Which formats are offered
//
// Four, out of the sixteen names in reporter.Formats. The picker is a question
// asked mid-read -- "give me this report as a file" -- and the answer is one of
// json, yaml, junit and sarif. Eleven rows made the reader choose between
// artifacts they would have to know cnspec's history to tell apart.
//
// What is left out, and why:
//
//   - the older shapes. json-v1, yaml-v1 and json-v2 are earlier reductions of
//     the same report, still reachable from `cnspec scan --output` for anything
//     already parsing them. A viewer opened to read one scan is not where a
//     reader picks a schema version, so each family offers its current one: json
//     is the full collection, yaml is yaml-v2.
//
//   - the four terminal renderings. compact, summary, full and report are the
//     report drawn for a screen -- which is what the CLI already prints, and
//     what this viewer is a richer replacement for. Writing one to a file was
//     also the only reason this package ever had to strip ANSI on the way out:
//     those writers take their color from colors.DefaultColorTheme, built from
//     termenv.EnvColorProfile() at init, so under a real terminal they emit SGR
//     escapes wherever their output is pointed. With none of them offered, the
//     strip has nothing to strip, and it is gone with them. Anything added back
//     here that renders for a screen has to bring it back.
//
//   - csv, which cannot be written at all. reporter.FormatCSV has no case in
//     (*Reporter).WriteReport: it falls through to the default branch and comes
//     back "unknown reporter type, don't recognize this format". The switch has
//     the CSV arm commented out, so this is a format that exists in the name
//     table and nowhere else. TestCSVIsExcludedBecauseItCannotBeWritten holds
//     that to it rather than trusting this paragraph.
//
// # Shown as, written as
//
// The name on a row is not always the reporter's name for it, because the
// reporter's names carry the history this list is dropping. `json` writes
// json-full and `yaml` writes yaml-v2; both are simply "the json one" and "the
// yaml one" to a reader who has never met a v1. Format is what reaches
// reporter.ParseConfig, Name is what the modal and the footer say, and the
// suffixes follow the shown name -- with one json on offer there is no reason
// for a .full.json.
//
// That makes `json` the row that round-trips: json-full is the whole
// *policy.ReportCollection rather than a reduction of it, so the file it writes
// is the file `cnspec report view` reopens. TestTheJSONRowRoundTrips proves it
// by reading one back through reporter.LoadCollectionFile.
//
// # Bundles
//
// junit and sarif are offered only when the collection carries a policy bundle.
// Both writers refuse a collection without one ("no policy bundle found",
// junit.go and sarif.go), which is exactly the shape of a scan where every asset
// failed before a policy could run. Rather than offer a row that is guaranteed
// to fail, exportable drops them, and such a report gets two rows instead of
// four.
//
// TestEveryOfferedFormatWrites proves the list rather than trusting any of this:
// it writes every offered format for both fixtures and reads the files back.

// exportFormat is one row of the export modal: what it is called, the reporter
// format behind it, the file it writes and the sentence that says what it is
// for.
type exportFormat struct {
	// Name is what the row is called on screen and in the footer notice.
	Name string
	// Format is the reporter format it writes, spelled the way
	// `cnspec scan --output` spells it. It is often the same string as Name and
	// deliberately not always -- see "Shown as, written as" above.
	Format string
	// Suffix is what is appended to the base name. Every offered format has a
	// distinct one, so exporting two formats of the same report never lands
	// them on the same path.
	Suffix string
	// Desc is the one-line explanation shown beside the name.
	Desc string
	// needsBundle marks a format whose writer refuses a collection with no
	// policy bundle.
	needsBundle bool
}

// exportFormats is every format the picker offers, machine-readable first: the
// reason to leave the viewer with a file is usually to feed it to something
// else, and json is what most things eat.
var exportFormats = []exportFormat{
	{Name: "json", Format: "json-full", Suffix: ".json", Desc: "everything, reopens in this viewer"},
	{Name: "yaml", Format: "yaml-v2", Suffix: ".yaml", Desc: "scores, data and errors, as yaml"},
	{Name: "junit", Format: "junit", Suffix: ".junit.xml", Desc: "one test case per check", needsBundle: true},
	{Name: "sarif", Format: "sarif", Suffix: ".sarif", Desc: "code scanning findings", needsBundle: true},
}

// exportable is the subset of exportFormats that can be written for this
// collection. A nil collection yields none: the viewer can be opened on an
// empty model (reportmodel.New(nil)), and writing four empty files is not a
// service to anybody.
func exportable(c *policy.ReportCollection) []exportFormat {
	if c == nil {
		return nil
	}
	res := make([]exportFormat, 0, len(exportFormats))
	for _, f := range exportFormats {
		if f.needsBundle && c.Bundle == nil {
			continue
		}
		res = append(res, f)
	}
	return res
}

// --- where the file goes ----------------------------------------------------

// exportFallbackName is the base name for a report that is not one asset.
const exportFallbackName = "cnspec-report"

// exportNameMax caps the base name. Asset names can be an ARN or a container
// digest, and a 200-character file name is not a name anybody can type.
const exportNameMax = 48

// exportBaseName is the file name the export is built from, without a suffix.
//
// A single-asset report is named after its asset, because that is what the user
// would call the file themselves. Anything else is named cnspec-report: the
// export is the whole collection, and naming a fifteen-asset file after the row
// the cursor happens to be on would be a lie about its contents.
func exportBaseName(r *reportmodel.Report) string {
	if r != nil && len(r.Assets) == 1 {
		if s := exportSlug(r.Assets[0].Name); s != "" {
			return s
		}
	}
	return exportFallbackName
}

// exportSlug reduces a name to lowercase letters, digits and dashes. The
// charset is the point: an asset name can hold a slash, a colon or a space, and
// the result of this is pasted straight into a path.
func exportSlug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if len(out) > exportNameMax {
		out = strings.TrimRight(out[:exportNameMax], "-")
	}
	return out
}

// --- writing ----------------------------------------------------------------

// writeExport renders the collection in one format and writes it to
// dir/base+suffix, returning the path and how many bytes landed there.
//
// The file is created with O_EXCL, so an export never replaces one that is
// already there. The alternative -- overwriting, or silently picking
// name-1.json -- trades a predictable path for a surprise, and the whole reason
// the name is derived from the report rather than typed is that the user should
// be able to say where the file will be before pressing the key. A collision is
// reported and the existing file is left exactly as it was.
func writeExport(ctx context.Context, c *policy.ReportCollection, f exportFormat, dir, base string) (string, int64, error) {
	path := filepath.Join(dir, base+f.Suffix)

	conf, err := reporter.ParseConfig(f.Format)
	if err != nil {
		return path, 0, errors.Wrapf(err, "cannot configure the %s reporter", f.Format)
	}

	// Rendered whole before the file is opened: a failure halfway through a
	// writer must not leave a truncated artifact behind under a name that says
	// it is complete.
	var buf bytes.Buffer
	if err := reporter.NewReporter(conf, false).WithOutput(&buf).WriteReport(ctx, c); err != nil {
		return path, 0, errors.Wrapf(err, "cannot render %s", f.Name)
	}

	// The bytes go to the file exactly as the writer produced them. Every format
	// offered here is a machine format, and an escape byte inside one came from
	// scanned data -- a command's captured output, say -- so stripping it would
	// corrupt the artifact rather than tidy it. See the package comment for the
	// terminal formats this used to have to strip, and no longer offers.
	out := buf.Bytes()

	fh, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return path, 0, errors.Newf("%s already exists, remove it first", filepath.Base(path))
		}
		return path, 0, errors.Wrapf(err, "cannot create %s", filepath.Base(path))
	}
	if _, err := fh.Write(out); err != nil {
		_ = fh.Close()
		_ = os.Remove(path)
		return path, 0, errors.Wrapf(err, "cannot write %s", filepath.Base(path))
	}
	if err := fh.Close(); err != nil {
		_ = os.Remove(path)
		return path, 0, errors.Wrapf(err, "cannot write %s", filepath.Base(path))
	}
	return path, int64(len(out)), nil
}

// ExportDoneMsg is the outcome of a write, on its way back to the frame.
//
// It is exported because a program embedding this viewer -- the launcher does
// -- can outlive the view: close the report while a large json-full is still
// being written and the result arrives with nobody left to show it. Such a
// program routes this to its own footer via ExportNotice.
type ExportDoneMsg struct {
	Format string
	Path   string
	Size   int64
	Err    error
}

// exportCmd is the write, as a command. Rendering a large collection is not
// instant -- json-full of a real scan is megabytes and sarif walks every
// finding -- and doing it inline would freeze the viewer mid-keystroke, so it
// happens off the event loop like any other slow thing in bubbletea.
//
// The collection is read, never modified, by every writer it is handed to,
// which is what lets the user keep browsing while the file is being written.
// TestExportDoesNotRaceTheView holds that to it.
func exportCmd(c *policy.ReportCollection, f exportFormat, base string) tea.Cmd {
	return func() tea.Msg {
		dir, err := os.Getwd()
		if err != nil {
			return ExportDoneMsg{Format: f.Name, Err: errors.Wrap(err, "cannot find the working directory")}
		}
		path, n, err := writeExport(context.Background(), c, f, dir, base)
		return ExportDoneMsg{Format: f.Name, Path: path, Size: n, Err: err}
	}
}

// ExportNotice is the one line to show when the modal is not there to say it: what was written and how big it is, or why nothing was.
func ExportNotice(msg ExportDoneMsg) string {
	if msg.Err != nil {
		return "export failed: " + tui.OneLine(msg.Err.Error())
	}
	return "wrote ./" + filepath.Base(msg.Path) + " (" + humanSize(msg.Size) + ")"
}

// humanSize renders a byte count the way a person reads one.
func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

// --- the modal --------------------------------------------------------------

// exportModal is the format picker. It is a value on Model rather than a pane:
// a pane occupies a slot in the layout and competes for focus, and a modal is
// the opposite of both -- it takes the body, and it takes every key until it is
// closed.
type exportModal struct {
	open   bool
	busy   bool
	cursor int
	// formats is fixed when the modal opens, because what can be written
	// depends on the collection (see exportable) and the collection does not
	// change while the viewer is up.
	formats []exportFormat
	// err is the last failed write, shown in the box. A failure leaves the
	// modal open on purpose: the footer notice is cleared by the next key, and
	// "your export did not happen" is not a message to show for one keystroke.
	err error
}

// exportHeadLines is the prompt and the blank line under it; exportTailLines is
// the status row, which is pinned to the bottom of the box.
const (
	exportHeadLines = 2
	exportTailLines = 1
)

// exportTitle names the box, and is also what the tests look for.
const exportTitle = "Export"

// The prompt above the list. The long form says the one thing the rest of the
// box does not -- that the path is relative to the working directory -- and the
// short one is what a box too narrow for that says instead of showing the same
// sentence with its end cut off.
const (
	exportPrompt      = "write this report to the working directory"
	exportPromptShort = "write to ./"
)

// exportCursorW is the "▸ " column in front of every row, which the unselected
// rows spend on spaces so the two kinds of row line up.
const exportCursorW = 2

// render draws the picker into rect, which is the whole body. The result is
// exactly rect.H lines of at most rect.W cells.
//
// The box is sized to its contents (exportBoxSize) and centered in the body
// rather than filling it. Four rows in a box the height of the terminal is
// mostly air with a question somewhere in it, and a modal that is visibly
// smaller than what it covers also reads as what it is -- something in front of
// the report, not a new page. The rest of the rect is left blank: the panes
// underneath are not drawn while the picker is up, because compositing a box
// over them would mean slicing styled lines in half.
func (x exportModal) render(base string, rect tui.Rect) []string {
	box := x.box(rect)
	inner := tui.InnerWidth(box.W)
	innerH := tui.InnerHeight(box.H)

	body := make([]string, 0, innerH)
	prompt := exportPrompt
	if tui.Width(prompt) > inner {
		prompt = exportPromptShort
	}
	body = append(body, tui.StyleFaint.Render(prompt), "")

	start, end := x.window(innerH)
	for i := start; i < end; i++ {
		row := exportRow(x.formats[i], inner-exportCursorW)
		if i == x.cursor {
			body = append(body, tui.Bar("▸ "+row, inner, tui.BandSelected))
			continue
		}
		body = append(body, strings.Repeat(" ", exportCursorW)+tui.StyleText.Render(row))
	}
	if more := tui.MoreRow(len(x.formats) - (end - start)); more != "" {
		body = append(body, more)
	}

	// The status sits on the last row inside the border rather than following
	// the list, so it does not move up and down the box as the list scrolls.
	for len(body) < innerH-exportTailLines {
		body = append(body, "")
	}
	body = append(body, x.status(base))

	panel := strings.Split(tui.Panel(
		tui.StyleAccent.Render(exportTitle), "", body, box.W, box.H, tui.BorderColor(true)), "\n")

	// Placed into the body rect by hand, because tui.Panel draws a box and knows
	// nothing about where it sits. Exactly rect.H lines go back, whatever the
	// box came out as: the frame's Fit would cut an overrun anyway, and a pane
	// that returns its own size is one less thing depending on it.
	pad := strings.Repeat(" ", max(box.X-rect.X, 0))
	out := make([]string, 0, rect.H)
	for i := box.Y - rect.Y; i > 0; i-- {
		out = append(out, "")
	}
	for _, ln := range panel {
		out = append(out, pad+ln)
	}
	for len(out) < rect.H {
		out = append(out, "")
	}
	return out[:rect.H]
}

// box is where the picker is drawn inside the body: as big as its contents want,
// never bigger than the body, and centered in whatever is left over.
func (x exportModal) box(rect tui.Rect) tui.Rect {
	w, h := exportBoxSize(x.formats)
	w = min(w, rect.W)
	h = min(h, rect.H)
	return tui.Rect{
		X: rect.X + (rect.W-w)/2,
		Y: rect.Y + (rect.H-h)/2,
		W: w,
		H: h,
	}
}

// exportBoxSize is the size the box wants: wide enough for the widest row it can
// draw and for the prompt above them, tall enough for one line per format plus
// the prompt, the blank under it and the status row.
//
// It is measured off the rows rather than written down, so a format with a
// longer description widens the box instead of being truncated by it.
func exportBoxSize(formats []exportFormat) (int, int) {
	inner := tui.Width(exportPrompt)
	for _, f := range formats {
		// A width nothing can exceed, so exportRow returns every column.
		if w := exportCursorW + tui.Width(exportRow(f, exportRowFullW)); w > inner {
			inner = w
		}
	}
	return inner + tui.BorderCols,
		len(formats) + exportHeadLines + exportTailLines + tui.BorderLines
}

// exportRowFullW is a width wide enough that exportRow keeps all three columns,
// which is how exportBoxSize asks for a row's full width.
const exportRowFullW = 1 << 10

// window is the slice of the list that is drawn, given the room inside the box.
// The centring is tui.Window's; what belongs here is the row budget.
//
// The "n more" marker costs a row of its own: leaving it out of this
// arithmetic is what pushes the status line off the bottom of a short box.
func (x exportModal) window(innerH int) (int, int) {
	rows := innerH - exportHeadLines - exportTailLines
	if rows < len(x.formats) && rows > 1 {
		rows--
	}
	if rows < 1 {
		rows = 1
	}
	return tui.Window(x.cursor, len(x.formats), rows)
}

// status is the bottom line of the box: what the write is doing, why it failed,
// or -- the ordinary case -- the exact path the highlighted row would write, so
// the answer to "where does it go" is on screen before the key is pressed.
func (x exportModal) status(base string) string {
	switch {
	case x.err != nil:
		return StatusStyle(reportmodel.StatusError).Render("! " + tui.OneLine(x.err.Error()))
	case x.busy:
		return tui.StyleDim.Render("writing " + x.current().Name + "…")
	default:
		return tui.StyleDim.Render("→ ./" + base + x.current().Suffix)
	}
}

// current is the highlighted format. It is never called on an empty modal --
// Model.openExport refuses to open one -- but a cursor left of range by a
// resize or a rebuild would panic in View, which takes the terminal with it.
func (x exportModal) current() exportFormat {
	if x.cursor < 0 || x.cursor >= len(x.formats) {
		return exportFormat{}
	}
	return x.formats[x.cursor]
}

// exportRow is one line of the list: the format, the file it writes and what it
// is for, in columns. The two tail columns drop off as the terminal narrows,
// rather than all three being cut to a stub by the panel's truncation.
func exportRow(f exportFormat, w int) string {
	row := tui.PadRight(f.Name, exportNameW)
	if w >= exportNameW+exportSuffixW+2 {
		row += " " + tui.PadRight(f.Suffix, exportSuffixW)
	}
	if w >= exportNameW+exportSuffixW+exportDescW+2 {
		row += " " + f.Desc
	}
	return row
}

// Column widths: the longest name, the longest suffix, and enough of a
// description to be a sentence rather than a hint.
// TestTheColumnsFitTheFormats keeps the first two matched to the list above.
const (
	exportNameW   = 5  // junit, sarif
	exportSuffixW = 10 // .junit.xml
	exportDescW   = 20
)

// exportHints are the footer bindings while the picker is open. They replace
// the focused pane's, because none of that pane's keys reach it right now.
func exportHints() []Hint {
	return []Hint{
		{Key: "↑/↓", Label: "format"},
		{Key: "enter", Label: "write"},
		{Key: "esc", Label: "cancel"},
	}
}

// --- frame integration ------------------------------------------------------

// openExport opens the picker, or explains in the footer why there is nothing
// to open it on.
func (m Model) openExport() (tea.Model, tea.Cmd) {
	formats := exportable(m.state.Report.Collection())
	if len(formats) == 0 {
		m.state.Notice = "nothing to export: this view was not opened on a report"
		return m, nil
	}
	m.export = exportModal{open: true, formats: formats}
	return m, nil
}

// exportKey drives the open picker. Nothing here reaches a pane, and nothing
// here quits: a modal that lets q end the program underneath it is a modal that
// loses the user's place in the report.
func (m Model) exportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	last := len(m.export.formats) - 1

	switch msg.String() {
	case "esc", "e", "q":
		m.export = exportModal{}
	case "up", "k", "ctrl+p":
		if m.export.cursor > 0 {
			m.export.cursor--
		}
	case "down", "j", "ctrl+n":
		if m.export.cursor < last {
			m.export.cursor++
		}
	case "home", "g":
		m.export.cursor = 0
	case "end", "G":
		m.export.cursor = last
	case "enter":
		if m.export.busy {
			return m, nil
		}
		f := m.export.current()
		if f.Name == "" {
			return m, nil
		}
		m.export.busy, m.export.err = true, nil
		return m, exportCmd(m.state.Report.Collection(), f, exportBaseName(m.state.Report))
	}
	return m, nil
}

// exportDone takes the result of a write.
//
// A failure that arrives while the box is still open is shown in it, in the
// error color, and the box stays up so the user can pick another format or read
// the reason twice. A failure that arrives after the box was closed has nowhere
// to go but the footer -- which is also where a success goes, because the file
// is written and the report is what the user wants to be looking at.
func (m Model) exportDone(msg ExportDoneMsg) (tea.Model, tea.Cmd) {
	if !m.export.open {
		m.state.Notice = ExportNotice(msg)
		return m, nil
	}
	m.export.busy = false
	if msg.Err != nil {
		m.export.err = msg.Err
		return m, nil
	}
	m.export = exportModal{}
	m.state.Notice = ExportNotice(msg)
	return m, nil
}
