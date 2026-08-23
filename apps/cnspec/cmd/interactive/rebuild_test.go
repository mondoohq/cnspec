// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	tuiform "go.mondoo.com/cnspec/cli/tui/form"
)

// installed is the message a finished provider install sends, carrying the
// connector metadata the form is rebuilt from.
func installed(provider, connector, use string) providerInstalledMsg {
	return providerInstalledMsg{
		provider: provider,
		conns:    []Connector{{Provider: provider, Name: connector, Use: use}},
	}
}

// A provider install lands asynchronously and rebuilds the form. It can land
// while a picker is open, at which point the picker's field index refers to a
// field list that no longer exists.
func TestInstallDoesNotStrandAnOpenPicker(t *testing.T) {
	m := selectEntry(t, sized(newTestModel(), 120, 40), "ssh")
	m.picker.modal.open = true
	m.picker.modal.field = len(m.detail.form.Fields()) - 1
	if m.picker.modal.field < 1 {
		t.Fatal("expected the ssh form to have several fields")
	}

	nm, _ := m.Update(installed("os", "ssh", "ssh user@host"))
	m = nm.(Model)

	if m.picker.modal.open && !m.picker.modal.valid(m.detail.form.Fields()) {
		t.Fatalf("picker left pointing at field %d of %d", m.picker.modal.field, len(m.detail.form.Fields()))
	}
	// The panic this guards is in the renderer, so render.
	_ = m.View()
}

// The key path has to survive a stale index too, not only the renderer.
//
// keyModal used to test that the index was in range and then index with it
// regardless: an out-of-range one fell past the kind check straight into
// m.detail.form.Fields()[md.field]. Nothing shipped could reach it -- only
// rebuildForm and syncSelection shrink the field slice, and both close the
// picker -- so this drives it directly rather than through a sequence.
func TestKeyModalSurvivesAStaleIndex(t *testing.T) {
	m := selectEntry(t, sized(newTestModel(), 120, 40), "ssh")
	m.picker.modal = modalState{open: true, field: len(m.detail.form.Fields()) + 5}

	for _, k := range []tea.KeyType{tea.KeyEnter, tea.KeyDown, tea.KeyUp, tea.KeySpace, tea.KeyEsc} {
		nm, _ := m.Update(tea.KeyMsg{Type: k})
		got := nm.(Model)
		if got.picker.modal.open {
			t.Fatalf("a picker pointing past the end of the form stayed open after %v", k)
		}
		_ = got.View()
	}
}

// The same rebuild must not discard what the user typed while waiting for it.
func TestInstallKeepsTypedInput(t *testing.T) {
	m := selectEntry(t, sized(newTestModel(), 120, 40), "ssh")

	idx := -1
	for i := range m.detail.form.Fields() {
		if m.detail.form.Fields()[i].Kind == fieldText && m.detail.form.Fields()[i].Prefilled() == "" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("expected a free-text field on the ssh form")
	}
	m.detail.form.Fields()[idx].SetValue("root@prod-db-01")
	want := m.detail.form.Fields()[idx].Identity()

	// A real install adds metadata rather than removing it, so the rebuild is
	// driven by the connector the catalog actually holds.
	c, ok := m.list.current()
	if !ok {
		t.Fatal("no connector selected")
	}
	nm, _ := m.Update(providerInstalledMsg{provider: c.Provider, conns: []Connector{c}})
	m = nm.(Model)

	for _, fd := range m.detail.form.Fields() {
		if fd.Identity() == want {
			if fd.Value() != "root@prod-db-01" {
				t.Errorf("typed value lost across the rebuild: got %q", fd.Value())
			}
			return
		}
	}
	t.Errorf("field %q is gone from the rebuilt form", want)
}

// A rebuild re-derives prefilled values rather than preserving the old ones:
// the point of rebuilding is that the provider metadata is now available.
func TestCarryOverLeavesPrefillsToTheRebuild(t *testing.T) {
	old := tuiform.New("k8s", []field{
		prefilled(tuiform.Decl{Flag: "context"}, "stale-cluster", "current"),
		valued(tuiform.Decl{Flag: "namespace"}, "typed-by-hand"),
	})
	fresh := tuiform.New("k8s", []field{
		prefilled(tuiform.Decl{Flag: "context"}, "real-current", "current"),
		tuiform.NewField(tuiform.Decl{Flag: "namespace"}),
	})
	carryOver(&fresh, old)

	if fresh.Fields()[0].Value() != "real-current" {
		t.Errorf("prefill was overwritten by the stale one: %q", fresh.Fields()[0].Value())
	}
	if fresh.Fields()[1].Value() != "typed-by-hand" {
		t.Errorf("user input was not carried: %q", fresh.Fields()[1].Value())
	}
}

// Carrying over is keyed on what a field asks, not where it sits: a rebuild
// that inserts a field must not shift values onto their neighbours.
func TestCarryOverSurvivesReordering(t *testing.T) {
	old := tuiform.New("aws", []field{
		valued(tuiform.Decl{Flag: "profile"}, "prod"),
		valued(tuiform.Decl{Flag: "region"}, "eu-central-1"),
	})
	fresh := tuiform.New("aws", []field{
		tuiform.NewField(tuiform.Decl{Flag: "endpoint-url"}),
		tuiform.NewField(tuiform.Decl{Flag: "region"}),
		tuiform.NewField(tuiform.Decl{Flag: "profile"}),
	})
	carryOver(&fresh, old)

	if got := fresh.Fields()[2].Value(); got != "prod" {
		t.Errorf("profile = %q, want prod", got)
	}
	if got := fresh.Fields()[1].Value(); got != "eu-central-1" {
		t.Errorf("region = %q, want eu-central-1", got)
	}
	if got := fresh.Fields()[0].Value(); got != "" {
		t.Errorf("endpoint-url picked up a value it never had: %q", got)
	}
}

// A different connector's form shares nothing with this one.
func TestCarryOverIgnoresADifferentConnector(t *testing.T) {
	old := tuiform.New("aws", []field{valued(tuiform.Decl{Flag: "profile"}, "prod")})
	fresh := tuiform.New("gcp", []field{tuiform.NewField(tuiform.Decl{Flag: "profile"})})
	carryOver(&fresh, old)

	if fresh.Fields()[0].Value() != "" {
		t.Errorf("carried a value across connectors: %q", fresh.Fields()[0].Value())
	}
}

// Pressing Scan with a required field still empty must land the cursor on that
// field. The launcher used to report the error and leave the cursor where it
// was, which says what is wrong without saying where.
func TestScanFocusesTheMissingField(t *testing.T) {
	m := selectEntry(t, sized(newTestModel(), 120, 40), "ssh")

	want := -1
	for i := range m.detail.form.Fields() {
		m.detail.form.Fields()[i].SetValue("filled")
		if m.detail.form.Fields()[i].Required && want < 0 {
			want = i
		}
	}
	if want < 0 {
		t.Skip("no required field on this form")
	}
	m.detail.form.Fields()[want].SetValue("") // the one thing standing in the way
	m.detail.form.SetCursor(len(m.detail.form.Fields()) - 1)

	nm, _ := m.launch()
	m = nm.(Model)

	if m.lastErr == "" {
		t.Fatal("expected a validation error")
	}
	if m.detail.form.Cursor() != want {
		t.Errorf("cursor = %d (%q), want %d (%q)",
			m.detail.form.Cursor(), m.detail.form.Fields()[m.detail.form.Cursor()].Label,
			want, m.detail.form.Fields()[want].Label)
	}
}
