// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/cli/tui"
)

// The scanning screen is what replaced the handover.
//
// It exists because a background child is invisible: the scan bar `cnspec scan`
// draws is gated on stdout being a terminal, and a backgrounded child's is not,
// so without this the launcher would sit on a still screen for however long a
// cloud scan takes. What it draws comes from the NDJSON progress stream (see
// cli/progress/stream.go), folded into scanProgress.
//
// It obeys the same invariant as the rest of the launcher: one rendered element
// is exactly one terminal line, and no style carries a margin. See layout.go.

// scanBarWidth is how wide an asset's progress bar is drawn. Fixed rather than
// proportional so the columns to its right line up down the pane.
const scanBarWidth = 20

// view draws the scanning screen.
//
// spin is the spinner's current frame rather than the spinner itself: the
// spinner belongs to the launcher and the value pickers animate with the same
// one, so this screen borrows a frame instead of owning something that would
// then have to be ticked separately.
func (s scanState) view(l layout, spin string) string {
	sess := s.session
	if sess == nil {
		return tui.Panel(tui.StyleDim.Render("Scanning"), "", nil, l.Width, l.BodyH, tui.BorderColor(true))
	}

	snap := sess.prog.snapshot()
	iw := tui.InnerWidth(l.Width)
	avail := tui.InnerHeight(l.BodyH)

	content := make([]string, 0, avail)
	content = append(content, tui.Bar(tui.Truncate("$ cnspec "+shellJoin(sess.args), iw), iw, bandCommand))
	content = append(content, "")
	content = append(content, "  "+s.headline(snap, sess.elapsed(), spin))

	if sess.warn != "" {
		content = append(content, "", "  "+tui.StyleWarn.Render(tui.Truncate("! "+sess.warn, iw-2)))
	}

	if len(snap.Assets) > 0 {
		content = append(content, "", "  "+tui.StyleLabel.Render("ASSETS"))
		for _, a := range snap.Assets {
			if len(content) >= avail {
				break
			}
			content = append(content, scanAssetRow(a, iw))
		}
	}

	title := tui.StyleAccent.Render(sess.label)
	return tui.Panel(title, s.statusPill(snap), content, l.Width, l.BodyH, tui.BorderColor(true))
}

// statusPill says which of the three states the screen is in: running, reading
// the report the child just wrote, or -- briefly -- finished.
func (s scanState) statusPill(snap scanSnapshot) string {
	switch {
	case s.loading:
		return tui.Pill("reading report…", tui.ColInk, tui.ColCyan)
	case snap.ScanDone:
		return tui.Pill("finishing…", tui.ColInk, tui.ColGood)
	default:
		return tui.Pill("scanning", tui.ColInk, tui.ColAccent)
	}
}

// headline is the one line that says how far along the scan is.
//
// Assets are discovered while a scan runs, so the total moves; saying "3 of 7"
// when the seven is still growing is honest as long as the discovered count is
// there next to it, which is why both are shown rather than a percentage of a
// number that is not final.
func (s scanState) headline(snap scanSnapshot, elapsed time.Duration, spin string) string {
	var parts []string

	switch {
	case s.loading:
		parts = append(parts, tui.StyleText.Render("reading the report"))
	case snap.Total == 0:
		parts = append(parts, tui.StyleText.Render("connecting"))
	default:
		parts = append(parts, tui.StyleText.Render(fmt.Sprintf("%d of %d assets", snap.Done, snap.Total)))
	}
	if snap.Errored > 0 {
		parts = append(parts, tui.StyleBad.Render(fmt.Sprintf("%d errored", snap.Errored)))
	}
	if snap.NotApplicable > 0 {
		parts = append(parts, tui.StyleFaint.Render(fmt.Sprintf("%d not applicable", snap.NotApplicable)))
	}
	parts = append(parts, tui.StyleFaint.Render(formatElapsed(elapsed)))

	return spin + strings.Join(parts, tui.StyleFaint.Render(tui.HintSep))
}

// scanAssetRow is one asset: its state, its name, its platform, and how far
// through it is. It reads nothing but the asset, so it is a function.
func scanAssetRow(a scanAsset, w int) string {
	mark := tui.StyleFaint.Render("◌")
	switch {
	case a.State == progress.StateErrored:
		mark = tui.StyleBad.Render("✗")
	case a.State == progress.StateNotApplicable:
		mark = tui.StyleFaint.Render("–")
	case a.Done:
		mark = tui.StyleGood.Render("●")
	}

	name := a.Label()
	nameW := 24
	if nameW > w/3 {
		nameW = w / 3
	}
	if nameW < 8 {
		nameW = 8
	}

	right := scanBar(a.Percent)
	if a.Done && a.Score != "" {
		right = tui.PadRight(a.Score, scanBarWidth)
	}

	line := "  " + mark + " " + tui.PadRight(tui.Truncate(name, nameW), nameW) + " "
	if a.Platform != "" {
		line += tui.StyleFaint.Render(tui.PadRight(tui.Truncate(a.Platform, 12), 12)) + " "
	}
	line += tui.StyleDim.Render(right)
	return ansi.Truncate(line, w, "")
}

// scanBar draws a fixed-width progress bar. A bar rather than a number because
// several assets scan at once and a column of percentages is harder to read at
// a glance than a column of bars.
func scanBar(percent float64) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	filled := int(percent * scanBarWidth)
	return strings.Repeat("█", filled) + strings.Repeat("░", scanBarWidth-filled)
}

// formatElapsed renders a duration the way a person reads a stopwatch.
func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int(d.Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	return fmt.Sprintf("%dm%02ds", secs/60, secs%60)
}

// viewFooter is the scanning screen's key line. It offers exactly one thing,
// because there is exactly one thing to do while waiting.
func (s scanState) viewFooter(l layout) string {
	hints := " " + tui.Kbd("esc", "cancel scan")
	return tui.Truncate(hints, l.Width)
}
