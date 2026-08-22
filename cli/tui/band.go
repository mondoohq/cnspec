// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

import "strings"

// A band is a full-width line with something at each end of it: the launcher's
// header, the viewer's summary row, the line that says what is being filtered.
//
// There were three implementations of this arithmetic and two of them were line
// for line identical. That is cheap to let happen and expensive to leave: the
// hit-tester for the launcher's header has to agree with the renderer about the
// column the right-hand fragment starts in, and it can only do that if both ask
// the same function. BandRightX is that agreement.

// Band lays a left and a right fragment across w cells, dropping the right one
// when there is not room for both. The result is always at most w cells wide.
//
// The right fragment is dropped whole rather than squeezed. A band whose two
// ends are touching says less than one that shows only its left end, and the
// left end is the side that carries the identity.
func Band(left, right string, w int) string {
	gap := w - Width(left) - Width(right)
	if gap < 1 {
		return Truncate(left, w)
	}
	return left + strings.Repeat(" ", gap) + right
}

// BandRightX is the column Band draws right at, or -1 when Band would drop it.
// A caller registering a click zone over the right-hand fragment reads its
// position from here, so a click can never land on something that was not
// drawn.
func BandRightX(left, right string, w int) int {
	if w-Width(left)-Width(right) < 1 {
		return -1
	}
	return w - Width(right)
}

// Band3 is Band with an optional fragment between the two, dropped whenever it
// would leave the band looking crowded rather than laid out.
func Band3(left, mid, right string, w int) string {
	if mid != "" {
		if gap := w - Width(left) - Width(mid) - Width(right); gap >= 4 {
			l := gap / 2
			return left + strings.Repeat(" ", l) + mid + strings.Repeat(" ", gap-l) + right
		}
	}
	return Band(left, right, w)
}
