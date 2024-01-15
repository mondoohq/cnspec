// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backgroundjob

import (
	"context"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/sysinfo"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/plugins/scope"
)

type checkinPinger struct {
	ctx        context.Context
	interval   time.Duration
	quit       chan struct{}
	wg         sync.WaitGroup
	endpoint   string
	httpClient *http.Client
	mrn        string
	creds      *upstream.ServiceAccountCredentials
}

func NewCheckinPinger(ctx context.Context, httpClient *http.Client, endpoint string, agentMrn string, config *upstream.UpstreamConfig, interval time.Duration) (*checkinPinger, error) {
	if agentMrn == "" {
		return nil, errors.New("could not determine agent MRN")
	}
	return &checkinPinger{
		ctx:        ctx,
		interval:   interval,
		quit:       make(chan struct{}),
		endpoint:   endpoint,
		httpClient: httpClient,
		mrn:        agentMrn,
		creds:      config.Creds,
	}, nil
}

func (c *checkinPinger) Start() {
	// determine information about the client
	sysInfo, err := sysinfo.Get()
	if err != nil {
		log.Error().Err(err).Msg("could not gather client information")
		return
	}
	c.wg.Add(1)
	runCheckIn := func() {
		err := c.checkIn(sysInfo)
		if err != nil {
			log.Info().Err(err).Msg("could not perform check-in")
		}
	}

	// run check-in once on startup
	runCheckIn()

	jitter := time.Duration(rand.Int63n(int64(c.interval)))
	ticker := time.NewTicker(c.interval + jitter)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ticker.C:
				runCheckIn()
			case <-c.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *checkinPinger) Stop() {
	close(c.quit)
	c.wg.Wait()
}

func (c *checkinPinger) checkIn(sysInfo *sysinfo.SystemInfo) error {
	// gather service account
	plugins := []ranger.ClientPlugin{}
	plugins = append(plugins, sysInfoHeader(sysInfo, cnquery.DefaultFeatures))

	credentials := c.creds
	if credentials != nil && len(credentials.Mrn) > 0 {
		certAuth, err := upstream.NewServiceAccountRangerPlugin(credentials)
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

	_, err = client.HealthCheck(context.Background(), &upstream.AgentInfo{
		Mrn:              c.mrn,
		Version:          sysInfo.Version,
		Build:            sysInfo.Build,
		PlatformName:     sysInfo.Platform.Name,
		PlatformRelease:  sysInfo.Platform.Version,
		PlatformArch:     sysInfo.Platform.Arch,
		PlatformIp:       sysInfo.IP,
		PlatformHostname: sysInfo.Hostname,
		Labels:           nil,
		PlatformId:       sysInfo.PlatformId,
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
