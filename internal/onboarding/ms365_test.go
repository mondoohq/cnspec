// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	subject "go.mondoo.com/cnspec/v11/internal/onboarding"
)

func TestGenerateMs365HCL_Basic(t *testing.T) {
	code, err := subject.GenerateMs365HCL(subject.Ms365Integration{
		Name:    "test-ms365-integration",
		Space:   "space-xyz",
		Primary: "00000000-0000-0000-0000-000000000000",
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
  space = "space-xyz"
}

provider "azuread" {
}

provider "azurerm" {
  subscription_id = "00000000-0000-0000-0000-000000000000"

  features {
  }
}

data "azuread_client_config" "current" {
}

resource "tls_private_key" "credential" {
  algorithm = "RSA"
  rsa_bits  = 4096
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
  app_role_assignment_required = false
  client_id                    = azuread_application.mondoo.client_id
  owners                       = [data.azuread_client_config.current.object_id]
}

resource "azuread_application" "mondoo" {
  display_name  = "mondoo_ms365"
  marketing_url = "https://www.mondoo.com/"
  owners        = [data.azuread_client_config.current.object_id]
}

resource "azuread_directory_role" "global_reader" {
  display_name = "Global Reader"
}

resource "azuread_directory_role_assignment" "global_reader" {
  depends_on          = [time_sleep.wait_time]
  principal_object_id = azuread_service_principal.mondoo.object_id
  role_id             = azuread_directory_role.global_reader.template_id
}

resource "azuread_directory_role" "exchange_admin" {
  display_name = "Exchange Administrator"
}

resource "azuread_directory_role_assignment" "exchange_admin" {
  depends_on          = [time_sleep.wait_time]
  principal_object_id = azuread_service_principal.mondoo.object_id
  role_id             = azuread_directory_role.exchange_admin.object_id
}

resource "time_sleep" "wait_time" {
  create_duration = "60s"
}

resource "mondoo_integration_ms365" "this" {
  client_id = azuread_application.mondoo.client_id
  credentials = {
    pem_file = join("\n", [tls_self_signed_cert.credential.cert_pem, tls_private_key.credential.private_key_pem])
  }
  depends_on = [azuread_service_principal.mondoo, azuread_application_certificate.mondoo, azuread_directory_role_assignment.global_reader]
  name       = "test-ms365-integration"
  tenant_id  = data.azuread_client_config.current.tenant_id
}
`
	assert.Equal(t, expected, code)
}

func TestGenerateMs365HCL_Minimal(t *testing.T) {
	code, err := subject.GenerateMs365HCL(subject.Ms365Integration{})
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

resource "tls_private_key" "credential" {
  algorithm = "RSA"
  rsa_bits  = 4096
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
  app_role_assignment_required = false
  client_id                    = azuread_application.mondoo.client_id
  owners                       = [data.azuread_client_config.current.object_id]
}

resource "azuread_application" "mondoo" {
  display_name  = "mondoo_ms365"
  marketing_url = "https://www.mondoo.com/"
  owners        = [data.azuread_client_config.current.object_id]
}

resource "azuread_directory_role" "global_reader" {
  display_name = "Global Reader"
}

resource "azuread_directory_role_assignment" "global_reader" {
  depends_on          = [time_sleep.wait_time]
  principal_object_id = azuread_service_principal.mondoo.object_id
  role_id             = azuread_directory_role.global_reader.template_id
}

resource "azuread_directory_role" "exchange_admin" {
  display_name = "Exchange Administrator"
}

resource "azuread_directory_role_assignment" "exchange_admin" {
  depends_on          = [time_sleep.wait_time]
  principal_object_id = azuread_service_principal.mondoo.object_id
  role_id             = azuread_directory_role.exchange_admin.object_id
}

resource "time_sleep" "wait_time" {
  create_duration = "60s"
}

resource "mondoo_integration_ms365" "this" {
  client_id = azuread_application.mondoo.client_id
  credentials = {
    pem_file = join("\n", [tls_self_signed_cert.credential.cert_pem, tls_private_key.credential.private_key_pem])
  }
  depends_on = [azuread_service_principal.mondoo, azuread_application_certificate.mondoo, azuread_directory_role_assignment.global_reader]
  name       = "subscription-"
  tenant_id  = data.azuread_client_config.current.tenant_id
}
`
	assert.Equal(t, expected, code)
}
