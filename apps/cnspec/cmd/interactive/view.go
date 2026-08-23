// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"go.mondoo.com/cnspec/cli/tui"
)

// What is left of the launcher's palette once cli/tui took the rest.
//
// The seven chrome colors, the six text styles and the two selection bands were
// declared here and, identically, in the report viewer; they come from tui now,
// so the two programs cannot drift apart by one of them being edited. Install
// state -- green, amber, red -- was picked by hand here and missed the rating
// palette on two of its three colors; it resolves through tui.Ratings now, like
// every other colored fact in cnspec.
//
// These two are what remains, and they are the launcher's own: nothing else in
// the repo draws them. Neither carries a margin, because a margin adds a line
// that layout.go cannot account for and the panels are sized in exact lines.
var (
	// styleSpark is the ✦ in front of the wordmark, which is the one place the
	// header spends the command colour on something that is not a command.
	styleSpark = lipgloss.NewStyle().Foreground(tui.ColCyan).Bold(true)

	// bandCommand is the command line at the foot of the detail panel, drawn as
	// a full-width bar so the thing the whole screen is building toward reads as
	// a command rather than as one more row.
	bandCommand = lipgloss.NewStyle().Foreground(tui.ColCyan).Background(lipgloss.Color("236"))
)

// categoryIcon gives each category a glyph.
var categoryIcon = map[string]string{
	catCloud:     "☁",
	catIdentity:  "◈",
	catSaaS:      "⬡",
	catAI:        "✦",
	catContainer: "◈",
	catHosts:     "▚",
	catDatabase:  "⛁",
	catNetwork:   "⟟",
	catIaC:       "❏",
	catDev:       "⚙",
	catOther:     "•",
}

// cursorMark prefixes the row under the cursor. It is a dot rather than an
// arrow because an arrow promises the row will open into something, and none of
// these rows do -- the "enter ▸ scan" hints keep their arrow, where the glyph
// means "leads to" rather than "is selected".
const cursorMark = "● "

// The launch affordance says what enter will do, because enter does two things:
// it runs when the command is complete and opens the fields when it is not.
// Both labels are the same width so the layout's clickable zone is fixed.
const (
	runButtonScan    = " enter ▸ scan    "
	runButtonOptions = " enter ▸ options "
)

func (m Model) runButton() string {
	if m.readyToRun() {
		return runButtonScan
	}
	return runButtonOptions
}

func (m Model) View() string {
	// The report viewer is a whole tea.Model, so it draws its own screen --
	// header, panes and footer -- rather than being framed by this one. See
	// viewer.go for how the user gets back.
	if m.phase == phaseViewing {
		return m.viewer.view()
	}

	l := computeLayout(m)

	// A scan in the background is invisible without a screen of its own: the
	// child has no terminal, so nothing it would normally print arrives here.
	if m.phase == phaseScanning {
		return m.viewHeader(l) + "\n" +
			clipLines(m.scan.view(l, m.spinner.View()), l.BodyH) + "\n" + m.scan.viewFooter(l)
	}

	// Authoring takes the body outright: it is a different verb from the one
	// the two panes behind it describe, and its keys mean what it says they do.
	if m.phase == phaseAuthoring {
		return m.viewHeader(l) + "\n" + clipLines(m.viewAuthor(l), l.BodyH) + "\n" + m.viewFooter(l)
	}

	// An open picker takes over the body: nothing moves behind it, and the
	// arrow keys mean one thing while it is up.
	// The reporting chooser takes the body the way a picker does: while it is
	// up, nothing behind it moves and its keys mean what it says they do.
	if m.upstream.modal.open {
		return m.viewHeader(l) + "\n" + clipLines(m.upstream.viewModal(l), l.BodyH) + "\n" + m.viewFooter(l)
	}
	// The export box takes the body the same way, and is checked before the
	// pickers because a picker cannot be opened from behind it.
	if m.export.open {
		return m.viewHeader(l) + "\n" + clipLines(m.viewExport(l), l.BodyH) + "\n" + m.viewFooter(l)
	}
	if m.picker.modal.open && m.picker.modal.valid(m.detail.form.Fields()) {
		return m.viewHeader(l) + "\n" + clipLines(m.viewModal(l), l.BodyH) + "\n" + m.viewFooter(l)
	}

	var body string
	switch {
	case l.TwoPane:
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			m.list.panel(l, m.focus == focusList), " ", m.detailPanel(l))
	case m.focus == focusList:
		body = m.list.panel(l, m.focus == focusList)
	default:
		body = m.detailPanel(l)
	}

	// Defensive clip: the panels are already sized in exact lines, but the
	// footer must stay on the last row even if one of them miscounts.
	body = clipLines(body, l.BodyH)
	return m.viewHeader(l) + "\n" + body + "\n" + m.viewFooter(l)
}

func clipLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// headerRight is the right-hand side of the header, built once so the renderer
// and the hit-tester cannot disagree about where the badge is. The badge leads
// it, so the badge starts exactly where this string starts.
func (m Model) headerRight() string {
	return m.upstream.badge() + tui.StyleFaint.Render(tui.HintSep) +
		tui.StyleDim.Render(fmt.Sprintf("%d connectors", m.list.count())) + " "
}

func (m Model) headerLeft() string {
	return " " + styleSpark.Render("✦") + " " + tui.StyleAccent.Render("cnspec") + "  " +
		tui.StyleFaint.Render("AI-native security for your entire stack")
}

// headerRightX is the column headerRight starts at, or -1 when the header is
// too narrow to draw it at all. computeLayout registers the badge's click zone
// from this, so a click cannot land on a badge that was never drawn.
//
// It and viewHeader ask tui the same question about the same two strings, which
// is the only reason the answer can be trusted: the renderer and the hit-tester
// each had their own copy of this arithmetic, in a package that also carried two
// more copies of it.
func (m Model) headerRightX(width int) int {
	return tui.BandRightX(m.headerLeft(), m.headerRight(), width)
}

func (m Model) viewHeader(l layout) string {
	return tui.Band(m.headerLeft(), m.headerRight(), l.Width)
}

// --- left panel -------------------------------------------------------------

// panel draws the connector list. It is a method on the list rather than on the
// Model because focus is the only thing outside the list it needs, and that is
// one bool rather than a whole launcher.
func (s listState) panel(l layout, focused bool) string {
	iw := tui.InnerWidth(l.LeftW)

	var content []string

	s.search.Width = iw - 3
	content = append(content, ansi.Truncate(s.search.View(), iw, ""))
	content = append(content, "")

	if len(s.selectable) == 0 {
		content = append(content, tui.StyleFaint.Render("no match — try a different term"))
	} else {
		off := l.offsetRow(s)
		end := off + l.ListH
		if end > len(s.rows) {
			end = len(s.rows)
		}
		for i := off; i < end; i++ {
			content = append(content, s.viewRow(s.rows[i], iw, focused))
		}
	}

	title := tui.StyleAccent.Render("Connectors")
	if !focused {
		title = tui.StyleDim.Render("Connectors")
	}
	status := tui.StyleFaint.Render(s.scrollStatus(l.ListH))
	return tui.Panel(title, status, content, l.LeftW, l.BodyH, tui.BorderColor(focused))
}

// scrollStatus reports the cursor's position in the list, so a long list says
// where you are rather than just running off the edge.
func (s listState) scrollStatus(height int) string {
	if len(s.selectable) == 0 {
		return "0"
	}
	if len(s.rows) <= height {
		return fmt.Sprintf("%d", len(s.selectable))
	}
	return fmt.Sprintf("%d/%d", s.cursor+1, len(s.selectable))
}

func (s listState) viewRow(r row, w int, focused bool) string {
	switch r.kind {
	case rowBlank:
		return ""
	case rowHeader:
		icon := categoryIcon[r.text]
		label := strings.ToUpper(r.text)
		head := tui.StyleFaint.Render(icon + "  " + label + " ")
		rule := w - tui.Width(head)
		if rule > 0 {
			head += tui.StyleFaint.Render(strings.Repeat("─", rule))
		}
		return head
	}

	e := s.filtered[r.idx]
	selected := r.sel == s.cursor

	// The install marker gets a reserved column whether or not it is drawn, so
	// it lines up down the panel instead of trailing each description.
	mark := " "
	if e.Installed {
		mark = "●"
	}

	const cursorW, markW = 2, 2
	avail := w - cursorW - markW
	nameW := 14
	if nameW > avail-6 {
		nameW = avail - 6
	}
	if nameW < 4 {
		nameW = 4
	}
	descW := max(avail-nameW-1, 0)

	name := tui.PadRight(tui.Truncate(e.Name, nameW), nameW)
	desc := tui.PadRight(tui.Truncate(e.Summary(), descW), descW)

	// A selected row is one band, so it is built unstyled and colored whole:
	// per-part colors would punch holes in the background.
	if selected {
		style := tui.BandSelected
		if !focused {
			style = tui.BandInactive
		}
		return tui.Bar(cursorMark+name+" "+desc+" "+mark, w, style)
	}

	return "  " + tui.StyleText.Render(name) + " " + tui.StyleFaint.Render(desc) + " " + tui.StyleGood.Render(mark)
}

// --- right panel ------------------------------------------------------------

func (m Model) detailPanel(l layout) string {
	c, ok := m.list.current()
	if !ok {
		return tui.Panel(tui.StyleDim.Render("Details"), "", []string{tui.StyleFaint.Render("Nothing selected.")},
			l.RightW, l.BodyH, tui.BorderColor(false))
	}
	return m.connectorPanel(c, l)
}

func (m Model) connectorPanel(c Connector, l layout) string {
	focused := m.focus != focusList
	iw := tui.InnerWidth(l.RightW)

	status := tui.Pill("installed", tui.ColInk, tui.ColGood)
	switch {
	case m.list.downloading(c):
		status = tui.Pill("installing…", tui.ColInk, tui.ColCyan)
	case !c.Installed:
		status = tui.Pill("not installed", tui.ColInk, tui.ColWarn)
	}

	plan, command := splitPlan(m.detailPlan())
	off := l.detailOffsetOf(m, len(plan))
	visible := tui.InnerHeight(l.BodyH) - 1 // the pinned command bar

	content := make([]string, 0, visible+1)
	for i := off; i < len(plan) && len(content) < visible; i++ {
		content = append(content, m.renderDetailItem(plan[i], c, iw))
	}
	// Pad so the command bar lands on the pane's last line every time.
	for len(content) < visible {
		content = append(content, "")
	}
	content = append(content, m.renderDetailItem(command, c, iw))

	title := tui.StyleAccent.Render(c.Name)
	if !focused {
		title = tui.StyleDim.Render(c.Name)
	}
	if off > 0 || off+visible < len(plan) {
		title += tui.StyleFaint.Render(scrollHint(off, visible, len(plan)))
	}
	return tui.Panel(title, status, content, l.RightW, l.BodyH, tui.BorderColor(focused))
}

func scrollHint(off, visible, total int) string {
	if total <= visible {
		return ""
	}
	pct := 100 * off / (total - visible)
	return fmt.Sprintf("  ↕%d%%", pct)
}

func (m Model) renderDetailItem(item detailItem, c Connector, w int) string {
	switch item.kind {
	case diBlank:
		return ""
	case diLabel:
		return tui.StyleLabel.Render(item.text)
	case diText:
		return tui.StyleDim.Render(tui.Truncate(item.text, w))

	case diField:
		return m.renderField(item.idx, w)

	case diMore:
		label := fmt.Sprintf("⋯ %d more options", item.idx)
		if m.detail.onMore() && m.focus == focusForm {
			return tui.Bar(cursorMark+label, w, tui.BandSelected)
		}
		return "  " + tui.StyleFaint.Render(label)

	case diButton:
		// Scanning is an explicit act, so it has a button of its own rather
		// than being what enter happens to do at the end of a form.
		label := "Scan"
		switch {
		case m.launching.preparing:
			// The credential now reaches the keychain from a command rather
			// than from Update, which is what stops a locked keychain freezing
			// the screen -- but it also means enter no longer does anything
			// visible on its own. Without this the button reads as a swallowed
			// keypress for as long as the OS dialog is up.
			label = "Preparing…"
		case !m.readyToRun():
			label = "Scan (fill in what is marked *)"
		}
		return "  " + xbutton(label, m.detail.onButton() && m.focus == focusForm)

	case diCommand:
		return tui.Bar(tui.Truncate("$ "+m.commandPreview(c), w), w, bandCommand)
	}
	return ""
}

// renderField draws one input row: label, current value, and -- when the field
// has focus -- whatever help it can offer about what may go in it.
func (m Model) renderField(idx, w int) string {
	fd := m.detail.form.Fields()[idx]
	focused := m.focus == focusForm && m.detail.form.Cursor() == idx

	label := fd.Label
	if fd.Required {
		label += "*"
	}
	labelW := 18
	if labelW > w/2 {
		labelW = w / 2
	}

	var value string
	switch fd.Kind {
	case fieldBool:
		value = "[ ] no"
		if fd.On() {
			value = "[x] yes"
		}
	case fieldMultiChoice:
		if picked := fd.Selected(); len(picked) > 0 {
			value = strings.Join(picked, ", ")
		} else {
			value = tui.StyleFaint.Render(fmt.Sprintf("%d to choose from", len(fd.Options)))
		}
		value += tui.StyleFaint.Render("  ▾")

	case fieldCredentialState:
		// A readout, not an input. It says whether a credential is present and
		// which variable supplied it, and what it holds is that variable's
		// name -- never the credential, which the launcher never reads.
		//
		// The mark is the same one the connector list uses for an installed
		// provider, because it answers the same question: this is here.
		if fd.IsSet() {
			value = tui.StyleGood.Render("●") + " " + fd.Display()
		} else {
			// Empty and "nothing to fill in here" look identical otherwise,
			// which is the failure this widget exists to end. The hint names
			// the variable to export.
			value = tui.StyleFaint.Render(credentialHint(fd))
		}

	case fieldPaste:
		// A paste box is the secret text field it behaves as: the live input
		// while focused -- already in password echo, see loadCursorField -- and
		// bullets otherwise. Its value is never rendered in the clear, and the
		// row says so while it is empty.
		switch {
		case focused:
			value = m.detail.input.View()
		case fd.IsSet():
			value = fd.Display()
		default:
			value = tui.StyleFaint.Render(fd.Placeholder())
		}

	case fieldChoice:
		// A picker has to show what it found. Rendering only the text input
		// left the values invisible and the field looking empty.
		//
		// A strict choice -- a fixed set the provider understands, like the
		// container kind -- is nothing but its options, so drawing a text box
		// beside them just repeats the selected value. A suggestion picker
		// (hosts, profiles, images) shows both, because a value it has not
		// heard of still has to be typeable.
		switch {
		case focused && fd.Strict:
			value = m.renderChoices(fd, w-labelW-6) + tui.StyleFaint.Render("  ▾")
		case focused:
			// The typed value and the offered values share the row, so the
			// input only reserves what it is using -- an empty one would
			// otherwise squeeze the list down to a single entry.
			inputW := tui.Width(fd.Value()) + 2
			if inputW < 6 {
				inputW = 6
			}
			if maxInput := (w - labelW) / 2; inputW > maxInput {
				inputW = maxInput
			}
			value = tui.PadRight(ansi.Truncate(m.detail.input.View(), inputW, ""), inputW) +
				" " + m.renderChoices(fd, w-labelW-inputW-8) + tui.StyleFaint.Render("  ▾")
		case fd.IsSet():
			// No room for the reason a live refresh failed here -- the row has
			// about forty columns and the message is longer. It goes in the
			// footer, which has the whole width, and in the picker itself.
			value = fd.Display() + prefillNote(fd) + tui.StyleFaint.Render("  ▾")
		default:
			value = tui.StyleFaint.Render(m.picker.choiceHint(m.detail.form, fd) + "  ▾")
		}

	default:
		if focused {
			// The live input carries the cursor, so it is what gets drawn.
			value = m.detail.input.View()
		} else if fd.IsSet() {
			value = fd.Display()
		} else {
			value = tui.StyleFaint.Render(fd.Placeholder())
		}
	}

	row := "  " + tui.PadRight(label, labelW) + " " + value
	if focused {
		style := tui.BandSelected
		plain := cursorMark + tui.PadRight(label, labelW) + " "
		// A focused text field keeps its own styling so the cursor stays
		// visible; only the label gets the band. A credential-state field has
		// no cursor either -- there is nothing to type into it -- so it takes
		// the band, while a paste field is a text field and does not.
		if fd.Kind == fieldBool || fd.Kind == fieldMultiChoice || fd.Kind == fieldChoice ||
			fd.Kind == fieldCredentialState {
			return tui.Bar(cursorMark+tui.PadRight(label, labelW)+" "+ansi.Strip(value), w, style)
		}
		return style.Render(plain) + " " + value
	}
	return tui.Truncate(row, w)
}

// renderChoices draws the values a picker offers, windowed around the one under
// the cursor.
func (m Model) renderChoices(fd field, w int) string {
	if len(fd.Options) == 0 {
		return tui.StyleFaint.Render(m.picker.choiceHint(m.detail.form, fd))
	}
	parts := make([]string, len(fd.Options))
	for i, opt := range fd.Options {
		if i == fd.OptCursor() {
			parts[i] = "[" + opt + "]"
			continue
		}
		parts[i] = opt
	}

	// Long values -- container names, EKS context ARNs -- mean the row often
	// holds only one. Say how many are out of view, or the list reads as
	// though it were the whole set. Only what is actually hidden is counted.
	shown, hidden := windowAround(parts, fd.OptCursor(), max(w, 8))
	if hidden == 0 {
		return tui.Truncate(shown, max(w, 8))
	}
	count := fmt.Sprintf("  +%d", hidden)
	avail := max(w-tui.Width(count), 8)
	shown, hidden = windowAround(parts, fd.OptCursor(), avail)
	if hidden == 0 {
		return tui.Truncate(shown, avail)
	}
	return tui.Truncate(shown, avail) + tui.StyleFaint.Render(fmt.Sprintf("  +%d", hidden))
}

// credentialHint says what to do about a credential that is not there. It is
// the field's own description, because only the declaration knows which
// variable this connector reads; the fallback exists so a readout built without
// one still says something rather than rendering as an empty row.
func credentialHint(fd field) string {
	if fd.Desc != "" {
		return fd.Desc
	}
	return "none found"
}

// prefillNote explains a value the user did not type.
//
// It reads the field and nothing else, so it takes one rather than a launcher.
func prefillNote(fd field) string {
	if fd.Prefilled() == "" {
		return ""
	}
	return tui.StyleFaint.Render("  (" + fd.Prefilled() + ")")
}

// windowAround renders as many entries as fit, keeping index i visible, and
// reports how many were left out.
func windowAround(parts []string, i, w int) (string, int) {
	if len(parts) == 0 {
		return "", 0
	}
	if i < 0 || i >= len(parts) {
		i = 0
	}
	start, end := i, i+1
	width := tui.Width(parts[i])
	for {
		grew := false
		if start > 0 && width+tui.Width(parts[start-1])+2 <= w {
			width += tui.Width(parts[start-1]) + 2
			start--
			grew = true
		}
		if end < len(parts) && width+tui.Width(parts[end])+2 <= w {
			width += tui.Width(parts[end]) + 2
			end++
			grew = true
		}
		if !grew {
			break
		}
	}
	return strings.Join(parts[start:end], "  "), len(parts) - (end - start)
}

// viewFooter is the last row of the screen, and it is exactly one row.
//
// The notice is flattened rather than merely truncated. A provider error is a
// string from somewhere else -- an exec failure, a Go client's wrapped chain --
// and several of them contain newlines; truncation cuts a line to width and
// leaves every newline in it, so a three-line error rendered a 24-row terminal
// as 26 rows and pushed this row off the bottom of the screen. tui.OneLine is
// what the report viewer already did to the same class of string.
func (m Model) viewFooter(l layout) string {
	if m.lastErr != "" {
		return tui.StyleBad.Render(tui.Truncate(" ✗ "+tui.OneLine(m.lastErr), l.Width))
	}
	if m.lastWarn != "" {
		return tui.StyleWarn.Render(tui.Truncate(" ! "+tui.OneLine(m.lastWarn), l.Width))
	}
	// Below the two that report trouble, and in the good color: this is a file
	// that was written and the command that reads it, which is neither an error
	// nor a weakened guarantee.
	if m.lastNote != "" {
		return tui.StyleGood.Render(tui.Truncate(" ✓ "+tui.OneLine(m.lastNote), l.Width))
	}

	var hints string
	switch m.focus {
	case focusForm:
		hints = tui.Kbd("↑/↓", "field") + tui.HintSep + tui.Kbd("enter", "choose / scan") + tui.HintSep +
			tui.Kbd("esc", "back")
	default:
		hints = tui.Kbd("↑/↓", "move") + tui.HintSep + tui.Kbd("type", "filter") + tui.HintSep +
			tui.Kbd("enter", "configure") + tui.HintSep + tui.Kbd("esc", "quit")
	}
	// Shown even without credentials: the chooser is where a user finds out
	// why their scans stay local and what to do about it.
	hints += tui.HintSep + tui.Kbd("^r", "reporting")
	hints += tui.HintSep + tui.Kbd("^g", "author")
	hints += m.exportHint()
	if m.viewer.loaded {
		// Only offered once there is one. A key hint for a report that does not
		// exist is a promise the launcher cannot keep.
		hints += tui.HintSep + tui.Kbd("^o", "report")
	}
	hints = " " + hints

	right := ""
	if m.lastRun != "" {
		right = tui.StyleFaint.Render("ran: "+m.lastRun) + " "
	}
	gap := l.Width - tui.Width(hints) - tui.Width(right)
	if right == "" || gap < 2 {
		return tui.Truncate(hints, l.Width)
	}
	return hints + strings.Repeat(" ", gap) + right
}

// xbutton renders a bracketed button, accented when focused.
func xbutton(label string, focused bool) string {
	if focused {
		return tui.StyleAccent.Render(cursorMark + "[ " + label + " ]")
	}
	return tui.StyleFaint.Render("  [ " + label + " ]")
}
