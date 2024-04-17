// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"fmt"
	"math"
	"strings"

	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v11/cli/theme/colors"
)

func sanitizeDatapoints(datapoints []float64) []float64 {
	for i := range datapoints {
		val := datapoints[i]
		// ensure the datapoint it a real value
		if math.IsNaN(val) {
			val = 0.0
			datapoints[i] = val
		}
	}
	return datapoints
}

// Option is used to set options in for StackBar. For example:
//
//	    progress := NewStackBar(
//		       WithWidth(65),
//	    )
type StackBarOption func(*StackBar)

func WithWidth(w int) StackBarOption {
	return func(m *StackBar) {
		m.Width = w
	}
}

func NewStackBar(opts ...StackBarOption) *StackBar {
	sb := &StackBar{
		Width: defaultWidth,
		// we use alternating characters for black-and-white rendering
		EntryChar: []rune{'█', '░', '▓', '░'},
	}

	for _, opt := range opts {
		opt(sb)
	}

	return sb
}

type StackBar struct {
	// total width of the stack bar, including title
	Width int

	// "EntryChar" visual representation for one entry in the stack bar
	EntryChar []rune
}

func (m StackBar) Render(colorMap []termenv.Color, datapoints []float64, textWidth int) string {
	b := strings.Builder{}

	datapoints = sanitizeDatapoints(datapoints)

	tw := m.Width - textWidth // total width

	// ensure we keep the correct length
	diff := make([]float64, len(datapoints))
	sizes := make([]int, len(datapoints))

	widthUsed := 0
	for i := range datapoints {
		percentage := (datapoints[i])
		// round down
		width := percentage * float64(tw)
		sizes[i] = int(width)
		widthUsed += sizes[i]
		diff[i] = width - float64(sizes[i])
	}

	// lets see if still have some ranges left
	if widthUsed < tw {
		left := tw - widthUsed

		for {
			maxIdx := 0
			maxDiff := float64(0)
			for j := range diff {
				if diff[j] > maxDiff {
					maxDiff = diff[j]
					maxIdx = j
				}
			}

			diff[maxIdx] = float64(0)
			sizes[maxIdx]++

			left--
			if left <= 0 {
				break
			}
		}
	}

	// render bar
	for i := range datapoints {
		idx := i % len(m.EntryChar)
		val := strings.Repeat(string(m.EntryChar[idx]), sizes[i])
		b.WriteString(termenv.String(val).Foreground(colorMap[i]).String())
	}

	return b.String()
}

type StackBarData struct {
	Title   string
	Color   []termenv.Color
	Labels  []string
	Entries []StackBarDataEntry
}

type StackBarDataEntry struct {
	Key    string
	Values []float64
}

type StackBarChart struct {
	bar               *StackBar
	displayDatapoints bool
	labelFunc         LabelFunc
}

// NewStackBarChart returns a model with default values.
func NewStackBarChart(labelF LabelFunc) *StackBarChart {
	entryChar := []rune{'█'}

	// fallback for no-color mode
	if colors.Profile == termenv.Ascii {
		entryChar = []rune{'█', '░', '▓', '░'}
	}

	m := &StackBarChart{
		bar: &StackBar{
			Width:     80,
			EntryChar: entryChar,
		},
		displayDatapoints: true,
		labelFunc:         labelF,
	}

	return m
}

type LabelFunc func(idx int, total float64, datapoints []float64, labels []string) string

func StackBarChartPercentageLabelFunc(idx int, total float64, datapoints []float64, labels []string) string {
	// add percentages for label
	datapointLabel := " ("
	for i := range datapoints {
		datapointLabel += fmt.Sprintf(" %s:%3.0f%%", labels[i], datapoints[i]/total*100)
	}
	datapointLabel += " )"
	return datapointLabel
}

func StackBarChartNoopLabelFunc(idx int, total float64, datapoints []float64, labels []string) string {
	return ""
}

// Render renders the chart.
func (m StackBarChart) Render(data StackBarData) string {
	b := strings.Builder{}

	maxLen := 0
	for i := range data.Entries {
		l := ansi.PrintableRuneWidth(data.Entries[i].Key)
		if l > maxLen {
			maxLen = l
		}
	}

	mayKeyWidth := m.bar.Width / 2
	if maxLen > mayKeyWidth {
		maxLen = mayKeyWidth
	}

	for i := range data.Entries {
		entry := data.Entries[i]

		datapoints := entry.Values
		total := float64(0)
		for i := range datapoints {
			total += datapoints[i]
		}

		l := ansi.PrintableRuneWidth(entry.Key)
		shouldWrap := l > maxLen

		if shouldWrap {
			// if the line is too long, we'll wrap it it and then make
			// the bar multiline as well
			wrappedKey := wordwrap.String(entry.Key, maxLen)
			bar := m.bar.Render(data.Color, datapoints, maxLen+1)
			lines := strings.Split(wrappedKey, "\n")
			first := true
			for _, line := range lines {
				ll := ansi.PrintableRuneWidth(line)
				indent := maxLen - ll
				if indent < 0 {
					indent = 0
				}
				b.WriteString(line + " ")
				b.WriteString(strings.Repeat(" ", indent))
				b.WriteString(bar)
				if first {
					b.WriteString(m.labelFunc(i, total, datapoints, data.Labels))
					first = false
				}
				b.WriteString("\n")
			}
			b.WriteString(strings.Repeat(" ", maxLen+1))
			b.WriteString(m.bar.Render(data.Color, datapoints, maxLen+1))
		} else {
			b.WriteString(entry.Key + " ")
			b.WriteString(strings.Repeat(" ", maxLen-l))
			b.WriteString(m.bar.Render(data.Color, datapoints, maxLen+1))
			b.WriteString(m.labelFunc(i, total, datapoints, data.Labels))
		}
		b.WriteString("\n")
	}

	return b.String()
}
