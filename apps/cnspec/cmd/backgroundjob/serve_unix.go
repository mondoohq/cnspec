// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package backgroundjob

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

// Serve runs the background scan service until the process is asked to stop,
// and returns whatever setup failed with so the caller can pick an exit code.
//
// setup runs inside the service rather than before it. That is required on
// Windows (see runService) and harmless here, so both platforms keep the same
// shape.
func Serve(setup Setup) error {
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

	stop := make(chan struct{})
	go func() {
		<-quitChannel // received SIGINT or SIGTERM
		log.Info().Msg("stop service gracefully")
		close(stop)
	}()

	err := runService(func() {
		log.Info().Msg("start cnspec background service")
	}, setup, stop)
	if err != nil {
		return err
	}

	log.Info().Msg("bye bye space cowboy")
	return nil
}
