// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"go.mondoo.com/cnspec/cli/tui"
)

// The badge is clickable, and a click that lands anywhere else is worse than no
// click at all -- it would toggle where results go without the user meaning to.
// Geometry and rendering are computed from one function so they cannot drift;
// this proves it across widths, including one too narrow to draw the badge.
func TestBadgeZoneLandsOnTheBadge(t *testing.T) {
	for _, w := range []int{80, 100, 120, 200} {
		m := sized(newTestModel(), w, 30)
		l := computeLayout(m)
		header := ansi.Strip(strings.Split(m.View(), "\n")[0])
		var found bool
		for _, z := range l.Zones {
			if z.Kind != tui.ZoneUpstream {
				continue
			}
			found = true
			// Zones are display columns; a Go string index is bytes, and the
			// header's ✦ is 3 bytes wide but 1 column.
			seg := colSlice(header, z.Rect.X, z.Rect.W)
			t.Logf("w=%3d zone x=%d w=%d -> %q", w, z.Rect.X, z.Rect.W, seg)
			if !strings.Contains(seg, "connected") && !strings.Contains(seg, "incognito") {
				t.Errorf("w=%d: zone does not cover the badge, covers %q", w, seg)
			}
		}
		// A width too narrow to draw the badge must register no zone: a click
		// on a badge that was never rendered would change where results go
		// without the user having seen the control.
		drawn := strings.Contains(header, "connected") || strings.Contains(header, "incognito")
		if found != drawn {
			t.Errorf("w=%d: zone present = %v but badge drawn = %v", w, found, drawn)
		}
	}
}

// colSlice takes w display columns starting at column x.
func colSlice(s string, x, w int) string {
	var out []rune
	col := 0
	for _, r := range s {
		rw := tui.Width(string(r))
		if col >= x && col < x+w {
			out = append(out, r)
		}
		col += rw
	}
	return string(out)
}
