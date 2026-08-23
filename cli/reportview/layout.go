// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import "go.mondoo.com/cnspec/cli/tui"

// The viewer's geometry is computed in exactly one place: computeLayout. It
// renders nothing and mutates nothing, and both View (to draw) and Update (to
// hit-test a mouse click) derive from it, so a drawn element and the region that
// responds to it cannot drift apart.
//
// The pane arithmetic itself is tui.Dims, shared with the launcher: the same
// 38% split, the same minimum width below which two panes stop being readable
// and only the focused one is shown. What is added here is a header band that
// can be more than one line tall, because the viewer's header carries a search
// field and filter chips rather than just a title.
//
// The invariant underneath all of it: every row a pane returns is exactly one
// terminal line. No style may carry a margin -- a margin adds a line this
// arithmetic cannot see, which is how a pane that "obviously" fits ends up
// scrolling the screen.

const (
	// MaxHeaderLines caps what the header pane may ask for. A header that eats
	// the body is a worse bug than a header that gets truncated.
	MaxHeaderLines = 6
	// MinBodyLines is what the body keeps even on a very short terminal.
	MinBodyLines = 1
)

// Layout is the viewer's geometry: the header band, the two body panels and the
// footer, plus the content rect inside each panel.
//
// Panel rects include the border; Content rects are the region a pane renders
// into and are what Pane.Render receives.
type Layout struct {
	tui.Layout

	// HeaderH is the height of the header band, at least 1.
	HeaderH int
	// Header is the header band: full width, no border.
	Header tui.Rect
	// Footer is the single hint line at the bottom.
	Footer tui.Rect

	// TreePanel and DetailPanel are the bordered boxes, in absolute cells.
	TreePanel   tui.Rect
	DetailPanel tui.Rect
	// TreeContent and DetailContent are the regions inside those borders.
	TreeContent   tui.Rect
	DetailContent tui.Rect

	// ShowTree and ShowDetail say which body panes are drawn. Below
	// tui.MinTwoPaneWidth only the focused one is.
	ShowTree, ShowDetail bool
}

// Content is the region inside a panel's border: one line down from the top,
// tui.InnerX in from the left, and tui.InnerWidth by tui.InnerHeight in size.
func Content(panel tui.Rect) tui.Rect {
	return tui.Rect{
		X: panel.X + tui.InnerX,
		Y: panel.Y + 1,
		W: tui.InnerWidth(panel.W),
		H: tui.InnerHeight(panel.H),
	}
}

// computeLayout derives every rect from the terminal size, the focus and the
// height the header asks for.
func computeLayout(width, height int, headerH int, focus PaneID) Layout {
	l := Layout{Layout: tui.Dims(width, height)}

	// The header takes what it asks for, within reason: at least one line, never
	// more than MaxHeaderLines, and never so much that the body is squeezed out.
	if headerH < 1 {
		headerH = 1
	}
	if headerH > MaxHeaderLines {
		headerH = MaxHeaderLines
	}
	if room := height - tui.FooterLines - MinBodyLines; headerH > room {
		headerH = room
	}
	if headerH < 1 {
		headerH = 1
	}
	l.HeaderH = headerH

	l.BodyTop = headerH
	l.BodyH = height - headerH - tui.FooterLines
	if l.BodyH < MinBodyLines {
		l.BodyH = MinBodyLines
	}

	l.Header = tui.Rect{X: 0, Y: 0, W: width, H: headerH}
	l.Footer = tui.Rect{X: 0, Y: height - tui.FooterLines, W: width, H: tui.FooterLines}

	l.ShowTree = l.TwoPane || focus != PaneDetail
	l.ShowDetail = l.TwoPane || focus == PaneDetail

	l.TreePanel = tui.Rect{X: 0, Y: l.BodyTop, W: l.LeftW, H: l.BodyH}
	l.DetailPanel = tui.Rect{X: l.RightX, Y: l.BodyTop, W: l.RightW, H: l.BodyH}
	if !l.TwoPane {
		// One pane at a time gets the whole width.
		l.TreePanel.W = width
		l.DetailPanel = tui.Rect{X: 0, Y: l.BodyTop, W: width, H: l.BodyH}
	}

	l.TreeContent = Content(l.TreePanel)
	l.DetailContent = Content(l.DetailPanel)

	// tui.Dims computed these for the launcher's list, which starts below a
	// search field inside the panel. The viewer's list starts at the top of its
	// panel, so restate them rather than leave the launcher's numbers here for
	// someone to trust.
	l.ListTop = l.TreeContent.Y
	l.ListH = l.TreeContent.H

	return l
}

// ContentFor returns the rect a pane renders into, and whether it is drawn at
// all.
func (l Layout) ContentFor(id PaneID) (tui.Rect, bool) {
	switch id {
	case PaneHeader:
		return l.Header, l.Header.H > 0
	case PaneTree:
		return l.TreeContent, l.ShowTree
	case PaneDetail:
		return l.DetailContent, l.ShowDetail
	default:
		return tui.Rect{}, false
	}
}

// PanelFor returns the bordered box a pane is drawn in. The header has no panel.
func (l Layout) PanelFor(id PaneID) (tui.Rect, bool) {
	switch id {
	case PaneTree:
		return l.TreePanel, l.ShowTree
	case PaneDetail:
		return l.DetailPanel, l.ShowDetail
	default:
		return tui.Rect{}, false
	}
}

// PaneAt returns the pane whose region contains (x, y), which is what a scroll
// wheel needs: the wheel scrolls what the pointer is over, not what has focus.
func (l Layout) PaneAt(x, y int) PaneID {
	if l.Header.Hit(x, y) {
		return PaneHeader
	}
	if l.ShowDetail && l.DetailPanel.Hit(x, y) {
		return PaneDetail
	}
	if l.ShowTree && l.TreePanel.Hit(x, y) {
		return PaneTree
	}
	return PaneNone
}
