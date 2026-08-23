// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"go.mondoo.com/cnspec/cli/tui"
)

// Picking a value happens in a modal that takes over the body, rather than by
// expanding the field in place.
//
// Expanding in place made two things ambiguous at once: the arrow keys drove
// either the field list or the values depending on invisible state, and the
// fields below jumped down the screen as a list opened. In a modal the arrows
// only ever mean one thing, nothing moves behind it, and the title says what is
// being chosen.

const (
	modalMaxWidth = 72
	// modalPadX is the horizontal padding in modalBox. lipgloss counts padding
	// inside the width it is given, so the usable content is that width minus
	// twice this -- getting it wrong is what wrapped every row by exactly six
	// columns and split long values across two lines.
	modalPadX = 3
	// modalBorder is one column each side.
	modalBorder = 1
)

// modalRows is how many values a picker shows before it scrolls.
const modalRows = 10

var modalBox = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(tui.ColAccent).
	Padding(1, modalPadX)

// modalGeom returns the width handed to the box style and the content width
// inside it, from the total the modal may occupy.
func modalGeom(total int) (boxW, contentW int) {
	boxW = modalWidth(total) - 2*modalBorder
	contentW = boxW - 2*modalPadX
	if contentW < 8 {
		contentW = 8
	}
	return boxW, contentW
}

// modalState is the picker currently open, if any.
type modalState struct {
	open bool
	// field is the index of the field being chosen for.
	field  int
	cursor int
	filter string
}

// move walks the cursor within a list of n values, refusing to leave it. The
// clamp lives with the cursor so that every key that moves it agrees about
// where the ends are.
func (s *modalState) move(delta, n int) {
	next := s.cursor + delta
	if next < 0 || next >= n {
		return
	}
	s.cursor = next
}

// typed extends the filter. The cursor returns to the top because the list
// under it has just changed, and leaving it where it was would select a value
// the user never looked at.
func (s *modalState) typed(runes []rune) {
	s.filter += string(runes)
	s.cursor = 0
}

// backspace shortens the filter, for the same reason typed lengthens it.
func (s *modalState) backspace() {
	if s.filter == "" {
		return
	}
	s.filter = trimLastRune(s.filter)
	s.cursor = 0
}

// trimLastRune drops the last character of a string, which is not the same as
// dropping its last byte.
//
// typed() appends whatever runes the terminal delivered, and a Kubernetes
// context, a container name or a file name is free to hold a multi-byte one.
// Slicing a byte off the end of that leaves a partial encoding: the filter then
// renders as a replacement glyph and matches nothing, and a file name typed
// into the export box would be a path with an invalid byte in it. Shared with
// the export box so both kinds of typing agree about what one backspace means.
func trimLastRune(s string) string {
	if s == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}

// options are the values the picker is offering for a field, after its filter.
func (s modalState) options(fd field) []string {
	if s.filter == "" {
		return fd.Options
	}
	needle := strings.ToLower(s.filter)
	out := make([]string, 0, len(fd.Options))
	for _, o := range fd.Options {
		if strings.Contains(strings.ToLower(o), needle) {
			out = append(out, o)
		}
	}
	return out
}

// valid reports whether the picker still points at a field that exists.
// Rebuilding the form is what can break this, and rebuildForm closes the picker
// itself. The check exists as well because the cost of being wrong is a panic
// in View, which takes the terminal down with it.
func (s modalState) valid(fields []field) bool {
	return s.field >= 0 && s.field < len(fields)
}

// modalWidth fits the box to the terminal, leaving a margin on narrow ones.
func modalWidth(total int) int {
	w := modalMaxWidth
	if fit := total - 8; fit > 24 && fit < w {
		w = fit
	}
	return w
}

// viewModal renders the open picker as a centered box occupying exactly bodyH
// lines, so the footer stays where it is.
func (m Model) viewModal(l layout) string {
	md := m.picker.modal
	fd := m.detail.form.Fields()[md.field]
	opts := md.options(fd)
	boxW, contentW := modalGeom(l.Width)

	var b strings.Builder
	b.WriteString(tui.StyleAccent.Render(tui.Truncate(fd.Label, contentW)))
	if fd.Desc != "" {
		b.WriteString("\n" + tui.StyleDim.Render(tui.Truncate(fd.Desc, contentW)))
	}
	b.WriteString("\n\n")

	switch {
	case len(fd.Options) == 0 && m.picker.waitingFor(m.detail.form, fd) != "":
		b.WriteString(m.spinner.View() + " " +
			tui.StyleDim.Render(tui.Truncate(m.picker.waitingFor(m.detail.form, fd)+"…", contentW-2)))
	case len(fd.Options) == 0:
		b.WriteString(tui.StyleFaint.Render(tui.Truncate(m.picker.choiceHint(m.detail.form, fd), contentW)))
	case len(opts) == 0:
		b.WriteString(tui.StyleFaint.Render("nothing matches " + strconv.Quote(md.filter)))
	default:
		start, end := tui.Window(md.cursor, len(opts), modalRows)

		// The cursor occupies two cells ahead of the value.
		valueW := max(contentW-2, 8)
		for i := start; i < end; i++ {
			text := opts[i]
			if fd.Kind == fieldMultiChoice {
				mark := "○ "
				if fd.Picked(text) {
					mark = "● "
				}
				text = mark + text
			}
			line := truncateTail(text, valueW)
			if i == md.cursor {
				b.WriteString(tui.Bar("▸ "+line, contentW, tui.BandSelected) + "\n")
				continue
			}
			b.WriteString("  " + tui.StyleText.Render(line) + "\n")
		}
		if more := tui.MoreRow(len(opts) - (end - start)); more != "" {
			b.WriteString(more + "\n")
		}
	}

	// A live refresh that failed has to say so even when the cheap read
	// succeeded: the list is showing, it is just incomplete, and the reason is
	// the only actionable thing on screen.
	if busy := m.picker.waitingFor(m.detail.form, fd); busy != "" && len(fd.Options) > 0 {
		b.WriteString(m.spinner.View() + " " +
			tui.StyleDim.Render(tui.Truncate(busy+"…", contentW-2)) + "\n")
	}
	if warn := m.picker.liveError(m.detail.form, fd); warn != "" {
		b.WriteString(tui.StyleWarn.Render(tui.Truncate("! "+warn, contentW)) + "\n")
	}

	b.WriteString("\n")
	if md.filter != "" {
		b.WriteString(tui.Truncate(tui.StyleAccent.Render("⌕ "+md.filter)+
			tui.StyleFaint.Render(fmt.Sprintf("   %d of %d", len(opts), len(fd.Options))), contentW) + "\n")
	}
	help := "↑/↓ choose · type to filter · enter select · esc cancel"
	if fd.Kind == fieldMultiChoice {
		help = "↑/↓ move · space toggle · type to filter · enter done · esc cancel"
	}
	b.WriteString(tui.StyleFaint.Render(tui.Truncate(help, contentW)))

	box := modalBox.Width(boxW).Render(b.String())
	return lipgloss.Place(l.Width, l.BodyH, lipgloss.Center, lipgloss.Center, box)
}

// truncateTail keeps the end of an over-long value rather than the start. The
// identifying part of a Kubernetes context lives at the end -- the cluster name
// in an EKS ARN, the account in an OpenShift URL -- so cutting the head leaves
// the reader something they can tell apart.
func truncateTail(s string, w int) string {
	width := tui.Width(s)
	if width <= w {
		return s
	}
	return ansi.TruncateLeft(s, width-w+1, "…")
}

// openModal starts choosing a value for the field under the cursor.
func (m Model) openModal() (tea.Model, tea.Cmd) {
	if m.detail.form.Cursor() < 0 || m.detail.form.Cursor() >= len(m.detail.form.Fields()) {
		return m, nil
	}
	fd := m.detail.form.Fields()[m.detail.form.Cursor()]
	// Only the two picker kinds have anything to choose from. The launcher's
	// own kinds -- a credential readout, a paste box -- deliberately fall out
	// here: neither has a list, and opening an empty modal over one would hide
	// the thing that does work.
	if fd.Kind != fieldChoice && fd.Kind != fieldMultiChoice {
		return m, nil
	}
	// Nothing to choose from and nowhere to get anything: this is a text field
	// wearing a picker's clothes, and opening an empty box over it only hides
	// the one thing that works, which is typing.
	if len(fd.Options) == 0 && fd.Source() == "" && fd.LiveSource == "" {
		return m, nil
	}
	md := modalState{open: true, field: m.detail.form.Cursor()}
	// Start on the current value so reopening a picker does not lose the place.
	for i, o := range fd.Options {
		if o == fd.Value() {
			md.cursor = i
		}
	}
	m.picker.modal = md
	return m, m.openPickerCmd(fd)
}

// keyModal drives the open picker. Nothing here can launch a scan.
func (m Model) keyModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// md points into the copy this method returns, so what it moves is what the
	// caller ends up with.
	md := &m.picker.modal

	// A modal is only ever opened over a picker (see openModal), but the field
	// it points at can be replaced underneath it by a rebuild. Closing rather
	// than editing is the safe answer for a kind that has no options: a paste
	// box or a credential readout has nothing this key handling can mean.
	//
	// The out-of-range index closes too, rather than being tested and then
	// indexed anyway. Only rebuildForm and syncSelection shrink the field
	// slice and both close the picker, so nothing today gets here with a stale
	// index -- but the guard used to prove the index was in range and then let
	// the line below index with it regardless, which is a panic that takes the
	// terminal down with it. View() has always closed this hole; this is the
	// same close, on the key path.
	if !md.valid(m.detail.form.Fields()) {
		m.picker.close()
		return m, nil
	}
	if k := m.detail.form.Fields()[md.field].Kind; k != fieldChoice && k != fieldMultiChoice {
		m.picker.close()
		return m, nil
	}

	opts := md.options(m.detail.form.Fields()[md.field])

	switch msg.String() {
	case "esc":
		m.picker.close()
		return m, nil
	case "up", "ctrl+p":
		md.move(-1, len(opts))
		return m, nil
	case "down", "ctrl+n":
		md.move(1, len(opts))
		return m, nil
	case " ":
		// Space toggles a multi-select and leaves the picker open, because
		// choosing several is the whole point of one.
		fd := &m.detail.form.Fields()[md.field]
		if fd.Kind == fieldMultiChoice && md.cursor < len(opts) {
			fd.TogglePick(opts[md.cursor])
		}
		return m, nil

	case "enter":
		fd := &m.detail.form.Fields()[md.field]
		if fd.Kind == fieldMultiChoice {
			// Enter closes; the toggling was done with space.
			m.picker.close()
			return m, nil
		}
		if md.cursor >= 0 && md.cursor < len(opts) {
			fd.SetValue(opts[md.cursor])
			fd.SetPrefilled("")
			if m.detail.form.Cursor() == md.field {
				m.detail.input.SetValue(fd.Value())
				m.detail.input.CursorEnd()
			}
		}
		m.picker.close()
		resolveSources(&m.detail.form)
		return m, tea.Batch(m.picker.pendingCmds(m.detail.form)...)
	case "backspace":
		md.backspace()
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		md.typed(msg.Runes)
	}
	return m, nil
}
