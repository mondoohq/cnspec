//go:build windows
// +build windows

package backgroundjob

import (
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/logger/eventlog"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

func Serve(timer time.Duration, handler JobRunner) {
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to determine if we are running in an interactive session")
	}
	// if it is an service ...
	if !isIntSess {
		// set windows eventlogger
		w, err := eventlog.NewEventlogWriter(SvcName)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to windows event log")
		}
		log.Logger = log.Output(w)

		// run service
		runService(false, timer, handler)
		return
	}
	runService(true, timer, handler)
}

type windowsService struct {
	Timer   time.Duration
	Handler JobRunner
}

// NOTE: we do not support svc.AcceptPauseAndContinue yet, we may reconsider this later
func (m *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	initTick := time.Tick(1 * time.Second)
	defaulttick := time.Tick(m.Timer)
	tick := initTick
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
		case <-tick:
			if tick == initTick {
				tick = defaulttick
			}
			select {
			case runChan <- struct{}{}:
			default:
				log.Error().Msg("scan not started. may be stuck")
			}
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

func runService(isDebug bool, timer time.Duration, handler JobRunner) {
	var err error

	log.Info().Msgf("starting %s service", SvcName)
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(SvcName, &windowsService{
		Handler: handler,
		Timer:   timer,
	})
	if err != nil {
		log.Info().Msgf("%s service failed: %v", SvcName, err)
		return
	}
	log.Info().Msgf("%s service stopped", SvcName)
}
