package components

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/upstream"
	"go.mondoo.com/ranger-rpc"
)

const sharedReportUrl = "https://report.api.mondoo.com"

func UploadSharedReport(report *policy.ReportCollection, reportUrl string, proxy *url.URL) error {
	if reportUrl == "" {
		reportUrl = sharedReportUrl
	}

	httpClient := ranger.NewHttpClient(ranger.WithProxy(proxy))
	sharedReportClient, err := upstream.NewReportingClient(reportUrl, httpClient)
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

func AskToUploadReport() bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("Do you want to view the report in the browser? [(y)es/(n)o]: ")

		answer, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal().Err(err).Msg("failed to read response")
		}

		answer = strings.ToLower(strings.TrimSpace(answer))

		if answer == "y" || answer == "yes" {
			return true
		} else if answer == "n" || answer == "no" {
			return false
		}
	}
}
