// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v12/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/v12/policy"
	"go.mondoo.com/cnspec/v12/policy/scandb"
)

func WithServices(ctx context.Context, runtime llx.Runtime, assetMrn string, upstreamClient *upstream.UpstreamClient, f func(*policy.LocalServices) error) error {
	err := withSqliteDataStore(assetMrn, func(scanDataStore scandb.ScanDataStore) error {
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
		if err := f(ls); err != nil {
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

func withSqliteDataStore(assetMrn string, f func(scanDataStore scandb.ScanDataStore) error) error {
	// create a temporary file for the scan data store
	tmpFile, err := os.CreateTemp("", "cnspec-scan-*.db")
	if err != nil {
		log.Error().Err(err).Msg("failed to create temporary file for scan data store")
		return err
	}
	tmpFile.Close() // nolint: errcheck
	defer func() {
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

	// Open the scan database file
	file, err := os.Open(scanDataPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create HTTP request for upload
	req, err := http.NewRequestWithContext(ctx, "PUT", url, file)
	if err != nil {
		return err
	}

	// Set required headers from the signed URL
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add file size header
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	req.ContentLength = fileInfo.Size()
	req.Header.Set("Content-Type", "application/octet-stream")

	// Perform the upload
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
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
