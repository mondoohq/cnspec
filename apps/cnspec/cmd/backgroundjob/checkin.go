// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backgroundjob

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/sysinfo"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/plugins/scope"
)

type CheckinHandler struct {
	endpoint   string
	httpClient *http.Client
	mrn        string
	creds      *upstream.ServiceAccountCredentials
	sysInfo    *sysinfo.SystemInfo
}

func newCheckinHandler(
	httpClient *http.Client,
	endpoint string,
	agentMrn string,
	config *upstream.UpstreamConfig,
) (*CheckinHandler, error) {
	if agentMrn == "" {
		return nil, errors.New("could not determine agent MRN")
	}
	return &CheckinHandler{
		endpoint:   endpoint,
		httpClient: httpClient,
		mrn:        agentMrn,
		creds:      config.Creds,
	}, nil
}

func NewCheckInHandlerWithInfo(httpClient *http.Client,
	endpoint string,
	agentMrn string,
	config *upstream.UpstreamConfig,
) (*CheckinHandler, error) {
	return newCheckinHandler(httpClient, endpoint, agentMrn, config)
}

func (c *CheckinHandler) CheckIn(ctx context.Context) error {
	if c.sysInfo == nil {
		sysInfo, err := sysinfo.Get()
		if err != nil {
			return errors.Wrap(err, "could not determine system information")
		}
		c.sysInfo = sysInfo
	}
	// gather service account
	plugins := []ranger.ClientPlugin{}
	plugins = append(plugins, sysInfoHeader(c.sysInfo, cnquery.DefaultFeatures))
	if c.creds != nil && len(c.creds.Mrn) > 0 {
		certAuth, err := upstream.NewServiceAccountRangerPlugin(c.creds)
		if err != nil {
			return errors.Wrap(err, "invalid credentials")
		}
		plugins = append(plugins, certAuth)
	} else {
		return errors.New("no credentials configured")
	}

	client, err := upstream.NewAgentManagerClient(c.endpoint, c.httpClient, plugins...)
	if err != nil {
		return errors.Wrap(err, "could not connect to mondoo platform")
	}

	_, err = client.HealthCheck(ctx, &upstream.AgentInfo{
		Mrn:              c.mrn,
		Version:          c.sysInfo.Version,
		Build:            c.sysInfo.Build,
		PlatformName:     c.sysInfo.Platform.Name,
		PlatformRelease:  c.sysInfo.Platform.Version,
		PlatformArch:     c.sysInfo.Platform.Arch,
		PlatformIp:       c.sysInfo.IP,
		PlatformHostname: c.sysInfo.Hostname,
		Labels:           nil,
		PlatformId:       c.sysInfo.PlatformId,
	})
	if err != nil {
		return errors.Wrap(err, "failed to check in upstream")
	}

	return nil
}

func sysInfoHeader(sysInfo *sysinfo.SystemInfo, features cnquery.Features) ranger.ClientPlugin {
	const (
		HttpHeaderUserAgent      = "User-Agent"
		HttpHeaderClientFeatures = "Mondoo-Features"
		HttpHeaderPlatformID     = "Mondoo-PlatformID"
	)

	h := http.Header{}
	info := map[string]string{
		"cnquery": cnquery.Version,
		"build":   cnquery.Build,
	}
	if sysInfo != nil {
		info["PN"] = sysInfo.Platform.Name
		info["PR"] = sysInfo.Platform.Version
		info["PA"] = sysInfo.Platform.Arch
		info["IP"] = sysInfo.IP
		info["HN"] = sysInfo.Hostname
		h.Set(HttpHeaderPlatformID, sysInfo.PlatformId)
	}
	h.Set(HttpHeaderUserAgent, scope.XInfoHeader(info))
	h.Set(HttpHeaderClientFeatures, features.Encode())
	return scope.NewCustomHeaderRangerPlugin(h)
}
