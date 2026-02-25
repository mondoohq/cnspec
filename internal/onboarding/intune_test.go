// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	subject "go.mondoo.com/cnspec/v13/internal/onboarding"
)

func TestGenerateIntuneHCL_Basic(t *testing.T) {
	code, err := subject.GenerateIntuneHCL(subject.IntuneIntegration{
		Name:  "test-intune-integration",
		Space: "space-xyz",
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

data "azuread_client_config" "current" {
}

data "azuread_application_published_app_ids" "well_known" {
}

resource "azuread_application_password" "mondoo" {
  application_id = azuread_application.mondoo.id
  display_name   = "mondoo-intune-credential"
}

resource "azuread_service_principal" "mondoo" {
  app_role_assignment_required = false
  client_id                    = azuread_application.mondoo.client_id
  owners                       = [data.azuread_client_config.current.object_id]
}

resource "azuread_application" "mondoo" {
  display_name  = "mondoo-intune"
  marketing_url = "https://www.mondoo.com/"
  owners        = [data.azuread_client_config.current.object_id]

  required_resource_access {
    resource_app_id = data.azuread_application_published_app_ids.well_known.result.MicrosoftGraph
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementManagedDevices.ReadWrite.All"]
      type = "Role"
    }
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementConfiguration.ReadWrite.All"]
      type = "Role"
    }
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementApps.ReadWrite.All"]
      type = "Role"
    }
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["Directory.Read.All"]
      type = "Role"
    }
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementServiceConfig.ReadWrite.All"]
      type = "Role"
    }
  }
}

resource "mondoo_integration_ms_intune" "this" {
  client_id = azuread_application.mondoo.client_id
  credentials = {
    client_secret = azuread_application_password.mondoo.value
  }
  depends_on = [azuread_service_principal.mondoo, azuread_application_password.mondoo]
  name       = "test-intune-integration"
  tenant_id  = data.azuread_client_config.current.tenant_id
}

resource "azuread_service_principal" "MicrosoftGraph" {
  client_id    = data.azuread_application_published_app_ids.well_known.result.MicrosoftGraph
  use_existing = true
}

resource "azuread_app_role_assignment" "DeviceManagementManagedDevices_ReadWrite_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementManagedDevices.ReadWrite.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}

resource "azuread_app_role_assignment" "DeviceManagementConfiguration_ReadWrite_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementConfiguration.ReadWrite.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}

resource "azuread_app_role_assignment" "DeviceManagementApps_ReadWrite_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementApps.ReadWrite.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}

resource "azuread_app_role_assignment" "Directory_Read_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["Directory.Read.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}

resource "azuread_app_role_assignment" "DeviceManagementServiceConfig_ReadWrite_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementServiceConfig.ReadWrite.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}
`
	assert.Equal(t, expected, code)
}

func TestGenerateIntuneHCL_Minimal(t *testing.T) {
	subject.UuidGenerator = func() string {
		return "bcb6e112-30f8-434a-926b-88afcea5fb91"
	}
	code, err := subject.GenerateIntuneHCL(subject.IntuneIntegration{})
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

data "azuread_client_config" "current" {
}

data "azuread_application_published_app_ids" "well_known" {
}

resource "azuread_application_password" "mondoo" {
  application_id = azuread_application.mondoo.id
  display_name   = "mondoo-intune-credential"
}

resource "azuread_service_principal" "mondoo" {
  app_role_assignment_required = false
  client_id                    = azuread_application.mondoo.client_id
  owners                       = [data.azuread_client_config.current.object_id]
}

resource "azuread_application" "mondoo" {
  display_name  = "mondoo-intune"
  marketing_url = "https://www.mondoo.com/"
  owners        = [data.azuread_client_config.current.object_id]

  required_resource_access {
    resource_app_id = data.azuread_application_published_app_ids.well_known.result.MicrosoftGraph
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementManagedDevices.ReadWrite.All"]
      type = "Role"
    }
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementConfiguration.ReadWrite.All"]
      type = "Role"
    }
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementApps.ReadWrite.All"]
      type = "Role"
    }
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["Directory.Read.All"]
      type = "Role"
    }
    resource_access {
      id   = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementServiceConfig.ReadWrite.All"]
      type = "Role"
    }
  }
}

resource "mondoo_integration_ms_intune" "this" {
  client_id = azuread_application.mondoo.client_id
  credentials = {
    client_secret = azuread_application_password.mondoo.value
  }
  depends_on = [azuread_service_principal.mondoo, azuread_application_password.mondoo]
  name       = "subscription-88afcea5fb91"
  tenant_id  = data.azuread_client_config.current.tenant_id
}

resource "azuread_service_principal" "MicrosoftGraph" {
  client_id    = data.azuread_application_published_app_ids.well_known.result.MicrosoftGraph
  use_existing = true
}

resource "azuread_app_role_assignment" "DeviceManagementManagedDevices_ReadWrite_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementManagedDevices.ReadWrite.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}

resource "azuread_app_role_assignment" "DeviceManagementConfiguration_ReadWrite_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementConfiguration.ReadWrite.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}

resource "azuread_app_role_assignment" "DeviceManagementApps_ReadWrite_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementApps.ReadWrite.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}

resource "azuread_app_role_assignment" "Directory_Read_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["Directory.Read.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}

resource "azuread_app_role_assignment" "DeviceManagementServiceConfig_ReadWrite_All" {
  app_role_id         = azuread_service_principal.MicrosoftGraph.app_role_ids["DeviceManagementServiceConfig.ReadWrite.All"]
  principal_object_id = azuread_service_principal.mondoo.object_id
  resource_object_id  = azuread_service_principal.MicrosoftGraph.object_id
}
`
	assert.Equal(t, expected, code)
}
