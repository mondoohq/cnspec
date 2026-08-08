// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// step is the current screen in the launcher wizard.
type step int

const (
	stepHome      step = iota // use-case landing screen
	stepConnector             // pick what to connect to
	stepAction                // pick what to do with it
	stepConfirm               // review + optional extra args, then launch
	stepSkills                // AI-agent skills highlight
)

type Model struct {
	catalog   []Connector
	homeTiles []homeTile

	step   step
	search textinput.Model

	// home
	homeCursor int

	// derived list state (stepConnector)
	categoryFilter string // when navigating in from a use-case tile
	filtered       []Connector
	rows           []displayRow
	selectable     []int
	cursor         int
	offset         int

	// selection
	connector Connector
	actions   []Action
	actionCur int

	// confirm step
	extra textinput.Model

	// skills step
	skillCursor int

	width  int
	height int

	// output
	result   []string
	launched bool
	aborted  bool
}

type rowKind int

const (
	rowHeader rowKind = iota
	rowItem
)

type displayRow struct {
	kind    rowKind
	text    string
	connIdx int
}

// NewModel builds a launcher model over the given catalog.
func NewModel(catalog []Connector) Model {
	s := textinput.New()
	s.Placeholder = "type to filter (e.g. aws, ssh, kubernetes, openai)"
	s.Prompt = "  ⌕ "
	s.CharLimit = 64

	e := textinput.New()
	e.Prompt = "  ❯ "
	e.CharLimit = 256

	m := Model{
		catalog:   catalog,
		homeTiles: buildHomeTiles(catalog),
		search:    s,
		extra:     e,
		width:     84,
		height:    30,
	}
	m.refilter()
	return m
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

// refilter recomputes the filtered list and display rows from the search text
// and the active category filter.
func (m *Model) refilter() {
	query := strings.ToLower(strings.TrimSpace(m.search.Value()))
	tokens := strings.Fields(query)

	m.filtered = m.filtered[:0]
	for _, c := range m.catalog {
		if m.categoryFilter != "" && c.Category != m.categoryFilter {
			continue
		}
		if matchesTokens(c.searchText(), tokens) {
			m.filtered = append(m.filtered, c)
		}
	}

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

func (m Model) visibleRows() int {
	v := m.height - 12
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

func (m *Model) ensureVisible() {
	if len(m.selectable) == 0 {
		return
	}
	sel := m.selectable[m.cursor]
	vis := m.visibleRows()
	if sel < m.offset {
		m.offset = sel
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
		case stepHome:
			return m.updateHome(msg)
		case stepConnector:
			return m.updateConnector(msg)
		case stepAction:
			return m.updateAction(msg)
		case stepConfirm:
			return m.updateConfirm(msg)
		case stepSkills:
			return m.updateSkills(msg)
		}
	}
	return m, nil
}

func (m Model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc", "q":
		m.aborted = true
		return m, tea.Quit
	case "up", "k", "ctrl+p":
		if m.homeCursor > 0 {
			m.homeCursor--
		}
		return m, nil
	case "down", "j", "ctrl+n":
		if m.homeCursor < len(m.homeTiles)-1 {
			m.homeCursor++
		}
		return m, nil
	case "enter", "right", "l":
		return m.selectHomeTile()
	}
	return m, nil
}

func (m Model) selectHomeTile() (tea.Model, tea.Cmd) {
	if m.homeCursor < 0 || m.homeCursor >= len(m.homeTiles) {
		return m, nil
	}
	tile := m.homeTiles[m.homeCursor]
	switch tile.key {
	case tileSkills:
		m.step = stepSkills
		m.skillCursor = 0
		return m, nil
	case tileSearch:
		m.categoryFilter = ""
	default:
		m.categoryFilter = tile.key
	}
	m.step = stepConnector
	m.search.SetValue("")
	m.search.Focus()
	m.cursor = 0
	m.refilter()
	m.ensureVisible()
	return m, textinput.Blink
}

func (m Model) updateConnector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.aborted = true
		return m, tea.Quit
	case "esc":
		m.step = stepHome
		m.search.Blur()
		return m, nil
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
		m.search.Focus()
		return m, textinput.Blink
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

func (m Model) updateSkills(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.aborted = true
		return m, tea.Quit
	case "esc", "left", "h":
		m.step = stepHome
		return m, nil
	case "up", "k", "ctrl+p":
		if m.skillCursor > 0 {
			m.skillCursor--
		}
		return m, nil
	case "down", "j", "ctrl+n":
		if m.skillCursor < len(Skills)-1 {
			m.skillCursor++
		}
		return m, nil
	}
	return m, nil
}

// argPlaceholder derives a helpful placeholder for the extra-args input.
func argPlaceholder(c Connector, a Action) string {
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
