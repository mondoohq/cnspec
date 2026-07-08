resource "azurerm_key_vault_key" "pass" {
  name         = "example-key"
  key_vault_id = "/subscriptions/00000000/resourceGroups/example-rg/providers/Microsoft.KeyVault/vaults/example-kv"
  key_type     = "RSA"
  key_size     = 2048

  key_opts = ["decrypt", "encrypt", "sign", "unwrapKey", "verify", "wrapKey"]

  rotation_policy {
    automatic {
      time_before_expiry = "P30D"
    }

    expire_after         = "P90D"
    notify_before_expiry = "P29D"
  }
}
