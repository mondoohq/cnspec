// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package tui holds the terminal primitives shared by cnspec's interactive
// programs -- the connector launcher and the report viewer. Nothing here knows
// about a particular model or view, so the two draw the same boxes, measure the
// same panes, scroll the same way and are colored out of the same palette, and
// end up looking like one program.
//
// It lives under cli/ because that is where this repo keeps reusable CLI
// components. It began under apps/cnspec/cmd/ holding only the box drawing and
// the pane split, and everything above geometry was written twice while it sat
// there: two palettes that disagreed on two colours, two text vocabularies, five
// scroll clamps, two justified bands, and a pad function defined locally in a
// file that already imported the package exporting it.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Box primitives. These draw their own borders rather than using lipgloss's
// Border(), for two reasons: a title and a status can be inlaid into the top
// edge, and the result is guaranteed to be exactly the requested number of
// lines, which is what the layout arithmetic depends on.

const (
	// BorderLines is the top and bottom edge; BorderCols is the left and right
	// edge plus one space of padding on each side.
	BorderLines = 2
	BorderCols  = 4
	// InnerX is the column where a panel's content starts, relative to the
	// panel's left edge: the border character plus one space of padding.
	InnerX = 2
)

// InnerWidth is the usable content width of a panel that is w columns wide.
func InnerWidth(w int) int {
	if w < BorderCols+1 {
		return 1
	}
	return w - BorderCols
}

// InnerHeight is the usable content height of a panel that is h lines tall.
func InnerHeight(h int) int {
	if h < BorderLines+1 {
		return 1
	}
	return h - BorderLines
}

// Panel draws a rounded box exactly w columns by h lines, with title inlaid at
// the top left and status at the top right. content supplies the inner lines;
// missing lines are blank and extra lines are dropped, so the caller cannot
// change the panel's height by accident.
func Panel(title, status string, content []string, w, h int, border lipgloss.Color) string {
	bs := lipgloss.NewStyle().Foreground(border)
	iw := InnerWidth(w)
	ih := InnerHeight(h)

	var b strings.Builder
	b.WriteString(PanelTop(title, status, w, bs))
	b.WriteString("\n")

	for i := 0; i < ih; i++ {
		line := ""
		if i < len(content) {
			line = content[i]
		}
		b.WriteString(bs.Render("│") + " " + PadRight(Truncate(line, iw), iw) + " " + bs.Render("│"))
		b.WriteString("\n")
	}

	b.WriteString(bs.Render("╰" + strings.Repeat("─", max(w-2, 0)) + "╯"))
	return b.String()
}

// PanelTop builds the top edge, fitting the title and status into it and
// falling back to a plain rule when there is not enough room for both.
func PanelTop(title, status string, w int, bs lipgloss.Style) string {
	rule := func(n int) string { return bs.Render(strings.Repeat("─", max(n, 0))) }

	left, right := "", ""
	if title != "" {
		left = bs.Render("─ ") + title + " "
	}
	if status != "" {
		right = " " + status + bs.Render(" ─")
	}

	fill := w - 2 - Width(left) - Width(right)
	if fill < 0 {
		// Not enough room for both; keep the title and drop the status.
		right = ""
		fill = w - 2 - Width(left)
	}
	if fill < 0 {
		return bs.Render("╭") + rule(w-2) + bs.Render("╮")
	}
	return bs.Render("╭") + left + rule(fill) + right + bs.Render("╮")
}

// Bar renders text as a full-width band, used for a selected row or a command
// line. The text must be unstyled: the band's own colors have to win.
func Bar(text string, w int, style lipgloss.Style) string {
	return style.Width(w).MaxWidth(w).Render(Truncate(text, w))
}

// Pill renders a small label with its own background, for status chips.
func Pill(text string, fg, bg lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true).Render(" " + text + " ")
}

// PadRight pads s with spaces so it occupies w cells, leaving it alone when it
// is already at least that wide.
func PadRight(s string, w int) string {
	if d := w - Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}
