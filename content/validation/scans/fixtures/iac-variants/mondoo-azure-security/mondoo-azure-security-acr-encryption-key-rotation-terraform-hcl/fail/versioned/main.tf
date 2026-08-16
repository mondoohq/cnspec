resource "azurerm_container_registry" "fail" {
  name                = "examplereg"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "Premium"

  encryption {
    key_vault_key_id   = "https://myvault.vault.azure.net/keys/mykey/abc123def456version"
    identity_client_id = "00000000-0000-0000-0000-000000000000"
  }
}
