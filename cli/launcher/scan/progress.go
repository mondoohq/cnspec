// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"sync"

	"go.mondoo.com/cnspec/cli/progress"
)

// Progress is what the launcher knows about a scan in flight.
//
// It is an aggregate rather than a log. The child emits one NDJSON event per
// whole percent per asset (cli/progress/stream.go), which on a large policy is
// still thousands of lines, and nothing on screen can use more than the latest
// state of each asset. Folding them here -- off the event loop, in the reader
// goroutine -- is what lets the display coalesce: a UI that falls behind
// redraws once with the current numbers instead of replaying a queue.
//
// It is written by exactly one goroutine and read by the renderer on the event
// loop, so every access takes the lock and the renderer works from a snapshot
// rather than from the live struct.
type Progress struct {
	mu sync.Mutex

	order   []string
	byIndex map[string]*Asset

	discovered    int
	filtered      int
	completed     int
	errored       int
	notApplicable int

	// scanDone records the summary event, which is the child's own statement
	// that it is finished -- distinct from the process exiting, which is all
	// the exit status can tell us.
	scanDone bool
}

// Asset is one asset's state, in the order the scan announced it.
type Asset struct {
	Index    string
	Name     string
	Platform string
	Num      int
	Percent  float64
	Score    string
	State    string
	Done     bool
}

// Label is what the asset is called on screen: its name, falling back to the
// platform id the scanner keys it by.
func (a Asset) Label() string {
	if a.Name != "" {
		return a.Name
	}
	return a.Index
}

// Snapshot is a consistent copy of the aggregate for one frame.
type Snapshot struct {
	Assets []Asset

	Total         int
	Done          int
	Completed     int
	Errored       int
	NotApplicable int
	Discovered    int
	Filtered      int

	ScanDone bool
}

func NewProgress() *Progress {
	return &Progress{byIndex: map[string]*Asset{}}
}

// apply folds one event into the aggregate.
//
// Unknown event names are ignored rather than rejected: the stream's schema
// says adding an event type is backwards compatible, so a launcher that
// refused one would break on the first scanner that emits more than it did.
func (p *Progress) apply(ev progress.StreamEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch ev.Event {
	case progress.EventScanStart:
		// Nothing to record: the session already knows when it started, and
		// the child's own clock is not the one the elapsed time is measured on.

	case progress.EventDiscovered:
		p.discovered = ev.Discovered

	case progress.EventFiltered:
		p.filtered = ev.Filtered

	case progress.EventAssetAdded:
		a := p.assetLocked(ev.Index, ev.Num)
		if ev.Name != "" {
			a.Name = ev.Name
		}
		if ev.Platform != "" {
			a.Platform = ev.Platform
		}

	case progress.EventAssetProgress:
		a := p.assetLocked(ev.Index, ev.Num)
		// Never backwards: events can only be reordered by a consumer, not by
		// the writer, but a bar that jumps back reads as a bug either way.
		if ev.Percent > a.Percent {
			a.Percent = ev.Percent
		}

	case progress.EventAssetScore:
		p.assetLocked(ev.Index, ev.Num).Score = ev.Score

	case progress.EventAssetDone:
		a := p.assetLocked(ev.Index, ev.Num)
		a.Done = true
		a.State = ev.State
		a.Percent = 1
		// The counters come from the event rather than being incremented here,
		// so a duplicate asset_done cannot double-count.
		p.completed, p.errored, p.notApplicable = p.tallyLocked()

	case progress.EventScanDone:
		p.scanDone = true
		p.completed = ev.Completed
		p.errored = ev.Errored
		p.notApplicable = ev.NotApplicable
		if ev.Discovered > 0 {
			p.discovered = ev.Discovered
		}
		if ev.Filtered > 0 {
			p.filtered = ev.Filtered
		}
	}
}

// assetLocked returns the tracked asset, announcing it first when the stream
// reports on one it never added. Num comes from the event because it is the
// writer's ordering, not ours.
func (p *Progress) assetLocked(index string, num int) *Asset {
	if a, ok := p.byIndex[index]; ok {
		if a.Num == 0 {
			a.Num = num
		}
		return a
	}
	a := &Asset{Index: index, Num: num}
	if a.Num == 0 {
		a.Num = len(p.order) + 1
	}
	p.byIndex[index] = a
	p.order = append(p.order, index)
	return a
}

// tallyLocked recounts the terminal states from the assets themselves.
func (p *Progress) tallyLocked() (completed, errored, notApplicable int) {
	for _, index := range p.order {
		a := p.byIndex[index]
		if !a.Done {
			continue
		}
		switch a.State {
		case progress.StateErrored:
			errored++
		case progress.StateNotApplicable:
			notApplicable++
		default:
			completed++
		}
	}
	return completed, errored, notApplicable
}

// Snapshot copies the aggregate for one frame. The renderer never touches the
// live struct, so a repaint cannot tear across an event.
func (p *Progress) Snapshot() Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := Snapshot{
		Assets:        make([]Asset, 0, len(p.order)),
		Total:         len(p.order),
		Completed:     p.completed,
		Errored:       p.errored,
		NotApplicable: p.notApplicable,
		Discovered:    p.discovered,
		Filtered:      p.filtered,
		ScanDone:      p.scanDone,
	}
	for _, index := range p.order {
		out.Assets = append(out.Assets, *p.byIndex[index])
	}
	out.Done = out.Completed + out.Errored + out.NotApplicable
	return out
}
