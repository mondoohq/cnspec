// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// step is the current screen in the launcher wizard.
type step int

const (
	stepConnector step = iota // pick what to connect to
	stepAction                // pick what to do with it
	stepConfirm               // review + optional extra args, then launch
)

// Palette lifted from cli/theme/colors (ANSI-256) so the launcher matches the
// rest of the cnspec CLI without pulling termenv into lipgloss.
var (
	colPrimary   = lipgloss.Color("75")  // blue
	colSecondary = lipgloss.Color("170") // purple/pink (matches list selection)
	colDisabled  = lipgloss.Color("248") // gray
	colSuccess   = lipgloss.Color("78")  // green
	colCommand   = lipgloss.Color("44")  // cyan
	colWarn      = lipgloss.Color("214") // amber

	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(colPrimary)
	styleTagline  = lipgloss.NewStyle().Foreground(colDisabled)
	styleHeader   = lipgloss.NewStyle().Bold(true).Foreground(colSecondary).MarginTop(1)
	styleCursor   = lipgloss.NewStyle().Bold(true).Foreground(colSecondary)
	styleItem     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleItemDesc = lipgloss.NewStyle().Foreground(colDisabled)
	styleSelName  = lipgloss.NewStyle().Bold(true).Foreground(colSecondary)
	styleBadge    = lipgloss.NewStyle().Foreground(colSuccess)
	styleBadgeOff = lipgloss.NewStyle().Foreground(colDisabled)
	styleHelp     = lipgloss.NewStyle().Foreground(colDisabled).MarginTop(1)
	styleCommand  = lipgloss.NewStyle().Bold(true).Foreground(colCommand)
	styleCount    = lipgloss.NewStyle().Foreground(colDisabled)
	styleWarn     = lipgloss.NewStyle().Foreground(colWarn)
)

type rowKind int

const (
	rowHeader rowKind = iota
	rowItem
)

type displayRow struct {
	kind    rowKind
	text    string // header title
	connIdx int    // index into model.filtered, for rowItem
}

// Model is the bubbletea model for the interactive launcher.
type Model struct {
	catalog []Connector

	step   step
	search textinput.Model

	// derived list state (stepConnector)
	filtered   []Connector
	rows       []displayRow // header + item rows for rendering
	selectable []int        // indices into rows that are selectable items
	cursor     int          // index into selectable
	offset     int          // first visible row index (windowing)

	// selection
	connector Connector
	actions   []Action
	actionCur int

	// confirm step
	extra textinput.Model

	width  int
	height int

	// output
	result   []string // final args, e.g. ["scan", "aws", "..."]
	launched bool
	aborted  bool
}

// NewModel builds a launcher model over the given catalog.
func NewModel(catalog []Connector) Model {
	s := textinput.New()
	s.Placeholder = "type to filter (e.g. aws, ssh, kubernetes, database)"
	s.Prompt = "  search: "
	s.Focus()
	s.CharLimit = 64

	e := textinput.New()
	e.Prompt = "  args: "
	e.CharLimit = 256

	m := Model{
		catalog: catalog,
		search:  s,
		extra:   e,
		width:   80,
		height:  24,
	}
	m.refilter()
	return m
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

// refilter recomputes the filtered list and display rows from the search text.
func (m *Model) refilter() {
	query := strings.ToLower(strings.TrimSpace(m.search.Value()))
	tokens := strings.Fields(query)

	m.filtered = m.filtered[:0]
	for _, c := range m.catalog {
		if matchesTokens(c.searchText(), tokens) {
			m.filtered = append(m.filtered, c)
		}
	}

	// Build header + item rows grouped by category in canonical order.
	m.rows = m.rows[:0]
	m.selectable = m.selectable[:0]
	for _, cat := range categoryOrder {
		var idxs []int
		for i, c := range m.filtered {
			if c.Category == cat {
				idxs = append(idxs, i)
			}
		}
		if len(idxs) == 0 {
			continue
		}
		m.rows = append(m.rows, displayRow{kind: rowHeader, text: cat})
		for _, i := range idxs {
			m.selectable = append(m.selectable, len(m.rows))
			m.rows = append(m.rows, displayRow{kind: rowItem, connIdx: i})
		}
	}

	if m.cursor >= len(m.selectable) {
		m.cursor = len(m.selectable) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.offset = 0
}

func matchesTokens(haystack string, tokens []string) bool {
	for _, t := range tokens {
		if !strings.Contains(haystack, t) {
			return false
		}
	}
	return true
}

// visibleRows is how many list rows fit given the terminal height, leaving
// room for header, search, details and help.
func (m Model) visibleRows() int {
	v := m.height - 11
	if v < 4 {
		return 4
	}
	return v
}

func (m *Model) moveCursor(delta int) {
	if len(m.selectable) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.selectable) {
		m.cursor = len(m.selectable) - 1
	}
	m.ensureVisible()
}

// ensureVisible scrolls the window so the current selection is on screen.
func (m *Model) ensureVisible() {
	if len(m.selectable) == 0 {
		return
	}
	sel := m.selectable[m.cursor]
	vis := m.visibleRows()
	if sel < m.offset {
		m.offset = sel
		// pull in the category header just above, if any
		if m.offset > 0 && m.rows[m.offset-1].kind == rowHeader {
			m.offset--
		}
	}
	if sel >= m.offset+vis {
		m.offset = sel - vis + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m Model) currentConnector() (Connector, bool) {
	if len(m.selectable) == 0 || m.cursor < 0 || m.cursor >= len(m.selectable) {
		return Connector{}, false
	}
	row := m.rows[m.selectable[m.cursor]]
	return m.filtered[row.connIdx], true
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureVisible()
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case stepConnector:
			return m.updateConnector(msg)
		case stepAction:
			return m.updateAction(msg)
		case stepConfirm:
			return m.updateConfirm(msg)
		}
	}
	return m, nil
}

func (m Model) updateConnector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.aborted = true
		return m, tea.Quit
	case "up", "ctrl+p":
		m.moveCursor(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveCursor(1)
		return m, nil
	case "pgup":
		m.moveCursor(-m.visibleRows())
		return m, nil
	case "pgdown":
		m.moveCursor(m.visibleRows())
		return m, nil
	case "enter":
		conn, ok := m.currentConnector()
		if !ok {
			return m, nil
		}
		m.connector = conn
		m.actions = ActionsFor(conn.Name)
		m.actionCur = 0
		m.step = stepAction
		return m, nil
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.refilter()
	m.ensureVisible()
	return m, cmd
}

func (m Model) updateAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.aborted = true
		return m, tea.Quit
	case "esc", "left", "h":
		m.step = stepConnector
		return m, nil
	case "up", "k", "ctrl+p":
		if m.actionCur > 0 {
			m.actionCur--
		}
		return m, nil
	case "down", "j", "ctrl+n":
		if m.actionCur < len(m.actions)-1 {
			m.actionCur++
		}
		return m, nil
	case "enter", "right", "l":
		m.step = stepConfirm
		m.extra.SetValue("")
		m.extra.Placeholder = argPlaceholder(m.connector, m.actions[m.actionCur])
		m.extra.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.aborted = true
		return m, tea.Quit
	case "esc":
		m.step = stepAction
		m.extra.Blur()
		return m, nil
	case "enter":
		m.result = append([]string{m.actions[m.actionCur].Name, m.connector.Name}, tokenize(m.extra.Value())...)
		m.launched = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.extra, cmd = m.extra.Update(msg)
	return m, cmd
}

// argPlaceholder derives a helpful placeholder for the extra-args input from
// the connector's usage hint and the chosen action.
func argPlaceholder(c Connector, a Action) string {
	// The part of Use after the connector word is the argument hint, e.g.
	// "ssh user@host" -> "user@host".
	hint := strings.TrimSpace(strings.TrimPrefix(c.Use, c.Name))
	switch a.Name {
	case "run":
		return "-c \"" + c.Provider + ".resources\"  (an MQL query to run)"
	case "shell":
		if hint != "" {
			return hint + "  (optional)"
		}
		return "(optional flags)"
	default:
		if hint != "" {
			return hint
		}
		return "(optional flags, e.g. --discover all)"
	}
}
