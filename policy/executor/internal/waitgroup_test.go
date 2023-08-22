// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package internal

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWaitGroupInvalidUsage(t *testing.T) {
	t.Run("calling Add with an active workID panics", func(t *testing.T) {
		wg := NewWaitGroup()
		wg.Add("foo")
		require.Panics(t, func() { wg.Add("foo") })
	})

	t.Run("calling Done with an id that was never added panics", func(t *testing.T) {
		wg := NewWaitGroup()
		require.Panics(t, func() { wg.Done("foo") })
	})
}

func TestWaitGroup(t *testing.T) {
	t.Run("finishing completed workIDs unblocks Wait", func(t *testing.T) {
		signalGoroutineStarted := &sync.WaitGroup{}
		signalFinished := &sync.WaitGroup{}
		wg := NewWaitGroup()
		wg.Add("foo")

		signalFinished.Add(1)
		signalGoroutineStarted.Add(1)
		go func() {
			signalGoroutineStarted.Done()
			wg.Wait()
			signalFinished.Done()
		}()

		wg.Done("foo")
		signalFinished.Wait()

		stats := wg.Stats()
		require.Equal(t, WaitGroupStats{
			NumAdded:  1,
			NumActive: 0,
			NumDone:   1,
		}, stats)

		require.Equal(t, wg.IsDecommissioned(), false)
	})

	t.Run("decommissioning unblocks Wait", func(t *testing.T) {
		signalGoroutineStarted := &sync.WaitGroup{}
		signalFinished := &sync.WaitGroup{}
		wg := NewWaitGroup()
		wg.Add("foo")
		wg.Add("bar")

		signalFinished.Add(1)
		signalGoroutineStarted.Add(1)
		go func() {
			signalGoroutineStarted.Done()
			wg.Wait()
			signalFinished.Done()
		}()

		signalGoroutineStarted.Wait()
		wg.Decommission()
		signalFinished.Wait()

		stats := wg.Stats()
		require.Equal(t, WaitGroupStats{
			NumAdded:  2,
			NumActive: 2,
			NumDone:   0,
		}, stats)

		require.Equal(t, wg.IsDecommissioned(), true)
	})

	t.Run("usable after decommission", func(t *testing.T) {
		// We want to make sure you can still done things even
		// after decommissioning. The wait group should still
		// never block on Wait after being decommissioned.

		wg := NewWaitGroup()
		wg.Add("foo")
		wg.Add("bar")

		wg.Decommission()
		wg.Wait() // Does not block

		wg.Done("foo")
		wg.Add("baz")
		wg.Wait()

		stats := wg.Stats()
		require.Equal(t, WaitGroupStats{
			NumAdded:  3,
			NumActive: 2,
			NumDone:   1,
		}, stats)

		require.Equal(t, wg.IsDecommissioned(), true)
	})
}
