// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/internal/bundle"
)

// authoringModel opens the authoring pane without requiring a coding agent on
// the machine: openAuthor builds a real generator, which needs an installed
// agent CLI, and nothing below exercises generation itself.
func authoringModel(t *testing.T) Model {
	t.Helper()
	m := newTestModel()
	m.phase = phaseAuthoring
	m.author = newAuthorState(1, nil, filepath.Join(t.TempDir(), "out.mql.yaml"))
	return m
}

func TestAuthorPaneTakesTheScreen(t *testing.T) {
	m := authoringModel(t)
	view := m.View()
	if !strings.Contains(view, "Author a check") {
		t.Fatalf("authoring pane is not on screen:\n%s", view)
	}
	// the launcher behind it must not also be drawing its list
	if strings.Contains(view, "a remote system via SSH") {
		t.Error("the connector list is visible behind the authoring pane")
	}
}

func TestAuthorFooterHintIsOffered(t *testing.T) {
	m := newTestModel()
	if !strings.Contains(m.View(), "^g") {
		t.Error("the launcher does not offer the authoring key")
	}
}

func TestCtrlGOpensAuthoring(t *testing.T) {
	m := newTestModel()
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	got := nm.(Model)
	// On a machine with no coding agent installed openAuthor reports why
	// instead of opening; either outcome is correct, silence is not.
	if got.phase != phaseAuthoring && got.lastErr == "" {
		t.Error("^g neither opened the authoring pane nor said why it could not")
	}
}

// TestAuthorEnterAdvancesRatherThanGenerating pins the guard on a billed call:
// enter walks the fields, and only the last one commits.
func TestAuthorEnterAdvancesRatherThanGenerating(t *testing.T) {
	m := authoringModel(t)
	m = typeString(m, "SSH must not permit root login")

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.author.cursor != fieldDesc {
		t.Fatalf("enter did not advance to the description, cursor=%d", m.author.cursor)
	}
	if m.author.step == authorWorking || cmd != nil {
		t.Error("enter on the first field started a generation")
	}
}

// TestAuthorProposesFilterFromTitle pins that the guess is shown for review
// rather than applied invisibly, and that it uses a platform name that exists.
func TestAuthorProposesFilterFromTitle(t *testing.T) {
	m := authoringModel(t)
	m = typeString(m, "kubernetes pod must not run privileged")

	got := m.author.fields[fieldFilter]
	if got == "" {
		t.Fatal("no asset filter was proposed for a Kubernetes check")
	}
	if strings.Contains(got, `asset.platform == "k8s"`) {
		t.Errorf("proposed a platform that does not exist: %q", got)
	}
	if !strings.Contains(got, "k8s-cluster") {
		t.Errorf("proposed filter does not scope to a real k8s platform: %q", got)
	}
}

// TestAuthorKeepsAHandEditedFilter: a proposal must never overwrite the user.
func TestAuthorKeepsAHandEditedFilter(t *testing.T) {
	m := authoringModel(t)
	m.author.cursor = fieldFilter
	m = typeString(m, "asset.platform == \"custom\"")

	m.author.cursor = fieldTitle
	m = typeString(m, "aws s3 bucket must be encrypted")

	if got := m.author.fields[fieldFilter]; got != `asset.platform == "custom"` {
		t.Errorf("a hand-written filter was overwritten by a proposal: %q", got)
	}
}

// TestAuthorDropsStaleGeneration is why authorState carries a sequence number.
//
// The hazard is not a result arriving after the pane closed -- the phase
// dispatch already drops that, without consulting seq. It is a result from an
// abandoned run arriving while a *later* run is in flight: both are addressed
// to an open authoring pane, and only the sequence number tells them apart. A
// test that merely cancels and delivers proves nothing; this one passed with
// the guard deleted until it was rewritten to start the second run.
func TestAuthorDropsStaleGeneration(t *testing.T) {
	m := authoringModel(t)
	m.author.step = authorWorking
	stale := m.author.seq

	// the user regenerates: run one is abandoned, run two is now in flight
	m.author.seq++
	current := m.author.seq

	nm, _ := m.Update(authorGeneratedMsg{seq: stale, mql: "should.not.appear"})
	m = nm.(Model)

	if m.author.mql == "should.not.appear" {
		t.Error("an abandoned run's answer replaced the one in flight")
	}
	if m.author.step != authorWorking {
		t.Errorf("an abandoned run's answer ended the wait for the current one, step=%d", m.author.step)
	}

	// and the run actually in flight still lands
	nm, _ = m.Update(authorGeneratedMsg{seq: current, mql: "expected.query"})
	m = nm.(Model)
	if m.author.mql != "expected.query" {
		t.Errorf("the current run's answer did not land, got %q", m.author.mql)
	}
}

// TestAuthorWritesScannableBundle is the end of the flow: what the pane accepts
// has to be a bundle cnspec can actually scan, which means a policy group and
// not a bare top-level query.
func TestAuthorWritesScannableBundle(t *testing.T) {
	m := authoringModel(t)
	file := m.author.fields[fieldFile]
	m.author.fields[fieldTitle] = "SSH must not permit root login"
	m.author.fields[fieldDesc] = "PermitRootLogin must be no"
	m.author.fields[fieldFilter] = `asset.family.contains("linux")`
	m.author.mql = `sshd.config.params["PermitRootLogin"] == "no"`
	m.author.step = authorReview

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("accepting the candidate produced no write")
	}
	msg := cmd()
	nm, _ = m.Update(msg)
	m = nm.(Model)

	if m.author.step != authorDone {
		t.Fatalf("write did not complete: step=%d err=%q", m.author.step, m.author.err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("nothing was written: %v", err)
	}
	b, err := bundle.ParseYaml(data)
	if err != nil {
		t.Fatalf("what was written does not parse: %v", err)
	}
	if len(b.Policies) == 0 {
		t.Fatal("no policy was written; a bare top-level query is not scannable")
	}
	var found bool
	for _, g := range b.Policies[0].Groups {
		for _, c := range g.Checks {
			if c != nil && strings.Contains(c.Mql, "PermitRootLogin") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("the accepted check is not in a policy group:\n%s", data)
	}
}

// TestAuthorRefusesAnEmptyQuery: accepting nothing must not write a check with
// an empty body, which lints clean and never fails.
func TestAuthorRefusesAnEmptyQuery(t *testing.T) {
	m := authoringModel(t)
	m.author.step = authorReview
	m.author.fields[fieldTitle] = "something"

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if cmd != nil {
		t.Error("accepting an empty candidate started a write")
	}
	if m.author.err == "" {
		t.Error("accepting an empty candidate said nothing")
	}
}

// TestAuthorSurfacesAWriteFailure: a check the user accepted and that did not
// land has to say so, not return to the launcher as though it had.
func TestAuthorSurfacesAWriteFailure(t *testing.T) {
	m := authoringModel(t)
	m.author.step = authorReview
	nm, _ := m.Update(authorWroteMsg{seq: m.author.seq, err: errors.New("disk full")})
	m = nm.(Model)
	if !strings.Contains(m.author.err, "disk full") {
		t.Errorf("a failed write was not reported: %q", m.author.err)
	}
	if m.author.step == authorDone {
		t.Error("a failed write was reported as done")
	}
}

// TestAuthorAnotherCheckAcceptsTyping pins a crash: "author another" rebuilt
// the pane by struct literal and left the text box zero-valued, so the next
// keystroke panicked on a nil cursor. Every field of authorState has a usable
// zero value except that one, which is why construction goes through
// newAuthorState.
func TestAuthorAnotherCheckAcceptsTyping(t *testing.T) {
	m := authoringModel(t)
	m.author.step = authorDone
	m.author.wrote = "out.mql.yaml"

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = nm.(Model)
	if m.author.step != authorIntent {
		t.Fatalf("'a' did not start another check, step=%d", m.author.step)
	}

	// the keystroke that used to panic
	m = typeString(m, "another check")
	if got := m.author.fields[fieldTitle]; got != "another check" {
		t.Errorf("typing after 'author another' did not reach the title: %q", got)
	}
}

// TestAuthorFocusedRowDrawsTheLiveInput pins the difference between a form
// field and a menu row.
//
// The focused row has to draw the text box, because that is what carries the
// cursor. Drawing the stored string instead -- or banding the whole row the way
// a list selection does -- leaves an empty field as a slab of accent colour
// with nothing to show where typing would go. view.go makes the same split for
// the connector form: only the label takes the band.
//
// The assertion is on the value drawn rather than on colour, because lipgloss
// emits no escape codes without a TTY and a test for them passes vacuously in
// CI.
func TestAuthorFocusedRowDrawsTheLiveInput(t *testing.T) {
	m := authoringModel(t)
	m.author.moveCursor(fieldTitle)

	// force a divergence the renderer has to resolve one way or the other
	m.author.fields[fieldTitle] = "STORED"
	m.author.input.SetValue("LIVE")

	view := m.View()
	if !strings.Contains(view, "LIVE") {
		t.Error("the focused row does not draw the text box, so it carries no cursor")
	}
	if strings.Contains(view, "STORED") {
		t.Error("the focused row drew the stored value instead of the live input")
	}
}

// TestAuthorEmptyFieldShowsAPlaceholder: a blank row must read as "not filled
// in yet", not as a value that failed to load.
func TestAuthorEmptyFieldShowsAPlaceholder(t *testing.T) {
	m := authoringModel(t)
	m.author.moveCursor(fieldTitle) // so Desc is unfocused and empty

	if !strings.Contains(m.View(), fieldDesc.placeholder()) {
		t.Errorf("an empty unfocused field shows nothing at all:\n%s", m.View())
	}
}
