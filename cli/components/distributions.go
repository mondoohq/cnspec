// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnspec/v11/policy"
)

var (
	// Custom table title style.
	titleStyle = lipgloss.NewStyle().
			Bold(false).
			MarginBottom(0)

	// Custom table cell style.
	cellStyle = lipgloss.NewStyle().Padding(0, 0)
)

// DistributionsModel renders two tables, scores and asset distribution tables.
//
// E.g.
//
// Score Distribution                        Asset Distribution
// -------------------------------------     ----------------------------------------
// CRITICAL             1 assets             Ubuntu 16.04.7 LTS             1
// HIGH                 1 assets
// MEDIUM               1 assets
// LOW                  1 assets
type DistributionsModel struct {
	scoresTable table.Model
	assetsTable table.Model
}

// NewDistributions creates a new DistributionsModel.
func NewDistributions(assetsByScore map[string]int, assetsByPlatform map[string][]*inventory.Asset) DistributionsModel {
	return DistributionsModel{
		scoresTable: createScoresTable(assetsByScore),
		assetsTable: createAssetsTable(assetsByPlatform),
	}
}

// createScoresTable renders the score distribution table.
func createScoresTable(assetsByScore map[string]int) table.Model {
	columns := []table.Column{
		{Title: "---------------------", Width: 21},
		{Title: "----------------", Width: 21},
	}

	rows := []table.Row{}
	for _, score := range []string{
		policy.ScoreRatingTextCritical,
		policy.ScoreRatingTextHigh,
		policy.ScoreRatingTextMedium,
		policy.ScoreRatingTextLow,
		policy.ScoreRatingTextUnrated,
	} {
		if score == policy.ScoreRatingTextError || score == policy.ScoreRatingTextUnrated {
			if _, ok := assetsByScore[score]; !ok {
				continue
			}
		}

		rows = append(rows, table.Row{
			DefaultScoreRatingColors.LipglossStyle(score).Render(score),
			fmt.Sprintf("%d assets", assetsByScore[score]),
		})
	}

	return createTable(columns, rows)
}

// createAssetsTable renders the asset distribution table.
func createAssetsTable(assetsByPlatform map[string][]*inventory.Asset) table.Model {
	columns := []table.Column{
		{Title: "-------------------------------", Width: 31},
		{Title: "---------", Width: 11},
	}

	rows := []table.Row{}
	for platform := range assetsByPlatform {
		rows = append(rows, table.Row{
			platform, fmt.Sprintf("%d", len(assetsByPlatform[platform])),
		})
	}

	return createTable(columns, rows)
}

// createTable is a helper method to render tables with an specific style.
func createTable(cols []table.Column, rows []table.Row) table.Model {
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)+1),
		table.WithStyles(table.Styles{
			Cell: cellStyle,
		}),
	)
	return t
}

func (m DistributionsModel) Init() tea.Cmd {
	return nil
}

func (m DistributionsModel) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renders the component.
func (m DistributionsModel) View() string {
	left := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Score Distribution"),
		m.scoresTable.View(),
	)

	right := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Asset Distribution"),
		m.assetsTable.View(),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}
