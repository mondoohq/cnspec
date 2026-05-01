// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scandb"
	"go.mondoo.com/cnspec/v13/upload"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/plugins/scope"
)

// UserAgent is sent on every upstream request the loadtest client makes so
// platform-side telemetry can distinguish synthetic load from real scans.
// Format mirrors cnspec's normal product-token style; the suffix marks the
// traffic as load-test origin.
var UserAgent = "cnspec/" + cnspec.Version + " (loadtest)"

// ScanPayload holds everything that goes into one scan database file: the
// mutated scores for this iteration plus the (unchanged) template data and
// risks. The asset proto carries the platform info written into the db's
// asset table.
type ScanPayload struct {
	Asset  *inventory.Asset
	Scores []*policy.Score
	Data   map[string]*llx.Result
	Risks  []*policy.ScoredRiskFactor
}

// Client abstracts the upstream calls the loadtest tool makes. The dryRun
// implementation logs but does not send, so users can validate flag values
// against a real config without producing server load.
//
// The server replaced the old StoreResults RPC with an upload-based flow:
// GetUploadURL → HTTP PUT a serialized scan database → ReportUploadCompleted.
// UploadScanDB encapsulates that three-step dance so the runner doesn't have
// to know about presigned URLs or temp files.
type Client interface {
	SynchronizeAsset(ctx context.Context, spaceMrn string, asset *inventory.Asset) (string, error)
	ResolveAndUpdateJobs(ctx context.Context, assetMrn string, filters *policy.Mqueries) error
	UploadScanDB(ctx context.Context, assetMrn string, payload *ScanPayload) error
}

// NewServicesClient builds a real upstream client using cnspec's standard
// service-account auth path. The credentials are read by the caller via the
// existing config loader (apps/cnspec/cmd) and passed in as a fully-formed
// UpstreamConfig — keeping this package free of CLI/config concerns.
//
// tempDir is where per-scan SQLite files are staged; pass "" for os.TempDir().
func NewServicesClient(cfg *upstream.UpstreamConfig, tempDir string) (Client, error) {
	if cfg == nil || cfg.ApiEndpoint == "" {
		return nil, errors.New("upstream config is required (use --dry-run for offline runs)")
	}

	httpClient := ranger.DefaultHttpClient()
	plugins := []ranger.ClientPlugin{}
	if cfg.Creds != nil {
		certAuth, err := upstream.NewServiceAccountRangerPlugin(cfg.Creds)
		if err != nil {
			return nil, errors.Wrap(err, "build service account auth")
		}
		plugins = append(plugins, certAuth)
	}

	// Stamp every request with a loadtest-flavored User-Agent so the platform
	// can filter synthetic traffic out of real scan analytics.
	headers := http.Header{}
	headers.Set("User-Agent", UserAgent)
	plugins = append(plugins, scope.NewCustomHeaderRangerPlugin(headers))

	services, err := policy.NewRemoteServices(cfg.ApiEndpoint, plugins, httpClient)
	if err != nil {
		return nil, errors.Wrap(err, "connect to upstream")
	}
	return &servicesClient{services: services, httpClient: httpClient, tempDir: tempDir}, nil
}

type servicesClient struct {
	services   *policy.Services
	httpClient *http.Client
	tempDir    string
}

func (c *servicesClient) SynchronizeAsset(ctx context.Context, spaceMrn string, asset *inventory.Asset) (string, error) {
	resp, err := c.services.SynchronizeAssets(ctx, &policy.SynchronizeAssetsReq{
		SpaceMrn: spaceMrn,
		List:     []*inventory.Asset{asset},
	})
	if err != nil {
		return "", err
	}
	for _, d := range resp.Details {
		return d.AssetMrn, nil
	}
	return "", errors.New("server returned no asset details")
}

// ResolveAndUpdateJobs replays the captured filter set against the synthetic
// asset's MRN. The server compiles each filter's MQL, so we send the full
// Mquery list captured at scan time — code_ids alone fail with "query is
// not implemented".
func (c *servicesClient) ResolveAndUpdateJobs(ctx context.Context, assetMrn string, filters *policy.Mqueries) error {
	var items []*policy.Mquery
	if filters != nil {
		items = filters.Items
	}
	_, err := c.services.ResolveAndUpdateJobs(ctx, &policy.UpdateAssetJobsReq{
		AssetMrn:     assetMrn,
		AssetFilters: items,
	})
	return err
}

// UploadScanDB writes payload into a fresh SQLite scan database, fetches a
// presigned upload URL, PUTs the file to it, and reports completion. This
// mirrors the real cnspec scan upload flow (internal/datalakes/sqlite) so
// the load test exercises the same server-side ingestion path.
func (c *servicesClient) UploadScanDB(ctx context.Context, assetMrn string, payload *ScanPayload) error {
	path, err := writeScanDB(ctx, c.tempDir, assetMrn, payload)
	if err != nil {
		return errors.Wrap(err, "build scan db")
	}
	defer os.Remove(path)

	urlResp, err := c.services.GetUploadURL(ctx, &policy.GetUploadURLReq{
		Kind:     policy.UploadURLKind_UPLOAD_URL_KIND_SCAN_DATABASE_V0,
		ScopeMrn: assetMrn,
	})
	if err != nil {
		return errors.Wrap(err, "get upload url")
	}
	if urlResp.UploadUrl == nil {
		return errors.New("server returned no upload URL")
	}

	// Inject our load-test User-Agent into the upload PUT. Presigned URL
	// uploads don't go through ranger, so this is the only place we can
	// stamp the header — without it, the request would carry Go's default UA.
	uploadHeaders := mergeHeaders(urlResp.UploadUrl.Headers, map[string]string{"User-Agent": UserAgent})
	resp, err := upload.UploadFile(ctx, urlResp.UploadUrl.Url, uploadHeaders, path, "application/octet-stream")
	if err != nil {
		return errors.Wrap(err, "upload file")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return errors.Newf("upload failed status=%d body=%s", resp.StatusCode, string(body))
	}

	if _, err := c.services.ReportUploadCompleted(ctx, &policy.ReportUploadCompletedReq{
		UploadSessionId: urlResp.UploadSessionId,
		ScopeMrn:        assetMrn,
	}); err != nil {
		return errors.Wrap(err, "report upload completed")
	}
	return nil
}

// mergeHeaders returns a new map that contains every entry from base plus
// every entry from extra, with extra winning on conflict. Used so we can
// add our load-test User-Agent without losing the presigned URL's auth
// headers.
func mergeHeaders(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// writeScanDB stages a scan database file containing payload's asset, scores,
// data, and risks. The returned path lives under tempDir (or os.TempDir() if
// empty) and the caller is responsible for removing it.
func writeScanDB(ctx context.Context, tempDir, assetMrn string, payload *ScanPayload) (string, error) {
	tmp, err := os.CreateTemp(tempDir, "loadtest-scan-*.db")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	tmp.Close()

	store, err := scandb.NewSqliteScanDataStore(path, assetMrn)
	if err != nil {
		os.Remove(path)
		return "", errors.Wrap(err, "open scan db")
	}

	if err := store.WriteAsset(ctx, payload.Asset); err != nil {
		store.Close()
		os.Remove(path)
		return "", errors.Wrap(err, "write asset")
	}
	if err := store.WriteScores(ctx, payload.Scores); err != nil {
		store.Close()
		os.Remove(path)
		return "", errors.Wrap(err, "write scores")
	}
	if len(payload.Data) > 0 {
		results := make([]*llx.Result, 0, len(payload.Data))
		for _, r := range payload.Data {
			results = append(results, r)
		}
		if err := store.WriteData(ctx, results); err != nil {
			store.Close()
			os.Remove(path)
			return "", errors.Wrap(err, "write data")
		}
	}
	for _, r := range payload.Risks {
		if err := store.WriteRisk(ctx, r); err != nil {
			store.Close()
			os.Remove(path)
			return "", errors.Wrap(err, "write risk")
		}
	}

	if _, err := store.Finalize(); err != nil {
		store.Close()
		os.Remove(path)
		return "", errors.Wrap(err, "finalize scan db")
	}
	if err := store.Close(); err != nil {
		os.Remove(path)
		return "", errors.Wrap(err, "close scan db")
	}
	return path, nil
}

// dryRunClient implements Client by logging the calls it would make. Useful
// for verifying flag combinations and template loading against a real config
// without producing server load.
type dryRunClient struct{}

// NewDryRunClient returns a Client that logs but does not send.
func NewDryRunClient() Client { return &dryRunClient{} }

func (d *dryRunClient) SynchronizeAsset(ctx context.Context, spaceMrn string, asset *inventory.Asset) (string, error) {
	log.Info().Str("space", spaceMrn).Strs("platform_ids", asset.PlatformIds).Msg("dry-run: SynchronizeAsset")
	return "//captain.api.mondoo.app/spaces/dryrun/assets/" + asset.PlatformIds[0], nil
}

func (d *dryRunClient) ResolveAndUpdateJobs(_ context.Context, assetMrn string, filters *policy.Mqueries) error {
	n := 0
	if filters != nil {
		n = len(filters.Items)
	}
	log.Info().Str("asset", assetMrn).Int("filters", n).Msg("dry-run: ResolveAndUpdateJobs")
	return nil
}

func (d *dryRunClient) UploadScanDB(_ context.Context, assetMrn string, payload *ScanPayload) error {
	log.Info().Str("asset", assetMrn).Int("scores", len(payload.Scores)).Int("data", len(payload.Data)).Int("risks", len(payload.Risks)).Msg("dry-run: UploadScanDB")
	return nil
}
