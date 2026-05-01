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

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scandb"
	"go.mondoo.com/cnspec/v13/upload"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/health"
)

// outputDir, when set via SetOutputDir, redirects per-asset scan databases
// to a user-specified directory and keeps them there instead of deleting.
// Dev-only feature wired to the `cnspec scan --output-scan-db` flag; used
// to capture realistic seeds for the cnspec loadtest tool.
var outputDir string

// SetOutputDir configures the destination directory for scan databases.
// Pass "" to restore the default temp-and-delete behavior. Not safe for
// concurrent calls — call once at startup before any scans run.
func SetOutputDir(dir string) { outputDir = dir }

func WithServices(ctx context.Context, runtime llx.Runtime, asset *inventory.Asset, upstreamClient *upstream.UpstreamClient, f func(context.Context, *policy.LocalServices) error) error {
	assetMrn := ""
	if asset != nil {
		assetMrn = asset.Mrn
	}
	err := withSqliteDataStore(assetMrn, func(scanDataStore *scandb.SqliteScanDataStore) error {
		// Persist the inventory.Asset proto so the scan database is self-contained
		// (consumed by the cnspec loadtest tool to replay against SynchronizeAssets).
		if asset != nil {
			if err := scanDataStore.WriteAsset(ctx, asset); err != nil {
				log.Warn().Err(err).Msg("failed to persist asset to scan data store")
			}
		}

		// When --output-scan-db is set, install the filter-capture hook so
		// the scanner's ResolveAndUpdateJobs filters land in the scan db too.
		// We deliberately gate this on outputDir so regular scans don't pay
		// the marshalling cost or send filters to the platform.
		scanCtx := ctx
		if outputDir != "" {
			scanCtx = scandb.WithFilterCapture(scanCtx, func(codeIDs []string) {
				if err := scanDataStore.WriteAssetFilters(context.Background(), codeIDs); err != nil {
					log.Warn().Err(err).Msg("failed to persist asset filters")
				}
			})
		}

		_, ls, err := inmemory.NewServices(runtime, inmemory.WithDataWriter(scandb.NewScanDataStoreWrapper(scanDataStore, assetMrn)))

		if err != nil {
			return err
		}

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
		if err := f(scanCtx, ls); err != nil {
			return err
		}

		if upstream != nil {
			scanDataPath, err := scanDataStore.Finalize()
			if err != nil {
				return err
			}

			return uploadScanDataStore(ctx, upstream, assetMrn, scanDataPath)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func withSqliteDataStore(assetMrn string, f func(scanDataStore *scandb.SqliteScanDataStore) error) error {
	// When SetOutputDir is configured, write the scan db to that directory and
	// keep it after upload — used by `cnspec scan --output-scan-db` to capture
	// seeds for the loadtest tool. Otherwise create a temp file we delete.
	dir := outputDir
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
	defer func() {
		if keep {
			log.Info().Str("path", tmpFile.Name()).Msg("scan database saved")
			return
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

func uploadScanDataStore(ctx context.Context, services *policy.Services, assetMrn string, scanDataPath string) error {
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

	// Confirm the upload
	_, err = services.ReportUploadCompleted(ctx, &policy.ReportUploadCompletedReq{
		UploadSessionId: urlResp.UploadSessionId,
		ScopeMrn:        assetMrn,
	})
	if err != nil {
		return err
	}

	log.Info().Str("session", urlResp.UploadSessionId).Msg("successfully uploaded scan data store")
	return nil
}
