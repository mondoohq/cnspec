resource "azurerm_key_vault_secret" "example" {
  name         = "db-password"
  value        = "szechuan"
  key_vault_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.KeyVault/vaults/example-kv"
}
