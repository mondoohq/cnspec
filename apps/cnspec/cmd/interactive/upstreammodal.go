// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnspec/cli/tui"
)

// Choosing where results go is not a toggle, it is a decision, and the two
// answers differ in a way a one-word badge cannot carry: one sends what the
// scan found off this machine, the other keeps it here. A blind flip would
// leave a user guessing which state they are now in and what it costs them, so
// the badge opens this instead and says what each option means.

// upstreamModalState is the reporting chooser, if it is open.
type upstreamModalState struct {
	open   bool
	cursor int // 0 = report to Mondoo Platform, 1 = stay local
}

// upstreamChoices are the two options in the order they are drawn. Reporting
// leads because it is the one with consequences outside this machine.
const (
	upstreamChoiceReport = 0
	upstreamChoiceLocal  = 1
)

// openModal opens the chooser with the current state under the cursor, so
// pressing enter immediately is a no-op rather than a surprise.
func (u *upstreamState) openModal() {
	u.modal.open = true
	u.modal.cursor = upstreamChoiceLocal
	if u.reporting() {
		u.modal.cursor = upstreamChoiceReport
	}
}

// keyModal handles the chooser's keys and returns what the user has to be told,
// which is empty for every key that simply did what it says.
//
// The warning is returned rather than written to the footer from here: the
// footer is the launcher's chrome, and a chooser that reached into it would be
// a second thing in the package deciding what the last line says.
//
// It is consulted before the rest of the launcher's keys, so while the chooser
// is open the keys mean what it says they do.
func (u *upstreamState) keyModal(msg tea.KeyMsg) string {
	switch msg.String() {
	case "esc", "q", "ctrl+r":
		u.modal = upstreamModalState{}

	case "up", "k", "shift+tab":
		u.modal.cursor = upstreamChoiceReport

	case "down", "j", "tab":
		u.modal.cursor = upstreamChoiceLocal

	case "enter", " ":
		// Reporting needs credentials. Choosing it without them cannot be
		// honoured, so the modal stays open and says why rather than closing on
		// a choice that did not take.
		if u.modal.cursor == upstreamChoiceReport && !u.canToggle() {
			return noCredentialsWarning
		}
		u.incognito = u.modal.cursor == upstreamChoiceLocal
		u.modal = upstreamModalState{}
	}
	return ""
}

// noCredentialsWarning is shared with the footer so the modal and the launcher
// say the same thing about the same situation.
const noCredentialsWarning = "no Mondoo Platform credentials on this machine, so scans stay local — run `cnspec login` to connect"

// viewModal draws the chooser, occupying exactly bodyH lines so the footer does
// not move.
func (u upstreamState) viewModal(l layout) string {
	boxW, contentW := modalGeom(l.Width)

	var b strings.Builder
	b.WriteString(tui.StyleAccent.Render(tui.Truncate("Where do scan results go?", contentW)))
	b.WriteString("\n\n")

	for _, opt := range []struct {
		idx         int
		label, desc string
		available   bool
	}{
		{
			idx:   upstreamChoiceReport,
			label: u.reportChoiceLabel(),
			desc: "Findings are uploaded, so they appear in the console, feed " +
				"compliance reports, and are visible to your team.",
			available: u.canToggle(),
		},
		{
			idx:       upstreamChoiceLocal,
			label:     "Keep results on this machine",
			desc:      "Nothing leaves this computer. The scan runs with --incognito and the report is yours alone.",
			available: true,
		},
	} {
		mark := "  "
		if (opt.idx == upstreamChoiceReport) == u.reporting() {
			mark = "● " // the state you are in now
		}
		line := mark + opt.label
		if opt.idx == u.modal.cursor {
			b.WriteString(tui.Bar(tui.Truncate(line, contentW), contentW, tui.BandSelected))
		} else if !opt.available {
			b.WriteString(tui.StyleFaint.Render(tui.Truncate(line, contentW)))
		} else {
			b.WriteString(tui.Truncate(line, contentW))
		}
		b.WriteString("\n")

		for _, wrapped := range tui.WrapWords(opt.desc, contentW-4) {
			b.WriteString(tui.StyleDim.Render("    " + wrapped))
			b.WriteString("\n")
		}
		if !opt.available {
			b.WriteString(tui.StyleWarn.Render(tui.Truncate("    needs credentials — run `cnspec login`", contentW)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(tui.StyleFaint.Render(tui.Truncate("↑/↓ choose · enter confirm · esc cancel", contentW)))

	box := modalBox.Width(boxW).Render(b.String())
	return lipgloss.Place(l.Width, l.BodyH, lipgloss.Center, lipgloss.Center, box)
}

// reportChoiceLabel names the space when the config declares one, because
// "report to Mondoo Platform" is a different decision depending on which space
// is on the other end.
func (u upstreamState) reportChoiceLabel() string {
	if u.scope != "" {
		return "Report to Mondoo Platform · " + u.scope
	}
	return "Report to Mondoo Platform"
}
