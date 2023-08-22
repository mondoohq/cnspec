// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"fmt"
	"sync"
)

// WaitGroup is a synchronization primitive that allows waiting
// for a collection of goroutines similar to sync.WaitGroup
// It differs in the following ways:
//   - Add takes in a workID instead of an increment. This workID is
//     passed to Done to finish it. This allows calling Done on
//     the same workID twice, making the second one a noop.
//   - There is a way to unblock all goroutines blocked on the
//     waitgroup without a normal completion. This is done through
//     Decommission
type WaitGroup struct {
	cond           *sync.Cond
	activeWorkIDs  map[string]struct{}
	seenWorkIDs    map[string]struct{}
	decommissioned bool

	numAdded int
	numDone  int
}

// NewWaitGroup returns a new WaitGroup
func NewWaitGroup() *WaitGroup {
	mutex := &sync.Mutex{}

	return &WaitGroup{
		cond:           sync.NewCond(mutex),
		activeWorkIDs:  make(map[string]struct{}),
		seenWorkIDs:    make(map[string]struct{}),
		decommissioned: false,
	}
}

// Done removes the given workID from the set of active work IDs. If the workID
// is not part of the active set, the call is a noop. Once removed, that workID
// can be reused.
// Passing a workID that was never added is an invalid operation and will cause a panic
func (w *WaitGroup) Done(workID string) {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()
	if _, ok := w.seenWorkIDs[workID]; !ok {
		// You are not allowed to complete an ID that has never been added
		panic(fmt.Sprintf("workID %q not found", workID))
	}
	if _, ok := w.activeWorkIDs[workID]; ok {
		delete(w.activeWorkIDs, workID)
		w.numDone++
	}
	if len(w.activeWorkIDs) == 0 {
		w.cond.Broadcast()
	}
}

// Add adds the workID to the set of active workIDs. Providing a workID
// that is already active is an invalid operation and will cause
// a panic. You must first Done it before reusing it.
func (w *WaitGroup) Add(workID string) {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()
	if _, ok := w.activeWorkIDs[workID]; ok {
		// You are not allowed to add the same thing to the waitgroup
		// multiple times without Doneing it
		panic(fmt.Sprintf("duplicate codeID %q", workID))
	}
	w.seenWorkIDs[workID] = struct{}{}
	w.activeWorkIDs[workID] = struct{}{}
	w.numAdded++
}

// Wait blocks the caller until there are either no more active
// workIDs in the wait group, or the wait group is decommissioned
func (w *WaitGroup) Wait() {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()

	for {
		if w.decommissioned || len(w.activeWorkIDs) == 0 {
			return
		} else {
			w.cond.Wait()
		}
	}
}

// Decommission notifies all blocked goroutines that the waitgroup
// is in a Done state, regardless of if there are still any active
// workIDs
func (w *WaitGroup) Decommission() []string {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()
	w.cond.Broadcast()
	w.decommissioned = true
	stillActivate := make([]string, len(w.activeWorkIDs))
	i := 0
	for w := range w.activeWorkIDs {
		stillActivate[i] = w
		i++
	}
	return stillActivate
}

func (w *WaitGroup) IsDecommissioned() bool {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()
	return w.decommissioned
}

type WaitGroupStats struct {
	NumAdded  int
	NumActive int
	NumDone   int
}

func (w *WaitGroup) Stats() WaitGroupStats {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()
	return WaitGroupStats{
		NumAdded:  w.numAdded,
		NumActive: len(w.activeWorkIDs),
		NumDone:   w.numDone,
	}
}
