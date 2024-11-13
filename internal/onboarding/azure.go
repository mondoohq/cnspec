// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnspec/v11/internal/tfgen"
)

// AzureIntegration represents the configuration of an Azure integration to be created.
type AzureIntegration struct {
	Name    string
	Space   string
	Primary string
	Allow   []string
	Deny    []string
	ScanVMs bool
}

// GenerateAzureHCL generates automation code to create an Azure integration.
func GenerateAzureHCL(integration AzureIntegration) (string, error) {
	// Validate integration name is not empty, if it is, generate a random one
	if integration.Name == "" {
		integration.Name = generateAzureIntegrationName(integration.Primary)
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
	customRolePermissionsBlock, err := tfgen.HclCreateGenericBlock("permissions", nil,
		tfgen.Attributes{
			"actions ": []string{
				"Microsoft.Compute/virtualMachines/runCommands/read",
				"Microsoft.Compute/virtualMachines/runCommands/write",
				"Microsoft.Compute/virtualMachines/runCommands/delete",
			},
			// not_actions = []
			// data_actions = []
			// not_data_actions = []
		},
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate custom role permissions block")
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
				"display_name":  "mondoo_security", // @afiune should we customize this?
				"owners":        []interface{}{dataADClientConfig.TraverseRef("object_id")},
				"marketing_url": "https://www.mondoo.com/",
			}),
		)
		resourceTLSPrivateKey = tfgen.NewResource("tls_private_key", "credential",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{"algorithm": "RSA"}),
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
				// see https://github.com/hashicorp/terraform-provider-azuread/issues/1227
				"application_id": resourceAdApplication.TraverseRef("id"),
				"type":           "AsymmetricX509Cert",
				"value":          resourceTLSSelfSignedCert.TraverseRef("cert_pem"),
			}),
		)
		resourceADServicePrincipal = tfgen.NewResource("azuread_service_principal", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"client_id": resourceAdApplication.TraverseRef("client_id"),
				"owners":    []interface{}{dataADClientConfig.TraverseRef("object_id")},
			}),
		)
		// TODO can we skip subscriptions if deny is provided
		dataRMAllSubscriptions         = tfgen.NewDataSource("azurerm_subscriptions", "available")
		resourceRMReaderRoleAssignment = tfgen.NewResource("azurerm_role_assignment", "reader",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"role_definition_name": "Reader",

				"count":        tfgen.NewFuncCall("length", dataRMAllSubscriptions.TraverseRef("subscriptions")),
				"scope":        dataRMAllSubscriptions.TraverseRef("subscriptions[count.index]", "id"),
				"principal_id": resourceADServicePrincipal.TraverseRef("object_id"),
			}),
		)
		// This is the way we avoid Grant Admin Consent issue.
		//
		// => https://docs.microsoft.com/en-us/azure/active-directory/roles/permissions-reference#directory-readers
		//
		resourceADReadersDirectoryRole = tfgen.NewResource("azuread_directory_role", "readers",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{"display_name": "Directory Readers"}),
		)
		resourceTimeSleep = tfgen.NewResource("time_sleep", "wait_time",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{"create_duration": "60s"}),
		)
		resourceADReadersRoleAssignment = tfgen.NewResource("azuread_directory_role_assignment", "readers",
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
			"scan_vms":  integration.ScanVMs,
			"credentials": tfgen.Attributes{
				"pem_file": tfgen.NewFuncCall("join",
					tfgen.CreateSimpleTraversal(`"\n", [tls_self_signed_cert.credential.cert_pem, tls_private_key.credential.private_key_pem]`),
				),
			},
			"depends_on": []interface{}{
				resourceADServicePrincipal.TraverseRef(),
				resourceRMReaderRoleAssignment.TraverseRef(),
				resourceADApplicationCertificate.TraverseRef(),
				resourceADReadersRoleAssignment.TraverseRef(),
			},
		}
		// Custom role needed only when scanning VMs
		resourceRMCustomRoleDefinition = tfgen.NewResource("azurerm_role_definition", "mondoo_security",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"name":              "tf-mondoo-security-role",
				"description":       "Allow Mondoo Security to use run commands for Virtual Machine scanning",
				"scope":             fmt.Sprintf("/subscriptions/%s", integration.Primary),
				"assignable_scopes": dataRMAllSubscriptions.TraverseRef("subscriptions[*]", "id"),
			}),
			tfgen.HclResourceWithGenericBlocks(customRolePermissionsBlock),
		)
		// Adds the custom role to all subscriptions
		resourceRMCustomRoleAssignment = tfgen.NewResource("azurerm_role_assignment", "mondoo_security",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"role_definition_id": resourceRMCustomRoleDefinition.TraverseRef("role_definition_resource_id"),

				"count":        tfgen.NewFuncCall("length", dataRMAllSubscriptions.TraverseRef("subscriptions")),
				"scope":        dataRMAllSubscriptions.TraverseRef("subscriptions[count.index]", "id"),
				"principal_id": resourceADServicePrincipal.TraverseRef("object_id"),
			}),
		)
	)

	// Allow and Deny are mutually exclusive and can't be added together
	if len(integration.Allow) != 0 {
		integrationAttributes["subscription_allow_list"] = integration.Allow
	} else if len(integration.Deny) != 0 {
		integrationAttributes["subscription_deny_list"] = integration.Deny
	}

	resourceMondooIntegration := tfgen.NewResource("mondoo_integration_azure", "this",
		tfgen.HclResourceWithAttributes(integrationAttributes),
	)

	blocks, err := tfgen.ObjectsToBlocks(
		providerMondoo,
		providerAzureAD,
		providerAzureRM,
		dataADClientConfig,
		dataRMAllSubscriptions,
		resourceTLSPrivateKey,
		resourceTLSSelfSignedCert,
		resourceADApplicationCertificate,
		resourceADServicePrincipal,
		resourceAdApplication,
		resourceADReadersDirectoryRole,
		resourceRMReaderRoleAssignment,
		resourceADReadersRoleAssignment,
		resourceTimeSleep,
		resourceMondooIntegration,
	)
	if err != nil {
		return "", err
	}

	hclBlocks := tfgen.CombineHclBlocks(requiredProvidersBlock, blocks)
	if integration.ScanVMs {
		vmScanningBlocks, err := tfgen.ObjectsToBlocks(
			resourceRMCustomRoleAssignment,
			resourceRMCustomRoleDefinition,
		)
		if err != nil {
			return "", err
		}
		hclBlocks = tfgen.CombineHclBlocks(hclBlocks, vmScanningBlocks)
	}

	return tfgen.CreateHclStringOutput(hclBlocks...), nil
}

type AzAccount struct {
	CloudName        string `json:"cloudName"`
	HomeTenantID     string `json:"homeTenantId"`
	ID               string `json:"id"`
	IsDefault        bool   `json:"isDefault"`
	ManagedByTenants []any  `json:"managedByTenants"`
	Name             string `json:"name"`
	State            string `json:"state"`
	TenantID         string `json:"tenantId"`
	User             struct {
		CloudShellID bool   `json:"cloudShellID"`
		Name         string `json:"name"`
		Type         string `json:"type"`
	} `json:"user"`
}

// Printable Keys and Values are used by the cnquery/cli/components package.
var assetPrintableKeys = []string{"name", "subscription-id"}

func (a AzAccount) PrintableKeys() []string {
	return assetPrintableKeys
}
func (a AzAccount) PrintableValue(index int) string {
	switch assetPrintableKeys[index] {
	case "name":
		return a.Name
	case "subscription-id":
		if a.IsDefault {
			return fmt.Sprintf("(default) %s", a.ID)
		}
		return a.ID
	default:
		return a.Display()
	}
}

// Display implements SelectableItem from the cnquery/cli/components package.
func (az AzAccount) Display() string {
	if az.IsDefault {
		return fmt.Sprintf("(%s) %s [default]", az.ID, az.Name)
	}
	return fmt.Sprintf("(%s) %s", az.ID, az.Name)
}

func generateAzureIntegrationName(subscription string) string {
	var subsSplit = strings.Split(subscription, "-")
	return "subscription-" + subsSplit[len(subsSplit)-1]
}
