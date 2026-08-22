// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
)

// Model is the viewer's bubbletea model: the frame around the panes.
//
// It owns the terminal size, the focus, the routing and the chrome. It does not
// own a single row of content -- every line inside the header band and the two
// panels comes from a Pane.
//
// The panes are pointers held by the frame, so Model stays copyable the way
// bubbletea expects while a pane's own state (its cursor, its scroll offset)
// survives the copy. State is shared for the same reason.
type Model struct {
	width, height int

	// leaveLabel is what the footer calls q -- "quit" standalone, "back" when
	// embedded in another program. See Embedded.
	leaveLabel string

	state  *State
	header Pane
	tree   Pane
	detail Pane

	// showHelp expands the footer into the full key map.
	showHelp bool

	// export is the format picker, when it is open. It is not a pane: it takes
	// the whole body and every key until it closes. See export.go.
	export exportModal
}

// NewModel builds a viewer for a report. Panes come from the registry (see
// RegisterTree and friends); passing a pane here overrides the registered one
// for its slot, which is what tests and embedders use.
func NewModel(report *reportmodel.Report, panes ...Pane) Model {
	st := NewState(report)
	m := Model{
		state:      st,
		leaveLabel: "quit",
		header:     buildPane(PaneHeader, st),
		tree:       buildPane(PaneTree, st),
		detail:     buildPane(PaneDetail, st),
	}
	for _, p := range panes {
		if p == nil {
			continue
		}
		switch p.ID() {
		case PaneHeader:
			m.header = p
		case PaneTree:
			m.tree = p
		case PaneDetail:
			m.detail = p
		}
	}
	m.state.Focus = m.firstFocusable()
	return m
}

// State exposes the shared state, so a caller that built the model can inspect
// or preselect through it.
func (m Model) State() *State { return m.state }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// panes returns the three panes in focus-cycle order: the body first, because
// that is where the user starts, then the header.
func (m Model) panes() []Pane {
	return []Pane{m.tree, m.detail, m.header}
}

func (m Model) pane(id PaneID) Pane {
	for _, p := range m.panes() {
		if p != nil && p.ID() == id {
			return p
		}
	}
	return nil
}

func (m Model) firstFocusable() PaneID {
	for _, p := range m.panes() {
		if p != nil && p.Focusable() {
			return p.ID()
		}
	}
	return PaneNone
}

// headerHeight asks the header pane how tall it wants to be. computeLayout
// clamps the answer.
func (m Model) headerHeight() int {
	if sp, ok := m.header.(SizedPane); ok {
		return sp.HeightFor(m.state, m.width)
	}
	return tui.HeaderLines
}

// layout is the frame's geometry for the current size and focus.
func (m Model) layout() Layout {
	return computeLayout(m.width, m.height, m.headerHeight(), m.state.Focus)
}

// frame is one complete pass: the geometry plus what each visible pane drew into
// it. Render is called exactly once per pane per frame, and both the renderer
// and the hit-tester read the result, which is what keeps a click on a row
// landing on the thing that row drew.
type frame struct {
	layout   Layout
	rendered map[PaneID]Render
	zones    []Zone
}

func (m Model) build() frame {
	f := frame{layout: m.layout(), rendered: map[PaneID]Render{}}
	for _, p := range m.panes() {
		if p == nil {
			continue
		}
		rect, visible := f.layout.ContentFor(p.ID())
		if !visible || rect.W < 1 || rect.H < 1 {
			continue
		}
		r := p.Render(m.state, rect)
		for i := range r.Zones {
			r.Zones[i].Pane = p.ID()
		}
		f.rendered[p.ID()] = r
		f.zones = append(f.zones, r.Zones...)
	}
	return f
}

// hit returns the topmost zone containing (x, y). Zones are appended in render
// order, so walk in reverse and let a later one win.
func (f frame) hit(x, y int) (Zone, bool) {
	for i := len(f.zones) - 1; i >= 0; i-- {
		if f.zones[i].Rect.Hit(x, y) {
			return f.zones[i], true
		}
	}
	return Zone{}, false
}

// Update implements tea.Model.
//
// Routing, in order:
//
//  1. ctrl+c always quits.
//  2. An open export modal takes every remaining key, and no pane sees any of
//     them. It is the one thing in the viewer that is modal in the literal
//     sense.
//  3. A resize, and any message that is neither a key nor a mouse event, is
//     broadcast to every pane.
//  4. A key claimed by a pane (Pane.Claims) moves focus to that pane and is
//     delivered there.
//  5. Otherwise the focused pane sees the key first.
//  6. A key no pane consumed falls through to the frame's own bindings, which is
//     how "esc" can clear a search in the pane that has one and quit everywhere
//     else.
//
// A mouse click is hit-tested against the zones of the frame just rendered: the
// owning pane takes focus and receives a ClickMsg. A scroll wheel goes to the
// pane under the pointer without moving focus, because scrolling something you
// are only looking at should not steal the keyboard.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, m.broadcast(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case ExportDoneMsg:
		return m.exportDone(msg)

	case copyDoneMsg:
		return m.copyDone(msg)

	case openDoneMsg:
		return m.openDone(msg)
	}

	return m, m.broadcast(msg)
}

// broadcast hands a message to every pane and batches whatever they return.
func (m Model) broadcast(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for _, p := range m.panes() {
		if p == nil {
			continue
		}
		if cmd, _ := p.Update(m.state, msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	// Any key dismisses whatever notice is showing.
	m.state.Notice = ""

	// The export picker is modal: while it is up it is the only thing keys can
	// mean, so no pane and none of the frame's own bindings see them.
	if m.export.open {
		return m.exportKey(msg)
	}

	// A pane that claims the key gets focus and the key, unless it already has
	// focus, in which case the ordinary path below delivers it once.
	if p := m.claimant(key); p != nil && p.ID() != m.state.Focus {
		m.state.Focus = p.ID()
		if cmd, handled := p.Update(m.state, msg); handled {
			return m, cmd
		}
	}

	if p := m.pane(m.state.Focus); p != nil {
		if cmd, handled := p.Update(m.state, msg); handled {
			return m, cmd
		}
	}

	switch key {
	case "tab":
		m.focusNext(1)
	case "shift+tab":
		m.focusNext(-1)
	case "e":
		// Free everywhere else: no pane switches on it, the header claims only
		// "/" and "f", and while the header's search field is open it consumes
		// every rune before the frame is reached, so typing an e into a search
		// still types an e.
		return m.openExport()
	case "y":
		// The yank key, and free: no pane switches on it -- the tree owns the
		// arrows, hjkl, g/G, enter, space and s, the detail pane the scroll set,
		// the header / f and c, and export e. It is handled here rather than
		// claimed by the detail pane so that it also copies while the tree has
		// focus, and, like e, it never reaches the frame while the header's
		// search field is open, because that field consumes every rune.
		return m.copySnippet()
	case "n":
		// Aiming y, and free for the same reasons it is: no pane switches on the
		// bare runes -- the scroll sets take ctrl+n and ctrl+p, and the modal is
		// modal. [ and ] would have been the other obvious pair, and are not
		// usable: both need AltGr on a German keyboard.
		return m.armCopy(1)
	case "p":
		return m.armCopy(-1)
	case "?":
		m.showHelp = !m.showHelp
	case "esc", "q":
		return m, tea.Quit
	}
	return m, nil
}

// claimant is the pane that claims a key regardless of focus.
func (m Model) claimant(key string) Pane {
	for _, p := range m.panes() {
		if p == nil {
			continue
		}
		for _, c := range p.Claims() {
			if c == key {
				return p
			}
		}
	}
	return nil
}

// focusNext moves focus by n places through the focusable panes.
func (m *Model) focusNext(n int) {
	var ids []PaneID
	for _, p := range m.panes() {
		if p != nil && p.Focusable() {
			ids = append(ids, p.ID())
		}
	}
	if len(ids) == 0 {
		m.state.Focus = PaneNone
		return
	}
	at := 0
	for i, id := range ids {
		if id == m.state.Focus {
			at = i
		}
	}
	next := ((at+n)%len(ids) + len(ids)) % len(ids)
	m.state.Focus = ids[next]
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// The modal covers the body, but the panes underneath it still render and
	// still publish their zones -- so without this, a click would select a row
	// the user cannot see. The picker is keyboard-driven; while it is up the
	// mouse does nothing.
	if m.export.open {
		return m, nil
	}

	f := m.build()

	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		// The wheel scrolls what the pointer is over and leaves focus alone.
		if p := m.pane(f.layout.PaneAt(msg.X, msg.Y)); p != nil {
			cmd, _ := p.Update(m.state, msg)
			return m, cmd
		}
		return m, nil
	}

	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	z, ok := f.hit(msg.X, msg.Y)
	if !ok {
		return m, nil
	}
	p := m.pane(z.Pane)
	if p == nil {
		return m, nil
	}
	if p.Focusable() {
		m.state.Focus = z.Pane
	}
	m.state.Notice = ""
	cmd, _ := p.Update(m.state, ClickMsg{Zone: z, Mouse: msg})
	return m, cmd
}

// View implements tea.Model. The result is always exactly m.height lines of at
// most m.width cells: the panes are sized in exact lines, and the final Fit is
// the belt to that pair of braces.
func (m Model) View() string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	f := m.build()
	l := f.layout

	lines := make([]string, 0, m.height)
	lines = append(lines, tui.Fit(m.rendered(f, PaneHeader), l.HeaderH)...)
	lines = append(lines, tui.Fit(m.body(f), l.BodyH)...)
	lines = append(lines, m.footer(l))

	lines = tui.Fit(lines, m.height)
	for i, ln := range lines {
		lines[i] = tui.Truncate(ln, m.width)
	}
	return strings.Join(lines, "\n")
}

func (m Model) rendered(f frame, id PaneID) []string {
	r, ok := f.rendered[id]
	if !ok {
		return nil
	}
	return r.Lines
}

// body draws the panels. tui.Panel is exactly the size it is given, so the two
// columns are the same height by construction and JoinHorizontal cannot grow
// the body.
func (m Model) body(f frame) []string {
	l := f.layout

	// The export picker takes the body whole, across both columns: it is a
	// question about the report, not about either pane, and a box floated over
	// the panels would have to slice styled lines in half to do it.
	if m.export.open {
		return m.export.render(exportBaseName(m.state.Report),
			tui.Rect{X: 0, Y: l.BodyTop, W: l.Width, H: l.BodyH})
	}

	panelOf := func(id PaneID, rect tui.Rect) string {
		r := f.rendered[id]
		title, status := r.Title, r.Status
		focused := m.state.Focus == id
		if focused {
			title = tui.StyleAccent.Render(title)
		} else {
			title = tui.StyleDim.Render(title)
		}
		if status != "" {
			status = tui.StyleFaint.Render(status)
		}
		return tui.Panel(title, status, r.Lines, rect.W, rect.H, tui.BorderColor(focused))
	}

	switch {
	case l.TwoPane:
		return strings.Split(lipgloss.JoinHorizontal(lipgloss.Top,
			panelOf(PaneTree, l.TreePanel), " ", panelOf(PaneDetail, l.DetailPanel)), "\n")
	case l.ShowDetail:
		return strings.Split(panelOf(PaneDetail, l.DetailPanel), "\n")
	default:
		return strings.Split(panelOf(PaneTree, l.TreePanel), "\n")
	}
}

// footer is the hint line: a notice if there is one, otherwise the focused
// pane's key hints plus the frame's own.
func (m Model) footer(l Layout) string {
	if m.state.Notice != "" {
		return tui.StyleDim.Render(tui.Truncate(" "+tui.Clean(m.state.Notice), l.Width))
	}

	// While the picker is up it owns every key, so the pane's hints and the
	// frame's would both be advertising keys that do nothing.
	if m.export.open {
		return m.hintLine(exportHints(), l.Width)
	}

	hints := []Hint{}
	if p := m.pane(m.state.Focus); p != nil {
		hints = append(hints, p.Hints(m.state)...)
	}
	hints = append(hints, m.frameHints()...)
	return m.hintLine(hints, l.Width)
}

// hintLine renders a set of bindings as the single footer row.
func (m Model) hintLine(hints []Hint, width int) string {
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, tui.Kbd(h.Key, h.Label))
	}
	return tui.Truncate(" "+strings.Join(parts, tui.HintSep), width)
}

// frameHints are the bindings the frame itself owns, and are shown after
// whatever the focused pane contributes.
//
// n and p are the frame's, and the ? list says so. The compact list leaves them
// to the detail pane's own hints instead, for room: the line is priority-ordered
// left to right and the frame's keys are last, so a sixth entry here pushes ?
// and q off an 80- or 120-column terminal -- and those are the two keys a reader
// who is lost needs most. The pane that draws the band is also the pane where
// aiming means anything, so that is where the short form belongs.
//
// ? is labelled "help" rather than "keys" because every other label on this line
// names what the key gets you rather than how it works, and a reader who is
// stuck is looking for help, not for a list of keys -- "keys" is the answer to a
// question they have not thought to ask yet. Its expanded form stays "less": the
// ? list is the one that already truncates at 120 columns, so it names the
// direction of the toggle rather than spending five more cells repeating the
// noun in front of ? and q.
func (m Model) frameHints() []Hint {
	if m.showHelp {
		return []Hint{
			{Key: "tab", Label: "next pane"},
			{Key: "shift+tab", Label: "previous pane"},
			{Key: "n/p", Label: "next / previous code block"},
			{Key: "y", Label: "copy the highlighted code block"},
			{Key: "e", Label: "export report"},
			{Key: "?", Label: "less"},
			{Key: "q", Label: m.leaveLabel},
		}
	}
	return []Hint{
		{Key: "tab", Label: "pane"},
		{Key: "y", Label: "copy"},
		{Key: "e", Label: "export"},
		{Key: "?", Label: "help"},
		{Key: "q", Label: m.leaveLabel},
	}
}

// Embedded says the viewer is running inside another program, which changes
// only what leaving is called: q still ends the view, but it hands control back
// rather than ending the process, and a footer that says "quit" would be
// telling the user something that does not happen.
func (m Model) Embedded() Model {
	m.leaveLabel = "back"
	return m
}
