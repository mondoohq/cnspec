// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestBandFillsExactlyTheWidth(t *testing.T) {
	require.Equal(t, "left      right", Band("left", "right", 15))
	require.Equal(t, "l         r", Band("l", "r", 11))
	for _, w := range []int{1, 3, 9, 10, 11, 40} {
		require.LessOrEqual(t, Width(Band("left side", "right", w)), w, "width %d", w)
	}
}

// The right-hand fragment is dropped whole rather than squeezed: a band whose
// two ends are touching says less than one showing only its left end, and the
// left end carries the identity.
func TestBandDropsTheRightSideWhenItCannotFit(t *testing.T) {
	require.Equal(t, "left", Band("left", "right", 4))
	require.NotContains(t, Band("left side here", "right", 14), "right")
}

// The renderer and the hit-tester have to agree on the column, which they can
// only do by asking the same function. -1 is how "nothing was drawn there" is
// said, so a click cannot land on a fragment that is not on screen.
func TestBandRightXAgreesWithBand(t *testing.T) {
	left, right := StyleDim.Render("✦ cnspec"), StyleAccent.Render("● connected")
	for w := 1; w <= 60; w++ {
		x := BandRightX(left, right, w)
		out := Band(left, right, w)
		if x < 0 {
			require.NotContains(t, ansi.Strip(out), ansi.Strip(right), "width %d dropped the right side", w)
			continue
		}
		require.Equal(t, x, Width(ansi.Strip(out))-Width(ansi.Strip(right)),
			"width %d: the right side must start where BandRightX says", w)
	}
}

// The middle fragment is a luxury: it appears only when the band has room to
// look laid out rather than crowded, and never at the cost of the other two.
func TestBand3DropsTheMiddleWhenTheBandIsTight(t *testing.T) {
	left, mid, right := "left", "middle", "right"
	require.Contains(t, Band3(left, mid, right, 40), mid)
	require.NotContains(t, Band3(left, mid, right, 16), mid)
	require.Equal(t, Band(left, right, 16), Band3(left, mid, right, 16))
	for w := 1; w <= 60; w++ {
		require.LessOrEqual(t, Width(Band3(left, mid, right, w)), w, "width %d", w)
	}
	require.Equal(t, Band(left, right, 40), Band3(left, "", right, 40),
		"no middle fragment is the two-fragment band")
}

// A band measures cells, not bytes, so a styled fragment must not push it over.
func TestBandMeasuresRenderedWidth(t *testing.T) {
	left := StyleAccent.Render(strings.Repeat("a", 10))
	right := StyleDim.Render(strings.Repeat("b", 10))
	require.Equal(t, 30, Width(ansi.Strip(Band(left, right, 30))))
}
