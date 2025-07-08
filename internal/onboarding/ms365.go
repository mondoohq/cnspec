// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnspec/v11/internal/tfgen"
)

// Ms365Integration represents the configuration of a Microsoft 365 integration to be created.
type Ms365Integration struct {
	Name  string
	Space string
}

// The full list of permissions required by Mondoo to scan a Microsoft 365 tenant
var Ms365AppPermissions = Permissions{
	{
		ResourceID: "MicrosoftGraph",
		Access: []ResourceAccess{
			{
				// Allows the app to read all organization's policies without a signed-in user.
				ID:   "Policy.Read.All",
				Type: "Role",
			},
			{
				// Allows the app to read organization's security events without a signed-in user.
				ID:   "SecurityEvents.Read.All",
				Type: "Role",
			},
		},
	},
	{
		ResourceID: "Office365SharePointOnline",
		Access: []ResourceAccess{
			{
				ID:   "Sites.FullControl.All",
				Type: "Role",
			},
		},
	},
}

// function wrapper to mock during testing
var UuidGenerator func() string = func() string {
	return uuid.New().String()
}

// GenerateMs365HCL generates automation code to create a Microsoft 365 integration.
func GenerateMs365HCL(integration Ms365Integration) (string, error) {
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

	permissionsBlocks := []*hclwrite.Block{}
	for _, permission := range Ms365AppPermissions {
		block, err := permission.HclBlock()
		if err != nil {
			return "", errors.Wrap(err, "failed to generate ms365 permissions block")
		}
		permissionsBlocks = append(permissionsBlocks, block)
	}

	var (
		providerAzureAD       = tfgen.NewProvider("azuread")
		providerMondoo        = tfgen.NewProvider("mondoo", mondooProviderHclModifier...)
		dataADClientConfig    = tfgen.NewDataSource("azuread_client_config", "current")
		resourceAdApplication = tfgen.NewResource("azuread_application", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"display_name":  "mondoo_ms365",
				"owners":        []interface{}{dataADClientConfig.TraverseRef("object_id")},
				"marketing_url": "https://www.mondoo.com/",
			}),
			tfgen.HclResourceWithGenericBlocks(permissionsBlocks...),
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
		dataADClientConfig,
		dataMicrosoftPublishedAppIDs(),
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

	// App Registration Permissions Blocks
	blocks, err = tfgen.ObjectsToBlocks(Ms365AppPermissions.HclResources(resourceADServicePrincipal)...)
	if err != nil {
		return "", err
	}
	hclBlocks = tfgen.CombineHclBlocks(hclBlocks, blocks)

	return tfgen.CreateHclStringOutput(hclBlocks...), nil
}

// Permissions are a collection of access and permissions to multiple Microsoft resources (e.g. MicrosoftGraph, SharePoint, etc.)
type Permissions []Permission

// HclBlocks returns all the hcl blocks for all the defined permissions for a Microsoft 365 integration
func (p *Permissions) HclBlocks() (blocks []*hclwrite.Block, err error) {
	for _, permission := range Ms365AppPermissions {
		block, err := permission.HclBlock()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate ms365 permissions block")
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// HclResources returns all the hcl resources (service principals) for all the defined permissions for a Microsoft 365 integration
func (p *Permissions) HclResources(mondooServicePrincipal *tfgen.HclResource) (resources []tfgen.Object) {
	for _, permission := range Ms365AppPermissions {
		resources = append(resources, permission.HclResource())
		for _, access := range permission.Access {
			resources = append(resources, access.HclResource(mondooServicePrincipal, permission.HclResource()))
		}
	}
	return resources
}

// Permission represents access and permissions to a single Microsoft resource (e.g. MicrosoftGraph, SharePoint, etc.)
type Permission struct {
	ResourceID string
	Access     []ResourceAccess

	hclResource *tfgen.HclResource
	hclBlock    *hclwrite.Block
}

// HclResource returns the service principal resource of the Microsoft resource we are granting permissions
func (p *Permission) HclResource() *tfgen.HclResource {
	if p.hclResource == nil {
		p.hclResource = tfgen.NewResource("azuread_service_principal", p.ResourceID,
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"client_id":    dataMicrosoftPublishedAppIDs().TraverseRef("result", p.ResourceID),
				"use_existing": true,
			}),
		)
	}
	return p.hclResource
}

// HclBlock returns the hcl block for a single permission of an App Registration
func (p *Permission) HclBlock() (*hclwrite.Block, error) {
	if p.hclBlock == nil {
		resourceAccessBlocks, err := p.ResourceAccessBlocks()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create required_resource_access block")
		}

		p.hclBlock, err = tfgen.HclCreateGenericBlock("required_resource_access", nil,
			map[string]any{"resource_app_id": dataMicrosoftPublishedAppIDs().TraverseRef("result", p.ResourceID)},
			resourceAccessBlocks...,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create required_resource_access block")
		}
	}

	return p.hclBlock, nil
}

// ResourceAccessBlocks returns hcl blocks for all defined access of a single resource
func (p *Permission) ResourceAccessBlocks() (blocks []*hclwrite.Block, err error) {
	var block *hclwrite.Block
	for _, access := range p.Access {
		block, err = access.HclBlock(p.HclResource())
		if err != nil {
			return
		}

		blocks = append(blocks, block)
	}
	return
}

// ResourceAccess defines a single access to a resource
type ResourceAccess struct {
	ID   string
	Type string

	hclResource *tfgen.HclResource
	hclBlock    *hclwrite.Block
}

func (r *ResourceAccess) AppRoleID() string {
	return `app_role_ids["` + r.ID + `"]`
}

// The hcl resource name, we expect IDs like 'Policy.Read.All' which will translate into 'Policy_Read_All'
func (r *ResourceAccess) ResourceName() string {
	return strings.ReplaceAll(r.ID, ".", "_")
}

// These resources are used to grant admin consent for application permissions
// https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/app_role_assignment
func (r *ResourceAccess) HclResource(mondooServicePrincipal, resourceServicePrincipal *tfgen.HclResource) *tfgen.HclResource {
	if r.hclResource == nil {
		r.hclResource = tfgen.NewResource("azuread_app_role_assignment", r.ResourceName(),
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"app_role_id":         resourceServicePrincipal.TraverseRef(r.AppRoleID()),
				"resource_object_id":  resourceServicePrincipal.TraverseRef("object_id"),
				"principal_object_id": mondooServicePrincipal.TraverseRef("object_id"),
			}),
		)
	}
	return r.hclResource
}

// These hcl blocks are used to generate the required resource access of the App Registration
func (r *ResourceAccess) HclBlock(resourceServicePrincipal *tfgen.HclResource) (*hclwrite.Block, error) {
	var err error
	if r.hclBlock == nil {
		r.hclBlock, err = tfgen.HclCreateGenericBlock("resource_access", nil, map[string]any{
			"id":   resourceServicePrincipal.TraverseRef(r.AppRoleID()),
			"type": r.Type,
		})
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create resource_access block")
	}
	return r.hclBlock, nil
}

// Used to discover application IDs for APIs published by Microsoft
var dataADPublishedAppIDs *tfgen.HclResource

// https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/data-sources/application_published_app_ids
func dataMicrosoftPublishedAppIDs() *tfgen.HclResource {
	if dataADPublishedAppIDs == nil {
		dataADPublishedAppIDs = tfgen.NewDataSource("azuread_application_published_app_ids", "well_known")
	}
	return dataADPublishedAppIDs
}
