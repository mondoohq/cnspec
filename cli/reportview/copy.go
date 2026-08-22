// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/tui"
)

// Copying a check's code to the clipboard.
//
// A remediation is not prose. It is a `yum remove`, an ansible play, a `defaults
// write` -- things whose only use is to be run somewhere else. Until this, the
// only way to get one out of the viewer was to select it with the mouse, and the
// pane had already folded it at the right edge and set it in two columns from
// the left, so what landed in the clipboard was not a command any more.
//
// So the thing that is copied is the *source*: the bytes between the fences,
// exactly as the policy author wrote them. Not what is on screen. No wrap, no
// indent, no ANSI, nothing glamour did to it. A pasted command runs.
//
// # What is copyable
//
// Everything the detail pane draws as a code block:
//
//   - the QUERY section, which is d.Mql -- a snippet in its own right, because
//     pasting it into `cnspec shell` is how you find out what the check saw;
//   - every fenced code block in the DESCRIPTION, the AUDIT and each
//     REMEDIATION body, all three of which are markdown source.
//
// The assessment is deliberately not on that list. It is the check's output,
// not something anybody runs.
//
// The blocks come from goldmark (fencedCodes), not from a scan for backticks.
// A fence may be written with tildes, may be indented, may carry no language and
// may never be closed at all, and CommonMark says something specific about each;
// goldmark is already in the module graph underneath glamour and already knows.
//
// # The clipboard
//
// github.com/atotto/clipboard, which shells out to pbcopy, xclip or wl-copy --
// already in the module graph, via bubbles/textinput. The reason to prefer it
// over an OSC 52 escape sequence is that it **returns an error**: it either put
// the text on the clipboard or it said why it could not, and the footer can
// therefore state a fact rather than a hope. An escape sequence is
// fire-and-forget -- many terminals ignore it by default, tmux and screen need
// explicit passthrough, and there is no reply -- so a viewer built on one would
// have to say "copied" without knowing.
//
// cnspec runs on the machine the user is sitting at, and that machine's
// clipboard is the one they will paste from, so the native clipboard is also the
// one that reaches the right place. Where there is no clipboard tool installed,
// WriteAll says so, and copyNotice shows that reason.

// Snippet is one copyable piece of a check: a code block, with enough about
// where it came from to name it in the footer afterwards.
//
// A check often has several -- the same fix as a CLI command, an ansible play
// and a bash script -- so "copied" on its own would not say which one.
type Snippet struct {
	// From is the section it came from, spelled the way the footer says it:
	// "description", "the query", "audit", "remediation [ansible]".
	From string
	// Lang is the fence's info word ("bash", "yaml"), empty when the fence
	// carried none. The query's is "mql".
	Lang string
	// Text is the source, exactly. This is what goes on the clipboard.
	Text string
	// Nth and Of place the snippet among the blocks of its own section, one
	// based. Of is 1 when the section had a single block, and Label leaves the
	// ordinal off in that case -- "remediation [cli]" needs no "#1 of 1".
	Nth, Of int
}

// Label names the snippet in a sentence: "remediation [cli] #2".
func (s Snippet) Label() string {
	if s.Of > 1 {
		return s.From + " #" + strconv.Itoa(s.Nth)
	}
	return s.From
}

// Describe is Label plus what it is and how much of it there is, which is the
// whole of what the footer needs to say: "remediation [cli] #2 (bash, 94 B)".
func (s Snippet) Describe() string {
	size := humanSize(int64(len(s.Text)))
	if s.Lang == "" {
		return s.Label() + " (" + size + ")"
	}
	return s.Label() + " (" + s.Lang + ", " + size + ")"
}

// checkSnippets is every copyable block of a check, in the order the detail pane
// draws them: description, query, audit, remediation. The order matters -- it is
// what lets the pane number the buttons it renders against this list rather than
// keeping a second one of its own.
func checkSnippets(c *reportmodel.Check) []Snippet {
	if c == nil {
		return nil
	}
	return detailSnippets(c.Detail())
}

// detailSnippets is checkSnippets over an already-composed detail. Detail()
// walks the raw results, so the pane computes it once per selection and passes
// it here rather than paying for it twice.
func detailSnippets(d reportmodel.CheckDetail) []Snippet {
	var res []Snippet
	res = append(res, markdownSnippets("description", d.Description)...)
	if mql := strings.TrimSpace(tui.Clean(d.Mql)); mql != "" {
		res = append(res, Snippet{From: "the query", Lang: "mql", Text: mql, Nth: 1, Of: 1})
	}
	res = append(res, markdownSnippets("audit", d.Audit)...)
	for _, item := range d.Remediation {
		res = append(res, markdownSnippets(remediationLabel(item.Id), item.Desc)...)
	}
	return res
}

// remediationLabel names a fix by its platform id, which is what tells three
// otherwise identical rows apart. "default" is reportmodel's stand-in for a fix
// with no platform, and is not a name worth showing.
func remediationLabel(id string) string {
	if id == "" || id == "default" {
		return "remediation"
	}
	return "remediation [" + id + "]"
}

// markdownSnippets is the fenced blocks of one markdown source, labelled.
func markdownSnippets(from, src string) []Snippet {
	codes := fencedCodes(mdSource(src))
	if len(codes) == 0 {
		return nil
	}
	res := make([]Snippet, 0, len(codes))
	for i, c := range codes {
		res = append(res, Snippet{
			From: from, Lang: c.Lang, Text: c.Text,
			Nth: i + 1, Of: len(codes),
		})
	}
	return res
}

// --- the clipboard -----------------------------------------------------------

// clipboardWrite is what puts text on the system clipboard.
//
// It is a variable so the tests can watch it. A test that called the real one
// would overwrite whatever the developer running it had copied, and on a build
// box with no pbcopy, xclip or wl-copy it would fail for a reason that has
// nothing to do with this viewer.
var clipboardWrite = clipboard.WriteAll

// copyDoneMsg is the outcome of a clipboard write on its way back to the frame.
//
// Unlike ExportDoneMsg this is not exported. A write to the clipboard is a
// subprocess that returns in milliseconds, not a multi-megabyte render, so there
// is no window in which the viewer is closed and the result still matters -- and
// exporting it would advertise a message an embedding program has to handle.
type copyDoneMsg struct {
	Snippet Snippet
	Err     error
}

// copyCmd is the write, as a command.
//
// Off the event loop for the same reason exportCmd is: clipboard.WriteAll forks
// pbcopy (or xclip, or wl-copy) and waits for it, and a fork-and-wait inside
// Update is a viewer that stops repainting mid-keystroke. Nothing here touches
// the terminal -- the escape-sequence route would have, and racing bubbletea's
// renderer from a command goroutine is exactly the thing this avoids.
func copyCmd(s Snippet) tea.Cmd {
	return func() tea.Msg {
		if s.Text == "" {
			return copyDoneMsg{Snippet: s, Err: errors.New("there is nothing in this block to copy")}
		}
		if err := clipboardWrite(s.Text); err != nil {
			return copyDoneMsg{Snippet: s, Err: errors.Wrap(err, "the system clipboard refused it")}
		}
		return copyDoneMsg{Snippet: s}
	}
}

// copyNotice is the one line the footer shows afterwards.
//
// Success says "copied", flatly, because clipboard.WriteAll returning nil means
// the text is on the clipboard -- there is nothing to hedge. Failure says why,
// in the words of whatever refused: a missing xclip and a wl-copy that exited
// non-zero are different problems and the user is the one who can fix either.
func copyNotice(msg copyDoneMsg) string {
	if msg.Err != nil {
		return "copy failed: " + tui.OneLine(msg.Err.Error())
	}
	return "copied " + msg.Snippet.Describe() + " to the clipboard"
}

// --- the button --------------------------------------------------------------

// The affordance is a web page's, on purpose: a small COPY at the top right of
// every code block, and you take the one you want. Clicking it copies that
// block.
//
// # The arming rule
//
// The keyboard needs a way to say *which* button, and the answer is the accent
// band: exactly one COPY on screen wears it, and y takes that block. The rule
// that decides which one, in full:
//
//  1. n and p move the arm to the next and previous code block of the check, and
//     scroll it into view. This is a real selection -- it is the reason the band
//     is worth drawing.
//  2. A plain scroll (the arrows, j/k, the page keys, g/G, the wheel) that
//     carries the armed block off the screen gives the arm back: it reverts to
//     the topmost block currently in view. Scrolling was the only way to aim
//     before n and p existed, and it is still an aim -- what it must not do is
//     leave the band somewhere the user cannot see it.
//  3. When no code block is on screen at all -- a long check opens on its
//     description, and the first fence may be forty rows down -- nothing is
//     armed. y says so and names n as the way to reach one.
//  4. A new selection resets the arm. The block you picked on the last check is
//     not on this one; the implicit rule takes over, and at the top of a fresh
//     page that is the check's first block.
//
// The invariant underneath all four: **y never copies a block that is not on
// screen**. It is enforced structurally rather than by agreement -- armedIn will
// not return a block outside the viewport, whatever put it there -- so a scroll,
// a resize or a reflow cannot leave y aimed at something invisible.
//
// The one exception is a terminal too narrow for two panes with the tree
// focused, where the detail pane is not drawn at all. There is no viewport to
// aim with and no band to have seen, and y still has to work from the tree, so
// it takes the check's first block. See detailPane.CopyTarget.
//
// A click is not arming and never has been: the button under the pointer is the
// answer to which block the user meant, so a click copies that one and leaves
// the arm where it was.

// copyLabel is the button. It is a constant width so that a column of them lines
// up down the pane and so the geometry can reserve exactly one strip for them.
const copyLabel = " COPY "

const (
	// copyLabelW is the rendered width of copyLabel.
	copyLabelW = 6
	// copyGutter is what a code block gives up to the button: the label plus one
	// space between it and the code. The block is *rendered* narrower by this
	// much rather than being overwritten by the button, so a button never hides
	// a character of the command it copies.
	copyGutter = copyLabelW + 1
	// copyMinCode is the narrowest strip of code worth keeping. Below
	// copyGutter+copyMinCode cells the pane draws no buttons at all: a terminal
	// where the button is wider than the command is one where the button is the
	// bug.
	copyMinCode = 8
)

// copyWidth is the width to render a code block at inside w cells, and whether
// there is room for a button beside it.
func copyWidth(w int) (int, bool) {
	if w < copyGutter+copyMinCode {
		return w, false
	}
	return w - copyGutter, true
}

// copyButtonRow puts the button at the right edge of a row w cells wide.
//
// Every button is drawn as a band, so every button reads as a button. The
// armed one -- the block y would take -- wears the accent band; the rest wear
// the inactive one. Drawing the unarmed ones as faint text instead was the
// first attempt and it read as *disabled*: a user seeing one lit button and one
// grey label concludes the second does not work, when in fact clicking it
// copies that block just as well.
//
// The armed one stays banded whether or not the detail pane has focus, because
// y is a frame binding -- it copies that block from the tree as well.
//
// The row is padded rather than overwritten: copyWidth already took the gutter
// out of the width the block was rendered at, so there is nothing under the
// button to lose.
func copyButtonRow(row string, w int, armed bool) string {
	if w < copyLabelW+1 {
		return row
	}
	style := tui.BandInactive
	if armed {
		style = tui.BandSelected
	}
	head := tui.PadRight(tui.Truncate(row, w-copyLabelW-1), w-copyLabelW-1)
	return head + " " + style.Render(copyLabel)
}

// copyZoneTag marks a Zone as a COPY button. The zone's Idx is the index of the
// snippet it copies.
const copyZoneTag = "copy"

// --- frame integration -------------------------------------------------------

// CopySource is the optional interface for a pane that can hand the frame
// something to put on the clipboard. It is optional in the same way SizedPane is:
// a pane that has no code in it simply does not implement it.
type CopySource interface {
	// CopyTarget is the snippet a copy key would take right now, and whether
	// there is one at all. It must not depend on having been rendered since the
	// last resize: below tui.MinTwoPaneWidth only the focused pane is drawn, and
	// the key still works from the other one.
	CopyTarget(st *State) (Snippet, bool)

	// ArmCopy moves the armed block delta places -- +1 for n, -1 for p -- and
	// scrolls it into view, which is not optional: an arm the user cannot see is
	// worse than no arm at all. It reports whether the pane had a block to arm;
	// at either end of the list it stays put and still reports true.
	ArmCopy(st *State, delta int) bool
}

// copySnippet is the frame's y binding: ask each pane for a target, and say so
// plainly when there is none. An empty picker is not offered -- this package has
// a rule against those.
func (m Model) copySnippet() (tea.Model, tea.Cmd) {
	for _, p := range m.panes() {
		cs, ok := p.(CopySource)
		if !ok {
			continue
		}
		if snip, ok := cs.CopyTarget(m.state); ok {
			return m, copyCmd(snip)
		}
	}
	m.state.Notice = "nothing to copy: " + noCopyReason(m.state)
	return m, nil
}

// armCopy is the frame's n and p binding: move the arm one block on.
//
// It is frame-level for the same reason y is, and the reason matters more than
// the line it saves. y copies while the tree has focus, which is where the
// cursor spends most of its time; a pane-local n and p would be dead in exactly
// that spot, so the key that aims would work in fewer places than the key that
// fires. It also never reaches the frame while the header's search field is
// open, because that field consumes every rune.
func (m Model) armCopy(delta int) (tea.Model, tea.Cmd) {
	for _, p := range m.panes() {
		cs, ok := p.(CopySource)
		if !ok {
			continue
		}
		if cs.ArmCopy(m.state, delta) {
			return m, nil
		}
	}
	// Silence would be indistinguishable from a key that does not exist, which is
	// the same reason y says something rather than nothing.
	m.state.Notice = "nothing to highlight: " + noCopyReason(m.state)
	return m, nil
}

// noCopyReason is why no pane offered a block, in the words the footer uses.
// It is only ever reached on the failing path, so recomputing the snippets to
// tell an empty check from a check scrolled away from costs nothing that the
// user is waiting on.
func noCopyReason(st *State) string {
	switch {
	case st.Sel.Check == nil:
		return "no check is selected"
	case len(checkSnippets(st.Sel.Check)) == 0:
		return "this check has no code blocks"
	default:
		return "no code block is in view (press n for the next one)"
	}
}

// copyDone puts the outcome in the footer. There is no box to keep open on a
// failure the way export has: the copy was one keystroke and the reason it
// failed fits on the line.
func (m Model) copyDone(msg copyDoneMsg) (tea.Model, tea.Cmd) {
	m.state.Notice = copyNotice(msg)
	return m, nil
}
