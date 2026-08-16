resource "azurerm_disk_encryption_set" "example" {
  name                      = "des-example"
  resource_group_name       = "example-rg"
  location                  = "eastus"
  key_vault_key_id          = azurerm_key_vault_key.example.id
  auto_key_rotation_enabled = true

  identity {
    type = "SystemAssigned"
  }
}
