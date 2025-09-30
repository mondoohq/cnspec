// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"bytes"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// Entry represents a single linting issue found.
type Entry struct {
	RuleID   string
	Level    string
	Message  string
	Location []Location
}

// Location specifies the file, line, and column of a linting issue.
type Location struct {
	File   string
	Line   int
	Column int
}

type SortResults []*Entry

func (s SortResults) Len() int {
	return len(s)
}

func (s SortResults) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SortResults) Less(i, j int) bool {
	if s[i].Level != s[j].Level {
		return s[i].Level < s[j].Level
	}
	return s[i].RuleID < s[j].RuleID
}

// Results holds all linting entries for a bundle.
type Results struct {
	BundleLocations []string
	Entries         []*Entry
}

// HasError checks if there are any error-level entries.
func (r *Results) HasError() bool {
	for i := range r.Entries {
		if r.Entries[i].Level == LevelError {
			return true
		}
	}
	return false
}

// HasWarning checks if there are any warning-level entries.
func (r *Results) HasWarning() bool {
	for i := range r.Entries {
		if r.Entries[i].Level == LevelWarning {
			return true
		}
	}
	return false
}

func (r *Results) ToCli() []byte {
	// lets not render the result table if no findings are present
	if r == nil || len(r.Entries) == 0 {
		return []byte{}
	}

	sort.Sort(SortResults(r.Entries))

	// render platform advisories
	var buf bytes.Buffer
	table := tablewriter.NewTable(&buf,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.Border{Left: tw.Off, Top: tw.Off, Right: tw.Off, Bottom: tw.Off},
			// Symbols: tw.NewSymbolCustom("cnspec").WithColumn(""),
			Settings: tw.Settings{
				Separators: tw.Separators{
					ShowHeader:     tw.Off,
					BetweenRows:    tw.Off,
					BetweenColumns: tw.Off,
				},
				Lines: tw.Lines{
					ShowHeaderLine: tw.Off,
					ShowFooterLine: tw.Off,
				},
			},
		})),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft}, // Left-align rows
			},
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft}, // Left-align rows
				Formatting: tw.CellFormatting{
					AutoWrap: tw.WrapNone,
				},
			},
			Footer: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignRight},
			},
		}),
	)

	header := []string{"Rule ID", "Level", "File", "Line", "Message"}
	table.Header(header)

	for i := range r.Entries {
		entry := r.Entries[i]
		// Ensure there's at least one location before accessing
		fileName := ""
		lineNumber := ""
		if len(entry.Location) > 0 {
			fileName = filepath.Base(entry.Location[0].File)
			lineNumber = strconv.Itoa(entry.Location[0].Line)
		}

		table.Append([]string{
			entry.RuleID,
			entry.Level,
			fileName,
			lineNumber,
			entry.Message,
		})
	}
	table.Render()
	return buf.Bytes()
}
