// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backgroundjob

import (
	"context"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/health"
)

type healthPinger struct {
	ctx        context.Context
	interval   time.Duration
	quit       chan struct{}
	wg         sync.WaitGroup
	endpoint   string
	httpClient *http.Client
}

func NewHealthPinger(ctx context.Context, httpClient *http.Client, endpoint string, interval time.Duration) *healthPinger {
	return &healthPinger{
		ctx:        ctx,
		interval:   interval,
		quit:       make(chan struct{}),
		endpoint:   endpoint,
		httpClient: httpClient,
	}
}

func (h *healthPinger) Start() {
	h.wg.Add(1)
	runHealthCheck := func() {
		_, err := health.CheckApiHealth(h.httpClient, h.endpoint)
		if err != nil {
			log.Info().Err(err).Msg("could not perform health check")
		}
	}

	// run health check once on startup
	runHealthCheck()

	jitter := time.Duration(rand.Int63n(int64(h.interval)))
	healthTicker := time.NewTicker(h.interval + jitter)
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-healthTicker.C:
				runHealthCheck()
			case <-h.quit:
				healthTicker.Stop()
				return
			}
		}
	}()
}

func (h *healthPinger) Stop() {
	close(h.quit)
	h.wg.Wait()
}
