// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v11/policy"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/awssnssqs"
)

var sqsRegex = regexp.MustCompile(`(https:\/\/|http:\/\/)?(sqs)[.][a-z]{2}[-][a-z]{3,}[-][0-9]{1}[.](amazonaws.com)[\/][0-9]{12}[\/]{1}[a-zA-Z0-9-_]*`)

type awsSqsHandler struct {
	sqsQueueUrl string
	format      Format
}

func (h *awsSqsHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	// the url may be passed in with a https:// or an http:// prefix, we can trim those
	trimmedUrl := strings.TrimPrefix(h.sqsQueueUrl, "https://")
	trimmedUrl = strings.TrimPrefix(trimmedUrl, "http://")
	topic, err := pubsub.OpenTopic(ctx, "awssqs://"+trimmedUrl)
	if err != nil {
		return err
	}
	defer topic.Shutdown(ctx) //nolint: errcheck
	data, err := h.convertReport(report)
	if err != nil {
		return err
	}
	err = topic.Send(ctx, &pubsub.Message{
		Body: data,
	})
	if err != nil {
		return err
	}
	log.Info().Str("url", h.sqsQueueUrl).Msg("sent report to SQS queue")
	return nil
}

func (h *awsSqsHandler) convertReport(report *policy.ReportCollection) ([]byte, error) {
	switch h.format {
	case FormatYAMLv1:
		return reportToYamlV1(report)
	case FormatJSONv1:
		return reportToJsonV1(report)
	case FormatYAMLv2:
		return reportToYamlV2(report)
	case FormatJSONv2:
		return reportToJsonV2(report)
	default:
		return nil, fmt.Errorf("'%s' is not supported in the aws sqs handler, please use one of the other formats", string(h.format))
	}
}
