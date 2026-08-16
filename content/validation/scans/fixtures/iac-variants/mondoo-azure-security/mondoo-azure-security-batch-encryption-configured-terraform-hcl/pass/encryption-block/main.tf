resource "azurerm_batch_account" "example" {
  name                = "examplebatchaccount"
  resource_group_name = "example-rg"
  location            = "eastus"

  encryption {
    key_vault_key_id = "https://myvault.vault.azure.net/keys/mykey/version"
  }
}
