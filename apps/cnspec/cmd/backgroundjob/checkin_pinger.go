// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backgroundjob

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type checkinPinger struct {
	ctx      context.Context
	interval time.Duration
	quit     chan struct{}
	wg       sync.WaitGroup
	handler  *CheckinHandler
}

func NewCheckinPinger(
	ctx context.Context,
	interval time.Duration,
	handler *CheckinHandler,
) *checkinPinger {
	return &checkinPinger{
		ctx:      ctx,
		interval: interval,
		quit:     make(chan struct{}),
		handler:  handler,
	}
}

func (c *checkinPinger) Start() {
	c.wg.Add(1)
	runCheckIn := func() {
		err := c.handler.CheckIn(c.ctx)
		if err != nil {
			log.Info().Err(err).Msg("could not perform check-in")
		}
	}

	// run check-in once on startup
	go func() {
		runCheckIn()
	}()

	jitter := time.Duration(rand.Int63n(int64(c.interval)))
	ticker := time.NewTicker(c.interval + jitter)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ticker.C:
				runCheckIn()
			case <-c.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *checkinPinger) Stop() {
	close(c.quit)
	c.wg.Wait()
}
