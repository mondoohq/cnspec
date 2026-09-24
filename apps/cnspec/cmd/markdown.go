// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Turning a command's documentation into terminal help.
//
// A command's long description is written once, in Markdown, and read in two
// places: the docs site renders it as a page, and cobra prints it for
// `cnspec <command> --help`. Markdown is the form that keeps the docs page,
// since headings, fenced examples and links into the rest of the docs are what
// make it navigable -- and none of that reads well in a terminal, where the
// markers show up literally and a relative link points nowhere.
//
// So the Markdown is the source and the terminal gets a rendering of it. The
// other direction does not work: deriving headings and code blocks from plain
// prose means guessing, and guessing wrong silently costs the docs page its
// structure.

package cmd

import (
	"regexp"
	"strings"
)

// docsBaseURL is where the pages these descriptions link to are served. A
// relative link resolves against it on the docs site; in a terminal there is
// nothing to resolve against, so the reader needs the whole address.
const docsBaseURL = "https://mondoo.com/docs"

var (
	mdHeading = regexp.MustCompile(`^#{1,6} +`)
	mdLink    = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
)

// plainText renders Markdown as the help text of a command.
//
// It is deliberately small: these descriptions are prose, examples and links,
// and a full renderer would pull in a dependency to handle constructs none of
// them use.
func plainText(md string) string {
	var b strings.Builder
	inCode := false

	for _, line := range strings.Split(strings.TrimSpace(md), "\n") {
		if strings.HasPrefix(line, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			// Indented rather than fenced, which is how a terminal shows a
			// command you are meant to type.
			b.WriteString("  " + line + "\n")
			continue
		}
		b.WriteString(plainTextLine(line) + "\n")
	}
	// Cobra puts its own blank line between the description and what follows,
	// so a trailing one here shows up as a gap.
	return strings.TrimRight(b.String(), "\n")
}

func plainTextLine(line string) string {
	line = mdHeading.ReplaceAllString(line, "")
	return mdLink.ReplaceAllStringFunc(line, func(match string) string {
		parts := mdLink.FindStringSubmatch(match)
		text, url := parts[1], parts[2]
		if strings.HasPrefix(url, "/") {
			url = docsBaseURL + url
		}
		if text == "" {
			return url
		}
		return text + " (" + url + ")"
	})
}
