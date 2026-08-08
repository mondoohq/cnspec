// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// sampleCatalog is a small hermetic catalog so model tests don't depend on the
// installed provider set.
func sampleCatalog() []Connector {
	return []Connector{
		{Provider: "os", Name: "local", Use: "local", Short: "your local system", Category: catHosts, Installed: true},
		{Provider: "os", Name: "ssh", Use: "ssh user@host", Short: "a remote system via SSH", Category: catHosts},
		{Provider: "os", Name: "docker", Use: "docker", Short: "a running Docker container", Category: catContainer},
		{Provider: "aws", Name: "aws", Use: "aws", Short: "an AWS account", Category: catCloud},
		{Provider: "k8s", Name: "k8s", Use: "k8s", Short: "a Kubernetes cluster", Category: catContainer},
		{Provider: "mongo", Name: "mongo", Use: "mongo [host]", Short: "a MongoDB server", Category: catDatabase},
	}
}

func typeString(m Model, s string) Model {
	for _, r := range s {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm.(Model)
	}
	return m
}

func key(m Model, t tea.KeyType) Model {
	nm, _ := m.Update(tea.KeyMsg{Type: t})
	return nm.(Model)
}

func TestFilterNarrowsResults(t *testing.T) {
	m := NewModel(sampleCatalog())
	m = typeString(m, "aws")
	if len(m.filtered) != 1 || m.filtered[0].Name != "aws" {
		t.Fatalf("expected only aws to match, got %d results", len(m.filtered))
	}

	m2 := NewModel(sampleCatalog())
	m2 = typeString(m2, "database")
	if len(m2.filtered) != 1 || m2.filtered[0].Name != "mongo" {
		t.Fatalf("expected category search to find mongo, got %d results", len(m2.filtered))
	}
}

func TestFullFlowProducesArgs(t *testing.T) {
	m := NewModel(sampleCatalog())
	m = typeString(m, "aws")
	m = key(m, tea.KeyEnter) // select connector -> action step
	if m.step != stepAction {
		t.Fatalf("expected action step, got %d", m.step)
	}
	m = key(m, tea.KeyEnter) // pick first action (scan) -> confirm
	if m.step != stepConfirm {
		t.Fatalf("expected confirm step, got %d", m.step)
	}
	m = key(m, tea.KeyEnter) // launch
	if !m.launched {
		t.Fatal("expected launched=true")
	}
	if got := strings.Join(m.result, " "); got != "scan aws" {
		t.Fatalf("expected 'scan aws', got %q", got)
	}
}

func TestActionFilteringForConnector(t *testing.T) {
	// aws does not support sbom; ensure it is not offered.
	got := ActionsFor("aws")
	for _, a := range got {
		if a.Name == "sbom" {
			t.Fatal("sbom should not be offered for aws")
		}
	}
	// local supports vuln, sbom and aibom.
	names := map[string]bool{}
	for _, a := range ActionsFor("local") {
		names[a.Name] = true
	}
	for _, want := range []string{"scan", "shell", "run", "discover", "vuln", "sbom", "aibom"} {
		if !names[want] {
			t.Fatalf("expected action %q for local", want)
		}
	}
}

func TestExtraArgsTokenized(t *testing.T) {
	m := NewModel(sampleCatalog())
	m = typeString(m, "ssh")
	m = key(m, tea.KeyEnter) // -> action
	m = key(m, tea.KeyEnter) // scan -> confirm
	m = typeString(m, "user@host")
	m = key(m, tea.KeyEnter)
	if got := strings.Join(m.result, " "); got != "scan ssh user@host" {
		t.Fatalf("expected 'scan ssh user@host', got %q", got)
	}
}

func TestTokenizeQuotes(t *testing.T) {
	got := tokenize(`-c "aws.regions.length"`)
	if len(got) != 2 || got[0] != "-c" || got[1] != "aws.regions.length" {
		t.Fatalf("unexpected tokenize result: %#v", got)
	}
}

func TestBackNavigation(t *testing.T) {
	m := NewModel(sampleCatalog())
	m = typeString(m, "aws")
	m = key(m, tea.KeyEnter) // action
	m = key(m, tea.KeyEsc)   // back to connector
	if m.step != stepConnector {
		t.Fatalf("expected to return to connector step, got %d", m.step)
	}
}
