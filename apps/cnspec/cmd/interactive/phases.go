// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// The launcher owns the terminal for the whole session now, so it has three
// things it can be doing with it rather than one.
//
// phaseForm is the launcher proper: the connector list and the fields. From
// there a scan moves to phaseScanning, which watches a background child, and
// then to phaseViewing, which delegates the whole screen to the report viewer.
// Both of the other two lead back to phaseForm, because the point of not
// handing the terminal to a child process is that the next scan does not need a
// new program.
type phase int

const (
	phaseForm phase = iota
	phaseScanning
	phaseViewing
	// phaseAuthoring is the check-authoring pane. It leads back to phaseForm
	// like the other two: the point of holding the terminal is that the next
	// thing the user does does not need a new program.
	phaseAuthoring
)

// startScan moves the launcher onto the scanning screen and forks the child.
func (m Model) startScan(plan launchPlan) (tea.Model, tea.Cmd) {
	label := "scan"
	if c, ok := m.list.current(); ok {
		label = c.Name
	}

	cmd := m.scan.begin(plan, label)
	m.phase = phaseScanning
	// The plan's cleanup belongs to the session now, so the launch path must
	// not also hold one: releasing the inventory twice is harmless, releasing
	// it early is not.
	m.launching.disown()

	return m, tea.Batch(cmd, m.spinner.Tick)
}

// scanReport opens the viewer on what the child wrote, or explains why it
// cannot. It stays on Model because those two outcomes are the footer and the
// viewer, neither of which is the scan's to decide.
func (m Model) scanReport(msg scanReportMsg) (tea.Model, tea.Cmd) {
	out, ok := m.scan.finish(msg)
	if !ok {
		return m, nil
	}
	m.phase = phaseForm

	if msg.err != nil {
		m.lastErr = scanFailure(msg.err, out)
		return m, textinput.Blink
	}

	cmd := m.viewer.open(msg.report, m.width, m.height)
	m.phase = phaseViewing
	return m, cmd
}

// scanFailure is what the user is told when a scan produced no readable report.
//
// The child's own output leads, because that is where the reason is: a provider
// that could not connect, a policy that would not resolve, a credential the
// remote end refused. The loader's error is about a file and explains nothing
// on its own.
func scanFailure(err error, out scanOutcome) string {
	msg := "the scan produced no report"
	if lines := lastLines(out.output, 1); len(lines) > 0 {
		return msg + ": " + strings.TrimSpace(lines[0])
	}
	if out.err != nil {
		return msg + ": " + out.err.Error()
	}
	if err != nil {
		return msg + ": " + err.Error()
	}
	return msg
}

// cancelScan kills the child and returns to the fields.
func (m Model) cancelScan() (tea.Model, tea.Cmd) {
	m.scan.abort()
	m.phase = phaseForm
	m.lastWarn = "scan cancelled"
	return m, textinput.Blink
}

// updateScanning drives the scanning screen. It handles only what that screen
// can do: cancel, quit, resize, and animate.
func (m Model) updateScanning(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.ensureVisible(m.listH())
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m.quit()
		case "esc", "q":
			return m.cancelScan()
		}
		return m, nil
	}
	return m, nil
}

// updateViewing delegates to the report viewer.
//
// Everything except ctrl+c goes to the viewer's own model, including the keys
// the launcher would otherwise claim: while a report is open the viewer's key
// map is the only one, which is what keeps `cnspec report view` and this screen
// the same program to learn.
func (m Model) updateViewing(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		// Handled before delegation: ctrl+c quits from anywhere, and the
		// interception below cannot tell the viewer's ctrl+c from its q.
		return m.quit()
	}

	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		m.list.ensureVisible(m.listH())
	}

	return m, m.viewer.update(msg)
}

// closeViewer returns from the report to the launcher, keeping the report so
// that leaving it is not the same as losing it.
func (m Model) closeViewer() (tea.Model, tea.Cmd) {
	m.phase = phaseForm
	return m, textinput.Blink
}

// reopenViewer shows the report of the most recent scan again.
func (m Model) reopenViewer() (tea.Model, tea.Cmd) {
	if !m.viewer.loaded {
		m.lastWarn = "no report yet — run a scan first"
		return m, nil
	}
	m.phase = phaseViewing
	return m, m.viewer.resize(m.width, m.height)
}

// interactiveAction reports whether an action has to own the terminal itself.
//
// `shell` is the whole list: it is an interactive REPL and it produces no
// report, so there is nothing for a background child to hand back and nothing
// for the viewer to draw. Everything else the launcher runs writes a report,
// and running it in the background is what keeps the launcher on screen.
func interactiveAction(name string) bool { return name == "shell" }
