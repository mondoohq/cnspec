// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v12/cli/theme"
	"go.mondoo.com/cnspec/v12/internal/tfgen"
)

// AwsIntegration represents the configuration of an AWS integration to be created.
type awsIntegration struct {
	Name      string
	Space     string
	AccessKey string
	SecretKey string
}

func integrationName(name string) string {
	if name == "" {
		log.Info().Msgf("integration name not provided, using %s", theme.DefaultTheme.Primary(name))
		return fmt.Sprintf("AWS Integration (%s)", uuid.New().String()[:7])
	}
	return name
}

func NewAwsIntegration(name, space, accessKey, secretKey string) awsIntegration {
	return awsIntegration{
		Name:      integrationName(name),
		Space:     space,
		AccessKey: accessKey,
		SecretKey: secretKey,
	}
}

func (a awsIntegration) Validate() []error {
	var errs []error
	if a.AccessKey == "" {
		errs = append(errs, errors.New("missing AWS access key"))
	}
	if a.SecretKey == "" {
		errs = append(errs, errors.New("missing AWS secret key"))
	}
	return errs
}

// GenerateAwsHCL generates automation code to create an AWS integration.
func GenerateAwsHCL(integration awsIntegration) (string, error) {
	if validationErrs := integration.Validate(); len(validationErrs) > 0 {
		return "", errors.Join(validationErrs...)
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

	mondooProviderHclModifier := []tfgen.HclProviderModifier{}
	if integration.Space != "" {
		mondooProviderHclModifier = append(mondooProviderHclModifier, tfgen.HclProviderWithAttributes(
			tfgen.Attributes{"space": integration.Space},
		))
	}

	providerMondoo := tfgen.NewProvider("mondoo", mondooProviderHclModifier...)

	accessKeyVariable := tfgen.NewVariable("aws_access_key",
		tfgen.HclVariableWithType("string"),
		tfgen.HclVariableWithDescription("AWS access key used for authentication"),
		tfgen.HclVariableWithSensitive(true),
	)
	secretKeyVariable := tfgen.NewVariable("aws_secret_key",
		tfgen.HclVariableWithType("string"),
		tfgen.HclVariableWithDescription("AWS secret key used for authentication"),
		tfgen.HclVariableWithSensitive(true),
	)

	resourceMondooIntegration := tfgen.NewResource("mondoo_integration_aws", "this",
		tfgen.HclResourceWithAttributes(tfgen.Attributes{
			"name": integration.Name,
			"credentials": tfgen.Attributes{
				"key": map[string]interface{}{
					"access_key": tfgen.CreateVariableReference("aws_access_key"),
					"secret_key": tfgen.CreateVariableReference("aws_secret_key"),
				},
			},
		}),
	)

	blocks, err := tfgen.ObjectsToBlocks(
		providerMondoo,
		accessKeyVariable,
		secretKeyVariable,
		resourceMondooIntegration,
	)
	if err != nil {
		return "", err
	}

	hclBlocks := tfgen.CombineHclBlocks(requiredProvidersBlock, blocks)
	return tfgen.CreateHclStringOutput(hclBlocks...), nil
}
