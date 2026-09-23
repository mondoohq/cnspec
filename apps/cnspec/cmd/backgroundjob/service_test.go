// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package backgroundjob

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the startup ordering that Windows' Service Control
// Manager requires, without needing Windows. runService is where the order
// lives; the platform files contribute only the handshake callback (a status
// write to the SCM on Windows, a log line elsewhere), so asserting the order
// here asserts it for both.
//
// The ordering matters because the SCM kills a service that has not reported
// itself within 30 seconds of process start (error 1053), and setup is not
// bounded by anything close to that. See runService's doc comment.

// waitFor fails the test if cond does not become true within a second. It
// polls rather than sleeping a fixed interval so the tests stay fast and do
// not depend on scheduler timing.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", msg)
}

// neverScan is a scan that fails the test if it is ever called. Every test
// using it sets a FirstScanDelay far longer than the test runs, so a scan
// firing means the schedule is wrong.
func neverScan(t *testing.T) JobRunner {
	return func() error {
		t.Error("scan ran, but no test here waits long enough for one")
		return nil
	}
}

// TestRunService_ReportsStartedBeforeSetup is the regression test for
// customer issue 285: cnspec was fingerprinting the platform before it ever
// connected to the SCM, so on a machine busy installing Windows updates the
// 30-second deadline expired first and the service was killed.
//
// Setup here blocks until the test releases it, standing in for the several
// sequential PowerShell processes platform detection really costs. The
// service must already have reported itself started by then.
func TestRunService_ReportsStartedBeforeSetup(t *testing.T) {
	var startedCalled atomic.Bool
	setupEntered := make(chan struct{})
	releaseSetup := make(chan struct{})

	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- runService(
			func() { startedCalled.Store(true) },
			func() (*ServiceConfig, error) {
				close(setupEntered)
				<-releaseSetup
				return &ServiceConfig{Timer: time.Hour, FirstScanDelay: time.Hour, Scan: neverScan(t)}, nil
			},
			stop,
		)
	}()

	select {
	case <-setupEntered:
	case <-time.After(time.Second):
		t.Fatal("setup was never entered")
	}

	// The assertion: setup is running and the handshake is already done. If
	// the two are ever reordered, this is false and the service dies on a
	// slow Windows box.
	assert.True(t, startedCalled.Load(),
		"setup started before the service reported itself; a slow setup will now trip the SCM's 30s deadline")

	close(releaseSetup)
	close(stop)
	require.NoError(t, <-done)
}

// TestRunService_ReportsStartedWhenSetupFails covers the same ordering on the
// failure path. A service that cannot read its config still has to complete
// the handshake, otherwise Windows reports the generic "did not respond in a
// timely fashion" instead of the service's own error.
func TestRunService_ReportsStartedWhenSetupFails(t *testing.T) {
	var startedCalled atomic.Bool
	setupErr := errors.New("could not load configuration")

	err := runService(
		func() { startedCalled.Store(true) },
		func() (*ServiceConfig, error) { return nil, setupErr },
		make(chan struct{}),
	)

	assert.True(t, startedCalled.Load(), "a failing setup must still report the service started first")
	assert.ErrorIs(t, err, setupErr, "the setup error has to reach the caller so it can set the exit code")
}

// TestRunService_RunsShutdownHookOnStop checks that whatever setup started
// (the upstream check-in pinger) is torn down when the service stops. Before
// this refactor that was a defer in the command's RunE; now setup runs inside
// the service, so the service owns the teardown.
func TestRunService_RunsShutdownHookOnStop(t *testing.T) {
	var shutdownCalls atomic.Int32
	setupDone := make(chan struct{})

	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- runService(
			func() {},
			func() (*ServiceConfig, error) {
				defer close(setupDone)
				return &ServiceConfig{
					Timer:          time.Hour,
					FirstScanDelay: time.Hour,
					Scan:           neverScan(t),
					Shutdown:       func() { shutdownCalls.Add(1) },
				}, nil
			},
			stop,
		)
	}()

	<-setupDone
	assert.Equal(t, int32(0), shutdownCalls.Load(), "shutdown ran before the service was asked to stop")

	close(stop)
	require.NoError(t, <-done)
	assert.Equal(t, int32(1), shutdownCalls.Load(), "shutdown hook did not run exactly once on stop")
}

// TestRunService_StopsWithoutWaitingForTheFirstScan makes sure a stop arriving
// during the initial randomized delay is honored immediately. The delay is up
// to a minute, and a service that ignored stop until it elapsed would blow
// through the SCM's stop timeout on every reboot.
func TestRunService_StopsWithoutWaitingForTheFirstScan(t *testing.T) {
	stop := make(chan struct{})
	done := make(chan error, 1)
	setupDone := make(chan struct{})

	go func() {
		done <- runService(
			func() {},
			func() (*ServiceConfig, error) {
				defer close(setupDone)
				// A first scan a minute out, the real fleet-spreading
				// delay, is what this test insists the stop does not wait
				// for.
				return &ServiceConfig{Timer: time.Hour, FirstScanDelay: time.Minute, Scan: neverScan(t)}, nil
			},
			stop,
		)
	}()

	<-setupDone
	close(stop)

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("service did not stop; it is waiting out the initial scan delay")
	}
}

// TestRunService_KeepsScanningAfterAFailedScan checks that one broken scan
// cycle does not take the service down. cnspec serve is a long-running agent;
// a scan that fails because the machine was briefly offline has to be
// followed by another one.
func TestRunService_KeepsScanningAfterAFailedScan(t *testing.T) {
	var scans atomic.Int32
	stop := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- runService(
			func() {},
			func() (*ServiceConfig, error) {
				return &ServiceConfig{
					// A zero first delay and a zero Timer scan immediately
					// and reschedule immediately, which is what makes this
					// test fast. Splay stays zero so nothing is randomized.
					FirstScanDelay: 0,
					Timer:          0,
					Scan: func() error {
						scans.Add(1)
						return errors.New("could not connect to asset")
					},
				}, nil
			},
			stop,
		)
	}()

	waitFor(t, func() bool { return scans.Load() >= 2 }, "a second scan after the first one failed")

	close(stop)
	require.NoError(t, <-done)
}

// TestFleetStartDelay_StaysWithinAMinute pins the fleet-spreading policy: the
// first scan is delayed, but never by more than a minute. A delay that could
// exceed the scan interval would stall the first report indefinitely.
func TestFleetStartDelay_StaysWithinAMinute(t *testing.T) {
	for i := 0; i < 100; i++ {
		d := FleetStartDelay()
		require.GreaterOrEqual(t, d, time.Duration(0))
		require.Less(t, d, time.Minute)
	}
}

// TestRunService_StopsWhileSetupIsStillBlocked covers a stop arriving before
// setup has finished. Setup can block for a long time -- a credential exchange
// against an upstream that accepts the connection and never answers costs the
// HTTP client its full 30-second timeout -- and a service that ignored the
// stop until then would miss the SCM's stop deadline the same way it used to
// miss the start one.
func TestRunService_StopsWhileSetupIsStillBlocked(t *testing.T) {
	setupEntered := make(chan struct{})
	releaseSetup := make(chan struct{})
	defer close(releaseSetup) // let the abandoned setup goroutine finish

	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- runService(
			func() {},
			func() (*ServiceConfig, error) {
				close(setupEntered)
				<-releaseSetup
				return &ServiceConfig{Timer: time.Hour, FirstScanDelay: time.Hour, Scan: neverScan(t)}, nil
			},
			stop,
		)
	}()

	<-setupEntered
	close(stop)

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("service did not stop; it is waiting for setup to unwind")
	}
}
