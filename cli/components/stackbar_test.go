// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"fmt"
	"testing"

	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v9/cli/theme/colors"
)

func TestStackBarGeneration(t *testing.T) {
	// initialize stack bar
	critical := colors.Profile.Color("#FD5CA4")
	high := colors.Profile.Color("#FF849C")
	good := colors.Profile.Color("#45C7BE")
	unknown := colors.Profile.Color("#ccc")

	data := StackBarData{
		Title:  "Stacked Data",
		Color:  []termenv.Color{critical, high, good, unknown},
		Labels: []string{"Error", "Failed", "Passed", "Unknown"},
		Entries: []StackBarDataEntry{
			{
				Key:    "policy 1",
				Values: []float64{.9, .1, 0, 0},
			},
			{
				Key:    "policy 2",
				Values: []float64{.2, .4, .1, .3},
			},
		},
	}

	stackBar := NewStackBar(WithWidth(65))
	output := stackBar.Render(data.Color, data.Entries[1].Values, 0)
	fmt.Println(output)
	expected := "█████████████░░░░░░░░░░░░░░░░░░░░░░░░░░▓▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░"
	assert.Equal(t, expected, output)

	chart := NewStackBarChart(StackBarChartPercentageLabelFunc)
	output = chart.Render(data)
	expected = "policy 1 ████████████████████████████████████████████████████████████████░░░░░░░ ( Error: 90% Failed: 10% Passed:  0% Unknown:  0% )\npolicy 2 ██████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░▓▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░░░ ( Error: 20% Failed: 40% Passed: 10% Unknown: 30% )\n"
	assert.Equal(t, expected, output)
}
