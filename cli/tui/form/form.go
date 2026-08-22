// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package form

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Form is a screen full of fields and the cursor moving over them.
type Form struct {
	subject string
	fields  []Field
	cursor  int
}

// New builds a form over the given fields. The subject is whatever the caller
// calls the thing being configured, and is carried only so a rebuilt form can
// recognise whether it is still the same screen; see CarryOver. This package
// never interprets it.
func New(subject string, fields []Field) Form {
	return Form{subject: subject, fields: fields}
}

// Subject is what this form is asking about.
func (f Form) Subject() string { return f.subject }

// Fields are the form's rows, in order. The slice header is a copy but the rows
// are the form's own, so a caller may edit a field's declaration through it --
// which is what an overlay does -- and still cannot reach an answer except
// through Field's methods.
func (f Form) Fields() []Field { return f.fields }

// SetFields replaces the rows wholesale, for an overlay that reorders or
// replaces what the generic layer derived.
func (f *Form) SetFields(fields []Field) { f.fields = fields }

// Cursor is the row the keyboard is on.
func (f Form) Cursor() int { return f.cursor }

// SetCursor moves the keyboard to a row.
func (f *Form) SetCursor(i int) { f.cursor = i }

// Visible reports whether a field applies given what has been chosen so far.
func (f Form) Visible(i int) bool {
	fd := f.fields[i]
	if len(fd.ShowIf) == 0 {
		return true
	}
	if fd.DependsOn < 0 || fd.DependsOn >= len(f.fields) {
		return true
	}
	return slices.Contains(fd.ShowIf, f.fields[fd.DependsOn].value)
}

// VisibleIndices are the fields that apply, in order.
func (f Form) VisibleIndices() []int {
	out := make([]int, 0, len(f.fields))
	for i := range f.fields {
		if f.Visible(i) {
			out = append(out, i)
		}
	}
	return out
}

// TypeEmptyLists turns a list field with nothing to pick from into one that can
// be typed into.
//
// A list flag becomes a multi-choice, and a multi-choice with no options is a
// row no keystroke can fill: a picker declines to open empty, and the shared
// input refuses to write itself back into a multi-choice. Six of helm's flags,
// mcp's --env and nd-ssh's --system-transport-args were all in that state, and
// the only answer available to the specs that met it was to hide the row --
// which is a reasonable thing to do about a field that cannot be filled and a
// poor thing to do about a field that matters.
//
// Comma-separated is not a convention invented here: pflag's own StringSlice
// splits its value on commas, which is how `--values a.yaml,b.yaml` already
// reaches a provider from a shell. So the typed form and the picked form
// produce the same command line, and Args already joins a multi-choice's picks
// the same way.
//
// A list that does have options -- a discovery flag always does, and an overlay
// may add choices -- keeps its picker. The two are not exclusive in principle,
// and letting a picker also take typed entries is a bigger change to the modal
// than this is; a list with options at least has a keystroke that works.
func (f *Form) TypeEmptyLists() {
	for i := range f.fields {
		fd := &f.fields[i]
		if fd.Kind != KindMultiChoice {
			continue
		}
		if len(fd.Options) > 0 || fd.source != "" || fd.LiveSource != "" ||
			fd.SourceBy != nil || fd.LiveSourceBy != nil {
			continue
		}
		fd.Kind = KindText
		fd.value = strings.Join(fd.Selected(), ",")
		fd.picks = nil
		if fd.Desc != "" && !strings.Contains(strings.ToLower(fd.Desc), "comma") {
			fd.Desc += " (comma separated)"
		}
	}
}

// visibleLabels is the set of labels the current choices leave on screen. A
// field the choices have hidden is not part of the command: a manifest path
// left over from before "live cluster" was chosen must not travel with it.
func (f Form) visibleLabels() map[string]bool {
	out := make(map[string]bool, len(f.fields))
	for _, i := range f.VisibleIndices() {
		out[f.fields[i].Label] = true
	}
	return out
}

// Positional is the form's positional arguments, in argument order.
//
// An unset one is skipped rather than emitted empty. Several connectors lead
// with an optional selector -- `terraform [plan|state] PATH` -- where leaving
// it blank has to collapse to just the path. A positional that must not be
// skipped is marked Required and caught by Validate.
func (f Form) Positional() []string {
	visible := f.visibleLabels()
	ordered := append([]Field(nil), f.fields...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Pos < ordered[j].Pos })

	var pos []string
	for _, fd := range ordered {
		// A caller-owned field is not an argument either. Special was only
		// honoured in the flag loop below, which was harmless while every such
		// field carried a flag -- but the ambient-credential widgets do not,
		// and a readout saying where a token came from would otherwise be
		// handed to the child as the thing to scan.
		if fd.Flag != "" || fd.Special != "" || !visible[fd.Label] {
			continue
		}
		if v := fd.Emitted(); v != "" {
			pos = append(pos, v)
		}
	}
	return pos
}

// FlagFields are the answered flag-carrying rows, in declaration order,
// secrets included.
//
// Secrets are included here and excluded by Args, and the difference is the
// whole point of this method existing. A secret does travel to the provider --
// it just travels in a protobuf field over the plugin's gRPC connection rather
// than in a word on a command line that `ps auxww` publishes. The two callers
// need the same answer to "which rows are part of this command" and disagree
// only about that one hop.
func (f Form) FlagFields() []Field {
	visible := f.visibleLabels()
	var out []Field
	for _, fd := range f.fields {
		if fd.Flag == "" || fd.Special != "" || !fd.IsSet() || !visible[fd.Label] {
			continue
		}
		out = append(out, fd)
	}
	return out
}

// Args emits the command line the form describes: positional arguments in
// order, then flags. Secret fields are skipped -- they travel by inventory
// file, never on the command line.
func (f Form) Args() []string {
	var flags []string
	for _, fd := range f.FlagFields() {
		if fd.Secret {
			continue
		}
		switch fd.Kind {
		case KindBool:
			flags = append(flags, "--"+fd.Flag)
		case KindMultiChoice:
			flags = append(flags, "--"+fd.Flag, strings.Join(fd.Selected(), ","))
		default:
			flags = append(flags, "--"+fd.Flag, fd.Emitted())
		}
	}
	return append(f.Positional(), flags...)
}

// Secrets returns the fields carrying a secret value, which decide how a launch
// delivers a credential and whether it can at all.
//
// Only visible fields count, the same rule Args applies. A value typed into a
// field that a later choice hid is not part of what the user is asking for, and
// counting it produced an outcome nobody could see the cause of: a unifi user
// who typed a password, switched the credential selector to "API key" and typed
// a key was told the launcher could not carry the credential on a screen
// showing exactly one. The stale value decided the route while being invisible
// -- which is also why this is a bug rather than a policy question. The form
// says what the command is; a row that is not on it is not part of the command.
func (f Form) Secrets() []Field {
	var out []Field
	for _, i := range f.VisibleIndices() {
		if fd := f.fields[i]; fd.Secret && fd.IsSet() {
			out = append(out, fd)
		}
	}
	return out
}

// ResolveSources points every dependent field at the picker its selector
// currently calls for, so choosing "local image" swaps the reference field from
// listing containers to listing images.
//
// emitFor is asked what a newly attached picker's displayed values mean on a
// command line. It is a parameter rather than a registry lookup because this
// package does not know what a picker is: it holds an opaque id and the mapping
// that came with it, and the two must never be separated. See Field.SetSource.
func (f *Form) ResolveSources(emitFor func(source string) Emit) {
	for i := range f.fields {
		fd := &f.fields[i]
		if fd.SourceBy == nil && fd.LiveSourceBy == nil && fd.DiscoverBy == nil {
			continue
		}
		key := ""
		if fd.DependsOn >= 0 && fd.DependsOn < len(f.fields) {
			key = f.fields[fd.DependsOn].value
		}
		if key == fd.resolvedKey {
			continue
		}
		fd.resolvedKey = key

		if fd.SourceBy != nil {
			// The offered values belong to the old picker; drop them rather
			// than suggest a container when an image was asked for.
			src := fd.SourceBy[key]
			fd.SetSource(src, emitFor(src))
			fd.Options = nil
			fd.optCursor = 0
		}
		if fd.LiveSourceBy != nil {
			fd.LiveSource = fd.LiveSourceBy[key]
		}
		if targets, ok := fd.DiscoverBy[key]; ok {
			f.setDiscover(targets)
		}
	}
}

// setDiscover preselects the discovery flag's targets.
func (f *Form) setDiscover(targets []string) {
	for i := range f.fields {
		fd := &f.fields[i]
		if fd.Flag != "discover" {
			continue
		}
		fd.picks = map[string]bool{}
		for _, t := range targets {
			fd.picks[t] = true
		}
	}
}

// Validate reports the first problem that would make the assembled command
// fail, so a caller can say so in place instead of tearing the screen down to
// show a usage error.
func (f Form) Validate() error {
	for _, i := range f.VisibleIndices() {
		if fd := f.fields[i]; fd.Required && !fd.IsSet() {
			return fmt.Errorf("%s is required", fd.Label)
		}
	}
	return nil
}

// Ordered returns the fields grouped into their sections, in section order,
// paired with the index each field has in the form so the cursor and the click
// zones keep referring to the same thing.
func (f Form) Ordered() (sections []string, bySection map[string][]int) {
	bySection = map[string][]int{}
	for i, fd := range f.fields {
		bySection[fd.Section] = append(bySection[fd.Section], i)
	}
	for _, s := range sectionOrder {
		if len(bySection[s]) > 0 {
			sections = append(sections, s)
		}
	}
	return sections, bySection
}

// CarryOver copies what the user entered on old onto this freshly built form.
//
// A form can be rebuilt underneath the user: a provider install lands
// asynchronously and the real metadata replaces the placeholder.
// Rebuilding is right -- the new form has fields the old one could not know
// about -- but discarding a half-typed target along the way is experienced as
// the launcher eating input.
//
// Only values the user is responsible for are carried. A prefilled value is
// the rebuild's own business: it re-derives it, possibly better, now that it
// has the provider's metadata.
func (f *Form) CarryOver(old Form) {
	if old.subject != f.subject || len(old.fields) == 0 {
		return
	}
	prev := make(map[string]Field, len(old.fields))
	for _, fd := range old.fields {
		prev[fd.Identity()] = fd
	}
	for i := range f.fields {
		o, ok := prev[f.fields[i].Identity()]
		if !ok || o.prefilled != "" {
			continue
		}
		if o.value != "" {
			f.fields[i].value = o.value
			f.fields[i].prefilled = ""
		}
		if o.on {
			f.fields[i].on = true
		}
		if len(o.picks) > 0 {
			f.fields[i].picks = o.picks
		}
	}
	if old.cursor > 0 && old.cursor < len(f.fields) {
		f.cursor = old.cursor
	}
}

// IndexOfFlag finds the field emitting a flag, or -1. A flag named "" is no
// flag: a positional field would otherwise match it.
func (f Form) IndexOfFlag(flag string) int {
	if flag == "" {
		return -1
	}
	for i := range f.fields {
		if f.fields[i].Flag == flag {
			return i
		}
	}
	return -1
}

// IndexOfSpecial finds a caller-owned field by its marker, or -1.
func (f Form) IndexOfSpecial(special string) int {
	if special == "" {
		return -1
	}
	for i := range f.fields {
		if f.fields[i].Special == special {
			return i
		}
	}
	return -1
}

// InsertField puts a field at an index, keeping every DependsOn pointing at the
// field it was pointing at before.
//
// Those indices are what make a form guided -- a selector at index 0 deciding
// which values the field after it offers -- and they are plain ints, so
// inserting anything ahead of one silently re-points it at its neighbour. Only
// a field that actually uses DependsOn is adjusted: the rest carry a zero that
// means nothing, and moving it would only make it look meaningful.
func (f *Form) InsertField(at int, fd Field) {
	if at < 0 || at > len(f.fields) {
		at = len(f.fields)
	}
	f.fields = append(f.fields, Field{})
	copy(f.fields[at+1:], f.fields[at:])
	f.fields[at] = fd

	for i := range f.fields {
		g := &f.fields[i]
		if g.ShowIf == nil && g.SourceBy == nil && g.LiveSourceBy == nil && g.DiscoverBy == nil {
			continue
		}
		if g.DependsOn >= at {
			g.DependsOn++
		}
	}
}

// SortFields orders fields by section, then by an overlay's ranking, then by
// whatever order they arrived in.
func SortFields(fields []Field, rank map[string]int) {
	sectionRank := map[string]int{}
	for i, s := range sectionOrder {
		sectionRank[s] = i
	}
	// Insertion sort keeps it stable and the slices here are tiny.
	for i := 1; i < len(fields); i++ {
		for j := i; j > 0; j-- {
			a, b := fields[j-1], fields[j]
			if fieldLess(b, a, sectionRank, rank) {
				fields[j-1], fields[j] = b, a
				continue
			}
			break
		}
	}
}

func fieldLess(a, b Field, sectionRank, rank map[string]int) bool {
	if sa, sb := sectionRank[a.Section], sectionRank[b.Section]; sa != sb {
		return sa < sb
	}
	// Positional fields always lead their section, in argument order.
	if (a.Flag == "") != (b.Flag == "") {
		return a.Flag == ""
	}
	if a.Flag == "" && b.Flag == "" {
		return a.Pos < b.Pos
	}
	ra, aok := rank[a.Flag]
	rb, bok := rank[b.Flag]
	if aok != bok {
		return aok
	}
	if aok && bok {
		return ra < rb
	}
	return false
}
