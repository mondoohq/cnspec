// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v10/policy"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/azuresb"
)

var sbusRegex = regexp.MustCompile(`(https:\/\/|http:\/\/)?[a-zA-Z0-9-_]*[.](servicebus.windows.net)[\/][a-zA-Z0-9-_]*`)

type azureSbusHandler struct {
	url    string
	format Format
}

func (h *azureSbusHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	// the url may be passed in with a https:// or an http:// prefix, we can trim those
	trimmedUrl := strings.TrimPrefix(h.url, "https://")
	trimmedUrl = strings.TrimPrefix(trimmedUrl, "http://")
	parts := strings.Split(trimmedUrl, "/")
	// we assume the last part of the url is the sender name, e.g. https://test-bus.servicebus.windows.net/msg-topic
	senderName := parts[len(parts)-1]
	sbusUrl := strings.TrimSuffix(trimmedUrl, "/"+senderName)

	client, err := azuresb.NewClientFromServiceBusHostname(sbusUrl, &azservicebus.ClientOptions{})
	if err != nil {
		return err
	}
	defer client.Close(ctx) //nolint: errcheck
	sender, err := azuresb.NewSender(client, senderName, &azservicebus.NewSenderOptions{})
	if err != nil {
		return err
	}
	defer sender.Close(ctx) //nolint: errcheck
	topic, err := azuresb.OpenTopic(ctx, sender, &azuresb.TopicOptions{})
	if err != nil {
		return err
	}
	defer topic.Shutdown(ctx) //nolint: errcheck
	data, err := h.convertReport(report)
	if err != nil {
		return err
	}
	msg := &pubsub.Message{
		Body: data,
	}
	err = topic.Send(ctx, msg)
	if err != nil {
		return err
	}
	log.Info().Str("url", h.url).Msg("sent report to azure service bus")
	return nil
}

func (h *azureSbusHandler) convertReport(report *policy.ReportCollection) ([]byte, error) {
	switch h.format {
	case YAML:
		return reportToYaml(report)
	case JSON:
		return reportToJson(report)
	default:
		return nil, fmt.Errorf("'%s' is not supported in the azure service bus handler, please use one of the other formats", string(h.format))
	}
}
