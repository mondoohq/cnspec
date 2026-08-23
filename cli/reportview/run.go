// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/policy"
)

// Run opens the viewer on a report and blocks until the user quits.
//
// The alt-screen is deliberate: the viewer takes the terminal, and the scrollback
// the user had before it is theirs again afterwards. Mouse cell motion is on so
// panes can report clicks through their zones.
func Run(report *reportmodel.Report) error {
	m := NewModel(report)
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return errors.Wrap(err, "report viewer failed")
	}
	return nil
}

// RunCollection builds the model from a report collection and opens the viewer
// on it. It is the whole of what a caller with a loaded scan has to do.
func RunCollection(collection *policy.ReportCollection) error {
	return Run(reportmodel.New(collection))
}
