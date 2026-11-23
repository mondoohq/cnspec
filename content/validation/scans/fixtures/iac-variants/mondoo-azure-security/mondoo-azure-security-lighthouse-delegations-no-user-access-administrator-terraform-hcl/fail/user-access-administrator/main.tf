# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1

data "azurerm_subscription" "current" {}

resource "azurerm_lighthouse_definition" "provider" {
  name               = "Managed operations"
  managing_tenant_id = "00000000-0000-0000-0000-000000000001"
  scope              = data.azurerm_subscription.current.id

  authorization {
    principal_id           = "00000000-0000-0000-0000-000000000002"
    role_definition_id     = "43d0d8ad-25c7-4714-9337-8ba259a9fe05"
    principal_display_name = "Provider operators"
  }

  authorization {
    principal_id                  = "00000000-0000-0000-0000-000000000003"
    role_definition_id            = "18d7d88d-d35e-4fb5-a5c3-7773c20a72d9"
    principal_display_name        = "Provider automation"
    delegated_role_definition_ids = ["b24988ac-6180-42a0-ab88-20f7382dd24c"]
  }
}
