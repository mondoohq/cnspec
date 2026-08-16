resource "azurerm_recovery_services_vault" "pass" {
  name                = "example-vault"
  location            = "eastus"
  resource_group_name = "example-rg"
  sku                 = "Standard"

  encryption {
    key_id                            = "https://example-kv.vault.azure.net/keys/example-key/00000000000000000000000000000000"
    infrastructure_encryption_enabled = true
    use_system_assigned_identity      = true
  }

  identity {
    type = "SystemAssigned"
  }
}
