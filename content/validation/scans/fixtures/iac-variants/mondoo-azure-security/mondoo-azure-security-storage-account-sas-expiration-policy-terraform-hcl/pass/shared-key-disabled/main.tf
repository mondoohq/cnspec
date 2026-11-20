# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1

resource "azurerm_storage_account" "data" {
  name                      = "contosodata"
  resource_group_name       = "rg-data"
  location                  = "eastus"
  account_tier              = "Standard"
  account_replication_type  = "LRS"
  shared_access_key_enabled = false
}
