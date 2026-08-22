// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestClampIndex(t *testing.T) {
	require.Equal(t, 3, ClampIndex(3, 10))
	require.Equal(t, 9, ClampIndex(40, 10))
	require.Equal(t, 0, ClampIndex(-5, 10))
	// An empty list leaves the cursor at 0 rather than at -1: three callers
	// clamp before they check for emptiness, and -1 is an index.
	require.Equal(t, 0, ClampIndex(4, 0))
	require.Equal(t, 0, ClampIndex(0, 0))
}

// Window is what both modal pickers compute. The rules it has to keep are that
// the cursor is always inside the window, the window never runs off either end
// of the list, and it is never wider than the list.
func TestWindowKeepsTheCursorInside(t *testing.T) {
	for _, total := range []int{0, 1, 2, 5, 12, 40} {
		for _, rows := range []int{1, 2, 4, 10, 60} {
			for cursor := 0; cursor < total; cursor++ {
				start, end := Window(cursor, total, rows)
				require.GreaterOrEqual(t, start, 0, "total=%d rows=%d cursor=%d", total, rows, cursor)
				require.LessOrEqual(t, end, total, "total=%d rows=%d cursor=%d", total, rows, cursor)
				require.LessOrEqual(t, end-start, rows, "total=%d rows=%d cursor=%d", total, rows, cursor)
				require.GreaterOrEqual(t, cursor, start, "total=%d rows=%d cursor=%d", total, rows, cursor)
				require.Less(t, cursor, end, "total=%d rows=%d cursor=%d", total, rows, cursor)
			}
		}
	}
}

// The cursor sits near the middle rather than being pinned to an edge, which is
// what makes paging through a long list readable.
func TestWindowCentresTheCursor(t *testing.T) {
	start, end := Window(20, 100, 10)
	require.Equal(t, 15, start)
	require.Equal(t, 25, end)

	start, end = Window(0, 100, 10)
	require.Equal(t, 0, start, "the top of the list does not scroll off the top")
	require.Equal(t, 10, end)

	start, end = Window(99, 100, 10)
	require.Equal(t, 90, start, "nor the bottom off the bottom")
	require.Equal(t, 100, end)

	start, end = Window(2, 4, 10)
	require.Equal(t, 0, start, "a list shorter than the window is shown whole")
	require.Equal(t, 4, end)
}

// The marker counts what is hidden, not what the list holds. "12 more" under a
// window showing all twelve is worse than no marker at all.
func TestMoreRow(t *testing.T) {
	require.Equal(t, "", MoreRow(0))
	require.Equal(t, "", MoreRow(-3))
	require.Equal(t, "  4 more", ansi.Strip(MoreRow(4)))
}
