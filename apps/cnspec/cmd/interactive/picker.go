// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"go.mondoo.com/cnspec/cli/tui"
	tuiform "go.mondoo.com/cnspec/cli/tui/form"
)

// sourceLoad is everything the launcher knows about one source key: what the
// picker answered with, why it answered with nothing, whether a fetch is still
// in flight, and how to stop it.
//
// It is one record because it used to be four maps -- sourceValues, sourceErrs,
// loading and cancelLoad -- all keyed by the same string, and four maps are
// four chances to key the same fact differently. One of those chances was
// already taken: a load registered under a key scoped by the form's parameters
// and deleted under an unscoped one left the field spinning for the rest of the
// session, because the delete never matched the insert. Registration, the
// answer and the release now go through begin, beginOwned and answer below, and
// each of those takes the key exactly once.
//
// It does not make a mis-keyed load impossible -- a caller that computes the
// key differently in two places still gets two entries. What it removes is the
// second failure mode, where the four facts about one key drift apart: a
// spinner with no load behind it, an error beside values that superseded it, a
// cancel for a load that already answered.
type sourceLoad struct {
	// answered is whether a fetch has come back. It is not the same as having
	// values: a picker that looked and found nothing has answered, and asking
	// again would re-run the gcloud that found nothing.
	answered bool
	values   []string
	// err explains an empty answer, so a field can say "cluster unreachable"
	// rather than look like a cluster with nothing in it.
	err error
	// loading is whether a fetch is in flight for this key.
	loading bool
	// cancel stops what a picker started. It is nil for the loads a form-open
	// read begins: those have no picker to close and nothing worth killing.
	cancel context.CancelFunc
}

// empty reports whether an entry holds nothing worth keeping, which is what a
// cancelled load leaves behind. Such an entry is dropped rather than kept as a
// key that answers "no" to every question.
func (l sourceLoad) empty() bool { return !l.answered && !l.loading && l.cancel == nil }

// pickerState is the value pickers: the one that is open, what each of them has
// found, and how to stop the ones still looking.
//
// The map is shared with every copy of the Model, which is what lets a value
// receiver register a load -- see pendingSourceCmds, which is `func (m Model)`
// and still has to be seen by the model it was called on. Nothing else in here
// is written from a value receiver.
type pickerState struct {
	// loads is one entry per source key: a picker id and the parameters it is
	// answering for. See sourceKey.
	loads map[string]sourceLoad
	// modal is the value picker currently open, if any.
	modal modalState
}

func newPickerState() pickerState {
	return pickerState{loads: map[string]sourceLoad{}}
}

// known reports whether a source key has already been answered.
func (p pickerState) known(key string) bool { return p.loads[key].answered }

// cached returns what a picker found for a key, and whether it has answered at
// all. A picker that answered with nothing returns (nil, true), which is what
// stops the launcher asking again.
func (p pickerState) cached(key string) ([]string, bool) {
	l := p.loads[key]
	return l.values, l.answered
}

// waiting reports whether a fetch is in flight for a key.
func (p pickerState) waiting(key string) bool { return p.loads[key].loading }

// failure is why a key came back empty, if it did.
func (p pickerState) failure(key string) error { return p.loads[key].err }

// busy reports whether anything at all is being fetched, which is what decides
// whether the spinner animates: one ticking over a settled screen is noise.
func (p pickerState) busy() bool {
	for _, l := range p.loads {
		if l.loading {
			return true
		}
	}
	return false
}

// begin registers a fetch the launcher started on its own account, when a form
// opened and its cheap file-backed pickers were read.
//
// Nothing is registered on a Model built as a literal rather than through
// NewModel. The load still runs and still answers; it is simply not tracked,
// which is where the cancellation bookkeeping already stood before these four
// maps became one -- refusing the load outright would be a worse trade.
func (p pickerState) begin(key string) {
	if p.loads == nil {
		return
	}
	l := p.loads[key]
	l.loading = true
	p.loads[key] = l
}

// beginOwned registers a fetch a picker started and the cancel that stops it.
//
// A picker owns what it starts: closing it cancels this context, which kills
// the gcloud or the cnspec discover underneath rather than leaving it to finish
// for an audience that has left.
func (p pickerState) beginOwned(key string, cancel context.CancelFunc) {
	if p.loads == nil {
		return
	}
	l := p.loads[key]
	if l.cancel != nil {
		l.cancel()
	}
	l.loading, l.cancel = true, cancel
	p.loads[key] = l
}

// answer records what a fetch came back with and releases its context.
//
// Cancelling a context that has already returned is how it is released, not an
// abort. A cancelled answer is filed under nothing at all: the picker that
// asked has been closed, and caching an empty list would leave a key a later
// picker would then trust.
func (p pickerState) answer(msg sourceValuesMsg) {
	if p.loads == nil {
		return
	}
	l := p.loads[msg.key]
	if l.cancel != nil {
		l.cancel()
	}
	l.loading, l.cancel = false, nil

	if !msg.cancelled {
		l.answered, l.values = true, msg.values
		// An error is only kept when it is the whole story. A live refresh that
		// failed after a cheap read succeeded has values to show, and the list
		// is what the field says.
		l.err = nil
		if msg.err != nil && len(msg.values) == 0 {
			l.err = msg.err
		}
	}

	if l.empty() {
		delete(p.loads, msg.key)
		return
	}
	p.loads[msg.key] = l
}

// close shuts the open picker and stops whatever it started.
//
// This is the whole of the cancellation policy: a load belongs to the picker
// that asked for it, and a picker that is gone is not owed an answer. Loads
// started when the form opened carry no cancel at all, so they run on -- they
// are file reads, and there is nothing there worth killing.
func (p *pickerState) close() {
	p.modal = modalState{}
	for key, l := range p.loads {
		if l.cancel == nil {
			continue
		}
		l.cancel()
		l.loading, l.cancel = false, nil
		if l.empty() {
			delete(p.loads, key)
			continue
		}
		p.loads[key] = l
	}
}

// inFlight names the keys still being fetched. Only assertions use it: the
// launcher asks busy() whether to animate and waiting(key) about one field.
func (p pickerState) inFlight() []string {
	var out []string
	for key, l := range p.loads {
		if l.loading {
			out = append(out, key)
		}
	}
	return out
}

// cancellable names the keys whose load a picker owns and could stop.
func (p pickerState) cancellable() []string {
	var out []string
	for key, l := range p.loads {
		if l.cancel != nil {
			out = append(out, key)
		}
	}
	return out
}

// --- what a picker has to say about a field ----------------------------------
//
// This is the half of the picker cluster that could not move when pickerState
// was first pulled out. A pickerState knows only keys; which key a field asks
// under is a fact about the *form*, because a source declares what it Needs and
// the answer is the values of those fields right now. The two are joined by
// sourceKeyFor below, and everything under it takes the form as an argument
// rather than reaching through a Model to find one.

// sourceParamsFor identify what a picker is answering for, from the fields the
// source declared it needs. They must be stable: keying the cache on anything
// generated means a load and its lookup never match, and a complete answer then
// never reaches the screen.
func sourceParamsFor(f form, source string) []string {
	s, ok := sourceByID(source)
	if !ok || len(s.Needs) == 0 {
		return nil
	}
	var params []string
	for _, need := range s.Needs {
		for _, fd := range f.Fields() {
			// A need names a field by identity, so a source can depend on a
			// positional. It used to compare against fd.flag, which positional
			// fields leave empty -- so every dependency worth having was
			// inexpressible: the gcp id, the container reference and the
			// github name are all positional, and a source keyed off one
			// silently answered for the whole connector instead of for the
			// chosen target.
			if fd.IsSet() && tuiform.MatchesIdentity(fd, need) {
				params = append(params, need+"="+fd.Emitted())
			}
		}
	}
	return params
}

// sourceKeyFor is the key a source answers under for the form as it now stands.
func sourceKeyFor(f form, source string) string {
	return sourceKey(source, sourceParamsFor(f, source))
}

// fill gives every field the options its picker has already found, for the
// parameters currently in the form. Pickers that have not run leave the field
// as free text.
func (p pickerState) fill(f *form) {
	for i := range f.Fields() {
		src := f.Fields()[i].Source()
		if src == "" {
			continue
		}
		if vals, ok := p.cached(sourceKeyFor(*f, src)); ok {
			// A live refresh adds to what the cheap read already found rather
			// than replacing it, so the locally configured answer never
			// disappears because a network call was slow or refused.
			if live := f.Fields()[i].LiveSource; live != "" {
				if extra, ok := p.cached(sourceKeyFor(*f, live)); ok {
					vals = sortedUnique(append(append([]string{}, vals...), extra...))
				}
			}
			f.Fields()[i].Options = vals
		} else {
			// The parameters changed, so whatever was offered belongs to a
			// different cluster or profile and must not be shown for this one.
			f.Fields()[i].Options = nil
		}
	}
}

// pendingCmds returns the loads a form's pickers still need.
//
// Only the file-backed ones run here. A source that needs a live connection can
// take seconds and can fail, so it waits until its field is actually opened
// rather than firing for every field of every form the user passes through.
func (p pickerState) pendingCmds(f form) []tea.Cmd {
	var cmds []tea.Cmd
	seen := map[string]bool{}
	for _, fd := range f.Fields() {
		if fd.Source() == "" || deferredSource(fd.Source()) {
			continue
		}
		// The parameters are computed once and used for both halves. They used
		// to differ: the key the wait was registered under was built from the
		// form's parameters and the load was started with nil, so the key the
		// answer came back under was a different string and the delete never
		// matched. The field then span for the rest of the session. Nothing
		// shipped hit it -- the only source with Needs was CostRemote and
		// skipped here -- but the enumerated and discovery sources put
		// dependent fields on this path, so the two are derived from one value
		// now rather than written out twice.
		params := sourceParamsFor(f, fd.Source())
		key := sourceKey(fd.Source(), params)
		if seen[key] || p.known(key) || p.waiting(key) {
			continue
		}
		seen[key] = true
		p.begin(key)
		cmds = append(cmds, loadSourceCmd(context.Background(), fd.Source(), params))
	}
	return cmds
}

// openCmds loads the values for the field being opened, if they are not already
// in hand for these parameters.
//
// It returns the loads rather than a batch of them: restarting the animation
// clock is the launcher's job, not the pickers'. See Model.openPickerCmd.
func (p pickerState) openCmds(f form, fd field) []tea.Cmd {
	var cmds []tea.Cmd
	for _, src := range []string{fd.Source(), fd.LiveSource} {
		if src == "" {
			continue
		}
		params := sourceParamsFor(f, src)
		key := sourceKey(src, params)
		if p.known(key) {
			continue
		}
		// A picker owns what it starts. Closing it cancels this context, which
		// kills the gcloud or the cnspec discover underneath rather than
		// leaving it to finish for an audience that has left.
		ctx, cancel := context.WithCancel(context.Background())
		p.beginOwned(key, cancel)
		cmds = append(cmds, loadSourceCmd(ctx, src, params))
	}
	return cmds
}

// waitingFor describes what a field is still waiting on, if anything.
func (p pickerState) waitingFor(f form, fd field) string {
	for _, src := range []string{fd.Source(), fd.LiveSource} {
		if src == "" {
			continue
		}
		if p.waiting(sourceKeyFor(f, src)) {
			return activityFor(src)
		}
	}
	return ""
}

// liveError reports why a field's live refresh came back with nothing.
func (p pickerState) liveError(f form, fd field) string {
	if fd.LiveSource == "" {
		return ""
	}
	if err := p.failure(sourceKeyFor(f, fd.LiveSource)); err != nil {
		return tui.OneLine(err.Error())
	}
	return ""
}

// choiceHint says why a picker is showing nothing: still working, genuinely
// empty, or never had a source to begin with.
func (p pickerState) choiceHint(f form, fd field) string {
	if fd.Source() == "" {
		return fd.Placeholder()
	}
	if busy := p.waitingFor(f, fd); busy != "" {
		return busy + "…"
	}
	key := sourceKeyFor(f, fd.Source())
	values, loaded := p.cached(key)
	switch {
	case !loaded && deferredSource(fd.Source()):
		return "press enter to look it up"
	case !loaded:
		return activityFor(fd.Source()) + "…"
	case p.failure(key) != nil:
		// An empty list and a failed lookup look identical otherwise, and the
		// reason is usually the only actionable thing on the screen. It is
		// flattened for the same reason the footer flattens: this goes into a
		// panel line, and a newline in one makes the panel taller than the
		// number of lines the layout measured it for.
		return tui.OneLine(p.failure(key).Error())
	case len(values) == 0:
		return "none found — type a value"
	}
	return fd.Placeholder()
}
