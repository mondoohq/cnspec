// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"sort"
	"strconv"
	"strings"
	"testing"

	"go.mondoo.com/cnspec/internal/connectors"
)

// What the generated half of a form is for, and what it is not.
//
// providerFormSpec reads internal/connectors, the artifact internal/connectorgen
// derives from mql provider source. The tempting summary is "the provider now
// declares its form", and that is not what happened. A Connector arriving from
// an installed provider already carries flags, argument counts, discovery
// targets and help text, and genericFields has always read them. The artifact
// adds exactly two facts on top: the environment variables behind each flag,
// and the sub-command vocabulary.
//
// The environment routes are not returned -- see providerFormSpec -- so the
// vocabulary is the whole of the generated contribution, and the tests below
// are about what that is worth. The answer turns out to be "as a gate, more
// than as a data source": every connector whose vocabulary the artifact records
// already has a curated spec that names those words with labels and an emit
// table, so the generated copy supplies nothing new today. What it can do is
// notice when a provider grows a word that nobody curated.

// vocabularyNotOffered names provider words the form deliberately does not
// offer, keyed "<connector>/<argument index>/<word>".
//
// The gate below reports a provider word with no option behind it, because
// that is what a provider growing a sub-command looks like. It cannot tell that
// case apart from a synonym of a word already offered, so the synonyms are
// written down here with the reason rather than the check being softened into
// something that would also miss the real thing.
var vocabularyNotOffered = map[string]string{
	// gcp accepts both, and its own help text uses `org` throughout --
	// `cnspec shell gcp org <ORGANIZATION-ID>`. The form offers one row
	// labelled "organization" that emits `org`; a second row for the spelling
	// would do the same thing twice.
	"gcp/0/organization": "an alias for `org`, which the form already emits",
}

// TestGeneratedVocabularyMatchesCuration is the gate.
//
// A provider adding a sub-command word is invisible to this package: the
// curated Options list is a Go literal, applySpec drops nothing that goes
// stale, and the new word simply never appears in the picker. Nothing failed,
// nothing was logged, and the launcher quietly stopped being able to reach part
// of the connector. This is the only thing that notices.
//
// It compares membership rather than order or spelling, because the two lists
// are answering different questions. The artifact records the literal words the
// provider compares against, in source order; a curated list is what a person
// reads on a screen, in a chosen order, and may add a display-only option that
// emits nothing at all -- nmap's "network range" is one, and there is no
// provider word behind it because there is not meant to be.
//
// So a curated word absent from the provider is a finding, and a provider word
// absent from curation is a finding, and each is reported separately with the
// side it came from.
func TestGeneratedVocabularyMatchesCuration(t *testing.T) {
	checked, skipped := 0, 0
	for _, rec := range connectors.All() {
		if len(rec.Positional) == 0 {
			continue
		}
		spec, ok := formSpecs[rec.Name]
		if !ok || len(spec.Positional) == 0 {
			skipped++
			continue
		}
		for _, p := range rec.Positional {
			if p.Index >= len(spec.Positional) {
				continue
			}
			curatedField := spec.Positional[p.Index]
			if len(curatedField.Options) == 0 {
				continue
			}
			checked++

			// What a curated option contributes on the command line is its
			// Emit entry when it has one, and its own text otherwise. That is
			// the value the provider will actually see, so it is the one to
			// compare.
			emitted := map[string]string{}
			for _, opt := range curatedField.Options {
				word := opt
				if curatedField.Emit != nil {
					word = curatedField.Emit[opt]
				}
				if word == "" {
					// A display-only option, steering the screen and adding no
					// argument. There is no provider word to match it against.
					continue
				}
				emitted[word] = opt
			}

			provider := map[string]bool{}
			for _, v := range p.Values {
				provider[v] = true
			}

			var unknown, uncovered []string
			for word, opt := range emitted {
				if !provider[word] {
					unknown = append(unknown, word+" (shown as "+opt+")")
				}
			}
			for _, v := range p.Values {
				if _, ok := emitted[v]; ok {
					continue
				}
				if _, allowed := vocabularyNotOffered[rec.Name+"/"+strconv.Itoa(p.Index)+"/"+v]; allowed {
					continue
				}
				uncovered = append(uncovered, v)
			}
			sort.Strings(unknown)
			sort.Strings(uncovered)

			if len(unknown) > 0 {
				t.Errorf("%s argument %d: the form offers %s, which the provider's own source never compares against. "+
					"Either the word changed or the option is a typo; a typo here is invisible at runtime because the "+
					"provider just rejects the argument.",
					rec.Name, p.Index, strings.Join(unknown, ", "))
			}
			if len(uncovered) > 0 {
				t.Errorf("%s argument %d: the provider accepts %s, which the form does not offer. "+
					"If the provider grew a sub-command, the launcher cannot reach it; if the omission is deliberate, "+
					"say so in the spec's comment.",
					rec.Name, p.Index, strings.Join(uncovered, ", "))
			}
		}
	}
	// Counts, because this reads generated data rather than installed
	// providers: it cannot silently check nothing the way a metadata test can,
	// but a future change to the artifact could empty it, and a green run that
	// checked zero things should say so.
	t.Logf("compared %d curated vocabularies against the provider source; %d recorded vocabularies had no curated positional to compare",
		checked, skipped)
	if checked == 0 {
		t.Error("compared no vocabularies at all; the artifact or the specs stopped lining up")
	}
}

// TestGeneratedSpecReachesAnUncuratedConnector proves providerFormSpec is
// wired rather than merely present.
//
// Everything the artifact records a vocabulary for happens to be curated today,
// so the generated path contributes nothing to any shipping form -- which is
// exactly the shape of a function that is quietly dead. This drives it with a
// connector that has no spec, which is the case it exists for: the next
// connector to arrive with a sub-command shape and nobody to curate it.
func TestGeneratedSpecReachesAnUncuratedConnector(t *testing.T) {
	// github is recorded with `org|user|repo` at argument 0.
	rec, ok := connectors.ByName("github")
	if !ok || len(rec.Positional) == 0 {
		t.Fatal("the artifact no longer records a positional vocabulary for github")
	}

	c := Connector{Provider: "github", Name: "github", Use: "github", Installed: true, MaxArgs: 2}
	spec, ok := providerFormSpec(c)
	if !ok {
		t.Fatal("providerFormSpec returned nothing for a connector the artifact describes")
	}
	if len(spec.Positional) == 0 {
		t.Fatal("the generated spec declares no positional fields")
	}
	got := spec.Positional[0].Options
	if len(got) != len(rec.Positional[0].Values) {
		t.Fatalf("generated options %v, artifact records %v", got, rec.Positional[0].Values)
	}

	// And the whole way through, not only out of the constructor: a connector
	// with no registered spec must reach applySpec with these fields.
	if _, curated := formSpecs["definitely-not-a-connector"]; curated {
		t.Fatal("test connector name is taken")
	}
	c2 := c
	c2.Name = "definitely-not-a-connector"
	if _, ok := specFor(c2); ok {
		t.Fatal("specFor invented a spec for a connector the artifact does not describe")
	}
}

// TestMergeKeepsCurationWhenTheProviderIsPartial is the regression this phase
// exists to prevent.
//
// specFor used to return the provider's declaration outright whenever there was
// one. That is safe only while providers declare nothing or everything, and the
// artifact declares a sliver -- a vocabulary and no labels, sections or
// pickers. Under the old rule a connector gaining a recorded vocabulary would
// have lost its entire curated form the same day, with no test failing and no
// message anywhere.
func TestMergeKeepsCurationWhenTheProviderIsPartial(t *testing.T) {
	generated := FormSpec{
		Positional: []PositionalSpec{{Label: "argument 1", Options: []string{"org", "repo"}}},
		Secret:     []string{"provider-said-secret"},
	}
	curated := FormSpec{
		Target:     []string{"host", "port"},
		Credential: []string{"ask-pass", "password"},
		Labels:     map[string]string{"port": "SSH port"},
		Secret:     []string{"curator-said-secret"},
		NotSecret:  []string{"key-path"},
		Choices:    map[string][]string{"port": {"22"}},
	}

	got := mergeSpec(generated, curated)

	if len(got.Target) != 2 || got.Target[0] != "host" {
		t.Errorf("Target came out %v; a partial provider spec discarded the curated grouping", got.Target)
	}
	if got.Labels["port"] != "SSH port" {
		t.Errorf("Labels came out %v; the curated labels were lost", got.Labels)
	}
	if len(got.Choices["port"]) != 1 {
		t.Errorf("Choices came out %v; the curated choice list was lost", got.Choices)
	}
	// Secret unions: both sides marking a flag secret keeps more values off the
	// command line, which is the safe resolution in both directions.
	if !containsString(got.Secret, "provider-said-secret") || !containsString(got.Secret, "curator-said-secret") {
		t.Errorf("Secret came out %v; both sides' entries must survive", got.Secret)
	}
	// NotSecret does not: it is the direction that puts a value onto the
	// command line, and only a person may assert it.
	if len(got.NotSecret) != 1 || got.NotSecret[0] != "key-path" {
		t.Errorf("NotSecret came out %v; it must be the curated list exactly", got.NotSecret)
	}
}

// TestProviderNotSecretIsNeverHonoured states the safety property on its own,
// because it is the one that puts a credential in the process table if it ever
// stops holding.
func TestProviderNotSecretIsNeverHonoured(t *testing.T) {
	generated := FormSpec{NotSecret: []string{"password", "token"}}
	got := mergeSpec(generated, FormSpec{})
	if len(got.NotSecret) != 0 {
		t.Fatalf("a generated NotSecret survived the merge as %v. That moves a value onto the command line, "+
			"where `ps auxww` reads it, on the strength of a classification nobody checked.", got.NotSecret)
	}
}

// TestTheThreeCorrectionsSurvive pins the per-connector secret corrections that
// the shared name-based classifier reads the wrong way.
//
// Each is a case where the flag's name or description misleads the classifier
// and a person checked the provider instead. They are the specific thing a
// generated spec must not be allowed to overrule, so this asserts the outcome
// rather than the mechanism: whatever specFor does, these three flags end up in
// CREDENTIAL and masked.
func TestTheThreeCorrectionsSurvive(t *testing.T) {
	corrections := []struct {
		connector, flag, why string
	}{
		{"datadog", "app-key", "an application key is a credential; the classifier reads '-key' as a reference"},
		{"stackit", "service-account-key", "the key JSON itself, not a path to it"},
		{"okta", "private-key", "the description says 'or a path to it', so the path detector matched first"},
	}
	byName := snapshotByName(t)
	for _, c := range corrections {
		snap, ok := byName[c.connector]
		if !ok {
			t.Errorf("%s is no longer in the artifact, so its correction is unverified", c.connector)
			continue
		}
		f := newForm(snap.connector())
		var found bool
		for _, fd := range f.Fields() {
			if fd.Flag != c.flag {
				continue
			}
			found = true
			if !fd.Secret {
				t.Errorf("%s --%s is not classified as a secret: %s", c.connector, c.flag, c.why)
			}
			if fd.Section != sectionCredential {
				t.Errorf("%s --%s sits in %q rather than CREDENTIAL", c.connector, c.flag, fd.Section)
			}
		}
		if !found {
			t.Errorf("%s declares no --%s any more; the correction may be stale", c.connector, c.flag)
		}
	}
}
