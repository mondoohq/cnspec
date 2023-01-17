//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package backgroundjob

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

func Serve(timer time.Duration, handler JobRunner) {
	log.Info().Msg("start cnspec background service")
	log.Info().Msgf("scan interval is %d minute(s)", int(timer.Minutes()))

	quitChannel := make(chan os.Signal)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

	shutdownChannel := make(chan struct{})
	waitGroup := &sync.WaitGroup{}

	initTick := time.Tick(1 * time.Second)
	defaultTick := time.Tick(timer)
	tick := initTick
	waitGroup.Add(1)

	go func(shutdownChannel chan struct{}, wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			// Give shutdown priority
			select {
			case <-shutdownChannel:
				log.Info().Msg("stop worker")
				return
			default:
			}

			select {
			case <-tick:
				if tick == initTick {
					tick = defaultTick
				}
				err := handler()
				if err != nil {
					log.Error().Err(err).Send()
				}
			case <-shutdownChannel:
				log.Info().Msg("stop worker")
				return
			}
		}
	}(shutdownChannel, waitGroup)

	<-quitChannel // received SIGINT or SIGTERM
	close(shutdownChannel)

	log.Info().Msg("stop service gracefully")

	waitGroup.Wait() // wait for all goroutines
	log.Info().Msg("bye bye space cowboy")
}
