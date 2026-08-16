resource "azurerm_container_registry" "pass" {
  name                = "examplereg"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "Premium"

  encryption {
    key_vault_key_id   = azurerm_key_vault_key.example.versionless_id
    identity_client_id = "00000000-0000-0000-0000-000000000000"
  }
}
