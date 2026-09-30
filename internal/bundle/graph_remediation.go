// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"strings"
)

// CodeBlock is one fenced code block of a remediation's markdown.
type CodeBlock struct {
	// Lang is the first word of the fence's info string, e.g. "bash".
	// Empty when the fence carries no language.
	Lang string `json:"lang,omitempty"`
	Code string `json:"code"`
}

// RemediationEntry is one docs.remediation item of a check, with the code
// blocks of its markdown parsed out.
type RemediationEntry struct {
	ID   string      `json:"id,omitempty"`
	Desc string      `json:"desc"`
	Code []CodeBlock `json:"code,omitempty"`
}

// NodeRemediations holds the remediations of one check or query.
type NodeRemediations struct {
	UID          string             `json:"uid"`
	QualName     string             `json:"qual_name"`
	Kind         NodeKind           `json:"kind"`
	Title        string             `json:"title,omitempty"`
	File         string             `json:"file"`
	Line         int                `json:"line"`
	Remediations []RemediationEntry `json:"remediations"`
}

// RemediationOpts narrows what PolicyGraph.Remediations returns. Both
// filters match case-insensitively; an empty filter matches everything.
type RemediationOpts struct {
	// IDs keeps only remediation entries with one of these ids, e.g. "bash".
	IDs []string
	// Langs keeps only code blocks in one of these languages and drops
	// entries left without a code block.
	Langs []string
}

// Remediations returns the remediations for a node. A check or query yields
// its own remediations; any other node (policy, group, framework, control)
// yields those of every check and query reachable from it. Nodes without a
// matching remediation are left out.
func (g *PolicyGraph) Remediations(nodeID string, opts RemediationOpts) []NodeRemediations {
	g.ensureBuilt()
	ids := []string{nodeID}
	if n := g.Node(nodeID); n == nil || (n.Kind != KindCheck && n.Kind != KindQuery) {
		ids = g.Reachable(nodeID)
	}

	var res []NodeRemediations
	for _, id := range ids {
		n := g.Node(id)
		if n == nil || (n.Kind != KindCheck && n.Kind != KindQuery) {
			continue
		}
		entries := filterRemediations(n.Remediations, opts)
		if len(entries) == 0 {
			continue
		}
		res = append(res, NodeRemediations{
			UID:          n.Name,
			QualName:     n.QualName,
			Kind:         n.Kind,
			Title:        n.Title,
			File:         n.File,
			Line:         n.Line,
			Remediations: entries,
		})
	}
	return res
}

func filterRemediations(items []GraphRemediation, opts RemediationOpts) []RemediationEntry {
	var res []RemediationEntry
	for _, item := range items {
		if len(opts.IDs) > 0 && !containsFold(opts.IDs, item.ID) {
			continue
		}
		var code []CodeBlock
		for _, block := range ParseCodeBlocks(item.Desc) {
			if len(opts.Langs) > 0 && !containsFold(opts.Langs, block.Lang) {
				continue
			}
			code = append(code, block)
		}
		if len(opts.Langs) > 0 && len(code) == 0 {
			continue
		}
		res = append(res, RemediationEntry{ID: item.ID, Desc: item.Desc, Code: code})
	}
	return res
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

// ParseCodeBlocks returns the fenced code blocks of a markdown text in order.
// It follows the CommonMark fence rules that remediation texts use: a fence
// is three or more backticks or tildes, the closing fence uses the same
// character and is at least as long, and a block left open runs to the end
// of the text. Fences may be indented, as they are inside list items; the
// fence's indentation is removed from each line of the block. Indented code
// blocks without a fence are not recognized.
func ParseCodeBlocks(markdown string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(markdown, "\n")

	var (
		inBlock bool
		fence   string
		indent  int
		lang    string
		body    []string
	)
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if !inBlock {
			f := fenceMarker(trimmed)
			if f == "" {
				continue
			}
			info := strings.TrimSpace(trimmed[len(f):])
			// A backtick fence's info string may not contain backticks,
			// otherwise the line is inline code.
			if f[0] == '`' && strings.Contains(info, "`") {
				continue
			}
			inBlock, fence, indent, body = true, f, len(line)-len(trimmed), nil
			lang = ""
			if fields := strings.Fields(info); len(fields) > 0 {
				lang = fields[0]
			}
			continue
		}

		if f := fenceMarker(trimmed); f != "" && f[0] == fence[0] && len(f) >= len(fence) &&
			strings.TrimSpace(trimmed[len(f):]) == "" {
			blocks = append(blocks, newCodeBlock(lang, body))
			inBlock = false
			continue
		}
		body = append(body, stripIndent(line, indent))
	}
	if inBlock {
		blocks = append(blocks, newCodeBlock(lang, body))
	}
	return blocks
}

// fenceMarker returns the run of three or more backticks or tildes a line
// starts with, or "" if it does not start with one.
func fenceMarker(line string) string {
	if line == "" || (line[0] != '`' && line[0] != '~') {
		return ""
	}
	n := 0
	for n < len(line) && line[n] == line[0] {
		n++
	}
	if n < 3 {
		return ""
	}
	return line[:n]
}

// stripIndent removes up to n leading spaces or tabs from a line.
func stripIndent(line string, n int) string {
	i := 0
	for i < n && i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return line[i:]
}

func newCodeBlock(lang string, body []string) CodeBlock {
	code := strings.Join(body, "\n")
	if code != "" {
		code += "\n"
	}
	return CodeBlock{Lang: lang, Code: code}
}
