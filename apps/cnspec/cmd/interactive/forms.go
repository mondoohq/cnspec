// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"

	"go.mondoo.com/cnspec/cli/launcher/forms"
	tuiform "go.mondoo.com/cnspec/cli/tui/form"
	"go.mondoo.com/cnspec/internal/connectors"
)

// The curated overlays themselves now live in cli/launcher/forms, one file per
// catalog category, and this file is what the launcher does with one.
//
// The split is by what each half needs. A spec needs the FormSpec shape, the
// source ids a picker is named by, and the launcher-owned field markers -- none
// of which is a screen. Applying a spec needs the connector's declared
// metadata, the field types and the sections they sort into, all of which are
// this package's. So the data went and the engine stayed: providerFormSpec,
// specFor, mergeSpec and applySpec are all still here and all unchanged.
//
// The names below are aliases rather than a rewrite at every use, for the same
// reason catalog.go's and source.go's are: the rename would otherwise have been
// the change, across every file in this package that tests a form.

type (
	// FormSpec describes a connector's input screen. See forms.FormSpec.
	FormSpec = forms.FormSpec
	// PositionalSpec describes one field in the TARGET section.
	PositionalSpec = forms.PositionalSpec
)

const (
	specialAlicloudProfile = forms.SpecialAlicloudProfile
	specialDockerContext   = forms.SpecialDockerContext
)

// The registry itself, read once. Every registration happens in the init of a
// file in cli/launcher/forms, and an imported package is fully initialized
// before this one, so there is nothing left to arrive after this runs.
var (
	formSpecs      = forms.Specs()
	duplicateSpecs = forms.Duplicates()
)

// uncuratedReason returns the recorded reason a connector has no spec.
func uncuratedReason(connector string) (string, bool) {
	return forms.UncuratedReason(connector)
}

// providerFormSpec returns the part of a connector's form that comes from the
// provider rather than from a person.
//
// The source is internal/connectors: the artifact internal/connectorgen derives
// by walking mql provider source, embedded in the binary. It stands in for the
// provider SDK declaring a form itself, and when the SDK grows that field this
// is still the one function that changes.
//
// # What it can supply, and what it cannot
//
// Very little, and the reason is worth writing down because the obvious
// assumption is the opposite. A Connector reaching here from an installed
// provider already carries Flags, MinArgs, MaxArgs, Discovery, Long and
// Aliases; genericFields reads all of them. So the artifact only adds value
// where it knows something the runtime metadata does not, and there are exactly
// two such facts: the environment variables behind each flag, and the
// sub-command vocabulary.
//
// The environment routes are deliberately not returned. Reading a variable to
// discover that a credential is already present is source_ambient.go's job and
// is fine; putting a secret into a variable to hand it to a child process is a
// delivery mechanism cnspec is moving away from, because providers resolve the
// inventory first and the environment last. Returning Env here would grow the
// route that is being retired.
//
// That leaves the vocabulary, and note what it does not include: no label, no
// description, no ordering, no grouping, no value picker, no choice list. Those
// are the whole content of a curated FormSpec and none of them is a fact about
// the provider -- they are decisions about a screen. This function therefore
// contributes Options and nothing else, and takes each field's label from the
// same positionalLabels() the generic layer uses, so it invents no prose.
func providerFormSpec(c Connector) (FormSpec, bool) {
	rec, ok := connectors.ByName(c.Name)
	if !ok || len(rec.Positional) == 0 {
		return FormSpec{}, false
	}

	// The vocabulary is indexed the way the provider counts arguments, so a
	// gap in the indices is a real gap: aws records `ec2` at 0 and
	// `instance-connect|ssm|ebs` at 1, and nothing at 2. Slots with no recorded
	// vocabulary still have to exist, or the ones after them shift left.
	vocab := map[int][]string{}
	highest := -1
	for _, p := range rec.Positional {
		if p.Index < 0 || len(p.Values) == 0 {
			continue
		}
		vocab[p.Index] = p.Values
		if p.Index > highest {
			highest = p.Index
		}
	}
	if highest < 0 {
		return FormSpec{}, false
	}

	// One slot past the last word, for the value those words are selecting --
	// `github org <ORG>` is the selector plus the thing selected -- but never
	// past what the connector says it accepts.
	n := highest + 2
	if max := int(c.MaxArgs); max > 0 && n > max {
		n = max
	}

	labels := positionalLabels(c)
	spec := FormSpec{Positional: make([]PositionalSpec, 0, n)}
	for i := 0; i < n; i++ {
		p := PositionalSpec{
			Label:    fmt.Sprintf("argument %d", i+1),
			Required: uint(i) < c.MinArgs,
			Options:  vocab[i],
		}
		if i < len(labels) {
			p.Label = labels[i]
		}
		spec.Positional = append(spec.Positional, p)
	}
	return spec, true
}

// specFor resolves a connector's form by merging what the provider supplies
// with what a person curated on top of it.
//
// It used to return the provider's declaration outright when there was one, and
// that was a trap rather than a simplification. A provider shipping a *partial*
// form -- a vocabulary and nothing else, which is exactly what the artifact
// supplies today -- would have discarded the connector's whole curated spec:
// every label, every section, every picker, silently, with a green build. The
// dangerous state was never an unmigrated provider, which returns false and
// changes nothing; it was a half-migrated one.
//
// So it merges, and the merge rules are in mergeSpec below.
func specFor(c Connector) (FormSpec, bool) {
	generated, hasGenerated := providerFormSpec(c)
	curated, hasCurated := formSpecs[c.Name]
	switch {
	case hasGenerated && hasCurated:
		return mergeSpec(generated, curated), true
	case hasCurated:
		return curated, true
	case hasGenerated:
		return generated, true
	}
	return FormSpec{}, false
}

// mergeSpec lays a curated spec over a generated one.
//
// The direction is fixed and is the point: generated data fills what nobody has
// said anything about, and never overrules what someone has. A disagreement
// between the two is either a generator bug or a deliberate correction, and
// both of those are resolved by a person reading them -- not by whichever
// source happened to run last.
//
// Three of the fields are not symmetric, and each asymmetry is a safety
// property rather than a style choice:
//
//   - Secret is unioned rather than overridden. Marking a flag secret keeps its
//     value off the command line, so the two sources agreeing to mark more is
//     always the safe resolution.
//   - NotSecret is taken from the curated spec only. It is the one field that
//     moves a value *onto* the command line, where `ps auxww` reads it, and a
//     provider is not allowed to assert that about itself. datadog --app-key,
//     stackit --service-account-key and okta --private-key are corrections a
//     person made after checking; a generated NotSecret would be a
//     classification nobody checked.
//   - Ordered lists (Positional, Target, Credential) replace rather than
//     concatenate. Each is a complete statement about order, and appending one
//     to the other produces an order neither source asked for.
func mergeSpec(generated, curated FormSpec) FormSpec {
	out := generated

	// Positional is the one field where a curated spec's silence is an
	// assertion rather than an absence, so generated slots are dropped whenever
	// a curated spec exists at all.
	//
	// applySpec reads "no Positional, and the connector requires no argument"
	// as "the derived slots are not worth showing", and that is a decision
	// someone made. aws is the case that proves it: the artifact records `ec2`
	// at argument 0 and `instance-connect|ssm|ebs` at argument 1, so a
	// generated Positional adds two pickers and a value box above --profile and
	// --region. The words are real, but a vocabulary carries no label and no
	// emit table, so what it renders is a box titled "argument 1" offering the
	// single choice `ec2`. That is worse than the screen it replaced, and the
	// spec that suppressed those slots said so first.
	//
	// Every other field merges key by key because a curated spec that does not
	// mention a flag is simply not talking about it. Only here does not
	// mentioning something mean "and I want it gone".
	out.Positional = curated.Positional
	if len(curated.Target) > 0 {
		out.Target = curated.Target
	}
	if len(curated.Credential) > 0 {
		out.Credential = curated.Credential
	}

	out.Secret = dedupeStrings(append(append([]string{}, generated.Secret...), curated.Secret...))
	out.Hide = dedupeStrings(append(append([]string{}, generated.Hide...), curated.Hide...))
	// Curated only; see the note above.
	out.NotSecret = curated.NotSecret

	out.Labels = mergeStringMap(generated.Labels, curated.Labels)
	out.Sources = mergeStringMap(generated.Sources, curated.Sources)
	out.LiveSources = mergeStringMap(generated.LiveSources, curated.LiveSources)
	out.Env = mergeStringMap(generated.Env, curated.Env)
	out.Choices = mergeListMap(generated.Choices, curated.Choices)
	out.ShowFlagsIf = mergeListMap(generated.ShowFlagsIf, curated.ShowFlagsIf)
	return out
}

// mergeStringMap overlays over onto base, key by key, without mutating either.
func mergeStringMap(base, over map[string]string) map[string]string {
	if len(base) == 0 {
		return over
	}
	if len(over) == 0 {
		return base
	}
	out := make(map[string]string, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func mergeListMap(base, over map[string][]string) map[string][]string {
	if len(base) == 0 {
		return over
	}
	if len(over) == 0 {
		return base
	}
	out := make(map[string][]string, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

// applySpec rewrites the generic form in place. Every lookup is guarded, so a
// flag a spec names but the installed provider does not declare simply has no
// effect -- a spec that goes stale degrades to the generic screen rather than
// emitting a flag that no longer exists.
func applySpec(f *form, c Connector, o FormSpec) {
	// A spec that declares no positional fields, for a connector that does not
	// require any, is saying the derived slots are not worth showing: they
	// exist only for sub-command forms the spec chose not to model.
	if len(o.Positional) == 0 && c.MinArgs == 0 {
		var kept []field
		for _, fd := range f.Fields() {
			if fd.Flag != "" {
				kept = append(kept, fd)
			}
		}
		f.SetFields(kept)
	}

	if len(o.Positional) > 0 {
		var kept []field
		for _, fd := range f.Fields() {
			if fd.Flag != "" {
				kept = append(kept, fd)
			}
		}
		pos := make([]field, len(o.Positional))
		// prev is the last field the connector itself owns, so a launcher-owned
		// field declared in the middle of the list does not break the chain
		// behind it: `docker context` sits between `kind` and `reference`
		// because it decides what the reference can be, and `reference` still
		// takes its SourceBy from `kind`.
		prev := -1
		for i, spec := range o.Positional {
			dependsOn := prev
			if spec.Special != "" {
				// A launcher-owned field steers nothing, and what steers it is
				// the leading selector -- the same convention ShowFlagsIf uses
				// for a flag.
				dependsOn = 0
			} else {
				prev = i
			}
			d := tuiform.Decl{
				Label:        spec.Label,
				Desc:         spec.Desc,
				Required:     spec.Required,
				Options:      spec.Options,
				SourceBy:     spec.SourceBy,
				LiveSourceBy: spec.LiveSourceBy,
				ShowIf:       spec.ShowIf,
				DiscoverBy:   spec.DiscoverBy,
				Special:      spec.Special,
				DependsOn:    dependsOn,
				Pos:          i,
				Section:      sectionTarget,
				Kind:         fieldText,
			}
			fd := tuiform.NewField(d)
			attachSource(&fd, spec.Source)
			if spec.Emit != nil {
				// A declared table wins over whatever the picker says, and the
				// two never meet: a table names the options a spec wrote down,
				// and a picker's values are not knowable when the spec is.
				fd.SetEmit(emitTable(spec.Emit))
			}
			if len(spec.Options) > 0 || spec.Source != "" || spec.SourceBy != nil {
				fd.Kind = fieldChoice
				// A declared option list is the whole set of valid values; a
				// picker only suggests.
				fd.Strict = len(spec.Options) > 0 && spec.Source == "" && spec.SourceBy == nil
			}
			pos[i] = fd
		}
		f.SetFields(append(pos, kept...))
	}

	byFlag := map[string]int{}
	for i, fd := range f.Fields() {
		if fd.Flag != "" {
			byFlag[fd.Flag] = i
		}
	}
	at := func(flag string) *field {
		if i, ok := byFlag[flag]; ok {
			return &f.Fields()[i]
		}
		return nil
	}

	// The classification overrides run first, because everything below sorts
	// and sections fields by what they are, and a Target entry deliberately
	// refuses to move a secret.
	for _, flag := range o.Secret {
		if fd := at(flag); fd != nil {
			fd.Secret = true
			fd.Section = sectionCredential
		}
	}
	for _, flag := range o.NotSecret {
		if fd := at(flag); fd != nil && fd.Secret {
			fd.Secret = false
			// It was in CREDENTIAL only because it was read as a secret; put
			// it back where an ordinary flag starts and let Target or
			// Credential below place it deliberately.
			fd.Section = sectionOptions
		}
	}

	for _, flag := range o.Hide {
		if fd := at(flag); fd != nil {
			fd.Section = ""
		}
	}
	for flag, label := range o.Labels {
		if fd := at(flag); fd != nil {
			fd.Label = label
		}
	}
	for flag, src := range o.Sources {
		if fd := at(flag); fd != nil {
			attachSource(fd, src)
			fd.PromoteToChoice()
		}
	}
	for flag, src := range o.LiveSources {
		if fd := at(flag); fd != nil {
			fd.LiveSource = src
			fd.PromoteToChoice()
		}
	}
	for flag, opts := range o.Choices {
		if fd := at(flag); fd != nil {
			fd.Options = opts
			fd.PromoteToChoice()
		}
	}
	for _, flag := range o.Target {
		// A secret never leaves the credential section, whatever a spec says.
		if fd := at(flag); fd != nil && !fd.Secret {
			fd.Section = sectionTarget
		}
	}
	for _, flag := range o.Credential {
		if fd := at(flag); fd != nil {
			fd.Section = sectionCredential
		}
	}
	for flag, when := range o.ShowFlagsIf {
		if fd := at(flag); fd != nil {
			fd.ShowIf = when
			fd.DependsOn = 0 // the leading selector
		}
	}
	// Env is keyed by field identity rather than by flag, because the value
	// that has no flag to travel in is often a positional -- the whole reason
	// the environment is being used at all.
	for want, envVar := range o.Env {
		for i := range f.Fields() {
			if tuiform.MatchesIdentity(f.Fields()[i], want) {
				f.Fields()[i].Env = envVar
			}
		}
	}

	rank := map[string]int{}
	for i, flag := range append(append([]string{}, o.Target...), o.Credential...) {
		rank[flag] = i
	}
	var kept []field
	for _, fd := range f.Fields() {
		if fd.Section != "" {
			kept = append(kept, fd)
		}
	}
	f.SetFields(kept)
	tuiform.SortFields(f.Fields(), rank)
}
