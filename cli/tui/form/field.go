// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package form is the input screen a terminal UI puts between choosing
// something to run and running it: a list of questions, the answers, and the
// command line the two produce.
//
// It knows nothing about connectors, providers or credentials sources. That is
// the point of it being a package: the function every field of every connector
// passes through on its way to argv used to open with a test for one cloud
// provider's profile picker, and there was no boundary to stop the second one
// being added beside it.
package form

import (
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Kind is what a field is answered with, which decides both the widget it gets
// and how its answer becomes a command-line word.
type Kind int

const (
	KindText        Kind = iota // free text
	KindBool                    // a toggle
	KindChoice                  // one of Options, or free text
	KindMultiChoice             // any number of Options
	KindKeyValue                // KEY=VALUE pairs, comma separated

	// KindCredentialState shows whether an ambient credential is present and
	// where it came from: an environment variable, a config file, or nothing.
	// It is not typed into -- the value it carries is a description of the
	// state, not the credential -- so it swallows keys rather than editing.
	KindCredentialState
	// KindPaste takes a secret the user pastes in, for the providers whose
	// credential is ambient or nothing: there is no list to pick from and the
	// only alternative is a shell they have already left.
	//
	// A paste field must be marked Secret or Special by whoever creates it.
	// The kind decides how it is drawn; only those two decide whether its
	// value can reach the command line, and Args reads them, not the kind.
	KindPaste
)

// Sections group fields on screen, in this order.
const (
	SectionTarget     = "TARGET"
	SectionCredential = "CREDENTIAL"
	SectionOptions    = "OPTIONS"
)

var sectionOrder = []string{SectionTarget, SectionCredential, SectionOptions}

// Emit maps a value as a field displays it to the value a command line takes.
// A value it maps to "" contributes nothing at all.
type Emit func(display string) string

// Decl is the question a field asks: what it is called, where it sits, what it
// offers, and what it contributes to the command. It is inert data the caller
// composes, and it is exported for exactly that reason -- hiding it behind
// nineteen accessors would buy nothing, because none of it is a decision this
// package makes.
//
// What the *user* did with the question is not in here. See Field.
type Decl struct {
	// Label is what the user sees.
	Label string
	// Desc is the one-line help.
	Desc string
	// Flag is the long flag name this field emits. Empty for a positional
	// argument, whose place is given by Pos instead.
	Flag string
	Pos  int

	Kind    Kind
	Options []string

	Required bool
	// Secret marks a field whose value must never reach the command line.
	Secret bool
	// Reference marks a field that names a file holding a credential rather
	// than holding one, so it becomes a path in an inventory rather than a
	// value.
	Reference bool
	// Special marks a field the caller owns rather than one the connector
	// declared, so it never reaches the command line -- neither as a flag nor
	// as a positional argument.
	Special string
	// Env names the environment variable this field's value travels in, for a
	// target that has no flag to carry it.
	Env string

	// The next three implement a selector: one field choosing what kind of
	// thing the target is, and the field after it adapting. The choice is a UI
	// distinction that changes which values are offered -- it must not put
	// extra words on the command line, which is what Emit is for.

	// SourceBy picks this field's value source from the value of the field
	// named by DependsOn.
	SourceBy map[string]string
	// LiveSourceBy picks the live source from the selector, the way SourceBy
	// picks the cheap one. Without this a live list stays attached across a
	// change of selector, and an organization id would be offered a list of
	// projects.
	LiveSourceBy map[string]string
	// DiscoverBy preselects discovery targets for a chosen value.
	DiscoverBy map[string][]string
	DependsOn  int

	Section string
	// LiveSource names a second picker, run only when the field is opened,
	// whose values are merged into the first's. It exists because the two
	// halves of "what can I scan" do not always come from the same place: a
	// cheap local read gives the obvious answer immediately, while the full
	// list needs the network and the user's patience.
	LiveSource string
	// ShowIf hides this field unless the field named by DependsOn holds one of
	// these values. This is what makes a form guided: choosing a live cluster
	// asks for a cluster, choosing a manifest asks for a path, and neither
	// screen shows the other's questions.
	ShowIf []string
	// Strict marks a choice whose Options are the only valid values, as opposed
	// to a picker that suggests what is on this machine but still accepts
	// anything typed.
	Strict bool
}

// Field is one row of a form: the question, and the answer to it.
//
// The answer is unexported and reachable only through the methods below, and
// that is the whole of what this boundary is for. Every accessor is a rule
// somebody has already got wrong by reading a member directly: Emitted is not
// Value, because a picker annotates what it found and a selector emits nothing
// at all; Display is not Value, because a secret is drawn as bullets; IsSet is
// not `!= ""`, because a toggle and a multi-choice answer differently.
//
// source and emit are unexported for the same reason and travel together, which
// SetSource is what enforces: a picker attached without its mapping puts the
// annotation on the command line, and the failure is a scan against something
// the provider has never heard of rather than anything the UI reports.
type Field struct {
	Decl

	// value holds text, choice and key/value input; on holds a toggle; picks
	// holds a multi-choice selection.
	value string
	on    bool
	picks map[string]bool

	// source names the value picker this field's options come from. It is an
	// opaque id: this package never resolves it, it only keeps it beside the
	// mapping the resolver handed back.
	source string
	emit   Emit

	// optCursor points at the option under the cursor for choice and
	// multi-choice fields.
	optCursor int
	// prefilled explains why this field filled itself in -- "current",
	// "default", "only one" -- so a value the user did not type is not a small
	// mystery.
	prefilled string
	// resolvedKey is the selector value the *By maps were last applied for, so
	// a user's own choices are not overwritten on every keystroke.
	resolvedKey string
}

// NewField builds a field from its declaration. A multi-choice starts with an
// empty selection rather than a nil one, because a nil map is not writable and
// the picker writes to it on the first keystroke.
func NewField(d Decl) Field {
	f := Field{Decl: d}
	if d.Kind == KindMultiChoice {
		f.picks = map[string]bool{}
	}
	return f
}

// Value is what the user typed or chose, as it is displayed. It is not what
// reaches the command line; see Emitted.
func (f Field) Value() string { return f.value }

// SetValue records an answer, and clears the note saying the field filled
// itself in -- once a user has typed here, "(default)" is no longer true.
func (f *Field) SetValue(v string) {
	f.value = v
	f.prefilled = ""
}

// Prefill records an answer the field derived for the user, and why.
func (f *Field) Prefill(v, why string) {
	f.value = v
	f.prefilled = why
}

// Prefilled is why this field filled itself in, or "" when the user did.
func (f Field) Prefilled() string { return f.prefilled }

// SetPrefilled overrides the note without touching the value, for a readout
// whose value and reason are derived together.
func (f *Field) SetPrefilled(why string) { f.prefilled = why }

// On is a toggle's state.
func (f Field) On() bool { return f.on }

// SetOn sets a toggle, for a declared default.
func (f *Field) SetOn(on bool) { f.on = on }

// Toggle flips a toggle.
func (f *Field) Toggle() { f.on = !f.on }

// Picked reports whether a multi-choice option is selected.
func (f Field) Picked(option string) bool { return f.picks[option] }

// TogglePick adds or removes one option from a multi-choice selection.
func (f *Field) TogglePick(option string) {
	if f.picks == nil {
		f.picks = map[string]bool{}
	}
	if f.picks[option] {
		delete(f.picks, option)
		return
	}
	f.picks[option] = true
}

// SetPicks replaces a multi-choice selection wholesale.
func (f *Field) SetPicks(picks map[string]bool) { f.picks = picks }

// OptCursor is the option under the cursor.
func (f Field) OptCursor() int { return f.optCursor }

// SetOptCursor moves the cursor over the options.
func (f *Field) SetOptCursor(i int) { f.optCursor = i }

// Source names the value picker this field's options come from, or "".
func (f Field) Source() string { return f.source }

// SetSource points a field at a value picker, and takes the picker's own answer
// to what its displayed values mean on a command line along with it.
//
// The two are one argument list because they are one decision. A picker may
// annotate what it found -- a cloud profile listed with the account id it names,
// because "which account is that?" is the question a profile name rarely
// answers -- and the annotation is not something the receiving command accepts.
// Passing nil says the picker's values are already the words the command takes.
func (f *Field) SetSource(id string, emit Emit) {
	f.source = id
	f.emit = emit
}

// SetEmit declares this field's own value mapping, for a selector whose options
// are fixed and some of which steer the screen without adding a word.
//
// A field never carries both this and a picker's mapping: a declaration cannot
// enumerate what a picker will find, so the two never describe the same field.
func (f *Field) SetEmit(e Emit) { f.emit = e }

// Emitted is what a field contributes to the command line, which is not always
// what it displays: a selector that only steers the UI emits nothing.
func (f Field) Emitted() string {
	if f.emit != nil {
		return f.emit(f.value)
	}
	return f.value
}

// IsSet reports whether this field has been answered.
func (f Field) IsSet() bool {
	switch f.Kind {
	case KindBool:
		return f.on
	case KindMultiChoice:
		return len(f.picks) > 0
	case KindCredentialState:
		// A credential-state field is set when a credential was found. The
		// value it carries describes where, which is exactly what makes an
		// empty one mean "nothing ambient here".
		return f.value != ""
	case KindPaste:
		return f.value != ""
	default:
		return f.value != ""
	}
}

// Display is the field's current value as shown on screen.
func (f Field) Display() string {
	switch f.Kind {
	case KindBool:
		if f.on {
			return "yes"
		}
		return "no"
	case KindMultiChoice:
		return strings.Join(f.Selected(), ", ")
	case KindCredentialState:
		// The value is already a sentence about where the credential came
		// from, so it is shown as it stands. Masking it would hide the one
		// thing this field exists to say.
		return f.value
	case KindPaste:
		if f.value != "" {
			return strings.Repeat("•", len(f.value))
		}
	case KindText, KindChoice, KindKeyValue:
		if f.Secret && f.value != "" {
			return strings.Repeat("•", len(f.value))
		}
	}
	return f.value
}

// Placeholder is the hint shown in an empty field: its own description, or the
// first thing a picker found, so the user can see what shape of value fits.
func (f Field) Placeholder() string {
	if f.Desc != "" {
		return f.Desc
	}
	if len(f.Options) > 0 {
		return "e.g. " + f.Options[0]
	}
	return ""
}

// Selected returns a multi-choice field's picks in the options' own order, so
// the rendered value is stable rather than map-ordered.
func (f Field) Selected() []string {
	if len(f.picks) == 0 {
		return nil
	}
	out := make([]string, 0, len(f.picks))
	for _, o := range f.Options {
		if f.picks[o] {
			out = append(out, o)
		}
	}
	// Anything picked that is not in Options (a curated addition) goes last.
	if len(out) != len(f.picks) {
		extra := make([]string, 0)
		for p := range f.picks {
			if !slices.Contains(f.Options, p) {
				extra = append(extra, p)
			}
		}
		sort.Strings(extra)
		out = append(out, extra...)
	}
	return out
}

// Identity names a field independently of its position, so the same question
// can be recognised across a rebuild of the form.
func (f Field) Identity() string {
	switch {
	case f.Special != "":
		return "s:" + f.Special
	case f.Flag != "":
		return "f:" + f.Flag
	default:
		return "p:" + strconv.Itoa(f.Pos)
	}
}

// MatchesIdentity reports whether a field is the one a declaration names.
//
// The canonical spelling is Identity -- "f:context", "p:0", "s:token" -- and a
// bare name is accepted as shorthand for a flag, because that is how a source's
// dependency has always spelled itself and there is no reason to break it. The
// shorthand is deliberately flags only: "0" meaning the first positional would
// be a name a flag could also have.
func MatchesIdentity(f Field, want string) bool {
	if want == "" {
		return false
	}
	if f.Identity() == want {
		return true
	}
	return !strings.ContainsRune(want, ':') && f.Flag == want
}

// PromoteToChoice turns a plain text field into a picker once something has
// given it values to offer.
//
// Only a plain text field is promoted, which is also what keeps the kinds a
// caller owns -- a credential-state readout, a paste box -- from being turned
// into pickers by an overlay that only meant to attach a list.
func (f *Field) PromoteToChoice() {
	if f.Kind == KindText {
		f.Kind = KindChoice
	}
}
