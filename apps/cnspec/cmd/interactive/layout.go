// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import "go.mondoo.com/cnspec/cli/tui"

// The launcher's geometry is computed in exactly one place: computeLayout. It
// renders nothing and mutates nothing, and both View (to draw) and Update (to
// hit-test mouse clicks) derive from it. Because they share the same numbers, a
// drawn element and the region that responds to it cannot drift apart.
//
// The invariant that makes this work: every row the launcher draws is exactly
// one terminal line. No style in view.go carries a margin, because a margin adds
// a line that the arithmetic here cannot see -- which is how a list that
// "obviously" fits ends up scrolling the screen.
//
// The pane arithmetic itself lives in package tui, shared with the report
// viewer. Only the parts that need the Model are here.

// layout is the launcher's handle on the shared geometry. It exists so that
// offsetRow and detailOffsetOf can stay methods: both need the Model, and Go
// does not allow a method on a type declared in another package.
type layout struct{ tui.Layout }

// computeLayout derives every rect and clickable zone from the model.
func computeLayout(m Model) layout {
	l := layout{tui.Dims(m.width, m.height)}

	// The reporting badge is clickable. Its position comes from the same
	// function the renderer uses, and headerRightX answers -1 when the header
	// is too narrow to draw the badge -- so a click can never land on a badge
	// that is not on screen.
	if x := m.headerRightX(l.Width); x >= 0 {
		if bw := m.upstream.badgeWidth(); bw > 0 {
			l.Zones = append(l.Zones, tui.Zone{
				Rect: tui.Rect{X: x, Y: 0, W: bw, H: 1},
				Kind: tui.ZoneUpstream,
			})
		}
	}

	showList := l.TwoPane || m.focus == focusList
	showDetail := l.TwoPane || m.focus != focusList

	if showList {
		for i := l.offsetRow(m.list); i < len(m.list.rows) && i < l.offsetRow(m.list)+l.ListH; i++ {
			r := m.list.rows[i]
			if r.kind != rowEntry {
				continue
			}
			y := l.ListTop + (i - l.offsetRow(m.list))
			l.Zones = append(l.Zones, tui.Zone{
				Rect: tui.Rect{X: tui.InnerX, Y: y, W: tui.InnerWidth(l.LeftW), H: 1},
				Kind: tui.ZoneEntry,
				Idx:  r.sel,
			})
		}
	}

	if showDetail {
		// The detail pane's geometry is not computed in parallel with the
		// renderer -- both walk the same plan, one line per item, so a zone
		// cannot land on a row the renderer put something else on.
		plan, _ := splitPlan(m.detailPlan())
		cx := l.RightX + tui.InnerX
		cw := tui.InnerWidth(l.RightW)
		off := l.detailOffsetOf(m, len(plan))
		visible := tui.InnerHeight(l.BodyH) - 1 // the pinned command bar

		for i, item := range plan {
			row := i - off
			if row < 0 || row >= visible {
				continue
			}
			y := l.BodyTop + 1 + row
			switch item.kind {
			case diField:
				l.Zones = append(l.Zones, tui.Zone{Rect: tui.Rect{X: cx, Y: y, W: cw, H: 1}, Kind: tui.ZoneField, Idx: item.idx})
			case diMore:
				l.Zones = append(l.Zones, tui.Zone{Rect: tui.Rect{X: cx, Y: y, W: cw, H: 1}, Kind: tui.ZoneMore})
			case diButton:
				l.Zones = append(l.Zones, tui.Zone{Rect: tui.Rect{X: cx, Y: y, W: cw, H: 1}, Kind: tui.ZoneRun})
			}
		}
	}

	return l
}

// offsetRow clamps the model's scroll offset into the current layout, so a
// resize that grows the terminal cannot leave the list scrolled past its end.
func (l layout) offsetRow(s listState) int {
	return s.offset.Clamp(len(s.rows), l.ListH)
}

// detailOffsetOf clamps the detail pane's scroll offset to the plan it is
// showing, so a resize cannot leave it scrolled past the end.
func (l layout) detailOffsetOf(m Model, planLen int) int {
	return m.detail.offset.Clamp(planLen, tui.InnerHeight(l.BodyH)-1)
}
