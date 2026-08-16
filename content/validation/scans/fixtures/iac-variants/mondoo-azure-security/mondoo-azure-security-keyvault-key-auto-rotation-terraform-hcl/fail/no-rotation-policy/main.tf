resource "azurerm_key_vault_key" "fail" {
  name         = "example-key"
  key_vault_id = "/subscriptions/00000000/resourceGroups/example-rg/providers/Microsoft.KeyVault/vaults/example-kv"
  key_type     = "RSA"
  key_size     = 2048

  key_opts = ["decrypt", "encrypt", "sign", "unwrapKey", "verify", "wrapKey"]
}
