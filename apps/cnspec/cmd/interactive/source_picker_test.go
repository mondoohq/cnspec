// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// What a picker's failure looks like on the screen.
//
// The pickers themselves are cli/launcher/source and these tests are not: what
// they assert is that a reason a source produced reaches a hint, a footer and a
// modal, which is the model's half of the bargain rather than the source's.

// Only the profile name is a command-line argument; the label the picker adds
// is for reading. The mapping is the source's, and this is the launcher
// honouring it.
func TestAWSProfileLabelNeverReachesArgv(t *testing.T) {
	f := newForm(awsConnector())
	fd := fieldByLabel(t, f, "profile")
	fd.SetValue("sso-prod  (123456789012)")
	if got := strings.Join(f.Args(), " "); got != "--profile sso-prod" {
		t.Fatalf("args = %q, want the bare profile name", got)
	}
}

// A failed lookup must say why on the screen, not just that it failed.
func TestPickerShowsTheReasonItIsEmpty(t *testing.T) {
	m := selectEntry(t, sized(newTestModel(), 140, 30), "ssh")
	fd := m.detail.form.Fields()[0]
	setSourceErr(&m, fd.Source(), errors.New("gcloud needs signing in again — run: gcloud auth login"))

	if got := m.picker.choiceHint(m.detail.form, fd); !strings.Contains(got, "gcloud auth login") {
		t.Fatalf("hint = %q, want the actual reason", got)
	}
}

// A live refresh that failed must say so even when the cheap read succeeded.
// The list is showing, it is just incomplete, and without this the failure is
// invisible: the field looks like a complete answer.
func TestLiveFailureIsVisibleBesideLocalValues(t *testing.T) {
	c := Connector{
		Provider: "gcp", Name: "gcp", Use: "gcp", Category: catCloud,
		Installed: true, MaxArgs: 2,
	}
	m := sized(NewModel([]Connector{c}), 140, 30)
	fieldByLabel(t, m.detail.form, "what to scan").SetValue("project")
	resolveSources(&m.detail.form)

	// The cheap read succeeded and prefilled.
	setSource(&m, srcGCPProject, []string{"attack-surface-scanner"})
	m.detail.applyPrefill(srcGCPProject, []string{"attack-surface-scanner"}, m.focus == focusForm)

	// The live one did not.
	liveErr := errors.New("gcloud needs signing in again — run: gcloud auth login")
	setSourceErr(&m, srcGCPProjectAll, liveErr)

	id := fieldByLabel(t, m.detail.form, "id")
	if got := m.picker.liveError(m.detail.form, *id); !strings.Contains(got, "gcloud auth login") {
		t.Fatalf("liveSourceError = %q", got)
	}

	// In the footer, which is the only place with room for a sentence.
	m.focus = focusForm
	m.lastWarn = liveErr.Error()
	if out := ansi.Strip(m.View()); !strings.Contains(out, "gcloud auth login") {
		t.Errorf("the footer should carry the reason:\n%s", out)
	}

	// And inside the picker, where the incomplete list is.
	for i, fd := range m.detail.form.Fields() {
		if fd.Label == "id" {
			m.detail.form.SetCursor(i)
		}
	}
	m.picker.modal = modalState{open: true, field: m.detail.form.Cursor()}
	if out := ansi.Strip(m.View()); !strings.Contains(out, "gcloud auth login") {
		t.Errorf("the picker should carry the reason:\n%s", out)
	}
}
