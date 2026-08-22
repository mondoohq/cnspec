// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.mondoo.com/cnspec/cli/tui"
)

// Finding the fenced code blocks in a markdown source, and splitting the source
// at them so the pane can draw a button on each.
//
// # Why a parser and not a scan for backticks
//
// Because CommonMark says something specific about every one of these and a
// scanner would have to reimplement all of it:
//
//	~~~sh          a fence may be tildes, and a tilde fence may contain backticks
//	```            a fence may carry no info string at all
//	  ```bash      a fence may be indented up to three spaces, and the same
//	               indent is then stripped from every line inside it
//	````           a longer fence closes only on one at least as long, so a
//	 ```           three-backtick line inside it is content, not the end
//	```bash        an unclosed fence runs to the end of the document rather
//	echo hi        than swallowing it as if the fence were never opened
//
// goldmark implements all of that, and it is already in the module graph
// underneath glamour -- the same parser glamour itself renders this markdown
// with, so what this finds and what the pane draws cannot disagree about where a
// block is.
//
// # Only the blocks at the top level of the document
//
// A fence inside a list item or a blockquote is skipped. The reason is the
// splitting, not the parsing: the pane renders the document in pieces, one per
// code block, and cutting a list in half to pull a block out of it would restart
// its numbering and drop its markers. Every fenced block in cnspec's own content
// is written at the top level, which is what the markdown a policy author writes
// looks like.

// mdParser is the parser every source goes through. Building one assembles the
// whole block and inline pipeline, and a Parser is reusable -- it takes no state
// from a parse, keeping what it needs in a per-call context -- so one is built
// and kept rather than one per markdown section of every check.
//
// Stock CommonMark, with no extension registered: a fenced code block is core
// syntax, and nothing an extension adds changes where a fence begins or ends.
var mdParser = sync.OnceValue(func() parser.Parser {
	return goldmark.New().Parser()
})

// mdCode is one fenced code block of a markdown source.
type mdCode struct {
	// Lang is the fence's info word: "bash" for ```bash, empty for a bare ```.
	Lang string
	// Text is the literal source between the fences, with the container indent
	// already stripped and the blank lines at either end removed. This is what
	// goes on the clipboard.
	Text string
	// Start and Stop bracket the whole block in the source it was found in,
	// fences included, so a caller can split the document at it. Source[Start:Stop]
	// is the block exactly as it was written.
	Start, Stop int
}

// mdSource normalizes a markdown source the way the renderer does, so an offset
// found here indexes the same string the renderer parses.
func mdSource(s string) string {
	return strings.TrimSpace(tui.Clean(s))
}

// fencedCodes is the top-level fenced code blocks of a normalized markdown
// source, in document order. Pass it mdSource(s).
//
// A block with no content between its fences is left out: there is nothing to
// copy, and a button on it would be a button that does nothing.
func fencedCodes(src string) []mdCode {
	if !strings.ContainsAny(src, "`~") {
		// The overwhelmingly common case for a description: no fence, no parse.
		return nil
	}

	b := []byte(src)
	doc := mdParser().Parse(text.NewReader(b))

	var res []mdCode
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		fence, ok := n.(*ast.FencedCodeBlock)
		if !ok {
			continue
		}
		code, ok := fenceSpan(src, b, fence)
		if !ok {
			continue
		}
		// Defensive: the spans must march forward and stay inside the source.
		// They do, for siblings of a document -- but every consumer of this
		// slices src with them, and a slice with a bad bound panics in a View.
		if code.Start < 0 || code.Stop > len(src) || code.Start >= code.Stop {
			continue
		}
		if len(res) > 0 && code.Start < res[len(res)-1].Stop {
			continue
		}
		res = append(res, code)
	}
	return res
}

// fenceSpan works out where a block starts and ends and what is inside it.
//
// goldmark hands back the *content* lines, not the fences: the block runs from
// the line before the first content line (the opening fence) to the end of the
// line after the last one (the closing fence), when there is a closing fence at
// all.
func fenceSpan(src string, b []byte, fence *ast.FencedCodeBlock) (mdCode, bool) {
	lines := fence.Lines()
	if lines.Len() == 0 {
		return mdCode{}, false
	}

	var body strings.Builder
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		body.Write(seg.Value(b))
	}
	// Blank lines at either end of a block are the author's spacing, not part of
	// the command. A leading newline in front of a pasted `yum remove` is noise.
	code := mdCode{
		Lang: string(fence.Language(b)),
		Text: strings.Trim(body.String(), "\n"),
	}
	if code.Text == "" {
		return mdCode{}, false
	}

	// The opening fence is the line before the content.
	contentAt := lineStart(src, lines.At(0).Start)
	if contentAt == 0 {
		return mdCode{}, false
	}
	code.Start = lineStart(src, contentAt-1)

	// The closing fence is the line after it, if the block was closed at all.
	code.Stop = lines.At(lines.Len() - 1).Stop
	if code.Stop < len(src) {
		if end := lineStop(src, code.Stop); isFenceLine(src[code.Stop:end]) {
			code.Stop = end
		}
	}
	if code.Stop > len(src) {
		code.Stop = len(src)
	}
	return code, true
}

// lineStart is the offset of the first byte of the line containing off.
func lineStart(s string, off int) int {
	if off > len(s) {
		off = len(s)
	}
	if off < 0 {
		off = 0
	}
	if i := strings.LastIndexByte(s[:off], '\n'); i >= 0 {
		return i + 1
	}
	return 0
}

// lineStop is the offset just past the line beginning at off, newline included.
func lineStop(s string, off int) int {
	if i := strings.IndexByte(s[off:], '\n'); i >= 0 {
		return off + i + 1
	}
	return len(s)
}

// isFenceLine reports whether a line is nothing but a closing fence: at least
// three of one fence character, indented no more than the three spaces
// CommonMark allows.
func isFenceLine(line string) bool {
	s := strings.TrimRight(strings.TrimRight(line, "\n"), " \t")
	indent := len(s) - len(strings.TrimLeft(s, " "))
	if indent > 3 {
		return false
	}
	s = s[indent:]
	if len(s) < 3 {
		return false
	}
	c := s[0]
	if c != '`' && c != '~' {
		return false
	}
	return strings.Trim(s, string(c)) == ""
}
