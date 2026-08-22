// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"fmt"
	"strconv"
	"strings"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// A form is what stands between picking a connector and running it: the
// positional target, the connector's flags, and the credential. It is built in
// two layers. The generic layer below derives every field from the connector's
// own declared metadata, so a connector nobody has curated still gets a usable
// screen. The curated layer -- forms.go here, applying the specs registered in
// cli/launcher/forms -- then reorders, relabels and attaches value pickers for
// the connectors people actually reach for.
//
// The engine underneath both -- what a field is, what a form emits, which rows
// apply -- is cli/tui/form, and it knows nothing about connectors. This file is
// the whole of the translation: everything here reads provider metadata, and
// nothing in the engine does.

// The launcher's names for the engine's types.
//
// They are aliases rather than a rewrite at every use because the rename would
// then have been the change, across eight hundred uses in this package. What
// the boundary is for survives an alias untouched: a field's answer is
// unexported in cli/tui/form, so nothing here can put a value on a command line
// without going through Emitted, and nothing here can attach a value picker
// without the mapping that says what its values mean.
type (
	field = tuiform.Field
	form  = tuiform.Form
)

const (
	fieldText            = tuiform.KindText
	fieldBool            = tuiform.KindBool
	fieldChoice          = tuiform.KindChoice
	fieldMultiChoice     = tuiform.KindMultiChoice
	fieldKeyValue        = tuiform.KindKeyValue
	fieldCredentialState = tuiform.KindCredentialState
	fieldPaste           = tuiform.KindPaste
)

const (
	sectionTarget     = tuiform.SectionTarget
	sectionCredential = tuiform.SectionCredential
	sectionOptions    = tuiform.SectionOptions
)

// newForm builds the input form for a connector: the generic layer from its
// declared metadata, then the curated overlay if one exists.
func newForm(c Connector) form {
	f := tuiform.New(c.Name, genericFields(c))
	if spec, ok := specFor(c); ok {
		applySpec(&f, c, spec)
	}
	// After the overlay, so a curated label and section for the token flag
	// survive and only the widget drawn over it changes; before resolveSources,
	// which is what fills the readouts in. See source_ambient.go.
	applyAmbient(&f, c)
	// After the overlay too, and for the same reason: whether a list has
	// anything to pick from is not known until a spec has had its say.
	f.TypeEmptyLists()
	resolveSources(&f)
	return f
}

// resolveSources re-points every dependent field at the picker its selector
// currently calls for, and re-reads the ambient credentials.
//
// The two are one operation, which is why this exists rather than a bare call
// into the engine. A readout is honest only because this runs when a form is
// built and after every keystroke: pasting a token flips the row from naming a
// variable to saying so, and clearing it flips the row back.
func resolveSources(f *form) {
	f.ResolveSources(sourceEmitFor)
	refreshAmbient(f)
}

// carryOver copies what the user entered on old onto a freshly built form, and
// re-derives the credential readouts on top of it.
//
// The readouts on the rebuilt form were derived before the carried values
// arrived. Without the second half a row would name an environment variable
// while a pasted token sat in the box below it.
func carryOver(f *form, old form) {
	f.CarryOver(old)
	refreshAmbient(f)
}

// genericFields derives fields straight from the connector's declaration.
func genericFields(c Connector) []field {
	var out []field

	// Show every required positional, plus at most one optional slot. A high
	// MaxArgs usually encodes sub-command forms rather than that many
	// independent arguments -- aws declares MaxArgs=4 for `aws ec2 ebs
	// snapshot <id>` -- and rendering one box per slot buries the fields that
	// matter. Connectors whose sub-command shape is worth modelling get a
	// curated overlay that replaces these outright.
	labels := positionalLabels(c)
	if n := int(c.MinArgs) + 1; len(labels) > n {
		labels = labels[:n]
	}
	for i, label := range labels {
		out = append(out, tuiform.NewField(tuiform.Decl{
			Label:    label,
			Flag:     "",
			Pos:      i,
			Kind:     fieldText,
			Required: uint(i) < c.MinArgs,
			Section:  sectionTarget,
		}))
	}

	for _, fl := range c.Flags {
		if fl.Option&plugin.FlagOption_Hidden != 0 || fl.Option&plugin.FlagOption_Deprecated != 0 {
			continue
		}
		if isInertPromptFlag(fl) {
			continue
		}
		out = append(out, fieldForFlag(fl))
	}

	// --discover is synthesized for every connector by the CLI, so it is never
	// in Flags; "all" and "auto" are always valid on top of what is declared.
	if len(c.Discovery) > 0 {
		opts := append([]string{"auto", "all"}, c.Discovery...)
		out = append(out, tuiform.NewField(tuiform.Decl{
			Label:   "discover",
			Desc:    "Discover nested assets",
			Flag:    "discover",
			Kind:    fieldMultiChoice,
			Options: dedupeStrings(opts),
			Section: sectionOptions,
		}))
	}

	return out
}

// builtinAskPass is the one prompt flag mql honours by name. Everything else
// has to declare itself; see isInertPromptFlag.
const builtinAskPass = "ask-pass"

// isInertPromptFlag reports whether a bool flag is named like a prompt that
// nothing will act on.
//
// mql prompts for --ask-pass, which it reads by name, and for any flag carrying
// FlagOption_AskInput, which it collects into an "ask-flags" annotation.
// clickhousecloud's --ask-secret and weaviate's --ask-api-key are declared as
// plain bools with neither property, so the CLI parses them and no one reads
// them. Verified against the shipped binary: `cnspec shell weaviate --host
// 127.0.0.1 --ask-api-key </dev/null` never asks anything and goes straight to
// a 404 from an unauthenticated request, where the same shape on ssh --ask-pass
// fails with "asking input is only supported when used with an interactive
// terminal", which is a prompt refusing a pipe.
//
// A toggle labelled "prompt for API key" that leads to an unauthenticated scan
// is worse than an absent row, so the row is absent. The flag it names is still
// on the form and still classified as a secret; what it loses is a partner that
// was never going to arrive.
func isInertPromptFlag(fl plugin.Flag) bool {
	return fl.Type == plugin.FlagType_Bool &&
		strings.HasPrefix(fl.Long, "ask-") &&
		fl.Long != builtinAskPass &&
		fl.Option&plugin.FlagOption_AskInput == 0
}

func fieldForFlag(fl plugin.Flag) field {
	d := tuiform.Decl{
		Label:     fl.Long,
		Desc:      fl.Desc,
		Flag:      fl.Long,
		Required:  fl.Option&plugin.FlagOption_Required != 0,
		Secret:    tuiform.IsSecretFlag(fl),
		Reference: tuiform.IsSecretReference(fl),
		Section:   sectionOptions,
	}
	if d.Secret {
		d.Section = sectionCredential
	}

	switch fl.Type {
	case plugin.FlagType_Bool:
		d.Kind = fieldBool
	case plugin.FlagType_List:
		d.Kind = fieldMultiChoice
	case plugin.FlagType_KeyValue:
		d.Kind = fieldKeyValue
	default: // String, Int and anything unrecognized are typed as text
		d.Kind = fieldText
	}

	f := tuiform.NewField(d)
	if fl.Type == plugin.FlagType_Bool {
		// A declared default is the connector's, not the user's, so it is set
		// rather than prefilled: nothing on the row claims the launcher chose
		// it.
		on, _ := strconv.ParseBool(fl.Default)
		f.SetOn(on)
	}
	return f
}

// positionalLabels names the connector's positional arguments. The usage string
// is the only description of them, and it is not machine-readable, so this
// falls back to numbered labels whenever it cannot line the tokens up with the
// declared argument count.
func positionalLabels(c Connector) []string {
	n := int(c.MaxArgs)
	if n < int(c.MinArgs) {
		n = int(c.MinArgs)
	}
	if n == 0 {
		return nil
	}

	hint := c.ArgHint()
	tokens := strings.Fields(hint)
	switch {
	case len(tokens) == n:
		return tokens
	case n == 1 && hint != "":
		return []string{hint}
	}

	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("argument %d", i+1)
	}
	return out
}

// attachSource points a field at a value picker, and at what that picker says
// its displayed values mean on a command line. See Source.Emit: the two are one
// decision, and a picker attached without its mapping emits the annotation it
// put there for the reader.
func attachSource(fd *field, id string) {
	fd.SetSource(id, sourceEmitFor(id))
}

// sourceEmitFor is what the engine asks when it attaches a picker to a field.
func sourceEmitFor(id string) tuiform.Emit {
	if e := sourceEmit(id); e != nil {
		return tuiform.Emit(e)
	}
	return nil
}

// emitTable is a spec's declared value mapping as a field carries it. A value
// with no entry emits nothing, which is how a selector steers the screen
// without adding a word cnspec would not understand.
func emitTable(table map[string]string) tuiform.Emit {
	return func(display string) string { return table[display] }
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
