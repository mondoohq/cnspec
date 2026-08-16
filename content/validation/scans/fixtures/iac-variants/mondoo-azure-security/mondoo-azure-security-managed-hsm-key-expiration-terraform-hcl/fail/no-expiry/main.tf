# A key with no expiration date stays valid indefinitely, so there is no
# forcing function for rotation.
resource "azurerm_key_vault_managed_hardware_security_module_key" "signing" {
  name           = "payment-signing"
  managed_hsm_id = azurerm_key_vault_managed_hardware_security_module.prod.id
  key_type       = "EC-HSM"
  curve          = "P-256"
  key_opts       = ["sign", "verify"]
}
