// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/cli/progress"
	"go.mondoo.com/mql/v13/discovery"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// TestRecoverAssetPanic_ContainsAndRecords verifies that a panic recovered from
// an asset scan is contained (does not propagate) and is recorded as a scan
// error, so one bad asset degrades to a single failed asset instead of crashing
// the whole scan.
func TestRecoverAssetPanic_ContainsAndRecords(t *testing.T) {
	reporter := NewAggregateReporter()
	d := &scanDispatcher{
		reporter:      reporter,
		multiprogress: progress.NoopMultiProgress{},
		spaceMrn:      "//spaces/abc",
		// explorer intentionally nil: the crash handler must tolerate it.
	}

	asset := &inventory.Asset{Mrn: "//assets/1", Name: "boom", PlatformIds: []string{"//p/1"}}
	tracked := &discovery.TrackedAsset{Asset: asset}

	// The panic must not escape the deferred recover — mirrors Submit's defer.
	assert.NotPanics(t, func() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					d.recoverAssetPanic(tracked, r, []byte("test stack"))
				}
			}()
			panic("provider exploded")
		}()
	})

	// The failure is recorded against the asset...
	require.Len(t, reporter.assetErrors, 1)
	err := reporter.assetErrors[asset.Mrn]
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic during scan")
	assert.Contains(t, err.Error(), "provider exploded")

	// ...and the asset reference is cleared for GC.
	assert.Nil(t, tracked.Asset)
}

// TestSubmit_PanicDoesNotCrashPool verifies end-to-end that a panic inside a
// submitted scan goroutine is contained by Submit's deferred recover, the
// worker/connection slots are released, and Wait returns normally so the rest
// of the scan can proceed. A nil tracked.Runtime makes scanSingleAsset panic on
// EnsureProvidersConnected, standing in for any provider-side panic.
func TestSubmit_PanicDoesNotCrashPool(t *testing.T) {
	reporter := NewAggregateReporter()
	connSem := make(chan struct{}, 1)
	d := &scanDispatcher{
		scanSem:       make(chan struct{}, 1),
		connSem:       connSem,
		reporter:      reporter,
		multiprogress: progress.NoopMultiProgress{},
		spaceMrn:      "//spaces/abc",
	}

	asset := &inventory.Asset{Mrn: "//assets/1", Name: "boom", PlatformIds: []string{"//p/1"}}
	tracked := &discovery.TrackedAsset{Asset: asset} // Runtime is nil → scan panics

	// Submit acquires the connSem slot (as the real walker does before submit).
	connSem <- struct{}{}

	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NotPanics(t, func() {
			d.Submit(context.Background(), tracked)
			d.Wait()
		})
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Submit/Wait did not return; panic likely escaped the worker")
	}

	// The connection slot must have been released back to the pool.
	assert.Len(t, connSem, 0, "connSem slot should be released after a panicking scan")
	// The panic should be recorded as a scan error.
	assert.Len(t, reporter.assetErrors, 1)
}
