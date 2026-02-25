// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding

import (
	"github.com/cockroachdb/errors"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnspec/v13/internal/tfgen"
	"go.mondoo.com/mql/v13/cli/theme"
)

// IntuneIntegration represents the configuration of a Microsoft Intune integration to be created.
type IntuneIntegration struct {
	Name  string
	Space string
}

// The full list of permissions required by Mondoo to scan a Microsoft Intune environment
var IntuneAppPermissions = Permissions{
	{
		ResourceID: "MicrosoftGraph",
		Access: []ResourceAccess{
			{
				// Allows the app to read and write the properties of devices managed by Microsoft Intune
				ID:   "DeviceManagementManagedDevices.ReadWrite.All",
				Type: "Role",
			},
			{
				// Allows the app to read and write Microsoft Intune device configuration and policies
				ID:   "DeviceManagementConfiguration.ReadWrite.All",
				Type: "Role",
			},
			{
				// Allows the app to read and write all Intune-managed app properties
				ID:   "DeviceManagementApps.ReadWrite.All",
				Type: "Role",
			},
			{
				// Allows the app to read directory data
				ID:   "Directory.Read.All",
				Type: "Role",
			},
			{
				// Allows the app to read and write Intune service properties including device enrollment and third-party service connection configuration
				ID:   "DeviceManagementServiceConfig.ReadWrite.All",
				Type: "Role",
			},
		},
	},
}

// GenerateIntuneHCL generates automation code to create a Microsoft Intune integration.
func GenerateIntuneHCL(integration IntuneIntegration) (string, error) {
	// Validate integration name is not empty, if it is, generate a random one
	if integration.Name == "" {
		integration.Name = generateAzureIntegrationName(UuidGenerator())
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

	mondooProviderHclModifier := []tfgen.HclProviderModifier{}
	if integration.Space != "" {
		mondooProviderHclModifier = append(mondooProviderHclModifier, tfgen.HclProviderWithAttributes(
			tfgen.Attributes{"space": integration.Space},
		))
	}

	permissionsBlocks := []*hclwrite.Block{}
	for _, permission := range IntuneAppPermissions {
		block, err := permission.HclBlock()
		if err != nil {
			return "", errors.Wrap(err, "failed to generate intune permissions block")
		}
		permissionsBlocks = append(permissionsBlocks, block)
	}

	var (
		providerAzureAD       = tfgen.NewProvider("azuread")
		providerMondoo        = tfgen.NewProvider("mondoo", mondooProviderHclModifier...)
		dataADClientConfig    = tfgen.NewDataSource("azuread_client_config", "current")
		resourceAdApplication = tfgen.NewResource("azuread_application", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"display_name":  "mondoo-intune",
				"owners":        []any{dataADClientConfig.TraverseRef("object_id")},
				"marketing_url": "https://www.mondoo.com/",
			}),
			tfgen.HclResourceWithGenericBlocks(permissionsBlocks...),
		)
		resourceADAppPassword = tfgen.NewResource("azuread_application_password", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"application_id": resourceAdApplication.TraverseRef("id"),
				"display_name":   "mondoo-intune-credential",
			}),
		)
		resourceADServicePrincipal = tfgen.NewResource("azuread_service_principal", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"client_id":                    resourceAdApplication.TraverseRef("client_id"),
				"app_role_assignment_required": false,
				"owners":                       []any{dataADClientConfig.TraverseRef("object_id")},
			}),
		)
		integrationAttributes = tfgen.Attributes{
			"name":      integration.Name,
			"tenant_id": dataADClientConfig.TraverseRef("tenant_id"),
			"client_id": resourceAdApplication.TraverseRef("client_id"),
			"credentials": tfgen.Attributes{
				"client_secret": resourceADAppPassword.TraverseRef("value"),
			},
			"depends_on": []any{
				resourceADServicePrincipal.TraverseRef(),
				resourceADAppPassword.TraverseRef(),
			},
		}
	)

	resourceMondooIntegration := tfgen.NewResource("mondoo_integration_ms_intune", "this",
		tfgen.HclResourceWithAttributes(integrationAttributes),
	)

	blocks, err := tfgen.ObjectsToBlocks(
		providerMondoo,
		providerAzureAD,
		dataADClientConfig,
		dataMicrosoftPublishedAppIDs(),
		resourceADAppPassword,
		resourceADServicePrincipal,
		resourceAdApplication,
		resourceMondooIntegration,
	)
	if err != nil {
		return "", err
	}
	hclBlocks := tfgen.CombineHclBlocks(requiredProvidersBlock, blocks)

	// App Registration Permissions Blocks (admin consent grants)
	var permissionResources []tfgen.Object
	for _, permission := range IntuneAppPermissions {
		permissionResources = append(permissionResources, permission.HclResource())
		for _, access := range permission.Access {
			permissionResources = append(permissionResources, access.HclResource(resourceADServicePrincipal, permission.HclResource()))
		}
	}
	blocks, err = tfgen.ObjectsToBlocks(permissionResources...)
	if err != nil {
		return "", err
	}
	hclBlocks = tfgen.CombineHclBlocks(hclBlocks, blocks)

	return tfgen.CreateHclStringOutput(hclBlocks...), nil
}
