// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package backgroundjob

import (
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

// JobRunner runs a single background scan cycle.
type JobRunner func() error

// ServiceConfig is what the background service needs once it is running: how
// often to scan, how far to randomize that interval, the scan itself, and a
// hook to release whatever Setup acquired.
type ServiceConfig struct {
	Timer time.Duration
	Splay time.Duration
	Scan  JobRunner
	// FirstScanDelay is how long to wait before the first scan. Callers set
	// it to FleetStartDelay(); it is a field rather than a constant inside
	// runService so the schedule is deterministic under test.
	FirstScanDelay time.Duration
	// Shutdown releases what Setup started -- today the upstream check-in
	// pinger. It may be nil, and it runs once, when the service stops.
	Shutdown func()
}

// FleetStartDelay is how long a freshly started service waits before its
// first scan: a random part of a minute, so a fleet that reboots together
// does not arrive at the platform together.
func FleetStartDelay() time.Duration {
	return time.Duration(rand.Int63n(int64(time.Minute)))
}

// Setup is the slow half of startup: reading the config file, building the
// upstream client, and fingerprinting the platform. It runs after the service
// has reported itself started, never before. See runService.
type Setup func() (*ServiceConfig, error)

// runService reports the service as started, then sets it up, then scans on a
// schedule until stop is closed.
//
// started is called first and nothing slow may precede it. Windows' Service
// Control Manager gives a service 30 seconds from process start to connect
// and report a state; a service that misses the deadline is killed with error
// 1053, "A timeout was reached (30000 milliseconds) while waiting for the
// service to connect". Setup has no comparable bound. Fingerprinting the
// platform shells out to PowerShell for each hardware fact -- twice over, as
// `powershell -c "powershell.exe -NoProfile -EncodedCommand ..."` -- and ends
// with a cloud-metadata probe. A process trace from an affected machine, taken
// while it was installing Windows updates, reaches that probe 25.6 seconds in
// and is killed at 30.0 seconds, with svc.Run never reached.
//
// So the handshake goes first and the work goes second. The service has
// nothing useful to report in between: its first scan is delayed by up to a
// minute regardless, so "running" is honest the moment the schedule is owned.
//
// This order was lost once already. #1487 moved the upstream check-in off the
// startup path for exactly this reason, and #2161 put platform detection back
// on it by way of the Mondoo-PlatformID request header, which InitClient
// resolves eagerly. Keeping the order in one function that both platform
// files call, with a test on it, is what stops a third round.
func runService(started func(), setup Setup, stop <-chan struct{}) error {
	started()

	// Setup runs on its own goroutine so that a stop arriving while it is
	// still blocked -- on an upstream that accepts the connection and never
	// answers, say, which costs the HTTP client a full 30 seconds -- is acted
	// on straight away rather than after it unwinds. A service that sat on a
	// stop request that long would miss the SCM's stop deadline the same way
	// it used to miss the start one. The goroutine is abandoned in that case;
	// the process is on its way out regardless.
	type setupResult struct {
		cfg *ServiceConfig
		err error
	}
	setupDone := make(chan setupResult, 1)
	go func() {
		cfg, err := setup()
		setupDone <- setupResult{cfg: cfg, err: err}
	}()

	var cfg *ServiceConfig
	select {
	case <-stop:
		return nil
	case res := <-setupDone:
		if res.err != nil {
			return res.err
		}
		cfg = res.cfg
	}

	if cfg.Shutdown != nil {
		defer cfg.Shutdown()
	}

	log.Info().Msgf("scan interval is %d minute(s) with a splay of %d minute(s)",
		int(cfg.Timer.Minutes()), int(cfg.Splay.Minutes()))

	t := time.NewTimer(cfg.FirstScanDelay)
	defer t.Stop()

	for {
		// Give stop priority over a timer that is ready in the same tick, so
		// a service asked to stop does not start one more scan first.
		select {
		case <-stop:
			return nil
		default:
		}

		select {
		case <-stop:
			return nil
		case <-t.C:
			log.Info().Msg("starting background scan")
			if err := cfg.Scan(); err != nil {
				log.Error().Err(err).Send()
			} else {
				log.Info().Msg("scan completed")
			}

			next := cfg.Timer
			if cfg.Splay > 0 {
				next += time.Duration(rand.Int63n(int64(cfg.Splay)))
			}
			log.Info().Time("next scan", time.Now().Add(next)).Msgf("next scan in %v", next)
			t.Reset(next)
		}
	}
}
