// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/ranger-rpc"
)

// Client abstracts the upstream calls the loadtest tool makes. The dryRun
// implementation logs but does not send, so users can validate flag values
// against a real config without producing server load.
type Client interface {
	SynchronizeAsset(ctx context.Context, spaceMrn string, asset *inventory.Asset) (string, error)
	ResolveAndUpdateJobs(ctx context.Context, assetMrn string, asset *inventory.Asset) error
	StoreResults(ctx context.Context, req *policy.StoreResultsReq) error
}

// NewServicesClient builds a real upstream client using cnspec's standard
// service-account auth path. The credentials are read by the caller via the
// existing config loader (apps/cnspec/cmd) and passed in as a fully-formed
// UpstreamConfig — keeping this package free of CLI/config concerns.
func NewServicesClient(cfg *upstream.UpstreamConfig) (Client, error) {
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

	services, err := policy.NewRemoteServices(cfg.ApiEndpoint, plugins, httpClient)
	if err != nil {
		return nil, errors.Wrap(err, "connect to upstream")
	}
	return &servicesClient{services: services, httpClient: httpClient}, nil
}

type servicesClient struct {
	services   *policy.Services
	httpClient *http.Client
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

func (c *servicesClient) ResolveAndUpdateJobs(ctx context.Context, assetMrn string, asset *inventory.Asset) error {
	filters := []*policy.Mquery{}
	if asset.Platform != nil {
		filters = append(filters, &policy.Mquery{
			Mql: assetPlatformFilter(asset),
		})
	}
	_, err := c.services.ResolveAndUpdateJobs(ctx, &policy.UpdateAssetJobsReq{
		AssetMrn:     assetMrn,
		AssetFilters: filters,
	})
	return err
}

func (c *servicesClient) StoreResults(ctx context.Context, req *policy.StoreResultsReq) error {
	_, err := c.services.StoreResults(ctx, req)
	return err
}

// assetPlatformFilter constructs the minimal platform-matching MQL the policy
// resolver uses to pick which queries apply to this asset. Using just the
// platform name covers the common case (e.g. ubuntu, amazonlinux); more
// elaborate filtering can be added if a load-test scenario needs it.
func assetPlatformFilter(asset *inventory.Asset) string {
	if asset.Platform == nil || asset.Platform.Name == "" {
		return "true"
	}
	return "asset.platform == \"" + asset.Platform.Name + "\""
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

func (d *dryRunClient) ResolveAndUpdateJobs(ctx context.Context, assetMrn string, asset *inventory.Asset) error {
	log.Info().Str("asset", assetMrn).Msg("dry-run: ResolveAndUpdateJobs")
	return nil
}

func (d *dryRunClient) StoreResults(ctx context.Context, req *policy.StoreResultsReq) error {
	log.Info().Str("asset", req.AssetMrn).Int("scores", len(req.Scores)).Int("data", len(req.Data)).Bool("last", req.IsLastBatch).Msg("dry-run: StoreResults")
	return nil
}

// dataMap converts a template's data map (keyed by code_id) into the format
// StoreResults expects. Pulled out so both the real client and tests can
// share the conversion.
func dataMap(t *Template) map[string]*llx.Result { return t.Data }
