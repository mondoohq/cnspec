// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding

import (
	"encoding/json"
	"fmt"
	"os/exec"
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
		}
	)

	dynamicObjects := []tfgen.Object{
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
	}
	integrationDependsOn := []any{
		resourceADServicePrincipal.TraverseRef(),
		resourceADApplicationCertificate.TraverseRef(),
		resourceADReadersRoleAssignment.TraverseRef(),
	}

	// Allow and Deny are mutually exclusive and can't be added together
	if len(integration.Allow) != 0 {
		// add the mondoo integration resource attribute to allow provided subscriptions
		integrationAttributes["subscription_allow_list"] = integration.Allow
		// grant reader role to only the allowed list of subscriptions
		readerRoleAssignmentBlocks, dependencies, err := azureRMReaderRoleAssignmentBlocks(integration, resourceADServicePrincipal)
		if err != nil {
			return "", err
		}
		dynamicObjects = append(dynamicObjects, readerRoleAssignmentBlocks...)
		integrationDependsOn = append(integrationDependsOn, dependencies...)
	} else {
		// grant reader role to all subscriptions
		allSubscriptionsBlocks, dependencies, err := generateAllSubscriptionsBlocks(integration, resourceADServicePrincipal)
		if err != nil {
			return "", err
		}
		dynamicObjects = append(dynamicObjects, allSubscriptionsBlocks...)
		integrationDependsOn = append(integrationDependsOn, dependencies...)
		if len(integration.Deny) != 0 {
			// add the mondoo integration resource attribute to deny provided subscriptions
			integrationAttributes["subscription_deny_list"] = integration.Deny
			// TODO can we skip subscriptions if deny is provided
		}
	}

	// add the main mondoo integration
	integrationAttributes["depends_on"] = integrationDependsOn
	resourceMondooIntegration := tfgen.NewResource("mondoo_integration_azure", "this",
		tfgen.HclResourceWithAttributes(integrationAttributes),
	)
	dynamicObjects = append(dynamicObjects, resourceMondooIntegration)

	blocks, err := tfgen.ObjectsToBlocks(dynamicObjects...)
	if err != nil {
		return "", err
	}

	hclBlocks := tfgen.CombineHclBlocks(requiredProvidersBlock, blocks)
	return tfgen.CreateHclStringOutput(hclBlocks...), nil
}

func generateAllSubscriptionsBlocks(integration AzureIntegration, resourceADServicePrincipal *tfgen.HclResource) ([]tfgen.Object, []any, error) {
	resources := []tfgen.Object{}
	dependsOn := []any{}

	// data source to fetch all available subscriptions
	dataRMAllSubscriptions := tfgen.NewDataSource("azurerm_subscriptions", "available")
	resources = append(resources, dataRMAllSubscriptions)
	dependsOn = append(dependsOn, dataRMAllSubscriptions.TraverseRef())

	// grant reader role to all subscriptions
	resourceRMReaderRoleAssignment := tfgen.NewResource("azurerm_role_assignment", "reader",
		tfgen.HclResourceWithAttributes(tfgen.Attributes{
			"role_definition_name": "Reader",

			"count":        tfgen.NewFuncCall("length", dataRMAllSubscriptions.TraverseRef("subscriptions")),
			"scope":        dataRMAllSubscriptions.TraverseRef("subscriptions[count.index]", "id"),
			"principal_id": resourceADServicePrincipal.TraverseRef("object_id"),
		}),
	)
	resources = append(resources, resourceRMReaderRoleAssignment)
	dependsOn = append(dependsOn, resourceRMReaderRoleAssignment.TraverseRef())

	// custom role needed only when scanning VMs
	if integration.ScanVMs {
		customRolePermissionsBlock, err := tfgen.HclCreateGenericBlock("permissions", nil,
			tfgen.Attributes{
				"actions ": []string{
					"Microsoft.Compute/virtualMachines/runCommands/read",
					"Microsoft.Compute/virtualMachines/runCommands/write",
					"Microsoft.Compute/virtualMachines/runCommands/delete",
					"Microsoft.Compute/virtualMachines/runCommand/action",
				},
				// not_actions = []
				// data_actions = []
				// not_data_actions = []
			},
		)
		if err != nil {
			return resources, dependsOn, errors.Wrap(err, "failed to generate custom role permissions block")
		}
		resourceRMCustomRoleDefinition := tfgen.NewResource("azurerm_role_definition", "mondoo_security",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"name":              "tf-mondoo-security-role",
				"description":       "Allow Mondoo Security to use run commands for Virtual Machine scanning",
				"scope":             fmt.Sprintf("/subscriptions/%s", integration.Primary),
				"assignable_scopes": dataRMAllSubscriptions.TraverseRef("subscriptions[*]", "id"),
			}),
			tfgen.HclResourceWithGenericBlocks(customRolePermissionsBlock),
		)
		resources = append(resources, resourceRMCustomRoleDefinition)
		dependsOn = append(dependsOn, resourceRMCustomRoleDefinition.TraverseRef())

		// adds the custom role to all subscriptions
		resourceRMCustomRoleAssignment := tfgen.NewResource("azurerm_role_assignment", "mondoo_security",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"role_definition_id": resourceRMCustomRoleDefinition.TraverseRef("role_definition_resource_id"),

				"count":        tfgen.NewFuncCall("length", dataRMAllSubscriptions.TraverseRef("subscriptions")),
				"scope":        dataRMAllSubscriptions.TraverseRef("subscriptions[count.index]", "id"),
				"principal_id": resourceADServicePrincipal.TraverseRef("object_id"),
			}),
		)
		resources = append(resources, resourceRMCustomRoleAssignment)
		dependsOn = append(dependsOn, resourceRMCustomRoleAssignment.TraverseRef())
	}

	return resources, dependsOn, nil
}

// azureRMReaderRoleAssignmentBlocks creates role assignment blocks for a list of subscription IDs,
// it returns the resources and the list of depends_on objects
func azureRMReaderRoleAssignmentBlocks(integration AzureIntegration, resourceADServicePrincipal *tfgen.HclResource) ([]tfgen.Object, []any, error) {
	resources := []tfgen.Object{}
	dependsOn := []any{}

	// For scope format see:
	// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment
	subscriptionIDs := make([]string, len(integration.Allow))
	for i, id := range integration.Allow {
		subscriptionIDs[i] = fmt.Sprintf("/subscriptions/%s", id)
	}

	// custom role needed only when scanning VMs
	var resourceRMCustomRoleDefinition *tfgen.HclResource
	if integration.ScanVMs {
		customRolePermissionsBlock, err := tfgen.HclCreateGenericBlock("permissions", nil,
			tfgen.Attributes{
				"actions ": []string{
					"Microsoft.Compute/virtualMachines/runCommands/read",
					"Microsoft.Compute/virtualMachines/runCommands/write",
					"Microsoft.Compute/virtualMachines/runCommands/delete",
					"Microsoft.Compute/virtualMachines/runCommand/action",
				},
				// not_actions = []
				// data_actions = []
				// not_data_actions = []
			},
		)
		if err != nil {
			return resources, dependsOn, errors.Wrap(err, "failed to generate custom role permissions block")
		}

		resourceRMCustomRoleDefinition = tfgen.NewResource("azurerm_role_definition", "mondoo_security",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"name":              "tf-mondoo-security-role",
				"description":       "Allow Mondoo Security to use run commands for Virtual Machine scanning",
				"scope":             fmt.Sprintf("/subscriptions/%s", integration.Primary),
				"assignable_scopes": subscriptionIDs,
			}),
			tfgen.HclResourceWithGenericBlocks(customRolePermissionsBlock),
		)
		resources = append(resources, resourceRMCustomRoleDefinition)
		dependsOn = append(dependsOn, resourceRMCustomRoleDefinition.TraverseRef())

	}

	for i, subscriptionID := range subscriptionIDs {
		resource := tfgen.NewResource("azurerm_role_assignment", fmt.Sprintf("reader-%d", i),
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"role_definition_name": "Reader",
				"scope":                subscriptionID,
				"principal_id":         resourceADServicePrincipal.TraverseRef("object_id"),
			}),
		)
		resources = append(resources, resource)
		dependsOn = append(dependsOn, resource.TraverseRef())

		// add the custom role to allowed subscriptions only when scanning VMs
		if integration.ScanVMs {
			resource := tfgen.NewResource("azurerm_role_assignment", fmt.Sprintf("mondoo_security-%d", i),
				tfgen.HclResourceWithAttributes(tfgen.Attributes{
					"role_definition_id": resourceRMCustomRoleDefinition.TraverseRef("role_definition_resource_id"),
					"scope":              subscriptionID,
					"principal_id":       resourceADServicePrincipal.TraverseRef("object_id"),
				}),
			)
			resources = append(resources, resource)
			dependsOn = append(dependsOn, resource.TraverseRef())
		}
	}

	return resources, dependsOn, nil
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

// VerifyUserRoleAssignments checks that the user running onboarding has the right role
// assignments to run the automation.
//
// We need one of "Privileged Role Administrator" or "Global Administrator"
// => https://learn.microsoft.com/en-us/entra/identity/role-based-access-control/permissions-reference#privileged-role-administrator
//
// Note that we run Azure CLI commands so that cnspec doesn't have to import the entire
// Azure SDK which will make the binary much bigger, which is unnecessary for the amount
// of checks we are running today.
func VerifyUserRoleAssignments() error {
	var BuiltInRoles = map[string]string{
		"Privileged Role Administrator": "e8611ab8-c189-46e8-94e1-60213ab1f814",
		"Global Administrator":          "62e90394-69f5-4237-9190-012177145e10",
	}

	principalID, err := getCurrentUserServicePrincipal()
	if err != nil {
		return err
	}
	log.Info().Msgf("user service principal %s", theme.DefaultTheme.Primary(principalID))

	var errs []error
	theUserWillFail := true
	for roleName, roleID := range BuiltInRoles {
		assignments, err := getRoleAssignmentsForRole(roleID)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "- %s", roleName))
			continue
		}

		if assignments.ContainsPrincipalID(principalID) {
			// the user will succeed
			log.Info().Msgf("role assignment %s found", theme.DefaultTheme.Success(roleName))
			theUserWillFail = false
			break
		}

		log.Info().Msgf("role assignment %s not found", theme.DefaultTheme.Secondary(roleName))
		errs = append(errs, errors.Newf("- %s: not found", roleName))
	}

	if theUserWillFail {
		return errors.Newf(
			"one of the following role assignments are required:\n\n%s",
			errors.Join(errs...),
		)
	}
	return nil
}

func getCurrentUserServicePrincipal() (string, error) {
	signedUserJSON, err := exec.Command("az", "ad", "signed-in-user", "show", "-o", "json").Output()
	if err != nil {
		return "", errors.Wrap(err, "unable to get details for the currently logged-in user")
	}
	var userSP struct {
		ID string `json:"id"`
	}
	err = json.Unmarshal(signedUserJSON, &userSP)
	return userSP.ID, errors.Wrap(err, "unable to parse user details")
}

type roleAssignmentResponse struct {
	Value []roleAssignment
}

func (r roleAssignmentResponse) ContainsPrincipalID(id string) bool {
	for _, assignment := range r.Value {
		if assignment.PrincipalID == id {
			return true
		}
	}
	return false
}

type roleAssignment struct {
	DirectoryScopeID string
	ID               string
	PrincipalID      string
	RoleDefinitionID string
}

func getRoleAssignmentsForRole(roleID string) (roleAssignmentResponse, error) {
	restURL := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/roleManagement/directory/roleAssignments?$filter=roleDefinitionId eq '%s'",
		roleID,
	)
	var roleAssignments roleAssignmentResponse
	restResponse, err := exec.Command("az", "rest", "--method", "get", "--url", restURL, "-o", "json").Output()
	if err != nil {
		return roleAssignments, errors.Wrap(err, "unable to list role assignments via REST")
	}

	err = json.Unmarshal(restResponse, &roleAssignments)
	return roleAssignments, errors.Wrap(err, "unable to parse role assignments")
}
