package upstream

//go:generate protoc --proto_path=../:../cnquery:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnspec_upstream.proto

import (
	"context"
	"fmt"
	"net/url"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/ranger-rpc"
)

const sharedReportUrl = "https://report.api.mondoo.com"

func UploadSharedReport(report *policy.ReportCollection, reportUrl string, proxy *url.URL) error {
	if reportUrl == "" {
		reportUrl = sharedReportUrl
	}

	httpClient := ranger.NewHttpClient(ranger.WithProxy(proxy))
	sharedReportClient, err := NewReportingClient(reportUrl, httpClient)
	if err != nil {
		log.Error().Err(err).Msg("error initializating shared report client")
		return err
	}

	reportId, err := sharedReportClient.StoreReport(context.Background(), report)
	if err != nil {
		log.Error().Err(err).Msg("error uploading shared report")
		return err
	}

	fmt.Printf("View your report at https://TBD.COM/space/overview?spaceId=%s\n", reportId.Id)
	return nil
}
