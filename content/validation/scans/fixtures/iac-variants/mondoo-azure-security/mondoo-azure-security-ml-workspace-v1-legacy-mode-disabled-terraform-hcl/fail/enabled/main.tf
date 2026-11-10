resource "azurerm_machine_learning_workspace" "example" {
  name                    = "example-workspace"
  location                = "eastus"
  resource_group_name     = "rg"
  application_insights_id = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.Insights/components/ai"
  key_vault_id            = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/kv"
  storage_account_id      = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa"
  v1_legacy_mode_enabled  = true

  identity {
    type = "SystemAssigned"
  }
}
