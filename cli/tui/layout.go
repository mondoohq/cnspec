// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tui

// Geometry is computed in exactly one place per view, and both the renderer
// (to draw) and the update loop (to hit-test mouse clicks) derive from it.
// Because they share the same numbers, a drawn element and the region that
// responds to it cannot drift apart.
//
// The invariant that makes this work: every row a view draws is exactly one
// terminal line. No style may carry a margin, because a margin adds a line that
// the arithmetic here cannot see -- which is how a list that "obviously" fits
// ends up scrolling the screen.

// Rect is a region of the terminal in cells.
type Rect struct{ X, Y, W, H int }

// Hit reports whether (x,y) falls inside the rect.
func (r Rect) Hit(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// ZoneKind identifies what a clickable region does.
type ZoneKind int

const (
	ZoneNone     ZoneKind = iota
	ZoneEntry             // a row in the left list; Idx is its selectable index
	ZoneField             // an input field in the detail pane; Idx is its field index
	ZoneMore              // the row that reveals the options
	ZoneRun               // the run button
	ZoneUpstream          // the reporting badge in the header
)

// Zone is a clickable region and the action it stands for.
type Zone struct {
	Rect Rect
	Kind ZoneKind
	Idx  int
}

// Chrome that frames the body: one header line at the top, one footer line at
// the bottom.
const (
	HeaderLines = 1
	FooterLines = 1
	// Inside the left panel, the first rows are the search field and a spacer.
	ListChromeLines = 2
	// Below this width the two panes are too narrow to be readable, so only the
	// focused one is shown.
	MinTwoPaneWidth = 76
)

// Layout is the pane geometry of a two-pane view plus the zones drawn into it.
type Layout struct {
	Width, Height int

	TwoPane bool
	LeftW   int
	RightW  int
	RightX  int

	BodyTop int // first body line
	BodyH   int // lines available to each pane

	ListTop int // absolute y of the first list row
	ListH   int // list rows that fit

	Zones []Zone
}

// Dims derives the pane geometry from the terminal size. It is the only place
// that turns a terminal size into pane sizes.
func Dims(width, height int) Layout {
	l := Layout{Width: width, Height: height}

	l.BodyTop = HeaderLines
	l.BodyH = height - HeaderLines - FooterLines
	if l.BodyH < 3 {
		l.BodyH = 3
	}

	l.TwoPane = width >= MinTwoPaneWidth
	if l.TwoPane {
		l.LeftW = width * 38 / 100
		if l.LeftW < 30 {
			l.LeftW = 30
		}
		if l.LeftW > 46 {
			l.LeftW = 46
		}
		l.RightW = width - l.LeftW - 1
		l.RightX = l.LeftW + 1
	} else {
		l.LeftW = width
		l.RightW = width
		l.RightX = 0
	}

	// Panels draw a border, so the list starts one line below the body top and
	// loses a line at the bottom as well.
	l.ListTop = l.BodyTop + 1 + ListChromeLines
	l.ListH = InnerHeight(l.BodyH) - ListChromeLines
	if l.ListH < 1 {
		l.ListH = 1
	}
	return l
}

// HitZone returns the topmost zone containing (x,y). Zones are appended in
// render order, so walk in reverse and let later ones win.
func (l Layout) HitZone(x, y int) Zone {
	for i := len(l.Zones) - 1; i >= 0; i-- {
		if l.Zones[i].Rect.Hit(x, y) {
			return l.Zones[i]
		}
	}
	return Zone{Kind: ZoneNone}
}
