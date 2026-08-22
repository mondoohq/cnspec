// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnspec/cli/tui"
)

// Markdown for the detail pane.
//
// A check's description, its audit steps and every remediation body are markdown
// *source*. Printed raw they read as markup -- asterisks around the words that
// should be bold, hashes in front of the headings, three backticks around the
// command you are meant to run -- so this renders them with glamour, which is
// the terminal markdown renderer from the same family as the TUI stack this
// viewer is already built on (bubbletea, lipgloss, x/ansi).
//
// # The stylesheet
//
// The base is glamour's *dark* style, for one concrete reason: it is written in
// the same ANSI-256 index vocabulary as this viewer's chrome. Its body color is
// literally 252, which is theme.go's colText, and its horizontal rule is 240,
// which is colFaint. The alternatives ship as hex themes (dracula, tokyo-night,
// pink) that would put a second, unrelated accent next to the pane's violet and
// cyan.
//
// Five changes to it, each because this is a pane and not a page:
//
//   - The document margin goes to zero. detailBuf already indents a section
//     body under its label, and glamour's own margin would indent it a second
//     time and eat two columns of an already narrow pane. The code block keeps
//     its margin, though: that indent is the only thing that still says "this is
//     a command, not a sentence" on a terminal with no colors.
//   - The document block prefix and suffix -- a blank line above and below the
//     whole document -- go away. The pane spaces its own sections.
//   - The heading prefixes ("## ", "### ", ...) go away. Keeping them is what
//     the raw source already did; a heading here is bold and colored instead.
//     H1 loses its filled background band as well, because in this viewer a
//     colored band means "selected" and nothing else.
//   - An inline code span loses the two padding cells glamour puts around it.
//     They cost real width in a 30-column pane, and with colors off -- NO_COLOR,
//     or any terminal lipgloss reports as Ascii -- they read as a stray double
//     space around every path and setting name. The color and the background
//     already set the span apart where there are colors to set it apart with.
//   - The document foreground color is dropped so body prose inherits the
//     terminal's own. That is the light-terminal answer: 252 is near-white and
//     would be invisible on a white background, and every color left in the
//     sheet (39 headings, 203/236 inline code, 244 code blocks, chroma's own
//     mid-tones) is legible on either. It also means this pane never queries the
//     terminal for its background color, which is what glamour's own auto style
//     does -- an OSC round trip in the middle of a running TUI, that also
//     degrades to the ASCII stylesheet whenever stdout is not a terminal, which
//     would put the literal asterisks straight back.
//
// # Width
//
// glamour word-wraps prose to the width it is given, and mostly gets fenced code
// too, but not reliably: it gives up on a token it cannot fit, at a very narrow
// width it stops wrapping altogether, and a code row it does wrap comes out
// without the indent the row above it has. So its output overflows exactly where
// a remediation is most likely to be wide. Every row it produces is therefore
// measured in cells and folded if it is over, which is the one thing the frame
// cannot forgive: a row wider than the pane wraps in the terminal and pushes the
// whole layout down by one.

// markdownProfile is the color profile the renderer emits for. It matches the
// rest of the viewer, which draws through lipgloss's detected profile, so a
// 16-color terminal is not sent 24-bit escapes. It is a variable because
// lipgloss detects Ascii whenever stdout is not a terminal -- which is every
// test run -- and a test that wants to see the colors has to say so.
var markdownProfile = sync.OnceValue(lipgloss.ColorProfile)

// markdown is the pane's renderer. One is enough: the detail pane renders on one
// goroutine, and the mutex is there so that a test rendering in parallel cannot
// interleave two documents through glamour's block stack.
var markdown markdownRenderer

// markdownRenderer renders markdown source to terminal rows, holding on to the
// glamour renderers it built. Building one compiles a goldmark pipeline and the
// word wrap is baked into it, so they are kept and reused; the set is thrown
// away only when the color profile changes, which happens once, in a test.
//
// There is one per width because Blocks renders a document at two of them --
// prose at the pane's width, a code block a button's strip narrower -- and a
// single-slot cache would rebuild the pipeline at every fence. markdownMaxTRs
// bounds the set so that dragging a terminal's edge cannot accumulate one
// renderer per column crossed.
type markdownRenderer struct {
	mu      sync.Mutex
	trs     map[int]*glamour.TermRenderer
	profile termenv.Profile

	// renders counts the documents that went through glamour. The detail pane
	// caches its body per selection and width, and this is how a test proves
	// that the markdown is rendered inside that cache rather than on every
	// frame and every mouse wheel notch.
	renders int
}

// markdownMaxTRs is how many widths are kept. Two is the working set of one
// pane; the rest of the room is for the other pane and for a resize in flight.
const markdownMaxTRs = 6

// Lines renders markdown source into rows no wider than w cells. It returns nil
// when there is nothing to draw or when glamour fails, so a caller can fall back
// to printing the source: a section that has source must never render empty.
func (m *markdownRenderer) Lines(src string, w int) []string {
	src = strings.TrimSpace(tui.Clean(src))
	if src == "" || w < 1 {
		return nil
	}

	out, err := m.render(src, w)
	if err != nil {
		return nil
	}

	var res []string
	for _, ln := range tui.Lines(out) {
		// glamour pads every row out to the wrap width. Where the padding is
		// plain it is dropped; where it is inside a styled run -- the band
		// behind a code block -- it is not, because that band is the thing that
		// sets the code off from the prose.
		ln = strings.TrimRight(ln, " ")
		if tui.Width(ln) <= w {
			res = append(res, ln)
			continue
		}
		// Over-wide rows are code and unbreakable tokens. They are cut at the
		// edge and continued on the next row rather than truncated: a command
		// you are meant to paste is worth more whole than tidy.
		res = append(res, strings.Split(ansi.Hardwrap(ln, w, false), "\n")...)
	}
	return trimBlankRows(res)
}

// mdBlock is one piece of a rendered document: the rows it drew, and, when the
// piece is a fenced code block, what was between the fences.
//
// The pane needs the split because the COPY button sits on a block's first row:
// a document rendered whole comes back as rows with no idea which of them is
// code, and matching them back up afterwards means guessing from the escape
// sequences glamour happened to use.
type mdBlock struct {
	// Lines are the rendered rows, one terminal row each, already folded to the
	// width they were rendered at.
	Lines []string
	// Code is the block's source when this piece is a fenced code block, and nil
	// when it is prose.
	Code *mdCode
}

// Blocks renders markdown source, split at every top-level fenced code block.
//
// Prose is rendered at w cells and code at codeW, which is how the button gets a
// strip of its own at the right of a block without any of the command
// disappearing under it. Pass codeW == w for no strip.
//
// Each piece goes through glamour separately. That costs a render per piece and
// gains the one thing this needs: the row a block starts on is known exactly,
// rather than being looked for in a wall of styled output. Prose either side of
// a fence is a paragraph either way -- what does *not* survive being split is a
// fence nested in a list or a blockquote, which is why fencedCodes only reports
// the ones at the top level.
func (m *markdownRenderer) Blocks(src string, w, codeW int) []mdBlock {
	src = mdSource(src)
	if src == "" || w < 1 {
		return nil
	}
	if codeW < 1 || codeW > w {
		codeW = w
	}

	codes := fencedCodes(src)
	if len(codes) == 0 {
		if lines := m.Lines(src, w); len(lines) > 0 {
			return []mdBlock{{Lines: lines}}
		}
		return nil
	}

	res := make([]mdBlock, 0, 2*len(codes)+1)
	prose := func(s string) {
		if lines := m.Lines(s, w); len(lines) > 0 {
			res = append(res, mdBlock{Lines: lines})
		}
	}

	at := 0
	for i := range codes {
		code := codes[i]
		prose(src[at:code.Start])

		// The block is re-rendered from its own source text rather than from a
		// fence rebuilt around code.Text, so what glamour is handed is byte for
		// byte what it would have been handed had the document gone through in
		// one piece.
		lines := m.Lines(src[code.Start:code.Stop], codeW)
		if len(lines) == 0 {
			// glamour declined it. The source is still worth showing, and it is
			// still worth copying, so it is drawn as plain rows.
			lines = codeRows(code.Text, codeW)
		}
		res = append(res, mdBlock{Lines: lines, Code: &code})
		at = code.Stop
	}
	prose(src[at:])

	return res
}

// codeRows is the fallback for a block glamour would not render: the source, cut
// at the right edge rather than folded, in the style the pane gives its other
// code.
func codeRows(text string, w int) []string {
	lines := tui.Lines(text)
	res := make([]string, 0, len(lines))
	for _, ln := range lines {
		res = append(res, tui.Truncate(tui.StyleText.Render(ln), w))
	}
	return res
}

// Renders is how many documents this renderer has put through glamour.
func (m *markdownRenderer) Renders() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.renders
}

func (m *markdownRenderer) render(src string, w int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	profile := markdownProfile()
	if m.trs == nil || m.profile != profile {
		m.trs, m.profile = map[int]*glamour.TermRenderer{}, profile
	}

	tr := m.trs[w]
	if tr == nil {
		var err error
		tr, err = glamour.NewTermRenderer(
			markdownStyle(),
			glamour.WithWordWrap(w),
			glamour.WithColorProfile(profile),
		)
		if err != nil {
			return "", err
		}
		if len(m.trs) >= markdownMaxTRs {
			// A resize walks through every intermediate width. Dropping the lot
			// costs one rebuild of the two widths in use and keeps this from
			// growing with the number of columns the user dragged through.
			clear(m.trs)
		}
		m.trs[w] = tr
	}

	m.renders++
	return tr.Render(src)
}

// markdownStyle is glamour's dark stylesheet with the five changes the package
// comment explains.
//
// styles.DarkStyleConfig is a package-level value in glamour and a copy of it
// shares glamour's own pointers, so every field below is *replaced* and none is
// written through: this pane must not edit the stylesheet every other glamour
// caller in the process is using.
func markdownStyle() glamour.TermRendererOption {
	cfg := styles.DarkStyleConfig
	none := uint(0)

	cfg.Document.Margin = &none
	cfg.Document.BlockPrefix = ""
	cfg.Document.BlockSuffix = ""
	cfg.Document.Color = nil

	cfg.H1.Prefix = ""
	cfg.H1.Suffix = ""
	cfg.H1.BackgroundColor = nil
	cfg.H2.Prefix = ""
	cfg.H3.Prefix = ""
	cfg.H4.Prefix = ""
	cfg.H5.Prefix = ""
	cfg.H6.Prefix = ""

	cfg.Code.Prefix = ""
	cfg.Code.Suffix = ""

	return glamour.WithStyles(cfg)
}

// trimBlankRows drops the blank rows at both ends of a rendered block. glamour
// ends a document with one, and the pane puts its own blank line between
// sections; two of them reads as a gap nobody asked for.
func trimBlankRows(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
