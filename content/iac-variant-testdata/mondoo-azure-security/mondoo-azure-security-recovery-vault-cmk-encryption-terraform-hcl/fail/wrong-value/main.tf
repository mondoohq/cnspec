resource "azurerm_recovery_services_vault" "fail" {
  name                = "example-vault"
  location            = "eastus"
  resource_group_name = "example-rg"
  sku                 = "Standard"

  encryption {
    key_id                            = "https://example-kv.vault.azure.net/keys/example-key/00000000000000000000000000000000"
    infrastructure_encryption_enabled = false
    use_system_assigned_identity      = true
  }

  identity {
    type = "SystemAssigned"
  }
}
