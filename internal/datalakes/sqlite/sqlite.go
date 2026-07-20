// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scandb"
	"go.mondoo.com/cnspec/v13/policy/scanstats"
	"go.mondoo.com/cnspec/v13/upload"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/health"
	"google.golang.org/protobuf/types/known/anypb"
)

// outputDirCtxKey is the unexported key used to thread the
// `cnspec scan --output-scan-db` directory through the scan ctx so the
// SQLite datalake knows where to keep the captured scan database. Set on
// the scan command's ctx via WithOutputDir; consumed inside withSqliteDataStore.
type outputDirCtxKey struct{}

// WithOutputDir returns a copy of ctx that, when threaded through the scan
// pipeline into WithServices, makes the SQLite datalake write its scan
// database to dir and keep it after upload (instead of using a temp file
// that gets deleted). Pass "" to restore the default behavior.
func WithOutputDir(ctx context.Context, dir string) context.Context {
	if dir == "" {
		return ctx
	}
	return context.WithValue(ctx, outputDirCtxKey{}, dir)
}

func outputDirFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(outputDirCtxKey{}).(string); ok {
		return v
	}
	return ""
}

func WithServices(ctx context.Context, runtime llx.Runtime, asset *inventory.Asset, upstreamClient *upstream.UpstreamClient, f func(context.Context, *policy.LocalServices) error) error {
	assetMrn := ""
	if asset != nil {
		assetMrn = asset.Mrn
	}
	err := withSqliteDataStore(ctx, assetMrn, func(scanDataStore *scandb.SqliteScanDataStore) error {
		// Persist the inventory.Asset proto so the scan database is self-contained
		// (consumed by the cnspec loadtest tool to replay against SynchronizeAssets).
		if asset != nil {
			if err := scanDataStore.WriteAsset(ctx, asset); err != nil {
				log.Warn().Err(err).Msg("failed to persist asset to scan data store")
			}
		}

		wrapper := scandb.NewScanDataStoreWrapper(scanDataStore, assetMrn)
		db, ls, err := inmemory.NewServices(runtime, inmemory.WithDataWriter(wrapper))
		if err != nil {
			return err
		}

		stats := scanstats.New()

		var upstream *policy.Services
		if upstreamClient != nil {
			var err error
			upstream, err = policy.NewRemoteServices(upstreamClient.ApiEndpoint, upstreamClient.Plugins, upstreamClient.HttpClient)
			if err != nil {
				return err
			}
			upstream.PolicyResolver = &policy.NoStoreResults{
				PolicyResolver: upstream.PolicyResolver,
			}
		}

		ls.Upstream = upstream
		scanStart := time.Now()
		if err := f(ctx, ls); err != nil {
			return err
		}
		stats.AddDuration(scanstats.MetricScanDuration, time.Since(scanStart))

		if upstream != nil {
			scanDataPath, err := scanDataStore.Finalize()
			if err != nil {
				return err
			}
			stats.AddInt(scanstats.MetricUploadSize, "bytes", fileSizeBytes(scanDataPath))
			recordKindMetrics(ctx, db, scanDataStore, assetMrn, stats)

			return uploadScanDataStore(ctx, upstream, assetMrn, scanDataPath, stats)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func withSqliteDataStore(ctx context.Context, assetMrn string, f func(scanDataStore *scandb.SqliteScanDataStore) error) error {
	// When the scan ctx carries an output dir (set via WithOutputDir from the
	// `cnspec scan --output-scan-db` flag), write the scan db there and keep
	// it after upload. Otherwise create a temp file we delete.
	dir := outputDirFromCtx(ctx)
	keep := dir != ""
	if keep {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Error().Err(err).Str("dir", dir).Msg("failed to create scan-db output directory")
			return err
		}
	}

	tmpFile, err := os.CreateTemp(dir, "cnspec-scan-*.db")
	if err != nil {
		log.Error().Err(err).Msg("failed to create file for scan data store")
		return err
	}
	tmpFile.Close() // nolint: errcheck
	debugMem := os.Getenv("DEBUG_PROVIDER_MEMORY") != ""
	if debugMem {
		log.Info().Str("path", tmpFile.Name()).Bool("keep", keep).Msg("created scan database")
	}
	defer func() {
		if keep {
			log.Info().Str("path", tmpFile.Name()).Msg("scan database saved")
			return
		}
		if debugMem {
			if info, err := os.Stat(tmpFile.Name()); err == nil {
				log.Info().Str("path", tmpFile.Name()).Int64("size_bytes", info.Size()).Msg("removing scan database")
			}
		}
		if err := os.Remove(tmpFile.Name()); err != nil {
			log.Warn().Err(err).Msg("failed to remove temporary scan data store file")
		}
	}()

	scanDataStore, err := scandb.NewSqliteScanDataStore(tmpFile.Name(), assetMrn)
	if err != nil {
		log.Error().Err(err).Msg("failed to create scan data store")
		return err
	}
	defer scanDataStore.Close()

	return f(scanDataStore)
}

// fileSizeBytes returns the size of the file at path, or 0 if it cannot be stat'd.
func fileSizeBytes(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// recordKindMetrics derives per-kind scan counts from the resolved policy and
// the finalized scan store, and records them on the collector. It is
// best-effort: any failure logs a warning and skips the affected metrics so the
// upload confirmation is never blocked.
func recordKindMetrics(ctx context.Context, db *inmemory.Db, store *scandb.SqliteScanDataStore, assetMrn string, stats *scanstats.Collector) {
	rp, err := db.GetResolvedPolicy(ctx, assetMrn)
	if err != nil {
		log.Warn().Err(err).Str("asset", assetMrn).Msg("no resolved policy for scan statistics; skipping per-kind metrics")
		return
	}

	var scores []*policy.Score
	if err := store.StreamScores(ctx, func(s *policy.Score) error {
		scores = append(scores, s)
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("failed to read scores for scan statistics")
	}

	var data []*llx.Result
	if err := store.StreamData(ctx, func(_ string, r *llx.Result) error {
		data = append(data, r)
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("failed to read data results for scan statistics")
	}

	counts := scanstats.CountByKind(rp, scores, data)
	stats.AddInt(scanstats.MetricChecks, "count", counts.Checks)
	stats.AddInt(scanstats.MetricDataQueries, "count", counts.DataQueries)
	stats.AddInt(scanstats.MetricPolicies, "count", counts.Policies)
	stats.AddInt(scanstats.MetricControls, "count", counts.Controls)
	stats.AddInt(scanstats.MetricFrameworks, "count", counts.Frameworks)
	stats.AddInt(scanstats.MetricChecksErrored, "count", counts.ChecksErrored)
	stats.AddInt(scanstats.MetricDataQueriesErrored, "count", counts.DataQueriesErrored)
}

func uploadScanDataStore(ctx context.Context, services *policy.Services, assetMrn string, scanDataPath string, stats *scanstats.Collector) error {
	urlResp, err := services.GetUploadURL(ctx, &policy.GetUploadURLReq{
		Kind:     policy.UploadURLKind_UPLOAD_URL_KIND_SCAN_DATABASE_V0,
		ScopeMrn: assetMrn,
	})
	if err != nil {
		return err
	}

	uploadUrl := urlResp.UploadUrl
	if uploadUrl == nil {
		return errors.New("no upload URL for scan data store")
	}

	headers := uploadUrl.Headers
	url := uploadUrl.Url

	resp, err := upload.UploadFile(ctx, url, headers, scanDataPath, "application/octet-stream")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Read a limited amount of the response body for diagnostics.
		// Truncate to 512 bytes to avoid leaking sensitive details in Sentry tags.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		health.ReportError("cnspec", cnspec.Version, cnspec.Build,
			fmt.Sprintf("upload failed with status %d", resp.StatusCode),
			health.WithTags(map[string]string{
				"assetMrn":     assetMrn,
				"responseBody": string(body),
			}),
		)
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	// Confirm the upload, attaching scan statistics as the completion payload.
	req := &policy.ReportUploadCompletedReq{
		UploadSessionId: urlResp.UploadSessionId,
		ScopeMrn:        assetMrn,
	}
	if s := stats.ToProto(); s != nil {
		if details, aerr := anypb.New(s); aerr != nil {
			log.Warn().Err(aerr).Msg("failed to encode scan statistics; sending upload confirmation without them")
		} else {
			req.Details = details
		}
	}
	if _, err = services.ReportUploadCompleted(ctx, req); err != nil {
		return err
	}

	log.Info().Str("session", urlResp.UploadSessionId).Msg("successfully uploaded scan data store")
	return nil
}
