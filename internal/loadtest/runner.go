// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

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

	// Continuous makes Run loop indefinitely, round-robining through every
	// assigned asset and re-scanning each one with mutated scores. ScansPerAsset
	// is ignored. The loop exits when ctx is cancelled (e.g. SIGINT).
	Continuous bool
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
	if !c.Continuous && c.ScansPerAsset <= 0 {
		return errors.New("scans-per-asset must be > 0 (or use --continuous)")
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

// assetRuntime owns the mutable per-asset state that evolves across scans.
// Workers acquire mu before mutating so two scans of the same asset never run
// concurrently — necessary because the score state and "have we synced yet"
// flag are not atomic individually and SynchronizeAssets must precede the
// first StoreResults for that asset.
type assetRuntime struct {
	mu       sync.Mutex
	idx      int
	template *Template
	state    *scoreState
	assetMrn string
	synced   bool
}

// Run executes the configured load. Per asset: SynchronizeAssets +
// ResolveAndUpdateJobs are issued once on the first scan; StoreResults is
// issued for every scan with the current mutated score state. Sharding is
// applied at asset granularity (assetIdx % totalShards == shardID), which
// keeps every asset's traffic on a single shard so per-asset state never
// crosses processes.
//
// In Continuous mode the producer round-robins through every assigned asset
// forever, so ScansPerAsset is ignored and Run exits only on context cancel.
func Run(ctx context.Context, cfg Config) (*Stats, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	stats := &Stats{}

	// Build per-asset runtimes for every asset assigned to this shard.
	runtimes := make([]*assetRuntime, 0, cfg.Assets)
	for i := 0; i < cfg.Assets; i++ {
		if i%cfg.TotalShards != cfg.ShardID {
			continue
		}
		runtimes = append(runtimes, &assetRuntime{
			idx:      i,
			template: cfg.Templates[i%len(cfg.Templates)],
			state:    newScoreState(cfg.Templates[i%len(cfg.Templates)], cfg.Seed, i),
		})
	}

	var limiter *rate.Limiter
	if cfg.ScansPerSecond > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.ScansPerSecond), int(cfg.ScansPerSecond)+1)
	}

	work := make(chan *assetRuntime, cfg.Workers)
	var wg sync.WaitGroup
	for w := 0; w < cfg.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rt := range work {
				if err := scanOnce(ctx, cfg, rt, limiter, stats); err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					log.Error().Err(err).Int("asset", rt.idx).Msg("scan failed")
				}
			}
		}()
	}

	produceErr := produce(ctx, cfg, runtimes, work)
	close(work)
	wg.Wait()

	// Count distinct assets that completed at least one scan.
	for _, rt := range runtimes {
		rt.mu.Lock()
		if rt.synced {
			atomic.AddInt64(&stats.AssetsHandled, 1)
		}
		rt.mu.Unlock()
	}

	if produceErr != nil && !errors.Is(produceErr, context.Canceled) {
		return stats, produceErr
	}
	return stats, ctx.Err()
}

// produce emits one work item per scan to be performed. Fixed mode emits
// exactly len(runtimes) * cfg.ScansPerAsset items; continuous mode loops the
// asset list forever and exits only when ctx is cancelled.
func produce(ctx context.Context, cfg Config, runtimes []*assetRuntime, work chan<- *assetRuntime) error {
	if len(runtimes) == 0 {
		return nil
	}

	send := func(rt *assetRuntime) bool {
		select {
		case <-ctx.Done():
			return false
		case work <- rt:
			return true
		}
	}

	if cfg.Continuous {
		for {
			for _, rt := range runtimes {
				if !send(rt) {
					return ctx.Err()
				}
			}
		}
	}

	for s := 0; s < cfg.ScansPerAsset; s++ {
		for _, rt := range runtimes {
			if !send(rt) {
				return ctx.Err()
			}
		}
	}
	return nil
}

// scanOnce runs a single scan iteration for the given asset: synchronize +
// resolve on the first scan, mutate scores then store on every scan. The
// per-asset mutex prevents two workers from racing on the same asset.
func scanOnce(ctx context.Context, cfg Config, rt *assetRuntime, limiter *rate.Limiter, stats *Stats) error {
	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	if !rt.synced {
		asset := SynthesizeAsset(rt.template, rt.idx, cfg.Seed)
		mrn, err := cfg.Client.SynchronizeAsset(ctx, cfg.SpaceMrn, asset)
		atomic.AddInt64(&stats.SyncCalls, 1)
		if err != nil {
			atomic.AddInt64(&stats.ErrorsSync, 1)
			return errors.Wrap(err, "synchronize")
		}
		asset.Mrn = mrn
		rt.assetMrn = mrn

		if err := cfg.Client.ResolveAndUpdateJobs(ctx, mrn, asset); err != nil {
			atomic.AddInt64(&stats.ErrorsResolve, 1)
			return errors.Wrap(err, "resolve")
		}
		atomic.AddInt64(&stats.ResolveCalls, 1)
		rt.synced = true
	} else {
		rt.state.applyChanges(cfg.ChangePct)
	}

	req := &policy.StoreResultsReq{
		AssetMrn:    rt.assetMrn,
		Scores:      rt.state.snapshot(),
		Data:        rt.template.Data,
		Risks:       rt.template.Risks,
		IsLastBatch: true,
	}
	if err := cfg.Client.StoreResults(ctx, req); err != nil {
		atomic.AddInt64(&stats.ErrorsStore, 1)
		return errors.Wrap(err, "store")
	}
	atomic.AddInt64(&stats.StoreCalls, 1)
	atomic.AddInt64(&stats.ScansSent, 1)
	return nil
}
