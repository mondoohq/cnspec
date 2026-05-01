// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	cnspec_config "go.mondoo.com/cnspec/v13/apps/cnspec/cmd/config"
	"go.mondoo.com/cnspec/v13/internal/loadtest"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
)

func init() {
	rootCmd.AddCommand(loadtestCmd)
	loadtestCmd.Flags().String("input", "", "scan-db (.db) templates source: a local directory or a gs://bucket/prefix/ URI")
	loadtestCmd.Flags().Int("assets", 0, "number of synthetic assets to simulate")
	loadtestCmd.Flags().Int("scans-per-asset", 1, "scans to send per asset (first is the baseline)")
	loadtestCmd.Flags().Float64("scans-per-second", 0, "global scan-rate cap (0 = unlimited)")
	loadtestCmd.Flags().Float64("change-pct", 0, "percent of scores to flip pass↔fail per non-baseline scan (0..100)")
	loadtestCmd.Flags().Int64("seed", 0, "RNG seed; identical seeds reproduce identical traffic")
	loadtestCmd.Flags().Int("shard-id", 0, "this shard's index, in [0, total-shards)")
	loadtestCmd.Flags().Int("total-shards", 1, "total shards across all parallel processes")
	loadtestCmd.Flags().Int("workers", 8, "concurrent worker goroutines per process")
	loadtestCmd.Flags().String("space-mrn", "", "target space MRN (defaults to service-account scope)")
	loadtestCmd.Flags().Bool("dry-run", false, "log calls instead of sending them upstream")
	loadtestCmd.Flags().Bool("continuous", false, "scan forever (round-robin through assigned assets) until interrupted; ignores --scans-per-asset")
	loadtestCmd.Flags().Bool("ingest-only", false, "run sync+resolve once per asset upfront, then drive only UploadScanDB during the load")
	loadtestCmd.Flags().String("metrics-addr", ":2113", "address for the Prometheus /metrics endpoint (empty disables)")
	loadtestCmd.Flags().Duration("status-interval", 5*time.Second, "how often to print a progress summary to stdout (0 disables)")
	_ = loadtestCmd.MarkFlagRequired("input")
	_ = loadtestCmd.MarkFlagRequired("assets")
}

var loadtestCmd = &cobra.Command{
	Use:    "loadtest",
	Short:  "Drive synthetic scan traffic against an upstream from scan-db templates (development only)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		input, _ := cmd.Flags().GetString("input")
		assets, _ := cmd.Flags().GetInt("assets")
		scansPerAsset, _ := cmd.Flags().GetInt("scans-per-asset")
		scansPerSecond, _ := cmd.Flags().GetFloat64("scans-per-second")
		changePct, _ := cmd.Flags().GetFloat64("change-pct")
		seed, _ := cmd.Flags().GetInt64("seed")
		shardID, _ := cmd.Flags().GetInt("shard-id")
		totalShards, _ := cmd.Flags().GetInt("total-shards")
		workers, _ := cmd.Flags().GetInt("workers")
		spaceMrn, _ := cmd.Flags().GetString("space-mrn")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		continuous, _ := cmd.Flags().GetBool("continuous")
		ingestOnly, _ := cmd.Flags().GetBool("ingest-only")
		metricsAddr, _ := cmd.Flags().GetString("metrics-addr")
		statusInterval, _ := cmd.Flags().GetDuration("status-interval")

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		seedDir, cleanup, err := loadtest.MaterializeSeeds(ctx, input)
		if err != nil {
			return errors.Wrap(err, "materialize seeds")
		}
		defer cleanup()

		templates, err := loadtest.LoadTemplates(ctx, seedDir)
		if err != nil {
			return errors.Wrap(err, "load templates")
		}
		log.Info().Int("templates", len(templates)).Str("source", input).Msg("loaded scan-db templates")

		var client loadtest.Client
		if dryRun {
			client = loadtest.NewDryRunClient()
		} else {
			opts, err := cnspec_config.ReadConfig()
			if err != nil {
				return errors.Wrap(err, "read cnspec config")
			}
			creds := opts.GetServiceCredential()
			if creds == nil {
				return errors.New("no service-account credentials found; run `cnspec login` or pass --dry-run")
			}
			if spaceMrn == "" {
				spaceMrn = opts.GetParentMrn()
				if spaceMrn == "" {
					spaceMrn = creds.ScopeMrn
				}
			}
			cfg := &upstream.UpstreamConfig{
				SpaceMrn:    spaceMrn,
				ApiEndpoint: opts.UpstreamApiEndpoint(),
				ApiProxy:    opts.APIProxy,
				Creds:       creds,
			}
			client, err = loadtest.NewServicesClient(cfg, "")
			if err != nil {
				return errors.Wrap(err, "build upstream client")
			}
		}
		if spaceMrn == "" {
			return errors.New("space-mrn is required (pass --space-mrn or configure a service account with a scope)")
		}

		metrics, metricsHandler, err := loadtest.NewMetrics(ctx)
		if err != nil {
			return errors.Wrap(err, "init metrics")
		}
		defer metrics.Shutdown(context.Background())

		// Run the Prometheus endpoint in a goroutine; an error from
		// ListenAndServe (e.g. port already taken) shouldn't kill the load test
		// — log it and continue without the metrics endpoint.
		if metricsAddr != "" {
			lis, err := net.Listen("tcp", metricsAddr)
			if err != nil {
				log.Warn().Err(err).Str("addr", metricsAddr).Msg("metrics endpoint unavailable; continuing without it")
			} else {
				mux := http.NewServeMux()
				mux.Handle("/metrics", metricsHandler)
				srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
				go func() { _ = srv.Serve(lis) }()
				defer srv.Shutdown(context.Background())
				log.Info().Str("addr", lis.Addr().String()).Msg("metrics endpoint listening at /metrics")
			}
		}

		runCfg := loadtest.Config{
			SpaceMrn:       spaceMrn,
			Templates:      templates,
			Assets:         assets,
			ScansPerAsset:  scansPerAsset,
			ScansPerSecond: scansPerSecond,
			ChangePct:      changePct,
			Seed:           seed,
			ShardID:        shardID,
			TotalShards:    totalShards,
			Workers:        workers,
			Client:         client,
			Continuous:     continuous,
			IngestOnly:     ingestOnly,
			Metrics:        metrics,
		}

		// The status reporter shares the runner's Stats by virtue of being
		// constructed AFTER Run starts populating it (Stats is created inside
		// Run today — for now the reporter polls a Stats we hand it). To keep
		// the runner's existing API, we need a stats handle up front.
		stats := &loadtest.Stats{}
		runCfg.StatsOut = stats
		reporter := loadtest.NewStatusReporter(stats, statusInterval)

		statusCtx, statusCancel := context.WithCancel(ctx)
		statusDone := make(chan struct{})
		go func() {
			defer close(statusDone)
			reporter.Run(statusCtx)
		}()

		start := time.Now()
		_, runErr := loadtest.Run(ctx, runCfg)
		elapsed := time.Since(start)

		statusCancel()
		<-statusDone
		reporter.Final(start, time.Now())

		log.Info().
			Int64("assets", stats.AssetsHandled).
			Int64("scans", stats.ScansSent).
			Int64("sync_calls", stats.SyncCalls).
			Int64("resolve_calls", stats.ResolveCalls).
			Int64("upload_calls", stats.UploadCalls).
			Int64("sync_errors", stats.ErrorsSync).
			Int64("resolve_errors", stats.ErrorsResolve).
			Int64("upload_errors", stats.ErrorsUpload).
			Dur("elapsed", elapsed).
			Msg("loadtest done")
		return runErr
	},
}
