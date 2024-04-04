// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package backgroundjob

import (
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

func Serve(timer time.Duration, splay time.Duration, handler JobRunner) {
	log.Info().Msg("start cnspec background service")
	log.Info().Msgf("scan interval is %d minute(s) with a splay of %d minutes(s)", int(timer.Minutes()), int(splay.Minutes()))

	quitChannel := make(chan os.Signal)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

	shutdownChannel := make(chan struct{})
	waitGroup := &sync.WaitGroup{}

	t := time.NewTimer(time.Duration(rand.Int63n(int64(time.Minute))))
	defer t.Stop()

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
			case <-t.C:
				log.Info().Msg("starting background scan")
				err := handler()
				if err != nil {
					log.Error().Err(err).Send()
				}
				splayDur := time.Duration(0)
				if splay > 0 {
					splayDur = time.Duration(rand.Int63n(int64(splay)))
				}
				nextRun := timer + splayDur
				log.Info().Time("next scan", time.Now().Add(nextRun)).Msgf("next scan in %v", nextRun)
				t.Reset(nextRun)
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
