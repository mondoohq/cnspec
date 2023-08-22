// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"fmt"
	"math"
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/theme/colors"
)

func maxWidth(labels []string) int {
	max := 0
	for i := range labels {
		l := len(labels[i])
		if l > max {
			max = l
		}
	}
	return max
}

type BarChartOption func(chart *BarChart)

func WithBarChartBorder() BarChartOption {
	return func(m *BarChart) {
		m.border = true
	}
}

func WithBarChartTitle(title string) BarChartOption {
	return func(m *BarChart) {
		m.title = title
	}
}

type BarChartLabelFunc func(i int, datapoints []float64) string

func WithBarChartLabelFunc(labelFunc BarChartLabelFunc) BarChartOption {
	return func(m *BarChart) {
		m.labelFunc = labelFunc
	}
}

func noLabelFunc(i int, datapoints []float64) string { return "" }

func BarChartPercentageLabelFunc(i int, datapoints []float64) string {
	// write percentage, handle case where we get a NaN
	p := datapoints[i] * 100
	if math.IsNaN(p) {
		p = 0
	}
	return fmt.Sprintf("%.1f%%", p)
}

func NewBarChart(opts ...BarChartOption) *BarChart {
	theme := AnsiPaperCars
	if colors.Profile == termenv.Ascii {
		theme = AsciiPaperCars
	}

	bc := &BarChart{
		width:       defaultWidth,
		EntryChar:   '█',
		border:      false,
		borderColor: colors.Profile.Color("#a9a9a9"),
		theme:       theme,
		labelFunc:   noLabelFunc,
	}

	for _, opt := range opts {
		opt(bc)
	}

	return bc
}

type BarChart struct {
	// title of the chart, if set is printed on the chart
	title string

	// function to calculate the label values
	labelFunc BarChartLabelFunc

	// total width of the bar chart including labels
	width int

	// visual representation for one entry in the stack bar
	EntryChar rune

	// chart with a border
	border bool

	// color of the border
	borderColor termenv.Color

	// character theme for border
	theme PaperCharsTheme
}

func (m BarChart) Render(datapoints []float64, colorMap []termenv.Color, labels []string) string {
	b := strings.Builder{}

	boxWidth := m.width

	// if its rendered with a box, we need to substract two char from each end
	if m.border {
		boxWidth = boxWidth - 4
	}

	// determine max size labels
	maxLabelWidth := maxWidth(labels)

	datapoints = sanitizeDatapoints(datapoints)

	// TODO: sorting changes the index for color and label
	//if m.SortByMax {
	//	sort.Sort(sort.Reverse(sort.Float64Slice(datapoints)))
	//}

	// maxLabelWidth + colon + one whitespace
	// 6 for percentages + 1 whitespace between box and label
	maxBarWidth := boxWidth - maxLabelWidth - 9
	sizes := make([]int, len(datapoints))
	maxBarWidthUsed := 0
	for i := range datapoints {
		percentage := datapoints[i]
		// round down
		width := int(percentage * float64(maxBarWidth))
		sizes[i] = width
		if width > maxBarWidthUsed {
			maxBarWidthUsed = width
		}
	}

	// calculate stretch factor
	whitespaceForBox := maxBarWidth - maxBarWidthUsed
	stretchFactor := float64(whitespaceForBox) / float64(maxBarWidth)

	if m.border {
		line := string(m.theme.TopLeft)

		// print title if it set and if it fits
		if m.title != "" && m.width-2-len(m.title)-2 > 0 {
			line += string(m.theme.Horizontal) + " " + m.title + " "
			line += strings.Repeat(string(m.theme.Horizontal), m.width-2-len(m.title)-3)
		} else {
			line += strings.Repeat(string(m.theme.Horizontal), m.width-2)
		}

		line += string(m.theme.TopRight)
		line += string('\n')

		b.WriteString(termenv.String(line).Foreground(m.borderColor).String())
	} else if m.title != "" {
		b.WriteString(m.title)
	}

	// render bar
	for i := range datapoints {
		if m.border {
			b.WriteString(termenv.String(string(m.theme.Vertical)).Foreground(m.borderColor).String())
			b.WriteRune(' ')
		}
		// write label
		b.WriteString(labels[i])
		b.WriteString(": ")

		// write gap
		gap := maxLabelWidth - len(labels[i])
		if gap > 0 {
			b.WriteString(strings.Repeat(" ", gap))
		}

		usedSpaces := maxLabelWidth + 2 // label + colon + space

		// render bar
		if sizes[i] > 0 {
			// stretch values if we still have space left
			size := sizes[i] + int(float64(sizes[i])*stretchFactor)
			val := strings.Repeat(string(m.EntryChar), size)
			b.WriteString(termenv.String(val).Foreground(colorMap[i]).String())
			b.WriteRune(' ')
			usedSpaces += size + 1
		}

		barLabel := m.labelFunc(i, datapoints)

		usedSpaces += len(barLabel)
		b.WriteString(barLabel)

		if m.border {
			gap := boxWidth - usedSpaces
			if gap > 0 {
				b.WriteString(strings.Repeat(" ", gap))
			}
			b.WriteRune(' ')
			b.WriteString(termenv.String(string(m.theme.Vertical)).Foreground(m.borderColor).String())
		}

		b.WriteRune('\n')
	}

	if m.border {
		b.WriteString(termenv.String(
			string(m.theme.BottomLeft) +
				strings.Repeat(string(m.theme.Horizontal), m.width-2) +
				string(m.theme.BottomRight) +
				string('\n'),
		).Foreground(m.borderColor).String())
	}

	return b.String()
}
