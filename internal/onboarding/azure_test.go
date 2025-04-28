// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	subject "go.mondoo.com/cnspec/v11/internal/onboarding"
)

func TestGenerateAzureHCL(t *testing.T) {
	code, err := subject.GenerateAzureHCL(subject.AzureIntegration{})
	assert.Nil(t, err)
	expected := `terraform {
  required_providers {
    mondoo = {
      source  = "mondoohq/mondoo"
      version = "~> 0.19"
    }
  }
}

provider "mondoo" {
}

provider "azuread" {
}

provider "azurerm" {

  features {
  }
}

data "azuread_client_config" "current" {
}

data "azurerm_subscriptions" "available" {
}

resource "tls_private_key" "credential" {
  algorithm = "RSA"
}

resource "tls_self_signed_cert" "credential" {
  allowed_uses          = ["key_encipherment", "digital_signature", "data_encipherment", "cert_signing"]
  early_renewal_hours   = 3
  private_key_pem       = tls_private_key.credential.private_key_pem
  validity_period_hours = 4096

  subject {
    common_name = "mondoo"
  }
}

resource "azuread_application_certificate" "mondoo" {
  application_id = azuread_application.mondoo.id
  type           = "AsymmetricX509Cert"
  value          = tls_self_signed_cert.credential.cert_pem
}

resource "azuread_service_principal" "mondoo" {
  client_id = azuread_application.mondoo.client_id
  owners    = [data.azuread_client_config.current.object_id]
}

resource "azuread_application" "mondoo" {
  display_name  = "mondoo_security"
  marketing_url = "https://www.mondoo.com/"
  owners        = [data.azuread_client_config.current.object_id]
}

resource "azuread_directory_role" "readers" {
  display_name = "Directory Readers"
}

resource "azuread_directory_role_assignment" "readers" {
  depends_on          = [time_sleep.wait_time]
  principal_object_id = azuread_service_principal.mondoo.object_id
  role_id             = azuread_directory_role.readers.template_id
}

resource "time_sleep" "wait_time" {
  create_duration = "60s"
}

resource "azurerm_role_assignment" "reader" {
  count                = length(data.azurerm_subscriptions.available.subscriptions)
  principal_id         = azuread_service_principal.mondoo.object_id
  role_definition_name = "Reader"
  scope                = data.azurerm_subscriptions.available.subscriptions[count.index].id
}

resource "mondoo_integration_azure" "this" {
  client_id = azuread_application.mondoo.client_id
  credentials = {
    pem_file = join("\n", [tls_self_signed_cert.credential.cert_pem, tls_private_key.credential.private_key_pem])
  }
  depends_on = [azuread_service_principal.mondoo, azuread_application_certificate.mondoo, azuread_directory_role_assignment.readers, azurerm_role_assignment.reader]
  name       = "subscription-"
  scan_vms   = false
  tenant_id  = data.azuread_client_config.current.tenant_id
}
`
	assert.Equal(t, expected, code)
}

func TestGenerateAzureHCLFull(t *testing.T) {
	code, err := subject.GenerateAzureHCL(subject.AzureIntegration{
		Name:    "test-integration",
		Space:   "hungry-poet-1234",
		Primary: "abc-123-xyz-456-1",
		Allow:   []string{"abc-123-xyz-456-1", "abc-123-xyz-456-2", "abc-123-xyz-456-3"},
		ScanVMs: true,
	})
	assert.Nil(t, err)
	expected := `terraform {
  required_providers {
    mondoo = {
      source  = "mondoohq/mondoo"
      version = "~> 0.19"
    }
  }
}

provider "mondoo" {
  space = "hungry-poet-1234"
}

provider "azuread" {
}

provider "azurerm" {
  subscription_id = "abc-123-xyz-456-1"

  features {
  }
}

data "azuread_client_config" "current" {
}

data "azurerm_subscriptions" "available" {
}

resource "tls_private_key" "credential" {
  algorithm = "RSA"
}

resource "tls_self_signed_cert" "credential" {
  allowed_uses          = ["key_encipherment", "digital_signature", "data_encipherment", "cert_signing"]
  early_renewal_hours   = 3
  private_key_pem       = tls_private_key.credential.private_key_pem
  validity_period_hours = 4096

  subject {
    common_name = "mondoo"
  }
}

resource "azuread_application_certificate" "mondoo" {
  application_id = azuread_application.mondoo.id
  type           = "AsymmetricX509Cert"
  value          = tls_self_signed_cert.credential.cert_pem
}

resource "azuread_service_principal" "mondoo" {
  client_id = azuread_application.mondoo.client_id
  owners    = [data.azuread_client_config.current.object_id]
}

resource "azuread_application" "mondoo" {
  display_name  = "mondoo_security"
  marketing_url = "https://www.mondoo.com/"
  owners        = [data.azuread_client_config.current.object_id]
}

resource "azuread_directory_role" "readers" {
  display_name = "Directory Readers"
}

resource "azuread_directory_role_assignment" "readers" {
  depends_on          = [time_sleep.wait_time]
  principal_object_id = azuread_service_principal.mondoo.object_id
  role_id             = azuread_directory_role.readers.template_id
}

resource "time_sleep" "wait_time" {
  create_duration = "60s"
}

resource "azurerm_role_assignment" "reader-0" {
  principal_id         = azuread_service_principal.mondoo.object_id
  role_definition_name = "Reader"
  scope                = "/subscriptions/abc-123-xyz-456-1"
}

resource "azurerm_role_assignment" "reader-1" {
  principal_id         = azuread_service_principal.mondoo.object_id
  role_definition_name = "Reader"
  scope                = "/subscriptions/abc-123-xyz-456-2"
}

resource "azurerm_role_assignment" "reader-2" {
  principal_id         = azuread_service_principal.mondoo.object_id
  role_definition_name = "Reader"
  scope                = "/subscriptions/abc-123-xyz-456-3"
}

resource "mondoo_integration_azure" "this" {
  client_id = azuread_application.mondoo.client_id
  credentials = {
    pem_file = join("\n", [tls_self_signed_cert.credential.cert_pem, tls_private_key.credential.private_key_pem])
  }
  depends_on              = [azuread_service_principal.mondoo, azuread_application_certificate.mondoo, azuread_directory_role_assignment.readers, azurerm_role_assignment.reader-0, azurerm_role_assignment.reader-1, azurerm_role_assignment.reader-2]
  name                    = "test-integration"
  scan_vms                = true
  subscription_allow_list = ["abc-123-xyz-456-1", "abc-123-xyz-456-2", "abc-123-xyz-456-3"]
  tenant_id               = data.azuread_client_config.current.tenant_id
}

resource "azurerm_role_assignment" "mondoo_security" {
  count              = length(data.azurerm_subscriptions.available.subscriptions)
  principal_id       = azuread_service_principal.mondoo.object_id
  role_definition_id = azurerm_role_definition.mondoo_security.role_definition_resource_id
  scope              = data.azurerm_subscriptions.available.subscriptions[count.index].id
}

resource "azurerm_role_definition" "mondoo_security" {
  assignable_scopes = data.azurerm_subscriptions.available.subscriptions[*].id
  description       = "Allow Mondoo Security to use run commands for Virtual Machine scanning"
  name              = "tf-mondoo-security-role"
  scope             = "/subscriptions/abc-123-xyz-456-1"

  permissions {
    actions  = ["Microsoft.Compute/virtualMachines/runCommands/read", "Microsoft.Compute/virtualMachines/runCommands/write", "Microsoft.Compute/virtualMachines/runCommands/delete", "Microsoft.Compute/virtualMachines/runCommand/action"]
  }
}
`
	assert.Equal(t, expected, code)
}
