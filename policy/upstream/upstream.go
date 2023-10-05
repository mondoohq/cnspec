// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package upstream

//go:generate protoc --proto_path=../../:../../cnquery:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnspec_upstream.proto

import (
	"context"
	"net/url"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v9/policy"
	"go.mondoo.com/ranger-rpc"
)

const sharedReportUrl = "https://report.api.mondoo.com"

func UploadSharedReport(report *policy.ReportCollection, reportUrl string, proxy *url.URL) (*ReportID, error) {
	if reportUrl == "" {
		reportUrl = sharedReportUrl
	}

	httpClient := ranger.NewHttpClient(ranger.WithProxy(proxy))
	sharedReportClient, err := NewReportingClient(reportUrl, httpClient)
	if err != nil {
		log.Error().Err(err).Msg("error initializing shared report client")
		return nil, err
	}

	return sharedReportClient.StoreReport(context.Background(), report)
}
