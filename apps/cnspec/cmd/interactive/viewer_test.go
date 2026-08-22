// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mondoo.com/cnspec/cli/reportview"

	tea "github.com/charmbracelet/bubbletea"
	"go.mondoo.com/cnspec/cli/reporter"
)

// recordedReport is the artifact a real scan wrote, read back the way the
// launcher reads it.
func recordedReport(t *testing.T) Model {
	t.Helper()

	collection, err := reporter.LoadCollectionFile(filepath.Join("testdata", "scan_report.json"))
	if err != nil {
		t.Fatalf("the recorded report no longer loads: %v", err)
	}

	m := newTestModel()
	m.width, m.height = 120, 40
	m.viewer.open(collection, m.width, m.height)
	m.phase = phaseViewing
	return m
}

// press delivers one key and runs whatever the model asked for, which is where
// a quit the viewer requested gets translated.
func press(t *testing.T, m Model, key string) Model {
	t.Helper()

	stroke := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	if key == "esc" {
		stroke = tea.KeyMsg{Type: tea.KeyEsc}
	}

	model, cmd := m.Update(stroke)
	m = model.(Model)
	if cmd == nil {
		return m
	}
	msg := cmd()
	if msg == nil {
		return m
	}
	model, _ = m.Update(msg)
	return model.(Model)
}

// The whole point of not handing the terminal over is scanning the next thing
// without leaving, so q and esc return to the launcher rather than ending the
// program -- even though the same viewer, run standalone by `cnspec report
// view`, quits on both.
func TestLeavingTheViewerReturnsToTheLauncher(t *testing.T) {
	for _, key := range []string{"q", "esc"} {
		t.Run(key, func(t *testing.T) {
			m := press(t, recordedReport(t), key)
			if m.phase != phaseForm {
				t.Fatalf("%q did not return to the launcher (phase %d)", key, m.phase)
			}
			if !m.viewer.loaded {
				t.Error("leaving the report is not the same as losing it")
			}
			if !strings.Contains(m.View(), "Connectors") {
				t.Error("the connector list is not back on screen")
			}
		})
	}
}

// ctrl+c is the exception, and it is handled before the viewer sees it: it ends
// the program from anywhere, which is what it does in every other pane.
func TestCtrlCQuitsFromTheViewer(t *testing.T) {
	m := recordedReport(t)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c did nothing")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c did not quit, it produced %T", cmd())
	}
}

// The report of the last scan stays in memory, so a stray esc does not throw
// away what the user just waited for.
func TestTheLastReportCanBeReopened(t *testing.T) {
	m := press(t, recordedReport(t), "q")

	if !strings.Contains(m.View(), "^o") {
		t.Error("the launcher does not offer the report it is holding")
	}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = model.(Model)
	if m.phase != phaseViewing {
		t.Fatal("the report did not reopen")
	}
	if !strings.Contains(m.View(), "launcher-test-host") {
		t.Error("the reopened report is not the one that was scanned")
	}
}

// Offering a key for a report that does not exist is a promise the launcher
// cannot keep.
func TestReopeningWithoutAReportSaysSo(t *testing.T) {
	m := newTestModel()
	m.width, m.height = 120, 40

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = model.(Model)
	if m.phase != phaseForm {
		t.Fatal("there is nothing to open")
	}
	if m.lastWarn == "" {
		t.Error("the key did nothing and said nothing, which reads as swallowed")
	}
	if strings.Contains(m.View(), "^o") {
		t.Error("a hint is offered for a report that does not exist")
	}
}

// The launcher's testing style: assert the geometry, not a screenshot. Every
// screen it draws is exactly the terminal's height, because the footer has to
// land on the last row.
func TestEveryPhaseFillsTheTerminalExactly(t *testing.T) {
	m := recordedReport(t)
	for name, model := range map[string]Model{
		"viewing":  m,
		"scanning": scanningModel(t),
		"form":     press(t, m, "q"),
	} {
		lines := strings.Split(model.View(), "\n")
		if len(lines) != model.height {
			t.Errorf("the %s screen drew %d lines into a %d-line terminal",
				name, len(lines), model.height)
		}
	}
}

// scanningModel is a launcher watching a scan, with the recorded progress
// stream already folded in.
func scanningModel(t *testing.T) Model {
	t.Helper()

	m := newTestModel()
	m.width, m.height = 120, 40
	m.phase = phaseScanning
	m.scan.session = newScanSession(launchPlan{args: []string{"scan", "local"}}, "local")
	t.Cleanup(m.scan.session.cancel)

	raw, err := os.Open(filepath.Join("testdata", "scan_progress.ndjson"))
	if err != nil {
		t.Fatalf("the recorded progress stream is missing: %v", err)
	}
	defer func() { _ = raw.Close() }()

	// The same reader the child's pipe feeds, over the recording instead.
	m.scan.session.readProgress(raw)
	return m
}

// The scanning screen exists because a background child is invisible: its
// stdout is not a terminal, so nothing it would normally print arrives. What it
// shows has to come from the progress stream, and this is that stream.
func TestTheRecordedProgressStreamReachesTheScreen(t *testing.T) {
	m := scanningModel(t)

	snap := m.scan.session.prog.snapshot()
	if snap.Total != 1 || snap.Done != 1 {
		t.Fatalf("the recording describes one finished asset, got %+v", snap)
	}

	view := m.View()
	if !strings.Contains(view, "launcher-test-host") {
		t.Errorf("the asset being scanned is not named:\n%s", view)
	}
	if !strings.Contains(view, "1 of 1 assets") {
		t.Errorf("the screen does not say how far along it is:\n%s", view)
	}
	if !strings.Contains(view, "cancel") {
		t.Errorf("a scan with no way out is worse than no scan:\n%s", view)
	}
}

// Esc while scanning cancels rather than backing out of a form.
func TestEscCancelsARunningScan(t *testing.T) {
	m := scanningModel(t)

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(Model)

	if m.phase != phaseForm {
		t.Fatal("cancelling did not return to the launcher")
	}
	if m.scan.active() {
		t.Error("the cancelled session is still attached to the model")
	}
	if m.lastWarn == "" {
		t.Error("a cancelled scan disappears without a word")
	}
}

// An event from a scan the user has already cancelled must not reanimate the
// screen they left.
func TestAStaleSessionIsIgnored(t *testing.T) {
	m := scanningModel(t)
	stale := m.scan.session

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(Model)

	for _, msg := range []tea.Msg{
		scanRunningMsg{session: stale},
		scanProgressMsg{session: stale},
		scanExitedMsg{session: stale},
		scanReportMsg{session: stale},
	} {
		model, _ := m.Update(msg)
		m = model.(Model)
		if m.phase != phaseForm {
			t.Fatalf("%T from a cancelled scan moved the launcher to phase %d", msg, m.phase)
		}
	}
	if m.viewer.loaded {
		t.Error("a cancelled scan produced a report")
	}
}

// A write can outlive the view that started it: close the report while a large
// export is still running and the result arrives with the viewer gone. It must
// land in the footer, not be dropped -- the user is waiting to hear that their
// file was written.
func TestALateExportNoticeReachesTheFooter(t *testing.T) {
	m := sized(newTestModel(), 100, 24)
	m.phase = phaseForm

	nm, _ := m.Update(reportview.ExportDoneMsg{
		Format: "json-full",
		Path:   "/tmp/report.full.json",
		Size:   1024,
	})
	got := nm.(Model)
	if got.lastNote == "" && got.lastWarn == "" && got.lastErr == "" {
		t.Fatal("the export result was dropped")
	}
	if !strings.Contains(got.lastNote+got.lastWarn+got.lastErr, "report.full.json") {
		t.Errorf("the notice does not name the file: %q%q%q", got.lastNote, got.lastWarn, got.lastErr)
	}
	// A file that was written is not a warning. This used to go out through
	// lastWarn, so the launcher announced a successful export in the warning
	// color behind a "!"; lastNote is the third channel that fixed it.
	if got.lastWarn != "" || got.lastErr != "" {
		t.Errorf("a successful export was reported as trouble: warn=%q err=%q", got.lastWarn, got.lastErr)
	}
}
