package backgroundjob

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/upstream/health"
)

type healthPinger struct {
	ctx      context.Context
	interval time.Duration
	quit     chan struct{}
	wg       sync.WaitGroup
	endpoint string
}

func NewHealthPinger(ctx context.Context, endpoint string, interval time.Duration) *healthPinger {
	return &healthPinger{
		ctx:      ctx,
		interval: interval,
		quit:     make(chan struct{}),
		endpoint: endpoint,
	}
}

func (h *healthPinger) Start() {
	h.wg.Add(1)
	runHealthCheck := func() {
		_, err := health.CheckApiHealth(h.endpoint)
		if err != nil {
			log.Info().Err(err).Msg("could not perform health check")
		}
	}

	// run health check once on startup
	runHealthCheck()

	// TODO we may want to add jitter and backoff
	healthTicker := time.NewTicker(h.interval)
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
