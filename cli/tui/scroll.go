// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

import (
	"fmt"
	"strconv"
)

// Scroll is the scroll offset of a pane. Every pane scrolls the same way because
// they all use this: the alternative is three subtly different clamps and one of
// them being off by one at the end of the list.
//
// A Scroll knows nothing about what it scrolls. The pane passes in the total
// number of rows and how many are visible, both of which it recomputes each
// frame, so a resize can never leave the pane scrolled past its end.
type Scroll struct {
	Off int
}

// Clamp returns the offset to render at, given the current geometry. It never
// mutates: call Apply to store the clamped value.
func (s Scroll) Clamp(total, visible int) int {
	maxOff := total - visible
	if maxOff < 0 {
		maxOff = 0
	}
	off := s.Off
	if off > maxOff {
		off = maxOff
	}
	if off < 0 {
		off = 0
	}
	return off
}

// Apply clamps the stored offset in place and returns it.
func (s *Scroll) Apply(total, visible int) int {
	s.Off = s.Clamp(total, visible)
	return s.Off
}

// Move scrolls by n rows and clamps.
func (s *Scroll) Move(n, total, visible int) {
	s.Off = Scroll{Off: s.Off + n}.Clamp(total, visible)
}

// EnsureVisible scrolls the minimum amount that brings row idx into view.
func (s *Scroll) EnsureVisible(idx, total, visible int) {
	if visible < 1 {
		visible = 1
	}
	if idx < s.Off {
		s.Off = idx
	} else if idx >= s.Off+visible {
		s.Off = idx - visible + 1
	}
	s.Apply(total, visible)
}

// Position renders the "12/240" indicator for a pane's title bar, or the plain
// total when everything fits.
func Position(cursor, total, visible int) string {
	if total == 0 {
		return "0"
	}
	if total <= visible {
		return strconv.Itoa(total)
	}
	return strconv.Itoa(cursor+1) + "/" + strconv.Itoa(total)
}

// ClampIndex holds a cursor inside a list of n rows. An empty list leaves the
// cursor at 0 rather than at -1, so a caller that clamps before it checks for
// emptiness does not index out of range.
func ClampIndex(i, n int) int {
	if i >= n {
		i = n - 1
	}
	if i < 0 {
		i = 0
	}
	return i
}

// Window is the slice of a list a box with room for rows lines can show, with
// the cursor kept near the middle of it so paging through the list does not pin
// it to an edge. It returns the half-open range [start, end).
//
// Both modal pickers in cnspec computed this, down to the same three clamps in
// the same order and the same "n more" marker underneath -- see MoreRow.
func Window(cursor, total, rows int) (start, end int) {
	start = cursor - rows/2
	if start > total-rows {
		start = total - rows
	}
	if start < 0 {
		start = 0
	}
	end = start + rows
	if end > total {
		end = total
	}
	return start, end
}

// MoreRow is the line a windowed list puts under itself to say how much of the
// list is out of view. It answers empty for a window that hid nothing, so a
// caller can append it unconditionally.
//
// The count is what is hidden, not what the list holds: a marker reading "12
// more" under a window showing all twelve is worse than no marker at all.
func MoreRow(hidden int) string {
	if hidden <= 0 {
		return ""
	}
	return StyleFaint.Render(fmt.Sprintf("  %d more", hidden))
}
