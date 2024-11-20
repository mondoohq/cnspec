// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package backgroundjob

import (
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/logger/eventlog"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

func Serve(timer time.Duration, splay time.Duration, handler JobRunner) {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to determine if we are running in an interactive session")
	}
	// if it is an service ...
	if isService {
		// set windows eventlogger
		w, err := eventlog.NewEventlogWriter(SvcName)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to windows event log")
		}
		log.Logger = log.Output(w)

		// run service
		runService(false, timer, splay, handler)
		return
	}
	runService(true, timer, splay, handler)
}

type windowsService struct {
	Timer   time.Duration
	Handler JobRunner
	Splay   time.Duration
}

// NOTE: we do not support svc.AcceptPauseAndContinue yet, we may reconsider this later
func (m *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	t := time.NewTimer(time.Duration(rand.Int63n(int64(time.Minute))))
	defer t.Stop()

	log.Info().Msg("schedule background scan")
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	runChan := make(chan struct{})
	go func() {
		// This goroutine doesn't stop cleanly.
		// This isn't great, but we cannot block the service event loop.
		// It would be good to make sure this shuts down cleanly, but
		// that's not possible right now and requires wiring through
		// context throughout the execution.
		for range runChan {
			log.Info().Msg("starting background scan")
			err := m.Handler()
			if err != nil {
				log.Error().Err(err).Send()
			} else {
				log.Info().Msg("scan completed")
			}
		}
	}()
loop:
	for {
		select {
		case <-t.C:
			select {
			case runChan <- struct{}{}:
			default:
				log.Error().Msg("scan not started. may be stuck")
			}
			splayDur := time.Duration(0)
			if m.Splay > 0 {
				splayDur = time.Duration(rand.Int63n(int64(m.Splay)))
			}
			nextRun := m.Timer + splayDur
			log.Info().Time("next scan", time.Now().Add(nextRun)).Msgf("next scan in %v", nextRun)
			t.Reset(nextRun)
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Info().Msg("stopping cnspec service")
				break loop
			default:
				log.Error().Msgf("unexpected control request #%d", c)
			}
		}
	}
	close(runChan)
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(isDebug bool, timer time.Duration, splay time.Duration, handler JobRunner) {
	var err error

	log.Info().Msgf("starting %s service", SvcName)
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(SvcName, &windowsService{
		Handler: handler,
		Timer:   timer,
		Splay:   splay,
	})
	if err != nil {
		log.Info().Msgf("%s service failed: %v", SvcName, err)
		return
	}
	log.Info().Msgf("%s service stopped", SvcName)
}
