// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"github.com/charmbracelet/bubbles/textinput"
	"go.mondoo.com/cnspec/cli/tui"
)

// spot is where the keys are inside the detail pane: on a field, on the row
// that reveals the folded options, or on the scan button.
//
// It is one field because it is one fact, and as two bools -- moreFocused and
// scanFocused -- it was three. The two could both be set, and when they were,
// nothing agreed about what that meant: moveFocus read "more" first, the
// renderer banded the reveal row *and* accented the button, and enter launched
// a scan. Two selection bands on one screen was the visible half of it.
type spot int

const (
	// spotField is the field at form.Cursor(). It is the resting state: a pane
	// with fields opens on one, and every other position is left deliberately.
	spotField spot = iota
	// spotMore is the row that unfolds the options section.
	spotMore
	// spotButton is the scan button, which sits after the last field. Scanning
	// is an explicit act: nothing else in the pane starts it.
	spotButton
)

// detailState is the detail pane: the form for the connector under the cursor,
// the box editing whichever of its fields has the keys, where in the pane those
// keys are, whether the long tail of provider flags is folded away, and how far
// the pane is scrolled.
//
// They are one type because they are one screen and they change together. The
// input mirrors the field under the cursor -- loadCursorField and
// storeCursorField are the whole of that invariant -- so a cursor that moved
// without them was silent input loss, which is a bug this package has already
// shipped once. showOptions decides which fields the cursor can reach at all,
// which is why spot cannot be read without it.
type detailState struct {
	// form is the input screen for the connector under the cursor.
	form form
	// input edits whichever form field has the cursor; its value is written
	// back to the field on every keystroke so the command bar stays live.
	input textinput.Model
	// spot is which of the pane's three kinds of row the keys are on.
	spot spot
	// showOptions reveals the long tail of provider flags, which stays folded
	// away until asked for.
	showOptions bool
	// offset scrolls the pane, which outgrows the screen as soon as a form has
	// more than a handful of fields.
	offset tui.Scroll
}

// onMore and onButton read the pane's cursor. They exist so that the two rows
// that are not fields are asked about by name rather than by comparing against
// a constant at twenty call sites.
func (d detailState) onMore() bool   { return d.spot == spotMore }
func (d detailState) onButton() bool { return d.spot == spotButton }

// onField reports whether the keys are on a field rather than on the reveal row
// or the button, which is what decides whether a keystroke is input.
func (d detailState) onField() bool { return d.spot == spotField }

// leaveMore and leaveButton put the keys back on a field, but only from the row
// named. They are two methods rather than one assignment because the callers
// are undoing one specific position -- the reveal row is gone because the
// options were just unfolded, the button is released because focus left the
// pane -- and an unconditional reset would move the keys off a row the caller
// said nothing about.
func (d *detailState) leaveMore() {
	if d.spot == spotMore {
		d.spot = spotField
	}
}

func (d *detailState) leaveButton() {
	if d.spot == spotButton {
		d.spot = spotField
	}
}

// focusableFields are the fields the keyboard can currently reach: the visible
// ones, minus the folded-away options.
func (d detailState) focusableFields() []int {
	out := make([]int, 0, len(d.form.Fields()))
	for _, i := range d.form.VisibleIndices() {
		if !d.showOptions && d.form.Fields()[i].Section == sectionOptions {
			continue
		}
		out = append(out, i)
	}
	return out
}

// hasMoreRow reports whether the folded options are worth a row of their own.
func (d detailState) hasMoreRow() bool {
	if d.showOptions {
		return false
	}
	for _, i := range d.form.VisibleIndices() {
		if d.form.Fields()[i].Section == sectionOptions {
			return true
		}
	}
	return false
}

// loadCursorField binds the shared text input to the field under the cursor.
func (d *detailState) loadCursorField() {
	if d.form.Cursor() < 0 || d.form.Cursor() >= len(d.form.Fields()) {
		return
	}
	fd := d.form.Fields()[d.form.Cursor()]
	// A credential-state field is a readout, not an input: binding the shared
	// text box to it would put a cursor on a sentence the user cannot edit.
	if fd.Kind == fieldCredentialState {
		d.input.SetValue("")
		d.input.Blur()
		return
	}
	d.input.SetValue(fd.Value())
	d.input.CursorEnd()
	d.input.EchoMode = textinput.EchoNormal
	if fd.Secret || fd.Kind == fieldPaste {
		d.input.EchoMode = textinput.EchoPassword
	}
	d.input.Focus()
}

// storeCursorField writes the shared input back into the focused field.
func (d *detailState) storeCursorField() {
	if d.form.Cursor() < 0 || d.form.Cursor() >= len(d.form.Fields()) {
		return
	}
	// A credential-state field's value is what the launcher discovered, not
	// what the shared input holds, so writing the input back over it would
	// erase the readout the moment the cursor passed through.
	if fd := &d.form.Fields()[d.form.Cursor()]; fd.Kind != fieldBool &&
		fd.Kind != fieldMultiChoice && fd.Kind != fieldCredentialState {
		fd.SetValue(d.input.Value())
	}
}

// applyPrefill fills a field whose picker came back with an unambiguous answer.
// It never overwrites what the user typed: a late-arriving source must not move
// the ground under someone who is already editing.
//
// focused is whether the pane has the keys, which the pane cannot answer for
// itself -- see Model.focus, which says which of the two panes they are in.
func (d *detailState) applyPrefill(source string, values []string, focused bool) {
	value, why := preferredValue(source, values)
	if value == "" {
		return
	}
	for i := range d.form.Fields() {
		fd := &d.form.Fields()[i]
		if fd.Source() != source || fd.Value() != "" {
			continue
		}
		fd.SetValue(value)
		fd.SetPrefilled(why)
		for j, opt := range fd.Options {
			if opt == value {
				fd.SetOptCursor(j)
			}
		}
		// The shared input mirrors the focused field, so it has to follow.
		if focused && d.form.Cursor() == i {
			d.loadCursorField()
		}
	}
}

// The command preview lives on launchRequest, not here. See preview in
// launch.go: it is the same words plan() runs, and it moved because assembling
// them twice is what let the bar and the command disagree.

// The detail pane is described once, as a flat list of items, and both the
// renderer and the layout walk that list. One item is exactly one line, so a
// click zone cannot land on a row the renderer drew something else on -- the
// two cannot drift because there is only one description to drift from.

type detailItemKind int

const (
	diBlank   detailItemKind = iota
	diLabel                  // a section heading
	diText                   // a plain line
	diField                  // idx is the form field index
	diMore                   // the row that reveals the options
	diButton                 // the scan button
	diCommand                // the command bar
)

// expandedRows is how many values an open picker shows at once. Enough to
// browse, not so many that the fields below it disappear.

type detailItem struct {
	kind detailItemKind
	idx  int
	text string
}

// detailPlan lays out the connector detail pane.
func (m Model) detailPlan() []detailItem {
	c, ok := m.list.current()
	if !ok {
		return nil
	}

	// No action list. The launcher is for scanning; showing six alternatives
	// above the fields meant a whole navigation layer, and a focus stop, before
	// reaching the thing the user came to fill in.
	out := []detailItem{
		{kind: diBlank},
		{kind: diText, text: c.Short},
	}

	// A pane with no fields on it has to say why, and there are two different
	// reasons -- which is what the single "install its provider" line used to
	// get wrong. `device` is installed and declares nine flags, all of them
	// Hidden or Deprecated, so there is nothing to ask and nothing to install;
	// `mondoo` is installed and declares nothing at all, because its target is
	// the workstation's own registration. Telling either of them to install a
	// provider they already have is the wrong sentence.
	//
	// The condition is what the form actually built, not what the connector
	// claims. A spec's positional fields survive on an uninstalled connector --
	// kustomize asks for its overlay path whether or not the provider is here
	// yet -- so a pane can be usefully full while the metadata behind it is
	// still the flagless static entry.
	if m.list.downloading(c) {
		out = append(out,
			detailItem{kind: diBlank},
			detailItem{kind: diText, text: "installing the " + c.Provider + " provider…"})
	} else if len(m.detail.form.Fields()) == 0 {
		text := "nothing to configure — press Scan"
		if !c.Installed {
			text = "open this connector to install its provider"
		}
		out = append(out, detailItem{kind: diBlank}, detailItem{kind: diText, text: text})
	}

	sections, bySection := m.detail.form.Ordered()
	for _, s := range sections {
		var shown []int
		for _, idx := range bySection[s] {
			if m.detail.form.Visible(idx) {
				shown = append(shown, idx)
			}
		}
		if len(shown) == 0 {
			continue
		}

		// Everything a connector declares is not a configuration screen. The
		// target and the credential are the questions worth asking; the rest is
		// a long tail that belongs behind one row until it is wanted -- k8s
		// alone puts eight of them between the target and the scan button.
		if s == sectionOptions && !m.detail.showOptions {
			out = append(out, detailItem{kind: diBlank},
				detailItem{kind: diMore, idx: len(shown)})
			continue
		}

		out = append(out, detailItem{kind: diBlank}, detailItem{kind: diLabel, text: s})
		for _, idx := range shown {
			out = append(out, detailItem{kind: diField, idx: idx})
		}
	}

	out = append(out, detailItem{kind: diBlank}, detailItem{kind: diButton})
	return out
}

// splitPlan separates the scrolling body from the command bar, which is pinned
// to the last line of the pane. It is what the whole screen is building toward,
// so it must never be the thing that scrolls out of view.
func splitPlan(plan []detailItem) (body []detailItem, command detailItem) {
	return plan, detailItem{kind: diCommand}
}

// ensureFieldVisible scrolls the detail pane so the focused field is on screen.
func (m *Model) ensureFieldVisible() {
	if m.focus != focusForm {
		return
	}
	plan := m.detailPlan()
	row := -1
	for i, item := range plan {
		if item.kind == diField && item.idx == m.detail.form.Cursor() {
			row = i
			break
		}
	}
	if row < 0 {
		return
	}
	l := tui.Dims(m.width, m.height)
	visible := tui.InnerHeight(l.BodyH) - 1 // the pinned command bar
	m.detail.offset.EnsureVisible(row, len(plan), visible)
}
