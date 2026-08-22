// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package form

import (
	"strings"
	"testing"
)

// What this package is for is the difference between the three ways of asking a
// field for its answer. Reading the wrong one is the bug the boundary exists to
// make unavailable, so each is stated here rather than only in the launcher's
// own suite.
func TestValueDisplayAndEmittedAreThreeDifferentQuestions(t *testing.T) {
	fd := NewField(Decl{Label: "profile", Flag: "profile", Kind: KindChoice})
	fd.SetSource("profiles", func(display string) string {
		return strings.TrimSuffix(display, "  (123456789012)")
	})
	fd.SetValue("prod  (123456789012)")

	if got := fd.Value(); got != "prod  (123456789012)" {
		t.Errorf("Value = %q, want what the user chose", got)
	}
	if got := fd.Display(); got != "prod  (123456789012)" {
		t.Errorf("Display = %q, want what the user chose", got)
	}
	if got := fd.Emitted(); got != "prod" {
		t.Errorf("Emitted = %q, want the annotation stripped", got)
	}

	secret := NewField(Decl{Label: "password", Flag: "password", Kind: KindText, Secret: true})
	secret.SetValue("must-never-reach-argv")
	if got := secret.Display(); got != strings.Repeat("•", len("must-never-reach-argv")) {
		t.Errorf("a secret rendered as %q", got)
	}
	if got := secret.Value(); got != "must-never-reach-argv" {
		t.Errorf("Value = %q, want the secret itself -- it has to reach a vault", got)
	}
}

// A secret never reaches the command line, whatever else the form is asked for.
func TestArgsNeverCarryASecret(t *testing.T) {
	password := NewField(Decl{Label: "password", Flag: "password", Kind: KindText, Secret: true})
	password.SetValue("must-never-reach-argv")
	host := NewField(Decl{Label: "host", Pos: 0, Kind: KindText})
	host.SetValue("example.com")

	f := New("ssh", []Field{host, password})
	line := strings.Join(f.Args(), " ")
	if strings.Contains(line, "must-never-reach-argv") {
		t.Fatalf("the secret reached argv: %q", line)
	}
	if line != "example.com" {
		t.Fatalf("args = %q, want just the target", line)
	}
	if got := f.Secrets(); len(got) != 1 || got[0].Flag != "password" {
		t.Fatalf("Secrets = %v, want the one filled secret", got)
	}
}

// A selector steers the screen without putting its own words on the command
// line, which is what a declared mapping to "" is for.
func TestASelectorCanEmitNothing(t *testing.T) {
	kind := NewField(Decl{Label: "kind", Pos: 0, Kind: KindChoice,
		Options: []string{"live cluster", "manifest file"}})
	kind.SetEmit(func(display string) string {
		return map[string]string{"live cluster": "", "manifest file": ""}[display]
	})
	kind.SetValue("live cluster")
	path := NewField(Decl{Label: "path", Pos: 1, Kind: KindText,
		ShowIf: []string{"manifest file"}, DependsOn: 0})
	path.SetValue("./left-over.yaml")

	f := New("k8s", []Field{kind, path})
	if got := f.Args(); len(got) != 0 {
		// The path is hidden by the selector, so it is not part of the command
		// either -- a value the user cannot see is not one they are asking for.
		t.Fatalf("args = %v, want nothing at all", got)
	}
}

// Inserting a row ahead of a selector must not silently re-point it at its new
// neighbour, which is what plain int indices invite.
func TestInsertFieldKeepsDependenciesPointingAtTheSameField(t *testing.T) {
	f := New("test", []Field{
		NewField(Decl{Label: "selector"}),
		NewField(Decl{Label: "dependent", ShowIf: []string{"x"}, DependsOn: 0}),
	})
	f.InsertField(0, NewField(Decl{Label: "readout", Kind: KindCredentialState}))

	if got := f.Fields()[2].DependsOn; got != 1 {
		t.Errorf("the dependent now points at field %d, want 1", got)
	}
	if got := f.Fields()[0].DependsOn; got != 0 {
		t.Errorf("the inserted row gained a dependency: %d", got)
	}
}

// A picker and the mapping that says what its values mean travel together, so
// switching a selector cannot leave the old mapping behind.
func TestResolveSourcesSwapsTheMappingWithThePicker(t *testing.T) {
	emitters := map[string]Emit{
		"annotated": func(display string) string { return strings.TrimSuffix(display, " (x)") },
		"plain":     nil,
	}
	f := New("test", []Field{
		NewField(Decl{Label: "kind", Pos: 0, Kind: KindChoice}),
		NewField(Decl{Label: "thing", Flag: "thing", Kind: KindChoice,
			SourceBy: map[string]string{"a": "annotated", "p": "plain"}}),
	})

	f.Fields()[0].SetValue("a")
	f.ResolveSources(func(id string) Emit { return emitters[id] })
	f.Fields()[1].SetValue("name (x)")
	if got := f.Fields()[1].Emitted(); got != "name" {
		t.Fatalf("Emitted = %q, want the annotated picker's mapping applied", got)
	}

	f.Fields()[0].SetValue("p")
	f.ResolveSources(func(id string) Emit { return emitters[id] })
	if got := f.Fields()[1].Emitted(); got != "name (x)" {
		t.Fatalf("Emitted = %q, want the old mapping gone with the old picker", got)
	}
}

// A multi-choice with nothing to pick from is a row no keystroke can fill, so
// it becomes a text box that takes the same comma-separated value.
func TestAnUnfillableListBecomesTypeable(t *testing.T) {
	f := New("helm", []Field{
		NewField(Decl{Label: "values", Flag: "values", Kind: KindMultiChoice, Desc: "value files"}),
		NewField(Decl{Label: "discover", Flag: "discover", Kind: KindMultiChoice,
			Options: []string{"auto", "all"}}),
	})
	f.TypeEmptyLists()

	if got := f.Fields()[0].Kind; got != KindText {
		t.Errorf("an optionless list is still a %v", got)
	}
	if got := f.Fields()[0].Desc; !strings.Contains(got, "comma") {
		t.Errorf("nothing tells the user how to type several: %q", got)
	}
	if got := f.Fields()[1].Kind; got != KindMultiChoice {
		t.Errorf("a list with options lost its picker: %v", got)
	}
}
