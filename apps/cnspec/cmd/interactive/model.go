// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"go.mondoo.com/cnspec/cli/reportview"
	"go.mondoo.com/cnspec/cli/tui"
)

// focusArea is which part of the launcher the keyboard drives. The list and the
// detail pane are always both on screen, so focus -- not a wizard step -- is
// what decides where a key lands.
type focusArea int

const (
	focusList focusArea = iota
	focusForm
)

// Model is the launcher. What it holds is the bubbletea surface -- Init, Update
// and View -- plus the states the screens are made of, each of which owns the
// facts that change together and the methods that change them.
//
// Nothing below is a lone fact that belongs to one of those states. What is
// left over is the chrome, and each field says why it is chrome.
type Model struct {
	// list is the catalog, the search over it, and the cursor in it. See
	// list.go.
	list listState
	// detail is the pane on the right: the form for the connector under the
	// cursor, the box editing it, where in the pane the keys are, and how far
	// it is scrolled. See detail.go.
	detail detailState
	// picker is the value pickers: the one that is open, and what each of them
	// has found for the parameters it was asked about. See picker.go.
	//
	// It is a sibling of detail rather than a part of it because it is the only
	// state here with a life of its own: a load outlives the form that started
	// it, and answers to a key rather than to a field. What joins the two is
	// sourceKeyFor, which turns a field into the key it asks under.
	picker pickerState
	// upstream is where a scan's results would go, whether that is the user's
	// choice, and the chooser that makes it. Read once at startup: it is a
	// config file read, and the answer does not change under us.
	upstream upstreamState
	// scan is the background scan being watched, if any. See scan.go.
	scan scanState
	// viewer holds the report of the most recent scan and the embedded model
	// that draws it. See viewer.go.
	viewer viewerState
	// launching is the command being assembled and what the last one left on
	// disk. See launch.go.
	launching launchState
	// export is the box that writes the configured target out as a reusable
	// inventory file. See export.go.
	//
	// It is a sibling of launching rather than part of it because the two go to
	// different places with the same material: launching writes a private file
	// for a child process and removes it, export writes one the user keeps. It
	// outlives a keypress the same way launching does, and for the same reason
	// -- the OS keychain dialog.
	export exportState

	// --- the launcher's own chrome ------------------------------------------

	// focus is which of the two panes the keyboard drives. It stays here for
	// the same reason phase does: the list asks it whether to draw itself
	// active, the detail pane asks it the same question, and the layout asks it
	// which pane to show when the terminal is too narrow for both. A pane that
	// owned it would be answering for its sibling.
	//
	// Where the keys are *inside* the detail pane is that pane's own business
	// and lives there, as detailState.spot.
	focus focusArea

	width  int
	height int

	// spinner animates a wait. A gcloud call can take tens of seconds, and a
	// still screen is indistinguishable from a hung one.
	//
	// It is chrome rather than the pickers' because the scanning screen borrows
	// its current frame, and a scan reaching into the pickers to find out what
	// a spinner looks like would be the coupling this split removed. One
	// animation clock, like width and height.
	spinner spinner.Model

	// lastErr is shown in the footer until the next key; lastRun is the command
	// that most recently ran, so returning from it says what just happened.
	lastErr string
	// lastWarn describes a weaker guarantee than usual -- a credential that
	// could not reach the keychain, say. The command still ran.
	lastWarn string
	// lastNote is something that simply worked and is worth saying: a file that
	// was written, and the command that reads it.
	//
	// It exists because there were two footer channels and three kinds of
	// message. A completed export went out through lastWarn, so the launcher
	// announced a file it had just written correctly in the warning color,
	// behind a "!". Three channels is the smaller lie.
	lastNote string
	lastRun  string

	// phase is what the launcher is doing with the terminal: showing the
	// fields, watching a background scan, or handing the screen to the report
	// viewer. See phases.go.
	//
	// It stays on Model rather than inside scan: two of the three phases are
	// not the scan's, and a viewer that had to read the scan's state to know
	// whether it is on screen would be the coupling this split removed.
	phase phase
}

// NewModel builds a launcher model over the given catalog.
func NewModel(catalog []Connector) Model {
	a := textinput.New()
	a.Prompt = ""
	a.CharLimit = 256

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = tui.StyleAccent

	const width, height = 100, 30
	m := Model{
		list:     newListState(catalog, tui.Dims(width, height).ListH),
		upstream: readUpstreamFn(),
		detail:   detailState{input: a},
		picker:   newPickerState(),
		spinner:  sp,
		width:    width,
		height:   height,
	}
	m.syncSelection()
	return m
}

func (m Model) Init() tea.Cmd { return tea.Batch(textinput.Blink, m.spinner.Tick) }

// listH is how many rows of the connector list fit on screen. Everything the
// list does to its own scroll offset needs it, and it is the launcher's
// geometry rather than the list's.
func (m Model) listH() int { return tui.Dims(m.width, m.height).ListH }

// syncSelection refreshes everything derived from the cursor: the action list
// and the argument field's placeholder.
func (m *Model) syncSelection() {
	c, ok := m.list.current()
	if !ok {
		m.detail.form = form{}
		return
	}

	// Rebuilding on every cursor move would throw away half-typed input, so
	// the form is only rebuilt when the selection actually changed connector.
	if m.detail.form.Subject() != c.Name {
		m.detail.form = newForm(c)
		m.picker.fill(&m.detail.form)
	}
}

// rebuildForm rebuilds the current form from scratch, keeping what the user
// typed and leaving no stale references behind.
//
// This is the path taken when a provider finishes installing: the connector
// gains the metadata the form is built from, so the form has to be rebuilt --
// but it is rebuilt underneath whatever the user is doing at that moment.
func (m *Model) rebuildForm() {
	c, ok := m.list.current()
	if !ok {
		m.detail.form = form{}
		m.picker.close()
		return
	}
	old := m.detail.form
	m.detail.form = newForm(c)
	carryOver(&m.detail.form, old)
	m.picker.fill(&m.detail.form)
	// An open picker indexes the field list it was opened against. The list
	// just changed, so the picker is closed rather than left pointing into it.
	if m.picker.modal.open {
		m.picker.close()
	}
}

// readyToRun reports whether the form has everything the command needs, which
// decides whether enter scans or opens the fields.
func (m Model) readyToRun() bool {
	if _, ok := m.list.current(); !ok {
		return false
	}
	return m.detail.form.Validate() == nil
}

// openPickerCmd starts the loads a field being opened needs, and restarts the
// animation clock for them.
//
// The loads are pickerState.openCmds; the clock is the launcher's one spinner.
// This is the seam between them and is all that is left on Model of what used
// to be the whole method: the pickers have no spinner to tick, and giving them
// one would be a second animation clock nothing would keep in step with the
// first.
func (m Model) openPickerCmd(fd field) tea.Cmd {
	cmds := m.picker.openCmds(m.detail.form, fd)
	if len(cmds) == 0 {
		return nil
	}
	// The spinner is only ticking while something is loading, so restart it.
	return tea.Batch(append(cmds, m.spinner.Tick)...)
}

// moveCursor walks the list and rebuilds the form under it.
//
// The two halves stay together on Model because they belong to different
// clusters: the list owns where the cursor is, and syncSelection owns what the
// detail pane shows for it. A list that rebuilt the form would be the list
// knowing what a form is.
func (m *Model) moveCursor(delta int) {
	if !m.list.move(delta, m.listH()) {
		return
	}
	m.syncSelection()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// A background scan reports in wherever the launcher happens to be, so its
	// messages are handled before the phase is consulted. Each handler checks
	// that the session is still the current one: an event from a scan the user
	// cancelled is dropped, and the session releases itself.
	switch msg := msg.(type) {
	case scanRunningMsg:
		return m, m.scan.running(msg)
	case scanProgressMsg:
		return m, m.scan.progressed(msg)
	case scanExitedMsg:
		return m, m.scan.exited(msg)
	case scanReportMsg:
		return m.scanReport(msg)
	case reportview.ExportDoneMsg:
		// A write can outlive the view that started it: close the report while
		// a large json-full is still being written and the result arrives here,
		// with the viewer gone. Say it in the footer rather than dropping a
		// confirmation the user is waiting for.
		if m.phase == phaseViewing {
			break // the viewer is up and says it itself
		}
		if msg.Err != nil {
			m.lastErr = reportview.ExportNotice(msg)
		} else {
			m.lastNote = reportview.ExportNotice(msg)
		}
		return m, nil

	case exportParsedMsg:
		// The two halves of an export answer from off the event loop and both
		// can outlive the box that asked, so they are handled before the phase
		// is consulted -- a scan started in the meantime must not swallow the
		// word that a file was written. Each checks for itself whether it is
		// still wanted; see exportParsed and exportWrote.
		return m.exportParsed(msg)
	case exportWroteMsg:
		return m.exportWrote(msg)

	case viewerClosedMsg:
		return m.closeViewer()
	}

	switch m.phase {
	case phaseScanning:
		return m.updateScanning(msg)
	case phaseViewing:
		return m.updateViewing(msg)
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		// Only animate while something is actually being fetched; a spinner
		// ticking over a settled screen is noise. The export box waits on a
		// provider and then on the OS keychain, so it is a second thing that
		// can be waiting -- an animation clock that only knew about the pickers
		// would leave the box still while it was the only thing on screen.
		if !m.picker.busy() && !m.export.busy() {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.ensureVisible(m.listH())
		return m, nil

	case launchPreparedMsg:
		m.launching.prepared()
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			return m, nil
		}
		m.lastWarn = msg.plan.warn
		m.lastRun = "cnspec " + strings.Join(msg.plan.args, " ")
		if interactiveAction(msg.action) {
			// A shell has no report and is genuinely interactive, so it still
			// gets the terminal handed to it. Everything else runs in the
			// background and comes back as a report.
			m.launching.cleanup = msg.plan.cleanup
			return m, launchCmd(msg.plan.args, msg.plan.env, msg.plan.warn)
		}
		return m.startScan(msg.plan)

	case launchDoneMsg:
		m.launching.release()
		if msg.err != nil {
			m.lastErr = msg.err.Error()
		}
		return m, textinput.Blink

	case sourceValuesMsg:
		m.picker.answer(msg)
		if msg.cancelled {
			// The picker that asked has been closed, and answer files nothing
			// for it: caching an empty list would leave a key a later picker
			// would then trust.
			return m, nil
		}
		if msg.err != nil && len(msg.values) == 0 {
			// Say it where there is room to say it. A picker that quietly
			// offers less than it should is the failure mode worth avoiding.
			m.lastWarn = msg.err.Error()
		}
		m.picker.fill(&m.detail.form)
		m.detail.applyPrefill(msg.source, msg.values, m.focus == focusForm)
		return m, nil

	case providerInstalledMsg:
		if m.list.installing == msg.provider {
			m.list.installing = ""
		}
		if msg.err != nil {
			m.lastErr = "could not install the " + msg.provider + " provider: " + msg.err.Error()
			return m, nil
		}
		if m.list.applyInstalled(msg, m.listH()) {
			m.syncSelection()
		}
		// The catalog entry now carries the metadata the form is built from,
		// so rebuild it and start any pickers it asks for.
		m.rebuildForm()
		return m, tea.Batch(m.picker.pendingCmds(m.detail.form)...)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m.quit()
	}
	// any key dismisses what is showing
	m.lastErr, m.lastWarn, m.lastNote = "", "", ""

	// Reporting is global: it decides where the results of anything launched
	// from here end up, so it is reachable from every pane. It is a control key
	// because the list treats bare letters as filter input.
	if m.upstream.modal.open {
		m.lastWarn = m.upstream.keyModal(msg)
		return m, nil
	}
	if msg.String() == "ctrl+r" {
		m.upstream.openModal()
		return m, nil
	}

	// The export box takes every key while it is up, for the same reason the
	// pickers do: what a key means there is what the box says it means.
	if m.export.open {
		return m.exportKey(msg)
	}

	if m.picker.modal.open {
		return m.keyModal(msg)
	}

	// Reachable from both panes, like reporting and the report: what would be
	// exported is the form the launcher is showing, whichever side the keys are
	// on. Not reachable from an open picker, which owns the screen while it is
	// up and must not be left standing behind another box.
	if msg.String() == "ctrl+e" {
		return m.openExport()
	}

	// The report of the last scan is still in memory, so leaving the viewer is
	// not the same as losing it. Reachable from both panes for the same reason
	// reporting is -- it is about the session, not about the field under the
	// cursor -- but not from an open picker, which owns the screen while it is
	// up and must not be left standing behind a report.
	if msg.String() == "ctrl+o" {
		return m.reopenViewer()
	}
	switch m.focus {
	case focusForm:
		return m.keyForm(msg)
	default:
		return m.keyList(msg)
	}
}

// quit leaves, removing what the launcher wrote on the way out.
//
// The generated inventory holds a plaintext credential whenever the keychain
// was unavailable, and it used to be removed on exactly one event -- the
// command it fed finishing. Quitting between assembling a plan and running it,
// or quitting at all after a scan that never happened, left that file in the
// system temp directory. See cleanupTempFiles for the rest of the story.
func (m Model) quit() (tea.Model, tea.Cmd) {
	m.scan.cancel()
	cleanupTempFiles()
	// Dropped rather than run: cleanupTempFiles has just removed what it would
	// remove, because trackTemp registered it when the plan was assembled.
	m.launching.disown()
	return m, tea.Quit
}

func (m Model) keyList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.list.clearSearch(m.listH()) {
			m.syncSelection()
			return m, nil
		}
		return m.quit()
	case "up", "ctrl+p":
		m.moveCursor(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveCursor(1)
		return m, nil
	case "pgup":
		m.moveCursor(-m.listH())
		return m, nil
	case "pgdown":
		m.moveCursor(m.listH())
		return m, nil
	case "home":
		m.moveCursor(-len(m.list.selectable))
		return m, nil
	case "end":
		m.moveCursor(len(m.list.selectable))
		return m, nil
	case "tab", "right", "enter", "shift+tab":
		// Enter never scans from here. Configuring a target and starting a scan
		// are separate acts, and a key that did both meant choosing a connector
		// could launch one by accident.
		return m.focusRight()
	}

	cmd := m.list.typed(msg, m.listH())
	m.syncSelection()
	return m, cmd
}

// focusRight moves focus into the fields.
func (m Model) focusRight() (tea.Model, tea.Cmd) {
	// Opening the detail pane is the first point where the user has committed
	// to a connector, so it is when the provider gets installed if it is
	// missing -- the form is built from metadata only an installed provider
	// carries. The download would happen on the first scan anyway.
	install := m.list.ensureProvider()
	model, cmd := m.enterForm()
	return model, tea.Batch(install, cmd)
}

// enterForm moves focus onto the input fields and kicks off any value pickers
// the form needs.
func (m Model) enterForm() (tea.Model, tea.Cmd) {
	m.focus = focusForm

	// A connector with nothing to configure -- local is the whole point of the
	// launcher's short path -- still needs somewhere for the keys to land, and
	// that is the scan button.
	if len(m.detail.focusableFields()) == 0 {
		m.detail.spot = spotButton
		m.list.search.Blur()
		m.detail.input.Blur()
		return m, nil
	}

	m.detail.leaveButton()
	m.list.search.Blur()
	if visible := m.detail.focusableFields(); len(visible) > 0 {
		m.detail.form.SetCursor(visible[0])
	}
	m.ensureFieldVisible()
	m.detail.loadCursorField()
	return m, tea.Batch(append(m.picker.pendingCmds(m.detail.form), textinput.Blink)...)
}

// focusFirstMissing puts the cursor on the first required field still empty,
// so opening the fields lands on the thing standing between here and a scan.
func (m *Model) focusFirstMissing() {
	for i, fd := range m.detail.form.Fields() {
		if fd.Required && !fd.IsSet() {
			m.detail.form.SetCursor(i)
			m.ensureFieldVisible()
			m.detail.loadCursorField()
			return
		}
	}
}

func (m Model) keyForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.detail.focusableFields()
	// With nothing reachable to type into, the keys cannot be on a field, so
	// they go to the button. The reveal row is a position in its own right and
	// is left alone -- a form whose every field is a folded option has no
	// reachable field and still has that row, and clobbering it here is what
	// stopped `up` from walking off the top of such a pane back to the list.
	if len(visible) == 0 && m.detail.onField() {
		m.detail.spot = spotButton
	}

	switch msg.String() {
	case "esc":
		m.detail.storeCursorField()
		m.focus = focusList
		// The button is released on the way out, so returning to the pane does
		// not land on it: the fields are what the user came back for.
		m.detail.leaveButton()
		m.detail.input.Blur()
		m.list.search.Focus()
		return m, textinput.Blink

	case "shift+tab", "up", "ctrl+p":
		if !m.moveFocus(-1) {
			// Off the top of the pane: hand the keys back to the list.
			m.detail.storeCursorField()
			m.focus = focusList
			m.detail.input.Blur()
			m.list.search.Focus()
		}
		return m, textinput.Blink

	case "tab", "down", "ctrl+n":
		m.moveFocus(1)
		return m, textinput.Blink

	case "enter":
		// The scan button is the only thing that scans.
		if m.detail.onButton() {
			return m.launch()
		}
		if m.detail.onMore() {
			m.detail.showOptions = true
			// The row the keys were on has just stopped existing, so they go
			// back to a field and moveFocus(0) settles them on a real one.
			m.detail.leaveMore()
			m.moveFocus(0)
			return m, textinput.Blink
		}
		return m.activateField()
	}

	if !m.detail.onField() || m.detail.form.Cursor() >= len(m.detail.form.Fields()) {
		return m, nil
	}

	fd := &m.detail.form.Fields()[m.detail.form.Cursor()]
	switch fd.Kind {
	case fieldBool:
		if msg.String() == " " {
			fd.Toggle()
		}
		return m, nil
	case fieldChoice, fieldMultiChoice:
		if msg.String() == " " {
			return m.openModal()
		}
		// A picker still accepts a value it has not heard of, so typing falls
		// through to the text input below.
	case fieldCredentialState:
		// Stub. There is nothing to type into a readout, so keys are swallowed
		// rather than falling through to the shared input and being written
		// back over the state. Navigation was handled above and still works.
		return m, nil
	case fieldPaste:
		// Stub. A paste field is a secret text field, so typing and pasting
		// fall through to the shared input below.
	}

	var cmd tea.Cmd
	m.detail.input, cmd = m.detail.input.Update(msg)
	m.detail.storeCursorField()
	resolveSources(&m.detail.form)
	return m, tea.Batch(append(m.picker.pendingCmds(m.detail.form), cmd)...)
}

// activateField does whatever the focused field needs: open a picker, flip a
// switch, or step past a text field, which is edited in place.
func (m Model) activateField() (tea.Model, tea.Cmd) {
	fd := &m.detail.form.Fields()[m.detail.form.Cursor()]
	switch fd.Kind {
	case fieldChoice, fieldMultiChoice:
		return m.openModal()
	case fieldBool:
		fd.Toggle()
		return m, nil
	case fieldCredentialState:
		// Stub. Enter on a readout will eventually re-check the environment;
		// stepping past it is the honest thing to do until it does, because
		// doing nothing at all reads as a key the launcher swallowed.
		m.moveFocus(1)
		return m, textinput.Blink
	}
	m.moveFocus(1)
	return m, textinput.Blink
}

// moveFocus walks the visible fields and then the scan button, which sits after
// the last field. It reports whether the focus stayed in the pane, so moving
// off the top can fall back to the list.
func (m *Model) moveFocus(delta int) bool {
	visible := m.detail.focusableFields()
	more := m.detail.hasMoreRow()

	// Positions: one per field, then the reveal row if there is one, then the
	// scan button.
	last := len(visible)
	if more {
		last++
	}

	pos := last // the button
	switch m.detail.spot {
	case spotMore:
		pos = len(visible)
	case spotField:
		pos = 0
		for i, idx := range visible {
			if idx == m.detail.form.Cursor() {
				pos = i
			}
		}
	}

	next := pos + delta
	if next < 0 {
		return false
	}
	if next > last {
		next = last
	}

	m.detail.storeCursorField()
	switch {
	case next == last:
		m.detail.spot = spotButton
		m.detail.input.Blur()
	case more && next == len(visible):
		m.detail.spot = spotMore
		m.detail.input.Blur()
	default:
		m.detail.spot = spotField
		m.detail.form.SetCursor(visible[next])
		m.ensureFieldVisible()
		m.detail.loadCursorField()
	}
	return true
}

// launch assembles the command and runs it. A form that is missing something
// required does not launch: focus moves to the field instead, because the
// alternative is tearing the screen down to show a usage error the launcher
// already knew was coming.
func (m Model) launch() (tea.Model, tea.Cmd) {
	c, ok := m.list.current()
	if !ok {
		return m, nil
	}
	if err := m.detail.form.Validate(); err != nil {
		m.lastErr = err.Error()
		m.focus = focusForm
		m.focusFirstMissing()
		return m, textinput.Blink
	}

	// The button says what is happening in the meantime; see the diButton arm
	// of renderDetailItem.
	if !m.launching.begin() {
		return m, nil
	}
	// Nothing else is batched in: the button says "Preparing…" and does not
	// animate, so the one repaint this Update already causes is all it needs.
	return m, prepareLaunchCmd(m, c, scanAction())
}

// launchPreparedMsg carries an assembled plan back to the event loop.
type launchPreparedMsg struct {
	// action is the verb the plan runs, which decides how it runs: a shell
	// takes the terminal, everything else goes to the background.
	action string
	plan   launchPlan
	err    error
}

// prepareLaunchCmd assembles the plan off the event loop.
//
// Everything launchArgs does is quick except one step, and that one has no
// bound at all: writing the credential to the OS keychain raises an
// authentication dialog when the keychain is locked, and the call does not
// return until the dialog is answered. Doing that from Update froze the whole
// TUI behind a dialog the launcher was not drawing -- alt screen, spinner and
// all -- so the launcher looked hung at exactly the moment it was asking for
// permission.
//
// m is a copy and launchArgs only reads it, so this is safe to run from a
// command. What it writes -- the inventory, a kubeconfig copy -- is returned in
// the plan rather than stored on the model.
func prepareLaunchCmd(m Model, c Connector, a Action) tea.Cmd {
	return func() tea.Msg {
		plan, err := m.launchArgs(c, a)
		return launchPreparedMsg{action: a.Name, plan: plan, err: err}
	}
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// A modal owns the body while it is up, and every zone computeLayout
	// registers belongs to a screen that is not drawn underneath it. The keys
	// have always respected that; the mouse did not, so a click landed on a row
	// nobody could see -- and a click where the scan button would have been
	// launched a scan from behind an open picker. The wheel is included: it
	// moved the selection under the box, which then rebuilt the form the box
	// was describing.
	if m.upstream.modal.open || m.export.open || m.picker.modal.open {
		return m, nil
	}

	l := computeLayout(m)

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.moveCursor(-1)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.moveCursor(1)
		return m, nil
	}

	// Dispatch on press rather than release: some terminals drop the release
	// event in cell-motion mode, which would swallow the click entirely.
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	switch z := l.HitZone(msg.X, msg.Y); z.Kind {
	case tui.ZoneEntry:
		if m.list.selectRow(z.Idx, m.listH()) {
			m.focus = focusList
			m.detail.input.Blur()
			m.list.search.Focus()
			m.syncSelection()
		}
		return m, textinput.Blink
	case tui.ZoneField:
		if z.Idx < 0 || z.Idx >= len(m.detail.form.Fields()) {
			return m, nil
		}
		// Clicking a toggle flips it; clicking anything else moves the cursor
		// there so it can be edited.
		if m.focus == focusForm {
			m.detail.storeCursorField()
		}
		m.detail.form.SetCursor(z.Idx)
		m.detail.spot = spotField
		if fd := &m.detail.form.Fields()[z.Idx]; fd.Kind == fieldBool {
			fd.Toggle()
		}
		m.focus = focusForm
		m.list.search.Blur()
		m.detail.loadCursorField()
		return m, tea.Batch(append(m.picker.pendingCmds(m.detail.form), textinput.Blink)...)
	case tui.ZoneUpstream:
		m.upstream.openModal()
		return m, nil
	case tui.ZoneMore:
		m.detail.showOptions = true
		// The row that was clicked has just stopped existing, so the keys
		// cannot stay on it. They are left where they are otherwise: a click
		// on the reveal row is not a claim about the button.
		m.detail.leaveMore()
		return m, nil
	case tui.ZoneRun:
		if m.focus == focusForm {
			m.detail.storeCursorField()
		}
		return m.launch()
	}
	return m, nil
}
