resource "azurerm_ai_foundry" "example" {
  name                = "example-hub"
  location            = "eastus"
  resource_group_name = "rg"
  storage_account_id  = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa"
  key_vault_id        = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/kv"

  managed_network {
    isolation_mode = "Disabled"
  }

  identity {
    type = "SystemAssigned"
  }
}
