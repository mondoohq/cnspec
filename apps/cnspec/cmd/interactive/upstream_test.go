// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// TestMain pins the two things a launch would otherwise read off the machine it
// is running on.
//
// Where a scan would report: without this the suite reads the developer's own
// Mondoo configuration, and every assertion about an assembled command line
// changes depending on whether whoever ran it happens to be logged in -- the
// same form yields "scan aws" on a connected machine and "scan aws --incognito"
// on a fresh one. Tests that care about the other states set m.upstream
// themselves.
//
// And what a provider says a form means: plan() asks the connector's own
// ParseCLI, which starts a plugin subprocess. A suite that did that per case
// would spend a subprocess on every credential assertion in the package, and
// would answer differently on CI, where PROVIDERS_PATH points at an empty
// directory and nothing is installed. So the default is a stand-in, and the
// tests that genuinely want a real provider -- TestEveryCredentialFieldRoundTrips
// and the ssh join in inventory_test.go -- ask deliverypkg.Parser directly
// rather than going through this.
func TestMain(m *testing.M) {
	readUpstreamFn = func() upstreamState {
		return upstreamState{configured: true, scope: "test-space"}
	}
	defaultTestParser()
	os.Exit(m.Run())
}

func TestReportingIsOnlyWithCredentialsAndConsent(t *testing.T) {
	for _, tc := range []struct {
		name      string
		state     upstreamState
		reporting bool
		canToggle bool
	}{
		{"connected", upstreamState{configured: true}, true, true},
		{"chose local", upstreamState{configured: true, incognito: true}, false, true},
		{"no credentials", upstreamState{incognito: true}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.state.reporting(); got != tc.reporting {
				t.Errorf("reporting() = %v, want %v", got, tc.reporting)
			}
			if got := tc.state.canToggle(); got != tc.canToggle {
				t.Errorf("canToggle() = %v, want %v", got, tc.canToggle)
			}
		})
	}
}

// The badge is the only thing on screen saying whether findings leave this
// machine, so each state has to be distinguishable at a glance.
func TestTheBadgeNamesEachState(t *testing.T) {
	m := newTestModel()

	m.upstream = upstreamState{configured: true, scope: "keen-tesla-123456"}
	if got := m.upstream.badge(); !strings.Contains(got, "connected") || !strings.Contains(got, "keen-tesla-123456") {
		t.Errorf("connected badge = %q, want the state and the space", got)
	}

	m.upstream = upstreamState{configured: true, incognito: true}
	if got := m.upstream.badge(); !strings.Contains(got, "incognito") {
		t.Errorf("incognito badge = %q", got)
	}

	m.upstream = upstreamState{incognito: true}
	if got := m.upstream.badge(); !strings.Contains(got, "incognito") {
		t.Errorf("no-credentials badge = %q", got)
	}
}

// Toggling has to change the command, not just the label -- otherwise the badge
// is decoration and the user cannot tell what will actually run.
func TestTogglingReportingChangesTheCommand(t *testing.T) {
	m := selectEntry(t, newTestModel(), "local")
	m.upstream = upstreamState{configured: true}

	plan, err := m.launchArgs(m.list.entries[m.list.cursor], scanAction())
	if err != nil {
		t.Fatal(err)
	}
	if contains(plan.args, "--incognito") {
		t.Errorf("connected, yet the command says --incognito: %v", plan.args)
	}

	m.upstream.openModal()
	m.upstream.keyModal(tea.KeyMsg{Type: tea.KeyDown})
	m.upstream.keyModal(tea.KeyMsg{Type: tea.KeyEnter})
	if m.upstream.reporting() {
		t.Fatal("choosing local did not take")
	}
	if m.upstream.modal.open {
		t.Error("the chooser stayed open after a choice")
	}
	plan, err = m.launchArgs(m.list.entries[m.list.cursor], scanAction())
	if err != nil {
		t.Fatal(err)
	}
	if !contains(plan.args, "--incognito") {
		t.Errorf("incognito, yet the command does not say so: %v", plan.args)
	}
}

// A user who presses the key and sees nothing happen would be right to think it
// broken, so the refusal has to say why.
func TestTogglingWithoutCredentialsExplainsItself(t *testing.T) {
	m := newTestModel()
	m.upstream = upstreamState{incognito: true}

	m.upstream.openModal()
	m.upstream.keyModal(tea.KeyMsg{Type: tea.KeyUp})
	warn := m.upstream.keyModal(tea.KeyMsg{Type: tea.KeyEnter})

	if m.upstream.reporting() {
		t.Fatal("reporting without credentials")
	}
	if !m.upstream.modal.open {
		t.Error("the chooser closed on a choice that could not be honoured")
	}
	if warn == "" {
		t.Fatal("refused silently")
	}
	if !strings.Contains(warn, "cnspec login") {
		t.Errorf("warning does not say how to fix it: %q", warn)
	}
}

func TestSpaceNameIsTheReadablePart(t *testing.T) {
	for in, want := range map[string]string{
		"//captain.api.mondoo.app/spaces/keen-tesla-123456": "keen-tesla-123456",
		"keen-tesla-123456": "keen-tesla-123456",
		"":                  "",
	} {
		if got := spaceName(in); got != want {
			t.Errorf("spaceName(%q) = %q, want %q", in, got, want)
		}
	}
}

func contains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

// The badge opens a chooser rather than flipping silently, because the two
// answers differ in whether findings leave the machine and a one-word badge
// cannot carry that.
func TestTheChooserExplainsBothOptions(t *testing.T) {
	m := sized(newTestModel(), 100, 22)
	m.upstream = upstreamState{configured: true, scope: "keen-tesla-123456"}
	m.upstream.openModal()
	view := ansiStrip(m.View())

	for _, want := range []string{
		"Where do scan results go?",
		"Report to Mondoo Platform",
		"keen-tesla-123456",
		"Keep results on this machine",
		"--incognito",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("the chooser never says %q", want)
		}
	}
}

// Without credentials the option is not merely absent -- it is shown, disabled,
// with the reason and the fix. A missing option teaches nothing.
func TestTheChooserSaysWhyReportingIsUnavailable(t *testing.T) {
	m := sized(newTestModel(), 100, 22)
	m.upstream = upstreamState{incognito: true}
	m.upstream.openModal()
	view := ansiStrip(m.View())

	if !strings.Contains(view, "Report to Mondoo Platform") {
		t.Error("the unavailable option is hidden rather than explained")
	}
	if !strings.Contains(view, "cnspec login") {
		t.Error("the chooser does not say how to get credentials")
	}
}

// Esc leaves the decision as it was.
func TestCancellingTheChooserChangesNothing(t *testing.T) {
	m := sized(newTestModel(), 100, 22)
	m.upstream = upstreamState{configured: true}
	m.upstream.openModal()
	m.upstream.keyModal(tea.KeyMsg{Type: tea.KeyDown})
	m.upstream.keyModal(tea.KeyMsg{Type: tea.KeyEsc})

	if !m.upstream.reporting() {
		t.Error("esc applied the choice it was cancelling")
	}
	if m.upstream.modal.open {
		t.Error("esc left the chooser open")
	}
}

// The chooser is a body overlay like a picker: the view stays exactly the
// terminal height so the footer does not move under it.
func TestTheChooserKeepsTheViewHeight(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {100, 30}, {120, 40}, {60, 15}} {
		m := sized(newTestModel(), size[0], size[1])
		m.upstream = upstreamState{configured: true, scope: "s"}
		m.upstream.openModal()
		if got := len(strings.Split(m.View(), "\n")); got != size[1] {
			t.Errorf("%dx%d: view is %d lines, want %d", size[0], size[1], got, size[1])
		}
	}
}

func ansiStrip(s string) string { return ansi.Strip(s) }
