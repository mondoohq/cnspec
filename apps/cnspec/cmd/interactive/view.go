// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.banner())

	switch m.step {
	case stepConnector:
		b.WriteString(m.viewConnector())
	case stepAction:
		b.WriteString(m.viewAction())
	case stepConfirm:
		b.WriteString(m.viewConfirm())
	}
	return b.String()
}

func (m Model) banner() string {
	title := styleTitle.Render("cnspec")
	tag := styleTagline.Render(" · security & compliance for your entire infrastructure")
	return "\n  " + title + tag + "\n"
}

func (m Model) viewConnector() string {
	var b strings.Builder

	count := styleCount.Render(fmt.Sprintf("%d of %d connectors", len(m.filtered), len(m.catalog)))
	b.WriteString("\n" + m.search.View())
	// right-align the count on the same visual block, keep it simple
	b.WriteString("  " + count + "\n")

	if len(m.selectable) == 0 {
		b.WriteString(styleItemDesc.Render("\n  no connectors match your search — try a different term\n"))
		b.WriteString(m.helpBar("↑/↓ move   type to filter   esc quit"))
		return b.String()
	}

	vis := m.visibleRows()
	end := m.offset + vis
	if end > len(m.rows) {
		end = len(m.rows)
	}

	selRow := -1
	if len(m.selectable) > 0 {
		selRow = m.selectable[m.cursor]
	}

	nameW := 16
	for i := m.offset; i < end; i++ {
		row := m.rows[i]
		if row.kind == rowHeader {
			b.WriteString(styleHeader.Render("  "+row.text) + "\n")
			continue
		}
		c := m.filtered[row.connIdx]
		cursor := "   "
		nameStyle := styleItem
		if i == selRow {
			cursor = styleCursor.Render("  ▸")
			nameStyle = styleSelName
		}
		name := nameStyle.Render(padRight(c.Name, nameW))
		desc := styleItemDesc.Render(truncate(c.Short, m.width-nameW-14))
		badge := ""
		if c.Installed {
			badge = "  " + styleBadge.Render("● installed")
		}
		b.WriteString(fmt.Sprintf("%s %s %s%s\n", cursor, name, desc, badge))
	}

	// scroll indicator
	if len(m.rows) > vis {
		b.WriteString(styleCount.Render(fmt.Sprintf("  … %d more (scroll with ↑/↓, PgUp/PgDn)\n", len(m.rows)-end+m.offset)))
	}

	b.WriteString(m.helpBar("↑/↓ move   type to filter   enter select   esc quit"))
	return b.String()
}

func (m Model) viewAction() string {
	var b strings.Builder
	b.WriteString("\n  target  " + styleSelName.Render(m.connector.Name))
	if m.connector.Short != "" {
		b.WriteString(styleItemDesc.Render("  " + m.connector.Short))
	}
	b.WriteString("\n")
	if !m.connector.Installed {
		b.WriteString(styleWarn.Render("  provider not installed yet — it downloads automatically on first use") + "\n")
	}

	b.WriteString(styleHeader.Render("  What would you like to do?") + "\n")
	for i, a := range m.actions {
		cursor := "   "
		nameStyle := styleItem
		if i == m.actionCur {
			cursor = styleCursor.Render("  ▸")
			nameStyle = styleSelName
		}
		b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
			nameStyle.Render(padRight(a.Name, 10)),
			styleItemDesc.Render(a.Short)))
	}

	b.WriteString(m.helpBar("↑/↓ move   enter select   esc back"))
	return b.String()
}

func (m Model) viewConfirm() string {
	var b strings.Builder
	action := m.actions[m.actionCur]

	b.WriteString(styleHeader.Render("  Review & launch") + "\n\n")
	b.WriteString("  " + styleItemDesc.Render(action.Short) + "\n\n")

	b.WriteString("  Add any target or flags below, then press enter:\n")
	b.WriteString(m.extra.View() + "\n\n")

	cmd := "cnspec " + action.Name + " " + m.connector.Name
	if extra := strings.TrimSpace(m.extra.Value()); extra != "" {
		cmd += " " + extra
	}
	b.WriteString("  " + styleItemDesc.Render("command:") + "\n")
	b.WriteString("  " + styleCommand.Render("$ "+cmd) + "\n")

	if !m.connector.Installed {
		b.WriteString("\n" + styleWarn.Render(fmt.Sprintf("  first run downloads the %q provider", m.connector.Provider)) + "\n")
	}

	b.WriteString(m.helpBar("enter launch   esc back   ctrl+c quit"))
	return b.String()
}

func (m Model) helpBar(text string) string {
	return "\n" + styleHelp.Render("  "+text)
}

func padRight(s string, w int) string {
	if lipgloss.Width(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

func truncate(s string, max int) string {
	if max < 4 {
		max = 4
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
