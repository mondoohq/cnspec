// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13/policy"
	"golang.org/x/time/rate"
)

// Config drives the load test. Validate by calling Config.Validate() before
// constructing a Runner; the Runner trusts the values it receives.
type Config struct {
	SpaceMrn       string
	Templates      []*Template
	Assets         int
	ScansPerAsset  int
	ScansPerSecond float64 // 0 = unlimited
	ChangePct      float64 // 0..100
	Seed           int64
	ShardID        int
	TotalShards    int
	Workers        int
	Client         Client
}

// Validate checks for the foot-guns we can catch at startup so the user gets
// an actionable error before any work begins.
func (c *Config) Validate() error {
	if c.SpaceMrn == "" {
		return errors.New("space-mrn is required")
	}
	if c.Client == nil {
		return errors.New("client is required")
	}
	if len(c.Templates) == 0 {
		return errors.New("at least one template is required")
	}
	if c.Assets <= 0 {
		return errors.New("assets must be > 0")
	}
	if c.ScansPerAsset <= 0 {
		return errors.New("scans-per-asset must be > 0")
	}
	if c.ChangePct < 0 || c.ChangePct > 100 {
		return errors.New("change-pct must be between 0 and 100")
	}
	if c.TotalShards <= 0 {
		return errors.New("total-shards must be > 0")
	}
	if c.ShardID < 0 || c.ShardID >= c.TotalShards {
		return fmt.Errorf("shard-id %d out of range for total-shards %d", c.ShardID, c.TotalShards)
	}
	if c.Workers <= 0 {
		return errors.New("workers must be > 0")
	}
	return nil
}

// Stats reports the cumulative outcome of a Run for printing/test assertions.
type Stats struct {
	AssetsHandled int64
	ScansSent     int64
	SyncCalls     int64
	ResolveCalls  int64
	StoreCalls    int64
	ErrorsSync    int64
	ErrorsResolve int64
	ErrorsStore   int64
}

// Run executes the configured load. Per asset: SynchronizeAssets +
// ResolveAndUpdateJobs are issued once on the first scan; StoreResults is
// issued for every scan with the current mutated score state. Sharding is
// applied at asset granularity (assetIdx % totalShards == shardID), which
// keeps every asset's traffic on a single shard so per-asset state never
// crosses processes.
func Run(ctx context.Context, cfg Config) (*Stats, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	stats := &Stats{}

	// Build the deterministic asset → template assignment up front.
	type assetAssignment struct {
		idx      int
		template *Template
	}
	assignments := make([]assetAssignment, 0, cfg.Assets)
	for i := 0; i < cfg.Assets; i++ {
		if i%cfg.TotalShards != cfg.ShardID {
			continue
		}
		assignments = append(assignments, assetAssignment{
			idx:      i,
			template: cfg.Templates[i%len(cfg.Templates)],
		})
	}

	var limiter *rate.Limiter
	if cfg.ScansPerSecond > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.ScansPerSecond), int(cfg.ScansPerSecond)+1)
	}

	work := make(chan assetAssignment, cfg.Workers)
	var wg sync.WaitGroup
	for w := 0; w < cfg.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for a := range work {
				if err := runAsset(ctx, cfg, a.idx, a.template, limiter, stats); err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					log.Error().Err(err).Int("asset", a.idx).Msg("asset run failed")
				}
				atomic.AddInt64(&stats.AssetsHandled, 1)
			}
		}()
	}

	for _, a := range assignments {
		select {
		case <-ctx.Done():
			break
		case work <- a:
		}
	}
	close(work)
	wg.Wait()

	return stats, ctx.Err()
}

func runAsset(ctx context.Context, cfg Config, assetIdx int, template *Template, limiter *rate.Limiter, stats *Stats) error {
	asset := SynthesizeAsset(template, assetIdx, cfg.Seed)
	state := newScoreState(template, cfg.Seed, assetIdx)

	var assetMrn string
	for scanIdx := 0; scanIdx < cfg.ScansPerAsset; scanIdx++ {
		if limiter != nil {
			if err := limiter.Wait(ctx); err != nil {
				return err
			}
		}

		if scanIdx == 0 {
			mrn, err := cfg.Client.SynchronizeAsset(ctx, cfg.SpaceMrn, asset)
			atomic.AddInt64(&stats.SyncCalls, 1)
			if err != nil {
				atomic.AddInt64(&stats.ErrorsSync, 1)
				return errors.Wrap(err, "synchronize")
			}
			assetMrn = mrn
			asset.Mrn = mrn

			if err := cfg.Client.ResolveAndUpdateJobs(ctx, assetMrn, asset); err != nil {
				atomic.AddInt64(&stats.ErrorsResolve, 1)
				return errors.Wrap(err, "resolve")
			}
			atomic.AddInt64(&stats.ResolveCalls, 1)
		} else {
			state.applyChanges(cfg.ChangePct)
		}

		req := &policy.StoreResultsReq{
			AssetMrn:    assetMrn,
			Scores:      state.snapshot(),
			Data:        template.Data,
			Risks:       template.Risks,
			IsLastBatch: true,
		}
		if err := cfg.Client.StoreResults(ctx, req); err != nil {
			atomic.AddInt64(&stats.ErrorsStore, 1)
			return errors.Wrap(err, "store")
		}
		atomic.AddInt64(&stats.StoreCalls, 1)
		atomic.AddInt64(&stats.ScansSent, 1)
	}
	return nil
}

// startTime is exposed so tests can construct rate-limited Runners with a
// known reference; production callers use time.Now() implicitly.
func startTime() time.Time { return time.Now() }
