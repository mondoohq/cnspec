// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/azauth"
	"go.mondoo.com/cnspec/v11/policy"
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

	cred, err := azauth.GetDefaultChainedToken(&azidentity.DefaultAzureCredentialOptions{})
	if err != nil {
		return err
	}
	client, err := azservicebus.NewClient(sbusUrl, cred, &azservicebus.ClientOptions{})
	if err != nil {
		return err
	}
	defer client.Close(ctx) //nolint: errcheck
	sender, err := client.NewSender(senderName, &azservicebus.NewSenderOptions{})
	if err != nil {
		return err
	}
	defer sender.Close(ctx) //nolint: errcheck
	data, err := h.convertReport(report)
	if err != nil {
		return err
	}

	msg := &azservicebus.Message{
		Body: data,
	}
	if h.format == FormatJSONv1 || h.format == FormatJSONv2 {
		typ := "application/json"
		msg.ContentType = &typ
	}
	cancelCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = sender.SendMessage(cancelCtx, msg, &azservicebus.SendMessageOptions{})
	if err != nil {
		return err
	}
	log.Info().Str("url", h.url).Msg("sent report to azure service bus")
	return nil
}

func (h *azureSbusHandler) convertReport(report *policy.ReportCollection) ([]byte, error) {
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
		return nil, fmt.Errorf("'%s' is not supported in the azure service bus handler, please use one of the other formats", string(h.format))
	}
}
