// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	tea "github.com/charmbracelet/bubbletea"
	"go.mondoo.com/cnspec/cli/reportmodel"
	"go.mondoo.com/cnspec/cli/reportview"
	"go.mondoo.com/cnspec/policy"
)

// The report viewer is a whole tea.Model of its own -- its own Init/Update/View,
// its own panes, its own key map -- so the launcher does not reimplement any of
// it. While a report is open the launcher delegates every message to it and
// draws whatever it returns.
//
// That leaves exactly one thing to decide: how the user gets back.
//
// `cnspec report view <file>` runs the same model standalone, where esc and q
// end the program, and the frame has no idea it is embedded. Rather than teach
// it (that package is finished and shared), the launcher intercepts the quit it
// asks for: a tea.Quit arriving out of the viewer means "done reading", and the
// launcher turns it into a return to the connector list. Nothing about the
// viewer changes, and a user who presses q lands back where they can scan the
// next thing -- which is the entire point of the background scan.
//
// ctrl+c is the exception and is handled before delegation: it quits the whole
// program from anywhere, which is what it does in every other pane too.

// viewerClosedMsg is a quit the viewer asked for, converted into a return.
type viewerClosedMsg struct{}

// viewerState is the report of the most recent scan and the embedded viewer
// drawing it.
//
// The two are one fact, not two: a viewer with no report draws nothing, and a
// report with no viewer cannot be shown. They were separate fields on Model and
// nothing but convention kept them in step -- the footer offers ^o on the
// strength of the flag and reopenViewer draws on the strength of the model.
type viewerState struct {
	model reportview.Model
	// loaded is whether there is a report to show. It stays true after the user
	// leaves the report, so that pressing esc is not the same as losing it.
	loaded bool
}

// open builds the viewer for a loaded report and hands it the terminal size,
// which it needs before it can lay anything out.
func (v *viewerState) open(collection *policy.ReportCollection, width, height int) tea.Cmd {
	// Embedded, because q hands control back to the launcher here rather than
	// ending the process, and a footer saying "quit" would promise otherwise.
	v.model = reportview.NewModel(reportmodel.New(collection)).Embedded()
	v.loaded = true
	// A model that has never seen a WindowSizeMsg has zero width and draws
	// nothing. The launcher already knows the size, so the viewer is told it at
	// construction rather than on the next resize the user happens to cause.
	// What that sizing returns is dropped: Init below is what the launcher runs.
	v.deliver(tea.WindowSizeMsg{Width: width, Height: height})
	return v.model.Init()
}

// deliver hands one message to the viewer and keeps what it returns.
//
// The type assertion is the whole reason this exists: reportview.Update returns
// a tea.Model, and three call sites each had their own copy of unwrapping it
// back into the concrete type. One that forgot would silently stop updating.
func (v *viewerState) deliver(msg tea.Msg) tea.Cmd {
	next, cmd := v.model.Update(msg)
	if vm, ok := next.(reportview.Model); ok {
		v.model = vm
	}
	return cmd
}

// update delegates a message to the viewer, translating a quit it asked for
// into a return to the launcher. See interceptQuit.
func (v *viewerState) update(msg tea.Msg) tea.Cmd { return interceptQuit(v.deliver(msg)) }

// resize tells the viewer the terminal size again, which is what reopening a
// report needs: the terminal may have changed shape while the report was away.
func (v *viewerState) resize(width, height int) tea.Cmd {
	return v.update(tea.WindowSizeMsg{Width: width, Height: height})
}

// view is the whole screen while a report is open: the viewer draws its own
// header, panes and footer rather than being framed by the launcher's.
func (v viewerState) view() string { return v.model.View() }

// interceptQuit rewrites a quit the viewer requested into viewerClosedMsg.
//
// A tea.Cmd is a function returning a message, so this wraps rather than
// inspects: the command still runs, and only its answer is translated. Batches
// are unwrapped because tea.Batch returns its children as a BatchMsg for the
// runtime to run, and the viewer's frame batches whatever its panes return.
func interceptQuit(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		return translateQuit(cmd())
	}
}

func translateQuit(msg tea.Msg) tea.Msg {
	switch msg := msg.(type) {
	case tea.QuitMsg:
		return viewerClosedMsg{}
	case tea.BatchMsg:
		out := make(tea.BatchMsg, 0, len(msg))
		for _, c := range msg {
			out = append(out, interceptQuit(c))
		}
		return out
	}
	return msg
}
