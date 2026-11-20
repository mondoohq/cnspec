# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1

data "azurerm_subscription" "current" {}

resource "azurerm_lighthouse_definition" "provider" {
  name               = "Managed monitoring"
  managing_tenant_id = "00000000-0000-0000-0000-000000000001"
  scope              = data.azurerm_subscription.current.id

  authorization {
    principal_id           = "00000000-0000-0000-0000-000000000002"
    role_definition_id     = "43d0d8ad-25c7-4714-9337-8ba259a9fe05"
    principal_display_name = "Provider operators"
  }
}
