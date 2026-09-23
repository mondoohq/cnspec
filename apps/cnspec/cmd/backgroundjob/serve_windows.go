// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package backgroundjob

import (
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/logger/eventlog"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

// shutdownGrace is how long Execute waits for the scan loop to unwind after
// the SCM asks the service to stop, so the check-in pinger gets torn down in
// the common case. It is deliberately short: a scan already in flight runs to
// completion, and the SCM has its own stop deadline that we must not sit on.
const shutdownGrace = 5 * time.Second

// Serve runs the background scan service, either under the Service Control
// Manager or, when started from a console, in the foreground.
//
// setup is not run here. It is handed to the service and run only after the
// SCM handshake completes -- see runService for why that order is not
// negotiable on Windows.
func Serve(setup Setup) error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to determine if we are running in an interactive session")
	}
	// if it is a service ...
	if isService {
		// set windows eventlogger
		w, err := eventlog.NewEventlogWriter(SvcName)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to windows event log")
		}
		log.Logger = log.Output(w)

		return dispatch(false, setup)
	}
	return dispatch(true, setup)
}

type windowsService struct {
	Setup Setup

	// err holds a setup failure. Execute cannot return it -- the SCM only
	// takes an exit code -- so dispatch reads it back off the handler once
	// svc.Run returns.
	err error
}

// NOTE: we do not support svc.AcceptPauseAndContinue yet, we may reconsider this later
func (m *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	stop := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		// Reporting Running is the handshake the SCM is waiting on, and
		// runService performs it before it touches setup. Everything slow --
		// config, upstream client, platform fingerprinting -- happens after
		// this status write, which is what keeps a busy machine from losing
		// the service to the 30-second connect deadline.
		done <- runService(func() {
			changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
		}, m.Setup, stop)
	}()

loop:
	for {
		select {
		case err := <-done:
			// Setup failed, so there is no service left to control.
			m.err = err
			break loop
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Info().Msg("stopping cnspec service")
				close(stop)
				select {
				case m.err = <-done:
				case <-time.After(shutdownGrace):
					// A scan is still running. It does not stop cleanly
					// today, so report stopped and let the process exit take
					// it down rather than sitting on the SCM's deadline.
					log.Warn().Msg("scan still running at shutdown; stopping anyway")
				}
				break loop
			default:
				log.Error().Msgf("unexpected control request #%d", c)
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	if m.err != nil {
		log.Error().Err(m.err).Msg("cnspec service failed")
		return false, 1
	}
	return false, 0
}

func dispatch(isDebug bool, setup Setup) error {
	log.Info().Msgf("starting %s service", SvcName)
	run := svc.Run
	if isDebug {
		run = debug.Run
	}

	handler := &windowsService{Setup: setup}
	if err := run(SvcName, handler); err != nil {
		log.Info().Msgf("%s service failed: %v", SvcName, err)
		return err
	}
	if handler.err != nil {
		return handler.err
	}

	log.Info().Msgf("%s service stopped", SvcName)
	return nil
}
