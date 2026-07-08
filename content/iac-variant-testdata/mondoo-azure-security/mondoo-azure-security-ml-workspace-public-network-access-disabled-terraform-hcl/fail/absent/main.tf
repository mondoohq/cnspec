resource "azurerm_machine_learning_workspace" "fail" {
  name                    = "example-ws"
  location                = "eastus"
  resource_group_name     = "example-rg"
  application_insights_id  = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.Insights/components/ai"
  key_vault_id            = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/kv"
  storage_account_id      = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa"

  identity {
    type = "SystemAssigned"
  }
}
