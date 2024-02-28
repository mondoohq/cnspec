// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	pol "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v10/policy"
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

	cred, err := h.GetTokenChain(ctx, &azidentity.DefaultAzureCredentialOptions{})
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
	if h.format == JSON {
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
	case YAML:
		return reportToYaml(report)
	case JSON:
		return reportToJson(report)
	default:
		return nil, fmt.Errorf("'%s' is not supported in the azure service bus handler, please use one of the other formats", string(h.format))
	}
}

// sometimes we run into a 'managed identity timed out' error when using a managed identity.
// according to https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/azidentity/TROUBLESHOOTING.md#troubleshoot-defaultazurecredential-authentication-issues
// we should instead use the NewManagedIdentityCredential directly. this function adds a bit more by
// also using other credentials to create a chained token credential
func (h *azureSbusHandler) GetTokenChain(ctx context.Context, options *azidentity.DefaultAzureCredentialOptions) (*azidentity.ChainedTokenCredential, error) {
	if options == nil {
		options = &azidentity.DefaultAzureCredentialOptions{}
	}

	chain := []azcore.TokenCredential{}

	cli, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{})
	if err == nil {
		chain = append(chain, cli)
	}
	envCred, err := azidentity.NewEnvironmentCredential(&azidentity.EnvironmentCredentialOptions{ClientOptions: options.ClientOptions})
	if err == nil {
		chain = append(chain, envCred)
	}
	mic, err := azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{ClientOptions: options.ClientOptions})
	if err == nil {
		timedMic := &TimedManagedIdentityCredential{mic: *mic, timeout: 5 * time.Second}
		chain = append(chain, timedMic)
	}
	wic, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		ClientOptions:            options.ClientOptions,
		DisableInstanceDiscovery: options.DisableInstanceDiscovery,
		TenantID:                 options.TenantID,
	})
	if err == nil {
		chain = append(chain, wic)
	}

	return azidentity.NewChainedTokenCredential(chain, nil)
}

type TimedManagedIdentityCredential struct {
	mic     azidentity.ManagedIdentityCredential
	timeout time.Duration
}

func (t *TimedManagedIdentityCredential) GetToken(ctx context.Context, opts pol.TokenRequestOptions) (azcore.AccessToken, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	var tk azcore.AccessToken
	var err error
	if t.timeout > 0 {
		c, cancel := context.WithTimeout(ctx, t.timeout)
		defer cancel()
		tk, err = t.mic.GetToken(c, opts)
		if err != nil {
			var authFailedErr *azidentity.AuthenticationFailedError
			if errors.As(err, &authFailedErr) && strings.Contains(err.Error(), "context deadline exceeded") {
				err = azidentity.NewCredentialUnavailableError("managed identity request timed out")
			}
		} else {
			// some managed identity implementation is available, so don't apply the timeout to future calls
			t.timeout = 0
		}
	} else {
		tk, err = t.mic.GetToken(ctx, opts)
	}
	return tk, err
}
