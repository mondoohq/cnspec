// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding

import (
	"github.com/cockroachdb/errors"
	"github.com/hashicorp/hcl/v2/hclwrite"
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

func createResourceAccessBlock(resourceAppID string, resourceAccesses []map[string]interface{}) (*hclwrite.Block, error) {
	resourceAccessBlocks := make([]*hclwrite.Block, len(resourceAccesses))
	for i, access := range resourceAccesses {
		block, err := tfgen.HclCreateGenericBlock("resource_access", nil, access)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create resource_access block")
		}
		resourceAccessBlocks[i] = block
	}

	requiredResourceAccessBlock, err := tfgen.HclCreateGenericBlock("required_resource_access", nil, map[string]interface{}{
		"resource_app_id": resourceAppID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create required_resource_access block")
	}

	for _, block := range resourceAccessBlocks {
		requiredResourceAccessBlock.Body().AppendBlock(block)
	}

	return requiredResourceAccessBlock, nil
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
	requiredResourceAccessBlock1, err := createResourceAccessBlock(
		"00000003-0000-0000-c000-000000000000",
		[]map[string]interface{}{
			{"id": "246dd0d5-5bd0-4def-940b-0421030a5b68", "type": "Role"},
			{"id": "e321f0bb-e7f7-481e-bb28-e3b0b32d4bd0", "type": "Role"},
			{"id": "5e0edab9-c148-49d0-b423-ac253e121825", "type": "Role"},
			{"id": "bf394140-e372-4bf9-a898-299cfc7564e5", "type": "Role"},
			{"id": "dc377aa6-52d8-4e23-b271-2a7ae04cedf3", "type": "Role"},
			{"id": "9e640839-a198-48fb-8b9a-013fd6f6cbcd", "type": "Role"},
			{"id": "37730810-e9ba-4e46-b07e-8ca78d182097", "type": "Role"},
		},
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate first resource access block")
	}

	requiredResourceAccessBlock2, err := createResourceAccessBlock(
		"00000003-0000-0ff1-ce00-000000000000",
		[]map[string]interface{}{
			{"id": "678536fe-1083-478a-9c59-b99265e6b0d3", "type": "Role"},
		},
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate second resource access block")
	}

	requiredResourceAccessBlock3, err := createResourceAccessBlock(
		"00000002-0000-0ff1-ce00-000000000000",
		[]map[string]interface{}{
			{"id": "dc50a0fb-09a3-484d-be87-e023b12c6440", "type": "Role"},
		},
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate third resource access block")
	}

	var (
		providerAzureAD = tfgen.NewProvider("azuread")
		providerAzureRM = tfgen.NewProvider("azurerm",
			tfgen.HclProviderWithAttributes(tfgen.Attributes{"subscription_id": integration.Primary}),
			tfgen.HclProviderWithGenericBlocks(featuresBlock),
		)
		providerMondoo = tfgen.NewProvider("mondoo", tfgen.HclProviderWithAttributes(
			tfgen.Attributes{"space": integration.Space},
		))
		dataADClientConfig    = tfgen.NewDataSource("azuread_client_config", "current")
		resourceAdApplication = tfgen.NewResource("azuread_application", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"display_name":  "mondoo_ms365",
				"owners":        []interface{}{dataADClientConfig.TraverseRef("object_id")},
				"marketing_url": "https://www.mondoo.com/",
			}),
			tfgen.HclResourceWithGenericBlocks(requiredResourceAccessBlock1, requiredResourceAccessBlock2, requiredResourceAccessBlock3),
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
		resourceTimeSleep,
		resourceMondooIntegration,
	)
	if err != nil {
		return "", err
	}
	hclBlocks := tfgen.CombineHclBlocks(requiredProvidersBlock, blocks)

	return tfgen.CreateHclStringOutput(hclBlocks...), nil
}
