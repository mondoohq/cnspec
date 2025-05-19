// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnspec/v11/internal/tfgen"
)

// AzureIntegration represents the configuration of an AWS integration to be created.
type AwsIntegration struct {
	Name       string
	Space      string
	AccessKey  string
	SecretKey  string
	RoleArn    string
	ExternalID string
}

// GenerateAzureHCL generates automation code to create an AWS integration.
func GenerateAwsHCL(integration AwsIntegration) (string, error) {
	// Validate integration name is not empty, if it is, generate a random one
	if integration.Name == "" {
		integration.Name = "AWS Integration"
		log.Info().Msgf(
			"integration name not provided, using %s",
			theme.DefaultTheme.Primary(integration.Name),
		)
	}

	requiredProvidersBlock, err := tfgen.CreateRequiredProviders(
		tfgen.NewRequiredProvider("mondoo",
			tfgen.HclRequiredProviderWithSource(mondooProviderSource),
			tfgen.HclRequiredProviderWithVersion(mondooProviderVersion),
		),
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate required providers")
	}

	var (
		providerMondoo = tfgen.NewProvider("mondoo", tfgen.HclProviderWithAttributes(
			tfgen.Attributes{"space": integration.Space},
		))

		integrationKeyAttributes = tfgen.Attributes{
			"name": integration.Name,
			"credentials": tfgen.Attributes{
				"key": map[string]interface{}{
					"access_key": integration.AccessKey,
					"secret_key": integration.SecretKey,
				},
			},
		}
		integrationRoleAttributes = tfgen.Attributes{
			"name": integration.Name,
			"credentials": tfgen.Attributes{
				"role": map[string]interface{}{
					"role_arn":    integration.RoleArn,
					"external_id": integration.ExternalID,
				},
			},
		}
	)

	var resourceMondooIntegration *tfgen.HclResource
	if integration.AccessKey != "" && integration.SecretKey != "" {
		resourceMondooIntegration = tfgen.NewResource("mondoo_integration_aws", "this",
			tfgen.HclResourceWithAttributes(integrationKeyAttributes),
		)
	} else {
		resourceMondooIntegration = tfgen.NewResource("mondoo_integration_aws", "this",
			tfgen.HclResourceWithAttributes(integrationRoleAttributes),
		)
	}

	blocks, err := tfgen.ObjectsToBlocks(
		providerMondoo,
		resourceMondooIntegration,
	)
	if err != nil {
		return "", err
	}

	hclBlocks := tfgen.CombineHclBlocks(requiredProvidersBlock, blocks)

	return tfgen.CreateHclStringOutput(hclBlocks...), nil
}
