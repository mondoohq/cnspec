// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package reportview is the terminal browser for a scan report: a two-pane view
// over the reportmodel tree, driven by `cnspec report view` and, later, by the
// interactive launcher once a scan it started has finished.
//
// # The seam
//
// This package is a frame, not a set of panes. The frame owns the terminal: the
// geometry, the focus, the key and mouse routing, the header band and the footer
// hint line. Everything inside a pane -- what a row looks like, how a detail is
// laid out, what a filter does -- belongs to a Pane, and each Pane lives in its
// own file so that panes can be written independently of one another.
//
// A pane plugs in through three things and nothing else:
//
//   - It reads and writes *State, the small amount of state that genuinely
//     crosses pane boundaries: the report, the selection and the filter. Per-pane
//     state (a scroll offset, a cursor, a set of collapsed nodes) belongs to the
//     pane's own struct, never here, so that adding a pane never edits a struct
//     another pane also has to edit.
//
//   - It answers Render(st, rect) with the lines to draw *and* the clickable
//     zones in one call. Both come out of the same pass over the same rect, which
//     is what keeps a click on row seven landing on the thing row seven drew. The
//     frame calls Render exactly once per frame and uses the result for drawing
//     and for hit-testing alike.
//
//   - It answers Update(st, msg) with whether it handled the message. Keys go to
//     the focused pane first; anything it does not claim falls through to the
//     frame's own bindings. See Model.Update for the full order.
//
// A pane installs itself by calling RegisterTree, RegisterDetail or
// RegisterHeader from an init function in its own file. There is no table of
// panes to append to.
//
// # Invariants
//
// One rendered line is exactly one terminal line. No style may carry a margin: a
// margin adds a line the geometry cannot see, and the panels are sized in exact
// lines. Render may return fewer lines than the rect is tall (the frame pads) or
// more (the frame drops the excess), but a single element of the slice is always
// one row.
//
// Color for a status or a severity comes from SeverityStyle / StatusStyle in
// theme.go, which resolve through components.DefaultScoreRatingColors. Do not
// hand-pick a color for a finding: the CLI already has one and the viewer must
// agree with it.
//
// Text that came from the reporter must be passed through Wrap or Clean before
// it reaches a rendered line. NewLineCharacter is "\r\n" on Windows and a stray
// carriage return corrupts every width calculation downstream.
package reportview

import (
	tea "github.com/charmbracelet/bubbletea"
	"go.mondoo.com/cnspec/cli/tui"
)

// PaneID names a pane. There is at most one pane per id, and the id is also its
// slot in the layout: the header is the band across the top, the tree is the
// left panel and the detail is the right one.
type PaneID int

const (
	// PaneNone is the zero value: no pane.
	PaneNone PaneID = iota
	// PaneHeader is the band above the body. It holds the summary, the search
	// field and the filter chips.
	PaneHeader
	// PaneTree is the left panel: asset -> policy -> check.
	PaneTree
	// PaneDetail is the right panel: everything about the selected check.
	PaneDetail
)

// String names the pane, for hints and tests.
func (id PaneID) String() string {
	switch id {
	case PaneHeader:
		return "header"
	case PaneTree:
		return "tree"
	case PaneDetail:
		return "detail"
	default:
		return "none"
	}
}

// Zone is a clickable region and the pane that owns it. Idx and Tag are the
// pane's own business: the frame only finds the zone, focuses its pane and hands
// the zone back in a ClickMsg.
//
// It deliberately does not reuse tui.ZoneKind, whose values name the launcher's
// widgets. tui.Rect, the part that is geometry rather than vocabulary, is shared.
type Zone struct {
	// Rect is the region in absolute terminal cells.
	Rect tui.Rect
	// Pane owns the zone. The frame fills this in from the pane that returned
	// it, so a pane may leave it zero.
	Pane PaneID
	// Idx identifies what was clicked within the pane, e.g. a row index.
	Idx int
	// Tag distinguishes kinds of target within one pane, e.g. "row" from
	// "twisty". It is free-form and only the owning pane reads it.
	Tag string
}

// ClickMsg is delivered to a pane when a click lands in one of its zones. The
// frame has already moved focus to that pane by the time it arrives.
type ClickMsg struct {
	Zone  Zone
	Mouse tea.MouseMsg
}

// Hint is one key binding shown in the footer.
type Hint struct {
	Key   string
	Label string
}

// Render is everything a pane produces for one frame.
//
// Title and Status are inlaid into the top edge of the pane's border (the header
// band has no border and ignores both). Lines are the content rows, one per
// terminal line. Zones are the regions of those rows that respond to a click, in
// absolute coordinates.
//
// Lines and Zones come out of the same call because they have to agree: a zone
// computed in a second pass is a zone that can drift off the row it names.
type Render struct {
	Title  string
	Status string
	Lines  []string
	Zones  []Zone
}

// Pane is one region of the viewer. Implementations live in their own file and
// keep their own state; the only state they share is *State.
//
// Every method may be called several times per frame and must not block: the
// frame calls Render on each visible pane once per View and once per mouse
// event.
type Pane interface {
	// ID is the pane's slot. It must be constant.
	ID() PaneID

	// Focusable reports whether the pane takes keyboard focus. A pane that only
	// displays -- a static summary band, say -- returns false and is skipped by
	// the focus cycle; it can still own clickable zones.
	Focusable() bool

	// Render draws the pane into rect, which is the pane's *content* area in
	// absolute terminal cells: for a bordered panel that is the region inside
	// the border, so rect.W is tui.InnerWidth of the panel and rect.H is its
	// tui.InnerHeight.
	//
	// Returning fewer than rect.H lines is fine (the frame pads with blanks) and
	// so is returning more (the frame drops them), but a pane that scrolls
	// should return exactly the rows it wants visible. Every element must be one
	// terminal row: no line may contain "\n".
	//
	// Zones must lie inside rect.
	Render(st *State, rect tui.Rect) Render

	// Update handles a message routed to this pane and reports whether it
	// consumed it. An unconsumed key falls through to the frame's own bindings,
	// which is how "esc" can clear a search in the pane that has one and quit
	// everywhere else.
	//
	// The pane mutates itself and st in place. The returned command is passed
	// back to bubbletea.
	Update(st *State, msg tea.Msg) (tea.Cmd, bool)

	// Hints are the key bindings the footer shows while this pane has focus.
	Hints(st *State) []Hint

	// Claims are keys the pane handles even when it does not have focus, e.g.
	// "/" to open a search. The frame moves focus to the pane and then delivers
	// the key. Return nil for none.
	Claims() []string
}

// SizedPane is the optional interface for a pane that decides its own height.
// Only the header slot honors it: the tree and detail panes are sized by the
// frame, which splits whatever the header leaves. The frame clamps the result so
// the body always keeps at least one line.
type SizedPane interface {
	Pane
	HeightFor(st *State, width int) int
}

// PaneFactory builds a pane for one viewer. It is called once per Model, so a
// pane may hold mutable state.
type PaneFactory func(st *State) Pane

// registry holds at most one factory per slot. A pane registers itself from its
// own file, so no two panes ever edit the same declaration.
var registry = map[PaneID]PaneFactory{}

// RegisterHeader installs the header pane: the summary band, search and filters.
func RegisterHeader(f PaneFactory) { register(PaneHeader, f) }

// RegisterTree installs the left pane: the asset -> policy -> check tree.
func RegisterTree(f PaneFactory) { register(PaneTree, f) }

// RegisterDetail installs the right pane: the detail of the selected check.
func RegisterDetail(f PaneFactory) { register(PaneDetail, f) }

// register panics on a second registration for the same slot, at init time,
// because two panes fighting over one slot is a wiring mistake that must not
// resolve itself by init order.
func register(id PaneID, f PaneFactory) {
	if f == nil {
		panic("reportview: nil pane factory for the " + id.String() + " slot")
	}
	if _, exists := registry[id]; exists {
		panic("reportview: a pane is already registered for the " + id.String() + " slot")
	}
	registry[id] = f
}

// buildPane returns the registered pane for a slot, or the placeholder that
// stands in until one is registered.
func buildPane(id PaneID, st *State) Pane {
	if f, ok := registry[id]; ok {
		return f(st)
	}
	return newPlaceholder(id, st)
}
