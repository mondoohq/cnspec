// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnspec/v11/internal/tfgen"
)

// Ms365Integration represents the configuration of a Microsoft 365 integration to be created.
type Ms365Integration struct {
	Name    string
	Space   string
	Primary string
}

// GenerateMs365HCL generates automation code to create a Microsoft 365 integration.
func GenerateMs365HCL(integration Ms365Integration) (string, error) {
	// Validate integration name is not empty, if it is, generate a random one
	if integration.Name == "" {
		integration.Name = generateAzureIntegrationName(integration.Primary)
		log.Info().Msgf(
			"integration name not provided, using %s",
			theme.DefaultTheme.Primary(integration.Name),
		)
	}

	// Generate required providers block
	requiredProvidersBlock, err := tfgen.CreateRequiredProviders(
		tfgen.NewRequiredProvider("mondoo",
			tfgen.HclRequiredProviderWithSource(mondooProviderSource),
			tfgen.HclRequiredProviderWithVersion(mondooProviderVersion),
		),
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate required providers")
	}

	featuresBlock, err := tfgen.HclCreateGenericBlock("features", []string{}, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to append features block to azurerm provider")
	}
	selfSignedCertSubjectBlock, err := tfgen.HclCreateGenericBlock("subject", nil,
		tfgen.Attributes{"common_name": "mondoo"},
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate self signed cert subject block")
	}

	mondooProviderHclModifier := []tfgen.HclProviderModifier{}
	if integration.Space != "" {
		mondooProviderHclModifier = append(mondooProviderHclModifier, tfgen.HclProviderWithAttributes(
			tfgen.Attributes{"space": integration.Space},
		))
	}

	azurermProviderHclModifier := []tfgen.HclProviderModifier{
		tfgen.HclProviderWithGenericBlocks(featuresBlock),
	}
	if integration.Primary != "" {
		azurermProviderHclModifier = append(azurermProviderHclModifier,
			tfgen.HclProviderWithAttributes(tfgen.Attributes{"subscription_id": integration.Primary}),
		)
	}

	var (
		providerAzureAD       = tfgen.NewProvider("azuread")
		providerAzureRM       = tfgen.NewProvider("azurerm", azurermProviderHclModifier...)
		providerMondoo        = tfgen.NewProvider("mondoo", mondooProviderHclModifier...)
		dataADClientConfig    = tfgen.NewDataSource("azuread_client_config", "current")
		resourceAdApplication = tfgen.NewResource("azuread_application", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"display_name":  "mondoo_ms365",
				"owners":        []interface{}{dataADClientConfig.TraverseRef("object_id")},
				"marketing_url": "https://www.mondoo.com/",
			}),
		)
		resourceTLSPrivateKey = tfgen.NewResource("tls_private_key", "credential",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"algorithm": "RSA",
				"rsa_bits":  4096,
			}),
		)
		resourceTLSSelfSignedCert = tfgen.NewResource("tls_self_signed_cert", "credential",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"private_key_pem":       resourceTLSPrivateKey.TraverseRef("private_key_pem"),
				"validity_period_hours": 4096,
				"early_renewal_hours":   3,
				"allowed_uses": []string{
					"key_encipherment",
					"digital_signature",
					"data_encipherment",
					"cert_signing",
				},
			}),
			tfgen.HclResourceWithGenericBlocks(selfSignedCertSubjectBlock),
		)
		resourceADApplicationCertificate = tfgen.NewResource("azuread_application_certificate", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"application_id": resourceAdApplication.TraverseRef("id"),
				"type":           "AsymmetricX509Cert",
				"value":          resourceTLSSelfSignedCert.TraverseRef("cert_pem"),
			}),
		)
		resourceADServicePrincipal = tfgen.NewResource("azuread_service_principal", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"client_id":                    resourceAdApplication.TraverseRef("client_id"),
				"app_role_assignment_required": false,
				"owners":                       []interface{}{dataADClientConfig.TraverseRef("object_id")},
			}),
		)
		resourceADReadersDirectoryRole = tfgen.NewResource("azuread_directory_role", "global_reader",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{"display_name": "Global Reader"}),
		)
		resourceADExchangeAdminDirectoryRole = tfgen.NewResource("azuread_directory_role", "exchange_admin",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"display_name": "Exchange Administrator",
			}),
		)
		resourceTimeSleep = tfgen.NewResource("time_sleep", "wait_time",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{"create_duration": "60s"}),
		)
		resourceADReadersRoleAssignment = tfgen.NewResource("azuread_directory_role_assignment", "global_reader",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"role_id":             resourceADReadersDirectoryRole.TraverseRef("template_id"),
				"principal_object_id": resourceADServicePrincipal.TraverseRef("object_id"),
				"depends_on":          []interface{}{resourceTimeSleep.TraverseRef()},
			}),
		)
		resourceADExchangeAdminRoleAssignment = tfgen.NewResource("azuread_directory_role_assignment", "exchange_admin",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"principal_object_id": resourceADServicePrincipal.TraverseRef("object_id"),
				"role_id":             resourceADExchangeAdminDirectoryRole.TraverseRef("object_id"),
				"depends_on":          []interface{}{resourceTimeSleep.TraverseRef()},
			}),
		)
		integrationAttributes = tfgen.Attributes{
			"name":      integration.Name,
			"tenant_id": dataADClientConfig.TraverseRef("tenant_id"),
			"client_id": resourceAdApplication.TraverseRef("client_id"),
			"credentials": tfgen.Attributes{
				"pem_file": tfgen.NewFuncCall("join",
					tfgen.CreateSimpleTraversal(`"\n", [tls_self_signed_cert.credential.cert_pem, tls_private_key.credential.private_key_pem]`),
				),
			},
			"depends_on": []interface{}{
				resourceADServicePrincipal.TraverseRef(),
				resourceADApplicationCertificate.TraverseRef(),
				resourceADReadersRoleAssignment.TraverseRef(),
			},
		}
	)

	resourceMondooIntegration := tfgen.NewResource("mondoo_integration_ms365", "this", // Changed to match the .tf file
		tfgen.HclResourceWithAttributes(integrationAttributes),
	)

	blocks, err := tfgen.ObjectsToBlocks(
		providerMondoo,
		providerAzureAD,
		providerAzureRM,
		dataADClientConfig,
		resourceTLSPrivateKey,
		resourceTLSSelfSignedCert,
		resourceADApplicationCertificate,
		resourceADServicePrincipal,
		resourceAdApplication,
		resourceADReadersDirectoryRole,
		resourceADReadersRoleAssignment,
		resourceADExchangeAdminDirectoryRole,
		resourceADExchangeAdminRoleAssignment,
		resourceTimeSleep,
		resourceMondooIntegration,
	)
	if err != nil {
		return "", err
	}
	hclBlocks := tfgen.CombineHclBlocks(requiredProvidersBlock, blocks)

	return tfgen.CreateHclStringOutput(hclBlocks...), nil
}
