// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AI-native palette. Magenta/violet accent for the AI-forward feel, cyan for
// the sparkle, gold for skills, matching the cnspec CLI's ANSI-256 theme.
var (
	colAccent   = lipgloss.Color("177") // violet/magenta (AI)
	colAccent2  = lipgloss.Color("51")  // cyan sparkle
	colGold     = lipgloss.Color("220") // skills
	colPrimary  = lipgloss.Color("75")  // blue
	colText     = lipgloss.Color("252")
	colDim      = lipgloss.Color("244")
	colGreen    = lipgloss.Color("78")
	colWarn     = lipgloss.Color("214")
	colFocusBox = lipgloss.Color("240")

	styleSpark    = lipgloss.NewStyle().Foreground(colAccent2).Bold(true)
	styleBrand    = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	styleTagline  = lipgloss.NewStyle().Foreground(colDim)
	styleChips    = lipgloss.NewStyle().Foreground(colPrimary)
	styleSection  = lipgloss.NewStyle().Foreground(colAccent).Bold(true).MarginTop(1)
	styleHeader   = lipgloss.NewStyle().Foreground(colDim).Bold(true).MarginTop(1)
	styleCursor   = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	styleItem     = lipgloss.NewStyle().Foreground(colText)
	styleItemDesc = lipgloss.NewStyle().Foreground(colDim)
	styleSelName  = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	styleAccentGl = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	styleGoldGl   = lipgloss.NewStyle().Foreground(colGold).Bold(true)
	stylePrimGl   = lipgloss.NewStyle().Foreground(colPrimary)
	styleBadge    = lipgloss.NewStyle().Foreground(colGreen)
	styleHelp     = lipgloss.NewStyle().Foreground(colDim).MarginTop(1)
	styleCommand  = lipgloss.NewStyle().Foreground(colAccent2).Bold(true)
	styleCount    = lipgloss.NewStyle().Foreground(colDim)
	styleWarn     = lipgloss.NewStyle().Foreground(colWarn)
	styleKbd      = lipgloss.NewStyle().Foreground(colAccent2)

	styleInstallBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colFocusBox).
			Foreground(colDim).
			Padding(0, 1).
			MarginTop(1).
			MarginLeft(2)
)

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.banner())

	switch m.step {
	case stepHome:
		b.WriteString(m.viewHome())
	case stepConnector:
		b.WriteString(m.viewConnector())
	case stepAction:
		b.WriteString(m.viewAction())
	case stepConfirm:
		b.WriteString(m.viewConfirm())
	case stepSkills:
		b.WriteString(m.viewSkills())
	}
	return b.String()
}

func (m Model) banner() string {
	line1 := "  " + styleSpark.Render("✦") + " " + styleBrand.Render("cnspec") +
		"  " + styleTagline.Render("AI-native security for your entire stack")
	line2 := "  " + styleChips.Render("cloud") + styleTagline.Render(" · ") +
		styleChips.Render("SaaS") + styleTagline.Render(" · ") +
		styleAccentGl.Render("AI services") + styleTagline.Render(" · ") +
		styleChips.Render("Kubernetes") + styleTagline.Render(" · ") +
		styleChips.Render("databases") + styleTagline.Render(" · ") +
		styleTagline.Render("70+ connectors")
	return "\n" + line1 + "\n" + line2 + "\n"
}

func (m Model) viewHome() string {
	var b strings.Builder
	b.WriteString(styleSection.Render("  What do you want to secure?") + "\n\n")

	for i, t := range m.homeTiles {
		selected := i == m.homeCursor
		cursor := "   "
		if selected {
			cursor = styleCursor.Render("  ▸")
		}

		iconStyle := stylePrimGl
		titleStyle := styleItem
		switch {
		case t.key == tileSkills:
			iconStyle, titleStyle = styleGoldGl, styleGoldGl
		case t.highlight:
			iconStyle, titleStyle = styleAccentGl, styleAccentGl
		}
		if selected {
			titleStyle = styleSelName
		}

		title := titleStyle.Render(t.title)
		count := ""
		if t.count > 0 {
			count = styleCount.Render(fmt.Sprintf("  (%d)", t.count))
		}
		blurb := styleItemDesc.Render(t.blurb)
		b.WriteString(fmt.Sprintf("%s %s  %s%s\n", cursor, iconStyle.Render(t.icon), title, count))
		b.WriteString(fmt.Sprintf("       %s\n", blurb))
	}

	b.WriteString(m.helpBar("↑/↓ move   enter open   q quit"))
	return b.String()
}

func (m Model) viewConnector() string {
	var b strings.Builder

	scope := "all connectors"
	if m.categoryFilter != "" {
		scope = categoryIcon[m.categoryFilter] + " " + homeTitle(m.categoryFilter)
	}
	b.WriteString("\n  " + styleItemDesc.Render(scope) + "  " +
		styleCount.Render(fmt.Sprintf("%d of %d", len(m.filtered), len(m.catalog))) + "\n")
	b.WriteString(m.search.View() + "\n")

	if len(m.selectable) == 0 {
		b.WriteString(styleItemDesc.Render("\n  no connectors match — try a different term\n"))
		b.WriteString(m.helpBar("type to filter   esc back"))
		return b.String()
	}

	vis := m.visibleRows()
	end := m.offset + vis
	if end > len(m.rows) {
		end = len(m.rows)
	}
	selRow := m.selectable[m.cursor]

	nameW := 16
	for i := m.offset; i < end; i++ {
		row := m.rows[i]
		if row.kind == rowHeader {
			icon := categoryIcon[row.text]
			b.WriteString(styleHeader.Render("  "+icon+"  "+homeTitle(row.text)) + "\n")
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
		desc := styleItemDesc.Render(truncate(c.Short, m.width-nameW-16))
		badge := ""
		if c.Installed {
			badge = "  " + styleBadge.Render("● installed")
		}
		b.WriteString(fmt.Sprintf("%s %s %s%s\n", cursor, name, desc, badge))
	}

	if len(m.rows) > vis {
		b.WriteString(styleCount.Render(fmt.Sprintf("  … %d more (↑/↓, PgUp/PgDn)\n", len(m.rows)-(end-m.offset))))
	}

	b.WriteString(m.helpBar("↑/↓ move   type to filter   enter select   esc back"))
	return b.String()
}

func (m Model) viewAction() string {
	var b strings.Builder
	icon := categoryIcon[m.connector.Category]
	b.WriteString("\n  " + styleItemDesc.Render("target") + "  " +
		styleAccentGl.Render(icon+" "+m.connector.Name))
	if m.connector.Short != "" {
		b.WriteString(styleItemDesc.Render("  " + m.connector.Short))
	}
	b.WriteString("\n")
	if !m.connector.Installed {
		b.WriteString(styleWarn.Render("  provider not installed yet — it downloads automatically on first use") + "\n")
	}

	b.WriteString(styleSection.Render("  What would you like to do?") + "\n")
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

	b.WriteString(styleSection.Render("  Review & launch") + "\n\n")
	b.WriteString("  " + styleItemDesc.Render(action.Short) + "\n\n")

	b.WriteString("  " + styleItemDesc.Render("Add a target or flags, then press enter:") + "\n")
	b.WriteString(m.extra.View() + "\n\n")

	cmd := "cnspec " + action.Name + " " + m.connector.Name
	if extra := strings.TrimSpace(m.extra.Value()); extra != "" {
		cmd += " " + extra
	}
	b.WriteString("  " + styleItemDesc.Render("command") + "\n")
	b.WriteString("  " + styleCommand.Render("$ "+cmd) + "\n")

	if !m.connector.Installed {
		b.WriteString("\n" + styleWarn.Render(fmt.Sprintf("  first run downloads the %q provider", m.connector.Provider)) + "\n")
	}

	b.WriteString(m.helpBar("enter launch   esc back   ctrl+c quit"))
	return b.String()
}

func (m Model) viewSkills() string {
	var b strings.Builder
	b.WriteString(styleSection.Render("  ✳ Skills") + "  " +
		styleItemDesc.Render("AI-agent skills for MQL & policy work") + "\n")
	b.WriteString("  " + styleItemDesc.Render("Plug cnspec expertise into Claude Code, Codex, Gemini CLI, and Cursor.") + "\n\n")

	for i, s := range Skills {
		cursor := "   "
		nameStyle := styleItem
		if i == m.skillCursor {
			cursor = styleCursor.Render("  ▸")
			nameStyle = styleGoldGl
		}
		b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
			nameStyle.Render(padRight(s.Name, 14)),
			styleItemDesc.Render(s.Summary)))
	}

	if m.skillCursor >= 0 && m.skillCursor < len(Skills) {
		b.WriteString("\n")
		for _, line := range Skills[m.skillCursor].Detail {
			b.WriteString("  " + styleItemDesc.Render(line) + "\n")
		}
	}

	install := styleGoldGl.Render("Add to your agent") + "\n" + strings.Join(skillInstall, "\n")
	b.WriteString(styleInstallBox.Render(install) + "\n")

	b.WriteString(m.helpBar("↑/↓ browse   esc back"))
	return b.String()
}

func (m Model) helpBar(text string) string {
	// Highlight the key names in the help line for an app-like feel.
	parts := strings.Split(text, "   ")
	for i, p := range parts {
		fields := strings.SplitN(p, " ", 2)
		if len(fields) == 2 {
			parts[i] = styleKbd.Render(fields[0]) + " " + styleItemDesc.Render(fields[1])
		} else {
			parts[i] = styleItemDesc.Render(p)
		}
	}
	return "\n" + styleHelp.Render("  "+strings.Join(parts, styleItemDesc.Render("   ")))
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
